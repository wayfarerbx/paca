type BlockLike = {
	type?: string;
	props?: Record<string, unknown>;
	content?: unknown;
	children?: unknown;
	text?: unknown;
};

function inlineToText(content: unknown): string {
	if (!content) return "";
	if (typeof content === "string") return content;
	if (Array.isArray(content)) return content.map(inlineToText).join("");
	if (typeof content !== "object") return String(content);

	const item = content as BlockLike;
	if (item.type === "text" && typeof item.text === "string") return item.text;

	if (item.type === "teamMention") {
		const name = String(item.props?.name || "Unknown");
		const id = String(item.props?.id || "");
		return id ? `@${name} (id: ${id})` : `@${name}`;
	}

	if (item.type === "taskReference") {
		const title = String(item.props?.title || "Unknown");
		const id = String(item.props?.id || "");
		return id ? `#${title} (id: ${id})` : `#${title}`;
	}

	if (item.type === "docReference") {
		const title = String(item.props?.title || "Unknown");
		const id = String(item.props?.id || "");
		return id ? `📄 ${title} (id: ${id})` : `📄 ${title}`;
	}

	if (typeof item.text === "string") return item.text;
	if (item.content) return inlineToText(item.content);
	return "";
}

function blockToMarkdown(block: unknown): string {
	if (!block || typeof block !== "object") return "";
	const b = block as BlockLike;
	const text = inlineToText(b.content || b.text);
	const children = Array.isArray(b.children)
		? b.children.map(blockToMarkdown).filter(Boolean).join("\n")
		: "";

	let current = text;
	if (b.type === "heading") {
		const rawLevel = Number(b.props?.level || 1);
		const level = Math.min(Math.max(rawLevel || 1, 1), 6);
		current = `${"#".repeat(level)} ${text}`.trim();
	} else if (b.type === "bulletListItem") {
		current = `- ${text}`.trim();
	} else if (b.type === "numberedListItem") {
		current = `1. ${text}`.trim();
	} else if (b.type === "checkListItem") {
		current = `- [ ] ${text}`.trim();
	}

	return [current, children].filter(Boolean).join("\n");
}

/**
 * Converts BlockNote-like JSON blocks to readable Markdown.
 * This intentionally avoids importing @blocknote/core/jsdom so the MCP server
 * can start reliably inside lightweight Linux agent sandboxes.
 */
export function blocknoteToMarkdown(
	blocks: unknown[] | { content?: unknown; text?: unknown } | null,
): string {
	if (!blocks) return "";
	if (!Array.isArray(blocks)) {
		if (typeof blocks.text === "string") return blocks.text;
		if (!Array.isArray(blocks.content)) return "";
		blocks = blocks.content;
	}
	if (blocks.length === 0) return "";
	return blocks.map(blockToMarkdown).filter(Boolean).join("\n\n");
}

/**
 * Converts plain Markdown into simple BlockNote-compatible paragraph/heading
 * blocks. It preserves the text agents write without needing a browser runtime.
 */
export function markdownToBlocknote(markdown: string): any[] {
	if (!markdown || markdown.trim() === "") return [];

	return markdown
		.split(/\n{2,}/)
		.map((part) => part.trim())
		.filter(Boolean)
		.map((part) => {
			const heading = /^(#{1,6})\s+(.*)$/.exec(part);
			if (heading) {
				return {
					type: "heading",
					props: { level: heading[1].length },
					content: [{ type: "text", text: heading[2], styles: {} }],
					children: [],
				};
			}
			return {
				type: "paragraph",
				content: [{ type: "text", text: part, styles: {} }],
				children: [],
			};
		});
}
