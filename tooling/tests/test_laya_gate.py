#!/usr/bin/env python3
"""Unit & backtest suite for agent/tooling/laya_gate.py (stdlib unittest + pytest compatible)."""
from __future__ import annotations

import io
import json
import sys
import tempfile
import unittest
import urllib.error
from contextlib import redirect_stdout
from pathlib import Path
from typing import Any, Callable, Dict, Tuple
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import laya_gate  # noqa: E402


def make_verdict(key: str, choice: str, confidence: float) -> Dict[str, Any]:
    return {"status": "success", "result": {"answers": {key: {"choice": choice, "confidence": confidence}}}}


class FakeHTTPResponse:
    def __init__(self, payload: Dict[str, Any]) -> None:
        self._bytes = json.dumps(payload).encode("utf-8")

    def read(self) -> bytes:
        return self._bytes

    def __enter__(self) -> "FakeHTTPResponse":
        return self

    def __exit__(self, *exc: Any) -> None:
        return None


def make_urlopen_mock(responder: Callable[[Dict[str, Any], int], Tuple[int, Dict[str, Any]]]):
    state = {"count": 0}

    def _urlopen(req: Any, timeout: float = 120.0) -> FakeHTTPResponse:
        state["count"] += 1
        body = json.loads(req.data.decode("utf-8"))
        status, payload = responder(body, state["count"])
        if status != 200:
            raise urllib.error.HTTPError(
                req.full_url, status, f"HTTP {status}", hdrs=None, fp=io.BytesIO(json.dumps(payload).encode("utf-8"))
            )
        return FakeHTTPResponse(payload)

    return _urlopen


