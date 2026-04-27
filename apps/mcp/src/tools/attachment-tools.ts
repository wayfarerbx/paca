import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type {
	PacaAPITaskExtendedClient,
	PacaAPIViewsClient,
} from "../api/index.js";
import { formatList } from "../utils/index.js";

const ListTaskAttachmentsSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const InitiateAttachmentUploadSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	fileName: z.string(),
	fileSize: z.number(),
	mimeType: z.string(),
});

const CompleteAttachmentUploadSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	uploadId: z.string(),
	key: z.string(),
	fileName: z.string(),
	fileSize: z.number(),
	mimeType: z.string(),
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

const ListBDDScenariosSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const CreateBDDScenarioSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	title: z.string(),
	given: z.string(),
	when: z.string(),
	// biome-ignore lint/suspicious/noThenProperty: BDD scenario uses "then" as domain terminology
	then: z.string(),
});

const GetBDDScenarioSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	scenarioId: z.string(),
});

const UpdateBDDScenarioSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	scenarioId: z.string(),
	title: z.string().optional(),
	given: z.string().optional(),
	when: z.string().optional(),
	// biome-ignore lint/suspicious/noThenProperty: BDD scenario uses "then" as domain terminology
	then: z.string().optional(),
});

const DeleteBDDScenarioSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	scenarioId: z.string(),
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
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
		{
			name: "initiate_attachment_upload",
			description: "Initiate an attachment upload for a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					fileName: {
						type: "string",
						description: "The name of the file",
					},
					fileSize: {
						type: "number",
						description: "The size of the file in bytes",
					},
					mimeType: {
						type: "string",
						description: "The MIME type of the file",
					},
				},
				required: ["projectId", "taskId", "fileName", "fileSize", "mimeType"],
			},
		},
		{
			name: "complete_attachment_upload",
			description: "Complete an attachment upload for a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					uploadId: {
						type: "string",
						description: "The upload ID",
					},
					key: {
						type: "string",
						description: "The file key",
					},
					fileName: {
						type: "string",
						description: "The name of the file",
					},
					fileSize: {
						type: "number",
						description: "The size of the file in bytes",
					},
					mimeType: {
						type: "string",
						description: "The MIME type of the file",
					},
				},
				required: [
					"projectId",
					"taskId",
					"uploadId",
					"key",
					"fileName",
					"fileSize",
					"mimeType",
				],
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
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					attachmentId: {
						type: "string",
						description: "The ID of the attachment",
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
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					attachmentId: {
						type: "string",
						description: "The ID of the attachment",
					},
				},
				required: ["projectId", "taskId", "attachmentId"],
			},
		},
	];
}

/**
 * Returns all BDD scenario-related MCP tools.
 */
export function getBDDScenarioTools(): Tool[] {
	return [
		{
			name: "list_bdd_scenarios",
			description: "List all BDD scenarios for a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
		{
			name: "create_bdd_scenario",
			description: "Create a new BDD scenario for a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					title: {
						type: "string",
						description: "The title of the scenario",
					},
					given: {
						type: "string",
						description: "The Given clause",
					},
					when: {
						type: "string",
						description: "The When clause",
					},
					// biome-ignore lint/suspicious/noThenProperty: BDD scenario uses "then" as domain terminology
					then: {
						type: "string",
						description: "The Then clause",
					},
				},
				required: ["projectId", "taskId", "title", "given", "when", "then"],
			},
		},
		{
			name: "get_bdd_scenario",
			description: "Get details of a specific BDD scenario",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					scenarioId: {
						type: "string",
						description: "The ID of the scenario",
					},
				},
				required: ["projectId", "taskId", "scenarioId"],
			},
		},
		{
			name: "update_bdd_scenario",
			description: "Update an existing BDD scenario",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					scenarioId: {
						type: "string",
						description: "The ID of the scenario",
					},
					title: {
						type: "string",
						description: "The new title",
					},
					given: {
						type: "string",
						description: "The new Given clause",
					},
					when: {
						type: "string",
						description: "The new When clause",
					},
					// biome-ignore lint/suspicious/noThenProperty: BDD scenario uses "then" as domain terminology
					then: {
						type: "string",
						description: "The new Then clause",
					},
				},
				required: ["projectId", "taskId", "scenarioId"],
			},
		},
		{
			name: "delete_bdd_scenario",
			description: "Delete a BDD scenario",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					taskId: {
						type: "string",
						description: "The ID of the task",
					},
					scenarioId: {
						type: "string",
						description: "The ID of the scenario",
					},
				},
				required: ["projectId", "taskId", "scenarioId"],
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

