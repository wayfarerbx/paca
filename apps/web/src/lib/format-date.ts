import i18n from "@/i18n";

export function formatDate(
	iso: string,
	options: Intl.DateTimeFormatOptions = {
		year: "numeric",
		month: "long",
		day: "numeric",
	},
): string {
	return new Intl.DateTimeFormat(i18n.language, options).format(new Date(iso));
}
