# 09. Giải Pháp Kỹ Thuật & Code Demo — Audit & Khắc Phục Toàn Diện Task "Tích Hợp Master MongoDB Destination" Trên `ai-orchestrator` & `data-hub`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `audit_master_mongo_orchestrator`
- **Tier:** `T2`
- **Ngày lập:** 2026-10-22
- **Trạng thái:** `APPROVED & EXECUTED ON AI-ORCHESTRATOR` (`DG1/DG2` & `DG3` `PASS` — Đã hoàn trả nguyên trạng 100% 3 repo `data-hub` theo `L001`)

---

## 1. Bối Cảnh & Tổng Hợp 12 Phát Hiện Audit Thực Tế (`VERIFIED` `file:line`)

Qua kiểm tra đối chiếu trực tiếp giữa hồ sơ đặc tả `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md`, cơ sở dữ liệu `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/queue.db`, các tạo tác trong `.orchestrator/tasks/` và mã nguồn thực tế trên 3 repository `data-hub` (`cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`), em xác nhận có **12 lỗ hổng thực tế** chia làm 2 nhóm:

### Nhóm A — 5 Lỗ Hổng Thực Thi & Quản Trị Task Trên `ai-orchestrator` (`AF-ORCH-01 .. AF-ORCH-05`)

1. **`AF-ORCH-01 (P0 — Split-Brain Duplicate Task: `TASK-11` vs `tích hợp Master MongoDB Destination`)`:**
   - **Bằng chứng (`queue.db` & `.orchestrator/tasks/`):** Trong bảng `task_queue` của `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/queue.db`, có **2 bản ghi riêng biệt** cùng mang tiêu đề `"tích hợp Master MongoDB Destination"`:
     - Bản ghi `id=32` (`external_id = "tích hợp Master MongoDB Destination"`, thư mục `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/`) chứa `diff.patch` (99KB, 2289 dòng) và `audit_report.md` (7.6KB).
     - Bản ghi `id=35` (`external_id = "TASK-11"`, `title = "tích hợp Master MongoDB Destination"`, thư mục `.orchestrator/tasks/TASK-11/`) là task anh đang bấm chọn trên UI lúc `10:23:58–10:27:35` (`/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-11/terminal.log:33-35`).
   - **Root Cause (`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/seed_data.go:278-285` & `task_inspector.go:1966-2055`):** Hàm `SyncRealWorkspaceState` đọc `08_tasks_*.md` của workspace orchestrator (vốn có `Task 11`) và đè trạng thái lên `TASK-11`; đồng thời hàm `restoreSmokeCompletedUserTasks` (`task_inspector.go:1966-2055`) khi khôi phục chỉ kiểm tra bản ghi đầu tiên hoặc bỏ qua đồng bộ chéo khi tồn tại 2 task cùng tiêu đề. Kết quả là `TASK-11/diff.patch` chỉ có **35 bytes** và `TASK-11/audit_report.md` chỉ có **531 bytes** cũ từ lúc `TASK-11` quét nhầm `ai-orchestrator` (`118 Go files in ai-orchestrator`, `TASK-11/terminal.log:6,12`).

2. **`AF-ORCH-02 (P0 — Fake Worktree Isolation & Passive Working-Tree Diff)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:490-514, 1133-1187` & `task_inspector.go:793`):**
     - `SubTaskDTO.WorktreePath` (`task_inspector.go:793`) khai báo đường dẫn `.orchestrator/worktrees/<safeParent>/<repoName>`, nhưng `ensureTaskIsolation` (`server.go:490-514`) chỉ chạy `git worktree add` nếu `repoDir == s.Root` (`ai-orchestrator`). Do đó `.orchestrator/worktrees/t-ch-h-p-Master-MongoDB-Destination` lại là git worktree của chính `ai-orchestrator` (`HEAD is now at 2d9a0a2`, `terminal.log:1-2`), không phải của 3 repo `data-hub`.
     - Tại `server.go:1133-1187` (`case "implement"`), orchestrator không tạo worktree cho từng repo trong `affectedRepos` mà chỉ đọc thụ động `git status --short` và `git diff` trực tiếp trên working tree chính `/Users/trainguyen/Documents/work/data-hub/{cdc-cms-service,cdc-cms-web,centralized-data-service}`.

3. **`AF-ORCH-03 (P1 — Test Pollution Làm Bẩn File Giải Pháp Thật Trong `/agent/memory/workspaces/`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:781-786` & `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md:320-360`):**
     - Khi `handleTaskPlanningChat` xử lý `action_mode == "revise_solution"`, các dòng `task_chat.go:781-786` gọi `s.FindMatchingAgentWorkspace(*qt)` (`task_inspector.go:697-704`) và **ghi nối trực tiếp vào file thật** `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md`!
     - Khi chạy `go test` trong `internal/webui/server_test.go` với task giả `TASK-DUAL-12` có từ khóa `"mongo"` và `"destination"`, test đã ghi đè 3 khối rác `TASK-DUAL-12` (`2026-10-22 10:01:49`, `10:03:21`, `10:04:26`) vào cuối file thật `09_tasks_solution_master_mongo.md:320-360`.

4. **`AF-ORCH-04 (P1 — False-Positive Sub-Task Verification Đánh Dấu `TASK-M10` Là `VERIFIED` Dù Chưa Có File Test)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1115-1139` & `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/audit_report.md:102`):**
     - Trong `audit_report.md:102`, `TASK-M10` được báo cáo: `VERIFIED (centralized-data-service:test/ ((command exited with 0, no output)))`.
     - Nguyên nhân: tại `task_inspector.go:1130-1139`, khi `git status --porcelain -- "test/"` không có thay đổi nào (trả về chuỗi mặc định `"(command exited with 0, no output)"` từ `server.go:381-383`), hàm `VerifySubTaskCodeOnDisk` rơi xuống nhánh fallback chạy `git status --porcelain` trên toàn bộ repo `centralized-data-service`; thấy repo có file khác đang sửa nên lấy dòng đầu tiên ghép với `firstRel = "test/"` và kết luận `TASK-M10` đã `done`!

