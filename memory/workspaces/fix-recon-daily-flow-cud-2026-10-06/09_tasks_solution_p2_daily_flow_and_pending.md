# 09 — Tasks Solution: Phase P2 & Pending Issues Resolution

Ngày: 2026-10-12

Tài liệu này chứa thiết kế chi tiết và demo code cụ thể cho toàn bộ các hạng mục tồn đọng.

---

## 1. T1: Fix Router `chain()` Capacity Aliasing (CMS)
**File**: `cdc-cms-service/internal/router/router.go`
**Root cause**: `out := make([]fiber.Handler, 0, len(mws))` có `cap = len(mws)`. Nếu có middleware `nil`, `len(out) < cap(out)`. Khi dùng `destructiveChain := chain(...)`, các lệnh `append(destructiveChain, handler)` cho các route khác nhau có thể ghi đè vào cùng 1 mảng nền do dung lượng còn thừa chưa dùng hết.
**Giải pháp**: Trả về 3-index slice `out[:len(out):len(out)]` để khóa dung lượng (capacity = length). Bất kỳ lệnh `append` nào từ bên ngoài đều buộc Go runtime cấp phát mảng mới.

```go
// Line 97 cdc-cms-service/internal/router/router.go
func chain(mws ...fiber.Handler) []fiber.Handler {
	out := make([]fiber.Handler, 0, len(mws))
	for _, m := range mws {
		if m != nil {
			out = append(out, m)
		}
	}
	// 3-index slice to freeze capacity and prevent backing-array aliasing on subsequent appends
	return out[:len(out):len(out)]
}
```

---

## 2. T2: Khắc phục đếm thừa DLQ khi Transmute Batch chia nhỏ (CDS)
**File**: `centralized-data-service/internal/handler/master/transmute_handler.go`
**Root cause**: Khi mẻ transmute bị lỗi poison pill, hệ thống chạy `binarySearchSplit`. Cả batch cha và các sub-batch trung gian đều log `status='error'`, khiến FE cứ thấy `error` là đếm `+1` DLQ.
**Giải pháp**:
1. Đánh dấu `"split_retry": true` trong `details` khi mẻ gặp lỗi và tiếp tục phân rã (`len(subBatch) > 1`).
2. Khi cô lập thành công (`len(batch) == 1`), ghi nhận rõ `"poison_pill": true`.

```go
// In runDebouncedGroup (khi lỗi non-transient và bắt đầu split):
if logEntry != nil {
	h.activity.FailWithDetails(logEntry, err.Error(), 0, map[string]interface{}{
		"master_binding_id": masterBindingID,
		"split_retry":       true,
	})
}
h.binarySearchSplit(ctx, masterTable, batch)

// In processSubBatch (khi mẻ con lỗi và tiếp tục split):
if err != nil {
	if logEntry != nil {
		h.activity.FailWithDetails(logEntry, err.Error(), 0, map[string]interface{}{
			"master_binding_id": masterBindingID,
			"split_retry":       true,
		})
	}
	h.binarySearchSplit(ctx, masterTable, subBatch)
	return
}

// In binarySearchSplit (khi cô lập được đúng 1 poison pill):
if len(batch) == 1 {
	task := batch[0]
	if h.activity != nil {
		logEntry := h.activity.Start("transmute_poison_pill", masterTable, task.Req.TriggeredBy)
		h.activity.FailWithDetails(logEntry, "poison pill isolated", 1, map[string]interface{}{
			"master_binding_id": task.Req.MasterBindingID,
			"poison_pill":       true,
			"source_ids":        task.Req.SourceIDs,
		})
	}
	h.replyErr(ctx, task.Msg, task.Req.CorrelationID, "poison_pill_isolated")
	h.publishCompleted(ctx, task.Req, master.TransmuteResult{}, fmt.Errorf("poison pill isolated"), "")
	return
}
```

---

## 3. T3 & T4: Model & SQL Aggregation Daily Flow (CMS)
**Files**:
- `cdc-cms-service/internal/app/queries/system/daily_flow.go` (DTO)
- `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` (SQL)

