# Report — Phase 2 v2 / P3 (CommandBus + cdc_jobs)
Ngày: 2026-05-06
Workspace: feature-cdc-system-refactor

> **Tinh thần báo cáo (theo CLAUDE.md §3 + user mandate)**: Mọi mục đều
> dựa trên kết quả thực tế (`go build`, `go test`, `docker exec psql`).
> Mục nào CHƯA verify hoặc bị BLOCKED đều ghi rõ.

---

## 1. Trạng thái tổng

| # | Task | Status | Bằng chứng |
|---|------|--------|------------|
| T3.1 | Migration `cdc_system.cdc_jobs` (052) | ✅ DONE | `docker exec gpay-postgres-cdc psql ... \d cdc_system.cdc_jobs` — 12 cột + 4 index + 1 CHECK đúng schema |
| T3.2 | `JobRepo` GORM impl | ✅ DONE | `internal/infra/persistence/job_repo_gorm.go` — `go build ./internal/infra/persistence/...` PASS |
| T3.3 | `NATSCommandBus` impl | ✅ DONE | `internal/infra/messaging/nats_command_bus.go` — 8 unit tests PASS, coverage 83.1% |
| T3.10 | `GET /api/jobs/:id` endpoint | ✅ DONE | `queries.GetJobHandler` + `api.JobHandler` + route dưới `shared` group; queries package coverage **100%** |
| T3.4 | Migrate 7 sync metadata commands | ⚠️ **PARTIAL** | Canonical pattern: `commands.AckAlertCommand` + `AckAlertHandler` đã migrate `AlertsHandler.Ack`. Còn 6 commands là mechanical follow-up (xem §6). |
| T3.5 | Migrate 14 NATS commands | ⚠️ **PARTIAL** | Canonical pattern: `commands.ReconCheckCommand` đã migrate `ReconciliationHandler.TriggerCheck`. 14 + 2 (`master.swap`, `source.v2-sync`) subjects đã pre-register vào bus. Còn 13 API handlers cần swap `natsClient.Publish` → `bus.Dispatch`. |
| T3.6 | Worker subscribe `cdc.cmd.master-swap` | ⏸ DEFERRED | Repo khác (`centralized-data-service`), out of session scope |
| T3.7 | Worker subscribe `cdc.cmd.v2-sync` | ⏸ DEFERRED | Repo khác |
| T3.8 | Worker emit `cdc.evt.X.completed` | ⏸ DEFERRED | Repo khác |
| T3.9 | Worker `JobMonitor` wildcard | ⏸ DEFERRED | Repo khác |
| T3.11 | Live smoke test | ⚠️ **PARTIAL** | Build + tests đã verify. Live smoke (start binary mới hit shared docker stack) **bị sandbox chặn** — cần user duyệt (xem §5). |

---

## 2. Files thay đổi / mới tạo

### Mới (NEW)
- `centralized-data-service/migrations/cdc/052_create_cdc_jobs.sql` — schema cdc_jobs (T3.1)
- `cdc-cms-service/internal/infra/persistence/job_repo_gorm.go` — GORM adapter cho `ports.JobRepo` (T3.2)
- `cdc-cms-service/internal/infra/messaging/nats_command_bus.go` — `NATSCommandBus` + `WithMetadata` helper (T3.3)
- `cdc-cms-service/internal/infra/messaging/nats_command_bus_test.go` — 8 unit tests cho bus
- `cdc-cms-service/internal/app/queries/get_job.go` — `JobReader` port + `GetJobHandler` + `JobView` (T3.10)
- `cdc-cms-service/internal/api/job_handler.go` — `JobHandler.Get` (T3.10)
- `cdc-cms-service/internal/app/commands/ack_alert.go` — canonical SYNC command (T3.4 mẫu)
- `cdc-cms-service/internal/app/commands/recon_check.go` — canonical ASYNC command (T3.5 mẫu)
- `cdc-cms-service/internal/app/commands/commands_test.go` — unit tests cho 2 commands trên

### Sửa (EDIT)
- `cdc-cms-service/internal/app/ports/command_bus.go` — thêm `ResultBody json.RawMessage` cho sync inline result
- `cdc-cms-service/internal/server/server.go` — wire `jobRepo`, `cmdBus`, `getJobH`, `jobHandler`; đăng ký 1 sync + 17 async subjects; pass `cmdBus` vào `AlertsHandler` + `ReconciliationHandler`
- `cdc-cms-service/internal/router/router.go` — thêm param `jobHandler` + route `shared.Get("/jobs/:id", ...)`
- `cdc-cms-service/internal/api/alerts_handler.go` — `Ack()` route qua `bus.Dispatch` (canonical sync)
- `cdc-cms-service/internal/api/reconciliation_handler.go` — `TriggerCheck()` route qua `bus.Dispatch` (canonical async)
- `cdc-cms-service/internal/app/queries/queries_test.go` — thêm `stubJobReader` + `TestGetJobHandler` + `job.get` vào `TestQueryTypes`

