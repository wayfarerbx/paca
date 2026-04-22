import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	AlertCircle,
	BookOpen,
	Check,
	GitBranch,
	GitPullRequest,
	KeyRound,
	Loader2,
	Plus,
	RefreshCw,
	Search,
	Trash2,
	Unlink,
	X,
} from "lucide-react";
import { useState } from "react";
import { GitHubIcon } from "@/components/icons/github-icon";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiErrorCode, getApiErrorCode } from "@/lib/api-error";
import {
	type AccessibleRepo,
	accessibleReposQueryOptions,
	deleteGitHubToken,
	githubIntegrationQueryOptions,
	type LinkedRepository,
	linkedRepositoriesQueryOptions,
	linkRepository,
	setGitHubToken,
	unlinkRepository,
} from "@/lib/github-api";
import { cn } from "@/lib/utils";

// ── Token card ────────────────────────────────────────────────────────────────

function TokenCard({
	projectId,
	hasIntegration,
	onTokenSet,
	canEdit,
}: {
	projectId: string;
	hasIntegration: boolean;
	onTokenSet: () => void;
	canEdit: boolean;
}) {
	const queryClient = useQueryClient();
	const [token, setToken] = useState("");
	const [showToken, setShowToken] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [confirmOpen, setConfirmOpen] = useState(false);

	const saveMutation = useMutation({
		mutationFn: () => setGitHubToken(projectId, token.trim()),
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: githubIntegrationQueryOptions(projectId).queryKey,
			});
			setToken("");
			setError(null);
			onTokenSet();
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.GitHubInvalidToken) {
				setError(
					"GitHub rejected the token. Check it has the required scopes (repo, admin:repo_hook).",
				);
				return;
			}
			setError("Failed to save token. Please try again.");
		},
	});

	const deleteMutation = useMutation({
		mutationFn: () => deleteGitHubToken(projectId),
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: githubIntegrationQueryOptions(projectId).queryKey,
			});
			await queryClient.removeQueries({
				queryKey: linkedRepositoriesQueryOptions(projectId).queryKey,
			});
			await queryClient.removeQueries({
				queryKey: accessibleReposQueryOptions(projectId).queryKey,
			});
			setConfirmOpen(false);
		},
		onError: () => {
			setError("Failed to remove token. Please try again.");
		},
	});

	if (hasIntegration) {
		return (
			<>
				<div className="flex items-center justify-between gap-4">
					<div className="flex items-center gap-3">
						<div className="flex size-8 items-center justify-center rounded-full bg-emerald-500/15">
							<Check className="size-4 text-emerald-500" />
						</div>
						<div>
							<p className="text-sm font-medium">Personal access token saved</p>
							<p className="text-xs text-muted-foreground mt-0.5">
								Token is stored encrypted. It is never returned by the API.
							</p>
						</div>
					</div>
					{canEdit && (
						<Button
							variant="outline"
							size="sm"
							className="shrink-0 gap-1.5 text-destructive/80 hover:text-destructive border-destructive/30 hover:border-destructive/50"
							onClick={() => setConfirmOpen(true)}
							disabled={deleteMutation.isPending}
						>
							{deleteMutation.isPending ? (
								<Loader2 className="size-3.5 animate-spin" />
							) : (
								<Trash2 className="size-3.5" />
							)}
							Remove token
						</Button>
					)}
				</div>

				<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
					<DialogContent className="sm:max-w-sm">
						<DialogHeader>
							<div className="flex size-10 items-center justify-center rounded-full bg-destructive/10 mb-2">
								<KeyRound className="size-5 text-destructive" />
							</div>
							<DialogTitle>Remove GitHub token</DialogTitle>
							<DialogDescription>
								Removing the token will also unlink all repositories and disable
								webhook events. This action cannot be undone.
							</DialogDescription>
						</DialogHeader>
						{deleteMutation.isError ? (
							<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
								Failed to remove token. Please try again.
							</p>
						) : null}
						<DialogFooter>
							<DialogClose
								render={
									<Button
										variant="outline"
										size="sm"
										disabled={deleteMutation.isPending}
									/>
								}
							>
								Cancel
							</DialogClose>
							<Button
								variant="destructive"
								size="sm"
								disabled={deleteMutation.isPending}
								onClick={() => deleteMutation.mutate()}
							>
								{deleteMutation.isPending ? (
									<>
										<Loader2 className="size-3.5 mr-1.5 animate-spin" />
										Removing…
									</>
								) : (
									"Remove token"
								)}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>
			</>
		);
	}

	return (
		<div className="space-y-4">
			<p className="text-sm text-muted-foreground">
				Add a GitHub personal access token with{" "}
				<code className="rounded bg-muted px-1 py-0.5 text-[11px] font-mono">
					repo
				</code>{" "}
				and{" "}
				<code className="rounded bg-muted px-1 py-0.5 text-[11px] font-mono">
					admin:repo_hook
				</code>{" "}
				scopes to link repositories, create branches, and track pull requests.
			</p>
			<div className="flex gap-2 max-w-lg">
				<div className="flex-1 relative">
					<Input
						type={showToken ? "text" : "password"}
						value={token}
						onChange={(e) => {
							setToken(e.target.value);
							setError(null);
						}}
						placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
						disabled={!canEdit || saveMutation.isPending}
						className={cn(
							"pr-8 font-mono text-sm",
							error
								? "border-destructive focus-visible:ring-destructive/30"
								: "",
						)}
						autoComplete="off"
						onKeyDown={(e) => {
							if (e.key === "Enter" && token.trim()) saveMutation.mutate();
						}}
					/>
					<button
						type="button"
						className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground/60 hover:text-muted-foreground text-[10px] font-medium"
						onClick={() => setShowToken((v) => !v)}
						tabIndex={-1}
					>
						{showToken ? "hide" : "show"}
					</button>
				</div>
				<Button
					size="sm"
					disabled={!token.trim() || !canEdit || saveMutation.isPending}
					onClick={() => saveMutation.mutate()}
					className="shrink-0"
				>
					{saveMutation.isPending ? (
						<Loader2 className="size-3.5 animate-spin mr-1.5" />
					) : null}
					Save token
				</Button>
			</div>
			{error ? <p className="text-xs text-destructive">{error}</p> : null}
		</div>
	);
}