class TestLayaGate(unittest.TestCase):
    def test_verify_evidence_checks_file_and_lines(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            tmp_path = Path(td)
            src = tmp_path / "sample.py"
            src.write_text("def add(a, b):\n    if a is None:\n        return 0\n    return a + b\n", encoding="utf-8")

            ok, err = laya_gate.verify_evidence(
                {
                    "id": "S1",
                    "kind": "solution",
                    "claim": "add returns 0 when a is None.",
                    "evidence": f"REF: {src}:1-3\ndef add(a, b):\n    if a is None:\n        return 0",
                }
            )
            self.assertTrue(ok)
            self.assertEqual(err, "")

            ok2, err2 = laya_gate.verify_evidence(
                {
                    "id": "S2",
                    "kind": "solution",
                    "claim": "add raises ValueError.",
                    "evidence": f"REF: {src}:1-3\ndef add(a, b):\n    raise ValueError('bad')",
                }
            )
            self.assertFalse(ok2)
            self.assertIn("trích dẫn không khớp", err2)

            ok3, err3 = laya_gate.verify_evidence(
                {"id": "S3", "kind": "solution", "claim": "x", "evidence": "no ref here"}
            )
            self.assertFalse(ok3)
            self.assertIn("thiếu REF", err3)

    def test_run_requires_minimum_controls(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            tmp_path = Path(td)
            claims_file = tmp_path / "bad_claims.json"
            claims_file.write_text(
                json.dumps([{"id": "S1", "kind": "solution", "claim": "x", "evidence": "REF: a:1-1\nx"}]),
                encoding="utf-8",
            )
            rc = laya_gate.main(
                ["run", "--stage", "dg1", "--workspace", "ws1", "--claims", str(claims_file), "--ws-root", str(tmp_path)]
            )
            self.assertEqual(rc, 3)

    def test_mode_a_retries_503_and_calibrates_rounds_and_resolves_flag(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            tmp_path = Path(td)
            src = tmp_path / "svc.py"
            src.write_text(
                "def process(items):\n    if not items:\n        return []\n    return [x * 2 for x in items]\n",
                encoding="utf-8",
            )
            claims = [
                {
                    "id": "S1",
                    "kind": "solution",
                    "basis": "code",
                    "claim": "process returns an empty list when items is empty.",
                    "evidence": f"REF: {src}:1-4\ndef process(items):\n    if not items:\n        return []",
                },
                {
                    "id": "S2",
                    "kind": "solution",
                    "basis": "code",
                    "claim": "process doubles each element.",
                    "evidence": f"REF: {src}:1-4\n    return [x * 2 for x in items]",
                },
                {
                    "id": "S3",
                    "kind": "solution",
                    "basis": "hypothesis",
                    "claim": "Latency spikes when items > 100000.",
                    "evidence": f"REF: {src}:1-4\n    return [x * 2 for x in items]",
                },
                {
                    "id": "K1",
                    "kind": "control_false",
                    "claim": "process returns None when items is empty.",
                    "evidence": f"REF: {src}:1-4\ndef process(items):\n    if not items:\n        return []",
                },
                {
                    "id": "K2",
                    "kind": "control_false",
                    "claim": "process triples each element.",
                    "evidence": f"REF: {src}:1-4\n    return [x * 2 for x in items]",
                },
                {
                    "id": "K3",
                    "kind": "control_true",
                    "claim": "process takes parameter items.",
                    "evidence": f"REF: {src}:1-4\ndef process(items):",
                },
            ]
            claims_file = tmp_path / "claims.json"
            claims_file.write_text(json.dumps(claims), encoding="utf-8")

            def responder(body: Dict[str, Any], count: int) -> Tuple[int, Dict[str, Any]]:
                if count <= 2:
                    return 503, {"detail": "Model loading"}
                st = body["state"]
                claim_txt = st["claim"]
                is_r3 = "code_excerpt" in st
                is_r2 = "feature_objective" in st
                if "returns None" in claim_txt:  # K1 control_false
                    if not is_r2 and not is_r3:  # r1 fails control!
                        return 200, make_verdict("verdict", "CORRECT", 0.80)
                    return 200, make_verdict("verdict", "NO" if is_r3 else "INCORRECT", 0.92)
                if "triples" in claim_txt:  # K2 control_false
                    return 200, make_verdict("verdict", "NO" if is_r3 else "INCORRECT", 0.90)
                if "parameter items" in claim_txt:  # K3 control_true
                    return 200, make_verdict("verdict", "YES" if is_r3 else "CORRECT", 0.95)
                if "doubles each element" in claim_txt:  # S2 -> FLAG on r2
                    if is_r2:
                        return 200, make_verdict("verdict", "INCORRECT", 0.65)
                    return 200, make_verdict("verdict", "YES" if is_r3 else "CORRECT", 0.85)
                return 200, make_verdict("verdict", "YES" if is_r3 else "CORRECT", 0.91)

            with patch("urllib.request.urlopen", side_effect=make_urlopen_mock(responder)):
                rc = laya_gate.main(
                    [
                        "run",
                        "--stage",
                        "dg1",
                        "--workspace",
                        "ws_test",
                        "--claims",
                        str(claims_file),
                        "--ws-root",
                        str(tmp_path),
                        "--retries",
                        "4",
                        "--retry-wait",
                        "0.001",
                    ]
                )
            self.assertEqual(rc, 2)

            summaries = list((tmp_path / "ws_test").glob("06_laya_summary_dg1_*.json"))
            self.assertEqual(len(summaries), 1)
            summary = json.loads(summaries[0].read_text(encoding="utf-8"))
            self.assertEqual(summary["trusted_rounds"], ["r2", "r3"])
            decisions = {c["id"]: c["decision"] for c in summary["claims"]}
            self.assertEqual(decisions, {"S1": "ACCEPT", "S2": "FLAG", "S3": "HYPOTHESIS"})
            self.assertEqual(summary["gate"], "NEEDS_RESOLUTION")

            rc_bad = laya_gate.main(
                [
                    "resolve",
                    "--workspace",
                    "ws_test",
                    "--stage",
                    "dg1",
                    "--id",
                    "S2",
                    "--reason",
                    "no ref provided",
                    "--ws-root",
                    str(tmp_path),
                ]
            )
            self.assertEqual(rc_bad, 3)

            rc_ok = laya_gate.main(
                [
                    "resolve",
                    "--workspace",
                    "ws_test",
                    "--stage",
                    "dg1",
                    "--id",
                    "S2",
                    "--reason",
                    f"List comprehension x * 2 confirmed at REF: {src}:4-4",
                    "--ws-root",
                    str(tmp_path),
                ]
            )
            self.assertEqual(rc_ok, 0)
            updated = json.loads(summaries[0].read_text(encoding="utf-8"))
            self.assertEqual(updated["gate"], "PASS")
            dec_files = list((tmp_path / "ws_test").glob("04_decisions*.md"))
            self.assertEqual(len(dec_files), 1)
            self.assertIn("DG1 S2 FLAG resolved", dec_files[0].read_text(encoding="utf-8"))

    def test_mode_b_debate_review_catches_rubber_stamp_and_clears_valid_fencing(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            tmp_path = Path(td)
            diff_file = tmp_path / "sample.diff"
            diff_file.write_text(
                "diff --git a/pkg/a.py b/pkg/a.py\n"
                "--- a/pkg/a.py\n+++ b/pkg/a.py\n"
                "@@ -1,3 +1,3 @@\n"
                " def check(x):\n"
                "-    return x == 1\n"
                "+    return x == 2\n"
                "diff --git a/pkg/b.py b/pkg/b.py\n"
                "--- a/pkg/b.py\n+++ b/pkg/b.py\n"
                "@@ -1,3 +1,3 @@\n"
                " def validate(y):\n"
                "-    return False\n"
                "+    return True\n",
                encoding="utf-8",
            )

            def responder(body: Dict[str, Any], count: int) -> Tuple[int, Dict[str, Any]]:
                st = body["state"]
                fn = st["filename"]
                code = st["full_function_code"]
                instr = body["questions"]["review_result"]["instructions"]
                is_fencing = "fencing context" in instr
                if not is_fencing:
                    return 200, make_verdict("review_result", "REJECT_LOGIC", 0.65)
                if fn == "pkg/a.py":
                    if "!=" in code:
                        return 200, make_verdict("review_result", "REJECT_LOGIC", 0.88)
                    return 200, make_verdict("review_result", "APPROVE", 0.97)
                return 200, make_verdict("review_result", "APPROVE", 0.92)

            with patch("urllib.request.urlopen", side_effect=make_urlopen_mock(responder)):
                rc = laya_gate.main(
                    [
                        "review",
                        "--workspace",
                        "ws_rev",
                        "--diff-file",
                        str(diff_file),
                        "--ws-root",
                        str(tmp_path),
                    ]
                )
            self.assertEqual(rc, 0)
            sum_file = next((tmp_path / "ws_rev").glob("06_laya_summary_dg3review_*.json"))
            summary = json.loads(sum_file.read_text(encoding="utf-8"))
            by_id = {c["id"]: c["decision"] for c in summary["claims"]}
            self.assertEqual(by_id["pkg/a.py"], "ACCEPT")
            self.assertEqual(by_id["pkg/b.py"], "UNVALIDATED")

    def test_backtest_reproduces_2026_10_16_batch(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            tmp_path = Path(td)
            r1 = [
                {"id": "S1", "kind": "solution", "claim": "c1", "r1": {"choice": "CORRECT", "confidence": 0.40}},
                {"id": "K1", "kind": "control_false", "claim": "k1", "r1": {"choice": "INCORRECT", "confidence": 0.31}},
                {"id": "K2", "kind": "control_false", "claim": "k2", "r1": {"choice": "INCORRECT", "confidence": 0.25}},
                {"id": "K3", "kind": "control_true", "claim": "k3", "r1": {"choice": "CORRECT", "confidence": 0.19}},
            ]
            r23 = [
                {
                    "id": "S1",
                    "r2": {"choice": "CORRECT", "confidence": 0.82},
                    "r3": {"choice": "YES", "confidence": 0.90},
                },
                {
                    "id": "K1",
                    "r2": {"choice": "CORRECT", "confidence": 0.76},
                    "r3": {"choice": "YES", "confidence": 0.91},
                },
                {
                    "id": "K2",
                    "r2": {"choice": "INCORRECT", "confidence": 0.80},
                    "r3": {"choice": "NO", "confidence": 0.85},
                },
                {
                    "id": "K3",
                    "r2": {"choice": "CORRECT", "confidence": 0.88},
                    "r3": {"choice": "YES", "confidence": 0.93},
                },
            ]
            f1, f2 = tmp_path / "r1.json", tmp_path / "r23.json"
            f1.write_text(json.dumps(r1), encoding="utf-8")
            f2.write_text(json.dumps(r23), encoding="utf-8")
            buf = io.StringIO()
            with redirect_stdout(buf):
                rc = laya_gate.main(["backtest", "--r1", str(f1), "--r23", str(f2)])
            self.assertEqual(rc, 0)
            out = json.loads(buf.getvalue())
            self.assertEqual(out["trusted_rounds"], [])
            self.assertEqual(out["claims"], [{"id": "S1", "decision": "UNVALIDATED"}])


if __name__ == "__main__":
    unittest.main()
