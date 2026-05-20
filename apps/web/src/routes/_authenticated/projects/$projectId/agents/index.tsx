import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
	Bot,
	ChevronRight,
	GitBranch,
	Loader2,
	MessageSquare,
	MoreHorizontal,
	Plus,
	Settings,
	Sparkles,
	Trash2,
	Zap,
} from "lucide-react";
import { useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { useProjectPermissions } from "@/hooks/use-project-permissions";
import {
	type Agent,
	AGENT_PRESETS,
	agentsQueryOptions,
	createAgent,
	deleteAgent,
	llmModelsQueryOptions,
} from "@/lib/agent-api";
import { projectRolesQueryOptions } from "@/lib/project-api";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/agents/",
)({
	loader: async ({ context: { queryClient }, params: { projectId } }) => {
		await Promise.all([
			queryClient.ensureQueryData(agentsQueryOptions(projectId)),
			queryClient.ensureQueryData(projectRolesQueryOptions(projectId)),
		]);
	},
	component: AgentsPage,
});

// ── Create Agent Dialog ───────────────────────────────────────────────────────

function CreateAgentDialog({
	projectId,
	open,
	onOpenChange,
}: {
	projectId: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	const qc = useQueryClient();
	const { data: roles = [] } = useQuery(projectRolesQueryOptions(projectId));
	const { data: llmModels = {} } = useQuery(llmModelsQueryOptions);

	const [name, setName] = useState("");
	const [handle, setHandle] = useState("");
	const [presetId, setPresetId] = useState("");
	const [roleId, setRoleId] = useState("");
	const [llmProvider, setLlmProvider] = useState("anthropic");
	const [llmModel, setLlmModel] = useState("claude-sonnet-4-5-20250929");
	const [llmApiKey, setLlmApiKey] = useState("");
	const [llmBaseUrl, setLlmBaseUrl] = useState("");
	const [systemPrompt, setSystemPrompt] = useState("");

	const onPresetChange = (id: string | null) => {
		if (!id) return;
		setPresetId(id);
		const preset = AGENT_PRESETS.find((p) => p.id === id);
		if (preset) {
			if (preset.defaultLLMProvider) setLlmProvider(preset.defaultLLMProvider);
			if (preset.defaultLLMModel) setLlmModel(preset.defaultLLMModel);
			if (preset.defaultSystemPrompt) setSystemPrompt(preset.defaultSystemPrompt);
		}
	};

	const providers = Object.keys(llmModels);
	const availableModels: string[] = llmModels[llmProvider] ?? [];

	const createMutation = useMutation({
		mutationFn: () =>
			createAgent(projectId, {
				name: name.trim(),
				handle: handle.trim(),
				llm_provider: llmProvider,
				llm_model: llmModel,
				llm_api_key: llmApiKey,
				llm_base_url: llmBaseUrl || null,
				system_prompt: systemPrompt,
				project_role_id: roleId,
			}),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents"],
			});
			onOpenChange(false);
			setName("");
			setHandle("");
			setPresetId("");
			setRoleId("");
			setLlmApiKey("");
			setLlmBaseUrl("");
			setSystemPrompt("");
		},
	});

	// Auto-derive handle from name
	const onNameChange = (v: string) => {
		setName(v);
		setHandle(
			v
				.toLowerCase()
				.replace(/[^a-z0-9]+/g, "-")
				.replace(/^-+|-+$/g, ""),
		);
	};

	const canSubmit =
		name.trim() &&
		handle.trim() &&
		roleId &&
		llmApiKey.trim() &&
		!createMutation.isPending;

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Bot className="size-5 text-primary" />
						Create AI Agent
					</DialogTitle>
					<DialogDescription>
						Add an AI agent to your project. The agent will join as a project
						member.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4 py-2">
					{/* Name */}
					<div className="space-y-1.5">
						<Label>Name</Label>
						<Input
							placeholder="Dev Bot"
							value={name}
							onChange={(e) => onNameChange(e.target.value)}
						/>
					</div>

					{/* Handle */}
					<div className="space-y-1.5">
						<Label>Handle</Label>
						<div className="flex items-center gap-1.5">
							<span className="text-muted-foreground text-sm">@</span>
							<Input
								placeholder="dev-bot"
								value={handle}
								onChange={(e) => setHandle(e.target.value)}
							/>
						</div>
					</div>

					{/* Preset (optional) */}
					<div className="space-y-1.5">
						<Label>
							Preset{" "}
							<span className="text-muted-foreground font-normal">
								(optional)
							</span>
						</Label>
						<Select value={presetId} onValueChange={onPresetChange}>
							<SelectTrigger>
								<SelectValue placeholder="Start from a preset…" />
							</SelectTrigger>
							<SelectContent>
								{AGENT_PRESETS.map((p) => (
									<SelectItem key={p.id} value={p.id}>
										{p.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						{presetId && (
							<p className="text-xs text-muted-foreground">
								{AGENT_PRESETS.find((p) => p.id === presetId)?.description}
							</p>
						)}
					</div>

					{/* Project Role */}
					<div className="space-y-1.5">
						<Label>Project Role</Label>
						<Select value={roleId} onValueChange={(v) => v && setRoleId(v)}>
							<SelectTrigger>
								<SelectValue placeholder="Select role…" />
							</SelectTrigger>
							<SelectContent>
								{roles.map((r) => (
									<SelectItem key={r.id} value={r.id}>
										{r.role_name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* LLM Provider */}
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<Label>LLM Provider</Label>
							<Select
								value={llmProvider}
								onValueChange={(v) => {								if (!v) return;									setLlmProvider(v);
									setLlmModel(llmModels[v]?.[0] ?? "");
								}}
							>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									{providers.map((p) => (
										<SelectItem key={p} value={p}>
											{p}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<Label>Model</Label>
							<Select value={llmModel} onValueChange={(v) => v && setLlmModel(v)}>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									{availableModels.map((m) => (
										<SelectItem key={m} value={m}>
											{m}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>

					{/* API Key */}
					<div className="space-y-1.5">
						<Label>API Key</Label>
						<Input
							type="password"
							placeholder="sk-ant-…"
							value={llmApiKey}
							onChange={(e) => setLlmApiKey(e.target.value)}
						/>
						<p className="text-xs text-muted-foreground">
							Stored encrypted. Never exposed in responses.
						</p>
					</div>

					{/* Base URL (optional) */}
					<div className="space-y-1.5">
						<Label>
							Base URL{" "}
							<span className="text-muted-foreground font-normal">
								(optional)
							</span>
						</Label>
						<Input
							placeholder="https://api.openai.com/v1"
							value={llmBaseUrl}
							onChange={(e) => setLlmBaseUrl(e.target.value)}
						/>
					</div>

					{/* System Prompt */}
					<div className="space-y-1.5">
						<Label>
							System Prompt{" "}
							<span className="text-muted-foreground font-normal">
								(optional)
							</span>
						</Label>
						<Textarea
							placeholder="You are a senior software engineer…"
							value={systemPrompt}
							onChange={(e) => setSystemPrompt(e.target.value)}
							rows={3}
						/>
					</div>

					{createMutation.isError && (
						<p className="text-sm text-destructive">
							Failed to create agent. Please try again.
						</p>
					)}
				</div>

				<DialogFooter>
					<Button
						variant="outline"
						onClick={() => onOpenChange(false)}
						disabled={createMutation.isPending}
					>
						Cancel
					</Button>
					<Button
						onClick={() => createMutation.mutate()}
						disabled={!canSubmit}
					>
						{createMutation.isPending ? (
							<>
								<Loader2 className="size-4 mr-2 animate-spin" />
								Creating…
							</>
						) : (
							<>
								<Bot className="size-4 mr-2" />
								Create Agent
							</>
						)}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Agent Card ────────────────────────────────────────────────────────────────

function AgentCard({
	agent,
	projectId,
	canWrite,
}: {
	agent: Agent;
	projectId: string;
	canWrite: boolean;
}) {
	const qc = useQueryClient();
	const [confirmDelete, setConfirmDelete] = useState(false);

	const deleteMutation = useMutation({
		mutationFn: () => deleteAgent(projectId, agent.id),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["projects", projectId, "agents"] });
			setConfirmDelete(false);
		},
	});

	const initials = agent.name
		.split(" ")
		.map((w) => w[0])
		.join("")
		.toUpperCase()
		.slice(0, 2);

	return (
		<>
			<div className="group relative flex flex-col gap-3 rounded-xl border border-border/60 bg-card p-5 transition-all hover:border-border hover:shadow-sm">
				{/* Header */}
				<div className="flex items-start justify-between gap-3">
					<div className="flex items-center gap-3">
						<Avatar className="size-10 rounded-lg bg-primary/10">
							<AvatarFallback className="rounded-lg bg-primary/10 text-primary font-semibold text-sm">
								{initials}
							</AvatarFallback>
						</Avatar>
						<div className="min-w-0">
							<p className="font-semibold text-sm leading-tight">{agent.name}</p>
							<p className="text-xs text-muted-foreground mt-0.5">
								@{agent.handle}
							</p>
						</div>
					</div>

					<div className="flex items-center gap-1.5 shrink-0">
						<Badge variant="secondary" className="text-[10px] font-medium">
						{agent.llm_provider}
						</Badge>
						{canWrite && (
							<DropdownMenu>
							<DropdownMenuTrigger className="inline-flex size-7 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity hover:bg-accent hover:text-foreground group-hover:opacity-100">
								<MoreHorizontal className="size-4" />
								</DropdownMenuTrigger>
								<DropdownMenuContent align="end" className="w-40">
									<DropdownMenuItem
										render={
											<Link
												to="/projects/$projectId/agents/$agentId"
												params={{ projectId, agentId: agent.id }}
											/>
										}
									>
										<Settings className="size-3.5 mr-2" />
										Configure
									</DropdownMenuItem>
									<DropdownMenuSeparator />
									<DropdownMenuItem
										className="text-destructive focus:text-destructive"
										onClick={() => setConfirmDelete(true)}
									>
										<Trash2 className="size-3.5 mr-2" />
										Delete
									</DropdownMenuItem>
								</DropdownMenuContent>
							</DropdownMenu>
						)}
					</div>
				</div>

				{/* Stats row */}
				<div className="flex items-center gap-4 text-xs text-muted-foreground">
					<span className="flex items-center gap-1">
						<Zap className="size-3" />
						{agent.llm_provider}/{agent.llm_model}
					</span>
					{agent.can_clone_repos && (
						<span className="flex items-center gap-1">
							<GitBranch className="size-3" />
							Repos
						</span>
					)}
				</div>

				{/* Footer link */}
				<Link
					to="/projects/$projectId/agents/$agentId"
					params={{ projectId, agentId: agent.id }}
					className="mt-1 flex items-center gap-1 text-xs font-medium text-primary hover:underline"
				>
					Configure & view activity
					<ChevronRight className="size-3" />
				</Link>
			</div>

			{/* Delete confirmation */}
			<Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
				<DialogContent className="max-w-sm">
					<DialogHeader>
						<DialogTitle>Delete {agent.name}?</DialogTitle>
						<DialogDescription>
							This permanently deletes the agent and removes it from the project.
							Running conversations will be stopped.
						</DialogDescription>
					</DialogHeader>
					<DialogFooter>
						<Button
							variant="outline"
							onClick={() => setConfirmDelete(false)}
							disabled={deleteMutation.isPending}
						>
							Cancel
						</Button>
						<Button
							variant="destructive"
							onClick={() => deleteMutation.mutate()}
							disabled={deleteMutation.isPending}
						>
							{deleteMutation.isPending ? (
								<Loader2 className="size-4 animate-spin" />
							) : (
								"Delete"
							)}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}

// ── Page ──────────────────────────────────────────────────────────────────────

function AgentsPage() {
	const { projectId } = Route.useParams();
	const { hasProjectPermission } = useProjectPermissions(projectId);
	const canWrite = hasProjectPermission("agents.write");

	const { data: agents = [], isLoading } = useQuery(
		agentsQueryOptions(projectId),
	);
	const [createOpen, setCreateOpen] = useState(false);

	return (
		<div className="flex flex-col flex-1 min-h-0">
			{/* Header */}
			<div className="relative overflow-hidden border-b border-border/50 shrink-0">
				<div
					className="pointer-events-none absolute inset-0 opacity-40"
					style={{
						backgroundImage:
							"radial-gradient(circle, color-mix(in oklch, var(--color-primary) 10%, transparent) 1px, transparent 1px)",
						backgroundSize: "20px 20px",
					}}
				/>
				<div className="relative flex items-center justify-between px-6 py-5">
					<div className="flex items-center gap-3">
						<div className="flex size-9 items-center justify-center rounded-lg bg-primary/10">
							<Sparkles className="size-4 text-primary" />
						</div>
						<div>
							<h1 className="text-lg font-semibold">AI Agents</h1>
							<p className="text-xs text-muted-foreground mt-0.5">
								Autonomous agents that work on tasks and chat with your team
							</p>
						</div>
					</div>
					{canWrite && (
						<Button size="sm" onClick={() => setCreateOpen(true)}>
							<Plus className="size-4 mr-1.5" />
							New Agent
						</Button>
					)}
				</div>
			</div>

			{/* Content */}
			<div className="flex-1 overflow-auto p-6">
				{isLoading ? (
					<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
						{Array.from({ length: 3 }).map((_, i) => (
							// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
							<Skeleton key={i} className="h-36 rounded-xl" />
						))}
					</div>
				) : agents.length === 0 ? (
					<div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
						<div className="flex size-16 items-center justify-center rounded-2xl bg-muted/50">
							<Bot className="size-8 text-muted-foreground/50" />
						</div>
						<div>
							<p className="font-medium text-sm">No agents yet</p>
							<p className="text-xs text-muted-foreground mt-1 max-w-xs">
								Add an AI agent to automate tasks, review code, write
								documentation, and more.
							</p>
						</div>
						{canWrite && (
							<Button size="sm" onClick={() => setCreateOpen(true)}>
								<Plus className="size-4 mr-1.5" />
								Create your first agent
							</Button>
						)}
					</div>
				) : (
					<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
						{agents.map((agent) => (
							<AgentCard
								key={agent.id}
								agent={agent}
								projectId={projectId}
								canWrite={canWrite}
							/>
						))}
					</div>
				)}
			</div>

			{/* Chat shortcut section */}
			{agents.length > 0 && (
				<div className="border-t border-border/50 px-6 py-4 shrink-0">
					<p className="text-xs text-muted-foreground flex items-center gap-1.5">
						<MessageSquare className="size-3.5" />
						Click <strong>Configure &amp; view activity</strong> on any agent to
						chat, view conversations, and manage MCP servers and skills.
					</p>
				</div>
			)}

			<CreateAgentDialog
				projectId={projectId}
				open={createOpen}
				onOpenChange={setCreateOpen}
			/>
		</div>
	);
}
