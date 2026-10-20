#!/usr/bin/env python3
"""Unit tests for tooling/verify_governance.py (T0-T3 + explicit workspace enforcement)."""
from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import verify_governance  # noqa: E402


class TestVerifyGovernance(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.ws_root = Path(self._tmp.name)
        self.today = "2026-10-16"

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def test_missing_workspace_arg_fails(self) -> None:
        code = verify_governance.main([])
        self.assertEqual(code, 1)

    def test_tier_t0_passes_without_workspace(self) -> None:
        code = verify_governance.main(["--tier", "T0"])
        self.assertEqual(code, 0)

    def test_tier_t1_passes_with_minimal_3_files(self) -> None:
        ws = self.ws_root / "hotfix-1"
        ws.mkdir(parents=True)
        (ws / "01_requirements_hf.md").write_text("tier: T1\n# Req\n", encoding="utf-8")
        (ws / "05_progress_hf.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Muscle] Fix bug\n", encoding="utf-8"
        )
        (ws / "08_tasks_hf.md").write_text("- [x] Task 1\n", encoding="utf-8")

        ok = verify_governance.verify_workspace_docs(
            "hotfix-1", workspaces_dir=self.ws_root, today_override=self.today
        )
        self.assertTrue(ok)

    def test_tier_t2_fails_when_missing_09_06_11_then_passes(self) -> None:
        ws = self.ws_root / "feat-2"
        ws.mkdir(parents=True)
        (ws / "01_requirements_f2.md").write_text("tier: T2\n# Req\n", encoding="utf-8")
        (ws / "05_progress_f2.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Brain] Start\n", encoding="utf-8"
        )
        (ws / "08_tasks_f2.md").write_text("- [x] Task 1\n", encoding="utf-8")

        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "feat-2", workspaces_dir=self.ws_root, today_override=self.today
            )
        )

        (ws / "09_tasks_solution_f2.md").write_text("# Solution\n", encoding="utf-8")
        (ws / "06_validation_f2.md").write_text("# Validation\n", encoding="utf-8")
        (ws / "11_report_f2.md").write_text("# Report\n", encoding="utf-8")
        (ws / "audit_report_f2.md").write_text(
            "# Final Adversarial Audit Report\n"
            "## 1. Line-by-Line git diff & task critique\n"
            "## 2. Anti-Smoke-Speculation: no mockup, no hallucination, runtime trace\n"
            "## 3. Concurrency, Multi-worker, Timeout/Heartbeat AckWait & Watermark Continuity\n"
            "## 4. Anti-Speculation & Honesty check (không suy diễn, không báo cáo láo)\n"
            "## 5. Self-Improvement Loop & lessons\n"
            "## 6. /laya_router DG3 verdict PASS\n",
            encoding="utf-8",
        )

        self.assertTrue(
            verify_governance.verify_workspace_docs(
                "feat-2", workspaces_dir=self.ws_root, today_override=self.today
            )
        )

    def test_incomplete_audit_report_fails_in_t2_s6(self) -> None:
        ws = self.ws_root / "feat-bad-audit"
        ws.mkdir(parents=True)
        (ws / "01_requirements_ba.md").write_text("tier: T2\n# Req\n", encoding="utf-8")
        (ws / "05_progress_ba.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Brain] Start\n", encoding="utf-8"
        )
        (ws / "08_tasks_ba.md").write_text("- [x] Task 1\n", encoding="utf-8")
        (ws / "09_tasks_solution_ba.md").write_text("# Solution\n", encoding="utf-8")
        (ws / "06_validation_ba.md").write_text("# Validation\n", encoding="utf-8")
        (ws / "11_report_ba.md").write_text("# Report\n", encoding="utf-8")
        # Incomplete audit report missing sections 2-6
        (ws / "audit_report_ba.md").write_text(
            "# Audit\n## 1. Line-by-Line git diff\n", encoding="utf-8"
        )

        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "feat-bad-audit", workspaces_dir=self.ws_root, today_override=self.today
            )
        )

    def test_malformed_progress_log_fails(self) -> None:
        ws = self.ws_root / "bad-log"
        ws.mkdir(parents=True)
        (ws / "01_requirements_b.md").write_text("tier: T1\n", encoding="utf-8")
        (ws / "05_progress_b.md").write_text(
            f"- [{self.today}] Missing agent tag\n", encoding="utf-8"
        )
        (ws / "08_tasks_b.md").write_text("- [x] T\n", encoding="utf-8")

        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "bad-log", workspaces_dir=self.ws_root, today_override=self.today
            )
        )

    def test_stage_s3_and_explicit_suffix_enforcement(self) -> None:
        ws = self.ws_root / "reused-ws"
        ws.mkdir(parents=True)
        (ws / "01_requirements_v1.md").write_text("tier: T2\n", encoding="utf-8")
        (ws / "05_progress_v1.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Brain] Start v2\n", encoding="utf-8"
        )
        (ws / "08_tasks_v1.md").write_text("- [x] Old task\n", encoding="utf-8")
        (ws / "09_tasks_solution_v2.md").write_text("# New Solution v2\n", encoding="utf-8")

        # Missing 01_requirements_v2.md and 08_tasks_v2.md for --suffix v2
        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "reused-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="v2",
                stage="S3",
            )
        )

        # Create 01_requirements_v2.md and 08_tasks_v2.md -> Stage S3 passes without 06/11
        (ws / "01_requirements_v2.md").write_text("tier: T2\n", encoding="utf-8")
        (ws / "08_tasks_v2.md").write_text("- [ ] New task v2\n", encoding="utf-8")
        self.assertTrue(
            verify_governance.verify_workspace_docs(
                "reused-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="v2",
                stage="S3",
            )
        )

    def test_reused_workspace_piggyback_fails_without_new_01_08(self) -> None:
        import os

        ws = self.ws_root / "piggyback-ws"
        ws.mkdir(parents=True)
        for name in (
            "01_requirements_old.md",
            "08_tasks_old.md",
            "09_tasks_solution_old.md",
            "06_validation_old.md",
            "11_report_old.md",
            "audit_report_old.md",
        ):
            p = ws / name
            p.write_text("tier: T2\n# Old\n", encoding="utf-8")
            os.utime(p, (1000000, 1000000))

        (ws / "05_progress_old.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Brain] Log\n", encoding="utf-8"
        )
        # Agent adds a new 09_tasks_solution_new.md without creating/updating 01 or 08
        sol_new = ws / "09_tasks_solution_new.md"
        sol_new.write_text("# New Solution\n", encoding="utf-8")
        os.utime(sol_new, (2000000, 2000000))

        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "piggyback-ws", workspaces_dir=self.ws_root, today_override=self.today
            )
        )

    def test_clean_room_s3_and_s6_enforcement(self) -> None:
        import json

        ws = self.ws_root / "clean-room-ws"
        ws.mkdir(parents=True)
        (ws / "01_requirements_cr.md").write_text(
            "tier: T2\nmode: clean-room\n", encoding="utf-8"
        )
        (ws / "05_progress_cr.md").write_text(
            f"- [{self.today}T10:00:00+07:00] [Agent:Brain] Start clean-room\n",
            encoding="utf-8",
        )
        (ws / "08_tasks_cr.md").write_text("- [x] Task\n", encoding="utf-8")
        sol = ws / "09_tasks_solution_cr.md"
        sol.write_text("# Solution without approved spec header\n", encoding="utf-8")

        # S3 fails without Clean-Room Approved Spec
        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "clean-room-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="cr",
                stage="S3",
            )
        )

        # Add Clean-Room Approved Spec -> S3 passes
        sol.write_text("# Solution\n## Clean-Room Approved Spec\n- CR-01\n", encoding="utf-8")
        self.assertTrue(
            verify_governance.verify_workspace_docs(
                "clean-room-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="cr",
                stage="S3",
            )
        )

        # Prepare S6 docs without 06_mutation_summary_*.json -> S6 fails
        (ws / "06_validation_cr.md").write_text("# Val\n", encoding="utf-8")
        (ws / "11_report_cr.md").write_text("# Rep\n", encoding="utf-8")
        (ws / "audit_report_cr.md").write_text(
            "# Audit\n"
            "1. Line-by-Line git diff\n"
            "2. No mockup, runtime trace\n"
            "3. Concurrency, timeout, watermark continuity\n"
            "4. Honesty check, không suy diễn\n"
            "5. Self-Improvement Loop lessons\n"
            "6. laya_router dg3 PASS\n",
            encoding="utf-8",
        )
        self.assertFalse(
            verify_governance.verify_workspace_docs(
                "clean-room-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="cr",
                stage="S6",
            )
        )

        # Add passing 06_mutation_summary_cr_*.json -> S6 passes
        (ws / "06_mutation_summary_cr_20261020T120000.json").write_text(
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
        self.assertTrue(
            verify_governance.verify_workspace_docs(
                "clean-room-ws",
                workspaces_dir=self.ws_root,
                today_override=self.today,
                suffix="cr",
                stage="S6",
            )
        )


if __name__ == "__main__":
    unittest.main()
