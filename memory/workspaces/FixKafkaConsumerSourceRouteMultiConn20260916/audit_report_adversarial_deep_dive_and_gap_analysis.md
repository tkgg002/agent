# BIÊN BẢN AUDIT PHẢN BIỆN CHUYÊN SÂU & TIẾN TRÌNH QC GẮT GAO
## TOÀN BỘ QUÁ TRÌNH THỰC THI, CÁC FILE ĐÃ SỬA VÀ TỪNG DÒNG MÃ NGUỒN (LINE-BY-LINE ADVERSARIAL AUDIT)

**Thời gian:** 2026-09-16T16:05:00+07:00  
**Thực hiện bởi:** Brain (Chairman & Architect)  
**Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`  
**Mục tiêu:** Áp dụng tư duy phản biện (Adversarial Review) đối soát 100% từng dòng mã nguồn, kiểm tra sai sót/thiếu sót so với Plan, phát hiện hành vi suy diễn/báo cáo láo, kích hoạt Vòng Lặp Phản Tỉnh (Self-Improvement Loop) và xuất bản giải pháp khắc phục triệt để.

---

## I. KIỂM ĐỊNH BÁO CÁO LÁO & SUY DIỄN (FALSE REPORT & ASSUMPTION AUDIT)

### 1. Kiểm định tính chân thực của mã nguồn trên đĩa (Physical Verification)
- **Đã kiểm tra 14/14 file** được đề cập trong toàn bộ quá trình:
  1. `cdc-cms-web/src/pages/SourceConnectors.tsx` (Dòng 480-494): **TỒN TẠI THẬT TRÊN ĐĨA**. Phân lập prefix `${name}` cho cả `sftp`, `mongodb`, `mysql`, `postgresql`.
  2. `centralized-data-service/internal/model/shadow/cdc_event.go` (Dòng 11): **TỒN TẠI THẬT TRÊN ĐĨA**. Đã có `SourceSchema string`.
  3. `centralized-data-service/internal/handler/shadow/kafka_consumer.go` (Dòng 652-695): **TỒN TẠI THẬT TRÊN ĐĨA**. Trích xuất `sourceSchema`, mở rộng `isGeneric`, bóc tách 5-parts/4-parts.
  4. `centralized-data-service/internal/handler/shadow/event_handler.go` (Dòng 167-270): **TỒN TẠI THẬT TRÊN ĐĨA**. Qualify `sourceSchema + "." + table` và fallback tra cứu.
  5. `centralized-data-service/internal/service/source/metadata_registry_utils.go` (Dòng 181, 188): **TỒN TẠI THẬT TRÊN ĐĨA**. Thêm `c:sourceDB.sourceTable` và `sourceDB.sourceTable`.
  6. `centralized-data-service/internal/service/source/metadata_registry_service.go` (Dòng 587-593): **TỒN TẠI THẬT TRÊN ĐĨA**. Sửa `return nil` khi `conn != ""` mà `len(filtered) == 0`.
  7. `centralized-data-service/internal/service/source/metadata_registry_service_test.go` (Dòng 260-468): **TỒN TẠI THẬT TRÊN ĐĨA**. Đã viết 3 test suites mới.
  8. `centralized-data-service/internal/service/shadow/dynamic_mapper.go` (Dòng 71-75, 402-464): **TỒN TẠI THẬT TRÊN ĐĨA**. Đã thêm `normalizeMongoExtJSONWithDepth`.
  9. `centralized-data-service/internal/handler/source/bridge_handler.go`: **TỒN TẠI THẬT TRÊN ĐĨA**. Truyền `connectorName`.
  10. `centralized-data-service/internal/handler/orchestration/snapshot_runner_utils.go`: **TỒN TẠI THẬT TRÊN ĐĨA**. Bổ sung `source_conn`.
  11. `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go`: **TỒN TẠI THẬT TRÊN ĐĨA**. Bổ sung `conn.ConnectionCode`.
  12. `centralized-data-service/internal/service/metadata/metadata_registry.go`: **TỒN TẠI THẬT TRÊN ĐĨA**.
  13. `centralized-data-service/internal/service/source/registry_service.go`: **TỒN TẠI THẬT TRÊN ĐĨA**.
  14. `centralized-data-service/internal/handler/orchestration/snapshot_runner_test.go`: **TỒN TẠI THẬT TRÊN ĐĨA**.

**Kết luận Kiểm định Chân thực:** Không có hành vi báo cáo khống việc sửa file. Mọi thay đổi vật lý đều có mặt trên ổ đĩa.

---

## II. ĐỐI SOÁT PHẢN BIỆN TỪNG DÒNG (ADVERSARIAL LINE-BY-LINE AUDIT) & PHÁT HIỆN LỖ HỔNG TIỀM ẨN

Mặc dù code đã được sửa đúng theo mô tả bề mặt, khi Brain tiến hành **Adversarial Execution Simulation (Mô phỏng thực thi phản biện)** đã phát hiện ra **3 LỖ HỔNG NGHIÊM TRỌNG (CRITICAL VULNERABILITIES)**:

### 1. LỖ HỔNG CHÍ MẠNG 1: Type Mismatch tại `dynamic_mapper.go:toTimestamp` làm crash câu lệnh SQL PostgreSQL
- **Vị trí:** [`centralized-data-service/internal/service/shadow/dynamic_mapper.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go#L296-L330)
- **Cơ chế phát sinh lỗi:**
  1. Ở đầu hàm `MapData`, ta thêm `normalizeMongoExtJSON(rawData)` để chuẩn hóa MongoDB Extended JSON.
  2. Trong `normalizeMongoExtJSONWithDepth`: Khi gặp `{"$date": 1770102689256}` hoặc `{"$date": "..."}`, hàm unwrap trả về giá trị kiểu **`int64`** (epoch milliseconds) hoặc `string`.
  3. Khi xử lý mapping rule cho cột có kiểu `dataType = "TIMESTAMP"` hoặc `"TIMESTAMPTZ"`:
     `convertType(val, "TIMESTAMP")` được gọi, và nó ủy quyền cho `toTimestamp(val)`.
  4. Xem xét kỹ mã nguồn hàm `toTimestamp(val interface{})`:
     ```go
     func toTimestamp(val interface{}) (interface{}, error) {
     	switch v := val.(type) {
     	case string:
     		...
     	case float64:
     		if v > 1e12 {
     			return time.UnixMilli(int64(v)), nil
     		}
     		return time.Unix(int64(v), 0), nil
     	case map[string]interface{}:
     		...
     	default:
     		return val, nil
     	}
     }
     ```
  5. **TỬ HUYỆT:** `toTimestamp` chỉ có `case float64:`, **HOÀN TOÀN KHÔNG CÓ `case int64:` hay `case int:`**!
  6. Vì `val` bây giờ là `int64` (do `normalizeMongoExtJSON` trả về), nó rơi thẳng vào `default: return val, nil` $\rightarrow$ Trả về chính giá trị nguyên thủy `int64(1770102689256)`.
  7. Khi pgx/GORM thực thi câu lệnh SQL INSERT/UPDATE vào PostgreSQL cho cột `TIMESTAMPTZ`:
     PostgreSQL lập tức ném lỗi:
     `ERROR: column "created_at" is of type timestamp with time zone but expression is of type bigint (SQLSTATE 42804)`!
  8. **Mức độ nghiêm trọng:** CỰC KỲ CAO. Mọi bản ghi Snapshot V2 hoặc CDC có trường ngày tháng map sang cột TIMESTAMP đều sẽ bị lỗi type mismatch!

