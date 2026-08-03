# 03 — Thiết Kế Kỹ Thuật Chi Tiết Refactor Phase 1

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** Phase 1 Refactor  

---

## 1. Struct & Interface Specifications

```go
package recon

import (
	"context"
	"time"
)

// StreamRecord represents a single record streamed from DB for hashing.
type StreamRecord struct {
	ID            string
	LastUpdatedAt time.Time
}

// StreamCallback is invoked for every row streamed from DB.
type StreamCallback func(rec StreamRecord) error

// SourceStreamAgent defines streaming capability for source dataset.
type SourceStreamAgent interface {
	StreamRangeData(ctx context.Context, tableName string, start, end time.Time, cb StreamCallback) error
}

// DestStreamAgent defines streaming capability for destination dataset.
type DestStreamAgent interface {
	StreamRangeData(ctx context.Context, tableName string, start, end time.Time, cb StreamCallback) error
}

// ChunkStreamBucketEngine implements Chunk-Based Stream-to-Bucket.
type ChunkStreamBucketEngine struct {
	sourceAgent SourceStreamAgent
	destAgent   DestStreamAgent
	chunkDays   int
	progressCb  ProgressCallback
}
```

---

## 2. Core Logic Workflow (`ExecuteStreamBucketDrillDown`)

1. **Outer Loop:**
   - Cắt dải $[startTime, endTime]$ thành các khoảng 1 ngày: `currStart` đến `currEnd = currStart.AddDate(0, 0, 1)`.
2. **Inner Loop (Go RAM Buckets):**
   - Khởi tạo `var srcBuckets [96]uint64` và `var dstBuckets [96]uint64`.
   - Streaming Source $\rightarrow$ Với mỗi record, `tsMilli := rec.LastUpdatedAt.UnixMilli()`, `idx := (tsMilli - currStartMilli) / (15 * 60 * 1000)`, `srcBuckets[idx] ^= hashRow(rec.ID, tsMilli)`.
   - Streaming Dest $\rightarrow$ Thực hiện tương tự cho `dstBuckets[idx]`.
3. **In-Memory Check:**
   - Duyệt `for i := 0; i < 96; i++`.
   - Nếu `srcBuckets[i] != dstBuckets[i]` $\rightarrow$ Tạo `DriftWindow` cho mốc 15 phút thứ `i` của ngày đó.
4. **Checkpoint Update:**
   - Gọi `progressCb(ctx, currEnd, dayIndex)` để ghi nhận `checkpoint_ts`.
