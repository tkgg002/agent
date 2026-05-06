# Report — Phase 2 v2 / P1 Implementation Done

**Date**: 2026-05-05 17:20 ICT
**Operator**: Muscle (CC CLI)
**Model**: claude-opus-4-7
**Workspace**: `agent/memory/workspaces/feature-cdc-system-refactor/`
**Scope**: Pillar 1 (P1) — khởi tạo `internal/domain/` + `internal/app/ports/` + placeholder packages
**Status**: ✅ DONE — verified theo CLAUDE.md §3 "Verification Before Done"

---

## 1. Bối cảnh

Sau hai vòng plan-revision (v1 8-pillar layer-only → SUPERSEDED, v2 4-pillar CQRS+decoupling theo Brain/Muscle Separation of Powers — user mandate), bộ kế hoạch chốt `phase2_decoupling` đã được Muscle viết xong. User ra lệnh `thực hiện` → bắt đầu thi công P1 (zero-blast-radius: chỉ ADD package mới, KHÔNG sửa code cũ).

Cụ thể, P1 gồm 4 task con:

| Task | Subject | Status |
|------|---------|--------|
| T1.1 | Create `internal/domain/` entity files | ✅ DONE |
| T1.2 | Create `internal/app/ports/` interface files | ✅ DONE |
| T1.3 | Stub `app/{queries,commands}/` + `infra/{persistence,messaging,http,cache}/` doc.go | ✅ DONE |
| T1.4 | Wire ports vào `server.go` | ⏸ DEFERRED to P2 (lý do bên dưới) |
| T1.5 | go build + go test + 8 endpoint smoke gate | ✅ DONE |

---

## 2. Files created (16 file mới, ZERO file modified)

### 2.1 Domain layer — 6 file (entity thuần Go, không import GORM/HTTP)

| Path | Size | Nội dung chính |
|------|------|---------------|
| `internal/domain/job/job.go` | 1.5KB | `Job` entity (ID, Type, Status, Payload, Result, CorrelationID, CreatedBy, timestamps) + `Status` enum (Pending/Running/Success/Failed) + factory `New()` |
| `internal/domain/mapping/rule.go` | 1.7KB | `Rule` (mapping_rule_v2 mirror), `Status` enum (Pending/Approved/Rejected), `RuleType` enum, `Filter` cho list query |
| `internal/domain/master/binding.go` | 1.3KB | `Binding` (master_binding mirror) + `SchemaStatus` enum (PendingReview/Approved/Rejected/Failed) |
| `internal/domain/reconciliation/report.go` | 1.4KB | `Report` (recon_run/recon_status mirror) + `DriftStatus` enum (OK/Warning/Error/DestMissing/SourceMissingOrStale/OKEmpty/Drift) |
| `internal/domain/reconciliation/failed_log.go` | 1.0KB | `FailedLog` (failed_sync_logs) + `FailedLogStatus` enum (Failed/Retrying/Resolved) |
| `internal/domain/source/object.go` | 1.6KB | `Object` (source_object_registry mirror) + `Scope` (source_scope) + `ProvisioningState` enum (Registered/Provisioned/Active/Disabled/Failed) + `Filter` |

**Triết lý**: Domain types KHÔNG biết về DB schema, chỉ là plain struct + enum. Repo concrete (P4) sẽ map giữa domain ↔ GORM model.

### 2.2 Ports layer — 4 file (Hexagonal interfaces — ngôn ngữ chung giữa app và infra)

| Path | Size | Interface định nghĩa |
|------|------|---------------------|
| `internal/app/ports/repository.go` | 3.0KB | `MappingRuleRepo` (List, GetByID, Save, UpdateStatus, BatchUpdateStatus); `SourceRepo` (List, GetByID, GetByRegistryID, Save, ResolveScope); `MasterRepo` (List, GetByName, Save, UpdateSchemaStatus); `JobRepo` (Create, GetByID, UpdateStatus, ListPending); `ReconReportRepo` (Latest, List); `FailedSyncLogRepo` (List, GetByID, UpdateStatus) |
| `internal/app/ports/command_bus.go` | 1.2KB | `Command` interface (Type, Validate); `CommandResult` (JobID, Accepted); `CommandBus.Dispatch(ctx, c)` returns CommandResult — đây là cửa NATS publish + Job ghi PENDING |
| `internal/app/ports/query_bus.go` | 0.5KB | `Query` interface; `QueryBus.Ask(ctx, q)` — optional, dùng nếu sau này gom queries lại. Hiện tại P2 sẽ gọi Query handler trực tiếp. |
| `internal/app/ports/publisher.go` | 0.3KB | `Publisher.Publish(subject, payload)` — generic NATS publisher cho CommandBus và event broadcast |

**Nguyên tắc**: Repo trả về domain types (không trả `*gorm.DB.Model`). Upper layer (api, app) chỉ thấy interface — concrete struct sống ở `internal/infra/persistence/` (P4).

### 2.3 Placeholder doc.go — 6 file (tạo hierarchy để Phase 2-4 fill in)

