import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	Check,
	ChevronDown,
	ChevronRight,
	ClipboardCopy,
	GitBranch,
	Loader2,
	Terminal,
} from "lucide-react";
import { useState } from "react";
import { ApiErrorCode, getApiErrorCode } from "@/lib/api-error";
import {
	createBranch,
	linkedRepositoriesQueryOptions,
	taskBranchesQueryOptions,
	type TaskBranch,
} from "@/lib/github-api";
import { cn } from "@/lib/utils";

// ── Copy button ───────────────────────────────────────────────────────────────

function CopyButton({ text }: { text: string }) {
	const [copied, setCopied] = useState(false);
	function copy() {
		navigator.clipboard?.writeText(text)?.catch(() => {});
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	}
	return (
		<button
			type="button"
			aria-label="Copy to clipboard"
			onClick={copy}
			className="shrink-0 text-muted-foreground/60 hover:text-muted-foreground transition-colors"
		>
			{copied ? (
				<Check className="size-3.5 text-emerald-500" />
			) : (
				<ClipboardCopy className="size-3.5" />
			)}
		</button>
	);
}

// ── Command block ─────────────────────────────────────────────────────────────

function CommandBlock({ command }: { command: string }) {
	return (
		<div className="flex items-center gap-2 rounded-md bg-muted/60 border border-border/50 px-3 py-2 mt-1.5">
			<Terminal className="size-3.5 shrink-0 text-muted-foreground/50" />
			<code className="flex-1 text-[11px] font-mono text-foreground/80 break-all">
				{command}
			</code>
			<CopyButton text={command} />
		</div>
	);
}

// ── Existing branch row ───────────────────────────────────────────────────────

function BranchRow({ branch }: { branch: TaskBranch }) {
	const cloneCmd = `git fetch origin && git checkout ${branch.branch_name}`;
	return (
		<div className="rounded-lg border border-border/50 bg-card px-3 py-2.5 space-y-1.5">
			<div className="flex items-center gap-2">
				<GitBranch className="size-3.5 shrink-0 text-muted-foreground/60" />
				<span className="text-[12px] font-mono truncate text-foreground/90 flex-1">
					{branch.branch_name}
				</span>
			</div>
			<CommandBlock command={cloneCmd} />
		</div>
	);
}

// ── Branch creation form ──────────────────────────────────────────────────────

const BRANCH_TYPES = [
	"feat",
	"fix",
	"chore",
	"docs",
	"test",
	"refactor",
] as const;

