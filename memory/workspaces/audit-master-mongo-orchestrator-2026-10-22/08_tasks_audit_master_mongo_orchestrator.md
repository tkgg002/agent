# 08 Tasks: Kế hoạch Audit & Khắc phục Task "Tích hợp Master MongoDB Destination" trên `ai-orchestrator`

## Danh mục Task
- [x] **TASK-A01 (S2 Investigate `ai-orchestrator` Runtime & Task State):**
  - **Context:** `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:490-514,1133-1187`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:697-704,1115-1139,1966-2055`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:781-786`.
  - **Mô tả:** Kiểm tra database `queue.db`, các thư mục `.orchestrator/tasks/*`, `.orchestrator/worktrees/*` và log thực thi của task `"tích hợp Master MongoDB Destination"` trên `ai-orchestrator`. Đã phát hiện 5 lỗ hổng `AF-ORCH-01 .. AF-ORCH-05`.
  - **Logs/Test:** Truy vấn read-only trạng thái task (`id=32` vs `id=35`), sub-tasks, worktrees, artifacts và đối chiếu code thực thi trong `ai-orchestrator`.
  - **Definition of Done:** Liệt kê đầy đủ bằng chứng `file:line` và trạng thái runtime thật của task trên `ai-orchestrator`.

- [x] **TASK-A02 (S2 Investigate `data-hub` Repositories & Workspace Artifacts):**
  - **Context:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/`, `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/`, `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/`.
  - **Mô tả:** Đối chiếu 10 Sub-Tasks (`TASK-M01 .. TASK-M10`) của `"Tích hợp Master MongoDB Destination"` với code thực tế trong 3 repo `data-hub`. Đã phát hiện 7 lỗ hổng `AF-DH-01 .. AF-DH-07`.
  - **Logs/Test:** Kiểm tra `git status`, `git diff` và nội dung từng file code trên 3 repo `data-hub` + `feat-master-mongo-destination-2026-10-20`.
  - **Definition of Done:** Bảng đối chiếu chi tiết 100% Sub-Tasks `TASK-M01..M10` kèm `file:line` thật.

- [x] **TASK-A03 (S3 Solution & Code Demo + DG1/DG2/DG3 Audit Report):**
  - **Context:** `09_tasks_solution_audit_master_mongo_orchestrator.md`, `audit_report_audit_master_mongo_orchestrator.md`.
  - **Mô tả:** Thiết kế 1 phương án khắc phục tối ưu nhất kèm Code Demo cho cả 12 lỗ hổng (`AF-ORCH-01..05` & `AF-DH-01..07`), chạy kiểm định `/laya_router` (`DG1/DG2` và `DG3`), xuất `audit_report_audit_master_mongo_orchestrator.md` và trình User `APPROVE`.
  - **Logs/Test:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` & `--stage dg3` đạt `PASS`.
  - **Definition of Done:** Hồ sơ `09_tasks_solution_*.md` và `audit_report_*.md` hoàn chỉnh, có biên lai Laya Gate `PASS`, giữ nguyên kỷ luật Propose-Only chờ User `APPROVE`.

