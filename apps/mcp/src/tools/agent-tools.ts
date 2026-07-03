import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPIExtendedClient } from "../api/index.js";
import type {
	Agent,
	AgentMCPServer,
	AgentSkill,
	CreateAgentInput,
	UpdateAgentInput,
} from "../types/index.js";
import { formatList } from "../utils/index.js";

const ListAgentsSchema = z.object({
	projectId: z.string(),
});

const GetAgentSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
});

const CreateAgentSchema = z.object({
	projectId: z.string(),
	name: z.string(),
	handle: z.string(),
	projectRoleId: z.string(),
	llmProvider: z.string().optional(),
	llmModel: z.string().optional(),
	llmApiKey: z.string().optional(),
	llmBaseURL: z.string().optional(),
	systemPrompt: z.string().optional(),
	taskTriggerPrompt: z.string().optional(),
	docCommentTriggerPrompt: z.string().optional(),
	chatTriggerPrompt: z.string().optional(),
	descriptionWriteTriggerPrompt: z.string().optional(),
	canCloneRepos: z.boolean().optional(),
	canCreatePRs: z.boolean().optional(),
	maxIterations: z.number().int().positive().optional(),
	timeoutMinutes: z.number().int().positive().optional(),
	gitCommitterName: z.string().optional(),
	gitCommitterEmail: z.string().optional(),
});

const UpdateAgentSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	name: z.string().optional(),
	handle: z.string().optional(),
	projectRoleId: z.string().optional(),
	llmProvider: z.string().optional(),
	llmModel: z.string().optional(),
	llmApiKey: z.string().optional(),
	llmBaseURL: z.string().optional(),
	systemPrompt: z.string().optional(),
	taskTriggerPrompt: z.string().optional(),
	docCommentTriggerPrompt: z.string().optional(),
	chatTriggerPrompt: z.string().optional(),
	descriptionWriteTriggerPrompt: z.string().optional(),
	canCloneRepos: z.boolean().optional(),
	canCreatePRs: z.boolean().optional(),
	maxIterations: z.number().int().positive().optional(),
	timeoutMinutes: z.number().int().positive().optional(),
	gitCommitterName: z.string().optional(),
	gitCommitterEmail: z.string().optional(),
});

const DeleteAgentSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
});

const AddMCPServerSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	serverName: z.string(),
	transport: z.enum(["stdio", "sse", "http"]),
	command: z.string().optional(),
	args: z.array(z.string()).optional(),
	url: z.string().optional(),
	env: z.record(z.string(), z.string()).optional(),
});

const UpdateMCPServerSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	serverId: z.string(),
	command: z.string().optional(),
	args: z.array(z.string()).optional(),
	url: z.string().optional(),
	env: z.record(z.string(), z.string()).optional(),
	isEnabled: z.boolean().optional(),
});

const DeleteMCPServerSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	serverId: z.string(),
});

const AddSkillSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	skillName: z.string(),
	skillSource: z.enum(["inline", "marketplace", "github_url"]).default("inline"),
	skillContent: z.string().optional(),
	sourceURL: z.string().optional(),
	triggers: z.array(z.string()).optional(),
});

const UpdateSkillSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	skillId: z.string(),
	skillContent: z.string().optional(),
	triggers: z.array(z.string()).optional(),
	isEnabled: z.boolean().optional(),
});

const DeleteSkillSchema = z.object({
	projectId: z.string(),
	agentId: z.string(),
	skillId: z.string(),
});

const projectIdProperty = {
	type: "string",
	description:
		"The technical UUID of the project. In single-project agent mode this must match the configured project.",
};

const agentIdProperty = {
	type: "string",
	description: "The technical UUID of the agent.",
};

function agentConfigProperties() {
	return {
		projectRoleId: {
			type: "string",
			description:
				"Project role UUID to assign to the agent. Use list_project_roles to find role IDs.",
		},
		llmProvider: {
			type: "string",
			description:
				"LLM provider. Use 'chatgpt' for the local ChatGPT subscription integration.",
		},
		llmModel: {
			type: "string",
			description:
				"Model name. Defaults to gpt-5.5 for ChatGPT subscription agents.",
		},
		llmApiKey: {
			type: "string",
			description:
				"API key for API-key providers. Leave empty or omit for ChatGPT subscription agents.",
		},
		llmBaseURL: {
			type: "string",
			description:
				"Base URL for API-key providers. Leave empty or omit for ChatGPT subscription agents.",
		},
		systemPrompt: { type: "string" },
		taskTriggerPrompt: { type: "string" },
		docCommentTriggerPrompt: { type: "string" },
		chatTriggerPrompt: { type: "string" },
		descriptionWriteTriggerPrompt: { type: "string" },
		canCloneRepos: { type: "boolean" },
		canCreatePRs: { type: "boolean" },
		maxIterations: { type: "number" },
		timeoutMinutes: { type: "number" },
		gitCommitterName: { type: "string" },
		gitCommitterEmail: { type: "string" },
	};
}

