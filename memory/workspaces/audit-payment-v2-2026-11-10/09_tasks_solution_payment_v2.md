---
tier: T2
suffix: payment_v2
workspace: audit-payment-v2-2026-11-10
---
# 09 Tasks Solution — Đánh Giá Tài Liệu `Payment-v2` & Thiết Kế Giải Pháp Đáp Ứng Trên `data-hub`

## 1. Tổng quan Nội dung Tài liệu `Payment-v2` (Confluence `512557312` & 3 Trang con)

Tài liệu `Payment-v2` gồm 4 trang Confluence (cùng **5 Inline Comments** đã chốt của Reviewer):
1. **Trang chính `Payment-v2` (`page_id=512557312`, v9):** Tổng quan kiến trúc chuyển đổi `payment-bill-service` (collection `payment-bills`) và `payment-service` (collection `payments`) từ MongoDB sang PostgreSQL theo mô hình tách **Core + Payload (`1:1`)** kế thừa từ `trans-his` (`465436729-transaction-history-schema.md`).
2. **Trang con 01 — `Hiện trạng và giải pháp` (`page_id=513179664`, v2):** Phân tích hiện trạng dữ liệu MongoDB (`payment-bills`, `payments`), danh sách các trường được giữ/bóc phẳng/bỏ, chiến lược lưu Canonical Extended JSON + `source_field_types` để khôi phục 100% lossless BSON gốc, và cơ chế đối soát/cách ly dữ liệu bẩn.
3. **Trang con 02 — `ERD và Data Dictionary` (`page_id=512590004`, v2):**
   - `public.payment_bill` (**48 cột**: 34 cột top-level scalar + 10 cột nested bóc phẳng + 4 cột dẫn xuất `DV`: `is_atomic`, `is_atomic_collection`, `payment_channel_ids`, `has_unapplied_lock_unavailable`).
   - `public.payment_bill_payload` (**13 cột**: `payment_bill_id BIGINT PK`, 11 cột `JSONB` giữ Canonical Extended JSON: `merchant_info, payments, payment_ids, payment_method, item, merchant_customer, embed_data, extra_info, routing_info, transaction_parties, device_info`, và `source_field_types JSONB NOT NULL` theo dõi 52 field top-level).
   - `public.payment` (**33 cột**: 27 cột top-level scalar + 6 cột nested bóc phẳng).
   - `public.payment_payload` (**9 cột**: `payment_id TEXT COLLATE "C" PK`, 7 cột `JSONB` giữ Canonical Extended JSON: `merchant, instrument, ewallet_info, extra_info, sender_bank_account, transaction_parties, reason`, và `source_field_types JSONB NOT NULL` theo dõi 39 field top-level).
4. **Trang con 03 — `Index, Query và DDL` (`page_id=512458941`, v2):** Danh sách 10 index trên `payment_bill`, 4 index trên `payment`, mapping các câu query nghiệp vụ (CMS, Open API, App, Kafka consumer, Cronjob, Export) và toàn bộ script PostgreSQL DDL (`CREATE SEQUENCE`, `CREATE TABLE`, `CREATE INDEX`).
5. **5 Inline Comments quan trọng trên Confluence (Thay đổi thiết kế so với văn bản gốc):**
   - **Comments #1–#4 (`512557312`, `513179664`, `512590004`):** Chốt **chấp nhận có downtime khi chuyển ghi** $\rightarrow$ **BỎ 2 bảng `payment_bill_control` và `payment_control` (bỏ `write_mode`)**. (Trang `512458941` đã xóa DDL của `*_control`, nhưng 3 trang kia vẫn còn nhắc tới `payment_bill_control` / `payment_control`).
   - **Comment #5 (`512590004` mục 2.1 ERD):** Chốt **`payment_bill*` và `payment*` nằm CÙNG 1 PostgreSQL database** (không tách 2 DB riêng như ghi chú trong mục 2.1).

---

## 2. Nhận định Mức độ Đáp ứng của Hệ thống & Các Việc Cần Làm Thêm (`F1..F5`)

