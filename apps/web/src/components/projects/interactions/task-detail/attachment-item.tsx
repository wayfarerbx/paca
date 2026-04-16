import { Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { timeAgo } from "./helpers";
import type { Attachment } from "./types";

interface AttachmentItemProps {
	attachment: Attachment;
	canDelete?: boolean;
	onDelete?: (id: string) => void;
}

export function AttachmentItem({
	attachment,
	canDelete,
	onDelete,
}: AttachmentItemProps) {
	const ext = attachment.name.split(".").pop()?.toUpperCase() ?? "FILE";
	return (
		<div className="group/att flex items-center gap-3.5 rounded-xl border border-border/20 bg-muted/15 px-4 py-3 transition-all duration-150 hover:bg-muted/25 hover:border-border/35">
			<div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-linear-to-br from-primary/12 to-primary/5 text-primary/80">
				<span className="text-[10px] font-bold tracking-tight">{ext}</span>
			</div>
			<div className="flex-1 min-w-0">
				<p className="text-[13px] font-medium text-foreground truncate">
					{attachment.name}
				</p>
				<p className="text-[11px] text-muted-foreground/60 mt-0.5">
					{timeAgo(attachment.uploaded_at)}
				</p>
			</div>
			{canDelete && (
				<button
					type="button"
					onClick={() => onDelete?.(attachment.id)}
					className={cn(
						"shrink-0 size-7 flex items-center justify-center rounded-lg",
						"text-muted-foreground/45 opacity-0 group-hover/att:opacity-100",
						"hover:text-destructive hover:bg-destructive/8 transition-all duration-150",
					)}
					aria-label={`Delete ${attachment.name}`}
				>
					<Trash2 className="size-3.5" />
				</button>
			)}
		</div>
	);
}
