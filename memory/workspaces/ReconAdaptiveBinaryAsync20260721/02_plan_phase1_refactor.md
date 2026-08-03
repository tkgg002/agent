# 02 — Kế Hoạch Roadmap Refactor Phase 1: Chunk-Based Stream-to-Bucket

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** Phase 1 Refactor  

---

## 🗺️ Roadmap Các Bước Thực Hiện

### Bước 1: Xây Dựng `ChunkStreamBucketEngine` (`recon_stream_bucket_engine.go`)
- Định nghĩa struct `ChunkStreamBucketEngine` thay thế cho `BinaryDrillDownEngine`.
- Cài đặt `ExecuteStreamBucketDrillDown(ctx, tableName, startTime, endTime)`.
- Cài đặt Outer Loop 1-day Chunking & Inner Loop 96 Go RAM Buckets.

### Bước 2: Bổ Sung Streaming Method Vào Source & Dest Agents
- Cập nhật Interface `SourceStreamAgent` và `DestStreamAgent`:
  `StreamRangeData(ctx, table, start, end, callbackFunc)`.
- Triển khai Index Scan streaming trên Postgres (GORM Rows / Cursor) và Mongo (Cursor Stream).

### Bước 3: Cập Nhật Background Worker `recon_job_worker.go`
- Chuyển đổi Worker từ việc gọi `BinaryDrillDownEngine` sang gọi `ChunkStreamBucketEngine`.
- Đảm bảo State Machine (`PENDING` -> `RUNNING` -> `COMPLETED`) và Checkpoint logging trơn tru.

### Bước 4: Viết Unit Test Suite (`recon_stream_bucket_engine_test.go`)
- Test Case 1: Match 100% (30 ngày).
- Test Case 2: Sparse Drift (Lệch đúng 1 sub-window 15m ở ngày thứ 14).
- Test Case 3: Boundary Skew (`00:00:00.000` nằm đúng Ngày 2).
- Test Case 4: Resumable Checkpoint.
