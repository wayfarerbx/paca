import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPITaskExtendedClient } from "../api/index.js";

const ListTaskActivitiesSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const AddTaskCommentSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	content: z.string(),
});

const UpdateTaskCommentSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	commentId: z.string(),
	content: z.string(),
});

const DeleteTaskCommentSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	commentId: z.string(),
});

const ListTaskPRsSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const LinkPRToTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	prId: z.number(),
	repositoryName: z.string(),
	owner: z.string(),
});

const UnlinkPRFromTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	prId: z.number(),
});

const CreateBranchForTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	branchName: z.string(),
	baseBranch: z.string().optional(),
});

const ListTaskBranchesSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

/**
 * Returns all task comment and activity related MCP tools.
 */
export function getTaskActivityTools(): Tool[] {
	return [
		{
			name: "list_task_activities",
			description: "List all activities for a task",
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
			name: "add_task_comment",
			description: "Add a comment to a task",
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
					content: {
						type: "string",
						description: "The comment content",
					},
				},
				required: ["projectId", "taskId", "content"],
			},
		},
		{
			name: "update_task_comment",
			description: "Update a task comment",
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
					commentId: {
						type: "string",
						description: "The ID of the comment",
					},
					content: {
						type: "string",
						description: "The new comment content",
					},
				},
				required: ["projectId", "taskId", "commentId", "content"],
			},
		},
		{
			name: "delete_task_comment",
			description: "Delete a task comment",
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
					commentId: {
						type: "string",
						description: "The ID of the comment",
					},
				},
				required: ["projectId", "taskId", "commentId"],
			},
		},
	];
}

/**
 * Returns all task GitHub related MCP tools.
 */
export function getTaskGitHubTools(): Tool[] {
	return [
		{
			name: "list_task_prs",
			description: "List pull requests linked to a task",
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
			name: "link_pr_to_task",
			description: "Link a pull request to a task",
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
					prId: {
						type: "number",
						description: "The pull request number",
					},
					repositoryName: {
						type: "string",
						description: "The repository name",
					},
					owner: {
						type: "string",
						description: "The repository owner",
					},
				},
				required: ["projectId", "taskId", "prId", "repositoryName", "owner"],
			},
		},
		{
			name: "unlink_pr_from_task",
			description: "Unlink a pull request from a task",
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
					prId: {
						type: "number",
						description: "The pull request number",
					},
				},
				required: ["projectId", "taskId", "prId"],
			},
		},
		{
			name: "create_branch_for_task",
			description: "Create a branch for a task",
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
					branchName: {
						type: "string",
						description: "The name of the branch to create",
					},
					baseBranch: {
						type: "string",
						description: "The base branch to branch from (optional)",
					},
				},
				required: ["projectId", "taskId", "branchName"],
			},
		},
		{
			name: "list_task_branches",
			description: "List branches for a task",
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
	];
}

function formatTaskActivity(activity: any): string {
	return `Activity: ${activity.activity_type}
ID: ${activity.id}
User: ${activity.actor_name} (${activity.actor_id})
Description: ${JSON.stringify(activity.content, null, 2)}
Created: ${activity.created_at}`;
}

function formatTaskComment(comment: any): string {
	return `Comment:
ID: ${comment.id}
User: ${comment.user_name} (${comment.user_id})
Content: ${comment.content}
Created: ${comment.created_at}
Updated: ${comment.updated_at}`;
}

function formatPullRequest(pr: any): string {
	return `Pull Request: #${pr.pr_number} - ${pr.title}
ID: ${pr.id}
State: ${pr.state}
Author: ${pr.author}
URL: ${pr.html_url}
Created: ${pr.created_at}
Merged: ${pr.merged_at ? `Yes (${pr.merged_at})` : "No"}`;
}

function formatBranch(branch: any): string {
	return `Branch: ${branch.branch_name}
Task ID: ${branch.task_id}
Repo ID: ${branch.repo_id}
Created: ${branch.created_at}`;
}

/**
 * Handles task activity, comment, and GitHub tool calls.
 */
export async function handleTaskActivityTool(
	toolName: string,
	args: any,
	client: PacaAPITaskExtendedClient,
): Promise<any> {
	switch (toolName) {
		case "list_task_activities": {
			const { projectId, taskId } = ListTaskActivitiesSchema.parse(args);
			const activities = await client.listTaskActivities(projectId, taskId);
			const formatted = activities.map(formatTaskActivity).join("\n\n---\n\n");
			return {
				content: [
					{
						type: "text",
						text: `Task Activities:\n\n${formatted}`,
					},
				],
			};
		}

		case "add_task_comment": {
			const { projectId, taskId, content } = AddTaskCommentSchema.parse(args);
			const comment = await client.addTaskComment(projectId, taskId, {
				text: content,
			});
			return {
				content: [
					{
						type: "text",
						text: `Comment added successfully:\n\n${formatTaskComment(comment)}`,
					},
				],
			};
		}

		case "update_task_comment": {
			const { projectId, taskId, commentId, content } =
				UpdateTaskCommentSchema.parse(args);
			const comment = await client.updateTaskComment(
				projectId,
				taskId,
				commentId,
				{
					text: content,
				},
			);
			return {
				content: [
					{
						type: "text",
						text: `Comment updated successfully:\n\n${formatTaskComment(comment)}`,
					},
				],
			};
		}

		case "delete_task_comment": {
			const { projectId, taskId, commentId } =
				DeleteTaskCommentSchema.parse(args);
			await client.deleteTaskComment(projectId, taskId, commentId);
			return {
				content: [
					{
						type: "text",
						text: `Comment ${commentId} deleted successfully`,
					},
				],
			};
		}

		case "list_task_prs": {
			const { projectId, taskId } = ListTaskPRsSchema.parse(args);
			const prs = await client.listTaskPRs(projectId, taskId);
			const formatted = prs.map(formatPullRequest).join("\n\n---\n\n");
			return {
				content: [
					{
						type: "text",
						text: `Pull Requests:\n\n${formatted}`,
					},
				],
			};
		}

		case "link_pr_to_task": {
			const { projectId, taskId, prId, repositoryName } =
				LinkPRToTaskSchema.parse(args);
			const pr = await client.linkPRToTask(projectId, taskId, {
				repo_id: repositoryName,
				pr_number: prId,
			});
			return {
				content: [
					{
						type: "text",
						text: `PR linked successfully:\n\n${formatPullRequest(pr)}`,
					},
				],
			};
		}

		case "unlink_pr_from_task": {
			const { projectId, taskId, prId } = UnlinkPRFromTaskSchema.parse(args);
			await client.unlinkPRFromTask(projectId, taskId, String(prId));
			return {
				content: [
					{
						type: "text",
						text: `PR #${prId} unlinked successfully`,
					},
				],
			};
		}

		case "create_branch_for_task": {
			const { projectId, taskId, branchName, baseBranch } =
				CreateBranchForTaskSchema.parse(args);
			const branch = await client.createBranch(projectId, taskId, {
				repo_id: "", // This needs to be provided by the user
				branch_name: branchName,
				source_branch: baseBranch,
			});
			return {
				content: [
					{
						type: "text",
						text: `Branch created successfully: ${branch.branch_name}`,
					},
				],
			};
		}

		case "list_task_branches": {
			const { projectId, taskId } = ListTaskBranchesSchema.parse(args);
			const branches = await client.listTaskBranches(projectId, taskId);
			const formatted = branches.map(formatBranch).join("\n\n---\n\n");
			return {
				content: [
					{
						type: "text",
						text: `Branches:\n\n${formatted}`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown task activity/GitHub tool: ${toolName}`);
	}
}
