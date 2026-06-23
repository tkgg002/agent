# Plan: Audit Logic Toàn Diện & Sửa Lỗi Scan Handlers

## Phase 1: Research & Audit
1. Đọc và phân tích kỹ lưỡng logic của file `internal/handler/recon/scan_handler.go` từ đầu đến cuối.
2. Kiểm tra chi tiết từng hàm:
   - `HandleScanSource`
   - `HandleScanFields`
   - `HandleScanRawData`
   - `HandleScanArrayFields`
   - `HandlePeriodicScan`
   - Các hàm helper liên quan (ví dụ: `flattenJSONWithTypes`, `insertPendingMappingRules`).
3. Xác định chính xác các điểm sai lệch logic, lỗi thứ tự xử lý, hoặc lỗi kiểu dữ liệu.

## Phase 2: Implementation
1. Sửa lỗi thứ tự Unmarshal trong `HandleScanArrayFields` (đưa block `json.Unmarshal` lên trước khi resolve reply subject).
2. Sửa bất kỳ lỗi logic nào khác được phát hiện trong quá trình audit ở Phase 1.
3. Cập nhật các thay đổi vào progress log trước khi thực thi.

## Phase 3: Verification
1. Chạy biên dịch dự án (`go build ./...`) để đảm bảo không bị lỗi cú pháp.
2. Chạy test suite của dự án (`go test ./...`) để xác minh tính ổn định của code.
3. Chạy công cụ rà soát bảo mật `/security-agent` trước khi bàn giao.
