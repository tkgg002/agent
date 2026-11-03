# 01 Requirements: Tích hợp đồng thời 2 Feature (Event Date Split & Batching + Master MongoDB Destination) trên Codebase `2026-10-28`

tier: T2
suffix: cdc_3repos_master_mongo
date: 2026-10-28

## 1. Yêu cầu của User
- **Lượt 1 (Đã hoàn tất):** Kiểm tra toàn bộ `git diff` (`status`, `diff`, `stash`, `log`) trên 3 repo CDC (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) đối chiếu với task **Tích hợp Master MongoDB Destination** (`TASK-M01..M10`).
- **Lượt 2 (Mục tiêu hiện tại):**
  1. **Lên đồng thời cả 2 Feature** trên codebase hiện tại (`2026-10-28`, `HEAD 786db91`):
     - **Feature 1:** *Event Date Split & Event Batching* (hiện đang nằm trên working tree của `centralized-data-service` và `activity_log_read_repo_gorm.go` ở `cdc-cms-service`).
     - **Feature 2:** *Master MongoDB Destination* (`TASK-M01` → `TASK-M10` xuyên suốt `cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`).
  2. **TUYỆT ĐỐI KHÔNG chạy `git stash apply` / `git stash pop`** vào phiên hiện tại, vì code trong 3 stash (`stash@{0..2}` ngày `2026-10-22`, base commit `43e5da7`) đã outdated (lệch 29 file, +2,977 / -949 dòng so với `HEAD 786db91`) và chưa được audit kỹ đối chiếu với kiến trúc mới (`ChunkStreamBucketEngine`, `transmuter_identity.go`, `transmuter_events_batch.go`, `recon_execute_heal_segb.go`).
  3. **Bóc tách, Audit từng dòng của 3 stash (`stash@{0..2}`) đối chiếu với code hiện tại (`2026-10-28`), chỉ rõ toàn bộ bug/điểm outdated, lên lại Plan (`09_tasks_solution_*.md` + `08_tasks_*.md`) và viết Code Demo hoàn chỉnh** để tích hợp sạch sẽ vào codebase hiện tại. Dừng chờ `APPROVE` trước khi chạm vào code 3 repo CDC.

## 2. Ràng buộc thực thi từ Bài học (Lesson-as-Enforced-Constraint — `L007`)
- **`L008` / `L028` / `L038` (`#never-touch-external-cdc-repo`, `#read-only-diff-check`, `#propose-only`):** Tuyệt đối giữ chế độ **Propose-Only** trên cả 3 repo `data-hub` khi chưa có `APPROVE` của User. Cấm chạy `git stash apply`, `git checkout`, hay sửa bất kỳ file nào trong `/Users/trainguyen/Documents/work/data-hub/*` ở bước S1–S3.
- **`L022` (`#direct-answer-summary-first`):** Mở đầu câu trả lời bằng khối Summary ngắn gọn trả lời trực tiếp câu hỏi của User trước khi đi vào bảng phân tích & Code Demo.
- **`L051` & `L079` (`#shared-state-side-effect-review`, `#mongodb-occ-pipeline-update`):**
  - Phải bảo toàn `_gpay_id` của document đã tồn tại trên MongoDB Master khi `conflictTarget != "_gpay_id"` (dùng `$ifNull: ["$_gpay_id", v]` trong aggregation pipeline update và `$setOnInsert` trong standard update), khớp 100% với hành vi `ON CONFLICT (<pk>) DO UPDATE` của PostgreSQL trong `transmuter_identity.go:15-17`.
  - Phải cập nhật đầy đủ `ChunkStreamBucketEngine.GetMasterAgent`, `ReconDestAgent.ColumnExists` / `GetRealColumnName` / `BucketCounts` cho MongoDB (`da.IsMongo()`), và `recon_execute_heal_segb.go` (file mới tách từ `786db91`).
- **Lượt 3 (Kiểm tra lỗi Step 1 — Thêm/Test kết nối Master MongoDB):**
  - User báo lỗi khi thêm/kiểm tra kết nối Master MongoDB ở Bước 1:
    `MongoDB ping failed: server selection error: context deadline exceeded, current topology: { Type: ReplicaSetNoPrimary, Servers: [{ Addr: gpay-mongo:27017, Type: Unknown, Last error: dial tcp: lookup gpay-mongo: no such host }, ] }`
  - Yêu cầu kiểm tra nguyên nhân gốc rễ (`file:line` + cấu hình container `gpay-mongo` + bản ghi `cdc_system.connection_registry`) và đề xuất giải pháp + Code Demo chuẩn xác trước khi sửa code.
- **Lượt 4 (Kiểm tra lỗi Step 2 — Thêm PostgreSQL Source Connector `devcoretrans`):**
  - User báo lỗi khi thêm connector PostgreSQL `devcoretrans`:
    `io.debezium.DebeziumException: Creation of replication slot failed ... Caused by: org.postgresql.util.PSQLException: FATAL: permission denied to start WAL sender Detail: Only roles with the REPLICATION attribute may start a WAL sender process.`
  - Yêu cầu kiểm tra nguyên nhân thực tế trên DB `10.200.187.110:5432` (`core-trans-proxy-history-service`) và đề xuất cách xử lý triệt để (cả cấp quyền DB lẫn preflight check tại `cdc-cms-service`).
- **Lượt 5 (Nâng cấp UX/API New Connect cho PostgreSQL Connector — `TASK-M13`):**
  - Yêu cầu: Trong modal **New Connect** của trang `SourceConnectors.tsx` (`cdc-cms-web`), khi chọn `Database Type = postgresql`, bổ sung:
    1. Nút **Check Connection** kiểm tra kết nối PostgreSQL đồng thời xác thực đầy đủ các điều kiện tiên quyết của Debezium PostgreSQL Connector (`wal_level = 'logical'`, user có quyền `REPLICATION` hoặc `SUPERUSER` trong `pg_roles`, tồn tại database và schema).
    2. Tự động **quét danh sách các bảng (`schema.table`) theo `Schema Include List` vừa điền** (mặc định `public`), hiển thị vào dropdown chọn nhiều `<Select mode="multiple">` của `Table Include List` (tự động chọn tất cả bảng tìm thấy khi Check PASS để user dễ dàng bỏ chọn bảng không cần CDC, đồng thời khóa nút `Create` ở chế độ tạo mới cho tới khi Check Connection đạt `ok` — đồng bộ UX với MongoDB).
  - **Ràng buộc bổ sung từ bài học (`L042` — `#react-rules-of-hooks`):** Mọi lời gọi `Form.useWatch(...)` và custom hook (`useCheckPostgresConnection`) trong `SourceConnectors.tsx` bắt buộc khai báo vô điều kiện ở top-level của component, tuyệt đối không gọi inline bên trong JSX điều kiện `dbKind === 'postgresql'`.


