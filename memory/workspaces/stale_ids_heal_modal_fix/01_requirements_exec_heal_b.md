# Requirements Spec: Fix Execute Heal Segment B (Shadow ➔ Master) ID Mapping

## 1. Context & Issue Summary
- **Hiện trạng**: Khi thực hiện `execute-heal` cho Chặng B (`shadow_master`), quá trình kết thúc nhưng không có bản ghi nào được heal (0 healed).
- **Root Cause**: 
  - Recon Engine Chặng B (`recon_stream_bucket_engine.go`, `recon_tier_b.go`) ghi nhận danh sách ID chênh lệch (`missing_ids`, `stale_ids`) theo cột `_source_id` / `id` (ví dụ Mongo `_id` `"44702"`).
  - Tuy nhiên, `ExecuteHealHandler` Chặng B ([recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)) lại lầm tưởng các ID đó là `_gpay_id` (Sonyflake int64 ID), và gọi `mapGpayToSourceIDs` chạy SQL `SELECT _source_id FROM shadow WHERE _gpay_id IN ('44702', ...)`.
  - Do `_gpay_id` là số Sonyflake còn `"44702"` là `_source_id`, câu SQL không match bản ghi nào (hoặc ném lỗi SQL mismatch type). Kết quả `mapGpayToSourceIDs` trả về mảng rỗng `[]`, Transmute Worker nhận `_source_ids: []` và không thực hiện transmute bất kỳ bản ghi nào.
  - Tương tự, lệnh Prune Master DB đang dùng `WHERE "_gpay_id" IN (?)` thay vì `WHERE "_source_id" IN (?)`.

## 2. Business & Technical Requirements
- **Req-1**: Sửa `executeHealSegB` nhận trực tiếp danh sách ID từ Report dưới dạng `_source_ids` (hoặc linh hoạt hỗ trợ cả `_source_id` lẫn `_gpay_id`), loại bỏ việc map nhầm từ `_gpay_id` sang `_source_id`.
- **Req-2**: Cập nhật hàm helper resolution ID để thử cả `_source_id` và `_gpay_id` nếu cần, đảm bảo mảng `sourceIDs` truyền sang Transmute Worker `cdc.cmd.transmute` luôn chính xác 100%.
- **Req-3**: Sửa lệnh Prune Master trong `executeHealSegB` dùng `WHERE "_source_id" = ANY(?)` (hoặc `_source_id IN (?)` / `_gpay_id::text IN (?)`), hỗ trợ xóa chính xác các bản ghi dư thừa.
- **Req-4**: Đảm bảo không làm ảnh hưởng đến Chặng A (`source_shadow`).
