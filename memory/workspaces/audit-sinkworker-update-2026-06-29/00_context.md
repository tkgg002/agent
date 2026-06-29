# Context: Audit SinkWorker Update / Bối cảnh: Đánh giá và so sánh SinkWorker và SinkWorker Backup

## 1. Goal / Mục tiêu
Đánh giá chi tiết các thay đổi (update) trong thư mục `internal/sinkworker` so với phiên bản backup trong thư mục `internal/sinkworker_bk`.
Phân tích những cải tiến, sự khác biệt, tác động đến hiệu năng, độ tin cậy, tính bảo mật, và tính tuân thủ với cấu trúc kiến trúc tổng thể.

## 2. Scope / Phạm vi
- So sánh các file nguồn trong `internal/sinkworker/` với `internal/sinkworker_bk/`.
- Rà soát các cấu trúc dữ liệu, thuật toán, cache, và SQL logic được chỉnh sửa.
- Đảm bảo tính toàn vẹn của logic CDC (Change Data Capture) / Debezium và hệ thống Fencing Guard.

## 3. Reference Files / Tài liệu liên quan
- `internal/sinkworker/`
- `internal/sinkworker_bk/`
- `agent/memory/global/lessons.md`
