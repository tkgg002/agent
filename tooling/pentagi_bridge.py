#!/usr/bin/env python3
"""PentAGI Bridge for /agent Master Workflow (S1 Intake & S5 DAST Trigger).

Responsibilities:
1. S1 Intake (`intake`):
   - Parses PentAGI Markdown penetration testing reports.
   - Masks all secrets, tokens, passwords, and DB credentials (Rule #20 / H3).
   - Generates standard S1 Intake workspace documents (`01_requirements_<suffix>.md`,
     `05_progress_<suffix>.md`, `08_tasks_<suffix>.md`) without overwriting existing
     memory files (Rule #17 / H2), ensuring every finding goes through
     `S2 Reproduce -> S3 Solution (DG1/DG2) -> Wait User APPROVE`.
2. S5 Dynamic Pentest Gate (`trigger`):
   - Validates target URLs strictly against `allowed_pentest_targets` from
     `<project-root>/.agent-project.json` (defaulting to localhost/127.0.0.1/[::1]).
   - Rejects non-whitelisted or external targets before any network call.
   - Triggers a scoped PentAGI flow via REST/GraphQL API.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

AGENT_ROOT = Path(os.environ.get("AGY_AGENT_ROOT", str(Path(__file__).resolve().parents[1])))
WS_ROOT = AGENT_ROOT / "memory" / "workspaces"
MANIFEST_NAME = ".agent-project.json"

# Rule #20 / H3 Secret & Credential Masking Regex
SECRET_RE = re.compile(
    r"(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ATATT[\w\-]{20,}|ghp_\w{30,}"
    r"|Bearer\s+[\w.\-]{20,}|(mongodb|postgres(ql)?|mysql|redis)://[^*\s:/]+:[^*\s@]+@"
    r"|password\s*[:=]\s*[^\s*'\"]{6,})",
    re.I,
)
DB_URI_CRED_RE = re.compile(r"((?:mongodb|postgres(?:ql)?|mysql|redis)://)[^*\s:/]+:[^*\s@]+@", re.I)

DEFAULT_ALLOWED_TARGETS = (
    "http://localhost",
    "https://localhost",
    "http://127.0.0.1",
    "https://127.0.0.1",
    "http://[::1]",
    "https://[::1]",
)

FINDING_HEADING_RE = re.compile(r"^(#{2,3})\s+(.+)$", re.M)
SEVERITY_RE = re.compile(r"\b(CRITICAL|HIGH|MEDIUM|LOW|INFO)\b", re.I)


def now_iso_vn() -> str:
    tz = dt.timezone(dt.timedelta(hours=7))
    return dt.datetime.now(tz).replace(microsecond=0).isoformat()


def mask_secrets(text: str) -> str:
    """Masks credentials, tokens, API keys, and DB URIs according to Rule #20."""
    masked = DB_URI_CRED_RE.sub(r"\1***:***@", text)
    return SECRET_RE.sub("<MASKED_SECRET>", masked)


def _matches_prefix_boundary(url_lower: str, prefix_lower: str) -> bool:
    clean_p = prefix_lower.rstrip("/")
    if not url_lower.startswith(clean_p):
        return False
    rem = url_lower[len(clean_p) :]
    return rem == "" or rem[0] in ("/", ":", "?", "#")


def validate_target_url(target_url: str, allowed_prefixes: Optional[Sequence[str]] = None) -> bool:
    """Validates that target_url uses http/https and matches allowed_pentest_targets."""
    raw = (target_url or "").strip()
    if not raw:
        return False
    parsed = urllib.parse.urlparse(raw)
    if parsed.scheme not in ("http", "https") or not parsed.netloc:
        return False
    prefixes = list(allowed_prefixes) if allowed_prefixes else list(DEFAULT_ALLOWED_TARGETS)
    u_low = raw.lower()
    return any(_matches_prefix_boundary(u_low, p.strip().lower()) for p in prefixes if p and p.strip())


def load_allowed_targets_from_manifest(project_root: Optional[Path]) -> Optional[List[str]]:
    """Loads allowed_pentest_targets from <project-root>/.agent-project.json if configured."""
    if not project_root:
        return None
    manifest_path = Path(project_root) / MANIFEST_NAME
    if not manifest_path.is_file():
        return None
    try:
        data = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    targets: List[str] = []
    root_targets = data.get("allowed_pentest_targets")
    if isinstance(root_targets, list):
        targets.extend(str(x) for x in root_targets if str(x).strip())
    for comp in data.get("components", []):
        if isinstance(comp, dict):
            comp_targets = comp.get("allowed_pentest_targets")
            if isinstance(comp_targets, list):
                targets.extend(str(x) for x in comp_targets if str(x).strip())
    return targets or None


