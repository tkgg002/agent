# 09 — Hồ Sơ Giải Pháp Kỹ Thuật: Adaptive Binary Drill-Down & Async Stateful Job

> **Workspace:** `ReconAuditPaymentBills20260720`  
> **Tác giả:** System Architect (Brain Role)  
> **Trạng thái:** PROPOSED — CHỜ APPROVE ĐỂ DELEGATE MUSCLE  

---

## 1. Đánh Giá Kiến Trúc Tổng Thể (Architectural Evaluation)

### 1.1 Giải pháp 1: Adaptive Binary Drill-Down (Chia để trị - Merkle Tree Hash Bisection)
* **Nhận định:** ĐÂY LÀ GIẢI PHÁP ĐÚNG ĐẮN VÀ TỐI ƯU NHẤT CHO NGHỆP VỤ RECONCILLATION TRÊN BIG DATA.
* **Cơ sở lý thuyết:** Thay vì chia cố định $N$ sub-windows ($O(N)$ queries), ta áp dụng Cây Hash (Merkle Tree Bisection) với độ phức tạp $O(K \cdot \log N)$ (trong đó $K$ là số lượng cửa sổ thực sự có dữ liệu lệch).
  - Khi dữ liệu 100% khớp: Chỉ tốn đúng **1 query (Global Hash Check)**.
  - Khi có 1 lỗi nằm rải rác: Tốn $\approx \log_2(2880) \approx 12$ queries.
  - **Mức độ giảm tải DB:** **95% - 99.6%** số lượng query không cần thiết.

* **Bẫy kỹ thuật (Tripwires & Edge Cases) cần xử lý:**
  1. **Điều kiện biên thời gian (Boundary Handling):** Cắt `[start, end]` thành `[start, mid]` và `[mid, end]`. Query DB phải tuân thủ nghiêm ngặt `lastUpdatedAt >= start AND lastUpdatedAt < mid` để tránh tính 2 lần các bản ghi nằm đúng mốc `mid`.
  2. **Hash Precision & Consistency:** Cả Source DB (Mongo) và Dest DB (Postgres) phải dùng cùng 1 thuật toán Hash (ví dụ XOR XXHash64 trên ID + Timestamp UTC chuẩn hóa).
  3. **Concurrency Optimization:** Hai nhánh con `[start, mid]` và `[mid, end]` hoàn toàn độc lập $\rightarrow$ có thể chạy song song (Parallel execution) qua `golang.org/x/sync/errgroup` để tận dụng I/O async.
  4. **Max Depth / Stop Criterion:** Đệ quy phải dừng khi:
     - Khớp Hash (Return `nil`).
     - Thời gian window $\le$ `minWindowDuration` (ví dụ 15 phút).
     - Độ sâu đệ quy $\ge$ `maxDepth` (để tránh tràn stack nếu có lỗi logic).

---

### 1.2 Giải pháp 2: Asynchronous Batch Job & State Machine
* **Nhận định:** BẮT BUỘC Phải chuyển từ Synchronous HTTP Request sang Asynchronous Event-Driven Job để chống HTTP 504 Gateway Timeout.
* **Cấu trúc Bảng Quản Lý Job (`recon_jobs`):**
  ```sql
  CREATE TABLE cdc_system.recon_jobs (
      job_id VARCHAR(64) PRIMARY KEY,
      target_table VARCHAR(128) NOT NULL,
      start_time TIMESTAMPTZ NOT NULL,
      end_time TIMESTAMPTZ NOT NULL,
      status VARCHAR(32) NOT NULL, -- PENDING, RUNNING, COMPLETED, FAILED
      progress_percent INT DEFAULT 0,
      total_diff_count BIGINT DEFAULT 0,
      checkpoint_ts TIMESTAMPTZ,
      result_summary JSONB,
      error_message TEXT,
      created_at TIMESTAMPTZ DEFAULT NOW(),
      updated_at TIMESTAMPTZ DEFAULT NOW()
  );
  ```
* **Luồng Vận Hành:**
  1. **Control Plane (API Gateway):** `POST /api/reconciliation/check` $\rightarrow$ Insert `recon_jobs` (PENDING), pub NATS Event `cdc.event.recon.job_created`, trả về `202 Accepted` kèm `job_id` trong **< 50ms**.
  2. **Worker Engine (Background Consumer):** Nhận NATS Event $\rightarrow$ Đổi status `RUNNING` $\rightarrow$ Thực thi `AdaptiveBinaryDrillDown` $\rightarrow$ Lưu `COMPLETED` / `FAILED` vào DB.
  3. **Client UI (Polling/Websocket):** Call `GET /api/reconciliation/jobs/{job_id}` để render Progress Bar & hiển thị kết quả.

---

