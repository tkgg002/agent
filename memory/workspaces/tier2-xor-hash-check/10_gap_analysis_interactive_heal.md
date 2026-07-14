# Báo Cáo Review Lần 3 — Architecture & Pattern Compliance (Rev.4 → Rev.5)

> **Mục tiêu**: Kiểm tra plan Rev.4 tuân thủ Pattern & Architecture của hệ thống. Mọi file/func mới phải tuân thủ tuyệt đối (Rule #12).

---

## Phát hiện 5 sai lệch Pattern

### ❌ Pattern Violation #1: Query endpoint dùng `reader.Method()` trực tiếp — Sai Pattern CQRS

**Pattern chuẩn của hệ thống**: Gateway dùng **CQRS Query Handler** (not direct repo calls). Xem 3 handler hiện có:

```go
// LatestReport — dùng Query Handler:
res, err := h.listLatestQ.Handle(ctx, recon.ListLatestReportsQuery{})

// TableHistory — dùng Query Handler:
res, err := h.getHistoryQ.Handle(ctx, recon.GetTableHistoryQuery{...})

// ListFailedLogs — dùng Query Handler:
res, err := h.listFailedQ.Handle(ctx, recon.ListFailedLogsQuery{...})
```

Mỗi query handler là 1 struct riêng biệt (ví dụ `ListLatestReportsHandler`) được inject vào `ReconciliationHandler` constructor.

**Plan Rev.4 sai**: Đề xuất gọi thẳng `h.reader.ListUnhealedReports()` — phá vỡ pattern CQRS.

**Fix**: Tạo **Query Handler struct** mới:
```go
// internal/app/queries/recon/list_unhealed_reports.go
type ListUnhealedReportsQuery struct { ShadowTable string }
type ListUnhealedReportsResult struct { Data []reconmodel.ReconciliationReport }
type ListUnhealedReportsHandler struct { reader ReconReader }
func (h *ListUnhealedReportsHandler) Handle(ctx, q) (*ListUnhealedReportsResult, error) { ... }
```

Inject vào [ReconciliationHandler](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler.go#L11):
```diff
 type ReconciliationHandler struct {
     reader         recon.ReconReader
     bus            ports.CommandBus
     listLatestQ    *recon.ListLatestReportsHandler
     getHistoryQ    *recon.GetTableHistoryHandler
     listFailedQ    *recon.ListFailedLogsHandler
+    listUnhealedQ  *recon.ListUnhealedReportsHandler  // NEW
     activityLogger ports.ActivityLogger
     logger         *zap.Logger
 }
```

---

### ❌ Pattern Violation #2: Thiếu đăng ký route trong `router.go`

**Plan Rev.4 nói**: "Đăng ký route trong `reconciliation_handler.go`" — SAI. Route không đăng ký ở handler, mà ở [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go).

**Pattern chuẩn**:
- **POST destructive** (heal, check, prune) → `registerDestructive("/reconciliation/...", h.Recon.Method)` ([L173-178](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go#L173-L178))
- **GET read** (report, history) → `dual("GET", shared, "/reconciliation/...", h.Recon.Method)` ([L277-280](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go#L277-L280))

**Fix**: Thêm 2 dòng vào [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go):
```go
// Destructive routes section (L178):
registerDestructive("/reconciliation/execute-heal", h.Recon.TriggerExecuteHeal)

// Read routes section (L280):
dual("GET", shared, "/reconciliation/report/:table/unhealed", h.Recon.GetUnhealedReports)
```

---

### ❌ Pattern Violation #3: Thiếu migration file theo quy ước đánh số

**Pattern chuẩn**: Migration files được đánh số tuần tự (migration 081, 083, 084, 085...). Plan không chỉ định số migration tiếp theo.

**Fix**: Cần xác định số migration hiện tại mới nhất và tạo file migration đúng format:
```
migrations/XXXX_add_heal_stats_columns.sql
```

---

### ❌ Pattern Violation #4: Parse `stale_ids` Segment B thiếu fallback flat array

**Hiện trạng thực tế**: Code hiện tại tại [recon_heal_v4.go L143-148](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L143-L148) có **fallback**:

```go
// Nếu parse {"stale_ids": [...], "orphan_in_master": [...]} THẤT BẠI
// → fallback parse flat array ["id1", "id2"] → gán vào orphanIDs
if err := json.Unmarshal(report.StaleIDs, &staleObj); err != nil {
    var orphanIDs []string
    if err2 := json.Unmarshal(report.StaleIDs, &orphanIDs); err2 == nil {
        staleObj.OrphanInMaster = orphanIDs
    }
}
```

**Plan Rev.4 sai**: Chỉ parse struct, không có fallback → sẽ bỏ sót data từ report cũ format flat array.

**Fix**: `executeHealSegmentB` phải copy chính xác logic fallback này.

---

### ❌ Pattern Violation #5: Query unhealed dùng sai key — phải dùng `shadow_table` (Migration 085 key)

**Hiện trạng**: Từ Migration 085, key ổn định là cặp `(shadow_schema, shadow_table)`, KHÔNG phải `target_table`. Bảng `cdc_reconciliation_report` có:
- `target_table`: **KHÔNG unique** — trùng tên across-schema, segment A ghi tên shadow / B ghi tên master.
- `shadow_table`: Key pipeline ổn định.

[GetTableHistory](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go#L151-L161) đã chuyển sang dùng `shadow_table OR master_table`:

```go
where := "shadow_table = ? OR master_table = ?"
if shadowSchema != "" {
    where = "shadow_schema = ? AND shadow_table = ?"
}
```

**Plan Rev.4 sai**: Query unhealed dùng `WHERE shadow_table = :table` — chỉ khớp Segment A. Segment B report có `master_table` chứ không có `shadow_table` = `:table`.

**Fix**: Query phải cover cả 2 segment:
```sql
WHERE (shadow_table = :table OR master_table = :table)
  AND healed_at IS NULL
  AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)
ORDER BY checked_at DESC
```

Hoặc nhận thêm param `shadow_schema` giống `GetTableHistory`.

---

## Tổng hợp: Bản đồ sửa chữa Rev.4 → Rev.5

| # | Vấn đề | Mức | Fix |
|---|--------|-----|-----|
| **PV-1** | Query dùng `reader.Method()` trực tiếp | 🔴 Pattern | Tạo CQRS `ListUnhealedReportsHandler`, inject vào constructor |
| **PV-2** | Route đăng ký sai chỗ | 🔴 Pattern | Thêm vào `router.go`: `registerDestructive` + `dual("GET"...)` |
| **PV-3** | Thiếu migration file + số thứ tự | 🟡 Convention | Xác định số migration tiếp theo, tạo file `.sql` |
| **PV-4** | Parse `stale_ids` Seg B thiếu fallback flat array | 🟡 Correctness | Copy logic fallback từ `recon_heal_v4.go L143-148` |
| **PV-5** | Query unhealed dùng sai key `shadow_table` | 🔴 Correctness | Dùng `shadow_table = ? OR master_table = ?` |

---

## Kết luận

Sau 3 vòng review, tổng cộng **14 lỗ hổng** đã được phát hiện:

| Vòng | Số lỗ | Phạm vi |
|------|-------|---------|
| Review 1 (Rev.2→3) | 5 | Business logic: command rename, Seg B, legacy, JSONB, migration |
| Review 2 (Rev.3→4) | 4 | Architecture: layer violation, NATS wiring, model divergence, subscription |
| Review 3 (Rev.4→5) | 5 | Pattern compliance: CQRS, router, migration convention, fallback, query key |

Anh xác nhận em cập nhật plan **Rev.5** với 5 fix mới không ạ?