function formatBDDScenario(scenario: any): string {
	return `BDD Scenario: ${scenario.title}
ID: ${scenario.id}

Given:
${scenario.given || "None"}

When:
${scenario.when || "None"}

Then:
${scenario.then || "None"}

Created: ${scenario.created_at}`;
}

/**
 * Handles attachment and BDD scenario tool calls.
 */
export async function handleAttachmentTool(
	toolName: string,
	args: any,
	viewsClient: PacaAPIViewsClient,
	taskClient: PacaAPITaskExtendedClient,
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

		case "initiate_attachment_upload": {
			const { projectId, taskId, fileName, fileSize, mimeType } =
				InitiateAttachmentUploadSchema.parse(args);
			const result = await viewsClient.initiateUpload(
				projectId,
				taskId,
				fileName,
				fileSize,
				mimeType,
			);
			return {
				content: [
					{
						type: "text",
						text: `Upload initiated:\n\n${JSON.stringify(result, null, 2)}`,
					},
				],
			};
		}

		case "complete_attachment_upload": {
			const { projectId, taskId, uploadId, key, fileName, fileSize, mimeType } =
				CompleteAttachmentUploadSchema.parse(args);
			const attachment = await viewsClient.completeUpload(projectId, taskId, {
				upload_id: uploadId,
				key,
				file_name: fileName,
				file_size: fileSize,
				content_type: mimeType,
			});
			return {
				content: [
					{
						type: "text",
						text: `Attachment uploaded successfully:\n\n${formatAttachment(attachment)}`,
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

		case "list_bdd_scenarios": {
			const { projectId, taskId } = ListBDDScenariosSchema.parse(args);
			const scenarios = await taskClient.listBDDScenarios(projectId, taskId);
			const formatted = formatList(scenarios, formatBDDScenario);
			return {
				content: [
					{
						type: "text",
						text: `BDD Scenarios:\n\n${formatted}`,
					},
				],
			};
		}

		case "create_bdd_scenario": {
			const { projectId, taskId, title, given, when, then } =
				CreateBDDScenarioSchema.parse(args);
			const scenario = await taskClient.createBDDScenario(projectId, taskId, {
				title,
				given,
				when,
				then,
			});
			return {
				content: [
					{
						type: "text",
						text: `BDD scenario created successfully:\n\n${formatBDDScenario(scenario)}`,
					},
				],
			};
		}

		case "get_bdd_scenario": {
			const { projectId, taskId, scenarioId } =
				GetBDDScenarioSchema.parse(args);
			const scenario = await taskClient.getBDDScenario(
				projectId,
				taskId,
				scenarioId,
			);
			return {
				content: [
					{
						type: "text",
						text: formatBDDScenario(scenario),
					},
				],
			};
		}

		case "update_bdd_scenario": {
			const { projectId, taskId, scenarioId, title, given, when, then } =
				UpdateBDDScenarioSchema.parse(args);
			const scenario = await taskClient.updateBDDScenario(
				projectId,
				taskId,
				scenarioId,
				{
					title,
					given,
					when,
					then,
				},
			);
			return {
				content: [
					{
						type: "text",
						text: `BDD scenario updated successfully:\n\n${formatBDDScenario(scenario)}`,
					},
				],
			};
		}

		case "delete_bdd_scenario": {
			const { projectId, taskId, scenarioId } =
				DeleteBDDScenarioSchema.parse(args);
			await taskClient.deleteBDDScenario(projectId, taskId, scenarioId);
			return {
				content: [
					{
						type: "text",
						text: `BDD scenario ${scenarioId} deleted successfully`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown attachment/BDD tool: ${toolName}`);
	}
}
