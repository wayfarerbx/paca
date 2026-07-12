import { useQuery } from "@tanstack/react-query";
import { ExternalLink, Workflow as WorkflowIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { workflowsForTaskQueryOptions } from "@/lib/workflow-api";

interface WorkflowsSectionProps {
	projectId: string;
	taskId: string;
	onNavigateToWorkflow?: (workflowId: string) => void;
}

const STATUS_BADGE_CLASSES: Record<string, string> = {
	draft: "bg-muted/50 text-muted-foreground",
	active: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
	archived: "bg-muted/30 text-muted-foreground/60",
};

export function WorkflowsSection({
	projectId,
	taskId,
	onNavigateToWorkflow,
}: WorkflowsSectionProps) {
	const { t } = useTranslation("projects");
	const { data: workflows = [] } = useQuery(
		workflowsForTaskQueryOptions(projectId, taskId),
	);

	return (
		<div className="space-y-3">
			<h3 className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
				<span>{t("taskDetail.workflows.title")}</span>
				<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
			</h3>

			{workflows.length > 0 ? (
				<div className="rounded-xl border border-border/25 bg-card/50 divide-y divide-border/15 overflow-hidden">
					{workflows.map((wf) => (
						<div
							key={wf.id}
							className="flex items-center gap-3 px-4 py-2.5 group"
						>
							<WorkflowIcon className="size-3 text-muted-foreground/40 shrink-0" />
							{/* biome-ignore lint/a11y/useSemanticElements: span for styling and truncation */}
							<span
								role="button"
								tabIndex={onNavigateToWorkflow ? 0 : -1}
								className={`flex-1 text-sm text-foreground truncate ${
									onNavigateToWorkflow
										? "cursor-pointer hover:text-primary transition-colors duration-100"
										: ""
								}`}
								onClick={() => onNavigateToWorkflow?.(wf.id)}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										e.preventDefault();
										onNavigateToWorkflow?.(wf.id);
									}
								}}
							>
								{wf.name}
							</span>
							<span
								className={`shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded-md ${
									STATUS_BADGE_CLASSES[wf.status] ??
									"bg-muted/50 text-muted-foreground"
								}`}
							>
								{t(`automation.status.${wf.status}`)}
							</span>
							{onNavigateToWorkflow && (
								<button
									type="button"
									onClick={() => onNavigateToWorkflow(wf.id)}
									className="size-6 rounded flex items-center justify-center text-muted-foreground/50 hover:text-foreground hover:bg-muted/30 transition-all duration-100 opacity-0 group-hover:opacity-100"
									title={t("taskDetail.workflows.open")}
								>
									<ExternalLink className="size-3" />
								</button>
							)}
						</div>
					))}
				</div>
			) : (
				<div className="flex items-center gap-3 px-1 py-3 text-muted-foreground/45">
					<WorkflowIcon className="size-4 opacity-70" />
					<p className="text-sm italic">{t("taskDetail.workflows.empty")}</p>
				</div>
			)}
		</div>
	);
}
