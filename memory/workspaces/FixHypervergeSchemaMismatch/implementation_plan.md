# Fix Schema Mismatch: Remove Forced "_id" -> "id" Overrides in Event Handlers

Phân tích nguyên nhân gốc rễ chuẩn xác 100% theo chỉ đạo của User đối với lỗi CDC sync `schema_mismatch` bảng `shadow_testhecs.hyperverge_face_match` (Record ID: `6a8d5cb4eeb3c73b3c973845`).

## User Review Required

> [!IMPORTANT]
> - **Phát hiện Nguyên nhân Gốc rễ Thực sự:** Cấu hình `PrimaryKeyField` trong Registry và TableConfig đã **ĐÚNG LÀ `_id`**. Tuy nhiên, tại các handler xử lý sự kiện CDC (`event_handler.go` và `bridge_handler.go`), tồn tại các dòng code ép đè cứng cưỡng chế:
>   - `event_handler.go:353`: `if pgPKField == "_id" { pgPKField = "id" }`
>   - `event_handler.go:384`: `if !mappedPK && pkField == "_id" { pgPKField = "id" }`
>   - `bridge_handler.go:281`: `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`
> - Chính các câu lệnh này đã tự ý đổi `pgPKField` từ `_id` thành `"id"`, làm cho `record.PrimaryKeyField` bị đổi thành `"id"`. Khi `BatchBuffer` sinh SQL `INSERT INTO shadow_testhecs.hyperverge_face_match ("id", ...)`, PostgreSQL văng lỗi `SQLSTATE 42703` do cột `"id"` không tồn tại trong bảng (bảng thực tế chứa cột `_id`).

## Proposed Changes

### `centralized-data-service`

#### [MODIFY] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
- Xóa dòng 353-355: `if pgPKField == "_id" { pgPKField = "id" }`.
- Xóa dòng 384-386: `if !mappedPK && pkField == "_id" { pgPKField = "id" }`.

#### [MODIFY] [bridge_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/bridge_handler.go)
- Xóa dòng 281-283: `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`.

## Verification Plan

### Automated Tests
- Chạy unit test suite `go test -v ./internal/handler/shadow/...` và `go test -v ./internal/handler/source/...` trong `centralized-data-service`.
- Chạy linter quy trình governance: `python3 agent/tooling/verify_governance.py`.
