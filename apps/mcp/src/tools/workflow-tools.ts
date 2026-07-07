import type { Tool } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import type { PacaAPIWorkflowClient } from "../api/index.js";
import type { Workflow, WorkflowGraph } from "../types/index.js";
import { formatToolError } from "../utils/index.js";
import {
	anyHasFailure,
	applyEdges,
	applyNodes,
	applyStatusRules,
	applyStatusTransitions,
	buildGraphIndexes,
	type CategoryResult,
	checkNodeSpacing,
	emptyGraphIndexes,
	extractApiErrorMessage,
	formatCategoryResult,
	type NodeSetInput,
	type OrchestrationContext,
	type PositionedNode,
	RECOMMENDED_NODE_GAP_X,
	RECOMMENDED_NODE_GAP_Y,
} from "./workflow-orchestration.js";

const PROJECT_ID_DESC =
	"The technical UUID of the project (e.g., '550e8400-e29b-41d4-a716-446655440000'). Use list_projects to get the project ID. Do NOT use the project name.";

const WORKFLOW_ID_DESC =
	"The technical UUID of the automation workflow. Use get_workflow (without workflowId) to list workflows and get their IDs.";

const TASK_ID_DESC =
	"The technical UUID of the task to wrap in this node. Use list_tasks to get the task ID.";

const STATUS_ID_DESC =
	"The technical UUID of a task status in this project. Use list_task_statuses to get status IDs.";

const MEMBER_ID_DESC =
	"The technical UUID of the project member (human or agent) to auto-assign. Use list_project_members to get member IDs.";

const WorkflowStatusEnum = z.enum(["draft", "active", "archived"]);

// .strict() on every item schema below is deliberate: agents occasionally
// invent a plausible-but-wrong field name (e.g. a single "position" rank
// instead of posX/posY, mirroring the position field used by other Paca
// tools like list_task_statuses/move_task). Without .strict(), Zod silently
// drops the unrecognized key and reports only "posX/posY is required" with
// no hint about what was actually sent; with .strict(), the caller also gets
// an explicit "Unrecognized key(s): 'position'" issue naming the mistake.

const NodeSetInputSchema = z
	.object({
		taskId: z.string(),
		posX: z.number(),
		posY: z.number(),
	})
	.strict();

const StatusRuleSetInputSchema = z
	.object({
		statusId: z.string(),
		assigneeMemberId: z.string(),
	})
	.strict();

const StatusTransitionSetInputSchema = z
	.object({
		statusId: z.string(),
		nextStatusId: z.string().nullable().optional(),
	})
	.strict();

const EdgeRefSchema = z
	.object({
		sourceTaskId: z.string(),
		targetTaskId: z.string(),
	})
	.strict();

const GetWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string().optional(),
	status: WorkflowStatusEnum.optional(),
});

const CreateWorkflowSchema = z.object({
	projectId: z.string(),
	name: z.string().trim().min(1),
	description: z.string().optional(),
	nodes: z.array(NodeSetInputSchema).optional(),
	statusRules: z.array(StatusRuleSetInputSchema).optional(),
	statusTransitions: z.array(StatusTransitionSetInputSchema).optional(),
	edges: z.array(EdgeRefSchema).optional(),
	activate: z.boolean().optional(),
});

const UpdateWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
	name: z.string().trim().min(1).optional(),
	description: z.string().optional(),
	status: WorkflowStatusEnum.optional(),
	nodes: z
		.object({
			set: z.array(NodeSetInputSchema).optional(),
			remove: z.array(z.string()).optional(),
		})
		.optional(),
	statusRules: z
		.object({
			set: z.array(StatusRuleSetInputSchema).optional(),
			remove: z.array(z.string()).optional(),
		})
		.optional(),
	statusTransitions: z
		.object({
			set: z.array(StatusTransitionSetInputSchema).optional(),
			remove: z.array(z.string()).optional(),
		})
		.optional(),
	edges: z
		.object({
			add: z.array(EdgeRefSchema).optional(),
			remove: z.array(EdgeRefSchema).optional(),
		})
		.optional(),
});

