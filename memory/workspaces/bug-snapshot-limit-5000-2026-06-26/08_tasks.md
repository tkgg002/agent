# Tasks: Bug Snapshot Limit 5000 Records

## Task: Fix MongoDB Numeric ID Pagination in Snapshot V2
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: centralized-data-service
- **Mô tả**: Sửa lỗi snapshot v2 chỉ đồng bộ 5000 bản ghi trên nguồn MongoDB có `_id` kiểu số (int32/int64/float64). Cần lấy mẫu `_id` để xác định kiểu dữ liệu thực tế và ép kiểu cho chuỗi `lastSeen` trong `buildResumeFilterWithSample`.
- **Trạng thái**: [x] DONE

### [Context]
- **Current state**: Kế hoạch đã được duyệt tại `implementation_plan.md`.
- **Dependencies**: `snapshot_runner_handler.go`, `snapshot_runner_utils.go`.
- **Logs/Error**: 
  `completed {"batches_total":2,"component":"snapshot_runner","rows_total":5000,"source_object_id":76,"target_table":"payment_bills"}`

### [Definition of Done]
- [x] Implement `buildResumeFilterWithSample(lastSeen string, sampleID interface{}) bson.M` trong `snapshot_runner_utils.go`.
- [x] Lấy sample document từ MongoDB collection trước khi vào loop cursor trong `snapshot_runner_handler.go` để lấy `sampleID` của `_id`.
- [x] Cập nhật loop cursor gọi `buildResumeFilterWithSample` thay vì `buildResumeFilter`.
- [x] **[QA Gate]**: Chạy thử nghiệm thành công:
  ```bash
  go test -v ./internal/handler/orchestration/...
  ```
- [x] **[Security Gate]**: Chạy rà soát bảo mật `/security-agent` (kiểm tra rà soát lỗ hổng và rủi ro cast kiểu dữ liệu an toàn - Đạt yêu cầu).
- [x] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.
