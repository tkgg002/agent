#!/usr/bin/env python3
"""Unit tests for tooling/pentagi_bridge.py."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import pentagi_bridge  # noqa: E402


class PentagiBridgeTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp_obj = tempfile.TemporaryDirectory()
        self.tmp = Path(self.tmp_obj.name)

    def tearDown(self) -> None:
        self.tmp_obj.cleanup()

    def test_01_mask_secrets_masks_tokens_passwords_and_db_uris(self) -> None:
        raw = (
            "Found token Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.secretpayload and "
            "AWS key AKIA1234567890ABCDEF and DB postgres://admin:supersecret123@10.0.0.1:5432/prod "
            "plus password = my_secret_pass_99"
        )
        masked = pentagi_bridge.mask_secrets(raw)
        self.assertNotIn("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", masked)
        self.assertNotIn("AKIA1234567890ABCDEF", masked)
        self.assertNotIn("supersecret123", masked)
        self.assertNotIn("my_secret_pass_99", masked)
        self.assertIn("postgres://***:***@10.0.0.1:5432/prod", masked)
        self.assertIn("<MASKED_SECRET>", masked)

    def test_02_validate_target_url_blocks_external_and_subdomain_spoofing(self) -> None:
        # Default localhost / 127.0.0.1 allowed
        self.assertTrue(pentagi_bridge.validate_target_url("http://localhost:8080/api/v1/orders"))
        self.assertTrue(pentagi_bridge.validate_target_url("https://127.0.0.1:8443/health"))
        self.assertTrue(pentagi_bridge.validate_target_url("http://localhost"))

        # External domain and subdomain spoofing (http://localhost.evil.com) blocked
        self.assertFalse(pentagi_bridge.validate_target_url("https://prod.example.com/api"))
        self.assertFalse(pentagi_bridge.validate_target_url("http://localhost.evil.com:8080/api"))
        self.assertFalse(pentagi_bridge.validate_target_url("ftp://localhost:21/file"))
        self.assertFalse(pentagi_bridge.validate_target_url(""))

        # Custom whitelist from .agent-project.json
        custom = ["https://staging-internal.local:9443"]
        self.assertTrue(
            pentagi_bridge.validate_target_url("https://staging-internal.local:9443/v2/pay", custom)
        )
        self.assertFalse(
            pentagi_bridge.validate_target_url("http://localhost:8080", custom)
        )

    def test_03_generate_s1_intake_creates_valid_docs_and_masks_secrets(self) -> None:
        report_md = (
            "## [HIGH] SQL Injection in /api/v1/users\n"
            "Exploit leaked URI mongodb://root: plainPass999@127.0.0.1:27017/admin "
            "and password=topsecretvalue123.\n\n"
            "### [MEDIUM] Missing Rate Limit on /api/v1/login\n"
            "Endpoint allows brute-force requests without throttling.\n"
        )
        ws_dir = self.tmp / "ws_pentest"
        res = pentagi_bridge.generate_s1_intake_from_report(
            report_md=report_md,
            workspace_dir=ws_dir,
            suffix="pentest_20261020",
            tier="T2",
        )
        self.assertEqual(res["findings_count"], "2")

        req_text = Path(res["requirements"]).read_text(encoding="utf-8")
        prog_text = Path(res["progress"]).read_text(encoding="utf-8")
        tasks_text = Path(res["tasks"]).read_text(encoding="utf-8")

        self.assertIn("tier: T2", req_text)
        self.assertNotIn("topsecretvalue123", req_text)
        self.assertIn("<MASKED_SECRET>", req_text)
        self.assertIn("F01 (HIGH)", tasks_text)
        self.assertIn("F02 (MEDIUM)", tasks_text)
        self.assertRegex(
            prog_text,
            r"- \[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}\] \[Agent:PentAGI-Bridge\] START S1 Intake",
        )

        # Rule #17: calling again on existing suffix raises FileExistsError
        with self.assertRaises(FileExistsError):
            pentagi_bridge.generate_s1_intake_from_report(
                report_md=report_md,
                workspace_dir=ws_dir,
                suffix="pentest_20261020",
                tier="T2",
            )

    def test_04_trigger_pentagi_flow_blocks_unlisted_target_and_supports_dry_run(self) -> None:
        with self.assertRaises(ValueError):
            pentagi_bridge.trigger_pentagi_flow(
                api_url="https://localhost:8443",
                api_token="secret_tok",
                target_url="https://production.bank.com/api",
                prompt="test",
                dry_run=True,
            )

        ok_res = pentagi_bridge.trigger_pentagi_flow(
            api_url="https://localhost:8443",
            api_token="secret_tok",
            target_url="http://127.0.0.1:8080/api/orders",
            prompt="Check auth with password=supersecret999",
            dry_run=True,
        )
        self.assertEqual(ok_res["status"], "DRY_RUN_VALIDATED")
        self.assertNotIn("supersecret999", ok_res["payload"]["prompt"])

    def test_05_load_allowed_targets_from_manifest(self) -> None:
        proj = self.tmp / "my_proj"
        proj.mkdir(parents=True, exist_ok=True)
        (proj / ".agent-project.json").write_text(
            json.dumps(
                {
                    "version": 1,
                    "allowed_pentest_targets": ["http://127.0.0.1:18080"],
                    "components": [
                        {
                            "name": "api",
                            "path": "services/api",
                            "allowed_pentest_targets": ["http://localhost:19090"],
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )
        allowed = pentagi_bridge.load_allowed_targets_from_manifest(proj)
        self.assertEqual(allowed, ["http://127.0.0.1:18080", "http://localhost:19090"])


if __name__ == "__main__":
    unittest.main()
