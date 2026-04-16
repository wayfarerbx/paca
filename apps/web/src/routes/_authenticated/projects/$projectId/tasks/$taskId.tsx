import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { AlertCircle, ArrowLeft, Loader2 } from "lucide-react";

import { TaskDetailModal } from "@/components/projects/interactions/task-detail-modal";
import { taskQueryOptions } from "@/lib/interaction-api";
import {
	projectMembersQueryOptions,
	projectQueryOptions,
	taskStatusesQueryOptions,
	taskTypesQueryOptions,
} from "@/lib/project-api";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/tasks/$taskId",
)({
	component: TaskDetailPage,
});

function TaskDetailPage() {
	const { projectId, taskId } = Route.useParams();

	const { data: project } = useQuery(projectQueryOptions(projectId));
	const { data: taskStatuses = [] } = useQuery(
		taskStatusesQueryOptions(projectId),
	);
	const { data: taskTypes = [] } = useQuery(taskTypesQueryOptions(projectId));
	const { data: members = [] } = useQuery(
		projectMembersQueryOptions(projectId),
	);
	const { data: task = null, isLoading } = useQuery(
		taskQueryOptions(projectId, taskId),
	);

	if (isLoading) {
		return (
			<div className="flex h-full items-center justify-center">
				<Loader2 className="size-6 animate-spin text-muted-foreground" />
			</div>
		);
	}

	if (!task) {
		return (
			<div className="flex h-full flex-col items-center justify-center gap-4 text-muted-foreground/60">
				<AlertCircle className="size-10" />
				<div className="text-center">
					<p className="text-base font-medium text-foreground/70">
						Task not found
					</p>
					<p className="text-sm mt-1">
						This task may have been deleted or the link is invalid.
					</p>
				</div>
				<Link
					to="/projects/$projectId"
					params={{ projectId }}
					className="flex items-center gap-1.5 rounded-lg border border-border/60 px-4 py-2 text-sm font-medium text-foreground/70 hover:bg-muted/50 transition-colors mt-2"
				>
					<ArrowLeft className="size-4" />
					Back to project
				</Link>
			</div>
		);
	}

	return (
		<div className="flex h-full flex-col overflow-hidden bg-background">
			{/* Back navigation strip */}
			<div className="shrink-0 border-b border-border/40 px-5 py-2.5 flex items-center gap-3">
				<Link
					to="/projects/$projectId"
					params={{ projectId }}
					className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
				>
					<ArrowLeft className="size-3.5" />
					{project?.name ?? "Project"}
				</Link>
			</div>

			{/* Task detail — full height below nav strip */}
			<div className="flex-1 overflow-hidden">
				<TaskDetailModal
					task={task}
					open
					onOpenChange={() => {}}
					statuses={taskStatuses}
					taskTypes={taskTypes}
					members={members}
					projectName={project?.name}
					projectId={projectId}
					mode="page"
					canEdit
				/>
			</div>
		</div>
	);
}