5. **`AF-ORCH-05 (P1 — Bỏ Sót 100% File Mới Untracked `??` Khỏi `diff.patch`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1137`):**
     - Lệnh `git diff` tại `server.go:1137` chỉ xuất diff của các file đã tracked (`M`), bỏ qua hoàn toàn các file untracked (`??`).
     - Toàn bộ 7 file mới được tạo cho tính năng Master MongoDB (`mongo_type_converter.go`, `transmuter_mongo.go`, `transmuter_mongo_test.go`, `recon_dest_mongo.go`, `recon_dest_mongo_test.go`, `connection_manager_mongo_test.go`, `master_registry_handler_connections_test.go`) đều ở trạng thái `??`, khiến `diff.patch` khuyết 100% code của 7 file này.

6. **`AF-ORCH-06 (P0 — Khung Co-Architect Chat Chỉ Nhai Lại Văn Mẫu Tĩnh & Không Làm Bất Cứ Gì Khi User Quăng Báo Cáo Audit Vào `TASK-11`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-11/chat_history.json:16-118`, `TASK-11/terminal.log:36-42`, và `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:677-920`):**
     - Tại `TASK-11/chat_history.json:85-118` (lúc `10:46:47`), anh đã dán nguyên văn kết quả Audit từ model AI khác (gồm 12 phát hiện `AF-ORCH-01..05`, `AF-DH-01..07` và 4 task khắc phục `TASK-A04..TASK-A07`) vào khung Co-Architect Chat của `TASK-11`.
     - Trước đó lúc `10:41:42` và `10:43:39` (`chat_history.json:17-83`), anh hỏi `"mày là ai, model nào"` và `"mày cảm nhận như thế nào khi mày làm việc ở đây, so với làm trên app antigravity..."`.
     - **Nguyên nhân gốc rễ (`task_chat.go:677-920`):**
       1. Dropdown trên giao diện mặc định gửi `action_mode: "discuss"`, nên khối `if actionMode == ""` (`task_chat.go:681-692`) bị bỏ qua hoàn toàn — hệ thống không hề phân tích nội dung tin nhắn để tự động nhận diện ý định (`Auto-Intent Classification`).
       2. Khi gặp một Báo cáo Audit / Danh sách lỗi (`AF-ORCH-*`, `AF-DH-*`, `TASK-A04..A07`, `audit_report_*.md`, `09_tasks_solution_*.md`), `handleTaskPlanningChat` không có hàm `detectAndIngestAuditReport`: nó rơi thẳng vào nhánh `default: // "discuss"` (`task_chat.go:874-919`), giữ nguyên `TASK-11` ở trạng thái `DONE (10/10 Sub-Tasks)`, không bóc tách `TASK-A04..TASK-A07` vào `subtasks.json` / `08_tasks.md`, không đồng bộ giải pháp từ workspace Audit sang `TASK-11/09_tasks_solution.md`, không mở lại vòng lặp tự chữa lành (`S5 -> S3 Re-Plan -> S4 Execute -> S5 QA`), mà chỉ nhai lại câu mẫu tĩnh: `"1. Trạng thái & Tiến độ thực tế hiện tại: Stage: DONE | Sub-Tasks hoàn tất: 10/10 ... Tất cả Sub-Tasks đã hoàn tất (100%)"`.

---

### Nhóm B — 7 Lỗ Hổng Kiến Trúc & Logic Runtime Trong Code `data-hub` (`AF-DH-01 .. AF-DH-07`)

1. **`AF-DH-01 (P0 — Crash `E11000 DuplicateKeyError` & Hủy Lô Gây Mất Dữ Liệu Khi Gặp Bản Ghi Out-of-Order Trên MongoDB Master)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go:126-148, 221-228` đối chiếu `09_tasks_solution_master_mongo.md:220-241`):**
     - Tại `transmuter_mongo.go:129-147`, hàm `BuildMongoOCCBulkModels` đưa điều kiện OCC `$or: [{"_source_ts": {"$exists": false}}, {"_source_ts": {"$lte": incomingTs}}]` **vào trong `SetFilter(filter)`** của `UpdateOneModel` cùng với `SetUpsert(true)`.
     - Khi một sự kiện cũ hơn (`incomingTs < existing._source_ts`) đi vào `BulkWrite` (do CDC out-of-order hoặc do 2 worker chạy song song sau bước `Find` ở dòng 196), filter `{conflictTarget: keyVal, $or: [...]}` không khớp document hiện có (`0 match`). Vì `Upsert: true`, MongoDB cố gắng **INSERT một document mới** với cùng `_gpay_id` / `conflictTarget`, va vào Unique Index `ux_<table>_gpay_id` và nổ lỗi `E11000 duplicate key error`!
     - Nghiêm trọng hơn, tại `transmuter_mongo.go:221-227`:
       ```go
       bwRes, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
       if err != nil {
           if mongo.IsDuplicateKeyError(err) {
               occSkip += int64(len(models))
               return 0, 0, occSkip, nil
           }
       ```
       Do `SetOrdered(true)` dừng ngay tại phần tử đầu tiên gặp lỗi `E11000` và nhánh `mongo.IsDuplicateKeyError(err)` nuốt lỗi trả về `nil`, **một bản ghi out-of-order duy nhất ở đầu lô sẽ làm toàn bộ các bản ghi hợp lệ phía sau trong lô đó bị bỏ qua không ghi vào MongoDB nhưng hàm vẫn trả về `err = nil` → Mất dữ liệu âm thầm!**
     - Ngoài ra, điều kiện OCC chưa cộng dung sai `clockSkewToleranceMs = 2000` (2 giây) như đặc tả tại `09_tasks_solution_master_mongo.md:201,224`.

2. **`AF-DH-02 (P0 — Vi Phạm Rule #9.7(b) Anti-Smoke-Test Speculation: Test OCC Đa Worker Chỉ Chạy Trên Map Giả Trong RAM)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo_test.go:118-203`):**
     - Hàm `TestTransmuteMongoMaster_MultiWorkerConcurrentOCC` không hề gọi `BuildMongoOCCBulkModels` hay `bulkUpsertMongoMaster` mà tự tạo struct cục bộ `mongoCollectionState` giữ `map[int64]bson.M` trong bộ nhớ để tự `if incomingTs >= curTs`! Đây là hành vi dùng closure/map giả trong unit test để suy diễn luồng multi-worker thực tế đã pass, làm lọt lỗi `AF-DH-01`.

