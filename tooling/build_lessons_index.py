#!/usr/bin/env python3
"""Build 2-tier Lessons Index (INDEX.md + topics/*.md) from memory/global/lessons.md.

Guarantees:
1. READ-ONLY on memory/global/lessons.md (never modifies or truncates lessons.md).
2. 100% entry conservation: count(INDEX entries) == sum(topic entries) == count(lessons.md entries).
3. INDEX.md size strictly < 15,000 bytes so S0 Boot fits comfortably in context budget.
"""
from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_SOURCE = AGENT_ROOT / "memory" / "global" / "lessons.md"
DEFAULT_OUT_DIR = AGENT_ROOT / "memory" / "global" / "lessons"
MAX_INDEX_BYTES = 15000

TOPICS: List[Tuple[str, str, str, re.Pattern[str]]] = [
    (
        "T1",
        "01_governance_and_workflow.md",
        "Governance, Brain/Muscle, Workspace & Process Discipline",
        re.compile(
            r"\b(governance|propose-only|propose\s+only|brain|muscle|workspace|05_progress|"
            r"09_tasks_solution|lessons\.md|overwrite|approve|audit\s+phase|rule\s+#?\d+|"
            r"pre-flight|mid-session-fix|blind-fixing|no-speculation|no-guessing)\b",
            re.I,
        ),
    ),
    (
        "T2",
        "02_golang_and_concurrency.md",
        "Golang, GORM, Concurrency, Redis & Backend Runtime",
        re.compile(
            r"\b(golang|\bgo\b|gorm|goroutine|channel|mutex|shutdown|redis|setnx|ttl|"
            r"ticker|race\s+condition|bind:|dto|pointer|panic|context\.|sqldb|waitgroup)\b",
            re.I,
        ),
    ),
    (
        "T3",
        "03_database_and_sql.md",
        "Database, PostgreSQL, MongoDB & SQL",
        re.compile(
            r"\b(postgres|postgresql|sql|sqlstate|unique|index|migration|mongodb|mongo|"
            r"bson|schema|deadlock|transaction|isolation|partition|upsert|\.scan\(|limit\s+1|ddl|table)\b",
            re.I,
        ),
    ),
    (
        "T4",
        "04_pipeline_cdc_and_kafka.md",
        "Data Pipeline, CDC, Debezium, Kafka & External APIs",
        re.compile(
            r"\b(cdc|debezium|kafka|outbox|pipeline|connector|confluence|webhook|"
            r"consumer|producer|topic|offset|watermark|recon|reconciliation|shadow|master|lineage|dlq|nats)\b",
            re.I,
        ),
    ),
    (
        "T5",
        "05_frontend_and_ui.md",
        "Frontend, React, AntD & UI/UX",
        re.compile(
            r"\b(react|jsx|tsx|antd|usewatch|useeffect|usestate|usememo|usecallback|"
            r"tailwind|css|dom|vite|playwright|modal|tooltip|cascading\s+select|frontend)\b",
            re.I,
        ),
    ),
]

H3_RE = re.compile(r"^###\s+\[([^\]]+)\]\s+(.*)$")
PAREN_EN_RE = re.compile(r"\(([^()]{10,})\)\s*$")


@dataclass
class LessonEntry:
    idx: int
    date: str
    title: str
    line_no: int
    raw_block: str
    topic_code: str
    topic_file: str
    short_label: str


def concise_label(title: str, max_chars: int = 38) -> str:
    """Extract concise ASCII/UTF-8 label (prefer English summary in parens if present)."""
    m = PAREN_EN_RE.search(title.strip())
    candidate = m.group(1).strip() if m else title.strip()
    candidate = candidate.replace("`", "").replace("|", "/")
    candidate = re.sub(r"\s+", " ", candidate)
    if len(candidate) > max_chars:
        candidate = candidate[: max_chars - 1].rstrip() + "…"
    return candidate


