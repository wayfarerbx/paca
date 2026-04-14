import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import {
	createTask,
	sprintsQueryOptions,
	subtasksQueryOptions,
	taskQueryOptions,
	updateTask,
} from "@/lib/interaction-api";
import { customFieldsQueryOptions, projectQueryOptions } from "@/lib/project-api";
import { cn } from "@/lib/utils";
import { getTaskTypeIconComponent } from "../../task-types/task-type-icons";
import { getPriority } from "../priority";
import { ActivityPane } from "./activity-pane";
import { AttachmentsSection } from "./attachments-section";
import { ChecklistsSection } from "./checklists-section";
import { DescriptionSection } from "./description-section";
import { mapApiFieldToUi } from "./helpers";
import { PropertiesPanel } from "./properties-panel";
import { SubtasksSection } from "./subtasks-section";
import { TaskHeader } from "./task-header";
import type { ActivityEntry, TaskDetailModalProps } from "./types";

// Re-exports for consumers
export type {
	ActivityEntry,
	Attachment,
	Checklist,
	ChecklistItem,
	CustomFieldDef,
	TaskDetailModalProps,
} from "./types";

const TITLE_CLASSES =
	"font-[Syne] text-xl lg:text-[26px] font-bold leading-snug text-foreground tracking-tight w-full";