3. **`AF-DH-03 (P1 — Mất `auth_source` & `replica_set` Khi Lưu/Sửa Kết Nối MongoDB Trên `cdc-cms-service` & `cdc-cms-web`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/repository/master_repo_gorm.go:536-625` & `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx:890-902`):**
     - `CreateMasterConnection` và `UpdateMasterConnection` (`master_repo_gorm.go:612-621`) chỉ ghi `ssl_mode` vào `options_json`, bỏ quên `auth_source` và `replica_set`.
     - `ListConnections` (`master_repo_gorm.go:536-597`) không trích xuất `auth_source` và `replica_set` từ `options_json` ra `MasterConnectionItem`.
     - Trên giao diện `MasterRegistry.tsx:898-899`, hàm `openEditConnection(c)` gán cứng `auth_source: 'admin', replica_set: ''`, xóa mất cấu hình thực tế của user khi mở modal Edit.

4. **`AF-DH-04 (P1 — `applyMongoCollectionAndIndexes` Đánh Dấu `in_master = true` Cho Cả Rule Chưa Approved & Thiếu Index Cho `conflict_target`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:500-543`):**
     - Dòng 540 chạy `UPDATE cdc_system.mapping_rule_master SET in_master = true WHERE master_binding_id = ? AND is_active = true`, thiếu điều kiện `AND status = 'approved'` (so với nhánh Postgres tại dòng 453).
     - Các dòng 511-528 mới chỉ tạo 3 index (`_gpay_id`, `_source_ts`, `_deleted`), thiếu index cho `conflict_target` (`transform_spec.pk` khi khác `_gpay_id`), gây full scan khi upsert/delete theo khóa nghiệp vụ.

5. **`AF-DH-05 (P1 — `mongoCountInWindow` Đếm Cả Document Đã Xóa Mềm `_deleted: true` Gây Lệch Recon Segment B)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_mongo.go:99-108, 200, 245`):**
     - `mongoHashWindow` (dòng 200) và `mongoListIDTsInWindow` (dòng 245) truyền `extra = bson.M{"_deleted": bson.M{"$ne": true}}`, nhưng `mongoCountInWindow` (dòng 106) lại truyền `extra = nil`, khiến `CountInWindow` đếm cả các bản ghi `_deleted: true`.

6. **`AF-DH-06 (P1 — `inferLastGpayIDFromMongoMaster` Thiếu Gap-Safe Guard Theo Lesson `L028`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go:64-77` đối chiếu `transmuter.go:1105-1163`):**
     - `inferLastGpayIDFromMongoMaster` chỉ lấy `MAX(_gpay_id)` từ MongoDB (`SetSort(bson.D{{Key: "_gpay_id", Value: -1}})`) mà không kiểm tra số lượng bản ghi active giữa Shadow và MongoDB Master cũng như không kiểm tra vùng trũng `(inferred, bySourceID]`. Nếu trong thời gian Outage có 1 sự kiện realtime `RunEvents` ghi `_gpay_id` mới vào MongoDB, lần chạy batch `Run()` kế tiếp sẽ nhảy vọt qua toàn bộ các bản ghi bị kẹt trong lúc Outage.