def parse_findings_from_markdown(md_text: str) -> List[Dict[str, str]]:
    """Extracts structured security findings from a PentAGI Markdown report."""
    clean_text = mask_secrets(md_text)
    matches = list(FINDING_HEADING_RE.finditer(clean_text))
    findings: List[Dict[str, str]] = []
    for idx, m in enumerate(matches):
        title = m.group(2).strip()
        start = m.end()
        end = matches[idx + 1].start() if idx + 1 < len(matches) else len(clean_text)
        body = clean_text[start:end].strip()
        sev_match = SEVERITY_RE.search(title) or SEVERITY_RE.search(body)
        severity = sev_match.group(1).upper() if sev_match else "MEDIUM"
        findings.append(
            {
                "id": f"F{idx + 1:02d}",
                "title": title,
                "severity": severity,
                "summary": body[:800].strip() if body else title,
            }
        )
    if not findings and clean_text.strip():
        findings.append(
            {
                "id": "F01",
                "title": "PentAGI Security Finding",
                "severity": "MEDIUM",
                "summary": clean_text.strip()[:800],
            }
        )
    return findings


def generate_s1_intake_from_report(
    report_md: str,
    workspace_dir: Path,
    suffix: str,
    tier: str = "T2",
    agent_model: str = "PentAGI-Bridge",
) -> Dict[str, str]:
    """Generates S1 Intake documents (01_requirements, 05_progress, 08_tasks) from PentAGI report."""
    workspace_dir.mkdir(parents=True, exist_ok=True)
    req_path = workspace_dir / f"01_requirements_{suffix}.md"
    prog_path = workspace_dir / f"05_progress_{suffix}.md"
    tasks_path = workspace_dir / f"08_tasks_{suffix}.md"

    # Rule #17: Never overwrite existing workspace requirement/task files
    if req_path.exists() or tasks_path.exists():
        raise FileExistsError(
            f"Rule 17: Không ghi đè file workspace đã tồn tại ({req_path.name} / {tasks_path.name})."
        )

    findings = parse_findings_from_markdown(report_md)
    req_lines = [
        f"# 01 — Requirements: PentAGI Security Intake (`{suffix}`)",
        "",
        f"tier: {tier}",
        f"suffix: {suffix}",
        "source: pentagi_bridge",
        "",
        "## 1. Mục tiêu & Quy trình Bắt buộc (Rule #9 & Rule #13)",
        "- Tiếp nhận báo cáo kiểm thử xâm nhập tự động từ PentAGI (đã mask secret theo Rule #20).",
        "- **CẤM sửa code trực tiếp từ báo cáo PentAGI!** Bắt buộc đi qua:",
        "  1. `S2 Investigate`: Tái hiện lỗi (Red test) và xác minh `file:line` thật (loại bỏ False Positive).",
        "  2. `S3 Solution`: Lập `09_tasks_solution_" + f"{suffix}.md` + qua `DG1`/`DG2` (`laya_gate.py`).",
        "  3. `Wait APPROVE`: Dừng chờ lệnh `APPROVE` của User trước khi thực thi `S4`.",
        "",
        "## 2. Danh sách Findings cần xác minh",
    ]
    task_lines = [
        f"# 08 — Tasks: PentAGI Security Remediation (`{suffix}`)",
        "",
    ]

    for f in findings:
        fid = f["id"]
        sev = f["severity"]
        title = f["title"]
        summary = f["summary"].replace("\n", " ")
        req_lines.append(f"- **[{fid}] [{sev}] {title}**: {summary}")
        task_lines.extend(
            [
                f"- [ ] **{fid} ({sev}) — {title}**",
                f"  - **Mô tả & Context:** Xác minh finding `{fid}` từ PentAGI ({summary}). Định vị chính xác `file:line` trong source code.",
                "  - **Logs / Test (Gate G2):** Viết integration/unit test tái hiện lỗ hổng (Red) trước khi đề xuất bản vá.",
                f"  - **Definition of Done (DoD):** Lập phương án vá trong `09_tasks_solution_{suffix}.md`, đạt `DG1/DG2 PASS`, chờ `APPROVE`, sửa code đạt Green test và pass `DG3`.",
            ]
        )

    req_path.write_text("\n".join(req_lines) + "\n", encoding="utf-8")
    tasks_path.write_text("\n".join(task_lines) + "\n", encoding="utf-8")

    log_entry = (
        f"- [{now_iso_vn()}] [Agent:{agent_model}] "
        f"START S1 Intake từ báo cáo PentAGI ({len(findings)} findings) vào suffix {suffix}\n"
    )
    if prog_path.exists():
        existing = prog_path.read_text(encoding="utf-8")
        sep = "" if existing.endswith("\n") else "\n"
        prog_path.write_text(existing + sep + log_entry, encoding="utf-8")
    else:
        header = f"# 05 — Progress Log: `{suffix}` (Append ONLY)\n\n"
        prog_path.write_text(header + log_entry, encoding="utf-8")

    return {
        "requirements": str(req_path),
        "progress": str(prog_path),
        "tasks": str(tasks_path),
        "findings_count": str(len(findings)),
    }


