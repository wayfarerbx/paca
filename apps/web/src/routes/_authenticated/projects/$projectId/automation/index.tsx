import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	Clock,
	Loader2,
	Plus,
	Trash2,
	Workflow as WorkflowIcon,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { useProjectPermissions } from "@/hooks/use-project-permissions";
import { projectQueryOptions } from "@/lib/project-api";
import { timeAgo } from "@/lib/time-ago";
import {
	createWorkflow,
	deleteWorkflow,
	type Workflow,
	workflowsQueryOptions,
} from "@/lib/workflow-api";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/automation/",
)({
	loader: async ({ context: { queryClient }, params: { projectId } }) => {
		await queryClient.ensureQueryData(workflowsQueryOptions(projectId));
	},
	component: AutomationListPage,
});

const STATUS_BADGE_VARIANT: Record<
	string,
	"default" | "secondary" | "outline"
> = {
	draft: "secondary",
	active: "default",
	archived: "outline",
};

function AutomationListPage() {
	const { t } = useTranslation("projects");
	const { t: tCommon } = useTranslation("common");
	const { projectId } = Route.useParams();
	const qc = useQueryClient();
	const navigate = useNavigate();
	const { hasProjectPermission } = useProjectPermissions(projectId);
	const canManage = hasProjectPermission("workflows.write");

	const { data: project } = useQuery(projectQueryOptions(projectId));
	const { data: workflows = [], isLoading } = useQuery(
		workflowsQueryOptions(projectId),
	);

	const [createOpen, setCreateOpen] = useState(false);
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [deleteTarget, setDeleteTarget] = useState<Workflow | null>(null);

	const listKey = workflowsQueryOptions(projectId).queryKey;

	const createMutation = useMutation({
		mutationFn: () =>
			createWorkflow(projectId, { name: name.trim(), description }),
		onSuccess: () => {
			setCreateOpen(false);
			setName("");
			setDescription("");
			qc.invalidateQueries({ queryKey: listKey });
		},
	});

	const deleteMutation = useMutation({
		mutationFn: (workflowId: string) => deleteWorkflow(projectId, workflowId),
		onSuccess: () => {
			setDeleteTarget(null);
			qc.invalidateQueries({ queryKey: listKey });
		},
	});

	return (
		<div className="flex flex-col">
			<div className="relative overflow-hidden border-b border-border/50">
				<div
					className="pointer-events-none absolute inset-0 opacity-50"
					style={{
						backgroundImage:
							"radial-gradient(circle, color-mix(in oklch, var(--color-primary) 12%, transparent) 1px, transparent 1px)",
						backgroundSize: "20px 20px",
						maskImage:
							"radial-gradient(ellipse 70% 100% at 0% 0%, black 20%, transparent 70%)",
					}}
				/>
				<div className="relative flex items-end justify-between px-6 py-8">
					<div>
						<h1 className="font-[Syne] text-2xl font-bold tracking-tight">
							{t("automation.list.title")}
						</h1>
						<p className="mt-1 text-sm text-muted-foreground">
							{project?.name} · {t("automation.list.subtitle")}
						</p>
					</div>
					{canManage ? (
						<Button
							size="sm"
							className="gap-1.5 shadow-sm shadow-primary/20"
							onClick={() => setCreateOpen(true)}
						>
							<Plus className="size-3.5" />
							{t("automation.list.newWorkflow")}
						</Button>
					) : null}
				</div>
			</div>

			<div className="p-6">
				{isLoading ? (
					<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
						{Array.from({ length: 3 }).map((_, i) => (
							// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
							<Skeleton key={i} className="h-40 rounded-xl" />
						))}
					</div>
				) : workflows.length === 0 ? (
					<div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
						<div className="flex size-16 items-center justify-center rounded-2xl bg-muted/50">
							<WorkflowIcon className="size-8 text-muted-foreground/50" />
						</div>
						<div>
							<p className="font-medium text-sm">
								{t("automation.list.empty.title")}
							</p>
							<p className="text-xs text-muted-foreground mt-1 max-w-xs">
								{t("automation.list.empty.description")}
							</p>
						</div>
						{canManage && (
							<Button size="sm" onClick={() => setCreateOpen(true)}>
								<Plus className="size-4 mr-1.5" />
								{t("automation.list.empty.createFirstWorkflow")}
							</Button>
						)}
					</div>
				) : (
					<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
						{workflows.map((wf) => (
							// biome-ignore lint/a11y/noStaticElementInteractions: card navigates to the workflow builder; the delete button below stays keyboard-reachable on its own
							// biome-ignore lint/a11y/useKeyWithClickEvents: click-to-navigate card, consistent with the agent list cards
							<div
								key={wf.id}
								onClick={() =>
									navigate({
										to: "/projects/$projectId/automation/$workflowId",
										params: { projectId, workflowId: wf.id },
									})
								}
								className="group relative flex flex-col gap-3 rounded-xl border border-border/60 bg-card p-5 transition-all hover:border-border hover:shadow-sm cursor-pointer"
							>
								<div className="flex items-start justify-between gap-3">
									<div className="flex items-center gap-3 min-w-0">
										<div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
											<WorkflowIcon className="size-4 text-primary" />
										</div>
										<div className="min-w-0">
											<p className="font-semibold text-sm leading-tight truncate">
												{wf.name}
											</p>
											<p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
												{wf.description || t("automation.list.noDescription")}
											</p>
										</div>
									</div>

									<div className="flex items-center gap-1.5 shrink-0">
										<Badge
											variant={STATUS_BADGE_VARIANT[wf.status] ?? "outline"}
										>
											{t(`automation.status.${wf.status}`)}
										</Badge>
										{canManage && (
											<button
												type="button"
												onClick={(e) => {
													e.stopPropagation();
													setDeleteTarget(wf);
												}}
												className="inline-flex size-7 items-center justify-center rounded-md text-muted-foreground/60 opacity-0 transition-all hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100"
											>
												<Trash2 className="size-3.5" />
											</button>
										)}
									</div>
								</div>

								{/* Meta row */}
								<div className="flex items-center gap-1.5 text-xs text-muted-foreground">
									<Clock className="size-3" />
									{t("automation.list.updated", {
										time: timeAgo(wf.updated_at, tCommon),
									})}
								</div>
							</div>
						))}
					</div>
				)}
			</div>

			<Dialog open={createOpen} onOpenChange={setCreateOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>{t("automation.createDialog.title")}</DialogTitle>
					</DialogHeader>
					<div className="space-y-4">
						<div className="space-y-1.5">
							<Label htmlFor="workflow-name">
								{t("automation.createDialog.nameLabel")}
							</Label>
							<Input
								id="workflow-name"
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder={t("automation.createDialog.namePlaceholder")}
							/>
						</div>
						<div className="space-y-1.5">
							<Label htmlFor="workflow-description">
								{t("automation.createDialog.descriptionLabel")}
							</Label>
							<Textarea
								id="workflow-description"
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								rows={3}
								placeholder={t(
									"automation.createDialog.descriptionPlaceholder",
								)}
							/>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setCreateOpen(false)}>
							{t("automation.createDialog.cancel")}
						</Button>
						<Button
							disabled={!name.trim() || createMutation.isPending}
							onClick={() => createMutation.mutate()}
						>
							{createMutation.isPending && (
								<Loader2 className="size-4 mr-2 animate-spin" />
							)}
							{t("automation.createDialog.create")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog
				open={!!deleteTarget}
				onOpenChange={(open) => !open && setDeleteTarget(null)}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>{t("automation.deleteDialog.title")}</DialogTitle>
					</DialogHeader>
					<p className="text-sm text-muted-foreground">
						{t("automation.deleteDialog.body", { name: deleteTarget?.name })}
					</p>
					<DialogFooter>
						<Button variant="outline" onClick={() => setDeleteTarget(null)}>
							{t("automation.deleteDialog.cancel")}
						</Button>
						<Button
							variant="destructive"
							disabled={deleteMutation.isPending}
							onClick={() =>
								deleteTarget && deleteMutation.mutate(deleteTarget.id)
							}
						>
							{deleteMutation.isPending && (
								<Loader2 className="size-4 mr-2 animate-spin" />
							)}
							{t("automation.deleteDialog.confirm")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