### DTO Definition
```go
package system

type DailyFlowParams struct {
	ShadowSchema    string `json:"shadow_schema"`
	ShadowTable     string `json:"shadow_table"`
	MasterTable     string `json:"master_table"`
	SourceDatabase  string `json:"source_database"`
	Days            int    `json:"days"`            // Default 14
	TzOffsetMinutes int    `json:"tz_offset_minutes"` // e.g. +420 for UTC+7
}

type DailyFlowRecord struct {
	Date     string `json:"date"`
	Snapshot struct {
		ShadowWritten int64 `json:"shadowWritten"`
		MasterWritten int64 `json:"masterWritten"`
	} `json:"snapshot"`
	Kafka struct {
		Total int64 `json:"total"`
		C     int64 `json:"c"`
		U     int64 `json:"u"`
		D     int64 `json:"d"`
	} `json:"kafka"`
	Shadow struct {
		Total   int64 `json:"total"`
		Written int64 `json:"written"`
		C       int64 `json:"c"`
		U       int64 `json:"u"`
		D       int64 `json:"d"`
	} `json:"shadow"`
	Master struct {
		Total    int64 `json:"total"`
		Written  int64 `json:"written"`
		Inserted int64 `json:"inserted"`
		Updated  int64 `json:"updated"`
		Skipped  int64 `json:"skipped"`
	} `json:"master"`
	Recon struct {
		TotalRuns int64 `json:"totalRuns"`
		DLQ       int64 `json:"dlq"`
		Missing   int64 `json:"missing"`
		Stale     int64 `json:"stale"`
		Healed    int64 `json:"healed"`
	} `json:"recon"`
	Status string `json:"status"` // BALANCED | DRIFT | WARNING
}
```

