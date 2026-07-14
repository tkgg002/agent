# Báo cáo thay đổi - Tracing Reconciliation Detail

Báo cáo này ghi lại chi tiết các thay đổi trong mã nguồn phục vụ việc cải thiện độ bao phủ OpenTelemetry Tracing cho dịch vụ đối soát dữ liệu (Reconciliation), tối ưu hóa tránh DDL Lock contention trong Transmuter, và tạo index hỗ trợ tăng tốc truy vấn.

## 1. Tóm tắt các File thay đổi

| Tên File | Thao tác | Mô tả |
| :--- | :--- | :--- |
| [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go) | `MODIFY` | Tích hợp OpenTelemetry ChildSpan cho các phương thức streaming/listing IDs. |
| [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go) | `MODIFY` | Tích hợp OpenTelemetry ChildSpan cho các hàm băm dữ liệu `HashWindow`, `BucketHash` của Source Agent. |
| [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go) | `MODIFY` | Tích hợp OpenTelemetry ChildSpan cho các hàm đếm và lấy mốc thời gian của Source Agent. |
| [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go) | `MODIFY` | Tích hợp OpenTelemetry ChildSpan cho các hàm băm dữ liệu `HashWindow`, `BucketHash` của Destination Agent (Postgres). |
| [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go) | `MODIFY` | Tích hợp OpenTelemetry ChildSpan cho các hàm truy vấn đếm, lấy thống kê của Destination Agent (Postgres). |
| [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go) | `MODIFY` | Thêm tracing spans cho các luồng đối soát Tier A, hỗ trợ động hóa phân giải timestamp Postgres. |
| [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go) | `MODIFY` | Thêm tracing spans cho các luồng đối soát Tier B. |
| [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) | `MODIFY` | Tích hợp cơ chế cache DDL `ensuredMasters` để ngăn DDL lock contention (lỗi 55P03) và tạo index `_source_id` concurrently bất đồng bộ. |
| [schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go) | `MODIFY` | Tự động tạo partial index cho cột `_deleted = true` và index `_source_id` trên Shadow DB. |
| [master_ddl_generator.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go) | `MODIFY` | Tự động tạo partial index cho cột `_deleted = true` và index cho timestamp nghiệp vụ trên Master DB. |

## 2. Chi tiết thay đổi cụ thể

### 2.1. OpenTelemetry Tracing bao phủ Database Agents
* **Source Agent (`ReconSourceAgent`):**
  * `HashWindow`, `BucketHash`, `CountDocuments`, `EstimatedCount`, `BucketCounts`, `CountInWindow`, `MaxWindowTs`, `ListIDTsInWindow`, `ListIDsInWindow`, `ListAllIDs`, `StreamAllIDs` (và các Postgres variants tương ứng) đều đã được bọc trong các child span thích hợp (ví dụ: `recon.source.bucket_hash`, `recon.source.stream_all_ids`), truyền context và span attributes (`db.database`, `db.collection`, `recon.t_lo`, `recon.t_hi`, `db.is_postgres`) đầy đủ.
* **Destination Agent (`ReconDestAgent`):**
  * Đã khắc phục khoảng trống tracing bằng cách tích hợp OTel child spans cho `CountRows`, `CountDeletedRows`, `EstimatedCountRows`, `CountInWindow`, `BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs`, `HashWindow`, `BucketHash` trên Postgres Destination DB.

### 2.2. Khắc phục DDL Lock Contention của Transmuter
* Thêm map thread-safe `ensuredMasters map[string]bool` vào `TransmuterModule`.
* Kiểm tra trạng thái cache trước khi chạy `EnsureMaster` nhằm tránh chạy DDL ở mỗi lô dữ liệu đồng bộ.
* Invalidate cache trong `InvalidateRuleCache` khi rule schema thay đổi.
* Tự động kiểm tra và tạo index `_source_id` concurrently trên Shadow table dưới nền (background goroutine) để không chặn luồng transmuter chính.

### 2.3. Tối ưu hóa Index
* Tạo index partial `CREATE INDEX ... ON <table> (_deleted) WHERE _deleted = true` trên cả Shadow và Master.
* Tạo index trên cột thời gian nghiệp vụ (business timestamp) được lấy từ `cdc_table_registry` để tối ưu các truy vấn phân dải mốc thời gian.

## 3. Kết quả xác minh (Verification Results)
* Chạy `go build ./cmd/... ./internal/...` thành công không có lỗi biên dịch.
* Chạy `go test ./internal/...` thành công, pass 100% tất cả các unit tests.
