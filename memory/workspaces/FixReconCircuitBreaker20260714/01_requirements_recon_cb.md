# Yêu cầu: Khắc phục lỗi Circuit Breaker khi đối soát 7 ngày (source_shadow)

## Bối cảnh
Khi chạy đối soát (reconciliation check) 7 ngày cho chặng `source_shadow` của bảng `schedule_histories` (collection `schedule_histories` trong MongoDB `scheduler-service`), hệ thống gặp lỗi:
`source max ts: circuit breaker is open (CIRCUIT_OPEN)`

## Yêu cầu
1. Tìm hiểu nguyên nhân tại sao truy vấn lấy Max TS trên nguồn MongoDB bị timeout hoặc thất bại liên tục dẫn tới Circuit Breaker mở.
2. Kiểm tra chỉ mục (index) của collection `schedule_histories` trên MongoDB đối với trường `lastUpdatedAt`.
3. Đề xuất và triển khai giải pháp tối ưu:
   - Tạo chỉ mục thích hợp trên MongoDB cho `lastUpdatedAt` (nếu chưa có).
   - Xem xét tăng `BreakerThreshold` hoặc cấu hình timeouts nếu cần thiết để phù hợp với các truy vấn dài ngày.
   - Kiểm tra xem khi truyền custom range (như đối soát 7 ngày) thì có cần thiết gọi `MaxWindowTs` hay không, hoặc có cách nào để bypass/handle lỗi đó mượt mà hơn.
