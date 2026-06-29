# Audit — Session 2026-06-25T14 (SigNoz Dashboard + Active Row Count)

> **Auditor**: Brain/Antigravity
> **Date**: 2026-06-25T14:44 +07:00
> **Scope**: `centralized-data-service` — Dashboard chart fixes + shadow active row count

---

## 1. Những gì session này đã làm

### Task A — Fix dashboard chart cardinality
**Vấn đề**: Chart "Transmute Ops by Type" group by `[master_table, op]` → 200 tables × 5 ops = 1000 lines, không đọc được.
**Fix**: Tách thành 2 panels:
- Graph panel: group by `[op]` only → 5 lines cố định
- Table panel w11 (drill-down): group by `[master_table, op]` — dùng khi debug
- Tương tự cho Sink: group by `[op]` only thay vì `[table, op]`

**File**: `deployments/signoz-dashboard-recon.json` — +2 panels (w11, w12, w13)

---

### Task B — Shadow/Master active row count (_deleted=false)
**Vấn đề user**: Source=452, Shadow/Master=457 (có 5 tombstones `_deleted=true`). Dashboard hiển thị diff=5 nhưng thực ra không có drift — chỉ là tombstones.
**Approach**: Emit 2 metrics mới:
- `cdc_shadow_active_row_count{table}` = pg_class_estimate - COUNT(WHERE _deleted=true)
- `cdc_master_active_row_count{table}` = (TBD — chưa có master dest agent trong recon path)

**Implementation thực tế**:
1. `prometheus.go`: Thêm `ShadowActiveRowCount` + `MasterActiveRowCount` (GaugeVec)
2. `recon_dest_query.go`: Thêm `CountDeletedRows()` → COUNT(*) WHERE _deleted=true (O(index scan))
3. `recon_tier_a.go`: Emit `shadowActive = dstTotal - deletedCount`

**User xóa `CountDeletedRows`** mid-session → emit block trong tier_a cũng bị xóa.

---

## 2. Sai sót & Thiếu sót phát hiện

### 🔴 CRITICAL — `ShadowActiveRowCount` + `MasterActiveRowCount` là DEAD CODE

**Vấn đề**: Sau khi user xóa `CountDeletedRows`, cả block emit trong `recon_tier_a.go` đã bị gỡ. Nhưng `prometheus.go` vẫn còn `ShadowActiveRowCount` + `MasterActiveRowCount` declarations.

**Kết quả**: Build pass nhưng 2 metrics này sẽ luôn = 0 trên SigNoz. Dashboard panels w12/w13 sẽ trống.

**Root cause của error trước đó**: CountDeletedRows phạm vào rule "không được thêm logic mới vào recon_dest_query.go" nếu user muốn giữ scope tối thiểu. User xóa thủ công.

**Fix cần làm**:
- Hoặc: Tái implement `CountDeletedRows` + emit (nếu user muốn feature)
- Hoặc: Xóa 2 metric declarations dead code ra khỏi `prometheus.go` + xóa panels w12/w13 khỏi dashboard

### 🟡 MEDIUM — `MasterActiveRowCount` chưa được emit

**Vấn đề**: Ngay cả khi `CountDeletedRows` còn tồn tại, chỉ emit được `ShadowActiveRowCount`. `MasterActiveRowCount` không có emit point vì `ReconCore` không có `masterDestAgent` (chỉ có 1 `destAgent` trỏ vào shadow DB).

**Root cause**: Master DB là DB riêng (khác shadow DB). Không thể dùng `destAgent` để query master. Cần `masterAgent` — đã tồn tại trong `recon_tier_b.go` thông qua `MasterBindingRef`.

