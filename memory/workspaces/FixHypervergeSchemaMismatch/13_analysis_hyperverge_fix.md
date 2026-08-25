# Technical Analysis: Hardcoded "_id" -> "id" Overrides

## 1. Trace Root Cause Details
- Bảng shadow: `shadow_testhecs.hyperverge_face_match` (nguồn MongoDB `hyperverge-face-match`).
- `PrimaryKeyField` được cấu hình/lưu trong Registry của `hyperverge-face-match` đã ĐÚNG LÀ `_id`.
- Tuy nhiên, khi Debezium event hoặc Bridge Oplog event được xử lý:
  - Trong `event_handler.go`:
    - Dòng 353-355: `if pgPKField == "_id" { pgPKField = "id" }`
    - Dòng 384-386: `if !mappedPK && pkField == "_id" { pgPKField = "id" }`
  - Trong `bridge_handler.go`:
    - Dòng 281-283: `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`
- Các dòng code ép đè cứng này khiến cho giá trị `_id` vốn đúng bị đổi trái phép thành `"id"`.
- `BatchBuffer` nhận `record.PrimaryKeyField = "id"`, sinh ra câu lệnh SQL `INSERT INTO shadow_testhecs.hyperverge_face_match ("id", ...)` -> PostgreSQL văng lỗi `SQLSTATE 42703 column "id" of relation "hyperverge_face_match" does not exist`.

## 2. Remediation Strategy
- Xóa bỏ 100% các câu lệnh `if ... == "_id" { ... = "id" }` trong `event_handler.go` và `bridge_handler.go`.
