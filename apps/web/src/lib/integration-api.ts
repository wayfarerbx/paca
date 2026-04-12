import { queryOptions } from "@tanstack/react-query";

import { apiClient } from "./api-client";
import type { SuccessEnvelope } from "./api-error";

// ── Shapes ────────────────────────────────────────────────────────────────────

export type SprintStatus = "planned" | "active" | "completed";

export interface Sprint {
	id: string;
	project_id: string;
	name: string;
	start_date?: string | null;
	end_date?: string | null;
	goal?: string | null;
	status: SprintStatus;
	created_at: string;
	updated_at: string;
}

export interface SprintListResult {
	items: Sprint[];
}

export interface Task {
	id: string;
	project_id: string;
	title: string;
	task_type_id?: string | null;
	status_id?: string | null;
	sprint_id?: string | null;
	parent_task_id?: string | null;
	description?: string | null;
	importance: number;
	assignee_id?: string | null;
	reporter_id?: string | null;
	custom_fields: Record<string, unknown>;
	start_date?: string | null;
	due_date?: string | null;
	tags?: string[];
	view_position?: number | null;
	view_group_key?: string | null;
	created_at: string;
	updated_at: string;
}

export interface TaskListResult {
	items: Task[];
	total: number;
	page: number;
	page_size: number;
}

// ── View types ─────────────────────────────────────────────────────────────────
export type ViewType = "table" | "board" | "roadmap";

export interface ViewConfig {
	fields?: string[];
	column_by?: string;
	swimlanes?: string;
	sort_by?: string;
	field_sum?: string;
	slice_by?: string;
}

export type ViewLayout = "Board" | "Table" | "Roadmap";

export interface IntegrationView {
	id: string;
	name: string;
	view_type: ViewType;
	layout: ViewLayout;
	config?: ViewConfig;
	position: number;
}

// ── View shape helpers ─────────────────────────────────────────────────────────
function viewTypeToLayout(vt: ViewType): ViewLayout {
	if (vt === "board") return "Board";
	if (vt === "roadmap") return "Roadmap";
	return "Table";
}

export function layoutToViewType(l: ViewLayout): ViewType {
	if (l === "Board") return "board";
	if (l === "Roadmap") return "roadmap";
	return "table";
}

function mapView(raw: Omit<IntegrationView, "layout">): IntegrationView {
	return { ...raw, layout: viewTypeToLayout(raw.view_type) };
}

// ── View API ──────────────────────────────────────────────────────────────────
interface ViewListResult {
	items: Omit<IntegrationView, "layout">[];
}

// Sprint views
export async function listViews(
	projectId: string,
	sprintId: string,
): Promise<IntegrationView[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<ViewListResult>
	>(`/projects/${projectId}/sprints/${sprintId}/views`);
	return data.data.items.map(mapView);
}

export async function createView(
	projectId: string,
	sprintId: string,
	payload: { name: string; view_type: ViewType; config?: ViewConfig },
): Promise<IntegrationView> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<Omit<IntegrationView, "layout">>
	>(`/projects/${projectId}/sprints/${sprintId}/views`, payload);
	return mapView(data.data);
}

export async function updateView(
	projectId: string,
	sprintId: string,
	viewId: string,
	payload: Partial<{
		name: string;
		view_type: ViewType;
		config: ViewConfig;
		position: number;
	}>,
): Promise<IntegrationView> {
	const { data } = await apiClient.instance.patch<
		SuccessEnvelope<Omit<IntegrationView, "layout">>
	>(`/projects/${projectId}/sprints/${sprintId}/views/${viewId}`, payload);
	return mapView(data.data);
}

