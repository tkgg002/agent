# 09_tasks_solution_kafka_consumer_source_route.md
## Hồ Sơ Kỹ Thuật & Code Demo Toàn Diện: Giải Quyết Bài Toán Đa Kết Nối Trùng Tên (Source & Master)

---

### PHẦN 1: TẦNG SOURCE — CÙNG (DB, COLLECTION) NHƯNG KHÁC CONNECTION

#### 1.1. `internal/handler/shadow/kafka_consumer.go`
**Mục tiêu:** Trích xuất định danh connector nguồn từ Debezium envelope hoặc topic name, đóng gói vào `cdcEvent["source_conn"]`.
```go
// Code Demo: internal/handler/shadow/kafka_consumer.go - trong func processMessage
// Trích xuất connector name từ Debezium envelope (event["source"]["name"])
var sourceConnCode string
if sMap, ok := sourceRaw.(map[string]interface{}); ok {
	if nameVal, exists := sMap["name"]; exists && nameVal != nil {
		sourceConnCode = strings.TrimSpace(fmt.Sprintf("%v", nameVal))
	}
}
// Fallback: nếu sourceRaw không có name, phân giải từ topic name
if sourceConnCode == "" {
	engine, _, _, _ := observability.ParseDebeziumTopic(msg.Topic)
	if engine != "" && engine != "mongodb" && engine != "postgres" && engine != "mysql" {
		sourceConnCode = engine
	}
}

// Đóng gói vào cdcEvent
cdcEvent := map[string]interface{}{
	"source":      "debezium",
	"source_conn": sourceConnCode, // <-- THÊM TRƯỜNG NÀY
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

---

#### 1.2. `internal/model/shadow/cdc_event.go` & `internal/handler/shadow/event_handler.go`
**Mục tiêu:** Đọc `SourceConn` từ CDCEvent và truyền vào `ResolveSourceRoutes`.
```go
// Code Demo: internal/model/shadow/cdc_event.go
type CDCEvent struct {
	SpecVersion string                 `json:"specversion"`
	Source      string                 `json:"source"`
	SourceConn  string                 `json:"source_conn,omitempty"` // <-- THÊM ĐỊNH DANH CONNECTION NGUỒN
	Type        string                 `json:"type"`
	Time        string                 `json:"time"`
	KafkaKey    string                 `json:"kafka_key,omitempty"`
	KafkaTopic  string                 `json:"kafka_topic,omitempty"`
	Data        CDCData                `json:"data"`
}
```

```go
// Code Demo: internal/handler/shadow/event_handler.go - trong func HandleRaw
	var db, table, sourceConn string

	// 1. Ưu tiên đọc trực tiếp từ CDCEvent nếu producer đã đóng gói
	if event.SourceDB != "" && event.SourceTable != "" {
		db = event.SourceDB
		table = event.SourceTable
		sourceConn = event.SourceConn
	}

	// 2. Nếu chưa có, trích xuất từ Debezium Source struct trong Data
	if db == "" || table == "" {
		var temp struct {
			Data struct {
				Source *struct {
					DB         string `json:"db"`
					Table      string `json:"table"`
					Collection string `json:"collection"`
					Name       string `json:"name"`
				} `json:"source"`
			} `json:"data"`
		}
		if errTemp := json.Unmarshal(data, &temp); errTemp == nil && temp.Data.Source != nil {
			db = temp.Data.Source.DB
			table = temp.Data.Source.Table
			if table == "" {
				table = temp.Data.Source.Collection
			}
			if sourceConn == "" && temp.Data.Source.Name != "" {
				sourceConn = temp.Data.Source.Name
			}
		}
	}

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

	if db == "" || table == "" {
		return 0, fmt.Errorf("rejecting CDC event: database and table name are empty for subject %q", subject)
	}

	rows, err = h.processEvent(ctx, start, &event, subject, db, table, sourceConn)
```

```go
// Code Demo: internal/handler/shadow/event_handler.go - trong func processEvent
func (h *EventHandler) processEvent(ctx context.Context, start time.Time, event *shadow.CDCEvent, subject, sourceDB, sourceTable, sourceConn string) (int, error) {

	// Phân giải chính xác route theo bộ ba (sourceDB, sourceTable, sourceConn)
	routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)
	if len(routes) == 0 {
		observability.Ctx(ctx, h.logger).Warn("event skipped: source not in registry cache",
			zap.String("source_conn", sourceConn),
			zap.String("source_db", sourceDB),
			zap.String("source_table", sourceTable),
		)
		return 0, nil
	}
	...
