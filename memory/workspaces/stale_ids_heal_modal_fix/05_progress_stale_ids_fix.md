# Audit Log & Progress: Fix Stale IDs Display & Execution Logic

## Audit Log
- [2026-07-22 11:57:30] [Agent:Gemini-3.6-Flash] Phân tích Root Cause: Frontend `ExecuteHealModal.tsx` đọc nhầm key `missing_from_shadow` và `missing_from_master` đối với `segment === 'shadow_master'`.
- [2026-07-22 11:57:30] [Agent:Gemini-3.6-Flash] Khởi tạo workspace docs: 01_requirements, 05_progress, 08_tasks, 12_implementation_plan, 13_analysis.
- [2026-07-22 13:11:40] [Agent:Gemini-3.6-Flash] Đã kiểm tra chi tiết 3 file Backend Go (`recon_base_handler.go`, `recon_tier_a.go`, `recon_tier_b.go`). Phát hiện sự không đồng nhất giữa các luồng engine backend.
- [2026-07-22 15:15:45] [Agent:Gemini-3.6-Flash] Rà soát và hoàn tất sửa lỗi trên Backend Go & Frontend React.
- [2026-07-22 16:14:40] [Agent:Gemini-3.6-Flash] Dọn dẹp triệt để 100% rác kỹ thuật cũ.
- [2026-07-22 16:25:22] [Agent:Gemini-3.6-Flash] Đã sửa triệt để 2 bug trên Frontend Popover (`ExecuteHealModal.tsx`):
  1. Fix logic merge `missing_ids`: Luôn nhét vào `missingFromDest` (thiếu ở Đích) thay vì nhét nhầm vào `missingFromSrc` khi `segment === 'shadow_master'`.
  2. Fix đúng 100% nhãn Popover: `missingFromDest` -> 'Thiếu ở Master (Missing from Dest)' cho Chặng B; `missingFromSrc` -> 'Thiếu ở Shadow (Missing from Src)' cho Chặng B.
