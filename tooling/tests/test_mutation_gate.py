#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Unit tests for tooling/mutation_gate.py (3-Mutant Test Verification Gate)."""
from __future__ import annotations

import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

TOOLING_DIR = Path(__file__).resolve().parents[1]
if str(TOOLING_DIR) not in sys.path:
    sys.path.insert(0, str(TOOLING_DIR))

import mutation_gate as mg  # noqa: E402


class TestMutationGate(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.ws_root = self.root / "workspaces"
        self.ws_root.mkdir(parents=True, exist_ok=True)
        self.proj = self.root / "sample_proj"
        self.proj.mkdir(parents=True, exist_ok=True)

        # Create .agent-project.json for auto-detection test
        (self.proj / ".agent-project.json").write_text(
            json.dumps(
                {
                    "name": "sample-proj",
                    "components": [
                        {
                            "name": "calc",
                            "path": ".",
                            "test": f"{sys.executable} -m unittest test_calc.py",
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )

        self.calc_py = self.proj / "calc.py"
        self.calc_py.write_text(
            "def add(a: int, b: int) -> int:\n"
            "    return a + b\n\n"
            "def sub(a: int, b: int) -> int:\n"
            "    return a - b\n\n"
            "def mul(a: int, b: int) -> int:\n"
            "    return a * b\n",
            encoding="utf-8",
        )
        self.orig_sha = hashlib.sha256(self.calc_py.read_bytes()).hexdigest()

        self.test_calc_py = self.proj / "test_calc.py"
        self.test_calc_py.write_text(
            "import unittest\n"
            "import calc\n\n"
            "class TestCalc(unittest.TestCase):\n"
            "    def test_ops(self):\n"
            "        self.assertEqual(calc.add(2, 3), 5)\n"
            "        self.assertEqual(calc.sub(5, 2), 3)\n"
            "        self.assertEqual(calc.mul(3, 4), 12)\n\n"
            "if __name__ == '__main__':\n"
            "    unittest.main()\n",
            encoding="utf-8",
        )

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_01_all_three_mutants_killed_passes_and_restores_sha256(self) -> None:
        mutations_path = self.proj / "mutations_ok.json"
        mutations_path.write_text(
            json.dumps(
                [
                    {
                        "id": "M1",
                        "description": "Break add()",
                        "file": str(self.calc_py),
                        "target_content": "return a + b",
                        "mutant_content": "return a - b",
                    },
                    {
                        "id": "M2",
                        "description": "Break sub()",
                        "file": str(self.calc_py),
                        "target_content": "return a - b",
                        "mutant_content": "return a + b",
                    },
                    {
                        "id": "M3",
                        "description": "Break mul()",
                        "file": str(self.calc_py),
                        "target_content": "return a * b",
                        "mutant_content": "return a + b",
                    },
                ]
            ),
            encoding="utf-8",
        )
        code, summary = mg.execute_mutation_gate(
            ws_name="ws-test",
            suffix="calc_v1",
            mutations_file=mutations_path,
            workspaces_dir=self.ws_root,
        )
        self.assertEqual(code, 0)
        self.assertEqual(summary["gate"], "PASS")
        self.assertEqual(summary["killed_count"], 3)
        self.assertEqual(summary["survived_count"], 0)
        self.assertTrue(summary["all_restored_sha256"])
        self.assertEqual(hashlib.sha256(self.calc_py.read_bytes()).hexdigest(), self.orig_sha)
        self.assertTrue(Path(summary["summary_file"]).is_file())

    def test_02_survived_mutant_fails_gate_and_restores_sha256(self) -> None:
        # Weak test that does not check mul(3, 4)
        self.test_calc_py.write_text(
            "import unittest\n"
            "import calc\n\n"
            "class TestCalc(unittest.TestCase):\n"
            "    def test_ops(self):\n"
            "        self.assertEqual(calc.add(2, 3), 5)\n"
            "        self.assertEqual(calc.sub(5, 2), 3)\n",
            encoding="utf-8",
        )
        mutations_path = self.proj / "mutations_weak.json"
        mutations_path.write_text(
            json.dumps(
                [
                    {
                        "id": "M1",
                        "file": str(self.calc_py),
                        "target_content": "return a + b",
                        "mutant_content": "return a - b",
                    },
                    {
                        "id": "M2",
                        "file": str(self.calc_py),
                        "target_content": "return a - b",
                        "mutant_content": "return a + b",
                    },
                    {
                        "id": "M3",
                        "file": str(self.calc_py),
                        "target_content": "return a * b",
                        "mutant_content": "return 999",
                    },
                ]
            ),
            encoding="utf-8",
        )
        code, summary = mg.execute_mutation_gate(
            ws_name="ws-test",
            suffix="calc_weak",
            mutations_file=mutations_path,
            workspaces_dir=self.ws_root,
        )
        self.assertEqual(code, 2)
        self.assertEqual(summary["gate"], "FAIL")
        self.assertEqual(summary["survived_ids"], ["M3"])
        self.assertEqual(hashlib.sha256(self.calc_py.read_bytes()).hexdigest(), self.orig_sha)

    def test_03_syntax_error_mutant_rejected_by_ast_parse(self) -> None:
        mutations_path = self.proj / "mutations_syntax.json"
        mutations_path.write_text(
            json.dumps(
                [
                    {
                        "id": "M1",
                        "file": str(self.calc_py),
                        "target_content": "return a + b",
                        "mutant_content": "return a + (",
                    },
                    {
                        "id": "M2",
                        "file": str(self.calc_py),
                        "target_content": "return a - b",
                        "mutant_content": "return a + b",
                    },
                    {
                        "id": "M3",
                        "file": str(self.calc_py),
                        "target_content": "return a * b",
                        "mutant_content": "return a + b",
                    },
                ]
            ),
            encoding="utf-8",
        )
        code, summary = mg.execute_mutation_gate(
            ws_name="ws-test",
            suffix="calc_syntax",
            mutations_file=mutations_path,
            workspaces_dir=self.ws_root,
        )
        self.assertEqual(code, 3)
        self.assertEqual(summary["gate"], "INVALID_INPUT")
        self.assertIn("SyntaxError", summary["error"])
        self.assertEqual(hashlib.sha256(self.calc_py.read_bytes()).hexdigest(), self.orig_sha)

    def test_04_timeout_mutant_classified_as_timeout_and_fails_gate(self) -> None:
        mutations_path = self.proj / "mutations_timeout.json"
        mutations_path.write_text(
            json.dumps(
                [
                    {
                        "id": "M1",
                        "file": str(self.calc_py),
                        "target_content": "return a + b",
                        "mutant_content": "return a - b",
                    },
                    {
                        "id": "M2",
                        "file": str(self.calc_py),
                        "target_content": "return a - b",
                        "mutant_content": "return a + b",
                    },
                    {
                        "id": "M3",
                        "file": str(self.calc_py),
                        "target_content": "return a * b",
                        "mutant_content": "import time; time.sleep(5); return a * b",
                    },
                ]
            ),
            encoding="utf-8",
        )
        code, summary = mg.execute_mutation_gate(
            ws_name="ws-test",
            suffix="calc_timeout",
            mutations_file=mutations_path,
            workspaces_dir=self.ws_root,
            timeout_sec=1,
        )
        self.assertEqual(code, 2)
        self.assertEqual(summary["gate"], "FAIL")
        self.assertEqual(summary["timeout_ids"], ["M3"])
        self.assertEqual(hashlib.sha256(self.calc_py.read_bytes()).hexdigest(), self.orig_sha)

    def test_05_fewer_than_three_mutants_returns_invalid_input(self) -> None:
        mutations_path = self.proj / "mutations_short.json"
        mutations_path.write_text(
            json.dumps(
                [
                    {
                        "id": "M1",
                        "file": str(self.calc_py),
                        "target_content": "return a + b",
                        "mutant_content": "return a - b",
                    }
                ]
            ),
            encoding="utf-8",
        )
        code, summary = mg.execute_mutation_gate(
            ws_name="ws-test",
            suffix="calc_short",
            mutations_file=mutations_path,
            workspaces_dir=self.ws_root,
        )
        self.assertEqual(code, 3)
        self.assertEqual(summary["gate"], "INVALID_INPUT")


if __name__ == "__main__":
    unittest.main()
