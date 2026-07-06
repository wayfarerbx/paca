import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import type { TaskStatus } from "@/lib/project-api";
import {
	setWorkflowStatusTransition,
	type WorkflowStatusTransition,
	workflowQueryOptions,
} from "@/lib/workflow-api";

interface WorkflowStatusTransitionsPanelProps {
	projectId: string;
	workflowId: string;
	transitions: WorkflowStatusTransition[];
	statuses: TaskStatus[];
	canEdit: boolean;
}

const TERMINAL = "__terminal__";

export function WorkflowStatusTransitionsPanel({
	projectId,
	workflowId,
	transitions,
	statuses,
	canEdit,
}: WorkflowStatusTransitionsPanelProps) {
	const { t } = useTranslation("projects");
	const qc = useQueryClient();

	const graphKey = workflowQueryOptions(projectId, workflowId).queryKey;
	const invalidate = () => qc.invalidateQueries({ queryKey: graphKey });

	const setTransitionMutation = useMutation({
		mutationFn: (input: { statusId: string; nextStatusId: string | null }) =>
			setWorkflowStatusTransition(projectId, workflowId, {
				status_id: input.statusId,
				next_status_id: input.nextStatusId,
			}),
		onSuccess: invalidate,
	});

	const orderedStatuses = useMemo(
		() => [...statuses].sort((a, b) => a.position - b.position),
		[statuses],
	);
	const transitionByStatus = useMemo(
		() => new Map(transitions.map((tr) => [tr.status_id, tr])),
		[transitions],
	);

	const terminalCount = transitions.filter((tr) => !tr.next_status_id).length;
	const isAmbiguous = transitions.length > 0 && terminalCount !== 1;

	return (
		<div className="space-y-3">
			<div>
				<p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-1">
					{t("automation.statusTransitions.title")}
				</p>
				<p className="text-xs text-muted-foreground/70">
					{t("automation.statusTransitions.hint")}
				</p>
			</div>

			{isAmbiguous && (
				<div className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
					<AlertTriangle className="size-3.5 shrink-0 mt-0.5" />
					<span>
						{t("automation.statusTransitions.ambiguousWarning", {
							count: terminalCount,
						})}
					</span>
				</div>
			)}

			<div className="space-y-2">
				{orderedStatuses.map((status) => {
					const transition = transitionByStatus.get(status.id);
					return (
						<div
							key={status.id}
							className="flex items-center gap-2 rounded-lg border border-border/30 bg-muted/20 px-3 py-2"
						>
							<span
								className="text-xs font-medium px-1.5 py-0.5 rounded-md shrink-0"
								style={{
									backgroundColor: `${status.color ?? "#6366f1"}1a`,
									color: status.color ?? "#6366f1",
								}}
							>
								{status.name}
							</span>
							<span className="text-xs text-muted-foreground shrink-0">→</span>
							<Select
								value={transition?.next_status_id ?? TERMINAL}
								disabled={!canEdit}
								onValueChange={(value) =>
									setTransitionMutation.mutate({
										statusId: status.id,
										nextStatusId: value === TERMINAL ? null : value,
									})
								}
								items={[
									{
										value: TERMINAL,
										label: t("automation.statusTransitions.terminalOption"),
									},
									...statuses
										.filter((s) => s.id !== status.id)
										.map((s) => ({ value: s.id, label: s.name })),
								]}
							>
								<SelectTrigger className="flex-1 h-8">
									<SelectValue
										placeholder={t(
											"automation.statusTransitions.nextStatusLabel",
										)}
									/>
								</SelectTrigger>
								<SelectContent>
									<SelectItem value={TERMINAL}>
										{t("automation.statusTransitions.terminalOption")}
									</SelectItem>
									{statuses
										.filter((s) => s.id !== status.id)
										.map((s) => (
											<SelectItem key={s.id} value={s.id}>
												{s.name}
											</SelectItem>
										))}
								</SelectContent>
							</Select>
						</div>
					);
				})}
			</div>
		</div>
	);
}
