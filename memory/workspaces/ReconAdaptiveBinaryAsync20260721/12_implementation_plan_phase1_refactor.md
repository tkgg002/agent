# 12 — Kế Hoạch Triển Khai Chi Tiết Cho Muscle: Refactor Phase 1

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** Phase 1 Refactor — Chunk-Based Stream-to-Bucket  
> **Trạng thái:** PROPOSED — AWAITING USER APPROVAL TO DELEGATE  

---

## 1. Các File Cần Tạo / Sửa Đổi Cho Muscle Sub-agent

1. **[NEW] [recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go):**
   - Triển khai `ChunkStreamBucketEngine` với Outer Loop (1-day Chunks), Inner Loop (Streaming & 96 Go RAM Buckets), và In-Memory Comparison.
2. **[MODIFY] [recon_job_worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go):**
   - Inject `ChunkStreamBucketEngine` vào worker.
3. **[NEW] [recon_stream_bucket_engine_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine_test.go):**
   - Unit test suite cover 100% Match, Sparse Drift, Boundary Skew, và Resumable Checkpoint.

---

## 2. Các Bước Thực Thi
1. Muscle đọc thiết kế trong `03_implementation_phase1_refactor.md` và `09_tasks_solution_chunk_stream_bucket.md`.
2. Sửa/Tạo mã nguồn Go.
3. Run `go build ./...` và `go test -v ./internal/service/recon/...`.