def trigger_pentagi_flow(
    api_url: str,
    api_token: str,
    target_url: str,
    prompt: str,
    allowed_prefixes: Optional[Sequence[str]] = None,
    timeout_sec: int = 30,
    dry_run: bool = False,
) -> Dict[str, Any]:
    """Validates target_url against whitelist and triggers a scoped PentAGI flow."""
    if not validate_target_url(target_url, allowed_prefixes):
        raise ValueError(
            f"Target URL '{target_url}' bị chặn bởi cổng an toàn: không nằm trong allowed_pentest_targets."
        )
    masked_prompt = mask_secrets(prompt)
    payload = {
        "target": target_url,
        "prompt": masked_prompt,
    }
    if dry_run:
        return {
            "status": "DRY_RUN_VALIDATED",
            "target": target_url,
            "api_url": api_url,
            "payload": payload,
        }

    endpoint = api_url.rstrip("/") + "/api/v1/flows"
    req_data = json.dumps(payload).encode("utf-8")
    headers = {
        "Content-Type": "application/json",
        "Accept": "application/json",
    }
    if api_token:
        headers["Authorization"] = f"Bearer {api_token}"

    req = urllib.request.Request(endpoint, data=req_data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout_sec) as resp:
            raw_body = resp.read().decode("utf-8", errors="replace")
            try:
                parsed_body = json.loads(raw_body)
            except json.JSONDecodeError:
                parsed_body = {"raw": mask_secrets(raw_body)}
            return {
                "status": "TRIGGERED",
                "http_status": resp.status,
                "target": target_url,
                "response": parsed_body,
            }
    except urllib.error.URLError as exc:
        return {
            "status": "ERROR",
            "target": target_url,
            "error": mask_secrets(str(exc)),
        }


def resolve_workspace_dir(ws_arg: str) -> Path:
    p = Path(ws_arg)
    if p.is_absolute():
        return p
    return WS_ROOT / ws_arg


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="PentAGI Bridge for /agent Workflow (S1 Intake & S5 DAST)")
    sub = parser.add_subparsers(dest="subcommand", required=True)

    p_intake = sub.add_parser("intake", help="Convert PentAGI Markdown report into S1 Intake workspace docs")
    p_intake.add_argument("--report", required=True, help="Path to PentAGI Markdown report file")
    p_intake.add_argument("--workspace", required=True, help="Workspace name or absolute directory path")
    p_intake.add_argument("--suffix", required=True, help="Task suffix (<suffix>)")
    p_intake.add_argument("--tier", default="T2", choices=["T1", "T2", "T3"], help="Task tier (default: T2)")

    p_trig = sub.add_parser("trigger", help="Validate target against whitelist and trigger scoped PentAGI flow")
    p_trig.add_argument("--target", required=True, help="Target URL to scan")
    p_trig.add_argument("--project-root", default=".", help="Project root containing .agent-project.json")
    p_trig.add_argument(
        "--api-url",
        default=os.environ.get("PENTAGI_API_URL", "https://localhost:8443"),
        help="PentAGI base URL",
    )
    p_trig.add_argument(
        "--prompt",
        default="Scoped DAST check on modified endpoints",
        help="Scoped pentest instruction for PentAGI",
    )
    p_trig.add_argument("--dry-run", action="store_true", help="Validate target and print payload without HTTP call")

    args = parser.parse_args(argv)

    if args.subcommand == "intake":
        report_path = Path(args.report)
        if not report_path.is_file():
            print(json.dumps({"error": f"Không tìm thấy file báo cáo: {report_path}"}, ensure_ascii=False))
            return 1
        ws_dir = resolve_workspace_dir(args.workspace)
        try:
            res = generate_s1_intake_from_report(
                report_md=report_path.read_text(encoding="utf-8"),
                workspace_dir=ws_dir,
                suffix=args.suffix,
                tier=args.tier,
            )
            print(json.dumps({"status": "OK", **res}, ensure_ascii=False, indent=2))
            return 0
        except Exception as exc:
            print(json.dumps({"status": "ERROR", "error": str(exc)}, ensure_ascii=False))
            return 1

    if args.subcommand == "trigger":
        allowed = load_allowed_targets_from_manifest(Path(args.project_root))
        token = os.environ.get("PENTAGI_API_TOKEN", "")
        try:
            res = trigger_pentagi_flow(
                api_url=args.api_url,
                api_token=token,
                target_url=args.target,
                prompt=args.prompt,
                allowed_prefixes=allowed,
                dry_run=args.dry_run,
            )
            print(json.dumps(res, ensure_ascii=False, indent=2))
            return 0 if res.get("status") in ("TRIGGERED", "DRY_RUN_VALIDATED") else 1
        except ValueError as exc:
            print(json.dumps({"status": "BLOCKED", "error": str(exc)}, ensure_ascii=False))
            return 2

    return 1


if __name__ == "__main__":
    sys.exit(main())
