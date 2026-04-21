// Hook that joins a project's realtime rooms and invalidates React Query
// caches when events arrive from the server.
//
// Usage: call once per project layout/page:
//
//   useProjectRealtime(projectId);
//
// The hook:
//   1. Joins the project rooms on mount (server grants tasks/docs room
//      access based on the caller's permissions).
//   2. Listens for "event" messages and maps event types to query
//      invalidations using the same query key prefixes as the API helpers.
//   3. Leaves the project rooms and removes the listener on unmount.
//
// Query invalidation strategy
// ---------------------------
// task.* events  → invalidate ["projects", projectId, "tasks"]
//                  This covers allTasksQueryOptions, taskQueryOptions,
//                  sprintTasksQueryOptions, epicTasksQueryOptions, etc.
// doc.* events   → invalidate ["projects", projectId, "docs"]
//                  This covers docFoldersQueryOptions, docListQueryOptions,
//                  docQueryOptions, etc.
//
// More granular invalidations (e.g. specific taskId) are avoided intentionally:
// the event payload fields are not yet stabilised and broad invalidation is
// safer and simpler.

import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import {
	connectSocket,
	joinProject,
	leaveProject,
	type RealtimeEvent,
} from "@/lib/socket-client";

export function useProjectRealtime(projectId: string): void {
	const queryClient = useQueryClient();

	useEffect(() => {
		const socket = connectSocket();

		// Subscribe to the project rooms.
		joinProject(projectId);

		function handleEvent(event: RealtimeEvent) {
			const { type } = event;

			if (type.startsWith("task.")) {
				void queryClient.invalidateQueries({
					queryKey: ["projects", projectId, "tasks"],
				});
				return;
			}

			if (type.startsWith("doc.")) {
				void queryClient.invalidateQueries({
					queryKey: ["projects", projectId, "docs"],
				});
				return;
			}
		}

		socket.on("event", handleEvent);

		return () => {
			socket.off("event", handleEvent);
			leaveProject(projectId);
		};
	}, [projectId, queryClient]);
}
