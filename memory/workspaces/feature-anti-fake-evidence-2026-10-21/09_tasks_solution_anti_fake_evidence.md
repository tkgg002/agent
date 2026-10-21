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
