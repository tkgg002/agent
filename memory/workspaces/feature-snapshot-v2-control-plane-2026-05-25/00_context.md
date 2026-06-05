# Context: Snapshot V2 Control Plane

## Business Objective
Triển khai hệ thống Control Plane cho luồng Snapshot V2 chuẩn Enterprise, nhằm đảm bảo khả năng kiểm soát tải, theo dõi tiến độ, xử lý lỗi tự động và bảo toàn tính toàn vẹn dữ liệu khi có concurrent events từ luồng CDC. 

## Vấn đề hiện tại
Luồng snapshot khi được kích hoạt với lượng dữ liệu lớn (như 50 triệu bản ghi) đang chạy tự do, không có các công cụ kiểm soát an toàn (không phanh, không giới hạn tải). Điều này có nguy cơ gây quá tải RAM/CPU cho Worker và DB.

## Yêu cầu cốt lõi (Core Requirements)
1. **Flow Control**: Cung cấp khả năng Pause/Resume (qua cờ trạng thái), Dynamic Batch Tuning, và Rate Limiting (MaxRPS).
2. **Resiliency**: Checkpoint bền vững (`last_id` / `$clusterTime`) và tính toán % tiến độ thực tế (Progress %).
3. **Fail-Safe**: Circuit Breaker (Auto-Pause khi tỷ lệ lỗi cao) và Dead Letter Queue (DLQ) cho bản ghi lỗi (chế độ Lenient/Strict).
4. **Data Integrity**: LWW Guard (Last Write Wins) để tránh ghi đè dữ liệu CDC mới hơn bằng dữ liệu Snapshot cũ, thông qua trường `source_ts_ms` và nhãn `_source: "snapshot:v2"`.
