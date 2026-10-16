# Kế Hoạch Triển Khai Chi Tiết (Implementation Plan): Fix Scan Array NATS Timeout

## 1. Mục tiêu
Giải quyết dứt điểm lỗi 503 Service Unavailable / NATS Timeout khi gọi API `/api/introspection/scan-array/trans_his` bằng cách đặt trần truy vấn an toàn và tối ưu hóa thời gian xử lý.

## 2. Kế hoạch từng bước

### Bước 1: Điều tra trạng thái tiến trình Worker trên Server
- Xác định trạng thái của worker (`centralized-data-service`) trên server staging/production (`10.200.198.100` / k8s cluster).
- Kiểm tra logs: xem worker có log nhận `cmd: scan_array_fields` với table `trans_his` không.
  - Nếu có log -> Worker bị nghẽn DB truy vấn vượt 15s.
  - Nếu không có log -> Worker pod đang down / không có subscriber nào trên queue group `cdc-worker-group`.

### Bước 2: Tối ưu hóa `centralized-data-service/internal/service/source/scan_service.go`
- Áp dụng trần bảo vệ `LIMIT 2000` cho trường hợp `mode == "full"` (hoặc bất kỳ mode nào không có limit).
- Thiết lập context timeout rõ ràng cho query database (10 giây) để tránh tình trạng query DB treo vô hạn.
- Đảm bảo nếu DB timeout thì trả về error message có cấu trúc thay vì để NATS ngắt kết nối âm thầm.

### Bước 3: Tối ưu hóa `cdc-cms-service/internal/api/system/introspection_handler.go`
- Trong `ScanArray` và `ScanArrayAll`, chuyển `mode` mặc định sang `"sample"`.
- Giữ timeout RPC ở mức 15-20s để tạo khoảng đệm an toàn sau khi query DB đã được tối ưu xuống < 2s.

### Bước 4: Kiểm thử và xác thực (Verification)
- Chạy unit test / integration test của `scan_service_test.go` và `introspection_handler_test.go`.
- Compile build binary kiểm tra cú pháp và kiểu dữ liệu.
- Đo thời gian phản hồi: đảm bảo query trên bảng lớn trả về kết quả trong vòng < 500ms.

## 3. Quy trình Governance & Phê duyệt
- Vai trò Brain: Đã lập kế hoạch và hồ sơ giải pháp chi tiết tại `09_tasks_solution_scan_array_timeout.md`.
- Trình User phê duyệt phương án trước khi chuyển giao Muscle thực thi chỉnh sửa mã nguồn.