### 2.1. Những hạng mục Hệ thống `data-hub` ĐÃ ĐÁP ỨNG (`READY`)
1. **Mô hình `1 Source -> 1 Shadow -> N Masters` & `N Shadows -> 1 Master` (`multi_shadow`):**
   - Đã vận hành thực tế cho `transaction_history` + `transaction_history_payload` tại `internal/service/master/transmute/multi_shadow.go:1-83`, `internal/service/master/transmuter_multi_shadow.go:22-252` và `internal/service/master/transmuter_multi_shadow_join_resolver.go:59-211` (hỗ trợ join qua `primary_join_field: "id"` và `secondary_join_field: "payment_bill_id"` hoặc `"payment_id"`).
2. **Bóc tách Nested Path nhiều cấp & Unwrap Extended JSON:**
   - `DynamicMapper.MapData` (`internal/service/shadow/dynamic_mapper.go:93`) và `TransmuterModule.extractColumns` (`internal/service/master/transmuter.go:1735-1753`) đã hỗ trợ bóc đường dẫn dấu chấm nhiều cấp (`merchantInfo.id`, `embedData.collectResend.nextAt`, `instrument.issuer.bankCode`...) và tự unwrap `$oid`, `$numberInt`, `$numberLong`, `$numberDecimal`, `$date`.
3. **Sinh cột JSON Subset (`Type/Presence` & `Value`) qua `@type_presence:{...}` / `{...}` & Bộ `transform_fn` cơ bản:**
   - Đã có sẵn `@type_presence:{...}` tại `internal/service/master/transmuter.go:1728-1733` và `internal/service/master/transmuter_utils.go:574-640` (`buildTypePresenceSubset`, `detectBsonType`) đúng chuẩn *"present $\rightarrow$ ghi tên kiểu BSON; explicit null $\rightarrow$ `"null"`; missing $\rightarrow$ omit key"*.
   - Đã có sẵn các hàm `bigint_str`, `numeric_cast`, `mongo_date_ms`, `jsonb_passthrough`, `extjson_relaxed`, `extjson_relaxed_num_str` tại `internal/service/master/transform_registry.go:31-43`.

### 2.2. Những hạng mục trong `data-hub` (CDC) CẦN DỌN SẠCH HARDCODE & BỔ SUNG TỔNG QUÁT (`F1..F2`)

