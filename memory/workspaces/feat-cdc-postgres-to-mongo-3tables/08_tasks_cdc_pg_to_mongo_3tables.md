# 08 Task Breakdown: CDC PostgreSQL → MongoDB & Multi-Table Aggregation

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Phân loại**: [Vận hành CMS] vs [Code thêm / Development] (Tuân thủ Lesson #2026-08-28)

---

## Danh sách Task Chi tiết

### Phase 1: Cấu hình Hạ tầng & Connector

- [ ] `TASK-01` **[Vận hành CMS]** Bật `wal_level = logical` trên PostgreSQL DB nguồn và gán `REPLICA IDENTITY FULL` cho 2 bảng `order_items`, `order_payments`.
- [ ] `TASK-02` **[Code thêm]** Tạo file cấu hình Debezium Connector `centralized-data-service/deployments/debezium/pg-orders-to-mongo-connector.json` tích hợp SMT `ByLogicalTableRouter` (route về `cdc.pg.unified.orders`) và `ExtractNewRecordState` (`delete.handling.mode: rewrite`).
- [ ] `TASK-03` **[Vận hành CMS]** Đăng ký Source Objects và Connector qua giao diện CMS (`cdc-cms-service` API / Web).

### Phase 2: Core Worker Ingestion (`centralized-data-service`)

- [ ] `TASK-04` **[Code thêm]** Tạo file `internal/sinkworker/mongo_type_converter.go`: Chuyển đổi chuẩn xác `UUID` $\rightarrow$ text, `NUMERIC` $\rightarrow$ `primitive.Decimal128`, `TIMESTAMPTZ` $\rightarrow$ `ISODate`, `JSONB` $\rightarrow$ `bson.M`.
- [ ] `TASK-05` **[Code thêm]** Tạo file `internal/sinkworker/mongo_aggregator_worker.go`:
  - Hiện thực `ProcessBatch` nhận `[]CDCEvent`.
  - Bảng cha `orders`: Soft-delete khi 'd', Upsert khi 'c','u'.
  - Bảng con `order_items`: Pull-then-Push với version `_v` ở từng item, `Upsert: false`.
  - Bảng con `order_payments`: Set payment subdocument, `Upsert: false`.
  - Thực thi `BulkWrite(SetOrdered(true))` và commit Kafka offset.
- [ ] `TASK-06` **[Code thêm]** Đăng ký và wire `MongoAggregatorWorker` vào entrypoint worker trong `cmd/sinkworker/main.go` hoặc `cmd/worker/main.go`.

### Phase 3: Snapshot Backfill Pipeline

- [ ] `TASK-07` **[Code thêm]** Mở rộng `internal/handler/orchestration/snapshot_runner_handler.go`: Viết hàm snapshot query PostgreSQL sử dụng `LEFT JOIN LATERAL` độc lập, stream cursor batch 1,000 orders và ghi `BulkWrite` sang MongoDB.

### Phase 4: Reconciliation & DoD Quality Gate

- [ ] `TASK-08` **[Code thêm]** Mở rộng `internal/service/recon/` hỗ trợ tính hash so sánh giữa 3 bảng PostgreSQL và 1 collection MongoDB với điều kiện `is_deleted != true`.
- [ ] `TASK-09` **[Code thêm]** Viết Unit Test & Integration Test trong `test/internal/service/`: Test ordering, test Pull-then-Push, test zombie prevention.
- [ ] `TASK-10` **[Vận hành CMS]** Thực thi test E2E, nghiệm thu các Gates G1 đến G8 và bàn giao vận hành.
