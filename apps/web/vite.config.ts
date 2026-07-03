import path from "node:path";
import { fileURLToPath } from "node:url";
import { federation } from "@module-federation/vite";
import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isDocker = process.env.DOCKER === "true";
const apiProxyTarget =
	process.env.VITE_API_PROXY_TARGET ?? "http://127.0.0.1:8080";
const realtimeProxyTarget =
	process.env.VITE_REALTIME_PROXY_TARGET ?? "http://127.0.0.1:3001";
const hostProxy = {
	"/api": {
		target: apiProxyTarget,
		changeOrigin: true,
	},
	"/ws": {
		target: realtimeProxyTarget,
		ws: true,
		changeOrigin: true,
		rewrite: (requestPath: string) => requestPath.replace(/^\/ws/, ""),
	},
};

// https://vite.dev/config/
export default defineConfig({
	plugins: [
		TanStackRouterVite({
			routesDirectory: "./src/routes",
			generatedRouteTree: "./src/routeTree.gen.ts",
			autoCodeSplitting: true,
		}),
		react(),
		tailwindcss(),
		federation({
			name: "host",
			dts: false,
			remotes: {
				// Plugin remotes are registered dynamically at runtime via the
				// PluginRegistryContext — no static remotes are declared here.
			},
			shared: {
				react: { requiredVersion: "^19.0.0" },
				"react-dom": { requiredVersion: "^19.0.0" },
			},
		}),
	],
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	server: {
		watch: isDocker ? { usePolling: true } : undefined,
		hmr: isDocker ? { clientPort: 3000 } : undefined,
		allowedHosts: process.env.VITE_ALLOWED_HOST
			? [process.env.VITE_ALLOWED_HOST]
			: [],
		proxy: isDocker ? undefined : hostProxy,
	},
	preview: {
		host: "127.0.0.1",
		port: 3000,
		allowedHosts: process.env.VITE_ALLOWED_HOST
			? [process.env.VITE_ALLOWED_HOST]
			: [],
		proxy: isDocker ? undefined : hostProxy,
	},
	test: {
		environment: "jsdom",
		globals: true,
		setupFiles: "./src/test/setup.ts",
	},
});