> **Lưu ý kiến trúc theo đúng luồng vận hành của hệ thống:**
> - Phạm vi đánh giá và thực thi chỉ nằm gọn trong **hệ thống CDC (`data-hub`)**.
> - Hệ thống có bước **Trigger chuyển tiếp từ bảng Master CDC $\rightarrow$ bảng Master Business**, do đó bảng Master CDC vẫn giữ nguyên cấu trúc cột chuẩn của CDC (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`), và các cột dẫn xuất mang tính quy tắc nghiệp vụ riêng của `payment_bill` (`is_atomic`, `is_atomic_collection`, `payment_channel_ids`, `has_unapplied_lock_unavailable`) được xử lý ở tầng Trigger `Master CDC -> Master Business` (hoặc trích xuất qua biểu thức `gjson` tổng quát như `payments.#.channelID`). **Tuyệt đối CẤM nhúng hàm hay directive chứa tên bảng/hằng số nghiệp vụ (`payment_bill`, `ATOMIC`, `VIET_QR`...) vào lõi `centralized-data-service`!**
> - Với cột `payment_ids` cần chuẩn hóa số thành chuỗi trong JSONB, `data-hub` **đã có sẵn** hàm tổng quát `extjson_relaxed_num_str` tại `internal/service/master/transform_registry.go:42` (`ExtJSONModeRelaxedNumStr` tại `internal/service/master/transmuter_utils.go:138,188-200`).

Chỉ còn **2 vấn đề cốt lõi trong lõi CDC (`centralized-data-service`)** cần xử lý theo hướng **100% tổng quát (Zero Hardcoding)**:

- **Phát hiện `F1` (Xử lý kiểu mảng PostgreSQL `TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]` tổng quát trong `coerceForColumn`):**
  - **Bằng chứng:** Tại `internal/service/master/transmuter_utils.go:361-375` (`coerceForColumn`) và `:456-466` (`sqlBindValueTransmute`), mọi slice `[]any` đều bị `json.Marshal(v)` thành chuỗi JSON `'["a","b"]'`. Khi cột đích trên Master PostgreSQL có kiểu mảng (`TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]` — vốn đã được whitelist tại `internal/service/shadow/type_resolver.go:41`), PostgreSQL báo lỗi `SQLSTATE 22P02: malformed array literal: "["a","b"]"` vì Postgres array literal yêu cầu định dạng `{"a","b"}`.
- **Phát hiện `F2` (Xóa bỏ 100% các mảng và câu lệnh `switch/case` HARDCODE tên cột trong `recon_multi_shadow_b.go`, `mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`, `mongo_stale_field_pruner.go` — Chuyển sang nhận diện qua cờ `Chế độ tạo: (JSON) Subset / Loại Subset: Type/Presence | Value` của Mapping Rule):**
  - **Bằng chứng:**
    1. Trên giao diện `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:111-113,533-536,1163-1174`, khi tạo cột với **"Chế độ tạo: JSON Subset"**:
       - Chọn **Loại Subset = `Type/Presence`** $\rightarrow$ `source_field` mang cờ tiền tố `@type_presence:{...}` (kiểm tra bằng `isTypePresenceRule(sourceField)` tại `internal/service/master/transmuter_utils.go:574-579` và `internal/service/master/transmuter.go:1728`).
       - Chọn **Loại Subset = `Value (JSON)`** $\rightarrow$ `source_field` mang biểu thức `{...}` (`strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")`).
    2. Tuy nhiên, tại 4 file backend hiện tại lại đang **hardcode cứng tên cột** thay vì kiểm tra cờ của Mapping Rule:
       - `internal/service/recon/recon_multi_shadow_b.go:154-173`: Hardcode cả mảng 25 cột `mongoStrippedRowDiffCols` của `transaction_history` **VÀ** hardcode `case "source_field_types", "sourceFieldTypes", "source_field_values", "sourceFieldValues": return true`.
       - `internal/service/master/mongo_bson_restorer.go:19-48,85-90,111-141,484-486`: Hardcode `doc["source_field_types"]` / `doc["sourceFieldTypes"]`, `snakeToCamelCoreMap` (11 cột), `denormalizedFlatColumns` (20 cột), `optionalNullableFields` (2 cột), `[]string{"info", "sender", "receiver", "error"}` (dòng `111`), và `"transaction_history_id"` (dòng `68, 76, 130, 484`).
       - `internal/service/master/mongo_extjson_unwrapper.go:185-192,351-356`: Hardcode `doc["source_field_values"]` / `doc["sourceFieldValues"]` và `doc["info"] != nil || doc["sender"] != nil || doc["receiver"] != nil || doc["transId"] != nil`.
       - `internal/service/master/mongo_stale_field_pruner.go:20-23`: Hardcode `k == "source_field_types" || k == "sourceFieldTypes"`.

---

## 3. Thiết kế Giải pháp Tối ưu Nhất trên `data-hub` (100% Tổng Quát — Zero Table/Column Hardcoding) & Code Demo

### 3.1. Giải pháp cho `F1`: Hỗ trợ chuẩn hóa PostgreSQL Array Literal (`TEXT[]`, `BIGINT[]`, `INTEGER[]`, `SMALLINT[]`, `UUID[]`) tổng quát trong `coerceForColumn` (`internal/service/master/transmuter_utils.go`)

- Thêm hàm `isPostgresArrayColumnType(dataType string) bool` và `formatPostgresArrayLiteral(v any) (string, bool)` trong `internal/service/master/transmuter_utils.go` ngay trước nhánh `if isJSONColumnType(dataType) || composite` (dòng `361`).

```go
// Code Demo 1 — internal/service/master/transmuter_utils.go
func isPostgresArrayColumnType(dataType string) bool {
	u := strings.ToUpper(strings.TrimSpace(dataType))
	return strings.HasSuffix(u, "[]") || strings.HasPrefix(u, "_")
}

func formatPostgresArrayLiteral(v any) (string, bool) {
	var items []any
	switch x := v.(type) {
	case []any:
		items = x
	case []string:
		items = make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
	case string:
		trimmed := strings.TrimSpace(x)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			return trimmed, true
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
				return "", false
			}
		} else {
			return "", false
		}
	default:
		return "", false
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		it = unwrapMongoExtJSON(it)
		if it == nil {
			parts = append(parts, "NULL")
			continue
		}
		s := fmt.Sprintf("%v", it)
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts = append(parts, `"`+escaped+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}", true
}
```

### 3.2. Giải pháp cho `F2`: Nhận diện cột qua cờ `Chế độ tạo: (JSON) Subset / Loại Subset: Type/Presence | Value` & Xóa sạch 100% hardcode tên cột trong `recon_multi_shadow_b.go`, `mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`, `mongo_stale_field_pruner.go`

1. **Export bộ hàm nhận diện cờ `JSON Subset` (`Type/Presence` vs `Value`) tại `internal/service/master/transmuter_utils.go:574-600` & truyền qua `colTypes` trong `transmuter.go`:**
   - `IsTypePresenceRule(sourceField string) bool`: kiểm tra `strings.HasPrefix(strings.TrimSpace(sourceField), "@type_presence:")` (cờ **Chế độ tạo: (JSON) Subset — Loại Subset: Type/Presence**).
   - `IsJSONValueSubsetRule(sourceField string) bool`: kiểm tra `strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")` (cờ **Chế độ tạo: (JSON) Subset — Loại Subset: Value**).
   - `IsJSONSubsetRule(sourceField string) bool`: `IsTypePresenceRule(sourceField) || IsJSONValueSubsetRule(sourceField)`.
   - Khi tạo `colTypes` từ `rules` (`buildColTypesFromRules(rules)` trong `transmuter.go:393,888,1164`), gắn hậu tố cờ `@type_presence` hoặc `@value_subset` vào `colTypes[r.TargetColumn]` để `CoerceRecordToBSONDoc`, `ApplySourceFieldTypesToBSONDoc` và `StaleMongoKeysToPrune` nhận diện chính xác cột metadata theo cờ của Rule bất kể người dùng đặt tên cột đích là gì.

