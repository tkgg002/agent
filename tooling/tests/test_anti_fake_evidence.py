#!/usr/bin/env python3
"""Tests anti_fake_evidence: evidence_run.py + mutation_gate --base."""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import evidence_run  # noqa: E402
import mutation_gate  # noqa: E402


def _git(repo: Path, *args: str) -> None:
    subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True)


class TestEvidenceRun(unittest.TestCase):
    def setUp(self) -> None:
        self._t = tempfile.TemporaryDirectory()
        self.root = Path(self._t.name)
        (self.root / "ws" / "w1").mkdir(parents=True)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        _git(self.repo, "init", "-q")
        (self.repo / "a.py").write_text("X = 1\n", encoding="utf-8")

    def tearDown(self) -> None:
        self._t.cleanup()

    def _evs(self) -> list:
        return sorted((self.root / "ws" / "w1" / "06_evidence").glob("EV-*.json"))

    def test_records_real_exit_and_output(self) -> None:
        rc = evidence_run.run("w1", "unit", ["python3", "-c", "'print(\"Ran 7 tests\"); raise SystemExit(1)'"], self.repo, self.root / "ws")
        self.assertEqual(rc, 1)
        rec = json.loads(self._evs()[0].read_text(encoding="utf-8"))
        self.assertEqual(rec["exit"], 1)
        self.assertIn("Ran 7 tests", rec["stdout_tail"])
        self.assertEqual(rec["kind"], "unit")
        self.assertRegex(rec["id"], r"^EV-\d{8}T\d{6}-[0-9a-f]{4}$")

    def test_runtime_kind_rejects_test_runner(self) -> None:
        rc = evidence_run.run("w1", "runtime", ["python3", "-m", "unittest"], self.repo, self.root / "ws")
        self.assertEqual(rc, 3)
        self.assertEqual(self._evs(), [])

    def test_fingerprint_stable_across_commit_changes_on_edit(self) -> None:
        f1 = evidence_run.source_fingerprint(self.repo)
        _git(self.repo, "add", "-A")
        _git(self.repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "x")
        self.assertEqual(evidence_run.source_fingerprint(self.repo), f1)
        (self.repo / "a.py").write_text("X = 2\n", encoding="utf-8")
        self.assertNotEqual(evidence_run.source_fingerprint(self.repo), f1)


class TestDiffAnchor(unittest.TestCase):
    def setUp(self) -> None:
        self._t = tempfile.TemporaryDirectory()
        self.repo = Path(self._t.name)
        _git(self.repo, "init", "-q")
        (self.repo / "m.py").write_text("def f():\n    return 1\n\ndef g():\n    return 2\n", encoding="utf-8")
        (self.repo / "n.py").write_text("Y = 0\n", encoding="utf-8")
        _git(self.repo, "add", "-A")
        _git(self.repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "base")
        (self.repo / "m.py").write_text("def f():\n    return 1\n\ndef g():\n    return 20\n", encoding="utf-8")
        (self.repo / "n.py").write_text("Y = 5\n", encoding="utf-8")

    def tearDown(self) -> None:
        self._t.cleanup()

    def _item(self, file: str, target: str) -> dict:
        return {"id": "M", "fpath": self.repo / file, "target_content": target}

    def test_mutant_on_unchanged_line_rejected(self) -> None:
        issues = mutation_gate.diff_anchor_issues([self._item("m.py", "    return 1"), self._item("n.py", "Y = 5")], "HEAD")
        self.assertTrue(any("không chạm" in i for i in issues))

    def test_changed_file_without_mutant_rejected(self) -> None:
        issues = mutation_gate.diff_anchor_issues([self._item("m.py", "    return 20")], "HEAD")
        self.assertTrue(any("n.py" in i for i in issues))

    def test_all_changed_files_anchored_ok(self) -> None:
        self.assertEqual(
            mutation_gate.diff_anchor_issues([self._item("m.py", "    return 20"), self._item("n.py", "Y = 5")], "HEAD"), []
        )


