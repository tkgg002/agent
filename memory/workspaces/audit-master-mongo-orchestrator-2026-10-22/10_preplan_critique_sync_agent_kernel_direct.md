# 10 Pre-Plan Adversarial Critique (S2.5 / DG0) — `sync_agent_kernel_direct`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `sync_agent_kernel_direct`
- **Stage:** `S2.5 Pre-Plan Adversarial Critique (DG0)`

## 1. Phản biện Giả định & Kiểm chứng Sự thật Vật lý (`C-REQ`, `C-REALITY`, `C-CONTRA`)

1. **Phản biện 1 (`C-REALITY-01` — Không gọi bừa `agy_guard.py` cho các event phụ thuộc `transcript.jsonl` của Antigravity IDE khi không có transcript):**
   - **Sự thật vật lý (`hooks/agy/agy_guard.py:180-189, 858-1130`):**
     - Trong `agy_guard.py`, các nhánh kiểm tra tĩnh trên nội dung file/command (`H2` Overwrite/Append-Only, `H2b` `validate_lesson_append_hygiene`, `H3` `SECRET`, `H4a` `spec_rule_issues`, `H4b` `PROD_MOCK_POLLUTION` + `FAKE_REAL_TEST_RE` + `H4t-Code` `code_tripwire_violations`, `H5` `DENY_CMD`/`ASK_CMD`) **hoạt động độc lập 100% trên payload JSON `{ "toolCall": { "args": { ... } } }`** mà KHÔNG cần `transcriptPath` của Antigravity IDE!
     - Ngược lại, phần cuối của `handle_pre_write` (dòng `1038-1116` kiểm tra `active_ws(st)` từ `transcriptPath`) chỉ kích hoạt sau khi đã chạy qua toàn bộ `PROD_MOCK_POLLUTION`, `FAKE_REAL_TEST_RE` và `code_tripwire_violations` (`1011-1036`). Nếu gọi `python3 agy_guard.py pre-write` với file nguồn `.go` mà `transcriptPath` rỗng, dòng `1040` (`if not ws: return {"decision": "deny", "reason": "Rule 4/16: chưa có workspace..."}`) sẽ luôn trả về `"Rule 4/16: chưa có workspace"`!
   - **Kết luận thiết kế (`DG0 Reality Check`):**
     - Đối với `pre-command` (`H5`) và `pre-write` trên `memory/` (`H2`, `H2b` `lessons.md`, `H3`, `H4a` `09_tasks_solution`): gọi trực tiếp `python3 agy_guard.py pre-command` / `pre-write` chạy hoàn hảo 100%.
     - Đối với kiểm tra file source code (`ScanProductionFiles` — `H3`, `H4b`, `H4t-Code`): để vừa chạy trực tiếp logic Python của `agy_guard.py` vừa không bị dính nhánh `if not ws:` của `transcriptPath` Antigravity IDE, ta gọi trực tiếp `agy_guard.py` (hoặc truyền synthetic step có `workspaces/<ws>/` + `05_progress` khi cần, hoặc phân biệt rõ `reason` từ `Rule 4/16: chưa có workspace` vs các lỗi thật `Rule 12/14`, `Rule 20`, `Code Tripwire Gate (H4t-Code)`, `Rule 5 (Global Lesson Hygiene Guard)`).

2. **Phản biện 2 (`C-CONTRA-01` — Phát hiện lỗi lệch schema nghiêm trọng trong `kernel_bridge.go` đối với `tripwires.json` và `.agent-project.json`):**
   - **Sự thật vật lý (`internal/agentgate/kernel_bridge.go:21-28, 44-51, 248-278, 450-465`):**
     - `tripwires.json` do `build_lessons_index.py:155-166` sinh ra là mảng JSON cấp cao nhất `[{"id": "...", "trigger": "...", "check": "...", "require": "..."}]`, nhưng `kernel_bridge.go:258-263` lại `json.Unmarshal` vào `struct { Tripwires []TripwireEntry }` với trường `Pattern` và `Require []string` -> làm `MatchTripwires` luôn trả về `0` kết quả!
     - `.agent-project.json` (`/Users/trainguyen/Documents/work/ai-orchestrator/.agent-project.json:1-14`) đặt `checks`, `test`, `runtime_check` trong `components[]`, nhưng `LoadProjectManifest` (`kernel_bridge.go:44-51`) lại unmarshal ở root -> làm rỗng `Checks`, `Test`, `RuntimeCheck`!
   - **Kết luận thiết kế:** Bắt buộc sửa cả 2 parser trong `kernel_bridge.go` để đọc chuẩn xác 100% file thật của `/agent` đồng thời vẫn giữ tương thích ngược với unit test cũ.

3. **Phản biện 3 (`C-REQ-01` — Ngưỡng bài học trong `internal/webui/server_test.go:406`):**
   - **Sự thật vật lý:** Sau khi User cập nhật `build_lessons_index.py` với chuẩn `G6 Global Lesson Hygiene` (`<= 12` dòng/bài, 100% Global), `memory/global/lessons/INDEX.md:2` hiện có đúng `45` bài học (`L001`–`L045`). Nhưng `internal/webui/server_test.go:406` đang kiểm tra `dashA.ActiveAgentKernel.TotalLessons >= 268`, gây fail test. Cần cập nhật ngưỡng kiểm tra về `>= 40` khớp với bộ 45 bài học Global mới.
