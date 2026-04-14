import type { Sprint, Task, ViewConfig } from "@/lib/interaction-api";
import type {
	CustomFieldDefinition,
	ProjectMember,
	TaskStatus,
	TaskType,
} from "@/lib/project-api";

import { IMPORTANCE_BUCKET_VALUES, PRIORITY_LEVELS, getImportanceBucket } from "./priority";

// ── Context ──────────────────────────────────────────────────────────────────

export interface ViewContext {
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members: ProjectMember[];
	customFields: CustomFieldDefinition[];
	sprints?: Sprint[];
}

// ── Column / Swimlane group definition ───────────────────────────────────────

export interface ColumnGroupDef {
	/** Unique key for this column */
	key: string;
	/** Display label */
	label: string;
	/** Optional badge colour */
	color?: string;
	/**
	 * The raw field value associated with this column.
	 * Used when dragging a task to this column to update the task's field.
	 */
	fieldValue: unknown;
}

// ── Built-in column groups per field ─────────────────────────────────────────

/**
 * Returns all possible column group definitions for the given column_by field.
 * All columns are returned even if they contain zero tasks.
 */
export function getColumnGroupDefs(
	columnBy: string | undefined,
	ctx: ViewContext,
): ColumnGroupDef[] {
	if (!columnBy || columnBy === "status") {
		return ctx.statuses
			.slice()
			.sort((a, b) => a.position - b.position)
			.map((s) => ({
				key: s.id,
				label: s.name,
				color: s.color ?? undefined,
				fieldValue: s.id,
			}));
	}

	if (columnBy === "sprint") {
		const sprintGroups = (ctx.sprints ?? [])
			.filter((s) => s.status !== "completed")
			.sort((a, b) => {
				if (a.status === b.status) return a.name.localeCompare(b.name);
				return a.status === "active" ? -1 : 1;
			})
			.map((s) => ({
				key: s.id,
				label: s.name,
				fieldValue: s.id,
			}));
		return [
			...sprintGroups,
			{ key: "__backlog", label: "Backlog", fieldValue: null },
		];
	}

	if (columnBy === "assignee") {
		return [
			...ctx.members.map((m) => ({
				key: m.id,
				label: m.full_name || m.username,
				fieldValue: m.id,
			})),
			{ key: "__unassigned", label: "Unassigned", fieldValue: null },
		];
	}

	if (columnBy === "importance") {
		return PRIORITY_LEVELS.map((p) => ({
			key: String(p.value),
			label: p.label,
			color: p.value > 0 ? p.color : undefined,
			// fieldValue is the representative raw importance to assign on drop
			fieldValue: IMPORTANCE_BUCKET_VALUES[p.value] ?? 0,
		}));
	}

	if (columnBy === "type") {
		return [
			...ctx.taskTypes.map((t) => ({
				key: t.id,
				label: t.name,
				color: t.color ?? undefined,
				fieldValue: t.id,
			})),
			{ key: "__none", label: "No Type", fieldValue: null },
		];
	}

	// Custom field
	const cf = ctx.customFields.find((f) => f.field_key === columnBy);
	if (cf) {
		if (cf.field_type === "select" || cf.field_type === "multi_select") {
			return [
				...cf.options.map((opt) => ({
					key: opt,
					label: opt,
					fieldValue: opt,
				})),
				{ key: "__none", label: "None", fieldValue: null },
			];
		}
		if (cf.field_type === "boolean") {
			return [
				{ key: "true", label: "Yes", fieldValue: true },
				{ key: "false", label: "No", fieldValue: false },
			];
		}
		// number / text / date → dynamic groups from task values (built at render time)
		return [];
	}

	return [];
}

/**
 * Returns the column key(s) for a task given the column_by field.
 * Returns an array because multi_select tasks can appear in multiple columns.
 */
export function getTaskColumnKeys(
	task: Task,
	columnBy: string | undefined,
	ctx: ViewContext,
): string[] {
	if (!columnBy || columnBy === "status") {
		return [task.status_id ?? "__none"];
	}
	if (columnBy === "sprint") {
		return [task.sprint_id ?? "__backlog"];
	}
	if (columnBy === "assignee") {
		return [task.assignee_id ?? "__unassigned"];
	}
	if (columnBy === "importance") {
		// Bucket the raw numeric importance into one of the 5 named levels
		return [String(getImportanceBucket(task.importance))];
	}
	if (columnBy === "type") {
		return [task.task_type_id ?? "__none"];
	}

	const cf = ctx.customFields.find((f) => f.field_key === columnBy);
	if (cf) {
		const val = task.custom_fields[cf.field_key];
		if (cf.field_type === "multi_select") {
			if (Array.isArray(val) && val.length > 0) {
				return val.map(String);
			}
			return ["__none"];
		}
		if (val === null || val === undefined || val === "") {
			return ["__none"];
		}
		if (cf.field_type === "boolean") {
			return [val ? "true" : "false"];
		}
		return [String(val)];
	}

	return ["__none"];
}

