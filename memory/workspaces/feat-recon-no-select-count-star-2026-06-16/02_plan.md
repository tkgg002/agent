# 02_plan: Recon — Loại bỏ Full Collscan Source khi Prune Orphan

## Scope

Worker `centralized-data-service`, activity reconcile, luồng `RunOrphanPrune`.

## Root Cause

`RunOrphanPrune` → `ListAllIDs` → `coll.Find({}, {_id:1})` = full collection scan Mongo.  
Với 100M+ records: ~30-60 phút, ~2.4GB RAM/Network. Không thể near-realtime.

## Giải pháp

**Watermark-bounded orphan prune** — thay full-collscan bằng 2 path:

- **Path A (normal)**: Chỉ diff shadow rows trong lookback window (7 ngày) vs source window  
  → O(window), không collscan, vài giây
- **Path B (re-seed guard)**: Khi source count <<< shadow count (nghi re-seed), stream source theo batch  
  → O(N) nhưng server-side, constant RAM, KHÔNG load all vào RAM

## Files thay đổi

1. `internal/service/recon_source_agent.go` — +`StreamAllIDsInBatches` (~40 LOC)
2. `internal/service/recon_core.go` — viết lại `RunOrphanPrune` + thêm `runOrphanPruneFull` + `batchSoftDeleteOrphans` (+100/-30 LOC)

## Definition of Done

- [ ] `go build ./...` = 0 error
- [ ] Logic Path A hoạt động đúng: chỉ prune ghost trong window
- [ ] Logic Path B (re-seed guard) hoạt động đúng: stream source batch, không RAM spike
- [ ] Existing test pass
- [ ] Verify bằng log: `orphan_prune v2` xuất hiện với `reseed=false` cho normal table

## Chi tiết kỹ thuật

Xem `09_tasks_solution_recon_no_full_scan.md`
