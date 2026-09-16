# 11 Change Overview Report: CDC PostgreSQL → MongoDB & Multi-Table Denormalization

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Mục đích**: Báo cáo tổng kết những file sẽ thay đổi, số lượng dòng code ước tính, và tổng quan cách thay đổi.

---

## 1. Bảng Kê Khai Thay Đổi Mã Nguồn (Code Changes Matrix)

| Đường dẫn File | Trạng thái | Số dòng (+/-) | Mô tả Thay đổi & Kiến trúc |
|:---|:---:|:---:|:---|
| `centralized-data-service/deployments/debezium/pg-orders-to-mongo-connector.json` | **NEW** | +45 / -0 | File cấu hình Debezium Postgres connector tích hợp SMT `ByLogicalTableRouter` và `ExtractNewRecordState`. |
| `centralized-data-service/internal/sinkworker/mongo_type_converter.go` | **NEW** | +95 / -0 | Bộ chuyển đổi dữ liệu an toàn từ Debezium sang BSON primitives (`Decimal128`, `ISODate`, `UUID`). |
| `centralized-data-service/internal/sinkworker/mongo_aggregator_worker.go` | **NEW** | +180 / -0 | Core Worker xử lý micro-batch Kafka, thực thi Pull-then-Push cho `order_items`, `$set` cho `orders`/`payments`, BulkWrite Ordered. |
| `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` | **MODIFY** | +60 / -0 | Bổ sung hàm execute snapshot streaming qua `LEFT JOIN LATERAL` đọc PostgreSQL và bulk insert MongoDB. |
| `centralized-data-service/internal/service/recon/recon_heal_fetch.go` | **MODIFY** | +40 / -5 | Bổ sung filter `is_deleted != true` khi fetch dữ liệu MongoDB để so sánh tính toàn vẹn. |
| `centralized-data-service/test/internal/service/mongo_aggregator_test.go` | **NEW** | +150 / -0 | Test suite kiểm thử tính năng: Strict FIFO, Idempotency Pull-then-Push, Zombie Prevention, Versioning. |

---

## 2. Tổng quan Tác động (Blast Radius Assessment)
- **Blast Radius**: Rất hẹp, các file mới nằm tách biệt trong `internal/sinkworker` và cấu hình deployment độc lập.
- **Không ảnh hưởng các luồng cũ**: Các pipeline hiện có (như `trans_proxy_history`, `payment_bills`, các bảng shadow hiện hành sang PG DW) hoàn toàn độc lập, không bị chạm vào.
