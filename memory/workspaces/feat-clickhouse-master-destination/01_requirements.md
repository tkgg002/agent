# 01 Requirements: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## 1. Yêu cầu Chức năng (Functional Requirements)

### FR-01: Quản lý Kết nối ClickHouse (Connection Management)
- Cho phép đăng ký kết nối ClickHouse trong `cdc_system.connection_registry` với `role_type: 'master'`, `engine_type: 'clickhouse'`.
- Hỗ trợ kết nối Native Protocol (port `9000`) qua driver `github.com/ClickHouse/clickhouse-go/v2`.
- Cấu hình qua YAML (`config-local.yml` / `config-production.yml`) hoặc dynamic resolution từ DB.

### FR-02: Sinh DDL Bảng Master trên ClickHouse (DDL Generator)
- Tự động sinh cú pháp DDL chuẩn ClickHouse dựa trên các `mapping_rule`:
  ```sql
  CREATE TABLE IF NOT EXISTS <db>.<table> (
      _gpay_id Int64,
      _source_id String,
      _source_ts DateTime64(3, 'UTC'),
      _deleted UInt8 DEFAULT 0,
      _version UInt64,
      <column_name> <ClickHouse_Type>,
      ...
  ) ENGINE = ReplacingMergeTree(_version, _deleted)
  ORDER BY (_gpay_id)
  PRIMARY KEY (_gpay_id)
  SETTINGS index_granularity = 8192;
  ```
- Chuyển đổi chuẩn xác kiểu dữ liệu:
  * `BIGINT` $\rightarrow$ `Int64`
  * `INT` $\rightarrow$ `Int32`
  * `NUMERIC / DECIMAL(P, S)` $\rightarrow$ `Decimal(P, S)`
  * `TIMESTAMPTZ` $\rightarrow$ `DateTime64(3, 'UTC')`
  * `VARCHAR / TEXT` $\rightarrow$ `String`
  * `BOOLEAN` $\rightarrow$ `Bool` (hoặc `UInt8`)
  * Cột nullable $\rightarrow$ `Nullable(T)`

### FR-03: Transmuter Batch Write sang ClickHouse (Ingestion Engine)
- Thay thế cú pháp `INSERT ... ON CONFLICT` (vốn không được ClickHouse hỗ trợ) bằng mô hình Append Batch Ingestion:
  * Mỗi batch sử dụng `PrepareBatch` của `clickhouse-go/v2` ghi block 10,000 - 50,000 records.
  * `_version` gán bằng `_source_ts` (epoch ms) để `ReplacingMergeTree` tự động deduplicate.
  * Thao tác Delete: Không chạy `ALTER TABLE DELETE` (mutation nặng), mà ghi dòng mới với `_deleted = 1` và `_version = source_ts`.

### FR-04: Đối soát Dữ liệu (Reconciliation for ClickHouse)
- Hỗ trợ truy vấn đối soát trên ClickHouse qua clause `FINAL`:
  `SELECT count(), sum(cityHash64(col1, col2, ...)) FROM <db>.<table> FINAL WHERE _deleted = 0`.
- So sánh hash với PostgreSQL Shadow Table để phát hiện và heal drift.

---

## 2. Yêu cầu Phi chức năng (Non-Functional Requirements)

- **NFR-01 (Throughput)**: Tốc độ insert ClickHouse đạt $\ge 50,000$ rows/s khi Transmuter xả batch.
- **NFR-02 (Zero Disruption)**: Không phá vỡ luồng Transmute sang PostgreSQL Master hiện hữu. ConnectionManager phân nhánh theo `engine_type`.
- **NFR-03 (Observability)**: Tích hợp OpenTelemetry tracing cho các thao tác ClickHouse batch write và DDL execution.
