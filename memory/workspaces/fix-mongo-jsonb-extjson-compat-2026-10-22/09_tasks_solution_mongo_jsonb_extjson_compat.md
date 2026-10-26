# 09 Technical Solution: Giải pháp Chuẩn hóa & Bóc tách Đệ quy MongoDB Extended JSON trong Cột `JSONB` cho các Service Cũ (`mongo_jsonb_extjson_compat`)

## 1. Phân tích Nguyên nhân Gốc rễ (`S2 Investigation — Verified with file:line`)

### 1.1. Lịch sử tiến hóa định dạng JSONB của MongoDB qua Shadow
1. **Giai đoạn trước commit `1473d0f` (Relaxed Extended JSON):**
   - Snapshot:v2 (`internal/handler/orchestration/snapshot_runner_handler.go:674`) và Heal Fetch (`internal/handler/recon/recon_heal_fetch.go:76`) dùng `bson.MarshalExtJSON(doc, false, false)` (`canonical = false`), còn Debezium MongoDB Connector phát chuỗi Relaxed Extended JSON.
   - Ở chế độ Relaxed:
     - `_id` được biểu diễn là `{"$oid": "67230efb7ed87a55a4ce8393"}`.
     - `date` được biểu diễn là `{"$date": "2024-10-31T05:00:43.935Z"}` (hoặc epoch ms).
     - Các trường số (`int32`, `int64`, `double`) là số JSON phẳng `{"number": 123123123}` (không có wrapper `$numberInt`, `$numberLong`, `$numberDouble`).
2. **Giai đoạn từ commit `1473d0f` (Canonical Extended JSON v2):**
   - Tại `internal/handler/shadow/kafka_consumer.go:603-618, 921-935` và `internal/sinkworker/utils.go:48-86`, hàm `toCanonicalMongoExtJSON` chuyển toàn bộ payload MongoDB sang **Canonical Extended JSON (v2)** thông qua `bson.MarshalExtJSON(doc, true, false)` (`canonical = true`).
   - Khi đó, trong `_raw_data` (kiểu `JSONB`) tại tầng Shadow:
     - `_id` -> `{"$oid": "67230efb7ed87a55a4ce8393"}`
     - `date` -> `{"$date": {"$numberLong": "1730350843935"}}` (thay vì chuỗi ISO `"2024-10-31T05:00:43.935Z"`)
     - `int32` / `int64` / `double` / `decimal128` -> bị bọc thành object `{"$numberInt": "123123123"}`, `{"$numberLong": "123123123"}`, `{"$numberDouble": "123.45"}`, `{"$numberDecimal": "123.45"}`
     - Các kiểu BSON khác -> `$timestamp`, `$binary`, `$regularExpression`, `$symbol`...

### 1.2. Điểm nghẽn khi đồng bộ từ Shadow sang cột `JSONB` cho các Service Cũ
1. **Master Transmuter bỏ qua bóc tách đối với cột `JSONB` (`internal/service/master/transmuter.go:1914-1917`):**
   ```go
   val := gjsonValueToGo(gres)
   if !isJSONColumnType(r.DataType) {
       val = unwrapMongoExtJSON(val)
   }
   ```
   - Khi `r.DataType` là `JSONB` hoặc `JSON`, `isJSONColumnType(r.DataType)` (`internal/service/master/transmuter_utils.go:101-104`) trả về `true`, nên `unwrapMongoExtJSON` bị bỏ qua hoàn toàn.
   - Hơn nữa, bản thân `unwrapMongoExtJSON` (`internal/service/master/transmuter_utils.go:45-85`) chỉ bóc object đơn trị ở tầng ngoài cùng (`len(m) == 1`), hoàn toàn **không duyệt đệ quy** vào các field con bên trong một sub-document hoặc mảng JSON (`map[string]any`, `[]any`).
