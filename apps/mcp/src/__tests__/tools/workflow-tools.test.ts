import { describe, expect, it, vi } from "vitest";

import {
	getWorkflowTools,
	handleWorkflowTool,
} from "../../tools/workflow-tools.js";

const workflow = {
	id: "wf1",
	project_id: "p1",
	name: "Release pipeline",
	description: "",
	status: "draft",
	created_at: "2024-01-01T00:00:00Z",
	updated_at: "2024-01-01T00:00:00Z",
};

const statusRule = {
	id: "r1",
	workflow_id: "wf1",
	status_id: "s-ready",
	assignee_member_id: "m1",
	created_at: "2024-01-01T00:00:00Z",
	updated_at: "2024-01-01T00:00:00Z",
};

const statusTransition = {
	id: "tr1",
	workflow_id: "wf1",
	status_id: "s-ready",
	next_status_id: "s-done",
	created_at: "2024-01-01T00:00:00Z",
	updated_at: "2024-01-01T00:00:00Z",
};

const graph = {
	workflow,
	nodes: [
		{
			id: "n1",
			workflow_id: "wf1",
			task_id: "t1",
			pos_x: 0,
			pos_y: 0,
			created_at: "2024-01-01T00:00:00Z",
			updated_at: "2024-01-01T00:00:00Z",
		},
	],
	edges: [
		{
			id: "e1",
			workflow_id: "wf1",
			source_node_id: "n1",
			target_node_id: "n2",
			created_at: "2024-01-01T00:00:00Z",
		},
	],
	status_rules: [statusRule],
	status_transitions: [statusTransition],
};

function makeWorkflowClient(overrides: Record<string, any> = {}) {
	return {
		listWorkflows: vi.fn().mockResolvedValue([workflow]),
		getWorkflow: vi.fn().mockResolvedValue(graph),
		createWorkflow: vi.fn().mockResolvedValue(workflow),
		updateWorkflow: vi.fn().mockResolvedValue({ ...workflow, name: "Renamed" }),
		deleteWorkflow: vi.fn().mockResolvedValue(undefined),
		activateWorkflow: vi
			.fn()
			.mockResolvedValue({ ...workflow, status: "active" }),
		archiveWorkflow: vi
			.fn()
			.mockResolvedValue({ ...workflow, status: "archived" }),
		revertWorkflowToDraft: vi
			.fn()
			.mockResolvedValue({ ...workflow, status: "draft" }),
		addWorkflowNode: vi.fn().mockResolvedValue(graph.nodes[0]),
		removeWorkflowNode: vi.fn().mockResolvedValue(undefined),
		setWorkflowStatusRule: vi.fn().mockResolvedValue(statusRule),
		removeWorkflowStatusRule: vi.fn().mockResolvedValue(undefined),
		setWorkflowStatusTransition: vi.fn().mockResolvedValue(statusTransition),
		removeWorkflowStatusTransition: vi.fn().mockResolvedValue(undefined),
		addWorkflowEdge: vi.fn().mockResolvedValue(graph.edges[0]),
		removeWorkflowEdge: vi.fn().mockResolvedValue(undefined),
		...overrides,
	} as any;
}

// ---------------------------------------------------------------------------
// getWorkflowTools
// ---------------------------------------------------------------------------

describe("getWorkflowTools", () => {
	it("returns 16 tools", () => {
		expect(getWorkflowTools()).toHaveLength(16);
	});

	it("includes all expected tool names", () => {
		const names = getWorkflowTools().map((t) => t.name);
		for (const n of [
			"list_workflows",
			"get_workflow",
			"create_workflow",
			"update_workflow",
			"delete_workflow",
			"activate_workflow",
			"archive_workflow",
			"revert_workflow_to_draft",
			"add_workflow_node",
			"remove_workflow_node",
			"set_workflow_status_rule",
			"remove_workflow_status_rule",
			"set_workflow_status_transition",
			"remove_workflow_status_transition",
			"add_workflow_edge",
			"remove_workflow_edge",
		]) {
			expect(names).toContain(n);
		}
	});
});

// ---------------------------------------------------------------------------
// handleWorkflowTool
// ---------------------------------------------------------------------------

