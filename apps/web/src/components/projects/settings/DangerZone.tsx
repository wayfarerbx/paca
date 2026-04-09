import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { AlertTriangle, Loader2, Trash2 } from "lucide-react";
import { useState } from "react";
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
import { deleteProject, projectQueryOptions } from "@/lib/project-api";

export function DangerZone({ projectId }: { projectId: string }) {
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const { data: project } = useQuery(projectQueryOptions(projectId));
	const [open, setOpen] = useState(false);
	const [confirmName, setConfirmName] = useState("");

	const deleteMutation = useMutation({
		mutationFn: () => deleteProject(projectId),
		onSuccess: async () => {
			queryClient.removeQueries({ queryKey: ["projects", projectId] });
			await navigate({ to: "/home" });
			await queryClient.invalidateQueries({ queryKey: ["projects"] });
		},
	});

	return (
		<div className="rounded-xl border border-destructive/30 bg-destructive/3 p-6">
			<h3 className="font-[Syne] text-base font-semibold text-destructive mb-4">
				Danger Zone
			</h3>
			<div className="flex items-center justify-between">
				<div>
					<p className="text-sm font-medium">Delete this project</p>
					<p className="text-xs text-muted-foreground mt-0.5">
						Permanently delete the project and all its data. This action cannot
						be undone.
					</p>
				</div>
				<Button
					variant="destructive"
					size="sm"
					className="shrink-0 ml-4 gap-1.5"
					onClick={() => setOpen(true)}
				>
					<Trash2 className="size-3.5" />
					Delete project
				</Button>
			</div>

			<Dialog
				open={open}
				onOpenChange={(o) => {
					setOpen(o);
					setConfirmName("");
				}}
			>
				<DialogContent className="sm:max-w-sm">
					<DialogHeader>
						<div className="flex size-10 items-center justify-center rounded-full bg-destructive/10 mb-2">
							<AlertTriangle className="size-5 text-destructive" />
						</div>
						<DialogTitle>Delete project</DialogTitle>
						<DialogDescription>
							This will permanently delete{" "}
							<span className="font-semibold text-foreground">
								{project?.name}
							</span>{" "}
							and all its data, including members, roles, and integrations. This
							action cannot be undone.
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-1.5">
						<Label
							htmlFor="confirm-name"
							className="text-xs text-muted-foreground"
						>
							Type{" "}
							<span className="font-semibold text-foreground">
								{project?.name}
							</span>{" "}
							to confirm
						</Label>
						<Input
							id="confirm-name"
							value={confirmName}
							onChange={(e) => setConfirmName(e.target.value)}
							placeholder={project?.name}
							autoComplete="off"
						/>
					</div>
					{deleteMutation.isError ? (
						<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
							Failed to delete project. Please try again.
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
							disabled={
								confirmName !== project?.name || deleteMutation.isPending
							}
							onClick={() => deleteMutation.mutate()}
						>
							{deleteMutation.isPending ? (
								<Loader2 className="size-3.5 animate-spin" />
							) : (
								<Trash2 className="size-3.5" />
							)}
							Delete permanently
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
