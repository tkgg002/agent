# Report — Metrics Hardening & Dashboard Gaps

> **Task**: feat-recon-hardening — Phase bổ sung 5 dashboard metrics  
> **Date**: 2026-06-25T11:25 +07:00  
> **Auditor**: Brain/Antigravity  
> **Scope**: `centralized-data-service`

---

## 1. Những File Thực Tế Đã Thay Đổi

| File | Thay đổi | Lines Added | Lines Removed |
|------|----------|-------------|---------------|
| `pkgs/metrics/prometheus.go` | Thêm 10 metric definitions (5 hardening + 5 dashboard) | +116 | 0 |
| `internal/service/recon/recon_tier_a.go` | Emit `SourceTableRowCount`, `ShadowTableRowCount` + `detachedSpanContext` helper | +28 | -5 |
| `internal/service/recon/recon_engine_run.go` | Emit `PipelineTableStatus`, `MasterTableRowCount` async, `CycleDuration`, `cycleStart` | +35 | -1 |
| `internal/handler/recon/dlq_handler.go` | Import metrics + emit `DLQDepth.Add(1)` | +5 | 0 |

**Tổng cộng**: ~+184 lines added, ~6 lines removed  
> *Dựa trên git diff HEAD (prometheus.go: +116) + wc -l cho untracked files*

---

## 2. Audit — So sánh với Plan (02_plan.md)

### ✅ Đã hoàn thành đúng kế hoạch

| Phase | Plan | Thực tế |
|-------|------|---------|
| Phase 4a | Fix dead code metrics (CycleTotal, CycleTablesChecked...) | ✅ Đã emit đủ 4 cycle metrics |
| Phase 4b | `cycleStart` timestamp | ✅ `cycleStart := now` — dùng lại biến sẵn có |
| Phase 1 | Advisory lock conn pinning | ✅ `withTableLock` dùng `sql.Conn` pinned |
| Phase 2 | Circuit breaker max lag 60m | ✅ `MaxTolerableLagMs` check + `ReconCircuitBreakerTrips` |
| Phase 3 | DrillDown semaphore | ✅ `drillDownSem` + `ReconDrillDownWaitMs` |
| Phase 5 | O(1) pg_class estimate | ✅ `EstimatedCountRows` + `GREATEST(reltuples,0)` |
| Phase 6 | Context hardening — detachedSpanContext | ✅ `context.WithTimeout(detachedSpanContext(ctx), 8m)` |
| Phase 7 | HotWindowLookback | ✅ `effectiveLookback()` + `RunMode` config |
| Gap 1 | `cdc_source_table_row_count` | ✅ Emit sau `srcEst` trong RunTier1 |
| Gap 2 | `cdc_shadow_table_row_count` | ✅ Emit sau `dstTotal` trong RunTier1 |
| Gap 3 | `cdc_master_table_row_count` | ✅ Emit async goroutine — `masterAgent` được wire qua `SetMasterAgent()` |
| Gap 4 | `cdc_dlq_depth` | ✅ Emit trong `dlq_handler.go` sau `PublishMsg` success |
| Gap 5 | `cdc_pipeline_table_status` | ✅ Emit per-table sau `wg.Wait()` trong `CheckAll` |

---

## 3. Audit — Phát hiện Sai sót & Thiếu sót

### 🔴 Issue #1 — `DLQDepth` dùng sai kiểu metric (NHẦM KIẾN TRÚC)

**Vấn đề**: `DLQDepth` được define là `GaugeVec` nhưng được gọi `.Add(1)` — đây là cách dùng Counter, không phải Gauge.

```go
// ❌ Hiện tại — DLQDepth là Gauge nhưng chỉ Add(1), không bao giờ Set() hay Sub()
metrics.DLQDepth.WithLabelValues(sourceTable).Add(1)
```

**Root cause**: Gauge có thể tăng/giảm (dùng cho queue depth thực tế). Nhưng nếu không có code để `Sub(1)` khi message được replay/cleared, thì `DLQDepth` chỉ một chiều tăng — hành xử giống Counter.

**Fix đúng**: 2 options:
- **Option A** (Đơn giản nhất): Đổi `DLQDepth` thành `CounterVec` và rename thành `DLQEnqueuedTotal` — semantics rõ ràng, không gây hiểu nhầm
- **Option B** (Đầy đủ hơn): Giữ `GaugeVec`, nhưng phải thêm `DLQDepth.Sub(1)` trong `ReplayDLQ()` khi message được replay thành công

**Quyết định**: Option B — đúng với naming "depth" (chiều sâu queue thực tế), và `ReplayDLQ()` đã có sẵn trong `dlq_handler.go`.

---

### 🟡 Issue #2 — `MasterTableRowCount` dùng `e.QualifiedTarget()` nhưng Master DB schema có thể khác Shadow

