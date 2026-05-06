# 05 — Progress (APPEND ONLY — CLAUDE.md §11)

---

## 2026-05-04 17:05+07 — Workspace khởi tạo

- User mandate: refactor toàn hệ thống cdc-system, chữa "nồi cám heo".
- Brain đã đọc:
  - `agent/memory/global/lessons.md` (TOC 1708 dòng, 70+ lesson)
  - `agent/memory/global/active_plans.md` (registry workspace)
  - `agent/memory/global/project_context.md` (template, chưa fill)
  - `agent/memory/global/tech_stack.md` (template, chưa fill)
  - `cdc-system/architecture.md` (architecture chi tiết 866 dòng)
  - `feature-cdc-system-refactor/07_status.md` + `10_gap_analysis.md`
- Brain đã scan disk:
  - 4 service: cdc-auth-service (9 .go), cdc-cms-service (76 .go), centralized-data-service (144 .go, 4 binary), cdc-cms-web (22 .tsx, 7634 LOC).
  - Build status: 3/3 Go service `go build ./... → exit 0`. FE `tsc --noEmit` → exit 0.
  - Live infra: 4 PG + Mongo + Mariadb + Redis + Kafka + Connect + SchemaRegistry + NATS + OTel — all healthy. cdc-worker docker, cms-server local, admin-api local đều running.
  - Uncommitted: 5 file Go = 286 line (Phase F1+F3 admin-api hardening + helpers.go fix).
- Workspace tạo: `agent/memory/workspaces/feature-system-refactor-2026-05/` (KHÔNG ghi đè `feature-cdc-system-refactor` cũ — Done từ Sprint trước).
- Đã viết `00_context.md` ghi sự thật disk + drift findings + 5 câu hỏi clarify scope.

## 2026-05-04 17:05+07 — Files vật lý đã tạo

- `00_context.md`
- `05_progress.md` (file này)

## 2026-05-04 17:05+07 — Pending

- `01_requirements.md` — sau khi User clarify scope.
- `02_plan.md` — sau khi user clarify scope.
- `08_tasks.md` + `09_tasks_solution.md` — khi plan chốt.
- `10_gap_analysis.md` — Brain sẽ điền sau khi user trả lời 5 câu hỏi.
- `report_initial_scan_20260504.md` — báo cáo findings + đề xuất scope, ĐANG VIẾT.

## Open questions cho User

Q1 Scope (1/4/n service first?), Q2 Function-broken (cụ thể nào đang đứt?), Q3 Goal-1-dòng, Q4 Risk tolerance (down-time?), Q5 Time budget.

---

## 2026-05-04 17:14+07 — B1 (Quick Wins) hoàn tất

### B1.1 — Commit 5 file Phase F1+F3 pending
- `git status` xác nhận 5 file (admin-api/main.go, helpers.go, server.go, server_test.go, source_register.go).
- `go test ./internal/admin/ -count=1` → ok 1.067s (21 assertion PASS).
- `git -c user.name="TraiNguyen" -c user.email="trainguyen@TraiNguyens-MBP.localdomain" commit` (per-command, không update `.git/config` per CLAUDE.md "NEVER update the git config").
- Commit hash: `92d78d3`. Tree clean post-commit.

### B1.2 — architecture.md gỡ Airbyte mention
- `grep -ic airbyte` BEFORE = 1 (line 64), 2 edge AIR (line 92, 95).
- Edit: `subgraph CDC` xóa node AIR, edge `MDB --> AIR` + `AIR --> NATS` xóa, đổi DBZ description "MongoDB / PostgreSQL / MariaDB CDC".
- `grep -ic airbyte` AFTER = 0. Verify ✅.

### B1.3 — Fill global memory templates
- `project_context.md` rewrite từ template → cdc-system thực tế (4 service, 250 .go + 22 tsx, 13 docker container, terminologies + business rules).
- `tech_stack.md` rewrite từ template → Go 1.26.1 + Vite/React/TS, 4 PG, multi-engine source, JobMonitor close-loop pattern.
- `grep -c "\[DATE\]\|\[Project Name\]\|\[e\.g\.\|\[Group A\]\|\[Term A\]"` cả 2 file = 0. Verify ✅.

### B1.4 — active_plans.md update
- Đổi `feature-cdc-integration` từ Active "Hybrid Debezium + Airbyte" → Done "Debezium-only sau commit 8ef7d71" + ghi Phase F (F1+F3) Done 2026-05-04.
- Thêm entry mới `feature-system-refactor-2026-05` Active 2026-05-04 (bucket B1+B2).

### B1.5 — APPEND lesson L-input-fallback-pattern
- Lessons.md: 1708 → 1778 dòng (+70).
- Heading mới `## L-input-fallback-pattern (2026-05-04, CDC Integration Phase F3 + System Refactor 2026-05)`.
- Format Global Pattern A/B/X/Y per CLAUDE.md §13. 6 dự án khác nhau apply pattern này (CDC orchestrator, K8s admission, Stripe webhook, multi-tenant DB sharding, image build, search indexer).

**B1 verdict**: ✅ ALL PASS. Files vật lý:
- 1 commit git: `92d78d3`.
- 1 file edit `cdc-system/architecture.md`.
- 2 file rewrite global memory: `project_context.md`, `tech_stack.md`.
- 1 file edit `active_plans.md`.
- 1 APPEND `lessons.md`.

Brain code prohibition CLAUDE.md §12: ✓ chỉ edit `.md` (Markdown không phải source code Go/TS/SQL/Python).

### Next: B2 — start cdc-auth-service + cdc-cms-web local + smoke E2E.

