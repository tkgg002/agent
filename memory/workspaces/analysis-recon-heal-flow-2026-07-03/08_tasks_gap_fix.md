# Tasks — Fix Rủi Ro Vận Hành Recon & Heal

> Nguồn: `10_gap_analysis.md` | Branch: `recon-heal` trên cả 3 repo

---

## Fix 1: Race Condition Guard 🔴
**Repo:** centralized-data-service

- [x] 1.1 Thêm method `ClaimForHealing(ctx, id)` + `ReleaseHealClaim()` vào `reconciliation_report_repo.go`
  - Atomic: `UPDATE ... SET status='healing' WHERE id=? AND status NOT IN ('healing','healed')`
  - Return (report, true, nil) nếu claim thành công, (nil, false, nil) nếu bị worker khác claim
- [x] 1.2 Sửa `executeHeal()` trong `recon_execute_heal.go`
  - Trước khi xử lý mỗi report: gọi `ClaimForHealing()`
  - Nếu claim fail → log warn + skip
  - Khi lỗi/unknown segment → `ReleaseHealClaim()` revert status

---

## Fix 2: Chunk SegA IDs 🟡
**Repo:** centralized-data-service

- [x] 2.1 Thêm constant `segAChunkSize = 1000` vào `recon_execute_heal.go`
- [x] 2.2 Tách helper `fetchAndWriteChunked()` — chunk `ids` thành batches 1000 trước khi gọi `FetchAndWriteByIDs()`
  - Sửa `executeHealSegA()` dùng helper thay vì call trực tiếp

---

## Fix 3: Safety Gate cho Interactive Heal 🔴
**Repo:** centralized-data-service + cdc-cms-service + cdc-cms-web

- [x] 3.1 Worker: Thêm `ForceHeal bool` vào `executeHealOpts` struct
- [x] 3.2 Worker: Thêm constant `interactiveHealMaxIDs = 50000`
- [x] 3.3 Worker: Trong `executeHeal()`, tính tổng IDs → nếu > threshold + !ForceHeal → return error
- [x] 3.4 API Gateway: Thêm `ForceHeal bool` vào `ExecuteHealCommand` struct trong `recon_async.go`
- [x] 3.5 API Gateway: Thêm `ForceHeal` mapping trong `TriggerExecuteHeal` handler
- [x] 3.6 FE: Thêm `force_heal` vào `ExecuteHealPayload` interface + `useExecuteHealMutation`
- [x] 3.7 FE: Thêm confirmation dialog trong `ExecuteHealModal.tsx` khi nhận error threshold

---

## Verify

- [x] 4.1 Build centralized-data-service (`go build ./internal/...`) ✅
- [x] 4.2 Build cdc-cms-service (`go build ./internal/... ./cmd/...`) ✅ 
- [x] 4.3 TypeScript type check cdc-cms-web (`tsc --noEmit`) ✅
