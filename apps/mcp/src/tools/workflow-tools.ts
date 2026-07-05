import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPIWorkflowClient } from "../api/index.js";

const PROJECT_ID_DESC =
	"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.";

const WORKFLOW_ID_DESC =
	"The technical UUID of the automation workflow. Use list_workflows to get the workflow ID.";

const NODE_ID_DESC =
	"The technical UUID of the workflow node. Use get_workflow to see all nodes in the workflow and the task_id each one wraps.";

const TASK_ID_DESC =
	"The technical UUID of the task to wrap in this node. Use list_tasks to get the task ID.";

const STATUS_ID_DESC =
	"The technical UUID of a task status in this project. Use list_task_statuses to get status IDs.";

const MEMBER_ID_DESC =
	"The technical UUID of the project member (human or agent) to auto-assign. Use list_project_members to get member IDs.";

const ListWorkflowsSchema = z.object({
	projectId: z.string(),
	status: z.enum(["draft", "active", "archived"]).optional(),
});

const GetWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
});

const CreateWorkflowSchema = z.object({
	projectId: z.string(),
	name: z.string(),
	description: z.string().optional(),
});

const UpdateWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	name: z.string().optional(),
	description: z.string().optional(),
});

const DeleteWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
});

const WorkflowLifecycleSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
});

const AddWorkflowNodeSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	taskId: z.string(),
});

const RemoveWorkflowNodeSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	nodeId: z.string(),
});

const SetWorkflowStatusRuleSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	statusId: z.string(),
	assigneeMemberId: z.string(),
});

const RemoveWorkflowStatusRuleSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	ruleId: z.string(),
});

const SetWorkflowStatusTransitionSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	statusId: z.string(),
	nextStatusId: z.string().nullable().optional(),
});

const RemoveWorkflowStatusTransitionSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	transitionId: z.string(),
});

const AddWorkflowEdgeSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	sourceNodeId: z.string(),
	targetNodeId: z.string(),
});

const RemoveWorkflowEdgeSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	edgeId: z.string(),
});

/**
 * Returns all automation-workflow MCP tools.
 */
