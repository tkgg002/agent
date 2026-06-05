# Report — scan-fields Diagnostics (Mongo empty error)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics
**Date**: 2026-05-19
**Status**: ✅ CODE COMPLETE — build/vet/test PASS — chờ user restart worker để thấy log mới.

## TL;DR

Error `"source collection X.Y is empty; no fields found"` ở `scanFieldsMongoSource` (`internal/handler/command_handler.go:347`) trước đây HỘP CHUNG 5 root cause khác nhau: (1) cluster unreachable, (2) DB không tồn tại, (3) collection không tồn tại, (4) collection thật sự 0 doc, (5) docs có nhưng chỉ có `_id`. Đã thêm probe + 5-case branching + sanitized DSN trong error/log. Pure-additive: chỉ thay block 16 dòng trong `scanFieldsMongoSource`, không touch handler khác. Build/vet/test PASS toàn bộ worker module.

## Files đã thay đổi (3 file Go)

```
centralized-data-service/internal/service/mongo_introspection.go
  + import: thêm "strings"
  + func SanitizeMongoDSN(uri string) string — top-level helper.
  + type IntrospectDiagnosis struct — 6 status code.
  + (s *MongoIntrospectionService) IntrospectCollectionDiagnose(...) — wrapper around old IntrospectCollection + slow-path probe.
  ~ Method cũ IntrospectCollection GIỮ NGUYÊN (zero-touch).

centralized-data-service/internal/service/mongo_introspection_test.go (NEW)
  + TestSanitizeMongoDSN — 6 case, không cần Mongo runtime.

centralized-data-service/internal/handler/command_handler.go
  ~ scanFieldsMongoSource (line ~341-419): block `IntrospectCollection + len(fieldMap)==0` thay bằng:
      • Log INFO upfront (connection_code, dispatch_path, sanitized_dsn, source_db, collection, registry_id).
      • Gọi IntrospectCollectionDiagnose.
      • 5-case switch: cluster_err | db_missing | coll_missing | empty | no_fields — error message phân biệt + zap structured log.
      • Fall-through ok → processDiscoveryRows như cũ.
```

## 5 trường hợp lỗi mới phân biệt được

| Status | Error message dùng được | Log structured |
|---|---|---|
| `cluster_err` | `mongo cluster unreachable for connection_code=X sanitized_dsn=Y: <driver err>` | ERROR `scan-fields cluster unreachable` + sanitized DSN |
| `db_missing` | `source database "X" not found on connection_code=Y; available DBs: [...]` | WARN `scan-fields db missing` + `available_dbs=[...]` |
| `coll_missing` | `collection "X" not found in database "Y"; available collections (first 50): [...]` | WARN `scan-fields collection missing` + `available_collections_first50=[...]` |
| `empty` | `collection X.Y exists on connection_code=Z but contains 0 documents; nothing to sample — load data into the source then retry` | WARN `scan-fields collection empty` + `doc_count=0` |
| `no_fields` | `collection X.Y has N documents but sampled docs contain no usable fields (only _id)` | WARN `scan-fields no fields` + `doc_count=N` |

## KHẲNG ĐỊNH KHÔNG BREAK CORE KHÁC

Tôi hiểu user lo regression. Đây là phạm vi cụ thể:

| Handler / Subsystem | Touch? |
|---|---|
| `scanFieldsMongoSource` (chính lỗi này) | ✅ chỉ block error path |
| `IntrospectCollection` (method cũ) | ❌ giữ nguyên byte-identical |
| `HandleCreateDefaultColumns` | ❌ |
| `HandleBatchTransform` | ❌ |
| `HandleDebeziumSignal` / `HandleDebeziumSnapshot` | ❌ |
| `MetadataRegistryService` / `connection_overrides` | ❌ chỉ READ |
| `source_object_v2_sync` (CMS) | ❌ |
| `registry_mirror` (CMS) | ❌ |
| FE | ❌ |
| SQL migration | ❌ |

Diff vật lý: 3 file worker, +138 dòng (mongo_introspection.go +88, test +27, handler +84 net thay 16). 0 dòng CMS/FE/SQL.

## Verification kết quả thật

| # | Command | Output |
|---|---|---|
| 1 | `cd centralized-data-service && go build ./...` | EXIT=0, no stderr |
| 2 | `go vet ./...` | EXIT=0, no stderr |
| 3 | `go test -count=1 ./internal/service/... -run TestSanitizeMongoDSN -v` | PASS 6/6 subtest, 0.299s |
| 4 | `go test -count=1 ./...` (whole worker) | service 1.338s, handler 3.332s, config 0.859s, activity 0.418s, admin 1.165s, sinkworker 1.816s, database 2.762s, idgen 42.345s, utils 3.205s — **TẤT CẢ PASS, 0 FAIL** |

