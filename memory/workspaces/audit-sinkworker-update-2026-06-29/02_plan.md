# Plan: Audit SinkWorker Update / Kế hoạch: Đánh giá và so sánh SinkWorker và SinkWorker Backup

## Kế hoạch hành động

### Phase 1: Research & Discovery / Tìm hiểu & So sánh
- [ ] Thực hiện so sánh cấu trúc thư mục giữa `sinkworker` và `sinkworker_bk`.
- [ ] Sử dụng công cụ `diff` hoặc công cụ so sánh file của Go/shell để xác định các phần khác biệt lớn trong từng file (như `worker.go`, `schema_manager.go`, `sql_builder.go`, `utils.go`, `avro_decode.go`, `test_exports.go`).
- [ ] Đọc hiểu mã nguồn của các phần thay đổi để xác định mục đích của việc update này (Ví dụ: tối ưu hóa cache, thêm log, nâng cấp fencing logic, tái cấu trúc file test helpers).

### Phase 2: Analysis & Evaluation / Phân tích & Đánh giá
- [ ] Phân tích tác động hiệu năng (ví dụ: cải tiến locking, buffering, caching).
- [ ] Phân tích độ an toàn & tin cậy (fencing guard, transaction isolation, error handling).
- [ ] Phân tích tính tương thích ngược (Backward Compatibility) đối với các unit test và các service khác phụ thuộc.

### Phase 3: Reporting / Báo cáo
- [ ] Viết tài liệu đánh giá kỹ thuật chi tiết.
- [ ] Phản hồi cho người dùng bằng tiếng Việt, phân tích rõ ràng ưu/nhược điểm và các điểm cần lưu ý của bản update này.
