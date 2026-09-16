# 00 Context: Pipeline CDC PostgreSQL → MongoDB & Denormalization 3 Bảng Quan Hệ

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Ngày khởi tạo**: 2026-09-10  
> **Chủ quản**: Brain (Antigravity) & Staff Engineer  
> **Dự án liên quan**: `centralized-data-service`, `cdc-cms-service`, `docker` (Kafka Connect/Debezium)

---

## 1. Bối cảnh Hệ thống Hiện tại (Current State)
- `cdc-system` (trong `data-hub`) hiện đang là hệ thống CDC đồng bộ dữ liệu từ các nguồn (MongoDB, PostgreSQL, MariaDB, SFTP) sang Data Warehouse PostgreSQL qua mô hình 2 tầng (`shadow` $\rightarrow$ `master`).
- `centralized-data-service`: Go service đóng vai trò worker xử lý CDC ingest từ Kafka (`sinkworker`), chuyển đổi schema (`transmuter`), snapshot lịch sử (`snapshot_runner`), và đối soát dữ liệu (`recon_heal`). Đã tích hợp sẵn driver PostgreSQL (`gorm`, `pgx`) và MongoDB (`go.mongodb.org/mongo-driver`).
- `cdc-cms-service`: Control Plane quản lý registry nguồn (`source_object_registry`), cấu hình bindings (`shadow_binding`, `master_binding`), điều phối worker qua NATS bus (`cdc.cmd.*`), và cung cấp REST API cho `cdc-cms-web`.
- `docker / deployments`: Cung cấp cụm Kafka Broker, Kafka Connect (với Debezium PostgreSQL, Debezium MongoDB, MongoDB Kafka Connect), Zookeeper/Schema Registry.

---

## 2. Nhu cầu Nghiệp vụ Mới (The New Requirement)
1. **Thiết lập luồng CDC PostgreSQL → MongoDB (Single Table 1:1)**:
   - Nguồn: PostgreSQL DB nghiệp vụ (ví dụ `goopay_source`).
   - Đích: MongoDB Replica Set (ví dụ collection `users`, `merchants`).
   - Yêu cầu bảo toàn kiểu dữ liệu (`UUID`, `NUMERIC/DECIMAL` sang `Decimal128`, `TIMESTAMPTZ` sang `ISODate`, `JSONB` sang nested document).
2. **Mở rộng Gom 3 Bảng PostgreSQL về 1 Document MongoDB (Multi-Table Denormalization)**:
   - Mô hình 3NF điển hình: Bảng cha `orders`, bảng con 1:N `order_items`, bảng con 1:1 `order_payments`.
   - Đích: Duy nhất 1 Collection `orders` trong MongoDB chứa root attributes, embedded array `items: [...]` và embedded subdocument `payment: { ... }`.
   - Đảm bảo 100% tính toàn vẹn thứ tự (**Strict FIFO**), loại bỏ race condition, triệt tiêu tài liệu rác (**No Zombie Document**), idempotency khi retry, và backfill dữ liệu lịch sử hiệu năng cao.
