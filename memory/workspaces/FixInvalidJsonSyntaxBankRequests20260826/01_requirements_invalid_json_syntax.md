# 01_requirements_invalid_json_syntax.md: Yêu cầu chi tiết xử lý lỗi SQLSTATE 22P02

## 1. Mục tiêu (Goals)
- Khắc phục triệt để lỗi `ERROR: invalid input syntax for type json (SQLSTATE 22P02)` khi ingestion/upsert bản ghi từ `vmg-ekyc-connector-service.bank-requests` (và các service MongoDB khác) vào PostgreSQL Shadow Table (`shadow_vmg_ekyc.bank_requests`) và Master Table.
- Bảo đảm 100% các cột kiểu `JSON` / `JSONB` (bao gồm cột metadata `_raw_data` và các cột nghiệp vụ mapped dynamic) được ép kiểu và format thành chuỗi JSON hợp lệ theo đúng quy chuẩn PostgreSQL parser trước khi truyền bind parameter xuống SQL engine.

## 2. Phạm vi thay đổi (Scope)
- Microservice: `centralized-data-service`
- Files bị tác động:
  1. `internal/service/shadow/schema_adapter.go`: Normalize `DataType` trong `loadSchemaInSchema` và phòng thủ `_raw_data` trong `getMetadataInsertPlaceholdersAndValues`.
  2. `internal/service/shadow/schema_adapter_coerce.go`: Mở rộng `IsJSONB` nhận diện chính xác các kiểu JSON/JSONB trong DB schema, sửa `decodeBase64JSON` không trả về raw `[]byte` hỏng, và bảo đảm `CoerceValue` cho JSONB luôn return valid JSON string (kể cả với empty string, raw scalar string, map/slice, BSON extended types).
  3. `internal/service/master/transmuter_utils.go`: Đảm bảo `coerceForColumn` validate và marshal JSON hợp lệ cho Master Table upsert path.

## 3. Tiêu chuẩn chấp nhận (Definition of Done)
- [ ] Không còn lỗi `SQLSTATE 22P02` đối với các bản ghi chứa ObjectId `6a8e3fc31b3d2729eb078a60` và các bản ghi MongoDB khác.
- [ ] `go test ./...` trong `centralized-data-service` pass xanh 100%.
- [ ] Bổ sung Unit Tests kiểm chứng toàn bộ các trường hợp biên của JSON coercion (`""`, base64 invalid, unwrapped MongoDB scalar, raw string, maps/slices).
