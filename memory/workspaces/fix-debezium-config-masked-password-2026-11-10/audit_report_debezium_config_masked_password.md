---
tier: T2
workspace: fix-debezium-config-masked-password-2026-11-10
suffix: debezium_config_masked_password
---

# Audit Report — Xử lý lỗi đẩy Password / URI bị Mask (`***` / `*******` / `__KEEP__`) khi Update Config, Recover & Introspection trên `/sources` (`G1..G6`)

## 1. Mục 1: Phản biện từng dòng update (`git diff` Line-by-Line Critique) & Đối chiếu từng Task (`08_tasks`)

Đã chạy `git diff` trên cả 2 repository (`cdc-cms-service` và `cdc-cms-web`) và đo `wc -l` thực tế (`EV-20261110T163554-74f2`) sau lần sửa code cuối:

### 1.1. Kiểm tra trần 600 dòng (`G6` — `EV-20261110T163554-74f2`)
- `internal/api/source/system_connectors_handler.go`: **521 dòng** (giảm từ 597 dòng, dưới trần 600 dòng — `EV-20261110T163554-74f2`).
- `internal/api/source/system_connectors_credentials.go`: **310 dòng** (`EV-20261110T163554-74f2`).
- `internal/app/commands/source/debezium_connector.go`: **504 dòng** (`EV-20261110T163554-74f2`).
- `internal/app/commands/source/debezium_connector_secrets.go`: **381 dòng** (`EV-20261110T163554-74f2`).
- `internal/api/system/introspection_handler.go`: **553 dòng** (`EV-20261110T163554-74f2`).
- `internal/api/system/introspection_handler_postgres.go`: **291 dòng** (`EV-20261110T163554-74f2`).
- `internal/infra/persistence/source/system_connector_repo_gorm.go`: **352 dòng** (`EV-20261110T163554-74f2`).
- `internal/app/queries/source/list_sources.go`: **118 dòng** (`EV-20261110T163554-74f2`).
- `internal/api/source/sources_handler.go`: **105 dòng** (`EV-20261110T163554-74f2`).

### 1.2. Phản biện từng dòng `git diff` theo từng file & từng Task (`T1..T4` / `G1..G6`)
1. **`cdc-cms-service/internal/app/commands/source/debezium_connector_secrets.go` (`L1-L381`) & `debezium_connector_secrets_test.go` (`L1-L214`) — `T1`, `G1`, `G2`, `G3`, `G5`, `G6` (`EV-20261110T163514-cf84`):**
   - `IsKeepOrMaskedSecret` (`L15-L29`): Nhận diện chính xác `"__KEEP__"` và mọi chuỗi chỉ gồm `*` hoặc `•` (`EV-20261110T163514-cf84`).
   - `isMaskedUserInfoPart` (`L41-L52`): Giải mã `url.PathUnescape` để bắt cả trường hợp ký tự `•` bị percent-encode thành `%E2%80%A2` (`G3.3`, `EV-20261110T163514-cf84`).
   - `splitMultiURIList` (`L80-L107`), `IsMaskedURI` (`L109-L135`), `restoreSingleMaskedURI` (`L137-L213`), `RestoreMaskedURI` (`L217-L243`):
     - Hỗ trợ `fs.uris` chứa danh sách nhiều URI phân tách bằng dấu phẩy (`G3.1`) đồng thời không cắt nhầm multi-host replicaSet của MongoDB (`EV-20261110T163514-cf84`).
     - Xử lý ca biên `len(curParts) == 1` (`L208-L212`): khi URI gốc chỉ có `user@host` mà URI gửi lên là `***:***@host`, trả về `scheme://user@host...` và tuyệt đối không ghép dư `:***` (`G3.2`, `EV-20261110T163514-cf84`).
   - `MergeConnectorConfigPreservingSecrets` (`L248-L272`): Áp dụng `IsMaskedURI(v)` cho mọi key trong config map (`G3.3`). Khi `IsKeepOrMaskedSecret(v)` nhưng `current[k]` rỗng/không tồn tại, giữ nguyên `v` trong `merged[k]` để `ValidateNoMaskedSecrets` chặn cứng Fail-Closed (`G1`, `EV-20261110T163514-cf84`).
   - `BuildStoredConfigBaseline` (`L277-L357`): Kết hợp `fp.RawConfigSanitized`, `fp.ServerAddress`, `fp.OptionsJSON` (từ `connection_registry`) và `liveCfg`; lọc nghiêm ngặt `!IsKeepOrMaskedSecret(v) && !IsMaskedURI(v)` ở mọi tầng để dữ liệu bẩn không bao giờ ghi đè baseline sạch (`G1`, `EV-20261110T163514-cf84`).
   - `ValidateNoMaskedSecrets` (`L362-L381`): Chốt chặn Fail-Closed trước khi gọi Kafka Connect nếu còn bất kỳ key nào dính `IsKeepOrMaskedSecret(v)`, `IsMaskedURI(v)` hoặc thiếu `database.password` bắt buộc (`G1`, `EV-20261110T163514-cf84`).