// ── Swimlane helpers ──────────────────────────────────────────────────────────

/**
 * Returns swimlane group definitions.
 * Returns a single "__all" group when swimlanes are disabled.
 */
export function getSwimlaneDefs(
	swimlanes: string | undefined,
	ctx: ViewContext,
): ColumnGroupDef[] {
	if (!swimlanes || swimlanes === "none") {
		return [{ key: "__all", label: "", fieldValue: null }];
	}
	const defs = getColumnGroupDefs(swimlanes, ctx);
	// Show Critical (highest bucket) first, None last
	if (swimlanes === "importance") return [...defs].reverse();
	return defs;
}

/** Returns the swimlane key for a task. */
export function getTaskSwimlaneKey(
	task: Task,
	swimlanes: string | undefined,
	ctx: ViewContext,
): string {
	if (!swimlanes || swimlanes === "none") return "__all";
	return getTaskColumnKeys(task, swimlanes, ctx)[0];
}

// ── Sort ──────────────────────────────────────────────────────────────────────

/**
 * Sorts tasks according to the view config's sort_by value.
 * When sort_by is "manual" or unset the original order is preserved.
 */
export function sortTasksByConfig(
	tasks: Task[],
	config: ViewConfig | undefined,
	ctx: ViewContext,
): Task[] {
	const sortBy = config?.sort_by;
	if (!sortBy || sortBy === "manual") return tasks;

	return [...tasks].sort((a, b) => {
		if (sortBy === "importance") return b.importance - a.importance;
		if (sortBy === "title") return a.title.localeCompare(b.title);
		if (sortBy === "created") return a.created_at.localeCompare(b.created_at);
		if (sortBy === "start_date") {
			const ad = a.start_date ?? "";
			const bd = b.start_date ?? "";
			if (!ad && !bd) return 0;
			if (!ad) return 1;
			if (!bd) return -1;
			return ad.localeCompare(bd);
		}
		if (sortBy === "due_date") {
			const ad = a.due_date ?? "";
			const bd = b.due_date ?? "";
			if (!ad && !bd) return 0;
			if (!ad) return 1;
			if (!bd) return -1;
			return ad.localeCompare(bd);
		}

		// Custom field sort
		const cf = ctx.customFields.find((f) => f.field_key === sortBy);
		if (cf) {
			const av = a.custom_fields[cf.field_key];
			const bv = b.custom_fields[cf.field_key];
			if (av === null || av === undefined) return 1; // nulls last
			if (bv === null || bv === undefined) return -1;
			if (cf.field_type === "number") {
				return (Number(av) || 0) - (Number(bv) || 0);
			}
			if (cf.field_type === "date") {
				return String(av).localeCompare(String(bv));
			}
			if (cf.field_type === "select") {
				const ai = cf.options.indexOf(String(av));
				const bi = cf.options.indexOf(String(bv));
				return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi);
			}
			return String(av).localeCompare(String(bv));
		}

		return 0;
	});
}

// ── Field sum ─────────────────────────────────────────────────────────────────

/**
 * Computes the aggregate value shown in the column/group heading.
 * When field_sum is "count" or unset the task count is returned.
 */
export function computeFieldSum(
	tasks: Task[],
	fieldSum: string | undefined,
	customFields: CustomFieldDefinition[],
): number {
	if (!fieldSum || fieldSum === "count") return tasks.length;
	const cf = customFields.find((f) => f.field_key === fieldSum);
	if (!cf) return tasks.length;
	return tasks.reduce((acc, t) => {
		const v = t.custom_fields[cf.field_key];
		return acc + (typeof v === "number" ? v : Number(v) || 0);
	}, 0);
}

// ── Option builders for ViewSettingsPanel dropdowns ──────────────────────────

export const BUILTIN_COLUMN_BY: { key: string; label: string }[] = [
	{ key: "status", label: "Status" },
	{ key: "sprint", label: "Sprint" },
	{ key: "assignee", label: "Assignee" },
	{ key: "importance", label: "Importance" },
	{ key: "type", label: "Type" },
	{ key: "reporter", label: "Reporter" },
];

