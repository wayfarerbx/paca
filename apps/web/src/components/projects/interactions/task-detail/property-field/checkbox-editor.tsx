import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

export function CheckboxEditor({
	checked = false,
	canEdit,
	onChange,
}: {
	checked: boolean;
	canEdit: boolean;
	onChange?: (value: boolean) => void;
}) {
	return (
		// biome-ignore lint/a11y/useSemanticElements: custom styled checkbox matching project design system
		<button
			type="button"
			role="checkbox"
			aria-checked={checked}
			disabled={!canEdit}
			onClick={() => canEdit && onChange?.(!checked)}
			className={cn(
				"size-4.5 rounded-md border-2 flex items-center justify-center transition-all duration-150",
				checked
					? "bg-primary border-primary text-primary-foreground"
					: "border-border/40 bg-muted/20 hover:border-border/60",
				!canEdit && "cursor-default opacity-60",
			)}
		>
			{checked && <Check className="size-3" />}
		</button>
	);
}
