# 05_progress — Audit Log (APPEND ONLY)

[2026-05-15 ICT] [Brain:claude-opus-4-7] Workspace khởi tạo. User yêu cầu audit `centralized-data-service/config/config-local.yml`.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Đọc `agent/memory/global/lessons.md` (đặc biệt 2026-05-05 validateConfig order, 2026-04-06 Airbyte retire). Đọc `project_context.md`, `tech_stack.md`, `active_plans.md`. Đọc `agent/GEMINI.md` để xác nhận role Brain (read-only audit).

[2026-05-15 ICT] [Brain:claude-opus-4-7] Tạo `00_context.md`, `01_requirements.md`, `02_plan.md`.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Đọc full `centralized-data-service/config/config.go` (622 lines). Extract struct AppConfig + sub-structs. Phát hiện KHÔNG có field `Airbyte` trong struct → YAML key `airbyte:` bị Viper silently drop.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Grep usage cho từng field:
- Airbyte: KHÔNG có field struct → DEAD (Viper drop).
- Worker.{FetchSize, TransformInterval, ScanInterval}: chỉ có ở config.go, KHÔNG có caller → DEAD downstream.
- JWT.Expiration: chỉ có ở config.go struct, KHÔNG có caller → DEAD.
- JWT.Secret: dùng để validate fail-fast guard tại boot, KHÔNG có sign/verify thực sự trong worker plane → ACTIVE-GUARD-ONLY.
- Debezium.ConnectorName: env override set vào cfg field nhưng `command_handler.detectConnectorName` hardcode `"goopay-mongodb-cdc"` (line 2021) → cfg field DEAD downstream.
- Sources["postgres_primary"]: `cfg.SourceURL()` defined nhưng KHÔNG có caller (grep `.SourceURL(` ngoài file config.go → 0 result) → DEAD.
- Sources["mongodb_primary"]: bridge fallback → hydrate cfg.MongoDB.URL → worker_server consume → ACTIVE-INDIRECT.
- Còn lại (server/db pool/systemDb/shadowDb/masterDb/controlPlane/nats/redis/kafka/otel/worker active subset/debezium active subset): ACTIVE, có file:line caller.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Tạo `report_config_local_audit_2026-05-15.md` với bảng tổng hợp.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Verify service work: build check skip (read-only audit, không sửa code) — verify bằng cách confirm `config-local.yml` HIỆN TẠI parse OK với loader hiện tại (DEAD keys chỉ là noise, không gây fail). Lesson 2026-05-05 đảm bảo validateConfig order đúng.

[2026-05-15 ICT] [Brain:claude-opus-4-7] Báo cáo hoàn thành. KHÔNG có file source thay đổi (audit only).

