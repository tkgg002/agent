# Context: ReconPostgresSource

## Yêu cầu/Vấn đề
Hệ thống đối soát (Reconciliation) đang gặp lỗi khi cố gắng parse chuỗi kết nối nguồn PostgreSQL bằng thư viện MongoDB Client trong tiến trình chạy đối soát:
`source max ts: connect to source postgres://<redacted>: error parsing uri: scheme must be "mongodb" or "mongodb+srv"`

## Nguyên nhân
- File `recon_tier_a.go` gọi `rc.sourceAgent.MaxWindowTs(...)` truyền `entry.SourceURL`, `entry.SourceDB`, `entry.SourceTable`, và `tsField(entry)`.
- Hiện tại, `ReconSourceAgent` (`recon_source_agent.go`) chỉ được thiết kế để kết nối và truy vấn MongoDB. Khi nhận được `sourceURL` có scheme `postgres://` hoặc `postgresql://`, nó vẫn cố gọi `mongodb.NewClient(ctx, ...)` dẫn tới lỗi parse URI từ MongoDB driver.
- Hệ thống hỗ trợ nguồn PostgreSQL (thông qua sync engine Debezium), nhưng cấu phần reconciliation chưa được trang bị cơ chế tự động chuyển đổi logic truy vấn dựa trên loại database nguồn (MongoDB hay PostgreSQL).

## Giải pháp đề xuất
- Refactor `ReconSourceAgent` để nhận diện scheme của `sourceURL`. Nếu URL bắt đầu bằng `postgres://` hoặc `postgresql://`, nó sẽ sử dụng driver PostgreSQL (thông qua GORM) để mở kết nối và thực hiện các câu lệnh SQL tương đương với MongoDB:
  - `MaxWindowTs`: Truy vấn `SELECT MAX(timestampField) FROM table`.
  - `CountDocuments`: Truy vấn `SELECT COUNT(*) FROM table`.
  - `EstimatedCount`: Sử dụng thống kê O(1) từ `pg_class.reltuples` (tương tự như `EstimatedCountRows` của `ReconDestAgent`).
  - `BucketCounts`: Nhóm các bản ghi theo giờ của timestamp và tính tổng số lượng.
  - `CountInWindow`: Truy vấn `SELECT COUNT(*) WHERE ts >= lo AND ts < hi`.
  - `HashWindow` / `BucketHash`: Sử dụng XOR xxhash trên ID và Timestamp của dữ liệu Postgres nguồn để đối soát chính xác với Destination (Shadow/Master).
