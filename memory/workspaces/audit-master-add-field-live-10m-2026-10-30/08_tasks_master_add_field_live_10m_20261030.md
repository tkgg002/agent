# 08. TASKS — `master_add_field_live_10m_20261030` (V2 — Post Adversarial Plan Audit)

## Danh sách Công việc (Audit & Architectural Readiness Analysis)

- [ ] **TASK-01 (S2 Investigate — CMS Mapping & Schema Evolution / DDL trên Master):**
  - **Context:** `cdc-cms-service` & `centralized-data-service` (luồng thêm field mapping mới cho `master_binding`, approve mapping, và chạy DDL `ALTER TABLE ... ADD COLUMN` trên PostgreSQL Master / thêm field trên MongoDB Master).
  - **Test/Verify:** Đối chiếu `file:line` xem khi thêm 1 field mới vào `master`, hệ thống thực thi `ALTER TABLE` hay `DROP/CREATE`, có giữ dữ liệu cũ không, và trên PostgreSQL 10 triệu dòng lệnh `ADD COLUMN` có gây lock luồng đọc live không.
  - **DoD:** Xác định chính xác `file:line` của luồng thêm field mapping và DDL Master.

- [ ] **TASK-02 (S2 Investigate — Luồng Live CDC Stream `source -> shadow -> master` khi thêm field):**
  - **Context:** `centralized-data-service` (cache `field_mappings`, reload mapping sau khi approve, và câu lệnh Upsert realtime sang PostgreSQL Master / MongoDB Master).
  - **Test/Verify:** Kiểm tra xem các bản ghi mới phát sinh từ `source` sau thời điểm thêm field có tự động lấy field mới từ `shadow` sang `master` không, và cache mapping bao lâu mới cập nhật.
  - **DoD:** Xác định chính xác `file:line` cơ chế cache/reload mapping và câu lệnh Upsert.

- [ ] **TASK-03 (S2 Investigate — Luồng Backfill / Full Re-Transmute 10 triệu bản ghi lịch sử từ `shadow -> master` trong lúc đang chạy Live):**
  - **Context:** `centralized-data-service` (luồng `Transmute` / `SyncMaster` / `Re-Transmute` từ `shadow` sang `master`).
  - **Test/Verify:** Kiểm tra 4 yếu tố sống còn ở quy mô 10 triệu dòng đang live:
    1. Cơ chế đọc phân trang từ `shadow` (cursor/keyset hay `OFFSET`, batch size bao nhiêu, có làm treo DB không).
    2. Cơ chế ghi Upsert vào `master` (`ON CONFLICT DO UPDATE` trên PG / `BulkWrite` trên Mongo) có khóa bảng chặn luồng đọc live của `master` không.
    3. **Xung đột OCC (Race Condition giữa Backfill và Live CDC):** Nếu luồng Live CDC vừa ghi bản ghi phiên bản mới (`T2`) vào `master`, sau đó luồng Backfill 10M đọc trúng bản ghi cũ hơn (`T1`) hoặc chạy đè lên, thì điều kiện `WHERE master._source_ts <= EXCLUDED._source_ts` trên PostgreSQL Master và `$cond` trên MongoDB Master có bảo vệ hoặc vô tình **chặn luôn việc cập nhật field mới** cho các dòng có `_source_ts` bằng nhau (`<=` vs `<`) không!
    4. Áp lực I/O, MVCC Dead Tuples (`VACUUM`) trên PostgreSQL Master hoặc WiredTiger cache trên MongoDB Master khi `UPDATE` toàn bộ 10 triệu dòng liên tục không có throttle/rate-limit.
  - **DoD:** Báo cáo đầy đủ bằng chứng `file:line` cho cả 4 yếu tố trên, qua cổng `DG0` và `DG1/DG3`.


---

## Bổ sung Kế hoạch Thực thi V2 (Sau Phản biện Plan V1 — SRP & Backward Compatibility)

