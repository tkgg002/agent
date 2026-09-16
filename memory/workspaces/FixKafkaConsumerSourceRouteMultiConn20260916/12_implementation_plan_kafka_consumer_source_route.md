# 12_implementation_plan_kafka_consumer_source_route.md
## Kế Hoạch Triển Khai Chi Tiết: Sửa Lỗi Routing Source -> Shadow Tại Kafka Consumer Khi Trùng Tên Database

---

### I. Mục Tiêu & Phạm Vi
Khắc phục triệt để lỗi phân phối nhầm dữ liệu từ **Source -> Shadow** tại tầng **Kafka Consumer** khi hệ thống kết nối với nhiều nguồn database có cùng tên (ở đây là 2 kết nối MongoDB `traitestmongodevct` và `traitestctphs` cùng có database `core-trans-proxy-history-service`). Đồng thời chuẩn hóa trường `_raw_data` phẳng 100% giữa Snapshot V2 và Debezium.

---

### II. Chi Tiết Thay Đổi Code

#### 1. File `internal/service/metadata/metadata_registry.go`
Mở rộng interface:
```go
ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*ResolvedSourceRoute
```

#### 2. File `internal/service/source/metadata_registry_utils.go`
Sửa `buildRouteLookupKeys`:
```go
func buildRouteLookupKeys(sourceDB, sourceTable string, sourceConn ...string) []string {
	sourceDB = strings.TrimSpace(sourceDB)
	sourceTable = strings.TrimSpace(sourceTable)
	connCode := ""
	if len(sourceConn) > 0 {
		connCode = strings.TrimSpace(sourceConn[0])
	}

	var keys []string
	if connCode != "" {
		keys = append(keys,
			fmt.Sprintf("%s:%s|%s", connCode, sourceDB, sourceTable),
			fmt.Sprintf("%s:%s", connCode, sourceTable),
		)
		trimmed := strings.TrimPrefix(connCode, "cdc.")
		if trimmed != connCode && trimmed != "" {
			keys = append(keys,
				fmt.Sprintf("%s:%s|%s", trimmed, sourceDB, sourceTable),
				fmt.Sprintf("%s:%s", trimmed, sourceTable),
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
```

#### 3. File `internal/service/source/metadata_registry_service.go`
Sửa `ResolveSourceRoutes`:
```go
func (rs *MetadataRegistryService) ResolveSourceRoutes(sourceDB, sourceTable string, sourceConn ...string) []*metadata.ResolvedSourceRoute {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	var masterRoutes []*metadata.ResolvedSourceRoute
	for _, key := range buildRouteLookupKeys(sourceDB, sourceTable, sourceConn...) {
		if routes, ok := rs.routeCache[key]; ok {
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

#### 4. File `internal/model/shadow/cdc_event.go`
Bổ sung `SourceConn string `json:"source_conn,omitempty"`` vào `CDCEvent`.

#### 5. File `internal/handler/shadow/kafka_consumer.go`
Trong hàm `processMessage`:
Trích xuất `sourceConnCode`:
```go
	var sourceConnCode string
	if sMap, ok := sourceRaw.(map[string]interface{}); ok {
		if nameVal, exists := sMap["name"]; exists {
			sourceConnCode = fmt.Sprintf("%v", nameVal)
		}
	}
	if sourceConnCode == "" {
		engine, _, _, _ := observability.ParseDebeziumTopic(msg.Topic)
		if engine != "" && engine != "mongodb" && engine != "postgres" && engine != "mysql" {
			sourceConnCode = engine
		}
	}
```
Gán vào `cdcEvent`:
```go
	cdcEvent := map[string]interface{}{
		"source":      "debezium",
		"source_conn": sourceConnCode,
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

#### 6. File `internal/handler/shadow/event_handler.go`
Trong `HandleRaw`:
- Đọc `sourceConn := event.SourceConn`. Nếu rỗng, fallback thử trích xuất từ `subject`.
- Truyền `sourceConn` vào `h.processEvent(ctx, start, &event, subject, db, table, sourceConn)`.
- Trong `processEvent`:
  Gọi `routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)`.

#### 7. File `internal/service/shadow/dynamic_mapper.go`
- Bổ sung hàm đệ quy `normalizeMongoExtJSON(v interface{}) interface{}` để unwrap triệt để `$oid` thành string và `$date` thành timestamp (hoặc ISO string).
- Trong `MapRecord`:
  ```go
  normalizedRaw := normalizeMongoExtJSON(rawData).(map[string]interface{})
  rawJSON, _ := json.Marshal(dm.maskRawData(bindingID, normalizedRaw))
  ```

---

### III. Kế Hoạch Kiểm Thử
1. Unit test `TestResolveSourceRoutes_MultiConnectionCollision`:
   - Mock 2 source connections (`connA`, `connB`) có cùng `sourceDB` và `sourceTable`.
   - Gọi `ResolveSourceRoutes(db, table, "connA")` -> Chỉ trả về route của `connA`.
   - Gọi `ResolveSourceRoutes(db, table, "connB")` -> Chỉ trả về route của `connB`.
2. Unit test `TestDynamicMapper_NormalizeMongoExtJSON`:
   - Payload chứa `{"$oid": "..."}`, `{"$date": "..."}` -> Output `_raw_data` phẳng, không còn key `$` lồng.
3. Build Worker: `go build ./cmd/worker`.
