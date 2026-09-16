# 05_progress_invalid_json_syntax.md: Audit Log Tiến độ & Phân tích Gốc rễ

[2026-08-26T09:34:00+07:00] [Agent:Gemini 3.6 Flash] Task Initiated: Fix invalid input syntax for type json (SQLSTATE 22P02) for topic vmg-ekyc-connector-service.bank-requests (record ID: 6a8e3fc31b3d2729eb078a60).

### Root Cause Analysis (Phân tích Gốc rễ):
1. **Lỗi 1 - Miss DataType Normalization trong `loadSchemaInSchema` (`schema_adapter.go`)**:
   - `loadSchemaInSchema` đọc `data_type` từ `information_schema.columns` và lưu trực tiếp `DataType: r.DataType` mà không gọi `strings.ToLower(strings.TrimSpace(r.DataType))`.
   - Hàm `IsJSONB` kiểm tra `info.DataType == "jsonb" || info.DataType == "json"`. Nếu `DataType` mang chữ hoa (`JSONB`/`JSON`) hoặc khoảng trắng, `IsJSONB` trả về `false`, khiến `CoerceValue` bỏ qua việc format JSON (`return val`).
   - Khi gorm/pgx bind Go `map[string]any` / `slice` không qua marshal vào placeholder SQL của PostgreSQL, Postgres văng `ERROR: invalid input syntax for type json (SQLSTATE 22P02)`.

2. **Lỗi 2 - `decodeBase64JSON` trả về raw `[]byte` hỏng trong `schema_adapter_coerce.go`**:
   - Trong `decodeBase64JSON`, khi decode string base64, nếu `json.Valid(decoded)` = true nhưng `json.Unmarshal` không thành công hoặc `json.Marshal` thất bại, hàm trả về `decoded` (kiểu `[]byte`).
   - Binding `[]byte` vào placeholder SQL dạng text/json làm pgx gửi bytea binary hoặc non-valid text, làm văng `SQLSTATE 22P02`.

3. **Lỗi 3 - Metadata `_raw_data` rỗng `""` không được fallback thành JSON object valid**:
   - `getMetadataInsertPlaceholdersAndValues` bind trực tiếp `rawData` (kiểu string) vào cột `_raw_data JSONB`. Nếu `rawData` bị empty string `""` hoặc invalid json text, PostgreSQL không thể parse chuỗi 0-byte thành `JSONB` và văng `SQLSTATE 22P02`.

4. **Lỗi 4 - `CoerceValue` cho cột kiểu `JSON`/`JSONB` khi nhận raw string scalar**:
   - Khi nhận một string không phải JSON valid (ví dụ: raw hex string `6a8e3fc31b3d2729eb078a60` hoặc string chưa quote), `CoerceValue` cần đảm bảo 100% kết quả trả về là một chuỗi JSON hợp lệ (ví dụ: `json.Marshal(v)`).

[2026-08-26T09:34:00+07:00] [Agent:Gemini 3.6 Flash] Analysis completed. Workspace documents created. Ready for implementation plan proposal.
