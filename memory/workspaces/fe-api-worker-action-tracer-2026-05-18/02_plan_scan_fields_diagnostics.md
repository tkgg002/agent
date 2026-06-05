# Plan — scan-fields Diagnostics

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics
**Date**: 2026-05-19
**Status**: APPROVED-by-Brain — chờ Muscle implement

## 1. Strategy

Tách "empty" thành 5 case phân biệt được bằng cách probe meta TRƯỚC khi tuyên bố "empty":

```
sample 10 docs → empty? →
  ListDatabaseNames(filter={name: sourceDB}) →
    miss → ERROR (case 2/3 — DB không tồn tại; list ALL DBs để user biết)
    hit  → ListCollectionNames(filter={name: collection}) →
      miss → ERROR (case 4 — collection không tồn tại; list collections để user biết)
      hit  → CountDocuments({}) →
        0   → ERROR (case 5 — empty thật sự)
        >0  → ERROR (case 1 race / driver bug — không thể, log + bubble)
```

Nếu `ListDatabaseNames` chính nó error → case 1 (cluster unreachable / auth fail) → ERROR riêng kèm sanitized DSN + nguyên nhân từ driver.

## 2. Files thay đổi (3 file Go + 1 test mới + 4 doc)

### 2.1. `centralized-data-service/internal/service/mongo_introspection.go`

Thêm 2 helper + 1 method enriched.

**Block 1** — `SanitizeMongoDSN` (sau `package` import block):

```go
// SanitizeMongoDSN strips credentials from a MongoDB connection URI so
// it can be embedded in logs/errors without leaking secrets. Accepts
// both "mongodb://" and "mongodb+srv://". Returns the input unchanged
// if it doesn't parse as a Mongo URI.
//
// Examples:
//   mongodb://user:pass@host:27017/?rs=rs0  →  mongodb://***@host:27017/?rs=rs0
//   mongodb+srv://u:p@cluster.mongo.net     →  mongodb+srv://***@cluster.mongo.net
//   mongodb://host:27017/                   →  mongodb://host:27017/
func SanitizeMongoDSN(uri string) string {
	for _, scheme := range []string{"mongodb+srv://", "mongodb://"} {
		if strings.HasPrefix(uri, scheme) {
			rest := uri[len(scheme):]
			at := strings.Index(rest, "@")
			if at < 0 {
				return uri
			}
			slash := strings.Index(rest, "/")
			if slash >= 0 && slash < at {
				// "@" lives after path (rare), don't touch.
				return uri
			}
			return scheme + "***@" + rest[at+1:]
		}
	}
	return uri
}
```

**Block 2** — `IntrospectDiagnosis` enum struct (sau `MongoIntrospectionService` type):

```go
// IntrospectDiagnosis describes WHY an introspection returned no fields.
// "ok"           — sample returned ≥1 doc with fields.
// "cluster_err"  — cluster unreachable / auth failed (see Err).
// "db_missing"   — DB name not present on cluster (see AvailableDBs).
// "coll_missing" — DB exists but collection not present (see AvailableColls).
// "empty"        — DB+coll exist but contains 0 documents.
// "no_fields"    — Has docs but every sampled doc had only _id / ignored keys.
type IntrospectDiagnosis struct {
	Status         string   // ok|cluster_err|db_missing|coll_missing|empty|no_fields
	SanitizedDSN   string
	DB             string
	Collection     string
	Err            error
	AvailableDBs   []string
	AvailableColls []string
	DocCount       int64
}
```

**Block 3** — `IntrospectCollectionDiagnose` method (after existing `IntrospectCollection`):

