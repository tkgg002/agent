# Implementation: Fallback Default Schema từ Connection Registry

## Chi tiết các sửa đổi

### 1. `metadata_registry_utils.go`
Sửa hàm `buildDSNFromFields` cho Postgres:
```go
		searchPath := ""
		if conn.DefaultSchema != nil && *conn.DefaultSchema != "" {
			searchPath = fmt.Sprintf("&search_path=%s", *conn.DefaultSchema)
		}
		return fmt.Sprintf("postgres://%s:%d/%s?sslmode=%s%s", host, port, db, sslmode, searchPath)
```

### 2. `snapshot_runner_handler.go`
Sửa logic xác định schema:
```go
		schema := "public"
		if so.SourceSchema != nil && *so.SourceSchema != "" {
			schema = *so.SourceSchema
		} else if conn.DefaultSchema != nil && *conn.DefaultSchema != "" {
			schema = *conn.DefaultSchema
		}
```

### 3. `snapshot_runner_test.go`
Cập nhật mock `ConnectionRegistry` để cung cấp `DefaultSchema`.