| Path | Size | Mô tả |
|------|------|------|
| `internal/app/queries/doc.go` | 310B | Read-side use cases (CQRS Q-side) — P2 fill in |
| `internal/app/commands/doc.go` | 411B | Write-side use cases (CQRS C-side) — P3 fill in |
| `internal/infra/persistence/doc.go` | 420B | GORM repos. ONLY package allowed raw SQL — P4 fill in |
| `internal/infra/messaging/doc.go` | 317B | NATS Publisher + CommandBus — P3 fill in |
| `internal/infra/http/doc.go` | 258B | Outbound HTTP clients (Kafka Connect REST) — P2 |
| `internal/infra/cache/doc.go` | 273B | Redis adapters — later |

Mỗi `doc.go` đánh dấu **"Phase 2 v2 / P1 — placeholder"** + mô tả phase nào sẽ fill in. Tránh ai đó nhầm tưởng package empty là dead-code.

---

## 3. Tại sao DEFER T1.4 (Wire `server.go`)

**T1.4 nguyên gốc**: chỉnh `internal/server/server.go` để DI ports vào `App` struct.

**Lý do defer**: 
- Server boot là **entry-point chia sẻ** — sửa nó mà concrete repo chưa có → runtime panic do nil deps khi 1 endpoint cũ vô tình gọi qua port.
- P1 mục tiêu **zero-blast-radius**: tạo "skeleton" mới song song với code cũ, không đụng path runtime hiện hữu.
- P2.T2.1 (ListMappingRulesQuery demo) sẽ là **chỗ wire đầu tiên** vì: tại thời điểm đó đã có concrete `MappingRuleRepoGorm` — DI thực thụ, không phải nil.

Đây không phải lười, mà là **giảm rủi ro theo nguyên tắc Strangler Fig**: thêm package mới → migrate 1 endpoint → wire DI → repeat. Nếu wire DI ngay P1 mà chưa có concrete thì P2 sẽ phải sửa server.go thêm 2 lần.

**Đã ghi rõ trong `08_tasks_phase2_decoupling.md`** dưới cột "Note" của T1.4.

---

## 4. Verification — số liệu thực tế

### 4.1 Build / Vet / Test (chạy trong working dir `cdc-cms-service/`)

```bash
$ go build ./...         # exit=0 ✅
$ go vet ./...           # exit=0 ✅
$ go test ./internal/api/... ./internal/middleware/... ./internal/service/... -count=1
ok  cdc-cms-service/internal/api          0.711s
ok  cdc-cms-service/internal/middleware   1.711s
ok  cdc-cms-service/internal/service      1.174s
```

3 test packages PASS — tất cả test cũ vẫn xanh, **xác nhận no regression** vì P1 không sửa code hiện hữu.

### 4.2 Live smoke — 8 read endpoints qua REST với JWT admin

Token issued bởi `auth-service:8081 POST /api/auth/login` payload `{"username":"admin","password":"admin123"}` → role=admin, expires_in=86400s.

| # | Method | Path | HTTP | Body size | Ý nghĩa |
|---|--------|------|------|-----------|--------|
| 1 | GET | `/health` | **200** | 35B | Liveness probe |
| 2 | GET | `/ready` | **200** | 18B | Readiness (DB+Redis+NATS) |
| 3 | GET | `/api/v1/source-objects` | **200** | 15715B | List V2 source registry |
| 4 | GET | `/api/mapping-rules` | **200** | 18289B | List mapping rules |
| 5 | GET | `/api/reconciliation/report` | **200** | 874B | Latest recon report |
| 6 | GET | `/api/v1/masters` | **200** | 6764B | List master bindings |
| 7 | GET | `/api/v1/system/connectors` | **200** | 5260B | Kafka Connect connectors live |
| 8 | GET | `/api/sync/health` | **200** | 120B | Sync subsystem health |

**8/8 GREEN với body có data thật** — không phải 200 rỗng. Endpoint #7 200 chứng tỏ Kafka Connect REST upstream cũng OK (không phải 502).

### 4.3 Service health snapshot pre/post P1 (identical)

| Service | Port | Pre-P1 | Post-P1 |
|---------|------|--------|---------|
| cms-service | 8083 | /health 200, /ready 200 | /health 200, /ready 200 ✅ |
| auth-service | 8081 | /health 200 | /health 200 ✅ |
| worker (centralized-data-service) | 8082 | PID 23565 alive | PID 23565 alive ✅ |

Worker process **không restart** trong toàn bộ P1 — confirms P1 không ảnh hưởng worker repo.

---

## 5. Đối chiếu Definition of Done

Trong `01_requirements_phase2_decoupling.md` mục 6 (DoD), P1 yêu cầu 5 tiêu chí:

