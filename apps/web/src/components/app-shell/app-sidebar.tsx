import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams, useRouterState } from "@tanstack/react-router";
import {
	ArrowLeft,
	BookOpen,
	ChevronDown,
	ChevronRight,
	FileText,
	FolderKanban,
	GanttChart,
	Home,
	KanbanSquare,
	LayoutDashboard,
	Monitor,
	Moon,
	Plus,
	Settings,
	Shield,
	Sun,
	Users,
} from "lucide-react";
import { type ComponentType, useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarRail,
	SidebarSeparator,
} from "@/components/ui/sidebar";
import { usePermissions } from "@/hooks/use-permissions";
import type { ThemeMode } from "@/hooks/use-theme-mode";
import { useThemeMode } from "@/hooks/use-theme-mode";
import {
	sprintsQueryOptions,
	updateTask,
} from "@/lib/interaction-api";
import { projectQueryOptions, projectsQueryOptions } from "@/lib/project-api";
import { cn } from "@/lib/utils";

import { UserMenu } from "./user-menu";

// ── Project Switcher ───────────────────────────────────────────────────────────
function ProjectSwitcher({
	currentProjectId,
	canCreate,
}: {
	currentProjectId?: string;
	canCreate: boolean;
}) {
	const [open, setOpen] = useState(false);
	const { data: projectsResult } = useQuery(projectsQueryOptions());
	const { data: currentProject } = useQuery({
		...projectQueryOptions(currentProjectId ?? ""),
		enabled: !!currentProjectId,
	});

	const projects = projectsResult?.items ?? [];
	const label = currentProject?.name ?? "Projects";
	const initials = currentProject?.name
		? currentProject.name.slice(0, 2).toUpperCase()
		: null;

	return (
		<DropdownMenu open={open} onOpenChange={setOpen}>
			<DropdownMenuTrigger
				className={cn(
					"flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-sm font-medium text-sidebar-foreground/80 transition-all duration-150 select-none cursor-pointer",
					open
						? "bg-primary/10 text-primary"
						: "hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
				)}
			>
				<div className="flex size-5 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary text-[10px] font-bold">
					{initials ?? <FolderKanban className="size-3" />}
				</div>
				<span className="flex-1 truncate text-left">{label}</span>
				<ChevronDown
					className={cn(
						"size-3.5 shrink-0 opacity-40 transition-transform duration-200",
						open && "rotate-180",
					)}
				/>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="start" sideOffset={6} className="w-60">
				<DropdownMenuGroup>
					<DropdownMenuLabel className="text-xs text-muted-foreground pb-1">
						Your Projects
					</DropdownMenuLabel>
				</DropdownMenuGroup>
				<DropdownMenuSeparator />
				{projects.length > 0 ? (
					<DropdownMenuGroup>
						{projects.map((p) => (
							<DropdownMenuItem
								key={p.id}
								render={
									<Link
										to="/projects/$projectId"
										params={{ projectId: p.id }}
										className="flex items-center gap-2"
									/>
								}
							>
								<div className="flex size-5 shrink-0 items-center justify-center rounded bg-primary/15 text-primary text-[9px] font-bold">
									{p.name.slice(0, 2).toUpperCase()}
								</div>
								<span className="truncate">{p.name}</span>
								{p.id === currentProjectId && (
									<span className="ml-auto size-1.5 rounded-full bg-primary" />
								)}
							</DropdownMenuItem>
						))}
					</DropdownMenuGroup>
				) : (
					<div className="flex flex-col items-center gap-1 px-3 py-4">
						<div className="flex size-8 items-center justify-center rounded-md bg-muted">
							<FolderKanban className="size-4 text-muted-foreground" />
						</div>
						<p className="text-xs text-muted-foreground mt-0.5">
							No projects yet
						</p>
					</div>
				)}
				<DropdownMenuSeparator />
				{canCreate ? (
					<DropdownMenuItem render={<Link to="/home" />}>
						<Plus className="size-3.5" />
						New project
					</DropdownMenuItem>
				) : null}
			</DropdownMenuContent>
		</DropdownMenu>
	);
}

