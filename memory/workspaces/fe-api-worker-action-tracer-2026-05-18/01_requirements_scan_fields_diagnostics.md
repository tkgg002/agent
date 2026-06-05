# Requirements — scan-fields Diagnostics (Mongo empty error)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / scan_fields_diagnostics
**Date**: 2026-05-19
**Owner**: Muscle (CC Opus 4.7)
**Status**: DRAFT — chờ implement

## 1. User report (verbatim)

> `source collection centralized-export-service.export-jobs is empty; no fields found`
> 16:19:35 19/5/2026 scan-fields
> hiện scanfile lại lỗi ở cả dev & local

## 2. Ngữ cảnh

- Subject `cdc.cmd.scan-fields` → `HandleScanFields` → `scanFieldsDebezium` → fallback nhánh V2 → `scanFieldsMongoSource` (`internal/handler/command_handler.go:287-358`).
- `scanFieldsMongoSource` build DSN từ 1 trong 2 path:
  - **Override hit**: `service.ApplyConnectionOverride(&conn, h.connectionOverrides, h.logger)` — match theo `lower(connection_code)`.
  - **Fallback**: `conn.Host` + `conn.Port` → `mongodb://host:port/` hoặc giữ nguyên URI nếu host đã có scheme `mongodb://` / `mongodb+srv://`.
- DSN được pass vào `service.MongoIntrospectionService.IntrospectCollection(uri, sourceDB, registry.SourceObjectName, 10)` (`internal/service/mongo_introspection.go:60-95`).
- Method này `mongo.Connect` → `Database(dbName).Collection(collectionName).Find(bson.M{}, SetLimit(10))` → trả `fieldMap` (sample doc). Nếu cursor 0 doc → trả map rỗng, KHÔNG error.
- Caller error chung: `"source collection %s.%s is empty; no fields found"`.

## 3. Vấn đề (Failure mode ambiguity)

Cùng 1 error message generic cho 4 root cause KHÁC NHAU — user/dev không cách nào phân biệt:

| # | Root cause | Symptom hiện tại | Cần message |
|---|---|---|---|
| 1 | DSN trỏ sai cluster (override map sai, host typo, port sai, port forward chưa start) | "is empty" | "cluster unreachable (host=X, sanitized DSN=Y, err=...)" |
| 2 | `connection_code` không có override + `conn.Host` của row khác cluster thật sự | "is empty" | "DB 'X' not found on cluster; available DBs: [...]" |
| 3 | Cluster đúng, DB sai (user gõ nhầm `source_database`) | "is empty" | "DB 'X' not found on cluster; available DBs: [...]" |
| 4 | Cluster đúng, DB đúng, tên collection sai (user nhầm `source_object_name`) | "is empty" | "collection 'X' not found in DB 'Y'; available collections (first 50): [...]" |
| 5 | Cluster + DB + collection ĐÚNG, nhưng collection thật sự 0 doc | "is empty" | "collection X.Y exists but contains 0 documents; nothing to sample" |

Cả 5 case hiện đều fail với cùng `len(fieldMap) == 0` → user không biết phải sửa cái gì.

## 4. Definition of Done

- [ ] `scanFieldsMongoSource` khi `fieldMap` empty: probe lần lượt `ListDatabaseNames` → `ListCollectionNames` → `CountDocuments` để biết case 1/2/3/4/5.
- [ ] Mỗi case có error message phân biệt, **bao gồm sanitized DSN** (strip `user:password@` khỏi `mongodb://`/`mongodb+srv://`).
- [ ] Log INFO trước introspect: `connection_code`, `dispatch_path` (`override`/`fallback`), sanitized DSN, `sourceDB`, `collection`.
- [ ] Log WARN/ERROR tách 5 case (zap structured fields), không chỉ chữ.
- [ ] `internal/service/mongo_introspection.go` thêm helper `IntrospectCollectionDiagnose(uri, db, coll, sampleSize) (fieldMap, diagnosis, err)` — fields gốc cộng metadata về case 1/2/3/4/5. Caller chọn dùng message thuần text.
- [ ] Helper `SanitizeMongoDSN(uri)` — strip credentials, dùng trong cả log + error message. Không leak password.
- [ ] Build + vet PASS.
- [ ] Test mới cho `SanitizeMongoDSN` (3 case: no-cred, basic auth, srv với cred).
- [ ] `go test -count=1 ./internal/handler/... ./internal/service/...` PASS.
- [ ] Workspace docs đầy đủ: `02_plan_*`, `03_implementation_*`, `08_tasks_*`, `09_tasks_solution_*`, `report_scan_fields_diagnostics.md`.
- [ ] `05_progress.md` APPEND timestamped + agent + model.
- [ ] Lesson global hóa cho `agent/memory/global/lessons.md` (pattern "Generic Empty Error Hides Multi-Cause Failure").
- [ ] User action: restart worker, click Scan Fields lại, đọc log mới để confirm root cause thật.

## 5. Out of scope

- Auto-discovery của DB/collection alternatives qua FE (chỉ log để dev đọc).
- Resize sample size (vẫn 10 doc).
- Postgres/MySQL introspection — chỉ Mongo path.
- DB DSN validation lúc Connector setup (overlay configure-time check) — riêng phase khác.

## 6. Non-functional

- **Backward compat**: cũ `IntrospectCollection` giữ nguyên signature, không break test khác.
- **Security**: Tuyệt đối không log raw URI có password — bắt buộc qua `SanitizeMongoDSN`.
- **Performance**: 3 probe DB call thêm chỉ chạy khi `fieldMap` empty (slow path). Happy path không tăng latency.
- **Idempotent**: Probe không write, chỉ read meta — gọi nhiều lần OK.
