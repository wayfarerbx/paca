import type { TFunction } from "i18next";

export function timeAgo(iso: string, t: TFunction<"common">): string {
	const diff = Date.now() - new Date(iso).getTime();
	const mins = Math.floor(diff / 60000);
	if (mins < 1) return t("timeAgo.justNow");
	if (mins < 60) return t("timeAgo.minutes", { count: mins });
	const hrs = Math.floor(mins / 60);
	if (hrs < 24) return t("timeAgo.hours", { count: hrs });
	return t("timeAgo.days", { count: Math.floor(hrs / 24) });
}