export function TaskDetailModal({
	task: taskProp,
	open,
	onOpenChange,
	statuses,
	taskTypes,
	members = [],
	projectName,
	interactionName,
	projectId,
	taskIdPrefix: taskIdPrefixProp = "",
	mode = "modal",
	canEdit = true,
}: TaskDetailModalProps) {
	const qc = useQueryClient();

	// Fetch project to get task ID prefix
	const { data: project } = useQuery({
		...projectQueryOptions(projectId ?? ""),
		enabled: !!projectId && (open || mode === "page"),
	});
	const taskIdPrefix = project?.task_id_prefix || taskIdPrefixProp;

	// Fetch fresh task data whenever the modal is open and we have a projectId
	const { data: freshTask } = useQuery({
		...taskQueryOptions(projectId ?? "", taskProp?.id ?? ""),
		enabled: !!projectId && !!taskProp?.id && (open || mode === "page"),
	});

	// Use fresh task if available, fall back to prop
	const task = freshTask ?? taskProp;

	// Fetch subtasks
	const { data: subtasks = [] } = useQuery({
		...subtasksQueryOptions(projectId ?? "", task?.id ?? ""),
		enabled: !!projectId && !!task?.id && (open || mode === "page"),
	});

	// Fetch sprints for sprint name display + assignment
	const { data: sprints = [] } = useQuery({
		...sprintsQueryOptions(projectId ?? ""),
		enabled: !!projectId && (open || mode === "page"),
	});

	// Fetch custom field definitions from API
	const { data: apiCustomFields = [] } = useQuery({
		...customFieldsQueryOptions(projectId ?? ""),
		enabled: !!projectId && (open || mode === "page"),
	});
	const customFieldDefs = useMemo(
		() => apiCustomFields.map(mapApiFieldToUi),
		[apiCustomFields],
	);

	const status = statuses.find((s) => s.id === task?.status_id);
	const taskType = taskTypes.find((t) => t.id === task?.task_type_id);
	const priority = getPriority(task?.importance ?? 0);
	const assignee = members.find((m) => m.id === task?.assignee_id);
	const reporter = members.find((m) => m.id === task?.reporter_id);

	// ── Title inline edit ─────────────────────────────────────────────────────
	const [editingTitle, setEditingTitle] = useState(false);
	const [titleDraft, setTitleDraft] = useState("");
	const titleInputRef = useRef<HTMLTextAreaElement>(null);

	// ── Update mutation ────────────────────────────────────────────────────────
	const updateMutation = useMutation({
		mutationFn: (payload: Parameters<typeof updateTask>[2]) => {
			if (!projectId || !task) throw new Error("missing context");
			return updateTask(projectId, task.id, payload);
		},
		onSuccess: (updated) => {
			if (!projectId) return;
			qc.setQueryData(
				taskQueryOptions(projectId, updated.id).queryKey,
				updated,
			);
			qc.invalidateQueries({
				queryKey: ["projects", projectId],
				predicate: (q) => {
					const key = q.queryKey as string[];
					return key.includes("tasks") || key.includes("backlog-tasks");
				},
			});
		},
	});

	const handleUpdate = canEdit ? updateMutation.mutate : undefined;

	// Mock activity entries
	const activities: ActivityEntry[] = task
		? [
				{
					id: "1",
					type: "created",
					author: reporter?.full_name || reporter?.username || "System",
					content: "created this task",
					timestamp: task.created_at,
				},
				...(task.assignee_id
					? [
							{
								id: "2",
								type: "assignee_change" as const,
								author: reporter?.full_name || reporter?.username || "System",
								content: `assigned this to ${assignee?.full_name || assignee?.username || "a member"}`,
								timestamp: task.updated_at,
							},
						]
					: []),
			]
		: [];

	// Close on Escape (modal mode only)
	useEffect(() => {
		if (!open || mode === "page") return;
		const handler = (e: KeyboardEvent) => {
			if (e.key === "Escape") onOpenChange(false);
		};
		document.addEventListener("keydown", handler);
		return () => document.removeEventListener("keydown", handler);
	}, [open, mode, onOpenChange]);

	if (mode === "modal" && !open) return null;

	// Resolve task type icon component
	const TypeIcon = taskType ? getTaskTypeIconComponent(taskType.icon) : null;

	// ── Content ────────────────────────────────────────────────────────────────
	const content = task ? (
		<div className="flex h-full flex-col">
			<TaskHeader
				task={task}
				mode={mode}
				projectName={projectName}
				interactionName={interactionName}
				projectId={projectId}
				taskIdPrefix={taskIdPrefix}
				onClose={() => onOpenChange(false)}
			/>

			{/* ── Body: stacks on mobile, side-by-side on lg+ ── */}
			<div className="flex flex-col lg:flex-row flex-1 min-w-0 overflow-y-auto lg:overflow-hidden">
				{/* Main content area: no own scroll on mobile (body scrolls), scrollable on lg+ */}
				<div className="lg:flex-1 lg:overflow-y-auto [scrollbar-gutter:stable] [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-border/60 [&::-webkit-scrollbar-thumb]:hover:bg-border">
					<div className="px-4 lg:px-8 py-5 lg:py-7 space-y-6 lg:space-y-8 max-w-3xl mx-auto">
						{/* Type badge + Status chip + Title */}
						<div className="space-y-4">
							<div className="flex items-center gap-2.5 flex-wrap">
								{taskType && (
									<span
										className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[11px] font-bold leading-tight tracking-wide border"
										style={{
											borderColor: taskType.color
												? `${taskType.color}44`
												: "var(--border)",
											backgroundColor: taskType.color
												? `${taskType.color}15`
												: "var(--muted)",
											color: taskType.color ?? "inherit",
										}}
									>
										{TypeIcon && <TypeIcon className="size-3.5 opacity-70" />}
										{taskType.name}
									</span>
								)}
								{status && (
									<span className="inline-flex items-center gap-2 rounded-full border border-border/40 bg-muted/40 px-3 py-1 text-[11px] font-semibold text-muted-foreground tracking-wide backdrop-blur-sm">
										<span
											className="size-1.75 rounded-full shrink-0 ring-2 ring-offset-1 ring-offset-background"
											style={{
												background: status.color ?? "var(--muted-foreground)",
												boxShadow: `0 0 6px ${status.color ?? "var(--muted-foreground)"}40`,
											}}
										/>
										{status.name}
									</span>
								)}
							</div>

							{editingTitle ? (
								<textarea
									ref={titleInputRef}
									value={titleDraft}
									onChange={(e) => setTitleDraft(e.target.value)}
									onBlur={() => {
										setEditingTitle(false);
										const trimmed = titleDraft.trim();
										if (trimmed && trimmed !== task.title)
											handleUpdate?.({ title: trimmed });
									}}
									onKeyDown={(e) => {
										if (e.key === "Enter" && !e.shiftKey) {
											e.preventDefault();
											e.currentTarget.blur();
										}
										if (e.key === "Escape") {
											setEditingTitle(false);
											setTitleDraft(task.title);
										}
									}}
									rows={1}
									className={cn(
										TITLE_CLASSES,
										"resize-none bg-transparent outline-none py-0",
									)}
									data-testid="task-title-input"
								/>
							) : (
								// biome-ignore lint/a11y/useKeyWithClickEvents: inline title click-to-edit
								<h1
									className={cn(
										TITLE_CLASSES,
										canEdit &&
											"cursor-text hover:bg-muted/15 rounded-md px-2 -ml-2 py-1 transition-all duration-150",
									)}
									data-testid="task-title"
									onClick={() => {
										if (!canEdit) return;
										setTitleDraft(task.title);
										setEditingTitle(true);
										setTimeout(() => titleInputRef.current?.focus(), 0);
									}}
								>
									{task.title}
								</h1>
							)}
						</div>

						{/* Properties */}
						<div>
							<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-3 flex items-center gap-2">
								<span>Properties</span>
								<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
							</h3>
							<PropertiesPanel
								task={task}
								status={status}
								taskType={taskType}
								priority={priority}
								assignee={assignee}
								reporter={reporter}
								statuses={statuses}
								taskTypes={taskTypes}
								members={members}
								sprints={sprints}
								projectId={projectId}
								initialCustomFields={customFieldDefs}
								canEdit={canEdit}
								onUpdate={handleUpdate}
							/>
						</div>

						{/* Description */}
						<DescriptionSection
							description={task.description}
							canEdit={canEdit}
							onUpdate={handleUpdate}
						/>

						{/* Subtasks */}
						<SubtasksSection
							projectId={projectId}
							parentTaskId={task.id}
							subtasks={subtasks}
							statuses={statuses}
							taskTypes={taskTypes}
							members={members}
							canEdit={canEdit}
							task={task}
							onSubtaskUpdate={(subtaskId, payload) => {
								if (!projectId) return;
								updateTask(projectId, subtaskId, payload).then(() => {
									qc.invalidateQueries({
										queryKey: subtasksQueryOptions(projectId, task.id).queryKey,
									});
								});
							}}
							onSubtaskCreate={(payload) => {
								if (!projectId) return;
								createTask(projectId, {
									...payload,
									parent_task_id: task.id,
								}).then(() => {
									qc.invalidateQueries({
										queryKey: subtasksQueryOptions(projectId, task.id).queryKey,
									});
								});
							}}
						/>

						{/* Checklists */}
						<ChecklistsSection />

						{/* Attachments */}
						<AttachmentsSection canEdit={canEdit} />

						{/* Bottom breathing room */}
						<div className="h-8" />
					</div>
				</div>

				{/* ── Right: Activity pane ── */}
				<ActivityPane activities={activities} />
			</div>
		</div>
	) : (
		<div className="flex h-full items-center justify-center">
			<div className="flex flex-col items-center gap-4 text-muted-foreground/70">
				<div className="flex size-14 items-center justify-center rounded-2xl bg-muted/50">
					<AlertCircle className="size-7 text-muted-foreground/60" />
				</div>
				<div className="text-center">
					<p className="text-base font-semibold text-foreground/80">
						Task not found
					</p>
					<p className="text-sm mt-1.5 text-muted-foreground/70">
						This task may have been deleted or the link is invalid.
					</p>
				</div>
			</div>
		</div>
	);

	// ── Modal wrapper ──────────────────────────────────────────────────────────
	if (mode === "page") {
		return (
			<div className="flex h-full flex-col overflow-hidden bg-background">
				{content}
			</div>
		);
	}

	return (
		<>
			{/* Backdrop */}
			<div
				className={cn(
					"fixed inset-0 z-50 bg-black/30 backdrop-blur-[3px] transition-opacity duration-200",
					open ? "opacity-100" : "opacity-0 pointer-events-none",
				)}
				onClick={() => onOpenChange(false)}
				aria-hidden="true"
			/>

			{/* Modal panel */}
			<div
				role="dialog"
				aria-modal="true"
				aria-label={task?.title ?? "Task detail"}
				className={cn(
					"fixed z-50",
					"inset-0 lg:left-1/2 lg:top-1/2 lg:-translate-x-1/2 lg:-translate-y-1/2",
					"flex flex-col overflow-hidden",
					"lg:h-[90vh] lg:w-[92vw] lg:max-w-6xl",
					"lg:rounded-2xl lg:border lg:border-border/50 bg-background",
					"lg:shadow-[0_25px_60px_-12px_rgba(0,0,0,0.25),0_0_0_1px_rgba(255,255,255,0.05)_inset]",
					"transition-all duration-200 origin-center",
					open
						? "opacity-100 scale-100"
						: "opacity-0 scale-[0.97] pointer-events-none",
				)}
			>
				{content}
			</div>
		</>
	);
}
