# 09 — Hồ Sơ Giải Pháp Kỹ Thuật Chi Tiết: Adaptive Binary Drill-Down & Async Stateful Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Tác giả:** System Architect (Brain Role)  
> **Trạng thái:** PROPOSED  

---

## 1. Giải Pháp 1: Adaptive Binary Drill-Down Engine

### Thuật toán Chia Để Trị (Merkle Tree Bisection)
```
                  [30 Days Window: start -> end]
                                 │
                     Hash Match? ──► [YES] ──► (RETURN CLEAN / PRUNED)
                                 │
                               [NO]
                                 ▼
                     Split Mid: [start -> mid] & [mid -> end]
                                 │
         ┌───────────────────────┴───────────────────────┐
         ▼                                               ▼
[Branch Left: start -> mid]                     [Branch Right: mid -> end]
    Hash Match? ──► [YES] ──► (PRUNED)              Hash Match? ──► [NO]
                                                                     │
                                                                     ▼
                                                             Recursive Split...
```

### Golang Core Implementation Pseudocode:
```go
package recon

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

type DriftWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	SrcCount  int64     `json:"src_count"`
	DstCount  int64     `json:"dst_count"`
	SrcHash   uint64    `json:"src_hash"`
	DstHash   uint64    `json:"dst_hash"`
}

type BinaryDrillDownEngine struct {
	sourceAgent       SourceAgent
	destAgent         DestAgent
	minWindowDuration time.Duration // 15m
	maxDepth          int           // 12
}

func NewBinaryDrillDownEngine(src SourceAgent, dst DestAgent) *BinaryDrillDownEngine {
	return &BinaryDrillDownEngine{
		sourceAgent:       src,
		destAgent:         dst,
		minWindowDuration: 15 * time.Minute,
		maxDepth:          12,
	}
}

func (e *BinaryDrillDownEngine) ExecuteDrillDown(ctx context.Context, tableName string, startTime, endTime time.Time) ([]DriftWindow, error) {
	return e.drillDownRecursive(ctx, tableName, startTime, endTime, 0)
}

func (e *BinaryDrillDownEngine) drillDownRecursive(ctx context.Context, tableName string, start, end time.Time, currentDepth int) ([]DriftWindow, error) {
	var srcHash, dstHash uint64
	var srcCount, dstCount int64

	g, ctxGroup := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		srcHash, srcCount, err = e.sourceAgent.GetRangeHashAndCount(ctxGroup, tableName, start, end)
		return err
	})

	g.Go(func() error {
		var err error
		dstHash, dstCount, err = e.destAgent.GetRangeHashAndCount(ctxGroup, tableName, start, end)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch hash/count: %w", err)
	}

	// Pruning check
	if srcHash == dstHash {
		return nil, nil
	}

	// Base case
	if end.Sub(start) <= e.minWindowDuration || currentDepth >= e.maxDepth {
		return []DriftWindow{{
			StartTime: start,
			EndTime:   end,
			SrcCount:  srcCount,
			DstCount:  dstCount,
			SrcHash:   srcHash,
			DstHash:   dstHash,
		}}, nil
	}

	// Bisection
	midNano := start.UnixNano() + (end.UnixNano()-start.UnixNano())/2
	mid := time.Unix(0, midNano).UTC()

	var leftDrifts, rightDrifts []DriftWindow
	gSub, ctxSub := errgroup.WithContext(ctx)

	gSub.Go(func() error {
		var err error
		leftDrifts, err = e.drillDownRecursive(ctxSub, tableName, start, mid, currentDepth+1)
		return err
	})

	gSub.Go(func() error {
		var err error
		rightDrifts, err = e.drillDownRecursive(ctxSub, tableName, mid, end, currentDepth+1)
		return err
	})

	if err := gSub.Wait(); err != nil {
		return nil, err
	}

	res := make([]DriftWindow, 0, len(leftDrifts)+len(rightDrifts))
	res = append(res, leftDrifts...)
	res = append(res, rightDrifts...)
	return res, nil
}
```

---

## 2. Giải Pháp 2: Async Stateful Job Workflow

### Dynamic Flow Chart:
1. `POST /api/reconciliation/check-async` $\rightarrow$ Handler tạo record `recon_jobs` (ID `job_xxx`, Status `PENDING`), publish event NATS `cdc.event.recon.job_created` $\rightarrow$ Trả về `202 Accepted` (50ms).
2. Worker nền nhận NATS event $\rightarrow$ Đổi Status thành `RUNNING`, cập nhật `progress_percent` $\rightarrow$ Chạy `BinaryDrillDownEngine`.
3. Khi hoàn tất $\rightarrow$ Đổi Status thành `COMPLETED`, lưu `result_summary` JSONB.
4. Client polling `GET /api/reconciliation/jobs/job_xxx` để hiển thị progress bar & danh sách lệch.
