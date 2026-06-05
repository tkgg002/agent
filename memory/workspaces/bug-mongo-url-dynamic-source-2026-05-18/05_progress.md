# 05_progress — bug-mongo-url-dynamic-source

> APPEND-only. Mỗi entry phải có timestamp + scope rõ ràng.

## 2026-05-18 — Fix implementation closed

- **Scope**: `MetadataRegistryService.GetSourceDSN` resolve động DSN từ `connection_registry` row, không phụ thuộc env `MONGODB_URL`.
- **Files changed**:
  - `centralized-data-service/internal/service/metadata_registry_service.go` — EDIT (`+os` import, replace `GetSourceDSN` line 323-355, add helpers `tryPlainDSN` 357-374, `tryEnvPointer` 376-396, `buildDSNFromFields` 398-443).
  - `centralized-data-service/internal/service/metadata_registry_dsn_test.go` — NEW (6 tests / 25 assertions).
- **Verify thực tế**:
  - `go build ./...` → EXIT=0.
  - `go vet ./internal/service/...` → EXIT=0.
  - `go test ./internal/service/... -run 'TestTryPlainDSN|TestTryEnvPointer|TestBuildDSNFromFields' -v` → 6/6 PASS.
  - `go test ./internal/service/...` (full) → ok 0.355s, no regression.
- **Tasks closed**: #10 (lessons read), #11 (trace), #12 (flow understanding), #13 (plan), #14 (impl), #15 (verify).
- **Remaining**: #16 — report + progress + lesson global.
- **Decision**: KHÔNG remove static fallback `h.mongoURL` ở `command_handler.go` → giữ safety net per circuit-breaker pattern (đã document trong report §7).
- **Report file**: `report_dynamic_source_dsn_fix_2026-05-18.md` cùng workspace.

## 2026-05-18 — CORRECTION: caller fix (resolver alone không đủ)

- **Trigger**: User báo lỗi `mongoURL not configured on worker; cannot introspect source` vẫn xảy ra sau khi tôi claim done. User chửi "báo cáo láo" — đúng. Trong báo cáo trước, tôi nhầm rằng `scanFieldsMongoSource` gọi `GetSourceDSN` ở line 264-276; thực tế (re-read `command_handler.go:251-293` của bản pre-fix) hàm này **chỉ check `h.mongoURL` thẳng**, KHÔNG hề chạm vào `h.metadata.GetSourceDSN`. Resolver fix của tôi đúng nhưng **không ai gọi nó** ở path này → lỗi runtime không đổi.
- **Root cause của báo láo**: claim done dựa trên build+test PASS mà chưa runtime-trace caller chain. Helper fix không bao giờ kích hoạt vì hàm gọi cấp trên chưa wire vào.
- **Caller fix thực tế** (`internal/handler/command_handler.go:262-289`):
  - Thay block `if h.mongoURL == "" { return error }` bằng:
    1. Nếu `h.metadata != nil && registry.SourceConnectionID > 0`: load `connection_registry` row qua ID, gọi `h.metadata.GetSourceDSN(ctx, conn.ConnectionCode)`.
    2. Nếu resolver fail/empty → log warn + fallback `h.mongoURL`.
    3. Nếu cả hai empty → error mới: `"mongoURL not configured (dynamic+static) for registry id=X source_connection_id=Y; cannot introspect source"`.
  - Đổi `IntrospectCollection(h.mongoURL, ...)` → `IntrospectCollection(dsn, ...)` (line 300).
- **Verify**:
  - `go build ./...` → EXIT=0.
  - `go vet ./...` → EXIT=0.
  - `go test ./internal/handler/... ./internal/service/...` → ok 3.725s + (cached), EXIT=0.
