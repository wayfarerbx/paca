import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPIClient } from "../api/index.js";
import { formatDocument, formatList } from "../utils/index.js";

const ListDocumentsSchema = z.object({
	projectId: z.string(),
});

const GetDocumentSchema = z.object({
	projectId: z.string(),
	docId: z.string(),
});

const CreateDocumentSchema = z.object({
	projectId: z.string(),
	title: z.string(),
	content: z.string().optional(),
	folderId: z.string().optional(),
});

const UpdateDocumentSchema = z.object({
	projectId: z.string(),
	docId: z.string(),
	title: z.string().optional(),
	content: z.string().optional(),
	folderId: z.string().optional(),
});

const DeleteDocumentSchema = z.object({
	projectId: z.string(),
	docId: z.string(),
});

/**
 * Returns all document-related MCP tools.
 * @returns Array of document tools
 */
export function getDocumentTools(): Tool[] {
	return [
		{
			name: "list_documents",
			description: "List all documents in a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "get_document",
			description: "Get details of a specific document",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					docId: {
						type: "string",
						description:
							"The technical UUID of the document (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_documents to get the document ID.",
					},
				},
				required: ["projectId", "docId"],
			},
		},
		{
			name: "create_document",
			description: "Create a new document",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					title: {
						type: "string",
						description: "The title of the document",
					},
					content: {
						type: "string",
						description:
							"The content of the document in markdown format (will be converted to BlockNote format)",
					},
					folderId: {
						type: "string",
						description:
							"The technical UUID of the folder (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_doc_folders to get the folder ID.",
					},
				},
				required: ["projectId", "title"],
			},
		},
		{
			name: "update_document",
			description: "Update an existing document",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					docId: {
						type: "string",
						description:
							"The technical UUID of the document (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_documents to get the document ID.",
					},
					title: {
						type: "string",
						description: "The new title of the document",
					},
					content: {
						type: "string",
						description:
							"The new content of the document in markdown format (will be converted to BlockNote format)",
					},
					folderId: {
						type: "string",
						description:
							"The technical UUID of the folder (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_doc_folders to get the folder ID.",
					},
				},
				required: ["projectId", "docId"],
			},
		},
		{
			name: "delete_document",
			description: "Delete a document",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					docId: {
						type: "string",
						description:
							"The technical UUID of the document (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_documents to get the document ID.",
					},
				},
				required: ["projectId", "docId"],
			},
		},
	];
}

/**
 * Handles document-related tool calls.
 * @param toolName - Name of the tool being called
 * @param args - Tool arguments
 * @param client - Paca API client instance
 * @returns Tool response
 */
export async function handleDocumentTool(
	toolName: string,
	args: any,
	client: PacaAPIClient,
): Promise<any> {
	switch (toolName) {
		case "list_documents": {
			const { projectId } = ListDocumentsSchema.parse(args);
			const documents = await client.listDocuments(projectId);
			const formatted = formatList(documents, formatDocument);
			return {
				content: [
					{
						type: "text",
						text: `Documents:\n\n${formatted}`,
					},
				],
			};
		}

		case "get_document": {
			const { projectId, docId } = GetDocumentSchema.parse(args);
			const document = await client.getDocument(projectId, docId);
			return {
				content: [
					{
						type: "text",
						text: formatDocument(document),
					},
				],
			};
		}

		case "create_document": {
			const { projectId, title, content, folderId } =
				CreateDocumentSchema.parse(args);
			const document = await client.createDocument({
				project_id: projectId,
				title,
				content,
				folder_id: folderId,
			});
			return {
				content: [
					{
						type: "text",
						text: `Document created successfully:\n\n${formatDocument(
							document,
						)}`,
					},
				],
			};
		}

		case "update_document": {
			const { projectId, docId, title, content, folderId } =
				UpdateDocumentSchema.parse(args);
			const document = await client.updateDocument(projectId, docId, {
				title,
				content,
				folder_id: folderId,
			});
			return {
				content: [
					{
						type: "text",
						text: `Document updated successfully:\n\n${formatDocument(
							document,
						)}`,
					},
				],
			};
		}

		case "delete_document": {
			const { projectId, docId } = DeleteDocumentSchema.parse(args);
			await client.deleteDocument(projectId, docId);
			return {
				content: [
					{
						type: "text",
						text: `Document ${docId} deleted successfully`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown document tool: ${toolName}`);
	}
}
