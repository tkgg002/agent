# 09_tasks_solution_invalid_json_syntax.md: Hồ sơ Giải pháp Kỹ thuật Chi tiết

## 1. Tổng quan Giải pháp (Single Best Approach)
Khắc phục tận gốc lỗi `SQLSTATE 22P02 invalid input syntax for type json` bằng cách nâng cấp cơ chế phòng thủ 4 lớp (4-Layer Defense Guard) trong `centralized-data-service`:

### Lớp 1: Chuẩn hóa Schema Metadata (`schema_adapter.go`)
Trong `loadSchemaInSchema`:
```go
	for _, r := range rows {
		schema.Columns[r.ColumnName] = ColumnInfo{
			Name:       r.ColumnName,
			DataType:   strings.ToLower(strings.TrimSpace(r.DataType)),
			IsNullable: r.IsNullable == "YES",
		}
	}
```
Giúp bảo đảm `DataType` luôn ở dạng chữ thường không có khoảng trắng thừa, ngăn ngừa việc bỏ sót nhận diện cột kiểu `json`/`jsonb`.

### Lớp 2: Nâng cấp `IsJSONB` & `CoerceValue` (`schema_adapter_coerce.go`)
- **Mở rộng `IsJSONB`**:
```go
func (sa *SchemaAdapter) IsJSONB(schema *TableSchema, colName string) bool {
	if info, ok := schema.Columns[colName]; ok {
		dt := strings.ToLower(strings.TrimSpace(info.DataType))
		return dt == "jsonb" || dt == "json" || strings.HasPrefix(dt, "json")
	}
	return false
}
```
- **Sửa `decodeBase64JSON`**: Không bao giờ trả về raw `[]byte` hỏng nếu `json.Unmarshal` không parse thành công. Trả về `nil` để `CoerceValue` xử lý qua `json.Marshal`.
- **Nâng cấp `CoerceValue` cho kiểu `JSON`/`JSONB`**:
  - Khi `val` là Go `map[string]interface{}` hoặc `[]interface{}` hoặc composite: marshal qua `normalizeMongoExtendedJSON`.
  - Khi `val` là `string`:
    - Nếu decodeBase64JSON ra valid JSON string -> return JSON string.
    - Nếu string là valid JSON text (e.g. `{"a":1}`) -> parse + normalize + marshal -> return JSON string.
    - Nếu string là chuỗi rỗng `""` hoặc string bình thường (e.g. `"6a8e3fc31b3d2729eb078a60"`) -> dùng `json.Marshal(v)` để tạo thành chuỗi JSON string hợp lệ (`"\"6a8e3fc31b3d2729eb078a60\""` hoặc `"\"\""`).
  - Với bất kỳ type nào khác: `jsonVal, _ := json.Marshal(v)` và return `string(jsonVal)`.
  - **Kết quả**: 100% giá trị trả về cho cột JSON/JSONB luôn luôn là một string đại diện cho JSON hợp lệ, PostgreSQL parser sẽ chấp nhận 100% mà không bao giờ văng 22P02.

### Lớp 3: Phòng thủ cột Metadata `_raw_data` (`schema_adapter.go`)
Trong `getMetadataInsertPlaceholdersAndValues`:
```go
	if _, ok := schema.Columns["_raw_data"]; ok {
		placeholders = append(placeholders, "?")
		rawStr := strings.TrimSpace(rawData)
		if rawStr == "" || !json.Valid([]byte(rawStr)) {
			rawStr = "{}"
		}
		vals = append(vals, rawStr)
	}
```
Đảm bảo cột `_raw_data` never nhận chuỗi 0-byte rỗng `""` hay string không valid JSON.

### Lớp 4: An toàn cho Master Transmute Path (`transmuter_utils.go`)
Trong `coerceForColumn`:
Gia cố nhánh `isJSONColumnType(dataType) || composite`: Nếu `v` là `string` mà không phải JSON valid, marshal nó thành valid JSON string thay vì trả về raw string.
