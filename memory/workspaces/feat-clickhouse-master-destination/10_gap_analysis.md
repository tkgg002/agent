# 10 Gap Analysis: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## Bảng Đối chiếu Năng lực (Gap Assessment)

| Thành phần | Năng lực Hiện có | Yêu cầu Kỹ thuật Mới | Gap & Hạng mục Phát triển |
|:---|:---|:---|:---|
| **Control Plane Schema** | `connection_registry` đã có `engine_type: 'clickhouse'` (migration 029/087) | Đã có sẵn | Không có gap DDL DB. |
| **Go Driver** | `clickhouse-go/v2` đã khai báo trong `go.mod` | Cần package wrapper | **Gap 1**: Thiếu package `pkgs/clickhouse/client.go`. |
| **Connection Manager** | `GetMasterDB` chỉ trả về `*gorm.DB` PostgreSQL | Cần trả về `driver.Conn` ClickHouse | **Gap 2**: Mở rộng `ConnectionManager` hỗ trợ ClickHouse connection pool. |
| **Master DDL Generator** | `MasterDDLGenerator` chỉ sinh PostgreSQL DDL (`CREATE TABLE ... ON CONFLICT`) | Cần sinh DDL ClickHouse `ReplacingMergeTree` | **Gap 3**: Tạo `clickhouse_ddl_generator.go`. |
| **Transmuter Ingestion** | Transmuter chỉ chạy `bulkUpsertMaster` với PostgreSQL `ON CONFLICT` | Cần batch insert vào ClickHouse qua `PrepareBatch` | **Gap 4**: Tạo `transmuter_clickhouse.go` với logic Append + Soft-delete. |
| **Recon Engine** | Chỉ query Postgres Master để tính hash | Cần query ClickHouse `FINAL` | **Gap 5**: Bổ sung hàm tính hash trong `recon_heal_fetch.go`. |
| **Docker Compose** | Chưa có container ClickHouse chạy local | Cần ClickHouse service | **Gap 6**: Thêm container `clickhouse/clickhouse-server` vào `docker/docker-compose.yml`. |