const DeleteWorkflowSchema = z.object({
	projectId: z.string(),
	workflowId: z.string(),
});

const NODE_ITEM_JSON_SCHEMA = {
	type: "object" as const,
	properties: {
		taskId: {
			type: "string",
			description: `${TASK_ID_DESC} Every node entry also needs posX AND posY — decide on both before calling this tool; there is no valid entry with just taskId.`,
		},
		posX: {
			type: "number",
			description: `REQUIRED canvas x-position — you choose it; there is no auto-placement. Use posX as the LANE within a row (see posY): tasks that are parallel/independent of each other at the same dependency stage share the SAME posY but get DIFFERENT posX values, spaced at least ${RECOMMENDED_NODE_GAP_X}px apart (canvas cards are 256px wide) so they read as clearly separate columns — e.g. posX = 0, ${RECOMMENDED_NODE_GAP_X}, ${RECOMMENDED_NODE_GAP_X * 2} for 3 parallel tasks in one row. This minimum is enforced by a post-call check: if any two of the nodes you set land closer than this on BOTH axes, the response includes a warning naming them — treat that warning as a required fix, not a suggestion. Also avoid choosing a posX/posY that lands ON the straight line between two OTHER nodes that are connected by an edge — that makes the edge line pass through/behind this node and is hard to read; shift it to its own lane instead. If the workflow already has nodes, call get_workflow first (its response lists every existing node's position) and continue past them rather than guessing or reusing one.`,
		},
		posY: {
			type: "number",
			description: `REQUIRED canvas y-position — you choose it. Use posY as the ROW/STAGE, matching dependency order (this must agree with the edges you declare): tasks with no predecessors — the ones that should be done first/in advance — go in the TOP row, posY = 0. Each task that depends on another (has an incoming edge from it) belongs in a LOWER row than everything it depends on; tasks that should happen last get the BOTTOM-most (largest) posY. Space rows at least ${RECOMMENDED_NODE_GAP_Y}px apart, e.g. posY = 0, ${RECOMMENDED_NODE_GAP_Y}, ${RECOMMENDED_NODE_GAP_Y * 2} for a 3-stage pipeline. Tasks sharing the same posY are read as the same stage/parallel — see posX for placing them side by side within that row. Like posX, this minimum is checked after the call — a warning naming any pair that's still too close means fix it, not just a nice-to-have.`,
		},
	},
	required: ["taskId", "posX", "posY"],
};

const STATUS_RULE_ITEM_JSON_SCHEMA = {
	type: "object" as const,
	properties: {
		statusId: { type: "string", description: STATUS_ID_DESC },
		assigneeMemberId: { type: "string", description: MEMBER_ID_DESC },
	},
	required: ["statusId", "assigneeMemberId"],
};

const STATUS_TRANSITION_ITEM_JSON_SCHEMA = {
	type: "object" as const,
	properties: {
		statusId: { type: "string", description: STATUS_ID_DESC },
		nextStatusId: {
			type: "string",
			description: `Optional — omit or pass null to mark statusId as terminal (the done status). ${STATUS_ID_DESC}`,
		},
	},
	required: ["statusId"],
};

const EDGE_ITEM_JSON_SCHEMA = {
	type: "object" as const,
	properties: {
		sourceTaskId: {
			type: "string",
			description: `The predecessor task's UUID — must also appear in nodes. ${TASK_ID_DESC}`,
		},
		targetTaskId: {
			type: "string",
			description: `The downstream task's UUID — must also appear in nodes. ${TASK_ID_DESC}`,
		},
	},
	required: ["sourceTaskId", "targetTaskId"],
};

/**
 * Returns all automation-workflow MCP tools.
 */
