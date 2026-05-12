import type {
	CreateBranchInput,
	CreateBranchResult,
	CreateCommentInput,
	LinkPRInput,
	PacaConfig,
	PullRequest,
	SuccessEnvelope,
	TaskActivity,
	TaskBranch,
	UpdateCommentInput,
} from "../types/index.js";

/**
 * Extended API client for Task Activities, Comments, and GitHub.
 */
export class PacaAPITaskExtendedClient {
	private config: PacaConfig;

	constructor(config: PacaConfig) {
		this.config = config;
	}

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

	// ==================== Task Activities ====================

	async listTaskActivities(
		projectId: string,
		taskId: string,
	): Promise<TaskActivity[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/tasks/${taskId}/activities`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.activities || response.data || [];
	}

	// ==================== Task Comments ====================

	async addTaskComment(
		projectId: string,
		taskId: string,
		input: CreateCommentInput,
	): Promise<TaskActivity> {
		return this.post(
			`/api/v1/projects/${projectId}/tasks/${taskId}/activities/comments`,
			input,
		);
	}

	async updateTaskComment(
		projectId: string,
		taskId: string,
		commentId: string,
		input: UpdateCommentInput,
	): Promise<TaskActivity> {
		return this.patch(
			`/api/v1/projects/${projectId}/tasks/${taskId}/activities/comments/${commentId}`,
			input,
		);
	}

	async deleteTaskComment(
		projectId: string,
		taskId: string,
		commentId: string,
	): Promise<void> {
		await this.delete(
			`/api/v1/projects/${projectId}/tasks/${taskId}/activities/comments/${commentId}`,
		);
	}

	// ==================== Task GitHub ====================

	async listTaskPRs(projectId: string, taskId: string): Promise<PullRequest[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/tasks/${taskId}/github/pull-requests`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return (
			response.items ||
			response.pullRequests ||
			response.prs ||
			response.data ||
			[]
		);
	}

	async linkPRToTask(
		projectId: string,
		taskId: string,
		input: LinkPRInput,
	): Promise<PullRequest> {
		return this.post(
			`/api/v1/projects/${projectId}/tasks/${taskId}/github/pull-requests`,
			input,
		);
	}

	async unlinkPRFromTask(
		projectId: string,
		taskId: string,
		prId: string,
	): Promise<void> {
		await this.delete(
			`/api/v1/projects/${projectId}/tasks/${taskId}/github/pull-requests/${prId}`,
		);
	}

	async createBranch(
		projectId: string,
		taskId: string,
		input: CreateBranchInput,
	): Promise<CreateBranchResult> {
		return this.post(
			`/api/v1/projects/${projectId}/tasks/${taskId}/github/branches`,
			input,
		);
	}

	async listTaskBranches(
		projectId: string,
		taskId: string,
	): Promise<TaskBranch[]> {
		const response = await this.get(
			`/api/v1/projects/${projectId}/tasks/${taskId}/github/branches`,
		);
		if (Array.isArray(response)) {
			return response;
		}
		return response.items || response.branches || response.data || [];
	}
}
