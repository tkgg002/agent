"""Tests for: laya_gate Vega-primary/Laya-fallback routing, hard_negative_feedback, evidence_run hook.

Real local HTTP servers (no mocks of our own code), isolated temp workspaces, no writes to live data.
"""
import contextlib
import io
import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Thread

AGENT_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(AGENT_ROOT / "tooling"))

import hard_negative_feedback as hnf  # noqa: E402
import laya_gate  # noqa: E402


class _Srv(BaseHTTPRequestHandler):
    """Fake decision sidecar: /health 200, records every POST body."""
    posts = []
    health = 200

    def log_message(self, *a):
        pass

    def do_GET(self):
        self.send_response(self.health if self.path == "/health" else 404)
        self.end_headers()
        self.wfile.write(b"{}")

    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(n) or b"{}")
        _Srv.posts.append((self.path, body))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"ok": True, "result": {"answers": {"verdict": {"choice": "CORRECT", "confidence": 0.9}}}}).encode())


def _start():
    srv = ThreadingHTTPServer(("127.0.0.1", 0), _Srv)
    Thread(target=srv.serve_forever, daemon=True).start()
    return srv, f"http://127.0.0.1:{srv.server_address[1]}"


class TestResolveGateUrl(unittest.TestCase):
    def setUp(self):
        _Srv.posts.clear()
        _Srv.health = 200
        self.srv, self.base = _start()
        self.orig = (laya_gate.VEGA_URL, laya_gate.LAYA_FALLBACK_URL)

    def tearDown(self):
        self.srv.shutdown()
        laya_gate.VEGA_URL, laya_gate.LAYA_FALLBACK_URL = self.orig

    def test_explicit_url_wins(self):
        self.assertEqual(laya_gate.resolve_gate_url("http://x/v1/predict"), "http://x/v1/predict")

    def test_vega_healthy_is_primary(self):
        laya_gate.VEGA_URL = self.base + "/v1/predict"
        laya_gate.LAYA_FALLBACK_URL = "http://127.0.0.1:1/v1/predict"
        self.assertEqual(laya_gate.resolve_gate_url(None), self.base + "/v1/predict")

    def test_vega_down_falls_back_to_laya_with_warning(self):
        laya_gate.VEGA_URL = "http://127.0.0.1:1/v1/predict"  # nothing listens
        laya_gate.LAYA_FALLBACK_URL = self.base + "/v1/predict"
        err = io.StringIO()
        with contextlib.redirect_stderr(err):
            got = laya_gate.resolve_gate_url(None)
        self.assertEqual(got, self.base + "/v1/predict")
        self.assertIn("GATE_FALLBACK", err.getvalue())

    def test_post_laya_sends_mode(self):
        laya_gate.post_laya(self.base + "/v1/predict", {"claim": "x"}, {"verdict": {"type": "choice"}}, retries=0)
        path, body = _Srv.posts[-1]
        self.assertEqual(path, "/v1/predict")
        self.assertEqual(body["mode"], laya_gate.GATE_MODE)
        self.assertEqual(body["state"], {"claim": "x"})


class _SlowSrv(BaseHTTPRequestHandler):
    """Vega with a hung /v1/predict: /health 200 but POST does not answer within the fail-fast window."""
    slow_posts = 0

    def log_message(self, *a):
        pass

    def do_GET(self):
        self.send_response(200)
        self.end_headers()

    def do_POST(self):
        _SlowSrv.slow_posts += 1
        time.sleep(2.5)


class TestFailFastFallback(unittest.TestCase):
    def setUp(self):
        _Srv.posts.clear()
        _SlowSrv.slow_posts = 0
        self.slow = ThreadingHTTPServer(("127.0.0.1", 0), _SlowSrv)
        Thread(target=self.slow.serve_forever, daemon=True).start()
        self.fast, self.fast_base = _start()
        self.slow_url = f"http://127.0.0.1:{self.slow.server_address[1]}/v1/predict"
        self.orig = (laya_gate.VEGA_URL, laya_gate.LAYA_FALLBACK_URL, laya_gate.VEGA_TIMEOUT,
                     laya_gate.VEGA_COOLDOWN, laya_gate._vega_skip_until)
        laya_gate.VEGA_URL = self.slow_url
        laya_gate.LAYA_FALLBACK_URL = self.fast_base + "/v1/predict"
        laya_gate.VEGA_TIMEOUT = 0.5
        laya_gate.VEGA_COOLDOWN = 60.0
        laya_gate._vega_skip_until = 0.0

    def tearDown(self):
        self.slow.shutdown()
        self.fast.shutdown()
        (laya_gate.VEGA_URL, laya_gate.LAYA_FALLBACK_URL, laya_gate.VEGA_TIMEOUT,
         laya_gate.VEGA_COOLDOWN, laya_gate._vega_skip_until) = self.orig

    def _post(self):
        err = io.StringIO()
        with contextlib.redirect_stderr(err):
            t = time.monotonic()
            raw = laya_gate.post_laya(self.slow_url, {"claim": "x"}, {"verdict": {"type": "choice"}}, retries=6, retry_wait=5.0)
        return raw, time.monotonic() - t, err.getvalue()

    def test_hung_vega_falls_to_laya_without_retry(self):
        raw, took, err = self._post()
        self.assertTrue(raw["ok"])
        self.assertLess(took, 2.0)
        self.assertEqual(_SlowSrv.slow_posts, 1)
        self.assertIn("GATE_FALLBACK", err)
        self.assertEqual(len(_Srv.posts), 1)

    def test_cooldown_skips_vega_on_next_call(self):
        self._post()
        raw, took, _ = self._post()
        self.assertTrue(raw["ok"])
        self.assertLess(took, 0.4)
        self.assertEqual(_SlowSrv.slow_posts, 1)

    def test_healthy_vega_does_not_touch_laya(self):
        laya_gate.VEGA_URL = self.fast_base + "/v1/predict"
        laya_gate.LAYA_FALLBACK_URL = self.slow_url
        raw = laya_gate.post_laya(laya_gate.VEGA_URL, {"claim": "x"}, {"verdict": {"type": "choice"}}, retries=0)
        self.assertTrue(raw["ok"])
        self.assertEqual(_SlowSrv.slow_posts, 0)

    def test_explicit_url_keeps_old_timeout_path(self):
        laya_gate.VEGA_URL = "http://127.0.0.1:1/v1/predict"
        raw = laya_gate.post_laya(self.fast_base + "/v1/predict", {"claim": "x"}, {"verdict": {"type": "choice"}}, retries=0)
        self.assertTrue(raw["ok"])


