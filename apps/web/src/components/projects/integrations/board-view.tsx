import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ChevronDown, Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { getTaskTypeIconComponent } from "@/components/projects/task-types/task-type-icons";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { type Task, updateTask } from "@/lib/integration-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";
import { cn } from "@/lib/utils";

import { TaskCard } from "./task-card";

interface BoardViewProps {
	projectId: string;
	tasks: Task[];
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members: ProjectMember[];
	canCreate: boolean;
	canEdit: boolean;
	searchQuery: string;
	assigneeFilter: string | null;
	tasksQueryKey: unknown[];
	onCreateTask: (statusId: string, title: string, taskTypeId?: string | null) => Promise<void>;
	onTaskClick: (task: Task) => void;
	onUpdateTask?: (taskId: string, payload: Partial<{ task_type_id: string | null; assignee_id: string | null }>) => void;
	manualSort?: boolean;
	onReorderTask?: (statusId: string, taskId: string, newIndex: number) => void;
}

interface ColumnAddProps {
	taskTypes: TaskType[];
	onAdd: (title: string, taskTypeId: string | null) => void;
}

function ColumnAddTask({ taskTypes, onAdd }: ColumnAddProps) {
	const [open, setOpen] = useState(false);
	const [value, setValue] = useState("");
	const [selectedTypeId, setSelectedTypeId] = useState<string | null>(null);
	const inputRef = useRef<HTMLInputElement>(null);

	const defaultType = taskTypes.find((tt) => tt.is_default) ?? taskTypes[0] ?? null;
	const selectedType = taskTypes.find((tt) => tt.id === selectedTypeId) ?? defaultType;

	const open_ = () => {
		setOpen(true);
		setTimeout(() => inputRef.current?.focus(), 0);
	};

	const submit = () => {
		const title = value.trim();
		if (!title) return;
		onAdd(title, selectedType?.id ?? null);
		setValue("");
		setSelectedTypeId(null);
		setOpen(false);
	};

	const cancel = () => {
		setValue("");
		setSelectedTypeId(null);
		setOpen(false);
	};

	if (!open) {
		return (
			<button
				type="button"
				onClick={open_}
				className="flex w-full items-center gap-1.5 rounded-lg bg-primary/8 text-primary/80 hover:bg-primary/15 hover:text-primary px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
			>
				<Plus className="size-3" />
				Add task
			</button>
		);
	}

	const SelectedIcon = getTaskTypeIconComponent(selectedType?.icon ?? null);

	return (
		<div className="rounded-xl border border-border/30 bg-card/50 p-2.5 shadow-sm">
			<div className="flex items-center gap-1.5 mb-2">
				{taskTypes.length > 0 && selectedType && (
					<DropdownMenu>
						<DropdownMenuTrigger
							className="flex items-center gap-1 rounded-lg px-1.5 py-0.5 text-[11px] font-semibold transition-all duration-150 hover:bg-muted/60 shrink-0"
							style={selectedType.color ? { color: selectedType.color } : undefined}
						>
							{SelectedIcon ? (
								<SelectedIcon className="size-3.5 opacity-70" />
							) : (
								<span className="text-[10px] font-bold">{selectedType.name.slice(0, 2)}</span>
							)}
							<ChevronDown className="size-3 text-muted-foreground/60" />
						</DropdownMenuTrigger>
						<DropdownMenuContent align="start" className="w-40 rounded-xl border border-border/40 shadow-lg p-1">
							{taskTypes.map((tt) => {
								const Icon = getTaskTypeIconComponent(tt.icon);
								return (
									<DropdownMenuItem
										key={tt.id}
										onClick={() => setSelectedTypeId(tt.id)}
										className={cn(
											"flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] hover:bg-muted/60 transition-colors duration-100",
											selectedType.id === tt.id && "bg-muted/40",
										)}
									>
										{Icon ? (
											<Icon
												className="size-3.5 shrink-0 text-muted-foreground/80"
												style={tt.color ? { color: tt.color } : undefined}
											/>
										) : (
											<span className="size-3.5 shrink-0 text-[10px] font-bold">
												{tt.name.slice(0, 2)}
											</span>
										)}
										{tt.name}
									</DropdownMenuItem>
								);
							})}
						</DropdownMenuContent>
					</DropdownMenu>
				)}
				<span className="text-[11px] text-muted-foreground/60 truncate">
					{selectedType?.name ?? "Task"}
				</span>
			</div>
			<input
				ref={inputRef}
				value={value}
				onChange={(e) => setValue(e.target.value)}
				onKeyDown={(e) => {
					if (e.key === "Enter") submit();
					if (e.key === "Escape") cancel();
				}}
				placeholder="Task title…"
				className="w-full rounded-lg border border-border/30 bg-muted/15 px-3 py-2 text-[13px] font-medium outline-none placeholder:text-muted-foreground/50 focus:border-primary/40 focus:ring-2 focus:ring-primary/15 transition-all duration-150"
			/>
			<div className="mt-2 flex items-center gap-1.5 justify-end">
				<button
					type="button"
					onClick={cancel}
					className="flex items-center gap-1.5 rounded-lg bg-muted/40 text-muted-foreground/80 hover:bg-muted/60 hover:text-foreground px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
				>
					Cancel
				</button>
				<button
					type="button"
					onClick={submit}
					disabled={!value.trim()}
					className="rounded-lg bg-primary px-3 py-1.5 text-[11px] font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm disabled:opacity-40 transition-all duration-150"
				>
					Create
				</button>
			</div>
		</div>
	);
}