```go
// Code Demo 2 — internal/service/master/transmuter_utils.go (Nhận diện cờ JSON Subset Type/Presence & Value)
const (
	typePresencePrefix = "@type_presence:"
	colFlagTypePres    = "@type_presence"
	colFlagValueSubset = "@value_subset"
)

func IsTypePresenceRule(sourceField string) bool {
	return strings.HasPrefix(strings.TrimSpace(sourceField), typePresencePrefix)
}

func IsJSONValueSubsetRule(sourceField string) bool {
	s := strings.TrimSpace(sourceField)
	return len(s) >= 2 && strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}

func IsJSONSubsetRule(sourceField string) bool {
	return IsTypePresenceRule(sourceField) || IsJSONValueSubsetRule(sourceField)
}

func isTypePresenceColType(dt string) bool {
	return strings.Contains(dt, colFlagTypePres)
}

func isValueSubsetColType(dt string) bool {
	return strings.Contains(dt, colFlagValueSubset)
}
```

2. **Tại `internal/service/recon/recon_multi_shadow_b.go` & `internal/service/recon/recon_tier_b.go` (`RunRowDiffB`):**
   - **Xóa bỏ hoàn toàn biến `mongoStrippedRowDiffCols` VÀ xóa bỏ `case "source_field_types", "sourceFieldTypes", "source_field_values", "sourceFieldValues":`!**
   - Nhận diện cột JSON Subset (`Type/Presence` hoặc `Value`) trực tiếp từ `master.IsJSONSubsetRule(rule.SourceField)`.
   - Với các cột bóc phẳng/khóa ngoại/optional bị loại bỏ khi ghi lên MongoDB Master: kiểm tra động trên `expectedBSON` sau khi qua `ApplySourceFieldTypesToBSONDoc`.

```go
// Code Demo 3 — internal/service/recon/recon_multi_shadow_b.go (0% hardcode tên cột)
func shouldSkipMongoRowDiffRule(isMongoMaster bool, mrow map[string]any, expectedBSON bson.M, sourceField, targetCol string) bool {
	if !isMongoMaster {
		return false
	}
	// Nhận diện cột JSON Subset (Type/Presence hoặc Value) qua cờ trên rule.SourceField
	if master.IsJSONSubsetRule(sourceField) {
		return true
	}
	col := strings.TrimSpace(targetCol)
	if mrow != nil && resolveMasterRowDiffVal(mrow, col) != nil {
		return false
	}
	if expectedBSON != nil {
		if _, exists := expectedBSON[col]; !exists {
			camel := snakeToCamelIdent(col)
			if _, existsCamel := expectedBSON[camel]; !existsCamel {
				return true
			}
		}
	}
	return false
}
```

