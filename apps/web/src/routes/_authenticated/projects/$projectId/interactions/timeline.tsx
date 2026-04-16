import { createFileRoute } from "@tanstack/react-router";

import { InteractionLayout } from "@/components/projects/interactions/interaction-layout";
import { usePermissions } from "@/hooks/use-permissions";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/interactions/timeline",
)({
	component: TimelinePage,
});

function TimelinePage() {
	const { projectId } = Route.useParams();
	const { hasPermission } = usePermissions();

	const canCreate = hasPermission("tasks.write");
	const canEdit = hasPermission("tasks.write");
	const canManageViews = hasPermission("projects.write");

	return (
		<InteractionLayout
			projectId={projectId}
			interactionKey={`timeline:${projectId}`}
			title="Timeline"
			description="Epics and long-horizon planning on a roadmap."
			canCreate={canCreate}
			canEdit={canEdit}
			canManageViews={canManageViews}
			sprintId={null}
			context="timeline"
		/>
	);
}
