import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	ChevronDown,
	ChevronRight,
	ExternalLink,
	GitMerge,
	GitPullRequest,
	GitPullRequestClosed,
	Loader2,
	Plus,
	Trash2,
} from "lucide-react";
import { useState } from "react";
import { ApiErrorCode, getApiErrorCode } from "@/lib/api-error";
import {
	type LinkedRepository,
	linkedRepositoriesQueryOptions,
	linkPRToTask,
	type PullRequest,
	taskPRsQueryOptions,
	unlinkPRFromTask,
} from "@/lib/github-api";
import { cn } from "@/lib/utils";

// ── PR state badge ────────────────────────────────────────────────────────────

function PRStateBadge({ state }: { state: PullRequest["state"] }) {
	if (state === "merged") {
		return (
			<span className="inline-flex items-center gap-1 rounded-full bg-violet-500/15 px-2 py-0.5 text-[10px] font-semibold text-violet-500">
				<GitMerge className="size-3" />
				Merged
			</span>
		);
	}
	if (state === "closed") {
		return (
			<span className="inline-flex items-center gap-1 rounded-full bg-destructive/15 px-2 py-0.5 text-[10px] font-semibold text-destructive/80">
				<GitPullRequestClosed className="size-3" />
				Closed
			</span>
		);
	}
	return (
		<span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-500">
			<GitPullRequest className="size-3" />
			Open
		</span>
	);
}

// ── Single PR row ─────────────────────────────────────────────────────────────

function PRRow({
	pr,
	projectId,
	taskId,
	canEdit,
}: {
	pr: PullRequest;
	projectId: string;
	taskId: string;
	canEdit: boolean;
}) {
	const queryClient = useQueryClient();

	const unlinkMutation = useMutation({
		mutationFn: () => unlinkPRFromTask(projectId, taskId, pr.id),
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: taskPRsQueryOptions(projectId, taskId).queryKey,
			});
		},
	});

	return (
		<div className="group flex items-start gap-2.5 rounded-lg border border-border/50 bg-card px-3 py-2.5 hover:border-border/80 transition-colors">
			<div className="mt-0.5 shrink-0">
				<PRStateBadge state={pr.state} />
			</div>
			<div className="min-w-0 flex-1">
				<a
					href={pr.html_url}
					target="_blank"
					rel="noopener noreferrer"
					className="flex items-center gap-1.5 text-sm font-medium hover:text-primary transition-colors"
				>
					<span className="truncate">{pr.title}</span>
					<ExternalLink className="size-3 shrink-0 opacity-50" />
				</a>
				<div className="flex items-center gap-2 mt-1 flex-wrap">
					<span className="text-[11px] text-muted-foreground font-mono">
						#{pr.pr_number}
					</span>
					<span className="text-muted-foreground/40">·</span>
					<span className="text-[11px] text-muted-foreground">
						{pr.head_branch}
					</span>
					{pr.author && (
						<>
							<span className="text-muted-foreground/40">·</span>
							<span className="text-[11px] text-muted-foreground">
								by {pr.author}
							</span>
						</>
					)}
				</div>
			</div>
			{canEdit && (
				<button
					type="button"
					aria-label="Unlink pull request"
					className="shrink-0 mt-0.5 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground/60 hover:text-destructive"
					onClick={() => unlinkMutation.mutate()}
					disabled={unlinkMutation.isPending}
				>
					{unlinkMutation.isPending ? (
						<Loader2 className="size-3.5 animate-spin" />
					) : (
						<Trash2 className="size-3.5" />
					)}
				</button>
			)}
		</div>
	);
}

// ── Helpers ───────────────────────────────────────────────────────────────────

/**
 * Parses a GitHub PR URL such as
 * `https://github.com/owner/repo/pull/42`
 * and returns `{ fullName: "owner/repo", prNumber: 42 }` or `null`.
 */
