import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ApiErrorCode, getApiErrorCode } from "@/lib/api-error";
import { projectQueryOptions, updateProject } from "@/lib/project-api";

export function GeneralSettings({
	projectId,
	canEdit,
}: {
	projectId: string;
	canEdit: boolean;
}) {
	const queryClient = useQueryClient();
	const { data: project } = useQuery(projectQueryOptions(projectId));

	const [name, setName] = useState(project?.name ?? "");
	const [description, setDescription] = useState(project?.description ?? "");
	const [nameError, setNameError] = useState<string | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [saved, setSaved] = useState(false);

	const mutation = useMutation({
		mutationFn: () =>
			updateProject(projectId, {
				name: name.trim(),
				description: description.trim(),
			}),
		onSuccess: async (updated) => {
			await queryClient.invalidateQueries({
				queryKey: projectQueryOptions(projectId).queryKey,
			});
			// Also update the projects list cache
			await queryClient.invalidateQueries({ queryKey: ["projects"] });
			setName(updated.name);
			setDescription(updated.description);
			setError(null);
			setNameError(null);
			setSaved(true);
			setTimeout(() => setSaved(false), 2500);
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.ProjectNameTaken) {
				setNameError("A project with this name already exists.");
				return;
			}
			if (code === ApiErrorCode.ProjectNameInvalid) {
				setNameError("Project name is empty or invalid.");
				return;
			}
			setError("Failed to update project. Please try again.");
		},
	});

	const isDirty =
		name.trim() !== (project?.name ?? "") ||
		description.trim() !== (project?.description ?? "");

	return (
		<div className="rounded-xl border border-border/60 bg-card p-6">
			<h3 className="font-[Syne] text-base font-semibold mb-4">General</h3>
			<div className="space-y-4 max-w-md">
				<div className="space-y-1.5">
					<Label htmlFor="project-name">Project name</Label>
					<Input
						id="project-name"
						value={name}
						onChange={(e) => {
							setName(e.target.value);
							setNameError(null);
						}}
						placeholder="My awesome project"
						disabled={!canEdit}
						className={
							nameError
								? "border-destructive focus-visible:ring-destructive/30"
								: ""
						}
					/>
					{nameError ? (
						<p className="text-xs text-destructive">{nameError}</p>
					) : null}
				</div>

				<div className="space-y-1.5">
					<Label htmlFor="project-description">Description</Label>
					<Textarea
						id="project-description"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
						placeholder="Describe what this project is about…"
						rows={3}
						disabled={!canEdit}
						className="resize-none"
					/>
				</div>

				{error ? (
					<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
						{error}
					</p>
				) : null}

				{canEdit ? (
					<div className="flex items-center gap-2 pt-1">
						<Button
							size="sm"
							disabled={!isDirty || mutation.isPending}
							onClick={() => mutation.mutate()}
							className="gap-1.5"
						>
							{mutation.isPending ? (
								<Loader2 className="size-3.5 animate-spin" />
							) : null}
							Save changes
						</Button>
						{saved ? (
							<span className="text-xs text-emerald-600 dark:text-emerald-400 font-medium">
								Saved ✓
							</span>
						) : null}
					</div>
				) : (
					<p className="text-xs text-muted-foreground">
						You don't have permission to edit this project.
					</p>
				)}
			</div>
		</div>
	);
}
