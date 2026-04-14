import { Check } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import type { Checklist } from "./types";

interface ChecklistSectionProps {
	checklist: Checklist;
	onUpdate: (updated: Checklist) => void;
}

export function ChecklistSection({
	checklist,
	onUpdate,
}: ChecklistSectionProps) {
	const [newItem, setNewItem] = useState("");
	const completed = checklist.items.filter((i) => i.checked).length;
	const pct =
		checklist.items.length > 0
			? Math.round((completed / checklist.items.length) * 100)
			: 0;

	const toggle = (id: string) => {
		onUpdate({
			...checklist,
			items: checklist.items.map((i) =>
				i.id === id ? { ...i, checked: !i.checked } : i,
			),
		});
	};

	const addItem = () => {
		const text = newItem.trim();
		if (!text) return;
		onUpdate({
			...checklist,
			items: [
				...checklist.items,
				{ id: crypto.randomUUID(), text, checked: false },
			],
		});
		setNewItem("");
	};

	return (
		<div className="space-y-3">
			{/* Header */}
			<div className="flex items-center gap-3">
				<span className="text-[13px] font-semibold text-foreground flex-1">
					{checklist.title}
				</span>
				<span
					className={cn(
						"text-[11px] font-bold tabular-nums rounded-full px-2 py-0.5",
						pct === 100
							? "bg-emerald-500/15 text-emerald-600"
							: "bg-muted/50 text-muted-foreground/80",
					)}
				>
					{completed}/{checklist.items.length}
				</span>
			</div>

			{/* Progress bar */}
			<div className="h-1.5 rounded-full bg-border/25 overflow-hidden">
				<div
					className={cn(
						"h-full rounded-full transition-all duration-500 ease-out",
						pct === 100
							? "bg-emerald-500 shadow-sm shadow-emerald-500/30"
							: "bg-primary/60",
					)}
					style={{ width: `${pct}%` }}
				/>
			</div>

			{/* Items */}
			<div className="space-y-0.5">
				{checklist.items.map((item) => (
					<div
						key={item.id}
						className="flex items-center gap-3 rounded-lg px-2 py-2 hover:bg-muted/30 transition-colors duration-150 group/ci"
					>
						<button
							type="button"
							onClick={() => toggle(item.id)}
							className={cn(
								"flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border-2 transition-all duration-200",
								item.checked
									? "border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/20"
									: "border-border/40 text-transparent hover:border-border/70 hover:bg-muted/40",
							)}
						>
							<Check className="size-2.5" strokeWidth={3} />
						</button>
						<span
							className={cn(
								"flex-1 text-[13px] transition-all duration-200",
								item.checked
									? "line-through text-muted-foreground/60"
									: "text-foreground",
							)}
						>
							{item.text}
						</span>
					</div>
				))}

				{/* Add item input */}
				<div className="flex items-center gap-3 px-2 pt-1">
					<div className="size-4.5 shrink-0 rounded-[5px] border-2 border-dashed border-border/25" />
					<input
						value={newItem}
						onChange={(e) => setNewItem(e.target.value)}
						onKeyDown={(e) => {
							if (e.key === "Enter") addItem();
						}}
						placeholder="Add an item…"
						className="flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted-foreground/45 py-1.5 focus:placeholder:text-muted-foreground/70 transition-colors"
					/>
					{newItem && (
						<button
							type="button"
							onClick={addItem}
							className="text-[11px] text-primary/80 font-semibold hover:text-primary transition-colors duration-150"
						>
							Add
						</button>
					)}
				</div>
			</div>
		</div>
	);
}
