import {
	ArrowRight,
	BookOpen,
	Clock,
	KanbanSquare,
	Link2,
	Plus,
} from "lucide-react";
import { useEffect, useState } from "react";
import type { Sprint, Task } from "@/lib/integration-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";
import { getTaskTypeIconComponent } from "../../task-types/task-type-icons";
import type { PriorityMeta } from "../priority";
import { AddFieldDialog } from "./add-field-dialog";
import { FieldRow, FieldValue } from "./primitives";
import type { SelectOption, UserOption } from "./property-field";
import { PropertyField } from "./property-field";
import type { CustomFieldDef } from "./types";

type UpdatePayload = Partial<{
	status_id: string | null;
	task_type_id: string | null;
	assignee_id: string | null;
	reporter_id: string | null;
	importance: number;
	start_date: string | null;
	due_date: string | null;
	tags: string[];
	sprint_id: string | null;
}>;

interface PropertiesPanelProps {
	task: Task;
	status: TaskStatus | undefined;
	taskType: TaskType | undefined;
	priority: PriorityMeta;
	assignee: ProjectMember | undefined;
	reporter: ProjectMember | undefined;
	statuses?: TaskStatus[];
	taskTypes?: TaskType[];
	members?: ProjectMember[];
	sprints?: Sprint[];
	initialCustomFields?: CustomFieldDef[];
	canEdit?: boolean;
	onUpdate?: (payload: UpdatePayload) => void;
}

function toUserOption(m: ProjectMember): UserOption {
	return {
		value: m.user_id,
		label: m.full_name || m.username,
		initials: (m.full_name || m.username).slice(0, 1).toUpperCase(),
	};
}

