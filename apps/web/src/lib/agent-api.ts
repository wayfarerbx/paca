import { queryOptions } from "@tanstack/react-query";
import { apiClient } from "./api-client";
import type { SuccessEnvelope } from "./api-error";

// ── Shapes ────────────────────────────────────────────────────────────────────

export interface AgentPreset {
	id: string;
	label: string;
	description: string;
	defaultLLMProvider: string;
	defaultLLMModel: string;
	defaultSystemPrompt: string;
}

export const AGENT_PRESETS: AgentPreset[] = [
	{
		id: "software-engineer",
		label: "Software Engineer",
		description: "An AI agent focused on implementing features and fixing bugs.",
		defaultLLMProvider: "anthropic",
		defaultLLMModel: "claude-sonnet-4-5-20250929",
		defaultSystemPrompt:
			"You are an expert software engineer. When assigned a task, follow this workflow:\n\n1. **Read the task**: Use the Paca MCP tool to fetch the full task details including description, acceptance criteria, and current status.\n2. **Assess readiness**: Based on the task details and status, determine whether you have enough information to begin implementation.\n   - If anything is unclear or you need input from the human (e.g. ambiguous requirements, missing context, architectural decisions), use the Paca MCP tool to add a comment on the task explaining what needs clarification. Then stop and wait for a response.\n   - If the task is clear and actionable, proceed to step 3.\n3. **Start the task**: Use the Paca MCP tool to update the task status to the appropriate in-progress status, then implement the feature or fix according to the requirements. Write clean, maintainable code and follow best practices.\n4. **Finish the task**: Once implementation is complete, use the Paca MCP tool to update the task status to the appropriate done/completed status, then add a comment summarising what was done, key decisions made, and any follow-up items the team should be aware of.",
	},
	{
		id: "code-reviewer",
		label: "Code Reviewer",
		description: "An AI agent that reviews code for quality, bugs, and best practices.",
		defaultLLMProvider: "anthropic",
		defaultLLMModel: "claude-sonnet-4-5-20250929",
		defaultSystemPrompt:
			"You are a meticulous code reviewer. When assigned a task, follow this workflow:\n\n1. **Read the task**: Use the Paca MCP tool to fetch the full task details including the code or pull request to review and the current status.\n2. **Assess readiness**: Based on the task details and status, determine whether you have enough context to begin the review.\n   - If the scope is unclear, the target branch/PR is not specified, or you need additional information from the human, use the Paca MCP tool to add a comment on the task asking for clarification. Then stop and wait for a response.\n   - If the task is clear, proceed to step 3.\n3. **Start the review**: Use the Paca MCP tool to update the task status to the appropriate in-progress status, then review the code for correctness, security vulnerabilities, performance issues, and adherence to best practices. Provide constructive and actionable feedback.\n4. **Finish the review**: Once the review is complete, use the Paca MCP tool to update the task status to the appropriate done/completed status, then add a comment summarising the findings, severity of issues found, and recommended next steps.",
	},
	{
		id: "qa-engineer",
		label: "QA Engineer",
		description: "An AI agent specialized in writing and running tests.",
		defaultLLMProvider: "anthropic",
		defaultLLMModel: "claude-sonnet-4-5-20250929",
		defaultSystemPrompt:
			"You are a quality assurance engineer. When assigned a task, follow this workflow:\n\n1. **Read the task**: Use the Paca MCP tool to fetch the full task details including the feature or component to test and the current status.\n2. **Assess readiness**: Based on the task details and status, determine whether you have enough information to begin writing or executing tests.\n   - If requirements are ambiguous, the acceptance criteria are missing, or you need clarification from the human, use the Paca MCP tool to add a comment on the task describing what is needed. Then stop and wait for a response.\n   - If the task is clear and actionable, proceed to step 3.\n3. **Start the task**: Use the Paca MCP tool to update the task status to the appropriate in-progress status, then write comprehensive test suites, identify edge cases, create test plans, and ensure software reliability through thorough testing strategies.\n4. **Finish the task**: Once testing is complete, use the Paca MCP tool to update the task status to the appropriate done/completed status, then add a comment summarising the tests written or executed, coverage achieved, any bugs discovered, and recommendations for the team.",
	},
	{
		id: "custom",
		label: "Custom",
		description: "Start from scratch with your own configuration.",
		defaultLLMProvider: "",
		defaultLLMModel: "",
		defaultSystemPrompt: "",
	},
];
export interface AgentMCPServer {
	id: string;
	agent_id: string;
	server_name: string;
	transport: "stdio" | "sse" | "http";
	command?: string | null;
	args?: string[];
	url?: string | null;
	env?: Record<string, string>;
	is_enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface AgentSkill {
	id: string;
	agent_id: string;
	skill_name: string;
	skill_source: "inline" | "marketplace" | "github_url";
	skill_content?: string | null;
	source_url?: string | null;
	triggers: string[];
	is_enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface Agent {
	id: string;
	project_id: string;
	name: string;
	handle: string;
	avatar_url?: string | null;
	llm_provider: string;
	llm_model: string;
	llm_base_url?: string | null;
	system_prompt: string;
	can_clone_repos: boolean;
	can_create_prs: boolean;
	max_iterations: number;
	timeout_minutes: number;
	member_id?: string | null;
	mcp_servers?: AgentMCPServer[];
	skills?: AgentSkill[];
	created_at: string;
	updated_at: string;
}

export type ConversationStatus =
	| "queued"
	| "running"
	| "paused"
	| "finished"
	| "failed"
	| "stopped";

export interface AgentConversation {
	id: string;
	agent_id: string;
	project_id: string;
	trigger_type: "task_assigned" | "comment_mention" | "chat_message";
	task_id?: string | null;
	comment_id?: string | null;
	chat_session_id?: string | null;
	triggered_by_member_id: string;
	status: ConversationStatus;
	iteration_count: number;
	error_message?: string | null;
	branch_name?: string | null;
	pr_url?: string | null;
	started_at?: string | null;
	finished_at?: string | null;
	created_at: string;
	updated_at: string;
}

export interface AgentConversationEvent {
	id: string;
	conversation_id: string;
	event_index: number;
	event_type: string;
	event_source: "agent" | "user" | "system";
	payload: Record<string, unknown>;
	created_at: string;
}

export interface AgentChatSession {
	id: string;
	agent_id: string;
	project_id: string;
	member_id: string;
	title?: string | null;
	last_message_at?: string | null;
	created_at: string;
	updated_at: string;
}

// ── Agents ────────────────────────────────────────────────────────────────────

export async function listAgents(projectId: string): Promise<Agent[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: Agent[] }>
	>(`/projects/${projectId}/agents`);
	return data.data.items;
}

export async function getAgent(
	projectId: string,
	agentId: string,
): Promise<Agent> {
	const { data } = await apiClient.instance.get<SuccessEnvelope<Agent>>(
		`/projects/${projectId}/agents/${agentId}`,
	);
	return data.data;
}

export async function createAgent(
	projectId: string,
	payload: {
		name: string;
		handle: string;
		llm_provider: string;
		llm_model: string;
		llm_api_key: string;
		llm_base_url?: string | null;
		system_prompt?: string;
		can_clone_repos?: boolean;
		can_create_prs?: boolean;
		max_iterations?: number;
		timeout_minutes?: number;
		project_role_id: string;
	},
): Promise<Agent> {
	const { data } = await apiClient.instance.post<SuccessEnvelope<Agent>>(
		`/projects/${projectId}/agents`,
		payload,
	);
	return data.data;
}

export async function updateAgent(
	projectId: string,
	agentId: string,
	payload: {
		name?: string;
		handle?: string;
		llm_provider?: string;
		llm_model?: string;
		llm_api_key?: string;
		llm_base_url?: string | null;
		system_prompt?: string;
		can_clone_repos?: boolean;
		can_create_prs?: boolean;
		max_iterations?: number;
		timeout_minutes?: number;
	},
): Promise<Agent> {
	const { data } = await apiClient.instance.patch<SuccessEnvelope<Agent>>(
		`/projects/${projectId}/agents/${agentId}`,
		payload,
	);
	return data.data;
}

export async function deleteAgent(
	projectId: string,
	agentId: string,
): Promise<void> {
	await apiClient.instance.delete(`/projects/${projectId}/agents/${agentId}`);
}

// ── MCP Servers ───────────────────────────────────────────────────────────────

export async function listMCPServers(
	projectId: string,
	agentId: string,
): Promise<AgentMCPServer[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: AgentMCPServer[] }>
	>(`/projects/${projectId}/agents/${agentId}/mcp-servers`);
	return data.data.items;
}

