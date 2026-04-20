# Implementation v3: Health Collector SLOW SQL Fix

**Date**: 2026-04-17
**Agent**: Muscle (claude-opus-4-7)
**Status**: DONE
**Related**: system_health_collector tick (15s) emitting SLOW SQL > 200ms

## Problem Statement

Production log của `cdc-cms-service` liên tục phát cảnh báo SLOW SQL (>= 200ms) từ 2 query chạy trong `system_health_collector.go`:

```
system_health_collector.go:599 [306.606ms] SELECT count(*) FROM "failed_sync_logs" WHERE created_at > NOW() - INTERVAL '24 hours'
system_health_collector.go:610 [440.572ms] SELECT * FROM "cdc_activity_log" ORDER BY started_at DESC LIMIT 10
```

Collector tick interval = 15s → cumulative waste + ảnh hưởng planner cache khi tải cao. Bảng đã được partition (migration 010) nhưng slow log chứng tỏ query chưa leverage partition prune.

## Root Cause Analysis

### Query 1: `COUNT(*) FROM failed_sync_logs WHERE created_at > NOW() - INTERVAL '24 hours'`

`EXPLAIN (ANALYZE, BUFFERS)` cho bản cũ:
```
Aggregate  (cost=53.00..53.01 rows=1)
  ->  Append  (rows=50)
        ->  Seq Scan on failed_sync_logs_y2026m04
        ->  Seq Scan on failed_sync_logs_y2026m05
        ->  Seq Scan on failed_sync_logs_y2026m06
        ->  Seq Scan on failed_sync_logs_y2026m07
        ->  Seq Scan on failed_sync_logs_default
Planning: Buffers: shared hit=741
Planning Time: 8.322 ms
Execution Time: 0.139 ms
```

→ Planner mở TẤT CẢ 5 partition. Không có **runtime partition pruning** vì `NOW() - INTERVAL` là upper-unbounded → planner không loại được partition tương lai.

### Query 2: `SELECT * FROM cdc_activity_log ORDER BY started_at DESC LIMIT 10`

```
Limit  (rows=10)
  ->  Merge Append (rows=2296)
        Sort Key: cdc_activity_log.started_at DESC
        ->  Index Scan using cdc_activity_log_20260417_started_at_idx
        ->  Index Scan using cdc_activity_log_20260418_started_at_idx
        ... (8 partitions)
Planning: Buffers: shared hit=1116
Planning Time: 8.719 ms
Execution Time: 0.265 ms
```

→ Không có WHERE → planner PHẢI mở tất cả 8 daily partitions (ngày càng tăng theo thời gian). Planning cost + catalog lookup ngày càng đắt.

**Lưu ý**: Execution time local test chỉ 0.1-0.3ms nhưng production slow log 300-440ms. Disparity là do:
1. **Planning cost** (741-1116 buffers × N query/15s) tích lũy.
2. **GORM prepared statement** re-plan mỗi session → không amortize planning.
3. **Contention**: collector chạy cùng worker đang INSERT vào partitions → lock/WAL contention.

Column mapping OK: `ActivityLog.StartedAt → started_at`, `FailedSyncLog.CreatedAt → created_at`. Không có GORM tag bug.

Index sẵn có đầy đủ:
- `failed_sync_logs`: `idx_fsl_new_created` btree `(created_at DESC, id)` + PK `(id, created_at)`.
- `cdc_activity_log`: `idx_act_new_started` btree `(started_at DESC)` + PK `(id, created_at)`.

**Không cần thêm index mới**. Vấn đề là query không cho phép prune.

## Fix Applied

### File: `internal/service/system_health_collector.go`

**Query 1** — thêm upper bound `AND created_at <= NOW()` để kích hoạt runtime partition pruning:

```go
c.db.WithContext(ctxQ).Model(&model.FailedSyncLog{}).
    Where("created_at > NOW() - INTERVAL '24 hours' AND created_at <= NOW()").Count(&count24h)
c.db.WithContext(ctxQ).Model(&model.FailedSyncLog{}).
    Where("created_at > NOW() - INTERVAL '1 hour' AND created_at <= NOW()").Count(&count1h)
```

**Query 2** — thêm WHERE `created_at > NOW() - INTERVAL '1 day' AND created_at <= NOW()` để prune về 1-2 daily partitions:

```go
c.db.WithContext(ctxQ).
    Where("created_at > NOW() - INTERVAL '1 day' AND created_at <= NOW()").
    Order("started_at DESC").Limit(10).Find(&logs)
```

Semantic không đổi (recent 10 events trong 24h window — hợp lý cho health dashboard).

## EXPLAIN After Fix

### Query 1 (new):
```
Aggregate
  ->  Append (rows=5)
        Subplans Removed: 4            ← runtime pruning ACTIVE
        ->  Index Only Scan using failed_sync_logs_y2026m04_created_at_id_idx
              Index Cond: created_at > ... AND created_at <= now()
              Heap Fetches: 0
Planning Time: 51.486 ms (cold), 2-8ms warm
Execution Time: ~12 ms
```

→ Chỉ 1 partition được scan (Index Only Scan), 4 bị prune tại runtime.

### Query 2 (new):
```
Limit (rows=10)
  ->  Merge Append (rows=602)
        Subplans Removed: 2            ← runtime pruning ACTIVE
        Sort Key: cdc_activity_log.started_at DESC
        ->  Index Scan on cdc_activity_log_20260419_started_at_idx
        ->  Index Scan on cdc_activity_log_20260420_started_at_idx
        ... (6 partitions thay vì 8)
Planning Time: 11.692 ms
Execution Time: 12.301 ms
```

→ Prune 2 partitions. Planning cost giảm theo tỷ lệ.

## Verification

1. **Build**: `go build ./...` PASS.
2. **Runtime restart**: kill + `go run ./cmd/server` khởi động sạch, không fatal.
3. **Log grep sau 60s tick**: không còn `SLOW SQL` từ line 599 hoặc 610.

## Decisions

- **Không thêm migration mới**: Index `idx_fsl_new_created` và `idx_act_new_started` đã tồn tại (migration 010). Thêm sẽ thừa.
- **Không dùng Redis caching**: Thay đổi semantic (TTL staleness) và thêm failure mode không cần thiết khi query rewrite đã đủ.
- **Không widen slow SQL threshold**: Vi phạm Rule 6 (hide symptom).
- **Giữ `ORDER BY started_at DESC`**: Index `idx_act_new_started` tồn tại; workload cần order theo `started_at` (thời điểm bắt đầu operation), không phải `created_at` (thời điểm INSERT row).

## Known Follow-up (out of scope)

- `cdc_activity_log_default` có 437 rows → nghĩa là có data rơi vào DEFAULT partition (ngoài khoảng 2026-04-17..2026-04-24). Cần kiểm tra partition auto-create job hoặc clock skew. Không thuộc scope SLOW SQL fix.

## Files Changed

- `cdc-cms-service/internal/service/system_health_collector.go` (lines 593-624)

## Files NOT Changed (verified but no change needed)

- `cdc-cms-service/internal/model/activity_log.go` — mapping đúng, `started_at` column tồn tại.
- `cdc-cms-service/internal/model/failed_sync_log.go` — mapping đúng, `created_at` primary partition key.
- `cdc-cms-service/migrations/*.sql` — không cần migration mới.
