import path from "node:path";
import { fileURLToPath } from "node:url";
import federation from "@originjs/vite-plugin-federation";
import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isDocker = process.env.DOCKER === "true";

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
		watch: isDocker ? { usePolling: true, interval: 2000, binaryInterval: 4000 } : undefined,
		hmr: isDocker ? { clientPort: 3000 } : undefined,
		allowedHosts: [
			"97d1-2001-ee0-4b7b-1210-c5f-19d3-601c-7dc0.ngrok-free.app",
		],
	},
	test: {
		environment: "jsdom",
		globals: true,
		setupFiles: "./src/test/setup.ts",
	},
});
