# Báo cáo — Phase 2 cdc-cms-service Refactor — Plan đã sẵn sàng

> **Ngày**: 2026-05-05 16:30 ICT
> **Operator**: Muscle (Claude Code, claude-opus-4-7)
> **Phạm vi**: Plan only — KHÔNG touch source code. Awaiting user approval.
> **Workspace**: `agent/memory/workspaces/feature-cdc-system-refactor/`

---

## 1. Tóm tắt yêu cầu user

> "lên plan refactor cdc-cms-service ... đầu tiên define cấu trúc source. xem các endpoint của nó, xem nó chạy những nhóm model tính năng nào. nó gọi qua những service nào. chức năng cốt lõi của nó là làm gì ... đọc lesson trước tất cả ... làm theo core /agent ... khi report phải dựa trên kết quả thực tế ... khi kết thúc luôn kiểm tra các service work mới báo done. Luôn có 1 file report_*.md ghi lại những gì thay đổi để tôi check lại."

Translation → 5 ràng buộc cứng:
1. Plan refactor (KHÔNG execute) — define source structure, endpoints, model groups, service deps, core function.
2. Đọc `lessons.md` TRƯỚC mọi việc.
3. Tuân thủ core `/agent` (CLAUDE.md §7, §11, §12).
4. Verification phải thực tế (exercise-driven, không health-driven — Lesson #1264).
5. Verify service work trước khi báo done. **Luôn có file `report_*.md`** để user review.

---

## 2. Inventory (đã khảo sát thực tế bằng Explore agent + grep)

### 2.1 Cấu trúc source

- **76 file `.go`** under `cdc-cms-service/`, 10 file test (coverage tổng ~13%).
- Layer hiện tại:
  - `cmd/server/main.go` — bootstrap.
  - `internal/api/` — 23 handler file (Fiber v2). 4 godfile vi phạm "≤200 line":
    - `reconciliation_handler.go` 894 dòng
    - `source_object_actions_handler.go` 693 dòng
    - `mapping_rule_handler.go` 689 dòng
    - `master_registry_handler.go` 667 dòng
  - `internal/service/` — 18 file. Mix workflow service + utility.
  - `internal/repository/` — **0 abstraction cho V2 schema** (`cdc_system.*`). Mọi query V2 raw SQL nằm trong handler.
  - `internal/middleware/` — JWT + RBAC tiers (admin, operator, ops-admin, destructive).
  - `internal/server/server.go` — DI graph + router register.

### 2.2 API surface (operator control plane)

8 nhóm endpoint chính:

| Nhóm | Path prefix | Handler | Endpoint count |
|---|---|---|---|
| Health | `/health`, `/api/system/health`, `/api/sync/health` | `health_collector.go`, `system_health_handler.go` | 4 |
| Source registry V1 | `/api/v1/source-objects` | `registry_handler.go` (V1 facade) | 6 |
| Source registry V2 | `/api/source-objects/*/{action}` | `source_object_actions_handler.go` | 13 |
| Mapping rule | `/api/mapping-rules*` | `mapping_rule_handler.go` | 9 |
| Master binding | `/api/v1/masters*`, `/api/master-bindings*` | `master_registry_handler.go` | 8 |
| Reconciliation | `/api/reconciliation/*`, `/api/failed-sync-logs` | `reconciliation_handler.go` | 4 |
| Connector ops | `/api/v1/system/connectors*`, `/api/connector-ops/*` | `connector_handler.go` | 6 |
| Wizard / alerts / users / admin | `/api/wizard/*`, `/api/alerts/*`, `/api/users/*`, `/api/admin/*` | `wizard_handler.go`, `alerts_handler.go`, `users_handler.go`, `audit_handler.go` | 12+ |

### 2.3 Model groups

- **V1 legacy** (`public.cdc_*`): `cdc_source_object_registry`, `cdc_master_registry`, `cdc_mapping_rule`, `cdc_failed_sync_log`, `cdc_activity_log` — đã settled.
- **V2 control plane** (`cdc_system.*`): `source_object_registry`, `shadow_binding`, `master_binding`, `mapping_rule_v2`, `connection_registry`, `wizard_session`, `transmute_schedule`, `alert_rule`, `alert_event`, `admin_action_log` — schema settled qua phase 38/39 cms-fe-overhaul.

### 2.4 Service deps (out-bound)

- **Postgres** (`cdc_dw` DB) qua GORM — lưu V1 + V2 metadata.
- **NATS** — publish 18 subject (cdc-cms-service là publisher only, không consume).
- **Redis** — 5 key pattern (snapshot cache, idempotency lock, rate limit, alert dedup, lockout).
- **Kafka Connect REST** — gọi GET/POST status connector (qua `connector_client.go`).
- **cdc-auth-service** — JWT verify (qua middleware), KHÔNG call HTTP.

### 2.5 Chức năng cốt lõi

> **Control plane CRUD** cho operator quản lý CDC pipeline: register source object, define mapping, manage master binding, monitor reconciliation drift, dispatch worker action, audit admin trace. KHÔNG xử lý CDC data flow — đó là worker (`centralized-data-service`).

---

## 3. 8 trụ refactor (P0 → P7)

| Pillar | Title | Risk | Effort | Verify |
|---|---|---|---|---|
| **P0** | Dead-code prune (`ReconciliationService` no-op + audit `cdc_event.go`) | LOW | 2h | grep=0, build PASS, 8 endpoint smoke |
| **P1** | V2 repository layer (6 repo: MappingRule/SourceObject/Master/Connection/Wizard/Alert/AdminAction) | MEDIUM | 1.5d | sqlmock test, raw SQL khỏi handler |
| **P2** | Service layer extraction (split 4 godfile → service/) | HIGH | 3-4d | handler ≤200 line, smoke endpoint |
| **P3** | ActivityLog helper dedup (8+ call site) | LOW | 4h | `grep "model.ActivityLog{" internal/api/`=0 |
| **P4** | V1↔V2 surface dedup (V2 không gọi V1 RegistryHandler) | MEDIUM | 1d | `grep "h.registry\." source_object_actions_handler.go`=0 |
| **P5** | Health collector probe split (`health/probes/*.go`) | MEDIUM | 1d | wc ≤300, errgroup parallel |
| **P6** | V2 sync atomicity (`SyncFromLegacy` → `SyncFromLegacyTx(tx)`) | HIGH | 1.5d | mock fail → V1 row rollback |
| **P7** | Test coverage uplift ≥35% (service + repo) | MEDIUM | 2d | `go test -cover` ≥0.35 |

**Sequential**: ~12-14 work day. **With parallel** (P5+P6 ‖ P2): ~10-12 work day. **Realistic**: 2-3 tuần với 1 engineer dedicated.

---

## 4. Nguyên tắc thi công (must hold)

1. **No API contract change** — path/request/response giữ nguyên (FE cdc-cms-web 22 .tsx 7634 LOC đang dùng).
2. **No NATS subject change** — 18 subject hiện tại (worker đã subscribe).
3. **No DB schema change** — schema settled.
4. **Per-pillar commit** — mỗi pillar = 1 PR/commit standalone, có thể revert riêng. KHÔNG mix pillar.
5. **Per-pillar gate**: build PASS + unit test PASS + ≥1 endpoint smoke PASS + `/security-agent` PASS + APPEND `05_progress.md` trước commit.
6. **APPEND-only memory** (CLAUDE.md §11) — `05_progress.md` không edit cũ.
7. **Real verification** (CLAUDE.md §3 + Lesson #1264) — exercise endpoint thực tế (`curl` token thật), không chỉ build pass.
8. **Brain-Code Prohibition** (CLAUDE.md §12) — Brain plan, Muscle code. Plan này = Brain output, chờ approve.

---

## 5. Lessons đã đọc trước plan (CLAUDE.md §7)

| Lesson | Tóm tắt áp dụng |
|---|---|
| #160 | Simplicity First — KHÔNG over-engineer base stable code, chỉ minimal-impact change. |
| #258 | Handler không trực tiếp cross-domain → cần repo abstraction. |
| #475 | Patch handler phải gán mọi field. |
| #1240 | Schema rename ↔ search_path → qualify SQL hoặc set search_path khi move schema. |
| #1253 | GORM Raw().Scan no nested struct → flat scan + manual transpose. |
| #1264 | PASS exercise-driven, không health-driven; bao phủ TẤT CẢ flow (operator + auto + cli). |
| #1277 | Tuân thủ rule user literal — không lý sự exception. |
| #1292 | Fire-and-forget cmd cần companion event — 3-actor pattern (publisher + handler + monitor). |
| Recent (2026-05-05) | Validation BEFORE fallback — config pipeline order matters. |

---

## 6. Service health snapshot (verified bằng curl/lsof thực tế)

### Pre-planning (ngay trước khi viết 4 file plan)
| Service | Port | Endpoint | Result |
|---|---|---|---|
| cms-service | 8083 | `/health` | 200 ✅ |
| cms-service | 8083 | `/ready` | 200 ✅ |
| auth-service | 8081 | `/health` | 200 ✅ |
| worker (centralized-data-service) | 8082 | TCP listen | PID 23565 alive ✅ |
| admin-api | 8084 | TCP | DOWN (pre-existing, không thuộc Phase 2) |

### Post-planning (sau khi viết 4 file plan + report này)
| Service | Port | Endpoint | Result |
|---|---|---|---|
| cms-service | 8083 | `/health` | 200 ✅ — không bị disturbed |
| auth-service | 8081 | `/health` | 200 ✅ — không bị disturbed |
| worker | 8082 | TCP listen | PID 23565 vẫn alive ✅ |

→ **Không có regression** từ thao tác planning (chỉ ghi memory, không touch source).

---

## 7. Files đã tạo / cập nhật

| Path | Action | Size |
|---|---|---|
| `01_requirements_phase2_cms_refactor.md` | NEW | 5,788 bytes |
| `02_plan_phase2_cms_refactor.md` | NEW | 15,620 bytes |
| `08_tasks_phase2_cms_refactor.md` | NEW | 7,879 bytes |
| `09_tasks_solution_phase2_cms_refactor.md` | NEW | 18,597 bytes |
| `05_progress.md` | APPEND (1 row Phase 2 kickoff) | +1 row |
| `report_phase2_cms_refactor_plan_2026-05-05.md` | NEW (file này) | — |

**Source code**: KHÔNG touch file `.go`/`.sql`/`.ts`/`.js` nào — đây thuần plan output.

---

## 8. Đề xuất bước kế tiếp (chờ user approval)

**Khuyến nghị**: bắt đầu với **P0 (T1)** — Dead-code prune.

Lý do:
- Effort thấp nhất (2h) — risk thấp nhất.
- Blocks T2-T17 → unblock toàn bộ pipeline planning.
- DoD rõ ràng: `internal/service/reconciliation_service.go` xóa, `server.go` field+alloc+goroutine remove, `cdc_event.go` audit, build PASS, 8 endpoint smoke PASS.

**Trình tự đề xuất**:
1. P0 (T1) — dead code → 2h
2. P1.0 (T2) — repo skeleton + MappingRuleV2Repo demo → 1d (validate pattern)
3. P1.1-P1.6 (T3-T8) — fan-out 6 repo → 2-3d (parallel-able)
4. P2.1-P2.4 (T9-T12) — split 4 godfile → 3-4d
5. Parallel: P5 (T15) probe split + P6 (T16) atomicity → 2d
6. P3 (T13) ActivityLog helper, P4 (T14) V1↔V2 dedup → 1.5d
7. P7 (T17) test uplift → 2d

**Cần user quyết định**:
- ✅ Approve plan và bắt đầu P0 ngay?
- ⏸ Hoặc xem lại 4 file plan, comment, rồi mới approve?
- 🔄 Hoặc thay đổi thứ tự pillar?

---

## 9. Nơi tham chiếu cho user check

```bash
# 4 file plan chính (xem theo thứ tự):
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/01_requirements_phase2_cms_refactor.md
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/02_plan_phase2_cms_refactor.md
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/08_tasks_phase2_cms_refactor.md
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/09_tasks_solution_phase2_cms_refactor.md

# Audit log (APPEND only):
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md

# Report này:
cat /Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-system-refactor/report_phase2_cms_refactor_plan_2026-05-05.md
```

---

## 10. Compliance checklist (CLAUDE.md §14 — Pre-flight)

- [x] §0 Trả lời tiếng Việt — ✅ (file này + chat)
- [x] §1 Brain plan / Muscle execute — Plan ghi vào workspace doc, code chưa touch.
- [x] §3 Verify trước khi báo done — service health đã verify trước+sau, không regression.
- [x] §7 Đọc lessons trước — 9 lesson đã reference trong `01_requirements`.
- [x] §7 Workspace governance — file đủ prefix (01, 02, 08, 09, 05_progress, report_).
- [x] §11 APPEND-only memory — `05_progress.md` chỉ thêm row mới, không edit row cũ.
- [x] §12 Brain Code Prohibition — không touch `.go`/`.sql`/`.ts`/`.js`.
- [x] User requirement: report_*.md để user review — ✅ đây.
- [x] User requirement: kiểm tra service trước khi báo done — ✅ pre+post snapshot.

**Status**: PLAN READY — awaiting user approval to proceed with P0.
