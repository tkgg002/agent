# Requirements: Reconcile Component Overhaul

## 1. Yêu cầu làm sạch Schema (Schema Consolidation)
- **Tình trạng hiện tại**: Bảng `cdc_reconciliation_report` chứa nhiều cột chắp vá qua các migration `008`, `081`, `082`, `083`, `084`, `085`.
- **Yêu cầu**:
  - Hợp nhất và định cấu trúc lại bảng `cdc_reconciliation_report` thành một schema sạch sẽ, có thứ tự cột khoa học.
  - Loại bỏ các cột trùng lặp hoặc không còn sử dụng.
  - Phân tách rõ ràng giữa:
    - **Định danh Pipeline**: `shadow_schema`, `shadow_table` làm khóa chính logic, kết hợp với `segment` (`source_shadow` vs `shadow_master`).
    - **Dữ liệu đối soát tức thời (Window Counts)**: `source_count`, `dest_count`, `diff`, `missing_count`, `stale_count`, `missing_ids`, `stale_ids`, `field_diffs`.
    - **Dữ liệu tổng thể (Totals)**: `total_source_count`, `total_dest_count`.
    - **metadata của phiên chạy (Metadata)**: `run_id`, `check_type`, `status`, `tier`, `duration_ms`, `error_code`, `error_message`, `checked_at`.
  - Thiết kế các chỉ mục (indexes) tối ưu để phục vụ dashboard hiển thị nhanh và tránh table scan lớn.

## 2. Yêu cầu tối ưu hóa lưu lượng ghi (Anti-Garbage Log Writing)
- **Tình trạng hiện tại**: Mỗi chu kỳ quét (cron/poller) chạy qua đều chèn mới một dòng vào `cdc_reconciliation_report` bất kể trạng thái là `ok` hay `drift`. Điều này tạo ra hàng ngàn dòng log `ok` giống hệt nhau, làm loãng dữ liệu thực tế và lãng phí dung lượng.
- **Yêu cầu**:
  - **Deduplication / State Update**: Nếu một pipeline đang ở trạng thái `ok` và phiên chạy mới tiếp tục có kết quả `ok` (với counts và watermark không thay đổi hoặc thay đổi trong ngưỡng), hệ thống nên **cập nhật** cột `checked_at` và `run_id` của bản ghi `ok` hiện tại thay vì `INSERT` bản ghi mới.
  - **Watermark/Checkpoint Table**: Hoặc tách biệt bảng lưu trạng thái hiện tại (State Table - 1 row per pipeline per segment) và bảng lưu lịch sử thay đổi thực tế/drift (History Table - chỉ ghi khi trạng thái thay đổi hoặc có lỗi).
  - **Retention & Pruning Policy**: Thiết kế cơ chế tự động dọn dẹp (pruning) các bản ghi `ok` cũ hơn N ngày để giữ cho bảng đối soát luôn nhẹ nhàng, sạch sẽ.

## 3. Yêu cầu nhất quán Kiến trúc & Pattern (Architecture Alignment)
- **Tình trạng hiện tại**: Mã nguồn reconcile nằm rải rác trong `internal/service/recon/`.
- **Yêu cầu**:
  - Bám sát nguyên lý "Simplicity First, minimal impact".
  - Không thay đổi cấu hình DB gốc của dự án hoặc dùng các cơ chế cheat để vượt qua test.
  - Đảm bảo các unit/integration tests được cập nhật tương ứng với schema mới và hoạt động trơn tru.
