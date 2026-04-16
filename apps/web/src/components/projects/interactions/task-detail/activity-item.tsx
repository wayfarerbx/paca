import { cn } from "@/lib/utils";
import { timeAgo } from "./helpers";
import type { ActivityEntry } from "./types";

export function ActivityItem({ entry }: { entry: ActivityEntry }) {
	const isComment = entry.type === "comment";
	return (
		<div className="flex gap-3">
			<div
				className={cn(
					"flex size-6 shrink-0 items-center justify-center rounded-full text-[10px] font-bold mt-0.5 ring-1",
					isComment
						? "bg-linear-to-br from-primary/20 to-primary/10 text-primary ring-primary/15"
						: "bg-muted/40 text-muted-foreground/80 ring-border/20",
				)}
			>
				{entry.author.slice(0, 1).toUpperCase()}
			</div>
			<div className="flex-1 min-w-0">
				{isComment ? (
					<div className="rounded-xl rounded-tl-lg border border-border/25 bg-card/70 px-3.5 py-2.5">
						<div className="mb-1 flex items-center gap-2">
							<span className="text-[12px] font-semibold text-foreground">
								{entry.author}
							</span>
							<span className="text-[10px] text-muted-foreground/50">
								{timeAgo(entry.timestamp)}
							</span>
						</div>
						<p className="text-[13px] text-foreground leading-relaxed">
							{entry.content}
						</p>
					</div>
				) : (
					<div className="flex flex-wrap items-baseline gap-1.5 py-0.5">
						<span className="text-[12px] font-medium text-foreground/80">
							{entry.author}
						</span>
						<span className="text-[12px] text-muted-foreground/70">
							{entry.content}
						</span>
						<span className="text-[10px] text-muted-foreground/45">
							{timeAgo(entry.timestamp)}
						</span>
					</div>
				)}
			</div>
		</div>
	);
}
