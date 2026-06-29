# Technical Design: Core Fix for Batch Transform Schema Drift

## Proposed Changes

### centralized-data-service

#### [MODIFY] [batch_transform_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go)

Di chuyển khai báo `execDB` lên trước vòng lặp xử lý mapping rules để dùng làm DB context kiểm tra cột:
```go
	execDB := h.DB
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}
```

Thêm kiểm tra cột tồn tại trong vòng lặp `for _, rule := range rules`:
```go
		// Kiểm tra xem cột đích có tồn tại trong schema DB thực tế hay không
		if !h.HasColumnInSchema(ctx, execDB, schemaName, targetTable, rule.TargetColumn) {
			h.Logger.Warn("batch transform: target_column does not exist in db, skipping rule",
				zap.String("table", targetTable),
				zap.String("target_column", rule.TargetColumn),
				zap.String("source_field", rule.SourceField),
			)
			continue
		}
```

Điều này đảm bảo nếu mapping rule chứa cột `__v` nhưng bảng đích trên Shadow DB chưa đồng bộ cột này, worker sẽ ghi nhận cảnh báo và tiếp tục transform các cột hợp lệ khác mà không bị crash câu lệnh UPDATE.

---

#### [MODIFY] [base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go)
Đồng bộ hóa biểu thức chính quy kiểm tra kiểu dữ liệu an toàn khớp với whitelist của `TypeResolver`:
```go
var reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|BIGINT|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)

func IsSafeType(t string) bool {
	u := strings.ToUpper(strings.TrimSpace(t))
	return reTypeWhitelist.MatchString(u)
}
```
Việc này sẽ chấp nhận các kiểu dữ liệu có tham số độ dài hoặc độ chính xác như `VARCHAR(24)`, `VARCHAR(255)`, `NUMERIC(10,2)` từ Registry đẩy về.

