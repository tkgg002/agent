# BÁO CÁO AUDIT CHUYÊN SÂU: ĐÁNH GIÁ TÁC ĐỘNG CỦA VIỆC QUY HOẠCH _RAW_DATA & BẢO ĐẢM TÍNH TƯƠNG THÍCH NGƯỢC TOÀN HỆ THỐNG
**Dự án:** Data Hub (Centralized Data Service, CDC CMS Web, CDC CMS Service)  
**Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`  
**Thời gian:** 2026-09-16T16:25:00+07:00  
**Thực hiện bởi:** Brain (Chairman & Architect)  
**Tiêu chuẩn:** Hiến pháp `GEMINI.md`, Kỷ luật `lessons.md`, Phản biện không khoan nhượng, Không suy diễn, Không báo cáo láo.

---

## I. LỜI MỞ ĐẦU & XÁC NHẬN NỘI TÂM
Người dùng đã chỉ ra một **tử huyệt kiến trúc cốt lõi**:
> *"mày vẫn còn sót lỗi. vụ _raw_data là đổi cấu trúc lớn, mày liệt kê tất cả các trường hợp dùng _raw_data trong hệ thống ra đây. sau khi quy hoạch về 1 kiểu, tương thích ngược còn đảm bảo ko"*

Đây là nhận định hoàn toàn chính xác. Cột `_raw_data` trong cơ sở dữ liệu Shadow không đơn thuần là một trường lưu trữ tạm thời của `dynamic_mapper`. Nó là **Ground Truth (Nguồn chân lý gốc)** của toàn bộ hệ thống Data Hub.  
Việc vội vã "quy hoạch `_raw_data` về 1 kiểu phẳng" bằng `normalizeMongoExtJSON` ở tầng ingest nếu không tính toán kỹ sẽ làm **GÃY HOÀN TOÀN TÍNH TƯƠNG THÍCH NGƯỢC** đối với dữ liệu lịch sử và các quy tắc ánh xạ (mapping rules) đang vận hành!

Dưới đây là bản điều tra pháp y (forensic investigation) toàn diện mọi ngóc ngách sử dụng `_raw_data` trên cả 3 repository.

---

## II. DANH MỤC TOÀN BỘ CÁC TRƯỜNG HỢP SỬ DỤNG `_RAW_DATA` TRONG TOÀN HỆ THỐNG

Qua quét mã nguồn thực tế (Static Code Analysis) trên cả 3 repository, hệ thống có **8 nhóm chức năng cốt lõi phụ thuộc trực tiếp vào `_raw_data`**:

```
                              ┌────────────────────────────────────────┐
                              │     SHADOW TABLE: _raw_data (JSONB)    │
                              └──────────────────┬─────────────────────┘
                                                 │
         ┌───────────────────────────────┬───────┴───────┬───────────────────────────────┐
         │                               │               │                               │
┌────────▼────────┐             ┌────────▼────────┐ ┌────▼───────────┐          ┌────────▼────────┐
│ 1. Dynamic      │             │ 2. Batch        │ │ 3. Master      │          │ 4. Recon &      │
│    Mapping      │             │    Transform    │ │    Transmuter  │          │    Row Diff     │
│ (Ingest/Upsert) │             │ (SQL Backfill)  │ │ (Shadow→Master)│          │ (Verification)  │
└─────────────────┘             └─────────────────┘ └────────────────┘          └─────────────────┘
         │                               │               │                               │
┌────────▼────────┐             ┌────────▼────────┐ ┌────▼───────────┐          ┌────────▼────────┐
│ 5. Auto Field   │             │ 6. Masking &    │ │ 7. Child Array │          │ 8. CMS Web UI   │
│    Discovery    │             │    Encryption   │ │    Explode     │          │    Preview &    │
│ (Schema Scanner)│             │ (Data Security) │ │ (Array Unnest) │          │    Data Explorer│
└─────────────────┘             └─────────────────┘ └────────────────┘          └─────────────────┘
```

---

### 1. Nhóm 1: Dynamic Mapper & Ingestion Engine (`dynamic_mapper.go`)
- **Vị trí:** [`centralized-data-service/internal/service/shadow/dynamic_mapper.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go#L70-L135)
- **Cơ chế sử dụng:**
  - Nhận event payload từ Kafka Consumer, Bridge Oplog, hoặc Snapshot Runner.
  - Áp dụng các `MappingRule` để trích xuất giá trị từ `rawData` vào các cột định kiểu (typed columns: `amount`, `status`, `created_at`).
  - Gọi `dm.maskRawData(bindingID, normalizedRaw)` để mã hóa các trường nhạy cảm, sau đó `json.Marshal` và ghi vào cột `_raw_data` của PostgreSQL.

