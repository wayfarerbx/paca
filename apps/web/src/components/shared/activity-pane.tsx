import {
	type QueryKey,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import {
	Bold,
	Hash,
	Italic,
	List,
	MessageSquare,
	MoreHorizontal,
	Paperclip,
	Pencil,
	Send,
	Smile,
	Trash2,
} from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";

export interface ActivityEntry {
	id: string;
	actor_id?: string | null;
	actor_name: string;
	actor_username: string;
	activity_type: string;
	content: Record<string, unknown> | string | null;
	created_at: string;
	updated_at: string;
}

export interface ActivityPaneConfig<T extends ActivityEntry> {
	projectId: string;
	entityId: string;
	queryKey: QueryKey;
	queryFn: () => Promise<T[]>;
	addComment?: (text: string) => Promise<unknown>;
	updateComment?: (commentId: string, text: string) => Promise<unknown>;
	deleteComment?: (commentId: string) => Promise<void>;
	describeActivity: (entry: T) => string;
	getCommentText: (content: T["content"]) => string;
	currentUserId?: string;
	sortAscending?: boolean;
	nameMaps?: Record<string, Record<string, string>>;
}

export function ActivityPane<T extends ActivityEntry>({
	queryKey,
	queryFn,
	addComment,
	updateComment,
	deleteComment,
	describeActivity,
	getCommentText,
	currentUserId,
	sortAscending = false,
}: ActivityPaneConfig<T>) {
	const [comment, setComment] = useState("");
	const [commentFocused, setCommentFocused] = useState(false);
	const qc = useQueryClient();

	const { data: activities = [] } = useQuery({
		queryKey,
		queryFn,
	});

	const sorted = useMemo(() => {
		if (!sortAscending) return activities;
		return [...activities].sort(
			(a, b) =>
				new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
		);
	}, [activities, sortAscending]);

	const addMutation = useMutation({
		mutationFn: (text: string) => {
			if (!addComment) return Promise.resolve();
			return addComment(text);
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey });
		},
	});

	const handleSend = () => {
		const text = comment.trim();
		if (!text) return;
		addMutation.mutate(text);
		setComment("");
		setCommentFocused(false);
	};

	return (
		<div className="flex w-full lg:w-80 lg:shrink-0 flex-col h-full lg:overflow-hidden border-t lg:border-t-0 lg:border-l border-border/25 bg-muted/10">
			<div className="flex shrink-0 items-center gap-2.5 border-b border-border/25 px-5 py-3 bg-muted/20">
				<MessageSquare className="size-3.5 text-muted-foreground/70" />
				<span className="text-[12px] font-semibold text-muted-foreground uppercase tracking-wide">
					Activity
				</span>
				{sorted.length > 0 && (
					<span className="ml-auto rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold text-muted-foreground/70 tabular-nums">
						{sorted.length}
					</span>
				)}
			</div>

			<ScrollArea className="lg:flex-1 lg:min-h-0 px-4 py-4">
				<div className="space-y-3">
					{sorted.length === 0 && (
						<div className="flex flex-col items-center py-8 text-muted-foreground/40">
							<MessageSquare className="size-6 mb-2" />
							<p className="text-[12px] font-medium">No activity yet</p>
						</div>
					)}
					{sorted.map((entry) => (
						<ActivityItemInner
							key={entry.id}
							entry={entry}
							describeActivity={describeActivity}
							getCommentText={getCommentText}
							updateComment={updateComment}
							deleteComment={deleteComment}
							queryKey={queryKey}
							currentUserId={currentUserId}
						/>
					))}
				</div>
			</ScrollArea>

			{addComment && (
				<div className="shrink-0 border-t border-border/25 p-3 space-y-1.5 bg-background/50">
					{commentFocused && (
						<div className="flex items-center gap-0.5 rounded-lg border border-border/25 bg-muted/25 px-2 py-1">
							{[
								{ icon: Bold, title: "Bold" },
								{ icon: Italic, title: "Italic" },
								{ icon: List, title: "List" },
							].map(({ icon: Icon, title }) => (
								<button
									key={title}
									type="button"
									title={title}
									className="flex size-6 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/50 transition-all duration-150"
								>
									<Icon className="size-3" />
								</button>
							))}
							<div className="mx-1 h-3.5 w-px bg-border/30" />
							{[
								{ icon: Smile, title: "Emoji" },
								{ icon: Paperclip, title: "Attach" },
								{ icon: Hash, title: "Mention" },
							].map(({ icon: Icon, title }) => (
								<button
									key={title}
									type="button"
									title={title}
									className="flex size-6 items-center justify-center rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/50 transition-all duration-150"
								>
									<Icon className="size-3" />
								</button>
							))}
						</div>
					)}

					<div
						className={cn(
							"flex items-end gap-2 rounded-xl border border-border/30 bg-card/80 px-3 py-2.5 transition-all duration-200",
							commentFocused && "border-primary/25 shadow-sm shadow-primary/5",
						)}
					>
						<textarea
							value={comment}
							onChange={(e) => setComment(e.target.value)}
							onFocus={() => setCommentFocused(true)}
							onBlur={() => !comment && setCommentFocused(false)}
							placeholder="Write a comment…"
							rows={commentFocused ? 3 : 1}
							className="flex-1 resize-none bg-transparent text-[13px] outline-none placeholder:text-muted-foreground/50 leading-relaxed"
							onKeyDown={(e) => {
								if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
									handleSend();
								}
							}}
						/>
						<button
							type="button"
							onClick={handleSend}
							disabled={!comment.trim() || addMutation.isPending}
							className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground disabled:opacity-40 hover:bg-primary/90 transition-all duration-150 shadow-sm disabled:shadow-none"
						>
							<Send className="size-3" />
						</button>
					</div>
					{commentFocused && (
						<p className="text-[10px] text-muted-foreground/40 text-right pr-1">
							⌘↵ to send
						</p>
					)}
				</div>
			)}
		</div>
	);
}

