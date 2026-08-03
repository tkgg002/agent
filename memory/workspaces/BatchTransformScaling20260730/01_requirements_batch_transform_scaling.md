# 01_requirements_batch_transform_scaling.md

## Yêu cầu chi tiết: Nâng cấp Batch Transform Engine (50M–500M Records)

### Bài toán
1. **Timeout 15 phút**: `HandleBatchTransform` hiện chạy **blocking** trực tiếp trên goroutine NATS subscription. Với bảng 50M+ records, job kéo dài > 15 phút → NATS reconnect timeout → job bị kill mid-stream.
2. **Hard cap maxIterations=100000**: 500M records / chunk 1000 = 500,000 iterations → bị cắt ngang.
3. **Không có progress tracking**: UI không biết transform đang ở % nào.
4. **Không có Pause/Cancel**: Goroutine chạy tới chết, không có cơ chế dừng.
5. **Chunk size cứng 1000**: Không adaptive với tải DB.

### Yêu cầu chức năng
- **FR1**: Dispatch transform không block NATS → spawn goroutine độc lập ngay khi nhận msg.
- **FR2**: Tái dùng `cdc_system.recon_jobs` để track trạng thái (job_id, status, progress_percent, rows_affected).
- **FR3**: Cập nhật `progress_percent` vào DB mỗi N chunks (heartbeat pattern từ ReconJobWorker).
- **FR4**: Xóa hard cap `maxIterations=100000`, thay bằng loop thoát khi `chunkCount == 0`.
- **FR5**: `recon_jobs` cần thêm cột `job_type` VARCHAR(32) để phân biệt `transform` vs `recon`.
- **FR6**: Adapter NATS payload: thay `[]byte(targetTable)` → JSON `{job_id, target_table}`.
- **FR7**: CMS API: endpoint GET `/api/v1/source-objects/:id/transform/status` trả job progress từ `recon_jobs`.
- **FR8**: CMS FE: Progress Bar + nút Cancel trên `TableRegistry.tsx`.

### Yêu cầu phi chức năng
- Tái sử dụng `ReconJobRepo`, `ReconJobRepository` interface hiện có — không tạo repo mới.
- Tuân thủ pattern `ReconJobWorker` (spawn goroutine trong NATS handler, update status RUNNING/COMPLETED/FAILED).
- Dynamic chunk size: tăng/giảm dựa trên thời gian thực thi mỗi chunk (100ms / 1000ms threshold).
