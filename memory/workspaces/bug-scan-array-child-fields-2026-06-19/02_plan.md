# Plan: Restore Scan Array Child Fields

## Objectives
1. Tìm lại logic `HandleScanArrayFields` gốc trong lịch sử git.
2. Thiết kế giải pháp tích hợp logic cũ vào cấu trúc mới của `scan_handler.go`.
3. Kiểm tra cách trích xuất đệ quy child fields của JSON lồng nhau/array.
4. Đảm bảo các rule được sinh ra ở trạng thái `pending`.
5. Tạo hoặc khôi phục unit test để xác minh.

## Checklist
- [ ] Research lịch sử git để lấy code gốc của `HandleScanArrayFields`.
- [ ] Soạn thảo Implementation Plan chi tiết.
- [ ] Sửa đổi `scan_handler.go` để tích hợp logic đệ quy.
- [ ] Viết / khôi phục unit test trong `scan_handler_test.go` hoặc file test tương ứng.
- [ ] Chạy kiểm thử tự động để verify kết quả.
