import { Paperclip, Upload } from "lucide-react";
import { useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { AttachmentItem } from "./attachment-item";
import type { Attachment } from "./types";

interface AttachmentsSectionProps {
	canEdit?: boolean;
}

export function AttachmentsSection({
	canEdit = true,
}: AttachmentsSectionProps) {
	const [attachments, setAttachments] = useState<Attachment[]>([]);
	const [isDragOver, setIsDragOver] = useState(false);
	const fileInputRef = useRef<HTMLInputElement>(null);

	const addFiles = (files: File[]) => {
		if (!canEdit) return;
		const newAttachments: Attachment[] = files.map((f) => ({
			id: crypto.randomUUID(),
			name: f.name,
			size: f.size,
			uploaded_at: new Date().toISOString(),
		}));
		setAttachments((a) => [...a, ...newAttachments]);
	};

	const handleFileDrop = (e: React.DragEvent) => {
		e.preventDefault();
		setIsDragOver(false);
		if (!canEdit) return;
		addFiles(Array.from(e.dataTransfer.files));
	};

	const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
		addFiles(Array.from(e.target.files ?? []));
		if (e.target) e.target.value = "";
	};

	const handleDelete = (id: string) => {
		setAttachments((a) => a.filter((att) => att.id !== id));
	};

	return (
		<div className="space-y-3">
			<div className="flex items-center justify-between">
				<span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground/60">
					Attachments
				</span>
				{canEdit && (
					<>
						<button
							type="button"
							onClick={() => fileInputRef.current?.click()}
							className="flex size-7 items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
							aria-label="Upload attachment"
						>
							<Upload className="size-4" />
						</button>
						<input
							ref={fileInputRef}
							type="file"
							multiple
							className="sr-only"
							onChange={handleFileInput}
						/>
					</>
				)}
			</div>

			{/* Attachment list */}
			{attachments.length > 0 && (
				<div className="space-y-2">
					{attachments.map((att) => (
						<AttachmentItem
							key={att.id}
							attachment={att}
							canDelete={canEdit}
							onDelete={handleDelete}
						/>
					))}
				</div>
			)}

			{/* Drop zone */}
			{canEdit && (
				<button
					type="button"
					onDragOver={(e) => {
						e.preventDefault();
						setIsDragOver(true);
					}}
					onDragLeave={() => setIsDragOver(false)}
					onDrop={handleFileDrop}
					onClick={() => fileInputRef.current?.click()}
					className={cn(
						"w-full rounded-xl border-2 border-dashed p-8 text-center transition-all duration-150 cursor-pointer",
						isDragOver
							? "border-primary/50 bg-primary/5 text-primary"
							: "border-border/40 bg-muted/20 text-muted-foreground/50 hover:border-border/60 hover:bg-muted/30",
					)}
				>
					<Paperclip className="size-6 mx-auto mb-2.5 opacity-60" />
					<p className="text-sm font-medium">Drop your files here to upload</p>
					<p className="text-xs mt-1 opacity-70">or click to browse</p>
				</button>
			)}
		</div>
	);
}
