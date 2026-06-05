# Giải pháp Kỹ thuật (Technical Solutions)

## Task 1: Fix `schema_inspector.go`

**Mục tiêu**: Ngăn việc cache empty schema khi không resolve được target route/schema.
**File**: `internal/service/schema_inspector.go`

1. Sửa hàm `resolveTargetSchema(tableName string)` thành `resolveTargetSchema(tableName string) (string, error)`:
```go
import "errors"

var ErrUnresolvableSchema = errors.New("unresolvable target schema")

func (si *SchemaInspector) resolveTargetSchema(tableName string) (string, error) {
	if si.metadata != nil {
		if route := si.metadata.ResolveTargetRoute(tableName); route != nil && route.ShadowBinding != nil {
			if v := strings.TrimSpace(route.ShadowBinding.ShadowSchema); v != "" {
				return v, nil
			}
		}
	}
	return "", ErrUnresolvableSchema
}
```

2. Cập nhật `getTableSchema`:
```go
func (si *SchemaInspector) getTableSchema(ctx context.Context, tableName string) (map[string]bool, error) {
	schemaName, err := si.resolveTargetSchema(tableName)
	if err != nil {
		return nil, err
	}
	// ... (phần còn lại giữ nguyên)
```

3. Cập nhật `InspectEvent` (bỏ qua drift nếu lỗi `ErrUnresolvableSchema`):
```go
	tableSchema, err := si.getTableSchema(ctx, tableName)
	if err != nil {
		if errors.Is(err, ErrUnresolvableSchema) {
			si.logger.Debug("skipping schema inspection: unresolvable schema", zap.String("table", tableName))
			return &SchemaDrift{Detected: false}, nil
		}
		return nil, fmt.Errorf("get table schema: %w", err)
	}
```

## Task 2: Fix `event_handler.go`

**Mục tiêu**: Gom batch DB insert, dừng bypass qua `WriteRecordSync`.
**File**: `internal/handler/event_handler.go`

1. Tại hàm `processEvent`, khoảng dòng `140`:
```go
		// Hủy WriteRecordSync để gom batch buffer
		h.batchBuffer.Add(record)
		written := 1 // Giả lập enqueue thành công 1 record
		totalWritten += written
		
		// Thay vì:
		// written, err := h.batchBuffer.WriteRecordSync(record)
		// if err != nil { ... }
		// totalWritten += written
```
**Lưu ý**: Việc này sẽ khiến `totalWritten` trả về số lượng record được đẩy vào memory queue thay vì số rows thực tế được ghi xuống DB (bỏ qua error tức thời do async).

## Task 3: Kafka Consumer Offset Reset (Manual)

**Mục tiêu**: Dừng xử lý các row rác cũ.
Mở Terminal, truy cập container/máy chạy Kafka và chạy:
```bash
kafka-consumer-groups.sh --bootstrap-server <broker:port> --group cdc.goopay.scheduler-service.schedule_histories.worker --reset-offsets --to-latest --execute --topic cdc.goopay.scheduler-service.schedule_histories
```
(Thay thế tên consumer group chính xác dựa trên config).
