import type {
	AddAgentMCPServerInput,
	AddAgentSkillInput,
	AddMemberInput,
	Agent,
	AgentMCPServer,
	AgentSkill,
	CreateCustomFieldInput,
	CreateAgentInput,
	CreateRoleInput,
	CreateTaskStatusInput,
	CreateTaskTypeInput,
	CustomFieldDefinition,
	PacaConfig,
	ProjectMember,
	ProjectRole,
	SuccessEnvelope,
	TaskStatus,
	TaskType,
	UpdateAgentInput,
	UpdateAgentMCPServerInput,
	UpdateAgentSkillInput,
	UpdateCustomFieldInput,
	UpdateMemberRoleInput,
	UpdateRoleInput,
	UpdateTaskStatusInput,
	UpdateTaskTypeInput,
} from "../types/index.js";

/**
 * Extended API client methods for additional Paca endpoints.
 * This extends the base PacaAPIClient with additional functionality.
 */
export class PacaAPIExtendedClient {
	private config: PacaConfig;

	constructor(config: PacaConfig) {
		this.config = config;
	}

	/**
	 * Makes an HTTP request to the Paca API.
	 * Handles SuccessEnvelope wrapper by extracting data.data.
	 */
	private async request(
		method: string,
		path: string,
		body?: any,
	): Promise<any> {
		const url = `${this.config.baseURL}${path}`;
		const headers: Record<string, string> = {
			"Content-Type": "application/json",
			"X-API-Key": this.config.apiKey,
		};
		if (this.config.agentId) {
			headers["X-Agent-ID"] = this.config.agentId;
		}

		const options: RequestInit = {
			method,
			headers,
		};

		if (body) {
			options.body = JSON.stringify(body);
		}

		const response = await fetch(url, options);

		if (!response.ok) {
			const errorText = await response.text();
			throw new Error(
				`API request failed: ${response.status} ${response.statusText} - ${errorText}`,
			);
		}

		const jsonResponse = await response.json();

		// Handle SuccessEnvelope wrapper
		if (
			jsonResponse &&
			typeof jsonResponse === "object" &&
			"success" in jsonResponse
		) {
			const envelope = jsonResponse as SuccessEnvelope<any>;
			if (envelope.success) {
				return envelope.data;
			}
		}

		// Fallback for responses not wrapped in SuccessEnvelope
		return jsonResponse;
	}

	private async get(path: string): Promise<any> {
		return this.request("GET", path);
	}

	private async post(path: string, body: any): Promise<any> {
		return this.request("POST", path, body);
	}

	private async patch(path: string, body: any): Promise<any> {
		return this.request("PATCH", path, body);
	}

	private async delete(path: string): Promise<any> {
		return this.request("DELETE", path);
	}

	private async put(path: string, body: any): Promise<any> {
		return this.request("PUT", path, body);
	}

	// ==================== Project Members ====================

	async listProjectMembers(projectId: string): Promise<ProjectMember[]> {
		const response = await this.get(`/api/v1/projects/${projectId}/members`);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.members || response.data || [];
	}

	async addProjectMember(
		projectId: string,
		input: AddMemberInput,
	): Promise<ProjectMember> {
		return this.post(`/api/v1/projects/${projectId}/members`, input);
	}

