#!/usr/bin/env python3
"""Claude Code -> agy_guard.py adapter (H1-H7 on Claude Code, zero change to agy_guard core).

Usage (from .claude/settings.json):
  python3 hooks/claude/claude_hook_adapter.py <event>
  <event> in: session-start | user-prompt | pre-tool | post-tool | stop

Flow:
  1. Read Claude Code hook JSON from stdin (session_id, transcript_path, tool_name, tool_input, ...).
  2. Convert the Claude transcript (JSONL {type:user|assistant, message:{content:[...]}}) into the
     Antigravity step list agy_guard expects ({type:"USER_INPUT", content} / {tool_calls:[{name,args}]}),
     written to $AGY_STATE_DIR/claude/<session_id>.transcript.jsonl.
  3. Map Claude tool names/args -> Antigravity vocabulary (Write->write_to_file, Edit->replace_file_content,
     MultiEdit->multi_replace_file_content, Bash->run_command, Task->invoke_subagent, Read->view_file, ...).
  4. agy_guard.dispatch(<antigravity-event>, payload), then translate the result back:
       {decision: deny|ask, reason}          -> hookSpecificOutput.permissionDecision
       {injectSteps:[{ephemeralMessage}]}    -> hookSpecificOutput.additionalContext
       {decision: continue, reason} (Stop)   -> {decision: "block", reason}
Antigravity (.agents/hooks.json) keeps calling agy_guard.py directly; nothing there changes.
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Any, Dict, List, Tuple

HERE = Path(__file__).resolve()
AGENT = Path(os.environ.get("AGY_AGENT_ROOT", str(HERE.parents[2])))
sys.path.insert(0, str(AGENT / "hooks" / "agy"))
import agy_guard  # noqa: E402

STATE = Path(os.environ.get("AGY_STATE_DIR", "/tmp/agy")) / "claude"

# Claude Code tool -> Antigravity tool name (agy_guard.WRITE_TOOLS / READ_ARG_KEYS vocabulary)
TOOL_MAP = {
    "Write": "write_to_file",
    "Edit": "replace_file_content",
    "MultiEdit": "multi_replace_file_content",
    "NotebookEdit": "replace_file_content",
    "Bash": "run_command",
    "Task": "invoke_subagent",
    "Read": "view_file",
    "Grep": "grep_search",
    "Glob": "find_by_name",
    "WebFetch": "read_url_content",
    "WebSearch": "search_web",
}
EVENT_MAP = {"session-start": "pre-invocation", "user-prompt": "pre-invocation",
             "post-tool": "post-invocation", "stop": "stop"}
HOOK_EVENT_NAMES = {"session-start": "SessionStart", "user-prompt": "UserPromptSubmit",
                    "pre-tool": "PreToolUse", "post-tool": "PostToolUse", "stop": "Stop"}
WRITE_TOOLS_CLAUDE = {"Write", "Edit", "MultiEdit", "NotebookEdit"}


def map_args(tool: str, inp: Dict[str, Any]) -> Dict[str, Any]:
    """Translate Claude tool_input keys into the keys agy_guard already understands."""
    inp = dict(inp or {})
    if tool == "Write":
        p = str(inp.get("file_path", ""))
        return {"TargetFile": p, "CodeContent": inp.get("content", ""),
                "Overwrite": bool(p) and Path(p).exists()}  # H2: Write onto an existing file == overwrite
    if tool in ("Edit", "NotebookEdit"):
        return {"TargetFile": inp.get("file_path") or inp.get("notebook_path", ""),
                "TargetContent": inp.get("old_string") or inp.get("old_source", ""),
                "ReplacementContent": inp.get("new_string") or inp.get("new_source", "")}
    if tool == "MultiEdit":
        return {"TargetFile": inp.get("file_path", ""),
                "ReplacementChunks": [{"TargetContent": e.get("old_string", ""),
                                       "ReplacementContent": e.get("new_string", "")}
                                      for e in (inp.get("edits") or []) if isinstance(e, dict)]}
    if tool == "Bash":
        return {"CommandLine": inp.get("command", ""), "Cwd": inp.get("cwd", "")}
    if tool == "Task":
        return {"Prompt": inp.get("prompt", ""), "Description": inp.get("description", "")}
    if tool == "Read":
        return {"AbsolutePath": inp.get("file_path", "")}
    if tool == "Grep":
        return {"SearchPath": inp.get("path", ""), "Query": inp.get("pattern", "")}
    if tool == "Glob":
        return {"SearchDirectory": inp.get("path", ""), "Pattern": inp.get("pattern", "")}
    if tool == "WebFetch":
        return {"Url": inp.get("url", "")}
    return inp



def convert_transcript(transcript_path: str) -> List[Dict[str, Any]]:
    """Claude JSONL -> Antigravity steps: USER_INPUT per real user text; one step per assistant message."""
    steps: List[Dict[str, Any]] = []
    if not transcript_path or not Path(transcript_path).is_file():
        return steps
    with open(transcript_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            msg = rec.get("message") or {}
            role = rec.get("type") or msg.get("role")
            content = msg.get("content")
            if role == "user":
                # tool_result blocks also arrive as role=user; only real text counts as USER_INPUT
                text = content if isinstance(content, str) else "\n".join(
                    b.get("text", "") for b in (content or []) if isinstance(b, dict) and b.get("type") == "text")
                if text.strip() and not rec.get("isMeta"):
                    steps.append({"type": "USER_INPUT", "content": text})
            elif role == "assistant":
                calls = [{"name": TOOL_MAP.get(b.get("name"), b.get("name")),
                          "args": map_args(str(b.get("name")), b.get("input") or {})}
                         for b in (content or []) if isinstance(b, dict) and b.get("type") == "tool_use"]
                steps.append({"type": "ASSISTANT", "tool_calls": calls})
    return steps


def build_payload(ev: str, cc: Dict[str, Any]) -> Tuple[Dict[str, Any], str]:
    sid = str(cc.get("session_id") or "default")
    STATE.mkdir(parents=True, exist_ok=True)
    steps = convert_transcript(str(cc.get("transcript_path") or ""))
    if ev == "user-prompt" and cc.get("prompt"):  # transcript lags; make the live prompt the last USER_INPUT
        steps.append({"type": "USER_INPUT", "content": str(cc["prompt"])})
    tr = STATE / f"{sid}.transcript.jsonl"
    tr.write_text("".join(json.dumps(s, ensure_ascii=False) + "\n" for s in steps), encoding="utf-8")
    payload: Dict[str, Any] = {"conversationId": sid, "transcriptPath": str(tr)}
    agy_event = EVENT_MAP.get(ev, "")
    if ev == "pre-tool":
        tool = str(cc.get("tool_name") or "")
        payload["toolCall"] = {"name": TOOL_MAP.get(tool, tool), "args": map_args(tool, cc.get("tool_input") or {})}
        agy_event = ("pre-write" if tool in WRITE_TOOLS_CLAUDE else
                     "pre-command" if tool == "Bash" else
                     "pre-subagent" if tool == "Task" else "")  # read-only tools: nothing to guard
    return payload, agy_event


def to_claude(ev: str, res: Dict[str, Any], hook_event_name: str) -> Dict[str, Any]:
    if not res:
        return {}
    inject = "\n\n".join(str(s.get("ephemeralMessage", "")) for s in res.get("injectSteps") or []
                         if s.get("ephemeralMessage"))
    decision, reason = res.get("decision"), str(res.get("reason") or "")
    if ev == "pre-tool":
        if decision in ("deny", "ask"):
            return {"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": decision,
                                           "permissionDecisionReason": reason or f"agy_guard: {decision}"}}
        return {}
    if ev == "stop":
        # Antigravity "continue" == keep working; Claude expresses that as blocking the stop
        return {"decision": "block", "reason": reason} if decision == "continue" else {}
    return {"hookSpecificOutput": {"hookEventName": hook_event_name, "additionalContext": inject}} if inject else {}


def main() -> int:
    if len(sys.argv) < 2:
        print("{}")
        return 0
    ev = sys.argv[1]
    raw = sys.stdin.read().strip()
    try:
        cc = json.loads(raw) if raw else {}
    except json.JSONDecodeError:
        cc = {}
    hook_event_name = str(cc.get("hook_event_name") or HOOK_EVENT_NAMES.get(ev, ""))
    if ev == "stop" and cc.get("stop_hook_active"):
        print("{}")  # already continuing because of a stop hook; agy_guard's own 3-block cap governs
        return 0
    payload, agy_event = build_payload(ev, cc)
    res = agy_guard.dispatch(agy_event, payload) if agy_event else {}
    print(json.dumps(to_claude(ev, res, hook_event_name), ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
