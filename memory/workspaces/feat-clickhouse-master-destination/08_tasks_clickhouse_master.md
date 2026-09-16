# 08 Task Breakdown: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`  
> **Phân loại**: [Vận hành CMS] vs [Code thêm / Development] (Tuân thủ Lesson #2026-08-28)

---

## Danh mục Task Chi tiết

### Phase 1: Hạ tầng Docker & ClickHouse Client Driver

- [ ] `TASK-CH-01` **[Code thêm]** Thêm service `clickhouse` vào `docker/docker-compose.yml` (Image `clickhouse/clickhouse-server:24.3-alpine`, port 9000 & 8123, env `CLICKHOUSE_DB=cdc_master_dw`).
- [ ] `TASK-CH-02` **[Code thêm]** Tạo package `centralized-data-service/pkgs/clickhouse/client.go`: Khởi tạo pool `clickhouse.Open` với cấu hình retry, timeouts, max connections và Otel tracing.
- [ ] `TASK-CH-03` **[Code thêm]** Cập nhật `config.go`: Thêm struct `ClickHouseConfig` vào `AppConfig`.

### Phase 2: DDL Generator cho ClickHouse Master

- [ ] `TASK-CH-04` **[Code thêm]** Tạo `centralized-data-service/internal/service/master/clickhouse_ddl_generator.go`:
  - Mapping kiểu dữ liệu từ `mappingRule` sang ClickHouse types (`Int64`, `Decimal`, `DateTime64`, `Nullable`).
  - Sinh câu lệnh `CREATE TABLE IF NOT EXISTS ... ENGINE = ReplacingMergeTree(_version, _deleted) ORDER BY (_gpay_id)`.
- [ ] `TASK-CH-05` **[Code thêm]** Cập nhật `MasterDDLGenerator`: Phân nhánh gọi ClickHouse DDL Generator khi `master_connection.engine_type == "clickhouse"`.

### Phase 3: Transmuter Engine Hỗ trợ ClickHouse

- [ ] `TASK-CH-06` **[Code thêm]** Cập nhật `ConnectionManager` trong `internal/service/source/connection_manager.go`: Thêm hàm `GetMasterClickHouseConn(ctx, key) (driver.Conn, error)`.
- [ ] `TASK-CH-07` **[Code thêm]** Tạo `internal/service/master/transmuter_clickhouse.go`:
  - Hiện thực `bulkInsertClickHouse` sử dụng `conn.PrepareBatch` và batch append.
  - Hiện thực `softDeleteClickHouse` bằng cách chèn dòng có `_deleted = 1` và `_version = source_ts`.
- [ ] `TASK-CH-08` **[Code thêm]** Phân nhánh trong `TransmuterModule.processBatch`: Nếu `binding.EngineType == "clickhouse"` thì gọi `bulkInsertClickHouse` thay cho `bulkUpsertMaster` của PostgreSQL.

### Phase 4: Reconciliation & Testing

- [ ] `TASK-CH-09` **[Code thêm]** Mở rộng `internal/service/recon/recon_heal_fetch.go`: Hỗ trợ fetch hash từ ClickHouse qua clause `FINAL` để so sánh với PostgreSQL Shadow.
- [ ] `TASK-CH-10` **[Vận hành CMS]** Đăng ký Connection Registry ClickHouse và tạo Master Binding trên giao diện CMS, kích hoạt Transmute và kiểm tra đối soát.