---

### 2. LỖ HỔNG CHÍ MẠNG 2: Over-filtering làm DROP 100% Traffic của các Topic cổ điển (Legacy 4-Parts Topics)
- **Vị trí:**
  - [`kafka_consumer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L673-L688)
  - [`event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L240-L242)
  - [`metadata_registry_service.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go#L587-L593)
- **Cơ chế phát sinh lỗi:**
  1. Trong `kafka_consumer.go`:
     ```go
     	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.")
     	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")

     	isGeneric := sourceConnCode == "" ||
     		sourceConnCode == "mongodb" ||
     		sourceConnCode == "postgres" ||
     		sourceConnCode == "postgresql" ||
     		sourceConnCode == "mysql" ||
     		sourceConnCode == "mariadb" ||
     		sourceConnCode == "gpay"
     ```
     **Phát hiện 2.1:** Tiền tố `"goopay"` bị **bỏ sót** trong danh sách `isGeneric`!
     Nếu connector Debezium gửi `sourceRaw["name"] = "cdc.goopay"`, sau khi cắt `cdc.`, `sourceConnCode` bằng `"goopay"`. Do `"goopay"` không có trong `isGeneric`, biến `isGeneric = false`. Hệ thống KHÔNG kiểm tra topic `msg.Topic` để lấy `parts[2]`!
  2. **Phát hiện 2.2 (Over-filtering Leak):**
     Nếu là một topic cổ điển 4-parts (chưa phân lập topic prefix theo connector, ví dụ `cdc.goopay.<db>.<coll>` hoặc `cdc.mariadb.<db>.<table>` hoặc `cdc.gpay.<schema>.<table>`):
     - `sourceConnCode` sẽ mang giá trị generic: `"goopay"`, `"mariadb"`, hoặc `"gpay"`.
     - Giá trị này được đóng gói vào `cdcEvent["source_conn"] = "goopay"`.
     - Tại `event_handler.go`: `sourceConn` được gán bằng `"goopay"`.
     - Tại `metadata_registry_service.go`: Hàm `ResolveSourceRoutes(db, coll, "goopay")` được gọi với `conn = "goopay"`.
     - Lớp lọc mới kiểm tra: `r.SourceConnectionKey == "goopay"`.
     - Trong cơ sở dữ liệu `connection_registry`, **KHÔNG CÓ KẾT NỐI NÀO** mang mã `connection_code = "goopay"`.
     - `len(filtered) == 0`.
     - Khối code mới vá trong turn trước kích hoạt:
       ```go
       if len(filtered) > 0 {
           return filtered
       }
       return nil // <--- BỊ TRẢ VỀ NIL!
       ```
     - **HẬU QUẢ:** Toàn bộ dữ liệu của các topic 4-parts cổ điển (chưa nâng cấp prefix riêng) sẽ bị `ResolveSourceRoutes` trả về `nil` và bị `event_handler` vứt bỏ (DROP 100%)!
  3. **Mức độ nghiêm trọng:** CỰC KỲ NGUY HIỂM vì phá vỡ Backward Compatibility.

---

### 3. LỖ HỔNG TIỀM ẨN 3: Block Subject Fallback do ràng buộc `db == "" || table == ""` trong `event_handler.go`
- **Vị trí:** [`centralized-data-service/internal/handler/shadow/event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L219-L228)
- **Cơ chế phát sinh lỗi:**
  - Đoạn code fallback bóc tách connector name từ Subject:
    ```go
    if db == "" || table == "" {
        parts := strings.Split(subject, ".")
        if len(parts) >= 4 {
            table = parts[len(parts)-1]
            db = parts[len(parts)-2]
            if len(parts) >= 5 && sourceConn == "" {
                sourceConn = parts[len(parts)-3]
            }
        }
    }
    ```
  - Nếu `db` và `table` đã được trích xuất thành công từ Debezium payload (`temp.Data.Source`), nhưng `temp.Data.Source.Name` bị rỗng hoặc là generic prefix, thì điều kiện `db == "" || table == ""` là FALSE!
  - Kết quả: Đoạn trích xuất `sourceConn = parts[len(parts)-3]` từ Subject KHÔNG BAO GIỜ ĐƯỢC CHẠY, khiến `sourceConn` bị mất dấu dù Subject có đầy đủ 5-parts (`cdc.goopay.<conn>.<db>.<coll>`).

