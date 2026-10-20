#!/usr/bin/env python3
"""Unit & fixture tests for hooks/agy/agy_guard.py (16 scenarios, H1-H7)."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import agy_guard  # noqa: E402


class TestAgyGuard(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = Path(self._tmp.name)
        self.mem_root = self.tmp / "memory"
        self.ws_root = self.mem_root / "workspaces"
        self.state_dir = self.tmp / "state"
        self.ws_root.mkdir(parents=True, exist_ok=True)
        self.state_dir.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def _write_transcript(self, lines: list[dict]) -> str:
        tr = self.tmp / "transcript.jsonl"
        tr.write_text("\n".join(json.dumps(x) for x in lines) + "\n", encoding="utf-8")
        return str(tr)

    # 1. H2: Overwrite existing memory file -> deny
    def test_01_h2_overwrite_existing_memory_denied(self) -> None:
        lessons = self.mem_root / "global" / "lessons.md"
        lessons.parent.mkdir(parents=True, exist_ok=True)
        lessons.write_text("# Lessons\n", encoding="utf-8")
        res = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(lessons), "Overwrite": True, "CodeContent": "x"},
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("Rule 17", res["reason"])

    # 2. H2: Append-only file without preserving TargetContent -> deny
    def test_02_h2_append_only_strip_target_denied(self) -> None:
        prog = self.ws_root / "demo" / "05_progress.md"
        prog.parent.mkdir(parents=True, exist_ok=True)
        prog.write_text("- [2026-10-16] [Agent:Brain] Init\n", encoding="utf-8")
        res = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {
                        "TargetFile": str(prog),
                        "TargetContent": "old line",
                        "ReplacementContent": "new line without old",
                    },
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("append-only", res["reason"])

    # 3. H2: Append-only file preserving TargetContent -> ask
    def test_03_h2_append_only_preserving_target_allowed(self) -> None:
        prog = self.ws_root / "demo" / "05_progress.md"
        prog.parent.mkdir(parents=True, exist_ok=True)
        prog.write_text("old line\n", encoding="utf-8")
        res = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {
                        "TargetFile": str(prog),
                        "TargetContent": "old line",
                        "ReplacementContent": "old line\n- [2026-10-16] [Agent:Brain] Added",
                    },
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "ask")

    # 4. H3: Unmasked secret/connection URI in memory file -> deny
    def test_04_h3_unmasked_secret_in_memory_denied(self) -> None:
        prog = self.ws_root / "demo" / "05_progress.md"
        res = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "write_to_file",
                    "args": {
                        "TargetFile": str(prog),
                        "Overwrite": False,
                        "CodeContent": "DB URI: postgresql://admin:123456@db:5432/prod",
                    },
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("Rule 20", res["reason"])

    # 5. H3: Masked URI in memory file -> ask
    def test_05_h3_masked_secret_in_memory_allowed(self) -> None:
        prog = self.ws_root / "demo" / "05_progress.md"
        res = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "write_to_file",
                    "args": {
                        "TargetFile": str(prog),
                        "Overwrite": False,
                        "CodeContent": "DB URI: postgresql://admin:***@db:5432/prod",
                    },
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "ask")

    # 6. H4: Source edit without workspace -> deny
    def test_06_h4_source_edit_without_workspace_denied(self) -> None:
        tr = self._write_transcript([])
        src_file = self.tmp / "app" / "service.go"
        res = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "package main"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("Rule 4/16", res["reason"])

    # 7. H4: Source edit when DG1 is BLOCK -> deny
    def test_07_h4_source_edit_dg1_block_denied(self) -> None:
        ws_dir = self.ws_root / "feat-x"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "BLOCK"}), encoding="utf-8"
        )
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "05_progress_x.md")},
                        }
                    ]
                }
            ]
        )
        src_file = self.tmp / "app" / "service.go"
        res = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "package main"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("DG1", res["reason"])

    # 8. H4: Source edit with DG1=PASS + 05_progress but no User APPROVE -> force_ask
    def test_08_h4_source_edit_without_approve_force_ask(self) -> None:
        ws_dir = self.ws_root / "feat-x"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "05_progress_x.md")},
                        },
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "09_tasks_solution_x.md")},
                        },
                    ]
                },
                {"type": "USER_INPUT", "content": "giải thích thêm chỗ này"},
            ]
        )
        src_file = self.tmp / "app" / "service.go"
        res = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "package main"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "force_ask")
        self.assertIn("Rule 13", res["reason"])

    # 9. H4: Source edit with DG1=PASS + 05_progress + User APPROVE -> ask
    def test_09_h4_source_edit_with_approve_allowed(self) -> None:
        ws_dir = self.ws_root / "feat-x"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "05_progress_x.md")},
                        },
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "09_tasks_solution_x.md")},
                        },
                    ]
                },
                {"type": "USER_INPUT", "content": "ok duyệt làm đi"},
            ]
        )
        src_file = self.tmp / "app" / "service.go"
        res = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "package main"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res["decision"], "ask")

    # 10. H5: git push -> deny
    def test_10_h5_git_push_denied(self) -> None:
        res = agy_guard.handle_pre_command(
            {"toolCall": {"name": "run_command", "args": {"CommandLine": "git push origin main"}}}
        )
        self.assertEqual(res["decision"], "deny")
        self.assertIn("Rule 15", res["reason"])

    # 11. H5: direct DB write CLI -> force_ask
    def test_11_h5_direct_db_delete_force_ask(self) -> None:
        res = agy_guard.handle_pre_command(
            {
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "psql -c 'DELETE FROM orders'"},
                }
            }
        )
        self.assertEqual(res["decision"], "force_ask")
        self.assertIn("Rule 12", res["reason"])

    # 12. H7: Stop missing component runtime_check -> continue
    def test_12_h7_stop_missing_runtime_check_continues(self) -> None:
        proj = self.tmp / "proj"
        proj.mkdir(parents=True, exist_ok=True)
        (proj / ".agent-project.json").write_text(
            json.dumps(
                {
                    "version": 1,
                    "components": [
                        {
                            "name": "api-svc",
                            "path": ".",
                            "checks": ["go test ./..."],
                            "runtime_check": "curl -sf http://127.0.0.1:18080/health",
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )
        src_file = proj / "service.go"
        src_file.write_text("package main\n", encoding="utf-8")
        ws_dir = self.ws_root / "feat-x"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg3_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {
                            "name": "replace_file_content",
                            "args": {"TargetFile": str(src_file)},
                        }
                    ]
                },
                {
                    "tool_calls": [
                        {
                            "name": "replace_file_content",
                            "args": {"TargetFile": str(ws_dir / "05_progress_x.md")},
                        },
                        {
                            "name": "write_to_file",
                            "args": {"TargetFile": str(ws_dir / "06_validation_x.md")},
                        },
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c1", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res.get("decision"), "continue")
        self.assertIn("api-svc", res.get("reason", ""))

    # 13. H7: Stop 3rd time when counter >= 2 -> {} (loop breaker)
    def test_13_h7_stop_third_time_loop_breaker(self) -> None:
        (self.state_dir / "c2.stop").write_text("2", encoding="utf-8")
        res = agy_guard.handle_stop(
            {"conversationId": "c2", "transcriptPath": ""},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res, {})

    # 14. H7: Stop on T0 (no source edits, no solution/audit files) -> {}
    def test_14_h7_stop_tier_t0_passes(self) -> None:
        tr = self._write_transcript(
            [{"type": "USER_INPUT", "content": "giải thích rule 0"}]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c3", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res, {})

    # 15. H6/H7: Repo without .agent-project.json -> graceful no-op
    def test_15_repo_without_manifest_graceful(self) -> None:
        orphan = self.tmp / "no_manifest" / "main.py"
        orphan.parent.mkdir(parents=True, exist_ok=True)
        orphan.write_text("print(1)\n", encoding="utf-8")
        root, comp = agy_guard.component_for(str(orphan))
        self.assertIsNone(root)
        self.assertIsNone(comp)

    # 16. H6/H7: Nested components in .agent-project.json -> longest path wins
    def test_16_nested_components_longest_prefix_match(self) -> None:
        proj = self.tmp / "monorepo"
        api_dir = proj / "services" / "api"
        api_dir.mkdir(parents=True, exist_ok=True)
        (proj / ".agent-project.json").write_text(
            json.dumps(
                {
                    "version": 1,
                    "components": [
                        {"name": "root", "path": ".", "checks": ["echo root"]},
                        {"name": "api", "path": "services/api", "checks": ["echo api"]},
                    ],
                }
            ),
            encoding="utf-8",
        )
        f = api_dir / "handler.go"
        f.write_text("package api\n", encoding="utf-8")
        _, comp = agy_guard.component_for(str(f))
        self.assertIsNotNone(comp)
        self.assertEqual(comp["name"], "api")

    # 17. Quoted TargetFile in real Antigravity transcript.jsonl -> properly unquoted & detected
    def test_17_quoted_target_file_in_transcript_detected(self) -> None:
        src_file = self.tmp / "app" / "handler.go"
        quoted_path = json.dumps(str(src_file))  # "\"/path/to/handler.go\""
        st = [
            {
                "tool_calls": [
                    {
                        "name": "replace_file_content",
                        "args": {"TargetFile": quoted_path},
                    }
                ]
            }
        ]
        edits = agy_guard.src_edits(st, mem_root=self.mem_root)
        self.assertEqual(len(edits), 1)
        self.assertEqual(edits[0][1], str(src_file))

    # 18. H1: Boot injects once per USER_INPUT turn and re-injects on a new USER_INPUT turn
    def test_18_h1_boot_reinjects_on_new_user_turn(self) -> None:
        tr1 = self._write_transcript([{"type": "USER_INPUT", "content": "task 1"}])
        r1 = agy_guard.handle_pre_invocation(
            {"conversationId": "c18", "transcriptPath": tr1}, state_dir=self.state_dir
        )
        self.assertIn("injectSteps", r1)

        # Second invocation in the same user turn -> no-op
        r2 = agy_guard.handle_pre_invocation(
            {"conversationId": "c18", "transcriptPath": tr1}, state_dir=self.state_dir
        )
        self.assertEqual(r2, {})

        # New USER_INPUT turn -> re-injects BOOT_MSG
        tr2 = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "task 1"},
                {"type": "PLANNER_RESPONSE", "content": "done 1"},
                {"type": "USER_INPUT", "content": "task 2"},
            ]
        )
        r3 = agy_guard.handle_pre_invocation(
            {"conversationId": "c18", "transcriptPath": tr2}, state_dir=self.state_dir
        )
        self.assertIn("injectSteps", r3)

    # 19. H4a: S1 Intake Gate blocks creating 09_tasks_solution_<suffix>.md without 01/08/05_progress
    def test_19_h4_s1_gate_blocks_solution_without_req_tasks_progress(self) -> None:
        ws_dir = self.ws_root / "feat-s1"
        ws_dir.mkdir(parents=True, exist_ok=True)
        sol_file = ws_dir / "09_tasks_solution_v2.md"
        tr_empty = self._write_transcript([{"type": "USER_INPUT", "content": "fix bug"}])

        # Missing 01, 08, and 05_progress -> deny
        res1 = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr_empty,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(sol_file), "Overwrite": False, "CodeContent": "# Sol"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res1["decision"], "deny")
        self.assertIn("Rule 9", res1["reason"])

        # After creating 01, 08 and touching 05_progress in current turn -> ask
        (ws_dir / "01_requirements_v2.md").write_text("tier: T2\n", encoding="utf-8")
        (ws_dir / "08_tasks_v2.md").write_text("- [ ] T1\n", encoding="utf-8")
        tr_ok = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "fix bug"},
                {
                    "tool_calls": [
                        {
                            "name": "replace_file_content",
                            "args": {"TargetFile": json.dumps(str(ws_dir / "05_progress_v2.md"))},
                        }
                    ]
                },
            ]
        )
        res2 = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr_ok,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(sol_file), "Overwrite": False, "CodeContent": "# Sol"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res2["decision"], "ask")

    # 20. H7: Blocks Shadow Analysis (>=4 project file reads without workspace docs)
    def test_20_h7_blocks_shadow_analysis_without_workspace_docs(self) -> None:
        proj = self.tmp / "data-hub"
        tr = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "check bug luồng đối soát"},
                {
                    "tool_calls": [
                        {"name": "view_file", "args": {"AbsolutePath": str(proj / "a.go")}},
                        {"name": "view_file", "args": {"AbsolutePath": str(proj / "b.go")}},
                        {"name": "view_file", "args": {"AbsolutePath": str(proj / "c.go")}},
                        {"name": "grep_search", "args": {"SearchPath": str(proj / "d.go")}},
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c20", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res.get("decision"), "continue")
        self.assertIn("No Shadow Files", res.get("reason", ""))

    # 21. H4: Denies production mock closure pollution (*Fn func, sqlmock) in prod code, allows in test files
    def test_21_h4_denies_production_mock_pollution_allows_in_test_files(self) -> None:
        prod_file = self.tmp / "app" / "service.go"
        res_prod = agy_guard.handle_pre_write(
            {
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {
                        "TargetFile": str(prod_file),
                        "ReplacementContent": "type Service struct { lookupMasterTargetsFn func() error }",
                    },
                }
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_prod["decision"], "deny")
        self.assertIn("Anti-Smoke-Speculation", res_prod["reason"])

        # Same pattern inside a *_test.go file -> not denied by mock pollution check
        ws_dir = self.ws_root / "feat-x"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "05_progress_x.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "09_tasks_solution_x.md")}},
                    ]
                },
                {"type": "USER_INPUT", "content": "APPROVE"},
            ]
        )
        test_file = self.tmp / "app" / "service_test.go"
        res_test = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {
                        "TargetFile": str(test_file),
                        "ReplacementContent": "type mockSvc struct { runEventsFn func() error }",
                    },
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_test["decision"], "ask")

    # 22. H6: Auto-injects FINAL_AUDIT_MSG whenever source code is edited
    def test_22_h6_auto_injects_final_audit_msg_on_source_edit(self) -> None:
        src_file = self.tmp / "app" / "worker.go"
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}
                    ]
                }
            ]
        )
        res = agy_guard.handle_post_invocation(
            {"conversationId": "c22", "transcriptPath": tr},
            mem_root=self.mem_root,
            state_dir=self.state_dir,
        )
        self.assertIn("injectSteps", res)
        msg = res["injectSteps"][0]["ephemeralMessage"]
        self.assertIn("MANDATORY FINAL ADVERSARIAL AUDIT", msg)
        self.assertIn("worker.go", msg)

    # 23. H7: Blocks when audit_report_<suffix>.md is missing or incomplete after source edits
    def test_23_h7_blocks_missing_or_incomplete_audit_report(self) -> None:
        ws_dir = self.ws_root / "feat-audit"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg3_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        # Write an incomplete audit_report (missing sections 2-6)
        audit_file = ws_dir / "audit_report_v1.md"
        audit_file.write_text("# Audit\n## 1. Line-by-Line git diff\n", encoding="utf-8")

        src_file = self.tmp / "app" / "worker.go"
        tr = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_v1.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "06_validation_v1.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(audit_file)}},
                        {
                            "name": "run_command",
                            "args": {"CommandLine": "python3 laya_gate.py run --stage dg3"},
                        },
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c23", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res.get("decision"), "continue")
        self.assertIn("thiếu nội dung bắt buộc", res.get("reason", ""))

    # 24. H7: Passes when audit_report_<suffix>.md has all 6 required sections and post-edit DG3 PASS
    def test_24_h7_passes_with_complete_audit_report_and_post_edit_dg3(self) -> None:
        ws_dir = self.ws_root / "feat-audit-ok"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg3_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        audit_file = ws_dir / "audit_report_v1.md"
        audit_file.write_text(
            "# Final Adversarial Audit Report\n"
            "## 1. Line-by-Line git diff & task critique\n"
            "## 2. Anti-Smoke-Speculation: no mockup, no hallucination, runtime trace verified\n"
            "## 3. Concurrency, Multi-worker, Timeout/Heartbeat AckWait & Watermark Continuity\n"
            "## 4. Anti-Speculation & Honesty check (không suy diễn, không báo cáo láo)\n"
            "## 5. Self-Improvement Loop & lessons check\n"
            "## 6. /laya_router DG3 verdict PASS\n",
            encoding="utf-8",
        )
        src_file = self.tmp / "app" / "worker.go"
        tr = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_v1.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "06_validation_v1.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(audit_file)}},
                        {
                            "name": "run_command",
                            "args": {"CommandLine": "python3 laya_gate.py run --stage dg3"},
                        },
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c24", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res, {})

    # 25. H4a: Clean-Room mode blocks 09_tasks_solution without Clean-Room Approved Spec, allows when present, and preserves replace_file_content partial edits (F1)
    def test_25_h4a_clean_room_requires_approved_spec_and_supports_partial_replace(self) -> None:
        ws_dir = self.ws_root / "cr-ws"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "01_requirements_cr1.md").write_text(
            "tier: T2\nmode: clean-room\n", encoding="utf-8"
        )
        (ws_dir / "08_tasks_cr1.md").write_text("- [ ] Task 1\n", encoding="utf-8")
        sol_file = ws_dir / "09_tasks_solution_cr1.md"
        tr = self._write_transcript(
            [
                {"type": "USER_INPUT", "content": "Rebuild feature"},
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_cr1.md")}}
                    ]
                },
            ]
        )
        # Missing Clean-Room Approved Spec -> deny
        res_deny = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(sol_file), "CodeContent": "# Solution without spec\n"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_deny.get("decision"), "deny")
        self.assertIn("Clean-Room Approved Spec", res_deny.get("reason", ""))

        # With Clean-Room Approved Spec -> ask
        sol_content = "# Solution\n## Clean-Room Approved Spec\n- CR-01\n## Footer\nOld footer\n"
        res_ok = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(sol_file), "CodeContent": sol_content},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_ok.get("decision"), "ask")

        # Partial replace_file_content on existing file (F1) -> ask
        sol_file.write_text(sol_content, encoding="utf-8")
        res_partial = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {
                        "TargetFile": str(sol_file),
                        "TargetContent": "Old footer",
                        "ReplacementContent": "New footer",
                    },
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_partial.get("decision"), "ask")

    # 26. H4b: Clean-Room mode blocks direct source edit if session read clean_room_forbidden_sources (F2)
    def test_26_h4b_clean_room_blocks_contaminated_session_from_writing_source(self) -> None:
        ws_dir = self.ws_root / "cr-iso"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "01_requirements_iso.md").write_text(
            "tier: T2\nmode: clean-room\nclean_room_forbidden_sources: legacy/old_payment.py\n",
            encoding="utf-8",
        )
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        sol_file = ws_dir / "09_tasks_solution_iso.md"
        src_file = self.tmp / "app" / "new_payment.py"
        tr = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {"name": "view_file", "args": {"AbsolutePath": "/repo/legacy/old_payment.py"}},
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_iso.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(sol_file)}},
                    ]
                },
                {"type": "USER_INPUT", "content": "APPROVE"},
            ]
        )
        res = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(src_file), "CodeContent": "def pay(): return True\n"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res.get("decision"), "deny")
        self.assertIn("Clean-Room Isolation", res.get("reason", ""))

    # 27. H7: Clean-Room mode blocks stop when mutation_gate summary is missing or stale (F6)
    def test_27_h7_clean_room_blocks_without_fresh_mutation_summary(self) -> None:
        ws_dir = self.ws_root / "cr-stop"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "01_requirements_cr2.md").write_text(
            "tier: T2\nmode: clean-room\n", encoding="utf-8"
        )
        (ws_dir / "06_laya_summary_dg3_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        audit_file = ws_dir / "audit_report_cr2.md"
        audit_file.write_text(
            "# Final Adversarial Audit Report\n"
            "## 1. Line-by-Line git diff & task critique\n"
            "## 2. Anti-Smoke-Speculation: no mockup, no hallucination, runtime trace verified\n"
            "## 3. Concurrency, Multi-worker, Timeout/Heartbeat AckWait & Watermark Continuity\n"
            "## 4. Anti-Speculation & Honesty check (không suy diễn, không báo cáo láo)\n"
            "## 5. Self-Improvement Loop & lessons check\n"
            "## 6. /laya_router DG3 verdict PASS\n",
            encoding="utf-8",
        )
        src_file = self.tmp / "app" / "service.py"
        tr = self._write_transcript(
            [
                {"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "09_tasks_solution_cr2.md")}}]},
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_cr2.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "06_validation_cr2.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(audit_file)}},
                        {"name": "run_command", "args": {"CommandLine": "python3 laya_gate.py run --stage dg3"}},
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c27", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res.get("decision"), "continue")
        self.assertIn("Mutation Gate", res.get("reason", ""))

    # 28. H7: Clean-Room mode passes when post-edit mutation_gate summary is PASS with >=3 killed
    def test_28_h7_clean_room_passes_with_post_edit_mutation_summary(self) -> None:
        ws_dir = self.ws_root / "cr-pass"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "01_requirements_cr3.md").write_text(
            "tier: T2\nmode: clean-room\n", encoding="utf-8"
        )
        (ws_dir / "06_laya_summary_dg3_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        (ws_dir / "06_mutation_summary_cr3_20261020T120000.json").write_text(
            json.dumps(
                {
                    "gate": "PASS",
                    "total_mutants": 3,
                    "killed_count": 3,
                    "survived_count": 0,
                    "all_restored_sha256": True,
                }
            ),
            encoding="utf-8",
        )
        audit_file = ws_dir / "audit_report_cr3.md"
        audit_file.write_text(
            "# Final Adversarial Audit Report\n"
            "## 1. Line-by-Line git diff & task critique\n"
            "## 2. Anti-Smoke-Speculation: no mockup, no hallucination, runtime trace verified\n"
            "## 3. Concurrency, Multi-worker, Timeout/Heartbeat AckWait & Watermark Continuity\n"
            "## 4. Anti-Speculation & Honesty check (không suy diễn, không báo cáo láo)\n"
            "## 5. Self-Improvement Loop & lessons check\n"
            "## 6. /laya_router DG3 verdict PASS\n",
            encoding="utf-8",
        )
        src_file = self.tmp / "app" / "service.py"
        tr = self._write_transcript(
            [
                {"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "09_tasks_solution_cr3.md")}}]},
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {
                    "tool_calls": [
                        {"name": "run_command", "args": {"CommandLine": "python3 mutation_gate.py --workspace cr-pass"}},
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_cr3.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(ws_dir / "06_validation_cr3.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(audit_file)}},
                        {"name": "run_command", "args": {"CommandLine": "python3 laya_gate.py run --stage dg3"}},
                    ]
                },
            ]
        )
        res = agy_guard.handle_stop(
            {"conversationId": "c28", "transcriptPath": tr},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
            state_dir=self.state_dir,
            run_linter=False,
        )
        self.assertEqual(res, {})

    # 29. H4b: Closed-Loop S5 -> S3 Re-Plan blocks source edit after audit_report_ until 09_tasks_solution is updated and re-approved
    def test_29_h4b_blocks_source_edit_after_s5_audit_until_s3_replan_and_new_approve(self) -> None:
        ws_dir = self.ws_root / "feat-replan"
        ws_dir.mkdir(parents=True, exist_ok=True)
        (ws_dir / "06_laya_summary_dg1_test.json").write_text(
            json.dumps({"gate": "PASS"}), encoding="utf-8"
        )
        sol_file = ws_dir / "09_tasks_solution_rp.md"
        aud_file = ws_dir / "audit_report_rp.md"
        src_file = self.tmp / "app" / "service.py"

        # Case A: audit_report_ written AFTER 09_tasks_solution -> direct source edit denied
        tr_locked = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_rp.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(sol_file)}},
                    ]
                },
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": str(aud_file)}}]},
            ]
        )
        res_locked = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr_locked,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "x = 2\n"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_locked.get("decision"), "deny")
        self.assertIn("S5 -> S3 Re-Plan Loop", res_locked.get("reason", ""))

        # Case B: Agent updates 09_tasks_solution AFTER audit_report_, and User sends new APPROVE -> ask
        tr_replanned = self._write_transcript(
            [
                {
                    "tool_calls": [
                        {"name": "replace_file_content", "args": {"TargetFile": str(ws_dir / "05_progress_rp.md")}},
                        {"name": "write_to_file", "args": {"TargetFile": str(sol_file)}},
                    ]
                },
                {"type": "USER_INPUT", "content": "APPROVE"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]},
                {"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": str(aud_file)}}]},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(sol_file)}}]},
                {"type": "USER_INPUT", "content": "APPROVE"},
            ]
        )
        res_unlocked = agy_guard.handle_pre_write(
            {
                "transcriptPath": tr_replanned,
                "toolCall": {
                    "name": "replace_file_content",
                    "args": {"TargetFile": str(src_file), "ReplacementContent": "x = 3\n"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res_unlocked.get("decision"), "ask")

    # 30. H6: Loop prevention counter escalates on >=3 consecutive check failures and resets to 0 on pass
    def test_30_h6_loop_prevention_escalates_on_three_consecutive_failures_and_resets_on_pass(self) -> None:
        proj = self.tmp / "loop_proj"
        proj.mkdir(parents=True, exist_ok=True)
        flag_file = proj / "pass.flag"
        (proj / ".agent-project.json").write_text(
            json.dumps(
                {
                    "version": 1,
                    "components": [
                        {
                            "name": "worker",
                            "path": ".",
                            "checks": [f"test -f {flag_file}"],
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )
        src_file = proj / "worker.py"
        src_file.write_text("x = 1\n", encoding="utf-8")

        steps_list = [{"type": "USER_INPUT", "content": "APPROVE"}]
        for attempt in range(1, 4):
            steps_list.append(
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]}
            )
            tr = self._write_transcript(steps_list)
            res = agy_guard.handle_post_invocation(
                {"conversationId": "c30", "transcriptPath": tr},
                mem_root=self.mem_root,
                state_dir=self.state_dir,
            )
            msg = res["injectSteps"][0]["ephemeralMessage"]
            if attempt < 3:
                self.assertNotIn("H6 EXECUTION MONITOR", msg)
            else:
                self.assertIn("H6 EXECUTION MONITOR", msg)
                self.assertIn("Rule #15 Escalation", msg)

        # Now make the check pass -> loop counter resets to 0
        flag_file.write_text("ok", encoding="utf-8")
        steps_list.append(
            {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src_file)}}]}
        )
        tr_pass = self._write_transcript(steps_list)
        res_pass = agy_guard.handle_post_invocation(
            {"conversationId": "c30", "transcriptPath": tr_pass},
            mem_root=self.mem_root,
            state_dir=self.state_dir,
        )
        msg_pass = res_pass["injectSteps"][0]["ephemeralMessage"]
        self.assertNotIn("H6 EXECUTION MONITOR", msg_pass)
        self.assertEqual((self.state_dir / "c30.0.loop").read_text(encoding="utf-8"), "0")


if __name__ == "__main__":
    unittest.main()
