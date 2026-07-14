# Hồ sơ giải pháp kỹ thuật cụ thể (Technical Solutions)

Tài liệu này mô tả chi tiết các đoạn code cần thay đổi để tích hợp granular OTel child spans và cơ chế smart tracing (bỏ qua trace cho các window sạch nhưng giữ trace chi tiết cho các window bị drifted).

## 1. File `pkgs/observability/trace_helpers.go`

```diff
diff --git a/pkgs/observability/trace_helpers.go b/pkgs/observability/trace_helpers.go
--- a/pkgs/observability/trace_helpers.go
+++ b/pkgs/observability/trace_helpers.go
@@ -126,3 +126,8 @@ func ContextWithSkipTrace(ctx context.Context) context.Context {
 
+// ContextWithoutSkipTrace returns a new context with the skip trace flag cleared.
+func ContextWithoutSkipTrace(ctx context.Context) context.Context {
+	return context.WithValue(ctx, skipTraceKey, false)
+}
+
 // IsTraceSkipped returns true if the context contains the skip trace flag.
 func IsTraceSkipped(ctx context.Context) bool {
```

---

## 2. File `internal/service/recon/recon_tier_a.go`

```diff
diff --git a/internal/service/recon/recon_tier_a.go b/internal/service/recon/recon_tier_a.go
--- a/internal/service/recon/recon_tier_a.go
+++ b/internal/service/recon/recon_tier_a.go
@@ -787,3 +787,4 @@ for _, w := range windows {
 		driftedWindows++
 
-		ctxDrift, spanDrift := observability.ChildSpan(ctxLoop, "cdc.recon.drift_drill_down",
+		ctxDrift := observability.ContextWithoutSkipTrace(ctxLoop)
+		ctxDrift, spanDrift := observability.ChildSpan(ctxDrift, "cdc.recon.drift_drill_down",
```

---

## 3. File `internal/service/recon/recon_tier_b.go`

```diff
diff --git a/internal/service/recon/recon_tier_b.go b/internal/service/recon/recon_tier_b.go
--- a/internal/service/recon/recon_tier_b.go
+++ b/internal/service/recon/recon_tier_b.go
@@ -202,3 +202,4 @@ for k := range bucketKeys {
 
-		ctxDrift, spanDrift := observability.ChildSpan(ctxLoop, "cdc.recon.drift_drill_down_b",
+		ctxDrift := observability.ContextWithoutSkipTrace(ctxLoop)
+		ctxDrift, spanDrift := observability.ChildSpan(ctxDrift, "cdc.recon.drift_drill_down_b",
@@ -414,3 +415,4 @@ for k := range bucketKeys {
 
-		ctxDrift, spanDrift := observability.ChildSpan(ctxLoop, "cdc.recon.drift_drill_down_b",
+		ctxDrift := observability.ContextWithoutSkipTrace(ctxLoop)
+		ctxDrift, spanDrift := observability.ChildSpan(ctxDrift, "cdc.recon.drift_drill_down_b",
```
