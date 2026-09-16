# 00 Context: Bổ sung ClickHouse làm Master Destination (Master Tier OLAP)

> **Workspace**: `feat-clickhouse-master-destination`  
> **Ngày khởi tạo**: 2026-09-11  
> **Chủ quản**: Brain (Antigravity) & Staff Engineer  
> **Dự án**: `centralized-data-service`, `cdc-cms-service`, `docker`

---

## 1. Bối cảnh Hệ thống (Current State)
- `cdc-system` vận hành pipeline CDC qua 2 tầng: `Shadow` (lưu trữ raw data) $\rightarrow$ `Master` (chuẩn hóa schema, typed columns, phục vụ truy vấn).
- Tầng `Master` hiện tại mới chỉ hỗ trợ đích là **PostgreSQL** (`RoleDestination`, GORM/pgx, `INSERT ... ON CONFLICT DO UPDATE`).
- Tuy nhiên, trong schema Control Plane `cdc_system.connection_registry`, trường `engine_type` đã được thiết kế sẵn hỗ trợ `clickhouse` (check constraint tại migration `029` và `087`).
- Bảng `cdc_system.master_binding` cũng hỗ trợ mô hình 1-N (Fan-out: 1 source object / shadow table có thể chiếu ra nhiều Master destination khác nhau).

---

## 2. Nhu cầu Nghiệp vụ (The Requirement)
- **Thêm ClickHouse ở tầng Master (Master Destination)**:
  * Cho phép người vận hành trên CMS UI chọn ClickHouse làm đích lưu trữ Master Table bên cạnh PostgreSQL.
  * Phục vụ các bài toán phân tích dữ liệu lớn (OLAP), báo cáo tài chính tốc độ cao, truy vấn aggregate trên hàng trăm triệu/hàng tỷ dòng giao dịch.
  * Transmuter trong `centralized-data-service` tự động sinh DDL trên ClickHouse (`ReplacingMergeTree`), chuyển đổi dữ liệu và thực thi batch insert hiệu năng cao.