function parseGitHubPRUrl(
	raw: string,
): { fullName: string; prNumber: number } | null {
	try {
		const url = new URL(raw.trim());
		if (url.hostname !== "github.com") return null;
		const parts = url.pathname.replace(/^\//, "").split("/");
		if (parts.length < 4 || parts[2] !== "pull") return null;
		const prNumber = Number(parts[3]);
		if (!Number.isInteger(prNumber) || prNumber <= 0) return null;
		return { fullName: `${parts[0]}/${parts[1]}`, prNumber };
	} catch {
		return null;
	}
}

// ── Link PR form ──────────────────────────────────────────────────────────────

function LinkPRForm({
	projectId,
	taskId,
	repos,
	onDone,
}: {
	projectId: string;
	taskId: string;
	repos: LinkedRepository[];
	onDone: () => void;
}) {
	const queryClient = useQueryClient();
	const [selectedRepoId, setSelectedRepoId] = useState(
		repos.length === 1 ? repos[0].id : "",
	);
	const [value, setValue] = useState("");
	const [error, setError] = useState<string | null>(null);

	// Detect when the user types a full GitHub PR URL.
	const parsed = parseGitHubPRUrl(value);
	const urlMatchedRepo = parsed
		? (repos.find((r) => r.full_name === parsed.fullName) ?? null)
		: null;
	const effectiveRepoId = parsed ? (urlMatchedRepo?.id ?? "") : selectedRepoId;

	const mutation = useMutation({
		mutationFn: () => {
			const prNum = parsed ? parsed.prNumber : Number(value);
			return linkPRToTask(projectId, taskId, effectiveRepoId, prNum);
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: taskPRsQueryOptions(projectId, taskId).queryKey,
			});
			setValue("");
			setError(null);
			onDone();
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.GitHubIntegrationNotFound) {
				setError("No GitHub token configured for this project.");
				return;
			}
			if (code === ApiErrorCode.GitHubRepositoryNotFound) {
				setError("Repository not found. It may have been unlinked.");
				return;
			}
			if (code === ApiErrorCode.GitHubPRNotFound) {
				const displayNum = parsed ? parsed.prNumber : value;
				setError(`PR #${displayNum} was not found in the selected repository.`);
				return;
			}
			if (code === ApiErrorCode.GitHubPRAlreadyLinked) {
				const displayNum = parsed ? parsed.prNumber : value;
				setError(`PR #${displayNum} is already linked to this task.`);
				return;
			}
			if (code === ApiErrorCode.GitHubTokenInsufficientPermissions) {
				setError(
					"Your GitHub token does not have permission to read pull requests. Update it in Project Settings > GitHub.",
				);
				return;
			}
			if (code === ApiErrorCode.BadRequest) {
				const msg = (err as { response?: { data?: { error?: string } } })
					?.response?.data?.error;
				setError(msg || "Failed to link pull request. Please try again.");
				return;
			}
			setError("Failed to link pull request. Please try again.");
		},
	});

	function submit() {
		if (parsed) {
			if (!urlMatchedRepo) {
				setError(
					`Repository "${parsed.fullName}" is not linked to this project.`,
				);
				return;
			}
		} else {
			if (!effectiveRepoId) {
				setError("Select a repository.");
				return;
			}
			const num = Number(value);
			if (!value.trim() || !Number.isInteger(num) || num <= 0) {
				setError("Enter a valid PR number or paste a GitHub PR URL.");
				return;
			}
		}
		mutation.mutate();
	}

	return (
		<div className="space-y-3 rounded-lg border border-border/50 bg-card px-3 py-3">
			{/* Repository selector */}
			<div>
				<p className="text-[11px] text-muted-foreground mb-1">Repository</p>
				<select
					value={parsed ? (urlMatchedRepo?.id ?? "") : selectedRepoId}
					onChange={(e) => {
						setSelectedRepoId(e.target.value);
						setError(null);
					}}
					disabled={mutation.isPending || !!parsed}
					className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
				>
					<option value="">Select repository…</option>
					{repos.map((r) => (
						<option key={r.id} value={r.id}>
							{r.full_name}
						</option>
					))}
				</select>
			</div>

			{/* PR number or URL */}
			<div>
				<p className="text-[11px] text-muted-foreground mb-1">
					PR number or GitHub URL
				</p>
				<input
					type="text"
					value={value}
					onChange={(e) => {
						setValue(e.target.value);
						setError(null);
					}}
					onKeyDown={(e) => {
						if (e.key === "Enter") submit();
						if (e.key === "Escape") onDone();
					}}
					placeholder="42 or https://github.com/owner/repo/pull/42"
					className={cn(
						"w-full rounded-md border bg-background px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring",
						error ? "border-destructive" : "border-border/60",
					)}
					// biome-ignore lint/a11y/noAutofocus: intentional for inline form
					autoFocus
					disabled={mutation.isPending}
				/>
				{parsed && urlMatchedRepo && (
					<p className="mt-1 text-[11px] text-muted-foreground">
						Will link PR #{parsed.prNumber} from{" "}
						<span className="font-medium">{urlMatchedRepo.full_name}</span>
					</p>
				)}
			</div>

			{error && (
				<p className="text-[11px] text-destructive/80 leading-relaxed">
					{error}
				</p>
			)}

			{/* Action buttons */}
			<div className="flex items-center gap-2 pt-0.5">
				<button
					type="button"
					onClick={submit}
					disabled={!value.trim() || mutation.isPending}
					className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-60 transition-colors"
				>
					{mutation.isPending ? (
						<Loader2 className="size-3.5 animate-spin" />
					) : (
						<GitPullRequest className="size-3.5" />
					)}
					Link pull request
				</button>
				<button
					type="button"
					onClick={onDone}
					className="text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors"
				>
					Cancel
				</button>
			</div>
		</div>
	);
}

