import type { Task } from "@/lib/interaction-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";

// ── Extended types (UI-first, wired to API later) ─────────────────────────────

export interface CustomFieldDef {
	id: string;
	display_name: string;
	field_key: string;
	field_type: "Text" | "Number" | "Date" | "Checkbox" | "Select";
	required?: boolean;
	options?: string[];
}

export interface Attachment {
	id: string;
	name: string;
	size?: number;
	uploaded_at: string;
	url?: string;
}

export interface ActivityEntry {
	id: string;
	type:
		| "created"
		| "field_change"
		| "comment"
		| "status_change"
		| "assignee_change";
	author: string;
	avatar?: string;
	content: string;
	timestamp: string;
}

export interface ChecklistItem {
	id: string;
	text: string;
	checked: boolean;
}

export interface Checklist {
	id: string;
	title: string;
	items: ChecklistItem[];
}

// ── Component props ────────────────────────────────────────────────────────────

export interface TaskDetailModalProps {
	task: Task | null;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members?: ProjectMember[];
	customFields?: CustomFieldDef[];
	projectName?: string;
	interactionName?: string;
	projectId?: string;
	taskIdPrefix?: string;
	mode?: "modal" | "page";
	canEdit?: boolean;
}
