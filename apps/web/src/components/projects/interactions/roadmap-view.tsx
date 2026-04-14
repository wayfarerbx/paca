import { CalendarDays } from "lucide-react";

import type { Task } from "@/lib/interaction-api";

import { AddTaskRow } from "./add-task-row";
import type { TaskStatus, TaskType } from "@/lib/project-api";
import { cn } from "@/lib/utils";

import { TaskRow } from "./task-row";

interface RoadmapViewProps {
	tasks: Task[];
	taskIdPrefix?: string;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	searchQuery: string;
	assigneeFilter: string | null;
	canCreate?: boolean;
	onCreateTask?: (
		statusId: string,
		title: string,
		taskTypeId?: string | null,
	) => Promise<void>;
	onTaskClick: (task: Task) => void;
}

function getBarStyle(task: Task, minMs: number, rangeMs: number) {
	const created = new Date(task.created_at).getTime();
	const updated = new Date(task.updated_at).getTime();
	const left = Math.max(0, Math.min(100, ((created - minMs) / rangeMs) * 100));
	const right = Math.max(0, Math.min(100, ((updated - minMs) / rangeMs) * 100));
	const width = Math.max(3, right - left);
	return { left: `${left}%`, width: `${width}%` };
}

export function RoadmapView({
	tasks,
	taskIdPrefix = "",
	statuses,
	taskTypes,
	searchQuery,
	assigneeFilter,
	canCreate = false,
	onCreateTask,
	onTaskClick,
}: RoadmapViewProps) {
	const filtered = tasks.filter((t) => {
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
	});

	const sortedStatuses = [...statuses].sort((a, b) => a.position - b.position);
	const defaultStatusId =
		statuses.find((status) => status.category === "backlog")?.id ??
		statuses.find((status) => status.category === "todo")?.id ??
		statuses[0]?.id ??
		"";

	const timestamps = filtered.flatMap((t) => [
		new Date(t.created_at).getTime(),
		new Date(t.updated_at).getTime(),
	]);
	const minMs = timestamps.length
		? Math.min(...timestamps)
		: Date.now() - 30 * 86400_000;
	const maxMs = timestamps.length ? Math.max(...timestamps) : Date.now();
	const rangeMs = Math.max(maxMs - minMs, 7 * 86400_000);

	const startDate = new Date(minMs);
	const endDate = new Date(maxMs + 86400_000 * 3);
	const months: { label: string; left: string }[] = [];
	const cur = new Date(startDate.getFullYear(), startDate.getMonth(), 1);
	while (cur <= endDate) {
		const pos = Math.max(0, ((cur.getTime() - minMs) / rangeMs) * 100);
		months.push({
			label: cur.toLocaleString("default", { month: "short", year: "2-digit" }),
			left: `${pos}%`,
		});
		cur.setMonth(cur.getMonth() + 1);
	}

	const hasVisibleTasks = filtered.length > 0;

	return (
		<div className="flex h-full flex-col overflow-hidden">
			{/* Timeline header */}
			<div className="shrink-0 border-b border-border/25 bg-muted/10">
				<div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/25">
					<CalendarDays className="size-3.5 text-muted-foreground/70" />
					<span className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-[0.08em]">
						Timeline
					</span>
					<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
				</div>
				<div className="relative h-7 overflow-hidden px-4">
					{months.map((m) => (
						<span
							key={m.label}
							className="absolute top-1.5 text-[10px] font-bold text-muted-foreground/50 whitespace-nowrap"
							style={{ left: m.left }}
						>
							{m.label}
						</span>
					))}
				</div>
			</div>

			{/* Task rows with bars */}
			<div className="flex-1 overflow-auto">
				{!hasVisibleTasks ? (
					<div className="flex flex-col items-center py-12 text-muted-foreground/40">
						<CalendarDays className="size-6 mb-2" />
						<p className="text-[12px] font-medium">No tasks to display</p>
					</div>
				) : (
					sortedStatuses.map((status) => {
						const groupTasks = filtered.filter(
							(t) => t.status_id === status.id,
						);
						if (groupTasks.length === 0) return null;

						return (
							<div
								key={status.id}
								className="border-b border-border/25 last:border-0"
							>
								{/* Group header */}
								<div className="flex items-center gap-2 px-4 py-2.5 bg-muted/20 border-b border-border/25">
									<span
										className="size-1.75 rounded-full shrink-0"
										style={{
											background:
												status.color ?? "oklch(var(--muted-foreground))",
											boxShadow: status.color
												? `0 0 6px ${status.color}40`
												: undefined,
										}}
									/>
									<span className="text-[11px] font-bold uppercase tracking-[0.08em] text-foreground/80">
										{status.name}
									</span>
									<span className="rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
										{groupTasks.length}
									</span>
								</div>

								{/* Rows: task info + timeline bar */}
								{groupTasks.map((task) => {
									const barStyle = getBarStyle(task, minMs, rangeMs);
									const type = task.task_type_id
										? taskTypes.find((t) => t.id === task.task_type_id)
										: null;
									return (
										<button
											type="button"
											key={task.id}
											className="grid w-full border-b border-border/15 last:border-0 hover:bg-muted/30 transition-colors duration-150 cursor-pointer text-left"
											style={{ gridTemplateColumns: "minmax(220px, 35%) 1fr" }}
											onClick={() => onTaskClick(task)}
										>
											{/* Left: task summary */}
											<div className="flex items-center gap-2 px-4 py-2.5 border-r border-border/25 min-w-0">
												{type && (
													<span
														className="shrink-0 size-1.5 rounded-full"
														style={{
															background:
																type.color ?? "oklch(var(--muted-foreground))",
														}}
													/>
												)}
												<span className="truncate text-[12px] font-medium text-foreground">
													{task.title}
												</span>
											</div>

											{/* Right: timeline bar */}
											<div className="relative px-4 flex items-center">
												<div className="absolute inset-x-4 h-px bg-border/20" />
												<div
													className={cn(
														"absolute h-5 rounded-full",
														"bg-primary/60",
													)}
													style={barStyle}
												/>
											</div>
										</button>
									);
								})}
							</div>
						);
					})
				)}

				{/* Unstatused tasks */}
				{filtered.filter((t) => !t.status_id).length > 0 && (
					<div className="border-b border-border/25 last:border-0">
						<div className="flex items-center gap-2 px-4 py-2.5 bg-muted/20 border-b border-border/25">
							<span className="size-1.75 rounded-full bg-muted-foreground/30 shrink-0" />
							<span className="text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground/50">
								No Status
							</span>
							<span className="rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
								{filtered.filter((t) => !t.status_id).length}
							</span>
						</div>
						{filtered
							.filter((t) => !t.status_id)
							.map((task) => (
								<TaskRow
									key={task.id}
									task={task}
									taskIdPrefix={taskIdPrefix}
									statuses={statuses}
									taskTypes={taskTypes}
									onClick={() => onTaskClick(task)}
								/>
							))}
					</div>
				)}

				{canCreate && defaultStatusId && onCreateTask && taskTypes.length > 0 && (
					<div className="border-t border-border/20 bg-muted/5">
						<AddTaskRow
							taskTypes={taskTypes}
							variant="list"
							onAdd={(title, taskTypeId) => {
								void onCreateTask(defaultStatusId, title, taskTypeId);
							}}
						/>
					</div>
				)}
			</div>
		</div>
	);
}