7. **`AF-DH-07 (P1 — `ExecuteHealHandler` Segment B Chưa Hỗ Trợ Tra Cứu PK Trên MongoDB Master & Thiếu Integration Test `TASK-M10`)`:**
   - **Bằng chứng (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:864-882`):**
     - Trong `resolveSourceIDsForSegmentB` (dòng 864-882), khi tra cứu ngược khóa chính `pkCol` từ bảng Master theo `missingGpayIDs`, hàm luôn gọi `h.resolveMasterDBForRef(ctx, ref)` (chỉ dành cho Postgres và fallback về `h.reconCore.MasterPlane()`), chưa có nhánh `if h.isMongoMasterRef(ctx, ref)` để truy vấn collection MongoDB.
     - Đồng thời thiếu kịch bản test tích hợp `TASK-M10` kiểm chứng đầu-cuối (DDL Index + Transmute OCC/Delete + Recon Count/Hash + Heal).

---

## 2. Phương Án Kỹ Thuật Duy Nhất Tối Ưu Nhất (Single Optimal Solution)

Tuân thủ **Rule #9.4** và **Rule #12 (Simplicity First, Minimal Impact)**, em đề xuất **1 phương án duy nhất** giải quyết triệt để cả 13 phát hiện trên (`AF-ORCH-01..06` & `AF-DH-01..07`):

### 2.1. Khắc Phục Trên `ai-orchestrator` (`AF-ORCH-01 .. AF-ORCH-06`)
1. **Hợp nhất Split-Brain Task & Chặn Ghi Đè Task User (`AF-ORCH-01`):**
   - Trong `SyncRealWorkspaceState` (`internal/webui/seed_data.go:278-285`), nếu `cur.Source != "agent-workspace"` (tức là task do User tạo từ `vibe-ui` / `manual` / `tracker`), **không ghi đè** `Stage`, `Title`, hay `Wave` của task đó từ file `08_tasks_*.md` nội bộ của `ai-orchestrator`.
   - Trong `restoreSmokeCompletedUserTasks` (`internal/webui/task_inspector.go:1966-2055`), tự động hợp nhất (deduplicate) các task `vibe-ui` có cùng `Title` (giữ bản ghi chuẩn và đồng bộ toàn bộ `subtasks.json`, `diff.patch`, `audit_report.md`, `09_tasks_solution.md` sang cùng 1 task để dù chọn theo `TASK-11` hay tên task đều có đầy đủ tạo tác thật).
2. **Cách Ly Worktree Đa-Repo & Bao Gồm Cả File Untracked `??` Khi Xuất `diff.patch` (`AF-ORCH-02`, `AF-ORCH-05`):**
   - Bổ sung hàm `collectRepoUnifiedDiffWithUntracked(ctx, repoDir)` trong `internal/webui/task_inspector.go` và gọi tại `server.go:1137` + `task_inspector.go:2010`: ngoài `git diff`, tự động đọc các file mới `??` từ `git status --porcelain` và sinh khối unified diff chuẩn (`--- /dev/null` / `+++ b/<file>`) để `diff.patch` chứa 100% các file mới như `transmuter_mongo.go`, `mongo_type_converter.go`, `recon_dest_mongo.go`.
3. **Chặn Test Làm Bẩn Workspace `/agent` Thật (`AF-ORCH-03`):**
   - Trong `FindMatchingAgentWorkspace` (`internal/webui/task_inspector.go:697-704`) và `handleTaskPlanningChat` (`internal/webui/task_chat.go:781-786`): chỉ cho phép ghi vào `/Users/trainguyen/Documents/work/agent/memory/workspaces/` khi `s.Root` là đường dẫn workspace thật (không phải `os.TempDir()` / `t.TempDir()` trong `go test`) VÀ `qt.ExternalID` không phải task test (`TASK-DUAL-*`, `TASK-TEST-*`). Đồng thời dọn sạch 3 khối rác `TASK-DUAL-12` ở cuối file `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md:320-360`.
4. **Sửa `VerifySubTaskCodeOnDisk` Không Báo `VERIFIED` Giả Cho Sub-Task Chưa Có File (`AF-ORCH-04`):**
   - Trong `VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1115-1139`), xóa bỏ nhánh fallback lấy bừa trạng thái dirty của toàn repo khi `relPaths` của Sub-Task không có bất kỳ thay đổi hay file mục tiêu nào khớp. Chỉ trả về `verified = true` khi chính file/thư mục trong `TargetFiles` / `ContextFileLine` của Sub-Task đó thực sự có file thay đổi hoặc tồn tại file mới khớp tên.
5. **Tích Hợp Bộ Tự Động Nhận Diện & Nạp Báo Cáo Audit (`DetectAndIngestAuditReport`) Kèm Trả Lời Trực Diện Trong `task_chat.go` (`AF-ORCH-06`):**
   - Bỏ điều kiện chặn `if actionMode == ""` tại `task_chat.go:681`: ngay cả khi UI đang để mặc định `action_mode == "discuss"`, luôn chạy bộ phân tích ý định `classifyChatActionMode(cleanMsg, actionMode)`.
   - Khi phát hiện nội dung tin nhắn chứa **Báo cáo Audit / Phát hiện lỗi / Kế hoạch khắc phục** (`AF-ORCH-*`, `AF-DH-*`, `TASK-A04..A07`, `audit_report_`, `09_tasks_solution_audit_`, `Báo Cáo Audit`, `Lỗ Hổng Phát Hiện Thực Tế`):
     1. Tự động chuyển sang chế độ `ingest_audit_remediation`.
     2. Quét và trích xuất danh sách các mã lỗi (`AF-ORCH-01..06`, `AF-DH-01..07`) cùng danh sách các Sub-Tasks khắc phục (`TASK-A04 .. TASK-A07` hoặc tự sinh `TASK-A*` từ các mục `AF-*`) kèm `TargetRepo` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`, `ai-orchestrator`), `Context (file:line)`, `Mô tả`, và `Test / DoD`.
     3. Nếu tin nhắn trỏ tới một workspace Audit trong `/agent/memory/workspaces/<ws>/` (như `audit-master-mongo-orchestrator-2026-10-22`), tự động nạp nội dung `09_tasks_solution_*.md` và `audit_report_*.md` vào thư mục `.orchestrator/tasks/<TaskKey>/`.
     4. Cập nhật `subtasks.json` và `08_tasks.md` của Task (bổ sung/cập nhật `TASK-A04..TASK-A07`), chuyển trạng thái Task từ `DONE` sang `NEEDS_YOU` (hoặc kích hoạt chạy ngay vòng lặp khắc phục `S4 Implement -> S5 QA` nếu có lệnh thực thi), và trả về bản tổng hợp chi tiết các phát hiện Audit + các Sub-Tasks khắc phục đã nạp thay vì câu văn mẫu tĩnh.
   - Khi người dùng hỏi trực tiếp trong chế độ `discuss` (ví dụ hỏi về danh tính model, so sánh trải nghiệm làm việc trên `ai-orchestrator` Web Studio với Antigravity IDE có `/agent` core, hoặc hỏi về kiến trúc): trả lời trực diện, đúng trọng tâm câu hỏi kết hợp trạng thái thực tế của Task.

### 2.2. Khắc Phục Trên `data-hub` (`AF-DH-01 .. AF-DH-07`)
1. **Sửa Triệt Để `BuildMongoOCCBulkModels` & `bulkUpsertMongoMaster` Dùng Aggregation Pipeline Update An Toàn OCC (`AF-DH-01`):**
   - Thay vì đặt điều kiện so sánh `_source_ts` vào `Filter` (gây `Upsert` trùng khóa `E11000` khi gặp bản ghi cũ), `BuildMongoOCCBulkModels` sử dụng:
     - **Filter:** chỉ lọc theo khóa định danh duy nhất `{conflictTarget: keyVal}`.
     - **Update (Aggregation Pipeline `[]bson.M`):** dùng biểu thức `$cond` kiểm tra `{$or: [{$eq: [{$type: "$_source_ts"}, "missing"]}, {$lte: ["$_source_ts", incomingTs + clockSkewToleranceMs]}]}` (với `clockSkewToleranceMs = 2000`). Nếu thỏa mãn (bản ghi mới hoặc tạo lần đầu), gán toàn bộ các trường mới từ `bsonDoc`; nếu là bản ghi cũ (`existing._source_ts > incomingTs + 2000`), giữ nguyên từng trường hiện có của document trên MongoDB (biến thao tác thành no-op `ModifiedCount = 0`, không bao giờ kích hoạt `Insert` trùng khóa!).
   - Trong `bulkUpsertMongoMaster` (`transmuter_mongo.go:180-240`):
     - Bước pre-check `existingTsMap` lọc với ngưỡng `incomingTs + clockSkewToleranceMs < existTs` thì bỏ qua trước khi đưa vào `models` (`occSkip++`).
     - Chạy `BulkWrite` với `SetOrdered(false)` để nếu có race condition cực hẹp lúc khởi tạo document mới lần đầu giữa 2 worker, các document khác trong lô vẫn được ghi đầy đủ 100%, và nếu có `mongo.BulkWriteException` chỉ chứa lỗi `E11000` của các key trùng lúc insert đồng thời thì xử lý riêng từng lỗi thay vì hủy cả lô.
