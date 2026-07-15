import { createFileRoute, notFound } from "@tanstack/react-router";
import { AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { pluginsQueryOptions } from "@/lib/plugin-api";
import { RemoteComponent } from "@/lib/plugins/loader";
import { usePluginRegistry } from "@/lib/plugins/registry";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/plugins/$pluginId/$slug",
)({
	loader: async ({ context: { queryClient } }) => {
		await queryClient.ensureQueryData(pluginsQueryOptions);
	},
	component: ProjectPluginPage,
});

/**
 * Full-page route that renders a plugin's `project.page` extension-point
 * component for the given plugin/nav-item slug, resolved from the sidebar
 * nav items contributed by enabled plugins. This is the routed counterpart
 * to `<ExtensionPoint point="project.page">` — instead of embedding a
 * fragment inside a host page, the plugin owns the entire route.
 */
function ProjectPluginPage() {
	const { t } = useTranslation("errors");
	const { projectId, pluginId, slug } = Route.useParams();
	const { getNavItems, isLoading } = usePluginRegistry();

	if (isLoading) return null;

	const navItem = getNavItems("project").find(
		(item) => item.pluginId === pluginId && item.slug === slug,
	);

	if (!navItem) {
		throw notFound();
	}

	return (
		<div className="flex flex-col h-full">
			<RemoteComponent
				registration={navItem.registration}
				componentProps={{ projectId }}
				fallback={
					<div className="flex items-center gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive m-6">
						<AlertCircle className="size-3.5 shrink-0" />
						<span>
							{t("pluginLoadFailedPrefix")}{" "}
							<strong>{navItem.pluginName}</strong>{" "}
							{t("pluginLoadFailedSuffix")}
						</span>
					</div>
				}
			/>
		</div>
	);
}