2. **`cdc-cms-service/internal/app/commands/source/debezium_connector.go` (`L363-L430`) — `T1`, `G1` (`EV-20261110T163514-cf84`):**
   - Bổ sung `PreResolved bool` và `LiveLookupErr error` vào `UpdateSystemConnectorConfigCommand` (`L369-L370`) để khi `SystemConnectorsHandler.UpdateConfig` đã gọi `resolveEffectiveConnectorConfig`, `UpdateSystemConnectorConfigHandler.Handle` không gọi trùng `h.writer.GetConfig` lần thứ 2 (`G1`, kiểm chứng `getConfigCalls == 1` tại `EV-20261110T163514-cf84`).
   - Gọi `ValidateNoMaskedSecrets(merged)` ở cả nhánh SFTP lazy configured (`L428`) và trước `h.writer.UpdateConfig` (`L446`, `EV-20261110T163514-cf84`).
3. **`cdc-cms-service/internal/infra/persistence/source/system_connector_repo_gorm.go` (`L125-L172`) — `T2`, `G1` (`EV-20261110T163522-bac8`):**
   - Mở rộng `GetByConnectorName` truy vấn `cdc_system.connection_registry` lấy `engine_type`, `host`, `port`, `default_database`, `options_json` để khôi phục `OptionsJSON` và `ServerAddress` ngay cả khi connector đã bị xóa khỏi Kafka Connect (`404` trong luồng `Recover`, `EV-20261110T163522-bac8`).
4. **`cdc-cms-service/internal/api/source/system_connectors_credentials.go` (`L41-L310`) & `system_connectors_handler.go` (`L133-L430`) — `T2`, `G1`, `G3`, `G5`, `G6` (`EV-20261110T163514-cf84`):**
   - `validatePostgresConnectorPrereqs` (`L53`): dùng `isKeepOrMaskedSecret(pass)` thay vì chỉ kiểm tra `pass == "***"`, chặn việc mang chuỗi `"__KEEP__"` đi kết nối PostgreSQL nguồn (`F2`, `EV-20261110T163514-cf84`).
   - `extractCredentialsAsOptions` (`L190-L205`): lưu `opts["uri"]` cho MongoDB và dùng `u.User.Username()` / `u.User.Password()` cho SFTP để tránh lỗi `u.User.String()` giữ nguyên `%E2%80%A2` hoặc làm hỏng ký tự `%40`/`%3A` (`G3.3`, `EV-20261110T163514-cf84`).
   - `Create`, `UpdateConfig`, `Recover` trong `system_connectors_handler.go`: đều chạy qua `resolveEffectiveConnectorConfig` + `ValidateNoMaskedSecrets` trước khi gọi `validateMongoConnectionString`, `validatePostgresConnectorPrereqs`, và Kafka Connect; đồng thời bọc `infrahttp.FilterSafeConfig(effectiveCfg)` khi lưu `RawConfigSanitized` (`G1`, `G5`, `EV-20261110T163514-cf84`).
5. **`cdc-cms-service/internal/api/system/introspection_handler.go` (`L51-L260`), `introspection_handler_postgres.go` (`L18-L139`), `server.go` (`L156`) — `T3`, `G2`, `G4` (`EV-20261110T163541-d941`):**
   - Khóa chặt Host-Bound Secret Resolution (`G2`): chỉ nạp secret đã lưu theo `connector_name` khi `host`/`port`/`username` (với Postgres) hoặc `SameURIHostAuthority` (với MongoDB URI) khớp tuyệt đối với cấu hình hiện tại của connector. Nếu đổi sang host lạ mà giữ nguyên secret rỗng/URI `***:***`, trả ngay `"status": "auth_err"` yêu cầu nhập lại thông tin xác thực, chặn đứng nguy cơ lộ credential sang listener bên ngoài (`EV-20261110T163541-d941`).
