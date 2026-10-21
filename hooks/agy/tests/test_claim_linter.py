#!/usr/bin/env python3
"""Tests H4c Claim ↔ Evidence linter."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import agy_guard  # noqa: E402

EV = "EV-20261021T103000-ab12"


class TestClaimLinter(unittest.TestCase):
    def setUp(self) -> None:
        self._t = tempfile.TemporaryDirectory()
        self.ws = Path(self._t.name) / "ws"
        (self.ws / "06_evidence").mkdir(parents=True)
        self.st = [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "python3 tooling/evidence_run.py --workspace w --kind unit -- python3 -m unittest"}}]}]

    def tearDown(self) -> None:
        self._t.cleanup()

    def _ev(self, **kw) -> None:
        rec = {"id": EV, "cmd": "python3 -m unittest", "kind": "unit", "exit": 0, "stdout_tail": "Ran 49 tests\nOK", "src_fingerprint": "fp1"}
        rec.update(kw)
        (self.ws / "06_evidence" / f"{EV}.json").write_text(json.dumps(rec), encoding="utf-8")

    def test_claim_without_ev_rejected(self) -> None:
        self.assertTrue(agy_guard.claim_issues("- hooks 49/49 PASS", self.ws, self.st, "fp1"))

    def test_valid_ev_passes(self) -> None:
        self._ev()
        self.assertEqual(agy_guard.claim_issues(f"- hooks 49/49 PASS ({EV})", self.ws, self.st, "fp1"), [])

    def test_number_not_in_output_rejected(self) -> None:
        self._ev()
        out = agy_guard.claim_issues(f"- hooks 50/50 PASS ({EV})", self.ws, self.st, "fp1")
        self.assertTrue(any("50/50" in i for i in out))

    def test_stale_fingerprint_rejected(self) -> None:
        self._ev()
        self.assertTrue(any("cũ" in i for i in agy_guard.claim_issues(f"- PASS {EV}", self.ws, self.st, "fp2")))

    def test_nonzero_exit_rejected_for_pass_claim(self) -> None:
        self._ev(exit=1)
        self.assertTrue(any("exit" in i for i in agy_guard.claim_issues(f"- PASS {EV}", self.ws, self.st, "fp1")))
        self.assertEqual(agy_guard.claim_issues(f"- Red trước fix: FAIL {EV}", self.ws, self.st, "fp1"), [])

    def test_runtime_claim_needs_runtime_kind(self) -> None:
        self._ev()
        self.assertTrue(any("runtime" in i for i in agy_guard.claim_issues(f"- chạy thật OK {EV}", self.ws, self.st, "fp1")))

    def test_ev_without_matching_run_command_rejected(self) -> None:
        self._ev(kind="runtime")
        self.assertTrue(any("--kind runtime" in i for i in agy_guard.claim_issues(f"- PASS {EV}", self.ws, self.st, "fp1")))

    def test_forged_evidence_file_rejected(self) -> None:
        self._ev()
        st = self.st + [{"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": str(self.ws / "06_evidence" / f"{EV}.json")}}]}]
        self.assertTrue(any("ghi tay" in i for i in agy_guard.claim_issues(f"- PASS {EV}", self.ws, st, "fp1")))

    def test_fingerprint_computed_from_ev_cwd(self) -> None:
        import subprocess
        repo = Path(self._t.name) / "repo"
        repo.mkdir()
        subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
        (repo / "a.py").write_text("X = 1\n", encoding="utf-8")
        sys.path.insert(0, str(Path(__file__).resolve().parents[3] / "tooling"))
        import evidence_run
        self._ev(cwd=str(repo), src_fingerprint=evidence_run.source_fingerprint(repo))
        self.assertEqual(agy_guard.claim_issues(f"- PASS {EV}", self.ws, self.st, None), [])
        (repo / "a.py").write_text("X = 2\n", encoding="utf-8")
        self.assertTrue(any("cũ" in i for i in agy_guard.claim_issues(f"- PASS {EV}", self.ws, self.st, None)))

    def test_exempt_section_and_code_fence_ignored(self) -> None:
        text = "## Chưa kiểm được\n- runtime thật chưa chạy 0/1\n## Khác\n```\nPASS\n```\n"
        self.assertEqual(agy_guard.claim_issues(text, self.ws, self.st, "fp1"), [])

    def test_h6_dedupes_audit_msg_and_respects_h6_auto_checks_false(self) -> None:
        root = Path(self._t.name)
        proj = root / "p"
        proj.mkdir()
        (proj / ".agent-project.json").write_text(
            json.dumps({"components": [{"name": "c", "path": ".", "h6_auto_checks": False, "checks": ["exit 1"]}]}),
            encoding="utf-8",
        )
        src = proj / "m.py"
        src.write_text("x = 1\n", encoding="utf-8")
        tr = root / "tr.jsonl"
        sdir = root / "state"
        tr.write_text(
            json.dumps({"type": "USER_INPUT", "content": "APPROVE"}) + "\n"
            + json.dumps({"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src)}}]}) + "\n",
            encoding="utf-8",
        )
        r1 = agy_guard.handle_post_invocation({"conversationId": "c1", "transcriptPath": str(tr)}, mem_root=root / "mem", state_dir=sdir)
        self.assertIn("MANDATORY FINAL ADVERSARIAL AUDIT", r1["injectSteps"][0]["ephemeralMessage"])
        with tr.open("a", encoding="utf-8") as f:
            f.write(json.dumps({"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(src)}}]}) + "\n")
        r2 = agy_guard.handle_post_invocation({"conversationId": "c1", "transcriptPath": str(tr)}, mem_root=root / "mem", state_dir=sdir)
        self.assertIn("🔍 H6 (m.py):", r2["injectSteps"][0]["ephemeralMessage"])
        self.assertNotIn("MANDATORY FINAL ADVERSARIAL AUDIT", r2["injectSteps"][0]["ephemeralMessage"])


if __name__ == "__main__":
    unittest.main()

