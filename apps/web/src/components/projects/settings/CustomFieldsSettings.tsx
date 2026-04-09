import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Edit2, Loader2, Plus, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";
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
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { ApiErrorCode, getApiErrorCode } from "@/lib/api-error";
import {
	createCustomFieldDefinition,
	customFieldsQueryOptions,
	type CustomFieldDefinition,
	deleteCustomFieldDefinition,
	type FieldType,
	updateCustomFieldDefinition,
} from "@/lib/project-api";
import { cn } from "@/lib/utils";

// ── Field type utilities ──────────────────────────────────────────────────────

const UI_FIELD_TYPES = ["Text", "Number", "Date", "Checkbox", "Select"] as const;
type UIFieldType = (typeof UI_FIELD_TYPES)[number];

const UI_TO_API_FIELD_TYPE: Record<UIFieldType, FieldType> = {
	Text: "text",
	Number: "number",
	Date: "date",
	Checkbox: "boolean",
	Select: "select",
};

function toUIFieldType(apiType: string): string {
	const map: Record<string, string> = {
		text: "Text",
		number: "Number",
		date: "Date",
		boolean: "Checkbox",
		select: "Select",
		multi_select: "Multi-Select",
		url: "URL",
	};
	return map[apiType] ?? apiType;
}

function slugify(s: string): string {
	return s
		.toLowerCase()
		.replace(/\s+/g, "_")
		.replace(/[^a-z0-9_]/g, "")
		.replace(/_+/g, "_")
		.slice(0, 64);
}

// ── Create dialog ─────────────────────────────────────────────────────────────