```

---

#### 1.3. `MetadataRegistry` & `MetadataRegistryService`
**Mục tiêu:** Mở rộng tra cứu key có `connCode:sourceDB|sourceTable`.
```go
// Code Demo: internal/service/metadata/metadata_registry.go
type MetadataRegistry interface {
	...
	ResolveSourceRoute(sourceDB, sourceTable string, sourceConn ...string) *ResolvedSourceRoute
	ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*ResolvedSourceRoute
	GetTableConfigBySource(sourceTable string, sourceConn ...string) *source.TableRegistry
	...
}
```

```go
// Code Demo: internal/service/source/metadata_registry_utils.go
func buildRouteLookupKeys(sourceDB, sourceTable string, sourceConn ...string) []string {
	sourceDB = strings.TrimSpace(sourceDB)
	sourceTable = strings.TrimSpace(sourceTable)
	connCode := ""
	if len(sourceConn) > 0 {
		connCode = strings.TrimSpace(sourceConn[0])
	}

	var keys []string
	if connCode != "" {
		// Ưu tiên 1: Khớp đích danh connection code
		keys = append(keys,
			fmt.Sprintf("%s:%s|%s", connCode, sourceDB, sourceTable),
			fmt.Sprintf("%s:%s", connCode, sourceTable),
		)
		// Hỗ trợ trường hợp connCode mang prefix "cdc."
		trimmed := strings.TrimPrefix(connCode, "cdc.")
		if trimmed != connCode && trimmed != "" {
			keys = append(keys,
				fmt.Sprintf("%s:%s|%s", trimmed, sourceDB, sourceTable),
				fmt.Sprintf("%s:%s", trimmed, sourceTable),
			)
		}
	}

	// Fallback chung: tra cứu theo db|table (chỉ dùng khi không có connCode)
	keys = append(keys,
		fmt.Sprintf("%s|%s", sourceDB, sourceTable),
		fmt.Sprintf("%s:%s", sourceDB, sourceTable),
		sourceTable,
	)
	return dedupeStrings(keys)
}
```

```go
// Code Demo: internal/service/source/metadata_registry_service.go
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
	return routes
}
```

---

#### 1.4. `internal/handler/source/bridge_handler.go`
**Mục tiêu:** Truyền `payload.ConnectorName` vào `resolveCollection` để không nhặt nhầm shadow schema của kết nối khác.
```go
// Code Demo: internal/handler/source/bridge_handler.go
func (h *BridgeHandler) resolveCollection(ctx context.Context, coll BridgeCollection, connectorName, payloadConnString string) resolvedCollection {
	resolved := resolvedCollection{
		BridgeCollection: coll,
		pgPKField:        coll.PKField,
		connString:       payloadConnString,
	}

	// Truyền connectorName vào ResolveSourceRoutes
	routes := h.registrySvc.ResolveSourceRoutes(coll.SourceDB, coll.SourceTable, connectorName)
	if len(routes) == 0 {
		...
		return resolved
	}

	route := routes[0]
	tc := route.TableConfig
	if tc.SourceURL != "" {
		resolved.connString = tc.SourceURL
	}
	resolved.TableName = tc.TargetTable
	resolved.PKField = tc.PrimaryKeyField
	if route.ShadowBinding != nil && strings.TrimSpace(route.ShadowBinding.ShadowSchema) != "" {
		resolved.SchemaName = strings.TrimSpace(route.ShadowBinding.ShadowSchema)
	} else {
		resolved.SchemaName = "public"
	}
	resolved.shadowConnectionKey = route.ShadowConnectionKey
	return resolved
}
```

---

#### 1.5. CMS Web `SourceConnectors.tsx`
**Mục tiêu:** Tránh việc mọi connector MongoDB đều bị ép cứng về chung `topic.prefix = cdc.goopay`.
```typescript
// Code Demo: cdc-cms-web/src/pages/SourceConnectors.tsx
useEffect(() => {
  if (!editorOpen || editorMode !== 'create') return;
  const name = slugifyForShadow(String(connectorNameValue || 'connector'));
  if (dbKind === 'sftp') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_SFTP}.${name}`);
  } else if (dbKind === 'mongodb') {
    // Tự động phân lập topic prefix theo tên connector để tránh đụng độ Kafka topic khi trùng tên DB
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
  } else if (dbKind === 'mysql') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MYSQL}.${name}`);
  } else if (dbKind === 'postgresql') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_POSTGRESQL}.${name}`);
  }
}, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
```

