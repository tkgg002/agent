# Bối cảnh Workspace: Phân tích Luồng Đối soát Tier 2 XOR-Hash

## Tổng quan
Workspace này được tạo ra để nghiên cứu và kiểm tra luồng đối soát Tier 2 trong hệ thống `cdc-system`.
Cụ thể, chúng ta cần kiểm tra xem luồng Tier 2 có thực hiện đối chiếu window-based XOR-hash trên cả hai bên (Source và Destination) hay không, và xác minh xem luồng này có strictly read-only (chỉ đọc, không ghi/sửa đổi dữ liệu) hay không.

## Thư mục dự án mục tiêu
- Đường dẫn: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`
- Các tệp tin cốt lõi:
  - `internal/service/recon/recon_tier_a.go`
  - `internal/service/recon/recon_hash.go`
  - `internal/service/recon/recon_dest_hash.go`
  - `internal/service/recon/recon_stream.go`
  - `internal/service/recon/recon_dest_query.go`
