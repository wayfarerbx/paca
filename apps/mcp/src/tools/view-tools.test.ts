import { describe, expect, it, vi } from "vitest";
import { handleViewTool } from "./view-tools.js";

const makeView = (overrides: object = {}) => ({
	id: "view-1",
	name: "Done",
	context: "backlog",
	sprint_id: null,
	position: 0,
	created_at: "2024-01-01T00:00:00Z",
	...overrides,
});

describe("handleViewTool — create_view", () => {
	it("passes context as the 3rd argument to client.createView, not in the body", async () => {
		const createView = vi.fn().mockResolvedValue(makeView());
		const client = { createView } as any;

		await handleViewTool(
			"create_view",
			{ projectId: "proj-1", name: "Done", context: "backlog", viewType: "table" },
			client,
		);

		expect(createView).toHaveBeenCalledOnce();
		expect(createView).toHaveBeenCalledWith(
			"proj-1",
			{ name: "Done", view_type: "table" },
			"backlog",
			null,
		);
	});

	it("does not embed context or sprint_id in the request body", async () => {
		const createView = vi.fn().mockResolvedValue(makeView());
		const client = { createView } as any;

		await handleViewTool(
			"create_view",
			{ projectId: "proj-1", name: "Done", context: "backlog", viewType: "table" },
			client,
		);

		const body = createView.mock.calls[0][1];
		expect(body).not.toHaveProperty("context");
		expect(body).not.toHaveProperty("sprint_id");
	});

	it("passes sprintId as the 4th argument when provided", async () => {
		const sprintId = "sprint-uuid-1";
		const createView = vi.fn().mockResolvedValue(makeView({ context: "sprint", sprint_id: sprintId }));
		const client = { createView } as any;

		await handleViewTool(
			"create_view",
			{ projectId: "proj-1", name: "Sprint Board", context: "sprint", viewType: "board", sprintId },
			client,
		);

		expect(createView).toHaveBeenCalledWith(
			"proj-1",
			{ name: "Sprint Board", view_type: "board" },
			"sprint",
			sprintId,
		);
	});
});
