# 08. Tasks Breakdown — `master_mysql_clickhouse`

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

### [x] TASK-01: Mở rộng Quản lý Kết nối & Nhận diện Engine Master (`mysql`, `clickhouse`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu cơ chế của MongoDB trong `centralized-data-service/internal/service/source/connection_manager.go:67-285` và `centralized-data-service/pkgs/database/multi.go:180-235`.
  - Tách file đơn trách nhiệm `centralized-data-service/internal/service/source/connection_manager_sql_masters.go:1-118` để quản lý kết nối `*gorm.DB` cho Master MySQL (`go-sql-driver/mysql`) và Master ClickHouse (`ClickHouse/clickhouse-go/v2`), cùng các helper `IsMySQLMasterConnection`, `IsClickHouseMasterConnection`, `detectMasterEngineFromDSNOverride`, `normalizeMasterSQLDSN`, `buildMySQLMasterDSNFromFields`, `buildClickHouseMasterDSNFromFields`.
- **Logs / Test:**
  - Unit test `TestMasterBindingDestinationDetectionAndBackwardCompatibility` trong `centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go:108-144` (`EV-20261030T165717-67dd`).
- **Definition of Done (DoD):**
  - `GetMasterDB` / `ResolveMasterEngineType` phân biệt chính xác `postgresql`, `mysql`, `clickhouse`, `mongodb`; tách sang file riêng `connection_manager_sql_masters.go`.

