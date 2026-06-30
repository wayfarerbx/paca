import type { TFunction } from "i18next";
import type { CustomFieldDefinition, FieldType } from "@/lib/project-api";
import type { CustomFieldDef } from "./types";

export function shortId(id: string): string {
	return id.slice(0, 8).toUpperCase();
}

export function timeAgo(iso: string, t: TFunction<"projects">): string {
	const diff = Date.now() - new Date(iso).getTime();
	const mins = Math.floor(diff / 60000);
	if (mins < 1) return t("taskDetail.timeAgo.justNow");
	if (mins < 60) return t("taskDetail.timeAgo.minutes", { count: mins });
	const hrs = Math.floor(mins / 60);
	if (hrs < 24) return t("taskDetail.timeAgo.hours", { count: hrs });
	return t("taskDetail.timeAgo.days", { count: Math.floor(hrs / 24) });
}

export function slugify(s: string): string {
	return s
		.toLowerCase()
		.replace(/\s+/g, "_")
		.replace(/[^a-z0-9_]/g, "")
		.replace(/_+/g, "_")
		.slice(0, 64);
}

const API_TO_UI_FIELD_TYPE: Record<FieldType, CustomFieldDef["field_type"]> = {
	text: "Text",
	number: "Number",
	date: "Date",
	boolean: "Checkbox",
	select: "Select",
	multi_select: "Select",
	url: "Text",
};

const UI_TO_API_FIELD_TYPE: Record<CustomFieldDef["field_type"], FieldType> = {
	Text: "text",
	Number: "number",
	Date: "date",
	Checkbox: "boolean",
	Select: "select",
};

export function mapApiFieldToUi(
	apiField: CustomFieldDefinition,
): CustomFieldDef {
	return {
		id: apiField.id,
		display_name: apiField.display_name,
		field_key: apiField.field_key,
		field_type: API_TO_UI_FIELD_TYPE[apiField.field_type] ?? "Text",
		required: apiField.is_required,
		options: apiField.options,
	};
}

export function mapUiFieldTypeToApi(
	uiType: CustomFieldDef["field_type"],
): FieldType {
	return UI_TO_API_FIELD_TYPE[uiType];
}