// ── Main export ───────────────────────────────────────────────────────────────

export function PullRequestsSection({
	projectId,
	taskId,
	canEdit = true,
}: {
	projectId: string;
	taskId: string;
	canEdit?: boolean;
}) {
	const { data: prs = [], isLoading } = useQuery(
		taskPRsQueryOptions(projectId, taskId),
	);
	const { data: linkedRepos = [] } = useQuery({
		...linkedRepositoriesQueryOptions(projectId),
		throwOnError: false,
	});
	const [expanded, setExpanded] = useState(true);
	const [linking, setLinking] = useState(false);

	const count = prs.length;
	const canLinkPR = canEdit && linkedRepos.length > 0;

	return (
		<div>
			{/* Section header */}
			<button
				type="button"
				className="flex w-full items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-3 hover:text-muted-foreground transition-colors"
				onClick={() => setExpanded((v) => !v)}
			>
				<GitPullRequest className="size-3.5 shrink-0" />
				<span>Pull Requests</span>
				{count > 0 && (
					<span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-bold text-muted-foreground normal-case tracking-normal">
						{count}
					</span>
				)}
				<div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
				{expanded ? (
					<ChevronDown className="size-3.5 shrink-0" />
				) : (
					<ChevronRight className="size-3.5 shrink-0" />
				)}
			</button>

			{expanded && (
				<div className="space-y-2">
					{isLoading ? (
						<div className="flex items-center gap-2 py-2 text-muted-foreground/60 text-xs">
							<Loader2 className="size-3.5 animate-spin" />
							Loading…
						</div>
					) : (
						<>
							{prs.map((pr) => (
								<PRRow
									key={pr.id}
									pr={pr}
									projectId={projectId}
									taskId={taskId}
									canEdit={canEdit}
								/>
							))}

							{count === 0 && !linking && (
								<p className="text-xs text-muted-foreground/50 italic py-1">
									No pull requests linked yet.
								</p>
							)}

							{linking ? (
								<LinkPRForm
									projectId={projectId}
									taskId={taskId}
									repos={linkedRepos}
									onDone={() => setLinking(false)}
								/>
							) : (
								canLinkPR && (
									<button
										type="button"
										className="flex items-center gap-1.5 text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors py-1"
										onClick={() => setLinking(true)}
									>
										<Plus className="size-3.5" />
										Link pull request
									</button>
								)
							)}
						</>
					)}
				</div>
			)}
		</div>
	);
}
