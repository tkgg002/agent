# Requirements: Khắc phục lỗi Blind Update khi Re-snapshot dữ liệu cũ tại Shadow Table

## 1. Bối cảnh & Vấn đề
- Khi chạy Re-snapshot, `SnapshotRunner` lấy `clusterTimeMs` mới (lớn hơn `_source_ts` cũ trong DB).
- Mệnh đề `buildOCCWhereClause` trong `schema_adapter.go` chỉ kiểm tra `_source_ts < EXCLUDED._source_ts` mà bỏ qua việc kiểm tra `_hash` khi `_source_ts` tồn tại.
- Hậu quả: 1 triệu bản ghi cũ không hề thay đổi nội dung vẫn bị PostgreSQL thực thi câu lệnh `UPDATE` đè, gây:
  + Bùng nổ 1 triệu Dead Tuples (PostgreSQL MVCC Table Bloat).
  + Tăng giả `_version` (`_version + 1`) và làm sai lệch mốc `_updated_at`.
  + Tải Disk I/O & WAL logs không cần thiết.
  + Gây quá tải trigger / transmute downstream.

## 2. Mục tiêu (Goals)
1. **No-Op cho dữ liệu không đổi:** Khi dữ liệu đầu vào có `_hash` trùng khớp với `_hash` hiện tại trong bảng Shadow, PostgreSQL phải thực hiện NO-OP (không ghi đè, không tăng `_version`, không đổi `_updated_at`, 0 rows affected).
2. **Cập nhật chính xác dữ liệu thay đổi:** Khi `_hash` khác biệt VÀ timestamp `_source_ts` mới hơn hoặc bằng, PostgreSQL thực thi UPDATE bình thường.
3. **Insert bình thường cho ID mới:** Bản ghi mới chưa có trong Shadow tiếp tục được INSERT bình thường.
4. **Giữ nguyên khả năng chống Out-of-order:** Vẫn từ chối các event có timestamp cũ hơn (`_source_ts < current._source_ts`).