3. **Tại `internal/service/master/mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go` và `mongo_stale_field_pruner.go`:**
   - **Xóa bỏ hoàn toàn mọi chuỗi hardcode `"source_field_types"`, `"sourceFieldTypes"`, `"source_field_values"`, `"sourceFieldValues"`, `snakeToCamelCoreMap`, `denormalizedFlatColumns`, `optionalNullableFields` và `[]string{"info", "sender", "receiver", "error"}`!**
   - Thay bằng cơ chế 100% động dựa trên cờ `colTypes` (`@type_presence`, `@value_subset`) và cấu trúc BSON:
     - **Trích xuất & xóa cột `Type/Presence` và `Value Subset` theo cờ:** Quét các key trong `doc` có `isTypePresenceColType(ctMap[k])` (hoặc thoả mãn cấu trúc bản đồ descriptor BSON `isTypePresenceDescriptorMap` khi chạy độc lập không truyền `colTypes`), nạp vào `typeMap` rồi `delete(doc, k)`. Tương tự với `isValueSubsetColType(ctMap[k])`, giải nén các trường con vào `doc` + `typeMap` rồi `delete(doc, k)`.
     - **Tại `StaleMongoKeysToPrune` (`mongo_stale_field_pruner.go`):** Bỏ qua các cột có `isTypePresenceColType(ctMap[k])` thay vì so sánh cứng tên cột.
     - **Dọn cột bóc phẳng (denormalized) & khóa ngoại bảng phụ động:** Tự động đối chiếu với các nested `bson.M` trong `doc` và khóa `_id` kết hợp `typeMap`.

```go
// Code Demo 4 — internal/service/master/mongo_bson_restorer.go & mongo_stale_field_pruner.go
func consumeTypePresenceAndValueSubsetCols(doc bson.M, ctMap map[string]string) map[string]sourceFieldTypeEntry {
	typeMap := make(map[string]sourceFieldTypeEntry)
	for k, rawVal := range doc {
		if strings.HasPrefix(k, "_") {
			continue
		}
		if isTypePresenceColType(ctMap[k]) || (len(ctMap) == 0 && isTypePresenceDescriptorMap(rawVal)) {
			for path, entry := range parseSourceFieldTypesMap(rawVal) {
				typeMap[path] = entry
			}
			delete(doc, k)
			continue
		}
		if isValueSubsetColType(ctMap[k]) {
			extractValueSubsetColumn(doc, rawVal, typeMap)
			delete(doc, k)
		}
	}
	return typeMap
}

func StaleMongoKeysToPrune(rec map[string]any, doc bson.M, targetKey string, colTypes ...map[string]string) []string {
	if doc == nil {
		return nil
	}
	var ctMap map[string]string
	if len(colTypes) > 0 {
		ctMap = colTypes[0]
	}
	var out []string
	for k, rawVal := range rec {
		if strings.HasPrefix(k, "_") || k == targetKey ||
			isTypePresenceColType(ctMap[k]) || (len(ctMap) == 0 && isTypePresenceDescriptorMap(rawVal)) {
			continue
		}
		if _, stillPresent := doc[k]; !stillPresent {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
```

---

## 4. Giải đáp & Đính chính về `F1` (Kiểu mảng PostgreSQL `TEXT[]` vs Chuẩn `JSONB` của Hệ thống `data-hub` CDC)

### 4.1. Bản chất kiểu mảng `TEXT[]` trong PostgreSQL
- **Khai báo DDL:** `TEXT[]` (hoặc `TEXT ARRAY`), `BIGINT[]`, `INTEGER[]`, `UUID[]`.
- **Trong `information_schema.columns` của PostgreSQL:**
  - Cột `data_type` trả về chuỗi `'ARRAY'`.
  - Cột `udt_name` trả về tên kiểu nội bộ có tiền tố gạch dưới: `'_text'` (`TEXT[]`), `'_int8'` (`BIGINT[]`), `'_int4'` (`INTEGER[]`), `'_int2'` (`SMALLINT[]`), `'_uuid'` (`UUID[]`).
- **Định dạng giá trị (Literal):** PostgreSQL Array Literal dùng dấu ngoặc nhọn `{"a","b"}` (khác với mảng JSON `["a","b"]` của kiểu `JSONB`).

