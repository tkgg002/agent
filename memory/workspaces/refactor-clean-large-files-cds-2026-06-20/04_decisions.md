# Architectural Decisions - Refactor Clean Large Files CDS

Tài liệu này ghi nhận các quyết định kiến trúc quan trọng được đưa ra trong quá trình tái cấu trúc các file lớn (`kafka_consumer.go`, `recon_source_agent.go`) của centralized-data-service.

## ADR 1: Phân tách recon_source_agent.go thành các cấu phần chuyên biệt

### Bối cảnh
File `recon_source_agent.go` ban đầu có kích thước lớn (1166 dòng), gộp chung nhiều trách nhiệm khác nhau:
- Quản lý MongoDB connection, Rate Limiter và Circuit Breaker.
- Logic băm XOR, xxhash và trích xuất Timestamp.
- Logic queries MongoDB (EstimatedCount, BucketCounts, CountInWindow).
- Logic streaming keyset pagination.
- Legacy shims tương thích ngược.

### Quyết định
Phân rã file này thành 6 file nhỏ chuyên biệt trong package `recon` dựa trên Single Responsibility Principle (SRP):
1.  `recon_source_agent.go`: Chỉ giữ vai trò quản lý vòng đời connection và circuit breaker.
2.  `recon_models.go`: Đóng gói các kiểu dữ liệu, cấu hình và ánh xạ mã lỗi MongoDB.
3.  `recon_hash.go`: Tập trung toàn bộ thuật toán băm XOR và helper trích xuất dữ liệu.
4.  `recon_query.go`: Thực thi các truy vấn count/aggregate.
5.  `recon_stream.go`: Xử lý streaming keyset pagination chống cursor timeout và OOM.
6.  `recon_legacy.go`: Giữ các legacy shims để đảm bảo tương thích ngược hoàn hảo với CMS.

### Hệ quả
- **Tích cực**: Kích thước file core giảm 88.5% (từ 1166 dòng xuống 134 dòng). Dễ dàng viết Unit Test độc lập cho các thuật toán băm mà không cần mock MongoDB client.
- **Tiêu cực**: Tăng số lượng file vật lý trong package `recon`.

---

## ADR 2: Phân tách kafka_consumer.go thành các helper chuyên biệt

### Bối cảnh
File `kafka_consumer.go` có kích thước 1521 dòng, quản lý cả vòng đời reader, adaptive batching, Avro schema validation, và DLQ logic.

### Quyết định
Phân rã thành:
1.  `kafka_consumer.go`: Giữ lại cấu trúc quản lý vòng đời consumer.
2.  `adaptive_batcher.go`: Logic tính toán kích thước batch động.
3.  `avro_helper.go`: Logic parse và validate Avro schema.
4.  `dlq_helper.go`: Logic ghi nhận và xử lý hàng đợi thư chết.
5.  `topic_helper.go`: Logic refresh và discovery topic tự động.
6.  `utils.go`: Các hàm helper và Otel tracing.

### Hệ quả
- **Tích cực**: Dễ bảo trì, phân tách rạch ròi trách nhiệm của consumer và các subsystem phụ trợ.

---

## ADR 3: Phân tách recon_dest_agent.go tương ứng với cấu trúc của recon_source_agent.go

### Bối cảnh
File `recon_dest_agent.go` ban đầu có kích thước 652 dòng, chứa các logic quản lý connection Postgres, XOR Hashing, queries Postgres, keyset pagination, legacy shims và SQL safety helpers. Để duy trì tính nhất quán kiến trúc 1-1 với `ReconSourceAgent` (đã phân rã ở ADR 1), chúng ta cần áp dụng cùng một pattern phân rã tương đương.

### Quyết định
Phân rã file này thành 7 file chuyên biệt:
1. `recon_dest_agent.go`: Chỉ giữ lại cấu trúc lõi `ReconDestAgent`, constructor và helper transaction read-only.
2. `recon_dest_models.go`: Chứa structs cấu hình, `BucketStat` và `IDTs`.
3. `recon_dest_hash.go`: Chứa logic băm XOR, xxhash.
4. `recon_dest_query.go`: Thực thi các truy vấn Postgres count/aggregate.
5. `recon_dest_stream.go`: Xử lý streaming / listing IDs.
6. `recon_dest_legacy.go`: Legacy shims cho ChunkHash.
7. `recon_dest_safety.go`: SQL identifier validation & quoting.

### Hệ quả
- **Tích cực**: Kích thước file chính giảm 90% (từ 652 xuống 66 dòng). Tăng tính rõ ràng, dễ dàng bảo trì và đồng bộ hóa cấu trúc giữa 2 store source-destination.