2. **Thiếu bộ `transform_fn` đệ quy cho `JSONB` trong `transformRegistry` (`internal/service/master/transform_registry.go:31-39`):**
   - Hiện tại `transformRegistry` chỉ có `jsonb_passthrough` (`internal/service/master/transform_registry.go:213-222`) thực hiện `json.Marshal(raw)` nguyên xi.
   - Trong khi đó, các **service cũ** đọc cột `JSONB` từ Master (hoặc qua view/query) lại có các hàm parse cố định theo các định dạng lịch sử:
     - **Nhóm A (Service cũ viết func đọc số dạng chuỗi `number: "123123123"` hoặc bóc phẳng toàn bộ wrapper):** Kỳ vọng field không có wrapper `$numberInt`/`$numberLong`/`$numberDouble` mà trả trực tiếp chuỗi `"123123123"` (như cách `unwrapMongoTypes` tại `internal/service/shadow/dynamic_mapper.go:346-360` trả `fmt.Sprintf("%v", num)`), `_id` là `"67230efb7ed87a55a4ce8393"`, `date` là `"2024-10-31T05:00:43.935Z"`.
     - **Nhóm B (Service cũ viết func đọc JSON phẳng chuẩn `{"number": 123123123}`):** Kỳ vọng số là JSON number `123123123` (không có wrapper `$number*`), `_id` là `"67230efb7ed87a55a4ce8393"`, `date` là `"2024-10-31T05:00:43.935Z"`.
     - **Nhóm C (Service cũ viết func theo Relaxed Extended JSON trước commit `1473d0f`):** Kỳ vọng giữ `{"$oid": "67230efb7ed87a55a4ce8393"}` và `{"$date": "2024-10-31T05:00:43.935Z"}` (ISO string), nhưng trường số phải là `{"number": 123123123}` (hoặc `"number": "123123123"`) không có wrapper `$numberInt`/`$numberLong`/`$numberDouble`.

---

## 2. Phương Án Kỹ Thuật Tối Ưu Nhất (`S3 Single Best Solution`)

### Nguyên tắc kiến trúc (Tuân thủ Rule #12 & Lesson L079)
- **KHÔNG sửa đổi tầng Shadow Ingestion (`kafka_consumer.go:926`, `sinkworker/utils.go:77`):** Giữ nguyên 100% Canonical Extended JSON v2 trong `_raw_data` của Shadow để bảo toàn tính nhất quán cho Recon Engine, Snapshot:v2 và Schema Scanner (`IsCanonicalMongoExtJSON`).
- **Giải quyết tại tầng Master Transmuter & Transform Registry (`internal/service/master/transmuter_utils.go` & `internal/service/master/transform_registry.go`):**
  - Cung cấp bộ chuẩn hóa đệ quy `NormalizeMongoExtJSONDeep(v any, mode ExtJSONCompatMode) any` xử lý mọi cấp lồng nhau (`map[string]any`, `[]any`, hoặc JSON string) của cột `JSONB`.
  - Đăng ký vào `transformRegistry` (`internal/service/master/transform_registry.go:31-39`) các hàm `transform_fn` chuyên biệt cho cột `JSONB` để bất kỳ bảng/cột nào phục vụ service cũ chỉ cần gắn `transform_fn` tương ứng trên rule mapping (hoặc bật theo cấu hình cột) là nhận đúng 100% định dạng JSONB mà service cũ mong đợi:
    1. `extjson_unwrap_plain`: Bóc đệ quy toàn bộ wrapper BSON sang Plain JSON (`$oid` -> hex string, `$date` -> ISO-8601 `"2024-10-31T05:00:43.935Z"`, `$numberInt`/`$numberLong`/`$numberDouble` -> JSON number `123123123`).
    2. `extjson_unwrap_num_str`: Bóc đệ quy toàn bộ wrapper BSON và xuất mọi trường số (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, hoặc số nguyên/thực) thành **JSON string** (`"number": "123123123"`) đúng theo hàm đọc của các service cũ.
    3. `extjson_relaxed`: Giữ `{"$oid": "..."}` và chuẩn hóa `{"$date": "2024-10-31T05:00:43.935Z"}`, nhưng bóc bỏ `$numberInt`/`$numberLong`/`$numberDouble` về số phẳng `{"number": 123123123}` (tương thích 100% với bản trước commit `1473d0f`).
    4. `extjson_relaxed_num_str`: Giữ `{"$oid": "..."}` và `{"$date": "2024-10-31T05:00:43.935Z"}`, nhưng bóc `$numberInt`/`$numberLong`/`$numberDouble`/`$numberDecimal` về chuỗi `"number": "123123123"`.

