import {
	Bot,
	BookOpen,
	Layers,
	ListTodo,
	type LucideIcon,
	Settings,
	Shield,
	Users,
} from "lucide-react";

export interface KnownPermission {
	key: string;
	label: string;
	description: string;
	domain: string;
}

export const PROJECT_KNOWN_PERMISSIONS: KnownPermission[] = [
	// projects
	{
		key: "projects.read",
		label: "Read Project",
		description: "View project details and settings",
		domain: "projects",
	},
	{
		key: "projects.write",
		label: "Edit Project",
		description: "Update project name, description, and settings",
		domain: "projects",
	},
	{
		key: "projects.delete",
		label: "Delete Project",
		description: "Permanently delete this project",
		domain: "projects",
	},
	// project members
	{
		key: "project.members.read",
		label: "View Members",
		description: "List and view project members",
		domain: "project.members",
	},
	{
		key: "project.members.write",
		label: "Manage Members",
		description: "Add, remove, and reassign project members",
		domain: "project.members",
	},
	// project roles
	{
		key: "project.roles.read",
		label: "View Roles",
		description: "List and view project role definitions",
		domain: "project.roles",
	},
	{
		key: "project.roles.write",
		label: "Manage Roles",
		description: "Create, edit, and delete project roles",
		domain: "project.roles",
	},
	// tasks
	{
		key: "tasks.read",
		label: "View Tasks",
		description: "Browse and read tasks in the project",
		domain: "tasks",
	},
	{
		key: "tasks.write",
		label: "Edit Tasks",
		description: "Create, update, and move tasks",
		domain: "tasks",
	},
	// sprints
	{
		key: "sprints.read",
		label: "View Sprints",
		description: "Browse sprint boards and backlogs",
		domain: "sprints",
	},
	{
		key: "sprints.write",
		label: "Manage Sprints",
		description: "Create, update, and close sprints",
		domain: "sprints",
	},
	// docs
	{
		key: "docs.read",
		label: "View Documents",
		description: "Browse and read documents in the project",
		domain: "docs",
	},
	{
		key: "docs.write",
		label: "Edit Documents",
		description: "Create, update, and delete documents and folders",
		domain: "docs",
	},
	// agents
	{
		key: "agents.read",
		label: "View Agents",
		description: "Browse AI agents and view their configuration",
		domain: "agents",
	},
	{
		key: "agents.write",
		label: "Manage Agents",
		description: "Create, configure, and delete AI agents",
		domain: "agents",
	},
];

export interface PermissionGroup {
	domain: string;
	label: string;
	Icon: LucideIcon;
}

export const PROJECT_PERMISSION_GROUPS: PermissionGroup[] = [
	{ domain: "projects", label: "Project", Icon: Settings },
	{ domain: "project.members", label: "Members", Icon: Users },
	{ domain: "project.roles", label: "Roles", Icon: Shield },
	{ domain: "tasks", label: "Tasks", Icon: ListTodo },
	{ domain: "sprints", label: "Sprints", Icon: Layers },
	{ domain: "docs", label: "Documents", Icon: BookOpen },
	{ domain: "agents", label: "AI Agents", Icon: Bot },
];
