# 12 Implementation Plan: Triển khai Nâng cấp Hệ thống CDC PostgreSQL → MongoDB (3 Bảng)

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Tài liệu**: Kế hoạch Triển khai Chi tiết của AI (AI Implementation Plan)  
> **Chủ quản**: Brain (Antigravity) & Staff Engineer  
> **Ngày phê duyệt**: 2026-09-10

---

## I. TỔNG QUAN YÊU CẦU & BỐI CẢNH KỸ THUẬT

### 1.1 Yêu cầu Nghiệp vụ
Nâng cấp hệ thống CDC `data-hub` để:
1. Đồng bộ CDC thời gian thực từ PostgreSQL sang MongoDB.
2. Mở rộng xử lý bài toán denormalization gom 3 bảng quan hệ 3NF (`orders`, `order_items`, `order_payments`) từ PostgreSQL về 1 collection tài liệu lồng nhau `orders` trong MongoDB.

### 1.2 Nguyên tắc Triển khai (Engineering Principles)
- **Simplicity First**: Giữ worker hoàn toàn stateless, không dùng stateful debounce buffer.
- **Minimal Impact**: Không sửa đổi hoặc phá vỡ cấu trúc của các pipeline CDC hiện hữu trong `centralized-data-service`.
- **Strict FIFO per Entity**: Gom 3 bảng về chung 1 topic duy nhất để toàn bộ event của 1 `order_id` đi vào 1 Kafka Partition.
- **Zero Zombie Documents**: Bảng cha soft-delete; bảng con tuyệt đối cấm `Upsert: true`.
- **Cartesian-Free Backfill**: Sử dụng `LEFT JOIN LATERAL` độc lập khi snapshot dữ liệu lịch sử.

---

## II. BẢN ĐỒ THAY ĐỔI & CÁC BƯỚC NÂNG CẤP HỆ THỐNG

### Bước 1: Nâng cấp Cấu hình PostgreSQL Nguồn & Debezium Connector
1. **Thiết lập PostgreSQL**:
   - Xác nhận `wal_level = logical`.
   - Cấu hình `REPLICA IDENTITY FULL` cho `order_items` và `order_payments` để đảm bảo khi `DELETE` hoặc `UPDATE` khóa ngoại, PostgreSQL phát ra toàn bộ giá trị cũ (`before` image).
2. **Triển khai Debezium Connector `cdc-pg-orders-to-mongo`**:
   - Vị trí: `centralized-data-service/deployments/debezium/pg-orders-to-mongo-connector.json`.
   - Định tuyến SMT `ByLogicalTableRouter`: Gom 3 bảng `orders`, `order_items`, `order_payments` về topic `cdc.pg.unified.orders`.
   - Thiết lập Kafka Key: `order_id`.
   - Unwrap SMT `ExtractNewRecordState`: Cấu hình `delete.handling.mode: rewrite` để giữ nguyên trường `order_id` khi delete.

### Bước 2: Xây dựng Stateless Mongo Aggregator Worker trong `centralized-data-service`
1. **Module `mongo_type_converter.go`**:
   - Vị trí: `centralized-data-service/internal/sinkworker/mongo_type_converter.go`.
   - Cung cấp các hàm parse và ép kiểu chuẩn:
     * `parseDecimal(val any) (primitive.Decimal128, error)`
     * `parseTime(val any) time.Time` (chuyển đổi UTC ISODate)
     * `parseBigInt(val any) int64`, `parseInt(val any) int32`
2. **Module `mongo_aggregator_worker.go`**:
   - Vị trí: `centralized-data-service/internal/sinkworker/mongo_aggregator_worker.go`.
   - Hàm cốt lõi: `ProcessBatch(ctx context.Context, coll *mongo.Collection, events []CDCEvent, logger *zap.Logger) error`.
   - Xử lý chi tiết:
     * `orders`: Soft-delete khi `Op == "d"` (`$set: { is_deleted: true, deleted_at: now() }`). Upsert khi `Op != "d"`.
     * `order_items`: Thực hiện Pull-then-Push (`$pull` item cũ theo `id`, sau đó `$push` item mới với `_v` per-element). **CẤM `SetUpsert(true)`**, điều kiện filter bắt buộc có `is_deleted: { $ne: true }`.
     * `order_payments`: Thực hiện `$set` subdocument `payment`, cấm `SetUpsert(true)`. Khi delete thực hiện `$unset: { payment: "" }`.
     * Thực thi 1 lệnh `coll.BulkWrite(ctx, writeModels, options.BulkWrite().SetOrdered(true))` duy nhất cho cả batch.
