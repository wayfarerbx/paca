import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type {
	PacaAPIClient,
	PacaAPITaskExtendedClient,
	PacaAPIViewsClient,
} from "../api/index.js";
import type { Task } from "../types/index.js";
import { formatList, formatTask, formatTaskDetail } from "../utils/index.js";

const ListTasksSchema = z.object({
	projectId: z.string(),
});

const GetTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

const GetTaskByNumberSchema = z.object({
	projectId: z.string(),
	taskNumber: z.number(),
});

const CreateTaskSchema = z.object({
	projectId: z.string(),
	title: z.string(),
	description: z.string().optional(),
	statusId: z.string().optional(),
	typeId: z.string().optional(),
	sprintId: z.string().optional(),
	assigneeId: z.string().optional(),
	importance: z.number().optional(),
	storyPoints: z.number().int().min(0).nullable().optional(),
	tags: z.array(z.string()).optional(),
	startDate: z.string().optional(),
	dueDate: z.string().optional(),
});

const UpdateTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
	title: z.string().optional(),
	description: z.string().optional(),
	statusId: z.string().optional(),
	typeId: z.string().optional(),
	sprintId: z.string().optional(),
	assigneeId: z.string().optional(),
	importance: z.number().optional(),
	storyPoints: z.number().int().min(0).nullable().optional(),
	tags: z.array(z.string()).optional(),
	startDate: z.string().optional(),
	dueDate: z.string().optional(),
});

const DeleteTaskSchema = z.object({
	projectId: z.string(),
	taskId: z.string(),
});

/**
 * Returns all task-related MCP tools.
 * @returns Array of task tools
 */
export function getTaskTools(): Tool[] {
	return [
		{
			name: "list_tasks",
			description: "List all tasks in a project",
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
			name: "get_task",
			description:
				"Get comprehensive details of a specific task including all properties, subtasks, attachments, and activities - everything that users can see in the task detail component of the web app",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description:
							"The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
		{
			name: "get_task_by_number",
			description:
				"Get comprehensive details of a task by its number within a project including all properties, subtasks, attachments, and activities",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskNumber: {
						type: "number",
						description: "The task number",
					},
				},
				required: ["projectId", "taskNumber"],
			},
		},
		{
			name: "create_task",
			description: "Create a new task",
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
						description: "The title of the task",
					},
					description: {
						type: "string",
						description:
							"The description of the task (will be converted from markdown to BlockNote format)",
					},
					statusId: {
						type: "string",
						description:
							"The technical UUID of the task status (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_statuses to get the status ID.",
					},
					typeId: {
						type: "string",
						description:
							"The technical UUID of the task type (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_types to get the type ID.",
					},
					sprintId: {
						type: "string",
						description:
							"The technical UUID of the sprint (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_sprints to get the sprint ID.",
					},
					assigneeId: {
						type: "string",
						description:
							"The technical UUID of the user to assign the task to (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_project_members to get user IDs.",
					},
					importance: {
						type: "number",
						description: "The importance of the task",
					},
					storyPoints: {
						type: ["number", "null"],
						description:
							"Story points estimate for the task (Fibonacci: 1, 2, 3, 5, 8, 13, ...)",
					},
					tags: {
						type: "array",
						items: { type: "string" },
						description: "Tags for the task",
					},
					startDate: {
						type: "string",
						description: "The start date (ISO 8601 format)",
					},
					dueDate: {
						type: "string",
						description: "The due date (ISO 8601 format)",
					},
				},
				required: ["projectId", "title"],
			},
		},
		{
			name: "update_task",
			description: "Update an existing task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description:
							"The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
					title: {
						type: "string",
						description: "The new title of the task",
					},
					description: {
						type: "string",
						description:
							"The new description of the task (will be converted from markdown to BlockNote format)",
					},
					statusId: {
						type: "string",
						description:
							"The technical UUID of the task status (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_statuses to get the status ID.",
					},
					typeId: {
						type: "string",
						description:
							"The technical UUID of the task type (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_task_types to get the type ID.",
					},
					sprintId: {
						type: "string",
						description:
							"The technical UUID of the sprint (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_sprints to get the sprint ID.",
					},
					assigneeId: {
						type: "string",
						description:
							"The technical UUID of the user to assign the task to (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_project_members to get user IDs.",
					},
					importance: {
						type: "number",
						description: "The importance of the task",
					},
					storyPoints: {
						type: ["number", "null"],
						description:
							"Story points estimate for the task (set to null to clear)",
					},
					tags: {
						type: "array",
						items: { type: "string" },
						description: "Tags for the task",
					},
					startDate: {
						type: "string",
						description: "The start date (ISO 8601 format)",
					},
					dueDate: {
						type: "string",
						description: "The due date (ISO 8601 format)",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
		{
			name: "delete_task",
			description: "Delete a task",
			inputSchema: {
				type: "object",
				properties: {
					projectId: {
						type: "string",
						description:
							"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.",
					},
					taskId: {
						type: "string",
						description:
							"The technical UUID of the task (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_tasks to get the task ID.",
					},
				},
				required: ["projectId", "taskId"],
			},
		},
	];
}

/**
 * Handles task-related tool calls.
 * @param toolName - Name of the tool being called
 * @param args - Tool arguments
 * @param client - Paca API client instance
 * @returns Tool response
 */
