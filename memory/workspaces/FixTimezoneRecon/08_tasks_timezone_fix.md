# Danh sách Task chi tiết: Sửa lỗi Timezone Drift trong Recon Pipeline

- `[x]` Sửa đổi mã nguồn trong `recon_query.go` để cập nhật hàm `parsePostgresTimestamp` hỗ trợ `.UTC()` cho `time.Time` và `*time.Time`.
- `[x]` Cập nhật unit test `TestParsePostgresTimestamp` trong `recon_postgres_source_test.go` để bổ sung test case múi giờ Local và FixedZone.
- `[x]` Chạy unit tests của package recon: `go test -v ./internal/service/recon/...`.
- `[x]` Chạy script `compare_hash.go` trên thực tế để xác nhận XOR Hash và số lượng count giữa MongoDB và Postgres Shadow khớp nhau hoàn toàn.
- `[x]` Cập nhật tài liệu tiến độ và tạo báo cáo nghiệm thu walkthrough.
