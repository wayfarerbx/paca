import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import {
	CallToolRequestSchema,
	ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import {
	PacaAPIClient,
	PacaAPIDocClient,
	PacaAPIExtendedClient,
	PacaAPITaskExtendedClient,
	PacaAPIViewsClient,
} from "./api/index.js";
import { loadPlugins } from "./plugin-loader.js";
import { getAllTools, handleToolCall } from "./tools/index.js";
import type { PacaConfig } from "./types/index.js";

/**
 * Creates and configures the Paca MCP server.
 * Loads plugin MCP modules from the Paca API before returning.
 *
 * @param config - Paca configuration
 * @returns Configured MCP server
 */
export async function createServer(config: PacaConfig): Promise<Server> {
	// Initialize all API clients
	const apiClient = new PacaAPIClient(config);
	const extendedClient = new PacaAPIExtendedClient(config);
	const viewsClient = new PacaAPIViewsClient(config);
	const taskExtendedClient = new PacaAPITaskExtendedClient(config);
	const docClient = new PacaAPIDocClient(config);

	const clients = {
		apiClient,
		extendedClient,
		viewsClient,
		taskExtendedClient,
		docClient,
	};

	// Load plugin MCP modules from the Paca API.
	// Failures for individual plugins are logged and skipped.
	const pluginRegistry = await loadPlugins(config);

	const server = new Server(
		{
			name: "paca",
			version: "0.1.0",
		},
		{
			capabilities: {
				tools: {},
			},
		},
	);

	// Handler for listing available tools (core + plugins)
	server.setRequestHandler(ListToolsRequestSchema, async () => {
		return {
			tools: [...getAllTools(), ...pluginRegistry.getAllTools()],
		};
	});

	// Handler for executing tool calls
	server.setRequestHandler(CallToolRequestSchema, async (request) => {
		const { name, arguments: args } = request.params;

		// Try plugin registry first (plugin tool names are chosen by developers,
		// so we check plugins before falling through to core tools to make
		// routing explicit).
		const pluginResult = await pluginRegistry.handleToolCall(
			name,
			(args ?? {}) as Record<string, unknown>,
			config,
		);
		if (pluginResult !== null) {
			return pluginResult;
		}

		// Fall through to core tool handlers
		return handleToolCall(request, clients);
	});

	return server;
}
