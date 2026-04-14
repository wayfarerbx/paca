import { createFileRoute } from "@tanstack/react-router";

import { InteractionLayout } from "@/components/projects/interactions/interaction-layout";
import { usePermissions } from "@/hooks/use-permissions";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/interactions/backlog",
)({
	component: BacklogPage,
});

function BacklogPage() {
	const { projectId } = Route.useParams();
	const { hasPermission } = usePermissions();

	const canCreate = hasPermission("tasks.write");
	const canEdit = hasPermission("tasks.write");
	const canManageViews = hasPermission("projects.write");

	return (
		<InteractionLayout
			projectId={projectId}
			interactionKey={`backlog:${projectId}`}
			title="Product Backlog"
			description="All work items not assigned to a sprint."
			canCreate={canCreate}
			canEdit={canEdit}
			canManageViews={canManageViews}
			sprintId={null}
			context="backlog"
		/>
	)
}
