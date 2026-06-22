import { Check } from "lucide-react";
import { getTaskTypeIconComponent } from "@/components/projects/task-types/task-type-icons";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { TaskType } from "@/lib/project-api";

interface TaskTypeSelectorProps {
	taskTypes: TaskType[];
	value: string | null | undefined;
	onChange?: (taskTypeId: string) => void;
	canEdit?: boolean;
	align?: "start" | "end" | "center";
}

/** Badge + dropdown for picking a task's type. Used anywhere a task type can be set or changed. */
export function TaskTypeSelector({
	taskTypes,
	value,
	onChange,
	canEdit = true,
	align = "start",
}: TaskTypeSelectorProps) {
	const taskType = taskTypes.find((tt) => tt.id === value);
	const Icon = taskType ? getTaskTypeIconComponent(taskType.icon) : null;

	const badgeStyle = taskType
		? {
				borderColor: taskType.color ? `${taskType.color}44` : "var(--border)",
				backgroundColor: taskType.color
					? `${taskType.color}15`
					: "var(--muted)",
				color: taskType.color ?? "inherit",
			}
		: undefined;

	const badgeContent = taskType ? (
		<>
			{Icon && <Icon className="size-3 shrink-0" />}
			<span className="truncate">{taskType.name}</span>
		</>
	) : (
		<span className="text-muted-foreground/50">—</span>
	);

	if (!canEdit || taskTypes.length === 0) {
		return (
			<span
				className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-bold leading-tight tracking-wide border truncate max-w-full"
				style={badgeStyle}
			>
				{badgeContent}
			</span>
		);
	}

	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				type="button"
				className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-bold leading-tight tracking-wide border truncate max-w-full hover:opacity-80 transition-opacity"
				style={badgeStyle}
			>
				{badgeContent}
			</DropdownMenuTrigger>
			<DropdownMenuContent
				className="w-44 p-1 rounded-xl border border-border/40 shadow-lg"
				align={align}
			>
				{taskTypes.map((tt) => {
					const TtIcon = getTaskTypeIconComponent(tt.icon);
					return (
						<DropdownMenuItem
							key={tt.id}
							className="rounded-lg px-3 py-2"
							onClick={() => onChange?.(tt.id)}
						>
							{TtIcon && (
								<TtIcon
									className="size-3.5 shrink-0"
									style={tt.color ? { color: tt.color } : undefined}
								/>
							)}
							<span className="flex-1 text-left truncate">{tt.name}</span>
							{tt.id === value && (
								<Check className="size-3.5 text-primary shrink-0" />
							)}
						</DropdownMenuItem>
					);
				})}
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
