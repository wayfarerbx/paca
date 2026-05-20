import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	Bot,
	Check,
	Clock,
	Code2,
	GitBranch,
	GitPullRequest,
	Loader2,
	MessageSquare,
	Plus,
	Save,
	Server,
	Trash2,
	Wand2,
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useProjectPermissions } from "@/hooks/use-project-permissions";
import {
	type Agent,
	type AgentConversation,
	type AgentMCPServer,
	type AgentSkill,
	addMCPServer,
	addSkill,
	agentMCPServersQueryOptions,
	agentQueryOptions,
	agentSkillsQueryOptions,
	CONVERSATION_STATUS_COLORS,
	CONVERSATION_STATUS_LABELS,
	conversationsQueryOptions,
	deleteMCPServer,
	deleteSkill,
	llmModelsQueryOptions,
	updateAgent,
	updateMCPServer,
	updateSkill,
} from "@/lib/agent-api";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/agents/$agentId/",
)({
	loader: async ({
		context: { queryClient },
		params: { projectId, agentId },
	}) => {
		await Promise.all([
			queryClient.ensureQueryData(agentQueryOptions(projectId, agentId)),
			queryClient.ensureQueryData(
				agentMCPServersQueryOptions(projectId, agentId),
			),
			queryClient.ensureQueryData(agentSkillsQueryOptions(projectId, agentId)),
			queryClient.ensureQueryData(
				conversationsQueryOptions(projectId, agentId),
			),
		]);
	},
	component: AgentDetailPage,
});

type Tab = "overview" | "mcp-servers" | "skills" | "conversations";

// ── Overview Tab ──────────────────────────────────────────────────────────────