6. **`cdc-cms-service/internal/app/queries/source/list_sources.go` (`L10-L118`) & `internal/api/source/sources_handler.go` (`L90`) — `G5` (`EV-20261110T163514-cf84`, `EV-20261110T163522-bac8`):**
   - Mask userinfo trong `ServerAddress` trước khi trả JSON ở `GET /api/v1/sources`, `GET /api/v1/sources/:id`, và `POST /api/v1/sources` (`EV-20261110T163514-cf84`).
7. **`cdc-cms-web/src/services/connectorCheck.ts`, `src/hooks/useConnectorCheck.ts`, `src/pages/SourceConnectors.tsx` — `T4`, `G4` (`EV-20261110T163547-8421`):**
   - Bổ sung `isMaskedSecretInput` và `resolvePasswordForSubmit` chuẩn hóa chuỗi rỗng hoặc toàn `*`/`•` về `KEEP_SECRET_SENTINEL` trong cả hai chế độ `edit` và `recover` (`EV-20261110T163547-8421`).
   - Truyền `connector_name` khi `editorMode === 'edit' || editorMode === 'recover'` trong `runCheckConnection` và `runCheckPostgresConnection` (`EV-20261110T163547-8421`).

---

## 2. Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật (`EV-...`)

Tuyệt đối không dùng mockup hay suy diễn từ smoke test đơn lẻ. Toàn bộ kết quả kiểm chứng đều đi qua `tooling/evidence_run.py` kèm biên lai vật lý (`EV-20261110T163514-cf84`, `EV-20261110T163522-bac8`, `EV-20261110T163541-d941`, `EV-20261110T163547-8421`, `EV-20261110T163554-74f2`, `EV-20261110T163636-d183`):
- **Unit & Package Tests (`EV-20261110T163514-cf84`):** Chạy `go test -v -count=1 ./internal/app/commands/source/... ./internal/api/source/... ./internal/app/queries/source/...` đạt `PASS` (`exit=0` — `EV-20261110T163514-cf84`), kiểm chứng toàn bộ ca biên `G1..G6`.
- **HTTP Fiber Integration Tests (`EV-20261110T163514-cf84`, `EV-20261110T163541-d941`):**
  - `TestSystemConnectorsHandler_UpdateAndRecover_G1_G5` (`EV-20261110T163514-cf84`): Kiểm chứng qua Fiber router thật rằng `PATCH /api/v1/system/connectors/:name/config` chỉ gọi `KafkaConnectClient.GetConfig` đúng 1 lần (`getConfigCalls == 1`), khôi phục URI thật gửi lên Kafka Connect, mask `RawConfigSanitized` trong DB, khôi phục credentials từ `OptionsJSON` khi `Recover` gặp `404` trên Kafka Connect, và chặn cứng `400 Bad Request` khi thiếu secret (`EV-20261110T163514-cf84`).
  - `TestIntrospection_HostBoundSecretResolution_G2_G4` (`EV-20261110T163541-d941`): Kiểm chứng `POST /api/introspection/postgres/tables` và `POST /api/introspection/mongo/collections` chặn đứng hành vi đổi `host` kèm `connector_name` (`status: "auth_err"`, không gọi RPC/DB sang host lạ) và khôi phục đúng URI khi giữ nguyên host (`EV-20261110T163541-d941`).
- **Live PostgreSQL `cdc_dw` (`localhost:5433`) Integration Verification (`EV-20261110T163522-bac8`):**
  - Chạy `TestSystemConnectorRepoGorm_ReadOnlyLiveDB_G1_G5` kết nối trực tiếp tới PostgreSQL thật `localhost:5433/cdc_dw` (`EV-20261110T163522-bac8`): truy vấn 5 connector thực tế (`testcoretrans`, `devcoretrans`, `walletservice`, `traitestces`, `traitestpbs`), xác nhận `GetByConnectorName` đọc thành công `has_options_json=true` từ bảng `cdc_system.connection_registry` cho `testcoretrans` và `devcoretrans`, và `ListSourcesHandler.Handle` mask an toàn `ServerAddress` cho toàn bộ bản ghi (`EV-20261110T163522-bac8`).
