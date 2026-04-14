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
	task_number: number;
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

export interface TaskPosition {
	view_id: string;
	task_id: string;
	position: number;
	group_key?: string | null;
}

interface TaskPositionListResult {
	items: TaskPosition[];
}

// ── View types ─────────────────────────────────────────────────────────────────
export type ViewType = "table" | "board" | "roadmap";

export interface ViewFilters {
	sprint_ids?: string[];
	status_ids?: string[];
	assignee_ids?: string[];
	task_type_ids?: string[];
}

export interface ViewConfig {
	fields?: string[];
	column_by?: string;
	swimlanes?: string;
	sort_by?: string;
	field_sum?: string;
	slice_by?: string;
	filters?: ViewFilters;
}

export type ViewLayout = "Board" | "Table" | "Roadmap";

export interface InteractionView {
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

function mapView(raw: Omit<InteractionView, "layout">): InteractionView {
	return { ...raw, layout: viewTypeToLayout(raw.view_type) };
}

// ── View API ──────────────────────────────────────────────────────────────────
interface ViewListResult {
	items: Omit<InteractionView, "layout">[];
}

// ── Unified context-based View API ───────────────────────────────────────────

export type ViewsContext = "sprint" | "backlog" | "timeline";

function viewsContextQuery(context: ViewsContext, sprintId?: string | null): string {
	if (context === "sprint" && sprintId) return `context=sprint&sprint_id=${sprintId}`;
	return `context=${context}`;
}

export async function listViewsByContext(
	projectId: string,
	context: ViewsContext,
	sprintId?: string | null,
): Promise<InteractionView[]> {
	const { data } = await apiClient.instance.get<SuccessEnvelope<ViewListResult>>(
		`/projects/${projectId}/views?${viewsContextQuery(context, sprintId)}`,
	);
	return data.data.items.map(mapView);
}

export async function createViewByContext(
	projectId: string,
	context: ViewsContext,
	payload: { name: string; view_type: ViewType; config?: ViewConfig },
	sprintId?: string | null,
): Promise<InteractionView> {
	const { data } = await apiClient.instance.post<
		SuccessEnvelope<Omit<InteractionView, "layout">>
	>(
		`/projects/${projectId}/views?${viewsContextQuery(context, sprintId)}`,
		payload,
	);
	return mapView(data.data);
}

export async function updateViewById(
	projectId: string,
	viewId: string,
	payload: Partial<{ name: string; view_type: ViewType; config: ViewConfig; position: number }>,
): Promise<InteractionView> {
	const { data } = await apiClient.instance.patch<
		SuccessEnvelope<Omit<InteractionView, "layout">>
	>(`/projects/${projectId}/views/${viewId}`, payload);
	return mapView(data.data);
}

export async function deleteViewById(
	projectId: string,
	viewId: string,
): Promise<void> {
	await apiClient.instance.delete(`/projects/${projectId}/views/${viewId}`);
}

export async function reorderViewsByContext(
	projectId: string,
	context: ViewsContext,
	viewIds: string[],
	sprintId?: string | null,
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/views/positions?${viewsContextQuery(context, sprintId)}`,
		{ view_ids: viewIds },
	);
}

export async function bulkMoveViewTaskPositions(
	projectId: string,
	viewId: string,
	items: Array<{ task_id: string; position: number; group_key?: string | null }>,
): Promise<void> {
	await apiClient.instance.put(
		`/projects/${projectId}/views/${viewId}/task-positions`,
		{ items },
	);
}

export const viewsByContextQueryOptions = (
	projectId: string,
	context: ViewsContext,
	sprintId?: string | null,
) => {
	const queryKey =
		context === "sprint" && sprintId
			? (["projects", projectId, "sprints", sprintId, "views"] as const)
			: context === "backlog"
				? (["projects", projectId, "backlog-views"] as const)
				: (["projects", projectId, "timeline-views"] as const);
	return queryOptions({
		queryKey,
		queryFn: () => listViewsByContext(projectId, context, sprintId),
		staleTime: 30_000,
	});
};

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

export interface UpdateSprintPayload {
	name?: string;
	status?: SprintStatus;
	goal?: string | null;
	start_date?: string | null;
	end_date?: string | null;
}

export async function updateSprint(
	projectId: string,
	sprintId: string,
	payload: UpdateSprintPayload,
): Promise<Sprint> {
	const { data } = await apiClient.instance.patch<SuccessEnvelope<Sprint>>(
		`/projects/${projectId}/sprints/${sprintId}`,
		payload,
	);
	return data.data;
}

// ── Task API ──────────────────────────────────────────────────────────────────

export interface ListTasksOptions {
	sprintId?: string | null;
	sprintIds?: string[];
	statusId?: string;
	statusIds?: string[];
	assigneeId?: string;
	assigneeIds?: string[];
	taskTypeIds?: string[];
	parentTaskId?: string;
	page?: number;
	pageSize?: number;
}

function buildTaskQueryParams(opts: ListTasksOptions = {}) {
	const params: Record<string, string | number | boolean> = {
		page: opts.page ?? 1,
		page_size: opts.pageSize ?? 200,
	};
	if (opts.sprintId === null) params.sprint_id = "null";
	else if (opts.sprintId) params.sprint_id = opts.sprintId;
	if (opts.sprintIds && opts.sprintIds.length > 0) params.sprint_ids = opts.sprintIds.join(",");
	if (opts.statusId) params.status_id = opts.statusId;
	if (opts.statusIds && opts.statusIds.length > 0) params.status_ids = opts.statusIds.join(",");
	if (opts.assigneeId) params.assignee_id = opts.assigneeId;
	if (opts.assigneeIds && opts.assigneeIds.length > 0) params.assignee_ids = opts.assigneeIds.join(",");
	if (opts.taskTypeIds && opts.taskTypeIds.length > 0) params.task_type_ids = opts.taskTypeIds.join(",");
	if (opts.parentTaskId) params.parent_task_id = opts.parentTaskId;
	return params;
}

export async function listAllTasks(
	projectId: string,
	opts: ListTasksOptions = {},
): Promise<TaskListResult> {
	const params = buildTaskQueryParams(opts);
	const { data } = await apiClient.instance.get<SuccessEnvelope<TaskListResult>>(
		`/projects/${projectId}/tasks`,
		{ params },
	);
	return data.data;
}

export async function listSprintTasks(
	projectId: string,
	sprintId: string,
	opts: ListTasksOptions = {},
): Promise<TaskListResult> {
	return listAllTasks(projectId, {
		...opts,
		sprintId,
	});
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

export async function listViewTaskPositions(
	projectId: string,
	viewId: string,
): Promise<TaskPosition[]> {
	const { data } = await apiClient.instance.get<
		SuccessEnvelope<TaskPositionListResult>
	>(`/projects/${projectId}/views/${viewId}/task-positions`);
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

export const allTasksQueryOptions = (
	projectId: string,
	opts: ListTasksOptions = {},
) =>
	queryOptions({
		queryKey: ["projects", projectId, "tasks", opts],
		queryFn: () => listAllTasks(projectId, opts),
		staleTime: 15_000,
	});

export const viewTaskPositionsQueryOptions = (
	projectId: string,
	viewId: string,
) =>
	queryOptions({
		queryKey: ["projects", projectId, "views", viewId, "task-positions"],
		queryFn: () => listViewTaskPositions(projectId, viewId),
		staleTime: 15_000,
	});

export const sprintTasksQueryOptions = (
	projectId: string,
	sprintId: string,
	filters?: ViewFilters,
) =>
	queryOptions({
		queryKey: ["projects", projectId, "sprints", sprintId, "tasks", filters],
		queryFn: () =>
			listSprintTasks(projectId, sprintId, {
				sprintIds: filters?.sprint_ids,
				statusIds: filters?.status_ids,
				assigneeIds: filters?.assignee_ids,
				taskTypeIds: filters?.task_type_ids,
			}),
		staleTime: 15_000,
	});

// end of interaction-api
