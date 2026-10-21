# 09 — Tasks Solution: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate vào Hệ thống `/agent`

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2` (`mode: clean-room`)
- **Ngày lập:** `2026-10-20` (Revision `R1` — Đã cập nhật sau Audit phản biện Plan & Code Demo `F1–F6`)
- **Tài liệu bổ trợ:** [`10_gap_analysis_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/10_gap_analysis_clean_room_and_mutation_gate.md), [`12_implementation_plan_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/12_implementation_plan_clean_room_and_mutation_gate.md)
- **Trạng thái:** `PROPOSE-ONLY (Chờ User APPROVE trước khi thực thi S4)`

---

## 1. Phân tích Vấn đề Cốt lõi, Khoảng trống Kiến trúc & 6 Điểm Mù Phát hiện qua Audit Phản biện (`VERIFIED` kèm `file:line`)

User yêu cầu tích hợp 2 kỹ thuật tinh hoa từ quy trình **REA + Jev + Clean-Room + Haiku 3-Mutant Testing** vào hệ thống `/Users/trainguyen/Documents/work/agent`, đồng thời **dùng tư duy phản biện để audit lại plan & code demo** theo nguyên tắc *Simplicity First, Minimal Impact, tuân thủ Core Systems & Design Pattern, Zero Speculation*:
1. **Clean-Room Specification & Sub-Agent Isolation (Cách ly Phòng Sạch khi Rebuild/Refactor):**
   > *"Model thấy code gốc thì không bao giờ được viết code, và model viết code thì không bao giờ được thấy code gốc!"*
2. **3-Mutant Test Verification Gate (Kiểm thử Đột biến với 3 Bản Cố tình Làm hỏng — Chống "Test Phế"):**
   > *"Chạy các bài kiểm thử và kiểm tra thêm 3 phiên bản được cố tình tạo lỗi để xác nhận hệ thống phát hiện đúng lỗi... Nếu 3 bản hỏng mà vẫn pass test thì test của bạn phế, phải viết lại!"*

### 1.1. Ba Khoảng trống Kiến trúc Hiện tại (`VERIFIED`)
1. **Khoảng trống 1 (`VERIFIED` — `GEMINI.md:18-22, 95-96`, `hooks/agy/agy_guard.py:379-455` & `tooling/verify_governance.py:21-23`):**
   - `GEMINI.md:18-22, 95-96` đã tách vai trò `Brain` và `Muscle`, nhưng khi **Rebuild / Refactor tính năng cũ**, cả Brain và Muscle vẫn dùng chung ngữ cảnh đã đọc code cũ (Contaminated Context), dễ bị neo tư duy hoặc copy lại bug cũ.
   - `hooks/agy/agy_guard.py:379-455` và `tooling/verify_governance.py:21-23` chưa nhận diện `mode: clean-room` và `clean_room_forbidden_sources: <csv_paths>` trong `01_requirements_<suffix>.md` để: (a) bắt buộc `09_tasks_solution_<suffix>.md` có mục `## Clean-Room Approved Spec` (chỉ giữ rule có `REF:` đã qua `/laya_router` `DG1`), và (b) chặn cứng tại `H4b` nếu phiên hiện tại đã đọc file trong `clean_room_forbidden_sources` mà tự sửa source code thay vì uỷ quyền cho Sub-agent mới tinh (`invoke_subagent`).
2. **Khoảng trống 2 (`VERIFIED` — `skills/qa-agent/SKILL.md:29-55` & thiếu công cụ `tooling/mutation_gate.py`):**
   - Ở tầng quyết định kiến trúc (`tooling/laya_gate.py:20-23`), hệ thống đã có cơ chế đối chứng âm tính (`>=2 control_false` ở Mode A và `control_corrupted_diff` ở Mode B).
   - Ở tầng **chạy bộ kiểm thử thực tế (`checks` / `test` tại S4/S5)**, hệ thống chưa có công cụ tự động tiêm **3 biến thể cố tình làm hỏng logic (`M1`, `M2`, `M3`)** vào code đích để kiểm chứng xem bộ test có thực sự bắt sống (`KILLED`) được cả 3 lỗi hay không (chống `L039 Tautological Validation`).
3. **Khoảng trống 3 (`VERIFIED` — `hooks/agy/agy_guard.py:560-574`):**
   - Trong `handle_stop` (`hooks/agy/agy_guard.py:560-574`), khối `if src:` chưa kiểm tra trạng thái biên bản kiểm thử đột biến (`06_mutation_summary_<suffix>_*.json`) và chưa kiểm tra tính tươi mới (`touched` sau lần sửa code cuối `last`).

### 1.2. Sáu Lỗi & Thiếu sót của Plan Ban đầu Phát hiện qua Audit Phản biện (`F1–F6` — Chi tiết tại [`10_gap_analysis_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/10_gap_analysis_clean_room_and_mutation_gate.md))
- **`F1` (`P0` — Lỗi `replace_file_content` tại `H4a` `hooks/agy/agy_guard.py:347, 379-416`):** Kiểm tra `re.search(r"clean-room\s+approved\s+spec", body)` trực tiếp trên `body` (`CodeContent + ReplacementContent`) sẽ chặn oan mọi lệnh `replace_file_content` cập nhật một phần nhỏ trong `09_tasks_solution_<suffix>.md`. **Khắc phục:** Ghép nội dung file hiện có trên đĩa với đoạn sửa (`existing_sol.replace(TargetContent, ReplacementContent, 1)`) rồi mới kiểm tra mục `Clean-Room Approved Spec`.
- **`F2` (`P0` — Cưỡng chế vật lý Luật Phòng Sạch tại `H4b` `hooks/agy/agy_guard.py:419-455`):** Luật *"Model thấy code gốc thì không bao giờ được viết code"* ở bản đầu mới dừng ở văn bản (`L194 Paper-Only Governance`). **Khắc phục:** Khai báo `clean_room_forbidden_sources: <csv_paths>` trong `01_requirements_<suffix>.md`; tại `H4b`, hàm `contaminated_clean_room_reads(st, forbidden)` quét `calls(st)` (`view_file`, `grep_search`, `find_by_name`) từ `transcript.jsonl` và `deny` ngay nếu phiên đã đọc code gốc cũ mà tự sửa source code, bắt buộc uỷ quyền qua `invoke_subagent`.
- **`F3` (`P0` — Lỗ hổng `SyntaxError` và `Timeout` gây False-Kill trong `tooling/mutation_gate.py`):** Bản demo đầu chỉ kiểm tra `returncode != 0`, nên nếu mutant cố tình viết sai cú pháp Python (`SyntaxError`) hoặc làm treo vô hạn (`timed_out == True`), test runner vẫn trả `returncode != 0` và bị tính nhầm là `KILLED`. **Khắc phục:** (1) Với file `.py`, kiểm tra `ast.parse(mutated_text)` trên bộ nhớ trước khi ghi xuống đĩa (nếu `SyntaxError` trả mã `3 INVALID_INPUT`); (2) Nếu `res["timed_out"] == True` thì phân loại `status = "TIMEOUT"` và đánh trượt cổng (`gate = "FAIL"`).
- **`F4` (`P1` — Tích hợp `.agent-project.json` và chuẩn hoá đường dẫn tương đối trong `tooling/mutation_gate.py`):** Bản demo đầu dùng `Path(m["file"]).resolve()` (lệ thuộc `Path.cwd()`) và bắt khai báo thủ công `test_cmd`/`cwd`. **Khắc phục:** Tái sử dụng pattern `resolve_target_file` (`tooling/laya_gate.py:97-105`) và `auto_detect_component` từ `.agent-project.json` (`hooks/agy/agy_guard.py:270-302`, tuân thủ `ADR-07`).
- **`F5` (`P1` — An toàn khôi phục file khi tiến trình nhận `SIGTERM` trong `tooling/mutation_gate.py`):** Khối `try...finally` trong Python không tự chạy khi nhận tín hiệu `SIGTERM` trừ khi đăng ký signal handler. **Khắc phục:** Đăng ký `signal.signal(signal.SIGTERM, _sigterm_to_exit)` chuyển `SIGTERM` thành `SystemExit(143)` quanh vùng ghi tạm file để `finally:` luôn khôi phục byte-for-byte và kiểm tra `SHA-256`.
- **`F6` (`P1` — Chống kết quả cũ `Stale Summary` & Giữ tương thích ngược tại `H7` `hooks/agy/agy_guard.py:560-574`):** Giữ nguyên 6 phần tử của `AUDIT_REQUIRED_PATTERNS` để tương thích 100% với 24 unit test hiện có (`L067`); tại `H7` (`if src:`), khi task có `mode: clean-room` (hoặc có file `06_mutation_summary_*.json`), bắt buộc cả `mutation_gate_status(...) == "PASS"` **VÀ** `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)` sau lần sửa code cuối (`L006/L007`).

---

## 2. Clean-Room Approved Spec (Source-Free Contract — Đã qua `/laya_router` DG1 & DG3)

> **Quy tắc Phòng Sạch (Clean-Room Rule):** Mọi quy tắc dưới đây đều được neo vào bằng chứng xác thực (`REF: path:line`) và đã được `/laya_router` (`DG1`, `DG2`, `DG3`) phê duyệt `PASS`. Khi khởi chạy Sub-agent Builder ở chặng `S4`, Builder chỉ nhận đặc tả này để xây dựng module mới.

- **R1**: (CR-01 — Giao thức Clean-Room & Cổng duyệt `BẮT ĐẦU` `G-01`) Khi `01_requirements_<suffix>.md` khai báo `mode: clean-room`, file `09_tasks_solution_<suffix>.md` bắt buộc có mục `Clean-Room Approved Spec` (kiểm tra trên nội dung file hoàn chỉnh sau khi áp dụng edit tại `H4a`), và chặng `S4` chỉ mở khóa khi User gửi riêng dòng lệnh `BẮT ĐẦU` hoặc `APPROVE` (`CLEAN_ROOM_START`, không khớp chữ `duyệt`/`BẮT ĐẦU` nằm giữa câu mô tả) — REF: `hooks/agy/agy_guard.py:48-60`
- **R2**: (CR-02 — Chặn đọc Code gốc qua mọi kênh `G-02`) Khi `01_requirements_<suffix>.md` khai báo `clean_room_forbidden_sources: <csv_paths>`, hàm `contaminated_clean_room_reads` theo dõi cả `view_file`, `view_file_outline`, `view_code_item`, `grep_search`, `find_by_name`, `run_command` (`CommandLine`, `Cwd`), và `read_url_content` (`Url`); nếu phiên hiện tại đã chạm vào bất kỳ nguồn cấm nào thì `H4b` chặn cứng việc trực tiếp sửa source code — REF: `hooks/agy/agy_guard.py:64-71`
- **R3**: (CR-03 — Kiểm duyệt từng Rule kèm `REF:` trong Spec `G-03`) Hàm `spec_rule_issues` trong `hooks/agy/agy_guard.py` (`H4a`) và `tooling/verify_governance.py` (`S3/S6`) bắt buộc mục `Clean-Room Approved Spec` có ít nhất 1 rule (`R<n>` hoặc `CR-<n>`), mọi rule và mọi dòng bullet trong mục này đều phải có trích dẫn `REF: <path>:<line>` và không chứa trạng thái bị bác bỏ (`REJECTED`/`NO_EVIDENCE`/`UNVERIFIED`/`BLOCK`) — REF: `hooks/agy/agy_guard.py:383-415`
- **R4**: (CR-04 — Chặn rò rỉ Code gốc vào Prompt Subagent `G-04`) Hàm `leaked_subagent_prompts` và `handle_pre_subagent` kiểm tra cả `invoke_subagent` (`Subagents[*].Prompt`) và `send_message` (`Message`); nếu chứa đường dẫn hoặc dòng code ($\ge 40$ ký tự) của file trong `clean_room_forbidden_sources` thì chặn ngay tại `PreToolUse` (`pre-subagent`) và tại `H7` (`handle_stop`) — REF: `hooks/agy/agy_guard.py:335-372`
- **R5**: (CR-05 — Công cụ `tooling/mutation_gate.py` & Điều kiện Thắng Kép 3-Mutant) Công cụ `tooling/mutation_gate.py` kiểm tra tối thiểu 3 biến thể cố tình làm hỏng (`len(mutants) >= 3`), tự động lấy `test`/`cwd` từ `.agent-project.json`, kiểm tra `ast.parse` chống `SyntaxError`, bọc `SIGTERM -> SystemExit(143)` và `try...finally` khôi phục 100% `SHA-256`, chỉ trả `gate = "PASS"` khi `baseline_pass == True` và `killed_count == total >= 3` — REF: `tooling/mutation_gate.py:157-313`
- **R6**: (CR-06 — Tích hợp Cưỡng chế tại `H6/H7`, `verify_governance.py` & Cài đặt `rea-agents` `G-06`) Cổng `H7` (`handle_stop`) và `verify_governance.py` bắt buộc `06_mutation_summary_<suffix>_*.json` đạt `PASS` sau lần sửa code cuối; đồng thời cài đặt `rea@6.7.0` (`rea-agents`) vào `PATH` và cấu hình MCP/skill `reverse-engineer-anything` cho `antigravity` & `claude_code` — REF: `hooks/agy/agy_guard.py:890-931`

---

## 3. ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT (Single Best Solution & Code Demo — Revision R2)

> Chi tiết vị trí dòng code (`file:line`) cho từng file trong 11 file mục tiêu được trình bày đầy đủ tại [`12_implementation_plan_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/12_implementation_plan_clean_room_and_mutation_gate.md).

### 3.1. Hạng mục 1 (`T8.1`): Cập nhật `GEMINI.md` (`L71-74`, `L95-99`) và 4 Core Skills (`brain-delegate`, `muscle-execute`, `qa-agent`, `workspace-docs`)
- **Tại `GEMINI.md` (tăng thêm đúng `+2` dòng, giữ tổng dung lượng `~16.2 KB < 18 KB`):**
  - **Rule #9 (`L71`, `L73-74`):** Bổ sung quy định `## Clean-Room Approved Spec` tại `S3 SOLUTION`, uỷ quyền Sub-agent phòng sạch (`clean_room_forbidden_sources`) tại `S4 EXECUTE`, và chạy `mutation_gate.py` tại `S4/S5`.
  - **Rule #13 (`L95-96`) & Rule #14 (`L99`):** Bổ sung **Clean-Room Rebuild Rule** và yêu cầu kiểm chứng 3 bản cố tình làm hỏng (`mutation_gate.py` — 3/3 `KILLED`) tại Gate `G3/G7`.

### 3.2. Hạng mục 2 (`T8.2`): Xây dựng Công cụ `tooling/mutation_gate.py` & Unit Tests `tooling/tests/test_mutation_gate.py`

#### Code Demo (Revision R1 — Đã vá `F3`, `F4`, `F5`) — `tooling/mutation_gate.py`:
```python
#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""mutation_gate.py — 3-Mutant Test Verification Gate (Chống 'Test Phế' / Tautological Tests).

Usage:
  python3 tooling/mutation_gate.py --workspace <ws> --suffix <suffix> --mutations <mutations.json>

Exit codes:
  0 : PASS (Baseline PASS + All >=3 Mutants KILLED + 100% SHA-256 Restored)
  2 : FAIL (Baseline FAIL or >=1 Mutant SURVIVED/TIMEOUT or Restore Mismatch)
  3 : INVALID_INPUT (<3 mutants, invalid target_content, or SyntaxError mutant)
"""
from __future__ import annotations

import argparse
import ast
import datetime as dt
import hashlib
import json
import signal
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WS_ROOT = AGENT_ROOT / "memory" / "workspaces"
MIN_MUTANTS = 3


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def resolve_target_file(raw_path: str, mutations_file: Path) -> Optional[Path]:
    p = Path(raw_path.strip())
    if p.is_absolute() and p.is_file():
        return p.resolve()
    for base in (Path.cwd(), AGENT_ROOT, mutations_file.resolve().parent):
        cand = (base / p).resolve()
        if cand.is_file():
            return cand
    return None


def auto_detect_component(fpath: Path) -> Tuple[Path, Optional[str]]:
    cur = fpath.resolve().parent
    while True:
        manifest = cur / ".agent-project.json"
        if manifest.is_file():
            try:
                cfg = json.loads(manifest.read_text(encoding="utf-8"))
                rel = str(fpath.resolve().relative_to(cur)).replace("\\", "/")
                for comp in cfg.get("components", []):
                    cpath = comp.get("path", "").strip("/")
                    if cpath and (rel == cpath or rel.startswith(cpath + "/")):
                        return (cur / cpath).resolve(), comp.get("test")
                if cfg.get("test"):
                    return cur, cfg.get("test")
            except Exception:
                pass
            return cur, None
        if cur.parent == cur:
            break
        cur = cur.parent
    return fpath.resolve().parent, None


def run_test_cmd(cmd: str, cwd: Path, timeout_sec: int = 60) -> Dict[str, Any]:
    try:
        r = subprocess.run(
            cmd,
            shell=True,
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=timeout_sec,
        )
        return {
            "returncode": r.returncode,
            "output_tail": (r.stdout + r.stderr)[-800:],
            "timed_out": False,
        }
    except subprocess.TimeoutExpired as e:
        return {
            "returncode": -1,
            "output_tail": f"TimeoutExpired after {timeout_sec}s: {e}",
            "timed_out": True,
        }


def execute_mutation_gate(
    ws_name: str,
    suffix: str,
    mutations_file: Path,
    workspaces_dir: Path = DEFAULT_WS_ROOT,
    timeout_sec: int = 60,
) -> Tuple[int, Dict[str, Any]]:
    try:
        raw = json.loads(mutations_file.read_text(encoding="utf-8"))
    except Exception as e:
        return 3, {"gate": "INVALID_INPUT", "error": f"Không đọc được file JSON mutations: {e}"}

    mutants: List[Dict[str, Any]] = raw if isinstance(raw, list) else raw.get("mutants", [])
    if len(mutants) < MIN_MUTANTS:
        return 3, {
            "gate": "INVALID_INPUT",
            "error": f"Bắt buộc tối thiểu {MIN_MUTANTS} biến thể cố tình làm hỏng (nhận được {len(mutants)}).",
        }

    prepared: List[Dict[str, Any]] = []
    for idx, m in enumerate(mutants, start=1):
        mid = str(m.get("id") or f"M{idx}")
        raw_file = str(m.get("file") or "")
        fpath = resolve_target_file(raw_file, mutations_file) if raw_file else None
        t_content = str(m.get("target_content") or "")
        m_content = str(m.get("mutant_content") or "")
        if not fpath or not t_content or not m_content:
            return 3, {"gate": "INVALID_INPUT", "error": f"Mutant {mid}: thiếu hoặc không tìm thấy file/target_content/mutant_content."}
        if t_content == m_content:
            return 3, {"gate": "INVALID_INPUT", "error": f"Mutant {mid}: mutant_content trùng hệt target_content."}

        comp_cwd, comp_test = auto_detect_component(fpath)
        t_cmd = str(m.get("test_cmd") or comp_test or "").strip()
        cwd = Path(m["cwd"]).resolve() if m.get("cwd") else comp_cwd
        if not t_cmd:
            return 3, {"gate": "INVALID_INPUT", "error": f"Mutant {mid}: không có test_cmd và không tìm thấy 'test' trong .agent-project.json."}

        text = fpath.read_text(encoding="utf-8")
        if text.count(t_content) != 1:
            return 3, {
                "gate": "INVALID_INPUT",
                "error": f"Mutant {mid}: target_content phải xuất hiện đúng 1 lần trong {fpath.name} (hiện tại: {text.count(t_content)}).",
            }

        mutated_text = text.replace(t_content, m_content, 1)
        if fpath.suffix == ".py":
            try:
                ast.parse(mutated_text, filename=str(fpath))
            except SyntaxError as se:
                return 3, {
                    "gate": "INVALID_INPUT",
                    "error": f"Mutant {mid}: biến thể làm hỏng gây lỗi cú pháp Python (SyntaxError: {se.msg} tại line {se.lineno}). Mutant phải là lỗi logic hợp lệ!",
                }

        prepared.append({
            "id": mid,
            "description": str(m.get("description") or ""),
            "fpath": fpath,
            "cwd": cwd,
            "test_cmd": t_cmd,
            "target_content": t_content,
            "mutant_content": m_content,
        })

    # Bước 1: Chạy Baseline trên các lệnh test duy nhất
    baseline_results: Dict[str, Dict[str, Any]] = {}
    for item in prepared:
        key = f"{item['cwd']}::{item['test_cmd']}"
        if key not in baseline_results:
            res = run_test_cmd(item["test_cmd"], item["cwd"], timeout_sec)
            baseline_results[key] = res
            if res["returncode"] != 0 or res["timed_out"]:
                return 2, {
                    "gate": "FAIL",
                    "reason": f"Baseline test FAILED trước khi tiêm lỗi: {item['test_cmd']}",
                    "baseline": res,
                }

    # Bước 2: Chạy từng Mutant, bảo vệ SIGTERM và luôn khôi phục nguyên trạng trong khối finally
    results: List[Dict[str, Any]] = []
    survived_ids: List[str] = []
    timeout_ids: List[str] = []
    all_restored = True

    prev_sigterm = signal.getsignal(signal.SIGTERM)

    def _sigterm_to_exit(signum: int, frame: Any) -> None:
        raise SystemExit(128 + signum)

    signal.signal(signal.SIGTERM, _sigterm_to_exit)
    try:
        for item in prepared:
            mid = item["id"]
            fpath = item["fpath"]
            orig_bytes = fpath.read_bytes()
            orig_sha = sha256_bytes(orig_bytes)
            mutated_text = orig_bytes.decode("utf-8").replace(item["target_content"], item["mutant_content"], 1)

            try:
                fpath.write_bytes(mutated_text.encode("utf-8"))
                res = run_test_cmd(item["test_cmd"], item["cwd"], timeout_sec)
            finally:
                fpath.write_bytes(orig_bytes)
                restored_sha = sha256_bytes(fpath.read_bytes())

            sha_ok = restored_sha == orig_sha
            if not sha_ok:
                all_restored = False

            if res["timed_out"]:
                status = "TIMEOUT"
                timeout_ids.append(mid)
            elif res["returncode"] != 0:
                status = "KILLED"
            else:
                status = "SURVIVED"
                survived_ids.append(mid)

            results.append({
                "id": mid,
                "description": item["description"],
                "file": str(fpath),
                "status": status,
                "returncode": res["returncode"],
                "timed_out": res["timed_out"],
                "sha256_before": orig_sha,
                "sha256_after": restored_sha,
                "sha256_restored": sha_ok,
                "output_tail": res["output_tail"],
            })
    finally:
        signal.signal(signal.SIGTERM, prev_sigterm)

    killed_count = sum(1 for r in results if r["status"] == "KILLED")
    gate = "PASS" if (killed_count == len(prepared) and not survived_ids and not timeout_ids and all_restored) else "FAIL"
    ts = dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).strftime("%Y%m%dT%H%M%S")
    ws_dir = workspaces_dir / ws_name
    ws_dir.mkdir(parents=True, exist_ok=True)
    summary_path = ws_dir / f"06_mutation_summary_{suffix}_{ts}.json"
    summary = {
        "workspace": ws_name,
        "suffix": suffix,
        "ts": ts,
        "total_mutants": len(prepared),
        "killed_count": killed_count,
        "survived_count": len(survived_ids),
        "survived_ids": survived_ids,
        "timeout_count": len(timeout_ids),
        "timeout_ids": timeout_ids,
        "all_restored_sha256": all_restored,
        "gate": gate,
        "mutants": results,
        "summary_file": str(summary_path),
    }
    summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return (0 if gate == "PASS" else 2), summary
```

### 3.3. Hạng mục 3 (`T8.3`): Nâng cấp `hooks/agy/agy_guard.py` (`H4a`, `H4b`, `H6`, `H7`) & `tooling/verify_governance.py`
1. **Trong `hooks/agy/agy_guard.py` (đã vá `F1`, `F2`, `F6`):**
   - Thêm `clean_room_config(ws_dir, suffix) -> Tuple[bool, List[str]]`, `contaminated_clean_room_reads(st, forbidden) -> List[str]`, và `mutation_gate_status(ws, ws_root, suffix) -> Optional[str]` tại `L238`.
   - **Tại `H4a` (`L401-415`):** Ghép nội dung file `09_tasks_solution_<suffix>.md` hiện có trên đĩa với `TargetContent -> ReplacementContent` trước khi kiểm tra `CLEAN_ROOM_SPEC_RE`.
   - **Tại `H4b` (`L436-454`):** Nếu task khai báo `mode: clean-room` và `clean_room_forbidden_sources`, kiểm tra `contaminated_clean_room_reads(st, forbidden)`; nếu phiên hiện tại đã đọc file trong danh sách cấm thì `deny` lệnh sửa code trực tiếp và yêu cầu dùng `invoke_subagent`.
   - **Tại `H7` (`L560-575`):** Giữ nguyên 6 phần tử `AUDIT_REQUIRED_PATTERNS`; trong khối `if src:`, nếu task có `mode: clean-room` hoặc có file `06_mutation_summary_*.json`, bắt buộc `mutation_gate_status == "PASS"` và `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)`.
2. **Trong `tooling/verify_governance.py` (`L21-23`, `L193-212`):**
   - Bổ sung `CLEAN_ROOM_MODE_RE` và `CLEAN_ROOM_SPEC_RE`.
   - Khi `01_requirements_<suffix>.md` khai báo `mode: clean-room`: kiểm tra `09_tasks_solution_<suffix>.md` có mục `Clean-Room Approved Spec` (tại `S3` & `S6`), và kiểm tra `06_mutation_summary_<suffix>_*.json` có `"gate": "PASS"` và `"killed_count" >= 3` (tại `S6`).

---

## 4. Phân tích Tác động & Kế hoạch Rollback (Blast Radius & Rollback Plan)

| Thành phần | File thay đổi / tạo mới | Tác động (`Blast Radius`) | Kế hoạch Rollback |
| :--- | :--- | :--- | :--- |
| **Hiến pháp & Core Skills** | `GEMINI.md`, `skills/{brain-delegate,muscle-execute,qa-agent,workspace-docs}/SKILL.md` | Bổ sung quy định `mode: clean-room` (`clean_room_forbidden_sources`) và `mutation_gate.py` (3-Mutant Verification). Hoàn toàn tương thích ngược với các task `T0–T3` thông thường. | Khôi phục từ git commit restore-point trước khi sửa (`git checkout -- ...`). |
| **Công cụ Mutation Gate** | `tooling/mutation_gate.py` *(mới)*, `tooling/tests/test_mutation_gate.py` *(mới)* | File độc lập mới; kiểm tra `ast.parse`, bảo vệ `SIGTERM`, luôn khôi phục byte-for-byte file gốc trong `try...finally` kèm kiểm tra `SHA-256`. | Xóa file mới hoặc revert commit. |
| **Lifecycle Hooks** | `hooks/agy/agy_guard.py`, `hooks/agy/tests/test_agy_guard.py` | Giữ nguyên `AUDIT_REQUIRED_PATTERNS`; chỉ kích hoạt kiểm tra bắt buộc khi task khai báo `mode: clean-room` (hoặc khi có file summary mutation trong workspace), không làm gãy 24 unit test hiện có. | Khôi phục từ git commit restore-point. |
| **Process Linter** | `tooling/verify_governance.py`, `tooling/tests/test_verify_governance.py` | Kiểm tra thêm điều kiện `mode: clean-room` ở `S3` và `S6`, giữ nguyên 100% hành vi với 14 unit test hiện có. | Khôi phục từ git commit restore-point. |

---

## 5. Bảng Kết quả Kiểm định Decision Gate (`DG1`, `DG2` & `DG3` Audit Plan)

| Cổng | Summary File | Trusted Rounds | Các Claims đã kiểm chứng | Kết quả Gate |
| :--- | :--- | :---: | :--- | :---: |
| **DG1 (Solution)** | `06_laya_summary_dg1_20261020T101742.json` | `r3` | `S-RC` (`ACCEPT` 0.9073), `S-FIX` (`ACCEPT` 0.6730), `S-IMPACT` (`ACCEPT` 0.9651), `S-PATTERN` (`ACCEPT` 0.8858) + `2 control_false` (`CF1`, `CF2`) + `1 control_true` (`CT1`) | **`PASS`** |
| **DG2 (Code Demo)** | `06_laya_summary_dg2_20261020T101746.json` | `r2, r3` | `D-BEHAV` (`ACCEPT` 0.8591), `D-EDGE` (`ACCEPT` 0.7791), `D-API` (`ACCEPT` 0.9503) + `2 control_false` (`CF1`, `CF2`) + `1 control_true` (`CT1`) | **`PASS`** |
| **DG3 (Adversarial Audit Plan & Code R1)** | `06_laya_summary_dg3_20261020T114944.json` | `r3` | `A-F1` (`FLAG -> RESOLVED` kèm `REF: hooks/agy/agy_guard.py:344-348`), `A-F2` (`ACCEPT` 0.7714), `A-F4` (`ACCEPT` 0.7889), `A-F6` (`ACCEPT` 0.5272), `A-SCOPE` (`ACCEPT` 0.6742) + `2 control_false` (`CF1`, `CF2`) + `1 control_true` (`CT1`) | **`PASS`** |


