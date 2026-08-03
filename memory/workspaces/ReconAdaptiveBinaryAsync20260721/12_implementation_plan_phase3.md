# 12 — Kế Hoạch Triển Khai Chi Tiết Của AI Cho Phase 3

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** 3 — Control Plane API Integration  
> **Trạng thái:** PROPOSED — AWAITING USER APPROVAL TO DELEGATE MUSCLE WORK  

---

## 1. Các File Cần Tạo / Sửa Đổi Cho Muscle Sub-agent

1. **[NEW] [recon_job_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_job_handler.go):**
   - Định nghĩa `JobHandler` với phương thức `HandleGetJobStatus`.
2. **[MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go):**
   - Refactor `HandleReconCheck` tích hợp Single Adaptive Endpoint (Sync $\le 2\text{h}$, Async $> 2\text{h}$).
3. **[NEW] [recon_job_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_job_handler_test.go):**
   - Unit test suite cho Polling Handler và Single Adaptive Endpoint.

---

## 2. Các Bước Thực Thi
1. Muscle đọc kỹ spec trong `03_implementation_phase3.md` và `09_tasks_solution_phase3.md`.
2. Sửa code, chạy `go build ./...`.
3. Run unit tests `go test -v ./internal/handler/recon/...` & `go test -v ./internal/service/recon/...`.