function CreateBranchForm({
	projectId,
	taskId,
	taskIdPrefix,
	taskNumber,
	taskTitle,
	repos,
	onDone,
}: {
	projectId: string;
	taskId: string;
	taskIdPrefix: string;
	taskNumber: number;
	taskTitle?: string;
	repos: { id: string; full_name: string }[];
	onDone: () => void;
}) {
	const queryClient = useQueryClient();

	const taskRef = taskIdPrefix
		? `${taskIdPrefix.toUpperCase()}-${taskNumber}`
		: `${taskNumber}`;

	// Build default slug from title: lowercase, replace non-word chars with `-`
	const defaultSlug = taskTitle
		? `-${taskTitle
				.toLowerCase()
				.replace(/[^a-z0-9]+/g, "-")
				.replace(/^-|-$/g, "")
				.slice(0, 30)}`
		: "";

	const [type, setType] = useState<(typeof BRANCH_TYPES)[number]>("feat");
	const [branchName, setBranchName] = useState(
		`${type}/${taskRef}${defaultSlug}`,
	);
	const [selectedRepoId, setSelectedRepoId] = useState(
		repos.length === 1 ? repos[0].id : "",
	);
	const [sourceBranch, setSourceBranch] = useState("");
	const [error, setError] = useState<string | null>(null);

	// When type changes, keep the rest of the branch name as-is (user may have edited it)
	function handleTypeChange(newType: (typeof BRANCH_TYPES)[number]) {
		setType(newType);
		// Replace only the leading type segment
		setBranchName((prev) => {
			const slash = prev.indexOf("/");
			const rest = slash >= 0 ? prev.slice(slash) : `/${taskRef}`;
			return `${newType}${rest}`;
		});
	}

	// ── Option A: create branch via API ──────────────────────────────────────

	const [createdBranchName, setCreatedBranchName] = useState<string | null>(
		null,
	);

	const createMutation = useMutation({
		mutationFn: () =>
			createBranch(
				projectId,
				taskId,
				selectedRepoId,
				branchName,
				sourceBranch || undefined,
			),
		onSuccess: (result) => {
			setCreatedBranchName(result.branch_name);
			queryClient.invalidateQueries({
				queryKey: taskBranchesQueryOptions(projectId, taskId).queryKey,
			});
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
			if (code === ApiErrorCode.GitHubBranchAlreadyLinked) {
				setError("This branch is already linked to the task.");
				return;
			}
			setError("Failed to create branch. Please try again.");
		},
	});

	// ── Option B: local command ───────────────────────────────────────────────

	const localCmd = `git checkout -b ${branchName} && git push -u origin ${branchName}`;

	function validateForm(): boolean {
		if (!branchName.trim()) {
			setError("Branch name is required.");
			return false;
		}
		if (repos.length > 1 && !selectedRepoId) {
			setError("Select a repository.");
			return false;
		}
		return true;
	}

	function handleCreate() {
		setError(null);
		if (!validateForm()) return;
		createMutation.mutate();
	}

	// After creation — show checkout command + done
	if (createdBranchName) {
		const checkoutCmd = `git fetch origin && git checkout ${createdBranchName}`;
		return (
			<div className="space-y-2">
				<p className="text-xs text-emerald-600 dark:text-emerald-400 font-medium flex items-center gap-1.5">
					<Check className="size-3.5" />
					Branch <code className="font-mono">{createdBranchName}</code> created on
					GitHub.
				</p>
				<p className="text-[11px] text-muted-foreground">
					Check it out locally:
				</p>
				<CommandBlock command={checkoutCmd} />
				<button
					type="button"
					className="text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors mt-1"
					onClick={onDone}
				>
					Done
				</button>
			</div>
		);
	}

	return (
		<div className="space-y-3 rounded-lg border border-border/50 bg-card px-3 py-3">
			{/* Branch type pills */}
			<div>
				<p className="text-[11px] text-muted-foreground mb-1.5">Type</p>
				<div className="flex flex-wrap gap-1.5">
					{BRANCH_TYPES.map((t) => (
						<button
							key={t}
							type="button"
							onClick={() => handleTypeChange(t)}
							className={cn(
								"rounded-full px-2.5 py-0.5 text-[11px] font-medium border transition-colors",
								t === type
									? "border-primary/60 bg-primary/10 text-primary"
									: "border-border/50 text-muted-foreground hover:border-border",
							)}
						>
							{t}
						</button>
					))}
				</div>
			</div>

			{/* Branch name */}
			<div>
				<p className="text-[11px] text-muted-foreground mb-1">Branch name</p>
				<input
					type="text"
					value={branchName}
					onChange={(e) => {
						setBranchName(e.target.value);
						setError(null);
					}}
					placeholder={`feat/${taskRef}`}
					className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-xs font-mono focus:outline-none focus:ring-1 focus:ring-ring"
					spellCheck={false}
				/>
			</div>

			{/* Repo selector — only when multiple repos */}
			{repos.length > 1 && (
				<div>
					<p className="text-[11px] text-muted-foreground mb-1">Repository</p>
					<select
						value={selectedRepoId}
						onChange={(e) => setSelectedRepoId(e.target.value)}
						className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
					>
						<option value="">Select repository…</option>
						{repos.map((r) => (
							<option key={r.id} value={r.id}>
								{r.full_name}
							</option>
						))}
					</select>
				</div>
			)}

			{/* Source branch (optional) */}
			<div>
				<p className="text-[11px] text-muted-foreground mb-1">
					Source branch{" "}
					<span className="opacity-60">(optional, defaults to repo default)</span>
				</p>
				<input
					type="text"
					value={sourceBranch}
					onChange={(e) => setSourceBranch(e.target.value)}
					placeholder="main"
					className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-xs font-mono focus:outline-none focus:ring-1 focus:ring-ring"
					spellCheck={false}
				/>
			</div>

			{error && (
				<p className="text-[11px] text-destructive/80">{error}</p>
			)}

			{/* Action buttons */}
			<div className="flex flex-col gap-2 pt-0.5">
				{/* Option A */}
				<button
					type="button"
					disabled={createMutation.isPending}
					onClick={handleCreate}
					className="flex items-center justify-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-60 transition-colors"
				>
					{createMutation.isPending ? (
						<Loader2 className="size-3.5 animate-spin" />
					) : (
						<GitBranch className="size-3.5" />
					)}
					Create branch on GitHub
				</button>

				{/* Option B: show local command immediately */}
				<div>
					<p className="text-[11px] text-muted-foreground mb-0.5">
						Or create locally:
					</p>
					<CommandBlock command={localCmd} />
				</div>
			</div>

			<button
				type="button"
				className="text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors"
				onClick={onDone}
			>
				Cancel
			</button>
		</div>
	);
}

// ── Public section component ──────────────────────────────────────────────────

export function BranchesSection({
	projectId,
	taskId,
	taskIdPrefix,
	taskNumber,
	taskTitle,
	canEdit = true,
}: {
	projectId: string;
	taskId: string;
	taskIdPrefix: string;
	taskNumber: number;
	taskTitle?: string;
	canEdit?: boolean;
}) {
	const [expanded, setExpanded] = useState(true);
	const [creating, setCreating] = useState(false);

	const { data: branches = [], isLoading } = useQuery(
		taskBranchesQueryOptions(projectId, taskId),
	);

	const { data: linkedRepos = [] } = useQuery(
		linkedRepositoriesQueryOptions(projectId),
	);

	const count = branches.length;
	const canCreate = canEdit && linkedRepos.length > 0;

	return (
		<div>
			{/* Section header */}
			<button
				type="button"
				className="flex w-full items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-3 hover:text-muted-foreground transition-colors"
				onClick={() => setExpanded((v) => !v)}
			>
				<GitBranch className="size-3.5 shrink-0" />
				<span>Branches</span>
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
							{branches.map((branch) => (
								<BranchRow key={branch.id} branch={branch} />
							))}

							{count === 0 && !creating && (
								<p className="text-xs text-muted-foreground/50 italic py-1">
									No branches linked yet.
								</p>
							)}

							{creating ? (
								<CreateBranchForm
									projectId={projectId}
									taskId={taskId}
									taskIdPrefix={taskIdPrefix}
									taskNumber={taskNumber}
									taskTitle={taskTitle}
									repos={linkedRepos}
									onDone={() => setCreating(false)}
								/>
							) : (
								canCreate && (
									<button
										type="button"
										className="flex items-center gap-1.5 text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors py-1"
										onClick={() => setCreating(true)}
									>
										<GitBranch className="size-3.5" />
										Create branch
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
