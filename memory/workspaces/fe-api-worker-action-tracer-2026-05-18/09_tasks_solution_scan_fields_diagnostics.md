# Solution — scan-fields Diagnostics

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics
**Date**: 2026-05-19

## Summary

3 file Go thay đổi (worker only). 0 file CMS. 0 SQL migration. 0 FE.

## Diff verbatim

### `centralized-data-service/internal/service/mongo_introspection.go`

- import block: thêm `strings`, sort lại alphabetical.
- top-level: thêm `SanitizeMongoDSN(uri string) string` — strip credentials.
- top-level: thêm `IntrospectDiagnosis` struct với 6 status code.
- method: thêm `(s *MongoIntrospectionService) IntrospectCollectionDiagnose(uri, db, coll, sampleSize) (map, IntrospectDiagnosis, error)`.
- method cũ `IntrospectCollection` GIỮ NGUYÊN — caller khác (DiscoverCollections etc.) không bị ảnh hưởng.

### `centralized-data-service/internal/service/mongo_introspection_test.go` (NEW)

- `TestSanitizeMongoDSN` — 6 case (no_creds, basic_auth, srv_auth, non_mongo_passthrough, empty, only_host_no_at).

### `centralized-data-service/internal/handler/command_handler.go`

- Hàm `scanFieldsMongoSource`: thay block introspect+empty-check bằng:
  - Log INFO upfront với `connection_code`/`dispatch_path`/`sanitized_dsn`.
  - Gọi `IntrospectCollectionDiagnose`.
  - 5-case switch: `cluster_err` / `db_missing` / `coll_missing` / `empty` / `no_fields` — mỗi case có error message phân biệt + log structured.
  - Fall-through `ok` tiếp tục `processDiscoveryRows` như cũ.

## Khẳng định KHÔNG break core

| Handler | Có touch không? | Lý do |
|---|---|---|
| `HandleScanFields` / `scanFieldsDebezium` / `scanFieldsMongoSource` | **CÓ** — chỉ block error path | Diagnostic refactor — fieldMap khi `ok` xử lý y nguyên. |
| `HandleCreateDefaultColumns` | **KHÔNG** | Không đụng tới file/function này. |
| `HandleBatchTransform` | **KHÔNG** | Không đụng. |
| `HandleDebeziumSignal` / `HandleDebeziumSnapshot` | **KHÔNG** | Không đụng. |
| `HandleRegister` (CMS-side) | **KHÔNG** | Worker không có handler register. |
| `MetadataRegistryService` / `connection_overrides` | **KHÔNG** | Chỉ READ qua `ApplyConnectionOverride`. |
| `IntrospectCollection` (old method) | **KHÔNG** | Giữ nguyên signature + body. |

## Verify

| # | Command | Result |
|---|---|---|
| 1 | `go build ./...` | EXIT=0 |
| 2 | `go vet ./...` | EXIT=0 |
| 3 | `go test -count=1 ./internal/service/... -run TestSanitizeMongoDSN -v` | 6/6 subtest PASS, 0.299s |
| 4 | `go test -count=1 ./...` (whole worker module) | ALL packages PASS — service 1.338s, handler 3.332s + 8 packages khác |