### [x] TASK-02: Trình sinh & Thực thi DDL cho Master MySQL và Master ClickHouse (SRP Split)
- **Mô tả & Context `file:line`:**
  - Đối chiếu `centralized-data-service/internal/service/master/master_ddl_generator.go:341-390` và `centralized-data-service/internal/service/master/master_ddl_generator_mongo.go:1-310`.
  - Tạo 2 file đơn trách nhiệm mới:
    1. `centralized-data-service/internal/service/master/master_ddl_generator_mysql.go:1-386`: Sinh và thực thi `CREATE TABLE IF NOT EXISTS` (`ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`), ánh xạ kiểu dữ liệu chuẩn MySQL (`BIGINT`, `VARCHAR(255)`, `TEXT`, `DECIMAL(38, 10)`, `DATETIME(3)`, `JSON`, `TINYINT(1)`), `ALTER TABLE ... ADD COLUMN`, và tạo Unique/Secondary Index idempotent qua kiểm tra `information_schema.statistics`.
    2. `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:1-318`: Sinh và thực thi `CREATE TABLE IF NOT EXISTS <db>.<table> (...) ENGINE = ReplacingMergeTree(_transmuted_at) ORDER BY (_gpay_id)`, ánh xạ kiểu ClickHouse (`Int64`, `String`, `Float64`, `Decimal(38, 10)`, `DateTime64(3, 'UTC')`, `UInt8`, `Nullable(...)`), `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.
- **Logs / Test:**
  - Unit test `TestMasterMySQLAndClickHouseDDLGeneration` trong `centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go:14-81` (`EV-20261030T165641-35b8`).
- **Definition of Done (DoD):**
  - `MasterDDLGenerator.Generate` và `Apply` điều phối sạch sang `master_ddl_generator_mysql.go` và `master_ddl_generator_clickhouse.go` với chỉ vài dòng hook tại `master_ddl_generator.go`.

### [x] TASK-03: Bộ Chuyển đổi Kiểu & Luồng Ghi Batch Upsert (`transmuter_mysql.go` & `transmuter_clickhouse.go`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu `centralized-data-service/internal/service/master/transmuter_mongo.go:1-380` và `centralized-data-service/internal/service/master/transmuter.go:1568-1655`.
  - Tạo các file đơn trách nhiệm:
    1. `centralized-data-service/internal/service/master/transmuter_mysql.go:1-331`: Chia chunk an toàn dưới trần `60,000` placeholders của MySQL, thực thi `INSERT INTO <db>.<table> (...) VALUES (?, ...) AS incoming ON DUPLICATE KEY UPDATE col = IF(COALESCE(incoming._source_ts, 0) >= COALESCE(<table>._source_ts, 0) - 2000, incoming.col, <table>.col), _updated_at = IF(..., CURRENT_TIMESTAMP(3), <table>._updated_at)` và `DELETE FROM <db>.<table> WHERE _gpay_id IN (?)`.
    2. `centralized-data-service/internal/service/master/transmuter_clickhouse.go:1-345`: Pre-read OCC `SELECT _gpay_id, _source_ts FROM <db>.<table> FINAL WHERE _gpay_id IN (?)` (`FilterClickHouseOCCRecords` với dung sai `2000ms`), chuẩn hóa kiểu ClickHouse (`DateTime64(3, 'UTC')`, `_transmuted_at UInt64`), ghi lô qua `INSERT INTO <db>.<table>` cho `ReplacingMergeTree(_transmuted_at)` và xử lý xóa qua `ALTER TABLE <db>.<table> DELETE WHERE _gpay_id IN (...) SETTINGS mutations_sync = 1`.
- **Logs / Test:**
  - Unit test `TestFilterClickHouseOCCRecords` trong `centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go:83-106` (`EV-20261030T165712-00c5`).
- **Definition of Done (DoD):**
  - `transmuter.go` chỉ gọi điều phối 3 dòng sang `bulkUpsertMySQLMaster` / `bulkUpsertClickHouseMaster` và `deleteFromMySQLMasterBySourceAndGpayIDs` / `deleteFromClickHouseMasterBySourceAndGpayIDs`, không làm phình `transmuter.go`.

### [x] TASK-04: Mở rộng Index Governance (`index_driver_mysql.go` & `index_driver_clickhouse.go`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu `centralized-data-service/internal/service/governance/index_driver_registry.go:116-126` và `index_driver_mongo.go:1-290`.
  - Tạo 2 file đơn trách nhiệm:
    1. `centralized-data-service/internal/service/governance/index_driver_mysql.go:1-201`: Thực thi `IndexEngineDriver` (`Engine() = "mysql"`, `ListIndexes` từ `information_schema.statistics`, `CreateIndex` với `ALGORITHM=INPLACE, LOCK=NONE` fallback chuẩn, `DropIndex` qua `DROP INDEX ... ON <db>.<table>`).
    2. `centralized-data-service/internal/service/governance/index_driver_clickhouse.go:1-153`: Thực thi `IndexEngineDriver` (`Engine() = "clickhouse"`, `ListIndexes` từ `system.data_skipping_indices`, `CreateIndex` qua `ALTER TABLE <db>.<table> ADD INDEX ... TYPE minmax/set/bloom_filter GRANULARITY 1`, `DropIndex` qua `ALTER TABLE <db>.<table> DROP INDEX ...`).
  - Tại `index_driver_registry.go:122-123`: Chỉ thêm 2 dòng đăng ký `im.RegisterDriver(newMySQLIndexDriver(connMgr, logger))` và `im.RegisterDriver(newClickHouseIndexDriver(connMgr, logger))`.
- **Logs / Test:**
  - Kiểm chứng biên dịch toàn bộ package governance (`EV-20261030T165650-98d6`).
- **Definition of Done (DoD):**
  - `NewIndexManagerWithConn` trả về driver hợp lệ cho cả 4 engine, giữ nguyên 100% hành vi của `postgresql` và `mongodb`.

### [x] TASK-05: Mở rộng Đối soát Recon Segment B, Heal Segment B & SourceCountRunner (`recon_dest_sql_dialects.go` & `recon_execute_heal_segb_sql_dialects.go`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu `centralized-data-service/internal/service/recon/recon_engine.go:202-230`, `recon_dest_agent.go:15-125`, `recon_dest_query.go:16-708`, `recon_dest_hash.go:24-230`, `recon_window_index_probe.go:34-76`, `recon_tier_b.go:606-685,786-864`, `internal/handler/recon/recon_execute_heal_segb.go:100-210,350-388,683-703,877-882`, và `internal/service/sourcecount/source_count_runner.go:165-181,290-305`.
  - Tạo 2 file đơn trách nhiệm:
    1. `centralized-data-service/internal/service/recon/recon_dest_sql_dialects.go:1-420`: Cung cấp `IsMySQL()`, `IsClickHouse()`, `IsSQLDialect()`, và thực thi trọn vẹn các hàm đọc đối soát cho MySQL (backtick `` `db`.`table` ``, `CAST(pk AS CHAR)`, Go-side XOR bucketing giống `mongoBucketCounts`) và ClickHouse (`FINAL`, `toString(pk)`, không gọi `SET TRANSACTION READ ONLY`).
    2. `centralized-data-service/internal/handler/recon/recon_execute_heal_segb_sql_dialects.go:1-262`: Thực thi `pruneSQLDialectMasterMissingFromSrc` và `resolveSourceIDsFromSQLDialectMasterByGpayIDs` cho MySQL & ClickHouse.
- **Logs / Test:**
  - Kiểm chứng biên dịch toàn bộ package `internal/service/recon`, `internal/handler/recon`, `internal/service/sourcecount` (`EV-20261030T165650-98d6`).
- **Definition of Done (DoD):**
  - Toàn bộ luồng Recon Segment B, RowDiff B, TimeBoundedDiff B, Heal Segment B và SourceCountRunner chạy đúng dialect MySQL & ClickHouse mà không ảnh hưởng nhánh PostgreSQL/MongoDB.

### [x] TASK-06: Mở rộng CMS Backend (`cdc-cms-service`) cho Master MySQL & ClickHouse
- **Mô tả & Context `file:line`:**
  - Đối chiếu `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go:160-165,240-245,350-368,440-442`, `master_read_repo_gorm.go:33-43`, `internal/app/commands/master/create_master.go:176-199`, `internal/api/master/master_registry_handler_connections.go:80-101,187-220`, và `internal/api/master/master_registry_handler_connections_probe.go:22-280`.
  - Tạo file đơn trách nhiệm `cdc-cms-service/internal/api/master/master_registry_handler_connections_probe_sql.go:1-174` (chứa `normalizeMasterConnectionEngine`, `isDatabaseScopedMasterEngine`, `defaultMasterPortByEngine`, `inferMasterEngineFromDSN`, `buildMySQLMasterDSN`, `buildClickHouseMasterDSN`, `probeMySQLMasterConnection`, `probeClickHouseMasterConnection`).
  - Sửa lỗi nối `&connect_timeout=5` vào DSN không phải PostgreSQL tại `master_registry_handler_connections_probe.go:196` (tránh lỗi `Unknown system variable 'connect_timeout'` trên MySQL).
  - Áp dụng quy tắc Database-scoped (`master_schema = default_database`, `master_database = default_database`) cho cả `mongodb`, `mysql`, `clickhouse` tại `create_master.go:180`, `master_read_repo_gorm.go:34-42`, và `master_connection_repo_gorm.go:356`.
- **Logs / Test:**
  - Unit test `TestCreateMasterHandler_MySQLAndClickHouse_EnforcesDatabaseScopeAndPreservesPostgres` trong `cdc-cms-service/internal/app/commands/master/create_master_sql_engines_test.go:138-240` (`EV-20261030T165646-e9e4`) và `go build ./internal/... ./cmd/...` (`EV-20261030T165700-97a6`).
- **Definition of Done (DoD):**
  - `cdc-cms-service` nhận diện, lưu trữ, probe ping (trả về version MySQL / ClickHouse) và gắn đúng `default_database` cho cả 4 engine, tương thích ngược 100% với PostgreSQL và MongoDB.

### [x] TASK-07: Mở rộng CMS Frontend (`cdc-cms-web`) theo SRP (`masterEngineConfig.ts`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu `cdc-cms-web/src/pages/MasterRegistry.tsx:365-566,644-665,1645-1935` và `cdc-cms-web/src/components/master/CreateMasterModal.tsx:30-246`.
  - Tạo file đơn trách nhiệm `cdc-cms-web/src/components/master/masterEngineConfig.ts:1-86` khai báo tập trung metadata cho 4 engine (`postgresql`, `mongodb`, `mysql`, `clickhouse`).
  - Cập nhật gọn `MasterRegistry.tsx` và `CreateMasterModal.tsx` dùng helper từ `masterEngineConfig.ts` (giữ `wc -l` của `MasterRegistry.tsx` chỉ tăng 9 dòng import/wiring).
- **Logs / Test:**
  - Chạy `npx tsc --noEmit` trên `cdc-cms-web` đạt `0 errors` (`EV-20261030T165703-f720`).
- **Definition of Done (DoD):**
  - Giao diện cho phép tạo/sửa/test kết nối Master `PostgreSQL`, `MongoDB`, `MySQL`, `ClickHouse` và tạo Master Table/Collection với cơ chế khóa Database tự động cho `mongodb`/`mysql`/`clickhouse`.
