import { describe, expect, it } from "vitest";
import { normalizeBlockContent } from "./comment-blocknote";

describe("normalizeBlockContent", () => {
	it("returns the array unchanged when content is already a block array", () => {
		const blocks = [{ type: "paragraph", content: [] }];
		expect(normalizeBlockContent(blocks)).toBe(blocks);
	});

	it("wraps a plain string into a paragraph block instead of dropping it", () => {
		// This is the exact shape reported in GitHub issue #233: a plain string
		// stored as description/comment content, which used to crash BlockNote's
		// `.map()` calls over blocks.
		const result = normalizeBlockContent("just a plain string") as Array<{
			type: string;
			content: Array<{ text: string }>;
		}>;
		expect(result).toHaveLength(1);
		expect(result[0].type).toBe("paragraph");
		expect(result[0].content[0].text).toBe("just a plain string");
	});

	it("returns an empty array for null, undefined, empty string, or other scalars", () => {
		expect(normalizeBlockContent(null)).toEqual([]);
		expect(normalizeBlockContent(undefined)).toEqual([]);
		expect(normalizeBlockContent("")).toEqual([]);
		expect(normalizeBlockContent(42)).toEqual([]);
		expect(normalizeBlockContent({ type: "paragraph" })).toEqual([]);
	});
});