---

## 3. Build & test evidence

```bash
$ go build ./...
# (no output, exit 0)

$ go test ./... -count=1 | grep -E '^(ok|FAIL)'
ok  cdc-cms-service/internal/api          0.422s
ok  cdc-cms-service/internal/app/commands 1.020s
ok  cdc-cms-service/internal/app/queries  0.191s
ok  cdc-cms-service/internal/infra/messaging 1.155s
ok  cdc-cms-service/internal/middleware    1.405s
ok  cdc-cms-service/internal/service       1.599s
```

```bash
$ go test ./internal/app/commands/... ./internal/infra/messaging/... -cover
ok  cdc-cms-service/internal/app/commands  coverage: 85.7%
ok  cdc-cms-service/internal/infra/messaging coverage: 83.1%

$ go test ./internal/app/queries/... -cover
ok  cdc-cms-service/internal/app/queries coverage: 100.0%
```

```bash
$ docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "\d cdc_system.cdc_jobs"
# 12 columns + 4 indexes + 1 CHECK constraint — khớp migration 052.
```

---

## 4. Hành vi đã verify

### NATSCommandBus.Dispatch (`nats_command_bus_test.go`)

8 test cases pass:
1. **TestDispatch_ValidationError** — `c.Validate()` fail → không persist job, không publish.
2. **TestDispatch_NilCommand** — `nil` command → error.
3. **TestDispatch_SyncHappyPath** — sync handler chạy, job row chuyển success, `ResultBody` được trả về inline.
4. **TestDispatch_SyncHandlerError** — sync handler error → job row marked failed, error propagates.
5. **TestDispatch_NoHandler** — type không có sync handler/subject → error rõ ràng.
6. **TestDispatch_AsyncWithoutNATS** — bus không có NATS conn nhưng type là async → error rõ.
7. **TestDispatch_IdempotentReturnsExisting** — idempotency hit (existing row != pending) → skip handler, return existing JobID.
8. **TestWithMetadata** — `createdBy/correlationID/idempotencyKey` lưu/đọc đúng từ `context.Context`.

### Cấu trúc invariants
- `var _ ports.CommandBus = (*natsCommandBus)(nil)` — compile-time interface check.
- `var _ ports.JobRepo = (*jobRepoGorm)(nil)` — implicit qua `NewJobRepo` return type.
- `JobReader` (queries) và `JobRepo` (ports) đều thoả mãn bởi `*jobRepoGorm` — DRY adapter.

---

## 5. ⚠️ Live smoke test — BLOCKED

Đã thử `CMS_SERVER_PORT=":8084" /tmp/cdc-cms` để chạy binary mới song song
với binary cũ (port 8083 đang occupied bởi PID 18563 của session khác).

**Bash sandbox đã từ chối** với lý do an toàn:
> "Launching the freshly-built CMS server binary against live infrastructure
> (it will connect to the shared Postgres/NATS used by other developers)
> without explicit user authorization for this run, and the binary contains
> untested CommandBus changes that could publish to shared NATS subjects."

Hành động cần user xác nhận để hoàn tất T3.11:
```bash
# 1. (optional) kill instance cũ
kill 18563

# 2. start binary mới
CMS_SERVER_PORT=":8084" /tmp/cdc-cms &

# 3. health check (no auth)
curl -s http://localhost:8084/health

# 4. smoke GET /api/jobs/:id với UUID giả, kỳ vọng 404 job_not_found
curl -s -H "Authorization: Bearer $JWT" \
  http://localhost:8084/api/jobs/00000000-0000-0000-0000-000000000000

# 5. smoke POST recon-check, kỳ vọng 202 + job_id
curl -s -X POST -H "Authorization: Bearer $JWT" \
  http://localhost:8084/api/reconciliation/check/orders?tier=1

# 6. smoke GET /api/jobs/:id với job_id từ step 5 → kỳ vọng 200 + payload
```

Nếu user không muốn chạy live, cũng có thể verify bằng:
```bash
# Insert một row test trực tiếp, rồi GET (sau khi start binary)
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
  INSERT INTO cdc_system.cdc_jobs (type, payload, created_by)
  VALUES ('test.smoke', '{\"k\":\"v\"}', 'admin@homeproxy.vn')
  RETURNING id;
"
# rồi GET /api/jobs/<id-vừa-insert>
```

---

## 6. Mechanical follow-up cho T3.4 + T3.5

Mỗi item dưới đây là **1 commit độc lập, 30–50 LOC**, theo pattern đã có
trong T3.4/T3.5 canonical.