### 4.2. Hiện trạng Hỗ trợ trên Hệ thống CMS (`cdc-cms-web` & `cdc-cms-service`) & Lý do KHÔNG CẦN làm `F1`
1. **Trên Hệ thống CMS hiện tại (`cdc-cms-web` & `cdc-cms-service`):**
   - **Frontend (`cdc-cms-web/src/constants/postgresDataTypes.ts:1-23`):** Danh sách `POSTGRES_DATA_TYPES` chỉ có 21 kiểu đơn và `JSONB`/`JSON`, **không có `TEXT[]`**.
   - **Backend (`cdc-cms-service/pkgs/utils/dbutil/validate.go:31-43` & `pkgs/utils/pg_validator.go:30-36`):** Map `allowed` trong `IsValidDataType` chỉ cho phép các kiểu đơn và `JSONB`/`JSON`, **không có `TEXT[]`**.
2. **Tại sao Hệ thống Đang Dùng `JSONB` (`["PAYOUT-48448-Cb061fPjdt-1786350427989"]`) và KHÔNG CẦN làm `F1`:**
   - Tại `centralized-data-service/internal/service/source/scan_service.go:153-155` và `internal/handler/source/discover_handler_mongo.go:114-115`, toàn bộ trường kiểu mảng (`array`) từ MongoDB khi quét sang PostgreSQL đều được chuẩn hóa thành **`JSONB`** (hiển thị trên Shadow và Master CDC dưới dạng mảng JSON chuẩn như `["PAYOUT-48448-Cb061fPjdt-1786350427989"]`).
   - Kiểu `TEXT[]` nêu trong tài liệu Confluence `Payment-v2` (như `payment_methods TEXT[]` hay `refunded_operation_ids TEXT[]`) là kiểu của **bảng Master Business** ở phía sau bước **Trigger `Master CDC -> Master Business`**, không phải kiểu của bảng Shadow hay bảng Master CDC!
   - Do đó, ở phạm vi **CDC (`Shadow -> Master CDC`)**, hệ thống giữ nguyên `data_type = "JSONB"` là **hoàn toàn đúng thiết kế và đã chạy chuẩn 100%**, **không cần sửa CMS sang `TEXT[]` và không cần làm `F1`**!

---

## 5. Phân Tích Khoảng Trống (Gap Analysis): Tại Sao `/shadow/:id/mappings` và Master Mapping Chưa Có Lựa Chọn Kiểu Mảng (`_TEXT` / `TEXT[]`)

### 5.1. Hiện tượng User thấy trên `/shadow/73/mappings?binding_id=192`
1. Cột **Data Type Current** hiển thị `<Tag color="orange">_TEXT ⚠️</Tag>` (hoặc `_INT8`, `_INT4`...).
2. Cột **Data Type Target** (và dropdown Master Data Type bên Master Mapping) khi bấm vào **hoàn toàn không có lựa chọn nào có dấu `_` (như `_TEXT`) cũng như không có `TEXT[]`**.
3. User muốn chọn khớp kiểu hoặc cấu hình kiểu mảng nhưng không có chỗ để chọn.

### 5.2. Nguyên nhân gốc rễ (Root Cause) & 3 Khoảng trống (Gaps) của Hệ thống

