---
tier: T2
suffix: missing_core_stream
---

# 01 Requirements — Kiểm tra lỗi `MISSING_CORE_STREAM` (`CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) trên `/nats-monitor` (`https://cdc-cms.goopay.vn/nats-monitor`)

## 1. Bối cảnh & Yêu cầu của Anh
Trên trang `https://cdc-cms.goopay.vn/nats-monitor`, mục **"Chẩn đoán Kiến trúc NATS (3 phát hiện cần lưu ý)"** hiển thị:
- `MISSING_CORE_STREAM[CDC_TRANSMUTE]` — Thiếu Core Stream bắt buộc `'CDC_TRANSMUTE'` trên NATS JetStream
- `MISSING_CORE_STREAM[SCHEMA_DRIFT]` — Thiếu Core Stream bắt buộc `'SCHEMA_DRIFT'` trên NATS JetStream
- `MISSING_CORE_STREAM[SCHEMA_CONFIG]` — Thiếu Core Stream bắt buộc `'SCHEMA_CONFIG'` trên NATS JetStream

Trong khi thực tế trên PROD, luồng `CDC_TRANSMUTE` vẫn đang bắn qua và chạy bình thường. Anh yêu cầu kiểm tra tại sao `/nats-monitor` lại có hiện tượng này và thiết kế giải pháp xử lý triệt để.

## 2. Phạm vi Kiểm tra & Giải quyết (Scope)
1. Truy vết chính xác `file:line` nơi sinh ra cảnh báo `MISSING_CORE_STREAM` (trong `cdc-cms-web` và/hoặc `cdc-cms-service` / `centralized-data-service`).
2. Kiểm tra cơ chế lấy danh sách Stream từ NATS Monitoring endpoint (`/jsz?...`) hoặc JetStream API, cũng như cơ chế `EnsureStreams` / cấu hình kết nối NATS của `cdc-cms-service` vs `centralized-data-service` trên PROD.
3. Giải thích chính xác vì sao `CDC_TRANSMUTE` vẫn bắn qua và chạy bình thường trong khi `/nats-monitor` báo thiếu 3 stream `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG` (mà không báo thiếu `CDC_EVENTS` hoặc chỉ thấy 1 phần).
4. Lập phương án kỹ thuật tối ưu duy nhất (`09_tasks_solution_missing_core_stream.md`) kèm Code Demo và kiểm định qua `/laya_router` (`DG1/DG2`) trước khi trình anh `APPROVE`.
