# 09 — Solution `anti_fake_evidence`

## Root cause (REF)
- `tooling/verify_governance.py:66` + `hooks/agy/agy_guard.py:144,569`: audit_report chỉ bị kiểm **từ khoá** ⇒ viết đủ chữ là qua.
- `tooling/mutation_gate.py:161`: mutant chỉ cần target_content xuất hiện 1 lần, **không** ràng vào dòng vừa đổi ⇒ chọn mutant né code mới.
- Không có liên kết claim ↔ lần chạy lệnh thật; output cũ/giả vẫn được tính (AF-5, stub harness2 AF-07).

## Phương án duy nhất
1. `tooling/evidence_run.py --workspace <ws> --kind unit|integration|runtime -- <cmd>`: chạy thật, ghi `06_evidence/EV-<ts>-<hex4>.json` {cmd, cwd, kind, exit, stdout_tail, src_fingerprint}. `kind=runtime` mà lệnh là test runner → từ chối (exit 3). `src_fingerprint` = sha256 nội dung file source (theo đuôi file, bỏ `memory/`, test vẫn tính) ⇒ commit không làm cũ bằng chứng, sửa code thì có.
2. H4c pre-write (`06_validation|11_report|audit_report`): chỉ lint phần **text mới ghi**. Dòng có claim (`PASS|OK|KILLED|xanh|chạy thật|runtime|n/n`) phải trích `EV-…`; mỗi EV phải: tồn tại; có `run_command` gọi `evidence_run.py` chứa đúng cmd trong transcript; fingerprint == hiện tại; claim PASS ⇒ exit 0; mọi số trong `n/n` có trong stdout; claim "chạy thật/runtime" ⇒ kind runtime|integration. Dòng dưới heading `Chưa kiểm|Giới hạn|Còn mở|Not verified` được miễn.
3. `mutation_gate.py --base <ref>`: mutant phải chạm dòng `+` của `git diff <ref> -- file`; mỗi file source đã đổi (trừ test) phải có ≥1 mutant; summary ghi `base`.
4. (Bổ sung S4) `mutation_gate.py:300` tính mọi exit≠0 là KILLED ⇒ mutant làm test crash/không chạy test nào cũng "bị bắt". Fix: output khớp `Ran 0 tests|NO TESTS RAN|no tests ran|collected 0 items|ImportError|ModuleNotFoundError|SyntaxError` → trạng thái `CRASHED` (không tính KILLED, gate FAIL).
5. (S5 runtime 10:35) Khớp nguyên văn cmd với transcript sai khi lệnh dùng biến shell/nháy ⇒ false deny. Đổi: EV hợp lệ nếu phiên có `run_command` chứa `evidence_run.py` và `--kind <kind của EV>`, và **không** có `write_to_file/replace_file_content` nhắm vào `06_evidence/` (chống tự viết biên lai).

## Code demo
```python
CLAIM_RE = re.compile(r"\b(PASS(?:ED)?|OK|KILLED|xanh|chạy thật|runtime)\b|\b\d+\s*/\s*\d+\b", re.I)
EV_RE = re.compile(r"\bEV-\d{8}T\d{6}-[0-9a-f]{4}\b")
def claim_issues(text, ws_dir, st, fingerprint):
    for line in claim_lines(text):          # bỏ dòng thuộc mục miễn
        evs = EV_RE.findall(line)
        if not evs: yield f"claim không có EV: {line[:80]}"; continue
        for ev in map(load_ev(ws_dir), evs): yield from check_ev(ev, line, st, fingerprint)
```

## S5 → S3 (10:40)
6. `current_fingerprint` dùng Cwd của run_command cuối phiên ⇒ sai repo thì báo "bằng chứng cũ" giả. Fix: khi H4c không truyền fingerprint, tính `source_fingerprint(rec["cwd"])` cho từng biên lai (cache theo cwd).

## Mở rộng theo yêu cầu User (11:11 — Tối ưu Quy trình & Tiết kiệm Token A–D)
7. **(A) Zero-Boilerplate Skill Loading (`GEMINI.md`):** Bỏ yêu cầu gọi `view_file` đọc lại các `SKILL.md` quy trình ở từng chặng S1–S6 vì luật đã tích hợp trong `GEMINI.md` và `agy_guard.py`.
8. **(B) Gộp `DG1 + DG2` trong 1 lần gọi (`tooling/laya_gate.py`, `hooks/agy/agy_guard.py`):** Cho phép `--stage dg12` hoặc `--stage dg1` có kèm claim `D-*`/`kind=demo` tự ghi cả summary `dg1` và `dg2` trong 1 batch HTTP duy nhất; `handle_stop` chỉ bắt buộc `dg1 == PASS` (và chặn nếu `dg2` tồn tại nhưng khác `PASS`).
9. **(C) Tắt H6 auto-checks lặp & chống bơm lặp prompt (`hooks/agy/agy_guard.py`, `.agent-project.json`):** Thêm cờ `h6_auto_checks` (mặc định `false` trong `.agent-project.json` để tránh chạy test ngầm lặp lại và tránh race với `mutation_gate.py`); chỉ bơm `FINAL_AUDIT_MSG` đầy đủ ở lần sửa đầu tiên trong lượt user; ở `handle_stop` chỉ chặn `n == 0` khi `miss` khác rỗng.
10. **(D) Gộp báo cáo Tier T2 vào `audit_report_<suffix>.md` (`tooling/verify_governance.py`, `hooks/agy/agy_guard.py`, `GEMINI.md`):** Với Tier T2, `audit_report_<suffix>.md` gộp luôn kết quả validation (biên lai `EV-...`) + report + 6 mục phản biện; không bắt tạo rời `06_validation_*` và `11_report_*`.

