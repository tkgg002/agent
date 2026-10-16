# Requirements: Khắc phục lỗi 503 Worker timeout (nats: timeout) khi Scan Array trên bảng trans_his

## 1. Hiện trạng & Bối cảnh
- Khi thực hiện gọi API: `POST https://cdc-cms-api.goopay.vn/api/introspection/scan-array/trans_his`
- Response trả về: `503 Service Unavailable` với nội dung `{"error":"Worker timeout or request error: nats: timeout"}`.

## 2. Yêu cầu chi tiết (Specs)
- **REQ-1:** Xác định chính xác luồng gọi và nguyên nhân gây NATS timeout giữa `cdc-cms-service` và `centralized-data-service` (worker).
- **REQ-2:** Khắc phục triệt để lỗ hổng hiệu năng trong `ScanArrayFields` (`centralized-data-service`):
  - Hiện tại `cdc-cms-service/internal/api/system/introspection_handler.go` đang truyền cứng `"mode": "full"`.
  - Trong `centralized-data-service/internal/service/source/scan_service.go`, khi `mode == "full"`, query SQL chạy full table scan không hề có `LIMIT` trên bảng lớn `trans_his`, duyệt tuần tự từng dòng và unmarshal JSON, dẫn đến thời gian thực thi vượt quá timeout 15s của NATS RPC.
  - Phải có cơ chế `LIMIT` an toàn (ví dụ mẫu 1,000 - 2,000 bản ghi tương tự `ScanRawData`) hoặc tối ưu hóa truy vấn để hoàn tất trong < 2 giây.
- **REQ-3:** Hướng dẫn kiểm tra trạng thái vận hành của tiến trình Worker (`centralized-data-service`) trên môi trường k8s/server để đảm bảo worker đang sống và đang lắng nghe `cdc.cmd.scan-array`.
- **REQ-4:** Đảm bảo khả năng tương thích ngược và không phá vỡ logic flatten discovery hiện có.
