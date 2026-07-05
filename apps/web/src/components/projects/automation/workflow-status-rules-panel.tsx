import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import type { ProjectMember, TaskStatus } from "@/lib/project-api";
import {
	removeWorkflowStatusRule,
	setWorkflowStatusRule,
	type WorkflowStatusRule,
	workflowQueryOptions,
} from "@/lib/workflow-api";

interface WorkflowStatusRulesPanelProps {
	projectId: string;
	workflowId: string;
	rules: WorkflowStatusRule[];
	statuses: TaskStatus[];
	members: ProjectMember[];
	canEdit: boolean;
}

export function WorkflowStatusRulesPanel({
	projectId,
	workflowId,
	rules,
	statuses,
	members,
	canEdit,
}: WorkflowStatusRulesPanelProps) {
	const { t } = useTranslation("projects");
	const qc = useQueryClient();
	const [newRuleStatusId, setNewRuleStatusId] = useState("");
	const [newRuleMemberId, setNewRuleMemberId] = useState("");

	const graphKey = workflowQueryOptions(projectId, workflowId).queryKey;
	const invalidate = () => qc.invalidateQueries({ queryKey: graphKey });

	const addRuleMutation = useMutation({
		mutationFn: () =>
			setWorkflowStatusRule(projectId, workflowId, {
				status_id: newRuleStatusId,
				assignee_member_id: newRuleMemberId,
			}),
		onSuccess: () => {
			setNewRuleStatusId("");
			setNewRuleMemberId("");
			invalidate();
		},
	});

	const updateAssigneeMutation = useMutation({
		mutationFn: ({
			statusId,
			assigneeMemberId,
		}: {
			statusId: string;
			assigneeMemberId: string;
		}) =>
			setWorkflowStatusRule(projectId, workflowId, {
				status_id: statusId,
				assignee_member_id: assigneeMemberId,
			}),
		onSuccess: invalidate,
	});

	const removeRuleMutation = useMutation({
		mutationFn: (ruleId: string) =>
			removeWorkflowStatusRule(projectId, workflowId, ruleId),
		onSuccess: invalidate,
	});

	const usedStatusIds = new Set(rules.map((r) => r.status_id));
	const availableStatuses = statuses.filter((s) => !usedStatusIds.has(s.id));

	return (
		<div className="space-y-3">
			<div>
				<p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-1">
					{t("automation.statusRules.title")}
				</p>
				<p className="text-xs text-muted-foreground/70">
					{t("automation.statusRules.hint")}
				</p>
			</div>

			<div className="space-y-2">
				{rules.map((rule) => {
					const status = statuses.find((s) => s.id === rule.status_id);
					return (
						<div
							key={rule.id}
							className="flex items-center gap-2 rounded-lg border border-border/30 bg-muted/20 px-3 py-2"
						>
							<span
								className="text-xs font-medium px-1.5 py-0.5 rounded-md shrink-0"
								style={{
									backgroundColor: `${status?.color ?? "#6366f1"}1a`,
									color: status?.color ?? "#6366f1",
								}}
							>
								{status?.name ?? rule.status_id}
							</span>
							<span className="text-xs text-muted-foreground shrink-0">→</span>
							<Select
								value={rule.assignee_member_id}
								disabled={!canEdit}
								onValueChange={(value) => {
									if (!value) return;
									updateAssigneeMutation.mutate({
										statusId: rule.status_id,
										assigneeMemberId: value,
									});
								}}
								items={members.map((m) => ({
									value: m.id,
									label: m.full_name || m.username,
								}))}
							>
								<SelectTrigger className="flex-1 h-8">
									<SelectValue
										placeholder={t("automation.statusRules.memberPlaceholder")}
									/>
								</SelectTrigger>
								<SelectContent>
									{members.map((m) => (
										<SelectItem key={m.id} value={m.id}>
											{m.full_name || m.username}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							{canEdit && (
								<button
									type="button"
									onClick={() => removeRuleMutation.mutate(rule.id)}
									className="shrink-0 text-muted-foreground/50 hover:text-destructive transition-colors"
								>
									<Trash2 className="size-3.5" />
								</button>
							)}
						</div>
					);
				})}
				{rules.length === 0 && (
					<p className="text-xs text-muted-foreground/50 italic py-2">
						{t("automation.statusRules.noRules")}
					</p>
				)}
			</div>

			{canEdit && availableStatuses.length > 0 && (
				<div className="flex items-center gap-2">
					<Select
						value={newRuleStatusId}
						onValueChange={(value) => setNewRuleStatusId(value ?? "")}
						items={availableStatuses.map((s) => ({
							value: s.id,
							label: s.name,
						}))}
					>
						<SelectTrigger className="flex-1">
							<SelectValue
								placeholder={t("automation.statusRules.statusPlaceholder")}
							/>
						</SelectTrigger>
						<SelectContent>
							{availableStatuses.map((s) => (
								<SelectItem key={s.id} value={s.id}>
									{s.name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
					<Select
						value={newRuleMemberId}
						onValueChange={(value) => setNewRuleMemberId(value ?? "")}
						items={members.map((m) => ({
							value: m.id,
							label: m.full_name || m.username,
						}))}
					>
						<SelectTrigger className="flex-1">
							<SelectValue
								placeholder={t("automation.statusRules.memberPlaceholder")}
							/>
						</SelectTrigger>
						<SelectContent>
							{members.map((m) => (
								<SelectItem key={m.id} value={m.id}>
									{m.full_name || m.username}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
					<Button
						size="icon"
						variant="secondary"
						disabled={
							!newRuleStatusId || !newRuleMemberId || addRuleMutation.isPending
						}
						onClick={() => addRuleMutation.mutate()}
					>
						{addRuleMutation.isPending ? (
							<Loader2 className="size-4 animate-spin" />
						) : (
							<Plus className="size-4" />
						)}
					</Button>
				</div>
			)}
		</div>
	);
}