export async function deleteView(
	projectId: string,
	sprintId: string,
	viewId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/sprints/${sprintId}/views/${viewId}`,
	);
}

// Backlog views
export async function listBacklogViews(
	projectId: string,
): Promise<IntegrationView[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<ViewListResult>
	>(`/projects/${projectId}/product-backlog/views`);
	return data.data.items.map(mapView);
}

export async function createBacklogView(
	projectId: string,
	payload: { name: string; view_type: ViewType; config?: ViewConfig },
): Promise<IntegrationView> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<Omit<IntegrationView, "layout">>
	>(`/projects/${projectId}/product-backlog/views`, payload);
	return mapView(data.data);
}

export async function updateBacklogView(
	projectId: string,
	viewId: string,
	payload: Partial<{
		name: string;
		view_type: ViewType;
		config: ViewConfig;
		position: number;
	}>,
): Promise<IntegrationView> {
	const { data } = await apiClient.instance.patch<
		SuccessEnvelope<Omit<IntegrationView, "layout">>
	>(`/projects/${projectId}/product-backlog/views/${viewId}`, payload);
	return mapView(data.data);
}

export async function deleteBacklogView(
	projectId: string,
	viewId: string,
): Promise<void> {
	await apiClient.instance.delete(
		`/projects/${projectId}/product-backlog/views/${viewId}`,
	);
}

export async function moveTaskPosition(
	projectId: string,
	sprintId: string,
	viewId: string,
	taskId: string,
	payload: { position: number; group_key?: string | null },
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/sprints/${sprintId}/views/${viewId}/task-positions/${taskId}`,
		payload,
	);
}

export async function moveBacklogTaskPosition(
	projectId: string,
	viewId: string,
	taskId: string,
	payload: { position: number; group_key?: string | null },
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/product-backlog/views/${viewId}/task-positions/${taskId}`,
		payload,
	);
}

export async function reorderViews(
	projectId: string,
	sprintId: string,
	viewIds: string[],
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/sprints/${sprintId}/views/positions`,
		{ view_ids: viewIds },
	);
}