describe("handleWorkflowTool", () => {
	it("list_workflows returns formatted workflow list", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"list_workflows",
			{ projectId: "p1" },
			client,
		);
		expect(client.listWorkflows).toHaveBeenCalledWith("p1", undefined);
		expect(result.content[0].text).toContain("Release pipeline");
	});

	it("list_workflows reports an empty project", async () => {
		const client = makeWorkflowClient({
			listWorkflows: vi.fn().mockResolvedValue([]),
		});
		const result = await handleWorkflowTool(
			"list_workflows",
			{ projectId: "p1" },
			client,
		);
		expect(result.content[0].text).toContain("No automation workflows");
	});

	it("get_workflow returns the formatted graph including nodes and edges", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"get_workflow",
			{ projectId: "p1", workflowId: "wf1" },
			client,
		);
		expect(client.getWorkflow).toHaveBeenCalledWith("p1", "wf1");
		expect(result.content[0].text).toContain("node n1");
		expect(result.content[0].text).toContain("edge e1");
	});

	it("create_workflow creates a draft workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"create_workflow",
			{ projectId: "p1", name: "Release pipeline" },
			client,
		);
		expect(client.createWorkflow).toHaveBeenCalledWith("p1", {
			name: "Release pipeline",
			description: undefined,
		});
		expect(result.content[0].text).toContain("Created draft workflow");
	});

	it("update_workflow renames a workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"update_workflow",
			{ projectId: "p1", workflowId: "wf1", name: "Renamed" },
			client,
		);
		expect(client.updateWorkflow).toHaveBeenCalledWith("p1", "wf1", {
			name: "Renamed",
			description: undefined,
		});
		expect(result.content[0].text).toContain("Renamed");
	});

	it("delete_workflow deletes a workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"delete_workflow",
			{ projectId: "p1", workflowId: "wf1" },
			client,
		);
		expect(client.deleteWorkflow).toHaveBeenCalledWith("p1", "wf1");
		expect(result.content[0].text).toContain("deleted successfully");
	});

	it("activate_workflow activates a workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"activate_workflow",
			{ projectId: "p1", workflowId: "wf1" },
			client,
		);
		expect(client.activateWorkflow).toHaveBeenCalledWith("p1", "wf1");
		expect(result.content[0].text).toContain("active");
	});

	it("archive_workflow archives a workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"archive_workflow",
			{ projectId: "p1", workflowId: "wf1" },
			client,
		);
		expect(client.archiveWorkflow).toHaveBeenCalledWith("p1", "wf1");
		expect(result.content[0].text).toContain("archived");
	});

	it("revert_workflow_to_draft reverts a workflow", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"revert_workflow_to_draft",
			{ projectId: "p1", workflowId: "wf1" },
			client,
		);
		expect(client.revertWorkflowToDraft).toHaveBeenCalledWith("p1", "wf1");
		expect(result.content[0].text).toContain("draft");
	});

	it("add_workflow_node adds a task as a node", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"add_workflow_node",
			{
				projectId: "p1",
				workflowId: "wf1",
				taskId: "t1",
			},
			client,
		);
		expect(client.addWorkflowNode).toHaveBeenCalledWith("p1", "wf1", {
			task_id: "t1",
		});
		expect(result.content[0].text).toContain("n1");
	});

	it("remove_workflow_node removes a node", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"remove_workflow_node",
			{ projectId: "p1", workflowId: "wf1", nodeId: "n1" },
			client,
		);
		expect(client.removeWorkflowNode).toHaveBeenCalledWith("p1", "wf1", "n1");
		expect(result.content[0].text).toContain("removed");
	});

	it("set_workflow_status_rule saves a rule", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"set_workflow_status_rule",
			{
				projectId: "p1",
				workflowId: "wf1",
				statusId: "s-ready",
				assigneeMemberId: "m1",
			},
			client,
		);
		expect(client.setWorkflowStatusRule).toHaveBeenCalledWith("p1", "wf1", {
			status_id: "s-ready",
			assignee_member_id: "m1",
		});
		expect(result.content[0].text).toContain("Rule saved");
	});

	it("remove_workflow_status_rule removes a rule", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"remove_workflow_status_rule",
			{ projectId: "p1", workflowId: "wf1", ruleId: "r1" },
			client,
		);
		expect(client.removeWorkflowStatusRule).toHaveBeenCalledWith(
			"p1",
			"wf1",
			"r1",
		);
		expect(result.content[0].text).toContain("removed");
	});

	it("set_workflow_status_transition saves a transition", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"set_workflow_status_transition",
			{
				projectId: "p1",
				workflowId: "wf1",
				statusId: "s-ready",
				nextStatusId: "s-done",
			},
			client,
		);
		expect(client.setWorkflowStatusTransition).toHaveBeenCalledWith(
			"p1",
			"wf1",
			{ status_id: "s-ready", next_status_id: "s-done" },
		);
		expect(result.content[0].text).toContain("Transition saved");
	});

	it("set_workflow_status_transition marks a status as terminal when nextStatusId is omitted", async () => {
		const client = makeWorkflowClient({
			setWorkflowStatusTransition: vi
				.fn()
				.mockResolvedValue({ ...statusTransition, next_status_id: null }),
		});
		const result = await handleWorkflowTool(
			"set_workflow_status_transition",
			{ projectId: "p1", workflowId: "wf1", statusId: "s-done" },
			client,
		);
		expect(client.setWorkflowStatusTransition).toHaveBeenCalledWith(
			"p1",
			"wf1",
			{ status_id: "s-done", next_status_id: null },
		);
		expect(result.content[0].text).toContain("done status");
	});

	it("remove_workflow_status_transition removes a transition", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"remove_workflow_status_transition",
			{ projectId: "p1", workflowId: "wf1", transitionId: "tr1" },
			client,
		);
		expect(client.removeWorkflowStatusTransition).toHaveBeenCalledWith(
			"p1",
			"wf1",
			"tr1",
		);
		expect(result.content[0].text).toContain("removed");
	});

	it("add_workflow_edge links two nodes", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"add_workflow_edge",
			{
				projectId: "p1",
				workflowId: "wf1",
				sourceNodeId: "n1",
				targetNodeId: "n2",
			},
			client,
		);
		expect(client.addWorkflowEdge).toHaveBeenCalledWith("p1", "wf1", {
			source_node_id: "n1",
			target_node_id: "n2",
		});
		expect(result.content[0].text).toContain("Linked");
	});

	it("remove_workflow_edge removes an edge", async () => {
		const client = makeWorkflowClient();
		const result = await handleWorkflowTool(
			"remove_workflow_edge",
			{ projectId: "p1", workflowId: "wf1", edgeId: "e1" },
			client,
		);
		expect(client.removeWorkflowEdge).toHaveBeenCalledWith("p1", "wf1", "e1");
		expect(result.content[0].text).toContain("removed");
	});

	it("throws for an unknown tool name", async () => {
		const client = makeWorkflowClient();
		await expect(
			handleWorkflowTool("not_a_real_tool", {}, client),
		).rejects.toThrow("Unknown workflow tool");
	});
});
