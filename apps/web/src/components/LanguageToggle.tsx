import { Globe } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuRadioGroup,
	DropdownMenuRadioItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLocale } from "@/hooks/use-locale";
import type { SupportedLanguage } from "@/i18n";

export default function LanguageToggle() {
	const { t } = useTranslation("appShell");
	const { locale, set, supportedLocales } = useLocale();
	const current =
		supportedLocales.find((option) => option.code === locale) ??
		supportedLocales[0];
	const label = t("language.toggleTooltip", {
		language: current.nativeLabel,
	});

	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				aria-label={label}
				title={label}
				className="flex items-center gap-1.5 rounded-full border border-(--chip-line) bg-(--chip-bg) px-3 py-1.5 text-sm font-semibold text-(--sea-ink) shadow-[0_8px_22px_rgba(30,90,72,0.08)] transition hover:-translate-y-0.5"
			>
				<Globe className="size-4" />
				{current.code.toUpperCase()}
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end" side="bottom" className="w-48">
				<DropdownMenuRadioGroup
					value={locale}
					onValueChange={(value) => set(value as SupportedLanguage)}
				>
					{supportedLocales.map((option) => (
						<DropdownMenuRadioItem key={option.code} value={option.code}>
							{option.nativeLabel}
						</DropdownMenuRadioItem>
					))}
				</DropdownMenuRadioGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