export function getWorkflowTools(): Tool[] {
	return [
		{
			name: "get_workflow",
			description:
				"Get information about automation workflows in a project. Pass workflowId to fetch one workflow's full graph: every node (the task it wraps), every edge (dependency link between two nodes), the workflow's single shared list of status->assignee rules, and its status-transition chain (the 'status workflow'). Omit workflowId to list all workflows in the project instead (optionally filtered by status) — e.g. to find a workflow's ID before fetching its graph.\n\n" +
				"An automation workflow is a dependency graph over EXISTING tasks, plus TWO shared, workflow-level lookup tables:\n" +
				"- Status rules: whenever any task in the workflow changes to a configured status, it's auto-assigned to that rule's member. create_workflow auto-seeds one of these per status (see its description) — a rule you didn't explicitly set is a valid default, not an error; change its assignee via statusRules.set (upserts by statusId) instead of removing it first.\n" +
				"- Status transitions ('status workflow'): for each status, which status comes next once work at that status is done. The workflow's done status is whichever status has no next status configured — used to unlock downstream tasks, and to tell an AI-agent assignee exactly what status to set next instead of guessing.\n" +
				"- Edges are plain links: once a source task reaches the workflow's done status, the target task is re-evaluated using ITS OWN current status against the same status rules (no status is changed on the target — only the assignment). If a target has multiple incoming edges, ALL predecessors must be done before it fires.\n\n" +
				"Call this before editing a workflow you didn't just create, so you know its current graph — create_workflow/update_workflow address nodes/edges by taskId, not by internal IDs, but you'll still want to see what's already there.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: {
						type: "string",
						description: `Optional. ${WORKFLOW_ID_DESC} Omit to list all workflows in the project instead of fetching one's graph.`,
					},
					status: {
						type: "string",
						enum: ["draft", "active", "archived"],
						description:
							"Optional filter by lifecycle status. Only used when workflowId is omitted.",
					},
				},
				required: ["projectId"],
			},
		},
		{
			name: "create_workflow",
			description:
				"IMPORTANT: every entry in nodes must include posX AND posY (both required numbers) in the SAME call — decide where each node goes before calling this tool. There is no way to add a node without a position and no way to add one later without a second call; omitting them just wastes a round trip on a validation error. Lay them out top-to-bottom by dependency order (posY = stage/row, matching the edges you declare — earlier/no-predecessor tasks on top, later/dependent tasks below) and side-by-side for parallel tasks at the same stage (same posY, different posX). See the posX/posY field descriptions below for exact spacing and how to avoid a node sitting on top of another edge.\n\n" +
				"Create a new automation workflow, optionally building out its whole graph in one call: nodes (tasks to wrap), status rules, status transitions, and edges. Starts in 'draft' state — the automation engine ignores it until activated (pass activate: true once the graph is complete, or activate later via update_workflow).\n\n" +
				"A default status-transition chain is auto-generated from the project's task statuses ordered by board position, chaining them sequentially — the last (highest-position) status becomes the workflow's done status. Pass statusTransitions to override entries in this chain.\n\n" +
				"A default status rule is ALSO auto-generated for every status — assigned to you if you're human, otherwise the project's first human member, since an agent can't hand its own work off to itself. This is so the workflow hands work off somewhere immediately instead of doing nothing until manually configured. These seeded rules are valid starting points, not invalid placeholders to delete — pass statusRules only for the statuses whose assignee you want to change; each entry upserts by statusId, reassigning the existing default in place.\n\n" +
				"Reference tasks/statuses/members by the IDs you already have from list_tasks/list_task_statuses/list_project_members — you never need an internal node/rule/transition/edge ID to build a workflow. edges reference the SAME task IDs used in nodes (both endpoints must also appear in nodes); this tool resolves them to the workflow's internal node IDs for you.\n\n" +
				"Each entry in nodes/statusRules/statusTransitions/edges is applied independently — one bad entry (e.g. an edge that would create a cycle) doesn't block the others; check the response for any 'failed' items. activate is only attempted if every requested item above succeeded.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					name: { type: "string", description: "Workflow name." },
					description: {
						type: "string",
						description: "Optional description.",
					},
					nodes: {
						type: "array",
						description:
							"Tasks to add as nodes in this workflow. Every entry requires taskId, posX, AND posY together — there is no valid entry with only taskId.",
						items: NODE_ITEM_JSON_SCHEMA,
					},
					statusRules: {
						type: "array",
						description:
							"Status->assignee rules: whenever a task in this workflow changes to statusId, auto-assign it to assigneeMemberId.",
						items: STATUS_RULE_ITEM_JSON_SCHEMA,
					},
					statusTransitions: {
						type: "array",
						description:
							"Overrides to the auto-generated status-transition chain: for statusId, what status comes next once work there is done.",
						items: STATUS_TRANSITION_ITEM_JSON_SCHEMA,
					},
					edges: {
						type: "array",
						description:
							"Dependency links between nodes: once sourceTaskId's task reaches the workflow's done status, targetTaskId's task is re-evaluated for auto-assignment. Both task IDs must also appear in nodes.",
						items: EDGE_ITEM_JSON_SCHEMA,
					},
					activate: {
						type: "boolean",
						description:
							"If true, activate the workflow immediately after building the graph above — only attempted if every item succeeded.",
					},
				},
				required: ["projectId", "name"],
			},
		},
		{
			name: "update_workflow",
			description:
				"IMPORTANT: every nodes.set entry must include posX AND posY (both required numbers), even when you are ONLY repositioning a node that already exists — decide on positions before calling this tool. There is no valid entry with just taskId, and omitting them just wastes a round trip on a validation error. Lay nodes out top-to-bottom by dependency order (posY = stage/row, matching the edges — earlier/no-predecessor tasks on top, later/dependent tasks below) and side-by-side for parallel tasks at the same stage (same posY, different posX); call get_workflow first to see every existing node's current position (and any edges) before repositioning, so the result stays consistent with the rest of the graph. See the posX/posY field descriptions below for exact spacing.\n\n" +
				"Update a workflow: rename/describe it, change its lifecycle status, and/or edit its graph (nodes, status rules, status transitions, edges) — all in one call. Like create_workflow, everything is addressed by taskId/statusId (no internal node/rule/transition/edge IDs needed); use get_workflow first if you need to see the workflow's current graph.\n\n" +
				"Every graph edit below (nodes/statusRules/statusTransitions/edges) technically requires the workflow to be in 'draft' state — including a nodes.set entry that ONLY repositions an existing node, which is NOT exempt just because it's purely visual. You don't need to manage this yourself, though: if you omit status entirely and the workflow is currently active, this call automatically reverts it to draft, applies every edit below, and re-activates it before returning — all in this ONE call. So for the common case ('just reposition/edit these nodes'), call this with only nodes/statusRules/statusTransitions/edges set and nothing else; do not call this tool once per node, pass every node you're touching as one nodes.set array in a single call.\n\n" +
				"Only pass status yourself when you actually want a different END state than what the workflow already had: 'draft' to revert AND stay in draft (e.g. you want to review before reactivating yourself later — this also unlocks edits in the same call, same as the automatic case above); 'active' to activate a currently-draft workflow (requires at least one node and exactly one status with no next status configured); 'archived' to archive a currently-active one (archived workflows can never be reverted — delete or build a new one instead). When status is given, it's applied around the edits the same way (draft first, active/archived last) but the workflow is left in whatever you asked for instead of being auto-restored.\n\n" +
				"nodes/statusRules/statusTransitions/edges each take set/remove (edges use add instead of set, since an edge has nothing to update — only exists or not): 'set' creates the entry if it doesn't exist yet, or updates it in place if it does (e.g. re-positioning an existing node, or changing a rule's assignee). 'remove' deletes it — removing a taskId/statusId/edge pair that doesn't currently exist is a no-op, not an error, so it's safe to retry. Every item in every list is attempted independently; one item failing (e.g. one bad edge) doesn't block its siblings.",
			inputSchema: {
				type: "object",
				properties: {
					projectId: { type: "string", description: PROJECT_ID_DESC },
					workflowId: { type: "string", description: WORKFLOW_ID_DESC },
					name: { type: "string", description: "New name." },
					description: { type: "string", description: "New description." },
					status: {
						type: "string",
						enum: ["draft", "active", "archived"],
						description:
							"New lifecycle status. See the tool description above for preconditions and ordering relative to graph edits in the same call.",
					},
					nodes: {
						type: "object",
						description:
							"Add/reposition or remove nodes, addressed by taskId. Put every node you're touching in ONE nodes.set array — e.g. repositioning 5 nodes is 1 update_workflow call with 5 entries, not 5 separate calls. (Requires 'draft' state, same as any other graph edit, but you don't need to handle that yourself — see the tool description above.)",
						properties: {
							set: {
								type: "array",
								description:
									"Tasks to add as a node (if new) or reposition (if already a node in this workflow). Every entry requires taskId, posX, AND posY together — there is no valid entry with only taskId, even for repositioning a node that already has a position.",
								items: NODE_ITEM_JSON_SCHEMA,
							},
							remove: {
								type: "array",
								description:
									"Task IDs whose node should be removed from this workflow (also removes that node's edges).",
								items: { type: "string", description: TASK_ID_DESC },
							},
						},
					},
					statusRules: {
						type: "object",
						description:
							"Set or remove status->assignee rules. Note: create_workflow already seeded one valid default rule per status — to change who a status is assigned to, use set with that statusId (it upserts in place); you do not need to remove the existing rule first.",
						properties: {
							set: {
								type: "array",
								description: "Rules to create or update.",
								items: STATUS_RULE_ITEM_JSON_SCHEMA,
							},
							remove: {
								type: "array",
								description: "Status IDs whose rule should be removed.",
								items: { type: "string", description: STATUS_ID_DESC },
							},
						},
					},
					statusTransitions: {
						type: "object",
						description: "Set or remove status-transition chain entries.",
						properties: {
							set: {
								type: "array",
								description: "Transition entries to create or update.",
								items: STATUS_TRANSITION_ITEM_JSON_SCHEMA,
							},
							remove: {
								type: "array",
								description:
									"Status IDs whose transition entry should be removed.",
								items: { type: "string", description: STATUS_ID_DESC },
							},
						},
					},
					edges: {
						type: "object",
						description: "Add or remove dependency links between nodes.",
						properties: {
							add: {
								type: "array",
								items: EDGE_ITEM_JSON_SCHEMA,
							},
							remove: {
								type: "array",
								items: EDGE_ITEM_JSON_SCHEMA,
							},
						},
					},
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
	];
}

/**
 * Positions for the node.set entries that actually succeeded (created or
 * updated), for feeding into checkNodeSpacing — a failed entry's requested
 * position isn't meaningfully "on the canvas" to warn about.
 */
function successfullyPositionedNodes(
	items: NodeSetInput[] | undefined,
	setResult: CategoryResult,
): PositionedNode[] {
	const successfulTaskIds = new Set(
		setResult.items
			.filter((i) => i.outcome === "created" || i.outcome === "updated")
			.map((i) => i.key),
	);
	return (items ?? [])
		.filter((n) => successfulTaskIds.has(n.taskId))
		.map((n) => ({ taskId: n.taskId, posX: n.posX, posY: n.posY }));
}

function formatWorkflow(workflow: Workflow): string {
	return (
		`- **${workflow.name}** (status: ${workflow.status})\n` +
		`  ID: ${workflow.id}\n` +
		`  Description: ${workflow.description || "(none)"}`
	);
}

function formatWorkflowGraph(graph: WorkflowGraph): string {
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
		lines.push(
			`- node ${n.id}: task_id=${n.task_id}, position=(${n.pos_x}, ${n.pos_y})`,
		);
	}
	for (const e of edges) {
		lines.push(`- edge ${e.id}: ${e.source_node_id} -> ${e.target_node_id}`);
	}
	if (rules.length > 0) {
		lines.push(
			"",
			"Status rules (apply to any task in this workflow; every status gets one by default when the workflow is created — these are valid rules, not errors, even if you didn't set them yourself; use update_workflow's statusRules.set to reassign one in place instead of removing it):",
		);
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
	// Wrapped locally (rather than relying on handleToolCall's try/catch)
	// because that outer catch only sees SYNCHRONOUS throws — every case
	// below (including each XxxSchema.parse(args) call) runs inside this
	// async function, so a thrown error becomes a promise rejection that
	// bare `return handleWorkflowTool(...)` in tools/index.ts does not
	// intercept. Without this, an invalid-arguments error (e.g. a Zod
	// validation failure) would surface as an unhandled rejection instead of
	// a clean, readable tool result the calling agent can act on.
	try {
		return await handleWorkflowToolInner(toolName, args, workflowClient);
	} catch (error) {
		return {
			content: [{ type: "text", text: `Error: ${formatToolError(error)}` }],
			isError: true,
		};
	}
}

async function handleWorkflowToolInner(
	toolName: string,
	args: any,
	workflowClient: PacaAPIWorkflowClient,
): Promise<any> {
	switch (toolName) {
		case "get_workflow": {
			const { projectId, workflowId, status } = GetWorkflowSchema.parse(args);

			if (workflowId) {
				const graph = await workflowClient.getWorkflow(projectId, workflowId);
				return {
					content: [{ type: "text", text: formatWorkflowGraph(graph) }],
				};
			}

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

		case "create_workflow": {
			const {
				projectId,
				name,
				description,
				nodes,
				statusRules,
				statusTransitions,
				edges,
				activate,
			} = CreateWorkflowSchema.parse(args);

			const workflow = await workflowClient.createWorkflow(projectId, {
				name,
				description,
			});
			const ctx: OrchestrationContext = {
				client: workflowClient,
				projectId,
				workflowId: workflow.id,
			};
			const indexes = emptyGraphIndexes();

			const nodesResult = await applyNodes(ctx, nodes, undefined, indexes);
			const rulesResult = await applyStatusRules(
				ctx,
				statusRules,
				undefined,
				indexes,
			);
			const transitionsResult = await applyStatusTransitions(
				ctx,
				statusTransitions,
				undefined,
				indexes,
			);
			const edgesResult = await applyEdges(ctx, edges, undefined, indexes);

			const hadFailure = anyHasFailure(
				nodesResult.set,
				rulesResult.set,
				transitionsResult.set,
				edgesResult.added,
			);

			const lines = [
				`Created draft workflow '${workflow.name}' (id: ${workflow.id}).`,
			];
			for (const block of [
				formatCategoryResult("Nodes", nodesResult.set),
				formatCategoryResult("Status rules", rulesResult.set),
				formatCategoryResult("Status transitions", transitionsResult.set),
				formatCategoryResult("Edges", edgesResult.added),
			]) {
				if (block) lines.push(block);
			}

			const spacingWarning = checkNodeSpacing(
				successfullyPositionedNodes(nodes, nodesResult.set),
			);
			if (spacingWarning) lines.push(spacingWarning);

			if (activate) {
				if (hadFailure) {
					lines.push(
						"Skipped activation because at least one item above failed — fix it via update_workflow, then activate separately.",
					);
				} else {
					try {
						const activated = await workflowClient.activateWorkflow(
							projectId,
							workflow.id,
						);
						lines.push(`Workflow is now ${activated.status}.`);
					} catch (err) {
						lines.push(`Activation failed: ${extractApiErrorMessage(err)}`);
					}
				}
			} else if (!nodes && !statusRules && !statusTransitions && !edges) {
				lines.push(
					"Add tasks as nodes, then link them with edges, via update_workflow.",
				);
			} else {
				lines.push(
					"Activate when ready with update_workflow (status: 'active').",
				);
			}

			return { content: [{ type: "text", text: lines.join("\n\n") }] };
		}

		case "update_workflow": {
			const {
				projectId,
				workflowId,
				name,
				description,
				status,
				nodes,
				statusRules,
				statusTransitions,
				edges,
			} = UpdateWorkflowSchema.parse(args);

			const lines: string[] = [];
			let hadFailure = false;

			if (name !== undefined || description !== undefined) {
				try {
					const workflow = await workflowClient.updateWorkflow(
						projectId,
						workflowId,
						{ name, description },
					);
					lines.push(`Renamed/updated:\n\n${formatWorkflow(workflow)}`);
				} catch (err) {
					hadFailure = true;
					lines.push(`Rename/update failed: ${extractApiErrorMessage(err)}`);
				}
			}

			let blockedByRevert = false;
			if (status === "draft") {
				try {
					const workflow = await workflowClient.revertWorkflowToDraft(
						projectId,
						workflowId,
					);
					lines.push(`Workflow is now ${workflow.status}.`);
				} catch (err) {
					blockedByRevert = true;
					hadFailure = true;
					lines.push(
						`Could not revert to draft: ${extractApiErrorMessage(err)}`,
					);
				}
			}

			const touchesGraph = Boolean(
				nodes || statusRules || statusTransitions || edges,
			);
			// Set when this call transparently reverts an active workflow to
			// draft on the caller's behalf (see below) — signals that we
			// should restore it to active afterward, so the caller doesn't
			// have to know about the draft requirement at all for the common
			// case of "just edit the graph, I don't care about lifecycle".
			let autoRevertedFromActive = false;
			if (touchesGraph && blockedByRevert) {
				hadFailure = true;
				lines.push(
					"Skipped all graph edits because the workflow could not be reverted to draft.",
				);
			} else if (touchesGraph) {
				let graph: WorkflowGraph | undefined;
				try {
					graph = await workflowClient.getWorkflow(projectId, workflowId);
				} catch (err) {
					hadFailure = true;
					lines.push(
						`Skipped all graph edits — could not load the current graph: ${extractApiErrorMessage(err)}`,
					);
				}

				// Every graph edit — including a node.set entry that ONLY
				// repositions an existing node — requires 'draft', with no
				// exception for position-only changes. Rather than making the
				// caller explicitly pass status: "draft" (and then a SEPARATE
				// follow-up call to reactivate) just to reposition a couple of
				// nodes, auto-revert here whenever no explicit status was
				// requested and the workflow is currently active, then restore
				// it to active at the end of this same call once the edits are
				// applied. If status was explicitly requested, respect that
				// instead of second-guessing it.
				if (
					graph &&
					status === undefined &&
					graph.workflow.status === "active"
				) {
					try {
						graph.workflow = await workflowClient.revertWorkflowToDraft(
							projectId,
							workflowId,
						);
						autoRevertedFromActive = true;
						lines.push(
							"Temporarily reverted to draft to apply the graph edits below (will re-activate once they're done).",
						);
					} catch (err) {
						hadFailure = true;
						lines.push(
							`Skipped all graph edits — could not revert to draft: ${extractApiErrorMessage(err)}`,
						);
						graph = undefined;
					}
				}

				if (graph && graph.workflow.status !== "draft") {
					hadFailure = true;
					lines.push(
						`Skipped all graph edits because the workflow is currently '${graph.workflow.status}', not 'draft'. Pass status: "draft" in this call to unlock edits.`,
					);
				} else if (graph) {
					const indexes = buildGraphIndexes(graph);
					const ctx: OrchestrationContext = {
						client: workflowClient,
						projectId,
						workflowId,
					};

					const nodesResult = await applyNodes(
						ctx,
						nodes?.set,
						nodes?.remove,
						indexes,
					);
					const rulesResult = await applyStatusRules(
						ctx,
						statusRules?.set,
						statusRules?.remove,
						indexes,
					);
					const transitionsResult = await applyStatusTransitions(
						ctx,
						statusTransitions?.set,
						statusTransitions?.remove,
						indexes,
					);
					const edgesResult = await applyEdges(
						ctx,
						edges?.add,
						edges?.remove,
						indexes,
					);

					for (const block of [
						formatCategoryResult("Nodes removed", nodesResult.removed),
						formatCategoryResult("Nodes set", nodesResult.set),
						formatCategoryResult("Status rules removed", rulesResult.removed),
						formatCategoryResult("Status rules set", rulesResult.set),
						formatCategoryResult(
							"Status transitions removed",
							transitionsResult.removed,
						),
						formatCategoryResult(
							"Status transitions set",
							transitionsResult.set,
						),
						formatCategoryResult("Edges removed", edgesResult.removed),
						formatCategoryResult("Edges added", edgesResult.added),
					]) {
						if (block) lines.push(block);
					}

					// Check the newly set nodes against the FULL graph — not just
					// against each other — so a node added in an earlier
					// update_workflow call still gets caught (see checkNodeSpacing's
					// doc comment for why this matters). A node is only excluded
					// from its OLD position when it was removed or successfully
					// repositioned this call — a nodes.set entry that FAILS for a
					// pre-existing node (e.g. an invalid taskId) must not vanish
					// from the check entirely; it keeps its old, still-current
					// position instead.
					const removedTaskIds = new Set(nodes?.remove ?? []);
					const newlyPositioned = successfullyPositionedNodes(
						nodes?.set,
						nodesResult.set,
					);
					const successfulTaskIds = new Set(
						newlyPositioned.map((n) => n.taskId),
					);
					const remainingAtOldPosition: PositionedNode[] = graph.nodes
						.filter(
							(n) =>
								!successfulTaskIds.has(n.task_id) &&
								!removedTaskIds.has(n.task_id),
						)
						.map((n) => ({ taskId: n.task_id, posX: n.pos_x, posY: n.pos_y }));

					const spacingWarning = checkNodeSpacing([
						...remainingAtOldPosition,
						...newlyPositioned,
					]);
					if (spacingWarning) lines.push(spacingWarning);

					if (
						anyHasFailure(
							nodesResult.removed,
							nodesResult.set,
							rulesResult.removed,
							rulesResult.set,
							transitionsResult.removed,
							transitionsResult.set,
							edgesResult.removed,
							edgesResult.added,
						)
					) {
						hadFailure = true;
					}
				}
			}

			if (status === "active" || status === "archived") {
				if (hadFailure) {
					lines.push(
						`Skipped ${status === "active" ? "activation" : "archiving"} because an earlier step in this call failed.`,
					);
				} else {
					try {
						const workflow =
							status === "active"
								? await workflowClient.activateWorkflow(projectId, workflowId)
								: await workflowClient.archiveWorkflow(projectId, workflowId);
						lines.push(`Workflow is now ${workflow.status}.`);
					} catch (err) {
						lines.push(
							`Could not ${status === "active" ? "activate" : "archive"}: ${extractApiErrorMessage(err)}`,
						);
					}
				}
			} else if (autoRevertedFromActive) {
				// No explicit status was requested, so restore the workflow to
				// the active state it was in before we temporarily reverted it
				// above — the caller asked for a graph edit, not a lifecycle
				// change, and shouldn't end up with the workflow stuck in draft
				// (and its automation paused) as a side effect of that edit.
				if (hadFailure) {
					lines.push(
						'Left the workflow in draft since a graph edit above failed — fix the failure and retry, or activate manually (status: "active") once ready.',
					);
				} else {
					try {
						const workflow = await workflowClient.activateWorkflow(
							projectId,
							workflowId,
						);
						lines.push(`Workflow is now ${workflow.status}.`);
					} catch (err) {
						lines.push(
							`Graph edits succeeded, but re-activating afterward failed: ${extractApiErrorMessage(err)}. The workflow is currently in draft — activate it manually once fixed.`,
						);
					}
				}
			}

			if (lines.length === 0) {
				lines.push("No changes requested.");
			}

			return { content: [{ type: "text", text: lines.join("\n\n") }] };
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

		default:
			throw new Error(`Unknown workflow tool: ${toolName}`);
	}
}
