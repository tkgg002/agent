# 09 — Giải pháp V4: Sửa tận gốc Clear Activity Log & Bảng Đối Soát (line-by-line)

> Trạng thái: **PROPOSE — chờ APPROVE** (Rule #13). Căn cứ: `13_analysis_audit_v4_clear_and_grid.md`.
> Nguyên tắc: additive, tương thích ngược, tái dùng pattern có sẵn (`destructiveChain`, `enrichmentFromClause`, `auditHeaders`, `FailWithDetails`).

## Tổng quan thay đổi

| Phase | Service | File | Loại | Mục tiêu |
|---|---|---|---|---|
| P1 | centralized-data-service | `handler/shadow/batch_buffer.go` | sửa ~8 dòng | Ghi `shadow_schema` vào details (success + error) — fix C1 |
| P1 | centralized-data-service | `handler/shadow/batch_buffer_fanout.go` | sửa ~4 dòng | Dấu snapshot vào trigger — fix C4 |
| P1 | centralized-data-service | `handler/master/transmute_handler.go` | sửa ~40 dòng | Forward dấu snapshot, group debounce theo (binding, trigger), ghi `master_binding_id`, details cho `transmute_sub` — fix C4/H1/H2 |
| P1 | cdc-cms-service | `infra/persistence/system/activity_log_read_repo_gorm.go` | sửa LATERAL + viết lại `ClearActivity` | Resolve theo định danh mới; xóa đúng cái UI thấy — fix C2/C5 |
| P1 | cdc-cms-service | `app/queries/system/list_activity_logs.go` | sửa struct | Bỏ from/to, `Before *time.Time` |
| P1 | cdc-cms-service | `api/system/activity_log_handler.go` | sửa `Clear` | Validate scope + RFC3339 |
| P1 | cdc-cms-service | `router/router.go` | đổi 1 → 2 dòng | Qua `destructiveChain` — fix C3 |
| P1 | cdc-cms-web | `hooks/useReconStatus.ts` | sửa | Payload mới, reason chỉ ở header, op `transmute,transmute_sub` |
| P1 | cdc-cms-web | `components/ReconPipelineGrid.tsx` | sửa | Snapshot master = `snapshot-hook`; ngày theo giờ local; bỏ đoán `'cdc_shadow'` |
| P2 | cdc-cms-service + web | query mới `daily-flow` | thêm | Tổng hợp theo ngày phía server (fix H5 — gốc rễ số liệu sai khi > 200 batch) |

---

## P1-1. `centralized-data-service/internal/handler/shadow/batch_buffer.go` (L616-652)

```go
	defer func() {
		if act != nil && logEntry != nil {
			if err != nil {
				// Ghi định danh pipeline cả ở nhánh lỗi để CMS cách ly đúng connector.
				act.FailWithDetails(logEntry, err.Error(), 0, map[string]any{
					"shadow_schema": schemaName,
					"buffer_type":   bufferType,
					"is_snapshot":   bb.isSnapshotBuffer,
				})
			} else {
				// ... giữ nguyên vòng đếm cntC/cntU/cntD ...
				details := map[string]any{
					"batch_size":    len(records),
					"written":       written,
					"c":             cntC,
					"u":             cntU,
					"d":             cntD,
					"buffer_type":   bufferType,
					"is_snapshot":   bb.isSnapshotBuffer,
					"trace_id":      traceID,
					"shadow_schema": schemaName, // NEW
				}
				act.Complete(logEntry, int64(written), details)
			}
		}
	}()
```
> `Fail(e,msg)` = `FailWithDetails(e,msg,0,nil)` (activity_logger.go) → hành vi rows_affected giữ nguyên.

## P1-2. `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go` (L56-63)

```go
	triggeredBy := "batch-buffer"
	if bb.isSnapshotBuffer {
		triggeredBy = "snapshot"
	}
	payload, err := json.Marshal(map[string]any{
		"shadow_table":          shadowTable,
		"shadow_schema":         shadowSchema,
		"shadow_connection_key": "default",
		"_source_ids":           sourceIDs,
		"triggered_by":          triggeredBy,
		"correlation_id":        fmt.Sprintf("batch-%s-%d", shadowTable, time.Now().UnixNano()),
	})
```
> Đã grep: không consumer nào đọc giá trị `"batch-buffer"` → đổi an toàn.

## P1-3. `centralized-data-service/internal/handler/master/transmute_handler.go`

**(a) `HandleTransmuteShadow` — struct `req` (L85-92) thêm field, và L136-143:**
```go
		CorrelationID       string   `json:"correlation_id,omitempty"`
		TriggeredBy         string   `json:"triggered_by,omitempty"` // NEW
	}
	// ...
	hookTrigger := "kafka-consumer-hook"
	if req.TriggeredBy == "snapshot" {
		hookTrigger = "snapshot-hook"
	}
	for _, target := range masterTargets {
		out, _ := json.Marshal(map[string]any{
			"master_table":      target.MasterFQN,
			"master_binding_id": target.ID,
			"_source_ids":       req.SourceIDs,
			"triggered_by":      hookTrigger,
			"correlation_id":    req.CorrelationID,
		})
```

**(b) Group mẻ debounce theo (binding, trigger) — thay đầu hàm `runDebouncedTransmute` (L398):**
```go
type debounceRunKey struct {
	masterBindingID int64
	triggeredBy     string
}

// groupDebouncedTasks tách mẻ debounce theo (master_binding_id, triggered_by), giữ thứ tự xuất hiện.
func groupDebouncedTasks(batch []NatsMsgTask) [][]NatsMsgTask {
	idx := make(map[debounceRunKey]int)
	var groups [][]NatsMsgTask
	for _, t := range batch {
		k := debounceRunKey{masterBindingID: t.Req.MasterBindingID, triggeredBy: t.Req.TriggeredBy}
		i, ok := idx[k]
		if !ok {
			i = len(groups)
			idx[k] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], t)
	}
	return groups
}

// runDebouncedTransmute chạy tuần tự từng nhóm trong CÙNG goroutine của debouncer
// → giữ nguyên concurrency=1/master table (tránh 40P01).
func (h *TransmuteHandler) runDebouncedTransmute(ctx context.Context, batch []NatsMsgTask) {
	for _, g := range groupDebouncedTasks(batch) {
		h.runDebouncedGroup(ctx, g)
	}
}

// runDebouncedGroup = NGUYÊN VẸN thân hàm runDebouncedTransmute hiện tại (chỉ đổi tên).
func (h *TransmuteHandler) runDebouncedGroup(ctx context.Context, batch []NatsMsgTask) {
	// ... body cũ L399-505 giữ nguyên ...
}
```

**(c) Ghi `master_binding_id` vào activity log:**
- L276 `h.activity.Fail(logEntry, err.Error())` → `h.activity.FailWithDetails(logEntry, err.Error(), 0, map[string]interface{}{"master_binding_id": req.MasterBindingID})`
- L286 Complete map: thêm `"master_binding_id": req.MasterBindingID,`
- L460 Fail → `FailWithDetails(..., map[string]interface{}{"master_binding_id": masterBindingID})`
- L482 Complete map: thêm `"master_binding_id": masterBindingID,`
- L567 Fail → `FailWithDetails(..., map[string]interface{}{"master_binding_id": masterBindingID})`
- L582 `Complete(logEntry, res.Inserted+res.Updated, nil)` →
```go
			h.activity.Complete(logEntry, res.Inserted+res.Updated, map[string]interface{}{
				"scanned":           res.Scanned,
				"inserted":          res.Inserted,
				"updated":           res.Updated,
				"skipped":           res.Skipped,
				"master_binding_id": masterBindingID,
				"trace_id":          traceID,
			})
```
> Không double-count: khi split, log cha `transmute` là `error` (không có số), số thật nằm ở các `transmute_sub` success.

## P1-4. `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`

**(a) `enrichmentFromClause` — LATERAL `sb` (sau L46) và `scope_counts` (sau L85) thêm cùng 1 predicate:**
```sql
			  AND (
			        NULLIF(al.details->>'shadow_schema', '') IS NULL
			     OR sb.shadow_schema = al.details->>'shadow_schema'
			  )
```
`scope_counts` thêm thêm predicate `shadow_binding_id` (đang thiếu so với LATERAL sb):
```sql
			  AND (
			        NULLIF(al.details->>'shadow_binding_id', '') IS NULL
			     OR sb.id = (al.details->>'shadow_binding_id')::bigint
			  )
```
**LATERAL `tm` (sau L65):**
```sql
			  AND (
			        NULLIF(al.details->>'master_binding_id', '') IS NULL
			     OR mb2.id = (al.details->>'master_binding_id')::bigint
			  )
```
> Log cũ (không có key) → hành vi y như hiện tại. Log mới → resolve tất định.

**(b) Viết lại `ClearActivity` (L306-405) — tái dùng `enrichmentFromClause` để "xóa đúng cái UI thấy":**
```go
// ErrClearScopeRequired: Clear bắt buộc có pipeline scope (không cho xóa toàn hệ thống theo ngày).
var ErrClearScopeRequired = errors.New("shadow_schema and shadow_table are required")

// ClearActivity xóa log của đúng 1 pipeline. Dùng CHUNG enrichmentFromClause với ListActivity
// để tập bị xóa == tập UI hiển thị. Bỏ qua bản ghi mơ hồ (scope_ambiguous) để không xóa
// nhầm log legacy của connector khác trùng tên bảng.
func (r *ActivityLogReadRepo) ClearActivity(ctx context.Context, p system.ClearActivityParams) (int64, error) {
	if p.ShadowSchema == "" || p.ShadowTable == "" {
		return 0, ErrClearScopeRequired
	}

	inner := `SELECT * FROM cdc_activity_log WHERE 1=1`
	var args []interface{}
	if p.Before != nil {
		inner += ` AND created_at < ? AND started_at < ?`
		args = append(args, *p.Before, *p.Before)
	}

	idQuery := `SELECT al.id` + r.enrichmentFromClause(inner) + `
		WHERE COALESCE(tm_sb.shadow_schema, sb.shadow_schema) = ?
		  AND COALESCE(tm_sb.shadow_table,  sb.shadow_table)  = ?
		  AND (al.operation IN ('transmute', 'transmute_sub') OR COALESCE(scope_counts.binding_count, 0) <= 1)
		  AND (al.operation NOT IN ('transmute', 'transmute_sub') OR (? <> '' AND tm.master_table = ?))`
	args = append(args, p.ShadowSchema, p.ShadowTable, p.MasterTable, p.MasterTable)
	if p.SourceDatabase != "" {
		idQuery += ` AND COALESCE(tm_so.source_database, so.source_database) = ?`
		args = append(args, p.SourceDatabase)
	}

	deleteSQL := `DELETE FROM cdc_activity_log WHERE id IN (` + idQuery + `)`
	var outer []interface{}
	if p.Before != nil {
		// Lặp lại điều kiện ở câu ngoài để planner prune partition (partition key = created_at).
		deleteSQL += ` AND created_at < ? AND started_at < ?`
		outer = append(outer, *p.Before, *p.Before)
	}
	res := r.db.WithContext(ctx).Exec(deleteSQL, append(args, outer...)...)
	return res.RowsAffected, res.Error
}
```
> Import thêm `errors`, `time` (nếu cần); bỏ `fmt` nếu không còn dùng. Bảng dùng `cdc_activity_log` (search_path) giống toàn bộ repo.

## P1-5. `cdc-cms-service/internal/app/queries/system/list_activity_logs.go` (L21-30)

```go
// ClearActivityParams — scope pipeline bắt buộc (ShadowSchema + ShadowTable).
type ClearActivityParams struct {
	ShadowSchema   string
	ShadowTable    string
	SourceDatabase string
	MasterTable    string     // bare master table (khớp master_binding.master_table)
	Before         *time.Time // nil = xóa toàn bộ log của pipeline
}
```

## P1-6. `cdc-cms-service/internal/api/system/activity_log_handler.go` — `Clear`

```go
func (h *ActivityLogHandler) Clear(c *fiber.Ctx) error {
	p := system.ClearActivityParams{
		ShadowSchema:   strings.TrimSpace(c.Query("shadow_schema")),
		ShadowTable:    strings.TrimSpace(c.Query("shadow_table")),
		SourceDatabase: strings.TrimSpace(c.Query("source_database")),
		MasterTable:    strings.TrimSpace(c.Query("master_table")),
	}
	if p.ShadowSchema == "" || p.ShadowTable == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "shadow_schema and shadow_table are required"})
	}
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "before must be RFC3339"})
		}
		p.Before = &t
	}
	rows, err := h.cleaner.ClearActivity(c.UserContext(), p)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "rows_deleted": rows})
}
```
> Swagger comment: thay `before_date/from_date/to_date` bằng `before (RFC3339)`.

## P1-7. `cdc-cms-service/internal/router/router.go`

```go
// XÓA L311:  dual("DELETE", shared, "/activity-log", h.System.ActivityLog.Clear)
// THÊM cạnh L195-196 (khối DESTRUCTIVE ROUTES):
	api.Delete("/activity-log", append(destructiveChain, h.System.ActivityLog.Clear)...)
	api.Delete("/v1/activity-log", append(destructiveChain, h.System.ActivityLog.Clear)...)
```
> Lưu ý: `append(destructiveChain, …)` đang được dùng y hệt ở L195-196 — giữ đúng pattern hiện hữu.

## P1-8. `cdc-cms-web/src/hooks/useReconStatus.ts`

```ts
export interface ClearActivityLogPayload {
  shadow_schema: string;
  shadow_table: string;
  source_database?: string;
  master_table?: string;
  before?: string; // RFC3339 (ISO) — mốc tuyệt đối, không phụ thuộc TZ server
  reason: string;
}

export function useClearActivityLogMutation() {
  const queryClient = useQueryClient();
  return useMutation<{ ok: boolean; rows_deleted: number }, Error, ClearActivityLogPayload>({
    mutationFn: async ({ reason, ...params }) => {
      const { data } = await cmsApi.delete<{ ok: boolean; rows_deleted: number }>('/api/activity-log', {
        params,
        headers: auditHeaders(reason),
      });
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-activity-log'] });
    },
    retry: 0,
  });
}
```
> Bỏ invalidate `table-history`/`recon-report` (không liên quan activity log).

## P1-9. `cdc-cms-web/src/components/ReconPipelineGrid.tsx`

**(a) Modal (L383-400):**
```tsx
    if (purgeMode === 'before') {
      if (!beforeDate) { message.warning('Vui lòng chọn ngày mốc để xóa'); return; }
      payload.before = beforeDate.startOf('day').toISOString();
      payload.reason = `Xóa nhật ký pipeline ${shadowSchema}.${shadowTable} trước ngày ${beforeDate.format('YYYY-MM-DD')}`;
    } else {
      payload.reason = `Xóa TOÀN BỘ nhật ký pipeline ${shadowSchema}.${shadowTable}`;
    }
```
`okButtonProps={{ danger: true, loading: clearMutation.isPending, disabled: !shadowSchema || !shadowTable }}`; thêm dòng cảnh báo: "Bản ghi legacy mơ hồ (trùng tên bảng, chưa có định danh) sẽ không bị xóa — để partition retention tự dọn."

**(b) L490 — lấy cả sub-batch:**
```tsx
usePipelineActivityLog(historyMaster, 'transmute,transmute_sub', sourceDb, historySchema, LOG_FETCH_LIMIT);
```
**(c) Phân loại snapshot (L542-546, L582-586):**
```tsx
// Shadow: writer ghi buffer_type/is_snapshot (batch_buffer.go L645-646)
const isSnapshot = details.buffer_type === 'snapshot' || details.is_snapshot === true;
// Master: transmute_handler forward triggered_by='snapshot-hook' khi trigger đến từ snapshot buffer
const isSnapshot = t.triggered_by === 'snapshot-hook';
```
**(d) Bucket ngày theo giờ local (L529, L572 và mọi chỗ `.slice(0, 10)` trên started_at):**
```tsx
const d = k.started_at ? dayjs(k.started_at).format('YYYY-MM-DD') : '';
```
**(e) L891 — bỏ đoán schema:** `const shadowSchema = pipeline.rowA?.shadow_schema || pipeline.rowB?.shadow_schema || '';` (rỗng → nút Clear disabled).
**(f) `isFailed`:** `t.status === 'error' || Boolean(t.error_message)` (writer ghi `"error"`, không phải `"failed"`).

---

## P2. Tổng hợp theo ngày phía server (gốc rễ H5) — thiết kế

UI hiện tính bảng ngày trên ≤ 200 log mới nhất → không thể đúng cho nhiều ngày. Thêm query đọc (đúng pattern CQRS Q-side như `ListActivityLogs`):

- `app/queries/system/get_daily_flow.go`: `GetDailyFlowQuery{Scope, Days int, TZ string}` + handler.
- Reader method trên `ActivityLogReadRepo`:
```sql
SELECT to_char(al.started_at AT TIME ZONE ?, 'YYYY-MM-DD')                 AS day,
       CASE WHEN al.operation IN ('transmute','transmute_sub') THEN 'master' ELSE 'shadow' END AS stage,
       (al.details->>'buffer_type' = 'snapshot' OR al.triggered_by = 'snapshot-hook') AS is_snapshot,
       al.status,
       COUNT(*)                                                       AS batches,
       SUM(COALESCE((al.details->>'c')::bigint, 0))                   AS c,
       SUM(COALESCE((al.details->>'u')::bigint, 0))                   AS u,
       SUM(COALESCE((al.details->>'d')::bigint, 0))                   AS d,
       SUM(COALESCE((al.details->>'written')::bigint, 0))             AS written,
       SUM(COALESCE((al.details->>'inserted')::bigint, 0))            AS inserted,
       SUM(COALESCE((al.details->>'updated')::bigint, 0))             AS updated,
       SUM(COALESCE((al.details->>'skipped')::bigint, 0))             AS skipped
  <enrichmentFromClause(`SELECT * FROM cdc_activity_log
                          WHERE created_at >= NOW() - (? * INTERVAL '1 day')
                            AND operation IN ('kafka-consumer','transmute','transmute_sub')`)>
 WHERE <cùng scope predicate với ClearActivity>
 GROUP BY 1, 2, 3, 4
 ORDER BY 1 DESC;
```
- Route `dual("GET", shared, "/activity-log/daily-flow", …)`; FE thay reducer `dailyStats` bằng dữ liệu này (giữ nguyên cột hiển thị).
- Ghi chú trung thực: Khối "2. KAFKA" vẫn suy ra từ log Shadow (c/u/d theo batch) — không phải đo độc lập phía Kafka; nhãn UI cần ghi "Kafka (theo batch Shadow nhận)".

---

## Giới hạn được chấp nhận (ghi ADR)
1. Log legacy (trước khi deploy P1) của 2 connector trùng tên bảng vẫn mơ hồ → UI gắn cờ `scope_ambiguous`, Clear bỏ qua; partition retention (`governance/partition_dropper.go`) tự dọn.
2. Binding đã `is_active = FALSE` không resolve được qua LATERAL → không Clear qua UI; để retention dọn.