// ── Add repository dialog ─────────────────────────────────────────────────────

function AddRepoDialog({
	projectId,
	open,
	onOpenChange,
}: {
	projectId: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	const queryClient = useQueryClient();
	const {
		data: repos = [],
		isLoading,
		isFetching,
	} = useQuery({ ...accessibleReposQueryOptions(projectId), enabled: open });
	const [search, setSearch] = useState("");
	const [error, setError] = useState<string | null>(null);

	const linkMutation = useMutation({
		mutationFn: (repo: AccessibleRepo) =>
			linkRepository(projectId, repo.owner, repo.repo_name),
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: linkedRepositoriesQueryOptions(projectId).queryKey,
			});
			setError(null);
			onOpenChange(false);
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.GitHubWebhookURLNotPublic) {
				setError(
					"Cannot register webhook because this API URL is not publicly reachable (for example localhost). Configure PUBLIC_URL to a public HTTPS URL and try again.",
				);
				return;
			}
			if (code === ApiErrorCode.GitHubWebhookCreationFailed) {
				setError(
					"Could not create the webhook. Make sure your token has the admin:repo_hook scope and you have admin access to the repository.",
				);
				return;
			}
			if (code === ApiErrorCode.GitHubRepoAlreadyLinked) {
				setError("This repository is already linked to the project.");
				return;
			}
			if (code === ApiErrorCode.GitHubRepoNotAccessible) {
				setError(
					"Repository not found or not accessible. Check that your token has the repo scope.",
				);
				return;
			}
			setError("Failed to link repository. Please try again.");
		},
	});

	const filtered = repos.filter((r) =>
		r.full_name.toLowerCase().includes(search.toLowerCase()),
	);

	function handleOpenChange(next: boolean) {
		if (!next) {
			setSearch("");
			setError(null);
		}
		onOpenChange(next);
	}

	return (
		<Dialog open={open} onOpenChange={handleOpenChange}>
			<DialogContent className="sm:max-w-lg">
				<DialogHeader>
					<div className="flex items-center gap-2.5 mb-1">
						<div className="flex size-9 items-center justify-center rounded-full bg-primary/10 shrink-0">
							<GitBranch className="size-4 text-primary" />
						</div>
						<DialogTitle>Add repository</DialogTitle>
					</div>
					<DialogDescription>
						Select a repository from your GitHub account to link to this
						project. A webhook will be registered automatically.
					</DialogDescription>
				</DialogHeader>

				{/* Search + reload */}
				<div className="flex items-center gap-2">
					<div className="relative flex-1">
						<Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground/60" />
						<Input
							placeholder="Search repositories…"
							value={search}
							onChange={(e) => setSearch(e.target.value)}
							className="pl-8 text-sm"
							autoFocus
						/>
					</div>
					<button
						type="button"
						aria-label="Reload repositories"
						disabled={isFetching}
						className="flex size-9 shrink-0 items-center justify-center rounded-md border border-input bg-background text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
						onClick={() =>
							queryClient.invalidateQueries({
								queryKey: accessibleReposQueryOptions(projectId).queryKey,
							})
						}
					>
						{isFetching ? (
							<Loader2 className="size-3.5 animate-spin" />
						) : (
							<RefreshCw className="size-3.5" />
						)}
					</button>
				</div>

				{error ? (
					<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
						{error}
					</p>
				) : null}

				{/* Repo list */}
				<div className="min-h-50 max-h-80 overflow-y-auto space-y-1 pr-0.5 [scrollbar-gutter:stable]">
					{isLoading ? (
						[...Array(4)].map((_, i) => (
							// biome-ignore lint/suspicious/noArrayIndexKey: static skeleton list
							<Skeleton key={i} className="h-14 rounded-lg mb-1" />
						))
					) : repos.length === 0 ? (
						<div className="flex flex-col items-center gap-2 py-10 text-muted-foreground/60">
							<BookOpen className="size-8" />
							<p className="text-sm">No accessible repositories found.</p>
						</div>
					) : filtered.length === 0 ? (
						<p className="text-center text-sm text-muted-foreground/60 py-10">
							No repositories match &ldquo;{search}&rdquo;
						</p>
					) : (
						filtered.map((repo) => (
							<button
								key={repo.full_name}
								type="button"
								disabled={linkMutation.isPending}
								onClick={() => linkMutation.mutate(repo)}
								className="w-full flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-card px-3.5 py-3 text-left hover:border-border hover:bg-muted/40 transition-colors disabled:opacity-60"
							>
								<div className="min-w-0">
									<div className="flex items-center gap-2">
										<GitBranch className="size-3.5 text-muted-foreground/70 shrink-0" />
										<span className="text-sm font-medium truncate">
											{repo.full_name}
										</span>
										{repo.private && (
											<span className="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold bg-muted text-muted-foreground">
												Private
											</span>
										)}
									</div>
									{repo.description ? (
										<p className="text-xs text-muted-foreground mt-0.5 truncate pl-5">
											{repo.description}
										</p>
									) : null}
								</div>
								{linkMutation.isPending &&
								linkMutation.variables?.full_name === repo.full_name ? (
									<Loader2 className="size-3.5 animate-spin shrink-0 text-muted-foreground" />
								) : (
									<Plus className="size-3.5 shrink-0 text-muted-foreground/40 group-hover:text-muted-foreground" />
								)}
							</button>
						))
					)}
				</div>

				<DialogFooter>
					<DialogClose
						render={
							<Button
								variant="outline"
								size="sm"
								disabled={linkMutation.isPending}
							/>
						}
					>
						Close
					</DialogClose>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Linked repo item ──────────────────────────────────────────────────────────

function LinkedRepoItem({
	projectId,
	repo,
	canEdit,
}: {
	projectId: string;
	repo: LinkedRepository;
	canEdit: boolean;
}) {
	const queryClient = useQueryClient();
	const [confirmOpen, setConfirmOpen] = useState(false);

	const unlinkMutation = useMutation({
		mutationFn: () => unlinkRepository(projectId, repo.id),
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: linkedRepositoriesQueryOptions(projectId).queryKey,
			});
			setConfirmOpen(false);
		},
	});

	return (
		<>
			<div className="flex items-center justify-between gap-4 rounded-lg border border-border/60 bg-muted/20 px-3.5 py-3">
				<div className="flex items-center gap-3 min-w-0">
					<div className="flex size-8 items-center justify-center rounded-full bg-primary/10 shrink-0">
						<GitBranch className="size-4 text-primary" />
					</div>
					<div className="min-w-0">
						<a
							href={`https://github.com/${repo.full_name}`}
							target="_blank"
							rel="noopener noreferrer"
							className="text-sm font-medium hover:underline truncate block"
						>
							{repo.full_name}
						</a>
						<p className="text-xs text-muted-foreground mt-0.5">
							Default branch:{" "}
							<code className="font-mono">{repo.default_branch}</code>
						</p>
					</div>
				</div>
				{canEdit && (
					<Button
						variant="ghost"
						size="sm"
						className="shrink-0 gap-1.5 text-muted-foreground hover:text-destructive"
						onClick={() => setConfirmOpen(true)}
					>
						<Unlink className="size-3.5" />
						Unlink
					</Button>
				)}
			</div>

			<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
				<DialogContent className="sm:max-w-sm">
					<DialogHeader>
						<div className="flex size-10 items-center justify-center rounded-full bg-destructive/10 mb-2">
							<Unlink className="size-5 text-destructive" />
						</div>
						<DialogTitle>Unlink repository</DialogTitle>
						<DialogDescription>
							This will remove the link to{" "}
							<span className="font-semibold text-foreground">
								{repo.full_name}
							</span>{" "}
							and attempt to delete the webhook from GitHub.
						</DialogDescription>
					</DialogHeader>
					{unlinkMutation.isError ? (
						<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
							Failed to unlink. Please try again.
						</p>
					) : null}
					<DialogFooter>
						<DialogClose
							render={
								<Button
									variant="outline"
									size="sm"
									disabled={unlinkMutation.isPending}
								/>
							}
						>
							Cancel
						</DialogClose>
						<Button
							variant="destructive"
							size="sm"
							disabled={unlinkMutation.isPending}
							onClick={() => unlinkMutation.mutate()}
						>
							{unlinkMutation.isPending ? (
								<>
									<Loader2 className="size-3.5 mr-1.5 animate-spin" />
									Unlinking…
								</>
							) : (
								"Unlink repository"
							)}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}

