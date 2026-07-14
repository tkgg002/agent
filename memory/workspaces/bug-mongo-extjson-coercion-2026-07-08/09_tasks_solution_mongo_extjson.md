# Hồ sơ giải pháp kỹ thuật - Sửa lỗi mapping MongoDB Ext-JSON Date/Timestamp vào Postgres

Hồ sơ giải pháp này mô tả chi tiết các phần mã nguồn sẽ được cập nhật trong `schema_adapter_coerce.go` để chuyển đổi các định dạng ngày/giờ từ MongoDB sang PostgreSQL.

## 1. File sửa đổi: `internal/service/shadow/schema_adapter_coerce.go`

### Bổ sung case trong `CoerceValue`
```go
	case "timestamp with time zone", "timestamp without time zone", "timestamptz", "timestamp", "date":
		return coerceToTimeOrNull(sa.logger, colName, val)
```

### Định nghĩa hàm helper `coerceToTimeOrNull`, `int64ToTime`, và `float64ToTime`
```go
func coerceToTimeOrNull(logger *zap.Logger, colName string, val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case time.Time:
		return v.UTC()
	case *time.Time:
		if v == nil {
			return nil
		}
		return v.UTC()
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		// Thử parse các định dạng phổ biến
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC()
			}
		}
		// Thử parse dạng chuỗi số (epoch seconds hoặc milliseconds)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return int64ToTime(n)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return float64ToTime(f)
		}
		if logger != nil {
			logger.Warn("coerce: time column got unrecognized string → NULL",
				zap.String("column", colName), zap.String("value", v))
		}
		return nil

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		var n int64
		switch x := v.(type) {
		case int:
			n = int64(x)
		case int8:
			n = int64(x)
		case int16:
			n = int64(x)
		case int32:
			n = int64(x)
		case int64:
			n = x
		case uint:
			n = int64(x)
		case uint8:
			n = int64(x)
		case uint16:
			n = int64(x)
		case uint32:
			n = int64(x)
		case uint64:
			n = int64(x)
		}
		return int64ToTime(n)

	case float32, float64:
		var f float64
		if x, ok := v.(float64); ok {
			f = x
		} else {
			f = float64(v.(float32))
		}
		return float64ToTime(f)

	case map[string]interface{}:
		// Hỗ trợ bóc tách đệ quy cho Ext-JSON {"$date": ...} hoặc {"$numberLong": ...}
		if dateVal, ok := v["$date"]; ok {
			return coerceToTimeOrNull(logger, colName, dateVal)
		}
		if numVal, ok := v["$numberLong"]; ok {
			return coerceToTimeOrNull(logger, colName, numVal)
		}
		if logger != nil {
			logger.Warn("coerce: time column got unrecognized map → NULL",
				zap.String("column", colName), zap.Any("value", val))
		}
		return nil

	default:
		if logger != nil {
			logger.Warn("coerce: time column got unsupported type → NULL",
				zap.String("column", colName), zap.Any("value", val))
		}
		return nil
	}
}

func int64ToTime(n int64) time.Time {
	absN := n
	if n < 0 {
		absN = -n
	}
	// Nếu giá trị tuyệt đối lớn hơn 20 tỷ, giả định là milliseconds
	if absN > 20000000000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}

func float64ToTime(f float64) time.Time {
	absF := f
	if f < 0 {
		absF = -f
	}
	// Nếu giá trị tuyệt đối lớn hơn 20 tỷ, giả định là milliseconds
	if absF > 20000000000 {
		return time.UnixMilli(int64(f)).UTC()
	}
	return time.Unix(int64(f), 0).UTC()
}
```

## 2. File kiểm thử: `test/internal/service/schema_adapter_coerce_test.go`
- Viết unit test để kiểm tra tính đúng đắn cho mọi case ở trên.