export async function addMCPServer(
	projectId: string,
	agentId: string,
	payload: {
		server_name: string;
		transport: "stdio" | "sse" | "http";
		command?: string | null;
		args?: string[];
		url?: string | null;
		env?: Record<string, string>;
	},
): Promise<AgentMCPServer> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<AgentMCPServer>
	>(`/projects/${projectId}/agents/${agentId}/mcp-servers`, payload);
	return data.data;
}

export async function updateMCPServer(
	projectId: string,
	agentId: string,
	serverId: string,
	payload: { is_enabled?: boolean; command?: string; args?: string[]; url?: string | null },
): Promise<AgentMCPServer> {
	const { data } = await apiClient.instance.patch<
		SuccessEnvelope<AgentMCPServer>
	>(
		`/projects/${projectId}/agents/${agentId}/mcp-servers/${serverId}`,
		payload,
	);
	return data.data;
}

export async function deleteMCPServer(
	projectId: string,
	agentId: string,
	serverId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/agents/${agentId}/mcp-servers/${serverId}`,
	);
}

// ── Skills ────────────────────────────────────────────────────────────────────

export async function listSkills(
	projectId: string,
	agentId: string,
): Promise<AgentSkill[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: AgentSkill[] }>
	>(`/projects/${projectId}/agents/${agentId}/skills`);
	return data.data.items;
}

export async function addSkill(
	projectId: string,
	agentId: string,
	payload: {
		skill_name: string;
		skill_source: "inline" | "marketplace" | "github_url";
		skill_content?: string;
		source_url?: string | null;
		triggers?: string[];
	},
): Promise<AgentSkill> {
	const { data } = await apiClient.instance.post<SuccessEnvelope<AgentSkill>>(
		`/projects/${projectId}/agents/${agentId}/skills`,
		payload,
	);
	return data.data;
}

export async function updateSkill(
	projectId: string,
	agentId: string,
	skillId: string,
	payload: { is_enabled?: boolean; triggers?: string[]; skill_content?: string },
): Promise<AgentSkill> {
	const { data } = await apiClient.instance.patch<SuccessEnvelope<AgentSkill>>(
		`/projects/${projectId}/agents/${agentId}/skills/${skillId}`,
		payload,
	);
	return data.data;
}

export async function deleteSkill(
	projectId: string,
	agentId: string,
	skillId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/agents/${agentId}/skills/${skillId}`,
	);
}