function OverviewTab({
	agent,
	projectId,
	canWrite,
}: {
	agent: Agent;
	projectId: string;
	canWrite: boolean;
}) {
	const qc = useQueryClient();
	const { data: llmModels = {} } = useQuery(llmModelsQueryOptions);
	const [name, setName] = useState(agent.name);
	const [llmProvider, setLlmProvider] = useState(agent.llm_provider);
	const [llmModel, setLlmModel] = useState(agent.llm_model);
	const [llmApiKey, setLlmApiKey] = useState("");
	const [llmBaseUrl, setLlmBaseUrl] = useState(agent.llm_base_url ?? "");
	const [systemPrompt, setSystemPrompt] = useState(agent.system_prompt);
	const [canClone, setCanClone] = useState(agent.can_clone_repos);
	const [canPrs, setCanPrs] = useState(agent.can_create_prs);
	const [maxIter, setMaxIter] = useState(String(agent.max_iterations));
	const [timeout, setTimeout] = useState(String(agent.timeout_minutes));

	const isDirty =
		name !== agent.name ||
		llmProvider !== agent.llm_provider ||
		llmModel !== agent.llm_model ||
		llmApiKey !== "" ||
		llmBaseUrl !== (agent.llm_base_url ?? "") ||
		systemPrompt !== agent.system_prompt ||
		canClone !== agent.can_clone_repos ||
		canPrs !== agent.can_create_prs ||
		maxIter !== String(agent.max_iterations) ||
		timeout !== String(agent.timeout_minutes);

	const saveMutation = useMutation({
		mutationFn: () =>
			updateAgent(projectId, agent.id, {
				name: name.trim(),
				llm_provider: llmProvider,
				llm_model: llmModel,
				...(llmApiKey ? { llm_api_key: llmApiKey } : {}),
				llm_base_url: llmBaseUrl || null,
				system_prompt: systemPrompt,
				can_clone_repos: canClone,
				can_create_prs: canPrs,
				max_iterations: Number(maxIter) || 30,
				timeout_minutes: Number(timeout) || 60,
			}),
		onSuccess: (updated) => {
			qc.setQueryData(
				["projects", projectId, "agents", agent.id],
				updated,
			);
			setLlmApiKey("");
		},
	});

	const availableModels: string[] = llmModels[llmProvider] ?? [];
	const providers = Object.keys(llmModels);

	return (
		<div className="space-y-6 max-w-2xl">
			<div className="space-y-1.5">
				<Label>Name</Label>
				<Input
					value={name}
					onChange={(e) => setName(e.target.value)}
					disabled={!canWrite}
				/>
			</div>

			<Separator />

			<div>
				<p className="text-sm font-medium mb-3">LLM Configuration</p>
				<div className="grid grid-cols-2 gap-3">
					<div className="space-y-1.5">
						<Label>Provider</Label>
						<Select
							value={llmProvider}
							onValueChange={(v) => {
								if (!v) return;
								setLlmProvider(v);
								setLlmModel(llmModels[v]?.[0] ?? "");
							}}
							disabled={!canWrite}
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
						<Select
							value={llmModel}
							onValueChange={(v) => v && setLlmModel(v)}
							disabled={!canWrite}
						>
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
				<div className="space-y-1.5 mt-3">
					<Label>
						API Key Update{" "}
						<span className="text-muted-foreground font-normal text-xs">
							(leave blank to keep current)
						</span>
					</Label>
					<Input
						type="password"
						placeholder="sk-ant-…"
						value={llmApiKey}
						onChange={(e) => setLlmApiKey(e.target.value)}
						disabled={!canWrite}
					/>
				</div>
				<div className="space-y-1.5 mt-3">
					<Label>
						Base URL{" "}
						<span className="text-muted-foreground font-normal text-xs">
							(optional)
						</span>
					</Label>
					<Input
						placeholder="https://api.openai.com/v1"
						value={llmBaseUrl}
						onChange={(e) => setLlmBaseUrl(e.target.value)}
						disabled={!canWrite}
					/>
				</div>
			</div>

			<Separator />

			<div className="space-y-1.5">
				<Label>System Prompt</Label>
				<Textarea
					value={systemPrompt}
					onChange={(e) => setSystemPrompt(e.target.value)}
					rows={5}
					disabled={!canWrite}
					className="font-mono text-xs"
				/>
			</div>

			<Separator />

			<div>
				<p className="text-sm font-medium mb-3">Capabilities</p>
				<div className="space-y-3">
					<div className="flex items-center justify-between">
						<div>
							<p className="text-sm">Clone repositories</p>
							<p className="text-xs text-muted-foreground">
								Allow agent to git clone repos locally
							</p>
						</div>
						<Switch
							checked={canClone}
							onCheckedChange={setCanClone}
							disabled={!canWrite}
						/>
					</div>
					<div className="flex items-center justify-between">
						<div>
							<p className="text-sm">Create pull requests</p>
							<p className="text-xs text-muted-foreground">
								Allow agent to open PRs on GitHub
							</p>
						</div>
						<Switch
							checked={canPrs}
							onCheckedChange={setCanPrs}
							disabled={!canWrite}
						/>
					</div>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-3">
				<div className="space-y-1.5">
					<Label>Max iterations</Label>
					<Input
						type="number"
						min={1}
						max={200}
						value={maxIter}
						onChange={(e) => setMaxIter(e.target.value)}
						disabled={!canWrite}
					/>
				</div>
				<div className="space-y-1.5">
					<Label>Timeout (minutes)</Label>
					<Input
						type="number"
						min={1}
						max={480}
						value={timeout}
						onChange={(e) => setTimeout(e.target.value)}
						disabled={!canWrite}
					/>
				</div>
			</div>

			{canWrite && (
				<div className="flex items-center gap-3 pt-2">
					<Button
						onClick={() => saveMutation.mutate()}
						disabled={!isDirty || saveMutation.isPending}
					>
						{saveMutation.isPending ? (
							<Loader2 className="size-4 mr-2 animate-spin" />
						) : (
							<Save className="size-4 mr-2" />
						)}
						Save changes
					</Button>
					{saveMutation.isSuccess && (
						<span className="flex items-center gap-1 text-xs text-emerald-600">
							<Check className="size-3" />
							Saved
						</span>
					)}
					{saveMutation.isError && (
						<span className="text-xs text-destructive">
							Failed to save. Please try again.
						</span>
					)}
				</div>
			)}
		</div>
	);
}

