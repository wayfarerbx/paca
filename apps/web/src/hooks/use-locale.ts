import { useEffect, useState } from "react";
import i18n, { SUPPORTED_LANGUAGES, type SupportedLanguage } from "@/i18n";

export interface LocaleOption {
	code: SupportedLanguage;
	label: string;
	nativeLabel: string;
}

export const SUPPORTED_LOCALES: LocaleOption[] = [
	{ code: "en", label: "English", nativeLabel: "English" },
	{ code: "vi", label: "Vietnamese", nativeLabel: "Tiếng Việt" },
	{ code: "ko", label: "Korean", nativeLabel: "한국어" },
	{ code: "zh-CN", label: "Chinese (Simplified)", nativeLabel: "简体中文" },
	{ code: "ja", label: "Japanese", nativeLabel: "日本語" },
	{ code: "es", label: "Spanish", nativeLabel: "Español" },
	{ code: "fr", label: "French", nativeLabel: "Français" },
];

function resolveLocale(language: string): SupportedLanguage {
	const exact = SUPPORTED_LANGUAGES.find((code) => code === language);
	if (exact) return exact;

	const base = language.split("-")[0];
	const baseMatch = SUPPORTED_LANGUAGES.find((code) => code === base);
	return baseMatch ?? "en";
}

export function useLocale() {
	const [locale, setLocale] = useState<SupportedLanguage>(() =>
		resolveLocale(i18n.language),
	);

	useEffect(() => {
		const onLanguageChanged = (language: string) => {
			setLocale(resolveLocale(language));
		};

		i18n.on("languageChanged", onLanguageChanged);
		return () => {
			i18n.off("languageChanged", onLanguageChanged);
		};
	}, []);

	function set(next: SupportedLanguage) {
		void i18n.changeLanguage(next);
	}

	return { locale, set, supportedLocales: SUPPORTED_LOCALES };
}