## 2. Mã Giả Golang (Golang Pseudo-Code): Adaptive Binary Drill-Down

```go
package recon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// DriftWindow đại diện cho cửa sổ bị phát hiện lệch dữ liệu
type DriftWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	SrcCount  int64     `json:"src_count"`
	DstCount  int64     `json:"dst_count"`
	SrcHash   uint64    `json:"src_hash"`
	DstHash   uint64    `json:"dst_hash"`
}

// BinaryDrillDownEngine chứa logic đệ quy cắt đôi
type BinaryDrillDownEngine struct {
	sourceAgent       SourceAgent
	destAgent         DestAgent
	minWindowDuration time.Duration // Ví dụ: 15 * time.Minute
	maxDepth          int           // Giới hạn đệ quy tối đa (VD: 10 levels)
	maxParallelism    int           // Số lượng goroutines song song tối đa
}

func NewBinaryDrillDownEngine(src SourceAgent, dst DestAgent) *BinaryDrillDownEngine {
	return &BinaryDrillDownEngine{
		sourceAgent:       src,
		destAgent:         dst,
		minWindowDuration: 15 * time.Minute,
		maxDepth:          10,
		maxParallelism:    4,
	}
}

// ExecuteDrillDown là entrypoint bắt đầu đệ quy từ toàn dải [startTime, endTime]
func (e *BinaryDrillDownEngine) ExecuteDrillDown(ctx context.Context, tableName string, startTime, endTime time.Time) ([]DriftWindow, error) {
	return e.drillDownRecursive(ctx, tableName, startTime, endTime, 0)
}

// drillDownRecursive thực hiện giải thuật Chia Để Trị (Bisection Merkle Tree)
func (e *BinaryDrillDownEngine) drillDownRecursive(
	ctx context.Context,
	tableName string,
	start, end time.Time,
	currentDepth int,
) ([]DriftWindow, error) {
	// 1. Lấy Hash & Count đồng thời từ cả Source (Mongo) và Dest (Postgres)
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
		return nil, fmt.Errorf("failed to fetch hash/count range [%s - %s]: %w", start.Format(time.RFC3339), end.Format(time.RFC3339), err)
	}

	// 2. PRUNING CHECK (Cắt tỉa nhánh an toàn)
	// Nếu Hash khớp HOẶC cả 2 bên đều rỗng (count = 0) -> Dữ liệu khớp 100%!
	if srcHash == dstHash {
		// Nhánh này hoàn toàn sạch, KHÔNG CẦN DRILL DOWN THÊM!
		return nil, nil
	}

	// 3. BASE CASE (Đã chạm ngưỡng tối thiểu 15m hoặc vượt maxDepth)
	windowDuration := end.Sub(start)
	if windowDuration <= e.minWindowDuration || currentDepth >= e.maxDepth {
		// Đây là cửa sổ lá (Leaf Window) bị DRIFT thật sự!
		leafDrift := DriftWindow{
			StartTime: start,
			EndTime:   end,
			SrcCount:  srcCount,
			DstCount:  dstCount,
			SrcHash:   srcHash,
			DstHash:   dstHash,
		}
		return []DriftWindow{leafDrift}, nil
	}

	// 4. RECURSIVE STEP (Chia đôi khoảng thời gian)
	midNano := start.UnixNano() + (end.UnixNano()-start.UnixNano())/2
	mid := time.Unix(0, midNano).UTC()

	var leftDrifts, rightDrifts []DriftWindow
	gSub, ctxSub := errgroup.WithContext(ctx)

	// Nhánh trái: [start, mid)
	gSub.Go(func() error {
		var err error
		leftDrifts, err = e.drillDownRecursive(ctxSub, tableName, start, mid, currentDepth+1)
		return err
	})

	// Nhánh phải: [mid, end)
	gSub.Go(func() error {
		var err error
		rightDrifts, err = e.drillDownRecursive(ctxSub, tableName, mid, end, currentDepth+1)
		return err
	})

	if err := gSub.Wait(); err != nil {
		return nil, err
	}

	// 5. Gộp kết quả lệch từ 2 nhánh
	result := make([]DriftWindow, 0, len(leftDrifts)+len(rightDrifts))
	result = append(result, leftDrifts...)
	result = append(result, rightDrifts...)

	return result, nil
}
```

---

## 3. Lộ Trình Phối Hợp Triển Khai (Roadmap)
1. **Brain:** Đã hoàn thành bản thiết kế kiến trúc và Pseudo-code Golang.
2. **User:** Review và đưa ra chỉ thị `APPROVE`.
3. **Muscle Sub-agent:** Thực thi code Golang chuẩn hóa vào `internal/service/recon/recon_bisection_engine.go`, viết Unit test và Integration test.
