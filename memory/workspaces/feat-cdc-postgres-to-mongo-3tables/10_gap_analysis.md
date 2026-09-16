# 10 Gap Analysis: Đánh giá Khoảng cách Hạ tầng Hiện tại vs Yêu cầu CDC Postgres -> Mongo (3 Bảng)

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Chủ quản**: Brain & Staff Engineer

---

## Bảng Đối chiếu Năng lực Hệ thống (System Capability Gap)

| Thành phần | Năng lực Hiện tại của Hệ thống | Yêu cầu Kỹ thuật Mới | Gap / Hạng mục Cần Phát triển |
|:---|:---|:---|:---|
| **Postgres CDC Connector** | Đã có Debezium Postgres Plugin, cấu hình mẫu đẩy từng bảng độc lập | Cần gom 3 bảng về 1 topic thống nhất với key `order_id` và giữ key khi DELETE | **Gap 1**: Thiếu config SMT `ByLogicalTableRouter` và unwrap rewrite cho connector. |
| **Kafka Pipeline** | Kafka topics được phân theo bảng: `<prefix>.<db>.<table>` | Topic unified: `cdc.pg.unified.orders` | **Gap 2**: Cần đăng ký topic mới và cấp quyền cho worker. |
| **Ingest Worker (CDS)** | `sinkworker` hiện tại chỉ tiêu thụ stream và ghi vào PostgreSQL Shadow tables | Cần tiêu thụ stream từ topic unified và ghi Idempotent sang MongoDB | **Gap 3**: Go engine chưa có `MongoAggregatorWorker` và BSON type converter. |
| **Snapshot Runner** | `snapshot_runner_handler.go` chỉ snapshot 1:1 từ Mongo -> Postgres hoặc Postgres -> Postgres | Cần snapshot gom 3 bảng quan hệ từ Postgres sang 1 collection Mongo | **Gap 4**: Thiếu hàm stream SQL LATERAL JOIN trong SnapshotRunner. |
| **Reconciliation** | Recon so sánh giữa Mongo collection và Postgres Master table 1:1 | So sánh hash giữa 3 bảng Postgres và 1 collection Mongo | **Gap 5**: Bổ sung adapter hash comparison với điều kiện `is_deleted != true`. |

---

## Kết luận & Đánh giá Rủi ro
Hệ thống hiện tại đã có đầy đủ driver và kết nối (`pgx`, `gorm`, `mongo-driver`, Kafka client). Các khoảng cách kỹ thuật trên hoàn toàn nằm trong khả năng phát triển mở rộng của Go engine (`centralized-data-service`) mà không cần thay đổi kiến trúc tổng thể (Clean/Hexagonal Architecture).
