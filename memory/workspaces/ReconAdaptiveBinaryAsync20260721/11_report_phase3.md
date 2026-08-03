# 11 — Báo Cáo Thay Đổi Mã Nguồn Phase 3 (Source Code Diff & Changes Overview)

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** 3 — Control Plane API Integration & Single Adaptive Endpoint  
> **Thời gian:** 2026-07-21  

---

## 1. Danh Sách File Tạo Mới & Chỉnh Sửa

| Tên File | Hành Động | Số Dòng | Mô Tả Thay Đổi |
| :--- | :--- | :--- | :--- |
| `internal/handler/recon/recon_job_handler.go` | **TẠO MỚI** | ~140 dòng | Triển khai `JobHandler` cho API Polling `GET /api/reconciliation/jobs/:job_id` (Gin & Fiber) và NATS command `cdc.cmd.recon-job-status`. |
| `internal/handler/recon/recon_check_handler.go` | **SỬA ĐỔI** | ~350 dòng | Tích hợp Fixed Immutable Bounds (Freeze Watermark) `resolveTimeRange` & Single Adaptive Endpoint Pattern (Sync Fast-path $\le 2\text{h}$ vs Async Job Path $> 2\text{h}$). |
| `internal/service/recon/recon_tier_a.go` | **SỬA ĐỔI** | +6 dòng | Export `GetScanRangeWithLag` cho `ReconCore` hỗ trợ tính toán mốc watermark cố định. |
| `internal/service/recon/recon_job_worker.go` | **SỬA ĐỔI** | +1 dòng | Bổ sung phương thức `Create` vào interface `ReconJobRepository`. |
| `internal/server/server_setup.go` | **SỬA ĐỔI** | +15 dòng | Đăng ký `ReconJobRepo`, `JobHandler`, NATS subscription `cdc.cmd.recon-job-status`, và Fiber GET `/api/reconciliation/jobs/:job_id`. |
| `internal/handler/recon/recon_job_handler_test.go` | **TẠO MỚI** | ~280 dòng | Unit Test Suite kiểm thử 100% Sync Fast-path, Async Job Path, và Polling Job Status qua Gin/Fiber/NATS. |
| `internal/service/recon/recon_job_worker_test.go` | **SỬA ĐỔI** | +7 dòng | Cập nhật `mockJobRepo` implement `Create`. |

---

## 2. Chi Tiết Thay Đổi Logic Nghiệp Vụ

1. **Khóa Mốc Cố Định (Fixed Immutable Bounds / Freeze Watermark)**:
   - Trong `recon_check_handler.go`, hàm `resolveTimeRange` tự động tính toán `upper = min(srcMax, dstMax) - lagBuffer` (mặc định 2 phút) và `lower = upper - lookback` (mặc định 24h hoặc từ payload) khi `start_time` / `end_time` không được truyền lên.

2. **Single Adaptive Endpoint Pattern (`POST /api/reconciliation/check`)**:
   - Nếu `rangeDuration <= 2h`: Thực thi **Sync Fast-path**, gọi `bisectionEngine.ExecuteDrillDown` trực tiếp, phản hồi `HTTP 200 OK` kèm danh sách `drifts`.
   - Nếu `rangeDuration > 2h`: Thực thi **Async Job Path**, ghi record `ReconJob` vào DB `cdc_system.recon_jobs` (trạng thái `PENDING`), publish NATS Event `cdc.event.recon.job_created`, phản hồi `HTTP 202 Accepted` kèm `job_id` và `status_url`.

3. **Job Polling Handler (`GET /api/reconciliation/jobs/:job_id`)**:
   - `JobHandler.HandleGetJobStatus` (Gin) và `HandleGetJobStatusFiber` (Fiber) truy vấn thông tin tiến độ `progress_percent`, `total_diff_count`, `checkpoint_ts`, `result_summary` JSONB, và `error_message`.
