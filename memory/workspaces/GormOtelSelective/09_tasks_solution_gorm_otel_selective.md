# Technical Solution: GORM OpenTelemetry Selective Tracing (HTTP API Support)

Tài liệu hướng dẫn chi tiết code thay đổi để khôi phục GORM spans cho HTTP APIs của CMS và Admin API.

## 1. centralized-data-service/pkgs/observability/trace_helpers.go
**File**: `pkgs/observability/trace_helpers.go`

- Thêm `"cdc"` vào map `enabledDBTraceModules` whitelist:
  ```go
  var enabledDBTraceModules = map[string]bool{
  	"cdc":             true, // Bật trace GORM cho HTTP APIs
  	"recon_heal":      true,
  	"batch_transform": true,
  	"discover":        true,
  	"scan_raw":        true,
  }
  ```

## 2. cdc-cms-service/pkgs/observability/otel.go
**File**: `pkgs/observability/otel.go`

- Thêm `"cdc"` vào map `enabledDBTraceModules` whitelist:
  ```go
  var enabledDBTraceModules = map[string]bool{
  	"cdc":             true, // Bật trace GORM cho HTTP APIs
  	"recon_heal":      true,
  	"batch_transform": true,
  	"discover":        true,
  	"scan_raw":        true,
  }
  ```

## 3. cdc-cms-service/internal/middleware/http_tracer.go
**File**: `internal/middleware/http_tracer.go`

- Import `"cdc-cms-service/pkgs/observability"`.
- Bọc `"cdc"` module cho context của HTTP request:
  ```go
  		ctx, span := tracer.Start(
  			c.UserContext(),
  			spanName,
  			trace.WithSpanKind(trace.SpanKindServer),
  			trace.WithAttributes(
  				semconv.HTTPMethod(method),
  				semconv.HTTPTarget(c.OriginalURL()),
  				semconv.HTTPRoute(path),
  				semconv.NetHostName(c.Hostname()),
  			),
  		)
  		ctx = observability.WithDBTraceModule(ctx, "cdc")
  		c.SetUserContext(ctx)
  ```

## 4. centralized-data-service/internal/admin/otel_middleware.go
**File**: `internal/admin/otel_middleware.go`

- Import `"centralized-data-service/pkgs/observability"`.
- Bọc `"cdc"` module cho context của HTTP request:
  ```go
  		ctx, span := tracer.Start(ctx, spanName,
  			trace.WithSpanKind(trace.SpanKindServer),
  			trace.WithAttributes(
  				attribute.String("http.method", method),
  				attribute.String("http.target", c.Request.RequestURI),
  				attribute.String("http.route", path),
  				attribute.String("net.host.name", c.Request.Host),
  			),
  		)
  		ctx = observability.WithDBTraceModule(ctx, "cdc")
  		defer span.End()

  		// Pass the trace context to Gin request context.
  		c.Request = c.Request.WithContext(ctx)
  ```
