# 09 — Tasks Solution: Clean-Room Hardening + Harness² Wiring (`rea_cleanroom_gap`)

status: PROPOSED — chờ User APPROVE (Propose-Only, chưa sửa code)
nguồn phân tích: `13_analysis_rea_cleanroom_gap.md`

## 1. Root Cause (S-RC)
- G-01: `APPROVE` regex không neo, chứa từ trần `duyệt`, thiếu `BẮT ĐẦU` — REF `hooks/agy/agy_guard.py:47-50`.
- G-02: `contaminated_clean_room_reads` chỉ xét `view_file|grep_search|find_by_name` — REF `hooks/agy/agy_guard.py:300-319`.
- G-03: `CLEAN_ROOM_SPEC_RE` chỉ khớp tiêu đề — REF `hooks/agy/agy_guard.py:54-56`.
- G-05: `harness2_engine.py` không được tham chiếu bởi GEMINI.md/OPERATOR_MAP.md/skills/hooks (grep exit=1).

## 2. Phương án duy nhất (S-FIX)
1. Thêm `CLEAN_ROOM_START` neo nguyên dòng; khi `mode: clean-room` chỉ mở S4 nếu User gửi đúng `BẮT ĐẦU`/`APPROVE`. Task thường: neo từ khoá ở đầu dòng.
2. `contaminated_clean_room_reads` xét thêm `run_command.CommandLine`, `read_url_content.Url`, `invoke_subagent.Prompt`.
3. Spec format `R<n>: <hành vi> — REF: path:start-end`; H4 + `verify_governance.py` từ chối rule thiếu REF.
4. **G-05 (cập nhật sau đọc source 09:37):** `harness2_engine.py run-recursion` bỏ proposer/scorer one-liner viết cứng (`:1020-1047`); thêm `--proposer-cmd` / `--rollout-cmd` (JSON list token) bắt buộc, thiếu → exit 2 với thông báo rõ. Chưa nối vào skill cho tới khi có CLI agent thật (tránh nối dây cho stub).
5. Test Red→Green trong `hooks/agy/tests/test_agy_guard.py` + `mutation_gate.py` cho gate mới.

## 3. Impact (S-IMPACT)
- Chỉ đổi `hooks/agy/agy_guard.py`, `tooling/verify_governance.py`, 1 skill, `OPERATOR_MAP.md`, test. Không hardcode dự án (ADR-07).
- Rủi ro: task thường cần User gõ từ khoá ở đầu dòng → giảm false-positive, có thể tăng 1 lượt hỏi lại.

## 4. Pattern (S-PATTERN)
Giữ pattern hiện có: regex hằng ở đầu file + hàm thuần nhận `st` trả `List[str]` (giống `contaminated_clean_room_reads`).

## 5. Code Demo (DG2)
```python
CLEAN_ROOM_START = re.compile(r"^\s*(BẮT\s*ĐẦU|APPROVE)\s*[.!]?\s*$", re.I | re.M)
READ_TOOLS = {"view_file": "AbsolutePath", "grep_search": "SearchPath", "find_by_name": "SearchDirectory",
              "run_command": "CommandLine", "read_url_content": "Url", "invoke_subagent": "Subagents"}
SPEC_RULE_RE = re.compile(r"^\s*-?\s*R(\d+):\s*(.+?)\s+—\s+REF:\s*\S+:\d+(?:-\d+)?\s*$", re.M)

def contaminated_clean_room_reads(st, forbidden):
    hits = []
    for _, n, a in calls(st):
        key = READ_TOOLS.get(n)
        if not key:
            continue
        blob = str(a.get(key) or "").replace("\\", "/")
        for fb in forbidden:
            fb_norm = fb.replace("\\", "/").rstrip("/")
            if fb_norm and fb_norm in blob and fb not in hits:
                hits.append(fb)
    return hits
```
Edge: path rỗng → bỏ qua; `forbidden` rỗng → `[]`; `BẮT ĐẦU` nằm giữa câu → không khớp.

## 6. Decision Gate
DG1/DG2: xem `05_progress_rea_cleanroom_gap.md`.

## 7. G-08 — Biến lesson thành Tripwire phòng ngừa (bổ sung 2026-10-21T09:45)
**Root cause:** `13_analysis_rea_cleanroom_gap.md` mục 7 (REF `tooling/build_lessons_index.py:93-101`, `hooks/agy/agy_guard.py:70-74`).

