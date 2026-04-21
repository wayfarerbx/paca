// Namespace-based permission model.
//
// Instead of checking permissions on every received message, sockets are placed
// into namespace-scoped rooms at join time:
//
//   project:<projectId>:tasks  — receives all task.* events
//   project:<projectId>:docs   — receives all doc.* events
//
// Room membership is determined once when the client emits "join": the server
// fetches the user's project permissions, then joins only the rooms the user is
// allowed to see.  The Pub/Sub subscriber routes events directly to the correct
// room with a plain io.to(room).emit() — no per-socket checks needed.

// EventNamespace identifies the two permission-gated sub-domains.
export type EventNamespace = "tasks" | "docs";

// NAMESPACE_PERMISSIONS maps each namespace to the project permission required
// to receive events in that namespace.
export const NAMESPACE_PERMISSIONS: Record<EventNamespace, string> = {
	tasks: "tasks.read",
	docs: "docs.read",
};

// projectRoomName returns the Socket.IO room name for a project + namespace pair.
export function projectRoomName(
	projectId: string,
	namespace: EventNamespace,
): string {
	return `project:${projectId}:${namespace}`;
}

// eventNamespace infers the namespace from the event type prefix.
// Returns undefined for unknown event types.
export function eventNamespace(type: string): EventNamespace | undefined {
	if (type.startsWith("task.")) return "tasks";
	if (type.startsWith("doc.")) return "docs";
	return undefined;
}

// hasProjectPermission checks whether a permission map grants the requested
// permission.  Wildcard permissions are honoured:
//
//   "*"        → grants everything
//   "tasks.*"  → grants "tasks.read", "tasks.write", etc.
export function hasProjectPermission(
	permissions: Record<string, boolean>,
	required: string,
): boolean {
	if (permissions["*"] === true) return true;
	if (permissions[required] === true) return true;

	// Check the namespace wildcard (e.g. "tasks.*" covers "tasks.read").
	const dotIdx = required.lastIndexOf(".");
	if (dotIdx !== -1) {
		const wildcard = `${required.slice(0, dotIdx)}.*`;
		if (permissions[wildcard] === true) return true;
	}

	return false;
}