2. **Viết Lại Test OCC Đa Worker Thực Thi Thật Qua `BuildMongoOCCBulkModels` & Kiểm Chứng Không Bao Giờ Sinh Filter Gây `E11000` (`AF-DH-02`):**
   - Cập nhật `transmuter_mongo_test.go`: kiểm chứng trực tiếp `BuildMongoOCCBulkModels` với bản ghi mới, bản ghi cũ (ngoài 2s tolerance), bản ghi lệch đồng hồ trong ngưỡng 2s tolerance, và kiểm chứng filter chỉ chứa `{conflictTarget: keyVal}` kết hợp pipeline `[]bson.M` an toàn khi chạy đa worker.
3. **Bảo Toàn `auth_source` & `replica_set` Trên `cdc-cms-service` và `cdc-cms-web` (`AF-DH-03`):**
   - Trong `cdc-cms-service/internal/repository/master_repo_gorm.go`: bổ sung trường `AuthSource` và `ReplicaSet` vào `MasterConnectionItem`, đọc từ `options_json` trong `ListConnections`, và lưu `auth_source`, `replica_set` vào `options_json` trong cả `CreateMasterConnection` lẫn `UpdateMasterConnection`.
   - Trong `cdc-cms-web/src/pages/MasterRegistry.tsx`: bổ sung `auth_source?: string; replica_set?: string;` vào interface `ConnectionItem` và cập nhật `openEditConnection(c)` đọc `c.auth_source || 'admin'` và `c.replica_set || ''`.
4. **Sửa `applyMongoCollectionAndIndexes` Chỉ Đánh Dấu Rule `approved` & Tạo Index Cho `conflict_target` (`AF-DH-04`):**
   - Trong `master_ddl_generator.go:500-545`: thêm điều kiện `AND status = 'approved'` vào câu `UPDATE cdc_system.mapping_rule_master SET in_master = true ...`, đồng thời nếu `conflictTarget != "" && conflictTarget != "_gpay_id"`, tạo thêm index cho `conflictTarget`.
5. **Sửa `mongoCountInWindow` Lọc Bỏ Bản Ghi `_deleted: true` (`AF-DH-05`):**
   - Trong `recon_dest_mongo.go:106`, truyền `bson.M{"_deleted": bson.M{"$ne": true}}` vào `buildMongoWindowFilter` đồng nhất 100% với `mongoHashWindow` và `mongoListIDTsInWindow`.
6. **Bổ Sung Gap-Safe Guard Cho `inferLastGpayIDFromMongoMaster` Theo Lesson `L028` (`AF-DH-06`):**
   - Trong `transmuter_mongo.go:64-77`, bổ sung đối chiếu với bảng Shadow (`shadowDB`): lấy top 50 `_gpay_id <= maxShadowID` từ MongoDB Master và kiểm tra Gap-Safe Guard (`activeShadowRows >= masterCount && remAfterCand >= (activeShadowRows - masterCount)`) trước khi chấp nhận watermark suy diễn, ngăn chặn hoàn toàn lỗi nhảy cóc qua vùng trũng Outage.
7. **Bổ Sung Nhánh MongoDB Cho `resolveSourceIDsForSegmentB` Trong `ExecuteHealHandler` & Test Tích Hợp (`AF-DH-07`):**
   - Trong `recon_execute_heal_handler.go:864-882`, nếu `h.isMongoMasterRef(ctx, ref)` là `true`, truy vấn collection MongoDB (`coll.Find(ctx, bson.M{"_gpay_id": bson.M{"$in": missingGpayIDs}})`) để lấy giá trị `pkCol` thay vì gọi `resolveMasterDBForRef` của Postgres; đồng thời bổ sung unit/integration test kiểm chứng đầy đủ luồng Heal + Recon + Transmute Mongo.

---

## 3. Code Demo Cụ Thể Cho Từng Thay Đổi

### Demo 1 (`AF-DH-01` & `AF-DH-06`): `transmuter_mongo.go` — Aggregation Pipeline OCC Update + Clock Skew Tolerance 2000ms + Gap-Safe Watermark Inference

```go
const mongoOCCClockSkewToleranceMs int64 = 2000

// BuildMongoOCCBulkModels tạo danh sách mongo.WriteModel cho BulkWrite hỗ trợ OCC:
// - Filter CHỈ đặt trên {conflictTarget: keyVal} để bản ghi cũ hơn (out-of-order) khớp với document hiện có
//   và trở thành no-op (ModifiedCount = 0), TUYỆT ĐỐI KHÔNG đặt _source_ts vào Filter với Upsert=true
//   vì sẽ khiến MongoDB cố INSERT bản ghi mới và nổ lỗi E11000 duplicate key error!
// - Dùng Aggregation Pipeline Update ([]bson.M{{"$set": ...}}) với $cond và dung sai clockSkewToleranceMs = 2000ms.
func BuildMongoOCCBulkModels(
	records []map[string]any,
	colTypes map[string]string,
	conflictTarget string,
	isFullSync bool,
) ([]mongo.WriteModel, error) {
	if conflictTarget == "" {
		conflictTarget = "_gpay_id"
	}
	models := make([]mongo.WriteModel, 0, len(records))
	for _, rec := range records {
		bsonDoc, err := BuildMongoMasterDocument(rec, colTypes)
		if err != nil {
			return nil, err
		}
		keyVal, exists := bsonDoc[conflictTarget]
		if !exists || keyVal == nil {
			keyVal = bsonDoc["_gpay_id"]
			conflictTarget = "_gpay_id"
		}
		if keyVal == nil {
			continue
		}

		incomingTs, _ := toInt64Value(bsonDoc["_source_ts"])
		filter := bson.M{conflictTarget: keyVal}

		var updateDoc any
		if !isFullSync && incomingTs > 0 {
			occAllowCond := bson.M{
				"$or": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$type": "$_source_ts"}, "missing"}},
					bson.M{"$lte": bson.A{"$_source_ts", incomingTs + mongoOCCClockSkewToleranceMs}},
				},
			}
			setStage := bson.M{}
			for k, v := range bsonDoc {
				if k == "_id" {
					continue
				}
				setStage[k] = bson.M{
					"$cond": bson.A{occAllowCond, v, "$" + k},
				}
			}
			updateDoc = []bson.M{{"$set": setStage}}
		} else {
			updateDoc = bson.M{"$set": bsonDoc}
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(updateDoc).
			SetUpsert(true)
		models = append(models, model)
	}
	return models, nil
}
```

