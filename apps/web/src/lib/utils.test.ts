import { describe, expect, it } from "vitest";
import { cleanBlocks, cn } from "./utils";

describe("cn", () => {
	it("joins class names and ignores falsy values", () => {
		const value = cn("px-4", undefined, null, false, "py-2");
		expect(value).toBe("px-4 py-2");
	});

	it("merges conflicting Tailwind classes", () => {
		const value = cn("px-2", "px-4", "text-sm", "text-lg");
		expect(value).toBe("px-4 text-lg");
	});
});

describe("cleanBlocks", () => {
	it("returns null when blocks is null", () => {
		expect(cleanBlocks(null)).toBeNull();
	});

	it("strips id field recursively and preserves other properties", () => {
		const input = [
			{
				id: "block-1",
				type: "paragraph",
				content: [{ text: "Hello" }],
				children: [
					{
						id: "block-1-sub1",
						type: "bullet",
						content: [],
					},
				],
			},
			{
				id: "block-2",
				type: "heading",
				content: [{ text: "Title" }],
			},
		];

		const expected = [
			{
				type: "paragraph",
				content: [{ text: "Hello" }],
				children: [
					{
						type: "bullet",
						content: [],
					},
				],
			},
			{
				type: "heading",
				content: [{ text: "Title" }],
			},
		];

		expect(cleanBlocks(input)).toEqual(expected);
	});
});