| Tầng | Hiện trạng Code | Khoảng trống (Gap) |
| :--- | :--- | :--- |
| **1. Shadow Schema Reader (`cdc-cms-service`)** | [`shadow_schema_reader_gorm.go:45-46`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go#L45-L46):<br>`case "USER-DEFINED", "ARRAY": return strings.ToUpper(udtName)` | Khi cột shadow vật lý là mảng PostgreSQL (`text[]`), `information_schema.columns` trả về `data_type = 'ARRAY'`, `udt_name = '_text'`. Code trả thẳng `_TEXT` thô từ catalog thay vì chuẩn hóa sang tên DDL chuẩn `TEXT[]` (trong khi [`discovery_utils_sql.go:131-144`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discovery_utils_sql.go#L131-L144) lại map `_text -> TEXT[]`). |
| **2. Frontend Constants & Dropdown (`cdc-cms-web`)** | [`postgresDataTypes.ts:1-23`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts#L1-L23) & [`clickhouseDataTypes.ts:123-141`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/clickhouseDataTypes.ts#L123-L141):<br>`POSTGRES_DATA_TYPES` chỉ có 21 kiểu scalar (`VARCHAR`, `TEXT`, `JSONB`...). | Cả Shadow Mapping ([`MappingFieldsPage.tsx:21,421-434`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx#L21)) và Master Mapping ([`MasterMappingFieldsPage.tsx:775`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx#L775)) đều lấy danh sách từ `POSTGRES_DATA_TYPES`. Do mảng này không chứa `TEXT[]` hay `_TEXT` nên dropdown không có để chọn. |
| **3. CMS Backend Validator (`cdc-cms-service`)** | [`dbutil/validate.go:31-44`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/utils/dbutil/validate.go#L31-L44) & [`pg_validator.go:30-36`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/utils/pg_validator.go#L30-36):<br>Whitelist map `allowed` chỉ chứa các kiểu đơn scalar. | Nếu user cố tình nhập hoặc gửi API `data_type = "TEXT[]"` hay `"_TEXT"`, handler [`master_mapping_rule_handler.go:101,214`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_mapping_rule_handler.go#L101) sẽ chặn và trả về lỗi HTTP 400 `"data_type không hợp lệ"`. |

> **Điểm nghịch lý trong kiến trúc hiện tại:**
> - Database Constraint tại [`001_init_schema.sql:147-149`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/core/001_init_schema.sql#L147-L149) (`mapping_rules_data_type_chk`) và Data Plane tại [`type_resolver.go:41`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/type_resolver.go#L41) **ĐÃ WHITELIST CÚ PHÁP `(SMALLINT|INTEGER|BIGINT|TEXT|UUID)[]`**!
> - Nhưng CMS UI và CMS Backend Validator lại bị bỏ quên, chưa mở các kiểu mảng này, đồng thời bộ đọc schema Shadow lại trả về tên thô `_TEXT` thay vì `TEXT[]`.

---

### 5.3. Phương Án Xử Lý Toàn Diện (Nếu Muốn Mở Cho Chọn Kiểu Mảng Trên CMS)

1. **Chuẩn hóa tầng Schema Reader (`cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go`):**
   Đổi từ `strings.ToUpper(udtName)` sang map chuẩn DDL PostgreSQL (`TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]`):
   ```go
   case "USER-DEFINED", "ARRAY":
       switch strings.ToLower(udtName) {
       case "_text", "_varchar", "_bpchar":
           return "TEXT[]"
       case "_int8":
           return "BIGINT[]"
       case "_int4":
           return "INTEGER[]"
       case "_int2":
           return "SMALLINT[]"
       case "_uuid":
           return "UUID[]"
       default:
           return strings.ToUpper(udtName)
       }
   ```
2. **Mở Whitelist Backend CMS (`cdc-cms-service/pkgs/utils/dbutil/validate.go` & `pg_validator.go`):**
   Hỗ trợ cú pháp hậu tố `[]` cho các kiểu mảng:
   ```go
   if strings.HasSuffix(s, "[]") {
       baseType := strings.TrimSuffix(s, "[]")
       switch baseType {
       case "TEXT", "VARCHAR", "CHAR", "BIGINT", "INTEGER", "INT", "SMALLINT", "UUID":
           return true
       }
   }
   ```
3. **Mở Dropdown Frontend (`cdc-cms-web/src/constants/postgresDataTypes.ts`):**
   Bổ sung các kiểu mảng phổ biến vào danh sách chọn:
   ```ts
   export const POSTGRES_DATA_TYPES = [
     'VARCHAR',
     'VARCHAR(50)',
     'VARCHAR(100)',
     'VARCHAR(255)',
     'CHAR',
     'TEXT',
     'INTEGER',
     'BIGINT',
     'SMALLINT',
     'NUMERIC',
     'DECIMAL(18,6)',
     'DOUBLE PRECISION',
     'REAL',
     'BOOLEAN',
     'TIMESTAMP',
     'TIMESTAMPTZ',
     'DATE',
     'JSONB',
     'JSON',
     'BYTEA',
     'OBJECTID',
     'TEXT[]',
     'BIGINT[]',
     'INTEGER[]',
     'UUID[]',
   ] as const;
   ```
