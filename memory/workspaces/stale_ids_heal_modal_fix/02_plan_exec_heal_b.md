# High-Level Plan: Fix Execute Heal Segment B ID Mapping

## 1. Objectives
Khắc phục triệt để lỗi không chạy heal được ở Chặng B (`shadow_master`) do sự bất đồng nhất về ID (`_source_id` vs `_gpay_id`) giữa Recon Report và Execute Heal Handler.

## 2. Technical Approach
1. **Refactor ID Resolution trong `recon_execute_heal_handler.go`**:
   - Viết hàm helper `resolveSourceIDsForSegmentB(ctx, shadowRel, reportIDs)`:
     - Kiểm tra nếu `reportIDs` đã là `_source_id` (hiện diện trong cột `_source_id` của Shadow DB hoặc match định dạng string ID) -> Sử dụng trực tiếp `reportIDs`.
     - Nếu `reportIDs` là `_gpay_id` (Sonyflake int64 string) -> Truy vấn `SELECT COALESCE(_source_id, _gpay_id::text) FROM shadowRel WHERE _gpay_id::text IN (?)` để chuyển đổi.
2. **Fix Prune Master SQL**:
   - Chuyển `DELETE FROM master WHERE "_gpay_id" IN (?)` thành:
     `DELETE FROM master WHERE "_source_id" IN (?) OR "_gpay_id"::text IN (?)`
3. **Verify & Test**:
   - Chạy `go test ./internal/handler/recon/...` để đảm bảo passthrough 100%.

## 3. Skills Declared
- `view_file`
- `replace_file_content`
- `run_command`
- `write_to_file`
