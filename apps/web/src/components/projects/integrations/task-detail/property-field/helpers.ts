export function toDateObject(iso?: string | null): Date | undefined {
	if (!iso) return undefined;
	const d = new Date(iso);
	return Number.isNaN(d.getTime()) ? undefined : d;
}

export function toISODate(date: Date): string {
	const y = date.getFullYear();
	const m = String(date.getMonth() + 1).padStart(2, "0");
	const d = String(date.getDate()).padStart(2, "0");
	return `${y}-${m}-${d}T00:00:00Z`;
}

export function displayDate(iso?: string | null) {
	if (!iso) return null;
	const d = new Date(iso);
	return d.toLocaleDateString(undefined, {
		month: "short",
		day: "numeric",
		year: "numeric",
	});
}
