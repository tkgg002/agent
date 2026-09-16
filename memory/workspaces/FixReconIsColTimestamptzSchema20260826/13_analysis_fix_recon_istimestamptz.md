# 13_analysis_fix_recon_istimestamptz.md: Kết quả Phân tích Kỹ thuật

## 1. Phân tích Nguyên nhân Gốc rễ
- Commit `f80b03e` (25/08) refactor `ReconJobWorker` để dùng `GetTableConfig(lookupKey)`.
- Khi `entry.ShadowSchema` rỗng, `entry.QualifiedTarget()` trả về `"bank_requests"`.
- `IsColTimestamptz` query `information_schema.columns WHERE table_schema = 'public'` bị `sql.ErrNoRows` $\rightarrow$ Fallback `isTZ = false` $\rightarrow$ Tự động cộng 7 tiếng (+7h) $\rightarrow$ Trôi lệch window 15-phút giữa Mongo và Postgres $\rightarrow$ Báo giả "Thừa ở Shadow".

## 2. Kết quả Phân tích sau khi Sửa
- Ngay khi `Execute` chạy, `entry.ShadowSchema` được tự động phân giải thành `"shadow_vmg_ekyc"`.
- `entry.QualifiedTarget()` trả về `"shadow_vmg_ekyc.bank_requests"`.
- `IsColTimestamptz` query đúng `table_schema = 'shadow_vmg_ekyc'` $\rightarrow$ Trả về `isTZ = true` chuẩn xác $\rightarrow$ Không trôi 7h $\rightarrow$ Lỗi báo giả "Thừa ở Shadow" bị dứt điểm 100%.
