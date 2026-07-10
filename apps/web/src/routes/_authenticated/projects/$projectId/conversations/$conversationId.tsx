import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ConversationView } from "@/components/projects/agents/conversation-view";
import { RouteErrorComponent } from "@/components/route-error-boundary";
import { useProjectRealtime } from "@/hooks/use-project-realtime";
import {
	conversationEventsQueryOptions,
	conversationQueryOptions,
} from "@/lib/agent-api";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/conversations/$conversationId",
)({
	loader: async ({
		context: { queryClient },
		params: { projectId, conversationId },
	}) => {
		await Promise.all([
			queryClient.ensureQueryData(
				conversationQueryOptions(projectId, conversationId),
			),
			queryClient.ensureQueryData(
				conversationEventsQueryOptions(projectId, conversationId),
			),
		]);
	},
	// Without an errorComponent, a loader failure (e.g. deleted conversation,
	// API 500) bubbles up and crashes the router's internal Lazy wrapper.
	errorComponent: ({ error }) => <RouteErrorComponent error={error} />,
	component: ConversationPage,
});

function ConversationPage() {
	const { t } = useTranslation("projects");
	const { projectId, conversationId } = Route.useParams();
	const { data: conversation } = useQuery(
		conversationQueryOptions(projectId, conversationId),
	);

	useProjectRealtime(projectId);

	return (
		<div className="flex flex-col h-full overflow-hidden bg-background">
			{/* Back navigation */}
			<div className="shrink-0 border-b border-border/40 px-5 py-2.5 flex items-center gap-3">
				<Link
					to="/projects/$projectId/agents/$agentId"
					params={{ projectId, agentId: conversation?.agent_id ?? "" }}
					hash="conversations"
					className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
				>
					<ArrowLeft className="size-3.5" />
					{t("common.back", "Back")}
				</Link>
			</div>

			{/* Conversation view */}
			<div className="flex-1 overflow-hidden">
				<ConversationView
					projectId={projectId}
					conversationId={conversationId}
				/>
			</div>
		</div>
	);
}
