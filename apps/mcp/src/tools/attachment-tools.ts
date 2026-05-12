import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type {
	PacaAPIViewsClient,
} from "../api/index.js";
import { formatList } from "../utils/index.js";

const ListTaskAttachmentsSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const GetAttachmentDownloadURLSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	attachmentId: z.string(),
});

const DeleteTaskAttachmentSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	attachmentId: z.string(),
});

/**
 * Returns all attachment-related MCP tools.
 */
export function getAttachmentTools(): Tool[] {
	return [
		{
			name: "list_task_attachments",
			description: "List all attachments for a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description: "The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
		{
			name: "get_attachment_download_url",
			description: "Get a download URL for an attachment",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description: "The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
					attachmentId: {
						type: "string",
						description: "The technical UUID of the attachment (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_attachments to get the attachment ID.",
					},
				},
				required: ["projectId", "taskId", "attachmentId"],
			},
		},
		{
			name: "delete_task_attachment",
			description: "Delete an attachment from a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description: "The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
					attachmentId: {
						type: "string",
						description: "The technical UUID of the attachment (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_attachments to get the attachment ID.",
					},
				},
				required: ["projectId", "taskId", "attachmentId"],
			},
		},
	];
}

function formatAttachment(attachment: any): string {
	return `Attachment: ${attachment.file?.file_name || "Unknown"}
ID: ${attachment.id}
Size: ${attachment.file?.file_size || 0} bytes
Type: ${attachment.file?.content_type || "Unknown"}
Uploaded by: ${attachment.created_by || "Unknown"}
Uploaded at: ${attachment.created_at}`;
}

/**
 * Handles attachment tool calls.
 */
export async function handleAttachmentTool(
	toolName: string,
	args: any,
	viewsClient: PacaAPIViewsClient,
): Promise<any> {
	switch (toolName) {
		case "list_task_attachments": {
			const { projectId, taskId } = ListTaskAttachmentsSchema.parse(args);
			const attachments = await viewsClient.listTaskAttachments(
				projectId,
				taskId,
			);
			const formatted = formatList(attachments, formatAttachment);
			return {
				content: [
					{
						type: "text",
						text: `Attachments:\n\n${formatted}`,
					},
				],
			};
		}

		case "get_attachment_download_url": {
			const { projectId, taskId, attachmentId } =
				GetAttachmentDownloadURLSchema.parse(args);
			const result = await viewsClient.getAttachmentDownloadURL(
				projectId,
				taskId,
				attachmentId,
			);
			return {
				content: [
					{
						type: "text",
						text: `Download URL: ${result}`,
					},
				],
			};
		}

		case "delete_task_attachment": {
			const { projectId, taskId, attachmentId } =
				DeleteTaskAttachmentSchema.parse(args);
			await viewsClient.deleteTaskAttachment(projectId, taskId, attachmentId);
			return {
				content: [
					{
						type: "text",
						text: `Attachment ${attachmentId} deleted successfully`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown attachment tool: ${toolName}`);
	}
}
