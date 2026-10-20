#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""verify_governance.py — Process Linter kiểm soát quy trình Agent theo Tier (T0–T3).

Usage:
  python3 tooling/verify_governance.py --workspace <workspace_name> [--tier T0|T1|T2|T3]
  python3 tooling/verify_governance.py --tier T0
"""
from __future__ import annotations

import argparse
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import List, Optional, Sequence, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WORKSPACES_DIR = AGENT_ROOT / "memory" / "workspaces"

TIER_RE = re.compile(r"^\s*tier\s*:\s*(T[0-3])\b", re.I | re.M)
LOG_LINE_RE = re.compile(r"^-\s*\[\d{4}-\d{2}-\d{2}[^\]]*\]\s*\[Agent:[a-zA-Z0-9.\-_]+\]\s+.+")

# Mỗi phần tử là (nhãn mô tả, tuple các glob pattern chấp nhận 1 trong số đó)
TIER_RULES: dict[str, List[Tuple[str, Tuple[str, ...]]]] = {
    "T1": [
        ("01_requirements_*.md", ("01_requirements*.md",)),
        ("05_progress_*.md", ("05_progress*.md",)),
        ("08_tasks_*.md", ("08_tasks*.md",)),
    ],
    "T2": [
        ("01_requirements_*.md", ("01_requirements*.md",)),
        ("05_progress_*.md", ("05_progress*.md",)),
        ("08_tasks_*.md", ("08_tasks*.md",)),
        ("09_tasks_solution_*.md", ("09_tasks_solution*.md",)),
        ("06_validation_*.md / 06_test_cases_*.md", ("06_validation*.md", "06_test_cases*.md")),
        ("11_report_*.md", ("11_report*.md",)),
        ("audit_report_*.md", ("audit_report*.md",)),
    ],
    "T3": [
        ("00_context_*.md", ("00_context*.md",)),
        ("01_requirements_*.md", ("01_requirements*.md",)),
        ("02_plan_*.md", ("02_plan*.md",)),
        ("04_decisions_*.md", ("04_decisions*.md",)),
        ("05_progress_*.md", ("05_progress*.md",)),
        ("06_validation_*.md / 06_test_cases_*.md", ("06_validation*.md", "06_test_cases*.md")),
        ("08_tasks_*.md", ("08_tasks*.md",)),
        ("09_tasks_solution_*.md", ("09_tasks_solution*.md",)),
        ("11_report_*.md", ("11_report*.md",)),
        ("12_implementation_plan_*.md", ("12_implementation_plan*.md", "implementation_plan.md")),
        ("13_analysis_*.md", ("13_analysis*.md",)),
        ("audit_report_*.md", ("audit_report*.md",)),
    ],
}

STAGE_S3_PREFIXES = ("01_requirements", "05_progress", "08_tasks", "09_tasks_solution")
SUFFIX_SCOPED_PREFIXES = ("01_requirements", "08_tasks", "09_tasks_solution", "audit_report")

AUDIT_REQUIRED_PATTERNS: List[Tuple[str, str]] = [
    (r"(line-by-line|từng dòng|git diff)", "Mục 1: Phản biện từng dòng update / git diff & từng task"),
    (r"(mockup|hallucination|runtime|trace|smoke)", "Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật"),
    (r"(concurrency|multi-worker|worker|timeout|heartbeat|ackwait|watermark|continuity|debouncer)", "Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity"),
    (r"(suy diễn|báo cáo láo|speculation|trung thực|honesty)", "Mục 4: Kiểm tra suy diễn & báo cáo láo"),
    (r"(self-improvement|phản tỉnh|lessons)", "Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop)"),
    (r"(laya_router|laya_gate|dg3)", "Mục 6: Kết quả QC & nhận xét từ /laya_router (DG3)"),
]


def log_info(msg: str) -> None:
    print(f"🟢 [GOVERNANCE] {msg}")


def log_error(msg: str) -> None:
    print(f"🔴 [GOVERNANCE] ERROR: {msg}", file=sys.stderr)


def match_any(ws_path: Path, patterns: Sequence[str]) -> List[Path]:
    found: List[Path] = []
    for pat in patterns:
        found.extend(sorted(ws_path.glob(pat)))
    return found


def extract_doc_suffix(path: Path, prefix: str) -> Optional[str]:
    m = re.match(rf"^{re.escape(prefix)}_(.+)\.md$", path.name)
    return m.group(1) if m else None


def detect_tier(req_files: Sequence[Path], tier_override: Optional[str] = None) -> str:
    if tier_override:
        return tier_override.upper()
    for rf in req_files:
        try:
            m = TIER_RE.search(rf.read_text(encoding="utf-8"))
            if m:
                return m.group(1).upper()
        except OSError:
            continue
    return "T2"


def verify_workspace_docs(
    ws_name: str,
    workspaces_dir: Path = DEFAULT_WORKSPACES_DIR,
    tier_override: Optional[str] = None,
    today_override: Optional[str] = None,
    suffix: Optional[str] = None,
    stage: str = "S6",
) -> bool:
    if tier_override and tier_override.upper() == "T0":
        log_info("Tier T0 — không yêu cầu tạo workspace.")
        return True

    ws_path = workspaces_dir / ws_name
    if not ws_path.is_dir():
        log_error(f"Workspace '{ws_name}' không tồn tại tại {ws_path}")
        return False

    req_files = match_any(ws_path, ("01_requirements*.md",))
    if not req_files:
        log_error(f"Thiếu file 01_requirements_*.md trong {ws_path}")
        return False

    tier = detect_tier(req_files, tier_override)
    if tier == "T0":
        log_info(f"Workspace '{ws_name}' khai báo tier: T0 — bỏ qua kiểm tra doc set.")
        return True

    stage_norm = (stage or "S6").upper()
    rules = TIER_RULES.get(tier, TIER_RULES["T2"])
    if stage_norm == "S3":
        rules = [
            (label, pats)
            for label, pats in rules
            if any(label.startswith(pfx) for pfx in STAGE_S3_PREFIXES)
        ]

    log_info(
        f"Đang kiểm tra workspace: '{ws_name}' (Tier: {tier}, Stage: {stage_norm}"
        + (f", Suffix: {suffix})" if suffix else ")")
    )

    missing: List[str] = []
    for label, patterns in rules:
        if not match_any(ws_path, patterns):
            missing.append(label)

    # Kiểm tra chặt theo suffix khi chỉ định --suffix hoặc khi tái sử dụng workspace có nhiều file 09_tasks_solution
    sol_files = match_any(ws_path, ("09_tasks_solution_*.md",))
    if suffix:
        for pfx in SUFFIX_SCOPED_PREFIXES:
            if stage_norm == "S3" and pfx not in STAGE_S3_PREFIXES:
                continue
            if pfx == "audit_report" and (stage_norm != "S6" or tier not in ("T2", "T3")):
                continue
            if stage_norm == "S3" or any(lbl.startswith(pfx) for lbl, _ in rules):
                if pfx == "09_tasks_solution" and tier == "T1":
                    continue
                target_doc = ws_path / f"{pfx}_{suffix}.md"
                if not target_doc.is_file():
                    missing.append(f"{pfx}_{suffix}.md (bắt buộc cho --suffix '{suffix}')")
    elif len(sol_files) > 1:
        latest_sol = max(sol_files, key=lambda p: p.stat().st_mtime)
        auto_suffix = extract_doc_suffix(latest_sol, "09_tasks_solution")
        if auto_suffix:
            check_pfxs: Tuple[str, ...] = ("01_requirements", "08_tasks")
            if stage_norm == "S6" and tier in ("T2", "T3"):
                check_pfxs = ("01_requirements", "08_tasks", "audit_report")
            for pfx in check_pfxs:
                exact_doc = ws_path / f"{pfx}_{auto_suffix}.md"
                if exact_doc.is_file():
                    continue
                candidates = match_any(ws_path, (f"{pfx}*.md",))
                newest = max(candidates, key=lambda p: p.stat().st_mtime) if candidates else None
                updated_for_task = bool(
                    newest
                    and (
                        newest.stat().st_mtime >= latest_sol.stat().st_mtime - 600
                        or auto_suffix in newest.read_text(encoding="utf-8", errors="ignore")
                    )
                )
                if not updated_for_task:
                    missing.append(
                        f"{pfx}_{auto_suffix}.md (hoặc cập nhật {pfx}_*.md cho task '{auto_suffix}')"
                    )

    if missing:
        for m in missing:
            log_error(f"[Tier {tier}] Thiếu tài liệu bắt buộc: {m} trong {ws_path}")
        return False

    log_info(f"✓ Đầy đủ bộ tài liệu bắt buộc cho Tier {tier} ({stage_norm}).")

    # Kiểm tra nội dung 6 phần bắt buộc trong audit_report_*.md ở chặng S6 (Tier T2/T3)
    if stage_norm == "S6" and tier in ("T2", "T3"):
        audit_candidates = match_any(ws_path, ("audit_report*.md",))
        if suffix and (ws_path / f"audit_report_{suffix}.md").is_file():
            audit_file = ws_path / f"audit_report_{suffix}.md"
        else:
            audit_file = max(audit_candidates, key=lambda p: p.stat().st_mtime)
        audit_text = audit_file.read_text(encoding="utf-8", errors="ignore")
        missing_sections = [
            label for pat, label in AUDIT_REQUIRED_PATTERNS if not re.search(pat, audit_text, re.I)
        ]
        if missing_sections:
            for ms in missing_sections:
                log_error(f"File {audit_file.name} thiếu phần bắt buộc: {ms}")
            return False
        log_info(
            f"✓ File {audit_file.name} đầy đủ 6 phần Audit Phản Biện & Chống Suy Diễn Smoke Test."
        )

    progress_files = match_any(ws_path, ("05_progress*.md",))
    progress_file = max(progress_files, key=lambda p: p.stat().st_mtime)
    today_str = today_override or datetime.now().strftime("%Y-%m-%d")
    today_re = re.compile(rf"^-\s*\[{re.escape(today_str)}[^\]]*\]\s*\[Agent:[a-zA-Z0-9.\-_]+\]")

    has_today_log = False
    malformed_lines: List[Tuple[int, str]] = []

    for idx, raw_line in enumerate(progress_file.read_text(encoding="utf-8").splitlines(), 1):
        line_str = raw_line.strip()
        if line_str.startswith("- ["):
            if not LOG_LINE_RE.match(line_str):
                malformed_lines.append((idx, line_str))
            if today_re.match(line_str):
                has_today_log = True

    if malformed_lines:
        for idx, line in malformed_lines:
            log_error(f"Dòng {idx} trong {progress_file.name} sai format audit log: '{line}'")
        return False

    if not has_today_log:
        log_error(
            f"Không tìm thấy dòng log nào của ngày hôm nay ({today_str}) trong {progress_file.name}"
        )
        return False

    log_info(f"✓ File {progress_file.name} hợp lệ và đã cập nhật ngày hôm nay ({today_str}).")
    return True


def parse_args(argv: Optional[Sequence[str]] = None) -> argparse.Namespace:
    ap = argparse.ArgumentParser(description="Verify Agent Workspace Governance by Tier (T0-T3)")
    ap.add_argument("--workspace", "-w", help="Tên thư mục workspace trong memory/workspaces/")
    ap.add_argument("--tier", "-t", choices=["T0", "T1", "T2", "T3", "t0", "t1", "t2", "t3"], help="Override tier")
    ap.add_argument("--suffix", "-s", help="Suffix định danh của task hiện tại (vd: config_Hierarchy_v2)")
    ap.add_argument("--stage", choices=["S3", "S6", "s3", "s6"], default="S6", help="Chặng kiểm tra: S3 (Solution) hoặc S6 (Close)")
    ap.add_argument("--workspaces-dir", default=str(DEFAULT_WORKSPACES_DIR), help="Thư mục chứa workspaces")
    ap.add_argument("positional_ws", nargs="?", help="Tên workspace (positional fallback)")
    return ap.parse_args(argv)


def main(argv: Optional[Sequence[str]] = None) -> int:
    args = parse_args(argv)
    tier = args.tier.upper() if args.tier else None
    ws_name = args.workspace or args.positional_ws

    if tier == "T0" and not ws_name:
        log_info("Tier T0 — không yêu cầu workspace.")
        print("════════════════════════════════════════════════")
        print(" ⛳ GOVERNANCE AUDIT PASSED 🟢 (Tier: T0)")
        print("════════════════════════════════════════════════")
        return 0

    if not ws_name:
        log_error("Bắt buộc chỉ định --workspace <workspace_name> (Rule #0: cấm tự đoán bằng mtime).")
        return 1

    success = verify_workspace_docs(
        ws_name=ws_name,
        workspaces_dir=Path(args.workspaces_dir),
        tier_override=tier,
        suffix=args.suffix,
        stage=args.stage,
    )
    if success:
        print("════════════════════════════════════════════════")
        print(f" ⛳ GOVERNANCE AUDIT PASSED 🟢 (Workspace: {ws_name})")
        print("════════════════════════════════════════════════")
        return 0

    print("════════════════════════════════════════════════")
    print(f" ⛳ GOVERNANCE AUDIT FAILED 🔴 (Workspace: {ws_name})")
    print("   Vui lòng bổ sung các tài liệu/log còn thiếu trước khi bàn giao task!")
    print("════════════════════════════════════════════════")
    return 1


if __name__ == "__main__":
    sys.exit(main())