### SQL Aggregation (Tận dụng Partition Pruning + Date Series)
```go
func (r *ActivityLogReadRepo) GetDailyFlow(ctx context.Context, p system.DailyFlowParams) ([]system.DailyFlowRecord, error) {
	if p.Days <= 0 {
		p.Days = 14
	}
	if p.Days > 90 {
		p.Days = 90
	}

	query := `
	WITH dates AS (
		SELECT TO_CHAR(d::date, 'YYYY-MM-DD') AS day_str
		FROM generate_series(CURRENT_DATE - (? * INTERVAL '1 day' - INTERVAL '1 day'), CURRENT_DATE, '1 day'::interval) d
	),
	act_filtered AS (
		SELECT al.*
		FROM cdc_activity_log al
		WHERE al.created_at >= NOW() - (? * INTERVAL '1 day')
		  AND (
		        (al.target_table = ? OR al.target_table LIKE ?)`
	args := []interface{}{p.Days, p.Days, p.ShadowTable, "%." + p.ShadowTable}
	if p.MasterTable != "" {
		query += ` OR (al.target_table = ? OR al.target_table LIKE ?)`
		args = append(args, p.MasterTable, "%." + p.MasterTable)
	}
	query += `)
	),
	enriched AS (
		SELECT al.*,
		       COALESCE(tm_sb.shadow_schema, sb.shadow_schema) AS derived_shadow_schema,
		       COALESCE(tm.master_table, mb.master_table) AS derived_master_table
	` + r.enrichmentFromClause("SELECT * FROM act_filtered") + `
		WHERE (
			-- Shadow logs phải khớp schema
			(al.operation = 'kafka-consumer' AND (COALESCE(tm_sb.shadow_schema, sb.shadow_schema) = ? OR al.details->>'shadow_schema' = ?))
			-- Transmute logs
			OR (al.operation IN ('transmute', 'transmute_sub') AND (?::text = '' OR COALESCE(tm.master_table, mb.master_table) = ?))
		)
	),
	act_daily AS (
		SELECT
			TO_CHAR(e.started_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD') AS day_str,
			-- 1. Snapshot
			SUM(CASE WHEN (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true) AND e.operation = 'kafka-consumer' AND e.status <> 'error'
			         THEN COALESCE((e.details->>'written')::bigint, e.rows_affected, 0) ELSE 0 END) AS snapshot_shadow,
			SUM(CASE WHEN e.triggered_by = 'snapshot-hook' AND e.operation IN ('transmute','transmute_sub') AND e.status <> 'error'
			         THEN COALESCE((e.details->>'inserted')::bigint, 0) + COALESCE((e.details->>'updated')::bigint, 0) + e.rows_affected ELSE 0 END) AS snapshot_master,
			-- 2. Kafka stream
			SUM(CASE WHEN e.operation = 'kafka-consumer' AND NOT (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true)
			         THEN COALESCE((e.details->>'c')::bigint, (e.details->>'inserted')::bigint, 0)
			            + COALESCE((e.details->>'u')::bigint, (e.details->>'updated')::bigint, 0)
			            + COALESCE((e.details->>'d')::bigint, (e.details->>'deleted')::bigint, 0)
			            + CASE WHEN (e.details->>'c' IS NULL AND e.details->>'inserted' IS NULL) THEN COALESCE((e.details->>'written')::bigint, e.rows_affected, 0) ELSE 0 END
			         ELSE 0 END) AS kafka_total,
			SUM(CASE WHEN e.operation = 'kafka-consumer' AND NOT (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true)
			         THEN COALESCE((e.details->>'c')::bigint, (e.details->>'inserted')::bigint, 0) ELSE 0 END) AS kafka_c,
			SUM(CASE WHEN e.operation = 'kafka-consumer' AND NOT (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true)
			         THEN COALESCE((e.details->>'u')::bigint, (e.details->>'updated')::bigint, 0) ELSE 0 END) AS kafka_u,
			SUM(CASE WHEN e.operation = 'kafka-consumer' AND NOT (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true)
			         THEN COALESCE((e.details->>'d')::bigint, (e.details->>'deleted')::bigint, 0) ELSE 0 END) AS kafka_d,
			-- 3. Shadow write
			SUM(CASE WHEN e.operation = 'kafka-consumer' AND e.status <> 'error' AND NOT (e.details->>'buffer_type' = 'snapshot' OR (e.details->>'is_snapshot')::boolean = true)
			         THEN COALESCE((e.details->>'written')::bigint, e.rows_affected, 0) ELSE 0 END) AS shadow_written,
			-- 4. Master transmute
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by <> 'snapshot-hook'
			         THEN COALESCE((e.details->>'inserted')::bigint, 0) ELSE 0 END) AS master_inserted,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by <> 'snapshot-hook'
			         THEN COALESCE((e.details->>'updated')::bigint, 0) ELSE 0 END) AS master_updated,
			SUM(CASE WHEN e.operation IN ('transmute','transmute_sub') AND e.status <> 'error' AND e.triggered_by <> 'snapshot-hook'
			         THEN COALESCE((e.details->>'skipped')::bigint, 0) ELSE 0 END) AS master_skipped,
			-- 5. DLQ (Bỏ qua split_retry để tránh đếm thừa)
			SUM(CASE WHEN e.status = 'error' AND (e.details->>'split_retry')::boolean IS NOT TRUE THEN 1 ELSE 0 END) AS dlq_count
		FROM enriched e
		GROUP BY 1
	),
	recon_daily AS (
		SELECT
			TO_CHAR(r.checked_at AT TIME ZONE 'UTC' + (? * INTERVAL '1 minute'), 'YYYY-MM-DD') AS day_str,
			COUNT(*) AS total_runs,
			SUM(COALESCE(r.missing_count, 0)) AS missing,
			SUM(COALESCE(r.stale_count, 0)) AS stale,
			SUM(COALESCE(r.healed_count, 0)) AS healed
		FROM cdc_system.cdc_reconciliation_report r
		WHERE r.checked_at >= NOW() - (? * INTERVAL '1 day')
		  AND r.shadow_schema = ?
		  AND r.shadow_table = ?
		GROUP BY 1
	)
	SELECT
		d.day_str AS date,
		COALESCE(a.snapshot_shadow, 0) AS "snapshot.shadow_written",
		COALESCE(a.snapshot_master, 0) AS "snapshot.master_written",
		COALESCE(a.kafka_total, 0)     AS "kafka.total",
		COALESCE(a.kafka_c, 0)         AS "kafka.c",
		COALESCE(a.kafka_u, 0)         AS "kafka.u",
		COALESCE(a.kafka_d, 0)         AS "kafka.d",
		COALESCE(a.shadow_written, 0)  AS "shadow.written",
		COALESCE(a.master_inserted, 0) AS "master.inserted",
		COALESCE(a.master_updated, 0)  AS "master.updated",
		COALESCE(a.master_skipped, 0)  AS "master.skipped",
		COALESCE(r.total_runs, 0)      AS "recon.total_runs",
		COALESCE(a.dlq_count, 0)       AS "recon.dlq",
		COALESCE(r.missing, 0)         AS "recon.missing",
		COALESCE(r.stale, 0)           AS "recon.stale",
		COALESCE(r.healed, 0)          AS "recon.healed"
	FROM dates d
	LEFT JOIN act_daily a ON a.day_str = d.day_str
	LEFT JOIN recon_daily r ON r.day_str = d.day_str
	ORDER BY d.day_str DESC;
	`
	args = append(args, p.ShadowSchema, p.ShadowSchema, p.MasterTable, p.MasterTable, p.TzOffsetMinutes, p.TzOffsetMinutes, p.Days, p.ShadowSchema, p.ShadowTable)

	// Scan results & derive status
	...
}
```