// ── Conversations ─────────────────────────────────────────────────────────────

export async function listConversations(
	projectId: string,
	agentId?: string,
): Promise<AgentConversation[]> {
	const params = agentId ? { agent_id: agentId } : undefined;
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: AgentConversation[] }>
	>(`/projects/${projectId}/conversations`, { params });
	return data.data.items;
}

export async function getConversation(
	projectId: string,
	conversationId: string,
): Promise<AgentConversation> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<AgentConversation>
	>(`/projects/${projectId}/conversations/${conversationId}`);
	return data.data;
}

export async function listConversationEvents(
	projectId: string,
	conversationId: string,
): Promise<AgentConversationEvent[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: AgentConversationEvent[] }>
	>(`/projects/${projectId}/conversations/${conversationId}/events`);
	return data.data.items;
}

// ── Chat Sessions ─────────────────────────────────────────────────────────────

export async function listChatSessions(
	projectId: string,
	agentId: string,
): Promise<AgentChatSession[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<{ items: AgentChatSession[] }>
	>(`/projects/${projectId}/agents/${agentId}/chat-sessions`);
	return data.data.items;
}

export async function startChatSession(
	projectId: string,
	agentId: string,
	payload: { message: string; title?: string },
): Promise<AgentChatSession> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<AgentChatSession>
	>(`/projects/${projectId}/agents/${agentId}/chat-sessions`, payload);
	return data.data;
}

