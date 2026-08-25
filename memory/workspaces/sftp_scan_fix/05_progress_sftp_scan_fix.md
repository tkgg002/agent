# 05_progress_sftp_scan_fix.md (Audit Log)

## [2026-08-11T14:52:00Z] [Brain:Gemini-3.6-Flash] Root Cause Analysis & Task Initialization

- **Root Cause Analysis**:
  - `centralized-data-service/internal/handler/source/discover_handler.go`: Hàm `ScanFieldsDebezium` kiểm tra sự tồn tại của cột `_raw_data` trong Shadow Table. Nếu bảng rỗng hoặc chưa tạo, code gọi `isSQLSource(sourceType)`.
  - Hàm `isSQLSource(sourceType)` trong `discover_handler_sql.go` chỉ check `postgres/mysql/mariadb`. Với `sourceType == "sftp"`, `isSQLSource` trả về `false`.
  - Nhưng do thiếu case guard cho `sftp`, code ở line 215 rơi xuống `scanFieldsSQLSource` hoặc báo lỗi "has no _raw_data column" / "SQL source returned 0 columns or connection failed".
  - Ngoài ra, `InferSourceColumns` trong `discovery_utils.go` thiếu case `engine == "sftp"`.
- **Action**: Lập kế hoạch chi tiết, khởi tạo tài liệu Workspace `sftp_scan_fix`.

## [2026-08-11T14:56:00Z] [Muscle:Go-Engineer] Execution & Verification Completed

- **Changes Applied**:
  - `discover_handler.go`: Bổ sung helper `isFileOrStreamSource(sourceType)` bẫy các nguồn `sftp`, `file`, `csv`, `json`, `kafka`. Ngăn không cho gọi `scanFieldsSQLSource` sai lầm khi Shadow Table chưa có dữ liệu.
  - `./docker/data/reconcile_final/reconcile_final_20260811.csv`: Nạp file CSV mẫu vào thư mục SFTP input.
- **Verification Result**:
  - Unit tests `go test -v ./internal/handler/source/...` PASS 100%.

## [2026-08-11T16:00:00Z] [Muscle:FullStack-Engineer] Fix SFTP Collection Name Extraction

- **Fix Applied**:
  - `cdc-cms-service/internal/api/source/system_connectors_handler.go`: Viết helper `extractSFTPCollectionName(cfg)` trích xuất tên collection sạch từ `topic` (ví dụ `cdc.sftplocal.reconcile.final` -> `reconcile_final`) thay vì lấy trực tiếp chuỗi RegEx `policy.regexp` (`^reconcile_final_.*\.csv$`).
  - `cdc-cms-web/src/pages/TableRegistry.tsx`: Bổ sung `showSearch` & `allowClear` cho Select component để người dùng tìm kiếm và chọn tên sạch.
- **Verification Result**:
  - Backend compile `go build ./cmd/server` PASS 100%.
  - Frontend TypeScript check `npx tsc --noEmit` PASS 100%.

## [2026-08-11T16:19:00Z] [Muscle:Go-Engineer] Auto-Create Kafka Topic for SFTP Connectors

- **Changes Applied**:
  - `debezium_connector.go`: Thêm helper `autoCreateKafkaTopic` trong `CreateSystemConnectorHandler`. Tự động tạo Kafka Topic (`cdc.sftplocal.<collection>`) trên Kafka Broker ngay khi đăng ký Connector thành công.
- **Verification Result**:
  - Backend `go build ./cmd/server` PASS 100%.

## [2026-08-12T13:17:00Z] [Muscle:Go-Engineer] Fix Dynamic Routing & Verify E2E SFTP CDC Sync

- **Root Cause of Sync Failure**:
  - SFTP events had flat JSON structure.
  - While flat record parsing was added, when `ResolveSourceRoutes` was called, it fell back to key `"reconcile_final"` which was registered by both the PostgreSQL shadow table source (using primary key `"transaction_id"`) and SFTP connectors.
  - Since the PostgreSQL route came first, the worker assumed `pkField == "transaction_id"`, which did not exist in the SFTP CSV flat schema, causing the events to be skipped with `"event missing PK, skipping routes"` warning.
- **Changes Applied**:
  - `metadata_registry_utils.go`: Modified `buildRouteLookupKeys` to generate colon-delimited lookup keys (e.g. `testsftp12:reconcile_final`) supporting connection code prefixing.
  - `event_handler.go`: Dynamically extracted the connector name (connection code) from SFTP Kafka topic to act as `db` (e.g., `testsftp12`) instead of hardcoding `"sftp"`.
- **Verification Result**:
  - All 162 records from topic `cdc.sftplocal.testsftp12.reconcile_final` were successfully consumed, primary key resolved correctly as `id`, and rows successfully upserted into 5 distinct records (due to uniqueness constraint of `id` in CSV seed files) in `shadow_testsftp12.reconcile_final`.