export async function handleTaskTool(
	toolName: string,
	args: any,
	client: PacaAPIClient,
	extendedClient?: PacaAPITaskExtendedClient,
	viewsClient?: PacaAPIViewsClient,
): Promise<any> {
	switch (toolName) {
		case "list_tasks": {
			const { projectId } = ListTasksSchema.parse(args);
			const tasks = await client.listTasks(projectId);
			const formatted = formatList(tasks, formatTask);
			return {
				content: [
					{
						type: "text",
						text: `Tasks:\n\n${formatted}`,
					},
				],
			};
		}

		case "get_task": {
			const { projectId, taskId } = GetTaskSchema.parse(args);
			return await getTaskDetail(
				projectId,
				taskId,
				client,
				extendedClient,
				viewsClient,
			);
		}

		case "get_task_by_number": {
			const { projectId, taskNumber } = GetTaskByNumberSchema.parse(args);
			const task = await client.getTaskByNumber(projectId, taskNumber);
			return await getTaskDetail(
				projectId,
				task.id,
				client,
				extendedClient,
				viewsClient,
			);
		}

		case "create_task": {
			const {
				projectId,
				title,
				description,
				statusId,
				typeId,
				sprintId,
				assigneeId,
				importance,
				storyPoints,
				tags,
				startDate,
				dueDate,
			} = CreateTaskSchema.parse(args);
			const task = await client.createTask({
				project_id: projectId,
				title,
				description,
				status_id: statusId,
				task_type_id: typeId,
				sprint_id: sprintId,
				assignee_id: assigneeId,
				importance,
				story_points: storyPoints,
				tags,
				start_date: startDate,
				due_date: dueDate,
			});
			return {
				content: [
					{
						type: "text",
						text: `Task created successfully:\n\n${formatTask(task)}`,
					},
				],
			};
		}

		case "update_task": {
			const {
				projectId,
				taskId,
				title,
				description,
				statusId,
				typeId,
				sprintId,
				assigneeId,
				importance,
				storyPoints,
				tags,
				startDate,
				dueDate,
			} = UpdateTaskSchema.parse(args);
			const task = await client.updateTask(projectId, taskId, {
				title,
				description,
				status_id: statusId,
				task_type_id: typeId,
				sprint_id: sprintId,
				assignee_id: assigneeId,
				importance,
				story_points: storyPoints,
				tags,
				start_date: startDate,
				due_date: dueDate,
			});
			return {
				content: [
					{
						type: "text",
						text: `Task updated successfully:\n\n${formatTask(task)}`,
					},
				],
			};
		}

		case "delete_task": {
			const { projectId, taskId } = DeleteTaskSchema.parse(args);
			await client.deleteTask(projectId, taskId);
			return {
				content: [
					{
						type: "text",
						text: `Task ${taskId} deleted successfully`,
					},
				],
			};
		}

		default:
			throw new Error(`Unknown task tool: ${toolName}`);
	}
}

async function getTaskDetail(
	projectId: string,
	taskId: string,
	client: PacaAPIClient,
	extendedClient?: PacaAPITaskExtendedClient,
	viewsClient?: PacaAPIViewsClient,
): Promise<any> {
	const task = await client.getTask(projectId, taskId);

	const [
		project,
		statuses,
		taskTypes,
		sprints,
		members,
		subtasks,
		attachments,
		activities,
		customFields,
	] = await Promise.all([
		client.getProject(projectId).catch(() => undefined),
		extendedClient?.listTaskStatuses(projectId)?.catch(() => []) ||
			Promise.resolve([]),
		extendedClient?.listTaskTypes(projectId)?.catch(() => []) ||
			Promise.resolve([]),
		client.listSprints(projectId).catch(() => []),
		extendedClient?.listProjectMembers(projectId)?.catch(() => []) ||
			Promise.resolve([]),
		extendedClient?.listSubtasks(projectId, taskId)?.catch(() => []) ||
			Promise.resolve([]),
		viewsClient?.listTaskAttachments(projectId, taskId)?.catch(() => []) ||
			Promise.resolve([]),
		extendedClient?.listTaskActivities(projectId, taskId)?.catch(() => []) ||
			Promise.resolve([]),
		viewsClient?.listCustomFieldDefinitions(projectId)?.catch(() => []) ||
			Promise.resolve([]),
	]);

	const status = statuses.find((s: any) => s.id === task.status_id);
	const taskType = taskTypes.find((t: any) => t.id === task.task_type_id);
	const sprint = sprints.find((s: any) => s.id === task.sprint_id);
	const assignee = members.find((m: any) => m.id === task.assignee_id);
	const reporter = members.find((m: any) => m.id === task.reporter_id);
	let parentTask: Task | undefined;
	if (task.parent_task_id) {
		try {
			parentTask = await client.getTask(projectId, task.parent_task_id);
		} catch (_e) {
			parentTask = undefined;
		}
	}

	const formatted = formatTaskDetail(
		task,
		project,
		status,
		taskType,
		sprint,
		assignee,
		reporter,
		parentTask,
		subtasks,
		attachments,
		activities,
		customFields,
	);

	return {
		content: [
			{
				type: "text",
				text: formatted,
			},
		],
	};
}
