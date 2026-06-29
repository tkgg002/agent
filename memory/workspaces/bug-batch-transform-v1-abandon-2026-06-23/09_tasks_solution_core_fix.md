# Technical Solution: Core Fix for Batch Transform Schema Drift

Đây là chi tiết thay đổi code cần thực hiện trên file `internal/handler/shadow/batch_transform_handler.go`.

## File Target
- File Path: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go`

## Code Diff Dự Kiến

```diff
diff --git a/internal/handler/shadow/batch_transform_handler.go b/internal/handler/shadow/batch_transform_handler.go
index 123456..789abc 100644
--- a/internal/handler/shadow/batch_transform_handler.go
+++ b/internal/handler/shadow/batch_transform_handler.go
@@ -95,6 +95,11 @@ func (h *BatchTransformHandler) HandleBatchTransform(msg *nats.Msg) {
 		return
 	}
 
+	execDB := h.DB
+	if h.shadowDB != nil {
+		execDB = h.shadowDB
+	}
+
 	var setClauses []string
 	var whereClauses []string
 	seenCols := make(map[string]struct{}, len(rules))
@@ -113,6 +118,16 @@ func (h *BatchTransformHandler) HandleBatchTransform(msg *nats.Msg) {
 			continue
 		}
 		seenCols[colKey] = struct{}{}
+
+		// Kiểm tra xem cột đích có tồn tại trong schema DB thực tế hay không
+		if !h.HasColumnInSchema(ctx, execDB, schemaName, targetTable, rule.TargetColumn) {
+			h.Logger.Warn("batch transform: target_column does not exist in db, skipping rule",
+				zap.String("table", targetTable),
+				zap.String("target_column", rule.TargetColumn),
+				zap.String("source_field", rule.SourceField),
+			)
+			continue
+		}
+
 		castExpr := metadata.BuildCastExpr(rule.SourceField, rule.DataType)
 		quotedCol := sqlutil.QuoteIdent(rule.TargetColumn)
 		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
@@ -130,11 +145,6 @@ func (h *BatchTransformHandler) HandleBatchTransform(msg *nats.Msg) {
 
 	setClauses = append(setClauses, "_updated_at = NOW()")
 
-	execDB := h.DB
-	if h.shadowDB != nil {
-		execDB = h.shadowDB
-	}
-
 	quotedTable := sqlutil.QualifiedTable(schemaName, targetTable)
 	whereExpr := strings.Join(whereClauses, " OR ")
 	setExpr := strings.Join(setClauses, ", ")
```

---

## File Target 2
- File Path: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go`

## Code Diff Dự Kiến 2

```diff
diff --git a/internal/handler/base/base_handler.go b/internal/handler/base/base_handler.go
index 112233..445566 100644
--- a/internal/handler/base/base_handler.go
+++ b/internal/handler/base/base_handler.go
@@ -3,6 +3,7 @@ package base
 import (
 	"context"
 	"encoding/json"
+	"regexp"
 	"strings"
 	"time"
 
@@ -184,14 +185,9 @@ func IsSafeIdentifier(s string) bool {
 	return true
 }
 
+var reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|BIGINT|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)
+
 func IsSafeType(t string) bool {
 	u := strings.ToUpper(strings.TrimSpace(t))
-	switch u {
-	case "TEXT", "VARCHAR", "VARCHAR(255)", "BIGINT", "INTEGER", "SMALLINT",
-		"NUMERIC", "DECIMAL", "REAL", "DOUBLE PRECISION", "BOOLEAN",
-		"TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMPTZ",
-		"DATE", "TIME", "JSONB", "JSON", "UUID":
-		return true
-	}
-	return false
+	return reTypeWhitelist.MatchString(u)
 }
```