**Vấn đề**: `entry.QualifiedTarget()` trả về qualified name của **Shadow** table (e.g., `shadow_schema.doctors`). Master DB có thể dùng schema khác (e.g., `public.doctors` hay `master.doctors`).

```go
// ❌ Có thể sai schema
rc.masterAgent.EstimatedCountRows(mCtx, e.QualifiedTarget())
// QualifiedTarget() → "shadow_schema.doctors"
// Nhưng master DB table là "public.doctors" hay schema khác!
```

**Kiểm tra**:

---

### 🟡 Issue #3 — Thiếu `report_*.md` theo quy tắc #10 & #35

Theo rule: *"Bắt buộc phải tạo file `report_*.md` ghi rõ lý do thay đổi và danh sách 'Những file thực tế đã sửa', 'Số lượng dòng code thay đổi'"*

File này là report đó — **nhưng cần được tạo trước khi báo DONE, không phải sau audit**.  
→ Sẽ không vi phạm nữa từ session tiếp theo.

---

### ✅ Không vi phạm các rule quan trọng

| Rule | Check | Kết quả |
|------|-------|---------|
| "Core Systems Only — no fix bẩn" | Tất cả emit points dùng đúng metric API, không hardcode | ✅ |
| "Minimal impact" | Chỉ thêm, không xóa/sửa logic business | ✅ |
| "Scope restriction — Shadow→Master vs Source→Shadow" | `DLQDepth` emit trong dlq_handler (cross-cutting), không chạm luồng Source→Shadow | ✅ |
| `masterAgent` nil-check | Code kiểm tra `if rc.masterAgent != nil` trước khi dùng | ✅ |
| Architecture pattern | `metrics.Xxx.WithLabelValues(...).Set/Inc/Observe()` — đúng pattern hiện tại | ✅ |
| Build + test | `go build ./internal/... ./pkgs/...` PASS; `go test -race` PASS (1.977s + 1.803s) | ✅ |

---

## 4. Fix Đã Thực Hiện Sau Audit

### Fix Issue #1 — `DLQDepth.Sub(1)` trong `ReplayDLQ()` ✅

**File**: `internal/handler/recon/dlq_handler.go`
**Thay đổi**: Thêm `metrics.DLQDepth.WithLabelValues(msg.SourceTable).Sub(1)` sau `PublishMsg` replay thành công.
**Kết quả**: `cdc_dlq_depth` bây giờ là Gauge thực sự — tăng khi vào DLQ, giảm khi replay ra. SigNoz panel sẽ phản ánh depth chính xác.

### Fix Issue #2 — `MasterTableRowCount` emit đúng schema ✅

**Vấn đề phát hiện**: Code ban đầu emit trong `CheckAll` với `e.QualifiedTarget()` (shadow schema) qua `masterAgent` → sai schema, metric sẽ luôn fail silently.

**Fix**: 
- **Xóa** goroutine sai trong `recon_engine_run.go`
- **Thêm** emit đúng chỗ trong `recon_tier_b.go:RunSegmentB` — nơi `masterFull` đã được đo từ `masterAgent.CountRows(ctx, masterRel, ...)` với `masterRel = MasterSchema.MasterTable` đúng

```go
// recon_tier_b.go — Sau Tier-0 match, dùng masterFull đã verified
if errMF == nil {
    metrics.MasterTableRowCount.WithLabelValues(ref.MasterTable).Set(float64(masterFull))
}
```

**Lý do đúng về architecture**: `RunSegmentB` nhận `MasterBindingRef` với `MasterSchema` + `MasterTable` từ DB binding — đây là source of truth duy nhất cho master table qualified name. `CheckAll` chỉ có `source.TableRegistry.QualifiedTarget()` là shadow schema.

---

## 5. Kết quả Verification Cuối

```
go build ./internal/... ./pkgs/... ./cmd/...     → PASS
go test -race ./internal/service/recon/...       → ok (2.199s)
go test -race ./internal/handler/recon/...       → ok (cached)
Race detector                                    → CLEAN
```

---

## 6. Files Thực Tế Đã Thay Đổi (Final)

| File | Lines Added | Lines Removed | Ghi chú |
|------|-------------|---------------|---------|
| `pkgs/metrics/prometheus.go` | +116 | 0 | 10 metric definitions |
| `internal/service/recon/recon_tier_a.go` | +28 | -5 | detachedSpanContext + row count emit |
| `internal/service/recon/recon_engine_run.go` | +22 | -17 | PipelineTableStatus + fix remove bad goroutine |
| `internal/service/recon/recon_tier_b.go` | +4 | 0 | MasterTableRowCount emit đúng chỗ |
| `internal/handler/recon/dlq_handler.go` | +9 | 0 | DLQDepth Add(1) + Sub(1) |
| **Tổng** | **+179** | **-22** | |
