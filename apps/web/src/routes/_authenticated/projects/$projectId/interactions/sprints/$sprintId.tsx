import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { AlertCircle, CheckCircle2, X } from "lucide-react";
import { useState } from "react";

import { InteractionLayout } from "@/components/projects/interactions/interaction-layout";
import { usePermissions } from "@/hooks/use-permissions";
import {
	sprintQueryOptions,
	sprintsQueryOptions,
	sprintTasksQueryOptions,
	updateSprint,
	updateTask,
} from "@/lib/interaction-api";
import { taskStatusesQueryOptions } from "@/lib/project-api";
import { cn } from "@/lib/utils";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/interactions/sprints/$sprintId",
)({
	loader: async ({
		context: { queryClient },
		params: { projectId, sprintId },
	}) => {
		await queryClient
			.ensureQueryData(sprintQueryOptions(projectId, sprintId))
			.catch(() => {
				throw redirect({
					to: "/projects/$projectId/interactions/backlog",
					params: { projectId },
				})
			})
	},
	component: SprintPage,
});

function SprintPage() {
	const { projectId, sprintId } = Route.useParams();
	const { hasPermission } = usePermissions();
	const qc = useQueryClient();
	const navigate = useNavigate();

	const { data: sprint, isError } = useQuery(
		sprintQueryOptions(projectId, sprintId),
	)
	const { data: allSprints = [] } = useQuery(sprintsQueryOptions(projectId));
	const { data: tasksResult } = useQuery(
		sprintTasksQueryOptions(projectId, sprintId),
	)
	const { data: taskStatuses = [] } = useQuery(
		taskStatusesQueryOptions(projectId),
	)

	const canCreate = hasPermission("tasks.write");
	const canEdit = hasPermission("tasks.write");
	const canManageViews = hasPermission("projects.write");
	const canManageSprints = hasPermission("sprints.write");

	const [completeOpen, setCompleteOpen] = useState(false);
	const [moveToSprintId, setMoveToSprintId] = useState<string | null>(null);

	const sprintTasks = tasksResult?.items ?? [];

	const doneStatusIds = new Set(
		taskStatuses
			.filter((s) => s.category === "done")
			.map((s) => s.id),
	)
	const incompleteTasks = sprintTasks.filter(
		(t) => !t.status_id || !doneStatusIds.has(t.status_id),
	)

	const otherSprints = allSprints.filter(
		(s) => s.id !== sprintId && s.status !== "completed",
	)

	const completeSprintMutation = useMutation({
		mutationFn: async () => {
			// Move only tasks that are NOT in a "done" status category
			if (incompleteTasks.length > 0) {
				await Promise.all(
					incompleteTasks.map((t) =>
						updateTask(projectId, t.id, {
							sprint_id: moveToSprintId ?? null,
						}),
					),
				)
			}
			// Mark sprint as completed
			return updateSprint(projectId, sprintId, { status: "completed" });
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["projects", projectId, "sprints"] });
			qc.invalidateQueries({ queryKey: ["projects", projectId, "backlog-tasks"] });
			qc.invalidateQueries({ queryKey: ["projects", projectId, "all-tasks"] });
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "sprints", sprintId, "tasks"],
			})
			setCompleteOpen(false);
			navigate({
				to: "/projects/$projectId/interactions/backlog",
				params: { projectId },
			})
		},
	})

	if (isError || !sprint) {
		return (
			<div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-muted-foreground">
				<AlertCircle className="size-8 opacity-40" />
				<p className="text-sm">Sprint not found or access denied.</p>
			</div>
		)
	}

	const statusBadge =
		sprint.status === "active"
			? "Active"
			: sprint.status === "planned"
				? "Planned"
				: "Completed";

	return (
        <>
            <InteractionLayout
				projectId={projectId}
				interactionKey={`sprint:${sprintId}`}
				title={sprint.name}
				description={
					sprint.goal
						? sprint.goal
						: `${statusBadge} sprint${sprint.start_date ? ` · started ${new Date(sprint.start_date).toLocaleDateString()}` : ""}`
				}
				canCreate={canCreate}
				canEdit={canEdit}
				canManageViews={canManageViews}
				sprintId={sprintId}
				context="sprint"
				headerActions={
					sprint.status === "active" && canManageSprints ? (
						<button
							type="button"
							onClick={() => setCompleteOpen(true)}
							className="flex items-center gap-1.5 rounded-lg bg-primary/10 px-3 py-1.5 text-[12px] font-semibold text-primary hover:bg-primary/20 transition-all duration-150"
						>
							<CheckCircle2 className="size-3.5 shrink-0" />
							Complete sprint
						</button>
					) : undefined
				}
			/>
            {/* Complete Sprint Modal */}
            {completeOpen && (
				// biome-ignore lint/a11y/noStaticElementInteractions: modal backdrop
				(<div
					className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
					onClick={(e) => {
						if (e.target === e.currentTarget) setCompleteOpen(false);
					}}
				>
                    {/* biome-ignore lint/a11y/noStaticElementInteractions: modal panel */}
                    <div
						className="relative w-full max-w-md rounded-xl border border-border/50 bg-background p-6 shadow-2xl mx-4"
						onClick={(e) => e.stopPropagation()}
					>
						<button
							type="button"
							onClick={() => setCompleteOpen(false)}
							className="absolute right-4 top-4 flex size-7 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 transition-all"
						>
							<X className="size-4" />
						</button>
						<h2 className="font-[Syne] text-[18px] font-bold tracking-tight mb-1">
							Complete sprint
						</h2>
						<p className="text-[13px] text-muted-foreground mb-5">
							{incompleteTasks.length > 0
								? `${incompleteTasks.length} incomplete task${incompleteTasks.length === 1 ? "" : "s"} will be moved to:`
								: "No incomplete tasks remain in this sprint."}
						</p>

						{incompleteTasks.length > 0 && (
							<div className="flex flex-col gap-2 mb-5">
								{/* Backlog option */}
								<label
									className={cn(
										"flex items-center gap-3 rounded-lg border p-3 cursor-pointer transition-all",
										moveToSprintId === null
											? "border-primary/50 bg-primary/5"
											: "border-border/40 hover:bg-muted/30",
									)}
								>
									<input
										type="radio"
										name="move-to-sprint"
										value=""
										checked={moveToSprintId === null}
										onChange={() => setMoveToSprintId(null)}
										className="accent-primary"
									/>
									<div>
										<p className="text-sm font-semibold">Product Backlog</p>
										<p className="text-[11px] text-muted-foreground">
											Tasks will be unassigned from any sprint
										</p>
									</div>
								</label>

								{/* Other sprints */}
								{otherSprints.map((s) => (
									<label
										key={s.id}
										className={cn(
											"flex items-center gap-3 rounded-lg border p-3 cursor-pointer transition-all",
											moveToSprintId === s.id
												? "border-primary/50 bg-primary/5"
												: "border-border/40 hover:bg-muted/30",
										)}
									>
										<input
											type="radio"
											name="move-to-sprint"
											value={s.id}
											checked={moveToSprintId === s.id}
											onChange={() => setMoveToSprintId(s.id)}
											className="accent-primary"
										/>
										<div>
											<p className="text-sm font-semibold">{s.name}</p>
											<p className="text-[11px] text-muted-foreground capitalize">
												{s.status}
											</p>
										</div>
									</label>
								))}
							</div>
						)}

						<div className="flex justify-end gap-2">
							<button
								type="button"
								onClick={() => setCompleteOpen(false)}
								className="rounded-lg border border-border/50 bg-muted/20 px-4 py-2 text-sm font-medium hover:bg-muted/40 transition-all"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={() => completeSprintMutation.mutate()}
								disabled={completeSprintMutation.isPending}
								className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-all"
							>
								{completeSprintMutation.isPending
									? "Completing…"
									: "Complete sprint"}
							</button>
						</div>
					</div>
                </div>)
			)}
        </>
    )
}
