import { ChevronDown, ChevronRight, Play, Plus } from "lucide-react";
import { useEffect, useState } from "react";

import type { Sprint, Task } from "@/lib/interaction-api";
import type {
	CustomFieldDefinition,
	ProjectMember,
	TaskStatus,
	TaskType,
} from "@/lib/project-api";
import { cn } from "@/lib/utils";

import { AddTaskRow } from "./add-task-row";
import { StartSprintModal } from "./start-sprint-modal";
import { TaskRow, getRowColConfig } from "./task-row";
import {
	type ColumnGroupDef,
	type TaskFieldUpdate,
	buildColumnDropUpdate,
	computeFieldSum,
	getTaskSwimlaneKey,
} from "./view-utils";

// ── Props ────────────────────────────────────────────────────────────────────

export interface ListGroupProps {
	groupDef: ColumnGroupDef;
	tasks: Task[];
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members: ProjectMember[];
	customFields: CustomFieldDefinition[];
	canCreate: boolean;
	defaultCollapsed?: boolean;
	fieldSum?: string;
	swimlaneDefs: ColumnGroupDef[];
	swimlaneBy: string | undefined;
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
	isStatusGrouping: boolean;
	sortBy?: string;
	onUpdateTaskField?: (taskId: string, update: TaskFieldUpdate) => void;
	visibleFields: string[];
	taskIdPrefix?: string;
	/** Sprint data when column_by === "sprint" */
	sprint?: Sprint;
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
	/** Extra fields merged into the create-task payload (e.g. sprint_id) */
	extraCreateFields?: TaskFieldUpdate;
	columnBy?: string;
}

// ── Component ────────────────────────────────────────────────────────────────

