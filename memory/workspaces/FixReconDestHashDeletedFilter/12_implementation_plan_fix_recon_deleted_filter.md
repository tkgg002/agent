# Sửa lỗi thiếu filter _deleted trong DestAgent HashWindow và BucketHash

Truy vấn đối soát ở phía Postgres (`ReconDestAgent.HashWindow` và `ReconDestAgent.BucketHash`) hiện không lọc các dòng đã xóa mềm (`_deleted = true`). Điều này dẫn đến sự không đồng nhất về số lượng bản ghi và mã băm XOR giữa nguồn và đích, gây ra các cảnh báo lệch giả (`false drift`) và liên tục kích hoạt drill down (`drift_drill_down`) đắt đỏ trong mọi cửa sổ. 

Mục tiêu của kế hoạch này là thêm điều kiện `AND NOT "_deleted"` (hoặc `AND "_deleted" = false`) vào các câu truy vấn tương ứng để bỏ qua các bản ghi đã xóa mềm khi đối soát.

## User Review Required

> [!IMPORTANT]
> Cần đảm bảo cột `_deleted` luôn tồn tại trong tất cả các bảng shadow và master. Theo thiết kế hệ thống cdc-system, cột này là bắt buộc cho tất cả các bảng shadow/master do sinkworker tạo ra. Do đó, việc hardcode `AND NOT "_deleted"` là an toàn và không gây lỗi cú pháp SQL.

## Open Questions

Không có câu hỏi mở nào.

## Proposed Changes

---

### Centralized Data Service (Recon Package)

#### [MODIFY] [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)

* **Hàm `HashWindow`:**
  - Thêm điều kiện `AND NOT "_deleted"` vào câu truy vấn ở nhánh timestamp `_source_ts` (dòng 52-58).
  - Thêm điều kiện `AND NOT "_deleted"` vào câu truy vấn ở nhánh domain timestamp (dòng 97-104).
* **Hàm `BucketHash`:**
  - Thêm điều kiện `AND NOT "_deleted"` vào các câu truy vấn keyset pagination (dòng 191-199).

## Verification Plan

### Automated Tests
- Chạy unit tests cho gói recon:
  ```bash
  go test -v ./internal/service/recon/...
  ```
- Viết thêm case test cụ thể kiểm thử tính năng bỏ qua dòng xóa mềm:
  - Tạo dữ liệu mẫu trên đích Postgres có `_deleted = true`.
  - Đảm bảo `HashWindow` và `BucketHash` không tính toán các bản ghi này vào XOR hash hoặc count.
