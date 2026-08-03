# 08 — Danh Mục Task Refactor Phase 1: Chunk-Based Stream-to-Bucket

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** Phase 1 Refactor  

---

- [x] `TASK-REF-1.1`: Tạo `internal/service/recon/recon_stream_bucket_engine.go` triển khai struct `ChunkStreamBucketEngine` và logic 3 Tầng.
- [x] `TASK-REF-1.2`: Cập nhật `SourceStreamAgent` & `DestStreamAgent` interfaces và implementations trong `recon_source_agent.go` & `recon_dest_agent.go` hỗ trợ streaming `[start, end)` với `UnixMilli()`.
- [x] `TASK-REF-1.3`: Refactor `recon_job_worker.go` tích hợp `ChunkStreamBucketEngine` thay thế cho bisection engine cũ.
- [x] `TASK-REF-1.4`: Tạo Unit Test Suite `recon_stream_bucket_engine_test.go` kiểm thử 100% Match, Sparse Drift, Boundary Skew (`00:00:00.000`), và Checkpointing.
- [x] `TASK-REF-1.5`: Thực hiện `go build ./...` và `go test -v ./internal/service/recon/...` PASS 100%.