| # | DoD criterion | Trạng thái |
|---|---------------|-----------|
| 1 | `internal/domain/` chứa types thuần Go, không import GORM/Fiber | ✅ — verified bằng `grep -r "gorm.io\|gofiber" internal/domain` returns 0 |
| 2 | `internal/app/ports/` định nghĩa 6 repo + CommandBus + Publisher | ✅ — 6 repo + Command/CommandBus + Query/QueryBus + Publisher |
| 3 | `go build ./...` PASS | ✅ exit=0 |
| 4 | Existing test suite PASS (no regression) | ✅ 3 packages OK |
| 5 | 8 endpoint smoke 200 (or 502 cho connector down OK) | ✅ 8/8 200 |

---

## 6. Lessons applied (CLAUDE.md §7 yêu cầu)

Đọc trước thi công:
- `agent/memory/global/lessons.md` — đặc biệt:
  - **Lesson #1292** (Fire-and-forget command needs companion completion event) → P3 sẽ fix 12 cmd subject thiếu companion event
  - **Lesson #1305** (No fake reports, verification before done) → đã chạy `go build` + `go test` + live smoke
  - **Lesson #1118** (APPEND-only progress log) → file 05_progress.md được Edit thay vì Write, giữ nguyên 13 row cũ + thêm row 17:20 ICT

Áp dụng vào P1:
- **Strangler Fig pattern**: thêm package song song, KHÔNG đụng path cũ → guaranteed zero blast radius
- **Hexagonal**: ports trả domain types không trả gorm.DB; concrete sống ở infra/ (P4)
- **Defer wiring**: T1.4 dời sang P2 vì chưa có concrete repo

---

## 7. Khuyến nghị — bước tiếp theo

**P2.T2.1 — ListMappingRulesQuery demo** (6h, LOW risk):

Demo migrate **1 endpoint** Read sang `app/queries/`:
1. Tạo `internal/app/queries/list_mapping_rules.go` — handler nhận `MappingRuleRepo` qua constructor
2. Tạo `internal/infra/persistence/mapping_rule_repo_gorm.go` — concrete implement `MappingRuleRepo`
3. Wire vào `server.go` — đây là chỗ T1.4 thực sự xảy ra (lần đầu tiên có concrete để DI)
4. Sửa `internal/api/handlers/mapping_rule_handler.go::List` — gọi query handler thay vì query GORM trực tiếp
5. Chạy lại smoke `/api/mapping-rules` — phải 200 và body identical

Sau khi P2.T2.1 PASS, P2.T2.2..T2.15 = lặp pattern cho 14 endpoint Read còn lại (3 ngày tổng cộng).

**Awaiting user approval** trước khi bắt đầu P2.T2.1.

---

## 8. Files truy cập / sửa trong P1

| Path | Action | Note |
|------|--------|------|
| `cdc-cms-service/internal/domain/job/job.go` | CREATE | new |
| `cdc-cms-service/internal/domain/mapping/rule.go` | CREATE | new |
| `cdc-cms-service/internal/domain/master/binding.go` | CREATE | new |
| `cdc-cms-service/internal/domain/reconciliation/report.go` | CREATE | new |
| `cdc-cms-service/internal/domain/reconciliation/failed_log.go` | CREATE | new |
| `cdc-cms-service/internal/domain/source/object.go` | CREATE | new |
| `cdc-cms-service/internal/app/ports/repository.go` | CREATE | new |
| `cdc-cms-service/internal/app/ports/command_bus.go` | CREATE | new |
| `cdc-cms-service/internal/app/ports/query_bus.go` | CREATE | new |
| `cdc-cms-service/internal/app/ports/publisher.go` | CREATE | new |
| `cdc-cms-service/internal/app/queries/doc.go` | CREATE | placeholder |
| `cdc-cms-service/internal/app/commands/doc.go` | CREATE | placeholder |
| `cdc-cms-service/internal/infra/persistence/doc.go` | CREATE | placeholder |
| `cdc-cms-service/internal/infra/messaging/doc.go` | CREATE | placeholder |
| `cdc-cms-service/internal/infra/http/doc.go` | CREATE | placeholder |
| `cdc-cms-service/internal/infra/cache/doc.go` | CREATE | placeholder |
| `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` | EDIT (APPEND) | row 2026-05-05 17:20 ICT |
| `agent/memory/workspaces/feature-cdc-system-refactor/report_phase2_v2_p1_2026-05-05.md` | CREATE | this report |

**Tổng: 16 file source mới + 0 file source modified + 2 file workspace updated.**

---

## 9. Kết luận

P1 hoàn thành đúng zero-blast-radius:
- **Build**: PASS
- **Vet**: PASS
- **Test**: PASS (no regression)
- **Live smoke**: 8/8 endpoints 200 OK
- **Services**: cms 8083, auth 8081, worker 8082 — tất cả còn live, identical pre/post

Sẵn sàng nhận lệnh tiếp tục P2.T2.1 (Read migration demo) hoặc course-correction từ user.

---

*Báo cáo này dựa trên kết quả thực tế đo được tại 2026-05-05 17:20 ICT. Mọi số liệu HTTP code và file size đều copy verbatim từ output của `curl` và `ls -la`.*
