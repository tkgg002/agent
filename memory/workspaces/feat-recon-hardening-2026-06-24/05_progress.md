# Progress Log — feat-recon-hardening-2026-06-24

## [2026-06-25T11:32 +07:00] [Agent: Brain/Antigravity] ✅ Dashboard Metrics + Audit DONE

### Mục tiêu phiên này
Implement 5 dashboard metrics còn thiếu (Gap 1-5 từ `09_missing_metrics_task.md`) + Audit toàn bộ quá trình.

### Thay đổi thực tế

| File | Thay đổi | Lines +/- |
|------|----------|-----------|
| `pkgs/metrics/prometheus.go` | +5 dashboard metric definitions (Source/Shadow/Master/DLQDepth/PipelineStatus) | +60 |
| `internal/service/recon/recon_tier_a.go` | Emit `SourceTableRowCount` + `ShadowTableRowCount` sau srcEst/dstTotal | +4 |
| `internal/service/recon/recon_engine_run.go` | Emit `PipelineTableStatus` per-table sau wg.Wait(); xóa goroutine sai schema | +6 / -17 |
| `internal/service/recon/recon_tier_b.go` | Emit `MasterTableRowCount` đúng chỗ (RunSegmentB + masterFull + MasterBindingRef) | +4 |
| `internal/handler/recon/dlq_handler.go` | Emit `DLQDepth.Add(1)` (sendToDLQ) + `DLQDepth.Sub(1)` (ReplayDLQ) | +9 |

### Audit phát hiện và đã fix
1. **Issue #1 — DLQDepth sai semantics**: Gauge không có `Sub(1)` → chỉ tăng như Counter → Fixed: thêm `Sub(1)` trong `ReplayDLQ()`
2. **Issue #2 — MasterTableRowCount sai schema**: `CheckAll` dùng `QualifiedTarget()` (shadow schema) với masterAgent → silent fail. Fixed: xóa goroutine, emit đúng trong `RunSegmentB` với `MasterBindingRef.masterFull`

### Build & Test Kết quả
- `go build ./internal/... ./pkgs/... ./cmd/...` → ✅ PASS
- `go test -race ./internal/service/recon/... ./internal/handler/recon/...` → ✅ PASS (2.199s + cached)
- Race detector → ✅ CLEAN

### Docs đã tạo/cập nhật
- `09_missing_metrics_task.md` — checklist [x] Done 2026-06-25T11:18
- `report_metrics_hardening_2026-06-25.md` — report đầy đủ +179/-22 lines

---

## [2026-06-25T08:57 +07:00] [Agent: Brain/Antigravity] ✅ EXECUTED — All phases DONE

### Build & Test
- `go build ./internal/... ./pkgs/... ./cmd/...` → ✅ PASS
- `go test -race ./internal/service/recon/...` → ✅ PASS (1.955s, no race)

### Phases
| Phase | Status | Files |
|-------|--------|-------|
| 4c | ✅ | `prometheus.go` — 5 metrics mới |
| 4a | ✅ | `recon_engine_run.go` — emit cycle metrics |
| 4b | ✅ | `recon_tier_a.go` — emit lag gauge |
| 2  | ✅ | `recon_engine.go` + `recon_tier_a.go` — circuit breaker |
| 5  | ✅ | `recon_dest_query.go` — EstimatedCountRows pg_class |
| 1  | ✅ | `recon_tier_a.go` — advisory lock conn pinning |
| 3  | ✅ | `recon_engine.go` + `recon_tier_a.go` — drillDownSem |
| 6  | ✅ | `recon_tier_a.go` — drillCtx 8m timeout |
| 7  | ✅ | `recon_engine.go` — Hot/Cold config fields |

### Tóm tắt phiên làm việc
- Đọc source code thực tế: `recon_tier_a.go`, `recon_engine_run.go`, `recon_engine.go`,
  `recon_dest_query.go`, `recon_query.go`, `recon_dest_models.go`, `prometheus.go`
