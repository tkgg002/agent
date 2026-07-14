# Kế hoạch Triển khai (AI Implementation Plan) - Hide Deleted/Inactive Pipelines

Dự án: `cdc-cms-service`
Workspace: `FixDataIntegrityDeletedPipeline20260707`
Mục tiêu: Chuyển đổi các `LEFT JOIN cdc_table_registry` sang `INNER JOIN cdc_table_registry` có lọc `is_active = TRUE` trong các query lấy báo cáo đối soát mới nhất (`listLatestPrimary` và `listLatestLegacy`) để loại bỏ các pipeline đã bị xóa (deleted) hoặc không hoạt động (inactive) khỏi Dashboard.

## Các bước thực hiện

### Bước 1: Chuẩn bị & Ghi nhận tiến độ
- Cập nhật file `05_progress_delete_pipeline.md` ghi nhận việc bắt đầu pha thực thi.

### Bước 2: Sửa đổi mã nguồn
- Tệp tin cần sửa đổi: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Sửa đổi 1**: Trong hằng số `listLatestPrimary` (dòng 138), thay thế:
  ```sql
  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table
  ```
  bằng:
  ```sql
  INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
  ```
- **Sửa đổi 2**: Trong hằng số `listLatestLegacy` (dòng 173), thay thế:
  ```sql
  LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table
  ```
  bằng:
  ```sql
  INNER JOIN cdc_table_registry reg ON reg.target_table = r.target_table AND reg.is_active = TRUE
  ```

### Bước 3: Xác minh bằng Unit Test
- Chạy lệnh test trong thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`:
  ```bash
  go test ./test/...
  ```
- Xác minh toàn bộ các test cases chạy thành công (PASS).

### Bước 4: Hoàn tất & Cập nhật Nhật ký Tiến độ
- Cập nhật nhật ký tiến độ trong `05_progress_delete_pipeline.md`.
- Ghi nhận báo cáo kết quả chi tiết.