// ── Nav Item ───────────────────────────────────────────────────────────────────
function NavItem({
	to,
	icon: Icon,
	label,
	badge,
}: {
	to: string;
	icon: ComponentType<{ className?: string }>;
	label: string;
	badge?: string;
}) {
	const location = useRouterState({ select: (s) => s.location.pathname });
	const isActive = location === to || location.startsWith(`${to}/`);

	return (
		<SidebarMenuItem>
			<SidebarMenuButton
				isActive={isActive}
				tooltip={label}
				render={<Link to={to} />}
				className={cn(
					"relative transition-all duration-150",
					isActive
						? "bg-primary/10 text-primary font-medium before:absolute before:left-0 before:inset-y-2 before:w-0.75 before:rounded-full before:bg-primary"
						: "hover:bg-sidebar-accent/60",
				)}
			>
				<Icon className="size-4" />
				<span>{label}</span>
				{badge ? (
					<Badge className="ml-auto text-xs" variant="secondary">
						{badge}
					</Badge>
				) : null}
			</SidebarMenuButton>
		</SidebarMenuItem>
	);
}

// ── Project Nav ───────────────────────────────────────────────────────────────
const PROJECT_NAV_ITEMS = [
	{ segment: "", icon: LayoutDashboard, label: "Dashboard" },
	{ segment: "interactions", icon: BookOpen, label: "Interactions" },
	{ segment: "docs", icon: FileText, label: "Docs" },
	{ segment: "team", icon: Users, label: "Team" },
	{ segment: "settings", icon: Settings, label: "Settings" },
] as const;

function ProjectNav() {
	return (
		<SidebarGroup>
			<SidebarGroupContent>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							tooltip="All Projects"
							render={<Link to="/home" />}
							className="text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/60 transition-all"
						>
							<ArrowLeft className="size-4" />
							<span>All Projects</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	);
}

function ProjectNavItems({ projectId }: { projectId: string }) {
	const location = useRouterState({ select: (s) => s.location.pathname });

	return (
		<SidebarGroup>
			<SidebarGroupLabel>Project</SidebarGroupLabel>
			<SidebarGroupContent>
				<SidebarMenu>
					{PROJECT_NAV_ITEMS.map(({ segment, icon: Icon, label }) => {
						const href = segment
							? `/projects/${projectId}/${segment}`
							: `/projects/${projectId}`;
						const isActive = segment
							? location.startsWith(href)
							: location === href || location === `${href}/`;
						return (
							<SidebarMenuItem key={label}>
								<SidebarMenuButton
									isActive={isActive}
									tooltip={label}
									render={<Link to={href} />}
									className={cn(
										"relative transition-all duration-150",
										isActive
											? "bg-primary/10 text-primary font-medium before:absolute before:left-0 before:inset-y-2 before:w-0.75 before:rounded-full before:bg-primary"
											: "hover:bg-sidebar-accent/60",
									)}
								>
									<Icon className="size-4" />
									<span>{label}</span>
								</SidebarMenuButton>
							</SidebarMenuItem>
						);
					})}
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	);
}

