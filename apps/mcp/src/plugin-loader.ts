/**
 * Plugin MCP loader.
 *
 * At MCP server startup this module:
 * 1. Fetches the list of enabled plugins from `GET /api/v1/plugins`.
 * 2. For each plugin that declares `manifest.mcp.remoteEntryUrl`, dynamically
 *    imports the remote entry module.
 * 3. Validates the module's default export against the PluginMCPEntry contract.
 * 4. Collects all tool definitions and registers dispatch mappings.
 *
 * The resulting {@link PluginRegistry} is consumed by `server.ts` to merge
 * plugin tools with core tools and route tool calls.
 */

import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import type { PacaConfig } from "./types/index.js";

// ── Types ─────────────────────────────────────────────────────────────────────

interface InstalledPlugin {
	id: string;
	name: string;
	version: string;
	enabled: boolean;
	manifest: {
		mcp?: {
			remoteEntryUrl: string;
		};
	};
}

interface PluginListResponse {
	plugins: InstalledPlugin[];
}

/** The contract a plugin's default export must satisfy. */
interface PluginMCPEntry {
	tools: Tool[];
	handleToolCall(
		name: string,
		args: Record<string, unknown>,
		context: PluginMCPContext,
	): Promise<PluginToolResult>;
}

interface PluginMCPContext {
	pluginId: string;
	baseURL: string;
	apiKey: string;
}

interface PluginToolResult {
	content: Array<{ type: string; [key: string]: unknown }>;
	isError?: boolean;
}

/** Loaded plugin: entry module + metadata. */
interface LoadedPlugin {
	pluginId: string;
	entry: PluginMCPEntry;
}

// ── Registry ──────────────────────────────────────────────────────────────────

/**
 * Holds all successfully loaded plugin MCP entries and their merged tool list.
 * Constructed by {@link loadPlugins}.
 */
export class PluginRegistry {
	private readonly loaded: LoadedPlugin[];
	/** Map from tool name → plugin ID (for fast dispatch). */
	private readonly toolOwner: Map<string, string>;
	/** Deduplicated tool definitions contributed by loaded plugins. */
	private readonly tools: Tool[];

	constructor(loaded: LoadedPlugin[]) {
		this.loaded = loaded;
		this.toolOwner = new Map();
		this.tools = [];
		for (const p of loaded) {
			for (const tool of p.entry.tools) {
				if (this.toolOwner.has(tool.name)) {
					const existingPluginId = this.toolOwner.get(tool.name);
					console.warn(
						`Duplicate plugin MCP tool name "${tool.name}" declared by plugin "${p.pluginId}"; already registered by plugin "${existingPluginId}". Skipping duplicate.`,
					);
					continue;
				}

				this.toolOwner.set(tool.name, p.pluginId);
				this.tools.push(tool);
			}
		}
	}

	/** All unique tool definitions contributed by loaded plugins. */
	getAllTools(): Tool[] {
		return this.tools;
	}

	/**
	 * Dispatch a tool call to the owning plugin.
	 * Returns `null` if no loaded plugin owns the given tool name.
	 */
	async handleToolCall(
		toolName: string,
		args: Record<string, unknown>,
		config: PacaConfig,
	): Promise<PluginToolResult | null> {
		const pluginId = this.toolOwner.get(toolName);
		if (!pluginId) return null;

		const plugin = this.loaded.find((p) => p.pluginId === pluginId);
		if (!plugin) return null;

		const context: PluginMCPContext = {
			pluginId,
			baseURL: config.baseURL,
			apiKey: config.apiKey,
		};

		try {
			return await plugin.entry.handleToolCall(toolName, args, context);
		} catch (error) {
			const message =
				error instanceof Error ? error.message : "Unknown plugin tool error";

			return {
				content: [
					{
						type: "text",
						text: `Plugin tool "${toolName}" failed: ${message}`,
					},
				],
				isError: true,
			};
		}
	}
}

// ── Loader ────────────────────────────────────────────────────────────────────

/**
 * Fetch all enabled plugins from the Paca API and load any that declare an
 * `mcp.remoteEntryUrl` in their manifest.
 *
 * Failures for individual plugins are logged and skipped so that a broken
 * third-party plugin does not prevent the server from starting.
 */
