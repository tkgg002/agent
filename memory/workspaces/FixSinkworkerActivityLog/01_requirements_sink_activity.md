# Yêu cầu: Bổ sung Ghi nhận Activity Log 'sink-upsert' cho CDC Pipeline

## 1. Mô tả yêu cầu
CDC pipeline của `cdc-worker` (chạy qua `cmd/worker/main.go`) thực hiện ghi dữ liệu CDC từ Kafka vào các bảng shadow DB thông qua `BatchBuffer`. Hiện tại, hoạt động này không được ghi nhận vào bảng `cdc_system.cdc_activity_log`, dẫn đến việc UI CMS tại trang `/activity-log` không hiển thị được tiến trình đồng bộ dữ liệu CDC.

Yêu cầu:
- Bổ sung việc ghi nhận log hoạt động `sink-upsert` vào bảng `cdc_system.cdc_activity_log` khi thực hiện flush/upsert dữ liệu CDC trong `BatchBuffer.batchUpsert`.
- Log phải được ghi nhận gom nhóm (1 dòng log duy nhất cho cả batch trên từng bảng shadow) để tối ưu hiệu năng và tránh gây nghẽn/lằng nhằng dữ liệu log.
- Cập nhật trạng thái `Complete` hoặc `Fail` tương ứng dựa trên kết quả thực thi batch.

## 2. Definition of Done (DoD)
- Code biên dịch thành công (`go build ./cmd/worker/...`).
- Chạy unit tests cho handler shadow pass (`go test -v ./internal/handler/shadow/...`).
- Log được ghi nhận thành công vào bảng `cdc_system.cdc_activity_log` với operation là `sink-upsert`.