---

## 2026-05-04 17:18+07 — B2.1 + B2.2 verify thực tế (live state)

### Phát hiện Drift quan trọng
- **3 service Go đã chạy local từ trước (~5 ngày uptime)** — `project_context.md` ghi "CHƯA chạy local" cho `cdc-auth-service` là SAI (hiện tại live).
  - PID 72078 `main` (cdc-auth-service) bind `*:8081`.
  - PID 13653 `cms-serve` (cdc-cms-service) bind `*:8083`.
  - PID 75578 `cdc-admin` (centralized-data-service admin-api) bind `localhost:8090`.
- **FE Vite cũng đã chạy** — PID 71945 `node vite`, bind IPv6 `localhost:5173` (IPv4 127.0.0.1 fail). Uptime ~5 ngày.

### B2.1 — cdc-auth-service smoke
- Migration `001_auth_users.sql` apply lên `gpay_auth` (NOT `goopay_dw` per Makefile drift) → schema `cdc_auth_service.auth_users` đã tồn tại + seed `admin@goopay.vn` role=admin.
- Build `go build -o /tmp/cdc-auth-service ./cmd/server` → PASS (38 MB binary). KHÔNG khởi động trùng do PID 72078 đã serve.
- `GET /health` → `200 {"service":"cdc-auth","status":"ok"}`.
- `POST /api/auth/login {admin/admin123}` → `200`, JWT issued (access_token + refresh_token, role=admin, expires_in=86400).
- B2.1 PASS. Note: Makefile target `migrate` outdated (-U user -d goopay_dw) — defer fix sang B3.

### B2.2 — cdc-cms-web smoke
- `package.json` Vite v8 + React 19 + AntD v6 + axios + react-router 7. node_modules đã cài.
- `GET http://localhost:5173/` → 200 HTML (`<!doctype html>` + `@vite/client` script).
- `src/services/api.ts` config 3 backend baseURL: AUTH 8081 ✅, CMS 8083 ✅, WORKER 8082 ❌.
- **Drift FIX**: `.env` `VITE_WORKER_API_URL=http://localhost:8082` → `8090` (admin-api thực bind 8090 per `cmd/admin-api/main.go:64`).
- `workerApi` instance không có call site nào → drift cosmetic, không gây bug runtime, nhưng vẫn fix defensive.
- B2.2 PASS.

### Backend health verify (4 service path)
- 8081 auth `/health` → 200 ✅
- 8083 cms `/health` → 200 ✅
- 8090 admin-api `/healthz` → 200 ✅; `/health` → 401 (auth-required, expected sau Phase F1)
- 5173 FE `/` → 200 ✅

### Files vật lý đã tạo/sửa B2 round 1
- `cdc-cms-web/.env` — fix VITE_WORKER_API_URL từ 8082 → 8090.
- (KHÔNG khởi tạo binary mới vì process đã live; KHÔNG sửa file .go/.ts/.sql).

### Next: B2.3 — E2E operator path live trên 4 service đã verify health.

---

## 2026-05-04 17:25+07 — B2.3 + B2.4 + B-final

### B2.3 — E2E operator path (data-plane verify, không UI click)
- Login `admin/admin123` → JWT 245 chars (cached `/tmp/cdc_jwt.txt`).
- `GET /api/v1/source-objects` → 200, 18 entry (`smoke_p02_*` + `p01_*` + Track D legacy).
- `GET /api/v1/masters` → 200, count=9 (mỗi master có shadow_binding_id + master_schema).
- `GET /api/v1/schedules` → 200, count=6. Schedule id=14 vừa tick `last_run_at=2026-05-04T17:24:13+07`, `last_status=success`, `duration_ms=2`.
- **Close-loop xác nhận live**: cms-server vừa restart 17:22:53 → JobMonitor (P4/D-39.A) sub `cdc.evt.transmute.completed` reload → next cron tick 17:24 update đúng `success`.

### Drift cross-service (cosmetic, defer B3)
1. **`cms-cms-service` config drift**: `config-local.yml:46 workerUrl: http://localhost:8082` → ĐÃ FIX → `8090`. Nguyên nhân: admin-api (centralized-data-service) bind 8090 (per `cmd/admin-api/main.go:64`), cms config viết 8082.
2. **`cms-service` code drift `/health` vs `/healthz`**: `internal/service/system_health_collector.go:267` gọi `<workerURL>/health` → admin-api Phase F1 đã auth-gate → 401. Phải đổi `/health` → `/healthz` (no-auth dev probe). **KHÔNG fix lần này** vì §12 cấm Brain edit `.go`. Document làm Muscle task B3-D-39.B.
3. **`cms-service` code drift `/metrics` auth**: `internal/service/prom_client.go:200` GET `<workerURL>/metrics` → admin-api Phase F1 auth-gate → 401. Cần đưa token hoặc move /metrics ra ngoài auth gate. Defer B3.
4. **kafka_exporter sidecar**: `:9308` không chạy → `consumer_lag` unknown. 3rd party container — defer.

### Restart cms-server log
- Old PID 13653 (uptime 5d) `kill 13653` clean.
- Build `/tmp/cms-server.new` từ `cdc-cms-service/cmd/server` PASS, atomic move → `/tmp/cms-server`.
- Boot log clean: PG/NATS JetStream/Redis connected, OTel bridge active, system_health_collector + audit + alert_resolver started.
- New PID 87728. /health, /ready, /api/system/health → 200.
- **Backup**: `/tmp/cms-server.bak.20260504-pid13653` giữ nguyên binary cũ.

