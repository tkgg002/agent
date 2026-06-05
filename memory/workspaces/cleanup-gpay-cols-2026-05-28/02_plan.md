# 02_plan — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Phase 1 — Audit (DONE)
1. Grep 104 references, phân loại theo path.
2. Đọc mỗi path để xác định semantic + data type.
3. Đối chiếu `_gpay_source_id` vs `source_id`, `_gpay_deleted` vs `_deleted` ở từng path.
4. Cross-check lesson `2026-05-20 "Bump dependency version anti-pattern"` về over-correct.

## Phase 2 — Đề xuất 3 cleanup option (Brain phase, DOCUMENT ONLY)
- **Option A (Conservative — Rollback only)**: chỉ revert Bug #2 portion ở `command_handler.go` (3 patch site). Không đụng master/sinkworker/transmuter.
- **Option B (Mid — FE Shadow Unify)**: bỏ `_gpay_*` ở tất cả FE-shadow paths (command_handler + event_handler tombstone INSERT). Cập nhật `mapping_preview_handler.go` chuyển sang đọc `source_id`. Không đụng master path.
- **Option C (Full — Master Refactor)**: bỏ `_gpay_*` cả ở master + sinkworker + transmuter + DDL. Yêu cầu migration data và prod-coordination.

Mỗi option có code demo + verify plan + risk profile trong `09_tasks_solution_cleanup.md`.

## Phase 3 — User decision gate
Trình 3 option ngắn gọn. User pick A/B/C → sang Phase 4.

## Phase 4 — Muscle apply (chỉ khi có user verb)
1. Apply patch theo option đã chọn.
2. Build verify 3 service.
3. Test handler + service package PASS.
4. Sanity check FE build.
5. Ghi `report_fix_cleanup_gpay_2026-05-28.md` + APPEND `05_progress.md`.

## Phase 5 — Security gate (§8)
- `/security-agent` sau Muscle phase (nếu user yêu cầu).

## Risk register
| Risk | Likelihood | Mitigation |
|---|---|---|
| User pick C → master tables prod đã có data, bỏ `_gpay_*` = phá UNIQUE INDEX | Med | Trình rõ trước; option C bao gồm DROP + recreate INDEX + data migration |
| Bỏ `_gpay_*` ở event_handler tombstone INSERT trong khi master path đọc `_gpay_*` từ shadow → mismatch | Med | Verify chain trước; nếu transmuter đọc `_gpay_source_id` từ shadow nhưng shadow ghi `source_id` → broken. Đó là drift hiện hữu, option B phải sửa transmuter shadowBatchRow column mapping |
| Pick A nhưng `mapping_preview_handler.go` đã broken vì shadow_automator path-A không tạo `_gpay_source_id` | Low | A không sửa preview; tiếp tục broken cho path-A. Đề xuất GAP item theo dõi |
| FE `MasterRegistry.tsx` placeholder hint user nhập `_gpay_source_id` → confuse khi backend đổi | Low | Cập nhật placeholder ở mỗi option |
