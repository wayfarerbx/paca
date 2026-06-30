import { useTranslation } from "react-i18next";

import { useThemeMode } from "@/hooks/use-theme-mode";

export default function ThemeToggle() {
	const { t } = useTranslation("appShell");
	const { mode, set } = useThemeMode();

	function toggleMode() {
		const nextMode =
			mode === "light" ? "dark" : mode === "dark" ? "auto" : "light";
		set(nextMode);
	}

	const label =
		mode === "auto"
			? t("theme.modeAutoTooltip")
			: t("theme.modeTooltip", { mode });

	const modeLabel =
		mode === "auto"
			? t("theme.auto")
			: mode === "dark"
				? t("theme.dark")
				: t("theme.light");

	return (
		<button
			type="button"
			onClick={toggleMode}
			aria-label={label}
			title={label}
			className="rounded-full border border-(--chip-line) bg-(--chip-bg) px-3 py-1.5 text-sm font-semibold text-(--sea-ink) shadow-[0_8px_22px_rgba(30,90,72,0.08)] transition hover:-translate-y-0.5"
		>
			{modeLabel}
		</button>
	);
}
