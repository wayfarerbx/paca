"""Minimal OpenAI-compatible chat-completions server for e2e tests.

No model weights, no network egress — just enough of the API surface for
litellm's openai/-prefixed client (running inside the real OpenHands
agent-server sandbox container) to get a usable response.

Each server instance is driven by a `script`: a list of ScriptedReply,
returned in order on successive /v1/chat/completions calls (the last entry
repeats once exhausted). Use text_reply()/tool_call_reply()/error_reply() to
build one, e.g. a tool-call-then-final-answer flow:

    FakeOpenAIServer(script=[
        tool_call_reply("terminal", {"command": "echo hi"}),
        text_reply("Done!"),
    ])

Binds to 0.0.0.0 so containers on the host's Docker bridge network can reach
it via the bridge gateway IP (see _bridge_gateway_ip in the e2e tests).
"""

from __future__ import annotations

import json
import threading
import time
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

REPLY_TEXT = "Hello! This is a canned reply from the fake LLM server."


@dataclass(frozen=True)
class ToolCall:
    name: str
    arguments: dict
    call_id: str = "call_fake_1"


@dataclass(frozen=True)
class ScriptedReply:
    """One scripted LLM turn. Exactly one of content / tool_call / error_status is set."""

    content: str | None = None
    tool_call: ToolCall | None = None
    error_status: int | None = None


def text_reply(content: str = REPLY_TEXT) -> ScriptedReply:
    return ScriptedReply(content=content)


def tool_call_reply(name: str, arguments: dict, call_id: str = "call_fake_1") -> ScriptedReply:
    return ScriptedReply(tool_call=ToolCall(name=name, arguments=arguments, call_id=call_id))


def error_reply(status: int = 500) -> ScriptedReply:
    return ScriptedReply(error_status=status)


class _Handler(BaseHTTPRequestHandler):
    server: FakeOpenAIServer

    def log_message(self, format: str, *args: object) -> None:  # noqa: A002
        pass  # silence default request logging to stderr

    def do_GET(self) -> None:
        if self.path.startswith(("/v1/models", "/health")):
            models = {"object": "list", "data": [{"id": "fake-model", "object": "model"}]}
            self._send_json(200, models)
        else:
            self._send_json(404, {"error": "not found"})

    def do_POST(self) -> None:
        if not self.path.startswith("/v1/chat/completions"):
            self._send_json(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length) if length else b"{}"
        try:
            payload = json.loads(body or b"{}")
        except json.JSONDecodeError:
            payload = {}

        reply = self.server.next_reply()
        if reply.error_status is not None:
            self._send_json(reply.error_status, {"error": {"message": "fake upstream failure"}})
        elif payload.get("stream"):
            self._send_streaming_reply(reply)
        else:
            self._send_non_streaming_reply(reply)

    def _send_streaming_reply(self, reply: ScriptedReply) -> None:
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        base = {
            "id": "fake-chunk",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "fake-model",
        }
        chunks = (
            self._tool_call_chunks(base, reply.tool_call)
            if reply.tool_call is not None
            else self._text_chunks(base, reply.content or "")
        )
        for chunk in chunks:
            self.wfile.write(f"data: {json.dumps(chunk)}\n\n".encode())
        self.wfile.write(b"data: [DONE]\n\n")

    def _text_chunks(self, base: dict, content: str) -> list[dict]:
        return [
            {
                **base,
                "choices": [
                    {
                        "index": 0,
                        "delta": {"role": "assistant", "content": ""},
                        "finish_reason": None,
                    }
                ],
            },
            {
                **base,
                "choices": [{"index": 0, "delta": {"content": content}, "finish_reason": None}],
            },
            {**base, "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]},
        ]

    def _tool_call_chunks(self, base: dict, tool_call: ToolCall) -> list[dict]:
        return [
            {
                **base,
                "choices": [
                    {
                        "index": 0,
                        "delta": {
                            "role": "assistant",
                            "content": None,
                            "tool_calls": [
                                {
                                    "index": 0,
                                    "id": tool_call.call_id,
                                    "type": "function",
                                    "function": {"name": tool_call.name, "arguments": ""},
                                }
                            ],
                        },
                        "finish_reason": None,
                    }
                ],
            },
            {
                **base,
                "choices": [
                    {
                        "index": 0,
                        "delta": {
                            "tool_calls": [
                                {
                                    "index": 0,
                                    "function": {"arguments": json.dumps(tool_call.arguments)},
                                }
                            ]
                        },
                        "finish_reason": None,
                    }
                ],
            },
            {**base, "choices": [{"index": 0, "delta": {}, "finish_reason": "tool_calls"}]},
        ]

    def _send_non_streaming_reply(self, reply: ScriptedReply) -> None:
        if reply.tool_call is not None:
            message = {
                "role": "assistant",
                "content": None,
                "tool_calls": [
                    {
                        "id": reply.tool_call.call_id,
                        "type": "function",
                        "function": {
                            "name": reply.tool_call.name,
                            "arguments": json.dumps(reply.tool_call.arguments),
                        },
                    }
                ],
            }
            finish_reason = "tool_calls"
        else:
            message = {"role": "assistant", "content": reply.content}
            finish_reason = "stop"
        self._send_json(
            200,
            {
                "id": "fake-completion",
                "object": "chat.completion",
                "created": int(time.time()),
                "model": "fake-model",
                "choices": [{"index": 0, "message": message, "finish_reason": finish_reason}],
                "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
            },
        )

    def _send_json(self, status: int, body: dict) -> None:
        data = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


class FakeOpenAIServer(ThreadingHTTPServer):
    """Runs _Handler on a background thread for the lifetime of a test."""

    def __init__(
        self,
        script: list[ScriptedReply] | None = None,
        host: str = "0.0.0.0",
        port: int = 0,
    ) -> None:
        super().__init__((host, port), _Handler)
        self._script = script or [text_reply()]
        self._call_index = 0
        self._lock = threading.Lock()
        self._thread = threading.Thread(target=self.serve_forever, daemon=True)

    def next_reply(self) -> ScriptedReply:
        with self._lock:
            reply = self._script[min(self._call_index, len(self._script) - 1)]
            self._call_index += 1
            return reply

    @property
    def call_count(self) -> int:
        with self._lock:
            return self._call_index

    @property
    def port(self) -> int:
        return self.server_address[1]

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self.shutdown()
        self.server_close()
        self._thread.join(timeout=5)
