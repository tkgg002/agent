---
tier: T2
---
# Solution — daily_flow_shadow_scope

## Phát hiện (VERIFIED bằng code)
- **F1 (read-side, gốc trực tiếp):** `GetDailyFlow` lọc dòng transmute CHỈ theo `master_table` trần, không theo shadow:
  `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:197-200`
  ```sql
  (al.operation = 'kafka-consumer' AND (?::text = '' OR COALESCE(tm_sb.shadow_schema, sb.shadow_schema) = ? OR al.details->>'shadow_schema' = ?))
  OR (al.operation IN ('transmute','transmute_sub','transmute_poison_pill') AND (?::text = '' OR COALESCE(tm.master_table, mb.master_table) = ?))
  ```
  Web còn bóc schema master: `cdc-cms-web/src/components/ReconPipelineGrid.tsx:475-476` (`rawMaster.split('.').pop()`), và `act_filtered` dùng `LIKE '%.' || master` (dòng 188-189) → mọi log transmute của mọi master tên `bank_requests` đều lọt vào CẢ 2 pipeline. Bằng chứng số: Master = tổng 2 pipeline (822+942=1,764; 626+494=1,120; 196+448=644; 372+1,160=1,532).
  Đối chứng: list activity log ĐÃ scope đúng (`activity_log_list_clear_gorm.go:116-117` `COALESCE(tm_sb.shadow_schema, sb.shadow_schema) = ?`) → Daily Flow là chỗ lệch duy nhất.
- **F2 (write-side, gốc sâu):** Log transmute chỉ ghi `master_binding_id`, không ghi shadow nguồn (`centralized-data-service/internal/handler/master/transmute_handler.go:750-764`, `transmute_poison_pill_split.go:54-60,200-240`). Với master binding `multi_shadow` (N shadow → 1 binding qua `transform_spec.shadow_sources`, `repository/master/master_binding_repo.go:109-119`) thì `tm_sb` chỉ trỏ về shadow chính → không thể tách theo shadow chỉ bằng SQL. Đồng thời debouncer gom chung task 2 shadow vào 1 nhóm (`transmute_event_date_split.go:14-18,126-130` khóa = binding+trigger+event_date).

## Giải pháp duy nhất (map 1-1)
- **F2 →** Mang danh tính shadow theo task: thêm `ShadowSchema/ShadowTable` vào `TransmuteRequest`, gán ở fan-out (`transmute_handler.go:228-235` và nhánh publish `:268-275`), thêm vào `debounceRunKey` để mỗi nhóm chỉ thuộc 1 shadow (vẫn chạy TUẦN TỰ trong cùng goroutine debouncer `runDebouncedTransmute` :615-619 → KHÔNG thêm concurrency, giữ chống 40P01), và ghi `shadow_schema/shadow_table` vào `details` của mọi log `transmute`/`transmute_sub`/`transmute_poison_pill` luồng CDC qua 1 helper file riêng.
- **F1 →** `GetDailyFlow`: dòng transmute phải khớp shadow: `COALESCE(NULLIF(al.details->>'shadow_schema',''), tm_sb.shadow_schema) = ?` và tương tự `shadow_table` (fallback `tm_sb` cho log cũ của binding 1:1).

## Code Demo
```go
// centralized-data-service/internal/handler/master/transmute_shadow_scope.go (MỚI)
package master

// withShadowScope gắn danh tính shadow nguồn vào details log transmute để
// báo cáo Daily Flow tách rạch ròi từng pipeline khi N shadow → 1 master.
func withShadowScope(req TransmuteRequest, details map[string]interface{}) map[string]interface{} {
	if req.ShadowSchema != "" {
		details["shadow_schema"] = req.ShadowSchema
	}
	if req.ShadowTable != "" {
		details["shadow_table"] = req.ShadowTable
	}
	return details
}
```
```go
// transmute_handler.go — TransmuteRequest
	ShadowSchema string `json:"shadow_schema,omitempty"` // shadow nguồn (scope báo cáo)
	ShadowTable  string `json:"shadow_table,omitempty"`
// fan-out :228 và payload publish :268
	tReq := TransmuteRequest{ MasterTable: target.MasterFQN, MasterBindingID: target.ID,
		ShadowSchema: req.ShadowSchema, ShadowTable: req.ShadowTable, ... }
// runDebouncedGroup :750 (và các FailWithDetails :718,:732)
	h.activity.Complete(logEntry, writtenRows, withShadowScope(batch[0].Req, map[string]interface{}{ ... }))
```
```go
// transmute_event_date_split.go
type debounceRunKey struct {
	masterBindingID int64
	shadowSchema    string
	shadowTable     string
	triggeredBy     string
	eventDate       string
}
	k := debounceRunKey{masterBindingID: t.Req.MasterBindingID,
		shadowSchema: t.Req.ShadowSchema, shadowTable: t.Req.ShadowTable,
		triggeredBy: t.Req.TriggeredBy, eventDate: taskEventDate(t, now)}
```
```go
// transmute_poison_pill_split.go :55, :203, :216, :238 — bọc details
	withShadowScope(task.Req, map[string]interface{}{ ... })      // :55
	withShadowScope(subBatch[0].Req, map[string]interface{}{ ... }) // processSubBatch
```
```sql
-- activity_log_read_repo_gorm.go:199 (GetDailyFlow) — args thêm p.ShadowSchema x3, p.ShadowTable x2
OR (al.operation IN ('transmute','transmute_sub','transmute_poison_pill')
    AND (?::text = '' OR COALESCE(tm.master_table, mb.master_table) = ?)
    AND (?::text = '' OR COALESCE(NULLIF(al.details->>'shadow_schema',''), tm_sb.shadow_schema) = ?)
    AND COALESCE(NULLIF(al.details->>'shadow_table',''), tm_sb.shadow_table, ?) = ?)
```

## Tác động / Rủi ro
- Không đổi dữ liệu master, không thêm goroutine; chỉ số nhóm/flush tăng khi binding là multi_shadow (mỗi shadow 1 lần RunEvents tuần tự).
- Dữ liệu lịch sử 11-15/11-16 (log cũ không có `shadow_schema`): nếu 2 shadow dùng 2 master binding riêng → SQL tách đúng ngay qua `tm_sb`; nếu chung 1 binding multi_shadow → log cũ dồn về shadow chính (không thể tách hồi tố), log mới từ lúc deploy tách đúng.
- Test: Postgres thật (pattern `activity_log_read_repo_gorm_test.go:51`) seed 2 shadow cùng `bank_requests` + log transmute có/không `shadow_schema`; unit `groupDebouncedTasksAt` 2 shadow → 2 nhóm.