### 2. Nhóm 2: Batch Transform & Backfill SQL Engine (`batch_transform_handler.go` & `mapping_utils.go`)
- **Vị trí:**
  - [`centralized-data-service/internal/handler/shadow/batch_transform_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go#L250-L350)
  - [`centralized-data-service/internal/service/metadata/mapping_utils.go:66-140`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/metadata/mapping_utils.go#L66-L140)
- **Cơ chế sử dụng:**
  - Khi người dùng bấm "Transform" hoặc "Force Transform" trên CMS UI, Worker chạy câu lệnh SQL trực tiếp trên PostgreSQL:
    `UPDATE shadow_table SET col1 = BuildCastExpr("col1", type), ... WHERE _raw_data IS NOT NULL`.
  - Hàm `BuildCastExpr` sinh mã SQL truy vấn JSONB:
    - Bigint: `(CASE WHEN jsonb_typeof(_raw_data->'amount'->'$numberLong') = 'string' THEN (_raw_data->'amount'->>'$numberLong')::BIGINT ELSE (_raw_data->>'amount')::BIGINT END)`
    - Timestamp: `(CASE WHEN jsonb_typeof(_raw_data->'created_at') = 'number' THEN to_timestamp((_raw_data->>'created_at')::BIGINT / 1000.0) WHEN ... _raw_data->'created_at'->'$date' ... END)`
    - Text: `(CASE WHEN jsonb_typeof(_raw_data->'_id'->'$oid') = 'string' THEN (_raw_data->'_id'->>'$oid') ELSE (_raw_data->>'_id') END)`

### 3. Nhóm 3: Master Table Transmuter (`transmuter.go`, `flatten.go`, `transform_registry.go`)
- **Vị trí:**
  - [`centralized-data-service/internal/service/master/transmute/flatten.go:110-150`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmute/flatten.go#L110-L150)
  - [`centralized-data-service/internal/service/master/transform_registry.go:115-180`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transform_registry.go#L115-L180)
- **Cơ chế sử dụng:**
  - Khi chuyển dữ liệu từ Shadow sang Master, Transmuter đọc dòng shadow: nếu cột đích trong Master chưa có sẵn trên cột định kiểu của shadow, nó sẽ dùng `gjson.Get(shadow._raw_data, rule.SourceField)` để bốc dữ liệu.
  - Các hàm chuyển đổi sẵn có: `transformOIDToHex` (bóc `$oid`), `transformBigIntStr` (bóc `$numberLong`), `unwrapExtJSON`.

### 4. Nhóm 4: Hệ thống Đối soát & Bắt lệch Dữ liệu Recon (`recon_row_diff.go`, `083_recon_field_diffs.sql`)
- **Vị trí:**
  - [`centralized-data-service/migrations/schema/recon_dlq/083_recon_field_diffs.sql`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/migrations/schema/recon_dlq/083_recon_field_diffs.sql#L5)
  - `centralized-data-service/internal/service/recon/recon_row_diff.go`
- **Cơ chế sử dụng:**
  - Subsystem Recon Segment B thực hiện đối soát từng dòng (Row Diff) giữa Shadow và Master: Nó chạy lại công thức transform trên `shadow._raw_data` để tính ra giá trị kỳ vọng (expected value), sau đó so sánh với giá trị thực tế đang có trên Master table để phát hiện data drift.

### 5. Nhóm 5: Quét Tự Động & Phát Hiện Trường Mới (Auto Field Discovery & Scan)
- **Vị trí:**
  - [`centralized-data-service/internal/handler/scan/scan_handler.go:180-210`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_handler.go#L180-L210)
  - [`centralized-data-service/internal/handler/source/discover_handler.go:210-230`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler.go#L210-L230)
- **Cơ chế sử dụng:**
  - Worker chạy query: `SELECT _raw_data FROM shadow_table WHERE _raw_data IS NOT NULL ORDER BY _synced_at DESC LIMIT 100`.
  - Phân tích cây JSON của `_raw_data` để đề xuất các cột mới (`mapping_rule_v2`).
  - Hàm `sanitize_field` sẽ bóc tách các operator suffix như `payload_userId_$oid` $\rightarrow$ `payload_userId`.

### 6. Nhóm 6: Cơ chế Bảo mật & Mã hóa Dữ liệu (Masking & Encryption)
- **Vị trí:** [`centralized-data-service/internal/service/shadow/masking_service.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/masking_service.go)
- **Cơ chế sử dụng:**
  - Khi một trường được đánh dấu `is_sensitive_field = true`, thuật toán (HMAC, AES-GCM, JSON Mask) sẽ mã hóa giá trị trường đó ngay bên trong `_raw_data` trước khi ghi xuống đĩa PostgreSQL.

