import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import i18n from "@/i18n";
import LanguageToggle from "./LanguageToggle";

describe("LanguageToggle", () => {
	beforeEach(async () => {
		window.localStorage.clear();
		await act(async () => {
			await i18n.changeLanguage("en");
		});
	});

	it("lists all supported languages and switches on selection", async () => {
		render(<LanguageToggle />);
		const user = userEvent.setup();

		const trigger = screen.getByRole("button", {
			name: /language: english/i,
		});
		expect(trigger).toHaveTextContent("EN");

		await user.click(trigger);
		expect(
			await screen.findByRole("menuitemradio", {
				name: "Русский",
			}),
		).toBeInTheDocument();
		const vietnamese = await screen.findByRole("menuitemradio", {
			name: "Tiếng Việt",
		});
		await user.click(vietnamese);

		await waitFor(() => {
			expect(i18n.language).toBe("vi");
		});
		expect(window.localStorage.getItem("locale")).toBe("vi");
	});
});
