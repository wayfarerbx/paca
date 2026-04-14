import { Check } from "lucide-react";
import { useState } from "react";
import type { Task } from "@/lib/interaction-api";
import type { TaskStatus } from "@/lib/project-api";
import { cn } from "@/lib/utils";

interface SubtaskRowProps {
	task: Task;
	statuses: TaskStatus[];
}

export function SubtaskRow({ task, statuses }: SubtaskRowProps) {
	const status = statuses.find((s) => s.id === task.status_id);
	const [done, setDone] = useState(false);
	return (
		<div className="group/sub flex items-center gap-3 px-3.5 py-2.5 hover:bg-muted/30 transition-colors duration-150">
			<button
				type="button"
				onClick={() => setDone(!done)}
				className={cn(
					"flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border-2 transition-all duration-200",
					done
						? "border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/20"
						: "border-border/40 text-transparent hover:border-border/70 hover:bg-muted/40",
				)}
			>
				<Check className="size-2.5" strokeWidth={3} />
			</button>
			<span
				className={cn(
					"flex-1 text-[13px] truncate transition-all duration-200",
					done ? "line-through text-muted-foreground/60" : "text-foreground",
				)}
			>
				{task.title}
			</span>
			{status && (
				<span className="shrink-0 inline-flex items-center gap-1.5 rounded-full border border-border/25 bg-muted/30 px-2.5 py-0.5 text-[10px] font-semibold text-muted-foreground/80 tracking-wide">
					<span
						className="size-1.5 rounded-full"
						style={{ background: status.color ?? "currentColor" }}
					/>
					{status.name}
				</span>
			)}
		</div>
	);
}