export function getAgentTools(): Tool[] {
	return [
		{
			name: "list_agents",
			description: "List AI agents in a project",
			inputSchema: {
				type: "object",
				properties: { projectId: projectIdProperty },
				required: ["projectId"],
			},
		},
		{
			name: "get_agent",
			description: "Get full configuration for an AI agent",
			inputSchema: {
				type: "object",
				properties: { projectId: projectIdProperty, agentId: agentIdProperty },
				required: ["projectId", "agentId"],
			},
		},
		{
			name: "create_agent",
			description:
				"Create an AI agent. Defaults to a ChatGPT subscription agent when llmProvider/llmBaseURL/llmApiKey are omitted.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					name: { type: "string" },
					handle: { type: "string" },
					...agentConfigProperties(),
				},
				required: ["projectId", "name", "handle", "projectRoleId"],
			},
		},
		{
			name: "update_agent",
			description:
				"Update an AI agent configuration, including its project role via projectRoleId",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					name: { type: "string" },
					handle: { type: "string" },
					...agentConfigProperties(),
				},
				required: ["projectId", "agentId"],
			},
		},
		{
			name: "delete_agent",
			description: "Delete an AI agent",
			inputSchema: {
				type: "object",
				properties: { projectId: projectIdProperty, agentId: agentIdProperty },
				required: ["projectId", "agentId"],
			},
		},
		{
			name: "list_agent_mcp_servers",
			description: "List MCP servers configured on an AI agent",
			inputSchema: {
				type: "object",
				properties: { projectId: projectIdProperty, agentId: agentIdProperty },
				required: ["projectId", "agentId"],
			},
		},
		{
			name: "add_agent_mcp_server",
			description: "Add an MCP server to an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					serverName: { type: "string" },
					transport: { type: "string", enum: ["stdio", "sse", "http"] },
					command: { type: "string" },
					args: { type: "array", items: { type: "string" } },
					url: { type: "string" },
					env: { type: "object", additionalProperties: { type: "string" } },
				},
				required: ["projectId", "agentId", "serverName", "transport"],
			},
		},
		{
			name: "update_agent_mcp_server",
			description: "Update an MCP server on an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					serverId: { type: "string" },
					command: { type: "string" },
					args: { type: "array", items: { type: "string" } },
					url: { type: "string" },
					env: { type: "object", additionalProperties: { type: "string" } },
					isEnabled: { type: "boolean" },
				},
				required: ["projectId", "agentId", "serverId"],
			},
		},
		{
			name: "delete_agent_mcp_server",
			description: "Delete an MCP server from an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					serverId: { type: "string" },
				},
				required: ["projectId", "agentId", "serverId"],
			},
		},
		{
			name: "list_agent_skills",
			description: "List skills configured on an AI agent",
			inputSchema: {
				type: "object",
				properties: { projectId: projectIdProperty, agentId: agentIdProperty },
				required: ["projectId", "agentId"],
			},
		},
		{
			name: "add_agent_skill",
			description: "Add a skill to an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					skillName: { type: "string" },
					skillSource: {
						type: "string",
						enum: ["inline", "marketplace", "github_url"],
					},
					skillContent: { type: "string" },
					sourceURL: { type: "string" },
					triggers: { type: "array", items: { type: "string" } },
				},
				required: ["projectId", "agentId", "skillName"],
			},
		},
		{
			name: "update_agent_skill",
			description: "Update a skill on an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					skillId: { type: "string" },
					skillContent: { type: "string" },
					triggers: { type: "array", items: { type: "string" } },
					isEnabled: { type: "boolean" },
				},
				required: ["projectId", "agentId", "skillId"],
			},
		},
		{
			name: "delete_agent_skill",
			description: "Delete a skill from an AI agent",
			inputSchema: {
				type: "object",
				properties: {
					projectId: projectIdProperty,
					agentId: agentIdProperty,
					skillId: { type: "string" },
				},
				required: ["projectId", "agentId", "skillId"],
			},
		},
	];
}

