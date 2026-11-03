# 04. Architecture Decision Records (ADRs) & Decision Gate Resolutions — Master MySQL & ClickHouse

- **Workspace:** `master-mysql-clickhouse-2026-10-30`
- **Suffix:** `master_mysql_clickhouse`
- **Tier:** `T2`

---

## ADR-01: Tách file Dialect theo Single-Responsibility Principle (SRP) đối xứng 1–1 với Master MongoDB
- **Context:** Các file điều phối trung tâm (`pkgs/database/multi.go`, `internal/service/source/connection_manager.go`, `internal/worker/master_ddl_generator.go`, `internal/worker/transmuter.go`, `internal/worker/index_driver.go`, `internal/worker/recon_sql_helpers.go`, `master_connection_repo_gorm.go`, `MasterRegistry.tsx`) đã có kích thước từ `250` đến `582` dòng.
- **Decision:** Không nhồi thêm logic SQL dialect của MySQL hay ClickHouse vào các file trên. Tách mỗi engine thành các file/module độc lập (`multi_dialector.go`, `connection_manager_sql_engines.go`, `master_ddl_generator_mysql.go`, `master_ddl_generator_clickhouse.go`, `sql_dialect_type_converter.go`, `transmuter_mysql.go`, `transmuter_clickhouse.go`, `index_driver_mysql.go`, `index_driver_clickhouse.go`, `recon_dest_sql_dialects.go`, `master_connection_engine_dsn.go`, `masterEngineConfig.ts`) và giữ dispatcher chỉ 3–8 dòng switch/case.
- **References:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/worker/master_ddl_generator.go:68-100`
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/worker/transmuter.go:62-97`

---

## ADR-02: Bảo toàn cơ chế Optimistic Concurrency Control (OCC) `_source_ts - 2000` trên MySQL & ClickHouse
- **Context:** PostgreSQL dùng `ON CONFLICT (...) DO UPDATE ... WHERE EXCLUDED._source_ts >= tbl._source_ts - 2000`, còn MongoDB dùng aggregation pipeline `$cond` trong `UpdateOneModel`. MySQL (`ON DUPLICATE KEY UPDATE`) không có mệnh đề `WHERE` ở tầng câu lệnh, còn ClickHouse (`ReplacingMergeTree`) là append-only columnar engine.
- **Decision:**
  1. **MySQL (InnoDB / MariaDB):** Thực thi ở tầng biểu thức cột bằng `IF(COALESCE(VALUES(_source_ts), 0) >= COALESCE(tbl._source_ts, 0) - 2000, VALUES(col), tbl.col)` trong `INSERT INTO ... ON DUPLICATE KEY UPDATE`, đồng thời tự động chia chunk batch để tổng số `?` không vượt quá `60,000` (`< 65,535` giới hạn `uint16` của MySQL prepared statement).
  2. **ClickHouse (`ReplacingMergeTree(_source_ts)`):** Thực hiện Pre-fetch OCC Guard (`SELECT <conflict_cols>, max(_source_ts) FROM <db>.<table> FINAL WHERE ... GROUP BY ...`) để loại bỏ các event cũ có `_source_ts < existing_ts - 2000` trước khi `INSERT` lô mới, kết hợp `ReplacingMergeTree(_source_ts)` tự động giữ bản ghi có `_source_ts` lớn nhất khi truy vấn `SELECT ... FINAL` hoặc sau background merge.
- **References:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/worker/transmuter.go:196-206`
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/worker/transmuter_mongo.go:322-375`

---

## Decision Gate Resolutions (`DG0` & `DG1`)

