# Audit Log & Progress: Fix Execute Heal Segment B & Performance Optimization

## Audit Log
- [2026-07-22 16:43:35] [Agent:Gemini-3.6-Flash] Ghi nhận bài học kinh nghiệm vào `lessons.md`.
- [2026-07-23 13:47:02] [Agent:Gemini-3.6-Flash] Phân tích log lỗi `SQLSTATE 42703`.
- [2026-07-23 13:47:40] [Agent:Gemini-3.6-Flash] Unit test dynamic column `TestExecuteHealSegB_PruneMasterSQL` PASS 100%.
- [2026-07-23 15:48:11] [Agent:Gemini-3.6-Flash] Tạo file auto-migration `096_optimize_recon_indexes.sql`.
- [2026-07-23 17:00:41] [Agent:Gemini-3.6-Flash] Đã loại bỏ hoàn toàn câu lệnh `COALESCE(NULLIF(_source_id, ''), _gpay_id::text)` khỏi `resolveSourceIDsForSegmentB`. Phân nhánh 100% theo `idType`:
  - `idType == "gpay"`: Query `SELECT "_gpay_id"::text FROM qualified WHERE "_gpay_id" IN (?)` dùng tham số `[]int64` ➔ B-Tree Index trên Shadow DB.
  - `idType == "id"`: Query `SELECT %s::text FROM qualified WHERE %s IN (?)` dùng tham số `[]int64` / `[]string` tương ứng theo `PrimaryKeyType` ➔ B-Tree Index trên Shadow DB.
  - Unit test `go test ./internal/handler/recon/...` PASS 100%.