function toCreateAgentInput(args: z.infer<typeof CreateAgentSchema>): CreateAgentInput {
	return {
		name: args.name,
		handle: args.handle,
		project_role_id: args.projectRoleId,
		llm_provider: args.llmProvider ?? "chatgpt",
		llm_model: args.llmModel ?? "gpt-5.5",
		llm_api_key: args.llmApiKey ?? "",
		llm_base_url: args.llmBaseURL ?? "",
		system_prompt: args.systemPrompt ?? "",
		task_trigger_prompt: args.taskTriggerPrompt ?? "",
		doc_comment_trigger_prompt: args.docCommentTriggerPrompt ?? "",
		chat_trigger_prompt: args.chatTriggerPrompt ?? "",
		description_write_trigger_prompt:
			args.descriptionWriteTriggerPrompt ?? "",
		can_clone_repos: args.canCloneRepos ?? true,
		can_create_prs: args.canCreatePRs ?? true,
		max_iterations: args.maxIterations ?? 500,
		timeout_minutes: args.timeoutMinutes ?? 30,
		git_committer_name: args.gitCommitterName ?? "",
		git_committer_email: args.gitCommitterEmail ?? "",
	};
}

function toUpdateAgentInput(args: z.infer<typeof UpdateAgentSchema>): UpdateAgentInput {
	return {
		name: args.name,
		handle: args.handle,
		project_role_id: args.projectRoleId,
		llm_provider: args.llmProvider,
		llm_model: args.llmModel,
		llm_api_key: args.llmApiKey,
		llm_base_url: args.llmBaseURL,
		system_prompt: args.systemPrompt,
		task_trigger_prompt: args.taskTriggerPrompt,
		doc_comment_trigger_prompt: args.docCommentTriggerPrompt,
		chat_trigger_prompt: args.chatTriggerPrompt,
		description_write_trigger_prompt: args.descriptionWriteTriggerPrompt,
		can_clone_repos: args.canCloneRepos,
		can_create_prs: args.canCreatePRs,
		max_iterations: args.maxIterations,
		timeout_minutes: args.timeoutMinutes,
		git_committer_name: args.gitCommitterName,
		git_committer_email: args.gitCommitterEmail,
	};
}

function formatAgent(agent: Agent): string {
	return `Agent: ${agent.name} (@${agent.handle})
ID: ${agent.id}
Project: ${agent.project_id}
Member ID: ${agent.member_id || "None"}
Role ID: ${agent.project_role_id || "None"}
Role: ${agent.project_role_name || "None"}
Provider: ${agent.llm_provider}
Model: ${agent.llm_model}
Base URL: ${agent.llm_base_url || "subscription/default"}
Can clone repos: ${agent.can_clone_repos}
Can create PRs: ${agent.can_create_prs}
Max iterations: ${agent.max_iterations}
Timeout minutes: ${agent.timeout_minutes}
Git committer: ${agent.git_committer_name} <${agent.git_committer_email}>
Created: ${agent.created_at}
Updated: ${agent.updated_at}

System prompt:
${agent.system_prompt || "None"}`;
}

function formatMCPServer(server: AgentMCPServer): string {
	return `MCP Server: ${server.server_name}
ID: ${server.id}
Agent ID: ${server.agent_id}
Transport: ${server.transport}
Command: ${server.command || "None"}
Args: ${server.args.length ? server.args.join(" ") : "None"}
URL: ${server.url || "None"}
Enabled: ${server.is_enabled}
Created: ${server.created_at}`;
}

function formatSkill(skill: AgentSkill): string {
	return `Skill: ${skill.skill_name}
ID: ${skill.id}
Agent ID: ${skill.agent_id}
Source: ${skill.skill_source}
Source URL: ${skill.source_url || "None"}
Triggers: ${skill.triggers.length ? skill.triggers.join(", ") : "None"}
Enabled: ${skill.is_enabled}
Created: ${skill.created_at}

Content:
${skill.skill_content || "None"}`;
}