**Phương án duy nhất:**
1. Mỗi lesson thêm 3 dòng máy đọc được: `- **ID:** LS-<YYYYMMDD>-<slug>` (ổn định, không theo vị trí), `- **Trigger:** <regex trên prompt User / tool action>`, `- **Check:** <1 câu tiền điều kiện phải làm TRƯỚC>`.
2. `build_lessons_index.py` sinh thêm `lessons/tripwires.json` = `[{id, trigger, check}]` (nhỏ, không bị cắt).
3. H1 (PreInvocation): match prompt User với `trigger` → tiêm nguyên văn `check` của các lesson khớp (tối đa 5) vào context **trước khi làm**. Không còn "đọc INDEX" chung chung.
4. Với tripwire có `- **Require:** <regex>` (bằng chứng hành động trong tool-call args): H4 pre-write chặn ghi `13_analysis_*`/`09_tasks_solution_*`/`audit_report_*` nếu prompt User trong phiên khớp `Trigger` mà chưa có tool call nào sau đó khớp `Require`.
   - **G-04 (vị trí thực thi):** dispatcher không có pre-hook cho `invoke_subagent` (`agy_guard.py:818-828`) ⇒ kiểm `invoke_subagent` Prompt chứa path cấm hoặc dòng ≥40 ký tự của file cấm tại H7 Done-gate cho task clean-room; `contaminated_clean_room_reads` (H4b) xét thêm `run_command.CommandLine`, `read_url_content.Url`.
5. Backfill: lesson cũ chưa có Trigger/Check → script liệt kê để bổ sung dần, ưu tiên các lesson tái phạm.
6. **Sửa đổi append-only (bổ sung S4 09:55):** `lessons.md` cấm sửa dòng cũ ⇒ trường tripwire lặp lại trong cùng lesson thì **dòng sau thắng** (amendment). Áp dụng siết `Require` của `LS-20261021-prior-workspace-lookup` sau khi replay transcript thật cho thấy `grep -v memory/workspaces` làm thoả giả.

## 8. S5 → S3 Re-Plan (2026-10-21T09:57)
- **AF-1 [P0][VERIFIED]:** `tripwire_violations` tính cả tool ghi (`write_to_file`/`replace_file_content`) làm bằng chứng ⇒ lần ghi `13_analysis` bị chặn tự thoả `Require` khi thử lại (repro: before=['LS-20261021-prior-workspace-lookup'], after=[]). **Fix:** chỉ tính tool đọc/lệnh (`TRIPWIRE_EVIDENCE_TOOLS` = view_file, grep_search, find_by_name, list_dir, run_command, read_url_content, search_web); test hồi quy.
- **AF-2 [P1][VERIFIED]:** `leaked_subagent_prompts` chỉ đọc được nội dung file cấm khi path tuyệt đối. **Fix:** path tương đối thử resolve theo các `Cwd` của `run_command` trong transcript; test hồi quy.
- **AF-3 [P1][VERIFIED] (G-06):** `skills/reverse-engineer-anything` (symlink → `~/.gemini/config/skills/`, tạo 09:56:58 bởi `rea-agents setup`, ngoài phiên này) làm `validate_agent_surface` báo dead-weight → `test_01_f3` fail, baseline mutation FAIL. **Fix:** đăng ký trigger trong `OPERATOR_MAP.md` (mục 1), giới hạn đúng phạm vi theo chính description của REA: chỉ dùng cho binary/APK/Electron/firmware/không có source; repo có source thì dùng `view_file`/`grep`.
- **AF-4 [P1][VERIFIED] (H7 stop-audit 10:02):** trace transcript thật: `LS-20261021-prior-workspace-lookup` được thoả ở step 102 bởi `run_command` chứa `/memory/workspaces/$WS/09_tasks_solution_rea_cleanroom_gap.md` — tài liệu của **chính task đang ghi**, biến shell chưa expand. **Fix:** `tripwire_violations(st, trips, exclude_ws=<ws đích>)`: duyệt từng match của `Require`, bỏ match chứa `/<ws đích>/` hoặc chứa `$` (không chứng minh được); H4t truyền tên workspace đích; test hồi quy + mutant.
- **AF-5 [P1][VERIFIED] (10:03):** sau AF-4, replay vẫn thoả ở step 244 bởi chuỗi giả `/x/memory/workspaces/w/13_analysis_s.md` trong script repro (không phải file thật). **Fix:** mở rộng match thành token (ranh giới khoảng trắng/quote/`;|<>()`/backslash); token là đường dẫn tuyệt đối thì bắt buộc `Path(token).exists()`; token khác (URL, `git clone`, `S=/...`) giữ nguyên. Test dùng file thật trong tmp.
- **Giới hạn đã biết (không sửa trong scope):** bằng chứng `Require` không gắn với đúng repo của Trigger (clone repo A thoả cho repo B trong cùng phiên); `run_command` có path cấm trong `grep -v` vẫn tính là đọc (thiên về an toàn).

**Code demo:**
```python
TRIP_RE = re.compile(r"^- \*\*(ID|Trigger|Check):\*\*\s*(.+)$", re.M)

def match_tripwires(prompt: str, trips: list[dict], limit: int = 5) -> list[dict]:
    hits = [t for t in trips if re.search(t["trigger"], prompt, re.I)]
    return hits[:limit]
```
Ví dụ: `ID: LS-20261020-readme-only` · `Trigger: github\.com/[\w.-]+/[\w.-]+` · `Check: Tra memory/workspaces/*<repo>* và mở source thật (clone) trước khi kết luận.`