---

## III. VÒNG LẶP PHẢN TỈNH & BÀI HỌC KINH NGHIỆM (SELF-IMPROVEMENT LOOP)

### 1. Phân tích Nguyên nhân Gốc rễ (Root Cause Analysis)
1. **Thiếu Tầm Nhìn Dữ Liệu Thực Tế (Data Flow Tunnel Vision):** Khi thêm một bước tiền xử lý unwrap Extended JSON (`normalizeMongoExtJSON`) ở tầng trên, ta đã không rà soát tầng chuyển đổi kiểu dữ liệu (`convertType` & `toTimestamp`) ở hạ lưu để xem kiểu dữ liệu đầu ra của tầng trên có được tầng dưới hỗ trợ hay không.
2. **Kỷ luật Bảo toàn Tính tương thích ngược (Backward Compatibility Discipline):** Khi áp dụng quy tắc cô lập nghiêm ngặt (`return nil` khi không tìm thấy kết nối), ta chưa phân biệt được giữa:
   - **Specific Connection Code:** Tên kết nối cụ thể (`traitestmongodevct`, `pg_conn_1`) $\rightarrow$ Phải cô lập 100%, không trùng khớp thì `return nil`.
   - **Generic Engine Prefix:** Tiền tố chung (`goopay`, `gpay`, `mariadb`, `mongodb`, `mysql`, `postgres`, `postgresql`, `sftp`) $\rightarrow$ Phải hiểu là "chưa chỉ định kết nối cụ thể", bắt buộc phải fallback trả về tất cả routes thay vì trả về `nil`.

### 2. Ghi nhận Bài học Tri thức (Lesson Promotion)
Đã đúc kết thành bài học chuẩn 5 phần theo Rule #6:
- `#unhandled-type-unwrapping-cascade`: Khi normalize Extended JSON sang số nguyên epoch (`int64`), hàm chuyển đổi timestamp phải hỗ trợ trực tiếp `int64`, `int`, `json.Number`.
- `#generic-engine-code-overfiltering`: Phải phân biệt rõ ràng giữa Generic Prefix và Connection Code thực sự trong mọi bộ lọc định tuyến.

---

## IV. GIẢI PHÁP KỸ THUẬT VÁ DỨT ĐIỂM (CONCRETE ARCHITECTURAL SOLUTIONS)

