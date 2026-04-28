import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	AlertCircle,
	Check,
	ChevronRight,
	History,
	MessageSquare,
	PanelRight,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { DocActivityPane } from "@/components/projects/docs/doc-activity-pane";
import {
	DocEditor,
	type DocEditorHandle,
} from "@/components/projects/docs/doc-editor";
import { DocHistoryPanel } from "@/components/projects/docs/doc-history-panel";
import { Button } from "@/components/ui/button";
import { useDebouncedCallback } from "@/hooks/use-debounced-callback";
import { useProjectPermissions } from "@/hooks/use-project-permissions";
import { currentUserQueryOptions } from "@/lib/auth-api";
import {
	docFoldersQueryOptions,
	docQueryKeys,
	docQueryOptions,
	updateDocument,
} from "@/lib/doc-api";
import { cn } from "@/lib/utils";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/docs/$docId",
)({
	component: DocEditorPage,
});

const TITLE_CLASSES =
	"font-[Syne] text-xl lg:text-[26px] font-bold leading-snug text-foreground tracking-tight w-full";

type RightPanel = "activity" | "history" | null;

type ProjectMemberMeResponse = {
	id: string;
};

function DocEditorPage() {
	const { projectId, docId } = Route.useParams();
	const { hasProjectPermission } = useProjectPermissions(projectId);
	const canWrite = hasProjectPermission("docs.write");
	const qc = useQueryClient();

	const { data: currentUser } = useQuery(currentUserQueryOptions);
	const currentAuthenticatedUserId = currentUser?.id;
	const { data: currentProjectMember } = useQuery<ProjectMemberMeResponse>({
		queryKey: ["projects", projectId, "members", "me"],
		enabled: !!projectId && !!currentAuthenticatedUserId,
		queryFn: async () => {
			const response = await fetch(`/api/projects/${projectId}/members/me`);
			if (!response.ok) {
				throw new Error("Failed to load current project member");
			}
			return (await response.json()) as ProjectMemberMeResponse;
		},
	});
	const currentUserId = currentProjectMember?.id;

	const { data: doc, isError } = useQuery(docQueryOptions(projectId, docId));
	const { data: allFolders = [] } = useQuery(docFoldersQueryOptions(projectId));

	const [rightPanel, setRightPanel] = useState<RightPanel>(null);
	const [editingTitle, setEditingTitle] = useState(false);
	const [titleDraft, setTitleDraft] = useState("");
	const [dirty, setDirty] = useState(false);
	const titleInputRef = useRef<HTMLTextAreaElement>(null);
	const editorRef = useRef<DocEditorHandle>(null);

	const updateMutation = useMutation({
		mutationFn: (payload: { title?: string; content?: unknown[] | null }) =>
			updateDocument(projectId, docId, payload),
		onSuccess: (updated) => {
			qc.setQueryData(docQueryKeys.detail(projectId, docId), updated);
			qc.invalidateQueries({
				queryKey: docQueryKeys.snapshots(projectId, docId),
			});
			qc.invalidateQueries({ queryKey: docQueryKeys.list(projectId) });
			if (updated.folder_id) {
				qc.invalidateQueries({
					queryKey: docQueryKeys.list(projectId, updated.folder_id),
				});
			}
		},
	});

	const commitTitle = useCallback(
		(value: string) => {
			const trimmed = (value ?? "").trim();
			if (!trimmed || trimmed === doc?.title) {
				setEditingTitle(false);
				setTitleDraft("");
				return;
			}
			updateMutation.mutate({ title: trimmed });
			setEditingTitle(false);
			setTitleDraft("");
		},
		[doc?.title, updateMutation],
	);

	useEffect(() => {
		const handler = (e: KeyboardEvent) => {
			if ((e.ctrlKey || e.metaKey) && e.key === "s") {
				e.preventDefault();
				if (editingTitle) commitTitle(titleDraft);
				editorRef.current?.save();
			}
		};
		window.addEventListener("keydown", handler);
		return () => window.removeEventListener("keydown", handler);
	}, [editingTitle, titleDraft, commitTitle]);

	const debouncedCommitTitle = useDebouncedCallback(
		(value: unknown) => commitTitle(value as string),
		5000,
	);

	const handleTitleChange = useCallback(
		(e: React.ChangeEvent<HTMLTextAreaElement>) => {
			const value = e.target.value;
			setTitleDraft(value);
			setDirty(true);
			debouncedCommitTitle(value);
		},
		[debouncedCommitTitle],
	);

	const handleContentSave = useCallback(
		(blocks: unknown[] | null) => {
			setDirty(false);
			updateMutation.mutate({ content: blocks });
		},
		[updateMutation],
	);

	const handleEditorDirtyChange = useCallback((isDirty: boolean) => {
		setDirty(isDirty);
	}, []);

	const togglePanel = (panel: RightPanel) => {
		setRightPanel((prev) => (prev === panel ? null : panel));
	};

	// Compute folder breadcrumb for this document
	const folderPath = (() => {
		if (!doc?.folder_id) return [];
		const path: typeof allFolders = [];
		let current: string | null = doc.folder_id;
		while (current) {
			const folder = allFolders.find((f) => f.id === current);
			if (!folder) break;
			path.unshift(folder);
			current = folder.parent_id ?? null;
		}
		return path;
	})();

	if (isError) {
		return (
			<div className="flex flex-1 flex-col items-center justify-center gap-4 text-muted-foreground">
				<div className="flex size-14 items-center justify-center rounded-xl bg-muted/50">
					<AlertCircle className="size-7 text-muted-foreground/60" />
				</div>
				<div className="text-center">
					<p className="text-base font-semibold text-foreground/80">
						Document not found
					</p>
					<p className="text-sm mt-1.5 text-muted-foreground/70">
						This document may have been deleted or the link is invalid.
					</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex flex-col h-full min-h-0">
			{/* ── Header bar ───────────────────────────────────────────────── */}
			<div className="flex items-center justify-between px-4 py-2 bg-muted/20 border-b border-border/30 shrink-0 gap-3 min-w-0">
				{/* Breadcrumb path */}
				<div className="flex items-center gap-1 min-w-0 text-[12px]">
					{folderPath.map((folder) => (
						<span key={folder.id} className="flex items-center gap-1 min-w-0">
							<span className="text-muted-foreground/50 truncate max-w-32">
								{folder.name}
							</span>
							<ChevronRight className="size-3.5 text-muted-foreground/30 shrink-0" />
						</span>
					))}
					{doc?.title && (
						<span className="text-foreground/70 font-medium truncate max-w-60">
							{doc.title}
						</span>
					)}
				</div>

				{/* Right: save status + panel toggles */}
				<div className="flex items-center gap-2 shrink-0">
					{dirty && !updateMutation.isPending && (
						<span className="text-[11px] text-muted-foreground/50">
							Unsaved
						</span>
					)}
					{updateMutation.isPending && (
						<span className="text-[11px] text-muted-foreground/60">
							Saving…
						</span>
					)}
					{!dirty && updateMutation.isSuccess && !updateMutation.isPending && (
						<span className="text-[11px] text-muted-foreground/60 flex items-center gap-1">
							<Check className="size-3 text-emerald-500" />
							Saved
						</span>
					)}

					<div className="flex items-center gap-0.5">
						<Button
							variant="ghost"
							size="icon"
							className={cn(
								"size-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 transition-all duration-150",
								rightPanel === "activity" && "bg-muted/40 text-foreground",
							)}
							title="Comments & activity"
							onClick={() => togglePanel("activity")}
						>
							<MessageSquare className="size-3.5" />
						</Button>
						<Button
							variant="ghost"
							size="icon"
							className={cn(
								"size-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 transition-all duration-150",
								rightPanel === "history" && "bg-muted/40 text-foreground",
							)}
							title="Version history"
							onClick={() => togglePanel("history")}
						>
							<History className="size-3.5" />
						</Button>
						{rightPanel !== null && (
							<Button
								variant="ghost"
								size="icon"
								className="size-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60 transition-all duration-150"
								title="Close panel"
								onClick={() => setRightPanel(null)}
							>
								<PanelRight className="size-3.5" />
							</Button>
						)}
					</div>
				</div>
			</div>

			{/* ── Body: editor + optional right panel ──────────────────────── */}
			<div className="flex flex-1 min-h-0 overflow-hidden">
				{/* Editor area */}
				<div className="flex-1 overflow-y-auto [scrollbar-gutter:stable] [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-border/60 [&::-webkit-scrollbar-thumb]:hover:bg-border">
					<div className="max-w-7xl mx-auto px-8 py-7 space-y-6">
						{/* Title — inline edit pattern */}
						{editingTitle ? (
							<textarea
								ref={titleInputRef}
								value={titleDraft}
								onChange={handleTitleChange}
								onBlur={() => commitTitle(titleDraft)}
								onKeyDown={(e) => {
									if (e.key === "Enter" && !e.shiftKey) {
										e.preventDefault();
										e.currentTarget.blur();
									}
									if (e.key === "Escape") {
										setEditingTitle(false);
										setTitleDraft("");
									}
								}}
								rows={1}
								className={cn(
									TITLE_CLASSES,
									"resize-none bg-transparent outline-none py-0",
								)}
							/>
						) : (
							<h1
								className={cn(
									TITLE_CLASSES,
									canWrite &&
										"cursor-text hover:bg-muted/15 rounded-md px-2 -ml-2 py-1 transition-all duration-150",
								)}
								onClick={() => {
									if (!canWrite || !doc) return;
									setTitleDraft(doc.title || "");
									setEditingTitle(true);
									setTimeout(() => titleInputRef.current?.focus(), 0);
								}}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										if (!canWrite || !doc) return;
										setTitleDraft(doc.title || "");
										setEditingTitle(true);
										setTimeout(() => titleInputRef.current?.focus(), 0);
									}
								}}
							>
								{doc?.title || (
									<span className="text-muted-foreground/30 italic">
										Untitled
									</span>
								)}
							</h1>
						)}

						{/* BlockNote editor */}
						{doc && (
							<DocEditor
								ref={editorRef}
								content={doc.content}
								editable={canWrite}
								onSave={handleContentSave}
								onDirtyChange={handleEditorDirtyChange}
								projectId={projectId}
								docId={docId}
							/>
						)}

						{!doc && (
							<div className="h-40 flex items-center justify-center">
								<span className="text-muted-foreground/50 text-sm animate-pulse">
									Loading…
								</span>
							</div>
						)}

						{/* Bottom breathing room */}
						<div className="h-8" />
					</div>
				</div>

				{/* Right panel: activity */}
				{rightPanel === "activity" && doc && (
					<div className="w-80 shrink-0 h-full overflow-hidden">
						<DocActivityPane
							projectId={projectId}
							docId={docId}
							currentUserId={currentUserId}
						/>
					</div>
				)}

				{/* Right panel: history */}
				{rightPanel === "history" && doc && (
					<div className="w-120 shrink-0 h-full overflow-hidden">
						<DocHistoryPanel
							projectId={projectId}
							docId={docId}
							onClose={() => setRightPanel(null)}
						/>
					</div>
				)}
			</div>
		</div>
	);
}