```go
// IntrospectCollectionDiagnose runs the sample-find AND, when the sample
// yields zero usable fields, probes the cluster to disambiguate WHY:
// wrong cluster, missing DB, missing collection, or truly-empty. Returns
// the same fieldMap as IntrospectCollection plus a diagnosis struct.
//
// Cheap path (sample has fields) skips the extra probes.
func (s *MongoIntrospectionService) IntrospectCollectionDiagnose(uri, dbName, collectionName string, sampleSize int) (map[string]interface{}, IntrospectDiagnosis, error) {
	sanitized := SanitizeMongoDSN(uri)
	diag := IntrospectDiagnosis{SanitizedDSN: sanitized, DB: dbName, Collection: collectionName}

	fieldMap, err := s.IntrospectCollection(uri, dbName, collectionName, sampleSize)
	if err != nil {
		diag.Status = "cluster_err"
		diag.Err = err
		return nil, diag, err
	}
	if len(fieldMap) > 0 {
		diag.Status = "ok"
		return fieldMap, diag, nil
	}

	// Slow path: confirm DB + collection existence + true doc count.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		diag.Status = "cluster_err"
		diag.Err = err
		return nil, diag, nil
	}
	defer client.Disconnect(ctx)

	allDBs, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		diag.Status = "cluster_err"
		diag.Err = err
		return nil, diag, nil
	}
	diag.AvailableDBs = allDBs
	foundDB := false
	for _, d := range allDBs {
		if d == dbName {
			foundDB = true
			break
		}
	}
	if !foundDB {
		diag.Status = "db_missing"
		return nil, diag, nil
	}

	allColls, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{})
	if err != nil {
		diag.Status = "cluster_err"
		diag.Err = err
		return nil, diag, nil
	}
	diag.AvailableColls = allColls
	foundColl := false
	for _, c := range allColls {
		if c == collectionName {
			foundColl = true
			break
		}
	}
	if !foundColl {
		diag.Status = "coll_missing"
		return nil, diag, nil
	}

	cnt, err := client.Database(dbName).Collection(collectionName).EstimatedDocumentCount(ctx)
	if err != nil {
		diag.Status = "cluster_err"
		diag.Err = err
		return nil, diag, nil
	}
	diag.DocCount = cnt
	if cnt == 0 {
		diag.Status = "empty"
	} else {
		diag.Status = "no_fields"
	}
	return nil, diag, nil
}
```

(Need `strings` import — verify file uses it; current file uses only `bson`, `mongo`, `options`, `context`, `time` — must add `strings`.)

### 2.2. `centralized-data-service/internal/handler/command_handler.go`

Refactor `scanFieldsMongoSource` (~ line 287-358) chỉ phần Introspect.

**Before** (line 333-348):

```go
sourceDB := ""
if registry.SourceDatabase != nil {
    sourceDB = *registry.SourceDatabase
}
if sourceDB == "" {
    return 0, 0, fmt.Errorf("source_database is missing in registry id=%d", registryID)
}

fieldMap, err := h.mongoSvc.IntrospectCollection(dsn, sourceDB, registry.SourceObjectName, 10)
if err != nil {
    return 0, 0, fmt.Errorf("failed to introspect mongo source: %v", err)
}

if len(fieldMap) == 0 {
    return 0, 0, fmt.Errorf("source collection %s.%s is empty; no fields found", sourceDB, registry.SourceObjectName)
}
```

**After** — gọi `IntrospectCollectionDiagnose` + log INFO upfront + branch theo `diag.Status`:

```go
sourceDB := ""
if registry.SourceDatabase != nil {
    sourceDB = *registry.SourceDatabase
}
if sourceDB == "" {
    return 0, 0, fmt.Errorf("source_database is missing in registry id=%d", registryID)
}

sanitizedDSN := service.SanitizeMongoDSN(dsn)
dispatchPath := "fallback"
if _, hit := service.ApplyConnectionOverride(&conn, h.connectionOverrides, nil); hit {
    dispatchPath = "override"
}
h.logger.Info("scan-fields mongo introspect",
    zap.String("connection_code", conn.ConnectionCode),
    zap.String("dispatch_path", dispatchPath),
    zap.String("sanitized_dsn", sanitizedDSN),
    zap.String("source_db", sourceDB),
    zap.String("collection", registry.SourceObjectName),
    zap.Int64("registry_id", registryID),
)

fieldMap, diag, err := h.mongoSvc.IntrospectCollectionDiagnose(dsn, sourceDB, registry.SourceObjectName, 10)
if err != nil || diag.Status == "cluster_err" {
    cause := ""
    if diag.Err != nil {
        cause = diag.Err.Error()
    } else if err != nil {
        cause = err.Error()
    }
    h.logger.Error("scan-fields cluster unreachable",
        zap.String("connection_code", conn.ConnectionCode),
        zap.String("sanitized_dsn", sanitizedDSN),
        zap.String("err", cause),
    )
    return 0, 0, fmt.Errorf("mongo cluster unreachable for connection_code=%s sanitized_dsn=%s: %s",
        conn.ConnectionCode, sanitizedDSN, cause)
}

switch diag.Status {
case "db_missing":
    h.logger.Warn("scan-fields db missing",
        zap.String("connection_code", conn.ConnectionCode),
        zap.String("sanitized_dsn", sanitizedDSN),
        zap.String("source_db", sourceDB),
        zap.Strings("available_dbs", diag.AvailableDBs),
    )
    return 0, 0, fmt.Errorf("source database %q not found on connection_code=%s (sanitized_dsn=%s); available DBs on cluster: %v",
        sourceDB, conn.ConnectionCode, sanitizedDSN, diag.AvailableDBs)

case "coll_missing":
    sample := diag.AvailableColls
    if len(sample) > 50 {
        sample = sample[:50]
    }
    h.logger.Warn("scan-fields collection missing",
        zap.String("connection_code", conn.ConnectionCode),
        zap.String("sanitized_dsn", sanitizedDSN),
        zap.String("source_db", sourceDB),
        zap.String("collection", registry.SourceObjectName),
        zap.Strings("available_collections_first50", sample),
    )
    return 0, 0, fmt.Errorf("collection %q not found in database %q on connection_code=%s (sanitized_dsn=%s); available collections (first 50): %v",
        registry.SourceObjectName, sourceDB, conn.ConnectionCode, sanitizedDSN, sample)

case "empty":
    h.logger.Warn("scan-fields collection empty",
        zap.String("connection_code", conn.ConnectionCode),
        zap.String("sanitized_dsn", sanitizedDSN),
        zap.String("source_db", sourceDB),
        zap.String("collection", registry.SourceObjectName),
        zap.Int64("doc_count", diag.DocCount),
    )
    return 0, 0, fmt.Errorf("collection %s.%s exists on connection_code=%s but contains 0 documents (sanitized_dsn=%s); nothing to sample — load data into the source then retry",
        sourceDB, registry.SourceObjectName, conn.ConnectionCode, sanitizedDSN)

case "no_fields":
    h.logger.Warn("scan-fields no fields",
        zap.String("connection_code", conn.ConnectionCode),
        zap.String("sanitized_dsn", sanitizedDSN),
        zap.String("source_db", sourceDB),
        zap.String("collection", registry.SourceObjectName),
        zap.Int64("doc_count", diag.DocCount),
    )
    return 0, 0, fmt.Errorf("collection %s.%s on connection_code=%s has %d documents but sampled docs contain no usable fields (only _id)",
        sourceDB, registry.SourceObjectName, conn.ConnectionCode, diag.DocCount)
}

// diag.Status == "ok" — fieldMap has fields. Proceed.
```