class TestHardNegativeFeedback(unittest.TestCase):
    def setUp(self):
        _Srv.posts.clear()
        self.srv, self.base = _start()
        self.tmp = tempfile.TemporaryDirectory()
        self.ws = Path(self.tmp.name) / "ws"
        self.ws.mkdir()
        raw = self.ws / "06_laya_raw_dg1_20260101T000000.json"
        raw.write_text(json.dumps([
            {"id": "S1", "kind": "solution", "claim": "a", "evidence": "REF: f.py:1-1\nx=1", "flow": "fl"},
            {"id": "S2", "kind": "solution", "claim": "b", "evidence": "REF: f.py:2-2\ny=2"},
            {"id": "K1", "kind": "control_true", "claim": "c", "evidence": "e"},
        ]), encoding="utf-8")
        (self.ws / "06_laya_summary_dg1_20260101T000000.json").write_text(json.dumps({
            "stage": "dg1", "raw_file": str(raw), "gate": "PASS",
            "claims": [{"id": "S1", "kind": "solution", "claim": "a", "decision": "ACCEPT"},
                       {"id": "S2", "kind": "solution", "claim": "b", "decision": "FLAG"},
                       {"id": "K1", "kind": "control_true", "claim": "c", "decision": "ACCEPT"}],
        }), encoding="utf-8")

    def tearDown(self):
        self.srv.shutdown()
        self.tmp.cleanup()

    def test_collect_only_accept_non_control(self):
        items = hnf.collect_hard_negatives(self.ws, "EV-1")
        self.assertEqual(len(items), 1)
        it = items[0]
        self.assertEqual(it["task_id"], "EV-1:S1")
        self.assertEqual(it["stage"], hnf.STAGE)
        self.assertEqual(it["labels"], {"verdict": "INCORRECT"})
        self.assertFalse(it["auto_fit"])
        self.assertEqual(it["state"], {"claim": "a", "evidence": "REF: f.py:1-1\nx=1", "flow": "fl"})

    def test_empty_workspace_yields_nothing(self):
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(hnf.collect_hard_negatives(Path(td), "EV-0"), [])
            self.assertEqual(hnf.feed_runtime_fail(Path(td), "EV-0", self.base + "/f")["candidates"], 0)

    def test_post_sends_and_counts(self):
        res = hnf.feed_runtime_fail(self.ws, "EV-2", self.base + "/v1/systemone/feedback")
        self.assertEqual((res["candidates"], res["sent"], res["error"]), (1, 1, ""))
        path, body = _Srv.posts[-1]
        self.assertEqual(path, "/v1/systemone/feedback")
        self.assertEqual(body["labels"], {"verdict": "INCORRECT"})

    def test_sidecar_down_is_fail_soft(self):
        res = hnf.feed_runtime_fail(self.ws, "EV-3", "http://127.0.0.1:1/f")
        self.assertEqual((res["candidates"], res["sent"]), (1, 0))
        self.assertTrue(res["error"])

    def test_evidence_run_attaches_hard_negatives_on_fail_only(self):
        env = dict(os.environ, VEGA_FEEDBACK_URL=self.base + "/v1/systemone/feedback")
        cmd = [sys.executable, str(AGENT_ROOT / "tooling" / "evidence_run.py"), "--workspace", "ws",
               "--kind", "unit", "--ws-root", str(self.ws.parent), "--cwd", self.tmp.name, "--"]
        ok = subprocess.run(cmd + ["true"], capture_output=True, text=True, env=env)
        self.assertEqual(ok.returncode, 0, ok.stderr)
        bad = subprocess.run(cmd + ["false"], capture_output=True, text=True, env=env)
        self.assertEqual(bad.returncode, 1, "exit code must be preserved")
        recs = sorted((self.ws / "06_evidence").glob("EV-*.json"), key=lambda p: p.stat().st_mtime)
        self.assertEqual(len(recs), 2)
        ok_rec, bad_rec = (json.loads(p.read_text(encoding="utf-8")) for p in recs)
        self.assertNotIn("hard_negatives", ok_rec)
        self.assertEqual(bad_rec["hard_negatives"]["sent"], 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)

