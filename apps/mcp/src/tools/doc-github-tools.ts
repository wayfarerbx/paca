import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPIDocClient } from "../api/index.js";
import { formatList } from "../utils/index.js";

const ListDocFoldersSchema = z.object({
	projectId: z.string(),
});

const CreateDocFolderSchema = z.object({
	projectId: z.string(),
	name: z.string(),
	parentId: z.string().optional(),
});

const UpdateDocFolderSchema = z.object({
	projectId: z.string(),
	folderId: z.string(),
	name: z.string().optional(),
	parentId: z.string().optional(),
	position: z.number().optional(),
});

const DeleteDocFolderSchema = z.object({
	projectId: z.string(),
	folderId: z.string(),
});

const ListDocSnapshotsSchema = z.object({
	projectId: z.string(),
	docId: z.string(),
});

const GetDocSnapshotSchema = z.object({
	projectId: z.string(),
	docId: z.string(),
	snapshotId: z.string(),
});

/**
 * Returns all document folder and file related MCP tools.
 */
export function getDocTools(): Tool[] {
	return [
		{
			name: "list_doc_folders",
			description: "List all folders in a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "create_doc_folder",
			description: "Create a new document folder",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					name: {
						type: "string",
						description: "The name of the folder",
					},
					parentId: {
						type: "string",
						description: "The parent folder ID (null for root level)",
					},
				},
				required: ["projectId", "name"],
			},
		},
		{
			name: "update_doc_folder",
			description: "Update a document folder",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					folderId: {
						type: "string",
						description: "The technical UUID of the folder (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_doc_folders to get the folder ID.",
					},
					name: {
						type: "string",
						description: "The new name",
					},
					parentId: {
						type: "string",
						description: "The new parent folder ID",
					},
					position: {
						type: "number",
						description: "The new position",
					},
				},
				required: ["projectId", "folderId"],
			},
		},
		{
			name: "delete_doc_folder",
			description: "Delete a document folder",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					folderId: {
						type: "string",
						description: "The technical UUID of the folder (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_doc_folders to get the folder ID.",
					},
				},
				required: ["projectId", "folderId"],
			},
		},
		{
			name: "list_doc_snapshots",
			description: "List all snapshots of a document",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					docId: {
						type: "string",
						description: "The technical UUID of the document (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_documents to get the document ID.",
					},
				},
				required: ["projectId", "docId"],
			},
		},
		{
			name: "get_doc_snapshot",
			description: "Get a specific document snapshot",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					docId: {
						type: "string",
						description: "The technical UUID of the document (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_documents to get the document ID.",
					},
					snapshotId: {
						type: "string",
						description: "The technical UUID of the snapshot (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_doc_snapshots to get the snapshot ID.",
					},
				},
				required: ["projectId", "docId", "snapshotId"],
			},
		},
	];
}

function formatDocFolder(folder: any): string {
	return `Folder: ${folder.name}
ID: ${folder.id}
Parent ID: ${folder.parent_id || "None (Root)"}
Position: ${folder.position}
Created by: ${folder.created_by || "Unknown"}
Created: ${folder.created_at}`;
}

function formatDocSnapshot(snapshot: any): string {
	return `Snapshot #${snapshot.snapshot_number}
ID: ${snapshot.id}
Title: ${snapshot.title}
Created by: ${snapshot.created_by_name}
Created: ${snapshot.created_at}`;
}

/**
 * Handles document tool calls.
 */
export async function handleDocTool(
	toolName: string,
	args: any,
	docClient: PacaAPIDocClient,
): Promise<any> {
	switch (toolName) {
		case "list_doc_folders": {
			const { projectId } = ListDocFoldersSchema.parse(args);
			const folders = await docClient.listFolders(projectId);
			const formatted = formatList(folders, formatDocFolder);
			return {
				content: [
					{
						type: "text",
						text: `Document Folders:\n\n${formatted}`,
					},
				],
			};
		}

		case "create_doc_folder": {
			const { projectId, name, parentId } = CreateDocFolderSchema.parse(args);
			const folder = await docClient.createFolder(projectId, {
				name,
				parent_id: parentId,
			});
			return {
				content: [
					{
						type: "text",
						text: `Folder created successfully:\n\n${formatDocFolder(folder)}`,
					},
				],
			};
		}

		case "update_doc_folder": {
			const { projectId, folderId, name, parentId, position } =
				UpdateDocFolderSchema.parse(args);
			const folder = await docClient.updateFolder(projectId, folderId, {
				name,
				parent_id: parentId,
				position,
			});
			return {
				content: [
					{
						type: "text",
						text: `Folder updated successfully:\n\n${formatDocFolder(folder)}`,
					},
				],
			};
		}

		case "delete_doc_folder": {
			const { projectId, folderId } = DeleteDocFolderSchema.parse(args);
			await docClient.deleteFolder(projectId, folderId);
			return {
				content: [
					{
						type: "text",
						text: `Folder ${folderId} deleted successfully`,
					},
				],
			};
		}

		case "list_doc_snapshots": {
			const { projectId, docId } = ListDocSnapshotsSchema.parse(args);
			const snapshots = await docClient.listSnapshots(projectId, docId);
			const formatted = formatList(snapshots, formatDocSnapshot);
			return {
				content: [
					{
						type: "text",
						text: `Document Snapshots:\n\n${formatted}`,
					},
				],
			};
		}

		case "get_doc_snapshot": {
			const { projectId, docId, snapshotId } = GetDocSnapshotSchema.parse(args);
			const snapshot = await docClient.getSnapshot(
				projectId,
				docId,
				snapshotId,
			);
			return {
				content: [
					{
						type: "text",
						text: formatDocSnapshot(snapshot),
					},
				],
			};
		}

		default:
			throw new Error(`Unknown doc tool: ${toolName}`);
	}
}
