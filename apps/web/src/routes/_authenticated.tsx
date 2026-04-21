import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { useEffect } from "react";

import { AppSidebar } from "@/components/app-shell/app-sidebar";
import { NotificationBell } from "@/components/app-shell/notification-bell";
import {
	SidebarInset,
	SidebarProvider,
	SidebarTrigger,
} from "@/components/ui/sidebar";
import { isPasswordChangeRequired } from "@/lib/api-error";
import { currentUserQueryOptions } from "@/lib/auth-api";
import { connectSocket, disconnectSocket } from "@/lib/socket-client";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Pathless layout route that guards every route nested under it.
 *
 * Any route placed inside `routes/_authenticated/` automatically requires the
 * user to be signed in.  Unauthenticated visitors are redirected to the login
 * page (`/`).  The check reuses the cached React Query result so it only hits
 * the network when the cache is stale.
 */
export const Route = createFileRoute("/_authenticated")({
	beforeLoad: async ({ context: { queryClient } }) => {
		const user = await queryClient
			.fetchQuery(currentUserQueryOptions)
			.catch((err: unknown) => {
				// 403 AUTH_PASSWORD_CHANGE_REQUIRED means the user IS authenticated
				// but the backend blocks all endpoints. Send them to the dedicated page.
				if (isPasswordChangeRequired(err)) {
					throw redirect({ to: "/change-password" });
				}
				return null;
			});

		if (!user) {
			throw redirect({ to: "/" });
		}

		if (user.must_change_password) {
			throw redirect({ to: "/change-password" });
		}
	},
	component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
	// Open the Socket.IO connection as soon as the user is authenticated and
	// close it when they log out (component unmounts).
	const queryClient = useQueryClient();

	useEffect(() => {
		const socket = connectSocket();

		// Listen for notification events delivered to the user's personal room.
		const handleNotification = ({ type }: { type: string }) => {
			if (type === "notification.created") {
				queryClient.invalidateQueries({ queryKey: ["notifications"] });
			}
		};
		socket.on("notification", handleNotification);

		return () => {
			socket.off("notification", handleNotification);
			disconnectSocket();
		};
	}, [queryClient]);

	return (
		<SidebarProvider className="h-svh">
			<AppSidebar />
			<SidebarInset className="min-w-0 overflow-hidden">
				<header className="flex h-12 shrink-0 items-center gap-2 bg-background/85 backdrop-blur-xl px-4 sticky top-0 z-10">
					<div className="absolute inset-x-0 bottom-0 h-px bg-linear-to-r from-transparent via-border to-transparent" />
					<SidebarTrigger className="-ml-1 text-muted-foreground hover:text-foreground transition-colors" />
					<div className="w-px h-4 bg-border/60" />
					<div className="ml-auto">
						<NotificationBell />
					</div>
				</header>
				<div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
					<Outlet />
				</div>
			</SidebarInset>
		</SidebarProvider>
	);
}
