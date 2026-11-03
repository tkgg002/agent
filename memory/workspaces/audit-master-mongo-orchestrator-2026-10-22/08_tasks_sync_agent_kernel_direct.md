# 08 Tasks — Đồng bộ Trực tiếp Kernel `/agent` vào `ai-orchestrator` (`sync_agent_kernel_direct`)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `sync_agent_kernel_direct`
- **Tier:** `T2`

## Danh sách Sub-Tasks

### [x] `TASK-KERN-01`: Đấu nối thực thi trực tiếp `hooks/agy/agy_guard.py` và `tooling/build_lessons_index.py` vào đúng điểm gọi Runtime (`S0`, `S4`, `S5`, `S6`)
- **Mô tả & Context `file:line`:**
  - Tạo module đơn trách nhiệm `/Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/direct_kernel_exec.go` và đấu nối trực tiếp vào các điểm gọi (call-sites) trong luồng vận hành của `ai-orchestrator`:
    1. **Tại `S0 Boot` (`cmd/orchestrator/daemon_cmd.go:64, 259`) và `S5/S6 Self-Heal & Finalize` (`internal/webui/task_self_heal_loop.go:215`, `internal/webui/server.go:1976`):** Gọi `Gatekeeper.RunBuildLessonsIndex(ctx)` (`python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py`) để kiểm chứng `Global Hygiene Guard` và đồng bộ `INDEX.md`, `topics/*.md`, `tripwires.json` trước khi cập nhật `IndexedLessons` và FTS5 BM25.
    2. **Tại `Hook H5 Pre-Command` (`internal/webui/server.go:386, 2188`, `internal/webui/task_chat.go:748` -> `internal/agentgate/gate.go:105`):** Đấu nối `ValidateCommandSafety` gọi trực tiếp `python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-command` để chặn cả `DENY_CMD` và `ASK_CMD`.
    3. **Tại `S4 Muscle CodeGen` (`internal/webui/subtask_codegen_muscle.go:280`) và `Step 11 S4 Gate Scan` (`internal/webui/server.go:1543-1566`, `internal/orchestrator/task_runner.go:538` -> `internal/agentgate/gate.go:126-177`):**
       - Ở `subtask_codegen_muscle.go:280` (ngay trước khi `os.WriteFile(absTarget, []byte(genCode), 0o644)`), gọi `s.Gatekeeper.RunAgyGuardPreWriteFile(ctx, absTarget, genCode)`.
       - Ở `server.go:1543-1556`, mở rộng danh sách file quét từ chỉ `.go` sang toàn bộ file source thay đổi (`.go`, `_test.go`, `.ts`, `.tsx`, `.js`, `.py`, `.sql`, `.css`) và gọi `s.Gatekeeper.ScanProductionFiles(execDir, changedSrcFiles)` (bên trong gọi `RunAgyGuardPreWriteFile` chạy trực tiếp `agy_guard.py pre-write` để bắt `H3`, `H4b` `PROD_MOCK_POLLUTION` + `FAKE_REAL_TEST_RE`, và `H4t-Code` `code_tripwire_violations`).
- **Logs / Test:**
  - `go test -v ./internal/agentgate -run TestDirectAgyGuardAndBuildLessonsIndex`
- **Definition of Done (DoD):**
  - Không có hàm wrapper chết; 100% các hàm mới đều có caller thực tế tại `daemon_cmd.go`, `subtask_codegen_muscle.go`, `server.go`, `task_self_heal_loop.go`, `gate.go`; mọi file đã có đạt `NOW <= HEAD`.

### [x] `TASK-KERN-02`: Đồng bộ schema `tripwires.json` (G-08), `.agent-project.json` (ADR-07) và ngưỡng `TotalLessons` trong `server_test.go` (Khắc phục `F2`, `F3`, `F4`)
- **Mô tả & Context `file:line`:**
  - Cập nhật `/Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/kernel_bridge.go:21-28, 44-51, 248-305, 450-465` và `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go:405-408`:
    1. Cập nhật `TripwireEntry` và `MatchTripwires` / `CheckUnmetTripwiresH4t` (`kernel_bridge.go:21-28, 248-305`) hỗ trợ trực tiếp định dạng mảng JSON `[...]` của `/agent/memory/global/lessons/tripwires.json` (`id`, `trigger`, `check`, `require` regex string) và kiểm tra `code_tripwire_violations` (`H4t-Code`).
    2. Cập nhật `LoadProjectManifest` (`kernel_bridge.go:44-51, 450-465`) unmarshal đầy đủ mảng `components[]` của `.agent-project.json` (hỗ trợ `test` và `runtime_check` dạng `string` hoặc `[]string`).
    3. Cập nhật `internal/webui/server_test.go:406` kiểm tra `TotalLessons >= 40` khớp với 45 bài học Global trong `/agent/memory/global/lessons/INDEX.md:2`.