export async function loadPlugins(config: PacaConfig): Promise<PluginRegistry> {
	let plugins: InstalledPlugin[] = [];

	try {
		plugins = await fetchInstalledPlugins(config);
	} catch (err) {
		console.error(
			"[plugin-loader] Failed to fetch plugin list — starting without plugins:",
			err,
		);
		return new PluginRegistry([]);
	}

	const mcpPlugins = plugins.filter(
		(p) => p.enabled && p.manifest?.mcp?.remoteEntryUrl,
	);

	if (mcpPlugins.length === 0) {
		return new PluginRegistry([]);
	}

	console.error(
		`[plugin-loader] Loading ${mcpPlugins.length} plugin(s) with MCP tools...`,
	);

	const loaded: LoadedPlugin[] = [];

	for (const plugin of mcpPlugins) {
		// biome-ignore lint/style/noNonNullAssertion: filtered above
		const url = plugin.manifest.mcp!.remoteEntryUrl;
		try {
			const entry = await loadPluginEntry(plugin.name, url);
			loaded.push({ pluginId: plugin.name, entry });
			console.error(
				`[plugin-loader] Loaded "${plugin.name}" (${entry.tools.length} tool(s))`,
			);
		} catch (err) {
			console.error(
				`[plugin-loader] Failed to load plugin "${plugin.name}" from ${url}:`,
				err,
			);
		}
	}

	return new PluginRegistry(loaded);
}

// ── Helpers ───────────────────────────────────────────────────────────────────

async function fetchInstalledPlugins(
	config: PacaConfig,
): Promise<InstalledPlugin[]> {
	const url = `${config.baseURL}/api/v1/plugins`;
	const response = await fetch(url, {
		headers: {
			"Content-Type": "application/json",
			"X-API-Key": config.apiKey,
		},
	});

	if (!response.ok) {
		const text = await response.text();
		throw new Error(
			`GET /api/v1/plugins failed: ${response.status} ${response.statusText} — ${text}`,
		);
	}

	const body = await response.json();

	// Handle SuccessEnvelope wrapper
	if (body && typeof body === "object" && "success" in body && body.success) {
		const data = body.data as PluginListResponse;
		return data.plugins ?? [];
	}

	// Direct array fallback
	if (Array.isArray(body)) return body;

	return [];
}

async function loadPluginEntry(
	pluginId: string,
	url: string,
): Promise<PluginMCPEntry> {
	// Dynamic import works for both file:// and https:// URLs in Node 18+.
	// For http:// URLs (common in local dev), we fetch the source first and
	// evaluate it via a data: URL import.
	const importUrl = await resolveImportUrl(url);
	const mod = await import(importUrl);

	const entry: unknown = mod.default ?? mod;

	assertPluginMCPEntry(pluginId, entry);
	return entry;
}

/**
 * Node.js `import()` supports `https://` URLs but NOT `http://` ones.
 * For local development we fetch the source over HTTP and re-expose it as a
 * `data:` URL so `import()` can evaluate it without network restrictions.
 */
async function resolveImportUrl(url: string): Promise<string> {
	if (url.startsWith("https://") || url.startsWith("file://")) {
		return url;
	}

	// http:// — fetch source and wrap in a data: URL
	const response = await fetch(url);
	if (!response.ok) {
		throw new Error(
			`Failed to fetch plugin module from ${url}: ${response.status} ${response.statusText}`,
		);
	}
	const source = await response.text();
	// Use base64 to avoid issues with special characters in the source
	const b64 = Buffer.from(source, "utf8").toString("base64");
	return `data:text/javascript;base64,${b64}`;
}

function assertPluginMCPEntry(
	pluginId: string,
	value: unknown,
): asserts value is PluginMCPEntry {
	if (!value || typeof value !== "object") {
		throw new Error(
			`Plugin "${pluginId}": default export is not an object`,
		);
	}
	const entry = value as Record<string, unknown>;
	if (!Array.isArray(entry.tools)) {
		throw new Error(
			`Plugin "${pluginId}": default export must have a "tools" array`,
		);
	}
	if (typeof entry.handleToolCall !== "function") {
		throw new Error(
			`Plugin "${pluginId}": default export must have a "handleToolCall" function`,
		);
	}
}
