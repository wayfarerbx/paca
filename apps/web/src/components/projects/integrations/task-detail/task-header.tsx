import { Check, ChevronRight, Hash, Share2, X } from "lucide-react";
import { useState } from "react";
import type { Task } from "@/lib/integration-api";
import { cn } from "@/lib/utils";
import { formatDate, shortId } from "./helpers";

interface TaskHeaderProps {
	task: Task;
	mode: "modal" | "page";
	projectName?: string;
	integrationName?: string;
	projectId?: string;
	onClose: () => void;
}

export function TaskHeader({
	task,
	mode,
	projectName,
	integrationName,
	projectId,
	onClose,
}: TaskHeaderProps) {
	const [linkCopied, setLinkCopied] = useState(false);

	const handleShare = () => {
		// Always share the canonical task detail page URL, not modal URLs with query params
		const taskUrl = projectId
			? `${window.location.origin}/projects/${projectId}/tasks/${task.id}`
			: window.location.href;
		navigator.clipboard?.writeText(taskUrl).catch(() => {});
		setLinkCopied(true);
		setTimeout(() => setLinkCopied(false), 2000);
	};

	return (
		<div className="flex shrink-0 items-center gap-2 lg:gap-3 border-b border-border/30 px-4 lg:px-6 py-2.5 bg-muted/20">
			{/* Breadcrumb (page mode only) */}
			{mode === "page" && projectName && (
				<nav className="hidden md:flex items-center gap-1.5 text-[12px] text-muted-foreground/80 mr-2">
					<span className="hover:text-foreground transition-colors cursor-pointer">
						{projectName}
					</span>
					<ChevronRight className="size-3 text-muted-foreground/45" />
					{integrationName && (
						<>
							<span className="hover:text-foreground transition-colors cursor-pointer">
								{integrationName}
							</span>
							<ChevronRight className="size-3 text-muted-foreground/45" />
						</>
					)}
					<span className="text-foreground/80 truncate max-w-36 font-medium">
						{task.title}
					</span>
				</nav>
			)}

			{/* Task short ID */}
			<div className="flex items-center gap-1.5 rounded-md bg-muted/60 px-2 py-1 border border-border/30">
				<Hash className="size-3 text-muted-foreground/60" />
				<span className="font-[JetBrains_Mono,monospace] text-[11px] font-semibold text-muted-foreground tracking-wider">
					{shortId(task.id)}
				</span>
			</div>

			<span className="hidden md:inline text-[11px] text-muted-foreground/60 font-medium">
				Created {formatDate(task.created_at)}
			</span>

			<div className="ml-auto flex items-center gap-1">
				<button
					type="button"
					onClick={handleShare}
					className={cn(
						"flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[12px] font-medium transition-all duration-150",
						linkCopied
							? "text-emerald-500 bg-emerald-500/10"
							: "text-muted-foreground/70 hover:text-foreground hover:bg-muted/60",
					)}
				>
					{linkCopied ? (
						<>
							<Check className="size-3 text-emerald-500" />
							<span>Copied!</span>
						</>
					) : (
						<>
							<Share2 className="size-3" />
							<span>Share</span>
						</>
					)}
				</button>

				{mode === "modal" && (
					<button
						type="button"
						onClick={onClose}
						className="flex size-7 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 transition-all duration-150"
						aria-label="Close task detail"
					>
						<X className="size-3.5" />
					</button>
				)}
			</div>
		</div>
	);
}
