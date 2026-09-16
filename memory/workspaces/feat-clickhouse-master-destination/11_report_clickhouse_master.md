# 11 Change Overview Report: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## 1. Bảng Kê Khai Thay Đổi Mã Nguồn (Code Changes Matrix)

| Đường dẫn File | Trạng thái | Số dòng (+/-) | Mô tả Thay đổi & Kiến trúc |
|:---|:---:|:---:|:---|
| `docker/docker-compose.yml` | **MODIFY** | +20 / -0 | Thêm service `clickhouse` (image `clickhouse/clickhouse-server:24.3-alpine`). |
| `centralized-data-service/pkgs/clickhouse/client.go` | **NEW** | +65 / -0 | Package khởi tạo connection pool ClickHouse Native TCP qua `clickhouse-go/v2`. |
| `centralized-data-service/config/config.go` | **MODIFY** | +15 / -0 | Bổ sung `ClickHouseConfig` vào struct `AppConfig`. |
| `centralized-data-service/internal/service/source/connection_manager.go` | **MODIFY** | +25 / -0 | Bổ sung `GetMasterClickHouseConn(ctx, key) (driver.Conn, error)`. |
| `centralized-data-service/internal/service/master/clickhouse_ddl_generator.go` | **NEW** | +80 / -0 | Generator sinh DDL ClickHouse chuẩn `ReplacingMergeTree(_version, _deleted)`. |
| `centralized-data-service/internal/service/master/transmuter_clickhouse.go` | **NEW** | +110 / -0 | Module thực thi `bulkInsertClickHouse` và `softDeleteClickHouse`. |
| `centralized-data-service/internal/service/master/transmuter.go` | **MODIFY** | +30 / -5 | Phân nhánh xử lý theo `binding.EngineType` trong `processBatch`. |
| `centralized-data-service/internal/service/recon/recon_heal_fetch.go` | **MODIFY** | +35 / -0 | Thêm logic query tính hash đối soát trên ClickHouse `FINAL`. |
| `centralized-data-service/test/internal/service/clickhouse_master_test.go` | **NEW** | +120 / -0 | Test suite kiểm tra DDL generation, batch insert, và deduplication. |

---

## 2. Tổng quan Tác động (Blast Radius Assessment)
- **Zero Regression**: Hoàn toàn không sửa đổi luồng ghi sang PostgreSQL Master hiện hữu. Tầng `ConnectionManager` và `Transmuter` chỉ mở rộng thêm nhánh `case "clickhouse"` khi `engine_type == "clickhouse"`.
- **Hiệu năng**: ClickHouse batch append độc lập, không tiêu tốn connection pool hay tài nguyên CPU của PostgreSQL Master DW.