export function getWorkflowTools(): Tool[] {
	return [
		{
			name: "list_workflows",
			description:
				"List automation workflows in a project, with their id/name/status (draft, active, or archived).",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					status: {
						type: "string",
						enum: ["draft", "active", "archived"],
						description: "Optional filter by lifecycle status.",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "get_workflow",
			description:
				"Fetch a workflow's full graph: every node (the task it wraps), every edge (dependency link between two nodes), the workflow's single shared list of status->assignee rules, and its status-transition chain (the 'status workflow' — see set_workflow_status_transition). Call this before editing a workflow you didn't just create, so you know its current graph.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "create_workflow",
			description:
				"Create a new automation workflow. It starts in 'draft' state — the automation engine ignores it until activate_workflow is called. A default status-transition chain (the 'status workflow') is auto-generated from the project's task statuses ordered by board position, chaining them sequentially — the last (highest-position) status becomes the workflow's done status. Customize it with set_workflow_status_transition.\n\n" +
				"An automation workflow is a dependency graph over EXISTING tasks, plus TWO shared, workflow-level lookup tables:\n" +
				"- Status rules: whenever any task in the workflow changes to a configured status, it's auto-assigned to that rule's member (see set_workflow_status_rule).\n" +
				"- Status transitions ('status workflow'): for each status, which status comes next once work at that status is done (see set_workflow_status_transition). The workflow's done status is whichever status has no next status configured — used to unlock downstream tasks, and to tell an AI-agent assignee exactly what status to set next instead of guessing.\n" +
				"- Edges are plain links: once a source task reaches the workflow's done status, the target task is re-evaluated using ITS OWN current status against the same status rules (no status is changed on the target — only the assignment). If a target has multiple incoming edges, ALL predecessors must be done before it fires.\n\n" +
				"Typical build order: create_workflow (chain auto-seeded) -> add_workflow_node for each task -> set_workflow_status_rule for the statuses that matter -> add_workflow_edge to wire dependencies -> activate_workflow.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					name: { type: "string", description: "Workflow name." },
					description: {
						type: "string",
						description: "Optional description.",
					},
				},
				required: ["projectId", "name"],
			},
		},
		{
			name: "update_workflow",
			description: "Rename or re-describe a workflow.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					name: { type: "string", description: "New name." },
					description: { type: "string", description: "New description." },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "delete_workflow",
			description: "Permanently delete a workflow.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "activate_workflow",
			description:
				"Activate a draft workflow so the automation engine starts running it. Requires at least one node, and the workflow's status-transition chain must have exactly one status with no next status configured (the done status) — see set_workflow_status_transition. If activation fails, the error explains what to fix.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "archive_workflow",
			description:
				"Archive an active workflow, stopping the automation engine from evaluating it.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "revert_workflow_to_draft",
			description:
				"Move an active workflow back to draft, stopping the engine and re-enabling graph edits (add/remove nodes, edges, and status rules). Archived workflows cannot be reverted — delete them or build a new workflow instead.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
				},
				required: ["projectId", "workflowId"],
			},
		},
		{
			name: "add_workflow_node",
			description:
				"Add an existing task as a node in a workflow. The workflow must be in draft state.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					taskId: { type: "string", description: TASK_ID_DESC },
				},
				required: ["projectId", "workflowId", "taskId"],
			},
		},
		{
			name: "remove_workflow_node",
			description:
				"Remove a node from a workflow (and its status rules and edges). The workflow must be in draft state. The underlying task itself is not affected.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					nodeId: { type: "string", description: NODE_ID_DESC },
				},
				required: ["projectId", "workflowId", "nodeId"],
			},
		},
		{
			name: "set_workflow_status_rule",
			description:
				"Set (create or update) the member to auto-assign a task to whenever it changes to a given status. This is ONE shared list per workflow, not per node — it applies to whichever task in the workflow currently has that status. It fires both when someone changes a task's status directly AND when a predecessor task in the workflow finishes and a downstream task is re-evaluated at its current status — so configure a rule for whatever status a task normally sits in when it's ready to be picked up (e.g. 'Ready' or 'To Do'), not just a terminal/done status. The workflow must be in draft state.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					statusId: { type: "string", description: STATUS_ID_DESC },
					assigneeMemberId: { type: "string", description: MEMBER_ID_DESC },
				},
				required: ["projectId", "workflowId", "statusId", "assigneeMemberId"],
			},
		},
		{
			name: "remove_workflow_status_rule",
			description: "Remove a status->assignee rule from the workflow.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					ruleId: {
						type: "string",
						description:
							"The technical UUID of the status rule. Use get_workflow to see the workflow's status_rules.",
					},
				},
				required: ["projectId", "workflowId", "ruleId"],
			},
		},
		{
			name: "set_workflow_status_transition",
			description:
				"Set (create or update) what status should come next once a task reaches a given status — the 'status workflow', distinct from the task-dependency graph. This is used to unlock downstream tasks (when a task reaches the workflow's done status — the one status with no next status configured) and to tell an AI-agent assignee exactly what status to set next instead of guessing. A default chain covering every project status is auto-generated when the workflow is created (see create_workflow); use this tool to customize it. Omit or pass null for nextStatusId to mark statusId as the workflow's done status — exactly one status must have no next status before the workflow can be activated. The workflow must be in draft state.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					statusId: { type: "string", description: STATUS_ID_DESC },
					nextStatusId: {
						type: "string",
						description: `Optional — omit or pass null to mark statusId as terminal (the done status). ${STATUS_ID_DESC}`,
					},
				},
				required: ["projectId", "workflowId", "statusId"],
			},
		},
		{
			name: "remove_workflow_status_transition",
			description:
				"Remove a status-transition entry from the workflow's status-workflow chain. The workflow must be in draft state.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					transitionId: {
						type: "string",
						description:
							"The technical UUID of the status transition. Use get_workflow to see the workflow's status_transitions.",
					},
				},
				required: ["projectId", "workflowId", "transitionId"],
			},
		},
		{
			name: "add_workflow_edge",
			description:
				"Link two nodes already in the workflow (add_workflow_node them first): once the source node's task reaches its done status, the target node's task is re-evaluated for auto-assignment based on ITS OWN current status (see set_workflow_status_rule). No status is changed on the target by this link — only its assignment. The workflow must be in draft state.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					sourceNodeId: {
						type: "string",
						description: `The predecessor node's UUID. ${NODE_ID_DESC}`,
					},
					targetNodeId: {
						type: "string",
						description: `The downstream node's UUID. ${NODE_ID_DESC}`,
					},
				},
				required: ["projectId", "workflowId", "sourceNodeId", "targetNodeId"],
			},
		},
		{
			name: "remove_workflow_edge",
			description: "Remove a dependency link between two nodes.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					edgeId: {
						type: "string",
						description:
							"The technical UUID of the edge. Use get_workflow to see all edges.",
					},
				},
				required: ["projectId", "workflowId", "edgeId"],
			},
		},
	];
}