// ── MCP Servers Tab ───────────────────────────────────────────────────────────

function AddMCPServerDialog({
	projectId,
	agentId,
	open,
	onOpenChange,
}: {
	projectId: string;
	agentId: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	const qc = useQueryClient();
	const [serverName, setServerName] = useState("");
	const [transport, setTransport] = useState<"stdio" | "sse" | "http">("stdio");
	const [command, setCommand] = useState("");
	const [args, setArgs] = useState("");
	const [url, setUrl] = useState("");

	const addMutation = useMutation({
		mutationFn: () =>
			addMCPServer(projectId, agentId, {
				server_name: serverName.trim(),
				transport,
				command: transport === "stdio" ? command.trim() || null : null,
				args:
					transport === "stdio"
						? args
								.split(/\s+/)
								.map((a) => a.trim())
								.filter(Boolean)
						: [],
				url:
					transport !== "stdio" ? url.trim() || null : null,
			}),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "mcp-servers"],
			});
			onOpenChange(false);
			setServerName("");
			setCommand("");
			setArgs("");
			setUrl("");
		},
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-md">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Server className="size-4 text-primary" />
						Add MCP Server
					</DialogTitle>
					<DialogDescription>
						Connect an MCP server to extend the agent's capabilities.
					</DialogDescription>
				</DialogHeader>
				<div className="space-y-4 py-2">
					<div className="space-y-1.5">
						<Label>Server name</Label>
						<Input
							placeholder="filesystem"
							value={serverName}
							onChange={(e) => setServerName(e.target.value)}
						/>
					</div>
					<div className="space-y-1.5">
						<Label>Transport</Label>
						<Select
							value={transport}
							onValueChange={(v) => setTransport(v as typeof transport)}
						>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="stdio">stdio</SelectItem>
								<SelectItem value="sse">SSE</SelectItem>
								<SelectItem value="http">HTTP</SelectItem>
							</SelectContent>
						</Select>
					</div>
					{transport === "stdio" ? (
						<>
							<div className="space-y-1.5">
								<Label>Command</Label>
								<Input
									placeholder="npx"
									value={command}
									onChange={(e) => setCommand(e.target.value)}
								/>
							</div>
							<div className="space-y-1.5">
								<Label>
									Args{" "}
									<span className="text-muted-foreground font-normal text-xs">
										(space-separated)
									</span>
								</Label>
								<Input
									placeholder="-y @modelcontextprotocol/server-filesystem /tmp"
									value={args}
									onChange={(e) => setArgs(e.target.value)}
								/>
							</div>
						</>
					) : (
						<div className="space-y-1.5">
							<Label>URL</Label>
							<Input
								placeholder="https://mcp.example.com/sse"
								value={url}
								onChange={(e) => setUrl(e.target.value)}
							/>
						</div>
					)}
				</div>
				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						onClick={() => addMutation.mutate()}
						disabled={!serverName.trim() || addMutation.isPending}
					>
						{addMutation.isPending ? (
							<Loader2 className="size-4 animate-spin" />
						) : (
							"Add server"
						)}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

