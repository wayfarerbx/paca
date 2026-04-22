import { queryOptions } from "@tanstack/react-query";

import { apiClient } from "./api-client";
import type { SuccessEnvelope } from "./api-error";

// ── Shapes ────────────────────────────────────────────────────────────────────

/** Stored GitHub integration record (no token is returned). */
export interface GitHubIntegration {
	id: string;
	project_id: string;
	created_at: string;
	updated_at: string;
}

/** A repository accessible via the project's PAT. */
export interface AccessibleRepo {
	full_name: string;
	owner: string;
	repo_name: string;
	default_branch: string;
	private: boolean;
	description: string;
}

/** The repository linked to a project. */
export interface LinkedRepository {
	id: string;
	project_id: string;
	integration_id: string;
	owner: string;
	repo_name: string;
	full_name: string;
	default_branch: string;
	webhook_id: number;
	created_at: string;
	updated_at: string;
}

/** A pull request cached from GitHub and linked to a project. */
export interface PullRequest {
	id: string;
	project_id: string;
	repo_id: string;
	pr_number: number;
	github_pr_id: number;
	title: string;
	state: "open" | "closed" | "merged";
	html_url: string;
	head_branch: string;
	base_branch: string;
	author: string;
	merged_at: string | null;
	created_at: string;
	updated_at: string;
}

export interface CreateBranchResult {
	branch_name: string;
}

// ── Integration endpoints ─────────────────────────────────────────────────────

export async function setGitHubToken(
	projectId: string,
	token: string,
): Promise<GitHubIntegration> {
	const { data } = await apiClient.instance.put<
		SuccessEnvelope<GitHubIntegration>
	>(`/projects/${projectId}/github/token`, { token });
	return data.data;
}

export async function getGitHubIntegration(
	projectId: string,
): Promise<GitHubIntegration> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<GitHubIntegration>
	>(`/projects/${projectId}/github`);
	return data.data;
}

export async function deleteGitHubToken(projectId: string): Promise<void> {
	await apiClient.instance.delete(`/projects/${projectId}/github/token`);
}

// ── Repository endpoints ──────────────────────────────────────────────────────

export async function listAccessibleRepos(
	projectId: string,
): Promise<AccessibleRepo[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<AccessibleRepo[]>
	>(`/projects/${projectId}/github/repositories`);
	return data.data;
}

export async function linkRepository(
	projectId: string,
	owner: string,
	repoName: string,
): Promise<LinkedRepository> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<LinkedRepository>
	>(`/projects/${projectId}/github/linked-repositories`, {
		owner,
		repo_name: repoName,
	});
	return data.data;
}

export async function listLinkedRepositories(
	projectId: string,
): Promise<LinkedRepository[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<LinkedRepository[]>
	>(`/projects/${projectId}/github/linked-repositories`);
	return data.data;
}

export async function unlinkRepository(
	projectId: string,
	repoId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/github/linked-repositories/${repoId}`,
	);
}

// ── Pull-request endpoints ────────────────────────────────────────────────────

export async function listTaskPRs(
	projectId: string,
	taskId: string,
): Promise<PullRequest[]> {
	const { data } = await apiClient.instance.get<SuccessEnvelope<PullRequest[]>>(
		`/projects/${projectId}/tasks/${taskId}/github/pull-requests`,
	);
	return data.data;
}

export async function linkPRToTask(
	projectId: string,
	taskId: string,
	repoId: string,
	prNumber: number,
): Promise<PullRequest> {
	const { data } = await apiClient.instance.post<SuccessEnvelope<PullRequest>>(
		`/projects/${projectId}/tasks/${taskId}/github/pull-requests`,
		{ repo_id: repoId, pr_number: prNumber },
	);
	return data.data;
}

export async function unlinkPRFromTask(
	projectId: string,
	taskId: string,
	prId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/tasks/${taskId}/github/pull-requests/${prId}`,
	);
}

// ── Branch endpoints ──────────────────────────────────────────────────────────

export async function createBranch(
	projectId: string,
	taskId: string,
	repoId: string,
	branchName: string,
	sourceBranch?: string,
): Promise<CreateBranchResult> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<CreateBranchResult>
	>(`/projects/${projectId}/tasks/${taskId}/github/branches`, {
		repo_id: repoId,
		branch_name: branchName,
		source_branch: sourceBranch,
	});
	return data.data;
}

// ── Query options ─────────────────────────────────────────────────────────────

export const githubIntegrationQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "github"],
		queryFn: () => getGitHubIntegration(projectId),
		retry: false,
	});

export const linkedRepositoriesQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "github", "linked-repositories"],
		queryFn: () => listLinkedRepositories(projectId),
		retry: false,
	});

export const accessibleReposQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "github", "repositories"],
		queryFn: () => listAccessibleRepos(projectId),
	});

export const taskPRsQueryOptions = (projectId: string, taskId: string) =>
	queryOptions({
		queryKey: [
			"projects",
			projectId,
			"tasks",
			taskId,
			"github",
			"pull-requests",
		],
		queryFn: () => listTaskPRs(projectId, taskId),
		enabled: !!projectId && !!taskId,
		retry: false,
	});
