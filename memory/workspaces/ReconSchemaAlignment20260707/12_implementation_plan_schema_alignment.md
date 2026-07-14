# Kế hoạch triển khai - Đồng bộ Schema Đối soát Shadow/Master

## 1. Các file thay đổi
- **File**: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Chi tiết thay đổi**:
  Bổ sung trường `master_schema` vào danh sách cột được SELECT của cả 2 phần trong UNION query tại hàm `GetTableHistory`.

## 2. Kế hoạch thực thi chi tiết
- **Bước 1**: Cập nhật file `recon_read_repo_gorm.go` (Ủy quyền cho Muscle thực thi).
- **Bước 2**: Chạy build dự án `cdc-cms-service` để đảm bảo compile thành công:
  ```bash
  go build ./...
  ```
- **Bước 3**: Gửi request đối soát Segment B (Shadow to Master) thông qua NATS/API để sinh data mới, hoặc sử dụng data đã có sẵn trong DB từ session trước.
- **Bước 4**: Kiểm tra endpoint API `/api/reconciliation/report/export_jobs` để verify xem payload trả về có chứa đầy đủ `"master_schema"` và `"master_table"` hay không.

## 3. Kế hoạch kiểm thử & Xác minh (Verification Plan)
- Chạy unit test trong `cdc-cms-service/test/internal/app/queries/...`.
- cURL API để kiểm tra kết quả thực tế.