[2026-05-15 ICT] [Muscle:claude-opus-4-7] User verb "làm đi" → switch sang execution mode. Tạo `08_tasks_cleanup.md` + `09_tasks_solution_cleanup.md` (technical solution + diff trước/sau + rollback plan).

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Backup `config-local.yml` → `/tmp/config-local.yml.bak.20260515` (128 lines). Note: file đã có local modifications so với HEAD (develop branch).

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Edit `config-local.yml` xoá 10 dòng:
- Line `postgres_primary: postgres://src_user:...@localhost:5435/...` (1 dòng).
- Worker dead fields: `fetchSize: 1000`, `transformInterval: 5m`, `scanInterval: 1h` (3 dòng).
- Toàn block `airbyte:` (apiUrl/clientId/clientSecret = 4 dòng).
- `jwt.expiration: 24h` (1 dòng).
- `debezium.connectorName: goopay-mongodb-cdc` (1 dòng).
Tổng: 10 dòng. Diff verify đúng plan trong `09_tasks_solution_cleanup.md`.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Verify: `cd centralized-data-service && go build ./...` → EXIT=0 (no compile error). `go test ./config/...` → 4 tests PASS (TestUnmarshalKafka_*). 0.957s.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Smoke load test (ad-hoc `/tmp/check_config.go` gọi `config.NewConfig()` với `cfgPath=./config/config-local.yml`): LOAD OK, validateConfig pass. Tất cả ACTIVE keys giữ giá trị đúng (server.port=:8082, systemDb/controlPlane/master/shadow set, kafka 4 topicPrefix, worker pool/batch/transformChunk/kafkaFlush đúng, mongodb.url bridge từ sources hoạt động, debezium.{kafkaConnect,signalDB,signalCollection,incr} đúng). Tất cả DEAD keys = empty/zero confirm xoá thành công (FetchSize=0, TransformInterval=0s, ScanInterval=0s, jwt.Expiration=0s, debezium.ConnectorName=""). Đã xoá file tạm.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] File thay đổi:
- `data-hub/centralized-data-service/config/config-local.yml` (128 → 118 lines, -10 lines, 7 mục DEAD removed).
- Workspace docs append: `05_progress.md` (file này), update `07_status_report.md`, append section "Đã thực thi" vào `report_config_local_audit_2026-05-15.md`, tạo `08_tasks_cleanup.md` + `09_tasks_solution_cleanup.md`.
- KHÔNG sửa `.go`. KHÔNG sửa `config-sample.yml`/`config-production.yml`.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] **ROUND 2 — Redundancy collapse**. User feedback: "mấy cái này là gì, sao nó giống nhau vậy. làm việc sao hời hợt, ngu đần vậy" → đào sâu phát hiện audit round 1 miss tầng redundancy giữa `db.{host..url}` + `systemDb.url` + `controlPlane.url` (cả 3 cùng trỏ về `localhost:5433/cdc_dw`).

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Trace fallback chain (config.go applyDBFallbacks:457-480 + multi.go dsnForRole:200-244 + cmd/admin-api/main.go:29-32):
- `multi.go GetDB("cdc")` → `cfg.ControlPlaneURL()` (primary reader).
- `cmd/admin-api/main.go:29-31` → `cfg.ControlPlaneURL()`, fallback `cfg.SystemDBURL()`.
- `applyDBFallbacks`: `ControlPlane.URL ← SystemDB.URL ← cfg.DB.PgxDSN()` (compose từ db.{host..}).
- `validateConfig`: yêu cầu `hasLegacy OR hasSplit`, KHÔNG check ControlPlane.
→ 3 layer là fallback chain runtime, KHÔNG phải 3 thứ khác nhau. Trên local rig cùng trỏ 1 PG → REDUNDANT.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] User verb "chỉ sửa config.yml" → Path A1 YAML-only:
- Xoá 7 dòng `db.{host,port,username,password,database,sslMode,url}` (giữ 3 dòng pool tuning `maxOpenConn/maxIdleConn/connMaxLifetime`).
- Xoá block `controlPlane:` (2 dòng + 4 dòng comment) — runtime sẽ hydrate từ `systemDb.url` qua applyDBFallbacks.
- Rewrite header comment giải thích single-source-of-truth per logical plane (systemDb/shadowDb/masterDb).
- Validator vẫn pass: `hasSplit = SystemDB.URL != ""` true vì `systemDb.url` còn.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Verify round 2: `go build ./...` EXIT=0. `go test ./config/...` PASS 0.572s. Smoke load `config.NewConfig()`:
- `systemDb.url` = real DSN ✅
- `controlPlane.url` hydrate runtime = systemDb.url ✅ (match=true)
- `destinationURL` = masterDb.default ✅
- `shadowDb default` = 5436/cdc_shadow ✅
- `db pool` open=50 idle=25 lifetime=5m0s ✅
- `db legacy DSN fields` = empty (host="" port=0 user="" db="" url="") ✅ confirm xoá thành công.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] File thay đổi round 2:
- `data-hub/centralized-data-service/config/config-local.yml` (117 → 114 lines, net -3 dòng vì rewrite comment thêm 9 dòng, xoá 12 dòng).
- Workspace `05_progress.md` (file này) APPEND.
- Sẽ APPEND lesson vào `agent/memory/global/lessons.md` về audit pattern phải verify cross-layer redundancy.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] Tổng kết 2 round:
- `config-local.yml` 128 → 114 lines (-14 dòng net, -7 mục DEAD + collapse 3 DSN layer redundancy).
- 0 lines code Go thay đổi.
- Build + test PASS cả 2 round.

[2026-05-15 ICT] [Muscle:claude-opus-4-7] APPEND lesson "Audit config phải verify cross-layer redundancy, không chỉ per-key DEAD" vào `agent/memory/global/lessons.md` (2875 → 2891 lines). Global Pattern: chain L1←L2←L3 có giá trị trùng nhau trong môi trường audit → flag REDUNDANT, không chỉ DEAD per-key. Tags: #audit #config #redundancy #fallback-chain #dry.
