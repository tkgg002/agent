# BÁO CÁO AUDIT PHẢN BIỆN CHUYÊN SÂU & TIẾN TRÌNH QC TOÀN TRÌNH (COMPREHENSIVE ADVERSARIAL AUDIT & GOVERNANCE REPORT)
**Dự án:** Data Hub (Centralized Data Service, CDC CMS Web, CDC CMS Service)  
**Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`  
**Thời gian lập:** 2026-09-16T16:15:00+07:00  
**Thực hiện bởi:** Brain (Chairman & Architect)  
**Tiêu chuẩn:** Hiến pháp `GEMINI.md`, Kỷ luật `lessons.md`, Quy tắc DoD (G1–G8), Không suy diễn, Không báo cáo láo.

---

## MỤC LỤC
1. [BỐI CẢNH & MỤC TIÊU TIẾN TRÌNH QC](#1-bối-cảnh--mục-tiêu-tiến-trình-qc)
2. [KIỂM ĐỊNH TÍNH CHÂN THỰC & CHỐNG BÁO CÁO LÁO (PHYSICAL ON-DISK VERIFICATION)](#2-kiểm-định-tính-chân-thực--chống-báo-cáo-láo)
3. [AUDIT CHI TIẾT TỪNG TASK, TỪNG TỆP TIN & TỪNG DÒNG MÃ NGUỒN (LINE-BY-LINE AUDIT)](#3-audit-chi-tiết-từng-task-từng-tệp-tin--từng-dòng-mã-nguồn)
4. [ĐỐI CHIẾU VỚI LOGIC KẾ HOẠCH & PHÁT HIỆN LỖ HỔNG TIỀM ẨN (GAPS & VULNERABILITIES)](#4-đối-chiếu-với-logic-kế-hoạch--phát-hiện-lỗ-hổng-tiềm-ẩn)
5. [VÒNG LẶP PHẢN TỈNH & BÀI HỌC KINH NGHIỆM (SELF-IMPROVEMENT LOOP)](#5-vòng-lặp-phản-tỉnh--bài-học-kinh-nghiệm)
6. [HỒ SƠ GIẢI PHÁP KỸ THUẬT VÁ DỨT ĐIỂM (CONCRETE CODE SOLUTIONS)](#6-hồ-sơ-giải-pháp-kỹ-thuật-vá-dứt-điểm)
7. [KẾ HOẠCH UỶ QUYỀN THỰC THI (RULE #13 COMPLIANCE)](#7-kế-hoạch-uỷ-quyền-thực-thi)

---

## 1. BỐI CẢNH & MỤC TIÊU TIẾN TRÌNH QC

Hệ thống Data Hub đang đối mặt với bài toán mở rộng quy mô lớn:
- **Tầng Source (Nguồn):** Hỗ trợ nhiều kết nối khác nhau (MongoDB, PostgreSQL, MariaDB, MySQL, SFTP) có thể trùng tên `(database, schema, table/collection)`. Yêu cầu dữ liệu từ kết nối nào phải được định tuyến chính xác tuyệt đối vào bảng Shadow của kết nối đó, không được ghi đè hay rò rỉ chéo.
- **Tầng Master (Đích):** Hỗ trợ nhiều kết nối Master độc lập (PostgreSQL 1, PostgreSQL 2) có thể trùng tên bảng Master `(master_schema, master_table)`. Yêu cầu mọi thao tác DDL, Transmute, Realtime Fanout, và Recon phải định danh chính xác theo `master_binding_id`.

**Mục tiêu của tiến trình QC gắt gao này:**
1. Rà soát lại toàn bộ "quá trình" thực hiện vừa qua từ đầu đến cuối.
2. Kiểm tra từng file, từng dòng code đã sửa xem có sai, thiếu sót gì so với Plan và so với kiến trúc Core Systems của dự án hay không.
3. Vạch trần mọi suy diễn, phỏng đoán hoặc báo cáo sai sự thật.
4. Áp dụng Vòng Lặp Phản Tỉnh (Self-Improvement Loop) và đưa ra giải pháp sửa chữa triệt để nhất.

---

## 2. KIỂM ĐỊNH TÍNH CHÂN THỰC & CHỐNG BÁO CÁO LÁO

Tiến hành đọc trực tiếp từ hệ thống tệp tin vật lý để đối soát 14 tệp tin đã được thay đổi:

| STT | Tệp tin mã nguồn | Trạng thái vật lý | Nội dung kiểm tra thực tế trên đĩa | Kết luận |
| :---: | :--- | :---: | :--- | :---: |
| 1 | `cdc-cms-web/src/pages/SourceConnectors.tsx` | **TỒN TẠI** | Dòng 480-494: gán `${TOPIC_PREFIX_*}.${name}` cho `sftp`, `mongodb`, `mysql`, `postgresql`. | **ĐÚNG SỰ THẬT** |
| 2 | `centralized-data-service/.../cdc_event.go` | **TỒN TẠI** | Dòng 11: `SourceSchema string json:"source_schema,omitempty"`. | **ĐÚNG SỰ THẬT** |
| 3 | `centralized-data-service/.../kafka_consumer.go` | **TỒN TẠI** | Dòng 652-695: trích xuất `sourceSchema`, mở rộng `isGeneric`, bóc tách 5-parts/4-parts topic. | **ĐÚNG SỰ THẬT** |
| 4 | `centralized-data-service/.../event_handler.go` | **TỒN TẠI** | Dòng 167-270: bóc tách `sourceSchema`, qualify `schema.table` và fallback tra cứu. | **ĐÚNG SỰ THẬT** |
| 5 | `centralized-data-service/.../metadata_registry_utils.go` | **TỒN TẠI** | Dòng 181, 188: thêm key `c:sourceDB.sourceTable` và fallback `sourceDB.sourceTable`. | **ĐÚNG SỰ THẬT** |
| 6 | `centralized-data-service/.../metadata_registry_service.go` | **TỒN TẠI** | Dòng 587-593: `return nil` khi `conn != ""` mà `len(filtered) == 0`. | **ĐÚNG SỰ THẬT** |
| 7 | `centralized-data-service/.../metadata_registry_service_test.go` | **TỒN TẠI** | Dòng 260-468: 3 test suites mới (PG multi-conn, Maria multi-conn, Negative test). | **ĐÚNG SỰ THẬT** |
| 8 | `centralized-data-service/.../dynamic_mapper.go` | **TỒN TẠI** | Dòng 71-75, 402-464: `normalizeMongoExtJSONWithDepth` với guard `depth > 32`. | **ĐÚNG SỰ THẬT** |
| 9 | `centralized-data-service/.../bridge_handler.go` | **TỒN TẠI** | Dòng 262-275: truyền `connectorName` vào `ResolveSourceRoutes`. | **ĐÚNG SỰ THẬT** |
| 10 | `centralized-data-service/.../snapshot_runner_utils.go` | **TỒN TẠI** | Dòng 144-155: `buildSnapshotEnvelope` đóng gói `source_conn`. | **ĐÚNG SỰ THẬT** |
| 11 | `centralized-data-service/.../snapshot_runner_handler.go` | **TỒN TẠI** | Dòng 511, 795: truyền `conn.ConnectionCode`. | **ĐÚNG SỰ THẬT** |
| 12 | `centralized-data-service/.../metadata_registry.go` | **TỒN TẠI** | Dòng 18-42: mở rộng interface variadic `sourceConn ...string`. | **ĐÚNG SỰ THẬT** |
| 13 | `centralized-data-service/.../registry_service.go` | **TỒN TẠI** | Dòng 29-41: đồng bộ interface signature. | **ĐÚNG SỰ THẬT** |
| 14 | `centralized-data-service/.../snapshot_runner_test.go` | **TỒN TẠI** | Dòng 65-85: mock struct đồng bộ interface signature. | **ĐÚNG SỰ THẬT** |

👉 **Khẳng định kiểm định:** Không có hành vi báo cáo láo (hallucination). Tất cả các file và dòng code báo cáo đều là code thật tồn tại trên ổ cứng.

---

## 3. AUDIT CHI TIẾT TỪNG TASK, TỪNG TỆP TIN & TỪNG DÒNG MÃ NGUỒN

### Task 1: Phân lập Topic Prefix trên giao diện CMS Web
- **Tệp tin:** [`SourceConnectors.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx#L480-L494)
- **Từng dòng mã nguồn:**
  ```typescript
  480:   useEffect(() => {
  481:     if (!editorOpen || editorMode !== 'create') return;
  482:     const name = slugifyForShadow(String(connectorNameValue || 'connector'));
  483:     if (dbKind === 'sftp') {
  484:       form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_SFTP}.${name}`);
  485:     } else if (dbKind === 'mongodb') {
  486:       form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
  487:     } else if (dbKind === 'mysql') {
  488:       form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MYSQL}.${name}`);
  489:     } else if (dbKind === 'postgresql') {
  490:       form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_POSTGRESQL}.${name}`);
  491:     }
  492:   }, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
  ```
- **Đánh giá phản biện:**
  - *Ưu điểm:* Tự động gắn connector name vào topic prefix khi tạo mới connector, triệt tiêu 100% việc 2 connector cùng loại ghi chung topic Kafka broker.
  - *Tác động biên:* Chỉ kích hoạt khi `editorMode === 'create'`, không ghi đè cấu hình topic của các connector hiện có (an toàn, Minimal Impact).

---

### Task 2: Mang Schema nguồn xuyên suốt Pipeline
- **Tệp tin:** [`cdc_event.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/shadow/cdc_event.go#L8-L13)
- **Từng dòng mã nguồn:**
  ```go
  8: 	Source          interface{}  `json:"source"`
  9: 	SourceConn      string       `json:"source_conn,omitempty"`
  10: 	SourceDB        string       `json:"source_db,omitempty"`
  11: 	SourceSchema    string       `json:"source_schema,omitempty"`
  12: 	SourceTable     string       `json:"source_table,omitempty"`
  ```