function formatWorkflow(workflow: any): string {
	return (
		`- **${workflow.name}** (status: ${workflow.status})\n` +
		`  ID: ${workflow.id}\n` +
		`  Description: ${workflow.description || "(none)"}`
	);
}

function formatWorkflowGraph(graph: any): string {
	const wf = graph.workflow;
	const nodes = graph.nodes || [];
	const edges = graph.edges || [];
	const rules = graph.status_rules || [];
	const transitions = graph.status_transitions || [];
	const lines = [
		`Workflow '${wf.name}' (status: ${wf.status}, id: ${wf.id})`,
		`${nodes.length} node(s), ${edges.length} edge(s), ${rules.length} status rule(s), ${transitions.length} status transition(s):`,
		"",
	];
	for (const n of nodes) {
		lines.push(`- node ${n.id}: task_id=${n.task_id}`);
	}
	for (const e of edges) {
		lines.push(`- edge ${e.id}: ${e.source_node_id} -> ${e.target_node_id}`);
	}
	if (rules.length > 0) {
		lines.push("", "Status rules (apply to any task in this workflow):");
		for (const r of rules) {
			lines.push(
				`- status=${r.status_id}->assignee=${r.assignee_member_id} (rule id: ${r.id})`,
			);
		}
	}
	if (transitions.length > 0) {
		lines.push(
			"",
			"Status workflow (next-status chain; a status with no next is the done status):",
		);
		for (const t of transitions) {
			const next = t.next_status_id ? t.next_status_id : "(done)";
			lines.push(
				`- status=${t.status_id} -> next=${next} (transition id: ${t.id})`,
			);
		}
	}
	return lines.join("\n");
}

/**
 * Handles automation-workflow tool calls.
 */