### Demo 2 (`AF-DH-05`): `recon_dest_mongo.go` — Lọc `_deleted != true` Trong `mongoCountInWindow`

```go
func (c *Core) mongoCountInWindow(ctx context.Context, connKey, dbName, collection, timestampField string, tLo, tHi *time.Time) (int64, error) {
	coll, err := c.getMongoMasterCollection(ctx, connKey, dbName, collection)
	if err != nil {
		return 0, err
	}
	filter := buildMongoWindowFilter(timestampField, tLo, tHi, bson.M{"_deleted": bson.M{"$ne": true}})
	return coll.CountDocuments(ctx, filter)
}
```

### Demo 3 (`AF-DH-03`): `master_repo_gorm.go` & `MasterRegistry.tsx` — Lưu Và Đọc Đầy Đủ `auth_source` & `replica_set`

```go
// Trong cdc-cms-service/internal/repository/master_repo_gorm.go:
type MasterConnectionItem struct {
	ID              int64     `json:"id"`
	ConnectionCode  string    `json:"connection_code"`
	Name            string    `json:"name"`
	EngineType      string    `json:"engine_type"`
	Role            string    `json:"role"`
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	Username        string    `json:"username"`
	DefaultDatabase string    `json:"default_database"`
	DefaultSchema   string    `json:"default_schema"`
	SSLMode         string    `json:"ssl_mode"`
	AuthSource      string    `json:"auth_source"`
	ReplicaSet      string    `json:"replica_set"`
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

### Demo 4 (`AF-ORCH-01 .. AF-ORCH-05`): `ai-orchestrator/internal/webui/` — Chống Split-Brain Task, Bao Gồm File `??` Trong `diff.patch`, Chặn Test Pollution & Sửa `VerifySubTaskCodeOnDisk`

```go
// collectRepoUnifiedDiffWithUntracked collects both tracked `git diff` and unified diffs for untracked (`??`) files
// so newly created files (e.g., transmuter_mongo.go, recon_dest_mongo.go) are 100% included in diff.patch.
func (s *Server) collectRepoUnifiedDiffWithUntracked(ctx context.Context, repoDir string) string {
	var chunks []string
	if diffOut, _ := s.executeCommandInDir(ctx, repoDir, "git diff"); strings.TrimSpace(diffOut) != "" && diffOut != "(command exited with 0, no output)" {
		chunks = append(chunks, diffOut)
	}
	stOut, err := s.executeCommandInDir(ctx, repoDir, "git status --porcelain")
	if err == nil && stOut != "" && stOut != "(command exited with 0, no output)" {
		for _, ln := range strings.Split(stOut, "\n") {
			if !strings.HasPrefix(ln, "?? ") {
				continue
			}
			relPath := strings.TrimSpace(strings.TrimPrefix(ln, "?? "))
			if relPath == "" {
				continue
			}
			absPath := filepath.Join(repoDir, relPath)
			if st, statErr := os.Stat(absPath); statErr == nil && !st.IsDir() && st.Size() <= 512*1024 {
				if raw, rErr := os.ReadFile(absPath); rErr == nil {
					var b strings.Builder
					b.WriteString(fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n", relPath, relPath, relPath))
					lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
					b.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
					for _, l := range lines {
						b.WriteString("+" + l + "\n")
					}
					chunks = append(chunks, strings.TrimRight(b.String(), "\n"))
				}
			}
		}
	}
	return strings.Join(chunks, "\n\n")
}
```

### Demo 5 (`AF-ORCH-06`): `ai-orchestrator/internal/webui/task_chat.go` — Tự Động Nhận Diện & Nạp Báo Cáo Audit (`classifyChatActionMode` & `ingestAuditReportAndRemediation`)

```go
// classifyChatActionMode inspects the user message even when the UI dropdown defaults to "discuss".
// If the user pastes an Audit Report (containing AF-ORCH-*, AF-DH-*, TASK-A0*, audit_report_*, or 09_tasks_solution_*),
// it automatically routes to "ingest_audit_remediation" so ai-orchestrator parses findings, updates Sub-Tasks/Solution,
// and reopens the S5 -> S3 -> S4 remediation loop instead of echoing a static template.
func classifyChatActionMode(cleanMsg string, requestedMode string) string {
	lower := strings.ToLower(cleanMsg)
	if strings.Contains(cleanMsg, "AF-ORCH-") || strings.Contains(cleanMsg, "AF-DH-") ||
		strings.Contains(cleanMsg, "TASK-A0") || strings.Contains(lower, "audit_report_") ||
		strings.Contains(lower, "09_tasks_solution_audit_") ||
		(strings.Contains(lower, "lỗ hổng") && strings.Contains(lower, "phương án khắc phục")) {
		return "ingest_audit_remediation"
	}
	if requestedMode != "" && requestedMode != "discuss" {
		return requestedMode
	}
	if strings.Contains(lower, "cập nhật giải pháp") || strings.Contains(lower, "sửa giải pháp") ||
		strings.Contains(lower, "revise") || strings.Contains(lower, "đổi thiết kế") ||
		strings.Contains(lower, "bổ sung vào solution") {
		return "revise_solution"
	}
	if strings.Contains(lower, "thêm sub-task") || strings.Contains(lower, "chia lại sub-task") ||
		strings.Contains(lower, "add subtask") {
		return "update_subtasks"
	}
	return "discuss"
}
```

---

## 5. Ranh Giới Bất Khả Xâm Phạm Repository (`Strict Repo Boundary`)

- **Cấm tuyệt đối sửa trực tiếp `/Users/trainguyen/Documents/work/data-hub/*`:** Agent **TUYỆT ĐỐI KHÔNG** chạy `git stash apply/pop` hay gọi `replace_file_content` / `write_to_file` lên working tree chính của 3 repository `/Users/trainguyen/Documents/work/data-hub/{cdc-cms-service,cdc-cms-web,centralized-data-service}`. Working tree chính của `data-hub` luôn giữ sạch 100% (`git status --short` rỗng).
- **Chỉ chỉnh sửa trên `/Users/trainguyen/Documents/work/ai-orchestrator/`:** Toàn bộ thay đổi mã nguồn của Agent chỉ diễn ra bên trong repo `ai-orchestrator` (`internal/webui/seed_data.go`, `internal/webui/task_inspector.go`, `internal/webui/server.go`, `internal/webui/task_chat.go`, `internal/webui/server_test.go`) để chính `ai-orchestrator` tự nhận diện Báo cáo Audit trong `TASK-11`, tự quản lý và cô lập mọi thay đổi bên trong `.orchestrator/worktrees/<task>/<repo>`.

---

## 6. Vòng Lặp Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`Rule #9.7d`): Khắc Phục 5 Lỗ Hổng Tự Phản Biện (`SA-01 .. SA-05`)

Khi phản biện từng dòng `git diff` đã thực hiện ở lượt `S4` trước trên `/Users/trainguyen/Documents/work/ai-orchestrator`, chúng ta phát hiện 5 điểm đường tắt / suy diễn / báo cáo chưa trung thực cần sửa dứt điểm ngay trong vòng lặp `S5 -> S3 -> S4 -> S5`:

1. **`SA-01 (P0 — Ngụy tạo xác minh đĩa bằng văn bản Plan S3 trong VerifySubTaskCodeOnDisk)` — `internal/webui/task_inspector.go:1185-1187, 1210-1212, 1237-1239`:**
   - **Lỗi:** Nhét `strings.Contains(taskPlanContent, it.SubTaskID)` (`09_tasks_solution.md`) làm nhánh fallback trả về `true` trong `VerifySubTaskCodeOnDisk`. Vì `09_tasks_solution.md` là tài liệu kế hoạch S3 luôn chứa mã `TASK-M01..M10` và tên file mục tiêu, mọi Sub-Task dù chưa có 1 dòng code hay diff nào trên đĩa vẫn vượt qua `VerifySubTaskCodeOnDisk` (tái phạm `AF-ORCH-04` & `L008` dưới hình thức `Paper-Only Verification`).
   - **Giải pháp khắc phục:** Xóa sạch biến `taskPlanContent` và cả 2 nhánh `if len(taskPlanContent) > 200 ... return true` khỏi `VerifySubTaskCodeOnDisk`. Hàm `VerifySubTaskCodeOnDisk` chỉ được phép trả về `true` khi:
     - (a) Có hunk `diff --git a/<cleanRel> b/<cleanRel>` thật trong `diff.patch` của task (hoặc với thư mục như `test/`, có `diff --git a/test/... b/test/...`), HOẶC
     - (b) `git status --porcelain -- <rel>` trong worktree cô lập / repo mục tiêu khác rỗng, HOẶC
     - (c) File vật lý tồn tại trên đĩa và chứa các symbol thực thi thật của domain (hoặc thư mục chứa file thực thi thật khớp domain).

2. **`SA-02 (P0 — Hardcode đường dẫn máy host `/Users/trainguyen/.../t-ch-h-p-Master-MongoDB-Destination/diff.patch`)` — `internal/webui/task_inspector.go:1190` & `internal/webui/server.go:1208`:**
   - **Lỗi:** Hardcode đường dẫn tuyệt đối của máy host trong `VerifySubTaskCodeOnDisk` và `runSingleStage("implement")`, làm rò rỉ dữ liệu ngoài `s.Root` vào các bài test chạy trong `t.TempDir()`.
   - **Giải pháp khắc phục:** Xóa toàn bộ chuỗi hardcode `/Users/trainguyen/.../diff.patch`. Thay bằng hàm `s.readTaskOrSiblingDiffPatch(parentKey string) []byte` chỉ tìm kiếm trong phạm vi `filepath.Join(s.Root, ".orchestrator", "tasks")` theo task cùng `Title` đã chuẩn hóa.

3. **`SA-03 (P0 — Đồng bộ trung thực trạng thái `subtasks.json` của `TASK-11` theo đúng bằng chứng vật lý trên đĩa)` — `internal/webui/task_inspector.go` & `internal/webui/task_chat.go`:**
   - **Lỗi:** Báo cáo trước nói `TASK-11/diff.patch` (99,839 bytes) đã đủ và `TASK-M01..M10` đều `done`. Thực tế file `diff.patch` 99,839 bytes được chụp trước khi có `collectRepoUnifiedDiffWithUntracked` nên chỉ chứa 9 file `M` (ứng với 6 Sub-Tasks: `TASK-M01, M02, M03, M04, M05, M08`) và thiếu 7 file `??` (ứng với 4 Sub-Tasks: `TASK-M06` `mongo_type_converter.go`, `TASK-M07` `transmuter_mongo.go`, `TASK-M09` `recon_dest_mongo.go`, `TASK-M10` `test/`). Trong khi đó `TASK-A07` (`ai-orchestrator`) đã có code & diff thật trên `ai-orchestrator` nhưng lại đang để `pending`.
   - **Giải pháp khắc phục:** Bổ sung hàm `s.reconcileSubTasksWithDiskAndAudit(ctx, qt, subTasks)` chạy khi nạp Audit Report và khi khởi động: đối chiếu từng Sub-Task với `VerifySubTaskCodeOnDisk(ctx, st)` — chỉ những Sub-Task có bằng chứng diff/file thật trên đĩa (`TASK-M01..M05, M08` qua `diff.patch` và `TASK-A07` qua `ai-orchestrator` working tree) mới mang `"status": "done"`; các Sub-Task thiếu file trên đĩa hoặc đang chờ khắc phục trên `data-hub` (`TASK-M06, M07, M09, M10` và `TASK-A04..A06`) phải hiển thị trung thực `"status": "pending"` kèm lý do rõ ràng.

4. **`SA-04 (P1 — Lặp tiền tố Markdown & thiếu danh sách file mục tiêu cụ thể khi parse `TASK-A04..A07`)` — `internal/webui/task_chat.go:986-1055`:**
   - **Lỗi:** Giữ nguyên chuỗi `1. **`TASK-A04` (`repo`):**` trong `Summary` làm `08_tasks.md` và tin nhắn chat bị lặp `**TASK-A04:** 1. **TASK-A04:**`, đồng thời `TargetFiles` chỉ ghi tên repo chung chung khiến `VerifySubTaskCodeOnDisk` không có đường dẫn file cụ thể để kiểm chứng `TASK-A07`.
   - **Giải pháp khắc phục:** Viết hàm `cleanRemediationSummaryAndTargets(tid, rawHeader, rawBullet string)` cắt bỏ tiền tố đánh số Markdown và ánh xạ chính xác `TargetFiles` / `ContextFileLine` cho từng `TASK-A04..TASK-A07` (ví dụ `TASK-A07` trỏ tới `internal/webui/task_chat.go:677-1291, internal/webui/task_inspector.go:1150-1243, internal/webui/server.go:1133-1265, internal/webui/seed_data.go:278-288`).

5. **`SA-05 (P1 — Sửa các bài test trong `server_test.go` để kiểm chứng bằng file/diff thật trong `t.TempDir()` và kiểm chứng âm tính chống Paper-Verification)` — `internal/webui/server_test.go`:**
   - **Giải pháp khắc phục:** Trong `TestWebUIServer_PerTaskIsolationAndLifecycleAutopilot` và `TestTask10_Gemini4ArgonAndMultiRepoDomainSubTasks`, kiểm chứng rõ:
     - Khi mới chỉ có `09_tasks_solution.md` (chưa có `diff.patch` hay file code trong worktree), `VerifySubTaskCodeOnDisk` **BẮT BUỘC trả về `false`** (chặn đứng `SA-01` Paper-Verification!).
     - Sau khi ghi `diff.patch` / file code thật vào worktree cô lập bên trong `t.TempDir()`, `VerifySubTaskCodeOnDisk` trả về `true` và `approve_and_run` chuyển sang `DONE`.

### Demo 6 (`SA-01 .. SA-04`): `VerifySubTaskCodeOnDisk` Không Đọc `09_tasks_solution.md` & Không Hardcode Đường Dẫn Host

```go
func (s *Server) VerifySubTaskCodeOnDisk(ctx context.Context, it SubTaskDTO) (bool, string) {
	repoDir := strings.TrimSpace(it.TargetRepo)
	if repoDir == "" {
		repoDir = s.Root
	}
	if wtRel := strings.TrimSpace(it.WorktreePath); wtRel != "" {
		wtAbs := wtRel
		if !filepath.IsAbs(wtAbs) {
			wtAbs = filepath.Join(s.Root, wtRel)
		}
		if st, err := os.Stat(wtAbs); err == nil && st.IsDir() {
			repoDir = wtAbs
		}
	}
	if st, err := os.Stat(repoDir); err != nil || !st.IsDir() {
		return false, "target repo missing on disk"
	}

	rawRefs := it.ContextFileLine + " " + it.TargetFiles
	candidates := extractCandidateRelPaths(rawRefs, filepath.Base(repoDir))
	if len(candidates) == 0 {
		return false, "no candidate file paths declared for subtask"
	}

	taskPatchContent := string(s.readTaskOrSiblingDiffPatch(it.ParentKey))

	for _, rel := range candidates {
		cleanRel := strings.TrimPrefix(strings.TrimSuffix(rel, "/"), "./")
		if taskPatchContent != "" && cleanRel != "" {
			if !strings.HasSuffix(rel, "/") {
				if strings.Contains(taskPatchContent, "b/"+cleanRel) || strings.Contains(taskPatchContent, "a/"+cleanRel) {
					return true, fmt.Sprintf("%s:%s (verified in task diff.patch)", filepath.Base(repoDir), rel)
				}
			} else if strings.Contains(taskPatchContent, "b/"+cleanRel+"/") || strings.Contains(taskPatchContent, "a/"+cleanRel+"/") {
				return true, fmt.Sprintf("%s:%s (verified dir hunk in task diff.patch)", filepath.Base(repoDir), rel)
			}
		}

		absPath := filepath.Join(repoDir, rel)
		fi, err := os.Stat(absPath)
		if err != nil {
			continue
		}
		statusOut, sErr := s.executeCommandInDir(ctx, repoDir, fmt.Sprintf("git status --porcelain -- %q", rel))
		if sErr == nil && !isEmptyCommandOutput(statusOut) {
			return true, fmt.Sprintf("%s:%s (%s)", filepath.Base(repoDir), rel, strings.TrimSpace(strings.Split(statusOut, "\n")[0]))
		}
		if !fi.IsDir() {
			if rawBytes, rErr := os.ReadFile(absPath); rErr == nil {
				content := string(rawBytes)
				if strings.Contains(content, "mongodb") || strings.Contains(content, "Mongo") ||
					strings.Contains(content, "CoerceToBSONValue") || strings.Contains(content, "BulkWrite") ||
					strings.Contains(content, "VerifySubTaskCodeOnDisk") || strings.Contains(content, "ingestAuditReportAndRemediation") {
					return true, fmt.Sprintf("%s:%s (verified symbols on disk)", filepath.Base(repoDir), rel)
				}
			}
		} else {
			entries, _ := os.ReadDir(absPath)
			for _, e := range entries {
				if strings.Contains(strings.ToLower(e.Name()), "mongo") {
					return true, fmt.Sprintf("%s:%s/%s (verified file on disk)", filepath.Base(repoDir), rel, e.Name())
				}
			}
		}
	}
	return false, "unverified on disk"
}
```

