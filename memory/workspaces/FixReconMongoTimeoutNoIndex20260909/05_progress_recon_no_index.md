# 05_progress_recon_no_index.md

## Audit Log (Append-Only)

- [2026-09-09 16:15:00] [Brain:Gemini-3.8-Flash] Khởi tạo workspace `FixReconMongoTimeoutNoIndex20260909`.
- [2026-09-09 16:15:30] [Brain:Gemini-3.8-Flash] Root Cause Analysis:
  - Lỗi `context deadline exceeded (30s)` phát sinh tại `recon.source.get_range_hash_and_count` (`ChunkStreamBucketEngine.checkDayChunk`).
  - `sa.cfg.QueryTimeout` mặc định là 30s (`recon_models.go:128`).
  - Source MongoDB không có index trên `timestampField` → MongoDB thực hiện `COLLSCAN` trên 15,000,000 documents.
  - Quét tuần tự 15M records vượt quá ngưỡng 30s → Context bị cancel.
- [2026-09-09 16:16:00] [Brain:Gemini-3.8-Flash] Adversarial Audit:
  - Phát hiện rủi ro nghiêm trọng của đề xuất cũ (chỉ đổi sang `_id: ObjectId`): Gây False Drift trên các bảng có in-place UPDATE và gây crash/0 docs nếu `_id` là String/UUID.
  - Chốt phương án cải tiến 3 lớp: Dynamic Timeout & BatchSize + Safe ObjectId Guard + Optimized Mongo Scan.