export async function reorderBacklogViews(
	projectId: string,
	viewIds: string[],
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/product-backlog/views/positions`,
		{ view_ids: viewIds },
	);
}

// ── Sprint API ────────────────────────────────────────────────────────────────

export async function listSprints(projectId: string): Promise<Sprint[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<SprintListResult>
	>(`/projects/${projectId}/sprints`);
	return data.data.items;
}

export async function getSprint(
	projectId: string,
	sprintId: string,
): Promise<Sprint> {
	const { data } = await apiClient.instance.get<SuccessEnvelope<Sprint>>(
		`/projects/${projectId}/sprints/${sprintId}`,
	);
	return data.data;
}

export interface CreateSprintPayload {
	name: string;
	status?: SprintStatus;
	goal?: string | null;
	start_date?: string | null;
	end_date?: string | null;
}

export async function createSprint(
	projectId: string,
	payload: CreateSprintPayload,
): Promise<Sprint> {
	const { data } = await apiClient.instance.post<SuccessEnvelope<Sprint>>(
		`/projects/${projectId}/sprints`,
		payload,
	);
	return data.data;
}

// ── Task API ──────────────────────────────────────────────────────────────────

export interface ListTasksOptions {
	sprintId?: string;
	statusId?: string;
	assigneeId?: string;
	page?: number;
	pageSize?: number;
	viewId?: string;
}

export async function listBacklogTasks(
	projectId: string,
	opts: ListTasksOptions = {},
): Promise<TaskListResult> {
	const params: Record<string, string | number> = {
		page: opts.page ?? 1,
		page_size: opts.pageSize ?? 200,
	};
	if (opts.statusId) params.status_id = opts.statusId;
	if (opts.assigneeId) params.assignee_id = opts.assigneeId;
	if (opts.viewId) params.view_id = opts.viewId;

	const { data } = await apiClient.instance.get<
		SuccessEnvelope<TaskListResult>
	>(`/projects/${projectId}/product-backlog`, { params });
	return data.data;
}

export async function listSprintTasks(
	projectId: string,
	sprintId: string,
	opts: ListTasksOptions = {},
): Promise<TaskListResult> {
	const params: Record<string, string | number> = {
		page: opts.page ?? 1,
		page_size: opts.pageSize ?? 200,
	};
	if (opts.statusId) params.status_id = opts.statusId;
	if (opts.assigneeId) params.assignee_id = opts.assigneeId;
	if (opts.viewId) params.view_id = opts.viewId;

	const { data } = await apiClient.instance.get<
		SuccessEnvelope<TaskListResult>
	>(`/projects/${projectId}/sprints/${sprintId}/tasks`, { params });
	return data.data;
}

export async function createTask(
	projectId: string,
	payload: {
		title: string;
		status_id?: string | null;
		sprint_id?: string | null;
		task_type_id?: string | null;
		assignee_id?: string | null;
		parent_task_id?: string | null;
	},
): Promise<Task> {
	const { data } = await apiClient.instance.post<SuccessEnvelope<Task>>(
		`/projects/${projectId}/tasks`,
		payload,
	);
	return data.data;
}

export async function getTask(
	projectId: string,
	taskId: string,
): Promise<Task> {
	const { data } = await apiClient.instance.get<SuccessEnvelope<Task>>(
		`/projects/${projectId}/tasks/${taskId}`,
	);
	return data.data;
}

export async function updateTask(
	projectId: string,
	taskId: string,
	payload: Partial<{
		title: string;
		status_id: string | null;
		sprint_id: string | null;
		task_type_id: string | null;
		assignee_id: string | null;
		reporter_id: string | null;
		parent_task_id: string | null;
		description: string | null;
		importance: number;
		start_date: string | null;
		due_date: string | null;
		tags: string[];
		custom_fields: Record<string, unknown>;
	}>,
): Promise<Task> {
	const { data } = await apiClient.instance.patch<SuccessEnvelope<Task>>(
		`/projects/${projectId}/tasks/${taskId}`,
		payload,
	);
	return data.data;
}

export async function listSubtasks(
	projectId: string,
	parentTaskId: string,
): Promise<Task[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<TaskListResult>
	>(`/projects/${projectId}/tasks`, {
		params: { parent_task_id: parentTaskId, page: 1, page_size: 200 },
	});
	return data.data.items;
}

// ── Query Options ─────────────────────────────────────────────────────────────

export const taskQueryOptions = (projectId: string, taskId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "tasks", taskId],
		queryFn: () => getTask(projectId, taskId),
		staleTime: 15_000,
	});

export const subtasksQueryOptions = (projectId: string, parentTaskId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "tasks", parentTaskId, "subtasks"],
		queryFn: () => listSubtasks(projectId, parentTaskId),
		staleTime: 15_000,
	});

export const sprintsQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "sprints"],
		queryFn: () => listSprints(projectId),
		staleTime: 30_000,
	});

export const sprintQueryOptions = (projectId: string, sprintId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "sprints", sprintId],
		queryFn: () => getSprint(projectId, sprintId),
		staleTime: 30_000,
	});

export const backlogTasksQueryOptions = (projectId: string, viewId?: string) =>
	queryOptions({
		queryKey: viewId
			? ["projects", projectId, "backlog-tasks", viewId]
			: ["projects", projectId, "backlog-tasks"],
		queryFn: () => listBacklogTasks(projectId, { viewId }),
		staleTime: 15_000,
	});

export const sprintTasksQueryOptions = (
	projectId: string,
	sprintId: string,
	viewId?: string,
) =>
	queryOptions({
		queryKey: viewId
			? ["projects", projectId, "sprints", sprintId, "tasks", viewId]
			: ["projects", projectId, "sprints", sprintId, "tasks"],
		queryFn: () => listSprintTasks(projectId, sprintId, { viewId }),
		staleTime: 15_000,
	});

export const viewsQueryOptions = (projectId: string, sprintId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "sprints", sprintId, "views"],
		queryFn: () => listViews(projectId, sprintId),
		staleTime: 30_000,
	});

export const backlogViewsQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: ["projects", projectId, "backlog-views"],
		queryFn: () => listBacklogViews(projectId),
		staleTime: 30_000,
	});