export async function sendChatMessage(
	projectId: string,
	agentId: string,
	sessionId: string,
	payload: { message: string },
): Promise<AgentConversation> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<AgentConversation>
	>(
		`/projects/${projectId}/agents/${agentId}/chat-sessions/${sessionId}/messages`,
		payload,
	);
	return data.data;
}

// ── Query Options ─────────────────────────────────────────────────────────────

export const agentsQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "agents"],
		queryFn: () => listAgents(projectId),
	});

export const agentQueryOptions = (projectId: string, agentId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "agents", agentId],
		queryFn: () => getAgent(projectId, agentId),
	});

export const agentMCPServersQueryOptions = (
	projectId: string,
	agentId: string,
) =>
	queryOptions({
		queryKey: ["projects", projectId, "agents", agentId, "mcp-servers"],
		queryFn: () => listMCPServers(projectId, agentId),
	});

export const agentSkillsQueryOptions = (projectId: string, agentId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "agents", agentId, "skills"],
		queryFn: () => listSkills(projectId, agentId),
	});

export const conversationsQueryOptions = (
	projectId: string,
	agentId?: string,
) =>
	queryOptions({
		queryKey: ["projects", projectId, "conversations", { agentId }],
		queryFn: () => listConversations(projectId, agentId),
		refetchInterval: 10_000,
	});

export const conversationEventsQueryOptions = (
	projectId: string,
	conversationId: string,
) =>
	queryOptions({
		queryKey: [
			"projects",
			projectId,
			"conversations",
			conversationId,
			"events",
		],
		queryFn: () => listConversationEvents(projectId, conversationId),
		refetchInterval: 5_000,
	});

export const chatSessionsQueryOptions = (projectId: string, agentId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "agents", agentId, "chat-sessions"],
		queryFn: () => listChatSessions(projectId, agentId),
	});

// ── LLM Models ────────────────────────────────────────────────────────────────

export interface LLMModelsResponse {
	[provider: string]: string[];
}

export async function listLLMModels(): Promise<LLMModelsResponse> {
	const { data } = await apiClient.instance.get<LLMModelsResponse>(
		"/agents/llm-models",
	);
	return data;
}

export const llmModelsQueryOptions = queryOptions({
	queryKey: ["agents", "llm-models"],
	queryFn: listLLMModels,
	staleTime: 10 * 60 * 1000, // 10 min — provider list rarely changes
});

// ── Helpers ───────────────────────────────────────────────────────────────────

export const CONVERSATION_STATUS_LABELS: Record<ConversationStatus, string> = {
	queued: "Queued",
	running: "Running",
	paused: "Paused",
	finished: "Finished",
	failed: "Failed",
	stopped: "Stopped",
};

export const CONVERSATION_STATUS_COLORS: Record<ConversationStatus, string> = {
	queued: "text-muted-foreground",
	running: "text-blue-500",
	paused: "text-amber-500",
	finished: "text-emerald-500",
	failed: "text-destructive",
	stopped: "text-muted-foreground",
};