def classify_topic(title: str, body: str) -> Tuple[str, str]:
    tags_line = "\n".join(
        l for l in body.splitlines() if "tags:" in l.lower()
    )
    header_zone = f"{title}\n{tags_line}"
    best_code = "T1"
    best_file = "01_governance_and_workflow.md"
    best_score = 0
    # Priority tie-break order: T5, T3, T2, T4, T1
    by_code = {code: (fname, pat) for code, fname, _, pat in TOPICS}
    for code in ("T5", "T3", "T2", "T4", "T1"):
        fname, pat = by_code[code]
        score = len(pat.findall(header_zone)) * 4 + len(pat.findall(body))
        if score > best_score:
            best_score = score
            best_code = code
            best_file = fname
    return best_code, best_file


def parse_lessons(source_text: str) -> List[LessonEntry]:
    lines = source_text.splitlines()
    h3_positions: List[Tuple[int, str, str]] = []
    for i, line in enumerate(lines):
        m = H3_RE.match(line)
        if m:
            h3_positions.append((i, m.group(1).strip(), m.group(2).strip()))

    entries: List[LessonEntry] = []
    for pos_idx, (start_line, date_str, title) in enumerate(h3_positions):
        end_line = h3_positions[pos_idx + 1][0] if pos_idx + 1 < len(h3_positions) else len(lines)
        block_lines: List[str] = []
        for l in lines[start_line:end_line]:
            if l.startswith("## ") and block_lines:
                break
            block_lines.append(l)
        while block_lines and not block_lines[-1].strip():
            block_lines.pop()
        raw_block = "\n".join(block_lines)
        tcode, tfname = classify_topic(title, raw_block)
        entries.append(
            LessonEntry(
                idx=pos_idx + 1,
                date=date_str,
                title=title,
                line_no=start_line + 1,
                raw_block=raw_block,
                topic_code=tcode,
                topic_file=tfname,
                short_label=concise_label(title, max_chars=28),
            )
        )
    return entries


def render_outputs(entries: List[LessonEntry]) -> Tuple[str, Dict[str, str]]:
    by_topic: Dict[str, List[LessonEntry]] = {code: [] for code, _, _, _ in TOPICS}
    for e in entries:
        by_topic[e.topic_code].append(e)

    topic_files: Dict[str, str] = {}
    for code, fname, desc, _ in TOPICS:
        bucket = by_topic[code]
        parts = [
            f"# {code}: {desc}",
            f"> Tổng số bài học trong chủ đề này: **{len(bucket)}** (trích xuất nguyên bản từ `memory/global/lessons.md`).",
            "",
        ]
        for e in bucket:
            parts.append(f"<!-- ID: L{e.idx:03d} | lessons.md:L{e.line_no} -->")
            parts.append(e.raw_block)
            parts.append("")
        topic_files[fname] = "\n".join(parts).rstrip() + "\n"

    idx_lines = [
        "# 🧭 LESSONS INDEX (2-Tier Memory Map)",
        f"> **Tổng số bài học:** `{len(entries)}` | **Nguồn gốc:** `memory/global/lessons.md` (Read-Only)",
        "> **Cách dùng tại S0 Boot:** Đọc bảng chủ đề dưới đây, chỉ mở file `topics/0X_*.md` tương ứng với domain của task.",
        "",
        "## 1. Bản đồ Chủ đề (Topic Routing)",
        "| Mã | File Chủ đề | Số lượng | Khi nào cần đọc |",
        "| :--- | :--- | :---: | :--- |",
    ]
    for code, fname, desc, _ in TOPICS:
        idx_lines.append(f"| `{code}` | `topics/{fname}` | {len(by_topic[code])} | {desc} |")

    idx_lines.extend(
        [
            "",
            "## 2. Top Luật Vàng Bất Biến (Core Tripwires)",
            "- **G1 (No Overwrite):** Cấm `Overwrite=true` lên file memory; chỉ append vào `05_progress`, `04_decisions`, `lessons.md`.",
            "- **G2 (Propose-Only & Audit):** Khi Audit hoặc chưa có `APPROVE` của User sau `09_tasks_solution` -> cấm sửa source code.",
            "- **G3 (Laya Gate):** Mọi giải pháp (DG1), code demo (DG2), audit (DG3) phải qua `laya_gate.py` (`PASS`).",
            "- **G4 (Runtime Exercise):** Build OK != Done. Phải chạy `checks` và `runtime_check` từ `.agent-project.json`.",
            "- **G5 (Clean Teardown & Locking):** Dừng goroutine nền trước khi đóng DB/Redis; TTL lock phải khớp chu kỳ ticker.",
            "",
            "## 3. Danh mục 100% Bài học (`L001`–`L{:03d}`)".format(len(entries)),
        ]
    )

    for e in entries:
        idx_lines.append(f"- L{e.idx:03d} [{e.date}] {e.topic_code}:L{e.line_no} {e.short_label}")

    index_md = "\n".join(idx_lines) + "\n"
    for budget in (24, 20, 16, 12, 8):
        if len(index_md.encode("utf-8")) < MAX_INDEX_BYTES:
            break
        idx_lines_tight = idx_lines[: -len(entries)]
        for e in entries:
            lbl = concise_label(e.title, max_chars=budget)
            idx_lines_tight.append(f"- L{e.idx:03d} [{e.date}] {e.topic_code}:L{e.line_no} {lbl}")
        index_md = "\n".join(idx_lines_tight) + "\n"

    return index_md, topic_files


