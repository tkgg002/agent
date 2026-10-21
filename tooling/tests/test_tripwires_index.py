#!/usr/bin/env python3
"""Tests for tripwires.json generation in build_lessons_index.py (G-08)."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import build_lessons_index  # noqa: E402

LESSONS = """# Lessons

### [2026-10-21] Lesson có tripwire (Has Tripwire)
- **Global Pattern:** [A] làm [B].
- **Tags:** #governance
- **ID:** LS-20261021-has-trip
- **Trigger:** github\\.com/[\\w.-]+/[\\w.-]+
- **Check:** Mở source thật trước khi kết luận.
- **Require:** (git\\s+clone|/scratch/)

### [2026-10-20] Lesson cũ chưa có tripwire (Legacy No Tripwire)
- **Global Pattern:** [A] làm [B].
- **Tags:** #governance
"""


class TestTripwiresIndex(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = Path(self._tmp.name)
        self.src = self.tmp / "lessons.md"
        self.out = self.tmp / "out"

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def test_tripwires_json_written_with_fields(self) -> None:
        self.src.write_text(LESSONS, encoding="utf-8")
        stats = build_lessons_index.build_index(self.src, self.out)
        trips = json.loads((self.out / "tripwires.json").read_text(encoding="utf-8"))
        self.assertEqual(len(trips), 1)
        t = trips[0]
        self.assertEqual(t["id"], "LS-20261021-has-trip")
        self.assertEqual(t["trigger"], r"github\.com/[\w.-]+/[\w.-]+")
        self.assertEqual(t["check"], "Mở source thật trước khi kết luận.")
        self.assertEqual(t["require"], r"(git\s+clone|/scratch/)")
        self.assertEqual(stats["tripwires"], 1)
        self.assertEqual(stats["lessons_without_tripwire"], 1)

    def test_invalid_trigger_regex_fails_build(self) -> None:
        self.src.write_text(LESSONS.replace(r"github\.com/[\w.-]+/[\w.-]+", "(["), encoding="utf-8")
        with self.assertRaises(RuntimeError):
            build_lessons_index.build_index(self.src, self.out)

    def test_duplicate_id_fails_build(self) -> None:
        dup = LESSONS + LESSONS.split("# Lessons\n", 1)[1].split("### [2026-10-20]")[0]
        self.src.write_text(dup, encoding="utf-8")
        with self.assertRaises(RuntimeError):
            build_lessons_index.build_index(self.src, self.out)

    def test_repeated_field_last_value_wins_for_append_only_amendment(self) -> None:
        amended = LESSONS.replace(
            "- **Require:** (git\\s+clone|/scratch/)\n",
            "- **Require:** (git\\s+clone|/scratch/)\n- **Require:** git\\s+clone\n",
        )
        self.src.write_text(amended, encoding="utf-8")
        build_lessons_index.build_index(self.src, self.out)
        trips = json.loads((self.out / "tripwires.json").read_text(encoding="utf-8"))
        self.assertEqual(trips[0]["require"], r"git\s+clone")

    def test_partial_tripwire_fields_fail_build(self) -> None:
        self.src.write_text(LESSONS.replace("- **Check:** Mở source thật trước khi kết luận.\n", ""), encoding="utf-8")
        with self.assertRaises(RuntimeError):
            build_lessons_index.build_index(self.src, self.out)


if __name__ == "__main__":
    unittest.main()