export function BoardView({
	projectId,
	tasks,
	statuses,
	taskTypes,
	members,
	canCreate,
	canEdit,
	searchQuery,
	assigneeFilter,
	tasksQueryKey,
	onCreateTask,
	onTaskClick,
	onUpdateTask,
	manualSort,
	onReorderTask,
}: BoardViewProps) {
	const qc = useQueryClient();
	const [draggingId, setDraggingId] = useState<string | null>(null);
	const [overStatusId, setOverStatusId] = useState<string | null>(null);
	const [overCardId, setOverCardId] = useState<string | null>(null);
	const [columnOrderMap, setColumnOrderMap] = useState<
		Record<string, string[]>
	>({});
	// biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset local order whenever the task list is refreshed from the server
	useEffect(() => {
		setColumnOrderMap({});
	}, [tasks]);

	const updateMutation = useMutation({
		mutationFn: ({
			taskId,
			statusId,
			sprintId,
		}: {
			taskId: string;
			statusId: string;
			sprintId: string | null | undefined;
		}) =>
			updateTask(projectId, taskId, {
				status_id: statusId,
				sprint_id: sprintId ?? null,
			}),
		onSuccess: () => qc.invalidateQueries({ queryKey: tasksQueryKey }),
	});

	const filteredTasks = tasks.filter((t) => {
		if (
			searchQuery &&
			!t.title.toLowerCase().includes(searchQuery.toLowerCase())
		)
			return false;
		if (assigneeFilter && t.assignee_id !== assigneeFilter) return false;
		return true;
	});

	const tasksByStatus = (statusId: string) => {
		const col = filteredTasks.filter((t) => t.status_id === statusId);
		if (manualSort) return col;
		return col.sort((a, b) => a.created_at.localeCompare(b.created_at));
	};
	const getColumnTasks = (statusId: string): Task[] => {
		const ids = columnOrderMap[statusId];
		if (ids) {
			return ids
				.map((id) => filteredTasks.find((t) => t.id === id))
				.filter((t): t is Task => t !== undefined);
		}
		return tasksByStatus(statusId);
	};

	const unassignedTasks = filteredTasks.filter((t) => !t.status_id);

	const handleDragStart = (e: React.DragEvent, taskId: string) => {
		if (!canEdit) return;
		setDraggingId(taskId);
		e.dataTransfer.effectAllowed = "move";
		e.dataTransfer.setData("text/plain", taskId);
		e.dataTransfer.setData("application/x-paca-task-id", taskId);
	};

	const handleDragEnd = () => {
		setDraggingId(null);
		setOverStatusId(null);
		setOverCardId(null);
	};

	const handleDrop = (e: React.DragEvent, statusId: string) => {
		e.preventDefault();
		const taskId = e.dataTransfer.getData("text/plain");
		if (!taskId || !canEdit) return;
		const task = tasks.find((t) => t.id === taskId);
		if (task && task.status_id !== statusId) {
			updateMutation.mutate({ taskId, statusId, sprintId: task.sprint_id });
		}
		setDraggingId(null);
		setOverStatusId(null);
		setOverCardId(null);
	};

	const handleDropOnCard = (
		e: React.DragEvent,
		targetStatusId: string,
		targetTaskId: string,
		targetIndex: number,
	) => {
		e.preventDefault();
		e.stopPropagation();
		const taskId = e.dataTransfer.getData("text/plain");
		if (!taskId || !canEdit) {
			setDraggingId(null);
			setOverCardId(null);
			return;
		}
		const task = tasks.find((t) => t.id === taskId);
		if (!task) {
			setDraggingId(null);
			setOverCardId(null);
			return;
		}
		if (task.status_id !== targetStatusId) {
			updateMutation.mutate({
				taskId,
				statusId: targetStatusId,
				sprintId: task.sprint_id,
			});
		} else if (manualSort && taskId !== targetTaskId) {
			const current = getColumnTasks(targetStatusId);
			const srcIdx = current.findIndex((t) => t.id === taskId);
			if (srcIdx !== -1) {
				const next = [...current];
				const [moved] = next.splice(srcIdx, 1);
				next.splice(targetIndex, 0, moved);
				setColumnOrderMap((prev) => ({
					...prev,
					[targetStatusId]: next.map((t) => t.id),
				}));
			}
			onReorderTask?.(targetStatusId, taskId, targetIndex);
		}
		setDraggingId(null);
		setOverStatusId(null);
		setOverCardId(null);
	};

	const handleDragOver = (e: React.DragEvent, statusId: string) => {
		e.preventDefault();
		e.dataTransfer.dropEffect = "move";
		setOverStatusId(statusId);
	};

	const sortedStatuses = [...statuses].sort((a, b) => a.position - b.position);

	return (
		<div className="flex flex-1 min-h-0 gap-4 overflow-x-auto px-6 py-5 pb-8">
			{sortedStatuses.map((status) => {
				const columnTasks = getColumnTasks(status.id);
				const isOver = overStatusId === status.id;

				return (
					// biome-ignore lint/a11y/noStaticElementInteractions: drag-and-drop column requires pointer events; keyboard reorder is handled separately
					<div
						key={status.id}
						data-status-id={status.id}
						className="flex w-72 shrink-0 flex-col gap-2.5"
						onDragOver={(e) => handleDragOver(e, status.id)}
						onDrop={(e) => handleDrop(e, status.id)}
					>
						{/* Column header */}
						<div className="flex items-center gap-2 px-2">
							<span
								className="size-1.75 rounded-full shrink-0"
								style={{
									background: status.color ?? "oklch(var(--muted-foreground))",
									boxShadow: status.color ? `0 0 6px ${status.color}40` : undefined,
								}}
							/>
							<span className="text-[11px] font-bold text-foreground/80 tracking-[0.08em] uppercase">
								{status.name}
							</span>
							<span className="ml-auto rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
								{columnTasks.length}
							</span>
						</div>

						{/* Drop zone */}
						<div
							className={cn(
								"flex flex-col gap-2 rounded-xl p-2 min-h-30 transition-all duration-200",
								isOver
									? "bg-primary/8 ring-2 ring-primary/20"
									: "bg-muted/40 dark:bg-black/30",
							)}
						>
							{columnTasks.length === 0 && (
								<div className="flex flex-1 flex-col items-center justify-center py-8 text-muted-foreground/40">
									<p className="text-[12px] font-medium">No tasks</p>
								</div>
							)}

							{columnTasks.map((task, index) => (
								// biome-ignore lint/a11y/noStaticElementInteractions: drag-and-drop card slot; pointer events only
								<div
									key={task.id}
									className={cn(
										"relative",
										manualSort &&
											overCardId === task.id &&
											draggingId !== task.id &&
											"border-t-2 border-primary/60",
									)}
									onDragOver={(e) => {
										e.preventDefault();
										e.stopPropagation();
										setOverStatusId(status.id);
										if (manualSort) setOverCardId(task.id);
									}}
									onDrop={(e) => handleDropOnCard(e, status.id, task.id, index)}
								>
									<TaskCard
										task={task}
										statuses={statuses}
										taskTypes={taskTypes}
										members={members}
										canEdit={canEdit}
										isDragging={draggingId === task.id}
										onDragStart={(e) => handleDragStart(e, task.id)}
										onDragEnd={handleDragEnd}
										onClick={() => onTaskClick(task)}
										onUpdate={onUpdateTask}
									/>
								</div>
							))}
							{canCreate && (
								<ColumnAddTask
									taskTypes={taskTypes}
									onAdd={(title, typeId) => onCreateTask(status.id, title, typeId)}
								/>
							)}
						</div>
					</div>
				);
			})}
			{/* Catch-all column for unstatused tasks */}
			{unassignedTasks.length > 0 && (
				<div className="flex w-72 shrink-0 flex-col gap-2.5">
					<div className="flex items-center gap-2 px-2">
						<span className="size-1.75 rounded-full bg-muted-foreground/30 shrink-0" />
						<span className="text-[11px] font-bold text-muted-foreground/50 tracking-[0.08em] uppercase">
							No Status
						</span>
						<span className="ml-auto rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
							{unassignedTasks.length}
						</span>
					</div>
					<div className="flex flex-col gap-2 rounded-xl bg-muted/30 dark:bg-black/30 p-2">
						{unassignedTasks.map((task) => (
							<TaskCard
								key={task.id}
								task={task}
								statuses={statuses}
								taskTypes={taskTypes}
								members={members}
								canEdit={false}
								isDragging={draggingId === task.id}
								onDragStart={(e) => handleDragStart(e, task.id)}
								onDragEnd={handleDragEnd}
								onClick={() => onTaskClick(task)}
							/>
						))}
					</div>
				</div>
			)}
		</div>
	);
}
