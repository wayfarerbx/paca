import { ListChecks, Plus } from "lucide-react";
import { useState } from "react";
import type { Task } from "@/lib/interaction-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";
import { SubtaskRow } from "./subtask-row";

interface SubtasksSectionProps {
	projectId?: string;
	parentTaskId: string;
	subtasks: Task[];
	statuses: TaskStatus[];
	taskTypes?: TaskType[];
	members?: ProjectMember[];
	canEdit?: boolean;
	task: Task;
	taskIdPrefix?: string;
	/** "subtasks" – normal task creating subtasks; "tasks" – epic creating child tasks */
	mode?: "subtasks" | "tasks";
	/** Type to auto-assign when mode="subtasks" */
	subtaskType?: TaskType;
	/** Available types for the type picker when mode="tasks" (epic) */
	normalTaskTypes?: TaskType[];
	onSubtaskUpdate?: (
		subtaskId: string,
		payload: Partial<{
			status_id: string | null;
			task_type_id: string | null;
			assignee_id: string | null;
			importance: number;
		}>,
	) => void;
	onSubtaskCreate?: (payload: {
		title: string;
		status_id?: string | null;
		task_type_id?: string | null;
	}) => void;
	onSubtaskClick?: (task: Task) => void;
}

export function SubtasksSection({
	subtasks,
	statuses,
	taskTypes = [],
	members = [],
	canEdit = true,
	taskIdPrefix = "",
	mode = "subtasks",
	subtaskType,
	normalTaskTypes = [],
	onSubtaskUpdate,
	onSubtaskCreate,
	onSubtaskClick,
}: SubtasksSectionProps) {
	const [adding, setAdding] = useState(false);
	const [newTitle, setNewTitle] = useState("");
	const [selectedTypeId, setSelectedTypeId] = useState<string>(
		() => normalTaskTypes[0]?.id ?? "",
	);

	const sectionLabel = mode === "tasks" ? "Tasks" : "Subtasks";
	const emptyLabel = mode === "tasks" ? "No tasks yet" : "No subtasks yet";
	const placeholder = mode === "tasks" ? "Task title..." : "Subtask title...";

	function handleCreate() {
		const trimmed = newTitle.trim();
		if (!trimmed) return;
		if (mode === "subtasks") {
			onSubtaskCreate?.({
				title: trimmed,
				task_type_id: subtaskType?.id ?? null,
			});
		} else {
			onSubtaskCreate?.({
				title: trimmed,
				task_type_id: selectedTypeId || null,
			});
		}
		setNewTitle("");
		setAdding(false);
	}

	return (
		<div className="space-y-3">
			<div className="flex items-center justify-between">
				<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
					<span>{sectionLabel}</span>
					<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
				</h3>
				{canEdit && (
					<button
						type="button"
						onClick={() => setAdding(true)}
						className="flex items-center gap-1.5 rounded-lg bg-primary/8 text-primary/80 hover:bg-primary/15 hover:text-primary px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
					>
						<Plus className="size-3" />
						Add Task
					</button>
				)}
			</div>

			{subtasks.length > 0 && (
				<div className="rounded-xl border border-border/25 bg-card/50 divide-y divide-border/15 overflow-hidden">
					{subtasks.map((sub) => (
						<SubtaskRow
							key={sub.id}
							task={sub}
							taskIdPrefix={taskIdPrefix}
							statuses={statuses}
							taskTypes={taskTypes}
							members={members}
							showTypeField={mode === "tasks"}
							canEdit={canEdit}
							onUpdate={onSubtaskUpdate}
							onClick={onSubtaskClick ? () => onSubtaskClick(sub) : undefined}
						/>
					))}
				</div>
			)}

			{adding && (
				<form
					className="flex flex-col gap-2"
					onSubmit={(e) => {
						e.preventDefault();
						handleCreate();
					}}
				>
					{mode === "tasks" && normalTaskTypes.length > 0 && (
						<select
							value={selectedTypeId}
							onChange={(e) => setSelectedTypeId(e.target.value)}
							className="rounded-lg border border-border/30 bg-muted/20 px-3 py-2 text-[12px] focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary/40 transition-all duration-150 self-start"
						>
							{normalTaskTypes.map((t) => (
								<option key={t.id} value={t.id}>
									{t.name}
								</option>
							))}
						</select>
					)}
					<div className="flex items-center gap-2">
						<input
							// biome-ignore lint/a11y/noAutofocus: intentional for inline form
							autoFocus
							type="text"
							value={newTitle}
							onChange={(e) => setNewTitle(e.target.value)}
							placeholder={placeholder}
							className="flex-1 rounded-lg border border-border/30 bg-muted/20 px-3 py-2.5 text-[13px] placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary/40 transition-all duration-150"
							onKeyDown={(e) => {
								if (e.key === "Escape") {
									setAdding(false);
									setNewTitle("");
								}
							}}
						/>
						<button
							type="submit"
							className="rounded-lg bg-primary px-3.5 py-2.5 text-[12px] font-semibold text-primary-foreground hover:bg-primary/90 transition-colors duration-150 shadow-sm"
						>
							Add
						</button>
						<button
							type="button"
							onClick={() => {
								setAdding(false);
								setNewTitle("");
							}}
							className="rounded-lg border border-border/30 px-3.5 py-2.5 text-[12px] text-muted-foreground/80 hover:text-foreground hover:bg-muted/30 transition-all duration-150"
						>
							Cancel
						</button>
					</div>
				</form>
			)}

			{!adding && subtasks.length === 0 && (
				<div className="flex items-center gap-3 px-1 py-3 text-muted-foreground/45">
					<ListChecks className="size-4 opacity-70" />
					<p className="text-[13px] italic">{emptyLabel}</p>
				</div>
			)}
		</div>
	);
}