export const BUILTIN_SORT_BY: { key: string; label: string }[] = [
	{ key: "manual", label: "Manual" },
	{ key: "importance", label: "Importance" },
	{ key: "title", label: "Title" },
	{ key: "created", label: "Created" },
	{ key: "start_date", label: "Start Date" },
	{ key: "due_date", label: "Due Date" },
];

export const BUILTIN_SWIMLANES: { key: string; label: string }[] = [
	{ key: "none", label: "None" },
	{ key: "assignee", label: "Assignee" },
	{ key: "importance", label: "Importance" },
	{ key: "type", label: "Type" },
];

export const BUILTIN_SLICE_BY: { key: string; label: string }[] = [
	{ key: "none", label: "None" },
	{ key: "assignee", label: "Assignee" },
	{ key: "importance", label: "Importance" },
	{ key: "type", label: "Type" },
];

export const FIELD_SUM_COUNT: { key: string; label: string } = {
	key: "count",
	label: "Count",
};

/** All built-in fields available for the Field Picker. Title is excluded — it is always visible. */
export const BUILTIN_FIELDS: { key: string; label: string }[] = [
	{ key: "assignee", label: "Assignee" },
	{ key: "status", label: "Status" },
	{ key: "importance", label: "Importance" },
	{ key: "type", label: "Type" },
	{ key: "reporter", label: "Reporter" },
	{ key: "start_date", label: "Start Date" },
	{ key: "due_date", label: "Due Date" },
	{ key: "created", label: "Created" },
];

/**
 * Default visible fields (excluding title, which is always shown).
 * Used when a view has no field config saved yet.
 */
export const DEFAULT_VISIBLE_FIELDS = [
	"assignee",
	"importance",
	"type",
];

export function buildColumnByOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields
		.filter((cf) =>
			["select", "multi_select", "boolean", "number"].includes(cf.field_type),
		)
		.map((cf) => ({ key: cf.field_key, label: cf.display_name }));
	return [...BUILTIN_COLUMN_BY, ...custom];
}

export function buildSortByOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields
		.filter((cf) => ["number", "date", "select"].includes(cf.field_type))
		.map((cf) => ({ key: cf.field_key, label: cf.display_name }));
	return [...BUILTIN_SORT_BY, ...custom];
}

export function buildSwimlaneOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields
		.filter((cf) =>
			["select", "multi_select", "boolean"].includes(cf.field_type),
		)
		.map((cf) => ({ key: cf.field_key, label: cf.display_name }));
	return [...BUILTIN_SWIMLANES, ...custom];
}

export function buildFieldSumOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields
		.filter((cf) => cf.field_type === "number")
		.map((cf) => ({ key: cf.field_key, label: cf.display_name }));
	return [FIELD_SUM_COUNT, ...custom];
}

export function buildSliceByOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields
		.filter((cf) => ["select", "multi_select"].includes(cf.field_type))
		.map((cf) => ({ key: cf.field_key, label: cf.display_name }));
	return [...BUILTIN_SLICE_BY, ...custom];
}

export function buildAllFieldOptions(
	customFields: CustomFieldDefinition[],
): { key: string; label: string }[] {
	const custom = customFields.map((cf) => ({
		key: cf.field_key,
		label: cf.display_name,
	}));
	return [...BUILTIN_FIELDS, ...custom];
}

// ── Task update payload builder for column drag ───────────────────────────────

export type TaskFieldUpdate = Partial<{
	status_id: string | null;
	assignee_id: string | null;
	importance: number;
	task_type_id: string | null;
	custom_fields: Record<string, unknown>;
	sprint_id: string | null;
}>;

/**
 * Builds the update payload when a task is dragged into a column whose
 * column_by field value is `columnFieldValue`.
 */
export function buildColumnDropUpdate(
	columnBy: string | undefined,
	columnFieldValue: unknown,
	customFields: CustomFieldDefinition[],
): TaskFieldUpdate {
	if (!columnBy || columnBy === "status") {
		return { status_id: (columnFieldValue as string | null) ?? null };
	}
	if (columnBy === "sprint") {
		return { sprint_id: (columnFieldValue as string | null) ?? null };
	}
	if (columnBy === "assignee") {
		return { assignee_id: (columnFieldValue as string | null) ?? null };
	}
	if (columnBy === "importance") {
		return { importance: Number(columnFieldValue) || 0 };
	}
	if (columnBy === "type") {
		return { task_type_id: (columnFieldValue as string | null) ?? null };
	}

	const cf = customFields.find((f) => f.field_key === columnBy);
	if (cf) {
		return { custom_fields: { [cf.field_key]: columnFieldValue } };
	}

	return {};
}
