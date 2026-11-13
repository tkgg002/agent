#!/usr/bin/env python3
"""End-to-end tests: Claude Code hook JSON on stdin -> adapter -> agy_guard -> Claude Code JSON on stdout.

Real subprocess, real agy_guard, isolated AGY_STATE_DIR and memory root. No mocks.
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ADAPTER = Path(__file__).resolve().parents[1] / "claude_hook_adapter.py"
AGENT = Path(__file__).resolve().parents[3]


def run_hook(event: str, payload: dict, env_extra: dict | None = None) -> dict:
    env = dict(os.environ, **(env_extra or {}))
    r = subprocess.run([sys.executable, str(ADAPTER), event], input=json.dumps(payload),
                       capture_output=True, text=True, env=env, timeout=60)
    assert r.returncode == 0, r.stderr[-1500:]
    return json.loads(r.stdout.strip() or "{}")


def write_transcript(path: Path, lines: list[dict]) -> None:
    path.write_text("".join(json.dumps(l) + "\n" for l in lines), encoding="utf-8")


class TestClaudeHookAdapter(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.env = {"AGY_STATE_DIR": os.path.join(self.tmp.name, "state")}
        self.transcript = Path(self.tmp.name) / "t.jsonl"
        write_transcript(self.transcript, [
            {"type": "user", "message": {"role": "user", "content": "sửa bug nhỏ"}},
            {"type": "assistant", "message": {"role": "assistant", "content": [
                {"type": "tool_use", "name": "Read", "input": {"file_path": "/x/a.py"}}]}},
        ])

    def tearDown(self):
        self.tmp.cleanup()

    def base(self, **kw):
        return {"session_id": "s-test", "transcript_path": str(self.transcript), "cwd": self.tmp.name, **kw}

    def test_h5_git_push_denied_via_bash(self):
        out = run_hook("pre-tool", self.base(hook_event_name="PreToolUse", tool_name="Bash",
                                             tool_input={"command": "git push origin main"}), self.env)
        hso = out["hookSpecificOutput"]
        self.assertEqual(hso["hookEventName"], "PreToolUse")
        self.assertEqual(hso["permissionDecision"], "deny")
        self.assertIn("git push", hso["permissionDecisionReason"])

    def test_h2_write_overwrite_existing_memory_file_denied(self):
        mem_file = AGENT / "memory" / "global" / "lessons" / "INDEX.md"
        self.assertTrue(mem_file.is_file())
        out = run_hook("pre-tool", self.base(hook_event_name="PreToolUse", tool_name="Write",
                                             tool_input={"file_path": str(mem_file), "content": "x"}), self.env)
        self.assertEqual(out["hookSpecificOutput"]["permissionDecision"], "deny")

    def test_read_only_tool_passes_through(self):
        out = run_hook("pre-tool", self.base(hook_event_name="PreToolUse", tool_name="Read",
                                             tool_input={"file_path": "/x/a.py"}), self.env)
        self.assertEqual(out, {})

    def test_harmless_bash_allowed(self):
        out = run_hook("pre-tool", self.base(hook_event_name="PreToolUse", tool_name="Bash",
                                             tool_input={"command": "ls -la"}), self.env)
        self.assertEqual(out, {})

    def test_h1_boot_injects_context_on_user_prompt(self):
        out = run_hook("user-prompt", self.base(hook_event_name="UserPromptSubmit", prompt="kiểm tra lỗi X"), self.env)
        hso = out["hookSpecificOutput"]
        self.assertEqual(hso["hookEventName"], "UserPromptSubmit")
        self.assertIn("BOOT S0", hso["additionalContext"])

    def test_stop_hook_active_short_circuits(self):
        out = run_hook("stop", self.base(hook_event_name="Stop", stop_hook_active=True), self.env)
        self.assertEqual(out, {})

    def test_transcript_conversion_maps_tools(self):
        sys.path.insert(0, str(ADAPTER.parent))
        import claude_hook_adapter as a
        write_transcript(self.transcript, [
            {"type": "user", "message": {"role": "user", "content": [{"type": "text", "text": "APPROVE"}]}},
            {"type": "user", "message": {"role": "user", "content": [{"type": "tool_result", "content": "ok"}]}},
            {"type": "assistant", "message": {"role": "assistant", "content": [
                {"type": "tool_use", "name": "Edit", "input": {"file_path": "/p/f.go", "old_string": "a", "new_string": "b"}},
                {"type": "tool_use", "name": "Bash", "input": {"command": "go test ./..."}}]}},
        ])
        steps = a.convert_transcript(str(self.transcript))
        self.assertEqual([s["type"] for s in steps], ["USER_INPUT", "ASSISTANT"])
        calls = steps[1]["tool_calls"]
        self.assertEqual(calls[0]["name"], "replace_file_content")
        self.assertEqual(calls[0]["args"], {"TargetFile": "/p/f.go", "TargetContent": "a", "ReplacementContent": "b"})
        self.assertEqual(calls[1], {"name": "run_command", "args": {"CommandLine": "go test ./...", "Cwd": ""}})


if __name__ == "__main__":
    unittest.main(verbosity=2)
