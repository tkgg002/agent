import argparse
import json
import re
import subprocess
import sys
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Thread

AGENT_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(AGENT_ROOT / "hooks" / "agy"))
sys.path.insert(0, str(AGENT_ROOT / "tooling"))

import agy_guard  # noqa: E402
import laya_gate  # noqa: E402
import verify_governance  # noqa: E402


class TestAgentCoreF01ToF08(unittest.TestCase):
    def test_f01_hooks_json_regex_alternation_and_6_write_tools(self) -> None:
        hook_files = [
            AGENT_ROOT / ".agents" / "hooks.json",
            AGENT_ROOT.parent / "data-hub" / ".agents" / "hooks.json",
            Path.home() / ".gemini" / "config" / "hooks.json",
        ]
        for hp in hook_files:
            if not hp.is_file():
                continue
            cfg = json.loads(hp.read_text(encoding="utf-8"))
            matcher = cfg["agy-pre-write"]["PreToolUse"][0]["matcher"]
            self.assertNotIn(r"\|", matcher, f"{hp} still contains escaped pipe")
            rx = re.compile(matcher)
            for tool in (
                "write_to_file",
                "replace_file_content",
                "multi_replace_file_content",
                "write_file",
                "edit_file",
                "multi_edit",
            ):
                self.assertIsNotNone(rx.fullmatch(tool), f"{tool} not matched by {matcher} in {hp}")
            sub_matcher = cfg["agy-pre-subagent"]["PreToolUse"][0]["matcher"]
            self.assertNotIn(r"\|", sub_matcher)
            sub_rx = re.compile(sub_matcher)
            self.assertIsNotNone(sub_rx.fullmatch("invoke_subagent"))
            self.assertIsNotNone(sub_rx.fullmatch("send_message"))

    def test_f02_and_r03_h2_blocks_write_file_and_empty_target_on_existing_memory(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            mem_root = Path(td) / "memory"
            ws_dir = mem_root / "workspaces" / "ws1"
            ws_dir.mkdir(parents=True)
            prog = ws_dir / "05_progress_t1.md"
            prog.write_text("# 05. Progress\n- [2026-11-10T09:00:00+07:00] [Agent:A] START\n", encoding="utf-8")
            req_doc = ws_dir / "01_requirements_t1.md"
            req_doc.write_text("# 01. Req\n", encoding="utf-8")

            # 1. write_file without Overwrite on existing memory file -> DENY via subprocess
            payload_wf = {
                "toolCall": {
                    "name": "write_file",
                    "args": {"Path": str(prog), "Content": "WIPED"},
                }
            }
            proc = subprocess.run(
                [sys.executable, str(AGENT_ROOT / "hooks" / "agy" / "agy_guard.py"), "pre-write"],
                input=json.dumps(payload_wf),
                text=True,
                capture_output=True,
                check=False,
            )
            out_wf = json.loads(proc.stdout)
            self.assertEqual(out_wf.get("decision"), "deny")

            # 2. edit_file with empty Target="" on existing APPEND_ONLY file -> DENY
            res_empty = agy_guard.handle_pre_write(
                {"toolCall": {"name": "edit_file", "args": {"Path": str(prog), "Target": "", "Replacement": "X"}}},
                mem_root=mem_root,
                ws_root=mem_root / "workspaces",
            )
            self.assertEqual(res_empty.get("decision"), "deny")

            # 3. edit_file with empty Target="" on regular existing memory file -> DENY (N-05 / R-03)
            res_empty_req = agy_guard.handle_pre_write(
                {"toolCall": {"name": "edit_file", "args": {"Path": str(req_doc), "Target": "", "Replacement": "X"}}},
                mem_root=mem_root,
                ws_root=mem_root / "workspaces",
            )
            self.assertEqual(res_empty_req.get("decision"), "deny")

            # 4. edit_file preserving Target on APPEND_ONLY file -> ALLOW
            old_txt = prog.read_text(encoding="utf-8")
            res_ok = agy_guard.handle_pre_write(
                {
                    "toolCall": {
                        "name": "edit_file",
                        "args": {
                            "Path": str(prog),
                            "Target": old_txt,
                            "Replacement": old_txt + "- [2026-11-10T09:05:00+07:00] [Agent:A] DONE\n",
                        },
                    }
                },
                mem_root=mem_root,
                ws_root=mem_root / "workspaces",
            )
            self.assertNotEqual(res_ok.get("decision"), "deny")

    def test_f03_and_r02_read_file_not_counted_as_write_or_touched(self) -> None:
        st = [
            {
                "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/ws/09_tasks_solution_x.md"}},
                    {"name": "read_file", "args": {"Path": "/tmp/ws/05_progress_x.md"}},
                    {"name": "read_file", "args": {"Path": "/tmp/ws/06_evidence/EV-1.json"}},
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/ws/memory/global/lessons.md"}},
                ]
            }
        ]
        self.assertIsNone(agy_guard.latest_solution_suffix(st))
        self.assertEqual(agy_guard.lessons_edits_since(st), [])
        self.assertEqual(agy_guard.src_edits(st), [])
        # R-02: read_file on 05_progress_*.md or 09_tasks_solution_*.md MUST NOT make touched() return True
        self.assertFalse(agy_guard.touched(st, r"05_progress"))
        self.assertFalse(agy_guard.touched(st, r"09_tasks_solution"))
        # Write tool (edit_file) or run_command for build_lessons_index.py MUST make touched() return True
        st_written = [
            {
                "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "/tmp/ws/05_progress_x.md", "Target": "a", "Replacement": "ab"}},
                    {"name": "run_command", "args": {"CommandLine": "python3 /path/to/build_lessons_index.py"}},
                ]
            }
        ]
        self.assertTrue(agy_guard.touched(st_written, r"05_progress"))
        self.assertTrue(agy_guard.touched(st_written, r"build_lessons_index\.py"))

    def test_f04_f06_n02_n03_n04_n05_r03_selective_demo_tripwire_and_pure_deletion(self) -> None:
        trips = [
            {
                "id": "go-only-rule",
                "ext": [".go"],
                "trigger_regex": r"BAD_FUNC\(",
                "forbid_regex": r"BAD_FUNC\(",
                "check": "No BAD_FUNC in Go",
            },
            {
                "id": "go-forbid-with-broad-trig",
                "ext": [".go"],
                "trigger_regex": r"func\s+\w+",
                "forbid": r"BAD_FUNC\(",
                "check": "Disjoint forbid check (N-02)",
            },
            {
                "id": "go-require-context",
                "ext": [".go"],
                "trigger_regex": r"RunQuery\(",
                "require_regex": r"ctx\s+context\.Context",
                "check": "Require checked against full_code_body (R-03)",
            },
        ]
        md = (
            "### Hiện tại (Before):\n```go\nfunc x() { BAD_FUNC() }\n```\n"
            "### Đề xuất (After):\n```diff\n--- a/pkg/a.go\n+++ b/pkg/a.go\n"
            "-func x() { BAD_FUNC() }\n+func x() { GOOD_FUNC() }\n```\n"
        )
        snips = agy_guard._extract_demo_snippets(md)
        self.assertEqual(len(snips), 1)
        self.assertEqual(snips[0][0], "pkg/a.go")
        self.assertNotIn("BAD_FUNC", snips[0][1])
        # N-02: clean Go function must NOT trigger forbid tripwire (no Catch-22 deadlock)
        self.assertEqual(agy_guard.code_tripwire_violations(snips[0][0], snips[0][1], trips), [])
        # N-02: Go function containing BAD_FUNC() MUST trigger forbid tripwire
        bad_hits = [t["id"] for t in agy_guard.code_tripwire_violations("pkg/a.go", "func x() { BAD_FUNC() }", trips)]
        self.assertIn("go-forbid-with-broad-trig", bad_hits)

        # R-03: editing 1 line inside a block that already has `ctx context.Context` in full_code_body -> no false positive
        full_go_block = "func Work(ctx context.Context) {\n    RunQuery(ctx)\n}"
        added_one_line = "    RunQuery(ctx)"
        self.assertEqual(
            agy_guard.code_tripwire_violations("pkg/a.go", added_one_line, trips, full_code_body=full_go_block),
            [],
        )

        # N-03: edit_file modifying 1 '+' line inside an existing 09_tasks_solution_*.md diff block MUST be caught by H4t-Demo
        with tempfile.TemporaryDirectory() as td:
            mem_root = Path(td) / "memory"
            (mem_root / "global" / "lessons").mkdir(parents=True)
            (mem_root / "global" / "lessons" / "tripwires.json").write_text(json.dumps(trips), encoding="utf-8")
            ws_dir = mem_root / "workspaces" / "ws_demo"
            ws_dir.mkdir(parents=True)
            (ws_dir / "01_requirements_s1.md").write_text("# Req\n", encoding="utf-8")
            (ws_dir / "08_tasks_s1.md").write_text("# Tasks\n", encoding="utf-8")
            (ws_dir / "05_progress_s1.md").write_text("# Prog\n", encoding="utf-8")
            sol_file = ws_dir / "09_tasks_solution_s1.md"
            sol_file.write_text(md, encoding="utf-8")

            edit_payload = {
                "steps": [
                    {
                        "tool_calls": [
                            {"name": "edit_file", "args": {"Path": str(ws_dir / "05_progress_s1.md"), "Target": "a", "Replacement": "b"}}
                        ]
                    }
                ],
                "toolCall": {
                    "name": "edit_file",
                    "args": {
                        "Path": str(sol_file),
                        "Target": "+func x() { GOOD_FUNC() }",
                        "Replacement": "+func x() { BAD_FUNC() }",
                    },
                },
            }
            res_demo = agy_guard.handle_pre_write(edit_payload, mem_root=mem_root, ws_root=mem_root / "workspaces")
            self.assertEqual(res_demo.get("decision"), "deny")
            self.assertIn("H4t-Demo", res_demo.get("reason", ""))

    def test_f05_r01_atomic_resolve_strict_ref_and_dg2_isolation(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            ws_root = Path(td)
            ws_dir = ws_root / "ws1"
            ws_dir.mkdir(parents=True)
            dec_file = ws_dir / "04_decisions_ws1.md"
            dec_file.write_text("# 04 — Decisions (Append ONLY)\n\n", encoding="utf-8")
            valid_ref_reason = f"Đã đối chiếu REF: {dec_file}:1"
            dg1_path = ws_dir / "06_laya_summary_dg1_20261110T090000.json"
            dg2_path = ws_dir / "06_laya_summary_dg2_20261110T090000.json"
            dg1_data = {
                "stage": "dg1",
                "gate": "FAIL",
                "claims": [
                    {"id": "S-1", "decision": "FLAG", "resolved": None},
                    {"id": "S-2", "decision": "BLOCK", "resolved": None},
                ],
            }
            dg2_data = {
                "stage": "dg2",
                "gate": "NEEDS_RESOLUTION",
                "claims": [
                    {"id": "S-1", "decision": "FLAG", "resolved": None},
                    {"id": "D-1", "decision": "FLAG", "resolved": None},
                ],
            }
            dg1_path.write_text(json.dumps(dg1_data), encoding="utf-8")
            dg2_path.write_text(json.dumps(dg2_data), encoding="utf-8")

            # 1. --id ALL must be rejected with exit 3
            rc_all = laya_gate.cmd_resolve(
                argparse.Namespace(ws_root=str(ws_root), workspace="ws1", stage="dg1", id="ALL", reason=valid_ref_reason)
            )
            self.assertEqual(rc_all, 3)

            # 2. R-01: --reason with 'dòng 10' but NO 'REF: <path>:<line>' must be rejected with exit 3
            rc_no_ref = laya_gate.cmd_resolve(
                argparse.Namespace(ws_root=str(ws_root), workspace="ws1", stage="dg1", id="S-1", reason="Đã xem tại dòng 10")
            )
            self.assertEqual(rc_no_ref, 3)

            # 3. R-01: --reason with REF: pointing to nonexistent file must be rejected with exit 3
            rc_missing_file = laya_gate.cmd_resolve(
                argparse.Namespace(
                    ws_root=str(ws_root), workspace="ws1", stage="dg1", id="S-1", reason="Đã xem REF: /nonexistent/file_xyz.py:10"
                )
            )
            self.assertEqual(rc_missing_file, 3)

            # 4. Atomic check: S-1,S-2 where S-2 is BLOCK -> exit 2 and S-1 MUST NOT be mutated
            rc_mix = laya_gate.cmd_resolve(
                argparse.Namespace(ws_root=str(ws_root), workspace="ws1", stage="dg1", id="S-1,S-2", reason=valid_ref_reason)
            )
            self.assertEqual(rc_mix, 2)
            after_mix = json.loads(dg1_path.read_text(encoding="utf-8"))
            self.assertIsNone(after_mix["claims"][0]["resolved"])

            # 5. Resolve only S-1 -> S-1 in dg2 is synced, but D-1 in dg2 stays FLAG (NEEDS_RESOLUTION)
            dg1_data["claims"] = [{"id": "S-1", "decision": "FLAG", "resolved": None}]
            dg1_path.write_text(json.dumps(dg1_data), encoding="utf-8")
            rc_ok = laya_gate.cmd_resolve(
                argparse.Namespace(ws_root=str(ws_root), workspace="ws1", stage="dg1", id="S-1", reason=valid_ref_reason)
            )
            self.assertEqual(rc_ok, 0)
            after_dg2 = json.loads(dg2_path.read_text(encoding="utf-8"))
            self.assertIsNotNone(after_dg2["claims"][0]["resolved"])
            self.assertIsNone(after_dg2["claims"][1]["resolved"])
            self.assertEqual(after_dg2["gate"], "NEEDS_RESOLUTION")

    def test_f07_n01_r03_bool_control_zero_controls_rejected_and_workers(self) -> None:
        # N-01: 0 control claims MUST be rejected by check_controls
        ok_zero, _, err_zero = laya_gate.check_controls([{"id": "C1", "kind": "claim"}])
        self.assertFalse(ok_zero)
        self.assertIn("control_false", err_zero)

        # R-03: _normalize_evidence_str preserves List[str]
        norm_ev = laya_gate._normalize_evidence_str({"evidence": ["REF: a.py:1-2", "line1"]}, [Path.cwd()])
        self.assertEqual(norm_ev, "REF: a.py:1-2\nline1")

        class _Handler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:
                length = int(self.headers.get("Content-Length", "0"))
                body = json.loads(self.rfile.read(length).decode("utf-8"))
                q = body["questions"]["verdict"]
                opts = list((q.get("criteria") or {}).keys()) or q.get("options") or ["CORRECT", "INCORRECT"]
                st = body["state"]
                is_false_ctrl = "FALSE_CTRL" in json.dumps(st)
                choice = opts[1] if is_false_ctrl else opts[0]
                resp = {"result": {"answers": {"verdict": {"choice": choice, "confidence": 0.95}}}}
                raw = json.dumps(resp).encode("utf-8")
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def log_message(self, format: str, *args: object) -> None:
                return

        srv = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
        t = Thread(target=srv.serve_forever, daemon=True)
        t.start()
        try:
            with tempfile.TemporaryDirectory() as td:
                td_path = Path(td)
                src = td_path / "sample.py"
                src.write_text("x = 1\n", encoding="utf-8")
                # N-01: cmd_run with 0 control claims MUST return exit 3
                zero_ctrl_file = td_path / "claims_zero.json"
                zero_ctrl_file.write_text(
                    json.dumps([{"id": "S-1", "claim": "ok", "evidence": f"REF: {src}:1-1\nx = 1"}]),
                    encoding="utf-8",
                )
                rc_zero = laya_gate.cmd_run(
                    argparse.Namespace(
                        stage="dg1",
                        workspace="ws_test",
                        claims=str(zero_ctrl_file),
                        objective="test",
                        url=f"http://127.0.0.1:{srv.server_port}/v1/predict",
                        ws_root=str(td_path),
                        retries=1,
                        retry_wait=0.1,
                        workers=2,
                    )
                )
                self.assertEqual(rc_zero, 3)

                claims_file = td_path / "claims.json"
                claims_file.write_text(
                    json.dumps(
                        [
                            {"id": "S-1", "type": "claim", "claim": "ok", "evidence": f"REF: {src}:1-1\nx = 1"},
                            {"id": "CF-1", "control": False, "claim": "FALSE_CTRL 1", "evidence": "bad"},
                            {"id": "CF-2", "type": "control_false", "claim": "FALSE_CTRL 2", "evidence": "bad"},
                            {"id": "CT-1", "control": True, "claim": "TRUE_CTRL 1", "evidence": "good"},
                        ]
                    ),
                    encoding="utf-8",
                )
                rc = laya_gate.cmd_run(
                    argparse.Namespace(
                        stage="dg1",
                        workspace="ws_test",
                        claims=str(claims_file),
                        objective="test",
                        url=f"http://127.0.0.1:{srv.server_port}/v1/predict",
                        ws_root=str(td_path),
                        retries=1,
                        retry_wait=0.1,
                        workers=2,
                    )
                )
                self.assertEqual(rc, 0)
        finally:
            srv.shutdown()
            srv.server_close()

    def test_f08_agent_tag_regex_supports_spaces_parens_and_slashes(self) -> None:
        for valid_tag in (
            "- [2026-11-10T09:00:00+07:00] [Agent:Gemini-4-Argon] START task",
            "- [2026-11-10T09:00:00+07:00] [Agent:Claude Opus 4.6 (Thinking)] START task",
            "- [2026-11-10T09:00:00+07:00] [Agent:google/gemini-3.1-pro] DONE task",
        ):
            self.assertIsNotNone(verify_governance.LOG_LINE_RE.match(valid_tag), f"Rejected valid log line: {valid_tag}")
        for invalid_tag in (
            "- [2026-11-10T09:00:00+07:00] [Agent:] START task",
            "- [2026-11-10T09:00:00+07:00] [Agent:   ] START task",
        ):
            self.assertIsNone(verify_governance.LOG_LINE_RE.match(invalid_tag), f"Accepted invalid log line: {invalid_tag}")


if __name__ == "__main__":
    unittest.main()
