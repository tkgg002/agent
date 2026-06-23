# Context: Restore Scan Array Child Fields

## Goal
Khôi phục và sửa lỗi trích xuất child fields của array/nested JSON trong quá trình scan array fields khi nhận lệnh `cdc.cmd.scan-array` qua NATS, sinh các `mapping_rule_v2` / `mapping_rule_master` ở trạng thái `pending` cho các child field phát hiện được.

## Active Directory
- [internal/handler/recon](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon)
- File ảnh hưởng chính: [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler.go)

## Current Status
- Hàm `HandleScanArrayFields` trong `scan_handler.go` đang sử dụng phiên bản mock/rút gọn chỉ quét root-level JSON và không xử lý `explode_path` hay đệ quy trích xuất child fields.
- Cần khôi phục lại logic gốc xịn từ lịch sử Git, tích hợp vào cấu trúc phân lớp hiện tại và viết thêm/khôi phục unit test để đảm bảo logic chạy đúng và sinh rule ở trạng thái `pending`.
