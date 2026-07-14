# Giải pháp Kỹ thuật: Fix P0 Issues trong Recon Module

Báo cáo này mô tả chi tiết các phần cần chỉnh sửa trong source code Go để sửa các lỗi Critical P0 đã phát hiện.

---

## 1. Sửa lỗi SQL Injection (P0)

### File: `internal/handler/recon/recon_execute_heal_handler.go`
- **Mục tiêu**: Thay thế việc định dạng chuỗi `%q` không an toàn bằng hàm trích dẫn an toàn `quoteRelation` tương thích PostgreSQL.
- **Chi tiết thay đổi**:
  - Khai báo hàm helper `quoteRelation` nội bộ trong package `recon`.
  - Thay đổi dòng build table reference động tại hàm `mapGpayToSourceIDs`.

```go
// Thêm helper quoteRelation vào file (hoặc trong recon_base_handler.go)
func quoteRelation(s string) string {
	quoteIdent := func(v string) string {
		return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
	}
	if i := strings.IndexByte(s, '.'); i > 0 {
		return quoteIdent(s[:i]) + "." + quoteIdent(s[i+1:])
	}
	return quoteIdent(s)
}
```

Tại hàm `mapGpayToSourceIDs`:
```go
// TRƯỚC:
qualified := fmt.Sprintf(`%q.%q`, parts[0], parts[1])

// SAU:
qualified := quoteRelation(shadowRel)
```

---

## 2. Sửa lỗi Context Key kiểu string (P0)

### File: `internal/service/recon/recon_models.go`
- **Mục tiêu**: Định nghĩa các keys kiểu struct ẩn và cung cấp accessors an toàn về kiểu để đọc/ghi context.
- **Chi tiết thay đổi**: Thêm vào cuối file:

```go
type manualLookbackKey struct{}
type coldLookbackKey struct{}

func WithManualLookback(ctx context.Context, val bool) context.Context {
	return context.WithValue(ctx, manualLookbackKey{}, val)
}

func GetManualLookback(ctx context.Context) (bool, bool) {
	val, ok := ctx.Value(manualLookbackKey{}).(bool)
	return val, ok
}

func WithColdLookback(ctx context.Context, val bool) context.Context {
	return context.WithValue(ctx, coldLookbackKey{}, val)
}

func GetColdLookback(ctx context.Context) (bool, bool) {
	val, ok := ctx.Value(coldLookbackKey{}).(bool)
	return val, ok
}
```

### File: `internal/service/recon/recon_engine.go`
- **Chi tiết thay đổi**: Tại hàm `effectiveLookback`:
```go
// TRƯỚC:
if val, ok := ctx.Value("cold_lookback").(bool); ok && val

// SAU:
if val, ok := GetColdLookback(ctx); ok && val
```

### File: `internal/service/recon/recon_tier_a.go`
- **Chi tiết thay đổi**: Tại hàm `RunSmokeCheck` hoặc `diffIDTsSegmentA` (dòng 359):
```go
// TRƯỚC:
if val, ok := ctx.Value("manual_lookback").(bool); ok && val

// SAU:
if val, ok := GetManualLookback(ctx); ok && val
```

### File: `internal/handler/recon/recon_check_handler.go`
- **Chi tiết thay đổi**: Thay các chỗ gán context key bằng string thành:
```go
// TRƯỚC:
ctx = context.WithValue(ctx, "manual_lookback", true)
ctx = context.WithValue(ctx, "cold_lookback", true)

// SAU (sử dụng import servicerecon):
ctx = servicerecon.WithManualLookback(ctx, true)
ctx = servicerecon.WithColdLookback(ctx, true)
```

### File: `internal/handler/recon/recon_check_heal_handler.go`
- **Chi tiết thay đổi**: Tương tự như check handler, thay đổi sang dùng accessor:
```go
// TRƯỚC:
ctx = context.WithValue(ctx, "cold_lookback", true)
ctx = context.WithValue(ctx, "manual_lookback", true)

// SAU:
ctx = servicerecon.WithColdLookback(ctx, true)
ctx = servicerecon.WithManualLookback(ctx, true)
```

---

## 3. Sửa lỗi ShadowPrefix Hardcode (P0)

### File: `internal/handler/recon/recon_base_handler.go`
- **Mục tiêu**: Dùng package `naming` để lấy ShadowPrefix thay vì hardcode.
- **Chi tiết thay đổi**:
  - Import `"centralized-data-service/internal/naming"`
  - Sửa hằng số `ShadowPrefix` thành một biến lấy từ naming package hoặc cập nhật callsites sử dụng `naming.ShadowSchemaPrefix()`.