function CreateCustomFieldDialog({
	projectId,
	open,
	onOpenChange,
}: {
	projectId: string;
	open: boolean;
	onOpenChange: (v: boolean) => void;
}) {
	const queryClient = useQueryClient();
	const [displayName, setDisplayName] = useState("");
	const [fieldKey, setFieldKey] = useState("");
	const [keyManuallyEdited, setKeyManuallyEdited] = useState(false);
	const [fieldType, setFieldType] = useState<UIFieldType>("Text");
	const [required, setRequired] = useState(false);
	const [options, setOptions] = useState<string[]>([]);
	const [newOption, setNewOption] = useState("");
	const [error, setError] = useState<string | null>(null);

	const reset = () => {
		setDisplayName("");
		setFieldKey("");
		setKeyManuallyEdited(false);
		setFieldType("Text");
		setRequired(false);
		setOptions([]);
		setNewOption("");
		setError(null);
	};

	const handleDisplayName = (v: string) => {
		setDisplayName(v);
		if (!keyManuallyEdited) setFieldKey(slugify(v));
	};

	const mutation = useMutation({
		mutationFn: () =>
			createCustomFieldDefinition(projectId, {
				display_name: displayName.trim(),
				field_key: fieldKey || slugify(displayName),
				field_type: UI_TO_API_FIELD_TYPE[fieldType],
				options: fieldType === "Select" ? options : [],
				is_required: required,
			}),
		onSuccess: () => {
			void queryClient.invalidateQueries({
				queryKey: customFieldsQueryOptions(projectId).queryKey,
			});
			reset();
			onOpenChange(false);
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.CustomFieldKeyTaken) {
				setError("A field with this key already exists in the project.");
				return;
			}
			if (code === ApiErrorCode.CustomFieldKeyInvalid) {
				setError("Field key is empty or invalid.");
				return;
			}
			if (code === ApiErrorCode.CustomFieldNameInvalid) {
				setError("Display name is empty or invalid.");
				return;
			}
			setError("Failed to create field. Please try again.");
		},
	});

	return (
		<Dialog
			open={open}
			onOpenChange={(o) => {
				if (!o) reset();
				onOpenChange(o);
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>Create custom field</DialogTitle>
					<DialogDescription>
						Define a new field to capture additional data on tasks in this
						project.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					{/* Display name */}
					<div className="space-y-1.5">
						<Label htmlFor="cf-display-name">
							Display name <span className="text-destructive">*</span>
						</Label>
						<Input
							id="cf-display-name"
							value={displayName}
							onChange={(e) => handleDisplayName(e.target.value)}
							placeholder="e.g. Release Tag"
							autoFocus
						/>
					</div>

					{/* Field key */}
					<div className="space-y-1.5">
						<Label htmlFor="cf-field-key">Field key</Label>
						<Input
							id="cf-field-key"
							value={fieldKey}
							onChange={(e) => {
								setKeyManuallyEdited(true);
								setFieldKey(slugify(e.target.value));
							}}
							placeholder="release_tag"
							className="font-mono text-sm"
						/>
						<p className="text-[10px] text-muted-foreground/60">
							Used as the identifier in the API and data exports.
						</p>
					</div>

					{/* Field type */}
					<div className="space-y-1.5">
						<Label>Field type</Label>
						<div className="flex flex-wrap gap-1.5">
							{UI_FIELD_TYPES.map((ft) => (
								<button
									key={ft}
									type="button"
									onClick={() => setFieldType(ft)}
									className={cn(
										"rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors",
										fieldType === ft
											? "border-primary bg-primary/10 text-primary"
											: "border-border/60 text-muted-foreground hover:border-border hover:bg-muted/50",
									)}
								>
									{ft}
								</button>
							))}
						</div>
					</div>

					{/* Options editor — only for Select type */}
					{fieldType === "Select" && (
						<div className="space-y-1.5">
							<Label>Options</Label>
							<div className="space-y-1">
								{options.map((opt, i) => (
									<div
										key={opt + i.toString()}
										className="flex items-center gap-2"
									>
										<Input
											value={opt}
											onChange={(e) => {
												const updated = [...options];
												updated[i] = e.target.value;
												setOptions(updated);
											}}
											className="text-xs h-8"
										/>
										<button
											type="button"
											onClick={() =>
												setOptions(options.filter((_, j) => j !== i))
											}
											className="shrink-0 text-muted-foreground hover:text-destructive transition-colors"
										>
											<X className="size-3.5" />
										</button>
									</div>
								))}
								<div className="flex gap-2">
									<Input
										value={newOption}
										onChange={(e) => setNewOption(e.target.value)}
										onKeyDown={(e) => {
											if (e.key === "Enter" && newOption.trim()) {
												setOptions([...options, newOption.trim()]);
												setNewOption("");
											}
										}}
										placeholder="Add option…"
										className="text-xs h-8"
									/>
									<button
										type="button"
										disabled={!newOption.trim()}
										onClick={() => {
											setOptions([...options, newOption.trim()]);
											setNewOption("");
										}}
										className="flex items-center gap-1 rounded-md bg-muted px-2.5 text-xs font-medium text-muted-foreground hover:bg-muted/80 disabled:opacity-40 transition-colors"
									>
										<Plus className="size-3" />
										Add
									</button>
								</div>
							</div>
						</div>
					)}

					{/* Required toggle */}
					<div className="flex items-center justify-between rounded-xl border border-border/50 bg-muted/20 px-4 py-3">
						<div>
							<p className="text-sm font-medium">Required</p>
							<p className="text-[11px] text-muted-foreground/70">
								Users must fill this field when creating or editing a task.
							</p>
						</div>
						<button
							type="button"
							role="switch"
							aria-checked={required}
							onClick={() => setRequired(!required)}
							className={cn(
								"relative inline-flex h-5 w-9 items-center rounded-full border-2 transition-colors",
								required
									? "border-primary bg-primary"
									: "border-border bg-muted",
							)}
						>
							<span
								className={cn(
									"inline-block size-3.5 rounded-full bg-white shadow transition-transform",
									required ? "translate-x-4" : "translate-x-0.5",
								)}
							/>
						</button>
					</div>

					{error ? (
						<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
							{error}
						</p>
					) : null}
				</div>

				<DialogFooter>
					<DialogClose
						render={
							<Button
								variant="outline"
								size="sm"
								disabled={mutation.isPending}
							/>
						}
					>
						Cancel
					</DialogClose>
					<Button
						size="sm"
						disabled={!displayName.trim() || mutation.isPending}
						onClick={() => mutation.mutate()}
					>
						{mutation.isPending ? (
							<Loader2 className="size-3.5 animate-spin" />
						) : null}
						Create field
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Edit dialog ───────────────────────────────────────────────────────────────

function EditCustomFieldDialog({
	projectId,
	field,
	open,
	onOpenChange,
}: {
	projectId: string;
	field: CustomFieldDefinition | null;
	open: boolean;
	onOpenChange: (v: boolean) => void;
}) {
	const queryClient = useQueryClient();
	const [displayName, setDisplayName] = useState(field?.display_name ?? "");
	const [options, setOptions] = useState<string[]>(field?.options ?? []);
	const [required, setRequired] = useState(field?.is_required ?? false);
	const [newOption, setNewOption] = useState("");
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		setDisplayName(field?.display_name ?? "");
		setOptions(field?.options ?? []);
		setRequired(field?.is_required ?? false);
		setError(null);
		setNewOption("");
	}, [field]);

	const uiFieldType = field ? toUIFieldType(field.field_type) : "";

	const mutation = useMutation({
		mutationFn: () => {
			if (!field) return Promise.resolve(field as unknown as CustomFieldDefinition);
			return updateCustomFieldDefinition(projectId, field.id, {
				display_name: displayName.trim(),
				options: field.field_type === "select" ? options : undefined,
				is_required: required,
			});
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({
				queryKey: customFieldsQueryOptions(projectId).queryKey,
			});
			onOpenChange(false);
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.CustomFieldNameInvalid) {
				setError("Display name is empty or invalid.");
				return;
			}
			setError("Failed to save changes. Please try again.");
		},
	});

	if (!field) return null;

	return (
		<Dialog
			open={open}
			onOpenChange={(o) => {
				if (!o) setError(null);
				onOpenChange(o);
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>Edit custom field</DialogTitle>
					<DialogDescription>Update the field settings.</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="space-y-1.5">
						<Label htmlFor="cf-edit-name">Display name</Label>
						<Input
							id="cf-edit-name"
							value={displayName}
							onChange={(e) => setDisplayName(e.target.value)}
							autoFocus
						/>
					</div>

					<div className="space-y-1.5">
						<Label>Field key</Label>
						<Input
							value={field.field_key}
							disabled
							className="font-mono text-sm opacity-60"
						/>
						<p className="text-[10px] text-muted-foreground/60">
							Field key cannot be changed after creation.
						</p>
					</div>

					<div className="space-y-1.5">
						<Label>Field type</Label>
						<div className="flex flex-wrap gap-1.5">
							<button
								type="button"
								disabled
								className="rounded-lg border border-primary bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary opacity-60 cursor-not-allowed"
							>
								{uiFieldType}
							</button>
						</div>
						<p className="text-[10px] text-muted-foreground/60">
							Field type cannot be changed after creation.
						</p>
					</div>

					{field.field_type === "select" && (
						<div className="space-y-1.5">
							<Label>Options</Label>
							<div className="space-y-1">
								{options.map((opt, i) => (
									<div
										key={opt + i.toString()}
										className="flex items-center gap-2"
									>
										<Input
											value={opt}
											onChange={(e) => {
												const updated = [...options];
												updated[i] = e.target.value;
												setOptions(updated);
											}}
											className="text-xs h-8"
										/>
										<button
											type="button"
											onClick={() =>
												setOptions(options.filter((_, j) => j !== i))
											}
											className="shrink-0 text-muted-foreground hover:text-destructive"
										>
											<X className="size-3.5" />
										</button>
									</div>
								))}
								<div className="flex gap-2">
									<Input
										value={newOption}
										onChange={(e) => setNewOption(e.target.value)}
										onKeyDown={(e) => {
											if (e.key === "Enter" && newOption.trim()) {
												setOptions([...options, newOption.trim()]);
												setNewOption("");
											}
										}}
										placeholder="Add option…"
										className="text-xs h-8"
									/>
									<button
										type="button"
										disabled={!newOption.trim()}
										onClick={() => {
											setOptions([...options, newOption.trim()]);
											setNewOption("");
										}}
										className="flex items-center gap-1 rounded-md bg-muted px-2.5 text-xs font-medium text-muted-foreground hover:bg-muted/80 disabled:opacity-40"
									>
										<Plus className="size-3" />
										Add
									</button>
								</div>
							</div>
						</div>
					)}

					{/* Required toggle */}
					<div className="flex items-center justify-between rounded-xl border border-border/50 bg-muted/20 px-4 py-3">
						<div>
							<p className="text-sm font-medium">Required</p>
							<p className="text-[11px] text-muted-foreground/70">
								Users must fill this field when creating or editing a task.
							</p>
						</div>
						<button
							type="button"
							role="switch"
							aria-checked={required}
							onClick={() => setRequired(!required)}
							className={cn(
								"relative inline-flex h-5 w-9 items-center rounded-full border-2 transition-colors",
								required
									? "border-primary bg-primary"
									: "border-border bg-muted",
							)}
						>
							<span
								className={cn(
									"inline-block size-3.5 rounded-full bg-white shadow transition-transform",
									required ? "translate-x-4" : "translate-x-0.5",
								)}
							/>
						</button>
					</div>

					{error ? (
						<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
							{error}
						</p>
					) : null}
				</div>

				<DialogFooter>
					<DialogClose
						render={
							<Button
								variant="outline"
								size="sm"
								disabled={mutation.isPending}
							/>
						}
					>
						Cancel
					</DialogClose>
					<Button
						size="sm"
						disabled={!displayName.trim() || mutation.isPending}
						onClick={() => mutation.mutate()}
					>
						{mutation.isPending ? (
							<Loader2 className="size-3.5 animate-spin" />
						) : null}
						Save changes
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Delete dialog ─────────────────────────────────────────────────────────────

function DeleteCustomFieldDialog({
	projectId,
	field,
	open,
	onOpenChange,
}: {
	projectId: string;
	field: CustomFieldDefinition | null;
	open: boolean;
	onOpenChange: (v: boolean) => void;
}) {
	const queryClient = useQueryClient();
	const [error, setError] = useState<string | null>(null);

	const mutation = useMutation({
		mutationFn: () => {
			if (!field) return Promise.resolve();
			return deleteCustomFieldDefinition(projectId, field.id);
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({
				queryKey: customFieldsQueryOptions(projectId).queryKey,
			});
			onOpenChange(false);
		},
		onError: (err: unknown) => {
			const code = getApiErrorCode(err);
			if (code === ApiErrorCode.CustomFieldNotFound) {
				void queryClient.invalidateQueries({
					queryKey: customFieldsQueryOptions(projectId).queryKey,
				});
				onOpenChange(false);
				return;
			}
			setError("Failed to delete field. Please try again.");
		},
	});

	if (!field) return null;

	return (
		<Dialog
			open={open}
			onOpenChange={(o) => {
				if (!o) setError(null);
				onOpenChange(o);
			}}
		>
			<DialogContent className="sm:max-w-sm">
				<DialogHeader>
					<div className="flex size-10 items-center justify-center rounded-full bg-destructive/10 mb-2">
						<Trash2 className="size-5 text-destructive" />
					</div>
					<DialogTitle>Delete custom field</DialogTitle>
					<DialogDescription>
						Delete{" "}
						<span className="font-semibold text-foreground">
							&ldquo;{field.display_name}&rdquo;
						</span>
						? Task data stored in this field will be lost. This action cannot
						be undone.
					</DialogDescription>
				</DialogHeader>

				{error ? (
					<p className="text-xs text-destructive bg-destructive/10 rounded-lg px-3 py-2">
						{error}
					</p>
				) : null}

				<DialogFooter>
					<DialogClose
						render={
							<Button
								variant="outline"
								size="sm"
								disabled={mutation.isPending}
							/>
						}
					>
						Cancel
					</DialogClose>
					<Button
						variant="destructive"
						size="sm"
						disabled={mutation.isPending}
						onClick={() => mutation.mutate()}
					>
						{mutation.isPending ? (
							<Loader2 className="size-3.5 animate-spin" />
						) : null}
						Delete field
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Main section component ────────────────────────────────────────────────────

export function CustomFieldsSettings({
	projectId,
	canWrite,
}: {
	projectId: string;
	canWrite: boolean;
}) {
	const { data: fields = [], isLoading } = useQuery(
		customFieldsQueryOptions(projectId),
	);
	const [createOpen, setCreateOpen] = useState(false);
	const [editField, setEditField] = useState<CustomFieldDefinition | null>(null);
	const [deleteField, setDeleteField] = useState<CustomFieldDefinition | null>(
		null,
	);

	return (
		<div className="rounded-xl border border-border/60 bg-card p-6">
			<div className="flex items-start justify-between mb-1">
				<div>
					<h3 className="font-[Syne] text-base font-semibold">Custom Fields</h3>
					<p className="text-xs text-muted-foreground mt-0.5 max-w-xs">
						Define project-level custom task fields that extend tasks with
						additional data specific to your workflow.
					</p>
				</div>
				{canWrite && (
					<Button
						size="sm"
						variant="outline"
						className="gap-1.5 border-border/60 shrink-0"
						onClick={() => setCreateOpen(true)}
					>
						<Plus className="size-3.5" />
						New custom field
					</Button>
				)}
			</div>

			{isLoading ? (
				<div className="rounded-xl border overflow-hidden mt-4">
					{["cf1", "cf2", "cf3"].map((k) => (
						<div
							key={k}
							className="flex items-center gap-4 border-b px-5 py-4 last:border-0"
						>
							<Skeleton className="h-4 w-36" />
							<Skeleton className="h-4 w-24 font-mono" />
							<Skeleton className="h-5 w-16 rounded-md" />
							<Skeleton className="h-4 w-8 ml-auto" />
						</div>
					))}
				</div>
			) : fields.length === 0 ? (
				<div className="mt-4 flex flex-col items-center gap-4 rounded-xl border border-dashed border-border/60 bg-muted/10 py-14 text-center">
					<div className="flex size-11 items-center justify-center rounded-xl bg-muted">
						<Plus className="size-5 text-muted-foreground/60" />
					</div>
					<div>
						<p className="text-sm font-medium">No custom fields yet</p>
						<p className="mt-1 text-xs text-muted-foreground max-w-xs mx-auto">
							Custom fields let you capture data specific to your workflow —
							sprints, severity levels, release tags, and more.
						</p>
					</div>
					{canWrite && (
						<Button
							size="sm"
							variant="outline"
							onClick={() => setCreateOpen(true)}
						>
							<Plus className="size-4 mr-1" />
							Create first field
						</Button>
					)}
				</div>
			) : (
				<div className="mt-4 overflow-x-auto rounded-xl border">
					<Table>
						<TableHeader>
							<TableRow className="bg-muted/40 hover:bg-muted/40">
								<TableHead className="px-5 text-xs font-semibold uppercase tracking-wide">
									Display Name
								</TableHead>
								<TableHead className="px-5 text-xs font-semibold uppercase tracking-wide">
									Field Key
								</TableHead>
								<TableHead className="px-5 text-xs font-semibold uppercase tracking-wide">
									Type
								</TableHead>
								<TableHead className="px-5 text-xs font-semibold uppercase tracking-wide">
									Required
								</TableHead>
								{canWrite && <TableHead className="w-20 px-5" />}
							</TableRow>
						</TableHeader>
						<TableBody>
							{fields.map((field) => (
								<TableRow key={field.id} className="group">
									<TableCell className="px-5 font-medium">
										{field.display_name}
									</TableCell>
									<TableCell className="px-5 font-mono text-xs text-muted-foreground">
										{field.field_key}
									</TableCell>
									<TableCell className="px-5">
										<span className="inline-flex items-center rounded-md border border-border/40 bg-muted/40 px-2 py-0.5 text-[10px] font-semibold text-muted-foreground">
											{toUIFieldType(field.field_type)}
										</span>
									</TableCell>
									<TableCell className="px-5">
										{field.is_required ? (
											<span className="inline-flex items-center gap-1 text-emerald-600 text-xs font-medium">
												<Check className="size-3" />
												Yes
											</span>
										) : (
											<span className="text-xs text-muted-foreground/50">
												No
											</span>
										)}
									</TableCell>
									{canWrite && (
										<TableCell className="px-5">
											<div className="flex items-center justify-end gap-0.5 opacity-100 transition-opacity sm:opacity-0 sm:group-hover:opacity-100">
												<Button
													variant="ghost"
													size="icon-sm"
													onClick={() => setEditField(field)}
													title="Edit field"
													aria-label="Edit field"
												>
													<Edit2 className="size-3.5" />
												</Button>
												<Button
													variant="ghost"
													size="icon-sm"
													className="text-destructive hover:text-destructive hover:bg-destructive/10"
													onClick={() => setDeleteField(field)}
													title="Delete field"
													aria-label="Delete field"
												>
													<Trash2 className="size-3.5" />
												</Button>
											</div>
										</TableCell>
									)}
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			<CreateCustomFieldDialog
				projectId={projectId}
				open={createOpen}
				onOpenChange={setCreateOpen}
			/>

			<EditCustomFieldDialog
				projectId={projectId}
				field={editField}
				open={!!editField}
				onOpenChange={(v) => {
					if (!v) setEditField(null);
				}}
			/>

			<DeleteCustomFieldDialog
				projectId={projectId}
				field={deleteField}
				open={!!deleteField}
				onOpenChange={(v) => {
					if (!v) setDeleteField(null);
				}}
			/>
		</div>
	);
}