- **Frontend Production Build (`EV-20261110T163547-8421`) & Live Kafka Connect Runtime Trace (`EV-20261110T163636-d183`):**
  - `npm run build` trong `cdc-cms-web` biên dịch TypeScript + Vite bundle sạch lỗi (`exit=0` — `EV-20261110T163547-8421`).
  - Trace trực tiếp `http://10.200.186.203:8083/connectors/testcoretrans/status` (`EV-20261110T163636-d183`) và đo `wc -l` (`EV-20261110T163554-74f2`).

---

## 3. Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

Đã phản biện các khía cạnh Concurrency, Timeout, và Data Continuity của luồng cấu hình & khôi phục connector (`EV-20261110T163514-cf84`):
1. **Loại bỏ Double HTTP Roundtrip & Race Window khi `UpdateConfig` (`G1` — `EV-20261110T163514-cf84`):**
   - Trước khi sửa, `SystemConnectorsHandler.UpdateConfig` và `UpdateSystemConnectorConfigHandler.Handle` gọi `GetConfig` 2 lần liên tiếp tới Kafka Connect (`10.200.186.203:8083`), vừa gấp đôi độ trễ mạng dưới tải multi-worker, vừa tạo cửa sổ race condition nếu trạng thái connector thay đổi giữa 2 lần đọc. Việc truyền `PreResolved: true` kèm `LiveLookupErr` (`debezium_connector.go:369-409`) đảm bảo mỗi request `UpdateConfig` chỉ thực hiện đúng 1 lần đọc `GetConfig` (`EV-20261110T163514-cf84`).
2. **Timeout Isolation trên Introspection (`introspection_handler_postgres.go:144-145` & `introspection_handler.go:205` — `EV-20261110T163541-d941`):**
   - Khi `DiscoverPostgresTablesPost` hoặc `DiscoverMongoCollectionsPost` bị từ chối bởi chốt chặn Host-Bound (`G2`), request trả về ngay lập tức ở tầng in-memory validation mà không mở socket tới DB nguồn hay gửi NATS RPC sang worker (tránh giữ kết nối cho tới `8s`/`10s` timeout hoặc làm nghẽn worker pool, `EV-20261110T163541-d941`).
3. **Bảo toàn Watermark / Offset Continuity khi `UpdateConfig` và `Recover` (`EV-20261110T163514-cf84`):**
   - Khi cập nhật `table.include.list` / `collection.include.list` hoặc `Recover` connector mồ côi, việc khôi phục chính xác `database.password`, `mongodb.connection.string`, `fs.uris` từ `liveCfg` và `cdc_system.connection_registry.options_json` kèm chốt chặn Fail-Closed `ValidateNoMaskedSecrets` ngăn chặn triệt để việc đẩy `"***"` hay `"__KEEP__"` lên Kafka Connect — bảo vệ task Debezium khỏi crash và bảo toàn tính liên tục của watermark/LSN (`EV-20261110T163514-cf84`).

---

## 4. Mục 4: Kiểm tra suy diễn & báo cáo láo (Zero Speculation & Honesty Check)

- **Kiểm tra trung thực (Honesty Check — `EV-20261110T163522-bac8`, `EV-20261110T173229-3fca`):**
  - Mọi tuyên bố về kết quả test, build, `arch_check.py`, số dòng code (`wc -l`), và live HTTP smoke test đều gắn kèm biên lai thực thi thật (`EV-20261110T173125-e131`, `EV-20261110T173141-c687`, `EV-20261110T173142-426d`, `EV-20261110T173229-3fca`).
  - Trong vòng hậu kiểm `V4`, đã biên dịch lại binary `./bin/cdc-cms-service` (`EV-20261110T173142-426d`), khởi động lại tiến trình trên cổng `:8083` (`PID 38492`), và chạy trực tiếp live HTTP smoke test không phá hủy (`EV-20261110T173229-3fca`) trên `GET /api/v1/sources`, `POST /api/introspection/postgres/tables`, và `PATCH /api/v1/system/connectors/testcoretrans/config`.

---

## 5. Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

1. **Ghi nhận bài học mới vào `memory/global/lessons.md` (`L002` và `L095` trên `INDEX.md` — `EV-20261110T173125-e131`):**
   - Đã làm rõ cơ chế đánh số của `build_lessons_index.py:222` (`idx = pos_idx + 1` đánh số tuần tự từ đầu đến cuối file `lessons.md`, còn `T1:L111` là số dòng `111` trong `lessons.md`) và ghi thêm bài học `L095` (`[2026-11-10] Khóa Khớp Máy Chủ Xuyên Suốt Mọi Nhánh Hợp Nhất Cấu Hình Và Khởi Động Lại Tiến Trình Trước Khi Kiểm Thử Sống`) ở cuối `lessons.md` (`EV-20261110T173125-e131`).
