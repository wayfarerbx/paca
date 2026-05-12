import { cn } from "@/lib/utils";
import { timeAgo } from "./helpers";
import type { ActivityEntry } from "./types";

export type ActivityNameMaps = {
	members: Record<string, string>; // user_id → display name
	sprints: Record<string, string>; // sprint_id → sprint name
};

type FieldChange = {
	field: string;
	old?: unknown;
	new?: unknown;
};

function label(value: unknown): string {
	if (value === null || value === undefined || value === "") return "none";
	if (Array.isArray(value)) return value.length > 0 ? value.join(", ") : "none";
	return String(value);
}

export function describeTaskChange(
	change: FieldChange,
	names: ActivityNameMaps,
): string {
	const oldVal = label(change.old);
	const newVal = label(change.new);
	const hasOld =
		change.old !== null && change.old !== undefined && change.old !== "";
	const hasNew =
		change.new !== null && change.new !== undefined && change.new !== "";

	const resolveMember = (id: unknown) =>
		(id && names.members[String(id)]) || (id ? String(id).slice(0, 8) : "none");
	const resolveSprint = (id: unknown) =>
		(id && names.sprints[String(id)]) || (id ? String(id).slice(0, 8) : "none");

	switch (change.field) {
		case "status":
			if (hasOld && hasNew)
				return `changed status from "${oldVal}" to "${newVal}"`;
			if (hasNew) return `set status to "${newVal}"`;
			return `cleared status`;
		case "task_type":
			if (hasOld && hasNew)
				return `changed type from "${oldVal}" to "${newVal}"`;
			if (hasNew) return `set type to "${newVal}"`;
			return `cleared type`;
		case "title":
			if (hasOld) return `renamed from "${oldVal}" to "${newVal}"`;
			return `set title to "${newVal}"`;
		case "importance":
			if (hasOld) return `changed priority from ${oldVal} to ${newVal}`;
			return `set priority to ${newVal}`;
		case "assignee": {
			const oldName = resolveMember(change.old);
			const newName = resolveMember(change.new);
			if (hasOld && hasNew)
				return `changed assignee from ${oldName} to ${newName}`;
			if (hasNew) return `assigned to ${newName}`;
			return `removed assignee ${oldName}`;
		}
		case "reporter": {
			const oldName = resolveMember(change.old);
			const newName = resolveMember(change.new);
			if (hasOld && hasNew)
				return `changed reporter from ${oldName} to ${newName}`;
			if (hasNew) return `set reporter to ${newName}`;
			return `removed reporter ${oldName}`;
		}
		case "sprint": {
			const oldSprint = resolveSprint(change.old);
			const newSprint = resolveSprint(change.new);
			if (hasOld && hasNew)
				return `moved from sprint "${oldSprint}" to "${newSprint}"`;
			if (hasNew) return `added to sprint "${newSprint}"`;
			return `removed from sprint "${oldSprint}"`;
		}
		case "parent_task":
			if (hasNew) return `set parent task`;
			return `removed parent task`;
		case "due_date":
			if (hasOld && hasNew)
				return `changed due date from ${oldVal} to ${newVal}`;
			if (hasNew) return `set due date to ${newVal}`;
			return `removed due date`;
		case "start_date":
			if (hasOld && hasNew)
				return `changed start date from ${oldVal} to ${newVal}`;
			if (hasNew) return `set start date to ${newVal}`;
			return `removed start date`;
		case "description":
			return `updated description`;
		case "tags":
			return `updated tags`;
		case "custom_fields":
			return `updated custom fields`;
		default:
			return `updated ${change.field.replace(/_/g, " ")}`;
	}
}

function activityDescription(
	entry: ActivityEntry,
	names: ActivityNameMaps,
): string {
	const c = entry.content ?? {};
	switch (entry.activity_type) {
		case "task.created":
			return "created this task";
		case "task.deleted":
			return "deleted this task";
		case "task.updated": {
			const changes = c.changes as FieldChange[] | undefined;
			if (changes && changes.length === 1) {
				return describeTaskChange(changes[0], names);
			}
			if (changes && changes.length > 1) {
				return changes.map((ch) => describeTaskChange(ch, names)).join("; ");
			}
			return "updated this task";
		}
		case "task.attachment.added":
			return `added attachment${c.file_name ? `: ${c.file_name}` : ""}`;
		case "task.attachment.removed":
			return `removed attachment${c.file_name ? `: ${c.file_name}` : ""}`;
		default:
			return (c._description as string | undefined) ?? "made a change";
	}
}

export function ActivityItem({
	entry,
	names = { members: {}, sprints: {} },
}: {
	entry: ActivityEntry;
	names?: ActivityNameMaps;
}) {
	const isComment = entry.activity_type === "comment";
	const displayName = entry.actor_name || entry.actor_username || "System";
	const initial = displayName.slice(0, 1).toUpperCase();

	return (
		<div className="flex gap-3">
			<div
				className={cn(
					"flex size-6 shrink-0 items-center justify-center rounded-full text-[10px] font-bold mt-0.5 ring-1",
					isComment
						? "bg-linear-to-br from-primary/20 to-primary/10 text-primary ring-primary/15"
						: "bg-muted/40 text-muted-foreground/80 ring-border/20",
				)}
			>
				{initial}
			</div>
			<div className="flex-1 min-w-0">
				{isComment ? (
					<div className="rounded-xl rounded-tl-lg border border-border/25 bg-card/70 px-3.5 py-2.5">
						<div className="mb-1 flex items-center gap-2">
							<span className="text-[12px] font-semibold text-foreground">
								{displayName}
							</span>
							<span className="text-[10px] text-muted-foreground/50">
								{timeAgo(entry.created_at)}
							</span>
						</div>
						<p className="text-[13px] text-foreground leading-relaxed">
							{(entry.content as { text?: string }).text ?? ""}
						</p>
					</div>
				) : (
					<div className="flex flex-wrap items-baseline gap-1.5 py-0.5">
						<span className="text-[12px] font-medium text-foreground/80">
							{displayName}
						</span>
						<span className="text-[12px] text-muted-foreground/70">
							{activityDescription(entry, names)}
						</span>
						<span className="text-[10px] text-muted-foreground/45">
							{timeAgo(entry.created_at)}
						</span>
					</div>
				)}
			</div>
		</div>
	);
}
