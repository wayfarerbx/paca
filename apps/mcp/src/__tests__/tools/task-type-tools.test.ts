import { describe, expect, it, vi } from "vitest";

vi.mock("../../utils/index.js", () => ({
	formatList: vi.fn((items: any[], fn: any) => items.map(fn).join("---")),
}));

import {
	getTaskStatusTools,
	getTaskTypeTools,
	handleTaskTypeTool,
} from "../../tools/task-type-tools.js";

const taskType = {
	id: "ty1",
	project_id: "p1",
	name: "Story",
	icon: "📖",
	color: "#blue",
	description: "User story",
	is_default: false,
	is_system: false,
	created_at: "2024-01-01T00:00:00Z",
	updated_at: "2024-01-01T00:00:00Z",
};

const taskStatus = {
	id: "st1",
	project_id: "p1",
	name: "In Progress",
	color: "#yellow",
	position: 1,
	category: "inprogress" as const,
	is_default: false,
	created_at: "2024-01-01T00:00:00Z",
	updated_at: "2024-01-01T00:00:00Z",
};

function makeClient(overrides: Record<string, any> = {}) {
	return {
		listTaskTypes: vi.fn().mockResolvedValue([taskType]),
		createTaskType: vi.fn().mockResolvedValue(taskType),
		updateTaskType: vi.fn().mockResolvedValue({ ...taskType, name: "Epic" }),
		deleteTaskType: vi.fn().mockResolvedValue(undefined),
		setDefaultTaskType: vi
			.fn()
			.mockResolvedValue({ ...taskType, is_default: true }),
		listTaskStatuses: vi.fn().mockResolvedValue([taskStatus]),
		createTaskStatus: vi.fn().mockResolvedValue(taskStatus),
		updateTaskStatus: vi
			.fn()
			.mockResolvedValue({ ...taskStatus, name: "Done" }),
		deleteTaskStatus: vi.fn().mockResolvedValue(undefined),
		setDefaultTaskStatus: vi
			.fn()
			.mockResolvedValue({ ...taskStatus, is_default: true }),
		...overrides,
	} as any;
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

describe("getTaskTypeTools", () => {
	it("returns 5 task type tools", () => {
		expect(getTaskTypeTools()).toHaveLength(5);
	});
});

describe("getTaskStatusTools", () => {
	it("returns 5 task status tools", () => {
		expect(getTaskStatusTools()).toHaveLength(5);
	});
});

// ---------------------------------------------------------------------------
// list_task_types
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – list_task_types", () => {
	it("calls client.listTaskTypes with projectId", async () => {
		const client = makeClient();
		await handleTaskTypeTool("list_task_types", { projectId: "p1" }, client);
		expect(client.listTaskTypes).toHaveBeenCalledWith("p1");
	});

	it("includes 'Task Types:' header and type name in response", async () => {
		const result = await handleTaskTypeTool(
			"list_task_types",
			{ projectId: "p1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("Task Types:");
		expect(result.content[0].text).toContain("Story");
	});
});

// ---------------------------------------------------------------------------
// create_task_type
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – create_task_type", () => {
	it("calls client.createTaskType with mapped input", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"create_task_type",
			{ projectId: "p1", name: "Bug", icon: "🐛", color: "#red" },
			client,
		);
		expect(client.createTaskType).toHaveBeenCalledWith("p1", {
			name: "Bug",
			icon: "🐛",
			color: "#red",
			description: undefined,
		});
	});

	it("includes 'created successfully' in response", async () => {
		const result = await handleTaskTypeTool(
			"create_task_type",
			{ projectId: "p1", name: "Bug" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("created successfully");
	});
});

// ---------------------------------------------------------------------------
// update_task_type
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – update_task_type", () => {
	it("calls client.updateTaskType with typeId and mapped input", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"update_task_type",
			{ projectId: "p1", typeId: "ty1", name: "Epic" },
			client,
		);
		expect(client.updateTaskType).toHaveBeenCalledWith("p1", "ty1", {
			name: "Epic",
			icon: undefined,
			color: undefined,
			description: undefined,
		});
	});

	it("includes 'updated successfully' in response", async () => {
		const result = await handleTaskTypeTool(
			"update_task_type",
			{ projectId: "p1", typeId: "ty1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("updated successfully");
	});
});