---

## 4. T5 & T6: API Handler & Frontend Hook Integration
**Endpoint**: `GET /api/reconciliation/daily-flow`
**Web Hook**: `src/hooks/useReconStatus.ts`
```typescript
export function useDailyFlow(
  shadowTable: string | null,
  shadowSchema?: string | null,
  masterTable?: string | null,
  sourceDb?: string | null,
  days = 14
) {
  const tzOffset = -new Date().getTimezoneOffset(); // e.g. +420 cho GMT+7
  return useQuery<{ data: DailyFlowRecord[] }>({
    queryKey: ['daily-flow', shadowTable, shadowSchema, masterTable, sourceDb, days, tzOffset],
    queryFn: async () => {
      const params: Record<string, string | number> = {
        shadow_table: shadowTable!,
        days,
        tz_offset: tzOffset,
      };
      if (shadowSchema) params.shadow_schema = shadowSchema;
      if (masterTable) params.master_table = masterTable;
      if (sourceDb) params.source_database = sourceDb;

      const { data } = await cmsApi.get('/api/reconciliation/daily-flow', { params });
      return data;
    },
    enabled: Boolean(shadowTable),
    refetchInterval: 15_000,
  });
}
```

Trong [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx):
- Thay thế 2 hook `usePipelineActivityLog` và in-memory reducer bằng `const { data: dailyFlowRes, isLoading } = useDailyFlow(historyTable, historySchema, historyMaster, sourceDb);`
- Dữ liệu `dailyStats` nhận trực tiếp `dailyFlowRes?.data || []`.
- Xóa bỏ hoàn toàn warning `isLogsTruncated` và 200 logs limit!

---

## 5. T7: Script Bảo trì Backfill `shadow_schema` cho Log cũ
**File**: `cdc-cms-service/migrations/scripts/enrich_legacy_activity_logs.sql`
```sql
-- One-time maintenance script:
-- Backfill shadow_schema cho các log cũ chưa có identity thuộc các bảng chỉ có duy nhất 1 active binding.
UPDATE cdc_system.cdc_activity_log al
SET details = COALESCE(al.details, '{}'::jsonb) || jsonb_build_object('shadow_schema', sb.shadow_schema)
FROM cdc_system.shadow_binding sb
WHERE al.created_at >= NOW() - INTERVAL '30 days'
  AND al.details->>'shadow_schema' IS NULL
  AND sb.is_active = TRUE
  AND sb.shadow_table = al.target_table
  -- Bảng chỉ có duy nhất 1 binding (không có nguy cơ gán nhầm sang connector khác)
  AND (
      SELECT COUNT(*)
      FROM cdc_system.shadow_binding sb_check
      WHERE sb_check.shadow_table = sb.shadow_table
        AND sb_check.is_active = TRUE
  ) = 1;
```