---

## 3. Code Demo Chi Tiết (`Before vs After`)

### 3.1. File 1: `centralized-data-service/internal/service/master/transmuter_utils.go`
Bổ sung `ExtJSONCompatMode` và hàm đệ quy `NormalizeMongoExtJSONDeep` ngay sau `unwrapMongoExtJSON` (`transmuter_utils.go:85`):

```go
// ExtJSONCompatMode controls how MongoDB Extended JSON wrappers inside a JSONB
// tree are normalized for downstream consumers and legacy services.
type ExtJSONCompatMode int

const (
	// ExtJSONModeUnwrapPlain unwraps $oid -> string, $date -> RFC3339 string,
	// and $numberInt/$numberLong/$numberDouble -> native JSON number (123123123).
	ExtJSONModeUnwrapPlain ExtJSONCompatMode = iota
	// ExtJSONModeUnwrapNumStr unwraps $oid -> string, $date -> RFC3339 string,
	// and converts numbers ($numberInt/$numberLong/$numberDouble/$numberDecimal
	// as well as numeric scalars) to JSON strings ("123123123").
	ExtJSONModeUnwrapNumStr
	// ExtJSONModeRelaxed preserves {"$oid":"..."} and normalizes {"$date":"<ISO-8601>"},
	// while unwrapping $numberInt/$numberLong/$numberDouble to native JSON numbers.
	ExtJSONModeRelaxed
	// ExtJSONModeRelaxedNumStr preserves {"$oid":"..."} and {"$date":"<ISO-8601>"},
	// while unwrapping $numberInt/$numberLong/$numberDouble/$numberDecimal to strings.
	ExtJSONModeRelaxedNumStr
)

// NormalizeMongoExtJSONDeep recursively traverses maps and slices (or parses a
// JSON string first) and normalizes MongoDB Extended JSON wrappers according to mode.
func NormalizeMongoExtJSONDeep(v any, mode ExtJSONCompatMode) any {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		trimmed := strings.TrimSpace(s)
		if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed)) {
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return NormalizeMongoExtJSONDeep(parsed, mode)
			}
		}
		return s
	}
	if arr, ok := v.([]any); ok {
		out := make([]any, len(arr))
		for i, elem := range arr {
			out[i] = NormalizeMongoExtJSONDeep(elem, mode)
		}
		return out
	}
	m, ok := v.(map[string]any)
	if !ok {
		if mode == ExtJSONModeUnwrapNumStr || mode == ExtJSONModeRelaxedNumStr {
			switch n := v.(type) {
			case int64:
				return strconv.FormatInt(n, 10)
			case int:
				return strconv.Itoa(n)
			case float64:
				if n == float64(int64(n)) {
					return strconv.FormatInt(int64(n), 10)
				}
				return strconv.FormatFloat(n, 'f', -1, 64)
			}
		}
		return v
	}

	keepOIDAndDateWrapped := mode == ExtJSONModeRelaxed || mode == ExtJSONModeRelaxedNumStr
	numAsString := mode == ExtJSONModeUnwrapNumStr || mode == ExtJSONModeRelaxedNumStr

	if len(m) == 1 {
		if oid, ok := m["$oid"].(string); ok {
			if keepOIDAndDateWrapped {
				return map[string]any{"$oid": oid}
			}
			return oid
		}
		if d, has := m["$date"]; has {
			iso := formatMongoDateISO(d)
			if keepOIDAndDateWrapped {
				return map[string]any{"$date": iso}
			}
			return iso
		}
		if dec, ok := m["$numberDecimal"].(string); ok {
			return dec
		}
		if n, has := m["$numberLong"]; has {
			if numAsString {
				return fmt.Sprintf("%v", n)
			}
			return mongoNumberToGo(n)
		}
		if n, has := m["$numberInt"]; has {
			if numAsString {
				return fmt.Sprintf("%v", n)
			}
			return mongoNumberToGo(n)
		}
		if n, has := m["$numberDouble"]; has {
			if numAsString {
				return fmt.Sprintf("%v", n)
			}
			if s, ok := n.(string); ok {
				if f, err := strconv.ParseFloat(s, 64); err == nil {
					return f
				}
			}
			return n
		}
	}

	out := make(map[string]any, len(m))
	for k, val := range m {
		out[k] = NormalizeMongoExtJSONDeep(val, mode)
	}
	return out
}

func formatMongoDateISO(d any) string {
	switch dv := d.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339Nano, dv); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05.000Z")
		}
		if ms, err := strconv.ParseInt(dv, 10, 64); err == nil {
			return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
		}
		return dv
	case float64:
		return time.UnixMilli(int64(dv)).UTC().Format("2006-01-02T15:04:05.000Z")
	case int64:
		return time.UnixMilli(dv).UTC().Format("2006-01-02T15:04:05.000Z")
	case map[string]any:
		if nl, ok := dv["$numberLong"].(string); ok {
			if ms, err := strconv.ParseInt(nl, 10, 64); err == nil {
				return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
			}
		}
	}
	return fmt.Sprintf("%v", d)
}
```
Đồng thời trong `coerceForColumn(v any, dataType string)` (`transmuter_utils.go:293-304`), khi `isJSONColumnType(dataType)` là `true` và giá trị sau `ApplyTransform` đã là chuỗi JSON object/array hợp lệ (`(strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed))`), trả về nguyên chuỗi `trimmed` thay vì gọi `json.Marshal(v)` lần thứ hai (tránh lỗi double-encoding chuỗi JSONB thành JSON string literal).