function MCPServersTab({
	projectId,
	agentId,
	canWrite,
}: {
	projectId: string;
	agentId: string;
	canWrite: boolean;
}) {
	const qc = useQueryClient();
	const { data: servers = [] } = useQuery(
		agentMCPServersQueryOptions(projectId, agentId),
	);
	const [addOpen, setAddOpen] = useState(false);

	const toggleMutation = useMutation({
		mutationFn: (s: AgentMCPServer) =>
			updateMCPServer(projectId, agentId, s.id, {
				is_enabled: !s.is_enabled,
			}),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "mcp-servers"],
			});
		},
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => deleteMCPServer(projectId, agentId, id),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "mcp-servers"],
			});
		},
	});

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<p className="text-sm text-muted-foreground">
					{servers.length} server{servers.length !== 1 && "s"} configured
				</p>
				{canWrite && (
					<Button size="sm" onClick={() => setAddOpen(true)}>
						<Plus className="size-4 mr-1.5" />
						Add server
					</Button>
				)}
			</div>

			{servers.length === 0 ? (
				<div className="flex flex-col items-center justify-center gap-3 py-14 rounded-xl border border-dashed border-border">
					<Server className="size-8 text-muted-foreground/40" />
					<p className="text-sm text-muted-foreground">No MCP servers added</p>
					{canWrite && (
						<Button size="sm" variant="outline" onClick={() => setAddOpen(true)}>
							<Plus className="size-3.5 mr-1" />
							Add your first server
						</Button>
					)}
				</div>
			) : (
				<div className="space-y-2">
					{servers.map((s) => (
						<div
							key={s.id}
							className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-card px-4 py-3"
						>
							<div className="flex items-center gap-3 min-w-0">
								<Server className="size-4 text-muted-foreground shrink-0" />
								<div className="min-w-0">
									<p className="text-sm font-medium truncate">{s.server_name}</p>
									<p className="text-xs text-muted-foreground font-mono truncate">
										{s.transport}
										{s.command ? ` · ${s.command}` : ""}
										{s.url ? ` · ${s.url}` : ""}
									</p>
								</div>
							</div>
							<div className="flex items-center gap-2 shrink-0">
								<Switch
									checked={s.is_enabled}
									onCheckedChange={() =>
										canWrite && toggleMutation.mutate(s)
									}
									disabled={!canWrite || toggleMutation.isPending}
								/>
								{canWrite && (
									<Button
										variant="ghost"
										size="icon"
										className="size-7 text-muted-foreground hover:text-destructive"
										onClick={() => deleteMutation.mutate(s.id)}
										disabled={deleteMutation.isPending}
									>
										<Trash2 className="size-3.5" />
									</Button>
								)}
							</div>
						</div>
					))}
				</div>
			)}

			<AddMCPServerDialog
				projectId={projectId}
				agentId={agentId}
				open={addOpen}
				onOpenChange={setAddOpen}
			/>
		</div>
	);
}

// ── Skills Tab ────────────────────────────────────────────────────────────────

function AddSkillDialog({
	projectId,
	agentId,
	open,
	onOpenChange,
}: {
	projectId: string;
	agentId: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	const qc = useQueryClient();
	const [skillName, setSkillName] = useState("");
	const [source, setSource] = useState<"inline" | "marketplace" | "github_url">(
		"inline",
	);
	const [skillContent, setSkillContent] = useState("");
	const [sourceUrl, setSourceUrl] = useState("");

	const addMutation = useMutation({
		mutationFn: () =>
			addSkill(projectId, agentId, {
				skill_name: skillName.trim(),
				skill_source: source,
				skill_content: source === "inline" ? skillContent : undefined,
				source_url: source !== "inline" ? sourceUrl.trim() : null,
			}),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "skills"],
			});
			onOpenChange(false);
			setSkillName("");
			setSkillContent("");
			setSourceUrl("");
		},
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-md">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Wand2 className="size-4 text-primary" />
						Add Skill
					</DialogTitle>
					<DialogDescription>
						Give the agent specialised instructions or capabilities.
					</DialogDescription>
				</DialogHeader>
				<div className="space-y-4 py-2">
					<div className="space-y-1.5">
						<Label>Skill name</Label>
						<Input
							placeholder="code-reviewer"
							value={skillName}
							onChange={(e) => setSkillName(e.target.value)}
						/>
					</div>
					<div className="space-y-1.5">
						<Label>Source</Label>
						<Select
							value={source}
							onValueChange={(v) =>
								setSource(v as typeof source)
							}
						>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="inline">Inline (write content here)</SelectItem>
								<SelectItem value="marketplace">Marketplace</SelectItem>
								<SelectItem value="github_url">GitHub URL</SelectItem>
							</SelectContent>
						</Select>
					</div>
					{source === "inline" ? (
						<div className="space-y-1.5">
							<Label>Skill content (Markdown)</Label>
							<Textarea
								placeholder="# Code Reviewer&#10;&#10;You review pull requests for security, performance…"
								value={skillContent}
								onChange={(e) => setSkillContent(e.target.value)}
								rows={5}
								className="font-mono text-xs"
							/>
						</div>
					) : (
						<div className="space-y-1.5">
							<Label>URL</Label>
							<Input
								placeholder={
									source === "marketplace"
										? "paca/code-reviewer@1.0.0"
										: "https://github.com/org/skills/blob/main/SKILL.md"
								}
								value={sourceUrl}
								onChange={(e) => setSourceUrl(e.target.value)}
							/>
						</div>
					)}
				</div>
				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						onClick={() => addMutation.mutate()}
						disabled={!skillName.trim() || addMutation.isPending}
					>
						{addMutation.isPending ? (
							<Loader2 className="size-4 animate-spin" />
						) : (
							"Add skill"
						)}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

