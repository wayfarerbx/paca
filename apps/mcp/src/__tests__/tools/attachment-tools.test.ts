import { describe, expect, it, vi } from "vitest";

vi.mock("../../utils/index.js", () => ({
	formatList: vi.fn((items: any[], fn: any) => items.map(fn).join("---")),
}));

import {
	getAttachmentTools,
	handleAttachmentTool,
} from "../../tools/attachment-tools.js";

const attachment = {
	id: "att-1",
	file: {
		file_name: "photo.png",
		file_size: 1024,
		content_type: "image/png",
	},
	created_by: "user-1",
	created_at: "2024-01-01T00:00:00Z",
};

function makeClient(overrides: Record<string, any> = {}) {
	return {
		listTaskAttachments: vi.fn().mockResolvedValue([attachment]),
		getAttachmentDownloadURL: vi
			.fn()
			.mockResolvedValue("https://cdn.example.com/photo.png"),
		deleteTaskAttachment: vi.fn().mockResolvedValue(undefined),
		...overrides,
	} as any;
}

// ---------------------------------------------------------------------------
// getAttachmentTools
// ---------------------------------------------------------------------------

describe("getAttachmentTools", () => {
	it("returns 3 tools", () => {
		expect(getAttachmentTools()).toHaveLength(3);
	});

	it("includes list_task_attachments, get_attachment_download_url, delete_task_attachment", () => {
		const names = getAttachmentTools().map((t) => t.name);
		expect(names).toContain("list_task_attachments");
		expect(names).toContain("get_attachment_download_url");
		expect(names).toContain("delete_task_attachment");
	});
});

// ---------------------------------------------------------------------------
// list_task_attachments
// ---------------------------------------------------------------------------

describe("handleAttachmentTool – list_task_attachments", () => {
	it("calls viewsClient.listTaskAttachments with projectId and taskId", async () => {
		const client = makeClient();
		await handleAttachmentTool(
			"list_task_attachments",
			{ projectId: "p1", taskId: "t1" },
			client,
		);
		expect(client.listTaskAttachments).toHaveBeenCalledWith("p1", "t1");
	});

	it("includes 'Attachments:' header in the response", async () => {
		const result = await handleAttachmentTool(
			"list_task_attachments",
			{ projectId: "p1", taskId: "t1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("Attachments:");
	});

	it("includes attachment file name in the formatted output", async () => {
		const result = await handleAttachmentTool(
			"list_task_attachments",
			{ projectId: "p1", taskId: "t1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("photo.png");
	});

	it("throws ZodError when projectId is missing", async () => {
		await expect(
			handleAttachmentTool(
				"list_task_attachments",
				{ taskId: "t1" },
				makeClient(),
			),
		).rejects.toThrow();
	});
});

// ---------------------------------------------------------------------------
// get_attachment_download_url
// ---------------------------------------------------------------------------

describe("handleAttachmentTool – get_attachment_download_url", () => {
	it("calls viewsClient.getAttachmentDownloadURL with all three IDs", async () => {
		const client = makeClient();
		await handleAttachmentTool(
			"get_attachment_download_url",
			{ projectId: "p1", taskId: "t1", attachmentId: "att-1" },
			client,
		);
		expect(client.getAttachmentDownloadURL).toHaveBeenCalledWith(
			"p1",
			"t1",
			"att-1",
		);
	});

	it("returns the download URL in the response text", async () => {
		const result = await handleAttachmentTool(
			"get_attachment_download_url",
			{ projectId: "p1", taskId: "t1", attachmentId: "att-1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain(
			"https://cdn.example.com/photo.png",
		);
	});
});

// ---------------------------------------------------------------------------
// delete_task_attachment
// ---------------------------------------------------------------------------

describe("handleAttachmentTool – delete_task_attachment", () => {
	it("calls viewsClient.deleteTaskAttachment with all three IDs", async () => {
		const client = makeClient();
		await handleAttachmentTool(
			"delete_task_attachment",
			{ projectId: "p1", taskId: "t1", attachmentId: "att-1" },
			client,
		);
		expect(client.deleteTaskAttachment).toHaveBeenCalledWith(
			"p1",
			"t1",
			"att-1",
		);
	});

	it("includes 'deleted successfully' in the response", async () => {
		const result = await handleAttachmentTool(
			"delete_task_attachment",
			{ projectId: "p1", taskId: "t1", attachmentId: "att-1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("deleted successfully");
		expect(result.content[0].text).toContain("att-1");
	});
});

// ---------------------------------------------------------------------------
// unknown tool
// ---------------------------------------------------------------------------

describe("handleAttachmentTool – unknown tool", () => {
	it("throws for an unknown tool name", async () => {
		await expect(
			handleAttachmentTool("bad_tool", {}, makeClient()),
		).rejects.toThrow("Unknown attachment tool");
	});
});
