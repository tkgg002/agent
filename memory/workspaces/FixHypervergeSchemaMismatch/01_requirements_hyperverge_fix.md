# Requirements: Fix Forced "_id" -> "id" Overrides in Event Handlers

## Context & Problem
Log báo lỗi trong hệ thống CDC:
```text
unknown.hyperverge-face-match
shadow_testhecs.hyperverge_face_match
6a8d5cb4eeb3c73b3c973845    schema_mismatch    ERROR: column "id" of relation "hyperverge_face_match" does not exist (SQLSTATE 42703)
```

- **Target Table:** `shadow_testhecs.hyperverge_face_match`
- **Record ID:** `6a8d5cb4eeb3c73b3c973845` (MongoDB ObjectID standard format)
- **Error Type:** `schema_mismatch`
- **Root Cause Thực Sự:** Trường `PrimaryKeyField` trong Registry và TableConfig đã ĐÚNG là `_id`. Tuy nhiên, trong code xử lý CDC event tại `internal/handler/shadow/event_handler.go` và `internal/handler/source/bridge_handler.go`, có các dòng code ép đè cứng cưỡng chế:
  - `event_handler.go:353`: `if pgPKField == "_id" { pgPKField = "id" }`
  - `event_handler.go:384`: `if !mappedPK && pkField == "_id" { pgPKField = "id" }`
  - `bridge_handler.go:281`: `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`
  
  Chính các dòng code này đã tự ý đổi `pgPKField` từ `_id` thành `"id"`, làm cho `record.PrimaryKeyField` bị sai. Khi `BatchBuffer` sinh câu SQL `INSERT INTO shadow_testhecs.hyperverge_face_match ("id", ...) VALUES (...)`, PostgreSQL văng lỗi `SQLSTATE 42703` vì bảng shadow chứa cột `_id` chứ không có cột `"id"`.

## Detailed Requirements
1. **Loại bỏ 100% các câu lệnh ép đè cứng `_id → id`:**
   - Xóa bỏ `if pgPKField == "_id" { pgPKField = "id" }` (dòng 353-355 trong `event_handler.go`).
   - Xóa bỏ `if !mappedPK && pkField == "_id" { pgPKField = "id" }` (dòng 384-386 trong `event_handler.go`).
   - Xóa bỏ `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }` (dòng 281-283 trong `bridge_handler.go`).
2. **Bảo toàn `PrimaryKeyField` của MongoDB:**
   - Khi `PrimaryKeyField` là `_id`, giữ nguyên 100% `_id`.
   - Giúp câu lệnh SQL sinh ra đúng định dạng `INSERT INTO shadow_testhecs.hyperverge_face_match ("_id", ...) VALUES (...) ON CONFLICT ("_id") DO UPDATE ...`.