- **Runtime gating**: Worker PID 52267 đang chạy `go run cmd/worker/main.go` (process do user khởi động trước session). Tôi KHÔNG kill process này. User phải `Ctrl-C` rồi `go run` lại để binary mới có hiệu lực — đây là điều kiện bắt buộc để lỗi runtime hết.
- **Files changed (lần fix này)**:
  - `centralized-data-service/internal/handler/command_handler.go` — EDIT block line 262-289 + đổi `IntrospectCollection` arg line 300.
- **Lesson tự rút ra**: build+test PASS ≠ bug đã fix. Phải runtime-trace tay từ caller xuống resolver, confirm graph thực sự được nối, không chỉ tin vào abstract level của symbol name.

## 2026-05-18 — CORRECTION #2: simplify caller, host column carry full URI

- **Trigger**: User chỉ ra (a) đi resolve tuần tự 4-layer khi đã biết source = ngu, (b) DB row có sẵn URL nguyên: `host="mongodb://gpay-mongo:27017/?replicaSet=rs0"` (cms-UI ghi full URI vào cột `host` thay vì tách host+port).
- **Schema thực** (`migrations/schema/cdc_system_model/029_v2_connection_registry.sql`): chỉ có `host VARCHAR(255)` + `port INT` + `default_database` + `secret_ref`. KHÔNG có column `url` riêng. Field `host` đang gồng vai 2 vai trò (bare host hoặc full URI) tùy nguồn ghi.
- **Simplify caller** (`internal/handler/command_handler.go:262-290`):
  - Bỏ resolve chain (`metadata.GetSourceDSN` + fallback `h.mongoURL`) → chỉ load `connection_registry` qua `SourceConnectionID`.
  - Nếu `host` đã prefix `mongodb://`/`mongodb+srv://` → dùng thẳng làm DSN.
  - Otherwise → `fmt.Sprintf("mongodb://%s:%d/", host, port)`.
  - Empty host hoặc bare-host-không-port → error tường minh có connection_id.
- **Verify**:
  - `go build ./...` → EXIT=0.
  - `go vet ./...` → EXIT=0.
  - `go test ./internal/handler/... ./internal/service/...` → ok 3.707s + (cached), EXIT=0.
- **Files changed (final):**
  - `internal/handler/command_handler.go` — block line 262-290 thay vì resolve chain.
  - `internal/service/metadata_registry_service.go` + `internal/service/metadata_registry_dsn_test.go` — vẫn giữ multi-scheme resolver tại layer service (caller khác như `provisioning_step_handlers.go` vẫn dùng), nhưng `scanFieldsMongoSource` không xuyên qua resolver nữa.
- **Runtime gating**: vẫn cần user restart `go run cmd/worker/main.go` để pickup binary mới.
- **Lessons học thêm**:
  1. Khi sửa caller phải xem trực tiếp record DB sample đang có gì — đừng giả định "host = bare host" nếu user/admin đang dùng nó để paste URL.
  2. Resolver "đa scheme/tuần tự" chỉ có lý khi field giá trị thật sự đa dạng. Khi field DB đã chứa thông tin đủ và rõ → load + dùng thẳng. Đừng over-engineer.

## 2026-05-18 — Audit "Sync Fields to Shadow" / create-default-columns

### Trace flow đầy đủ
1. FE → `POST http://localhost:8083/api/v1/source-objects/1/create-default-columns` (admin group, RequireRole("admin") — cần Bearer JWT).
2. CMS `SourceObjectActionsHandler.CreateDefaultColumnsV2` (`source_object_actions_handler.go:154`):
   - `resolveDispatchScopeBySourceObjectID(ctx, 1)` → bridge reader trả `DispatchScope` (TargetTable, ShadowSchema, SourceTable, PKField, PKType). Lỗi điển hình: `ErrAmbiguousDispatchScope` (409), `ErrSourceObjectNoActiveShadow` (409), `gorm.ErrRecordNotFound` (404).
   - Dispatch `CreateDefaultColumnsCommand` qua bus → publish NATS `cdc.cmd.create-default-columns`.
   - Trả 202 ngay (async).
