# Giải Pháp Kỹ Thuật Chi Tiết - Fix PostgreSQL Scan Fields ID

## 1. `centralized-data-service/internal/handler/source/discovery_utils.go`
Sửa `inferPGCols`:
```go
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, err
		}
		// XÓA BỎ skip pkColumn để giữ lại tất cả các cột bao gồm cả id:
		// if strings.EqualFold(name, pkColumn) { continue }
		out = append(out, shadow.BusinessColumn{
			Name:     name,
			DataType: pgSafeType(dataType),
			Nullable: strings.EqualFold(isNullable, "YES"),
		})
	}
```
Tương tự cho `inferMySQLCols` và `inferMongoCols`.

## 2. `centralized-data-service/internal/handler/source/discover_handler_utils.go`
Trong `processDiscoveryRows`, unwrap `after` map nếu dữ liệu trong `_raw_data` chứa Debezium payload envelope:
```go
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(row), &doc); err != nil {
			continue
		}
		targetMap := doc
		if after, ok := doc["after"].(map[string]interface{}); ok && after != nil {
			targetMap = after
		}
		for k, v := range targetMap {
			if k == "_raw_data" || k == "_synced_at" || k == "_source" {
				continue
			}
			...
```