	async getMyProjectPermissions(
		projectId: string,
	): Promise<Record<string, boolean>> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/members/me/permissions`,
		);
		return response.permissions || {};
	}

	async updateProjectMemberRole(
		projectId: string,
		userId: string,
		input: UpdateMemberRoleInput,
	): Promise<ProjectMember> {
		return this.patch(`/api/v1/projects/${projectId}/members/${userId}`, input);
	}

	async removeProjectMember(projectId: string, userId: string): Promise<void> {
		await this.delete(`/api/v1/projects/${projectId}/members/${userId}`);
	}

	// ==================== Project Roles ====================

	async listProjectRoles(projectId: string): Promise<ProjectRole[]> {
		const response = await this.get(`/api/v1/projects/${projectId}/roles`);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.roles || response.data || [];
	}

	async createProjectRole(
		projectId: string,
		input: CreateRoleInput,
	): Promise<ProjectRole> {
		return this.post(`/api/v1/projects/${projectId}/roles`, input);
	}

	async updateProjectRole(
		projectId: string,
		roleId: string,
		input: UpdateRoleInput,
	): Promise<ProjectRole> {
		return this.patch(`/api/v1/projects/${projectId}/roles/${roleId}`, input);
	}

	async deleteProjectRole(projectId: string, roleId: string): Promise<void> {
		await this.delete(`/api/v1/projects/${projectId}/roles/${roleId}`);
	}

	// ==================== AI Agents ====================

	async listAgents(projectId: string): Promise<Agent[]> {
		const response = await this.get(`/api/v1/projects/${projectId}/agents`);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.agents || response.data || [];
	}

	async getAgent(projectId: string, agentId: string): Promise<Agent> {
		return this.get(`/api/v1/projects/${projectId}/agents/${agentId}`);
	}

	async createAgent(
		projectId: string,
		input: CreateAgentInput,
	): Promise<Agent> {
		return this.post(`/api/v1/projects/${projectId}/agents`, input);
	}

	async updateAgent(
		projectId: string,
		agentId: string,
		input: UpdateAgentInput,
	): Promise<Agent> {
		return this.patch(`/api/v1/projects/${projectId}/agents/${agentId}`, input);
	}

	async deleteAgent(projectId: string, agentId: string): Promise<void> {
		await this.delete(`/api/v1/projects/${projectId}/agents/${agentId}`);
	}

	async listAgentMCPServers(
		projectId: string,
		agentId: string,
	): Promise<AgentMCPServer[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/agents/${agentId}/mcp-servers`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.mcp_servers || response.data || [];
	}

	async addAgentMCPServer(
		projectId: string,
		agentId: string,
		input: AddAgentMCPServerInput,
	): Promise<AgentMCPServer> {
		return this.post(
			`/api/v1/projects/${projectId}/agents/${agentId}/mcp-servers`,
			input,
		);
	}

	async updateAgentMCPServer(
		projectId: string,
		agentId: string,
		serverId: string,
		input: UpdateAgentMCPServerInput,
	): Promise<AgentMCPServer> {
		return this.patch(
			`/api/v1/projects/${projectId}/agents/${agentId}/mcp-servers/${serverId}`,
			input,
		);
	}

	async deleteAgentMCPServer(
		projectId: string,
		agentId: string,
		serverId: string,
	): Promise<void> {
		await this.delete(
			`/api/v1/projects/${projectId}/agents/${agentId}/mcp-servers/${serverId}`,
		);
	}

	async listAgentSkills(
		projectId: string,
		agentId: string,
	): Promise<AgentSkill[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/agents/${agentId}/skills`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.skills || response.data || [];
	}

	async addAgentSkill(
		projectId: string,
		agentId: string,
		input: AddAgentSkillInput,
	): Promise<AgentSkill> {
		return this.post(
			`/api/v1/projects/${projectId}/agents/${agentId}/skills`,
			input,
		);
	}

	async updateAgentSkill(
		projectId: string,
		agentId: string,
		skillId: string,
		input: UpdateAgentSkillInput,
	): Promise<AgentSkill> {
		return this.patch(
			`/api/v1/projects/${projectId}/agents/${agentId}/skills/${skillId}`,
			input,
		);
	}

	async deleteAgentSkill(
		projectId: string,
		agentId: string,
		skillId: string,
	): Promise<void> {
		await this.delete(
			`/api/v1/projects/${projectId}/agents/${agentId}/skills/${skillId}`,
		);
	}

	// ==================== Task Types ====================

	async listTaskTypes(projectId: string): Promise<TaskType[]> {
		const response = await this.get(`/api/v1/projects/${projectId}/task-types`);
		if (Array.isArray(response)) {
			return response;
		}
		return (
			response.items ||
			response.taskTypes ||
			response.types ||
			response.data ||
			[]
		);
	}

	async createTaskType(
		projectId: string,
		input: CreateTaskTypeInput,
	): Promise<TaskType> {
		return this.post(`/api/v1/projects/${projectId}/task-types`, input);
	}

	async updateTaskType(
		projectId: string,
		typeId: string,
		input: UpdateTaskTypeInput,
	): Promise<TaskType> {
		return this.patch(
			`/api/v1/projects/${projectId}/task-types/${typeId}`,
			input,
		);
	}

	async deleteTaskType(projectId: string, typeId: string): Promise<void> {
		await this.delete(`/api/v1/projects/${projectId}/task-types/${typeId}`);
	}

	async setDefaultTaskType(
		projectId: string,
		typeId: string,
	): Promise<TaskType> {
		return this.put(
			`/api/v1/projects/${projectId}/task-types/${typeId}/set-default`,
			{},
		);
	}

	// ==================== Task Statuses ====================

	async listTaskStatuses(projectId: string): Promise<TaskStatus[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/task-statuses`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return (
			response.items ||
			response.taskStatuses ||
			response.statuses ||
			response.data ||
			[]
		);
	}

	async createTaskStatus(
		projectId: string,
		input: CreateTaskStatusInput,
	): Promise<TaskStatus> {
		return this.post(`/api/v1/projects/${projectId}/task-statuses`, input);
	}

	async updateTaskStatus(
		projectId: string,
		statusId: string,
		input: UpdateTaskStatusInput,
	): Promise<TaskStatus> {
		return this.patch(
			`/api/v1/projects/${projectId}/task-statuses/${statusId}`,
			input,
		);
	}

	async deleteTaskStatus(projectId: string, statusId: string): Promise<void> {
		await this.delete(
			`/api/v1/projects/${projectId}/task-statuses/${statusId}`,
		);
	}

	async setDefaultTaskStatus(
		projectId: string,
		statusId: string,
	): Promise<TaskStatus> {
		return this.put(
			`/api/v1/projects/${projectId}/task-statuses/${statusId}/set-default`,
			{},
		);
	}

	// ==================== Custom Field Definitions ====================

	async listCustomFieldDefinitions(
		projectId: string,
	): Promise<CustomFieldDefinition[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/custom-fields`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.customFields || response.data || [];
	}

	async getCustomFieldDefinition(
		projectId: string,
		fieldId: string,
	): Promise<CustomFieldDefinition> {
		return this.get(`/api/v1/projects/${projectId}/custom-fields/${fieldId}`);
	}

	async createCustomFieldDefinition(
		projectId: string,
		input: CreateCustomFieldInput,
	): Promise<CustomFieldDefinition> {
		return this.post(`/api/v1/projects/${projectId}/custom-fields`, input);
	}

	async updateCustomFieldDefinition(
		projectId: string,
		fieldId: string,
		input: UpdateCustomFieldInput,
	): Promise<CustomFieldDefinition> {
		return this.patch(
			`/api/v1/projects/${projectId}/custom-fields/${fieldId}`,
			input,
		);
	}

	async deleteCustomFieldDefinition(
		projectId: string,
		fieldId: string,
	): Promise<void> {
		await this.delete(`/api/v1/projects/${projectId}/custom-fields/${fieldId}`);
	}
}
