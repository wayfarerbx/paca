import { FileText, Sparkles } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type UpdateFn = (payload: { description?: string | null }) => void;

interface DescriptionSectionProps {
	description?: string | null;
	canEdit?: boolean;
	onUpdate?: UpdateFn;
}

export function DescriptionSection({
	description,
	canEdit = true,
	onUpdate,
}: DescriptionSectionProps) {
	const [editing, setEditing] = useState(false);
	const [draft, setDraft] = useState(description ?? "");
	const textareaRef = useRef<HTMLTextAreaElement>(null);

	// Sync draft when description changes externally (e.g. after a save)
	useEffect(() => {
		if (!editing) setDraft(description ?? "");
	}, [description, editing]);

	const commitEdit = () => {
		setEditing(false);
		const value = draft.trim() || null;
		if (value !== (description ?? null)) {
			onUpdate?.({ description: value });
		}
	};

	return (
		<div className="space-y-3">
			<div className="flex items-center justify-between">
				<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
					<span>Description</span>
					<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
				</h3>
				<button
					type="button"
					className="flex items-center gap-1.5 text-[11px] text-muted-foreground/60 hover:text-muted-foreground transition-colors duration-150 font-medium"
				>
					<Sparkles className="size-3" />
					Write with AI
				</button>
			</div>

			{editing ? (
				<textarea
					ref={textareaRef}
					value={draft}
					onChange={(e) => setDraft(e.target.value)}
					onBlur={commitEdit}
					onKeyDown={(e) => {
						if (e.key === "Escape") {
							setDraft(description ?? "");
							setEditing(false);
						}
					}}
					placeholder="Add description…"
					className="w-full min-h-35 resize-y rounded-xl border-2 border-primary/30 bg-muted/20 px-5 py-4 text-[14px] text-foreground leading-relaxed whitespace-pre-wrap outline-none focus:border-primary/50 focus:bg-muted/30 transition-all duration-150"
				/>
			) : description ? (
				// biome-ignore lint/a11y/noStaticElementInteractions: click-to-edit description
				// biome-ignore lint/a11y/useKeyWithClickEvents: click-to-edit description
				<div
					className="rounded-xl border border-border/25 bg-card/50 px-5 py-4 cursor-text hover:border-border/50 hover:bg-card/80 transition-all duration-200 group/desc"
					onClick={() => {
						if (!canEdit) return;
						setDraft(description);
						setEditing(true);
					}}
				>
					<p className="text-[14px] text-foreground whitespace-pre-wrap leading-relaxed">
						{description}
					</p>
					{canEdit && (
						<span className="block mt-2 text-[11px] text-muted-foreground/45 opacity-0 group-hover/desc:opacity-100 transition-opacity duration-200">
							Click to edit
						</span>
					)}
				</div>
			) : (
				<button
					type="button"
					onClick={() => {
						if (!canEdit) return;
						setDraft("");
						setEditing(true);
					}}
					className="w-full rounded-xl border-2 border-dashed border-border/25 bg-muted/10 px-5 py-6 text-left hover:border-border/50 hover:bg-muted/20 transition-all duration-200 group/add"
				>
					<div className="flex items-center gap-3">
						<div className="flex size-8 items-center justify-center rounded-lg bg-muted/40 text-muted-foreground/45 group-hover/add:text-muted-foreground/70 transition-colors">
							<FileText className="size-4" />
						</div>
						<span className="text-[13px] text-muted-foreground/60 group-hover/add:text-muted-foreground/80 font-medium transition-colors">
							Add a description…
						</span>
					</div>
				</button>
			)}
		</div>
	);
}
