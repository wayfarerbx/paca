import type { ThreadMessageLike } from "@assistant-ui/react";
import type { AgentConversationEvent } from "@/lib/agent-api";

// Extract plain text from a content block array [{type:"text", text:"..."}] or a bare string.
export function extractContentText(content: unknown): string | null {
	if (Array.isArray(content)) {
		const parts = (content as Array<unknown>)
			.map((c) => {
				if (typeof c === "object" && c !== null && "text" in c) {
					const t = (c as { text: unknown }).text;
					return typeof t === "string" ? t : null;
				}
				return null;
			})
			.filter((t): t is string => t !== null && t.trim().length > 0);
		return parts.length > 0 ? parts.join("\n\n").trim() : null;
	}
	if (typeof content === "string" && content.trim().length > 0) {
		return content.trim();
	}
	return null;
}

type MutableToolCallPart = {
	type: "tool-call";
	toolCallId: string;
	toolName: string;
	argsText: string;
	result?: unknown;
};

type MutablePart =
	| { type: "text"; text: string }
	| { type: "reasoning"; text: string }
	| MutableToolCallPart;

interface InProgressMessage {
	id: string;
	createdAt: Date;
	parts: MutablePart[];
	// Keyed by tool_call_id so a later ObservationEvent can attach its result
	// to the ActionEvent's tool-call part within the same turn.
	openToolCalls: Map<string, MutableToolCallPart>;
}

function startAssistantMessage(id: string, createdAt: Date): InProgressMessage {
	return { id, createdAt, parts: [], openToolCalls: new Map() };
}

function toThreadMessage(
	msg: InProgressMessage,
	isTrailingAndRunning: boolean,
): ThreadMessageLike {
	return {
		id: msg.id,
		role: "assistant",
		createdAt: msg.createdAt,
		status: isTrailingAndRunning
			? { type: "running" }
			: { type: "complete", reason: "stop" },
		content: msg.parts,
	};
}

/**
 * Converts raw conversation events into assistant-ui's ThreadMessageLike[],
 * grouping each agent turn's thought/tool-calls/reply into one assistant
 * message's content parts (rather than one bubble per raw event) — the
 * `reasoning`/`tool-call` part types are auto-grouped into collapsible
 * sections by the Thread component's `MessagePrimitive.GroupedParts`.
 *
 * `isThreadRunning` marks the trailing in-progress assistant message (if
 * any) as still running so the Thread shows its "working" indicator.
 */
export function eventsToThreadMessages(
	events: AgentConversationEvent[],
	isThreadRunning: boolean,
): ThreadMessageLike[] {
	const messages: ThreadMessageLike[] = [];
	let current: InProgressMessage | null = null;

	const flushCurrent = () => {
		if (current && current.parts.length > 0) {
			messages.push(toThreadMessage(current, false));
		}
		current = null;
	};

	for (const ev of events) {
		const p = ev.payload;
		const t = ev.event_type;

		// Non-user-visible bookkeeping events — already filtered server-side,
		// skipped here too as a defensive guard against legacy/unfiltered data.
		if (
			t === "ConversationStateUpdateEvent" ||
			t === "SystemPromptEvent" ||
			t === "StreamingDeltaEvent"
		) {
			continue;
		}

		if (t === "MessageEvent") {
			const llmMsg = p.llm_message as { content?: unknown } | undefined;
			const text =
				extractContentText(llmMsg?.content) ?? extractContentText(p.content);
			if (!text) continue;

			if (ev.event_source === "user") {
				flushCurrent();
				messages.push({
					id: ev.id,
					role: "user",
					createdAt: new Date(ev.created_at),
					content: [{ type: "text", text }],
				});
				continue;
			}

			// Agent's natural-language reply — part of the current turn.
			if (!current)
				current = startAssistantMessage(ev.id, new Date(ev.created_at));
			current.parts.push({ type: "text", text });
			continue;
		}

		if (t === "ActionEvent") {
			if (!current)
				current = startAssistantMessage(ev.id, new Date(ev.created_at));

			const thoughtText = extractContentText(p.thought);
			if (thoughtText)
				current.parts.push({ type: "reasoning", text: thoughtText });

			const toolCall = p.tool_call as
				| { name?: string; arguments?: unknown }
				| undefined;
			const toolCallId =
				typeof p.tool_call_id === "string" ? p.tool_call_id : ev.id;
			const toolName =
				(typeof p.tool_name === "string" ? p.tool_name : undefined) ??
				toolCall?.name ??
				"tool";
			const argsText =
				(typeof toolCall?.arguments === "string" ? toolCall.arguments : null) ??
				JSON.stringify(toolCall?.arguments ?? toolCall ?? {}, null, 2);

			const part: MutableToolCallPart = {
				type: "tool-call",
				toolCallId,
				toolName,
				argsText,
			};
			current.parts.push(part);
			current.openToolCalls.set(toolCallId, part);
			continue;
		}

		if (t === "ObservationEvent") {
			const obs = p.observation as Record<string, unknown> | undefined;
			const resultText =
				(obs &&
					(extractContentText(obs.content) ??
						(typeof obs.message === "string" ? obs.message : null))) ??
				extractContentText(p.content) ??
				(typeof p.message === "string" ? p.message : null) ??
				(typeof p.output === "string" ? p.output : null) ??
				"";

			const toolCallId =
				typeof p.tool_call_id === "string" ? p.tool_call_id : undefined;
			const openPart =
				toolCallId && current
					? current.openToolCalls.get(toolCallId)
					: undefined;

			if (openPart) {
				openPart.result = resultText;
			} else {
				// No matching open tool-call in this turn (history gap) — append
				// a standalone, already-complete tool-call part.
				if (!current)
					current = startAssistantMessage(ev.id, new Date(ev.created_at));
				const toolName = typeof p.tool_name === "string" ? p.tool_name : "tool";
				current.parts.push({
					type: "tool-call",
					toolCallId: toolCallId ?? ev.id,
					toolName,
					argsText: "",
					result: resultText,
				});
			}
			continue;
		}

		// Fallback for other/legacy event types: surface as plain text so
		// nothing silently disappears from the transcript.
		const fallbackText =
			extractContentText(p.content) ??
			extractContentText(p.thought) ??
			(typeof p.message === "string" ? p.message : null);
		if (fallbackText) {
			if (ev.event_source === "user") {
				flushCurrent();
				messages.push({
					id: ev.id,
					role: "user",
					createdAt: new Date(ev.created_at),
					content: [{ type: "text", text: fallbackText }],
				});
			} else {
				if (!current)
					current = startAssistantMessage(ev.id, new Date(ev.created_at));
				current.parts.push({ type: "text", text: fallbackText });
			}
		}
	}

	if (current && current.parts.length > 0) {
		messages.push(toThreadMessage(current, isThreadRunning));
	}

	return messages;
}
