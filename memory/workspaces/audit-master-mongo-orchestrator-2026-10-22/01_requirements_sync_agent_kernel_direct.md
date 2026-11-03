# 01 Requirements — Đồng bộ Trực tiếp Kernel `/agent` (`agy_guard.py`, `build_lessons_index.py`, `tripwires.json`, `.agent-project.json`) vào `ai-orchestrator`
tier: T2
status: IN_PROGRESS
suffix: sync_agent_kernel_direct
workspace: audit-master-mongo-orchestrator-2026-10-22

## 1. Bối cảnh & Yêu cầu của User
- **User Prompt:**
  1. *"hiện tại ai-orchestrator đang dùng kenel /agent của anh. nhưng đang copy ra để dùng hay dùng trực tiếp. vì a đang có vài update vào agent ."*
  2. *"còn mớ build_lessons_index.py, agy_guard.py của agent thì sao, có chạy tụi nó ở ai-orchestrator ko"*
  3. *"giải pháp cho anh"*
- **Mục tiêu cốt lõi:**
  - Chuyển `ai-orchestrator` từ cơ chế "copy/port rời rạc logic sang Go" sang **gọi thực thi trực tiếp (Direct Execution Bridge)** các công cụ lõi của `/Users/trainguyen/Documents/work/agent` (`hooks/agy/agy_guard.py` và `tooling/build_lessons_index.py`), đồng thời đồng bộ 100% schema dữ liệu (`tripwires.json`, `INDEX.md` 45 bài học Global `G6`, `.agent-project.json`) để mọi cập nhật hiện tại và tương lai của User vào `/agent` tự động có hiệu lực ngay lập tức trên `ai-orchestrator`.

## 2. Ràng buộc Kỹ thuật từ Bài học Lịch sử (`lessons/INDEX.md` — 45 Global Lessons)
- **`L001` (Propose-Only Discipline & Explicit Approval Gate):** Giữ chế độ 100% Read-Only đối với source code của `ai-orchestrator` ở bước `S1 -> S3`, trình bày Summary trực diện + Giải pháp 1-1 theo từng phát hiện `F1..F4` + Code Demo trên chat và dừng chờ lệnh `APPROVE` tường minh của User.
- **`L002` (Pure Agnostic Core Engine Design — `LS-GLOBAL-002-agnostic-core`):** Xóa bỏ các hằng số tên hàm đặc thù dự án con (`lookupMasterTargetsFn`, `runEventsFn`) đang bị hardcode trong `internal/agentgate/gate.go:25`; đọc động `.agent-project.json` (`components[]`) đúng chuẩn Agnostic Core.
- **`L004` (Single-Responsibility Module Split & Line-Budget Guard):** Tách các hàm bridge trực tiếp sang module đơn trách nhiệm (`internal/agentgate/direct_kernel_exec.go`) và giữ 100% file hiện có (`gate.go`, `kernel_bridge.go`, `server_test.go`) đạt `NOW <= HEAD`.
- **`L018` (Direct-Answer Summary First, Macro Topology & 1-to-1 Finding Mapping):** Mở đầu bằng Summary trực diện, trình bày Macro Runtime Topology (`ai-orchestrator <-> /agent Kernel`), và ánh xạ giải pháp 1-1 theo từng sơ hở `F1 -> F4`.

## 3. Tiêu chí Nghiệm thu (Definition of Done — Gates G1–G8)
1. **G1 (Direct `agy_guard.py` Execution):** `Gatekeeper` trong `ai-orchestrator` gọi trực tiếp `python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py` (`pre-write`, `pre-command`, `pre-subagent`) để thừa hưởng tự động 100% các rule trong `agy_guard.py` (`H2b` Global Lesson Hygiene, `H3` Secret Guard, `H4t-Code` Code Tripwires, `H4b` Agnostic Anti-Mock & `FAKE_REAL_TEST_RE`, `H5` Command Guard).
2. **G2 (Direct `build_lessons_index.py` Execution & Lesson Hygiene):** `Gatekeeper` cung cấp hàm gọi trực tiếp `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` (kèm kiểm tra `pre-write` của `agy_guard.py` trước khi append bài học vào `lessons.md`).
3. **G3 (Schema Alignment cho `tripwires.json`, `.agent-project.json` & `INDEX.md`):**
   - `MatchTripwires` và `CheckUnmetTripwiresH4t` trong `kernel_bridge.go` đọc trực tiếp mảng JSON `[...]` của `tripwires.json` mới (`id`, `trigger`, `check`, `require`) đồng thời tương thích ngược với schema cũ.
   - `LoadProjectManifest` trong `kernel_bridge.go` đọc đúng mảng `components[]` của `.agent-project.json`.
   - Cập nhật ngưỡng kiểm tra số bài học trong `internal/webui/server_test.go:406` khớp với bộ 45 bài học Global (`>= 40` bài học từ `INDEX.md`) thay vì số cũ `>= 268`.
4. **G4 (Line Budget & Real Verification):** 100% file đã có đạt `NOW <= HEAD`, `go vet ./...` và `go test ./internal/agentgate/... ./internal/webui/...` đạt `PASS` với biên lai vật lý `EV-...` (sau khi User `APPROVE`).