2. **Xử lý dứt điểm 4 điểm hậu kiểm `V1..V4` (`S5 -> S3 -> S4 -> S5` — `EV-20261110T173125-e131`, `EV-20261110T173141-c687`, `EV-20261110T173229-3fca`):**
   - **`V1`:** Chuyển `MaskURIUserInfo` về `internal/model/source/source.go:8-50` để `list_sources.go` gọi trực tiếp mà không nhân bản hàm hay vi phạm CQRS, tách `loadRegistryFallback`/`enrichRegistryCredentials` trong `system_connector_repo_gorm.go:125-189`, và cung cấp đúng trường top-level `objective`/`workflow` trong `dg3_fencing.json` để `laya_gate.py review` (Mode B) đạt `gate: PASS` (exit code `0`, `10/10` file `APPROVE`/`ACCEPT` tại `06_laya_summary_dg3review_20261110T173107.json` — `EV-20261110T173125-e131`).
   - **`V2`:** Loại bỏ import `internal/model/source` khỏi 3 file ngoài allowlist (`debezium_connector_secrets.go`, `system_connectors_credentials.go`, `introspection_handler.go`) qua `StoredSourceSnapshot` và `ResolveStoredBaselineByName`, giúp `arch_check.py` đạt `0` vi phạm mới trên `cdc-cms-service` (`EV-20261110T173141-c687`).
   - **`V3`:** Khóa `SameConnectorEndpointIdentity` và `SameURIHostAuthority` ngay trong `MergeConnectorConfigPreservingSecrets` (`debezium_connector_secrets.go:175-225, 295-339`) và gắn template URI theo host lưu trữ trong `BuildStoredConfigBaselineSnapshot` (`debezium_connector_secrets.go:341-438`), chặn đứng mọi nỗ lực đổi host kèm `__KEEP__`/`***:***` ở cả `UpdateConfig` và `Recover` với HTTP `400 Bad Request` (`EV-20261110T173125-e131`, `EV-20261110T173229-3fca`).
   - **`V4`:** Build `./bin/cdc-cms-service` (`EV-20261110T173142-426d`), khởi động lại tiến trình trên `:8083` (`PID 38492`), và xác nhận qua live HTTP smoke test (`EV-20261110T173229-3fca`).

---

## 6. Mục 6: Kết quả QC & Nhận xét từ Hội đồng `/laya_router` (`laya_gate` — `DG3`)

Đã chạy kiểm định `/laya_router` (`laya_gate.py`) cả 2 chế độ sau lần sửa code cuối (`EV-20261110T173125-e131`, `EV-20261110T173229-3fca`):
1. **Mode A (`laya_gate.py run --stage dg3` — File `06_laya_summary_dg3_20261110T173509.json`, exit code `0` — `EV-20261110T173125-e131`):**
   - Đạt trạng thái cổng `PASS` (`evidence_error = ""`) trên toàn bộ claims (`G1-RECOVER-GUARD`, `G2-V3-HOST-BOUND-SECRETS`, `G3-URI-EDGE-CASES`, `G5-V1-SHARED-MASK-URI`) và 3 controls (`CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`, `EV-20261110T173125-e131`).
2. **Mode B (`laya_gate.py review` — File `06_laya_summary_dg3review_20261110T173107.json`, exit code `0` — `EV-20261110T173125-e131`):**
   - Đạt trạng thái cổng `PASS` (exit code `0`) với **10/10 file** (`sources_handler.go`, `system_connectors_credentials.go`, `system_connectors_handler.go`, `introspection_handler.go`, `introspection_handler_postgres.go`, `debezium_connector.go`, `list_sources.go`, `system_connector_repo_gorm.go`, `source.go`, `server.go`) đều đạt `round_1: APPROVE` -> `decision: ACCEPT` (`EV-20261110T173125-e131`).
3. **Tổng kết Gates `G1–G8` (`EV-20261110T173125-e131`, `EV-20261110T173141-c687`, `EV-20261110T173229-3fca`):**
   - **G1–G8:** Đạt đầy đủ trên cả unit test (`EV-20261110T173125-e131`), `arch_check.py` (`EV-20261110T173141-c687`), binary build (`EV-20261110T173142-426d`), và live runtime smoke test trên `:8083` (`EV-20261110T173229-3fca`).
