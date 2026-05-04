import { ListChecks, Plus } from "lucide-react";
import { useState } from "react";
import { ChecklistSection } from "./checklist-section";
import type { Checklist } from "./types";

export function ChecklistsSection({ canEdit = true }: { canEdit?: boolean }) {
	const [checklists, setChecklists] = useState<Checklist[]>([]);

	const handleCreate = () => {
		setChecklists((c) => [
			...c,
			{
				id: crypto.randomUUID(),
				title: `Checklist ${c.length + 1}`,
				items: [],
			},
		]);
	};

	return (
		<div className="space-y-3">
			<div className="flex items-center justify-between">
				<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
					<span>Checklists</span>
					<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
				</h3>
				{canEdit && (
					<button
						type="button"
						onClick={handleCreate}
						className="flex items-center gap-1.5 rounded-lg bg-muted/40 text-muted-foreground/80 hover:bg-muted/60 hover:text-foreground px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
					>
						<Plus className="size-3" />
						Create checklist
					</button>
				)}
			</div>

			{checklists.length > 0 ? (
				<div className="space-y-3">
					{checklists.map((cl) => (
						<div
							key={cl.id}
							className="rounded-xl border border-border/25 bg-card/50 p-4"
						>
							<ChecklistSection
								checklist={cl}
								onUpdate={(updated) =>
									setChecklists((all) =>
										all.map((c) => (c.id === updated.id ? updated : c)),
									)
								}
							/>
						</div>
					))}
				</div>
			) : (
				<div className="flex items-center gap-3 px-1 py-3 text-muted-foreground/45">
					<ListChecks className="size-4 opacity-70" />
					<p className="text-[13px] italic">No checklists yet</p>
				</div>
			)}
		</div>
	);
}