### 3.2. File 2: `centralized-data-service/internal/service/master/transform_registry.go`
Đăng ký 4 hàm transform cho cột `JSONB` vào `transformRegistry` (`transform_registry.go:31-39`):

```go
var transformRegistry = map[string]TransformFunc{
	"mongo_date_ms":          transformMongoDateMs,
	"oid_to_hex":             transformOIDToHex,
	"bigint_str":             transformBigIntStr,
	"numeric_cast":           transformNumericCast,
	"lowercase":              transformLowercase,
	"jsonb_passthrough":      transformJSONBPassthrough,
	"null_if_empty":          transformNullIfEmpty,
	"extjson_unwrap_plain":    makeExtJSONTransform(ExtJSONModeUnwrapPlain),
	"extjson_unwrap_num_str": makeExtJSONTransform(ExtJSONModeUnwrapNumStr),
	"extjson_relaxed":        makeExtJSONTransform(ExtJSONModeRelaxed),
	"extjson_relaxed_num_str": makeExtJSONTransform(ExtJSONModeRelaxedNumStr),
}

func makeExtJSONTransform(mode ExtJSONCompatMode) TransformFunc {
	return func(raw any) (any, error) {
		if raw == nil {
			return nil, nil
		}
		normalized := NormalizeMongoExtJSONDeep(raw, mode)
		b, err := json.Marshal(normalized)
		if err != nil {
			return nil, fmt.Errorf("extjson transform marshal: %w", err)
		}
		return string(b), nil
	}
}
```

### 3.3. File 3: `centralized-data-service/internal/service/master/transmuter.go` & `master_ddl_generator.go`
Tại `transmuter.go:998` và `master_ddl_generator.go:116`, cập nhật câu lệnh `SELECT` để ưu tiên lấy `m.transform_fn` được cấu hình trực tiếp trên từng field của Master (`mapping_rule_master`), fallback về `v2.transform_fn` của Shadow (`mapping_rule_v2`):

```sql
-- Trước:
v2.transform_fn, COALESCE(v2.is_nullable, true) AS is_nullable, v2.default_value
-- Sau:
COALESCE(m.transform_fn, v2.transform_fn) AS transform_fn, COALESCE(v2.is_nullable, true) AS is_nullable, v2.default_value
```