- [x] **TASK-04 (S2.5 & S3 V2 — Phản biện Kế hoạch V1, Thiết kế Tách File Đơn Trách Nhiệm SRP & Kiểm định `/laya_router`):**
  - **Context:** Phản biện từng mục tiêu, từng file và từng dòng update của Plan V1; đảm bảo `NOW <= HEAD` trên mọi file lớn (`transmuter.go` `1835` dòng, `master_ddl_generator.go` `743` dòng, `activity_log_read_repo_gorm.go` `599` dòng) và tương thích ngược 100% với toàn bộ test hiện hữu.
  - **Test/Verify:** Chạy `/laya_router` (`laya_gate.py run --stage dg1` & `dg3` + `review`) kiểm định toàn bộ claims của Plan V2.
  - **DoD:** Hoàn tất cập nhật `09_tasks_solution_master_add_field_live_10m_20261030.md` (V2) và `audit_report_master_add_field_live_10m_20261030.md` (V2), trình User chờ `APPROVE`.

- [ ] **TASK-05 (S4 — Tách file `transmuter_batch_guard.go` & cập nhật `transmuter.go` xử lý `F1`, `F2`, `F4`):**
  - **Context:**
    - Tạo mới `centralized-data-service/internal/service/master/transmuter_batch_guard.go` (`~135` dòng): chứa `dedupAndSortMasterBatchRecords`, `bulkUpsertMasterWithSchemaGuard`, `filterRecordsToExistingMasterColumns`, `paceFullSyncBatch`.
    - Cập nhật `centralized-data-service/internal/service/master/transmuter.go` (giảm từ `1835` xuống `~1775` dòng, `NOW < HEAD`): sửa dòng `451` (`runType == "full" && len(chunkIDs) > 0`), dòng `576` (`paceFullSyncBatch`), dòng `1202-1270` (gọi `dedupAndSortMasterBatchRecords`), dòng `1292` (gọi `bulkUpsertMasterWithSchemaGuard`).
  - **Test/Verify:** `go test -v ./internal/service/master/...` PASS 100% (bao gồm `transmuter_orphan_test.go`, `transmuter_events_batch_test.go`, `transmuter_mongo_test.go`, `transmuter_gpayid_test.go`).
  - **DoD:** `wc -l transmuter.go <= 1785` (`< 1835`), toàn bộ unit/integration tests PASS.

- [ ] **TASK-06 (S4 — Tách file `master_ddl_generator_pg.go` & cập nhật `master_ddl_generator.go` xử lý `F3`):**
  - **Context:**
    - Tạo mới `centralized-data-service/internal/service/master/master_ddl_generator_pg.go` (`~395` dòng): tách `applyPostgresTableAndIndexes`, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable` từ `master_ddl_generator.go`; bổ sung kiểm tra `information_schema.columns` (chỉ `ALTER TABLE ADD COLUMN` cho cột mới chưa có, giữ transaction `< 5ms`) và tách `CREATE INDEX CONCURRENTLY IF NOT EXISTS` cho các index chưa có trong `pg_indexes` ra ngoài transaction.
    - Cập nhật `centralized-data-service/internal/service/master/master_ddl_generator.go` (giảm từ `743` xuống `~360` dòng, `NOW < HEAD`).
  - **Test/Verify:** `go test -v ./internal/service/master/... ./internal/handler/master/...` PASS 100%.
  - **DoD:** `wc -l master_ddl_generator.go <= 370` (`< 743`), `wc -l master_ddl_generator_pg.go <= 420` (`< 500`).

- [ ] **TASK-07 (S4 — Cập nhật `PublishMasterCreate` truyền `master_binding_id` (`F2`) & sửa `GetDailyFlow` nhận `'full'` (`F5`) trong `cdc-cms-service`):**
  - **Context:**
    - `cdc-cms-service/internal/domain/master/publisher.go:13`
    - `cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go:124-136`
    - `cdc-cms-service/internal/app/commands/master/save_master_rule.go:70-74`
    - `cdc-cms-service/internal/app/commands/master/batch_update_rules.go:47-53`
    - `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:428, 438, 454-462` (sửa in-place `599/599` dòng `NOW == HEAD`).
  - **Test/Verify:** `go test ./...` trên `cdc-cms-service` PASS 100%.
  - **DoD:** Không phát sinh dòng mới trên `activity_log_read_repo_gorm.go` (`599` dòng), truyền đúng `master_binding_id` và phân loại đúng `run_type IN ('full', 'full_sync')`.