// ---------------------------------------------------------------------------
// delete_task_type
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – delete_task_type", () => {
	it("calls client.deleteTaskType with projectId and typeId", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"delete_task_type",
			{ projectId: "p1", typeId: "ty1" },
			client,
		);
		expect(client.deleteTaskType).toHaveBeenCalledWith("p1", "ty1");
	});

	it("includes 'deleted successfully' and typeId in response", async () => {
		const result = await handleTaskTypeTool(
			"delete_task_type",
			{ projectId: "p1", typeId: "ty1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("ty1");
		expect(result.content[0].text).toContain("deleted successfully");
	});
});

// ---------------------------------------------------------------------------
// set_default_task_type
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – set_default_task_type", () => {
	it("calls client.setDefaultTaskType with projectId and typeId", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"set_default_task_type",
			{ projectId: "p1", typeId: "ty1" },
			client,
		);
		expect(client.setDefaultTaskType).toHaveBeenCalledWith("p1", "ty1");
	});

	it("includes 'Default task type set' in response", async () => {
		const result = await handleTaskTypeTool(
			"set_default_task_type",
			{ projectId: "p1", typeId: "ty1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("Default task type set");
	});
});

// ---------------------------------------------------------------------------
// list_task_statuses
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – list_task_statuses", () => {
	it("calls client.listTaskStatuses with projectId", async () => {
		const client = makeClient();
		await handleTaskTypeTool("list_task_statuses", { projectId: "p1" }, client);
		expect(client.listTaskStatuses).toHaveBeenCalledWith("p1");
	});

	it("includes 'Task Statuses:' header and status name in response", async () => {
		const result = await handleTaskTypeTool(
			"list_task_statuses",
			{ projectId: "p1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("Task Statuses:");
		expect(result.content[0].text).toContain("In Progress");
	});
});

// ---------------------------------------------------------------------------
// create_task_status
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – create_task_status", () => {
	it("calls client.createTaskStatus with mapped input", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"create_task_status",
			{
				projectId: "p1",
				name: "Review",
				color: "#purple",
				category: "inprogress",
			},
			client,
		);
		expect(client.createTaskStatus).toHaveBeenCalledWith("p1", {
			name: "Review",
			color: "#purple",
			category: "inprogress",
			position: 0,
		});
	});

	it("includes 'created successfully' in response", async () => {
		const result = await handleTaskTypeTool(
			"create_task_status",
			{ projectId: "p1", name: "Review", category: "inprogress" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("created successfully");
	});
});

// ---------------------------------------------------------------------------
// update_task_status
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – update_task_status", () => {
	it("calls client.updateTaskStatus with statusId and mapped input", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"update_task_status",
			{ projectId: "p1", statusId: "st1", name: "Done", position: 5 },
			client,
		);
		expect(client.updateTaskStatus).toHaveBeenCalledWith("p1", "st1", {
			name: "Done",
			color: undefined,
			category: undefined,
			position: 5,
		});
	});

	it("includes 'updated successfully' in response", async () => {
		const result = await handleTaskTypeTool(
			"update_task_status",
			{ projectId: "p1", statusId: "st1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("updated successfully");
	});
});

// ---------------------------------------------------------------------------
// delete_task_status
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – delete_task_status", () => {
	it("calls client.deleteTaskStatus with projectId and statusId", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"delete_task_status",
			{ projectId: "p1", statusId: "st1" },
			client,
		);
		expect(client.deleteTaskStatus).toHaveBeenCalledWith("p1", "st1");
	});

	it("includes 'deleted successfully' and statusId in response", async () => {
		const result = await handleTaskTypeTool(
			"delete_task_status",
			{ projectId: "p1", statusId: "st1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("st1");
		expect(result.content[0].text).toContain("deleted successfully");
	});
});

// ---------------------------------------------------------------------------
// set_default_task_status
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – set_default_task_status", () => {
	it("calls client.setDefaultTaskStatus with projectId and statusId", async () => {
		const client = makeClient();
		await handleTaskTypeTool(
			"set_default_task_status",
			{ projectId: "p1", statusId: "st1" },
			client,
		);
		expect(client.setDefaultTaskStatus).toHaveBeenCalledWith("p1", "st1");
	});

	it("includes 'Default task status set' in response", async () => {
		const result = await handleTaskTypeTool(
			"set_default_task_status",
			{ projectId: "p1", statusId: "st1" },
			makeClient(),
		);
		expect(result.content[0].text).toContain("Default task status set");
	});
});

// ---------------------------------------------------------------------------
// unknown tool
// ---------------------------------------------------------------------------

describe("handleTaskTypeTool – unknown tool", () => {
	it("throws for an unknown tool name", async () => {
		await expect(
			handleTaskTypeTool("unknown", {}, makeClient()),
		).rejects.toThrow();
	});
});
