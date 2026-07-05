#!/usr/bin/env node

import { createRequire } from "node:module";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { createServer } from "./server.js";
import type { PacaConfig } from "./types/index.js";

const require = createRequire(import.meta.url);

/**
 * Main entry point for the Paca MCP server.
 * Initializes the API clients and starts the MCP server.
 */
async function main() {
	// Handle --version flag
	if (process.argv.includes("--version") || process.argv.includes("-v")) {
		const { version } = require("../package.json") as { version: string };
		console.log(version);
		process.exit(0);
	}

	// Get configuration from environment variables
	const apiKey = process.env.PACA_API_KEY;
	const baseURL = process.env.PACA_API_URL || "http://localhost:8080";
	const gatewayURL = process.env.PACA_GATEWAY_URL || undefined;
	const agentId = process.env.PACA_AGENT_ID || undefined;
	const projectId = process.env.PACA_PROJECT_ID || undefined;

	// Validate required configuration
	if (!apiKey) {
		console.error(
			"PACA_API_KEY environment variable is required. Please set it to your Paca API key.",
		);
		console.error("\nExample:");
		console.error("  export PACA_API_KEY='your-api-key-here'");
		console.error("  export PACA_API_URL='http://localhost:8080'");
		process.exit(1);
	}

	// If agent ID is provided, project ID is required
	if (agentId && !projectId) {
		console.error(
			"PACA_PROJECT_ID environment variable is required when using PACA_AGENT_ID.",
		);
		console.error("\nExample:");
		console.error("  export PACA_AGENT_ID='your-agent-id-here'");
		console.error("  export PACA_PROJECT_ID='your-project-id-here'");
		console.error("  export PACA_API_KEY='your-api-key-here'");
		console.error("  export PACA_API_URL='http://localhost:8080'");
		process.exit(1);
	}

	// Create configuration object
	const config: PacaConfig = {
		apiKey,
		baseURL,
		gatewayURL,
		agentId,
		projectId,
	};

	// Create and configure MCP server (loads plugin modules asynchronously)
	const server = await createServer(config);

	// Connect to stdio transport
	const transport = new StdioServerTransport();
	await server.connect(transport);
}

// Handle errors and exit gracefully
main().catch((error) => {
	console.error("Server error:", error);
	process.exit(1);
});
