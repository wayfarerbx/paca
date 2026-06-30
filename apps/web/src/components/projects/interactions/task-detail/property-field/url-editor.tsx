import { ExternalLink } from "lucide-react";
import { FieldValue } from "../primitives";
import { TextEditor } from "./text-editor";

export function UrlEditor({
	value,
	canEdit,
	onChange,
}: {
	value: string | null;
	canEdit: boolean;
	onChange?: (value: string) => void;
}) {
	if (!canEdit) {
		if (!value) return <FieldValue empty />;
		if (!/^https?:\/\//i.test(value)) return <FieldValue>{value}</FieldValue>;
		return (
			<a
				href={value}
				target="_blank"
				rel="noreferrer"
				className="inline-flex items-center gap-1.5 text-sm font-medium text-primary/80 hover:text-primary hover:underline underline-offset-2 transition-colors duration-150 truncate max-w-full"
			>
				<ExternalLink className="size-3 shrink-0" />
				<span className="truncate">{value}</span>
			</a>
		);
	}
	return <TextEditor value={value} canEdit={canEdit} onChange={onChange} />;
}
