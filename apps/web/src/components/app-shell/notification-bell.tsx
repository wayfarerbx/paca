import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Bell } from "lucide-react";
import { useCallback } from "react";

import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
	type Notification,
	markAllNotificationsAsRead,
	markNotificationAsRead,
	notificationsQueryOptions,
} from "@/lib/notification-api";

function timeAgo(date: string): string {
	const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
	if (seconds < 60) return "just now";
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

function notificationText(n: Notification): string {
	if (n.type === "assigned") {
		return `${n.actor_full_name} assigned you to task #${n.task_number} — ${n.task_title}`;
	}
	return `${n.actor_full_name} mentioned you in #${n.task_number} — ${n.task_title}`;
}

export function NotificationBell() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const { data } = useQuery(notificationsQueryOptions);

	const unreadCount = data?.unread_count ?? 0;
	const notifications = data?.items ?? [];

	const { mutate: markRead } = useMutation({
		mutationFn: markNotificationAsRead,
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["notifications"] }),
	});

	const { mutate: markAllRead } = useMutation({
		mutationFn: markAllNotificationsAsRead,
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["notifications"] }),
	});

	const handleNotificationClick = useCallback(
		(n: Notification) => {
			if (!n.read_at) markRead(n.id);
			navigate({
				to: "/projects/$projectId/tasks/$taskId",
				params: { projectId: n.project_id, taskId: n.task_id },
			});
		},
		[markRead, navigate],
	);

	return (
		<Popover>
			<PopoverTrigger
				className="relative inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
				aria-label="Notifications"
			>
				<Bell className="h-4 w-4" />
				{unreadCount > 0 && (
					<span className="absolute -top-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] font-medium text-primary-foreground leading-none">
						{unreadCount > 9 ? "9+" : unreadCount}
					</span>
				)}
			</PopoverTrigger>
			<PopoverContent
				align="end"
				sideOffset={8}
				className="w-80 p-0 shadow-lg"
			>
				<div className="flex items-center justify-between px-4 py-3 border-b">
					<span className="text-sm font-semibold">Notifications</span>
					{unreadCount > 0 && (
						<button
							type="button"
							onClick={() => markAllRead()}
							className="text-xs text-muted-foreground hover:text-foreground transition-colors"
						>
							Mark all as read
						</button>
					)}
				</div>
				{notifications.length === 0 ? (
					<div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
						<Bell className="h-8 w-8 mb-2 opacity-30" />
						<p className="text-sm">No notifications yet</p>
					</div>
				) : (
					<ScrollArea className="max-h-96">
						<ul className="divide-y">
							{notifications.map((n) => (
								<li key={n.id}>
									<button
										type="button"
										onClick={() => handleNotificationClick(n)}
										className={`w-full text-left px-4 py-3 hover:bg-muted/50 transition-colors flex gap-3 ${!n.read_at ? "bg-primary/5" : ""}`}
									>
										{!n.read_at && (
											<span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary" />
										)}
										<div className={!n.read_at ? "" : "pl-5"}>
											<p className="text-sm leading-snug">
												{notificationText(n)}
											</p>
											<p className="mt-0.5 text-xs text-muted-foreground">
												{n.project_name} · {timeAgo(n.created_at)}
											</p>
										</div>
									</button>
								</li>
							))}
						</ul>
					</ScrollArea>
				)}
			</PopoverContent>
		</Popover>
	);
}