### B2.4 — `scripts/dev-up.sh`
- Tạo file mới `cdc-system/scripts/dev-up.sh` (chmod +x).
- 3 sub-command: `up | status | stop`.
- Idempotent: skip nếu port đã LISTEN. Build go bằng `go build`, FE bằng vite via node_modules.
- Logs `/tmp/<svc>.log`, pidfiles `/tmp/cdc-dev-pids/<svc>.pid`.
- Smoke 4/4 service post-up: auth/health 200, cms/health 200, admin/healthz 200, fe 200.
- ĐÚNG §12: chỉ là `.sh`, không phải Go/TS/SQL/Python.

### Service health snapshot 17:25+07
| Service | Port | PID | Uptime | Verify |
|---|---|---|---|---|
| cdc-auth-service | 8081 | 72078 | 5d 8h | /health 200, login admin/admin123 → JWT |
| cdc-cms-service | 8083 | 87728 | 2 phút (fresh restart) | /health 200, /api/v1/* 200 với JWT |
| centralized-data-service admin-api | 8090 | 75578 | (?) | /healthz 200, /metrics + /health 401 (auth-gate) |
| centralized-data-service worker | docker | gpay-cdc-worker | 2h | JobMonitor close-loop 17:24:13 success |
| cdc-cms-web (Vite) | 5173 | 71945 | 5d 8h | / 200 HTML |
| 4 PG + Mongo + MariaDB + Redis + Kafka + KConnect + SchemaReg + NATS + OTel | docker | various | up | infra heath-collector probe up |

### B-final verify all services work
- ✅ 4/4 backend service LISTEN + health 200.
- ✅ FE Vite serve HTML.
- ✅ JWT issue + verify từ login → CMS protected endpoint.
- ✅ CDC pipeline live (transmute schedule cron tick close-loop success).
- ⚠ /api/system/health overall=critical do drift cosmetic cross-service (worker probe path) — KHÔNG ảnh hưởng data plane, defer B3.


## 2026-05-05 10:07+07 — Phase B5 (Config-Env Extract + Docker Split) DONE

**Trigger**: User trainguyen 2-task assignment lúc 09:30+07.

**Goal 1 — endpoint/secret → ENV (3 repos)**:
- `cdc-auth-service/config/config.go` +1 import strconv +30 lines `applyEnvOverrides()` cho 7 env (AUTH_DB_*, AUTH_SERVER_PORT, JWT_SECRET).
- `cdc-cms-service/config/config.go` +35 lines cho 11 env (CMS_DB_*, CMS_SERVER_PORT, NATS_URL, REDIS_URL, JWT_SECRET, OTEL_EXPORTER_OTLP_ENDPOINT).
- `cdc-cms-service/config/config-local.yml` xoá block `airbyte:` (chứa real `apiKey: trai.nguyen@goopay.vn:knF1jh...` — secret leak), `controlPlane:`, `destination:` (dead config — AppConfig không bind).
- `centralized-data-service/config/config.go` +18 lines cho `SOURCE_DSN_POSTGRES_PRIMARY` + `SOURCE_DSN_MONGODB_PRIMARY` (đặt trước applyDBFallbacks để fallback bridge thấy giá trị mới).
- 4 file `.env.example` mới (auth/cms/central/cdc-docker-dev).

**Goal 2 — Docker split**:
- Rewrite `centralized-data-service/docker-compose.yml` 369→188 lines, giữ 10 service core (NATS, postgres-cdc, Redis, Kafka, schema-registry, kafka-connect, kafka-exporter, redpanda-console, otel-collector, cdc-worker). Hardcoded password → `${VAR:-default}` interpolation.
- Tạo `cdc-system/cdc-docker-dev/docker-compose.yml` với 6 dev DB (postgres-auth, postgres-source, postgres-dest, mongo, mysql, mariadb). External volumes trỏ tới `centralized-data-service_*` để bảo toàn data.
- External network `cdc-bridge` chia sẻ giữa 2 compose. Loại `depends_on` chéo (worker không depend dest, kafka-connect không depend mongo).
- README + .env.example cho cdc-docker-dev.

**Verify (exercise-driven, lesson 2026-04-28)**:
- ✅ 3× `go build ./...` PASS.
- ✅ Env override smoke (3 repos) — override giá trị set, fallback YAML cho var rỗng.
- ✅ 16 containers up: 14 healthy / running, kafka-exporter restarted ổn sau Kafka warm-up.
- ✅ Data preserved: source 65→68, shadow 18→21 (+3), master 37→40 (+3) sau insert smoke.
- ✅ Cross-bridge resolution: worker resolves gpay-postgres-dest/source/mongo, kafka-connect resolves postgres-source.
- ✅ Track D close-loop từ B3 vẫn hoạt động: 5 schedule recent có last_status='success', last_error=NULL.
- ✅ Worker log clean (last 1m): 0 warn/error/fatal.
- ✅ 3 host service /health endpoint trả 200 (auth :8081, cms :8083, admin :8090).

**Out of scope ghi nhận**:
- MariaDB connector FAILED do plugin install lỗi pre-existing (`debezium-connector-mysql:2.5.4` không có trên Confluent Hub). PG + Mongo connector vẫn RUNNING — pipeline core không ảnh hưởng.
- Git history vẫn chứa airbyte secret cũ — purge cần `git filter-repo` task riêng.
- `.gitignore` chưa có `.env` — em note trong report.

**Files**:
- 5 doc workspace: `01_requirements_b5_config_env_docker.md`, `02_plan_b5_config_env_docker.md`, `08_tasks_b5_config_env_docker.md`, `09_tasks_solution_b5_config_env_docker.md`, `report_phase_b5_config_env_docker_20260505_1007.md`.
- 6 file mới ngoài workspace: 3× `.env.example` + `cdc-docker-dev/{docker-compose.yml, .env.example, README.md}`.

**TaskList**: #115-#122 đều completed.

**Lesson reinforcement**: external volumes `name: centralized-data-service_*` cho dev compose là cách an toàn migrate volumes giữa các compose project mà KHÔNG mất data. Pattern: nếu split compose project, declare volumes external trỏ tới existing namespaced names → preserve data, đổi project ownership tách bạch.

---

## 2026-05-05 10:24+07 — Phase B5.5b: decouple cross-repo init mount

**Trigger**: anh trainguyen pick up regression — `cdc-docker-dev/docker-compose.yml` mount `../centralized-data-service/deployments/sql/source` và `../centralized-data-service/deployments/mariadb/init` qua relative path → tái lập coupling chính việc split B5.5 vốn đi tách. Anti-pattern.

**Fix**:
- `cdc-docker-dev/init/postgres-source/01_init_source_local.sql` MOVE từ `centralized-data-service/deployments/sql/source/`.
- `cdc-docker-dev/init/mariadb/01_seed.sql` MOVE từ `centralized-data-service/deployments/mariadb/init/`.
- `cdc-docker-dev/docker-compose.yml` EDIT 2 dòng mount: `../centralized-data-service/deployments/sql/source` → `./init/postgres-source`; `../centralized-data-service/deployments/mariadb/init` → `./init/mariadb`.
- `cdc-docker-dev/README.md` EDIT note: thay "mount qua relative path từ centralized-data-service/..." bằng "self-contained, KHÔNG dependency cross-repo".
- `centralized-data-service/deployments/sql/source/` DELETE (rỗng sau mv).
- `centralized-data-service/deployments/mariadb/init/` + `centralized-data-service/deployments/mariadb/` DELETE (rỗng sau mv).

**Verify**:
- `docker compose -f cdc-docker-dev/docker-compose.yml config --quiet` → EXIT=0.
- `grep -rn 'deployments/sql/source\|deployments/mariadb' cdc-system/` → 0 hit.
- Source data alive (volumes external không re-init): postgres-source `orders=70 users=10 payments=10`, mariadb `legacy_orders=5`.

**Lesson abstract → lessons.md**: Khi split compose project A → B + C (cùng repo umbrella), KHÔNG được mount asset của A bằng relative path `../A/...` trong compose của B/C — coupling cross-project trá hình. Pattern đúng: di chuyển asset (init scripts, configs, secrets samples) vào repo own của B/C → mount relative `./...`. Test: `grep -rn '\.\./<other-project>'` trong YAML compose phải 0 hit.

---
## 2026-05-05 — B5.5c: Schema-prefix env refactor + postgres-shadow container scaffold

**Trigger A (refactor)**: anh trainguyen — *"code hiện tại (refs `shadow_<src>`) => thêm cái prefix ở env trong source cdc-worker đi, rồi dùng nó khi tạo schema. để ko ép buộc nó dính với từ shadow"*. Mục tiêu: gỡ cứng từ `shadow_` ra env → đổi convention (`lake_`, `raw_`) chỉ cần đổi env, không đụng code.

**Trigger B (postgres-shadow container)**: anh trainguyen — *"cdc-docker-dev vẫn chưa có postgres shadow nhé"*. Mục tiêu: bổ sung container `gpay-postgres-shadow` vào cdc-docker-dev compose theo plan A đã duyệt (shadow = data-lake = config-able destination, sibling của master).

### A. Prefix env refactor (centralized-data-service)

**Done**:
- NEW `internal/naming/naming.go` (35 lines): `ShadowSchemaPrefix()` đọc `CDC_SHADOW_SCHEMA_PREFIX` qua `sync.Once`, fallback `shadow_`. Helper `ShadowSchemaName(suffix)` = prefix + suffix.
- Edit 4 sites hardcode `"shadow_"` → `naming.ShadowSchemaName(...)`:
  - `internal/admin/helpers.go::shadowSchemaFor` (3 case: postgresql/mongodb/mariadb-mysql + default).
  - `internal/handler/provisioning_step_handlers.go:276` (HandleShadowBind fallback derive schema).
  - `internal/sinkworker/sinkworker.go::normalizeShadowSchema` (line 299).
- `centralized-data-service/.env.example`: add `CDC_SHADOW_SCHEMA_PREFIX=shadow_` block (top of file, trên DSNs).
- `go build ./...` PASS.

**KHÔNG đụng** (xác nhận grep):
- State machine enums chứa từ `shadow_*` (ví dụ `shadow_pending`, `shadow_done`) — đây là state names, không phải schema names.
- NATS subjects `cdc.cmd.shadow.bind` / `cdc.evt.shadow.*` — protocol identifier.
- Log keys & test fixtures literal `shadow_<src>` — chỉ verify, không tạo schema.

### B. postgres-shadow container scaffold (cdc-docker-dev)

**Done**:
- `cdc-docker-dev/docker-compose.yml`: thêm service `postgres-shadow` (image postgres:15-alpine, container `gpay-postgres-shadow`, port 5436, db `cdc_shadow`, healthcheck pg_isready). Renumber comment 2-6.
- `cdc-docker-dev/docker-compose.yml::volumes`: thêm `pg_shadow_data` named volume (LOCAL fresh, không external — không có data inherit). Comment ghi rõ data hiện vẫn ở cdc_dw (5433), cutover manual.
- `cdc-docker-dev/.env.example`: block `PG_SHADOW_USER/PASSWORD/DATABASE` + comment giải thích "data-lake layer per architectural reframing".
- `cdc-docker-dev/README.md`: row mới trong service table + blockquote ghi prefix là env-driven + cutover plan.
- `centralized-data-service/.env.example`: thêm comment đường opt-in `CDC_SHADOW_DB_URL=...localhost:5436/cdc_shadow...` (commented out, default vẫn trỏ cdc_dw).

**Verify**:
- `docker compose -f cdc-docker-dev/docker-compose.yml config --quiet` → EXIT=0.
- `docker compose up -d postgres-shadow` → container Created → Started → healthcheck=healthy trong ~10s.
- `psql -U gpay_admin -d cdc_shadow -c "SELECT current_database();"` → `cdc_shadow` (PG 15.17 alpine).
- Old shadow data tại cdc_dw vẫn intact (không touch CDC_SHADOW_DB_URL trong dev .env hiện tại).

**Pending (không thuộc scope)**:
- Migration script `migrations/cdc/0NN_split_shadow_db.sql`: pg_dump `shadow_*` schemas từ cdc_dw → restore vào cdc_shadow. Manual cutover khi anh duyệt.
- Switch `CDC_SHADOW_DB_URL` trong dev .env sang port 5436. Sau khi migration script chạy.

**Lesson abstract → lessons.md (Global Pattern)**: Khi convention naming (prefix/suffix/separator) bị hardcode rải rác qua N call sites trong codebase X → tạo package `naming` (hoặc `convention`) tập trung, expose helper `<Convention>Name(parts...)` đọc env (`<DOMAIN>_<CONVENTION>_<PART>`) qua `sync.Once`, default fallback giữ behavior cũ. Lý do: thay convention chỉ cần đổi env (1 dòng), không cần code review N file. Anti-pattern: tìm sửa từng `"shadow_"` mỗi khi đổi convention → drift, sót hit (state enum / log key / test fixture lẫn lộn với schema name).

---
## 2026-05-05 11:08 — B5.5c LIVE CUTOVER: shadow_* schemas migrated to gpay-postgres-shadow + RoleShadow first-class

**Trigger**: anh trainguyen — *"duyệt"* (cutover authorization).

**Outcome (real-verified, not fabricated)**:
- 9 shadow_* tables / 54 rows pg_dump'd from `cdc_dw` → restore vào `cdc_shadow`. Count matches 100% per-table.
- Smoke #1 (env-only DSN switch): FAILED — new rows landed in OLD `cdc_dw`. Root cause: code architecture had `GetShadowDB("default") → RoleControlPlane`, ignoring `cfg.ShadowDB.URLs[default]`.
- **Architectural code fix (essential, not bypassable)**:
  - `pkgs/database/multi.go`: thêm `RoleShadow = "shadow"` const + `case RoleShadow` trong `dsnForRole` resolving từ `cfg.ShadowDB.URLs[default-key]`, fallback `RoleControlPlane` để giữ legacy collocated layout work.
  - `internal/service/connection_manager.go::GetShadowDB`: 2 sites `RoleControlPlane` → `RoleShadow`. Update routing-rule comment.
  - `go build ./...` PASS.
- Smoke #2 (after code fix + rebuild): PASS. INSERT 4 source rows (cutover2-1..4 id 75-78) → log `batch upsert ok group=shadow|shadow_local_pg_cdc count=4` + `group=shadow|legacy_shadow_default count=4` → NEW shadow=27 (+4), OLD shadow=26 (unchanged), 4 rows visible in NEW only.
- Cron tick (65s): schedule id=1 `last_status=success scanned=27 inserted=22 updated=4 errors=0 duration_ms=137`. Master `dw_orders.orders_fact`: 45 → 49 (+4). Cả 6 enabled schedules SUCCESS.
- Pipeline E2E: source(5435) → debezium → kafka → worker → shadow_NEW(5436/cdc_shadow) → master(5434/goopay_dest) ✓.
- Rollback path documented: rm `.env` + recreate worker (fallback chain RoleShadow→RoleControlPlane). 4 cutover2 rows would be lost (not in cdc_dw); re-snapshot trivial.

**Files added (cumulative B5.5c)**:
- NEW `centralized-data-service/.env` (runtime override, gitignore) — `CDC_SHADOW_DB_URL=...gpay-postgres-shadow:5432/cdc_shadow`.
- EDIT `pkgs/database/multi.go` (RoleShadow const + dsnForRole case + doc).
- EDIT `internal/service/connection_manager.go` (GetShadowDB 2 sites + comment).
- NEW `report_phase_b5_5c_shadow_split_20260505_1108.md` (anh requested for review).

**Lesson reinforced**: env-only "split" without architectural role promotion = ineffective. The code architecture is the contract — env vars mean nothing if not wired through Registry roles. Pattern: when introducing a new physical-instance separation, FIRST add the role to Registry (with backwards-compat fallback), THEN switch the env. Reverse order = silent regression to old physical destination.

**Pre-existing test failure noted (NOT regression)**: `TestSchemaValidatorDriftDetection` panic = nil zap logger in test setup, file `internal/service/schema_validator.go:126`. I did NOT touch that file. Tracked for future cleanup.


---
## 2026-05-05 11:58 — B5.6 cdc-auth-service prod-config alignment (vs reconcile-service pattern)

**Trigger**: anh trainguyen — *"sao repo auth hiện tại nó có cảm giác ko lên prod đc vậy. em lên cai plan đi check các repo, và fix chỗ này cho anh xem. xon con reconcile-service trước anh làm."* (reconcile-service đã prod-ready trước đó, dùng làm pattern đối chiếu).

**Diagnosis (thực tế, dựa file đã đọc)**:
- `cdc-auth-service/config/config.go`: path resolver chỉ accept config-NAME, `applyEnvOverrides` hardcoded 8 fields, KHÔNG có `validateConfig()`.
- `cdc-auth-service/config/`: chỉ có `config-local.yml`, KHÔNG có `config-production.yml` hay `config-sample.yml`.
- `cdc-auth-service/deployments/docker/Dockerfile:12`: `COPY --from=builder /app/config/config-local.yml ...` → image prod nuốt creds DEV + JWT secret `change-me-in-production` + pool 10/5.
- vs `reconcile-service`: dual-mode path resolver, AutomaticEnv binding, validateConfig, đầy đủ 3 yml (local/production/sample), Dockerfile copy cả repo.

**Fixes applied**:
- REWRITE `cdc-auth-service/config/config.go`:
  - Path resolver dual-mode (file path absolute / config name + multi search paths).
  - `viper.SetEnvPrefix("AUTH")` + `SetEnvKeyReplacer(".", "_")` + `AutomaticEnv()`.
  - Explicit `BindEnv` map 15 keys → single source of truth, viper handle Duration parsing tự động.
  - Backwards-compat: `cfgPath` (reconcile-style) lẫn `CFG_PATH` (cdc-system existing) đều support; `JWT_SECRET` legacy + `AUTH_JWT_SECRET` preferred.
  - `validateConfig()` 6 rules: server.port, db.host/database/username, jwt.secret non-empty, refuse `change-me-in-production` khi `mode==production`.
- EDIT `config-local.yml`: thêm `mode: dev`.
- NEW `config-production.yml`: prod tunables (pool 50/25, sslMode require, JWT 1h/24h), fields rỗng cho secret → env override điền.
- NEW `config-sample.yml`: clone local cho dev/staging template.
- EDIT `Dockerfile`: `COPY config ./config` (cả folder), comment runtime hint cho `cfgPath`.
- REWRITE `.env.example`: actionable env vars theo Global Pattern lesson 2026-05-05 (#-header + var, no prose), thêm `cfgPath` (commented) + `AUTH_JWT_SECRET`.

**Verification (real-exercised, không health-only)**:
- `go build ./...` EXIT=0; `go test ./...` EXIT=0 (no test files trong repo).
- Smoke #1 (local default): config path log → DB connect → server start. Bind :8081 fail vì auth-service host process đang chiếm — config path đã pass.
- Smoke #2 (prod yml, no env): `validate config: db.host required` ← validation refuse rỗng đúng.
- Smoke #3 (prod yml + full env, port :19999): DB connect → server start → `curl /health → 200` → graceful shutdown. ✅ E2E.
- Smoke #4 (prod mode + JWT placeholder): `jwt.secret must not use default placeholder in production mode` ← đóng được hole "deploy nhầm secret default".
- Existing services: 5 postgres containers `(healthy)`, auth-service host process `curl /health → 200` — không regression.

**Files added (cumulative B5.6)**:
- NEW `agent/memory/workspaces/feature-system-refactor-2026-05/02_plan_auth_prod_config.md`
- REWRITE `cdc-auth-service/config/config.go` (96 → 121 lines, +25)
- EDIT `cdc-auth-service/config/config-local.yml` (+1 line `mode: dev`)
- NEW `cdc-auth-service/config/config-production.yml` (18 lines)
- NEW `cdc-auth-service/config/config-sample.yml` (19 lines)
- EDIT `cdc-auth-service/deployments/docker/Dockerfile` (-1 line single-file copy → +2 lines folder copy + comment)
- REWRITE `cdc-auth-service/.env.example` (8 → 15 lines, restructured theo Global Pattern actionable)
- NEW `report_phase_b5_6_auth_prod_config_20260505_1158.md` (anh requested for review)

**Lesson abstract → lessons.md (Global Pattern)**: "Image bake `config-local.yml` only" = production deploy nuốt DEV creds + default secrets — root cause: Dockerfile chọn lựa file thay vì copy folder. Pattern: image phải copy CẢ thư mục `config/`, runtime chọn file qua env var (`cfgPath`), prod yml để rỗng cho secrets (env override điền), validateConfig refuse default-placeholder trong production mode. Anti-pattern: 1 image / 1 environment (rebuild image cho từng env).

**Pattern verification**: Áp dụng được cho ≥3 dự án — bất kỳ Go service nào dùng viper + Dockerfile multi-stage. centralized-data-service & cdc-cms-service trong cdc-system có cùng `CFG_PATH` pattern → cùng risk, có thể cần audit tương tự (out of scope phase này).

---
## 2026-05-05 12:00 — B5.6 follow-up: DELETE cdc-auth-service/.env.example (dead weight)

**Trigger**: anh trainguyen — *".env.example đang có cảm giác nó ko xài vì đang dùng go mà"*.

**Verification (real)**:
- `grep -r godotenv cdc-auth-service/` → 0 hit. Go binary KHÔNG auto-load `.env`.
- `go.mod` không import dotenv lib.
- `docker-compose.yml` có defaults `${AUTH_DB_USERNAME:-gpay_admin}` etc. cho cả 3 vars → compose chạy OK không cần `.env`.
- `grep -r ".env.example" cdc-system/` → 0 docs/scripts reference.

**Action**: `rm cdc-auth-service/.env.example`. File untracked (chưa commit), 0 history loss.

**Verify post-delete**:
- `docker compose config` → POSTGRES_USER=gpay_admin, POSTGRES_PASSWORD=gpay_pass, POSTGRES_DB=gpay_auth (defaults work) ✅.
- `curl /health` → 200 (auth-service không regression) ✅.

**Files removed**:
- DELETE `cdc-auth-service/.env.example` (15 lines).

**Lesson abstract → lessons.md**: "Go service `.env.example` = dead weight nếu (no godotenv import) ∧ (compose có defaults)". Decision tree 3 bước: grep godotenv → check compose defaults → check docs references. Anti-pattern: copy `.env.example` template từ Node project sang Go project không check runtime loading.

**Source-of-truth post-cleanup cho cdc-auth-service**:
- DEV: `config/config-local.yml` (committed) — anh `./auth-service` là chạy.
- PROD: `config/config-production.yml` (committed, secrets rỗng) + env injected qua orchestrator (`AUTH_DB_HOST` etc.).
- TEMPLATE: `config/config-sample.yml` cho dev clone.
- Compose: defaults inline trong yml — không cần `.env`.

---
## 2026-05-05 13:35 — B5.6.2 + B5.6.3: centralized-data-service + cdc-cms-service prod-config alignment

**Trigger**: anh trainguyen — *"sau khi làm xong, làm vụ prod-config alignment cho các service còn lại đi"*. Áp pattern B5.6.1 (cdc-auth-service đã verify pass) sang 2 service còn lại.

**Plan trước thi công**: `02_plan_remaining_services_prod_config.md` (NEW, theo CLAUDE.md §7 "Mỗi phase mới → 02_plan_{phase}.md").

### B5.6.2 — centralized-data-service

**Diagnosis (audit thực tế)**:
- `config.go:197-204`: path resolver chỉ NAME (`SetConfigName(path)` với path chứa `/`).
- `config.go:273-392`: `applyEnvOverrides` 25+ blocks `os.Getenv` — **nhưng có parsing logic phức tạp** (`parseNamedURLs` JSON/semicolon, sources↔mongodb.url bridge, DB_SINK_URL → SSLMode=disable side-effect) → KHÔNG thể chuyển sang BindEnv.
- 0 hit "validate" trong file.
- `config/` chỉ có `config-local.yml`.
- `Dockerfile.worker:12`: bake `config-local.yml only` → image prod nuốt creds DEV + JWT placeholder.
- `.env.example` ACTIVE (compose có 16+ `${VAR:-default}` references) → KEEP.

**Fixes applied**:
- Edit `centralized-data-service/config/config.go`:
  - Path resolver dual-mode (cfgPath / CFG_PATH legacy / fallback `./config/config-local.yml`); detect `/` hoặc `.yml/.yaml/.json` suffix → SetConfigFile, else SetConfigName + SetConfigType + multi search paths.
  - KEEP applyEnvOverrides nguyên trạng — CHỈ tách `applyDBFallbacks(cfg)` call ra khỏi cuối hàm (chuyển vào NewConfig).
  - NEW `validateConfig` 5 rules: server.port required; (db.host+db.database) OR db.url OR systemDb.url required; masterDB.urls[default-key] required; jwt.secret required; refuse `change-me-in-production` khi `mode==production`.
  - Sequence trong NewConfig: ReadInConfig → Unmarshal → mergeTopicPrefixAlias → applyEnvOverrides → **validateConfig** → applyDBFallbacks. **Critical**: validation chạy BEFORE fallback để bắt được "cfg.DB.PgxDSN() literal-non-empty garbage" (nếu chạy AFTER, fallback đã set SystemDB.URL = legacy garbage DSN → validation slip past).
- NEW `config/config-production.yml`: prod tunables (pool 100/50, sslMode require, otel sampleRatio 0.1), DSN/secret rỗng, mode: production.
- NEW `config/config-sample.yml`: clone local cho dev/staging template.
- Edit `deployments/docker/Dockerfile.worker`: `COPY config ./config` (cả folder) thay vì single file; EXPOSE 8082 (was 8080); runtime hint comment cho `cfgPath`.

**Verification (real-exercised, 4 smoke)**:
- `go build ./...` EXIT=0; smoke binary `/tmp/cdc-worker-smoke` 50 MB.
- Smoke #1 (default local): config path log → multi-pg connect (control_plane + destination) → V2 metadata reload (sources:7, connections:8, shadow_bindings:8) → NATS/Redis/Mongo connect → scheduler + JobMonitor registered → port :8082.
- Smoke #2 (prod yml, no env): `validate config: DB connection required (set db.host+db.database, db.url, systemDb.url, or env CDC_SYSTEM_DB_URL/DB_SINK_URL)` ← refuse rỗng đúng. Validation BEFORE fallback ngăn được PgxDSN garbage slip.
- Smoke #3 (prod yml + 14 env vars, port :19998): boot E2E → DB connect + NATS + Redis + Mongo + 18 NATS subjects registered. Note: `:9090 metrics` bind fail vì gpay-cdc-worker container đang giữ — không phải config issue.
- Smoke #4 (prod mode + JWT placeholder): `jwt.secret must not use default placeholder in production mode` ← reject.

### B5.6.3 — cdc-cms-service

**Diagnosis**:
- `config.go:87-95`: path resolver chỉ NAME — y hệt centralized.
- `config.go:112-148`: `applyEnvOverrides` 11 simple `os.Getenv → cfg.X = v` mappings, **KHÔNG có parsing đặc biệt** → safe to refactor sang BindEnv.
- ServerConfig thiếu `Mode` field (cần thêm để validateConfig check production mode).
- 0 hit "validate".
- `config/` chỉ có `config-local.yml`.
- `Dockerfile:12`: bake config-local only.
- `.env.example` DEAD WEIGHT: `grep godotenv` 0 hit, KHÔNG có `docker-compose.yml`, KHÔNG xuất hiện như service trong compose nào khác, 0 docs reference → DELETE per decision tree.

**Fixes applied**:
- Rewrite `cdc-cms-service/config/config.go` (149 → 196 lines):
  - ADD `Mode string` field tới `ServerConfig`.
  - Path resolver dual-mode (giống auth/centralized).
  - `v.SetEnvPrefix("CMS")` + `SetEnvKeyReplacer(".", "_")` + `AutomaticEnv()`.
  - Explicit `BindEnv` map 30 keys (CMS_*) với multi-name array cho legacy back-compat: `nats.url` accepts `CMS_NATS_URL` OR `NATS_URL`; `redis.url` accepts `CMS_REDIS_URL` OR `REDIS_URL`; `jwt.secret` accepts `CMS_JWT_SECRET` OR `JWT_SECRET`; `otel.endpoint` accepts `CMS_OTEL_ENDPOINT` OR `OTEL_EXPORTER_OTLP_ENDPOINT`.
  - DELETE `applyEnvOverrides` + `import strconv` (unused).
  - NEW `validateConfig` 6 rules: server.port, db.host, db.database, db.username, jwt.secret required + production-mode placeholder reject.
- Edit `config/config-local.yml`: add `mode: dev`.
- NEW `config/config-production.yml`: pool 50/25, sslMode require, prod tunables, mode: production, DSN/secret rỗng.
- NEW `config/config-sample.yml`: clone local.
- Edit `deployments/docker/Dockerfile`: `COPY config ./config`; EXPOSE 8083 (was 8080); runtime hint comment.
- DELETE `cdc-cms-service/.env.example`.

**Verification (real-exercised, 4 smoke)**:
- `go build ./...` EXIT=0; smoke binary `/tmp/cms-smoke` 57 MB.
- Smoke #1 (default local): config path log → DB connect → NATS/Redis connect → CMS Service started :8083 → system_health_collector + audit_logger + alert_resolver started. Bind :8083 fail vì host PID 18563 đang giữ — config đã pass.
- Smoke #2 (prod yml, no env): `validate config: db.host required (set in YAML or CMS_DB_HOST)` ← refuse đúng.
- Smoke #3 (prod yml + 13 env vars, port :19997): DB + NATS + Redis connect + CMS Service started + `curl /health → 200` + graceful shutdown. ✅ E2E.
- Smoke #4 (prod mode + JWT placeholder): `jwt.secret must not use default placeholder in production mode` ← reject.

### Regression check (post-fix baseline match)

- Host services /health → 200: auth :8081, cms :8083, admin :8090, FE :5173.
- 17 docker containers UP (no restart trigger): gpay-cdc-worker (2h), 4 PG instances (healthy), Mongo + Mariadb + MySQL + Kafka + Connect + SchemaRegistry + NATS + OTel + Redis + 3 SigNoz.
- Host PIDs intact: `cdc-admin-api-f3v2` (PID 21133, uptime 3h59m), `cdc-worker-host` (PID 23565, 3h54m).
- CDC pipeline E2E LIVE: 5 schedules tick 13:34:03 với `last_status='success'` → JobMonitor close-loop (P4/D-39.A từ Phase B3) vẫn hoạt động.

### Files added (cumulative B5.6.2 + B5.6.3)

- NEW `agent/memory/workspaces/feature-system-refactor-2026-05/02_plan_remaining_services_prod_config.md`.
- EDIT `centralized-data-service/config/config.go` (~+45 lines: const + path resolver dual-mode + sequence change + validateConfig).
- NEW `centralized-data-service/config/config-production.yml` (84 lines).
- NEW `centralized-data-service/config/config-sample.yml` (clone local).
- EDIT `centralized-data-service/deployments/docker/Dockerfile.worker` (-1 single-file COPY → +2 folder COPY + comment + EXPOSE 8082).
- REWRITE `cdc-cms-service/config/config.go` (149 → 196 lines, BindEnv pattern + Mode + validateConfig).
- EDIT `cdc-cms-service/config/config-local.yml` (+1 line `mode: dev`).
- NEW `cdc-cms-service/config/config-production.yml` (50 lines).
- NEW `cdc-cms-service/config/config-sample.yml` (clone local).
- EDIT `cdc-cms-service/deployments/docker/Dockerfile` (folder COPY + EXPOSE 8083 + comment).
- DELETE `cdc-cms-service/.env.example` (dead weight, decision tree confirmed).
- NEW `report_phase_b5_6_remaining_services_20260505_1335.md`.

### Lesson abstract → lessons.md

**Global Pattern (mới)**: "Validation BEFORE fallback merging — fallbacks gốc trộn 'derived' vào 'user-set' state, validateConfig sẽ thấy non-empty derived value và miss được intent rỗng của user. Order đúng: ReadConfig → Unmarshal → applyEnvOverrides → **validateConfig** → applyFallbacks/derives."

Anti-pattern: validateConfig chạy AFTER fallbacks → garbage DSN literal-non-empty (`postgres://:@:0/?sslmode=`) slip past validation, app boot OK rồi crash khi connect.

**Tasks completed**: #128-#136 (9 tasks, 8 smoke tests + 1 regression check ALL PASS).
