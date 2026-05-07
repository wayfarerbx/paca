import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "./api-client";
import type { SuccessEnvelope } from "./api-error";

// ── Shapes ────────────────────────────────────────────────────────────────────

export interface PluginRoute {
	method: string;
	path: string;
}

export interface BackendManifest {
	routes?: PluginRoute[];
	eventSubscriptions?: string[];
}

export interface ExtensionPointRegistration {
	point: string;
	component: string;
	order?: number;
}

export interface FrontendManifest {
	remoteEntryUrl?: string;
	extensionPoints?: ExtensionPointRegistration[];
}

export interface PluginManifest {
	id: string;
	displayName: string;
	description?: string;
	version: string;
	backend?: BackendManifest;
	frontend?: FrontendManifest;
	permissions?: string[];
}

export interface Plugin {
	id: string;
	name: string;
	version: string;
	manifest: PluginManifest;
	enabled: boolean;
	installed_at: string;
	updated_at: string;
}

export interface PluginExtensionSetting {
	id: string;
	plugin_id: string;
	extension_point: string;
	settings: {
		hidden: boolean;
		order?: number;
	};
	updated_at: string;
}

// ── API ───────────────────────────────────────────────────────────────────────

export async function listPlugins(): Promise<Plugin[]> {
	const { data } =
		await apiClient.instance.get<SuccessEnvelope<{ plugins: Plugin[] }>>(
			"/plugins",
		);
	return data.data.plugins;
}

export async function updatePluginExtensionSetting(payload: {
	plugin_id: string;
	extension_point: string;
	settings: { hidden: boolean; order?: number };
}): Promise<PluginExtensionSetting> {
	const { data } = await apiClient.instance.patch<
		SuccessEnvelope<PluginExtensionSetting>
	>("/admin/plugin-extension-settings", payload);
	return data.data;
}

// ── Query options ─────────────────────────────────────────────────────────────

export const pluginsQueryOptions = queryOptions({
	queryKey: ["plugins"],
	queryFn: listPlugins,
	staleTime: 5 * 60 * 1000, // 5 min — plugins don't change often
});

// ── Registry helpers ──────────────────────────────────────────────────────────

export type ExtensionPointId =
	| "sidebar.general.section"
	| "sidebar.project.section"
	| "task.detail.section"
	| "project.settings.tab"
	| "view";

export interface PluginRegistration {
	pluginUUID: string; // The database UUID for API calls
	pluginId: string; // The reverse-DNS identifier (e.g., "com.paca.checklist")
	pluginName: string;
	remoteEntryUrl: string;
	component: string;
	order: number;
	hidden?: boolean;
}

/** Build a Map<ExtensionPointId, PluginRegistration[]> from the plugins list. */
export function buildRegistryMap(
	plugins: Plugin[],
): Map<ExtensionPointId, PluginRegistration[]> {
	const map = new Map<ExtensionPointId, PluginRegistration[]>();

	for (const plugin of plugins) {
		if (!plugin.enabled) continue;
		const ext = plugin.manifest.frontend?.extensionPoints;
		const remoteEntryUrl = plugin.manifest.frontend?.remoteEntryUrl;
		if (!ext || !remoteEntryUrl) continue;

		for (const reg of ext) {
			const point = reg.point as ExtensionPointId;
			const regs = map.get(point) ?? [];
			regs.push({
				pluginUUID: plugin.id, // UUID for API calls
				pluginId: plugin.manifest.id, // reverse-DNS for display/keys
				pluginName: plugin.manifest.displayName,
				remoteEntryUrl,
				component: reg.component,
				order: reg.order ?? 0,
			});
			map.set(point, regs);
		}
	}

	// Sort each point's registrations by order
	for (const [, regs] of map) {
		regs.sort((a, b) => a.order - b.order);
	}

	return map;
}
