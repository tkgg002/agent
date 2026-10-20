#!/usr/bin/env python3
"""Unit tests for tooling/build_lessons_index.py."""
from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import build_lessons_index  # noqa: E402


SAMPLE_LESSONS = """# 📚 Hard-Tech Patterns & Tripwires

### [2026-10-16] Lỗi quên truyền ancestors khi PUT Confluence (Missing ancestors on Confluence Page Update)
- **Global Pattern:** [A] gọi PUT [B] thiếu `ancestors`.
- **Tags:** #confluence #pipeline

### [2026-10-15] Lỗi gọi Form.useWatch trong nhánh JSX có điều kiện (Conditional React Hook Call)
- **Global Pattern:** [A] gọi hook trong ternary JSX.
- **Tags:** #react #frontend

### [2026-10-14] Lỗi UNIQUE constraint PostgreSQL SQLSTATE 23505 (Postgres Unique Constraint Collision)
- **Global Pattern:** [A] insert trùng key.
- **Tags:** #postgres #sql

### [2026-10-13] Lỗi đóng sqlDB.Close() trước khi goroutine dừng hẳn (Goroutine Shutdown Race)
- **Global Pattern:** [A] close DB khi worker đang chạy.
- **Tags:** #golang #concurrency

### [2026-10-12] Tự ý sửa code khi đang Audit (Violation of Propose-Only Rule #13)
- **Global Pattern:** [A] sửa code khi chưa có APPROVE.
- **Tags:** #governance #audit-propose-only
"""


class TestBuildLessonsIndex(unittest.TestCase):
    def test_build_index_conserves_entries_and_respects_size(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            src = tmp_path / "lessons.md"
            out = tmp_path / "lessons"
            src.write_text(SAMPLE_LESSONS, encoding="utf-8")

            stats = build_lessons_index.build_index(src, out)
            self.assertEqual(stats["source_entries"], 5)
            self.assertEqual(stats["index_entries"], 5)
            self.assertEqual(stats["topic_entries"], 5)
            self.assertLess(stats["index_bytes"], build_lessons_index.MAX_INDEX_BYTES)
            # Source file must remain 100% untouched
            self.assertEqual(src.read_text(encoding="utf-8"), SAMPLE_LESSONS)
            self.assertTrue((out / "INDEX.md").is_file())
            self.assertEqual(len(list((out / "topics").glob("*.md"))), 5)


if __name__ == "__main__":
    unittest.main()
