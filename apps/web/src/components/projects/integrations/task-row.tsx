import { GripVertical, User } from "lucide-react";

import type { Task } from "@/lib/integration-api";
import type { TaskStatus, TaskType } from "@/lib/project-api";
import { cn } from "@/lib/utils";

import { getPriority } from "./priority";

interface TaskRowProps {
	task: Task;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	onClick?: () => void;
	showDragHandle?: boolean;
	isDragging?: boolean;
}

export function TaskRow({
	task,
	statuses,
	taskTypes,
	onClick,
	showDragHandle,
	isDragging,
}: TaskRowProps) {
	const taskType = taskTypes.find((t) => t.id === task.task_type_id);
	const status = statuses.find((s) => s.id === task.status_id);

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: draggable list row with click; converting to button breaks drag-and-drop
		// biome-ignore lint/a11y/useKeyWithClickEvents: drag-and-drop row; keyboard nav handled by parent
		<div
			onClick={onClick}
			className={cn(
				"group flex items-center gap-3 px-4 py-2.5 cursor-pointer",
				"hover:bg-muted/30 transition-colors duration-150 border-b border-border/20 last:border-0",
				isDragging && "opacity-40 bg-muted/20",
			)}
		>
			{showDragHandle && (
				<GripVertical className="size-3.5 shrink-0 -ml-1.5 text-muted-foreground/30 group-hover:text-muted-foreground/70 cursor-grab opacity-0 group-hover:opacity-100 transition-opacity duration-200" />
			)}
			<div className="w-16 shrink-0">
				{taskType ? (
					<span
						className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[11px] font-bold leading-tight tracking-wide border truncate max-w-full"
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
						{taskType.name}
					</span>
				) : (
					<span className="text-xs text-muted-foreground/50">—</span>
				)}
			</div>

			<div className="hidden sm:flex w-20 shrink-0 items-center gap-1">
				{(() => {
					const p = getPriority(task.importance);
					return task.importance > 0 ? (
						<>
							<span
								className="size-2 rounded-full shrink-0"
								style={{ background: p.color }}
							/>
							<span className="text-[11px] font-medium truncate" style={{ color: p.color }}>
								{p.label}
							</span>
						</>
					) : (
						<span className="text-[11px] text-muted-foreground/50">—</span>
					);
				})()}
			</div>

			<span className="flex-1 text-[13px] font-medium text-foreground truncate">
				{task.title}
			</span>

			<div className="hidden sm:flex w-24 shrink-0 items-center gap-1.5">
				{status ? (
					<span className="inline-flex items-center gap-1.5 rounded-full border border-border/40 bg-muted/40 px-2.5 py-0.5 text-[11px] font-semibold text-muted-foreground tracking-wide">
						<span
							className="size-1.5 rounded-full shrink-0"
							style={{
								background: status.color ?? "oklch(var(--muted-foreground))",
								boxShadow: status.color ? `0 0 4px ${status.color}40` : undefined,
							}}
						/>
						{status.name}
					</span>
				) : (
					<span className="text-[11px] text-muted-foreground/50">—</span>
				)}
			</div>

			<div className="shrink-0">
				{task.assignee_id ? (
					<div className="flex size-6 items-center justify-center rounded-full bg-linear-to-br from-primary/20 to-primary/10 text-primary text-[10px] font-bold ring-1 ring-primary/20">
						<User className="size-3" />
					</div>
				) : (
					<div className="flex size-6 items-center justify-center rounded-full bg-linear-to-br from-muted/80 to-muted/40 text-muted-foreground text-[10px] font-bold ring-1 ring-border/25">
						<User className="size-3" />
					</div>
				)}
			</div>
		</div>
	);
}
