# 03 — Thiết Kế Kỹ Thuật Chi Tiết: Adaptive Binary & Async Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Trạng thái:** TECHNICAL DESIGN APPROVED  

---

## 1. Kiến Trúc Lớp (Layered Architecture)

```
[ HTTP Client / Frontend ]
       │
       │ POST /api/reconciliation/check-async (HTTP 202 Accepted)
       ▼
[ ReconHandler ] ──── (Insert PENDING Job) ────► [ DB: cdc_system.recon_jobs ]
       │
       │ Pub Event: cdc.event.recon.job_created
       ▼
[ NATS JetStream / Queue ]
       │
       │ Sub Event (Async)
       ▼
[ ReconJobWorker ] ─── (Update RUNNING / Progress) ───► [ DB: cdc_system.recon_jobs ]
       │
       │ Execute Drill-Down
       ▼
[ BinaryDrillDownEngine ]
   ├── [ SourceAgent (Mongo) ] ── GetRangeHashAndCount
   └── [ DestAgent (Postgres) ] ── GetRangeHashAndCount
```

---

## 2. Thiết Kế Chi Tiết Thuật Toán Adaptive Binary Drill-Down

```go
// Data structures
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
    maxDepth          int           // 12 levels
}
```

### Các Bước Thực Thi Đệ Quy:
1. `GetRangeHashAndCount`: Khởi tạo 2 goroutine qua `errgroup` để tính Hash & Count từ Source và Dest đồng thời.
2. `Pruning Check`: If `SrcHash == DstHash` $\Rightarrow$ Return `nil, nil` (Nhánh sạch 100%).
3. `Base Case Check`: If `end.Sub(start) <= minWindowDuration` hoặc `depth >= maxDepth` $\Rightarrow$ Trả về `DriftWindow` đại diện cho lá bị lệch.
4. `Bisection`: Tính điểm giữa `mid = start + (end - start)/2`.
5. `Sub-branches`: Gọi đệ quy nhánh trái $[start, mid]$ và nhánh phải $[mid, end]$ song song.
6. `Merge`: Tổng hợp kết quả từ 2 nhánh.

---

## 3. Thiết Kế Schema Database (`cdc_system.recon_jobs`)

```sql
CREATE TABLE IF NOT EXISTS cdc_system.recon_jobs (
    job_id VARCHAR(64) PRIMARY KEY,
    target_table VARCHAR(128) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    progress_percent INT DEFAULT 0,
    total_diff_count BIGINT DEFAULT 0,
    checkpoint_ts TIMESTAMPTZ,
    result_summary JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recon_jobs_status ON cdc_system.recon_jobs(status);
CREATE INDEX IF NOT EXISTS idx_recon_jobs_created_at ON cdc_system.recon_jobs(created_at DESC);
```