class TestCrashNotKill(unittest.TestCase):
    def test_import_error_is_crash_not_kill(self) -> None:
        with tempfile.TemporaryDirectory() as d:
            r = mutation_gate.run_test_cmd("python3 -c 'import no_such_mod_xyz'", Path(d))
            self.assertNotEqual(r["returncode"], 0)
            self.assertTrue(r["crashed"])

    def test_assertion_failure_is_kill(self) -> None:
        with tempfile.TemporaryDirectory() as d:
            r = mutation_gate.run_test_cmd("python3 -c 'assert 1 == 2'", Path(d))
            self.assertFalse(r["crashed"])

    def test_gate_marks_crash_mutant_and_fails(self) -> None:
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "m.py").write_text("X = 1\nY = 2\nZ = 3\n", encoding="utf-8")
            (root / "ws" / "w").mkdir(parents=True)
            cmd = "python3 -c 'import m; assert m.X == 1 and m.Y == 2 and m.Z == 3'"
            muts = [
                {"id": "C1", "file": str(root / "m.py"), "target_content": "X = 1", "mutant_content": "import no_such_mod_xyz", "cwd": d, "test_cmd": cmd},
                {"id": "K2", "file": str(root / "m.py"), "target_content": "Y = 2", "mutant_content": "Y = 9", "cwd": d, "test_cmd": cmd},
                {"id": "K3", "file": str(root / "m.py"), "target_content": "Z = 3", "mutant_content": "Z = 9", "cwd": d, "test_cmd": cmd},
            ]
            mf = root / "muts.json"
            mf.write_text(json.dumps({"mutants": muts}), encoding="utf-8")
            code, summary = mutation_gate.execute_mutation_gate("w", "s", mf, root / "ws")
            status = {m["id"]: m["status"] for m in summary["mutants"]}
            self.assertEqual(status["C1"], "CRASHED")
            self.assertEqual(summary["gate"], "FAIL")


class TestWorkflowOpt(unittest.TestCase):
    def test_t2_consolidated_audit_report_without_06_or_11(self) -> None:
        import verify_governance

        with tempfile.TemporaryDirectory() as d:
            ws_root = Path(d)
            ws = ws_root / "t2-merged"
            ws.mkdir()
            today = "2026-10-21"
            (ws / "01_requirements_m.md").write_text("tier: T2\n", encoding="utf-8")
            (ws / "05_progress_m.md").write_text(f"- [{today}T10:00:00+07:00] [Agent:Opus] START\n", encoding="utf-8")
            (ws / "08_tasks_m.md").write_text("- [x] T1\n", encoding="utf-8")
            (ws / "09_tasks_solution_m.md").write_text("# Sol\n", encoding="utf-8")
            self.assertFalse(verify_governance.verify_workspace_docs("t2-merged", workspaces_dir=ws_root, today_override=today))
            self.assertTrue(any("gộp validation + report + audit" in lbl for lbl, _ in verify_governance.TIER_RULES["T2"]))
            (ws / "audit_report_m.md").write_text(
                "# Audit\n1. git diff\n2. runtime trace\n3. concurrency timeout watermark\n4. không suy diễn\n5. lessons\n6. dg3\n",
                encoding="utf-8",
            )
            self.assertTrue(verify_governance.verify_workspace_docs("t2-merged", workspaces_dir=ws_root, today_override=today))

    def test_laya_gate_dg1_with_demo_claim_writes_dg2_summary(self) -> None:
        from unittest import mock
        import laya_gate

        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            src = root / "s.py"
            src.write_text("X = 1\n", encoding="utf-8")
            claims = [
                {"id": "S-RC", "kind": "claim", "basis": "code", "claim": "X is 1", "evidence": f"REF: {src}:1-1\nX = 1", "flow": "f"},
                {"id": "D-BEHAV", "kind": "demo", "basis": "code", "claim": "demo ok", "evidence": "DEMO: X = 1", "flow": "f"},
                {"id": "CF-1", "kind": "control_false", "claim": "false 1", "evidence": "x", "flow": "f"},
                {"id": "CF-2", "kind": "control_false", "claim": "false 2", "evidence": "x", "flow": "f"},
                {"id": "CT-1", "kind": "control_true", "claim": "true 1", "evidence": "x", "flow": "f"},
            ]
            cf = root / "claims.json"
            cf.write_text(json.dumps(claims), encoding="utf-8")

            def fake_ask(_url, rnd, c, _obj, **_kw):
                is_neg = c["kind"] == "control_false"
                ch = laya_gate.NEG_CHOICE[rnd] if is_neg else laya_gate.POS_CHOICE[rnd]
                return {"choice": ch, "confidence": 0.9, "raw": {}, "request": {}}

            with mock.patch.object(laya_gate, "ask_round_mode_a", side_effect=fake_ask):
                rc = laya_gate.main(["run", "--stage", "dg1", "--workspace", "w", "--claims", str(cf), "--ws-root", str(root / "ws")])
            self.assertEqual(rc, 0)
            self.assertEqual(len(list((root / "ws" / "w").glob("06_laya_summary_dg1_*.json"))), 1)
            self.assertEqual(len(list((root / "ws" / "w").glob("06_laya_summary_dg2_*.json"))), 1)


if __name__ == "__main__":
    unittest.main()

