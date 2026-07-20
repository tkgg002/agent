# Kế hoạch khắc phục lỗi Timezone Drift trong Recon Pipeline

Mục tiêu là sửa lỗi hàm `parsePostgresTimestamp` ở file `recon_query.go` để chuyển đổi múi giờ `time.Time` của PostgreSQL (do driver pgx trả về dưới dạng Local timezone) sang UTC chuẩn xác mà không bị dịch lệch giờ vật lý (timezone shift). Điều này sẽ giải quyết dứt điểm các báo cáo lệch `HashWindow` giả mạo (False Positive).

## User Review Required

> [!IMPORTANT]
> Sửa đổi này ảnh hưởng trực tiếp đến cách parse các cột timestamp từ PostgreSQL Shadow DB trong Recon Service. Toàn bộ các cột timestamp trong PostgreSQL Shadow DB đều được thiết lập kiểu dữ liệu `timestamp with time zone`. Do đó việc dùng `.UTC()` thay vì shift timezone là giải pháp đúng đắn về mặt vật lý và thống nhất hệ thống.

## Open Questions
- Không có câu hỏi nào mở. Phương án khắc phục đã được kiểm chứng thực nghiệm bằng script so sánh dữ liệu thực tế.

## Proposed Changes

### Centralized Data Service

#### [MODIFY] [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go)
- Sửa đổi hàm `parsePostgresTimestamp` tại case `time.Time` và `*time.Time` để sử dụng `.UTC()` thay vì `time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), time.UTC)`.

#### [MODIFY] [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go)
- Thêm unit test `TestParsePostgresTimestamp_Timezone` để kiểm chứng việc parse đúng múi giờ Local và FixedZone về múi giờ UTC chuẩn xác.

## Verification Plan

### Automated Tests
- Chạy unit tests của package recon:
  `go test -v ./internal/service/recon/...`
- Chạy scratch script để kiểm chứng XOR Hash thực tế của MongoDB và Postgres Shadow khớp hoàn toàn 100%:
  `go run /Users/trainguyen/.gemini/antigravity/brain/b4e8b28d-f986-4efe-85f0-36d3b6f8667f/scratch/compare_hash.go`