export function PropertiesPanel({
	task,
	status,
	taskType,
	assignee,
	reporter,
	statuses = [],
	taskTypes = [],
	members = [],
	sprints = [],
	initialCustomFields = [],
	canEdit = true,
	onUpdate,
}: PropertiesPanelProps) {
	const [localCustomFields, setLocalCustomFields] =
		useState<CustomFieldDef[]>(initialCustomFields);
	const [addFieldOpen, setAddFieldOpen] = useState(false);

	useEffect(() => {
		setLocalCustomFields(initialCustomFields);
	}, [initialCustomFields]);

	const statusOptions: SelectOption[] = statuses.map((s) => ({
		value: s.id,
		label: s.name,
		colorDot: s.color ?? undefined,
	}));

	const taskTypeOptions: SelectOption[] = taskTypes.map((tt) => {
		const Ic = getTaskTypeIconComponent(tt.icon);
		return {
			value: tt.id,
			label: tt.name,
			icon: Ic ? (
				<Ic className="size-3.5 text-muted-foreground/80 shrink-0" />
			) : undefined,
		};
	});

	const sprintOptions: SelectOption[] = [
		{
			value: "__backlog__",
			label: "Product Backlog",
			icon: <BookOpen className="size-3 shrink-0 opacity-60" />,
		},
		...sprints.map((s) => ({
			value: s.id,
			label: s.name,
			icon: (
				<KanbanSquare className="size-3 shrink-0 text-muted-foreground/70" />
			),
		})),
	];

	const memberUserOptions: UserOption[] = members.map(toUserOption);
	const assigneeUserOption = assignee ? toUserOption(assignee) : null;
	const reporterUserOption = reporter ? toUserOption(reporter) : null;

	return (
		<>
			<div className="divide-y divide-border/20 rounded-xl border border-border/30 bg-card/50 px-4 py-0.5">
				<PropertyField
					label="Status"
					mode="select"
					value={status?.id}
					options={statusOptions}
					onChange={(v) => onUpdate?.({ status_id: typeof v === "string" ? v : null })}
					canEdit={canEdit && statuses.length > 0}
				/>

				<PropertyField
					label="Dates"
					mode="date-range"
					startDate={task.start_date}
					dueDate={task.due_date}
					onStartDateChange={(v) => onUpdate?.({ start_date: v })}
					onDueDateChange={(v) => onUpdate?.({ due_date: v })}
					canEdit={canEdit}
				/>

				<FieldRow label="Track Time">
					<button
						type="button"
						className="inline-flex items-center gap-1.5 text-[12px] text-muted-foreground/70 hover:text-foreground transition-colors duration-150 font-medium"
					>
						<Clock className="size-3.5 opacity-70" />
						Add time
					</button>
				</FieldRow>

				<PropertyField
					label="Type"
					mode="select"
					value={taskType?.id}
					options={taskTypeOptions}
					onChange={(v) => onUpdate?.({ task_type_id: typeof v === "string" ? v : null })}
					canEdit={canEdit && taskTypes.length > 0}
					hidden={!taskType && !(canEdit && taskTypes.length > 0)}
				/>

				<FieldRow label="Relationships">
					<button
						type="button"
						className="inline-flex items-center gap-1.5 text-[12px] text-muted-foreground/70 hover:text-foreground transition-colors duration-150 font-medium"
					>
						<Link2 className="size-3.5 opacity-70" />
						<FieldValue empty />
					</button>
				</FieldRow>

				<PropertyField
					label="Assignees"
					mode="user"
					userValue={assigneeUserOption}
					users={memberUserOptions}
					onUserChange={(v) => onUpdate?.({ assignee_id: v })}
					canEdit={canEdit && members.length > 0}
					showUnassigned
				/>

				<PropertyField
					label="Importance"
					mode="number"
					numberValue={task.importance ?? 0}
					onNumberChange={(v) => onUpdate?.({ importance: v })}
					canEdit={canEdit}
				/>

				<PropertyField
					label="Tags"
					mode="tags"
					tags={task.tags ?? []}
					onTagsChange={(t) => onUpdate?.({ tags: t })}
					canEdit={canEdit}
				/>

				<PropertyField
					label="Reporter"
					mode="user"
					userValue={reporterUserOption}
					users={[]}
					canEdit={false}
					hidden={!reporter}
				/>

				<PropertyField
					label="Sprint"
					mode="select"
					value={task.sprint_id ?? "__backlog__"}
					options={sprintOptions}
					onChange={(v) =>
						onUpdate?.({
							sprint_id: v === "__backlog__" ? null : typeof v === "string" ? v : null,
						})
					}
					canEdit={canEdit && sprints.length > 0}
					hidden={!task.sprint_id && !(canEdit && sprints.length > 0)}
				/>

				<PropertyField
					label="Parent task"
					mode="link"
					linkValue={task.parent_task_id ?? ""}
					linkIcon={<ArrowRight className="size-3 shrink-0" />}
					hidden={!task.parent_task_id}
				/>

				{localCustomFields.map((cf) => (
					<PropertyField
						key={cf.id}
						label={cf.display_name}
						mode="custom"
						customType={cf.field_type}
						customRawValue={task.custom_fields?.[cf.field_key]}
						onCustomChange={(v) => {
							onUpdate?.({
								custom_fields: {
									...task.custom_fields,
									[cf.field_key]: v,
								},
							} as unknown as UpdatePayload);
						}}
						customOptions={cf.options}
						canEdit={canEdit}
					/>
				))}
			</div>

			{canEdit && (
				<button
					type="button"
					onClick={() => setAddFieldOpen(true)}
					className="mt-3 flex items-center gap-2 text-[12px] text-muted-foreground/60 hover:text-muted-foreground transition-colors duration-150 font-medium"
				>
					<Plus className="size-3.5" />
					Add fields
				</button>
			)}

			<AddFieldDialog
				open={addFieldOpen}
				onOpenChange={setAddFieldOpen}
				onAdd={(field) => setLocalCustomFields((prev) => [...prev, field])}
			/>
		</>
	);
}