export async function handleWorkflowTool(
	toolName: string,
	args: any,
	workflowClient: PacaAPIWorkflowClient,
): Promise<any> {
	switch (toolName) {
		case "list_workflows": {
			const { projectId, status } = ListWorkflowsSchema.parse(args);
			const workflows = await workflowClient.listWorkflows(projectId, status);
			if (workflows.length === 0) {
				return {
					content: [
						{
							type: "text",
							text: "No automation workflows in this project yet.",
						},
					],
				};
			}
			return {
				content: [
					{
						type: "text",
						text: `Found ${workflows.length} workflow(s):\n\n${workflows.map(formatWorkflow).join("\n")}`,
					},
				],
			};
		}

		case "get_workflow": {
			const { projectId, workflowId } = GetWorkflowSchema.parse(args);
			const graph = await workflowClient.getWorkflow(projectId, workflowId);
			return {
				content: [{ type: "text", text: formatWorkflowGraph(graph) }],
			};
		}

		case "create_workflow": {
			const { projectId, name, description } = CreateWorkflowSchema.parse(args);
			const workflow = await workflowClient.createWorkflow(projectId, {
				name,
				description,
			});
			return {
				content: [
					{
						type: "text",
						text: `Created draft workflow '${workflow.name}' (id: ${workflow.id}). Add tasks as nodes with add_workflow_node, then link them with add_workflow_edge.`,
					},
				],
			};
		}

		case "update_workflow": {
			const { projectId, workflowId, name, description } =
				UpdateWorkflowSchema.parse(args);
			const workflow = await workflowClient.updateWorkflow(
				projectId,
				workflowId,
				{ name, description },
			);
			return {
				content: [
					{
						type: "text",
						text: `Workflow updated:\n\n${formatWorkflow(workflow)}`,
					},
				],
			};
		}

		case "delete_workflow": {
			const { projectId, workflowId } = DeleteWorkflowSchema.parse(args);
			await workflowClient.deleteWorkflow(projectId, workflowId);
			return {
				content: [
					{
						type: "text",
						text: `Workflow ${workflowId} deleted successfully.`,
					},
				],
			};
		}

		case "activate_workflow": {
			const { projectId, workflowId } = WorkflowLifecycleSchema.parse(args);
			const workflow = await workflowClient.activateWorkflow(
				projectId,
				workflowId,
			);
			return {
				content: [
					{
						type: "text",
						text: `Workflow '${workflow.name}' is now ${workflow.status}.`,
					},
				],
			};
		}

		case "archive_workflow": {
			const { projectId, workflowId } = WorkflowLifecycleSchema.parse(args);
			const workflow = await workflowClient.archiveWorkflow(
				projectId,
				workflowId,
			);
			return {
				content: [
					{
						type: "text",
						text: `Workflow '${workflow.name}' is now ${workflow.status}.`,
					},
				],
			};
		}

		case "revert_workflow_to_draft": {
			const { projectId, workflowId } = WorkflowLifecycleSchema.parse(args);
			const workflow = await workflowClient.revertWorkflowToDraft(
				projectId,
				workflowId,
			);
			return {
				content: [
					{
						type: "text",
						text: `Workflow '${workflow.name}' is now ${workflow.status}.`,
					},
				],
			};
		}

		case "add_workflow_node": {
			const { projectId, workflowId, taskId } =
				AddWorkflowNodeSchema.parse(args);
			const node = await workflowClient.addWorkflowNode(projectId, workflowId, {
				task_id: taskId,
			});
			return {
				content: [
					{
						type: "text",
						text: `Added task as node ${node.id} in the workflow.`,
					},
				],
			};
		}

		case "remove_workflow_node": {
			const { projectId, workflowId, nodeId } =
				RemoveWorkflowNodeSchema.parse(args);
			await workflowClient.removeWorkflowNode(projectId, workflowId, nodeId);
			return {
				content: [
					{ type: "text", text: `Node ${nodeId} removed from the workflow.` },
				],
			};
		}

		case "set_workflow_status_rule": {
			const { projectId, workflowId, statusId, assigneeMemberId } =
				SetWorkflowStatusRuleSchema.parse(args);
			const rule = await workflowClient.setWorkflowStatusRule(
				projectId,
				workflowId,
				{ status_id: statusId, assignee_member_id: assigneeMemberId },
			);
			return {
				content: [
					{
						type: "text",
						text: `Rule saved (id: ${rule.id}): when status -> ${rule.status_id}, assign to ${rule.assignee_member_id}.`,
					},
				],
			};
		}

		case "remove_workflow_status_rule": {
			const { projectId, workflowId, ruleId } =
				RemoveWorkflowStatusRuleSchema.parse(args);
			await workflowClient.removeWorkflowStatusRule(
				projectId,
				workflowId,
				ruleId,
			);
			return {
				content: [{ type: "text", text: `Status rule ${ruleId} removed.` }],
			};
		}

		case "set_workflow_status_transition": {
			const { projectId, workflowId, statusId, nextStatusId } =
				SetWorkflowStatusTransitionSchema.parse(args);
			const transition = await workflowClient.setWorkflowStatusTransition(
				projectId,
				workflowId,
				{ status_id: statusId, next_status_id: nextStatusId ?? null },
			);
			const nextText = transition.next_status_id
				? `next -> ${transition.next_status_id}`
				: "marked as the done status (no next status)";
			return {
				content: [
					{
						type: "text",
						text: `Transition saved (id: ${transition.id}): status ${transition.status_id} ${nextText}.`,
					},
				],
			};
		}

		case "remove_workflow_status_transition": {
			const { projectId, workflowId, transitionId } =
				RemoveWorkflowStatusTransitionSchema.parse(args);
			await workflowClient.removeWorkflowStatusTransition(
				projectId,
				workflowId,
				transitionId,
			);
			return {
				content: [
					{ type: "text", text: `Status transition ${transitionId} removed.` },
				],
			};
		}

		case "add_workflow_edge": {
			const { projectId, workflowId, sourceNodeId, targetNodeId } =
				AddWorkflowEdgeSchema.parse(args);
			const edge = await workflowClient.addWorkflowEdge(projectId, workflowId, {
				source_node_id: sourceNodeId,
				target_node_id: targetNodeId,
			});
			return {
				content: [
					{
						type: "text",
						text: `Linked (edge id: ${edge.id}): once the source task is done, the target task will be re-evaluated.`,
					},
				],
			};
		}

		case "remove_workflow_edge": {
			const { projectId, workflowId, edgeId } =
				RemoveWorkflowEdgeSchema.parse(args);
			await workflowClient.removeWorkflowEdge(projectId, workflowId, edgeId);
			return {
				content: [{ type: "text", text: `Edge ${edgeId} removed.` }],
			};
		}

		default:
			throw new Error(`Unknown workflow tool: ${toolName}`);
	}
}