---

### PHẦN 2: TẦNG MASTER — CÙNG (SCHEMA, TABLE) NHƯNG KHÁC MASTER CONNECTION

#### 2.1. Triệt tiêu `LIMIT 1` và Định danh bằng `master_binding_id`
Khi 2 Master bindings cùng có `master_schema = "public"` và `master_table = "trans_his"`, một binding ghi vào DB Master 1 (`default`), một binding ghi vào DB Master 2 (`master_2`):
- **CMS Backend (`master_repo_gorm.go`)**:
  Hàm `findBindingForAction` ưu tiên:
  1. `bindingID > 0`: `WHERE id = ?` (chính xác 100%).
  2. `name` là chuỗi số: `WHERE id = ?`.
  3. `name` là `schema.table`: Nếu có > 1 dòng trên các connection khác nhau mà không truyền `binding_id`, **báo lỗi ngay `ambiguous_master_name`**, bắt buộc client cung cấp `binding_id`.
- **Transmuter (`transmuter.go`)**:
  `TransmuteRequest` mang `MasterBindingID int64`. `loadMaster` query trực tiếp `WHERE mb.id = ?`.
  Kết nối target database động qua `connMgr.GetMasterDB(ctx, mb.MasterConnectionKey)`.
- **Master DDL Generator (`master_ddl_generator.go`)**:
  `masterCreateRequest` mang `MasterBindingID int64`. `loadBinding` query trực tiếp `WHERE mb.id = ?`.
  Thực thi DDL trên đúng target database qua `connMgr.GetMasterDB(ctx, mb.MasterConnectionKey)`.
- **Transmute Scheduler (`transmute_scheduler.go`)**:
  Claim due jobs đọc `ts.master_binding_id`, NATS command `cdc.cmd.transmute` luôn mang `master_binding_id`.
- **Realtime Fanout (`transmute_handler.go`)**:
  Khi Shadow table nhận data, gọi `ListMasterTargetsByShadowIdentity(shadowSchema, shadowTable)` trả về danh sách các `MasterTargetIdentity { ID, MasterFQN, MasterConnectionKey }`. Duyệt từng target và phát lệnh transmute riêng biệt kèm đúng `master_binding_id`.

---

### PHẦN 3: CHUẨN HÓA ĐỆ QUY `_raw_data` TRONG DYNAMIC MAPPER

**Vị trí:** `internal/service/shadow/dynamic_mapper.go`
```go
// Code Demo: internal/service/shadow/dynamic_mapper.go
// normalizeMongoExtJSON unwrap đệ quy các kiểu dữ liệu MongoDB Extended JSON ($oid, $date)
func normalizeMongoExtJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Trường hợp object mang khóa đặc thù của Extended JSON
		if len(val) == 1 {
			if oid, ok := val["$oid"]; ok {
				return fmt.Sprintf("%v", oid)
			}
			if d, ok := val["$date"]; ok {
				switch dt := d.(type) {
				case float64:
					return int64(dt)
				case string:
					if t, err := time.Parse(time.RFC3339, dt); err == nil {
						return t.UnixMilli()
					}
					return dt
				case json.Number:
					if ms, err := dt.Int64(); err == nil {
						return ms
					}
				}
				return d
			}
		}
		// Đệ quy cho map thông thường
		normalized := make(map[string]interface{}, len(val))
		for k, item := range val {
			normalized[k] = normalizeMongoExtJSON(item)
		}
		return normalized
	case []interface{}:
		// Đệ quy cho slice
		normalized := make([]interface{}, len(val))
		for i, item := range val {
			normalized[i] = normalizeMongoExtJSON(item)
		}
		return normalized
	default:
		return v
	}
}

// Trong func MapRecord:
func (dm *DynamicMapper) MapRecord(bindingID int64, rawData map[string]interface{}) (*MappedData, error) {
	...
	// Chuẩn hóa đệ quy Extended JSON trước khi lưu vào _raw_data
	normalizedRaw, ok := normalizeMongoExtJSON(rawData).(map[string]interface{})
	if !ok {
		normalizedRaw = rawData
	}
	rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, normalizedRaw))

	return &MappedData{
		Columns:      columns,
		EnrichedData: enriched,
		RawJSON:      rawJSON,
	}, nil
}
```
