# 13_analysis_batch_transform_scaling.md: Phân tích Kỹ thuật Nâng cấp Batch Engine 50M-500M Records

## 📌 1. Bối cảnh & Bài toán Thực tế

Khi vận hành CDC Data Hub với dung lượng dữ liệu lớn:
1. **Câu chuyện 1 (Hiệu năng 50M - 500M records)**:
   - Với 3 triệu bản ghi, chạy lặp qua các chunk (10,000 bản ghi/chunk) mất ~30 giây.
   - Nếu bảng tăng lên **50 triệu - 500 triệu bản ghi**, chạy đơn luồng tuần tự sẽ mất **hàng giờ**, phình WAL log Postgres và chiếm giữ kết nối DB quá lâu.
2. **Câu chuyện 2 (Cơ chế Async Background Job & Quản lý Timeout > 15 phút)**:
   - Cơ chế cũ gọi `batch-transform` trực tiếp trên luồng nhận NATS message. Khi dữ liệu lớn chạy > 15 phút, đứt kết nối NATS/HTTP.
   - Hệ thống thiếu cơ chế **Async Background Job Worker** để theo dõi trạng thái, tính `% progress`, và cho phép **Pause / Resume / Cancel** job từ UI.

---

## 🔍 2. Phân tích Chi tiết Lỗ hổng & Giải pháp Nâng cấp

### A. Chuyển đổi mô hình Sync sang Async Background Worker Engine
```
Luồng xử lý Transform mới:
1. Client gửi POST /api/v1/source-objects/:id/transform
2. CMS Service tạo recon_jobs record (job_id, status='pending', progress_percent=0)
3. CMS Service phát NATS event 'cdc.cmd.batch-transform' chứa job_id
4. CDC Worker nhận event -> Spawn Goroutine độc lập (non-blocking)
5. Worker cập nhật status='running', lặp theo chunk và bắn Heartbeat + Update % tiến độ vào DB sau mỗi N chunks.
6. Khi hoàn tất -> Đổi status='completed'.
```

### B. Giải pháp Tối ưu SQL cho DB 500M Records
1. **Keyset Pagination với Composite Partial Index**:
   Khuyến nghị tạo Partial Index cho bảng Shadow lớn:
   `CREATE INDEX CONCURRENTLY idx_<table_name>_pending_transform ON <schema>.<table_name> (_gpay_id) WHERE (_raw_data IS NOT NULL AND (<target_col_is_null>));`
   Giúp PostgreSQL chuyển từ Seq Scan sang **Index Only Scan** với latency < 5ms mỗi chunk!
2. **Dynamic Batch Size Tuning**:
   Thuật toán tự điều chỉnh chunk size: Nếu execution time < 100ms -> tăng chunk size từ 10,000 lên 20,000. Nếu execution time > 1000ms -> giảm chunk size xuống 5,000 để giải phóng lock table.

---

## 🏁 3. Lộ trình Triển khai (Roadmap)
- **Phase 1**: Nâng cấp `BatchTransformHandler` thành Async Goroutine Worker & Tích hợp `cdc_system.recon_jobs` tracking.
- **Phase 2**: Cập nhật CMS UI (`TableRegistry.tsx`) hỗ trợ thanh Progress Bar hiển thị % tiến độ transform real-time và nút Pause/Cancel.

---

## 📊 4. Audit Code Hiện Tại (batch_transform_handler.go)

### Vấn đề phát hiện:
1. **Blocking NATS Handler**: `HandleBatchTransform` chạy đồng bộ trên goroutine NATS subscription → timeout NATS sau 15 phút
2. **maxIterations = 100000**: Hard-cap cứng, 500M records / 1000 chunk_size = 500,000 iterations → bị cut-off!
3. **Không có progress tracking**: Không có cách nào biết % hoàn thành từ UI
4. **Không có Pause/Cancel**: Goroutine chạy tới chết, không thể dừng
5. **chunkSize = 1000 cứng**: Không adaptive với tải DB thực tế