// ── Project Integrations Section ───────────────────────────────────────────────
function ProjectInteractionsSection({ projectId }: { projectId: string }) {
	const location = useRouterState({ select: (s) => s.location.pathname });
	const { hasPermission } = usePermissions();
	const qc = useQueryClient();
	const [collapsed, setCollapsed] = useState(() => {
		try {
			return (
				localStorage.getItem(
					`paca:sidebar-interactions-collapsed:${projectId}`,
				) === "true"
			);
		} catch {
			return false;
		}
	});

	const canViewSprints = hasPermission("sprints.read");
	const canEditTasks = hasPermission("tasks.write");

	const [dragOverInteractionId, setDragOverInteractionId] = useState<
		string | null
	>(null);

	// Clear the drop-target highlight whenever any drag ends (covers drag-cancel
	// and mouse-release outside a valid target, where dragleave may not fire).
	useEffect(() => {
		const handleDragEnd = () => setDragOverInteractionId(null);
		document.addEventListener("dragend", handleDragEnd);
		return () => document.removeEventListener("dragend", handleDragEnd);
	}, []);

	const updateSprintMutation = useMutation({
		mutationFn: ({
			taskId,
			sprintId,
		}: {
			taskId: string;
			sprintId: string | null;
		}) => updateTask(projectId, taskId, { sprint_id: sprintId }),
		onSuccess: () => {
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "backlog-tasks"],
			});
			qc.invalidateQueries({
				queryKey: ["projects", projectId, "all-tasks"],
			});
			qc.invalidateQueries({ queryKey: ["projects", projectId, "sprints"] });
		},
	});

	const handleInteractionDragOver = (
		e: React.DragEvent,
		interactionId: string,
	) => {
		if (!canEditTasks) return;
		if (!e.dataTransfer.types.includes("application/x-paca-task-id")) return;
		e.preventDefault();
		e.dataTransfer.dropEffect = "move";
		setDragOverInteractionId(interactionId);
	};

	const handleInteractionDragLeave = (e: React.DragEvent) => {
		// Clear whenever leaving the item. If the cursor moves to a child element
		// within the same item, dragover immediately re-fires on the parent and
		// restores the highlight, so the brief gap is imperceptible.
		if (
			!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node | null)
		) {
			setDragOverInteractionId(null);
		}
	};

	const handleInteractionDrop = (
		e: React.DragEvent,
		sprintId: string | null,
	) => {
		e.preventDefault();
		setDragOverInteractionId(null);
		if (!canEditTasks) return;
		const taskId = e.dataTransfer.getData("text/plain");
		if (!taskId) return;
		updateSprintMutation.mutate({ taskId, sprintId });
	};

	const { data: sprints = [] } = useQuery({
		...sprintsQueryOptions(projectId),
		enabled: canViewSprints,
		retry: false,
		refetchInterval: 30_000,
	});

	// Hide entire section if user lacks the "View Sprints" permission
	if (!canViewSprints) return null;

	const openSprints = sprints
		.filter((s) => s.status === "active")
		.sort((a, b) => a.name.localeCompare(b.name));

	const backlogHref = `/projects/${projectId}/interactions/backlog`;
	const isBacklogActive = location.startsWith(backlogHref);

	const timelineHref = `/projects/${projectId}/interactions/timeline`;
	const isTimelineActive = location.startsWith(timelineHref);

	const toggle = () => {
		setCollapsed((prev) => {
			const next = !prev;
			try {
				localStorage.setItem(
					`paca:sidebar-interactions-collapsed:${projectId}`,
					String(next),
				);
			} catch {
				/* ignore */
			}
			return next;
		});
	};

	return (
		<>
			<SidebarGroup>
				<SidebarGroupLabel
					className="flex cursor-pointer items-center justify-between hover:text-sidebar-foreground transition-colors"
					onClick={toggle}
				>
					<span>Integrations</span>
					<ChevronRight
						className={cn(
							"size-3.5 transition-transform duration-200 text-sidebar-foreground/40",
							!collapsed && "rotate-90",
						)}
					/>
				</SidebarGroupLabel>

				{!collapsed && (
					<SidebarGroupContent>
						<SidebarMenu>
							{/* Product Backlog — always shown */}
							<SidebarMenuItem
								onDragOver={(e) => handleInteractionDragOver(e, "backlog")}
								onDragLeave={handleInteractionDragLeave}
								onDrop={(e) => handleInteractionDrop(e, null)}
							>
								<SidebarMenuButton
									isActive={isBacklogActive}
									tooltip="Product Backlog"
									render={<Link to={backlogHref} />}
									className={cn(
										"relative transition-all duration-150",
										isBacklogActive
											? "bg-primary/10 text-primary font-medium before:absolute before:left-0 before:inset-y-2 before:w-0.75 before:rounded-full before:bg-primary"
											: "hover:bg-sidebar-accent/60",
										dragOverInteractionId === "backlog" &&
											"ring-2 ring-primary/40 bg-primary/5 text-primary",
									)}
								>
									<BookOpen className="size-4" />
									<span>Product Backlog</span>
								</SidebarMenuButton>
							</SidebarMenuItem>
						{/* Timeline */}
						<SidebarMenuItem>
							<SidebarMenuButton
								isActive={isTimelineActive}
								tooltip="Timeline"
								render={<Link to={timelineHref} />}
								className={cn(
									"relative transition-all duration-150",
									isTimelineActive
										? "bg-primary/10 text-primary font-medium before:absolute before:left-0 before:inset-y-2 before:w-0.75 before:rounded-full before:bg-primary"
										: "hover:bg-sidebar-accent/60",
								)}
							>
								<GanttChart className="size-4" />
								<span>Timeline</span>
							</SidebarMenuButton>
						</SidebarMenuItem>
							{/* Open sprints */}
							{openSprints.map((sprint) => {
								const sprintHref = `/projects/${projectId}/interactions/sprints/${sprint.id}`;
								const isActive = location.startsWith(sprintHref);
								return (
									<SidebarMenuItem
										key={sprint.id}
										onDragOver={(e) => handleInteractionDragOver(e, sprint.id)}
										onDragLeave={handleInteractionDragLeave}
										onDrop={(e) => handleInteractionDrop(e, sprint.id)}
									>
										<SidebarMenuButton
											isActive={isActive}
											tooltip={sprint.name}
											render={<Link to={sprintHref} />}
											className={cn(
												"relative transition-all duration-150",
												isActive
													? "bg-primary/10 text-primary font-medium before:absolute before:left-0 before:inset-y-2 before:w-0.75 before:rounded-full before:bg-primary"
													: "hover:bg-sidebar-accent/60",
												dragOverInteractionId === sprint.id &&
													"ring-2 ring-primary/40 bg-primary/5 text-primary",
											)}
										>
											<KanbanSquare className="size-4" />
											<span className="flex-1 truncate">{sprint.name}</span>
											{sprint.status === "active" && (
												<span className="ml-auto flex size-1.5 shrink-0 rounded-full bg-emerald-500" />
											)}
										</SidebarMenuButton>
									</SidebarMenuItem>
								);
							})}
						</SidebarMenu>
					</SidebarGroupContent>
				)}
			</SidebarGroup>
		</>
	);
}

