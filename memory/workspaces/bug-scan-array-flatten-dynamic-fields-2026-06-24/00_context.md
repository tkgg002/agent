# Context: Scan Array Flatten Dynamic Fields Bug

## Goal
Ngăn chặn việc quét và tạo các `mapping_rule_v2` tĩnh cho các phần tử mảng động hoặc toàn bộ nội dung mảng stringified JSON trong quá trình scan flatten array fields. Bản chất của các child fields trong mảng này không phải tĩnh (ví dụ: `[0,1,2,3,4]`), do đó không được tạo các mapping rule tĩnh tương ứng.

## Active Directory
- Thư mục làm việc chính: `data-hub/centralized-data-service`
- File liên quan:
  - `internal/service/source/scan_service.go`
  - `internal/handler/recon/scan_handler_discover.go` (hoặc các file handler scan khác)

## Current Status
- Khi scan field flatten, nếu gặp trường dữ liệu dạng mảng (array) chứa các JSON object (ví dụ logs dạng các step `REQUEST_LINK`, `CONFIRM_LINK`), hệ thống đang cố gắng parse hoặc flatten và tạo trực tiếp mapping rule cho dạng dữ liệu này (hoặc tạo thẳng mapping rule dạng string representation).
- Cần xác định vị trí thực hiện scan flatten array và lọc bỏ/bỏ qua các trường động hoặc cấu trúc mảng này.
