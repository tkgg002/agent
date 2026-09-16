# BÁO CÁO AUDIT PHẢN BIỆN CHI TIẾT TỪNG DÒNG MÃ NGUỒN (LINE-BY-LINE ADVERSARIAL AUDIT REPORT)
## KIỂM ĐỊNH TOÀN TRÌNH: ĐỊNH TUYẾN ĐA KẾT NỐI SOURCE & MASTER VÀ CHUẨN HÓA `_raw_data`

---
- **Thời gian thực hiện:** 2026-09-16T15:26:00+07:00
- **Thực thể thực hiện:** Brain (Chairman & Architect)
- **Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`
- **Tài liệu đối chiếu gốc:**
  1. `01_requirements_kafka_consumer_source_route.md`
  2. `08_tasks_kafka_consumer_source_route.md`
  3. `09_tasks_solution_kafka_consumer_source_route.md`
  4. `12_implementation_plan_kafka_consumer_source_route.md`
  5. `13_analysis_kafka_consumer_source_route.md`
  6. `implementation_plan.md`

---

## MỤC LỤC
1. [Đối Soát Tính Chân Thực Mã Nguồn (Anti-Speculation & Truth Verification)](#i-đối-soát-tính-chân-thực-mã-nguồn)
2. [Audit Chi Tiết Từng Task, Từng File, Từng Dòng Code Update](#ii-audit-chi-tiết-từng-task-từng-file-từng-dòng-code-update)
3. [Tư Duy Phản Biện: Phân Tích Lỗ Hổng & Điểm Lệch Kế Hoạch](#iii-tư-duy-phản-biện-phân-tích-lỗ-hổng--điểm-lệch-kế-hoạch)
4. [Rà Soát Tuân Thủ Kiến Trúc & Core Systems](#iv-rà-soát-tuân-thủ-kiến-trúc--core-systems)
5. [Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop)](#v-vòng-lặp-phản-tỉnh--khắc-phục-self-improvement-loop)

---

## I. ĐỐI SOÁT TÍNH CHÂN THỰC MÃ NGUỒN

Tiến trình QC độc lập đã đọc trực tiếp 14 tệp tin mã nguồn trên đĩa, đối chiếu snapshot vật lý:
- **Tất cả các tệp tin báo cáo đều tồn tại thực tế:** Không có tình trạng tạo file ảo hay báo cáo khống.
- **Tất cả các hàm, struct, interface được khai báo:** Đều có mặt trên mã nguồn và khớp chữ ký giữa các package (`model/shadow`, `service/metadata`, `service/source`, `handler/shadow`, `handler/source`, `handler/orchestration`, `service/shadow`).
- **Không có "cheat DB" hay workaround bẩn:** Không có câu lệnh SQL nào sửa data trực tiếp, không có hardcode tên bảng hay connection trong mã nguồn.

---

## II. AUDIT CHI TIẾT TỪNG TASK, TỪNG FILE, TỪNG DÒNG CODE UPDATE

### Task 1: Mở rộng Model CDCEvent để lưu trữ thông tin Connection nguồn
📁 **File:** `centralized-data-service/internal/model/shadow/cdc_event.go`
- **Vị trí dòng:** Dòng 9–11
- **Diff chi tiết:**
  ```go
  <<<< BEFORE:
  type CDCEvent struct {
  	SpecVersion string      `json:"specversion"`
  	ID          string      `json:"id"`
  	Source      interface{} `json:"source"`
  	Type        string      `json:"type"`
  ==== AFTER:
  type CDCEvent struct {
  	SpecVersion string      `json:"specversion"`
  	ID          string      `json:"id"`
  	Source      interface{} `json:"source"`
  	SourceConn  string      `json:"source_conn,omitempty"`
  	SourceDB    string      `json:"source_db,omitempty"`
  	SourceTable string      `json:"source_table,omitempty"`
  	Type        string      `json:"type"`
  >>>>
  ```
- **Phân tích từng dòng:**
  + Dòng 9: `SourceConn string `json:"source_conn,omitempty"``: Lưu mã định danh connection nguồn (vd: `traitestmongodevct`, `traitestctphs`). Thẻ `omitempty` giúp các producer cũ không bị nhồi null/rỗng.
  + Dòng 10: `SourceDB string `json:"source_db,omitempty"``: Lưu database nguồn.
  + Dòng 11: `SourceTable string `json:"source_table,omitempty"``: Lưu collection/table nguồn.
- **Đánh giá:** Hoàn toàn khớp với Mục 1.2 của Plan. Thiết kế Minimal Impact, bảo toàn tính tương thích ngược với OCC (`SourceTsMs`, `KafkaOffset`).

---

### Task 2: Nâng cấp Interface MetadataRegistry & ResolvedSourceRoute
📁 **File:** `centralized-data-service/internal/service/metadata/metadata_registry.go`
- **Vị trí dòng:** Dòng 14, 18, 22, 41
- **Diff chi tiết:**
  ```go
  <<<< BEFORE:
  	GetTableConfigBySource(sourceTable string) *source.TableRegistry
  	ResolveSourceRoute(sourceDB, sourceTable string) *ResolvedSourceRoute
  	ResolveSourceRoutes(sourceDB, sourceTable string) []*ResolvedSourceRoute
  ...
  type ResolvedSourceRoute struct {
  	SourceObject        *source.SourceObjectRegistry
  	ShadowBinding       *shadow.ShadowBinding
  	TableConfig         *source.TableRegistry
  	ShadowConnectionKey string
  }
  ==== AFTER:
  	GetTableConfigBySource(sourceTable string, sourceConn ...string) *source.TableRegistry
  	ResolveSourceRoute(sourceDB, sourceTable string, sourceConn ...string) *ResolvedSourceRoute
  	ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*ResolvedSourceRoute
  ...
  type ResolvedSourceRoute struct {
  	SourceObject        *source.SourceObjectRegistry
  	ShadowBinding       *shadow.ShadowBinding
  	TableConfig         *source.TableRegistry
  	ShadowConnectionKey string
  	SourceConnectionKey string
  }
  >>>>
  ```
- **Phân tích từng dòng:**
  + Sử dụng tham số variadic `sourceConn ...string`: Cho phép người gọi truyền hoặc không truyền `sourceConn`. Các callsite cũ không bị vỡ biên dịch.
  + Bổ sung trường `SourceConnectionKey string` vào `ResolvedSourceRoute`: Giúp downstream nhận biết chính xác route này gắn với connection nguồn nào để lọc bảo vệ.
- **Đánh giá:** Hoàn toàn khớp với Mục 1.3 của Plan.

---

### Task 3: Sinh Lookup Keys ưu tiên Connection Code
📁 **File:** `centralized-data-service/internal/service/source/metadata_registry_utils.go`
- **Vị trí dòng:** Dòng 160–192
- **Diff chi tiết:**
  ```go
  <<<< BEFORE:
  func buildRouteLookupKeys(sourceDB, sourceTable string) []string {
  	sourceDB = strings.TrimSpace(sourceDB)
  	sourceTable = strings.TrimSpace(sourceTable)
  	return dedupeStrings([]string{
  		fmt.Sprintf("%s|%s", sourceDB, sourceTable),
  		fmt.Sprintf("%s:%s", sourceDB, sourceTable),
  		sourceTable,
  	})
  }
  ==== AFTER:
  func buildRouteLookupKeys(sourceDB, sourceTable string, sourceConn ...string) []string {
  	sourceDB = strings.TrimSpace(sourceDB)
  	sourceTable = strings.TrimSpace(sourceTable)
  	var conn string
  	if len(sourceConn) > 0 {
  		conn = strings.TrimSpace(sourceConn[0])
  	}

  	var keys []string
  	if conn != "" {
  		candidates := []string{conn}
  		if trimmed := strings.TrimPrefix(conn, "cdc.goopay."); trimmed != conn && trimmed != "" {
  			candidates = append(candidates, trimmed)
  		}
  		if trimmed := strings.TrimPrefix(conn, "cdc."); trimmed != conn && trimmed != "" {
  			candidates = append(candidates, trimmed)
  		}

  		for _, c := range candidates {
  			keys = append(keys,
  				fmt.Sprintf("%s:%s|%s", c, sourceDB, sourceTable),
  				fmt.Sprintf("%s:%s", c, sourceTable),
  			)
  		}
  	}

  	keys = append(keys,
  		fmt.Sprintf("%s|%s", sourceDB, sourceTable),
  		fmt.Sprintf("%s:%s", sourceDB, sourceTable),
  		sourceTable,
  	)
  	return dedupeStrings(keys)
  }
  >>>>
  ```
- **Phân tích từng dòng:**
  + Dòng 170-176: Xử lý tập `candidates` với các biến thể tiền tố: nguyên bản, cắt `cdc.goopay.`, cắt `cdc.`. Đảm bảo dù caller truyền chuỗi từ Kafka topic (`cdc.goopay.traitestmongodevct`) hay từ DB connection registry (`traitestmongodevct`), key sinh ra đều khớp trúng nhau.
  + Dòng 179-182: Sinh các key ưu tiên tuyệt đối: `c:sourceDB|sourceTable`, `c:sourceTable`.
  + Dòng 186-190: Giữ nguyên các key fallback chung để tương thích ngược khi không có `sourceConn`.
- **Đánh giá:** Khớp Plan 100%.

---

### Task 4: Khởi tạo Cache và Định tuyến Đa Kết Nối trong Registry Service
📁 **File:** `centralized-data-service/internal/service/source/metadata_registry_service.go`
- **Vị trí dòng 218:**
  Gán `SourceConnectionKey: sourceConnCode` khi khởi tạo `ResolvedSourceRoute`.
- **Vị trí dòng 386–401 (`GetTableConfigBySource`):**
  ```go
  func (rs *MetadataRegistryService) GetTableConfigBySource(sourceTable string, sourceConn ...string) *source.TableRegistry {
  	rs.mu.RLock()
  	defer rs.mu.RUnlock()
  	sourceTable = strings.TrimSpace(sourceTable)
  	if len(sourceConn) > 0 && strings.TrimSpace(sourceConn[0]) != "" {
  		conn := strings.TrimSpace(sourceConn[0])
  		if cfg, ok := rs.sourceCache[fmt.Sprintf("%s:%s", conn, sourceTable)]; ok {
  			return cfg
  		}
  		clean := strings.TrimPrefix(strings.TrimPrefix(conn, "cdc.goopay."), "cdc.")
  		if cfg, ok := rs.sourceCache[fmt.Sprintf("%s:%s", clean, sourceTable)]; ok {
  			return cfg
  		}
  	}
  	return rs.sourceCache[sourceTable]
  }
  ```
- **Vị trí dòng 553–592 (`ResolveSourceRoutes`):**
  ```go
  func (rs *MetadataRegistryService) ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*metadata.ResolvedSourceRoute {
  	rs.mu.RLock()
  	defer rs.mu.RUnlock()
  	var masterRoutes []*metadata.ResolvedSourceRoute
  	for _, key := range buildRouteLookupKeys(sourceDB, sourceTable, sourceConn...) {
  		if routes, ok := rs.routeCache[key]; ok && len(routes) > 0 {
  			masterRoutes = routes
  			break
  		}
  	}
  	if len(masterRoutes) == 0 {
  		return nil
  	}
  	routes := append([]*metadata.ResolvedSourceRoute(nil), masterRoutes...)
  	if masterRoutes[0].SourceObject != nil {
  		clones := rs.cloneRoutes[masterRoutes[0].SourceObject.ID]
  		routes = append(routes, clones...)
  	}

  	var conn string
  	if len(sourceConn) > 0 {
  		conn = strings.TrimSpace(sourceConn[0])
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
  	}
  	return routes
  }
  ```
- **Phân tích từng dòng & Phát hiện lỗi logic:**
  + Dòng 557: Duyệt các key từ `buildRouteLookupKeys`.
  + Dòng 578-586: Lọc theo `r.SourceConnectionKey == conn || cleanConn`.
  + **⚠️ LỖI LOGIC TẠI DÒNG 587–591:**
    Nếu caller truyền `sourceConn` cụ thể (ví dụ `conn = "traitestmongodevct"`), nhưng registry KHÔNG CÓ route nào của connection này (ví dụ do chưa active hoặc cấu hình sai), thì `len(filtered) == 0`.
    Khi đó, hàm **bỏ qua `if len(filtered) > 0` và trôi xuống dòng 591 `return routes`** (vốn là route của connection khác tìm được từ key fallback chung `db|table`).
    Điều này làm dữ liệu của connection lạ bị chèn nhầm vào bảng shadow của connection hiện có!
  + **Sửa chuẩn:** Tại dòng 590, nếu `conn != ""` mà `len(filtered) == 0`, BẮT BUỘC trả về `nil`!

---

### Task 5: Đồng bộ RegistryService Legacy
📁 **File:** `centralized-data-service/internal/service/source/registry_service.go`
- **Vị trí dòng:** Dòng 29, 40, 177
- **Phân tích:** Mở rộng chữ ký `sourceConn ...string` để thỏa mãn interface `MetadataRegistry`.

---

### Task 6: Trích xuất Connector Name & Envelope trong Kafka Consumer
📁 **File:** `centralized-data-service/internal/handler/shadow/kafka_consumer.go`
- **Vị trí dòng:** Dòng 652–692
- **Diff chi tiết:**
  ```go
  	var sourceConnCode, sourceDB, sourceTable string
  	if sm, ok := sourceRaw.(map[string]interface{}); ok {
  		if n, ok := sm["name"].(string); ok {
  			sourceConnCode = strings.TrimSpace(n)
  		}
  		if d, ok := sm["db"].(string); ok {
  			sourceDB = strings.TrimSpace(d)
  		}
  		if t, ok := sm["table"].(string); ok {
  			sourceTable = strings.TrimSpace(t)
  		} else if c, ok := sm["collection"].(string); ok {
  			sourceTable = strings.TrimSpace(c)
  		}
  	}

  	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.")
  	sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")

  	if sourceConnCode == "" || sourceConnCode == "mongodb" {
  		parts := strings.Split(msg.Topic, ".")
  		if len(parts) >= 5 && parts[0] == "cdc" {
  			sourceConnCode = parts[2]
  		}
  	}

  	cdcEvent := map[string]interface{}{
  		"source":       "debezium",
  		"source_conn":  sourceConnCode,
  		"source_db":    sourceDB,
  		"source_table": sourceTable,
  		"data": map[string]interface{}{
  			"op":           opStr,
  			"before":       beforeField,
  			"after":        afterData,
  			"source_ts_ms": sourceTsMs,
  		},
  		"kafka_key":       keyStr,
  		"kafka_topic":     msg.Topic,
  		"kafka_partition": msg.Partition,
  		"kafka_offset":    msg.Offset,
  	}
  ```
- **Phân tích từng dòng:**
  + Dòng 653-665: Trích xuất `source.name` từ Debezium envelope (`sourceRaw`). Hỗ trợ cả `table` lẫn `collection`.
  + Dòng 667-668: Cắt tiền tố để lấy mã connection thuần.
  + Dòng 670-675: Fallback qua topic name nếu payload không có `name`.
  + Dòng 679-681: Đóng gói tường minh `source_conn`, `source_db`, `source_table` vào JSON event.
- **Đánh giá:** Hoàn toàn khớp với Mục 1.1 của Plan. (Điểm cần hoàn thiện: bổ sung nhận diện cho topic 4-parts khi thiếu name).

---

### Task 7: Bóc tách an toàn & Định tuyến trong EventHandler
📁 **File:** `centralized-data-service/internal/handler/shadow/event_handler.go`
- **Vị trí dòng:** Dòng 170–250
- **Diff chi tiết:**
  ```go
  	// 1. Ưu tiên đọc trực tiếp từ CDCEvent nếu producer đã đóng gói
  	if event.SourceDB != "" && event.SourceTable != "" {
  		db = event.SourceDB
  		table = event.SourceTable
  		sourceConn = event.SourceConn
  	}
  ...
  	// 3. Fallback phân giải từ Subject an toàn (tính từ cuối chuỗi ngược lên):
  	if db == "" || table == "" {
  		parts := strings.Split(subject, ".")
  		if len(parts) >= 4 {
  			table = parts[len(parts)-1] // Luôn là table/collection
  			db = parts[len(parts)-2]    // Luôn là database
  			if len(parts) >= 5 && sourceConn == "" {
  				sourceConn = parts[len(parts)-3] // Là connection/connector name
  			}
  		}
  	}

  	if event.SourceConn != "" {
  		sourceConn = event.SourceConn
  	}

  	sourceConn = strings.TrimPrefix(sourceConn, "cdc.goopay.")
  	sourceConn = strings.TrimPrefix(sourceConn, "cdc.")
  ...
  	routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)
  ```
- **Phân tích từng dòng:**
  + Dòng 170-174: Đọc trực tiếp từ struct `CDCEvent` đã được parse, không cần parse lại string.
  + Dòng 216-223: Công thức bóc tách tương đối: `table = parts[len-1]`, `db = parts[len-2]`, `sourceConn = parts[len-3]`. Đây là cải tiến xuất sắc so với code cũ (code cũ dùng index tĩnh `parts[2]`, `parts[3]` gây lỗi khi topic có 5 segments).
  + Dòng 249: Truyền `sourceConn` vào `ResolveSourceRoutes`.

---

### Task 8: Phân lập Connector trong Oplog Bridge Handler
📁 **File:** `centralized-data-service/internal/handler/source/bridge_handler.go`
- **Vị trí dòng:** Dòng 201, 262, 274
- **Phân tích:**
  + Dòng 201: Truyền `payload.ConnectorName` vào `h.resolveCollection(ctx, coll, payload.ConnString, payload.ConnectorName)`.
  + Dòng 274: `routes := h.registrySvc.ResolveSourceRoutes(coll.SourceDB, coll.SourceTable, conn)`.
  + Đảm bảo bridge xử lý đúng schema của connector tương ứng.

---

### Task 9: Đóng gói Connection & Phân lập trong Snapshot Runner
📁 **Files:**
- `centralized-data-service/internal/handler/orchestration/snapshot_runner_utils.go:L144-155`
- `centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go:L511, L795`
- **Phân tích:**
  + `buildSnapshotEnvelope`: Đóng gói `"source_conn": conn` vào envelope JSON của snapshot event.
  + `snapshot_runner_handler.go:L511`: Truyền `conn.ConnectionCode` vào `ResolveSourceRoutes` khi kiểm tra pre-flight routes.
  + `snapshot_runner_handler.go:L795`: Truyền `conn.ConnectionCode` vào `buildSnapshotEnvelope` khi duyệt từng record MongoDB.
  + Triệt tiêu hoàn toàn lỗi snapshot runner ghi dữ liệu chéo bảng shadow khi chạy sync không truyền binding scope.

---

### Task 10: Chuẩn hóa Toàn Trình Extended JSON trong Dynamic Mapper
📁 **File:** `centralized-data-service/internal/service/shadow/dynamic_mapper.go`
- **Vị trí dòng:** Dòng 71–75, Dòng 394–463
- **Diff chi tiết:**
  ```go
  // MapData đầu vào:
  	normalizedRaw := rawData
  	if nMap, ok := normalizeMongoExtJSON(rawData).(map[string]interface{}); ok {
  		normalizedRaw = nMap
  	}
  ...
  // Hàm unwrap đệ quy:
  func normalizeMongoExtJSONWithDepth(v interface{}, depth int) interface{} {
  	if depth > 32 { // Guard chống stack overflow
  		return v
  	}
  	switch val := v.(type) {
  	case map[string]interface{}:
  		// 1. MongoDB {"$oid": "..."}
  		if oid, ok := val["$oid"]; ok && len(val) == 1 {
  			return fmt.Sprintf("%v", oid)
  		}
  		// 2. MongoDB {"$date": ...}
  		if dateVal, ok := val["$date"]; ok && len(val) == 1 {
  			switch dt := dateVal.(type) {
  			case float64:
  				return int64(dt)
  			case string:
  				if t, err := time.Parse(time.RFC3339, dt); err == nil {
  					return t.UnixMilli()
  				}
  				return dt
  			case map[string]interface{}:
  				if numStr, ok := dt["$numberLong"].(string); ok {
  					if n, err := strconv.ParseInt(numStr, 10, 64); err == nil {
  						return n
  					}
  				}
  			case json.Number:
  				if n, err := dt.Int64(); err == nil {
  					return n
  				}
  			}
  			return dateVal
  		}
  		// 3. MongoDB {"$numberLong": "..."}
  		if numStr, ok := val["$numberLong"].(string); ok && len(val) == 1 {
  			if n, err := strconv.ParseInt(numStr, 10, 64); err == nil {
  				return n
  			}
  		}
  		// 4. MongoDB {"$numberInt": "..."}
  		if numStr, ok := val["$numberInt"].(string); ok && len(val) == 1 {
  			if n, err := strconv.ParseInt(numStr, 10, 64); err == nil {
  				return n
  			}
  		}

  		// Đệ quy cho map thông thường
  		normalized := make(map[string]interface{}, len(val))
  		for k, child := range val {
  			normalized[k] = normalizeMongoExtJSONWithDepth(child, depth+1)
  		}
  		return normalized
  	case []interface{}:
  		normalized := make([]interface{}, len(val))
  		for i, child := range val {
  			normalized[i] = normalizeMongoExtJSONWithDepth(child, depth+1)
  		}
  		return normalized
  	default:
  		return v
  	}
  }
  ```
- **Phân tích từng dòng:**
  + Dòng 71-75: Áp dụng ngay đầu hàm `MapData`. Cả mapping rules lẫn `_raw_data` đều nhận data đã unwrap sạch.
  + Dòng 403: Guard độ sâu `depth > 32` triệt tiêu nguy cơ tấn công DoS / Stack Overflow bằng JSON lồng nhau sâu.
  + Dòng 409-446: Unwrap toàn bộ các kiểu Extended JSON thông dụng (`$oid`, `$date`, `$numberLong`, `$numberInt`).
  + Dòng 448-460: Đệ quy xử lý triệt để các cấu trúc lồng nhau (nested objects và arrays).

---

### Task 11: Phân lập Topic Prefix trên Giao diện CMS Web
📁 **File:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Vị trí dòng:** Dòng 488
- **Code:** `form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);`
- **Phân tích:** Tự động điền prefix mang tên connector khi tạo mới connector MongoDB trên giao diện.

---

### Task 12: Kiểm thử Unit Test Đa Kết Nối & Unwrap JSON
📁 **File:** `centralized-data-service/internal/service/source/metadata_registry_service_test.go`
- **Vị trí dòng:** Dòng 280–420
- **Phân tích:** Đã có 3 test suites:
  1. `TestBuildRouteLookupKeys_MultiConnection`: Kiểm thử sinh các key lookup có connection.
  2. `TestMetadataRegistryService_ResolveSourceRoutes_MultiConnIsolation`: Kiểm thử query đúng connection 1 và 2.
  3. `TestNormalizeMongoExtJSON_FullSuite`: Kiểm thử unwrap primitives, extended json, nested map, mảng, và depth guard.

---

## III. TƯ DUY PHẢN BIỆN: PHÂN TÍCH LỖ HỔNG & ĐIỂM LỆCH KẾ HOẠCH

Dựa trên việc kiểm tra chi tiết từng dòng code, Brain chỉ ra 3 điểm cần khắc phục:

| STT | Mức độ | Tệp tin & Dòng code | Mô tả Lỗ hổng & Kịch bản Rủi ro | Giải pháp chuẩn hóa |
|:---:|:---:|:---|:---|:---|
| 1 | **CRITICAL** | `metadata_registry_service.go`<br>`L587-591` | **Rò rỉ route khi connection lạ (Route Leakage):** Khi caller truyền `sourceConn = "unknown_conn"`, hàm lọc ra `len(filtered) == 0`. Nhưng vì thiếu nhánh `return nil`, hàm rơi xuống `return routes` chung $\rightarrow$ Trả về route của connection khác, gây ghi chép dữ liệu chéo bảng shadow! | Nếu `conn != ""` mà `len(filtered) == 0` thì **BẮT BUỘC `return nil`**. |
| 2 | **MEDIUM** | `kafka_consumer.go`<br>`L670-675` | **Điểm mù topic 4-parts khi payload thiếu name:** Điều kiện `len(parts) >= 5` bỏ sót topic format 4 segments `cdc.<conn>.<db>.<table>` nếu Debezium envelope không mang trường `name`. | Bổ sung fallback cho 4-parts: `if len(parts) == 4 && parts[0] == "cdc" && parts[1] != "goopay" { sourceConnCode = parts[1] }`. |
| 3 | **LOW** | `metadata_registry_service_test.go`<br>`L310-328` | **Thiếu Negative Test Case:** Test suite mới chỉ kiểm thử trường hợp thành công (happy-path), chưa kiểm thử trường hợp truyền connection không tồn tại để xác nhận hàm trả về `nil`. | Bổ sung test case `ResolveSourceRoutes("core_db", "trans_his", "unknown_conn")` assert trả về `nil`. |

---

## IV. RÀ SOÁT TUÂN THỦ KIẾN TRÚC & CORE SYSTEMS

- **Nguyên tắc "Simplicity First, Minimal Impact" (Rule #12):**
  Mã nguồn mới không tạo thêm framework hay abstraction phức tạp. Giữ nguyên cấu trúc `MetadataRegistry`, chỉ mở rộng tham số variadic và bổ sung bộ lọc an toàn.
- **Nguyên tắc "Tư duy Core Systems" (Rule #12):**
  Giải quyết tận gốc rễ từ tầng dữ liệu vào (Debezium Consumer) cho tới tầng định tuyến (Metadata Registry) và tầng lưu trữ (Dynamic Mapper). Tuyệt đối không can thiệp sửa trực tiếp DB.
- **Nguyên tắc "No Shadow Files" (Rule #4):**
  Mọi phân tích và nhật ký audit đều được ghi nhận vào `audit_report_*.md` và `05_progress.md` trong workspace vật lý.

---

## V. VÒNG LẶP PHẢN TỈNH & KHẮC PHỤC (SELF-IMPROVEMENT LOOP)

### 1. Bài học rút ra (Self-Reflection):
- Khi thực hiện kiểm thử và nghiệm thu, không được chỉ nhìn vào **Happy-Path** (các trường hợp chạy đúng kỳ vọng). Bắt buộc phải đặt mình vào vị thế **Adversarial / Negative-Path** (nếu client truyền connection không tồn tại, nếu payload thiếu trường, nếu topic có format dị biệt thì hệ thống ứng xử ra sao?).
- Thiếu sót tại dòng 591 `metadata_registry_service.go` chính là hậu quả của việc chỉ kiểm tra xem connection 1 và connection 2 có lọc đúng không, mà quên mất trường hợp connection 3 không tồn tại sẽ bị tuột fallback!

### 2. Kế hoạch khắc phục (Refinement Patch):
Cần thực hiện ngay 3 chỉnh sửa cụ thể:
1. Sửa dòng 587-592 `metadata_registry_service.go`: trả về `nil` khi `conn != ""` và `len(filtered) == 0`.
2. Sửa dòng 670-675 `kafka_consumer.go`: hỗ trợ fallback topic 4-parts.
3. Bổ sung Negative Test trong `metadata_registry_service_test.go`.