### 3.4. File 4 (Repo `cdc-cms-service`): Migration & Repository + API cập nhật `transform_fn` cho Master Rule
1. **Migration `cdc-cms-service/migrations/schema/cdc_system_model/081_add_transform_fn_to_mapping_rule_master.sql`:**
   - Do migration `075_redesign_mapping_rule_master.sql:9-23` đã `DROP TABLE` và tạo lại `cdc_system.mapping_rule_master` mà thiếu cột `transform_fn` (vốn có ở `074_v2_mapping_rule_master.sql:13`), cần bổ sung lại:
   ```sql
   -- +goose Up
   ALTER TABLE cdc_system.mapping_rule_master
       ADD COLUMN IF NOT EXISTS transform_fn VARCHAR(100);

   -- +goose Down
   ALTER TABLE cdc_system.mapping_rule_master
       DROP COLUMN IF EXISTS transform_fn;
   ```
2. **Repository `cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:87-102`:**
   - Sửa `masterRuleSelectJoin` dòng 100 từ `v2.transform_fn` thành `COALESCE(m.transform_fn, v2.transform_fn) AS transform_fn`.
3. **Command & Handler (`cdc-cms-service/internal/app/commands/master/update_column_metadata.go:11-72` & `internal/api/master/master_mapping_rule_handler.go:163-186`):**
   - Mở rộng `UpdateColumnMetadataCommand` thêm `TransformFn *string` và cho phép truyền `transform_fn` trong `PUT /api/v1/master-mapping-rules/:id/data-type` (hoặc khi đổi kiểu dữ liệu / chọn hàm transform trên UI):
   ```go
   type UpdateColumnMetadataCommand struct {
   	ID           int64
   	TargetColumn *string
   	DataType     *string
   	TransformFn  *string
   }
   ```
   ```go
   // Case 3: Cập nhật hàm chuyển đổi (TransformFn)
   if cmd.TransformFn != nil {
   	fn := strings.TrimSpace(*cmd.TransformFn)
   	if fn == "" {
   		updates["transform_fn"] = nil
   	} else {
   		updates["transform_fn"] = fn
   	}
   	result["transform_fn"] = fn
   }
   ```

### 3.5. File 5 (Repo `cdc-cms-web`): Chọn `transform_fn` trên mỗi Field khi `Master Data Type` là `JSONB` / `JSON` (`MasterMappingFieldsPage.tsx:768-826`)
Tại `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:768-826` (trang `http://localhost:5173/masters/events/mappings?binding_id=59`), trong cột **`Master Data Type`**, ngay dưới/cạnh `<Select>` chọn kiểu dữ liệu, khi `baseType === 'JSONB' || baseType === 'JSON'`, hiển thị `<Select>` chọn `transform_fn` cho field đó:

```tsx
const JSONB_TRANSFORM_OPTIONS = [
  { label: 'Mặc định (Canonical ExtJSON từ Shadow)', value: '' },
  { label: 'extjson_relaxed — Giữ $oid/$date, số phẳng {"n": 123}', value: 'extjson_relaxed' },
  { label: 'extjson_relaxed_num_str — Giữ $oid/$date, số chuỗi {"n": "123"}', value: 'extjson_relaxed_num_str' },
  { label: 'extjson_unwrap_plain — Bóc hết $oid/$date/$number*, số phẳng {"n": 123}', value: 'extjson_unwrap_plain' },
  { label: 'extjson_unwrap_num_str — Bóc hết $oid/$date/$number*, số chuỗi {"n": "123"}', value: 'extjson_unwrap_num_str' },
];
```
Và trong `render` của cột `Master Data Type` (`MasterMappingFieldsPage.tsx:794-825`):
```tsx
        const handleTransformChange = async (newTransformFn: string) => {
          try {
            await cmsApi.put(`/api/v1/master-mapping-rules/${record.id}/data-type`, {
              data_type: t || baseType || 'JSONB',
              transform_fn: newTransformFn,
            });
            message.success(
              newTransformFn
                ? `Đã gắn transform_fn "${newTransformFn}" cho cột ${record.target_column}`
                : `Đã về mặc định (Canonical ExtJSON) cho cột ${record.target_column}`
            );
            fetchRules();
          } catch (err) {
            message.error(humanizeApiError(err, 'Cập nhật transform_fn thất bại'));
          }
        };

        return (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Space>
              <Select
                size="small"
                value={baseType || 'TEXT'}
                style={{ width: 180 }}
                onChange={(val) => {
                  if (val === 'VARCHAR' || val === 'CHAR') {
                    handleTypeChange(val, length || 255);
                  } else {
                    handleTypeChange(val);
                  }
                }}
                options={types.map(type => ({ label: type, value: type }))}
              />
              {(baseType === 'VARCHAR' || baseType === 'CHAR') && (
                <InputNumber
                  size="small"
                  style={{ width: 60 }}
                  min={1}
                  max={65535}
                  value={length || 255}
                  onChange={(val) => {
                    if (val) {
                      handleTypeChange(baseType, val);
                    }
                  }}
                />
              )}
            </Space>
            {(baseType === 'JSONB' || baseType === 'JSON') && (
              <Select
                size="small"
                value={record.transform_fn || ''}
                style={{ width: 260 }}
                placeholder="Chọn chuẩn JSONB (transform_fn)"
                onChange={(val) => handleTransformChange(val)}
                options={JSONB_TRANSFORM_OPTIONS}
              />
            )}
          </Space>
        );
```