// ── Theme Switcher ─────────────────────────────────────────────────────────────
const THEME_MODES = [
	{ mode: "light" as ThemeMode, Icon: Sun, label: "Light" },
	{ mode: "dark" as ThemeMode, Icon: Moon, label: "Dark" },
	{ mode: "auto" as ThemeMode, Icon: Monitor, label: "Auto" },
] as const;

function ThemeSwitcher() {
	const { mode, set } = useThemeMode();
	const cycle = () =>
		set(mode === "light" ? "dark" : mode === "dark" ? "auto" : "light");
	const CurrentIcon = mode === "light" ? Sun : mode === "dark" ? Moon : Monitor;

	return (
		<>
			{/* Collapsed: single cycling icon button with tooltip */}
			<SidebarMenu className="hidden group-data-[collapsible=icon]:flex">
				<SidebarMenuItem>
					<SidebarMenuButton
						tooltip={`Theme: ${mode} — click to cycle`}
						onClick={cycle}
					>
						<CurrentIcon className="size-4" />
					</SidebarMenuButton>
				</SidebarMenuItem>
			</SidebarMenu>

			{/* Expanded: segmented 3-way control */}
			<div className="flex items-center justify-between px-2 py-1.5 group-data-[collapsible=icon]:hidden">
				<span className="text-xs font-medium text-sidebar-foreground/50 tracking-wide">
					Theme
				</span>
				<div className="flex items-center gap-0.5 rounded-md border border-sidebar-border bg-sidebar p-0.5">
					{THEME_MODES.map(({ mode: m, Icon, label }) => (
						<button
							key={m}
							type="button"
							onClick={() => set(m)}
							title={label}
							className={cn(
								"flex size-6 items-center justify-center rounded transition-all duration-150",
								mode === m
									? "bg-sidebar-accent text-sidebar-accent-foreground shadow-sm"
									: "text-sidebar-foreground/40 hover:text-sidebar-foreground/70",
							)}
						>
							<Icon className="size-3.5" />
						</button>
					))}
				</div>
			</div>
		</>
	);
}

