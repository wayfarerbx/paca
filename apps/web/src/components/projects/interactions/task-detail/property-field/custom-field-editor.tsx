import { CalendarDays } from "lucide-react";
import { FieldValue } from "../primitives";
import { CheckboxEditor } from "./checkbox-editor";
import { SingleDateEditor } from "./date-editor";
import { displayDate } from "./helpers";
import { NumberEditor } from "./number-editor";
import { SelectEditor } from "./select-editor";
import { TextEditor } from "./text-editor";
import type { SelectOption } from "./types";

export function CustomFieldEditor({
	customType,
	rawValue,
	canEdit,
	options = [],
	onChange,
}: {
	customType: "Text" | "Number" | "Date" | "Checkbox" | "Select";
	rawValue: unknown;
	canEdit: boolean;
	options?: string[];
	onChange?: (value: unknown) => void;
}) {
	switch (customType) {
		case "Text":
			return (
				<TextEditor
					value={rawValue != null ? String(rawValue) : null}
					canEdit={canEdit}
					onChange={(v) => onChange?.(v)}
				/>
			);
		case "Number": {
			const num =
				typeof rawValue === "number" ? rawValue : Number(rawValue) || 0;
			if (!canEdit) {
				return (
					<span className="text-[13px] tabular-nums font-medium text-foreground">
						{num}
					</span>
				);
			}
			return <NumberEditor value={num} onChange={(v) => onChange?.(v)} />;
		}
		case "Date":
			if (!canEdit) {
				return (
					<span className="inline-flex items-center gap-1.5 rounded-lg border border-border/25 bg-muted/25 px-2.5 py-1.5 text-[11px] text-muted-foreground font-medium">
						<CalendarDays className="size-3 shrink-0 opacity-70" />
						{displayDate(rawValue as string | null) ?? "Empty"}
					</span>
				);
			}
			return (
				<SingleDateEditor
					value={rawValue as string | null}
					onChange={(v) => onChange?.(v)}
				/>
			);
		case "Checkbox":
			return (
				<CheckboxEditor
					checked={Boolean(rawValue)}
					canEdit={canEdit}
					onChange={(v) => onChange?.(v)}
				/>
			);
		case "Select": {
			const selectOptions: SelectOption[] = options.map((o) => ({
				value: o,
				label: o,
			}));
			const currentVal = rawValue != null ? String(rawValue) : null;
			if (!canEdit) {
				return <FieldValue empty={!currentVal}>{currentVal}</FieldValue>;
			}
			return (
				<SelectEditor
					value={currentVal}
					options={selectOptions}
					onChange={(v) => onChange?.(v)}
				/>
			);
		}
	}
}
