"use client";

import {
	AttachmentPrimitive,
	ComposerPrimitive,
	MessagePrimitive,
	useAui,
	useAuiState,
} from "@assistant-ui/react";
import { AlertCircleIcon, FileText, Loader2Icon, XIcon } from "lucide-react";
import { type FC, type PropsWithChildren, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/shallow";
import { TooltipIconButton } from "@/components/assistant-ui/tooltip-icon-button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
	Dialog,
	DialogContent,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

const useFileSrc = (file: File | undefined) => {
	const [src, setSrc] = useState<string | undefined>(undefined);

	useEffect(() => {
		if (!file) {
			setSrc(undefined);
			return;
		}

		const objectUrl = URL.createObjectURL(file);
		setSrc(objectUrl);

		return () => {
			URL.revokeObjectURL(objectUrl);
		};
	}, [file]);

	return src;
};

const useAttachmentSrc = () => {
	const { file, src } = useAuiState(
		useShallow((s): { file?: File; src?: string } => {
			if (s.attachment.type !== "image") return {};
			if (s.attachment.file) return { file: s.attachment.file };
			const src = s.attachment.content?.filter((c) => c.type === "image")[0]
				?.image;
			if (!src) return {};
			return { src };
		}),
	);

	return useFileSrc(file) ?? src;
};

type AttachmentPreviewProps = {
	src: string;
};

const AttachmentPreview: FC<AttachmentPreviewProps> = ({ src }) => {
	const { t } = useTranslation("projects");
	const [isLoaded, setIsLoaded] = useState(false);
	return (
		<img
			src={src}
			alt={t("agents.thread.attachmentPreview")}
			className={cn(
				"block h-auto max-h-[80vh] w-auto max-w-full object-contain",
				isLoaded
					? "aui-attachment-preview-image-loaded"
					: "aui-attachment-preview-image-loading invisible",
			)}
			onLoad={() => setIsLoaded(true)}
		/>
	);
};

const AttachmentPreviewDialog: FC<PropsWithChildren> = ({ children }) => {
	const { t } = useTranslation("projects");
	const src = useAttachmentSrc();

	if (!src) return children;

	return (
		<Dialog>
			<DialogTrigger className="aui-attachment-preview-trigger hover:bg-accent/50 cursor-pointer transition-colors">
				{children}
			</DialogTrigger>
			<DialogContent className="aui-attachment-preview-dialog-content [&>button]:bg-foreground/60 [&_svg]:text-background [&>button]:hover:[&_svg]:text-destructive p-2 sm:max-w-3xl [&>button]:rounded-full [&>button]:p-1 [&>button]:opacity-100 [&>button]:ring-0!">
				<DialogTitle className="aui-sr-only sr-only">
					{t("agents.thread.imageAttachmentPreviewTitle")}
				</DialogTitle>
				<div className="aui-attachment-preview bg-background relative mx-auto flex max-h-[80dvh] w-full items-center justify-center overflow-hidden">
					<AttachmentPreview src={src} />
				</div>
			</DialogContent>
		</Dialog>
	);
};

const AttachmentThumb: FC = () => {
	const { t } = useTranslation("projects");
	const src = useAttachmentSrc();

	return (
		<Avatar className="aui-attachment-tile-avatar h-full w-full rounded-none">
			<AvatarImage
				src={src}
				alt={t("agents.thread.attachmentPreview")}
				className="aui-attachment-tile-image object-cover"
			/>
			<AvatarFallback>
				<FileText className="aui-attachment-tile-fallback-icon text-muted-foreground size-8" />
			</AvatarFallback>
		</Avatar>
	);
};

const AttachmentUI: FC = () => {
	const { t } = useTranslation("projects");
	const aui = useAui();
	const isComposer = aui.attachment.source !== "message";

	const isImage = useAuiState((s) => s.attachment.type === "image");
	const attachmentType = useAuiState((s) => s.attachment.type);
	const typeLabel =
		attachmentType === "image"
			? t("agents.thread.attachmentTypeImage")
			: attachmentType === "document"
				? t("agents.thread.attachmentTypeDocument")
				: attachmentType === "file"
					? t("agents.thread.attachmentTypeFile")
					: attachmentType;

	const uploadState = useAuiState((s) =>
		s.attachment.status.type === "running"
			? "uploading"
			: s.attachment.status.type === "incomplete" &&
					s.attachment.status.reason === "error"
				? "error"
				: undefined,
	);
	const isUploading = uploadState === "uploading";
	const isError = uploadState === "error";
	const attachmentAriaLabel = isError
		? t("agents.thread.attachmentAriaLabelError", { type: typeLabel })
		: isUploading
			? t("agents.thread.attachmentAriaLabelUploading", { type: typeLabel })
			: t("agents.thread.attachmentAriaLabel", { type: typeLabel });

	return (
		<Tooltip>
			<AttachmentPrimitive.Root
				className={cn(
					"aui-attachment-root relative",
					isImage &&
						!isComposer &&
						"aui-attachment-root-message only:*:first:size-24",
				)}
			>
				<AttachmentPreviewDialog>
					<TooltipTrigger
						render={
							<button
								type="button"
								className={cn(
									"aui-attachment-tile bg-muted relative size-14 cursor-pointer overflow-hidden rounded-[calc(var(--composer-radius)-var(--composer-padding))] border transition-opacity hover:opacity-75",
									isError && "border-destructive",
								)}
								aria-label={attachmentAriaLabel}
							/>
						}
					>
						<AttachmentThumb />
						{isUploading && (
							<div
								aria-hidden="true"
								className="aui-attachment-tile-uploading bg-background/60 absolute inset-0 flex items-center justify-center backdrop-blur-[1px]"
							>
								<Loader2Icon className="text-muted-foreground size-5 animate-spin" />
							</div>
						)}
						{isError && (
							<div
								aria-hidden="true"
								className="aui-attachment-tile-error bg-destructive/10 absolute inset-0 flex items-center justify-center"
							>
								<AlertCircleIcon className="text-destructive size-5" />
							</div>
						)}
					</TooltipTrigger>
				</AttachmentPreviewDialog>
				{isComposer && <AttachmentRemove />}
			</AttachmentPrimitive.Root>
			<TooltipContent side="top">
				<AttachmentPrimitive.Name />
			</TooltipContent>
		</Tooltip>
	);
};

const AttachmentRemove: FC = () => {
	const { t } = useTranslation("projects");
	return (
		<AttachmentPrimitive.Remove
			render={
				<TooltipIconButton
					tooltip={t("agents.thread.removeFile")}
					className="aui-attachment-tile-remove text-muted-foreground hover:[&_svg]:text-destructive absolute end-1.5 top-1.5 size-3.5 rounded-full bg-white opacity-100 shadow-sm hover:bg-white! [&_svg]:text-black"
					side="top"
				/>
			}
		>
			<XIcon className="aui-attachment-remove-icon size-3 dark:stroke-[2.5px]" />
		</AttachmentPrimitive.Remove>
	);
};

export const UserMessageAttachments: FC = () => {
	return (
		<div className="aui-user-message-attachments-end col-span-full col-start-1 row-start-1 flex w-full flex-row justify-end gap-2">
			<MessagePrimitive.Attachments>
				{() => <AttachmentUI />}
			</MessagePrimitive.Attachments>
		</div>
	);
};

export const ComposerAttachments: FC = () => {
	return (
		<div className="aui-composer-attachments flex w-full flex-row items-center gap-2 overflow-x-auto empty:hidden">
			<ComposerPrimitive.Attachments>
				{() => <AttachmentUI />}
			</ComposerPrimitive.Attachments>
		</div>
	);
};