**Fix đúng**: Emit `MasterActiveRowCount` trong `RunSegmentB` — cùng chỗ emit `MasterTableRowCount` (đã xác nhận tại report cũ issue #2).

### 🟡 MEDIUM — `CountDeletedRows` không phải pattern của hệ thống

**Vấn đề**: Tất cả methods trong `recon_dest_query.go` đều là O(1) hoặc index-scan với _source_ts. `CountDeletedRows` là query COUNT bình thường — nhưng ổn nếu có partial index `WHERE _deleted = true`.

**Thiếu sót**: Code không có comment/doc về việc cần index. Nếu deploy mà chưa tạo index:
```sql
CREATE INDEX CONCURRENTLY ON <shadow_table> (_deleted) WHERE _deleted = true;
```
thì `CountDeletedRows` vẫn là full table scan trên mặc định.

**Kết luận**: Không phải "fix bẩn" nhưng cần đảm bảo precondition (index) trước khi activate.

### ✅ KHÔNG vi phạm — chart cardinality fix

Dashboard fix (Task A) hoàn toàn đúng:
- Không chạm code Go
- Không ảnh hưởng Source→Shadow pipeline
- Đúng architecture: graph cho overview, table cho drill-down
- JSON valid, build không liên quan

---

## 3. Architecture & Pattern Compliance

| Rule | Kiểm tra | Kết quả |
|------|----------|---------|
| "Core Systems Only" | Không thêm external deps, không thay đổi config | ✅ |
| "Minimal impact" | metrics.go chỉ thêm definitions, không sửa logic | ✅ |
| "No fix bẩn" | CountDeletedRows dùng đúng breaker+readOnlyDB pattern | ✅ |
| "Scope restriction Source→Shadow vs Shadow→Master" | ShadowActiveRowCount emit từ recon (không phải CDC handler) | ✅ |
| "Pattern: metrics.Xxx.WithLabelValues(...).Set()" | Đúng pattern | ✅ |
| "Build pass trước khi done" | Build ✅ PASS | ✅ |
| `recon_dest_query.go` pattern | `da.breaker.Execute` + `da.readOnlyDB(ctx)` + `validateIdent` | ✅ |

---

## 4. Files Thực Tế Đã Thay Đổi (Session này)

| File | Thay đổi | Lines +/- |
|------|----------|-----------|
| `pkgs/metrics/prometheus.go` | +2 metric defs: ShadowActiveRowCount + MasterActiveRowCount | +22 / 0 |
| `internal/service/recon/recon_tier_a.go` | +emit block (sau đó user xóa CountDeletedRows → block bị gỡ luôn) | +0 thực tế |
| `internal/service/recon/recon_dest_query.go` | +CountDeletedRows → user xóa | +0 thực tế |
| `deployments/signoz-dashboard-recon.json` | Chart cardinality fix + panels w11/w12/w13 | +90 /-30 |

**Thực tế còn lại sau session**: `prometheus.go` +22 lines dead code, dashboard +90/-30.

---

## 5. Action Items cần làm ngay

### AIM-1 🔴 — Quyết định số phận 2 metrics dead code

**Chọn 1 trong 2**:

**A — Giữ feature, implement đúng**:
1. Re-add `CountDeletedRows` vào `recon_dest_query.go` (pattern đã có)
2. Emit `ShadowActiveRowCount` tại `recon_tier_a.go` sau `dstTotal`
3. Emit `MasterActiveRowCount` tại `recon_tier_b.go` sau `masterFull` (cùng chỗ emit MasterTableRowCount)
4. Document precondition index `WHERE _deleted=true`

**B — Remove dead code**:
1. Xóa `ShadowActiveRowCount` + `MasterActiveRowCount` khỏi `prometheus.go`
2. Xóa panels w12/w13 khỏi `signoz-dashboard-recon.json`
3. Dùng dashboard logic: `source_row_count - shadow_row_count` để thấy diff

### AIM-2 🟡 — Cập nhật 05_progress.md với session này

Chưa ghi session 14:44 vào progress log.

---

## 6. Verdict

**Session này**: 
- ✅ Chart cardinality fix — ĐÚNG, clean, production-ready
- 🔴 Active row count feature — INCOMPLETE (metrics registered nhưng không được emit)
- 🟡 Không có report_*.md tạo cho session này (vi phạm rule #10)

**Build**: ✅ PASS
**Test**: Chưa chạy lại sau thay đổi session này