### 1. File: `internal/service/shadow/dynamic_mapper.go`
Bổ sung hỗ trợ `int64`, `int`, `json.Number` vào `toTimestamp`:
```go
func toTimestamp(val interface{}) (interface{}, error) {
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case int64:
		if v > 1e12 {
			return time.UnixMilli(v), nil
		}
		return time.Unix(v, 0), nil
	case int:
		v64 := int64(v)
		if v64 > 1e12 {
			return time.UnixMilli(v64), nil
		}
		return time.Unix(v64, 0), nil
	case json.Number:
		if n, err := v.Int64(); err == nil {
			if n > 1e12 {
				return time.UnixMilli(n), nil
			}
			return time.Unix(n, 0), nil
		}
		return nil, fmt.Errorf("cannot parse json.Number %q to timestamp", v.String())
	case string:
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				return t, nil
			}
		}
		return v, nil
	case float64:
		if v > 1e12 {
			return time.UnixMilli(int64(v)), nil
		}
		return time.Unix(int64(v), 0), nil
	case map[string]interface{}:
		if dateStr, ok := v["$date"].(string); ok {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				return t, nil
			}
		}
		return nil, fmt.Errorf("cannot parse date from map")
	default:
		return val, nil
	}
}
```

### 2. File: `internal/service/source/metadata_registry_utils.go`
Bổ sung hàm kiểm tra Generic Connection Code chuẩn hóa:
```go
// IsGenericConnectionCode kiểm tra xem chuỗi có phải là tiền tố chung/tên engine hay không
func IsGenericConnectionCode(code string) bool {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(code), "cdc.goopay."), "cdc."))
	switch clean {
	case "", "goopay", "gpay", "mongodb", "postgres", "postgresql", "mysql", "mariadb", "sftp":
		return true
	default:
		return false
	}
}
```

### 3. File: `internal/service/source/metadata_registry_service.go`
Chuẩn hóa bộ lọc `ResolveSourceRoutes`:
```go
	var conn string
	if len(sourceConn) > 0 {
		conn = strings.TrimSpace(sourceConn[0])
	}
	// Nếu conn là generic prefix (goopay, gpay, mariadb, mongodb...), reset về rỗng để fallback về tất cả routes
	if IsGenericConnectionCode(conn) {
		conn = ""
	}
	if conn != "" {
		cleanConn := strings.TrimPrefix(strings.TrimPrefix(conn, "cdc.goopay."), "cdc.")
		filtered := make([]*metadata.ResolvedSourceRoute, 0, len(routes))
		for _, r := range routes {
			if r != nil {
				rConn := r.SourceConnectionKey
				if rConn == conn || rConn == cleanConn {
					filtered = append(filtered, r)
				}
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
		// Caller chỉ định đích danh connection cụ thể nhưng không có route khớp -> Trả về nil!
		return nil
	}
	return routes
```

### 4. File: `internal/handler/shadow/kafka_consumer.go`
Mở rộng kiểm tra và reset generic code:
```go
	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.")
	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")

	if sourcemodel.IsGenericConnectionCode(sourceConnCode) {
		parts := strings.Split(msg.Topic, ".")
		if len(parts) >= 5 && parts[0] == "cdc" {
			sourceConnCode = parts[2]
		} else if len(parts) == 4 && parts[0] == "cdc" && !sourcemodel.IsGenericConnectionCode(parts[1]) {
			sourceConnCode = parts[1]
		}
	}

	// Nếu sau khi kiểm tra topic mà sourceConnCode vẫn là generic, reset về rỗng để không bị filter sai ở hạ lưu
	if sourcemodel.IsGenericConnectionCode(sourceConnCode) {
		sourceConnCode = ""
	}
```

### 5. File: `internal/handler/shadow/event_handler.go`
Bóc tách Subject độc lập và reset generic code:
```go
	if sourcemodel.IsGenericConnectionCode(sourceConn) {
		parts := strings.Split(subject, ".")
		if len(parts) >= 5 && !sourcemodel.IsGenericConnectionCode(parts[len(parts)-3]) {
			sourceConn = parts[len(parts)-3]
		} else {
			sourceConn = ""
		}
	}
```

---

## V. KẾ HOẠCH BÀN GIAO & UỶ QUYỀN THỰC THI (RULE #13 COMPLIANCE)

Tuân thủ nghiêm ngặt **Rule #13 (Brain Code Prohibition)**:
1. Brain chỉ lập bản phân tích, vạch rõ sai sót/thiếu sót, và thiết kế giải pháp chi tiết vào file này.
2. Tuyệt đối KHÔNG tự ý chỉnh sửa file mã nguồn `.go`.
3. Trình toàn bộ kết quả audit và giải pháp cho User phê duyệt.
4. Sau khi User phê duyệt, Brain sẽ uỷ quyền cho Role: **MUSCLE (Chief Engineer)** thực thi sửa đổi mã nguồn và chạy kiểm thử toàn trình.
