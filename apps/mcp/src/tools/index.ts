import type { CallToolRequest, Tool } from "@modelcontextprotocol/sdk/types.js";
import type {
	PacaAPIClient,
	PacaAPIDocClient,
	PacaAPIExtendedClient,
	PacaAPITaskExtendedClient,
	PacaAPIViewsClient,
} from "../api/index.js";
import {
	getAttachmentTools,
	handleAttachmentTool,
} from "./attachment-tools.js";
import { getDocTools, handleDocTool } from "./doc-github-tools.js";
import { getDocumentTools, handleDocumentTool } from "./document-tools.js";
import {
	getProjectMemberTools,
	getProjectRoleTools,
	handleProjectMemberTool,
} from "./member-tools.js";
import { getProjectTools, handleProjectTool } from "./project-tools.js";
import { getSprintTools, handleSprintTool } from "./sprint-tools.js";
import {
	getTaskActivityTools,
	handleTaskActivityTool,
} from "./task-activity-tools.js";
import { getTaskTools, handleTaskTool } from "./task-tools.js";
import {
	getTaskStatusTools,
	getTaskTypeTools,
	handleTaskTypeTool,
} from "./task-type-tools.js";
import {
	getCustomFieldTools,
	getViewTools,
	handleViewTool,
} from "./view-tools.js";

/**
 * Returns all available MCP tools.
 */
export function getAllTools(): Tool[] {
	return [
		...getProjectTools(),
		...getTaskTools(),
		...getSprintTools(),
		...getDocumentTools(),
		...getProjectMemberTools(),
		...getProjectRoleTools(),
		...getTaskTypeTools(),
		...getTaskStatusTools(),
		...getViewTools(),
		...getCustomFieldTools(),
		...getAttachmentTools(),
		...getDocTools(),
		...getTaskActivityTools(),
	];
}

/**
 * Handles incoming tool calls.
 * Routes the call to the appropriate handler based on tool name.
 */
export async function handleToolCall(
	request: CallToolRequest,
	clients: {
		apiClient: PacaAPIClient;
		extendedClient: PacaAPIExtendedClient;
		viewsClient: PacaAPIViewsClient;
		taskExtendedClient: PacaAPITaskExtendedClient;
		docClient: PacaAPIDocClient;
	},
): Promise<any> {
	const { name, arguments: args } = request.params;

	try {
		// Project tools
		if (
			name === "list_projects" ||
			name === "get_project" ||
			name === "create_project" ||
			name === "update_project" ||
			name === "delete_project"
		) {
			return handleProjectTool(name, args, clients.apiClient);
		}

		// Task tools
		if (
			name === "list_tasks" ||
			name === "get_task" ||
			name === "get_task_by_number" ||
			name === "create_task" ||
			name === "update_task" ||
			name === "delete_task"
		) {
			return handleTaskTool(
				name,
				args,
				clients.apiClient,
				clients.taskExtendedClient,
				clients.viewsClient,
			);
		}

		// Sprint tools
		if (
			name === "list_sprints" ||
			name === "get_sprint" ||
			name === "create_sprint" ||
			name === "update_sprint" ||
			name === "delete_sprint" ||
			name === "complete_sprint"
		) {
			return handleSprintTool(name, args, clients.apiClient);
		}

		// Document tools
		if (
			name === "list_documents" ||
			name === "get_document" ||
			name === "create_document" ||
			name === "update_document" ||
			name === "delete_document"
		) {
			return handleDocumentTool(name, args, clients.apiClient);
		}

		// Project member and role tools
		if (
			name === "list_project_members" ||
			name === "add_project_member" ||
			name === "get_my_project_permissions" ||
			name === "update_project_member_role" ||
			name === "remove_project_member" ||
			name === "list_project_roles" ||
			name === "create_project_role" ||
			name === "update_project_role" ||
			name === "delete_project_role"
		) {
			return handleProjectMemberTool(name, args, clients.extendedClient);
		}

		// Task type and status tools
		if (
			name === "list_task_types" ||
			name === "create_task_type" ||
			name === "update_task_type" ||
			name === "delete_task_type" ||
			name === "set_default_task_type" ||
			name === "list_task_statuses" ||
			name === "create_task_status" ||
			name === "update_task_status" ||
			name === "delete_task_status" ||
			name === "set_default_task_status"
		) {
			return handleTaskTypeTool(name, args, clients.extendedClient);
		}

		// View and custom field tools
		if (
			name === "list_views" ||
			name === "create_view" ||
			name === "reorder_views" ||
			name === "get_view" ||
			name === "update_view" ||
			name === "delete_view" ||
			name === "list_task_positions" ||
			name === "bulk_move_tasks" ||
			name === "move_task" ||
			name === "list_custom_fields" ||
			name === "create_custom_field" ||
			name === "get_custom_field" ||
			name === "update_custom_field" ||
			name === "delete_custom_field"
		) {
			return handleViewTool(name, args, clients.viewsClient);
		}

		// Attachment tools
		if (
			name === "list_task_attachments" ||
			name === "get_attachment_download_url" ||
			name === "delete_task_attachment"
		) {
			return handleAttachmentTool(name, args, clients.viewsClient);
		}

		// Document tools
		if (
			name === "list_doc_folders" ||
			name === "create_doc_folder" ||
			name === "update_doc_folder" ||
			name === "delete_doc_folder" ||
			name === "list_doc_snapshots" ||
			name === "get_doc_snapshot"
		) {
			return handleDocTool(name, args, clients.docClient);
		}

		// Task activity tools
		if (
			name === "list_task_activities" ||
			name === "add_task_comment" ||
			name === "update_task_comment" ||
			name === "delete_task_comment"
		) {
			return handleTaskActivityTool(name, args, clients.taskExtendedClient);
		}

		throw new Error(`Unknown tool: ${name}`);
	} catch (error) {
		const errorMessage =
			error instanceof Error ? error.message : "Unknown error";
		return {
			content: [
				{
					type: "text",
					text: `Error: ${errorMessage}`,
				},
			],
			isError: true,
		};
	}
}
