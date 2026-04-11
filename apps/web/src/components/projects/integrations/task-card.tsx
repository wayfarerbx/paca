import { Check, GripVertical, User } from "lucide-react";

import { getTaskTypeIconComponent } from "@/components/projects/task-types/task-type-icons";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import type { Task } from "@/lib/integration-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";
import { cn } from "@/lib/utils";

type UpdatePayload = Partial<{
	task_type_id: string | null;
	assignee_id: string | null;
}>;

interface TaskCardProps {
	task: Task;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members?: ProjectMember[];
	onClick?: () => void;
	onDragStart?: (e: React.DragEvent) => void;
	onDragEnd?: (e: React.DragEvent) => void;
	isDragging?: boolean;
	canEdit?: boolean;
	onUpdate?: (taskId: string, payload: UpdatePayload) => void;
}

export function TaskCard({
	task,
	taskTypes,
	members = [],
	onClick,
	onDragStart,
	onDragEnd,
	isDragging,
	canEdit,
	onUpdate,
}: TaskCardProps) {
	const taskType = taskTypes.find((t) => t.id === task.task_type_id);
	const assignee = task.assignee_id
		? members.find((m) => m.user_id === task.assignee_id)
		: undefined;

	return (
		<div
			data-task-id={task.id}
			draggable={canEdit}
			onDragStart={onDragStart}
			onDragEnd={onDragEnd}
			onClick={onClick}
			className={cn(
				"group relative rounded-xl border border-border/30 bg-card p-3 shadow-xs cursor-pointer transition-all duration-150 select-none",
				"hover:border-border/50 hover:shadow-sm",
				isDragging && "opacity-50 ring-2 ring-primary/30 shadow-lg rotate-1",
				canEdit && "cursor-grab active:cursor-grabbing",
			)}
		>
			{canEdit && (
				<div className="absolute left-1.5 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
					<GripVertical className="size-3.5 text-muted-foreground/60" />
				</div>
			)}

			<span className="text-sm font-medium leading-snug text-foreground line-clamp-2">
				{task.title}
			</span>

			<div className="mt-2 flex items-center gap-1.5">
				{canEdit && members.length > 0 ? (
					<Popover>
						<PopoverTrigger
							type="button"
							onClick={(e) => e.stopPropagation()}
							className="flex size-5 items-center justify-center rounded-full transition-all duration-150 hover:ring-2 hover:ring-primary/30"
						>
							{assignee ? (
								<div className="flex size-5 items-center justify-center rounded-full bg-linear-to-br from-primary/20 to-primary/10 text-primary text-[10px] font-bold ring-1 ring-primary/20">
									{(assignee.full_name || assignee.username)
										.slice(0, 1)
										.toUpperCase()}
								</div>
							) : (
								<div className="flex size-5 items-center justify-center rounded-full bg-linear-to-br from-muted/80 to-muted/40 text-muted-foreground text-[10px] font-bold ring-1 ring-border/25">
									<User className="size-2.5" />
								</div>
							)}
						</PopoverTrigger>
						<PopoverContent
							className="w-48 p-1 rounded-xl border border-border/40 shadow-lg"
							align="start"
						>
							<button
								type="button"
								className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] text-muted-foreground hover:bg-muted/60 transition-colors duration-100"
								onClick={(e) => {
									e.stopPropagation();
									onUpdate?.(task.id, { assignee_id: null });
								}}
							>
								<User className="size-3.5 opacity-60" />
								<span className="flex-1 text-left">Unassigned</span>
								{!assignee && <Check className="size-3.5 text-primary" />}
							</button>
							{members.map((m) => (
								<button
									key={m.user_id}
									type="button"
									className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] hover:bg-muted/60 transition-colors duration-100"
									onClick={(e) => {
										e.stopPropagation();
										onUpdate?.(task.id, { assignee_id: m.user_id });
									}}
								>
									<div className="flex size-5 items-center justify-center rounded-full bg-linear-to-br from-primary/20 to-primary/10 text-primary text-[9px] font-bold">
										{(m.full_name || m.username)
											.slice(0, 1)
											.toUpperCase()}
									</div>
									<span className="flex-1 text-left truncate">
										{m.full_name || m.username}
									</span>
									{m.user_id === assignee?.user_id && (
										<Check className="size-3.5 text-primary" />
									)}
								</button>
							))}
						</PopoverContent>
					</Popover>
				) : task.assignee_id ? (
					<div className="flex size-5 items-center justify-center rounded-full bg-linear-to-br from-primary/20 to-primary/10 text-primary text-[10px] font-bold ring-1 ring-primary/20">
						<User className="size-2.5" />
					</div>
				) : (
					<div className="flex size-5 items-center justify-center rounded-full bg-linear-to-br from-muted/80 to-muted/40 text-muted-foreground text-[10px] font-bold ring-1 ring-border/25">
						<User className="size-2.5" />
					</div>
				)}

				{canEdit && taskTypes.length > 0 ? (
					<Popover>
						<PopoverTrigger
							type="button"
							onClick={(e) => e.stopPropagation()}
							className="flex items-center justify-center rounded-md p-0.5 transition-all duration-150 hover:bg-muted/60"
						>
							{taskType ? (() => {
								const Icon = getTaskTypeIconComponent(taskType.icon);
								return Icon ? (
									<Icon
										className="size-3.5"
										style={taskType.color ? { color: taskType.color } : undefined}
									/>
								) : (
									<span
										className="text-[10px] font-bold"
										style={taskType.color ? { color: taskType.color } : undefined}
									>
										{taskType.name.slice(0, 2)}
									</span>
								);
							})() : (
								<span className="text-[10px] text-muted-foreground/50">--</span>
							)}
						</PopoverTrigger>
						<PopoverContent
							className="w-44 p-1 rounded-xl border border-border/40 shadow-lg"
							align="start"
						>
							{taskTypes.map((tt) => {
								const TtIcon = getTaskTypeIconComponent(tt.icon);
								return (
									<button
										key={tt.id}
										type="button"
										className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-[13px] hover:bg-muted/60 transition-colors duration-100"
										onClick={(e) => {
											e.stopPropagation();
											onUpdate?.(task.id, { task_type_id: tt.id });
										}}
									>
										{TtIcon && (
											<TtIcon
												className="size-3.5 text-muted-foreground/80 shrink-0"
												style={tt.color ? { color: tt.color } : undefined}
											/>
										)}
										<span className="flex-1 text-left">{tt.name}</span>
										{tt.id === taskType?.id && (
											<Check className="size-3.5 text-primary" />
										)}
									</button>
								);
							})}
						</PopoverContent>
					</Popover>
				) : (
					taskType && (() => {
						const Icon = getTaskTypeIconComponent(taskType.icon);
						return Icon ? (
							<Icon
								className="size-3.5"
								style={taskType.color ? { color: taskType.color } : undefined}
							/>
						) : (
							<span
								className="text-[10px] font-bold"
								style={taskType.color ? { color: taskType.color } : undefined}
							>
								{taskType.name.slice(0, 2)}
							</span>
						);
					})()
				)}
			</div>
		</div>
	);
}