function SkillsTab({
	projectId,
	agentId,
	canWrite,
}: {
	projectId: string;
	agentId: string;
	canWrite: boolean;
}) {
	const qc = useQueryClient();
	const { data: skills = [] } = useQuery(
		agentSkillsQueryOptions(projectId, agentId),
	);
	const [addOpen, setAddOpen] = useState(false);

	const toggleMutation = useMutation({
		mutationFn: (s: AgentSkill) =>
			updateSkill(projectId, agentId, s.id, { is_enabled: !s.is_enabled }),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "skills"],
			});
		},
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => deleteSkill(projectId, agentId, id),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "agents", agentId, "skills"],
			});
		},
	});

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<p className="text-sm text-muted-foreground">
					{skills.length} skill{skills.length !== 1 && "s"} attached
				</p>
				{canWrite && (
					<Button size="sm" onClick={() => setAddOpen(true)}>
						<Plus className="size-4 mr-1.5" />
						Add skill
					</Button>
				)}
			</div>

			{skills.length === 0 ? (
				<div className="flex flex-col items-center justify-center gap-3 py-14 rounded-xl border border-dashed border-border">
					<Wand2 className="size-8 text-muted-foreground/40" />
					<p className="text-sm text-muted-foreground">No skills yet</p>
					{canWrite && (
						<Button size="sm" variant="outline" onClick={() => setAddOpen(true)}>
							<Plus className="size-3.5 mr-1" />
							Add first skill
						</Button>
					)}
				</div>
			) : (
				<div className="space-y-2">
					{skills.map((s) => (
						<div
							key={s.id}
							className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-card px-4 py-3"
						>
							<div className="flex items-center gap-3 min-w-0">
								<Code2 className="size-4 text-muted-foreground shrink-0" />
								<div className="min-w-0">
									<p className="text-sm font-medium truncate">{s.skill_name}</p>
									<p className="text-xs text-muted-foreground">
										{s.skill_source}
										{s.source_url ? ` · ${s.source_url}` : ""}
									</p>
								</div>
							</div>
							<div className="flex items-center gap-2 shrink-0">
								<Switch
									checked={s.is_enabled}
									onCheckedChange={() => canWrite && toggleMutation.mutate(s)}
									disabled={!canWrite || toggleMutation.isPending}
								/>
								{canWrite && (
									<Button
										variant="ghost"
										size="icon"
										className="size-7 text-muted-foreground hover:text-destructive"
										onClick={() => deleteMutation.mutate(s.id)}
										disabled={deleteMutation.isPending}
									>
										<Trash2 className="size-3.5" />
									</Button>
								)}
							</div>
						</div>
					))}
				</div>
			)}

			<AddSkillDialog
				projectId={projectId}
				agentId={agentId}
				open={addOpen}
				onOpenChange={setAddOpen}
			/>
		</div>
	);
}

// ── Conversations Tab ─────────────────────────────────────────────────────────