def _write_bundle(target_dir: Path, index_md: str, topic_files: Dict[str, str]) -> None:
    topics_dir = target_dir / "topics"
    topics_dir.mkdir(parents=True, exist_ok=True)
    (target_dir / "INDEX.md").write_text(index_md, encoding="utf-8")
    for fname, content in topic_files.items():
        (topics_dir / fname).write_text(content, encoding="utf-8")


def build_index(source_path: Path = DEFAULT_SOURCE, out_dir: Path = DEFAULT_OUT_DIR) -> Dict[str, int]:
    before_text = source_path.read_text(encoding="utf-8")
    before_lines = len(before_text.splitlines())

    entries = parse_lessons(before_text)
    index_md, topic_files = render_outputs(entries)

    try:
        _write_bundle(out_dir, index_md, topic_files)
    except PermissionError:
        fallback_dir = AGENT_ROOT / ".agents" / "lessons_index"
        _write_bundle(fallback_dir, index_md, topic_files)

    after_text = source_path.read_text(encoding="utf-8")
    after_lines = len(after_text.splitlines())
    if before_lines != after_lines or before_text != after_text:
        raise RuntimeError("CRITICAL: lessons.md was modified during build_index!")

    index_entry_count = sum(
        1 for line in index_md.splitlines() if re.match(r"^-\s+L\d{3}\b", line)
    )
    topic_entry_count = sum(
        sum(1 for line in content.splitlines() if H3_RE.match(line))
        for content in topic_files.values()
    )
    index_bytes = len(index_md.encode("utf-8"))

    if index_entry_count != len(entries) or topic_entry_count != len(entries):
        raise RuntimeError(
            f"Entry count mismatch: source={len(entries)}, index={index_entry_count}, topics={topic_entry_count}"
        )
    if index_bytes >= MAX_INDEX_BYTES:
        raise RuntimeError(f"INDEX.md size {index_bytes} exceeds limit {MAX_INDEX_BYTES} bytes")

    return {
        "source_lines": after_lines,
        "source_entries": len(entries),
        "index_entries": index_entry_count,
        "topic_entries": topic_entry_count,
        "index_bytes": index_bytes,
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Build 2-tier lessons index (INDEX.md + topics/*.md)")
    ap.add_argument("--source", default=str(DEFAULT_SOURCE), help="Path to lessons.md")
    ap.add_argument("--out-dir", default=str(DEFAULT_OUT_DIR), help="Output directory for INDEX.md and topics/")
    args = ap.parse_args()

    stats = build_index(Path(args.source), Path(args.out_dir))
    print(
        f"🟢 [LESSONS INDEX] Built successfully: "
        f"source_lines={stats['source_lines']}, "
        f"entries={stats['source_entries']}/{stats['index_entries']}/{stats['topic_entries']}, "
        f"index_bytes={stats['index_bytes']} (<{MAX_INDEX_BYTES})"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
