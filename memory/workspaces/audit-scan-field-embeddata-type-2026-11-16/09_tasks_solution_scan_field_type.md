# 09 Solution — scan_field_type (Propose-Only, chờ APPROVE)

## Luồng thật
`cdc-cms-service` `ScanFieldsCommand` (publish `cdc.cmd.scan-fields`) → worker `DiscoverHandler.HandleScanFields` (discover_handler.go:286) → `ScanFieldsDebezium` (discover_handler.go:195-250):
- Shadow chưa có bảng/`_raw_data` hoặc 0 row → `scanFieldsMongoSource` (đọc thẳng Mongo).
- Ngược lại → `processDiscoveryRows` với 100 row `_raw_data` mới nhất (`ORDER BY _synced_at DESC LIMIT 100`, discover_service.go:65).

## Findings
- **F1 (root cause chính — Mongo path):** `IntrospectCollection` gộp mọi doc mẫu vào 1 map bằng `fieldMap[k] = v` (mongo_introspection.go:142-147, 160-162) → **last-write-wins**, doc sau có `embedData: null` đè mất giá trị object. `scanFieldsMongoSource` marshal map đã gộp thành **1 row duy nhất** (discover_handler_mongo.go:63-66) → cơ chế "bỏ phiếu" kiểu ở `discoverResolveMongoSampledType` chỉ thấy đúng 1 giá trị. Pass 2 quét 20 doc mới nhất `_id DESC` chạy SAU nên doc mới nhất quyết định kiểu. Giá trị `nil` → `InferTypeFromRawData` trả `"TEXT"` (source_router.go:38-40).
- **F2 (Shadow path):** `processDiscoveryRows` cho `nil` bỏ phiếu `"TEXT"` (discover_handler_utils.go:28-38 + source_router.go:38-40). Nếu 100 row mới nhất toàn `embedData = null` → kết quả TEXT.
- **Bằng chứng data thật (read-only):** `cdc_shadow.shadow_traitestpbs.payment_bills`: `embedData` = object 32,358 / null 26,204 / string 3,159 / thiếu 1,193; 100 row mới nhất = **93 null / 7 object**. `mapping_rule_v2` id 743 (binding 196, `created_by` NULL = luồng processDiscoveryRows/Mongo) = `TEXT`; id 688 hiện JSONB (updated 2026-11-16 14:57, sửa tay).
- **Rủi ro phụ (ngoài phạm vi):** 3,159 row `embedData` là string thuần (`"FEE_CLIENT"`) → field đa hình; JSONB column cần ghi dạng JSON string. Liên quan workspace sync-fields ALTER.

## Phương án (1 duy nhất)
1. Tách `SampleDocuments()` trả `[]bson.M`; `IntrospectCollection` merge **không cho nil đè giá trị thật**; `IntrospectDiagnosis` mang thêm `Samples`.
2. `scanFieldsMongoSource` marshal **mỗi doc thành 1 row** → bỏ phiếu thật trên toàn bộ mẫu.
3. `processDiscoveryRows`: `nil` không bỏ phiếu (vẫn ghi nhận field tồn tại; field toàn null → set rỗng → TEXT như cũ, discover_handler_mongo.go:95-97).

## Code Demo
```go
// mongo_introspection.go
type IntrospectDiagnosis struct {
	// ...giữ nguyên
	Samples []bson.M
}

func (s *MongoIntrospectionService) SampleDocuments(uri, dbName, collectionName string, sampleSize int) ([]bson.M, error) {
	// = thân IntrospectCollection hiện tại (line 105-165), nhưng `docs = append(docs, doc)` thay vì `fieldMap[k] = v`
}

func (s *MongoIntrospectionService) IntrospectCollection(uri, dbName, collectionName string, sampleSize int) (map[string]interface{}, error) {
	docs, err := s.SampleDocuments(uri, dbName, collectionName, sampleSize)
	if err != nil {
		return nil, err
	}
	return mergeSampleDocs(docs), nil
}

func mergeSampleDocs(docs []bson.M) map[string]interface{} {
	out := make(map[string]interface{})
	for _, d := range docs {
		for k, v := range d {
			if v == nil {
				if _, seen := out[k]; !seen {
					out[k] = nil
				}
				continue // null KHÔNG đè giá trị thật
			}
			out[k] = v
		}
	}
	return out
}

// IntrospectCollectionDiagnose: docs, err := s.SampleDocuments(...); fieldMap := mergeSampleDocs(docs); diag.Samples = docs

// discover_handler_mongo.go (thay line 63-66)
rows := make([]string, 0, len(diag.Samples))
for _, d := range diag.Samples {
	if b, err := json.Marshal(d); err == nil {
		rows = append(rows, string(b))
	}
}

// discover_handler_utils.go (trong vòng for k, v := range targetMap)
set, ok := perField[k]
if !ok {
	set = map[string]struct{}{}
	perField[k] = set
}
if v == nil {
	continue // null không bỏ phiếu TEXT
}
set[servicesource.InferTypeFromRawData(v)] = struct{}{}
```

## Test (S4, Red→Green)
- `processDiscoveryRows` rows = `[{"embedData":null} x9, {"embedData":{"promotionInfo":{"campaignCode":"blackfriday"}}}]` → kỳ vọng JSONB (hiện tại vẫn JSONB vì set {TEXT,JSONB}); rows toàn null + 1 object ở Mongo path → hiện tại TEXT (Red) → sau fix JSONB (Green).
- `mergeSampleDocs([{a:{x:1}}, {a:nil}])` → `a` là map, không nil.