3. **Đăng ký và Wiring Worker**:
   - Khởi tạo kết nối MongoDB thông qua `pkgs/mongodb/client.go`.
   - Khởi tạo consumer loop trong `cmd/sinkworker/main.go` hoặc worker pool, lắng nghe topic `cdc.pg.unified.orders`.

### Bước 3: Nâng cấp SnapshotRunner Backfill Dữ liệu Lịch sử
1. Mở rộng `internal/handler/orchestration/snapshot_runner_handler.go`:
   - Bổ sung hàm snapshot multi-table streaming cursor.
   - Sử dụng câu lệnh SQL tối ưu:
     ```sql
     SELECT 
         o.id AS order_id, o.order_code, o.customer_id, o.status, o.total_amount, o.created_at, o.updated_at,
         COALESCE(i.items_json, '[]'::jsonb) AS items,
         p.payment_json AS payment
     FROM orders o
     LEFT JOIN LATERAL (
         SELECT jsonb_agg(jsonb_build_object(
             'id', id, 'product_id', product_id, 'quantity', quantity, 'price', unit_price,
             '_v', extract(epoch from updated_at)::bigint * 1000
         )) AS items_json
         FROM order_items WHERE order_id = o.id
     ) i ON TRUE
     LEFT JOIN LATERAL (
         SELECT jsonb_build_object(
             'id', id, 'payment_method', payment_method, 'amount', amount, 'status', status,
             'transaction_code', transaction_code, '_v', extract(epoch from updated_at)::bigint * 1000
         ) AS payment_json
         FROM order_payments WHERE order_id = o.id
         ORDER BY id DESC LIMIT 1
     ) p ON TRUE
     WHERE o.id >= $1 AND o.id < $2;
     ```
   - Chuyển đổi trực tiếp các dòng SQL sang BSON document và thực thi `bulkWrite(ReplaceOneModel, upsert: true)` vào MongoDB collection `orders`.

### Bước 4: Tích hợp Reconciliation & Data Integrity
1. Cập nhật logic fetch trong `internal/service/recon/recon_heal_fetch.go`:
   - Lọc bỏ các document có `is_deleted == true` khi tính hash đối soát.
2. Kiểm tra hash khớp giữa PostgreSQL (tổng hợp 3 bảng) và MongoDB collection `orders`.

---

## III. DEFINITION OF DONE & FEATURE QUALITY GATES (G1 - G8)

- **(G1) Requirement Traceability**: Đáp ứng 100% các yêu cầu FR-01 đến FR-07 và NFR-01 đến NFR-03.
- **(G2) Red → Green Test**: Có test case tái hiện lỗi Zombie Document khi bật `Upsert: true` ở bảng con, và chứng minh lỗi biến mất khi chuyển sang `Upsert: false`.
- **(G3) Test Thật**: Chạy test suite `go test -v ./test/internal/service/... -run TestMongoAggregator` PASS 100%.
- **(G4) Edge-cases**: Phủ kín các trường hợp: DELETE parent trước khi con tới, duplicate event do replay, trường số tiền lớn không tràn số (Decimal128), payment null/rỗng.
- **(G5) Chống Regression**: Không làm ảnh hưởng các luồng PostgreSQL Shadow/Master hiện hành.
- **(G6) Output Correctness**: Đối soát số lượng: 100,000 orders trên Postgres sau khi snapshot phải ra đúng 100,000 documents trên Mongo, tổng số items và payments khớp 100%.
- **(G7) Adversarial Review**: Đã audit và vượt qua toàn bộ 6 tiêu chí phản biện của Staff Engineer.
- **(G8) Bằng chứng vật lý**: Toàn bộ tài liệu được lưu trữ đầy đủ trong workspace `agent/memory/workspaces/feat-cdc-postgres-to-mongo-3tables`.