function ConversationRow({
	conv,
	onClick,
}: {
	conv: AgentConversation;
	onClick: () => void;
}) {
	const statusColor = CONVERSATION_STATUS_COLORS[conv.status];
	const statusLabel = CONVERSATION_STATUS_LABELS[conv.status];

	return (
		<button
			type="button"
			onClick={onClick}
			className="w-full flex items-center gap-4 rounded-lg border border-border/60 bg-card px-4 py-3 text-left transition-colors hover:border-border hover:bg-accent/30"
		>
			<div className="flex flex-col gap-0.5 min-w-0 flex-1">
				<div className="flex items-center gap-2">
					<span className="text-sm font-medium truncate">
						{conv.trigger_type === "chat_message" ? "Chat" : "Task"} ·{" "}
						{conv.id.slice(0, 8)}
					</span>
					<Badge
						variant="outline"
						className={`text-[10px] font-semibold shrink-0 ${statusColor}`}
					>
						{statusLabel}
					</Badge>
				</div>
				<div className="flex items-center gap-3 text-xs text-muted-foreground">
					<span className="flex items-center gap-1">
						<Zap className="size-3" />
						{conv.iteration_count} iterations
					</span>
					{conv.branch_name && (
						<span className="flex items-center gap-1 truncate">
							<GitBranch className="size-3" />
							{conv.branch_name}
						</span>
					)}
					{conv.pr_url && (
						<span className="flex items-center gap-1">
							<GitPullRequest className="size-3" />
							PR opened
						</span>
					)}
					<span className="flex items-center gap-1 ml-auto">
						<Clock className="size-3" />
						{new Date(conv.created_at).toLocaleDateString()}
					</span>
				</div>
			</div>
		</button>
	);
}

