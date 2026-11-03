# 08 Tasks: Audit & Fix Scan Fields PostgreSQL (`scan_fields_postgres`)

- **Workspace**: `audit-scan-fields-postgres-2026-10-28`
- **Suffix**: `scan_fields_postgres`
- **Tier**: `T2`

---

- [x] **TASK-01 — Truy vết toàn trình hàm xử lý `cmd-scan-fields` / `scan-fields` cho PostgreSQL (`S2 Investigate`)**
  - **Context (`file:line`)**:
    - `centralized-data-service/internal/handler/source/discover_handler.go:195-328`
    - `centralized-data-service/internal/handler/source/discover_handler_sql.go:13-99`
    - `centralized-data-service/internal/handler/source/discovery_utils.go:112-240`
    - `cdc-cms-service/internal/api/source/system_connectors_handler.go:245, 567-612`
  - **Logs/Test**: Đối chiếu activity log `16:30:15 28/10/2026 cmd-scan-fields / scan-fields` (`id=134184, 134185`) và kiểm tra `cdc_system.connection_registry` (`id=407`).
  - **Definition of Done (DoD)**: Chỉ rõ `file:line` của toàn bộ callchain và 4 phát hiện gốc rễ (`F1`–`F4`, `VERIFIED`).

- [x] **TASK-02 — Thiết kế giải pháp Detect DDL trực tiếp cho PostgreSQL (`S2.5 DG0` & `S3 Solution + DG1/DG2`)**
  - **Context (`file:line`)**: `09_tasks_solution_scan_fields_postgres.md`.
  - **Logs/Test**: Chạy `laya_gate.py run --stage dg0` và `laya_gate.py run --stage dg1`.
  - **Definition of Done (DoD)**: Hoàn thành `09_tasks_solution_scan_fields_postgres.md` kèm Code Demo, đạt `DG0 PASS` và `DG1+DG2 PASS`, trình bày trên chat và dừng chờ `APPROVE`.

- [x] **TASK-03 — Chặn ghi đè mật khẩu `"__KEEP__"` / `"***"` vào `connection_registry.options_json` (`Fix F2`)**
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go:567-612`
  - **Logs/Test**: `go test -v ./internal/api/source/...`
  - **Definition of Done (DoD)**: Khi Edit Connector trên CMS mà không nhập lại mật khẩu (`"__KEEP__"`), `extractCredentialsAsOptions` không ghi đè mật khẩu thật trong `cdc_system.connection_registry.options_json`.

- [x] **TASK-04 — Rẽ nhánh Detect DDL trực tiếp cho SQL Source trong `ScanFieldsDebezium` (`Fix F1 & F4`)**
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler.go:195-244`
  - **Logs/Test**: `go test -v ./internal/handler/source/...`
  - **Definition of Done (DoD)**: Với mọi nguồn SQL (`postgres`, `postgresql`, `pg`, `mysql`, `mariadb`), `ScanFieldsDebezium` gọi thẳng `scanFieldsSQLSource` để detect DDL mà không quét `_raw_data`.

