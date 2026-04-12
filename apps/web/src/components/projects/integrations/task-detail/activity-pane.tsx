import {
	Bold,
	Hash,
	Italic,
	List,
	MessageSquare,
	Paperclip,
	Send,
	Smile,
} from "lucide-react";
import { useState } from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { ActivityItem } from "./activity-item";
import type { ActivityEntry } from "./types";

interface ActivityPaneProps {
	activities: ActivityEntry[];
}

export function ActivityPane({ activities }: ActivityPaneProps) {
	const [comment, setComment] = useState("");
	const [commentFocused, setCommentFocused] = useState(false);

	const handleSend = () => {
		const text = comment.trim();
		if (!text) return;
		setComment("");
		setCommentFocused(false);
	};

	return (
		<div className="flex w-full lg:w-80 lg:shrink-0 flex-col lg:overflow-hidden border-t lg:border-t-0 lg:border-l border-border/25 bg-muted/10">
			{/* Header */}
			<div className="flex shrink-0 items-center gap-2.5 border-b border-border/25 px-5 py-3 bg-muted/20">
				<MessageSquare className="size-3.5 text-muted-foreground/70" />
				<span className="text-[12px] font-semibold text-muted-foreground uppercase tracking-wide">
					Activity
				</span>
				{activities.length > 0 && (
					<span className="ml-auto rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
						{activities.length}
					</span>
				)}
			</div>

			{/* Activity feed */}
			<ScrollArea className="lg:flex-1 px-4 py-4 max-h-[40vh] lg:max-h-none">
				<div className="space-y-3">
					{activities.map((entry) => (
						<ActivityItem key={entry.id} entry={entry} />
					))}
					{activities.length === 0 && (
						<div className="flex flex-col items-center py-8 text-muted-foreground/40">
							<MessageSquare className="size-6 mb-2" />
							<p className="text-[12px] font-medium">No activity yet</p>
						</div>
					)}
				</div>
			</ScrollArea>

			{/* Comment input */}
			<div className="shrink-0 border-t border-border/25 p-3 space-y-1.5 bg-background/50">
				{/* Formatting toolbar (shown when focused) */}
				{commentFocused && (
					<div className="flex items-center gap-0.5 rounded-lg border border-border/25 bg-muted/25 px-2 py-1">
						{[
							{ icon: Bold, title: "Bold" },
							{ icon: Italic, title: "Italic" },
							{ icon: List, title: "List" },
						].map(({ icon: Icon, title }) => (
							<button
								key={title}
								type="button"
								title={title}
								className="flex size-6 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/50 transition-all duration-150"
							>
								<Icon className="size-3" />
							</button>
						))}
						<div className="mx-1 h-3.5 w-px bg-border/30" />
						{[
							{ icon: Smile, title: "Emoji" },
							{ icon: Paperclip, title: "Attach" },
							{ icon: Hash, title: "Mention" },
						].map(({ icon: Icon, title }) => (
							<button
								key={title}
								type="button"
								title={title}
								className="flex size-6 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/50 transition-all duration-150"
							>
								<Icon className="size-3" />
							</button>
						))}
					</div>
				)}

				{/* Textarea + send */}
				<div
					className={cn(
						"flex items-end gap-2 rounded-xl border border-border/30 bg-card/80 px-3 py-2.5 transition-all duration-200",
						commentFocused && "border-primary/25 shadow-sm shadow-primary/5",
					)}
				>
					<textarea
						value={comment}
						onChange={(e) => setComment(e.target.value)}
						onFocus={() => setCommentFocused(true)}
						onBlur={() => !comment && setCommentFocused(false)}
						placeholder="Write a comment…"
						rows={commentFocused ? 3 : 1}
						className="flex-1 resize-none bg-transparent text-[13px] outline-none placeholder:text-muted-foreground/50 leading-relaxed"
						onKeyDown={(e) => {
							if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
								handleSend();
							}
						}}
					/>
					<button
						type="button"
						onClick={handleSend}
						disabled={!comment.trim()}
						className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground disabled:opacity-40 hover:bg-primary/90 transition-all duration-150 shadow-sm disabled:shadow-none"
					>
						<Send className="size-3" />
					</button>
				</div>
				{commentFocused && (
					<p className="text-[10px] text-muted-foreground/40 text-right pr-1">
						⌘↵ to send
					</p>
				)}
			</div>
		</div>
	);
}
