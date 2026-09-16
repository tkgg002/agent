# 02 Plan: Kế hoạch Nâng cấp Master Tier Hỗ trợ ClickHouse

> **Workspace**: `feat-clickhouse-master-destination`

---

## Roadmap Triển khai 4 Giai đoạn

```mermaid
gantt
    title Roadmap Tích hợp ClickHouse Master Tier
    dateFormat  YYYY-MM-DD
    section Phase 1: Hạ tầng & Driver
    Thêm ClickHouse Service vào docker-compose (image 24.3-alpine) :p1_1, 2026-09-11, 1d
    Tạo package pkgs/clickhouse/client.go (clickhouse-go/v2)       :p1_2, after p1_1, 1d
    section Phase 2: DDL Generator
    Phát triển ClickHouse DDL Generator (ReplacingMergeTree)       :p2_1, after p1_2, 2d
    Ánh xạ kiểu dữ liệu MappingRule -> ClickHouse Types            :p2_2, after p2_1, 1d
    section Phase 3: Transmuter Engine
    Mở rộng ConnectionManager: GetMasterClickHouseConn             :p3_1, after p2_2, 1d
    Hiện thực ClickHouse Batch Transmute & Soft-Delete (is_deleted):p3_2, after p3_1, 2d
    section Phase 4: Recon & Kiểm thử
    Tích hợp Recon Hash tính toán trên ClickHouse FINAL            :p4_1, after p3_2, 1d
    E2E Test Transmute 100k rows Shadow -> ClickHouse Master       :p4_2, after p4_1, 1d
```

---

## Chi tiết Từng Giai đoạn

### Phase 1: Hạ tầng & Driver
1. Thêm service `clickhouse` vào `docker/docker-compose.yml` (Port 9000 native, 8123 http).
2. Tạo package `pkgs/clickhouse/client.go` trong `centralized-data-service` để khởi tạo connection pool an toàn.

### Phase 2: DDL Generator cho ClickHouse
1. Mở rộng `MasterDDLGenerator` để nhận biết `engine_type` của master binding.
2. Sinh DDL bảng ClickHouse chuẩn `ENGINE = ReplacingMergeTree(_version, _deleted) ORDER BY (_gpay_id)`.

### Phase 3: Transmuter Engine Phân nhánh Đích
1. Cập nhật `ConnectionManager` để cung cấp `clickhouse.Conn` khi `master_connection.engine_type == "clickhouse"`.
2. Tạo module `transmuter_clickhouse.go` xử lý `PrepareBatch` và batch append thay cho câu lệnh `ON CONFLICT` của Postgres.

### Phase 4: Reconciliation & Đo lường
1. Viết adapter query đối soát hash trên ClickHouse.
2. Kiểm thử tải và xác nhận tốc độ ghi đạt tiêu chuẩn OLAP.