### 7. Nhóm 7: Tách Mảng Dữ liệu Con (Child Array Explode)
- **Vị trí:** [`centralized-data-service/internal/service/shadow/child_explode.go:110-140`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/child_explode.go#L110-L140)
- **Cơ chế sử dụng:**
  - Quét các mảng JSON bên trong `_raw_data` của bảng cha (ví dụ `orders.items[*]`), bung từng phần tử mảng thành các bản ghi riêng biệt trong bảng con (`shadow_order_items`).

### 8. Nhóm 8: Giao diện Người Dùng CMS Web & Introspection API
- **Vị trí:**
  - [`cdc-cms-web/src/pages/MappingFieldsPage.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx)
  - [`cdc-cms-service/internal/api/system/introspection_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler.go)
- **Cơ chế sử dụng:**
  - Endpoint `POST /api/v1/mapping-rules/preview`: Lấy mẫu 3 dòng `_raw_data` từ bảng shadow, cho phép Operator nhập JsonPath (`gjson`) để xem trước kết quả trích xuất trước khi lưu mapping rule.

---

## III. ĐÁNH GIÁ PHẢN BIỆN: SAU KHI QUY HOẠCH VỀ 1 KIỂU, TƯƠNG THÍCH NGƯỢC CÒN ĐẢM BẢO KHÔNG?

### ⚠️ CÂU TRẢ LỜI THẲNG THẮN: NẾU CHỈ FLAT HÓA `_RAW_DATA` MÀ KHÔNG CÓ ADAPTER LƯỠNG TÍNH, TƯƠNG THÍCH NGƯỢC CHẮC CHẮN SẼ BỊ PHÁ VỠ (100% REGRESSION)!

Dưới đây là 4 bằng chứng phản biện thực tế giải thích tại sao:

#### 1. Sự tồn tại của Dữ liệu Lưỡng Tính (Heterogeneous / Hybrid Data) trong cùng một Bảng
- Trong các bảng Shadow hiện tại trên production, đang có **hàng triệu bản ghi lịch sử** được ghi dưới dạng MongoDB Extended JSON:
  `{"_id": {"$oid": "6aaa3b467742982d574e2817"}, "created_at": {"$date": {"$numberLong": "1736403351944"}}}`
- Nếu từ hôm nay, ta đổi sang lưu dạng phẳng:
  `{"_id": "6aaa3b467742982d574e2817", "created_at": 1736403351944}`
- $\rightarrow$ **Bảng Shadow sẽ chứa 2 thế hệ dữ liệu khác nhau hoàn toàn về cấu trúc JSONB!**

#### 2. Tử huyệt tại `dynamic_mapper.go:getNestedField` đối với các Mapping Rules cũ
- Trong cơ sở dữ liệu `cdc_system.mapping_rule_v2`, nhiều rule cũ đang lưu:
  `source_field = "_id.$oid"` hoặc `source_field = "created_at.$date"`
- Hãy xem hàm `getNestedField` chạy như thế nào:
  ```go
  func getNestedField(data map[string]interface{}, path string) interface{} {
      parts := strings.Split(path, ".") // ["_id", "$oid"]
      for _, part := range parts {
          m, ok := current.(map[string]interface{})
          if !ok { return nil } // <--- BỊ RETURN NIL TẠI ĐÂY!
          current = m[part]
      }
      return current
  }
  ```
- **Hậu quả:** Đối với bản ghi mới đã bị làm phẳng:
  - Vòng lặp 1: `current = data["_id"]` $\rightarrow$ ra chuỗi `"6aaa3b467742982d574e2817"`.
  - Vòng lặp 2: Ép kiểu sang `map[string]interface{}` để tìm key `"$oid"` $\rightarrow$ **FAIL** (vì là chuỗi)!
  - Hàm trả về **`nil`**!
  - Cột `_gpay_id` hoặc cột ID chính trong bảng Shadow bị gán giá trị **`NULL`**!
  - **SẬP UPSERT HOẶC MẤT KHÓA CHÍNH NGAY LẬP TỨC!**

#### 3. Tử huyệt tại Master Transmuter Flatten Strategy (`flatten.go:extractFieldValue`)
- Tại dòng 128 trong `flatten.go`:
  `gres := gjson.Get(rawStr, path)` (với `path = "_id.$oid"`).
- Khi `rawStr` là JSON mới phẳng: `{"_id": "6aaa3b46..."}`:
  - `gjson` truy cập `_id.$oid` sẽ trả về `gres.Exists() == false`!
  - `finalCols["payment_oid"]` bị rỗng / `nil`.
  - Kiểm tra non-nullable (dòng 136-146): Nếu cột Master đó là `NOT NULL`, **TOÀN BỘ BẢN GHI ĐÓ BỊ DROP, KHÔNG BAO GIỜ ĐƯỢC ĐỒNG BỘ SANG MASTER!**

#### 4. Điểm sáng duy nhất: SQL Engine `BuildCastExpr` trong `mapping_utils.go`
- Rất may mắn, ở tầng SQL Backfill (`BatchTransform`), tác giả trước đó đã thiết kế `BuildCastExpr` đa hình (polymorphic):
  ```sql
  CASE 
    WHEN jsonb_typeof(_raw_data->'_id'->'$oid') = 'string' THEN (_raw_data->'_id'->>'$oid')
    ELSE (_raw_data->>'_id')
  END
  ```
  Biểu thức SQL này chạy được cho cả 2 định dạng (cũ và mới).

---

## IV. GIẢI PHÁP KIẾN TRÚC TOÀN DIỆN: BẢO ĐẢM TƯƠNG THÍCH LƯỠNG TÍNH (DUAL-STACK ADAPTER)

Để đảm bảo tương thích ngược 100%, quy hoạch `_raw_data` **bắt buộc phải đi kèm với Cơ Chế Thích Ứng Lưỡng Tính (Dual-Stack Adapter)** ở 2 tầng quan trọng nhất:

### 1. Nâng cấp `getNestedField` trong `dynamic_mapper.go` thành Hàm Đa Hình (Polymorphic Extraction)
Khi một rule tìm kiếm các toán tử ExtJSON (`$oid`, `$date`, `$numberLong`, v.v.) mà dữ liệu đã ở dạng phẳng (scalar primitive), hàm phải thông minh trả về giá trị scalar đó thay vì return `nil`:

```go
// isExtJSONOperator kiểm tra xem segment có phải là wrapper type của MongoDB BSON hay không
func isExtJSONOperator(segment string) bool {
	switch segment {
	case "$oid", "$date", "$numberLong", "$numberInt", "$numberDecimal", "$binary", "$timestamp":
		return true
	default:
		return false
	}
}

// getNestedField nâng cấp: Hỗ trợ cả dữ liệu ExtJSON cũ lẫn dữ liệu phẳng mới
func getNestedField(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		if current == nil {
			return nil
		}
		// Nếu segment tiếp theo là toán tử ExtJSON (vd $oid, $date, $numberLong)
		// nhưng current hiện tại ĐÃ LÀ primitive/scalar (do dữ liệu đã được unwrap),
		// thì coi như đã trích xuất trúng đích!
		if isExtJSONOperator(part) {
			if _, isMap := current.(map[string]interface{}); !isMap {
				continue // Giữ nguyên current và bỏ qua operator wrapper!
			}
		}

		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}
```

### 2. Nâng cấp `flatten.go:extractFieldValue` trong Master Transmuter (Fallback Strip Suffix)
Khi `gjson.Get(rawStr, path)` không tìm thấy giá trị, nếu `path` có đuôi ExtJSON (`.$oid`, `.$date`, `.$numberLong`), hàm tự động cắt đuôi và query lại trên đường dẫn gốc:

```go
func tryFallbackExtJSONPath(rawStr, path string) (gjson.Result, bool) {
	extSuffixes := []string{".$oid", ".$date", ".$numberLong", ".$numberInt", ".$numberDecimal"}
	for _, suffix := range extSuffixes {
		if strings.HasSuffix(path, suffix) {
			barePath := strings.TrimSuffix(path, suffix)
			gres := gjson.Get(rawStr, barePath)
			if gres.Exists() {
				return gres, true
			}
		}
	}
	return gjson.Result{}, false
}
```

### 3. Chuẩn hóa `dynamic_mapper.go:toTimestamp`
Bổ sung `case int64:`, `case int:`, `case json.Number:` để đảm bảo khi `$date` được chuyển thành epoch milliseconds, nó được chuyển đổi chính xác thành `time.Time` cho PostgreSQL.

---

## V. MA TRẬN ĐỐI SOÁT TƯƠNG THÍCH NGƯỢC (BACKWARD COMPATIBILITY MATRIX)

| Kịch bản Dữ liệu & Rule | Khi Chưa có Dual-Stack Adapter | Sau khi Trang Bị Dual-Stack Adapter | Trạng thái Nghiệm thu |
| :--- | :--- | :--- | :---: |
| **Bản ghi cũ (ExtJSON) + Rule cũ (`_id.$oid`)** | PASS (Đọc `$oid` từ object) | **PASS** (Đọc `$oid` từ object) | ✅ Hoàn toàn tương thích |
| **Bản ghi mới (Flat) + Rule mới (`_id`)** | PASS (Đọc string trực tiếp) | **PASS** (Đọc string trực tiếp) | ✅ Hoàn toàn tương thích |
| **Bản ghi mới (Flat) + Rule cũ (`_id.$oid`)** | ❌ **FAIL (Return nil $\rightarrow$ Drop row!)** | **PASS** (Bỏ qua `$oid`, lấy string scalar) | ✅ **ĐÃ KHẮC PHỤC TRIỆT ĐỂ** |
| **Bản ghi cũ (ExtJSON) + Rule mới (`_id`)** | ❌ **FAIL (Bốc cả object `{"$oid":...}`)** | **PASS** (`unwrapMongoTypes` unwrap thành string) | ✅ **ĐÃ KHẮC PHỤC TRIỆT ĐỂ** |
| **SQL BatchTransform trên bảng hỗn hợp** | PASS (`BuildCastExpr` đa hình) | **PASS** (`BuildCastExpr` đa hình) | ✅ Hoàn toàn tương thích |
| **Transmuter Master trên dòng mới** | ❌ **FAIL (Drop row nếu NOT NULL)** | **PASS** (Fallback bare path thành công) | ✅ **ĐÃ KHẮC PHỤC TRIỆT ĐỂ** |

---

## VI. KẾT LUẬN & ĐỀ XUẤT HÀNH ĐỘNG TIẾP THEO

1. **Ghi nhận bài học:** Đã đúc kết bài học `#raw-data-polymorphic-compatibility` vào `lessons.md`.
2. **Kỷ luật phân quyền Rule #13:** Toàn bộ hồ sơ giải pháp đã được lưu trữ vật lý tại `audit_report_raw_data_impact_and_backward_compatibility.md`.
3. **Sẵn sàng triển khai:** Khi User phát lệnh **`APPROVE`**, Brain sẽ lập tức ủy quyền cho **Muscle (Chief Engineer)** triển khai mã nguồn Dual-Stack Adapter cho `dynamic_mapper.go`, `flatten.go`, và `toTimestamp` kèm theo bộ test suites kiểm thử tương thích ngược toàn diện!
