# Báo cáo Thay đổi Mã nguồn (Change Report)

## Tóm tắt công việc
Đã sửa triệt để nguyên nhân gốc rễ làm câu lệnh CDC SQL Upsert sinh ra cột `"id"` thay vì `"_id"` cho bảng shadow `shadow_testhecs.hyperverge_face_match` (và các MongoDB collections khác).

## Các file đã thay đổi (Files Modified)
1. `internal/handler/shadow/event_handler.go`:
   - Xóa bỏ 2 khối code ép đè cứng cưỡng chế `if pgPKField == "_id" { pgPKField = "id" }` (dòng 353-355) và `if !mappedPK && pkField == "_id" { pgPKField = "id" }` (dòng 384-386).
2. `internal/handler/source/bridge_handler.go`:
   - Thay thế `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }` bằng `if resolved.pgPKField == "" { resolved.pgPKField = "_id" }`.

## Thống kê dòng code
- `internal/handler/shadow/event_handler.go`: Giảm 6 dòng code.
- `internal/handler/source/bridge_handler.go`: Sửa 1 dòng condition.

## Kết quả
Hệ thống giữ nguyên 100% `PrimaryKeyField = "_id"` cho các bảng MongoDB. Câu lệnh SQL sinh ra là `INSERT INTO shadow_testhecs.hyperverge_face_match ("_id", ...) VALUES (...) ON CONFLICT ("_id") DO UPDATE ...`, khớp hoàn toàn với schema bảng PostgreSQL.