// ── Main export ───────────────────────────────────────────────────────────────

export function GitHubSettings({
	projectId,
	canEdit,
}: {
	projectId: string;
	canEdit: boolean;
}) {
	const { data: integration, isLoading: integrationLoading } = useQuery({
		...githubIntegrationQueryOptions(projectId),
		throwOnError: false,
	});
	const { data: linkedRepos = [], isLoading: reposLoading } = useQuery({
		...linkedRepositoriesQueryOptions(projectId),
		throwOnError: false,
		enabled: !!integration,
	});

	const queryClient = useQueryClient();
	const hasIntegration = !!integration;
	const hasRepos = linkedRepos.length > 0;

	const [addRepoOpen, setAddRepoOpen] = useState(false);

	const steps = [
		{
			num: 1,
			label: "Connect a GitHub token",
			done: hasIntegration,
		},
		{
			num: 2,
			label: "Link a repository",
			done: hasRepos,
		},
	];

	return (
		<div className="space-y-6">
			{/* Header card */}
			<div className="rounded-xl border border-border/60 bg-card p-6">
				<div className="flex items-center gap-3 mb-1">
					<GitHubIcon className="size-5 text-foreground/80" />
					<h3 className="font-[Syne] text-base font-semibold">
						GitHub Integration
					</h3>
				</div>
				<p className="text-sm text-muted-foreground mb-5">
					Link GitHub repositories to track pull requests, create branches from
					tasks, and receive webhook events automatically. You can link multiple
					repositories to a single project.
				</p>

				{/* Progress steps */}
				<div className="flex items-center gap-4 mb-6">
					{steps.map((step, idx) => (
						<div key={step.num} className="flex items-center gap-3">
							<div
								className={cn(
									"flex size-6 items-center justify-center rounded-full text-[11px] font-bold shrink-0 transition-colors",
									step.done
										? "bg-emerald-500 text-white"
										: "border-2 border-border text-muted-foreground/60",
								)}
							>
								{step.done ? <Check className="size-3.5" /> : step.num}
							</div>
							<span
								className={cn(
									"text-xs font-medium",
									step.done ? "text-foreground" : "text-muted-foreground/70",
								)}
							>
								{step.label}
							</span>
							{idx < steps.length - 1 && (
								<div className="h-px w-8 bg-border/60 shrink-0" />
							)}
						</div>
					))}
				</div>

				{/* Token section */}
				<div className="space-y-4">
					<div className="flex items-center gap-2">
						<KeyRound className="size-3.5 text-muted-foreground/70" />
						<Label className="text-[13px] font-semibold text-foreground/80">
							Personal Access Token
						</Label>
					</div>
					{integrationLoading ? (
						<Skeleton className="h-10 rounded-lg max-w-xs" />
					) : (
						<TokenCard
							projectId={projectId}
							hasIntegration={hasIntegration}
							onTokenSet={() => setAddRepoOpen(true)}
							canEdit={canEdit}
						/>
					)}
				</div>
			</div>

			{/* Repository section — only after token */}
			{hasIntegration && (
				<div className="rounded-xl border border-border/60 bg-card p-6">
					<div className="flex items-center justify-between mb-1">
						<div className="flex items-center gap-2">
							<GitPullRequest className="size-4 text-foreground/80" />
							<h3 className="font-[Syne] text-base font-semibold">
								Linked Repositories
							</h3>
							<button
								type="button"
								aria-label="Reload linked repositories"
								disabled={reposLoading}
								className="text-muted-foreground/40 hover:text-muted-foreground transition-colors disabled:opacity-30"
								onClick={() =>
									queryClient.invalidateQueries({
										queryKey:
											linkedRepositoriesQueryOptions(projectId).queryKey,
									})
								}
							>
								{reposLoading ? (
									<Loader2 className="size-3.5 animate-spin" />
								) : (
									<RefreshCw className="size-3.5" />
								)}
							</button>
						</div>
						{canEdit && (
							<Button
								variant="outline"
								size="sm"
								className="gap-1.5"
								onClick={() => setAddRepoOpen(true)}
							>
								<Plus className="size-3.5" />
								Add repository
							</Button>
						)}
					</div>
					<p className="text-sm text-muted-foreground mb-4">
						{hasRepos
							? "Webhooks are registered automatically for each linked repository."
							: "No repositories linked yet. Link a repository to track pull requests and branches."}
					</p>

					{reposLoading ? (
						<div className="space-y-2">
							<Skeleton className="h-14 rounded-lg" />
							<Skeleton className="h-14 rounded-lg" />
						</div>
					) : (
						<div className="space-y-2">
							{linkedRepos.map((repo) => (
								<LinkedRepoItem
									key={repo.id}
									projectId={projectId}
									repo={repo}
									canEdit={canEdit}
								/>
							))}

							{!hasRepos && (
								<div className="flex flex-col items-center gap-3 py-8 text-muted-foreground/60">
									<GitBranch className="size-8" />
									<p className="text-sm">No repositories linked yet.</p>
									{canEdit && (
										<Button
											variant="outline"
											size="sm"
											className="gap-1.5"
											onClick={() => setAddRepoOpen(true)}
										>
											<Plus className="size-3.5" />
											Add your first repository
										</Button>
									)}
								</div>
							)}
						</div>
					)}

					{canEdit && (
						<AddRepoDialog
							projectId={projectId}
							open={addRepoOpen}
							onOpenChange={setAddRepoOpen}
						/>
					)}
				</div>
			)}

			{/* Webhook hint */}
			{hasRepos && (
				<div className="flex items-start gap-2.5 rounded-lg bg-muted/40 border border-border/40 px-4 py-3">
					<AlertCircle className="size-4 text-muted-foreground/70 shrink-0 mt-0.5" />
					<p className="text-xs text-muted-foreground leading-relaxed">
						Webhooks are registered on all linked repositories. GitHub will push{" "}
						<code className="font-mono">pull_request</code> events to keep PR
						status in sync automatically.
					</p>
				</div>
			)}

			{/* No integration hint */}
			{!hasIntegration && !integrationLoading && (
				<div className="flex items-start gap-2.5 rounded-lg bg-muted/30 border border-dashed border-border/50 px-4 py-3">
					<X className="size-4 text-muted-foreground/50 shrink-0 mt-0.5" />
					<p className="text-xs text-muted-foreground">
						No GitHub integration configured. Add a personal access token to get
						started.
					</p>
				</div>
			)}
		</div>
	);
}
