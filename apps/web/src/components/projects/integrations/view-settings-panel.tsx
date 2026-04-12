import { Settings } from "lucide-react";
import { useEffect, useState } from "react";

import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import type { IntegrationView, ViewConfig } from "@/lib/integration-api";
import { cn } from "@/lib/utils";

const SORT_OPTIONS = ["Manual", "Priority", "Title", "Created"];
const FIELD_SUM_OPTIONS = ["Count", "Story Points"];
const COLUMN_BY_OPTIONS = ["Status", "Assignee", "Priority"];
const SWIMLANE_OPTIONS = ["None", "Assignee", "Priority", "Type"];
const SLICE_BY_OPTIONS = ["None", "Assignee", "Priority", "Type"];

const labelToKey = (label: string) => label.toLowerCase().replace(/\s+/g, "_");

interface ViewSettingsPanelProps {
	view: IntegrationView | null;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onSave: (viewId: string, config: ViewConfig) => Promise<unknown>;
	onPreview: (config: ViewConfig) => void;
	isPending?: boolean;
}

function SettingRow({
	label,
	children,
}: {
	label: string;
	children: React.ReactNode;
}) {
	return (
		<div className="flex items-center justify-between gap-3 py-2">
			<span className="text-[12px] font-medium text-muted-foreground shrink-0 w-20">
				{label}
			</span>
			{children}
		</div>
	);
}

function SettingSelect({
	value,
	options,
	onChange,
	placeholder = "Default",
}: {
	value: string | undefined;
	options: string[];
	onChange: (v: string | undefined) => void;
	placeholder?: string;
}) {
	return (
		<select
			value={value ?? ""}
			onChange={(e) =>
				onChange(e.target.value === "" ? undefined : e.target.value)
			}
			className="flex-1 rounded-lg border border-border/30 bg-muted/25 px-2.5 py-1.5 text-[12px] font-medium outline-none focus:border-primary/40 focus:ring-2 focus:ring-primary/15 transition-all duration-150 min-w-0"
		>
			<option value="">{placeholder}</option>
			{options.map((o) => (
				<option key={o} value={labelToKey(o)}>
					{o}
				</option>
			))}
		</select>
	);
}

function SortSelect({
	value,
	options,
	onChange,
}: {
	value: string | undefined;
	options: string[];
	onChange: (v: string) => void;
}) {
	return (
		<select
			value={value || "manual"}
			onChange={(e) => onChange(e.target.value)}
			className="flex-1 rounded-lg border border-border/30 bg-muted/25 px-2.5 py-1.5 text-[12px] font-medium outline-none focus:border-primary/40 focus:ring-2 focus:ring-primary/15 transition-all duration-150 min-w-0"
		>
			{options.map((o) => (
				<option key={o} value={o.toLowerCase()}>
					{o}
				</option>
			))}
		</select>
	);
}

export function ViewSettingsPanel({
	view,
	open,
	onOpenChange,
	onSave,
	onPreview,
	isPending,
}: ViewSettingsPanelProps) {
	const [draft, setDraft] = useState<ViewConfig>(() => view?.config ?? {});

	// biome-ignore lint/correctness/useExhaustiveDependencies: intentionally keyed on view?.id so config is re-read only when the view itself changes, not on every config mutation
	useEffect(() => {
		if (open) setDraft(view?.config ?? {});
	}, [open, view?.id]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: onPreview is a stable callback; adding it would cause infinite re-renders
	useEffect(() => {
		if (open) onPreview(draft);
	}, [draft, open]);

	const handleOpenChange = (newOpen: boolean) => {
		if (!newOpen && view) onPreview(view.config ?? {});
		onOpenChange(newOpen);
	};

	const update = (patch: Partial<ViewConfig>) => {
		setDraft((prev) => {
			const next = { ...prev, ...patch };
			for (const key of Object.keys(patch) as (keyof ViewConfig)[]) {
				if (patch[key] === undefined) delete next[key];
			}
			return next;
		});
	};

	const handleSave = async () => {
		if (!view) return;
		await onSave(view.id, draft);
		onOpenChange(false);
	};

	const handleReset = () => {
		const saved = view?.config ?? {};
		setDraft(saved);
		onPreview(saved);
	};

	return (
		<Popover open={open} onOpenChange={handleOpenChange}>
			<PopoverTrigger
				render={
					<button
						type="button"
						aria-label="View settings"
						className={cn(
							"flex size-7 items-center justify-center rounded-md transition-all duration-150",
							open
								? "bg-primary/8 text-primary/80"
								: "text-muted-foreground/60 hover:text-foreground hover:bg-muted/60",
						)}
					/>
				}
			>
				<Settings className="size-3.5" />
			</PopoverTrigger>
			<PopoverContent
				side="bottom"
				align="end"
				className="w-72 p-0 gap-0 rounded-xl border border-border/40 shadow-lg"
				sideOffset={6}
			>
				<div className="px-3 py-2.5 border-b border-border/30">
					<p className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70">
						View settings
					</p>
				</div>
				<div className="px-3 py-1 flex flex-col divide-y divide-border/20">
					<SettingRow label="Fields">
						<span className="text-[12px] font-medium text-foreground flex-1 truncate">
							{draft.fields?.join(", ") || "Title, Assignees, Status"}
						</span>
					</SettingRow>
					<SettingRow label="Column by">
						<SettingSelect
							value={draft.column_by}
							options={COLUMN_BY_OPTIONS}
							onChange={(v) => update({ column_by: v })}
							placeholder="Status"
						/>
					</SettingRow>
					<SettingRow label="Swimlanes">
						<SettingSelect
							value={draft.swimlanes}
							options={SWIMLANE_OPTIONS}
							onChange={(v) => update({ swimlanes: v })}
							placeholder="None"
						/>
					</SettingRow>
					<SettingRow label="Sort by">
						<SortSelect
							value={draft.sort_by}
							options={SORT_OPTIONS}
							onChange={(v) => update({ sort_by: v })}
						/>
					</SettingRow>
					<SettingRow label="Field sum">
						<SettingSelect
							value={draft.field_sum}
							options={FIELD_SUM_OPTIONS}
							onChange={(v) => update({ field_sum: v })}
							placeholder="Count"
						/>
					</SettingRow>
					<SettingRow label="Slice by">
						<SettingSelect
							value={draft.slice_by}
							options={SLICE_BY_OPTIONS}
							onChange={(v) => update({ slice_by: v })}
							placeholder="None"
						/>
					</SettingRow>
				</div>
				<div className="flex items-center justify-end gap-2 px-3 py-2.5 border-t border-border/30">
					<button
						type="button"
						onClick={handleReset}
						className="flex items-center gap-1.5 rounded-lg bg-muted/40 text-muted-foreground/80 hover:bg-muted/60 hover:text-foreground px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
					>
						Reset
					</button>
					<button
						type="button"
						onClick={handleSave}
						disabled={isPending}
						className="rounded-lg bg-primary px-3 py-1.5 text-[11px] font-semibold text-primary-foreground hover:bg-primary/90 shadow-sm disabled:opacity-40 transition-all duration-150"
					>
						{isPending ? "Saving…" : "Save"}
					</button>
				</div>
			</PopoverContent>
		</Popover>
	);
}
