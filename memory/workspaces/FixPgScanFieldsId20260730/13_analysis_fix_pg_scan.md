# Phân Tích Nguyên Nhân - Scan Fields Thiếu Cột ID Bảng PostgreSQL

## Root Cause
Khi gọi API Scan Fields:
1. `InferSourceColumns` truy vấn PostgreSQL `information_schema.columns`.
2. Vòng lặp `inferPGCols` kiểm tra `if strings.EqualFold(name, pkColumn)` (với `pkColumn` = `"id"`), cố tình bỏ qua cột `id`.
3. Do đó, cột `id` bị loại bỏ và không được ghi nhận vào danh sách mapping rules (`mapping_rules_v2`).
