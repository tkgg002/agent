# 01 — Yêu Cầu Kỹ Thuật Refactor Phase 1: Chunk-Based Stream-to-Bucket

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** Phase 1 Refactor — Engine Transition  
> **Trạng thái:** DRAFT SPECIFICATION  

---

## 1. Mục Tiêu Refactor
Thay thế hoàn toàn Core Engine đệ quy Top-Down (`recon_bisection_engine.go`) bằng kiến trúc **`ChunkStreamBucketEngine` (Chunk-Based Stream-to-Bucket)** để triệt tiêu 100% rủi ro DB CPU Overload, $12 \times N$ I/O Read Amplification, và Network Blip/Long-lived Cursor.

---

## 2. Danh Mục Yêu Cầu Kỹ Thuật (Functional & Technical Specs)

### [FR-P1-REF-01] Outer Loop Chunking 1-Ngày & Resumable Checkpoint
- Hệ thống chẻ dải thời gian $[startTime, endTime]$ thành các Chunks 1-ngày độc lập ($[startDay, endDay)$).
- Sau mỗi Chunk 1-ngày xử lý xong, hệ thống tự động ghi Checkpoint (`checkpoint_ts` và `progress_percent`) vào `cdc_system.recon_jobs`.
- Nếu bị crash/restart, Engine tự khôi phục và tiếp tục chạy từ ngày kế tiếp.

### [FR-P1-REF-02] Inner Loop Streaming & $O(1)$ RAM Buckets
- DB Agent mở Stream bằng Index Scan nhẹ: `WHERE last_updated_at >= $1 AND last_updated_at < $2 ORDER BY last_updated_at ASC`.
- Bắt buộc dùng nửa khoảng $[start, end)$ triệt tiêu toán tử `BETWEEN`.
- Go App nhận stream, cắt đuôi `UnixMilli()`, phân bổ vào mảng tĩnh `srcBuckets[96]` và `dstBuckets[96]` (RAM $O(1) \approx 768\text{ bytes}$).
- Tích lũy bitwise XOR: `Buckets[idx] ^= hashRow(id, tsMilli)`.

### [FR-P1-REF-03] In-Memory Comparison ($< 0.001\text{ms}$)
- So sánh `srcBuckets[i] == dstBuckets[i]` cho 96 khay trên RAM Go.
- Khay nào lệch $\rightarrow$ Ghi nhận cửa sổ lá 15 phút bị drift.