export function ListGroup({
	groupDef,
	tasks,
	statuses,
	taskTypes,
	members,
	customFields,
	canCreate,
	defaultCollapsed,
	fieldSum,
	swimlaneDefs,
	swimlaneBy,
	onCreateTask,
	onTaskClick,
	manualSort,
	onReorderTask,
	onStatusChange,
	canEdit,
	isStatusGrouping,
	onUpdateTaskField,
	visibleFields,
	taskIdPrefix = "",
	sprint,
	onStartSprint,
	onCreateSprint,
	extraCreateFields,
	columnBy,
}: ListGroupProps) {
	const [collapsed, setCollapsed] = useState(defaultCollapsed ?? false);
	const [draggingId, setDraggingId] = useState<string | null>(null);
	const [dragOverId, setDragOverId] = useState<string | null>(null);
	const [dragOverSwimKey, setDragOverSwimKey] = useState<string | null>(null);
	const [isDropTarget, setIsDropTarget] = useState(false);
	const [orderedTasks, setOrderedTasks] = useState<Task[]>(tasks);
	const [startSprintOpen, setStartSprintOpen] = useState(false);

	useEffect(() => {
		setOrderedTasks(tasks);
	}, [tasks]);

	const isDraggable = !!(canEdit || manualSort);
	const sumValue = computeFieldSum(tasks, fieldSum, customFields);
	const getViewCtxTasks = () => orderedTasks;

	// Builds onCreateTask arguments for either grouping mode
	const handleAdd = (title: string, typeId: string | null) => {
		if (isStatusGrouping) {
			return onCreateTask(groupDef.key as string, title, typeId);
		}
		const defaultStatus = statuses.find((s) => s.category !== "done") ?? statuses[0];
		return onCreateTask(defaultStatus?.id ?? "", title, typeId, extraCreateFields);
	};

	// ── Drag helpers ─────────────────────────────────────────────────────────

	const handleIntraGroupDrop = (
		e: React.DragEvent,
		targetTask: Task,
		targetIndex: number,
	) => {
		e.preventDefault();
		e.stopPropagation();
		const taskId = e.dataTransfer.getData("text/plain");
		const sourceGroupKey = e.dataTransfer.getData("application/x-source-group-key");

		if (sourceGroupKey && sourceGroupKey !== groupDef.key) {
			if (canEdit) {
				if (isStatusGrouping) {
					onStatusChange?.(taskId, groupDef.key as string);
				} else {
					const colUpdate = buildColumnDropUpdate(columnBy, groupDef.fieldValue, customFields);
					if (Object.keys(colUpdate).length > 0)
						onUpdateTaskField?.(taskId, colUpdate);
				}
			}
			setDraggingId(null);
			setDragOverId(null);
			setIsDropTarget(false);
			return;
		}

		if (!manualSort) {
			setDraggingId(null);
			setDragOverId(null);
			setIsDropTarget(false);
			return;
		}

		const currentDraggingId = draggingId;
		if (!currentDraggingId || currentDraggingId === targetTask.id) return;
		const sourceIndex = orderedTasks.findIndex((t) => t.id === currentDraggingId);
		if (sourceIndex === -1) return;
		const updated = [...orderedTasks];
		const [moved] = updated.splice(sourceIndex, 1);
		updated.splice(targetIndex, 0, moved);
		setOrderedTasks(updated);
		onReorderTask?.(groupDef.key, currentDraggingId, targetIndex);
		setDraggingId(null);
		setDragOverId(null);
	};

	const handleGroupDragOver = (e: React.DragEvent) => {
		if (!isDraggable) return;
		e.preventDefault();
		e.dataTransfer.dropEffect = "move";
		setIsDropTarget(true);
	};

	const handleGroupDrop = (e: React.DragEvent) => {
		e.preventDefault();
		const taskId = e.dataTransfer.getData("text/plain");
		const sourceGroupKey = e.dataTransfer.getData("application/x-source-group-key");
		setIsDropTarget(false);
		setDraggingId(null);
		setDragOverId(null);
		if (!taskId || !canEdit) return;
		if (sourceGroupKey && sourceGroupKey !== groupDef.key) {
			if (isStatusGrouping) {
				onStatusChange?.(taskId, groupDef.key as string);
			} else {
				const colUpdate = buildColumnDropUpdate(columnBy, groupDef.fieldValue, customFields);
				if (Object.keys(colUpdate).length > 0) onUpdateTaskField?.(taskId, colUpdate);
			}
		}
	};

	const hasSwimlanes = swimlaneBy && swimlaneBy !== "none";

	// ── Column-header row ─────────────────────────────────────────────────────

	const columnHeaders = (
		<div className="flex items-center gap-3 px-4 py-1.5 bg-muted/20 border-b border-border/25">
			{isDraggable && <div className="w-3 shrink-0" />}
			<div className="w-20 shrink-0 text-[10px] font-bold uppercase tracking-[0.08em] text-muted-foreground/60">
				ID
			</div>
			<div className="flex-1 text-[10px] font-bold uppercase tracking-[0.08em] text-muted-foreground/60">
				Title
			</div>
			{visibleFields.map((fk) => {
				const col = getRowColConfig(fk, customFields);
				return (
					<div
						key={fk}
						className={cn(
							col.className,
							"text-[10px] font-bold uppercase tracking-[0.08em] text-muted-foreground/60",
							col.responsive ? "hidden sm:block" : "",
						)}
					>
						{col.headerLabel}
					</div>
				);
			})}
		</div>
	);

	// ── Task row renderer ─────────────────────────────────────────────────────

	const renderTaskRow = (task: Task, index: number, groupKey: string, swimKey?: string) => (
		// biome-ignore lint/a11y/noStaticElementInteractions: drag-and-drop row slot
		<div
			key={task.id}
			className={cn(
				"relative",
				manualSort &&
					dragOverId === task.id &&
					draggingId !== task.id &&
					"border-t-2 border-primary/60",
			)}
			draggable={isDraggable}
			onDragStart={(e) => {
				e.dataTransfer.effectAllowed = "move";
				e.dataTransfer.setData("text/plain", task.id);
				e.dataTransfer.setData("application/x-paca-task-id", task.id);
				e.dataTransfer.setData("application/x-source-group-key", groupKey);
				if (swimKey) {
					e.dataTransfer.setData("application/x-source-swim-key", swimKey);
				}
				setDraggingId(task.id);
			}}
			onDragEnd={() => {
				setDraggingId(null);
				setDragOverId(null);
				setIsDropTarget(false);
			}}
			onDragOver={(e) => {
				e.preventDefault();
				if (manualSort) setDragOverId(task.id);
			}}
			onDrop={
				swimKey
					? (e) => {
							e.preventDefault();
							e.stopPropagation();
							const taskId = e.dataTransfer.getData("text/plain");
							const sourceGroupKey = e.dataTransfer.getData(
								"application/x-source-group-key",
							);
							const sourceSwimKey = e.dataTransfer.getData(
								"application/x-source-swim-key",
							);
							setDragOverSwimKey(null);
							// Cross-band drop
							if (swimKey !== "__all" && sourceSwimKey && sourceSwimKey !== swimKey) {
								setDraggingId(null);
								setDragOverId(null);
								if (canEdit) {
									const swimUpdate = buildColumnDropUpdate(
										swimlaneBy,
										swimlaneDefs.find((s) => s.key === swimKey)?.fieldValue,
										customFields,
									);
									if (Object.keys(swimUpdate).length > 0)
										onUpdateTaskField?.(taskId, swimUpdate);
								}
								return;
							}
							if (sourceGroupKey && sourceGroupKey !== groupDef.key) {
								if (canEdit) {
									if (isStatusGrouping) {
										onStatusChange?.(taskId, groupDef.key as string);
									} else {
										const colUpdate = buildColumnDropUpdate(
											columnBy,
											groupDef.fieldValue,
											customFields,
										);
										if (Object.keys(colUpdate).length > 0)
											onUpdateTaskField?.(taskId, colUpdate);
									}
								}
								setDraggingId(null);
								setDragOverId(null);
								setIsDropTarget(false);
								return;
							}
							if (!manualSort) {
								setDraggingId(null);
								setDragOverId(null);
								setIsDropTarget(false);
								return;
							}
							const currentDraggingId = draggingId;
							if (!currentDraggingId || currentDraggingId === task.id) return;
							const sourceOrderedIndex = orderedTasks.findIndex(
								(t) => t.id === currentDraggingId,
							);
							const targetOrderedIndex = orderedTasks.findIndex(
								(t) => t.id === task.id,
							);
							if (sourceOrderedIndex === -1 || targetOrderedIndex === -1) return;
							const updated = [...orderedTasks];
							const [moved] = updated.splice(sourceOrderedIndex, 1);
							updated.splice(targetOrderedIndex, 0, moved);
							setOrderedTasks(updated);
							onReorderTask?.(groupDef.key, currentDraggingId, targetOrderedIndex);
							setDraggingId(null);
							setDragOverId(null);
						}
					: (e) => handleIntraGroupDrop(e, task, index)
			}
		>
			<TaskRow
				task={task}
				taskIdPrefix={taskIdPrefix}
				statuses={statuses}
				taskTypes={taskTypes}
				members={members}
				customFields={customFields}
				visibleFields={visibleFields}
				onClick={() => onTaskClick(task)}
				showDragHandle={isDraggable}
				isDragging={draggingId === task.id}
				canEdit={canEdit}
				onUpdateTaskField={onUpdateTaskField}
			/>
		</div>
	);

	const showAddTask =
		canCreate && groupDef.key !== "__none" && (isStatusGrouping || !!extraCreateFields);

	// ── Render ────────────────────────────────────────────────────────────────

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: drag-and-drop group
		<div
			className={cn(
				"border-b border-border/25 last:border-0 transition-all duration-150",
				isDropTarget && "bg-primary/5 ring-inset ring-2 ring-primary/20",
			)}
			onDragOver={handleGroupDragOver}
			onDragLeave={() => setIsDropTarget(false)}
			onDrop={handleGroupDrop}
		>
			{/* Group header */}
			{/* biome-ignore lint/a11y/noStaticElementInteractions: composite group header with action buttons */}
			<div
				onClick={() => setCollapsed((v) => !v)}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.key === " ") setCollapsed((v) => !v);
				}}
				role="button"
				tabIndex={0}
				className="flex w-full items-center gap-2.5 px-4 py-3 hover:bg-muted/30 transition-colors duration-150 cursor-pointer"
			>
				{collapsed ? (
					<ChevronRight className="size-3.5 text-muted-foreground/60 shrink-0" />
				) : (
					<ChevronDown className="size-3.5 text-muted-foreground/60 shrink-0" />
				)}
				{groupDef.color && (
					<span
						className="size-1.75 rounded-full shrink-0"
						style={{
							background: groupDef.color,
							boxShadow: `0 0 6px ${groupDef.color}40`,
						}}
					/>
				)}
				<span className="text-[11px] font-bold uppercase tracking-[0.08em] text-foreground/80 flex-1 text-left truncate">
					{groupDef.label}
				</span>

				{/* Sprint: "Start sprint" button */}
				{sprint && sprint.status === "planned" && onStartSprint && (
					// biome-ignore lint/a11y/noStaticElementInteractions: stop click propagation
					<button
						type="button"
						onClick={(e) => {
							e.stopPropagation();
							setStartSprintOpen(true);
						}}
						className="flex items-center gap-1.5 rounded-md bg-emerald-500 px-2.5 py-1 text-[11px] font-semibold text-white shadow-sm hover:bg-emerald-600 active:scale-95 transition-all duration-150 shrink-0"
					>
						<Play className="size-3 fill-white" />
						Start sprint
					</button>
				)}

				{/* Backlog: "New sprint" button */}
				{groupDef.key === "__backlog" && onCreateSprint && (
					// biome-ignore lint/a11y/noStaticElementInteractions: stop click propagation
					<button
						type="button"
						onClick={(e) => {
							e.stopPropagation();
							onCreateSprint();
						}}
						className="flex items-center gap-1.5 rounded-md border border-dashed border-primary/40 bg-primary/5 px-2.5 py-1 text-[11px] font-semibold text-primary hover:bg-primary/10 hover:border-primary/60 active:scale-95 transition-all duration-150 shrink-0"
					>
						<Plus className="size-3" />
						New sprint
					</button>
				)}

				{/* Task count / field sum badge */}
				<span className="rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
					{fieldSum && fieldSum !== "count" ? `${sumValue}` : tasks.length}
				</span>
			</div>

			{/* Start Sprint modal */}
			{sprint && onStartSprint && (
				<StartSprintModal
					sprint={sprint}
					open={startSprintOpen}
					onOpenChange={setStartSprintOpen}
					onSubmit={onStartSprint}
				/>
			)}

			{!collapsed && (
				<>
					{hasSwimlanes ? (
						<>
							{/* Swimlane bands */}
							{swimlaneDefs.map((swimDef) => {
								const viewCtxForSwim = { statuses, taskTypes, members, customFields };
								const laneTasks =
									swimDef.key === "__all"
										? getViewCtxTasks()
										: getViewCtxTasks().filter(
												(t) =>
													getTaskSwimlaneKey(t, swimlaneBy, viewCtxForSwim) ===
													swimDef.key,
											);

								const handleSwimBandDragOver = (e: React.DragEvent) => {
									if (!isDraggable || swimDef.key === "__all") return;
									e.preventDefault();
									e.dataTransfer.dropEffect = "move";
									setDragOverSwimKey(swimDef.key);
								};

								const handleSwimBandDrop = (e: React.DragEvent) => {
									e.preventDefault();
									const taskId = e.dataTransfer.getData("text/plain");
									const sourceSwimKey = e.dataTransfer.getData(
										"application/x-source-swim-key",
									);
									setDragOverSwimKey(null);
									setDraggingId(null);
									setDragOverId(null);
									if (!taskId || !canEdit || swimDef.key === "__all") return;
									if (sourceSwimKey && sourceSwimKey !== swimDef.key) {
										const swimUpdate = buildColumnDropUpdate(
											swimlaneBy,
											swimDef.fieldValue,
											customFields,
										);
										if (Object.keys(swimUpdate).length > 0)
											onUpdateTaskField?.(taskId, swimUpdate);
									}
								};

								return (
									// biome-ignore lint/a11y/noStaticElementInteractions: drag-and-drop swimlane band
									<div
										key={swimDef.key}
										className={cn(
											"border-t border-border/15",
											dragOverSwimKey === swimDef.key &&
												isDraggable &&
												"bg-primary/5 ring-inset ring-1 ring-primary/20",
										)}
										onDragOver={handleSwimBandDragOver}
										onDragLeave={(e) => {
											if (!e.currentTarget.contains(e.relatedTarget as Node)) {
												setDragOverSwimKey(null);
											}
										}}
										onDrop={handleSwimBandDrop}
									>
										{swimDef.key !== "__all" && (
											<div className="flex items-center gap-2 px-8 py-1.5 bg-muted/10">
												<span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/50">
													{swimDef.label}
												</span>
											</div>
										)}
										{/* Column headers shown once for the first band */}
										{swimDef.key === (swimlaneDefs[0]?.key ?? "__all") &&
											columnHeaders}
										{laneTasks.length === 0 ? (
											<div className="flex flex-col items-center py-5 text-muted-foreground/40">
												<p className="text-[12px] font-medium">No tasks</p>
											</div>
										) : (
											laneTasks.map((task, index) =>
												renderTaskRow(task, index, groupDef.key, swimDef.key),
											)
										)}
									</div>
								);
							})}

							{/* Add task button after all swimlane bands */}
							{showAddTask && (
								<AddTaskRow
									variant="list"
									taskTypes={taskTypes}
									onAdd={handleAdd}
								/>
							)}
						</>
					) : (
						<>
							{columnHeaders}
							{orderedTasks.length === 0 ? (
								<div className="flex flex-col items-center py-8 text-muted-foreground/40">
									<p className="text-[12px] font-medium">No tasks</p>
								</div>
							) : (
								orderedTasks.map((task, index) =>
									renderTaskRow(task, index, groupDef.key),
								)
							)}
							{showAddTask && (
								<AddTaskRow
									variant="list"
									taskTypes={taskTypes}
									onAdd={handleAdd}
								/>
							)}
						</>
					)}
				</>
			)}
		</div>
	);
}