---

## 4. Bảng Đối Chiếu Kết Quả Biến Đổi JSONB Theo Từng Chế Độ

Giả sử dữ liệu gốc trong Shadow `_raw_data` (Canonical Extended JSON v2):
```json
{
  "_id": {"$oid": "67230efb7ed87a55a4ce8393"},
  "createdAt": {"$date": {"$numberLong": "1730350843935"}},
  "number": {"$numberLong": "123123123"},
  "rate": {"$numberDouble": "12.5"}
}
```

| Chế độ (`transform_fn`) | Kết quả lưu vào cột `JSONB` ở Master | Phù hợp với Service nào |
| :--- | :--- | :--- |
| *(Không gắn `transform_fn`)* | `{"_id":{"$oid":"67230efb7ed87a55a4ce8393"},"createdAt":{"$date":{"$numberLong":"1730350843935"}},"number":{"$numberLong":"123123123"},"rate":{"$numberDouble":"12.5"}}` | Service mới đọc chuẩn Canonical Extended JSON v2 |
| `extjson_relaxed` | `{"_id":{"$oid":"67230efb7ed87a55a4ce8393"},"createdAt":{"$date":"2024-10-31T05:00:43.935Z"},"number":123123123,"rate":12.5}` | Service quen với bản Relaxed ExtJSON trước commit `1473d0f` (`{"number": 123123123}`) |
| `extjson_relaxed_num_str` | `{"_id":{"$oid":"67230efb7ed87a55a4ce8393"},"createdAt":{"$date":"2024-10-31T05:00:43.935Z"},"number":"123123123","rate":"12.5"}` | Service cũ giữ `$oid`/`$date` nhưng viết func đọc `"number": "123123123"` |
| `extjson_unwrap_plain` | `{"_id":"67230efb7ed87a55a4ce8393","createdAt":"2024-10-31T05:00:43.935Z","number":123123123,"rate":12.5}` | Service đọc Plain JSON chuẩn với số nguyên bản `{"number": 123123123}` |
| `extjson_unwrap_num_str` | `{"_id":"67230efb7ed87a55a4ce8393","createdAt":"2024-10-31T05:00:43.935Z","number":"123123123","rate":"12.5"}` | Service cũ đọc Plain JSON và viết func theo `"number": "123123123"` |

---

## 5. Decision Gate Claims (`DG1 + DG2` Verification)

