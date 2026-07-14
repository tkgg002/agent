# Kế hoạch triển khai - Khắc phục Hiển thị Dữ liệu ID Diff

## 1. Các file thay đổi
- **File**: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Chi tiết thay đổi**:
  Bổ sung các trường dữ liệu ID diff (`missing_count`, `missing_ids`, `stale_count`, `stale_ids`, `field_diffs`, `orphan_count`) và các trường heal (`healed_at`, `healed_count`, `healed_duration_ms`, `healed_mismatched_count`, `healed_mismatched_duration_ms`, `healed_missing_dest_count`, `healed_missing_dest_duration_ms`, `pruned_missing_src_count`, `pruned_missing_src_duration_ms`) vào mệnh đề SELECT của UNION query ở hàm `GetTableHistory`.

## 2. Kế hoạch thực thi chi tiết
- **Bước 1**: Cập nhật file `recon_read_repo_gorm.go` (Ủy quyền cho Muscle thực thi).
- **Bước 2**: Chạy build dự án `cdc-cms-service` để đảm bảo compile thành công:
  ```bash
  go build ./internal/... ./cmd/...
  ```
- **Bước 3**: Chạy unit test của queries:
  ```bash
  go test -v ./test/internal/app/queries/...
  ```
- **Bước 4**: Khởi động lại service và cURL API để kiểm tra kết quả:
  ```bash
  curl -s -H "Authorization: Bearer dev-token" "http://localhost:8083/api/reconciliation/report/export_jobs?page=1&page_size=10" | jq .
  ```

## 3. Kế hoạch kiểm thử & Xác minh (Verification Plan)
- Đảm bảo JSON API phản ánh chính xác các thông tin diff và heal (không bị null/0 giả tạo đối với các bản ghi chuẩn).
