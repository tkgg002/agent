# 00 — Context: Workspace Audit cho ReconAdaptiveBinaryAsync20260721

> **Workspace Audit:** `ReconAuditWorkspace20260721`  
> **Workspace được Audit:** `ReconAdaptiveBinaryAsync20260721`  
> **Ngày thực hiện:** 2026-07-21  
> **Role thực hiện:** Brain (Architect & Chairman)  

---

## I. BỐI CẢNH & MỤC TIÊU AUDIT

Chiến dịch Refactor Big Data Reconciliation Engine tại workspace `ReconAdaptiveBinaryAsync20260721` nhằm giải quyết triệt để các hạn chế của hệ thống đối soát dữ liệu quy mô lớn (lookback >= 30 ngày) giữa MongoDB và PostgreSQL. 

Workspace này được khởi tạo độc lập nhằm kiểm tra, đánh giá toàn diện (Audit) chất lượng thiết kế kiến trúc, mã nguồn triển khai, tính tuân thủ quy trình Governance (G1–G8), và hiệu năng thực tế của chiến dịch nâng cấp đối soát dữ liệu.

---

## II. PHẠM VI AUDIT (AUDIT SCOPE)

### 1. Kiến trúc Engine lõi:
- **Phase 1:** Chunk-Based Stream-to-Bucket Engine (`ChunkStreamBucketEngine`) — Duyệt mảng 96 buckets RAM 15-phút theo từng chunk 1-ngày, đảm bảo bộ nhớ hằng số $O(1)$.
- **Phase 2:** Async Job Worker State Machine (`ReconJobWorker`) — Lắng nghe NATS topic `cdc.event.recon.job_created`, chuyển giao trạng thái `PENDING` → `RUNNING` → `COMPLETED`/`FAILED`, tích hợp OpenTelemetry tracing và lưu kết quả vào `cdc_reconciliation_report`.
- **Phase 3:** Single Adaptive Endpoint (`CheckHandler`) — Định tuyến tự động luồng Fast-path Sync (nếu $\le 2\text{h}$) hoặc Async Job Path (nếu $> 2\text{h}$), chốt cứng Watermarks với Lag Buffer 120s.

### 2. Các Thành Phần Code Cần Rà Soát:
- `internal/service/recon/recon_stream_bucket_engine.go`
- `internal/service/recon/recon_job_worker.go`
- `internal/repository/recon_job_repo.go`
- `internal/handler/recon/recon_check_handler.go`
- `internal/handler/recon/recon_job_handler.go`
- `internal/server/server_setup.go`