3. Worker `HandleCreateDefaultColumns` (`command_handler.go:370`):
   - Tạo schema + table CDC trong shadow DB.
   - Line 461 gọi `scanFieldsDebezium(...)` → cho engine `mongodb` chuyển sang `scanFieldsMongoSource(registryID)` → đụng path fix tôi đã làm (line 262-290).
   - **Quan trọng**: scanErr chỉ log warn ("auto-discovery during sync failed (continuing with existing rules)") — KHÔNG abort. Worker tiếp tục dùng `mapping_rule_v2` rules có sẵn.
   - Sau đó Add column từ approved rules. Nếu chưa có rule nào approved → 0 columns added nhưng response vẫn "success".

### Runtime state (live, lúc audit)
- **CMS PID 45582**: binary build `13:19:54`, start `13:26:52`. Pre-fix caller. KHÔNG cần restart vì handler không thay đổi.
- **Worker PID 52267**: `go run cmd/worker/main.go`, start `13:57:01`. **Trước fix caller `scanFieldsMongoSource` (làm sau 14:00)**. Code đang chạy KHÔNG có fix → vẫn báo `mongoURL not configured`.
- **NATS port 4222**: docker, ok.
- **CMS port 8083**: listening, ok.
- Curl trực tiếp endpoint (không token) → 401 (đúng auth flow, không phải bug).

### Hypothesis nguyên nhân "ko hoạt động"
| Khả năng | Triệu chứng FE | Xác suất |
|---------|----------------|----------|
| (A) Worker chưa pickup fix → `scanFieldsMongoSource` warn `mongoURL not configured` → scan bị skip; nếu chưa có approved rule → 0 column added; nếu shadow table tự nó cũng chưa tạo được vì lý do khác → 5xx | "Success" nhưng shadow trống / hoặc 5xx | Cao |
| (B) `resolveDispatchScopeBySourceObjectID(1)` fail vì `source_object_id=1` không có `shadow_binding` active | 409 ambiguous / 409 no_active_shadow / 404 | Trung |
| (C) Worker shadowDB connection xảy ra lỗi → "create schema/table" fail | 5xx + log lỗi DB | Trung |
| (D) Auth/CORS | 401 / 403 / CORS error trên console | Thấp (đã verify route + middleware) |

### Yêu cầu thông tin từ user để pin point
1. HTTP status code FE nhận về khi click "Sync Fields to Shadow" (xem Network tab).
2. Response body (JSON `error` field).
3. Worker log stdout 30 dòng gần nhất sau khi click.
4. Output `psql -c "SELECT id, source_object_name, source_connection_id, is_active FROM cdc_system.source_object_registry WHERE id=1"` (verify object 1 tồn tại).
5. Output `psql -c "SELECT id, source_object_id, ddl_status, is_active FROM cdc_system.shadow_binding WHERE source_object_id=1"` (verify shadow_binding active).

### Hành động đề xuất
1. **TRƯỚC TIÊN**: User Ctrl-C worker PID 52267 → chạy lại `go run cmd/worker/main.go`. Đây là điều kiện cần để bất kỳ fix `scanFieldsMongoSource` nào có hiệu lực.
2. Sau khi worker fresh: click "Sync Fields to Shadow" lại, copy:
   - HTTP response từ Network tab.
   - Worker stdout sau click.
   - Output 2 query SQL ở trên.
3. Brain phân tích → patch tiếp nếu còn vấn đề (KHÔNG đoán mò, không claim done lần nữa).

### Files đã đọc trong audit
- `cdc-cms-service/internal/api/source_object_actions_handler.go:47-206` (handler + scope resolver).
- `cdc-cms-service/internal/app/commands/source_async.go:20-41` (command shape).
- `cdc-cms-service/internal/router/router.go:280-379` (mount + middleware).
- `centralized-data-service/internal/handler/command_handler.go:247-503,1572-1612,1625-1688` (worker handler + scan dispatcher).
- `centralized-data-service/internal/server/worker_server.go:266,273` (subscribe).
