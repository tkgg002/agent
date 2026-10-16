# 08_tasks — Audit miss record

## Brain (đã xong)
- [x] Đọc luồng Kafka → BatchBuffer → Shadow
- [x] Xác định F1-F8, loại giả thuyết sai
- [x] Viết `13_analysis`, `09_tasks_solution`

## Chờ User
- [ ] Chạy checklist xác minh production (mục 6 của `13_analysis`) và gửi kết quả
- [ ] APPROVE giải pháp `09_tasks_solution_miss_record_audit.md`

## Muscle (sau APPROVE) — Red → Green
- [ ] T1. Viết test RED: ép `batchUpsert` trả lỗi retryable (deadlock) → assert offset KHÔNG được commit qua record lỗi (hiện tại fail)
- [ ] T2. Sửa `FlushWithContext`: commit theo contiguous watermark chỉ gồm record đã persist hoặc đã DLQ bền vững
- [ ] T3. Retry có backoff cho lỗi retryable; hết retry → trả record về buffer, không commit
- [ ] T4. Không drop khi `Prepare`/`schema nil` — giữ lại + không commit
- [ ] T5. Serialize flush/commit (loại race timerLoop vs snapshot runner)
- [ ] T6. Backpressure cho `BatchBuffer.Add`
- [ ] T7. Debounce `RefreshTopics` + jitter, chỉ rebuild khi tập topic thật sự đổi ổn định
- [ ] T8. Chạy lại test suite hiện có (G5), đối soát count Source = Shadow trên dữ liệu thật (G6), `/security-agent` rà soát