### 2.3. `centralized-data-service/internal/service/mongo_introspection_test.go` (NEW)

Test cho `SanitizeMongoDSN` — 4 case không cần Mongo runtime.

```go
package service

import "testing"

func TestSanitizeMongoDSN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no_creds", "mongodb://host:27017/", "mongodb://host:27017/"},
		{"basic_auth", "mongodb://user:pass@host:27017/?rs=rs0", "mongodb://***@host:27017/?rs=rs0"},
		{"srv_auth", "mongodb+srv://u:p@cluster.mongo.net", "mongodb+srv://***@cluster.mongo.net"},
		{"non_mongo_passthrough", "postgresql://u:p@h:5432/db", "postgresql://u:p@h:5432/db"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeMongoDSN(tc.in)
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}
```

### 2.4. Workspace docs

- `01_requirements_scan_fields_diagnostics.md` (đã viết)
- `02_plan_scan_fields_diagnostics.md` (file này)
- `03_implementation_scan_fields_diagnostics.md` — sau khi run
- `08_tasks_scan_fields_diagnostics.md` — checklist
- `09_tasks_solution_scan_fields_diagnostics.md` — diff verbatim
- `report_scan_fields_diagnostics.md` — TL;DR + verify

## 3. Risks & rejected alternatives

**Rejected — Auto-create DB**: probe → tự tạo DB nếu missing. Bị từ chối: dữ liệu nguồn không thuộc team này; tạo silently sẽ che bug Connector setup.

**Rejected — đổi `IntrospectCollection` signature**: sẽ break test/caller khác (`DiscoverDatabases`, `DiscoverCollections`). Method mới riêng `IntrospectCollectionDiagnose` an toàn hơn.

**Risk — ListDatabaseNames quyền**: User Mongo có thể không có quyền `listDatabases` (admin role). Mitigation: nếu probe fail → `cluster_err` với err verbatim → user thấy `not authorized on admin to execute command` → biết cần grant role. Không silent ignore.

**Risk — Large collection list**: cluster có 10K collections. Mitigation: log `first50` + dùng `[:50]` trong error message. Không truncate trong `IntrospectDiagnosis` struct (caller chọn truncate khi format).

## 4. Verify steps

```bash
cd centralized-data-service
go build ./...
go vet ./...
go test -count=1 ./internal/service/... ./internal/handler/...
```

User action sau khi merge:
1. Ctrl-C worker `tty003` → `go run cmd/worker/main.go`.
2. Click Scan Fields lại trên `export-jobs`.
3. Đọc worker log — sẽ thấy 1 trong 5 message phân biệt thay vì "is empty".
4. Báo lại log để Muscle xác định DB/host/permission/data đâu sai.