export async function handleAgentTool(
	toolName: string,
	args: any,
	client: PacaAPIExtendedClient,
): Promise<any> {
	switch (toolName) {
		case "list_agents": {
			const { projectId } = ListAgentsSchema.parse(args);
			const agents = await client.listAgents(projectId);
			return {
				content: [{ type: "text", text: `Agents:\n\n${formatList(agents, formatAgent)}` }],
			};
		}

		case "get_agent": {
			const { projectId, agentId } = GetAgentSchema.parse(args);
			const agent = await client.getAgent(projectId, agentId);
			return { content: [{ type: "text", text: formatAgent(agent) }] };
		}

		case "create_agent": {
			const parsed = CreateAgentSchema.parse(args);
			const agent = await client.createAgent(
				parsed.projectId,
				toCreateAgentInput(parsed),
			);
			return {
				content: [
					{
						type: "text",
						text: `Agent created successfully:\n\n${formatAgent(agent)}`,
					},
				],
			};
		}

		case "update_agent": {
			const parsed = UpdateAgentSchema.parse(args);
			const agent = await client.updateAgent(
				parsed.projectId,
				parsed.agentId,
				toUpdateAgentInput(parsed),
			);
			return {
				content: [
					{
						type: "text",
						text: `Agent updated successfully:\n\n${formatAgent(agent)}`,
					},
				],
			};
		}

		case "delete_agent": {
			const { projectId, agentId } = DeleteAgentSchema.parse(args);
			await client.deleteAgent(projectId, agentId);
			return {
				content: [
					{ type: "text", text: `Agent ${agentId} deleted successfully` },
				],
			};
		}

		case "list_agent_mcp_servers": {
			const { projectId, agentId } = GetAgentSchema.parse(args);
			const servers = await client.listAgentMCPServers(projectId, agentId);
			return {
				content: [
					{
						type: "text",
						text: `Agent MCP Servers:\n\n${formatList(servers, formatMCPServer)}`,
					},
				],
			};
		}

		case "add_agent_mcp_server": {
			const parsed = AddMCPServerSchema.parse(args);
			const server = await client.addAgentMCPServer(parsed.projectId, parsed.agentId, {
				server_name: parsed.serverName,
				transport: parsed.transport,
				command: parsed.command,
				args: parsed.args ?? [],
				url: parsed.url,
				env: parsed.env ?? {},
			});
			return {
				content: [
					{
						type: "text",
						text: `MCP server added successfully:\n\n${formatMCPServer(server)}`,
					},
				],
			};
		}

		case "update_agent_mcp_server": {
			const parsed = UpdateMCPServerSchema.parse(args);
			const server = await client.updateAgentMCPServer(
				parsed.projectId,
				parsed.agentId,
				parsed.serverId,
				{
					command: parsed.command,
					args: parsed.args,
					url: parsed.url,
					env: parsed.env,
					is_enabled: parsed.isEnabled,
				},
			);
			return {
				content: [
					{
						type: "text",
						text: `MCP server updated successfully:\n\n${formatMCPServer(server)}`,
					},
				],
			};
		}

		case "delete_agent_mcp_server": {
			const { projectId, agentId, serverId } = DeleteMCPServerSchema.parse(args);
			await client.deleteAgentMCPServer(projectId, agentId, serverId);
			return {
				content: [
					{
						type: "text",
						text: `MCP server ${serverId} deleted successfully`,
					},
				],
			};
		}

		case "list_agent_skills": {
			const { projectId, agentId } = GetAgentSchema.parse(args);
			const skills = await client.listAgentSkills(projectId, agentId);
			return {
				content: [
					{
						type: "text",
						text: `Agent Skills:\n\n${formatList(skills, formatSkill)}`,
					},
				],
			};
		}

		case "add_agent_skill": {
			const parsed = AddSkillSchema.parse(args);
			const skill = await client.addAgentSkill(parsed.projectId, parsed.agentId, {
				skill_name: parsed.skillName,
				skill_source: parsed.skillSource,
				skill_content: parsed.skillContent ?? "",
				source_url: parsed.sourceURL,
				triggers: parsed.triggers ?? [],
			});
			return {
				content: [
					{
						type: "text",
						text: `Skill added successfully:\n\n${formatSkill(skill)}`,
					},
				],
			};
		}

		case "update_agent_skill": {
			const parsed = UpdateSkillSchema.parse(args);
			const skill = await client.updateAgentSkill(
				parsed.projectId,
				parsed.agentId,
				parsed.skillId,
				{
					skill_content: parsed.skillContent,
					triggers: parsed.triggers,
					is_enabled: parsed.isEnabled,
				},
			);
			return {
				content: [
					{
						type: "text",
						text: `Skill updated successfully:\n\n${formatSkill(skill)}`,
					},
				],
			};
		}

		case "delete_agent_skill": {
			const { projectId, agentId, skillId } = DeleteSkillSchema.parse(args);
			await client.deleteAgentSkill(projectId, agentId, skillId);
			return {
				content: [
					{ type: "text", text: `Skill ${skillId} deleted successfully` },
				],
			};
		}

		default:
			throw new Error(`Unknown agent tool: ${toolName}`);
	}
}
