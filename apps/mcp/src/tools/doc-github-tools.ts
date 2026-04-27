import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type {
	PacaAPIDocClient,
	PacaAPIGitHubClient,
} from "../api/index.js";
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

const GetGitHubIntegrationSchema = z.object({
	projectId: z.string(),
});

const SetGitHubTokenSchema = z.object({
	projectId: z.string(),
	token: z.string(),
});

const DeleteGitHubTokenSchema = z.object({
	projectId: z.string(),
});

const ListGitHubRepositoriesSchema = z.object({
	projectId: z.string(),
});

const ListLinkedGitHubReposSchema = z.object({
	projectId: z.string(),
});

const LinkGitHubRepositorySchema = z.object({
	projectId: z.string(),
	owner: z.string(),
	repo: z.string(),
});

const UnlinkGitHubRepositorySchema = z.object({
	projectId: z.string(),
	repoId: z.number(),
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
						description: "The ID of the project",
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
						description: "The ID of the project",
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
						description: "The ID of the project",
					},
					folderId: {
						type: "string",
						description: "The ID of the folder",
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
						description: "The ID of the project",
					},
					folderId: {
						type: "string",
						description: "The ID of the folder",
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
						description: "The ID of the project",
					},
					docId: {
						type: "string",
						description: "The ID of the document",
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
						description: "The ID of the project",
					},
					docId: {
						type: "string",
						description: "The ID of the document",
					},
					snapshotId: {
						type: "string",
						description: "The ID of the snapshot",
					},
				},
				required: ["projectId", "docId", "snapshotId"],
			},
		},
	];
}

/**
 * Returns all GitHub integration related MCP tools.
 */
export function getGitHubTools(): Tool[] {
	return [
		{
			name: "get_github_integration",
			description: "Get GitHub integration status for a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "set_github_token",
			description: "Set GitHub token for a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					token: {
						type: "string",
						description: "The GitHub personal access token",
					},
				},
				required: ["projectId", "token"],
			},
		},
		{
			name: "delete_github_token",
			description: "Delete GitHub token for a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "list_github_repositories",
			description: "List available GitHub repositories",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "list_linked_github_repos",
			description: "List linked GitHub repositories",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "link_github_repository",
			description: "Link a GitHub repository to a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					owner: {
						type: "string",
						description: "The repository owner",
					},
					repo: {
						type: "string",
						description: "The repository name",
					},
				},
				required: ["projectId", "owner", "repo"],
			},
		},
		{
			name: "unlink_github_repository",
			description: "Unlink a GitHub repository from a project",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description: "The ID of the project",
					},
					repoId: {
						type: "number",
						description: "The repository ID",
					},
				},
				required: ["projectId", "repoId"],
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

function formatGitHubIntegration(integration: any): string {
	return `GitHub Integration:
ID: ${integration.id}
Project ID: ${integration.project_id}
Created: ${integration.created_at}
Updated: ${integration.updated_at}`;
}

function formatGitHubRepo(repo: any): string {
	return `Repository: ${repo.full_name}
ID: ${repo.id}
Owner: ${repo.owner}
Repo Name: ${repo.repo_name}
Full Name: ${repo.full_name}
Private: ${repo.private ? "Yes" : "No"}
Default Branch: ${repo.default_branch}
Webhook ID: ${repo.webhook_id}
Created: ${repo.created_at}`;
}

/**
 * Handles document and GitHub tool calls.
 */
export async function handleDocTool(
	toolName: string,
	args: any,
	docClient: PacaAPIDocClient,
	githubClient: PacaAPIGitHubClient,
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

		case "get_github_integration": {
			const { projectId } = GetGitHubIntegrationSchema.parse(args);
			const integration = await githubClient.getGitHubIntegration(projectId);
			return {
				content: [
					{
						type: "text",
						text: formatGitHubIntegration(integration),
					},
				],
			};
		}

		case "set_github_token": {
			const { projectId, token } = SetGitHubTokenSchema.parse(args);
			await githubClient.setGitHubToken(projectId, { token });
			return {
				content: [
					{
						type: "text",
						text: `GitHub token set successfully`,
					},
				],
			};
		}

		case "delete_github_token": {
			const { projectId } = DeleteGitHubTokenSchema.parse(args);
			await githubClient.deleteGitHubToken(projectId);
			return {
				content: [
					{
						type: "text",
						text: `GitHub token deleted successfully`,
					},
				],
			};
		}

		case "list_github_repositories": {
			const { projectId } = ListGitHubRepositoriesSchema.parse(args);
			const repos = await githubClient.listRepositories(projectId);
			const formatted = formatList(repos, formatGitHubRepo);
			return {
				content: [
					{
						type: "text",
						text: `GitHub Repositories:\n\n${formatted}`,
					},
				],
			};
		}

		case "list_linked_github_repos": {
			const { projectId } = ListLinkedGitHubReposSchema.parse(args);
			const repos = await githubClient.listLinkedRepositories(projectId);
			const formatted = formatList(repos, formatGitHubRepo);
			return {
				content: [
					{
						type: "text",
						text: `Linked GitHub Repositories:\n\n${formatted}`,
					},
				],
			};
		}

		case "link_github_repository": {
			const { projectId, owner, repo } = LinkGitHubRepositorySchema.parse(args);
			const linkedRepo = await githubClient.linkRepository(projectId, {
				owner,
				repo_name: repo,
			});
			return {
				content: [
					{
						type: "text",
						text: `Repository linked successfully:\n\n${formatGitHubRepo(linkedRepo)}`,
					},
				],
			};
		}

		case "unlink_github_repository": {
			const { projectId, repoId } = UnlinkGitHubRepositorySchema.parse(args);
			await githubClient.unlinkRepository(projectId, String(repoId));
			return {
				content: [
					{
						type: "text",
						text: `Repository ${repoId} unlinked successfully`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown doc/GitHub tool: ${toolName}`);
	}
}
