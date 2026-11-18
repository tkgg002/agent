#!/usr/bin/env python3
"""Build 3-tier Lessons Index (INDEX.md + topics/*.summary.md + topics/*.md + tripwires.json).

Tiers (read in this order to keep input tokens low):
  Tier 1 `INDEX.md`          — topic routing + core tripwires only (< 4,000 bytes, always read at S0).
  Tier 2 `topics/*.summary.md` — ONE line per lesson (id, full title, pattern gist, tags); open the
                                 single file matching the task domain (< 8,000 bytes each).
  Tier 3 `topics/*.md`       — full 5-bullet lesson blocks; NEVER read whole, grep by id:
                                 `grep -A12 'ID: L0xx' topics/0X_*.md`.

Guarantees:
1. READ-ONLY on memory/global/lessons.md (never modifies or truncates lessons.md).
2. 100% entry conservation: count(summary entries) == sum(topic entries) == count(lessons.md entries).
3. INDEX.md < 15,000 bytes and every summary file < 8,000 bytes so S0 Boot stays cheap.
4. Global Hygiene Guard: Enforces 5-bullet schema, max line budget per entry, English summary in parens,
   and blocks local path/task leakage when Agent appends new lessons.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_SOURCE = AGENT_ROOT / "memory" / "global" / "lessons.md"
DEFAULT_OUT_DIR = AGENT_ROOT / "memory" / "global" / "lessons"
MAX_INDEX_BYTES = 15000
MAX_SUMMARY_BYTES = 8000  # per topics/*.summary.md (Tier 2) — keeps a domain lookup ~1-3k tokens
MAX_ENTRY_LINES = 12
SUMMARY_SUFFIX = ".summary.md"

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

# Global Hygiene Guard constants
REQUIRED_BULLETS: Tuple[str, ...] = (
    "- **Global Pattern:**",
    "- **Bối cảnh (Trigger):**",
    "- **Root Cause:**",
    "- **Fix/Correct Flow:**",
    "- **Tags:**",
)
LOCAL_PATH_LEAK_RE = re.compile(r"/Users/\w+/Documents/work/", re.I)

# G-08: lesson tripwire fields (máy đọc được) — ID ổn định, Trigger regex, Check tiền điều kiện, Require bằng chứng tuỳ chọn.
TRIP_FIELD_RE = re.compile(r"^- \*\*(ID|Trigger|Check|Require):\*\*\s*(.+?)\s*$", re.M)
TRIP_REQUIRED = ("id", "trigger", "check")


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


def validate_global_hygiene(entries: List["LessonEntry"]) -> None:
    """Enforce concise, 100% global structure on every lesson entry."""
    if not entries:
        raise RuntimeError("CRITICAL: No lesson entries parsed from lessons.md!")
    for e in entries:
        block_lines = [l for l in e.raw_block.splitlines() if l.strip()]
        if len(block_lines) > MAX_ENTRY_LINES:
            raise RuntimeError(
                f"[GLOBAL HYGIENE BLOCK] Bài học tại lessons.md:L{e.line_no} dài {len(block_lines)} dòng "
                f"(vượt trần {MAX_ENTRY_LINES} dòng). Phải tóm tắt gọn đúng 1 dòng cho mỗi trường!"
            )
        for req_bullet in REQUIRED_BULLETS:
            if req_bullet not in e.raw_block:
                raise RuntimeError(
                    f"[GLOBAL HYGIENE BLOCK] Bài học tại lessons.md:L{e.line_no} thiếu trường bắt buộc '{req_bullet}'."
                )
        if LOCAL_PATH_LEAK_RE.search(e.raw_block):
            raise RuntimeError(
                f"[GLOBAL HYGIENE BLOCK] Bài học tại lessons.md:L{e.line_no} chứa đường dẫn máy cục bộ '/Users/...'. "
                f"Phải trừu tượng hóa 100% thành Global Pattern!"
            )
        if not PAREN_EN_RE.search(e.title):
            raise RuntimeError(
                f"[GLOBAL HYGIENE BLOCK] Tiêu đề tại lessons.md:L{e.line_no} thiếu cụm tóm tắt Tiếng Anh "
                f"trong ngoặc đơn '(...)' (tối thiểu 10 ký tự) ở cuối dòng để tạo nhãn cho INDEX.md."
            )


def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:
    fields = {k.lower(): v for k, v in TRIP_FIELD_RE.findall(entry.raw_block)}
    if not fields:
        return None
    missing = [k for k in TRIP_REQUIRED if not fields.get(k)]
    if missing:
        raise RuntimeError(f"Tripwire thiếu {missing} tại lessons.md:L{entry.line_no}")
    for key in ("trigger", "require"):
        if fields.get(key):
            try:
                re.compile(fields[key])
            except re.error as exc:
                raise RuntimeError(f"Tripwire {key} regex lỗi tại lessons.md:L{entry.line_no}: {exc}") from exc
    return {k: fields[k] for k in ("id", "trigger", "check", "require") if fields.get(k)}


def build_tripwires(entries: List["LessonEntry"]) -> List[Dict[str, str]]:
    trips: List[Dict[str, str]] = []
    seen: Dict[str, int] = {}
    for e in entries:
        t = parse_tripwire(e)
        if not t:
            continue
        if t["id"] in seen:
            raise RuntimeError(f"Tripwire ID trùng {t['id']} tại lessons.md:L{seen[t['id']]} và L{e.line_no}")
        seen[t["id"]] = e.line_no
        trips.append(t)
    return trips


def concise_label(title: str, max_chars: int = 38) -> str:
    """Extract concise ASCII/UTF-8 label (prefer English summary in parens if present)."""
    m = PAREN_EN_RE.search(title.strip())
    candidate = m.group(1).strip() if m else title.strip()
    candidate = candidate.replace("`", "").replace("|", "/")
    candidate = re.sub(r"\s+", " ", candidate)
    if len(candidate) > max_chars:
        candidate = candidate[: max_chars - 1].rstrip() + "…"
    return candidate


def _bullet_value(body: str, bullet: str) -> str:
    """Return the text right after a '- **Name:**' bullet, single-line and squeezed."""
    for line in body.splitlines():
        stripped = line.strip()
        if stripped.startswith(bullet):
            return re.sub(r"\s+", " ", stripped[len(bullet) :]).strip()
    return ""


def pattern_gist(body: str, max_chars: int = 150) -> str:
    """One-sentence gist of the lesson, taken from 'Global Pattern' (fallback: 'Fix/Correct Flow')."""
    raw = _bullet_value(body, "- **Global Pattern:**") or _bullet_value(body, "- **Fix/Correct Flow:**")
    raw = raw.replace("|", "/").replace("`", "")
    # The pattern bullet is written as "Khi ... -> Hậu quả [Y]: ..."; the part before "->" carries the trigger.
    head = raw.split("->", 1)[0].strip() if "->" in raw else raw
    gist = head if len(head) >= 40 else raw
    if len(gist) > max_chars:
        gist = gist[: max_chars - 1].rstrip() + "…"
    return gist


def entry_tags(body: str, max_chars: int = 60) -> str:
    """The '#kebab-case' tag list, used to grep a lesson without opening the full block."""
    tags = _bullet_value(body, "- **Tags:**").replace("|", "/").replace("`", "")
    if len(tags) > max_chars:
        tags = tags[: max_chars - 1].rstrip() + "…"
    return tags


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


def render_summary(code: str, desc: str, fname: str, bucket: List[LessonEntry]) -> str:
    """Render a Tier-2 summary table, shrinking the gist budget until it fits MAX_SUMMARY_BYTES."""
    header = [
        f"# {code} SUMMARY: {desc}",
        f"> **{len(bucket)}** bài học, mỗi dòng 1 bài. Đọc file này TRƯỚC; chỉ mở bài đầy đủ khi cần:",
        f"> `grep -A12 'ID: L0xx' memory/global/lessons/topics/{fname}`",
        "",
        "| ID | Bài học | Khi nào trúng (Global Pattern) | Tags |",
        "| :--- | :--- | :--- | :--- |",
    ]
    out = ""
    for title_max, gist_max, tags_max in ((70, 150, 60), (60, 110, 48), (52, 80, 40), (44, 56, 32), (40, 40, 24)):
        rows = [
            f"| `L{e.idx:03d}` | {concise_label(e.title, max_chars=title_max)} "
            f"| {pattern_gist(e.raw_block, max_chars=gist_max)} | {entry_tags(e.raw_block, max_chars=tags_max)} |"
            for e in bucket
        ]
        out = "\n".join(header + rows).rstrip() + "\n"
        if len(out.encode("utf-8")) < MAX_SUMMARY_BYTES:
            break
    return out


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

        # Tier 2: one line per lesson so a domain lookup costs ~1-3k tokens instead of ~50k.
        topic_files[fname.replace(".md", SUMMARY_SUFFIX)] = render_summary(code, desc, fname, bucket)

    idx_lines = [
        "# 🧭 LESSONS INDEX (3-Tier Memory Map)",
        f"> **Tổng số bài học:** `{len(entries)}` | **Nguồn gốc:** `memory/global/lessons.md` (Read-Only)",
        "> **Cách dùng tại S0 Boot (tiết kiệm token):** (1) Đọc bảng dưới; (2) mở ĐÚNG MỘT file"
        " `topics/0X_*.summary.md` khớp domain của task; (3) chỉ khi cần chi tiết mới trích 1 bài theo ID:"
        " `grep -A12 'ID: L0xx' topics/0X_*.md`. **CẤM** đọc nguyên `topics/0X_*.md` hay `lessons.md`.",
        "",
        "## 1. Bản đồ Chủ đề (Topic Routing)",
        "| Mã | Tier 2 — đọc file này | Số lượng | Khi nào cần đọc |",
        "| :--- | :--- | :---: | :--- |",
    ]
    for code, fname, desc, _ in TOPICS:
        summary_name = fname.replace(".md", SUMMARY_SUFFIX)
        idx_lines.append(f"| `{code}` | `topics/{summary_name}` | {len(by_topic[code])} | {desc} |")

    idx_lines.extend(
        [
            "",
            "## 2. Top Luật Vàng Bất Biến (Core Tripwires)",
            "- **G1 (No Overwrite):** Cấm `Overwrite=true` lên file memory; chỉ append vào `05_progress`, `04_decisions`, `lessons.md`.",
            "- **G2 (Propose-Only & Audit):** Khi Audit hoặc chưa có `APPROVE` của User sau `09_tasks_solution` -> cấm sửa source code.",
            "- **G3 (Laya Gate):** Mọi giải pháp (DG0/DG1), code demo (DG2), audit (DG3) phải qua `laya_gate.py` (`PASS`).",
            "- **G4 (Runtime Exercise):** Build OK != Done. Phải chạy `checks` và `runtime_check` từ `.agent-project.json`.",
            "- **G5 (Clean Teardown & Locking):** Dừng goroutine nền trước khi đóng DB/Redis; TTL lock phải khớp chu kỳ ticker.",
            "- **G6 (Global Lesson Hygiene):** Mọi bài học append vào `lessons.md` phải <= 12 dòng, đủ 5 trường, 100% Global (cấm chứa `/Users/...`).",
            "",
            "## 3. Tra cứu một bài học cụ thể",
            f"Danh mục `L001`–`L{len(entries):03d}` nằm trong các file Tier 2 ở mục 1 (mỗi bài 1 dòng kèm"
            " Global Pattern + Tags). Khi đã biết ID, lấy nguyên bản 5 trường bằng:",
            "```bash",
            "grep -A12 'ID: L0xx' /Users/trainguyen/Documents/work/agent/memory/global/lessons/topics/*.md",
            "```",
        ]
    )

    index_md = "\n".join(idx_lines) + "\n"
    if len(index_md.encode("utf-8")) >= MAX_INDEX_BYTES:
        raise RuntimeError(
            f"INDEX.md render {len(index_md.encode('utf-8'))} bytes >= {MAX_INDEX_BYTES}: "
            "mục 1/2 đã phình, cắt bớt thủ công thay vì nén nhãn."
        )

    return index_md, topic_files


def _write_bundle(
    target_dir: Path, index_md: str, topic_files: Dict[str, str], tripwires: Optional[List[Dict[str, str]]] = None
) -> None:
    topics_dir = target_dir / "topics"
    topics_dir.mkdir(parents=True, exist_ok=True)
    (target_dir / "INDEX.md").write_text(index_md, encoding="utf-8")
    for fname, content in topic_files.items():
        (topics_dir / fname).write_text(content, encoding="utf-8")
    (target_dir / "tripwires.json").write_text(
        json.dumps(tripwires or [], ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


def build_index(
    source_path: Path = DEFAULT_SOURCE,
    out_dir: Path = DEFAULT_OUT_DIR,
    enforce_hygiene: bool = True,
) -> Dict[str, int]:
    before_text = source_path.read_text(encoding="utf-8")
    before_lines = len(before_text.splitlines())

    entries = parse_lessons(before_text)
    if enforce_hygiene:
        validate_global_hygiene(entries)
    index_md, topic_files = render_outputs(entries)
    tripwires = build_tripwires(entries)

    try:
        _write_bundle(out_dir, index_md, topic_files, tripwires)
    except PermissionError:
        fallback_dir = AGENT_ROOT / ".agents" / "lessons_index"
        _write_bundle(fallback_dir, index_md, topic_files, tripwires)

    after_text = source_path.read_text(encoding="utf-8")
    after_lines = len(after_text.splitlines())
    if before_lines != after_lines or before_text != after_text:
        raise RuntimeError("CRITICAL: lessons.md was modified during build_index!")

    # Tier 2 rows (`| `L0xx` | ...`) replace the old flat INDEX catalogue as the per-lesson listing.
    summary_entry_count = sum(
        sum(1 for line in content.splitlines() if re.match(r"^\|\s+`L\d{3}`\s+\|", line))
        for name, content in topic_files.items()
        if name.endswith(SUMMARY_SUFFIX)
    )
    topic_entry_count = sum(
        sum(1 for line in content.splitlines() if H3_RE.match(line))
        for name, content in topic_files.items()
        if not name.endswith(SUMMARY_SUFFIX)
    )
    index_bytes = len(index_md.encode("utf-8"))
    summary_bytes = {
        name: len(content.encode("utf-8"))
        for name, content in topic_files.items()
        if name.endswith(SUMMARY_SUFFIX)
    }

    if summary_entry_count != len(entries) or topic_entry_count != len(entries):
        raise RuntimeError(
            f"Entry count mismatch: source={len(entries)}, summaries={summary_entry_count}, topics={topic_entry_count}"
        )
    if index_bytes >= MAX_INDEX_BYTES:
        raise RuntimeError(f"INDEX.md size {index_bytes} exceeds limit {MAX_INDEX_BYTES} bytes")
    for name, size in summary_bytes.items():
        if size >= MAX_SUMMARY_BYTES:
            raise RuntimeError(f"{name} size {size} exceeds limit {MAX_SUMMARY_BYTES} bytes")

    return {
        "source_lines": after_lines,
        "source_entries": len(entries),
        "summary_entries": summary_entry_count,
        "topic_entries": topic_entry_count,
        "index_bytes": index_bytes,
        "summary_bytes_max": max(summary_bytes.values()) if summary_bytes else 0,
        "tripwires": len(tripwires),
        "lessons_without_tripwire": len(entries) - len(tripwires),
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Build 2-tier lessons index (INDEX.md + topics/*.md + tripwires.json)")
    ap.add_argument("--source", default=str(DEFAULT_SOURCE), help="Path to lessons.md")
    ap.add_argument("--out-dir", default=str(DEFAULT_OUT_DIR), help="Output directory for INDEX.md and topics/")
    ap.add_argument(
        "--skip-hygiene",
        action="store_true",
        help="Skip Global Hygiene validation (only use for legacy uncleaned files)",
    )
    args = ap.parse_args()

    stats = build_index(
        Path(args.source),
        Path(args.out_dir),
        enforce_hygiene=not args.skip_hygiene,
    )
    print(
        f"🟢 [LESSONS INDEX] Built successfully: "
        f"source_lines={stats['source_lines']}, "
        f"entries={stats['source_entries']}/{stats['summary_entries']}/{stats['topic_entries']}, "
        f"tripwires={stats['tripwires']}, "
        f"index_bytes={stats['index_bytes']} (<{MAX_INDEX_BYTES}), "
        f"summary_bytes_max={stats['summary_bytes_max']} (<{MAX_SUMMARY_BYTES})"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())