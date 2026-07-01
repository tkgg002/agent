# Implementation Plan: Recon Self-Healing Fix

## Mục tiêu
1. Phân tích nguyên nhân tại sao Debezium không nhận tín hiệu incremental snapshot hoặc ignore tín hiệu.
2. Kiểm tra định dạng signal (payload.Filter, Kafka message key, value) và so khớp với yêu cầu của Debezium MongoDB connector.
3. Sửa lỗi SQL syntax error tại `recon_engine_run.go:38` nếu có.
4. Đảm bảo toàn bộ luồng upstream đồng bộ trơn tru và không bị gián đoạn.

## Các bước thực hiện
- [ ] Bước 1: Điều tra cấu hình Debezium Connector thực tế và định dạng signal (Kafka message key, Kafka topic, payload).
- [ ] Bước 2: Kiểm tra kiểu dữ liệu của `_id` trong MongoDB và format filter tương ứng cho MongoDB connector.
- [ ] Bước 3: Sửa lỗi format filter trong `debezium_signal.go` và `recon_heal_v4.go` nếu không khớp chuẩn MongoDB.
- [ ] Bước 4: Kiểm tra và sửa lỗi SQL Syntax Error tại `recon_engine_run.go:38` (nếu có).
- [ ] Bước 5: Verify toàn bộ luồng hoạt động bằng cách chạy local và kiểm tra logs.
