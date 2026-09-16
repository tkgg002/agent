# 01_requirements_fix_recon_istimestamptz.md: Yêu cầu Kỹ thuật Fix Lỗi IsColTimestamptz & Recon Deduplication

## 1. Bối cảnh
- Khi thực hiện Recon Chặng A (`source_shadow`), hệ thống phát hiện các ID vừa nằm ở `MissingFromDest` vừa nằm ở `MissingFromSrc` (ví dụ 8 ID: `49974`, `49977`, `49982`, `50002`, `50007`, `50008`, `50010`, `50013`).
- Mặc dù cột `last_updated_at` ở Shadow DB là `timestamptz`, hàm `IsColTimestamptz` trong `recon_dest_query.go` bị văng `sql.ErrNoRows` do `splitSchemaTable` fallback schema rỗng về `"public"`, trong khi tên bảng ở Shadow thuộc schema khác (ví dụ `shadow_vmg_ekyc`).
- Hậu quả: `IsColTimestamptz` văng error $\rightarrow$ Recon fallback `isTZ = false` $\rightarrow$ Tự động cộng 7 tiếng (+7h) vào tham số query Postgres $\rightarrow$ Trôi lệch sub-window 15-phút giữa Source Mongo và Shadow Postgres $\rightarrow$ Báo giả "Thừa ở Shadow (Missing from Src)".
- Đồng thời `ChunkStreamBucketEngine` thiếu bước Deduplicate giữa `MissingFromDest` và `MissingFromSrc`.

## 2. Yêu cầu Sửa đổi (Requirements)
- **R1:** Nâng cấp `IsColTimestamptz` trong `internal/service/recon/recon_dest_query.go`: Khi query theo `table_schema` rỗng/default `"public"` không tìm thấy cột, tự động fallback query `information_schema.columns` theo `table_name` + `column_name` (loại trừ `pg_catalog`, `information_schema`).
- **R2:** Bổ sung bước Khử trùng lặp (Deduplication / Cross-Check) trong `ChunkStreamBucketEngine` (`recon_stream_bucket_engine.go`) để di chuyển các ID bị trùng giữa `MissingFromDest` và `MissingFromSrc` sang `Mismatched`.
- **R3:** Đảm bảo 100% Unit tests hiện có chạy PASS.