### 5.1. Solution & Demo Claims (`Real Claims`)
- **S-RC-01:** Tại `kafka_consumer.go` dòng 921-935, hàm `toCanonicalMongoExtJSON` sử dụng `bson.UnmarshalExtJSON([]byte(jsonStr), false, &doc)` và `bson.MarshalExtJSON(doc, true, false)` để chuyển chuỗi JSON của MongoDB sang chuẩn Canonical Extended JSON (v2), trong đó `int32`/`int64`/`double` biến thành `{"$numberInt": "..."}`, `{"$numberLong": "..."}`, `{"$numberDouble": "..."}` và `date` biến thành `{"$date": {"$numberLong": "..."}}`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go:921-935`
- **S-RC-02:** Tại `transmuter.go` dòng 1914-1920, hàm `extractColumns` kiểm tra `if !isJSONColumnType(r.DataType)` trước khi gọi `unwrapMongoExtJSON(val)`, sau đó mới gọi `ApplyTransform(*r.TransformFn, val)` nếu `r.TransformFn` được cấu hình.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1914-1920`
- **S-FIX-01:** Tại `transmuter_utils.go` dòng 45-85, hàm `unwrapMongoExtJSON(v any)` chỉ xử lý `map[string]any` có `len(m) == 1` ở tầng ngoài cùng cho các cột scalar mà chưa có hàm đệ quy duyệt sâu vào các phần tử con của `map[string]any` hoặc `[]any` cho cột `JSONB`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:45-85`
- **S-IMPACT-01:** Tại `transform_registry.go` dòng 31-39, `transformRegistry` hiện chỉ đăng ký 7 hàm (`mongo_date_ms`, `oid_to_hex`, `bigint_str`, `numeric_cast`, `lowercase`, `jsonb_passthrough`, `null_if_empty`), trong đó `transformJSONBPassthrough` (dòng 213-222) chỉ gọi `json.Marshal(raw)` mà không bóc tách các wrapper Extended JSON bên trong cấu trúc `JSONB`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transform_registry.go:31-39`
- **S-PATTERN-01:** Tại `master_mapping_rule_repo_gorm.go` dòng 87-102, hằng số `masterRuleSelectJoin` hiện đang lấy `v2.transform_fn` trực tiếp từ bảng `cdc_system.mapping_rule_v2` thay vì `COALESCE(m.transform_fn, v2.transform_fn)`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:87-102`
- **D-BEHAV-01:** Tại `transmuter_utils.go` dòng 129-136, hàm `coerceForColumn(v any, dataType string)` kiểm tra `if isJSONColumnType(dataType) || composite` để marshal giá trị `map[string]any` hoặc `[]any` thành chuỗi JSON hợp lệ trước khi upsert vào Master DB.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:129-136`
- **D-EDGE-01:** Tại `MasterMappingFieldsPage.tsx` dòng 768-826, cột `Master Data Type` hiện chỉ hiển thị `<Select>` chọn kiểu dữ liệu từ `POSTGRES_DATA_TYPES` và `<InputNumber>` khi `baseType === 'VARCHAR' || baseType === 'CHAR'`, chưa có `<Select>` chọn `transform_fn` khi `baseType === 'JSONB' || baseType === 'JSON'`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:768-826`
- **D-API-01:** Tại `transform_registry.go` dòng 43-62, `ApplyTransform` và `IsTransformWhitelisted` dùng trực tiếp map `transformRegistry` làm Single Source of Truth để xác thực và thực thi `transform_fn`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transform_registry.go:43-62`

### 5.2. Control Claims (`Calibration Ground Truth`)
- **CTRL-F1 (`control_false`):** Tại `kafka_consumer.go` dòng 921-935, hàm `toCanonicalMongoExtJSON` gọi `bson.MarshalExtJSON(doc, false, true)` với `canonical = false` để xóa bỏ toàn bộ wrapper `$oid` và `$numberLong`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go:921-935`
- **CTRL-F2 (`control_false`):** Tại `transmuter.go` dòng 1914-1920, hàm `extractColumns` luôn gọi `unwrapMongoExtJSON(val)` chỉ khi `isJSONColumnType(r.DataType)` bằng `true` và bỏ qua tất cả các cột scalar.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1914-1920`
- **CTRL-T1 (`control_true`):** Tại `avro_helper.go` dòng 67-82, hàm `unwrapAvroUnion` có nhánh kiểm tra `if strings.HasPrefix(k, "$") { return v }` để không bóc nhầm các wrapper MongoDB BSON Extended JSON như `$oid`, `$date`, `$numberLong`.
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/avro_helper.go:67-82`

---

## 6. Kế Hoạch Cô Lập Trạng Thái Git (`Restore-Point` — Rule #19) Trước Khi Chạy `S4`
- Hiện tại trên nhánh `recon-heal` của cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`), có 16 file đang sửa (`M`) và 7 file mới (`??`) của task **"Tích hợp Master MongoDB Destination"**, trong đó giao nhau tại 2 file `centralized-data-service/internal/service/master/transmuter.go` và `master_ddl_generator.go`.
- Ngay khi anh phản hồi `APPROVE` (kèm chọn **Cách A: WIP commit checkpoint local** hoặc **Cách B: `git stash push -u`**), bước đầu tiên của `S4` sẽ cô lập trạng thái Git của 3 repo để bảo toàn 100% phần việc "Master MongoDB Destination" và đảm bảo `git diff` của task `mongo_jsonb_extjson_compat` sạch tuyệt đối.

