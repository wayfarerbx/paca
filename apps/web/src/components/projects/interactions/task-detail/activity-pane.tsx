import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { ActivityPane } from "@/components/shared/activity-pane";
import {
	type Activity,
	addComment,
	listTaskActivities,
	sprintsQueryOptions,
} from "@/lib/interaction-api";
import { projectMembersQueryOptions } from "@/lib/project-api";
import { describeTaskChange } from "./activity-item";

type FieldChange = {
	field: string;
	old?: unknown;
	new?: unknown;
};

interface TaskActivityPaneProps {
	projectId: string;
	taskId: string;
	canEdit?: boolean;
}

export function TaskActivityPane({
	projectId,
	taskId,
	canEdit = true,
}: TaskActivityPaneProps) {
	const { data: membersData } = useQuery(projectMembersQueryOptions(projectId));
	const { data: sprintsData } = useQuery(sprintsQueryOptions(projectId));

	const nameMaps = useMemo(() => {
		const members: Record<string, string> = {};
		for (const m of membersData ?? []) {
			members[m.id] = m.full_name || m.username;
		}
		const sprints: Record<string, string> = {};
		for (const s of sprintsData ?? []) {
			sprints[s.id] = s.name;
		}
		return { members, sprints };
	}, [membersData, sprintsData]);

	const describeActivity = useCallback(
		(entry: Activity): string => {
			const c = entry.content ?? {};
			switch (entry.activity_type) {
				case "task.created":
					return "created this task";
				case "task.deleted":
					return "deleted this task";
				case "task.updated": {
					const changes = c.changes as FieldChange[] | undefined;
					if (changes && changes.length === 1) {
						return describeTaskChange(changes[0], nameMaps);
					}
					if (changes && changes.length > 1) {
						return changes
							.map((ch) => describeTaskChange(ch, nameMaps))
							.join("; ");
					}
					return "updated this task";
				}
				case "task.attachment.added":
					return `added attachment${c.file_name ? `: ${c.file_name}` : ""}`;
				case "task.attachment.removed":
					return `removed attachment${c.file_name ? `: ${c.file_name}` : ""}`;
				default:
					return (c._description as string | undefined) ?? "made a change";
			}
		},
		[nameMaps],
	);

	const queryKey = [
		"projects",
		projectId,
		"tasks",
		taskId,
		"activities",
	] as const;

	return (
		<ActivityPane<Activity>
			projectId={projectId}
			entityId={taskId}
			queryKey={queryKey}
			queryFn={() => listTaskActivities(projectId, taskId)}
			addComment={
				canEdit ? (text) => addComment(projectId, taskId, text) : undefined
			}
			describeActivity={describeActivity}
			getCommentText={(content) => (content as { text?: string }).text ?? ""}
		/>
	);
}