- [x] **TASK-A04 (S4 — Sau APPROVE: Chỉ sửa trên `ai-orchestrator` để khắc phục `AF-ORCH-01 .. AF-ORCH-06`, Cô lập Worktree `.orchestrator/worktrees/TASK-11/<repo>` & Dọn Rác Test Trong `09_tasks_solution_master_mongo.md` — TUYỆT ĐỐI CẤM chạm vào `/Users/trainguyen/Documents/work/data-hub/*`):**
  - **Context:** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/seed_data.go:278-285`, `internal/webui/task_inspector.go:697-704,1115-1139,1966-2055`, `internal/webui/server.go:490-514,1133-1187`, `internal/webui/task_chat.go:677-920`, `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-11/chat_history.json:16-118`, `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md:320-360`.
  - **Mô tả:**
    1. **Bất khả xâm phạm Repo `data-hub`:** Giữ nguyên working tree `/Users/trainguyen/Documents/work/data-hub/{cdc-cms-service,cdc-cms-web,centralized-data-service}` sạch 100% (`git status --short` rỗng). Không bao giờ gọi `git stash apply` hay sửa file trực tiếp trên `/Users/trainguyen/Documents/work/data-hub/*`.
    2. (`AF-ORCH-01`) Hợp nhất 2 task trùng tên `"tích hợp Master MongoDB Destination"` (`TASK-11` `id=35` và `t-ch-h-p-Master-MongoDB-Destination` `id=32`), chặn `SyncRealWorkspaceState` ghi đè task của User (`cur.Source != "agent-workspace"`).
    3. (`AF-ORCH-02` & `AF-ORCH-05`) Cô lập Multi-Repo Worktree tại `.orchestrator/worktrees/<safeTask>/<repoName>` và bổ sung `collectRepoUnifiedDiffWithUntracked` để `diff.patch` lấy từ worktree cô lập của task chứa đủ 100% cả file tracked (`M`) lẫn file mới untracked (`??`).
    4. (`AF-ORCH-03`) Chặn unit test ghi đè workspace `/agent` thật và xóa 3 khối rác `TASK-DUAL-12` khỏi `09_tasks_solution_master_mongo.md:320-360`.
    5. (`AF-ORCH-04`) Sửa `VerifySubTaskCodeOnDisk` không báo `VERIFIED` giả khi `relPaths` của Sub-Task chưa có file khớp.
    6. (`AF-ORCH-06`) Xây dựng bộ **`DetectAndIngestAuditReport` (`classifyChatActionMode` & `ingestAuditReportAndRemediation`)** trong `internal/webui/task_chat.go`: bất kể dropdown đang để `discuss`, tự động nhận diện khi User dán Báo cáo Audit (`AF-ORCH-*`, `AF-DH-*`, `TASK-A04..A07`, `audit_report_*.md`, `09_tasks_solution_*.md`), tự động bóc tách danh sách Sub-Tasks khắc phục (`TASK-A04..TASK-A07`) nạp vào `subtasks.json` + `08_tasks.md`, đồng bộ `09_tasks_solution.md` & `audit_report.md`, mở lại vòng lặp tự chữa lành (`S5 -> S3 Re-Plan -> S4 Execute -> S5 QA`) bên trong `ai-orchestrator`, và trả lời trực diện theo đúng nội dung câu hỏi/báo cáo Audit của User thay vì nhai lại văn mẫu tĩnh.
  - **Logs/Test:** `go test -v -race ./internal/webui/...` trên `/Users/trainguyen/Documents/work/ai-orchestrator` qua `evidence_run.py` + kiểm thử runtime `POST /api/tasks/chat` trên `127.0.0.1:8090` + kiểm tra `git status --short` trên 3 repo `data-hub` bằng 0 dòng.
  - **Definition of Done:** Working tree 3 repo `data-hub` của anh sạch 100%; chọn `TASK-11` trên `ai-orchestrator` (`http://127.0.0.1:8090/`) hiển thị đầy đủ Sub-Tasks (`TASK-M01..M10` + `TASK-A04..A07`), `diff.patch`, `09_tasks_solution.md` và phản hồi xử lý đầy đủ cho Báo cáo Audit anh đã dán vào khung chat.

- [x] **TASK-A05 (S5 -> S3 -> S4 -> S5 Autonomous Self-Healing Loop: Khắc phục 5 lỗi tự phản biện `SA-01 .. SA-05` trên `ai-orchestrator`):**
  - **Context:** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1164-1325,2085-2158`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1205-1233`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:967-1166`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go:451-498,633-686`.
  - **Mô tả:**
    1. (`SA-01`) Xóa bỏ hoàn toàn nhánh `taskPlanContent` (`09_tasks_solution.md`) khỏi `VerifySubTaskCodeOnDisk` để chặn đứng việc xác minh giấy tờ (`Paper-Only Verification`).
    2. (`SA-02`) Xóa bỏ đường dẫn hardcode `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/diff.patch` khỏi `task_inspector.go` và `server.go`; thay bằng `readTaskOrSiblingDiffPatch` chỉ tra cứu động bên trong `filepath.Join(s.Root, ".orchestrator", "tasks")`.
    3. (`SA-03`) Đối soát trung thực trạng thái `subtasks.json` của `TASK-11` theo đúng bằng chứng vật lý qua `VerifySubTaskCodeOnDisk`: `10 done` (`TASK-M01..M07, M09, A04, A07`) và `4 pending` (`TASK-M08, M10, A05, A06`), giữ `TASK-11` ở trạng thái `NEEDS_YOU`.
    4. (`SA-04`) Làm sạch bộ parser `ingestAuditReportAndRemediation` để không lặp tiền tố `1. **TASK-A04...**` và gắn đúng danh sách file mục tiêu `TargetFiles` / `ContextFileLine` cho `TASK-A04..A07`.
    5. (`SA-05`) Cập nhật `server_test.go` kiểm chứng chặt cả chiều âm (có `09_tasks_solution.md` nhưng chưa có `diff.patch`/code trên đĩa thì `VerifySubTaskCodeOnDisk` trả về `false` và `approve_and_run` chặn ở `NEEDS_YOU`) lẫn chiều dương (khi có `diff.patch`/code thật trong `t.TempDir()` thì PASS).
  - **Logs/Test:** Chạy `go vet ./...` (`EV-20261022T121634-6a24`), `go test -v -race ./internal/webui/...` (`EV-20261022T121635-f246`), `go run ./cmd/orchestrator daemon --once` (`EV-20261022T121700-c6b1`) và kiểm thử live HTTP API `:8090` (`EV-20261022T121803-27b9`) qua `evidence_run.py`; gọi `/laya_router` (`dg3` + `review`) và xuất `audit_report_audit_master_mongo_orchestrator.md`.
  - **Definition of Done:** 0% Paper-Verification, 0% Hardcoded Host Path, 100% trung thực trạng thái Sub-Tasks trên đĩa, pass toàn bộ test + runtime + `DG3` + `verify_governance.py`.