- **Đánh giá phản biện:** Chuẩn CloudEvent model, bảo toàn thông tin PostgreSQL schema (`public`, `finance`, v.v.).

---

### Task 3: Bóc tách Namespace & Connection tại Kafka Consumer
- **Tệp tin:** [`kafka_consumer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L652-L695)
- **Từng dòng mã nguồn:**
  ```go
  652: 	var sourceConnCode, sourceDB, sourceSchema, sourceTable string
  653: 	if sm, ok := sourceRaw.(map[string]interface{}); ok {
  654: 		if n, ok := sm["name"].(string); ok {
  655: 			sourceConnCode = strings.TrimSpace(n)
  656: 		}
  657: 		if d, ok := sm["db"].(string); ok {
  658: 			sourceDB = strings.TrimSpace(d)
  659: 		}
  660: 		if s, ok := sm["schema"].(string); ok {
  661: 			sourceSchema = strings.TrimSpace(s)
  662: 		}
  663: 		if t, ok := sm["table"].(string); ok {
  664: 			sourceTable = strings.TrimSpace(t)
  665: 		} else if c, ok := sm["collection"].(string); ok {
  666: 			sourceTable = strings.TrimSpace(c)
  667: 		}
  668: 	}
  669: 
  670: 	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.")
  671: 	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")
  672: 
  673: 	isGeneric := sourceConnCode == "" ||
  674: 		sourceConnCode == "mongodb" ||
  675: 		sourceConnCode == "postgres" ||
  676: 		sourceConnCode == "postgresql" ||
  677: 		sourceConnCode == "mysql" ||
  678: 		sourceConnCode == "mariadb" ||
  679: 		sourceConnCode == "gpay"
  680: 
  681: 	if isGeneric {
  682: 		parts := strings.Split(msg.Topic, ".")
  683: 		if len(parts) >= 5 && parts[0] == "cdc" {
  684: 			sourceConnCode = parts[2]
  685: 		} else if len(parts) == 4 && parts[0] == "cdc" && parts[1] != "goopay" && parts[1] != "gpay" {
  686: 			sourceConnCode = parts[1]
  687: 		}
  688: 	}
  ```
- **Đánh giá phản biện (PHÁT HIỆN SƠ HỞ):**
  - **Sơ hở A:** Ở dòng 673-679, biến `isGeneric` thiếu `"goopay"`. Khi `sourceRaw["name"] = "cdc.goopay"`, sau `TrimPrefix("cdc.")` ta được `"goopay"`. Vì không có trong `isGeneric`, code bỏ qua không bóc tách topic `msg.Topic`!
  - **Sơ hở B:** Nếu topic là 4-parts cổ điển (ví dụ `cdc.mariadb.<db>.<table>`), dòng 686 gán `sourceConnCode = "mariadb"`. Chuỗi `"mariadb"` này là tên engine chứ không phải connector name. Truyền chuỗi này xuống tầng dưới sẽ gây lỗi over-filtering!

---

### Task 4: Xử lý Namespace & Fallback tại Event Handler
- **Tệp tin:** [`event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L240-L270)
- **Từng dòng mã nguồn:**
  ```go
  248: 	if sourceSchema != "" && sourceSchema != "public" && !strings.Contains(table, ".") {
  249: 		table = sourceSchema + "." + table
  250: 	}
  ...
  264: 	routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)
  265: 	if len(routes) == 0 && event.SourceSchema != "" && !strings.Contains(sourceTable, ".") {
  266: 		qualified := event.SourceSchema + "." + sourceTable
  267: 		routes = h.registrySvc.ResolveSourceRoutes(sourceDB, qualified, sourceConn)
  268: 	}
  ```
