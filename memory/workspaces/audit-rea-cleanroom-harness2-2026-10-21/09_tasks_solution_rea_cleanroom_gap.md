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
4. Với tripwire có `enforce: hook` (vd. URL repo tham chiếu → phải có `git clone`/`view_file` source trước khi ghi `13_analysis`), H4/H7 chặn cứng nếu chưa thấy hành động `check` trong transcript.
5. Backfill: lesson cũ chưa có Trigger/Check → script liệt kê để bổ sung dần, ưu tiên các lesson tái phạm.

**Code demo:**
```python
TRIP_RE = re.compile(r"^- \*\*(ID|Trigger|Check):\*\*\s*(.+)$", re.M)

def match_tripwires(prompt: str, trips: list[dict], limit: int = 5) -> list[dict]:
    hits = [t for t in trips if re.search(t["trigger"], prompt, re.I)]
    return hits[:limit]
```
Ví dụ: `ID: LS-20261020-readme-only` · `Trigger: github\.com/[\w.-]+/[\w.-]+` · `Check: Tra memory/workspaces/*<repo>* và mở source thật (clone) trước khi kết luận.`

