# High-Level Plan: Fix Stale IDs across Backend and Frontend

## 1. Technical Audit Findings
- `recon_base_handler.go`: `staleSegmentA` thiếu tag `missing_from_dest`, `staleSegmentB` thiếu fallback key chuẩn `missing_from_dest` / `missing_from_src`.
- `recon_tier_a.go`: Marshal JSON `stale_ids` với 3 key chuẩn (`missing_from_dest`, `missing_from_src`, `mismatched`).
- `recon_tier_b.go`: Marshal JSON `stale_ids` chỉ có legacy keys (`missing_from_master`, `missing_from_shadow`). Cần bổ sung alias chuẩn (`missing_from_dest`, `missing_from_src`).
- `ExecuteHealModal.tsx` (FE): Chỉ đọc key legacy ở Chặng B. Cần đọc linh hoạt cả key legacy lẫn key chuẩn.

## 2. Roadmap
- Phase 1: Update Go Backend (`recon_base_handler.go`, `recon_tier_a.go`, `recon_tier_b.go`).
- Phase 2: Update Frontend React (`ExecuteHealModal.tsx`).
- Phase 3: Verification (Go tests + FE build + UI check).