- **Logs / Test:**
  - `go test -v ./internal/agentgate/... ./internal/webui -run TestDashboardActiveAgentKernelAndMatchedLessons`
- **Definition of Done (DoD):**
  - `MatchTripwires` đọc đúng 14 tripwires từ `/agent/memory/global/lessons/tripwires.json`; `LoadProjectManifest` đọc đúng `checks`, `test`, `runtime_check` từ `/Users/trainguyen/Documents/work/ai-orchestrator/.agent-project.json`; `server_test.go` và `kernel_bridge.go` đạt `NOW <= HEAD`.

### [x] `TASK-KERN-03`: Vòng lặp Phản tỉnh & Tự chữa lành `S5 -> S3 -> S4 -> S5` (`AF-KERN-01 .. AF-KERN-04`)
- **Mô tả & Context `file:line`:**
  1. **`AF-KERN-01` (Tách module đơn trách nhiệm — Single-Responsibility Principle):** Tách phần giải mã schema (`tripwires.json`, `.agent-project.json`) và kiểm tra tripwire mã nguồn ra khỏi `internal/agentgate/direct_kernel_exec.go` (`359` dòng) thành file đơn trách nhiệm `internal/agentgate/live_schema_parser.go` (`~195` dòng), giữ `direct_kernel_exec.go` (`~185` dòng) thuần túy chỉ thực thi subprocess sang `/agent`.
  2. **`AF-KERN-02` (Đồng bộ `s.IndexedLessons` & `RunBootProtocolS0` đúng 100% cam kết trong `09_tasks_solution`):** Trong `internal/agentgate/kernel_bridge.go:177` (`RunBootProtocolS0`), gọi `_, _ = g.RunBuildLessonsIndex(ctx)` trước khi đọc `INDEX.md`; tại `internal/webui/server.go:1970` và `internal/webui/task_self_heal_loop.go:214`, cập nhật `s.IndexedLessons = CountTotalLessonsInAgentIndex(s.Gatekeeper.AgentCoreDir)` ngay sau khi `RunBuildLessonsIndex` chạy thành công.
  3. **`AF-KERN-03` (Xóa điểm mù ngưỡng lô `len(relFiles) <= 12` trong `ScanProductionFiles`):** Trong `internal/agentgate/gate.go:138-148`, gọi `EvaluateCodeTripwiresAndFakeRealTest(cleanRel, content, tripwires)` trên **100% số lượng file** bất kể `len(relFiles)` lớn hay nhỏ (đảm bảo khi `TaskRunner.implementAndReview` truyền `> 12` file vẫn bắt đủ `H4t-Code` và `FAKE_REAL_TEST_RE`).
  4. **`AF-KERN-04` (Kiểm chứng Không Mockup & Trace Runtime Giá trị Thật trên `:8090`):** Bổ sung bài test thực tế không dùng `httptest.NewServer` trong `internal/agentgate/direct_kernel_exec_test.go` kiểm chứng lô `16` file (`> 12`) và `RunBootProtocolS0`, đồng thời khởi động lại `./bin/orchestrator ui --addr :8090` và trace trực tiếp giá trị JSON runtime của `active_agent_kernel`.
- **Logs / Test:**
  - `go test -v -race ./internal/agentgate/...`
  - `go build -o bin/orchestrator ./cmd/orchestrator && ./bin/orchestrator daemon --once`
- **Definition of Done (DoD):**
  - 100% file hiện hữu đạt `NOW <= HEAD`; `direct_kernel_exec.go` và `live_schema_parser.go` tách biệt rõ trách nhiệm đơn; pass toàn bộ biên lai vật lý `EV-...` và `/laya_router` `DG3` Mode A + Mode B.
