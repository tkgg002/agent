# Requirements — Transmute Hard Delete & Toast Trace ID

## 1. Problem Statement
1. **Master Hard Delete on Oplog Delete:**
   - Khi có oplog delete từ nguồn, Shadow lưu `_deleted = true` (tombstone).
   - Khi Transmute chạy (cả incremental và full scan), trên Master phải thực hiện **Hard Delete (DELETE CỨNG)** thay vì upsert `_deleted = true`.
2. **Master Cascade Hard Delete on Segment A Orphan Prune:**
   - Khi user chạy "Xóa bản ghi thừa ở đích (orphan)" cho Luồng A (`executeHealSegA`) hoặc daemon `RunOrphanPrune` phát hiện bản ghi không còn ở nguồn:
     - Shadow cập nhật `_deleted = true`.
     - Hệ thống phải tự động phát NATS event `cdc.cmd.transmute-shadow` để Master nhận biết và **Hard Delete** ngay lập tức.
3. **Frontend Toast Trace ID:**
   - Action Toast khi bấm "Bắt đầu đối soát" / "Heal" phải hiển thị đồng thời cả **`Job ID`** và **`Trace ID`** (đều có nút copy) để tiện tra cứu trên SigNoz/Logs.

## 2. Definition of Done (DoD)
- [x] Transmuter `processBatch` tách riêng `_deleted = true` không đưa vào Bulk Upsert, mà gom `_gpay_id` để chạy `DELETE FROM master WHERE _gpay_id IN (?)`.
- [x] `executeHealSegA` và `RunOrphanPrune` phát NATS `cdc.cmd.transmute-shadow` sau khi `UPDATE ... SET _deleted = TRUE` thành công trên Shadow DB.
- [x] `actionToast.tsx` hiển thị đồng thời `Job ID` và `Trace ID`.
- [x] `DataIntegrity.tsx` tạo `createActionTrace` và truyền `traceId` vào mutation + toast.
- [x] Go worker build pass, Frontend vite build pass 100%.