---

## 7. Bổ Sung & Khắc Phục Sau Phản Biện `S5` (`S5 -> S3 -> S4 -> S5` Self-Healing Loop)

Qua quá trình phản biện từng dòng `git diff` và kiểm tra trực tiếp dữ liệu thật của `binding_id=59` (`shadow_walletservice.events`, rule `id=1293` `payload JSONB`), bổ sung 4 điểm hoàn thiện kỹ thuật:
1. **Bảo toàn độ chính xác `int64 > 2^53` ngay tại `gjsonValueToGo` (`transmuter_utils.go:31-38`):**
   - Trong `case gjson.JSON`, sử dụng `dec := json.NewDecoder(strings.NewReader(r.Raw)); dec.UseNumber()` thay vì `json.Unmarshal` thường, đồng thời hỗ trợ `json.Number` trong `unwrapMongoExtJSON` và `mongoNumberToGo`. Nhờ đó các số nguyên lớn trong JSONB (kể cả không bọc `$numberLong`) không bị ép sang `float64` trước khi đi vào `ApplyTransform`.
2. **Tách riêng giải mã chuỗi JSON ở tầng gốc khỏi hàm đệ quy phần tử con (`transmuter_utils.go:122-240`) & hỗ trợ `$timestamp` + quoted JSON scalar:**
   - Entry-point `NormalizeMongoExtJSONDeep(v any, mode ExtJSONCompatMode)` chỉ giải mã `string` sang JSON khi `v` ở tầng root là một chuỗi JSON object/array hợp lệ, sau đó gọi `normalizeMongoExtJSONValue(parsed, mode)`. Ở các tầng con bên trong object/array, các trường có kiểu `string` được giữ nguyên là `string`, không tự ý biến trường `string` con thành `object/array`.
   - Bổ sung bóc wrapper `{"$timestamp": {"t": ..., "i": ...}}` và trong `coerceForColumn` chấp nhận cả chuỗi JSON scalar đã được quote hợp lệ (`"..."`).
3. **Kiểm tra Whitelist tại `MasterMappingRuleHandler.Save` & bọc `NULLIF(m.transform_fn, '')` trong SQL (`cdc-cms-service` & `centralized-data-service`):**
   - Tại `master_mapping_rule_handler.go:82-115`, kiểm tra `allowedMasterTransformFns` cho `req.TransformFn` ngay trong `Save` và chuẩn hóa chuỗi rỗng `""` về `nil`.
   - Tại `master_mapping_rule_repo_gorm.go:100`, `transmuter.go:960`, `master_ddl_generator.go:112`, sử dụng `COALESCE(NULLIF(m.transform_fn, ''), v2.transform_fn) AS transform_fn` để phòng ngừa trường hợp cột `m.transform_fn` lưu chuỗi rỗng `''`.
4. **Tự động dọn `transform_fn` đặc thù của `JSONB` khi đổi `DataType` rời khỏi `JSONB`/`JSON` (`update_column_metadata.go` & `MasterMappingFieldsPage.tsx`):**
   - Khi người dùng đổi `DataType` từ `JSONB`/`JSON` sang kiểu khác (`TEXT`, `BIGINT`, ...) mà không truyền `transform_fn` mới, `UpdateColumnMetadataHandler.Handle` và `handleTypeChange` tự động xóa `transform_fn` của `JSONB` về `nil`/`""`.