### DG0 Resolutions (`06_laya_summary_dg0_20261030T153817.json`)
- **`C-REALITY` (`FLAG` -> `RESOLVED`):** Verified directly in `multi.go`: `openGorm` calls `gorm.Open(postgres.Open(dsn), ...)`. `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/database/multi.go:205-216`
- **`C-CONTRA` (`FLAG` -> `RESOLVED`):** Verified directly in `master_connection_repo_gorm.go`: `CreateMasterConnection` only checks `mongo`/`mongodb` and defaults all others to `postgresql`. `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go:160-165`
- [2026-10-30T15:42:25+07:00] [Agent:laya_gate] DG1 S-RC FLAG resolved: Verified directly in connection_manager.go: GetMasterDB calls m.reg.OpenGorm(dsn, 'master:'+key). REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:195-248
- [2026-10-30T15:42:25+07:00] [Agent:laya_gate] DG1 D-EDGE FLAG resolved: Verified directly in 09_tasks_solution_master_mysql_clickhouse.md: FilterClickHouseOCCRecords checks !isFullSync && prevTs > incomingTs+clickhouseOCCClockSkewToleranceMs and increments occSkipped. REF: /Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/09_tasks_solution_master_mysql_clickhouse.md:306-346
- [2026-10-30T15:42:26+07:00] [Agent:laya_gate] DG2 D-EDGE FLAG resolved: Verified directly in 09_tasks_solution_master_mysql_clickhouse.md: FilterClickHouseOCCRecords checks !isFullSync && prevTs > incomingTs+clickhouseOCCClockSkewToleranceMs and increments occSkipped. REF: /Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/09_tasks_solution_master_mysql_clickhouse.md:306-346
- [2026-10-30T15:43:19+07:00] [Agent:laya_gate] DG1 S-FIX FLAG resolved: Verified directly in master_ddl_generator.go: Apply branches on MongoDB to applyMongoCollectionAndIndexes, otherwise calls GetMasterDB. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:352-357
- [2026-10-30T15:43:36+07:00] [Agent:laya_gate] DG2 S-RC FLAG resolved: Verified directly in connection_manager.go: GetMasterDB calls m.reg.OpenGorm(dsn, 'master:'+key). REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:195-248
- [2026-10-30T15:43:36+07:00] [Agent:laya_gate] DG2 S-FIX FLAG resolved: Verified directly in master_ddl_generator.go: Apply branches on MongoDB to applyMongoCollectionAndIndexes, otherwise calls GetMasterDB. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:352-357
- [2026-10-30T16:27:59+07:00] [Agent:laya_gate] DG1 S-RC FLAG resolved: Verified directly in connection_manager.go: GetMasterDB calls m.reg.OpenGorm(dsn, 'master:'+key). REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:399-406
- [2026-10-30T16:27:59+07:00] [Agent:laya_gate] DG1 S-FIX FLAG resolved: Verified directly in master_ddl_generator.go: Apply branches on MongoDB to applyMongoCollectionAndIndexes, otherwise calls GetMasterDB. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:352-357
- [2026-10-30T16:27:59+07:00] [Agent:laya_gate] DG1 S-RECON-TX FLAG resolved: Verified directly in recon_dest_agent.go: readOnlyDB calls da.replica.WithContext(ctx).Begin() and tx.Exec('SET TRANSACTION READ ONLY'). REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go:121-125
- [2026-10-30T16:27:59+07:00] [Agent:laya_gate] DG1 S-CMS-DB-SCOPE FLAG resolved: Verified directly in create_master.go: when isMongoConn is true, Handle sets masterSchema = mongoDB and masterDatabase = &mongoDB. REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:176-191
- [2026-10-30T16:27:59+07:00] [Agent:laya_gate] DG1 D-EDGE FLAG resolved: Verified directly in 09_tasks_solution_master_mysql_clickhouse.md: FilterClickHouseOCCRecords checks !isFullSync && prevTs > incomingTs+clickhouseOCCClockSkewToleranceMs and increments occSkipped. REF: /Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/09_tasks_solution_master_mysql_clickhouse.md:613-651
- [2026-10-30T16:33:38+07:00] [Agent:laya_gate] DG2 S-RC FLAG resolved: Verified call sites REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:338-401
- [2026-10-30T16:33:41+07:00] [Agent:laya_gate] DG2 S-FIX FLAG resolved: Verified call sites REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:277-322
- [2026-10-30T16:33:44+07:00] [Agent:laya_gate] DG2 S-RECON-TX FLAG resolved: Verified call sites REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go:121-125
- [2026-10-30T16:33:48+07:00] [Agent:laya_gate] DG2 S-CMS-DB-SCOPE FLAG resolved: Verified call sites REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:176-199
- [2026-10-30T16:33:50+07:00] [Agent:laya_gate] DG2 D-EDGE FLAG resolved: Verified call sites REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go:166-181
- [2026-10-30T17:01:33+07:00] [Agent:laya_gate] DG3 A-01 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:402-409 where GetMasterDB calls normalizeMasterSQLDSN and OpenGormWithEngine.
- [2026-10-30T17:01:35+07:00] [Agent:laya_gate] DG3 A-02 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_registry.go:121-125 where NewIndexManagerWithConn registers all four drivers.
- [2026-10-30T17:01:39+07:00] [Agent:laya_gate] DG3 A-03 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:180-195 where isDBScopedConn sets masterSchema and masterDatabase to scopedDB.
- [2026-10-30T17:12:17+07:00] [Agent:laya_gate] DG3 A-01 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:402-409 where GetMasterDB calls normalizeMasterSQLDSN and OpenGormWithEngine.
- [2026-10-30T17:12:19+07:00] [Agent:laya_gate] DG3 A-02 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_registry.go:121-125 where NewIndexManagerWithConn registers all four drivers.
- [2026-10-30T17:12:23+07:00] [Agent:laya_gate] DG3 A-03 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:180-195 where isDBScopedConn sets masterSchema and masterDatabase to scopedDB.
- [2026-10-30T17:12:25+07:00] [Agent:laya_gate] DG3 A-04 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:362-368 where Apply dispatches MySQL and ClickHouse bindings.
- [2026-10-30T17:12:29+07:00] [Agent:laya_gate] DG3 A-05 FLAG resolved: Verified in REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1636-1643 where bulkUpsertMaster routes MySQL and ClickHouse bindings.
- [2026-10-30T17:33:50+07:00] [Agent:laya_gate] DG3 A-01 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:402-409
- [2026-10-30T17:33:54+07:00] [Agent:laya_gate] DG3 A-02 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_registry.go:116-126
- [2026-10-30T17:33:56+07:00] [Agent:laya_gate] DG3 A-03 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:180-195
- [2026-10-30T17:34:00+07:00] [Agent:laya_gate] DG3 A-04 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:118-127
- [2026-10-30T17:34:03+07:00] [Agent:laya_gate] DG3 A-05 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:612-624