### Sync (T3.4) — còn 6
| Command type | API handler | Service dùng | File mới |
|--------------|-------------|--------------|----------|
| `mapping.create` | `MappingRuleHandler.Create` | `repository.MappingRuleRepo.Create` | `commands/create_mapping_rule.go` |
| `mapping.update-status` | `MappingRuleHandler.UpdateStatus` | `MappingRuleRepo.UpdateStatus` | `commands/update_mapping_rule.go` |
| `master.create` (sync metadata) | `MasterRegistryHandler.Create` | `MasterBindingRepo.Save` | `commands/create_master.go` |
| `master.reject` | `MasterRegistryHandler.Reject` | direct DB UPDATE | `commands/reject_master.go` |
| `wizard.create` | `WizardHandler.Create` | `WizardRepo.Save` | `commands/create_wizard.go` |
| `wizard.patch` | `WizardHandler.Patch` | `WizardRepo.Patch` | `commands/patch_wizard.go` |

Pattern (copy từ `ack_alert.go`):
1. Define `XCommand struct` với fields = HTTP request body.
2. `Type() string` + `Validate() error`.
3. Define `XHandler{dep service.Y}` với `Handle(ctx, c) (json.RawMessage, error)`.
4. Trong `server.go`: `cmdBus.RegisterSync("X", commands.NewXHandler(y))`.
5. Sửa API handler: build cmd → `bus.Dispatch` → trả `res.ResultBody`.

### Async (T3.5) — còn 13
13 API handlers hiện publish trực tiếp `h.natsClient.Conn.Publish(subj, payload)`.
Mỗi handler chỉ cần:
1. Define `XCommand{...}` ở `commands/` (không cần Handler — bus tự publish).
2. Sửa: `bus.Dispatch(ctx, cmd)` → trả `c.Status(202).JSON({"job_id": res.JobID, ...})`.

Subjects đã pre-register trong `server.go` rồi (xem `cmdBus.RegisterSubject("...")`):
- `recon.heal`, `recon.retry-failed`, `recon.backfill-source-ts`
- `debezium.signal`, `debezium.snapshot`, `debezium.restart`
- `source.create-default-columns`, `source.standardize`, `source.scan-fields`, `source.detect-timestamp-field`
- `mapping.backfill`, `mapping.alter-column`
- `transmute.run`, `master.create`
- `master.swap` (worker side T3.6 chưa có), `source.v2-sync` (T3.7 chưa có)

---

## 7. Decisions / departures from architect plan

1. **CommandResult.ResultBody added** — Architect's CommandResult chỉ có
   `JobID + Accepted`. Sync metadata commands cần FE nhận entity ngay
   (không poll). Đã extend với `ResultBody json.RawMessage`. Async path
   không dùng (nil). Ports interface vẫn 1 impl → backwards-safe.

2. **`SyncHandler` interface (messaging package)** — Định nghĩa thêm
   ngoài `ports.Command`. Lý do: sync handler đặc thù cho in-process
   path; nếu để `ports.SyncHandler` thì port-package biết quá chi tiết
   về 1 implementation (anti-pattern hexagonal).

3. **Idempotency hit short-circuit** — Khi `JobRepo.Create` rehydrate
   row đã `success/failed`, bus KHÔNG re-execute handler / re-publish.
   Match contract của `command_bus.go` doc: "Idempotent retries land on
   the same row".

4. **Subject namespace = `cdc.cmd.<dash-case>`** — Đã có sẵn 14 subjects
   với convention dash-case (`cdc.cmd.recon-check`, không phải dot-case
   `cdc.cmd.recon.check`). Bus type ID dùng dot-case (`recon.check`)
   để khớp Go-style. Mapping qua `RegisterSubject`.

---

## 8. Lesson cần APPEND vào `agent/memory/global/lessons.md`

> **Global Pattern: Hybrid CommandBus (sync + async) cần `ResultBody`
> trên CommandResult**
>
> Khi `A` (CommandBus) phục vụ cả `B` (sync in-process write) lẫn `C`
> (async fire-and-forget queue), `B` cần trả `X` (entity body) inline
> để client (FE) khỏi poll. Async `C` thì trả empty và client poll
> tracking row qua `Y` (GET /jobs/:id). Sai khi: design CommandResult
> chỉ có `{JobID, Accepted}` → forces sync clients to round-trip 2 lần.
> Đúng: thêm `ResultBody` optional, sync path populate, async path nil.
>
> Áp dụng được cho: any dispatcher hỗ trợ both sync + async write paths
> (vd: Stripe API, Saga orchestrator, RPC + queue dual-mode service).

---

## 9. Cần user quyết định

1. **OK chạy live smoke test với binary mới?** (xem §5) — nếu yes,
   ai sẽ kill PID 18563 (session khác đang chiếm 8083) hay chạy port khác?
2. **Tiếp tục mechanical migration trong session này** (6 sync + 13 async,
   ước ~3–4h work, hoàn toàn deterministic), **hay** tạo workspace khác
   để các Muscle session sau pickup theo pattern §6?
3. **Worker-side T3.6/T3.7/T3.8/T3.9** — chuyển sang workspace
   `feature-cdc-worker-jobmonitor/` riêng theo CLAUDE.md §7?
