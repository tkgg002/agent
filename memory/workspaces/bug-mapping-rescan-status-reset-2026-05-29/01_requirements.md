# 01_requirements

## Functional
1. **Khi user nhấn "Scan Unmapped Fields"** (`/api/introspection/scan-raw/:table`), TẤT CẢ mapping rule của shadow table đó RESET về `status='pending', is_active=false`.
   - User explicit: "khi quét cái mới thì phải về pending hết"
2. **Thêm 1 cột "In Shadow"** trên bảng Mapping Rules:
   - `true` nếu `target_column` tồn tại trong `information_schema.columns` của `shadow_schema.shadow_table`
   - `false` nếu chưa tồn tại
3. **System Default Fields card** hiển thị đủ 11 entries (hiện 8) — kèm cờ "đã tạo ở table shadow chưa":
   - 1 PK (`id` hoặc `_id` rename → `id`)
   - 10 CDC cột: `source_id, _raw_data, _source, _source_ts, _synced_at, _version, _hash, _deleted, _created_at, _updated_at`
4. Sau khi scan thành công, FE refetch rules + shadow-columns để hiển thị status mới.

## Non-functional
- Theo core-systems pattern, không cheat DB hay config.
- Minimal impact: chỉ chạm file cần thiết.
- Verify build BE worker + BE cms + FE tsc trước khi report.
- Report file `report_*.md` với danh sách file thay đổi + LOC delta.