// ── App Sidebar ────────────────────────────────────────────────────────────────
export function AppSidebar() {
	const { hasPermission } = usePermissions();
	const { resolvedMode } = useThemeMode();
	const { projectId } = useParams({ strict: false });

	const canAccessGlobalRoles =
		hasPermission("global_roles.read") || hasPermission("global_roles.write");

	const canAccessUsers =
		hasPermission("users.read") || hasPermission("users.write");

	const canCreateProject = hasPermission("projects.create");

	const showAdminSection = canAccessGlobalRoles || canAccessUsers;
	const isProjectContext = !!projectId;

	return (
		<Sidebar collapsible="icon">
			{/* Brand */}
			<SidebarHeader className="gap-2 pb-2">
				<div className="flex items-center gap-2.5 px-2 pt-1">
					<Link to="/home">
						<img
							src={
								resolvedMode === "dark"
									? "/paca-logo-dark.svg"
									: "/paca-logo.svg"
							}
							alt="Paca Logo"
							className="size-8 shrink-0"
						/>
					</Link>
					<span className="font-[Syne] font-bold text-[15px] tracking-tight text-sidebar-foreground group-data-[collapsible=icon]:hidden">
						paca
					</span>
				</div>
				<div className="group-data-[collapsible=icon]:hidden">
					<ProjectSwitcher
						currentProjectId={projectId}
						canCreate={canCreateProject}
					/>
				</div>
			</SidebarHeader>

			<SidebarSeparator />

			{/* Navigation — switches between workspace and project context */}
			<SidebarContent>
				{isProjectContext ? (
					<>
						<ProjectNav />
						<SidebarSeparator />
						<ProjectNavItems projectId={projectId} />
						<SidebarSeparator />
						<ProjectInteractionsSection projectId={projectId} />
					</>
				) : (
					<>
						<SidebarGroup>
							<SidebarGroupContent>
								<SidebarMenu>
									<NavItem to="/home" icon={Home} label="Home" />
								</SidebarMenu>
							</SidebarGroupContent>
						</SidebarGroup>

						{/* Admin section */}
						{showAdminSection ? (
							<>
								<SidebarSeparator />
								<SidebarGroup>
									<SidebarGroupLabel>Administration</SidebarGroupLabel>
									<SidebarGroupContent>
										<SidebarMenu>
											{canAccessGlobalRoles ? (
												<NavItem
													to="/admin/global-roles"
													icon={Shield}
													label="Global Roles"
												/>
											) : null}
											{canAccessUsers ? (
												<NavItem to="/admin/users" icon={Users} label="Users" />
											) : null}
										</SidebarMenu>
									</SidebarGroupContent>
								</SidebarGroup>
							</>
						) : null}
					</>
				)}
			</SidebarContent>

			{/* Footer: theme toggle + user menu */}
			<SidebarSeparator />
			<SidebarFooter className="gap-1 pb-3">
				<ThemeSwitcher />
				<UserMenu />
			</SidebarFooter>

			<SidebarRail />
		</Sidebar>
	);
}