- [x] **TASK-05 — Chuẩn hóa `inferPGCols` (`schema.table`, fallback `public`, `udt_name` mảng, trả lỗi tường minh) (`Fix F3`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discovery_utils.go:112-240, 368-386`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler_sql.go:35-42`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go:560-585`
  - **Logs/Test**: `go test -v ./internal/handler/source/... ./internal/service/source/...`
  - **Definition of Done (DoD)**: `inferPGCols` đọc đầy đủ `column_name, data_type, udt_name, is_nullable`, tự tách `schema.table`, fallback sang `public` nếu `source_schema` bị lệch, và trả về lỗi kết nối/SQL rõ ràng.

- [x] **TASK-06 — Chuẩn hóa UI `Create Master` Modal cho MongoDB (`1 Connection = 1 Database`, đổi label `Database / Collection`, auto-select & lock Database) (`Fix F5`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx:291-360, 548-626, 809-839, 938-951, 1258-1278, 1295-1456`
  - **Logs/Test**: `npm run build` trong `cdc-cms-web` + kiểm tra UI modal `Create Master` khi chọn kết nối `mongo_local` (`engine_type: 'mongodb'`) vs `default_master` (`engine_type: 'postgresql'`).
  - **Definition of Done (DoD)**:
    1. Khi chọn `Target Database Connection` là MongoDB (`isMongoMasterTarget === true`), đổi label sang **`Master Database (MongoDB)`** và **`Master Collection Name`**, đổi tiêu đề modal sang **`Create Master Collection (MongoDB)`**.
    2. Tự động gán `form.master_schema = selectedMasterConn.default_database` (ví dụ `devtest`) và **khóa không cho nhập tay (`disabled={true}`)**.
    3. Khi chọn `Shadow Table` trong lúc kết nối đích là MongoDB, giữ nguyên `master_schema = selectedMasterConn.default_database` (không ghi đè bằng `normalizeMasterSchema(b?.source_db)`). Khi chuyển ngược về kết nối PostgreSQL, mở khóa (`disabled={false}`) và tự động điền lại schema gợi ý `master_<source_db>`.

- [x] **TASK-07 — Cưỡng chế `1 Kết nối Master Mongo = 1 Database` ở Backend `cdc-cms-service` (`CreateMasterHandler`, `ListEnriched`, `UpdateMasterConnection`) (`Fix F6`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go:44-60, 150-228`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler.go:27-31`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:532-577, 750-852`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go:27-96`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/master/list_masters.go:15-53`
  - **Logs/Test**: `go test -v ./internal/app/commands/master/... ./internal/api/master/... ./internal/infra/persistence/master/...`
  - **Definition of Done (DoD)**:
    1. Khi tạo Master (`POST /api/v1/masters`) với kết nối đích `engine_type == 'mongodb'`, backend tự động cưỡng chế `master_schema = default_database` và `master_database = default_database` từ `cdc_system.connection_registry` (bất kể client gửi gì lên), đồng thời cho phép tên database/collection chứa dấu gạch ngang `-` hợp lệ của MongoDB.
    2. `GET /api/v1/masters` (`ListEnriched`) trả thêm `master_engine_type` và khi `UpdateMasterConnection` cập nhật `default_database` của kết nối MongoDB thì tự động đồng bộ `master_database`/`master_schema` của các `master_binding` thuộc kết nối đó.

- [x] **TASK-08 — Đồng bộ `master_database` & `master_schema` trong `MasterDDLGenerator.applyMongoCollectionAndIndexes` (`centralized-data-service`) (`Fix F7`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:480-582`
  - **Logs/Test**: `go test -v ./internal/service/master/...`
  - **Definition of Done (DoD)**: Khi `applyMongoCollectionAndIndexes` tạo collection và indexes trên database `dbName` (`default_database` của kết nối MongoDB), tự động đồng bộ `master_database = dbName`, `master_schema = dbName`, `physical_table_fqn = dbName + "." + reg.MasterTable` trên `cdc_system.master_binding` để UI và MongoDB khớp nhau 100%.

- [x] **TASK-BUG-09 — Tách file theo nguyên tắc Trách nhiệm Đơn (`Single-Responsibility`) và đảm bảo `NOW <= HEAD` trên toàn bộ 6 file lớn (`Fix F8` — `L006`)**
  - **Context (`file:line`)**:
    - `cdc-cms-web/src/pages/MasterRegistry.tsx` (`2177` dòng -> tách `CreateMasterModal.tsx` sang `cdc-cms-web/src/components/master/CreateMasterModal.tsx` để `MasterRegistry.tsx <= 2019` dòng)
    - `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go` (`1135` dòng -> tách Master Connection CRUD & `syncMongoMasterBindingsDatabase` sang `master_connection_repo_gorm.go` để `master_repo_gorm.go <= 956` dòng)
    - `cdc-cms-service/internal/api/source/system_connectors_handler.go` (`773` dòng -> tách `extractCredentialsAsOptions`, `isKeepOrMaskedSecret`, `verifyPostgresReplicationPreflight`, `injectDebeziumSignalDefaults` sang `system_connectors_credentials.go` để `system_connectors_handler.go <= 701` dòng)
    - `centralized-data-service/internal/service/master/master_ddl_generator.go` (`935` dòng -> tách `applyMongoCollectionAndIndexes` sang `master_ddl_generator_mongo.go` để `master_ddl_generator.go <= 750` dòng)
    - `centralized-data-service/internal/handler/source/discovery_utils.go` (`488` dòng -> tách `inferPGCols`, `inferMySQLCols`, `pgSafeTypeWithUDT`, `pgSafeType`, `mysqlToPGType` sang `discovery_utils_sql.go` để `discovery_utils.go <= 446` dòng)
    - `centralized-data-service/internal/service/source/metadata_registry_service.go` (`739` dòng -> tách `GetSourceDSN`, `resolveSourceURIFromConn`, `buildDSNFromFieldsPatched` sang `metadata_registry_dsn.go` để `metadata_registry_service.go <= 735` dòng)
  - **Logs/Test**: Đo `wc -l` so với `HEAD` + chạy `go test` & `npm run build` qua `evidence_run.py`.
  - **Definition of Done (DoD)**: Tất cả 6 file lớn đều đạt `NOW <= HEAD` và mỗi file mới có đúng 1 trách nhiệm đơn.

- [x] **TASK-BUG-10 — Đặt đúng Layer Repository cho truy vấn đồng bộ `cdc_system.master_binding` (`Fix F9` — Strict Layer Separation)**
  - **Context (`file:line`)**:
    - `centralized-data-service/internal/repository/master/sync_runtime_state_repo.go:11-50`
    - `centralized-data-service/internal/service/master/master_ddl_generator_mongo.go`
  - **Logs/Test**: `go test -v ./internal/service/master/...`
  - **Definition of Done (DoD)**: Chuyển câu lệnh SQL `UPDATE cdc_system.master_binding SET master_database = ?, master_schema = ?, physical_table_fqn = ?` từ tầng Service (`MasterDDLGenerator`) về phương thức `SyncMasterBindingDatabase` tại tầng Repository (`SyncRuntimeStateRepo`).

- [x] **TASK-BUG-11 — Xóa bỏ Mockup (`fakeMasterRepoForCreate`) và Kiểm thử Tích hợp Thật trên PostgreSQL `:5433` & `10.200.187.110:5432` (`Fix F10` — `L016` & `L019`)**
  - **Context (`file:line`)**:
    - `cdc-cms-service/internal/app/commands/master/update_column_metadata_test.go:162-238`
    - `cdc-cms-service/internal/app/commands/master/create_master_mongo_real_test.go`
    - `centralized-data-service/internal/handler/source/discover_handler_live_pg_test.go`
  - **Logs/Test**: Chạy integration test thật qua `evidence_run.py --kind integration` & `--kind runtime`.
  - **Definition of Done (DoD)**: Xóa hoàn toàn `fakeMasterRepoForCreate`; kiểm chứng `CreateMasterHandler` trên PostgreSQL `:5433` thật (rollback transaction) và kiểm chứng `ScanFieldsDebezium` / `InferSourceColumnsWithError` trực tiếp trên nguồn PostgreSQL `10.200.187.110:5432` (`source_object_id = 214`, `shadow_binding_id = 219`).
