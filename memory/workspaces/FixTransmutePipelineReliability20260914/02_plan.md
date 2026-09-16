# Roadmap Cao Tầng: Fix Transmute Pipeline Reliability

## Mục Tiêu
Khắc phục hiện tượng rớt trigger transmute từ CDC và tối ưu hoá hệ thống chống timeout/deadlock, giúp pipeline Shadow -> Master chạy ổn định, giảm thiểu phụ thuộc vào Recon Heal.

## Các Giai Đoạn Triển Khai

### Phase 1: Nâng cấp Observability & Error Handling NATS (Hot-Path Ingest)
- Tăng timeout tra cứu master binding trong `transmute_handler.go` từ 5s lên 10s.
- Bắt lỗi NATS publish `cdc.cmd.transmute` và ghi nhận metric khi publish thất bại (chấm dứt tình trạng nuốt lỗi `_ =`).
- Bổ sung metric và trace khi không tìm thấy master binding hoặc lookup DB gặp lỗi.

### Phase 2: Nâng cấp TableDebouncer (Gom Lô & Chống Nghẽn)
- Bổ sung cảnh báo warn và metric `__backpressure_<master>` khi hàng đợi tasks đạt ngưỡng tải (`len >= maxSize * 2`).
- Bảo vệ channel `flushCh` bằng cơ chế non-blocking send kèm ghi nhận `__flush_drop_<master>` để không bị treo worker khi downstream tắc nghẽn.
- Import thư viện metric nội bộ vào `debounce.go`.

### Phase 3: Cơ Chế Tự Phục Hồi Transmute Scheduler (Cron & Heartbeat)
- Bổ sung ticker định kỳ (10 phút/lần) trong `transmute_scheduler.go` để quét và giải phóng các schedule bị kẹt ở trạng thái `running` quá 30 phút mà không cần đợi restart service.

### Phase 4: Khắc Phục Bottleneck Recon Smoke Scan
- Thay thế hàm scan chính xác `CountRows()` (full table scan gây timeout 30s) bằng `EstimatedCountRows()` (O(1) qua `pg_stat`) trong `recon_smoke.go`.

### Phase 5: Hạ Tầng Database Index
- Tạo chỉ mục `CONCURRENTLY` cho các trường timestamp nghiệp vụ (`updatedAt`) trên cả hai bảng Shadow và Master của các quan hệ dữ liệu lớn (`bank_requests_bvb`, `order_updates_bvb`).

### Phase 6: Kiểm Thử & Nghiệm Thu
- Build kiểm tra toàn bộ service (`go build ./internal/...`).
- Chạy unit tests cho handler, scheduler và debouncer.