- **Đánh giá phản biện (PHÁT HIỆN SƠ HỞ):**
  - Dòng 220-229: Khối code bóc tách connector name từ Subject bị đặt lồng trong `if db == "" || table == ""`. Khi `db` và `table` đã có trong payload nhưng `sourceConn` bị rỗng, hệ thống không bóc tách Subject!

---

### Task 5: Chuẩn hóa Lookup Keys
- **Tệp tin:** [`metadata_registry_utils.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_utils.go#L180-L192)
- **Từng dòng mã nguồn:**
  ```go
  178: 		for _, c := range candidates {
  179: 			keys = append(keys,
  180: 				fmt.Sprintf("%s:%s|%s", c, sourceDB, sourceTable),
  181: 				fmt.Sprintf("%s:%s.%s", c, sourceDB, sourceTable),
  182: 				fmt.Sprintf("%s:%s", c, sourceTable),
  183: 			)
  184: 		}
  ...
  187: 	keys = append(keys,
  188: 		fmt.Sprintf("%s|%s", sourceDB, sourceTable),
  189: 		fmt.Sprintf("%s:%s", sourceDB, sourceTable),
  190: 		fmt.Sprintf("%s.%s", sourceDB, sourceTable),
  191: 		sourceTable,
  192: 	)
  ```
- **Đánh giá phản biện:** Bổ sung chính xác định dạng `schema.table` cho PostgreSQL.

---

### Task 6: Vá Logic Leak tại Registry Service
- **Tệp tin:** [`metadata_registry_service.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go#L587-L593)
- **Từng dòng mã nguồn:**
  ```go
  587: 		if len(filtered) > 0 {
  588: 			return filtered
  589: 		}
  590: 		// BẮT BUỘC: Caller đòi đích danh connection nhưng không có route khớp -> Trả về nil!
  591: 		// CẤM tuyệt đối fallback xuống trả về routes của connection khác!
  592: 		return nil
  593: 	}
  594: 	return routes
  ```
- **Đánh giá phản biện (PHÁT HIỆN SƠ HỞ CỰC KỲ NGUY HIỂM):**
  - Khi `conn != ""`, nếu `conn` là một chuỗi generic (`"goopay"`, `"mariadb"`, `"gpay"` do các topic 4-parts gửi tới), hàm sẽ tìm route có `SourceConnectionKey == "goopay"`.
  - Trong DB không có connection nào tên là `"goopay"`, dẫn đến `len(filtered) == 0`.
  - Hàm thực thi dòng 592: `return nil`!
  - **Hậu quả:** Toàn bộ traffic của các topic cổ điển 4-parts bị DROP 100%!

---

### Task 7: Unwrap MongoDB Extended JSON tại Dynamic Mapper
- **Tệp tin:** [`dynamic_mapper.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go#L70-L75, #L402-L464)
- **Từng dòng mã nguồn:**
  ```go
  71: 	normalizedRaw := rawData
  72: 	if nMap, ok := normalizeMongoExtJSON(rawData).(map[string]interface{}); ok {
  73: 		normalizedRaw = nMap
  74: 	}
  ...
  413: 		if dateVal, ok := val["$date"]; ok && len(val) == 1 {
  414: 			switch dt := dateVal.(type) {
  415: 			case float64:
  416: 				return int64(dt)
  417: 			case string:
  418: 				if t, err := time.Parse(time.RFC3339, dt); err == nil {
  419: 					return t.UnixMilli()
  420: 				}
  421: 				return dt
  ...
  ```
- **Đánh giá phản biện (PHÁT HIỆN LỖ HỔNG TYPE MISMATCH POSTGRESQL):**
  - `normalizeMongoExtJSON` unwrap `$date` thành số nguyên epoch **`int64`** (milliseconds).
  - Nhưng hàm `toTimestamp(val)` (dòng 296-330) của `dynamic_mapper.go` chỉ có `case float64:` và `case string:`, **KHÔNG CÓ `case int64:`**.
  - Kết quả: Giá trị rơi vào `default` và giữ nguyên kiểu `int64`.
  - Khi đưa vào câu lệnh SQL INSERT của PostgreSQL cho cột `TIMESTAMPTZ`, PostgreSQL báo lỗi:  
    `ERROR: column ... is of type timestamp with time zone but expression is of type bigint`.

---

## 4. ĐỐI CHIẾU VỚI LOGIC KẾ HOẠCH & PHÁT HIỆN LỖ HỔNG TIỀM ẨN

Qua quá trình QC phản biện chuyên sâu, Brain chỉ ra **3 LỖ HỔNG HỆ THỐNG CẦN KHẮC PHỤC NGAY**:

| Mã Lỗ Hổng | Tên Lỗ Hổng | Tệp tin bị ảnh hưởng | Hậu quả kỹ thuật nếu không fix | Giải pháp triệt để |
| :---: | :--- | :--- | :--- | :--- |
| **GAP-01** | Type Mismatch trong `toTimestamp` | `internal/service/shadow/dynamic_mapper.go` | PostgreSQL từ chối ghi dữ liệu với lỗi type mismatch khi cột là TIMESTAMP/TIMESTAMPTZ. | Bổ sung `case int64:`, `case int:`, `case json.Number:` vào `toTimestamp`. |
| **GAP-02** | Over-filtering làm Drop Traffic Topic Cổ Điển | `internal/service/source/metadata_registry_service.go` & `kafka_consumer.go` | Các topic Debezium 4-parts cổ điển (`cdc.goopay`, `cdc.mariadb`) bị coi là connection riêng và bị trả về `nil` (Drop 100% data). | Tạo hàm `IsGenericConnectionCode()`; nếu generic thì reset về `""` để kích hoạt fallback trả về tất cả routes. |
| **GAP-03** | Bị chặn bóc tách Subject Fallback | `internal/handler/shadow/event_handler.go` | Khi payload có `db/table` nhưng mất connection, subject fallback 5-parts không được chạy. | Độc lập hóa khối trích xuất subject connection name. |

---

## 5. VÒNG LẶP PHẢN TỈNH & BÀI HỌC KINH NGHIỆM (SELF-IMPROVEMENT LOOP)

### Root Cause Analysis (Phân tích Nguyên nhân Gốc rễ):
1. **Bệnh Tunnel Vision chuỗi kiểu dữ liệu (Data Pipeline Type Cascade):** Khi sửa đổi format dữ liệu ở tầng tiền xử lý (unwrap sang `int64`), ta đã không đối soát đến tầng chuyển đổi kiểu của adapter (`toTimestamp`). Đây là bài học xương máu về việc phải lần theo toàn bộ hành trình của dữ liệu (data lifecycle tracing).
2. **Bệnh Over-filtering:** Khi áp dụng tư duy "Zero-Trust" cho định tuyến (`return nil` khi không tìm thấy kết nối), ta đã quên mất bài toán tương thích ngược (Backward Compatibility) đối với các topic cổ điển chưa có connector prefix riêng.

### Đã ghi nhận bài học vào `lessons.md`:
- `### [2026-09-16] Unhandled Type Cascade khi unwrap Extended JSON và Over-filtering làm drop traffic topic cổ điển (#type-cascade-crash, #generic-prefix-overfiltering)`

---

## 6. HỒ SƠ GIẢI PHÁP KỸ THUẬT VÁ DỨT ĐIỂM (CONCRETE CODE SOLUTIONS)

### Fix 1: Sửa `centralized-data-service/internal/service/shadow/dynamic_mapper.go`
Tại hàm `toTimestamp(val interface{}) (interface{}, error)` (dòng 296), bổ sung hỗ trợ `int64`, `int`, `json.Number`:
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

### Fix 2: Bổ sung Helper chuẩn trong `centralized-data-service/internal/service/source/metadata_registry_utils.go`
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

### Fix 3: Sửa `centralized-data-service/internal/service/source/metadata_registry_service.go`
Trong `ResolveSourceRoutes`:
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

### Fix 4: Sửa `centralized-data-service/internal/handler/shadow/kafka_consumer.go`
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

### Fix 5: Sửa `centralized-data-service/internal/handler/shadow/event_handler.go`
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

## 7. KẾ HOẠCH UỶ QUYỀN THỰC THI (RULE #13 COMPLIANCE)

Tuân thủ nghiêm ngặt **Rule #13 (Brain Code Prohibition)**:
1. Brain không tự ý chỉnh sửa file mã nguồn `.go`.
2. Toàn bộ phát hiện và code demo đã được trình bày minh bạch trong báo cáo này.
3. Ngay khi User phát lệnh **`APPROVE`**, Brain sẽ ủy quyền cho **Muscle (Chief Engineer)** áp dụng các chỉnh sửa này và chạy unit tests kiểm thử toàn trình.