function ConversationsTab({
	projectId,
	agentId,
}: {
	projectId: string;
	agentId: string;
}) {
	const { data: conversations = [], isLoading } = useQuery(
		conversationsQueryOptions(projectId, agentId),
	);
	const [expandedId, setExpandedId] = useState<string | null>(null);

	if (isLoading) {
		return (
			<div className="space-y-2">
				{Array.from({ length: 3 }).map((_, i) => (
					// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
					<Skeleton key={i} className="h-16 rounded-lg" />
				))}
			</div>
		);
	}

	if (conversations.length === 0) {
		return (
			<div className="flex flex-col items-center justify-center gap-3 py-14 rounded-xl border border-dashed border-border">
				<MessageSquare className="size-8 text-muted-foreground/40" />
				<p className="text-sm text-muted-foreground">No conversations yet</p>
				<p className="text-xs text-muted-foreground max-w-xs text-center">
					Conversations start when a task is assigned to this agent or someone
					messages it.
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-2">
			{conversations.map((conv) => (
				<div key={conv.id}>
					<ConversationRow
						conv={conv}
						onClick={() =>
							setExpandedId(expandedId === conv.id ? null : conv.id)
						}
					/>
					{expandedId === conv.id && (
						<ConversationEventList
							projectId={projectId}
							conversationId={conv.id}
						/>
					)}
				</div>
			))}
		</div>
	);
}

function ConversationEventList({
	projectId,
	conversationId,
}: {
	projectId: string;
	conversationId: string;
}) {
	const { data: events = [], isLoading } = useQuery({
		queryKey: [
			"projects",
			projectId,
			"conversations",
			conversationId,
			"events",
		],
		queryFn: async () => {
			const { listConversationEvents } = await import("@/lib/agent-api");
			return listConversationEvents(projectId, conversationId);
		},
		refetchInterval: 5000,
	});

	if (isLoading) {
		return (
			<div className="ml-4 mt-1 space-y-1">
				{Array.from({ length: 3 }).map((_, i) => (
					// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
					<Skeleton key={i} className="h-8 rounded" />
				))}
			</div>
		);
	}

	if (events.length === 0) {
		return (
			<div className="ml-4 mt-1 rounded-lg border border-dashed border-border bg-muted/20 px-4 py-3 text-xs text-muted-foreground">
				No events recorded yet
			</div>
		);
	}

	return (
		<div className="ml-4 mt-1 rounded-lg border border-border/40 bg-muted/20 divide-y divide-border/40 text-xs font-mono max-h-64 overflow-y-auto">
			{events.map((ev) => (
				<div key={ev.id} className="px-3 py-2 flex items-start gap-2">
					<span className="text-muted-foreground shrink-0 tabular-nums">
						{ev.event_index.toString().padStart(3, "0")}
					</span>
					<span
						className={
							ev.event_source === "agent"
								? "text-primary"
								: ev.event_source === "user"
									? "text-amber-500"
									: "text-muted-foreground"
						}
					>
						{ev.event_type}
					</span>
					{!!ev.payload?.message && (
						<span className="text-foreground/70 truncate">
							{String(ev.payload.message).slice(0, 120)}
						</span>
					)}
				</div>
			))}
		</div>
	);
}

// ── Page ──────────────────────────────────────────────────────────────────────

const TABS: { id: Tab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
	{ id: "overview", label: "Overview", icon: Bot },
	{ id: "mcp-servers", label: "MCP Servers", icon: Server },
	{ id: "skills", label: "Skills", icon: Wand2 },
	{ id: "conversations", label: "Conversations", icon: MessageSquare },
];

function AgentDetailPage() {
	const { projectId, agentId } = Route.useParams();
	const { hasProjectPermission } = useProjectPermissions(projectId);
	const canWrite = hasProjectPermission("agents.write");

	const { data: agent } = useQuery(agentQueryOptions(projectId, agentId));
	const [activeTab, setActiveTab] = useState<Tab>("overview");

	if (!agent) {
		return (
			<div className="flex flex-col gap-4 p-6">
				<Skeleton className="h-16 w-full rounded-xl" />
				<Skeleton className="h-64 w-full rounded-xl" />
			</div>
		);
	}

	const initials = agent.name
		.split(" ")
		.map((w) => w[0])
		.join("")
		.toUpperCase()
		.slice(0, 2);

	return (
		<div className="flex flex-col flex-1 min-h-0">
			{/* Agent header */}
			<div className="border-b border-border/50 px-6 py-5 shrink-0">
				<div className="flex items-center gap-4">
					<Avatar className="size-12 rounded-xl bg-primary/10">
						<AvatarFallback className="rounded-xl bg-primary/10 text-primary font-bold text-base">
							{initials}
						</AvatarFallback>
					</Avatar>
					<div>
						<h1 className="text-lg font-semibold">{agent.name}</h1>
						<div className="flex items-center gap-2 mt-0.5">
							<span className="text-sm text-muted-foreground">
								@{agent.handle}
							</span>
							<span className="text-muted-foreground/40">·</span>
							<Badge variant="secondary" className="text-[10px]">
								{agent.llm_provider}
							</Badge>
						</div>
					</div>
				</div>
			</div>

			{/* Tabs */}
			<div className="border-b border-border/50 px-6 shrink-0">
				<div className="flex items-center gap-1 -mb-px">
					{TABS.map((tab) => {
						const Icon = tab.icon;
						const isActive = activeTab === tab.id;
						return (
							<button
								key={tab.id}
								type="button"
								onClick={() => setActiveTab(tab.id)}
								className={`flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium border-b-2 transition-colors ${
									isActive
										? "border-primary text-primary"
										: "border-transparent text-muted-foreground hover:text-foreground"
								}`}
							>
								<Icon className="size-3.5" />
								{tab.label}
							</button>
						);
					})}
				</div>
			</div>

			{/* Tab content */}
			<div className="flex-1 overflow-auto p-6">
				{activeTab === "overview" && (
					<OverviewTab
						agent={agent}
						projectId={projectId}
						canWrite={canWrite}
					/>
				)}
				{activeTab === "mcp-servers" && (
					<MCPServersTab
						projectId={projectId}
						agentId={agentId}
						canWrite={canWrite}
					/>
				)}
				{activeTab === "skills" && (
					<SkillsTab
						projectId={projectId}
						agentId={agentId}
						canWrite={canWrite}
					/>
				)}
				{activeTab === "conversations" && (
					<ConversationsTab projectId={projectId} agentId={agentId} />
				)}
			</div>
		</div>
	);
}
