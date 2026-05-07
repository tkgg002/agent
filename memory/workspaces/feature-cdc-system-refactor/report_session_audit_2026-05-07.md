# Report — Session Audit 2026-05-07

> **Mục đích**: Tổng hợp những gì Muscle đã thay đổi trong session này (đợt G + H Task #19) + audit repo state thực tế (backup vs current) để Boss check lại. Lập theo yêu cầu Boss "Luôn có 1 file report_*.md ghi lại những gì thay đổi để tôi check lại".
> **Author**: Muscle (CC CLI), opus 4.7
> **Date**: 2026-05-07 ICT
> **Trigger feedback**: Boss reject "đợt nhỏ pattern" (đợt G/H), yêu cầu quét repo gốc + report thực tế.

---

## 1. Trạng thái thực tế (verified, không láo)

| Probe | Command | Kết quả thực tế |
|---|---|---|
| Build | `go build ./...` ở `cdc-cms-service` | **PASS** (no output) |
| Tests | `go test ./... -count=1` | **PASS** all packages: `api 3.969s`, `app/commands 1.113s`, `app/queries 3.475s`, `infra/http 2.109s`, `infra/messaging 3.024s`, `infra/persistence 4.415s`, `middleware 1.641s`, `service 2.567s`, `service/health/probes 0.644s` |
| cms-server process | `ps aux \| grep cms` | **CHẠY** PID `33841` binary `/tmp/cdc-cms-service-t27` từ Mon 9:36AM. **CẢNH BÁO**: binary này build TRƯỚC commit đợt G/H — chưa rebuild + restart sau commit |
| cdc-worker process | `ps aux \| grep cdc-worker` | **CHẠY** PID `23565` binary `/tmp/cdc-worker-host` từ Tue 9AM |
| cdc-admin-api process | `ps aux \| grep cdc-admin-api` | **CHẠY** PID `21133` binary `/tmp/cdc-admin-api-f3v2` từ Tue 9AM |
| FE Vite dev server | `ps aux \| grep vite` | **CHẠY** PID `10343` từ Tue 8AM |
| `cdc_internal` schema (Boss bảo đã bỏ) | `grep -rln cdc_internal --include='*.go'` cả hai repo | **0 hit Go code** ✅. SQL còn references trong migrations 018-027 (lịch sử). Migration 028+037+038 đã rename `cdc_internal` → `cdc_system` — Boss đúng. |
| `integration` directory trong cms | `find cdc-cms-service-bk cdc-cms-service -maxdepth 4 -type d -name "*integration*"` | **0 hit** trong cms (gốc + current). Chỉ có ở `centralized-data-service/test/integration` (worker, không thuộc cms) |

**Kết luận trạng thái**: build/test xanh; cms-server đang chạy binary CŨ (trước session đợt G/H). Cần rebuild + restart để verify code commit mới hoạt động trong runtime.

---

## 2. Diff `cdc-cms-service-bk/` (gốc) vs `cdc-cms-service/` (current)

### 2.1. File / dir BỎ khỏi current (`Only in cdc-cms-service-bk/`)

| Path (backup) | Trạng thái | Lý do |
|---|---|---|
| `internal/repository/` (toàn dir) | XÓA | Drained vào `internal/infra/persistence/` qua đợt C→F |
| `internal/service/master_swap.go` | XÓA | Đợt G commit `3424764` — move sang `infra/persistence/` |
| `internal/service/shadow_automator.go` | XÓA | Đợt G commit `3424764` — move sang `infra/persistence/` |
| `internal/service/prom_client.go` (+test) | XÓA | Đợt E commit `55b3afc` — move sang `infra/http/` |
| `internal/service/provisioning_orchestrator.go` | XÓA | Đợt H commit `ff16e38` — move sang `infra/persistence/` |
| `internal/service/provisioning_state_machine.go` | XÓA | Đợt H commit `ff16e38` — move sang `infra/persistence/` |
| `internal/service/reconciliation_service.go` | XÓA | Đợt C — drop dead-code (no-op service) |

### 2.2. File / dir THÊM vào current (`Only in cdc-cms-service/`)

| Path (current) | Loại | Lý do |
|---|---|---|
| `internal/app/` (toàn dir: `commands`, `ports`, `queries`) | MỚI | Hexagonal app layer (CQRS commands + queries + ports) |
| `internal/domain/` (toàn dir: `job`, `mapping`, `master`, `reconciliation`, `source`) | MỚI | Hexagonal domain layer |
| `internal/infra/` (toàn dir: `cache`, `http`, `messaging`, `persistence`) | MỚI | Hexagonal infra layer |
| `internal/api/job_handler.go` | MỚI | JobMonitor close-loop endpoint |
| `internal/middleware/deprecation.go` | MỚI | Deprecation header middleware |
| `internal/service/health/` (probes/) | MỚI | External health probes (debezium, kafka_connect, kafka_lag, nats, postgres, redis, worker, deps) |
| `internal/service/approval_service_test.go` | MỚI | Test uplift T17 P7 đợt 5 |
| `internal/service/source_object_v2_sync_test.go` | MỚI | Test uplift |
| `internal/service/system_health_alerts_test.go` | MỚI | Test uplift T17 P7 |
| `internal/service/system_health_compute.go` (+test) | MỚI | Compute helpers split |
| `internal/service/system_health_queries.go` | MỚI | Query helpers split |
| `pkgs/utils/pg_ident.go` | MỚI | PG identifier helpers |

### 2.3. File MODIFIED (giữ tên, có sửa)

20 file ở `internal/api/`, `internal/middleware/`, `internal/router/router.go`, `internal/server/server.go`, `internal/service/{approval_service,source_object_v2_sync,system_health_collector,system_health_collector_test}.go`, `go.mod`. Đa số mod là caller-update theo các đợt drain.

---

## 3. Những gì Muscle đã commit trong session này

### 3.1. Đợt G — `3424764` (cdc-system repo)
- **Subject**: `refactor(cms): Task #19 đợt G — bulk migrate master_swap + shadow_automator to infra/persistence`
- **Files**: 7 (4 renames @98-99% + 3 caller mods, +17/-19)
- **Renames**:
  - `internal/service/master_swap.{go,_test.go}` → `internal/infra/persistence/...` (99%/99%)
  - `internal/service/shadow_automator.{go,_test.go}` → `internal/infra/persistence/...` (99%/98%)
- **Callers**: `internal/api/master_registry_handler.go`, `internal/api/registry_handler.go`, `internal/server/server.go`
- **Verify**: build PASS, vet clean, tests PASS, DoD grep `service.{MasterSwap,NewMasterSwap,ShadowAutomator,NewShadowAutomator}` → 0 hit

### 3.2. Đợt H — `ff16e38` (cdc-system repo)
- **Subject**: `refactor(cms): Task #19 đợt H — bulk migrate provisioning_orchestrator + state_machine to infra/persistence`
- **Files**: 7 (4 renames @99% + 3 caller mods, +20/-20)
- **Renames**:
  - `internal/service/provisioning_orchestrator.{go,_test.go}` → `internal/infra/persistence/...` (99%/99%)
  - `internal/service/provisioning_state_machine.{go,_test.go}` → `internal/infra/persistence/...` (99%/99%)
- **Callers**: `internal/api/provisioning_handler.go`, `internal/api/provisioning_handler_test.go`, `internal/server/server.go`
- **Technique**: `cp + sed -i '' 's/^package service$/package persistence/'` byte-equivalent move; bulk `sed` cross-3-files cho 14 token sites
- **Verify**: build PASS, vet clean, tests PASS, DoD grep `service.{Provisioning,NewProvisioning,ErrProvisioning,SourceProvisioning}` → 0 hit
- **Cross-module note**: `centralized-data-service/internal/service/provisioning_state_machine.go` là worker-side copy (separate Go module), giữ byte-equivalent với cms copy. Đợt này KHÔNG touch worker-side.

### 3.3. Doc commits (agent repo)
- `df20830` docs(workspace): Task #19 đợt G APPEND progress
- `742ae13` docs(workspace): Task #19 đợt H APPEND progress

### 3.4. Lesson APPEND (agent repo, sau Boss feedback hôm nay)
- `agent/memory/global/lessons.md` — `L-PRE-PLAN-AUDIT` (chưa commit, đợi Boss approve trước khi commit)

---

## 4. Trạng thái `internal/service/` còn lại (7 file source + 7 test + probes/)

```
internal/service/
├── alert_manager.go              (DB-only, 14 hit cdc_dw, 0 external)
├── alert_manager_test.go
├── approval_service.go           (DB-only, 4 hit cdc_dw, 0 external)
├── approval_service_test.go
├── source_object_v2_sync.go      (DB-only, 11 hit cdc_dw, 0 external)
├── source_object_v2_sync_test.go
├── system_health_alerts.go       (pure-fn Collector method, 0 DB, 0 HTTP)
├── system_health_alerts_test.go
├── system_health_collector.go    (DB + 5 hit external HTTP — probe worker /metrics)
├── system_health_collector_test.go
├── system_health_compute.go      (pure-fn, 0 DB, 0 HTTP)
├── system_health_compute_test.go
├── system_health_queries.go      (DB Collector method, 4 hit cdc_dw)
└── health/
    └── probes/
        ├── debezium.go        (external HTTP — Debezium connector REST API)
        ├── deps.go            (generic HTTP probe)
        ├── kafka_connect.go   (external HTTP — Kafka Connect REST)
        ├── kafka_lag.go       (external HTTP — kafka-exporter Prometheus)
        ├── nats.go            (NATS connect probe)
        ├── postgres.go        (DB ping)
        ├── redis.go           (Redis ping)
        ├── worker.go          (external HTTP — worker /health)
        └── *_test.go          (mỗi file 1 test)
```

---

## 5. Boss's 3-bucket model — phân loại

Theo Boss feedback: cms code chia 3 nhóm:
- **Bucket A**: API động cdc_dw (DB metadata) → giữ pattern hexagonal `infra/persistence/`
- **Bucket B**: API cần touch source/dest/shadow → trigger qua cdc-worker (NATS publish)
- **Bucket C**: API gọi external (Debezium, Kafka, Prometheus...) → tách thư mục riêng

| File | Bucket | Action đề xuất |
|---|---|---|
| `alert_manager.go` (+ test) | A | Move `infra/persistence/` |
| `approval_service.go` (+ test) | A | Move `infra/persistence/` |
| `source_object_v2_sync.go` (+ test) | A | Move `infra/persistence/` |
| `system_health_queries.go` | A* (Collector method, đi cùng cluster) | Co-locate với collector |
| `system_health_alerts.go` (+ test) | A* (pure Collector method) | Co-locate với collector |
| `system_health_compute.go` (+ test) | A* (pure-fn) | Co-locate với collector |
| `system_health_collector.go` (+ test) | C (HTTP client gọi worker) | Move sang `infra/external/health/` (cluster cùng nhau) |
| `service/health/probes/*` (8 file) | C | Move sang `infra/external/probes/` |

**NATS publish sites hiện tại** (Bucket B đã đúng pattern):
- `infra/persistence/provisioning_orchestrator.go:250` — fire `cdc.cmd.{shadow.bind,master.bind,discover,schedule.enable}`
- `app/commands/approve_master.go:112` — fire dispatched event
- `api/registry_handler.go:465` — fire `cdc.cmd.batch-transform`

**DB connection**: 1 (`cdc_dw` qua `pkgs/database/postgres.go:22`). Cms KHÔNG động source/dest/shadow trực tiếp ✅.

---

## 6. Verification checklist (cho Boss check lại từng claim)

| # | Claim | Lệnh verify | Expected |
|---|---|---|---|
| 1 | Build cdc-cms-service xanh | `cd cdc-cms-service && go build ./...` | exit 0, no output |
| 2 | Tests cdc-cms-service xanh | `cd cdc-cms-service && go test ./... -count=1` | tất cả `ok` |
| 3 | Đợt G commit landed | `cd cdc-system && git log --oneline -- cdc-cms-service/ \| grep 3424764` | 1 line match |
| 4 | Đợt H commit landed | `cd cdc-system && git log --oneline -- cdc-cms-service/ \| grep ff16e38` | 1 line match |
| 5 | `master_swap` ở persistence | `ls cdc-cms-service/internal/infra/persistence/master_swap.go` | exists |
| 6 | `master_swap` không còn ở service | `ls cdc-cms-service/internal/service/master_swap.go` | "No such file" |
| 7 | `provisioning_orchestrator` ở persistence | `ls cdc-cms-service/internal/infra/persistence/provisioning_orchestrator.go` | exists |
| 8 | `provisioning_state_machine` ở persistence | `ls cdc-cms-service/internal/infra/persistence/provisioning_state_machine.go` | exists |
| 9 | DoD grep stale refs | `grep -rn "service\.\(MasterSwap\|ShadowAutomator\|ProvisioningOrchestrator\)" cdc-cms-service/ --include="*.go"` | 0 hit |
| 10 | `cdc_internal` đã bỏ trong code | `grep -rln cdc_internal cdc-cms-service/ --include="*.go"` | 0 hit |

---

## 7. Đề xuất plan tiếp (PAUSE chờ Boss approve)

**Quan điểm Boss đã rõ**: dừng "đợt-nhỏ pattern". Output 1-2 commit cuối, đóng Task #19.

**Plan đơn giản — 2 commit cuối**:

### Plan A (giữ approach hiện tại, hoàn thiện drainage):
- **Commit cuối #1**: 3 service file DB-only → `infra/persistence/` (alert_manager, approval_service, source_object_v2_sync; 6 file rename)
- **Commit cuối #2**: cluster external + probes → `infra/external/{health,probes}/` (4 system_health_* + 8 probes/* file; ~12 file rename)
- Sau đó: `internal/service/` rỗng, drainage Task #19 đóng hoàn toàn.

### Plan B (skip nếu đã đủ):
- Boss có thể muốn KHÔNG drain tiếp 7 file cuối — vì cms hiện tại đã hoạt động đúng pattern (cms động cdc_dw qua persistence; cần touch source/dest/shadow đã đi qua NATS).
- Drain 7 file cuối = cosmetic refactor, không thay đổi behavior.
- Nếu Boss chọn Plan B → đóng Task #19 ngay tại HEAD `ff16e38`.

### Câu hỏi cho Boss
1. **Q1**: Drain tiếp 7 file (Plan A) hay đóng Task #19 ngay (Plan B)?
2. **Q2**: Nếu Plan A — `infra/external/` thực ra đã có pattern qua `infra/http/` (kafka_connect.go, prom_client.go). Move probes/ vào `infra/external/probes/` là dir mới hay reuse `infra/http/probes/`?
3. **Q3**: Có cần rebuild + restart cms-server (`/tmp/cdc-cms-service-t27`) sau đợt G+H để verify runtime hoạt động?

---

## 8. Lessons mới ghi (CLAUDE.md §7 mid-session)

- `agent/memory/global/lessons.md` — APPEND lesson `L-PRE-PLAN-AUDIT` về việc plan refactor mà không quét backup repo trước → làm rối + đợt nhỏ kéo lê. Pattern: bất kỳ refactor task nào lên codebase X có backup B song song → diff(B,X) là step #0 BẮT BUỘC.

---

## 9. Skills used trong session này
- Bash (grep / diff -rq / find / ps / git log / go build / go test)
- Read (lessons.md, project_context.md, active_plans.md, source files)
- Edit (caller updates trong api/* + server.go)
- Write (4 file mới ở infra/persistence + report này)
- Git rename detection (≥98% similarity giữ history)
- §3 Plan & Verify (mid-session re-plan sau Boss feedback)
- §7 Knowledge retention (lesson APPEND mid-session)
- §11 APPEND-only memory (workspace progress + report)

---

**Status**: Pause chờ Boss xác nhận Plan A hay Plan B + Q2/Q3 trước khi tiếp tục.