## Activity log analysis (chain failure user vừa show)

User dump activity_log show 4 chain failure cho 2 connector (`goopay-dev` + `goopay-local`) cùng `(centralized-export-service, export-jobs)`:

| id | op | target | status | error | Diễn giải |
|---|---|---|---|---|---|
| 6/7 | register | dev/local | accepted | — | OK |
| 8/9 | cmd-create-default-columns | dev/local | success rows=0 | — | **PHẢI XEM LẠI**: `success rows_affected=0` có thể nghĩa "table chưa có rule nào để alter" — không tạo table base. |
| 13 | scan-fields | dev | error | "is empty; no fields found" | Cluster đến được; DB/collection/data nào sai — diagnostic mới sẽ phân biệt. |
| 15 | scan-fields | local | error | "context deadline ... lookup host.docker.internal: no such" | **Override map miss** cho `goopay-local` → fallback `conn.Host = host.docker.internal:27017` → worker chạy ngoài docker không resolve được. |
| 18/22/26 | cmd-batch-transform | dev | error | "no active mapping rules" | Chain effect: scan-fields fail → 0 rule approved → batch-transform không có rule để chạy. |
| 19/23/27 | cmd-batch-transform | local | skipped | "table does not exist" | Shadow table chưa được tạo physical. |

**Chuỗi nhân quả**:
1. scan-fields fail → mapping_rule_v2 chưa có row nào active.
2. create-default-columns chạy với rules_total=0 → chỉ "success" rỗng, không alter table nào.
3. batch-transform cho `dev` query mapping_rule_v2 → 0 row → `no active mapping rules`.
4. batch-transform cho `local` query shadow table → table chưa tồn tại → `table does not exist`.

→ **Tất cả đều bắt nguồn từ scan-fields fail**. Cho nên fix root chỗ này = chữa toàn chain.

## Test thủ công user cần làm

```bash
# 1. Restart worker để load code diagnostics mới.
#    Ctrl-C tty003 → cd centralized-data-service && go run cmd/worker/main.go
#    Boot log không thay đổi (chỉ thêm log path khi scan-fields chạy).

# 2. Click Scan Fields trên FE cho cả 2 connector (sd_export_jobs_dev + sd_export_jobs_local).

# 3. Đọc worker log mới. Expected 1 trong 5 dòng:
#    (a) cluster_err  → kiểm tra override map / port forward / VPN.
#    (b) db_missing   → log liệt kê available_dbs → user thấy DB đúng tên là gì.
#    (c) coll_missing → log liệt kê available_collections_first50 → user thấy tên collection thật.
#    (d) empty        → collection 0 doc, cần insert data.
#    (e) no_fields    → docs có nhưng chỉ _id, kiểm tra schema bằng Compass.

# 4. Capture log paste lại để tôi tiếp tục diagnose chain (mapping rule + batch transform).
```

## Out of scope (chưa làm — chờ user xác nhận root cause thật bằng diagnostic mới)

- Fix `connection_overrides.goopay-local` (cần user/ops decide URI).
- Tạo physical shadow table cho `sd_export_jobs_local` (cần biết schema đúng trước).
- Fix `cmd-create-default-columns` để không silent "success" khi 0 rules (cần plan riêng).

## DoD checklist

- [x] `01_requirements_scan_fields_diagnostics.md` tạo file vật lý.
- [x] `02_plan_scan_fields_diagnostics.md` tạo file vật lý, có code demo.
- [x] `03_implementation_scan_fields_diagnostics.md` tạo file vật lý.
- [x] `08_tasks_scan_fields_diagnostics.md` tạo file vật lý.
- [x] `09_tasks_solution_scan_fields_diagnostics.md` tạo file vật lý.
- [x] `report_scan_fields_diagnostics.md` tạo file vật lý (file này).
- [x] 3 file Go thay đổi đúng theo plan.
- [x] `go build` + `go vet` PASS toàn worker.
- [x] `go test -count=1 ./...` PASS toàn worker.
- [x] 0 file CMS / FE / SQL thay đổi (chứng minh không break "core").
- [ ] `05_progress.md` APPEND 5 dòng timestamped + agent + model (sắp làm).
- [ ] `agent/memory/global/lessons.md` APPEND lesson global hóa (sắp làm).
- [ ] User restart worker + retry scan-fields + paste log mới (chờ).
