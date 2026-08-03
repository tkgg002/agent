# Tasks: Fix Stale IDs Display & Execution Logic

- [x] Task 1: Sửa struct & parse helper trong `recon_base_handler.go` (bổ sung `MissingFromDest` vào `staleSegmentA`, fallback key chuẩn trong `parseStaleSegmentB`).
- [x] Task 2: Bổ sung alias key chuẩn (`missing_from_dest`, `missing_from_src`) trong `recon_tier_b.go`.
- [x] Task 3: Bổ sung `MissingFromDest` vào danh sách ID missing khi Heal trong `recon_execute_heal_handler.go` (Chặng A & Chặng B).
- [x] Task 4: Bổ sung `MissingFromDest` vào danh sách ID propose Heal trong `recon_check_heal_handler.go`.
- [x] Task 5: Cập nhật hàm `getDiffIDs` và render Popover trong `ExecuteHealModal.tsx` (FE) hỗ trợ đọc linh hoạt cả 2 bộ key.
- [x] Task 6: Run verification tests (`go test` & `npx tsc`) — PASS 100%.
