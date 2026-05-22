import { BlockNoteEditor } from "@blocknote/core";
import { JSDOM } from "jsdom";

// Initialize JSDOM to provide a browser-like environment for BlockNote
const dom = new JSDOM("");
const { window } = dom;

const globals = {
	window,
	document: window.document,
	navigator: window.navigator,
	HTMLElement: window.HTMLElement,
	Node: window.Node,
};

for (const [key, value] of Object.entries(globals)) {
	Object.defineProperty(global, key, {
		value,
		writable: true,
		configurable: true,
	});
}

let editor: BlockNoteEditor | null = null;

function getEditor(): BlockNoteEditor {
	if (!editor) {
		editor = BlockNoteEditor.create();
	}
	return editor;
}

/**
 * Recursively converts mentions to text format with IDs in the blocks.
 * @param content - Inline content array or single content item
 * @returns Converted content
 */
function convertMentionsToText(content: any): any {
	if (!content) return content;

	if (Array.isArray(content)) {
		return content.map((item) => convertMentionsToText(item));
	}

	if (typeof content === "object" && content !== null) {
		const result = { ...content };

		if (result.type === "teamMention") {
			const memberName = result.props?.name || "Unknown";
			const memberId = result.props?.id || "";
			const text = memberId
				? `@${memberName} (id: ${memberId})`
				: `@${memberName}`;
			return {
				type: "text",
				text,
				styles: {},
			};
		}

		if (result.type === "taskReference") {
			const taskTitle = result.props?.title || "Unknown";
			const taskId = result.props?.id || "";
			const text = taskId ? `#${taskTitle} (id: ${taskId})` : `#${taskTitle}`;
			return {
				type: "text",
				text,
				styles: {},
			};
		}

		if (result.type === "docReference") {
			const docTitle = result.props?.title || "Unknown";
			const docId = result.props?.id || "";
			const text = docId ? `📄 ${docTitle} (id: ${docId})` : `📄 ${docTitle}`;
			return {
				type: "text",
				text,
				styles: {},
			};
		}

		if (result.content) {
			result.content = convertMentionsToText(result.content);
		}

		if (result.children) {
			result.children = convertMentionsToText(result.children);
		}

		return result;
	}

	return content;
}

/**
 * Checks if blocks contain any mention types.
 * @param blocks - Array of BlockNote block objects
 * @returns True if mentions are found
 */
function hasMentions(blocks: any[] | null): boolean {
	if (!blocks || blocks.length === 0) return false;

	for (const block of blocks) {
		if (!block) continue;

		if (
			block.type === "teamMention" ||
			block.type === "taskReference" ||
			block.type === "docReference"
		) {
			return true;
		}

		if (block.content) {
			if (hasMentionsInContent(block.content)) {
				return true;
			}
		}

		if (block.children) {
			if (hasMentions(block.children)) {
				return true;
			}
		}
	}

	return false;
}

function hasMentionsInContent(content: any): boolean {
	if (!content) return false;

	if (Array.isArray(content)) {
		for (const item of content) {
			if (hasMentionsInContent(item)) {
				return true;
			}
		}
		return false;
	}

	if (typeof content === "object" && content !== null) {
		if (
			content.type === "teamMention" ||
			content.type === "taskReference" ||
			content.type === "docReference"
		) {
			return true;
		}

		if (content.content && hasMentionsInContent(content.content)) {
			return true;
		}

		if (content.children && hasMentions(content.children)) {
			return true;
		}
	}

	return false;
}

/**
 * Converts BlockNote JSON blocks to Markdown string.
 * @param blocks - Array of BlockNote block objects
 * @returns Markdown string representation
 */
export function blocknoteToMarkdown(
	blocks: any[] | { content?: unknown; text?: unknown } | null,
): string {
	if (!blocks) return "";
	if (!Array.isArray(blocks)) {
		if (typeof blocks.text === "string") {
			return blocks.text;
		}
		if (!Array.isArray(blocks.content)) {
			return "";
		}
		blocks = blocks.content;
	}
	if (blocks.length === 0) return "";

	let _blocksToConvert = blocks;

	if (hasMentions(blocks)) {
		_blocksToConvert = convertMentionsToText(blocks);
	}

	const e = getEditor();
	return (e as any)._exportManager.blocksToMarkdownLossy(_blocksToConvert);
}

/**
 * Converts Markdown string to BlockNote JSON blocks.
 * @param markdown - Markdown string
 * @returns Array of BlockNote block objects
 */
export function markdownToBlocknote(markdown: string): any[] {
	if (!markdown || markdown.trim() === "") return [];
	const e = getEditor();
	return (e as any)._exportManager.tryParseMarkdownToBlocks(markdown);
}