function timeAgo(iso: string): string {
	const diff = Date.now() - new Date(iso).getTime();
	const mins = Math.floor(diff / 60000);
	if (mins < 1) return "just now";
	if (mins < 60) return `${mins}m ago`;
	const hrs = Math.floor(mins / 60);
	if (hrs < 24) return `${hrs}h ago`;
	return `${Math.floor(hrs / 24)}d ago`;
}

interface ActivityItemInnerProps<T extends ActivityEntry> {
	entry: T;
	describeActivity: (entry: T) => string;
	getCommentText: (content: T["content"]) => string;
	updateComment?: (commentId: string, text: string) => Promise<unknown>;
	deleteComment?: (commentId: string) => Promise<void>;
	queryKey: QueryKey;
	currentUserId?: string;
}

function ActivityItemInner<T extends ActivityEntry>({
	entry,
	describeActivity,
	getCommentText,
	updateComment,
	deleteComment,
	queryKey,
	currentUserId,
}: ActivityItemInnerProps<T>) {
	const qc = useQueryClient();
	const [editing, setEditing] = useState(false);
	const commentText = getCommentText(entry.content);
	const [editText, setEditText] = useState(commentText);

	const isComment = entry.activity_type === "comment";
	const isOwn = entry.actor_id === currentUserId;
	const displayName = entry.actor_name || entry.actor_username || "System";
	const initial = displayName.slice(0, 1).toUpperCase();

	const canEdit = isComment && isOwn && !!updateComment;
	const canDelete = isComment && isOwn && !!deleteComment;

	const updateMutation = useMutation({
		// biome-ignore lint/style/noNonNullAssertion: guarded by canEdit
		mutationFn: (text: string) => updateComment!(entry.id, text),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey });
			setEditing(false);
		},
	});

	const deleteMutation = useMutation({
		// biome-ignore lint/style/noNonNullAssertion: guarded by canDelete
		mutationFn: () => deleteComment!(entry.id),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey });
		},
	});

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
					<div className="group rounded-xl rounded-tl-lg border border-border/25 bg-card/70 px-3.5 py-2.5">
						<div className="mb-1 flex items-center gap-2">
							<span className="text-[12px] font-semibold text-foreground">
								{displayName}
							</span>
							<span className="text-[10px] text-muted-foreground/50">
								{timeAgo(entry.created_at)}
							</span>
							{canEdit && canDelete && (
								<DropdownMenu>
									<DropdownMenuTrigger className="inline-flex items-center justify-center ml-auto size-5 rounded-md text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 opacity-0 group-hover:opacity-100 transition-all duration-150">
										<MoreHorizontal className="size-3" />
									</DropdownMenuTrigger>
									<DropdownMenuContent align="end" className="w-36">
										<DropdownMenuItem onClick={() => setEditing(true)}>
											<Pencil className="size-3.5 mr-2" />
											Edit
										</DropdownMenuItem>
										<DropdownMenuItem
											className="text-destructive focus:text-destructive"
											onClick={() => deleteMutation.mutate()}
										>
											<Trash2 className="size-3.5 mr-2" />
											Delete
										</DropdownMenuItem>
									</DropdownMenuContent>
								</DropdownMenu>
							)}
						</div>

						{editing ? (
							<div className="space-y-1.5 mt-1">
								<textarea
									value={editText}
									onChange={(e) => setEditText(e.target.value)}
									className="w-full rounded-lg border border-border/30 bg-muted/15 px-3 py-2 text-[13px] outline-none focus:border-primary/40 focus:ring-2 focus:ring-primary/15 resize-none min-h-16 placeholder:text-muted-foreground/50 leading-relaxed transition-all duration-150"
								/>
								<div className="flex gap-1.5">
									<Button
										size="sm"
										className="h-6 text-[11px] gap-1 rounded-md"
										onClick={() => updateMutation.mutate(editText)}
										disabled={!editText.trim()}
									>
										Save
									</Button>
									<Button
										variant="ghost"
										size="sm"
										className="h-6 text-[11px] rounded-md"
										onClick={() => {
											setEditing(false);
											setEditText(commentText);
										}}
									>
										Cancel
									</Button>
								</div>
							</div>
						) : (
							<p className="text-[13px] text-foreground leading-relaxed">
								{commentText}
							</p>
						)}
					</div>
				) : (
					<div className="flex flex-wrap items-baseline gap-1.5 py-0.5">
						<span className="text-[12px] font-medium text-foreground/80">
							{displayName}
						</span>
						<span className="text-[12px] text-muted-foreground/70">
							{describeActivity(entry)}
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
