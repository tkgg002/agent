# Implementation — scan-fields Diagnostics

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics
**Date**: 2026-05-19
**Status**: CODE COMPLETE — build/vet/test PASS

## File 1: `centralized-data-service/internal/service/mongo_introspection.go`

### Thay đổi 1 — Add `strings` import + reorder bson alphabet

```diff
 import (
 	"context"
+	"strings"
 	"time"

-	"go.mongodb.org/mongo-driver/mongo"
-	"go.mongodb.org/mongo-driver/mongo/options"
 	"go.mongodb.org/mongo-driver/bson"
+	"go.mongodb.org/mongo-driver/mongo"
+	"go.mongodb.org/mongo-driver/mongo/options"
 )
```

### Thay đổi 2 — Add `SanitizeMongoDSN` (top-level helper)

Strip `user:password@` từ URI cho safe logging. Hỗ trợ `mongodb://` + `mongodb+srv://`. Trả input nguyên vẹn nếu prefix không match.

### Thay đổi 3 — Add `IntrospectDiagnosis` struct (top-level type)

6 status: `ok`, `cluster_err`, `db_missing`, `coll_missing`, `empty`, `no_fields`. Mỗi status carry context-relevant fields (Err, AvailableDBs, AvailableColls, DocCount).

### Thay đổi 4 — Add `IntrospectCollectionDiagnose` method

Wraps existing `IntrospectCollection`:
1. Sample 10 doc. Nếu có field → status `ok`, return ngay (happy path, không thêm latency).
2. Nếu sample empty: `mongo.Connect` → `ListDatabaseNames` → check sourceDB exists.
3. Nếu DB miss → `db_missing` (kèm `AvailableDBs`).
4. Nếu DB hit → `ListCollectionNames` → check collection exists.
5. Nếu collection miss → `coll_missing` (kèm `AvailableColls`).
6. Nếu collection hit → `EstimatedDocumentCount`.
7. Count = 0 → `empty`; Count > 0 → `no_fields`.

## File 2: `centralized-data-service/internal/service/mongo_introspection_test.go` (NEW)

Test cho `SanitizeMongoDSN` — 6 case không cần Mongo runtime:
- `no_creds` → passthrough
- `basic_auth` → strip `user:pass@`
- `srv_auth` → strip với `mongodb+srv://`
- `non_mongo_passthrough` → postgres URI giữ nguyên
- `empty` → empty input
- `only_host_no_at` → URI không có `@` giữ nguyên

Result: 6/6 PASS, 0.299s.

## File 3: `centralized-data-service/internal/handler/command_handler.go`

### Refactor `scanFieldsMongoSource` (line 287-429)

Block trước (line 333-348):
```go
fieldMap, err := h.mongoSvc.IntrospectCollection(dsn, sourceDB, registry.SourceObjectName, 10)
if err != nil { ... }
if len(fieldMap) == 0 {
    return 0, 0, fmt.Errorf("source collection %s.%s is empty; no fields found", ...)
}
```

Block sau:
- Log INFO upfront với `connection_code` + `dispatch_path` (override/fallback) + `sanitized_dsn` + `source_db` + `collection` + `registry_id`.
- Gọi `IntrospectCollectionDiagnose`.
- 5-case switch: `cluster_err` / `db_missing` / `coll_missing` / `empty` / `no_fields`.
- Mỗi case có log riêng (zap structured) + error message phân biệt.
- `case ok` (fallthrough): tiếp tục xử lý `fieldMap` như cũ.

## Verification (real output)

| # | Command | Result |
|---|---|---|
| 1 | `cd centralized-data-service && go build ./...` | EXIT=0, no stderr |
| 2 | `go vet ./...` | EXIT=0, no stderr |
| 3 | `go test -count=1 ./internal/service/... -run TestSanitizeMongoDSN -v` | PASS 6/6 subtest, 0.299s |
| 4 | `go test -count=1 ./...` (worker module) | ALL packages PASS — service 1.338s, handler 3.332s, config 0.859s, activity 0.418s, admin 1.165s, sinkworker 1.816s, database 2.762s, idgen 42.345s, utils 3.205s |

## Trace — Cách user dùng output mới để fix

User click Scan Fields lần tới, worker log một trong 5 dòng:

```
# Case 1 — cluster_err
ERROR scan-fields cluster unreachable connection_code=goopay sanitized_dsn=mongodb://localhost:27017/ err="no reachable servers"
# → User: kiểm tra override map / port forward / VPN

# Case 2 — db_missing
WARN scan-fields db missing connection_code=goopay sanitized_dsn=mongodb://localhost:17017/?rs=rs0 source_db=centralized-export-service available_dbs=[admin local config goopay-payment-bill ...]
# → User: sửa source_database trong CMS hoặc đổi connector

# Case 3 — coll_missing
WARN scan-fields collection missing connection_code=goopay source_db=centralized-export-service collection=export-jobs available_collections_first50=[users orders ...]
# → User: sửa source_object_name hoặc tạo collection thật

# Case 4 — empty (truly empty)
WARN scan-fields collection empty connection_code=goopay source_db=centralized-export-service collection=export-jobs doc_count=0
# → User: load data vào collection rồi retry

# Case 5 — no_fields (count > 0 nhưng sample chỉ có _id)
WARN scan-fields no fields connection_code=goopay source_db=centralized-export-service collection=export-jobs doc_count=12345
# → User: doc bất thường, mở Mongo Compass kiểm tra schema
```

Phân biệt 5 case này → user/dev biết chính xác cần sửa lớp nào (config / DB schema / data ingest / Mongo permission).