- Phân tích và xác nhận 7 bug với evidence file:line cụ thể
- Thiết kế 7-phase fix với code demo đầy đủ
- Lưu plan vào workspace tại `02_plan.md`

### Bugs đã xác nhận (root cause có evidence)

| ID | Bug | File:Line | Status |
|----|-----|-----------|--------|
| BUG-1 | Advisory Lock Leak | `recon_tier_a.go:40` | 📋 Planned (Phase 1) |
| BUG-2 | False Drift @ lag > 60m | `recon_tier_a.go:138-207` | 📋 Planned (Phase 2) |
| BUG-3 | Thundering Herd BucketCounts | `recon_engine_run.go:169` | 📋 Planned (Phase 3) |
| BUG-4 | Cycle Metrics Dead Code | `prometheus.go:249-285` | 📋 Planned (Phase 4a) |
| BUG-5 | COUNT(*) Full Scan 50M rows | `recon_dest_query.go:22` | 📋 Planned (Phase 5) |
| BUG-6 | Timeout 45s quá ngắn cho drill-down | `recon_engine_run.go:221` | 📋 Planned (Phase 6) |
| BUG-7 | WindowLookback 7d scan lại data đã soát | `recon_engine.go:45` | 📋 Planned (Phase 7) |

### Execution Order (đã quyết định)
```
4a (fix dead-code metrics) →
4b (emit lag prometheus) →
4c (thêm 5 metrics mới) →
2  (lag circuit breaker) →
5  (pg_class estimate count) →
6  (adaptive timeout split) →
1  (advisory lock pinning) →
7  (hot/cold schedule) →
3  (drill-down semaphore)
```

### Files cần sửa (checklist cho Muscle)
- [ ] `pkgs/metrics/prometheus.go` — Thêm 5 metrics mới (Phase 4c)
- [ ] `internal/service/recon/recon_engine_run.go` — Emit cycle metrics sau wg.Wait() (Phase 4a), tách timeout (Phase 6)
- [ ] `internal/service/recon/recon_tier_a.go` — Emit lag (4b), circuit breaker (2), drillDown ctx (6), lock pinning (1), acquire drillDownSem (3)
- [ ] `internal/service/recon/recon_engine.go` — Thêm config fields (2, 7, 3), drillDownSem field + init
- [ ] `internal/service/recon/recon_dest_query.go` — Thêm EstimatedCountRows + splitSchemaTable (Phase 5)
- [ ] `internal/service/recon/recon_dest_models.go` — Split QueryTimeout (Phase 6)

### Ghi chú quan trọng khi tiếp tục
- **Phase 1** (lock pinning) cần verify với `go test -race` bắt buộc trước khi xem là done
- **Phase 5** (pg_class) cần manual verify: bảng target phải đã chạy ANALYZE hoặc autovacuum
  → `SELECT reltuples FROM pg_class WHERE relname = 'table_name'` trước khi switch
- **Phase 6** timeout split: `drillCtx` phải link parent span để trace liên tục trên SigNoz
- Context bắt buộc đọc trước khi execute: `00_context.md`, `02_plan.md` (plan đầy đủ)

---

## [2026-06-24T17:14 +07:00] [Agent: Brain/Antigravity] Workspace khởi tạo
- Đọc `active_plans.md` — không có workspace nào đang active cho task này
- Tạo workspace `feat-recon-hardening-2026-06-24`
- Đọc code thực tế, xác nhận BUG-1/2/3 từ phiên 2026-06-22
- Tạo plan v1 (3 phases)

## [2026-06-24T17:19 +07:00] [Agent: Brain/Antigravity] Deep-dive metrics
- Đọc toàn bộ `prometheus.go` (287 lines)
- Phát hiện BUG-4: dead code metrics
- Update plan v2 (4 phases)

## [2026-06-24T17:22 +07:00] [Agent: Brain/Antigravity] Scale analysis
- Đọc `recon_dest_query.go`, `recon_query.go`, `recon_dest_models.go`
- Phát hiện BUG-5/6/7 ở tầng data/I/O khi scale 50M rows
- Update plan v3 (7 phases) — hoàn chỉnh
