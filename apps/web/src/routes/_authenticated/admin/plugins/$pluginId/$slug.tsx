import { createFileRoute, notFound, redirect } from "@tanstack/react-router";
import { AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { myPermissionsQueryOptions } from "@/lib/admin-api";
import { hasPermission } from "@/lib/permissions";
import { pluginsQueryOptions } from "@/lib/plugin-api";
import { RemoteComponent } from "@/lib/plugins/loader";
import { usePluginRegistry } from "@/lib/plugins/registry";

export const Route = createFileRoute(
	"/_authenticated/admin/plugins/$pluginId/$slug",
)({
	beforeLoad: async ({ context: { queryClient } }) => {
		const permissions = await queryClient
			.fetchQuery(myPermissionsQueryOptions)
			.catch(() => [] as string[]);

		if (!hasPermission(permissions, "users.write")) {
			throw redirect({ to: "/home" });
		}
	},
	loader: async ({ context: { queryClient } }) => {
		await queryClient.ensureQueryData(pluginsQueryOptions);
	},
	component: AdminPluginPage,
});

/**
 * Full-page route that renders a plugin's `admin.page` extension-point
 * component for the given plugin/nav-item slug — the admin/global-scope
 * counterpart to `ProjectPluginPage`. Used for cross-project plugin
 * dashboards (e.g. a "total logged time across all projects" summary).
 */
function AdminPluginPage() {
	const { t } = useTranslation("errors");
	const { pluginId, slug } = Route.useParams();
	const { getNavItems, isLoading } = usePluginRegistry();

	if (isLoading) return null;

	const navItem = getNavItems("admin").find(
		(item) => item.pluginId === pluginId && item.slug === slug,
	);

	if (!navItem) {
		throw notFound();
	}

	return (
		<div className="flex flex-col h-full">
			<RemoteComponent
				registration={navItem.registration}
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
