import { useMemo } from "react";

import type { Sprint, Task, ViewConfig } from "@/lib/interaction-api";
import type {
	CustomFieldDefinition,
	ProjectMember,
	TaskStatus,
	TaskType,
} from "@/lib/project-api";

import { ListGroup } from "./list-group";
import {
	type ColumnGroupDef,
	type TaskFieldUpdate,
	DEFAULT_VISIBLE_FIELDS,
	getColumnGroupDefs,
	getSwimlaneDefs,
	getTaskColumnKeys,
} from "./view-utils";

// ── Props ─────────────────────────────────────────────────────────────────────

export interface ListViewProps {
	tasks: Task[];
	taskIdPrefix?: string;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members?: ProjectMember[];
	customFields?: CustomFieldDefinition[];
	viewConfig?: ViewConfig;
	canCreate: boolean;
	searchQuery: string;
	assigneeFilter: string | null;
	onCreateTask: (
		statusId: string,
		title: string,
		taskTypeId?: string | null,
		extraFields?: TaskFieldUpdate,
	) => Promise<void>;
	onTaskClick: (task: Task) => void;
	manualSort?: boolean;
	onReorderTask?: (groupKey: string, taskId: string, newIndex: number) => void;
	onStatusChange?: (taskId: string, newStatusId: string) => void;
	canEdit?: boolean;
	sortBy?: string;
	onUpdateTaskField?: (taskId: string, update: TaskFieldUpdate) => void;
	sprints?: Sprint[];
	onStartSprint?: (
		sprintId: string,
		payload: {
			name: string;
			goal: string | null;
			start_date: string | null;
			end_date: string | null;
			status: "active";
		},
	) => Promise<void>;
	onCreateSprint?: () => void;
}

// ── Component ─────────────────────────────────────────────────────────────────

export function ListView({
	tasks,
	taskIdPrefix = "",
	statuses,
	taskTypes,
	members = [],
	customFields = [],
	viewConfig,
	canCreate,
	searchQuery,
	assigneeFilter,
	onCreateTask,
	onTaskClick,
	manualSort,
	onReorderTask,
	onStatusChange,
	canEdit,
	sortBy,
	onUpdateTaskField,
	sprints,
	onStartSprint,
	onCreateSprint,
}: ListViewProps) {
	const columnBy = viewConfig?.column_by ?? "status";
	const swimlaneBy = viewConfig?.swimlanes;
	const fieldSum = viewConfig?.field_sum;
	const visibleFields: string[] =
		viewConfig?.fields && viewConfig.fields.length > 0
			? viewConfig.fields
			: DEFAULT_VISIBLE_FIELDS;
	const isStatusGrouping = !viewConfig?.column_by || viewConfig.column_by === "status";

	const viewCtx = useMemo(
		() => ({ statuses, taskTypes, members, customFields, sprints }),
		[statuses, taskTypes, members, customFields, sprints],
	);

	const filtered = useMemo(
		() =>
			tasks.filter((t) => {
				if (searchQuery) {
					const q = searchQuery.toLowerCase();
					const taskId = taskIdPrefix
						? `${taskIdPrefix}-${t.task_number}`
						: `#${t.task_number}`;
					if (
						!t.title.toLowerCase().includes(q) &&
						!taskId.toLowerCase().includes(q)
					)
						return false;
				}
				if (assigneeFilter && t.assignee_id !== assigneeFilter) return false;
				return true;
			}),
		[tasks, searchQuery, assigneeFilter, taskIdPrefix],
	);

	const groupDefs = useMemo(
		() => getColumnGroupDefs(columnBy, viewCtx),
		[columnBy, viewCtx],
	);

	const effectiveGroupDefs = useMemo((): ColumnGroupDef[] => {
		if (groupDefs.length > 0) return groupDefs;
		const seen = new Set<string>();
		const dynamic: ColumnGroupDef[] = [];
		for (const t of filtered) {
			for (const k of getTaskColumnKeys(t, columnBy, viewCtx)) {
				if (!seen.has(k)) {
					seen.add(k);
					dynamic.push({ key: k, label: k === "__none" ? "None" : k, fieldValue: k });
				}
			}
		}
		if (!seen.has("__none")) {
			dynamic.push({ key: "__none", label: "None", fieldValue: null });
		}
		return dynamic;
	}, [groupDefs, filtered, columnBy, viewCtx]);

	const swimlaneDefs = useMemo(
		() => getSwimlaneDefs(swimlaneBy, viewCtx),
		[swimlaneBy, viewCtx],
	);

	const getGroupTasks = (groupKey: string): Task[] =>
		filtered.filter((t) => getTaskColumnKeys(t, columnBy, viewCtx).includes(groupKey));

	return (
		<div className="flex flex-col overflow-auto">
			{effectiveGroupDefs.map((grp) => {
				const groupTasks = getGroupTasks(grp.key);
				const status = isStatusGrouping
					? statuses.find((s) => s.id === grp.key)
					: undefined;
				const isDone = status?.category === "done";

				return (
					<ListGroup
						key={grp.key}
						groupDef={grp}
						tasks={groupTasks}
						statuses={statuses}
						taskTypes={taskTypes}
						members={members}
						customFields={customFields}
						canCreate={canCreate}
						defaultCollapsed={isDone}
						fieldSum={fieldSum}
						swimlaneDefs={swimlaneDefs}
						swimlaneBy={swimlaneBy}
						onCreateTask={onCreateTask}
						onTaskClick={onTaskClick}
						manualSort={manualSort}
						onReorderTask={onReorderTask}
						onStatusChange={onStatusChange}
						canEdit={canEdit}
						isStatusGrouping={isStatusGrouping}
						sortBy={sortBy}
						onUpdateTaskField={onUpdateTaskField}
						visibleFields={visibleFields}
						taskIdPrefix={taskIdPrefix}
						sprint={sprints?.find((s) => s.id === grp.key)}
						onStartSprint={onStartSprint}
						onCreateSprint={onCreateSprint}
						columnBy={columnBy}
						extraCreateFields={
							!isStatusGrouping && columnBy === "sprint"
								? { sprint_id: grp.key === "__backlog" ? null : (grp.key as string) }
								: undefined
						}
					/>
				);
			})}
		</div>
	);
}
