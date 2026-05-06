# Báo cáo — Phase 2 v2 — CDC-CMS-SERVICE Decoupling + CQRS Refactor — Plan đã sẵn sàng

> **Ngày**: 2026-05-05 17:00 ICT
> **Operator**: Muscle (Claude Code, claude-opus-4-7)
> **Phạm vi**: Plan only — KHÔNG touch source code. Awaiting user approval.
> **Workspace**: `agent/memory/workspaces/feature-cdc-system-refactor/`
> **Supersedes**: `report_phase2_cms_refactor_plan_2026-05-05.md` (v1 — layer-only refactor)

---

## 0. Tóm tắt 1 dòng

Phase 2 v1 chỉ tách layer trong CMS — chưa đủ "refactor". User mandate kiến trúc mới: **Separation of Powers** giữa CMS (Brain — Control Plane) và Worker (Muscle — Data Plane), kết hợp **CQRS / Clean Architecture nội bộ** trong CMS với 4 Pillar P1→P2→P3→P4.

---

## 1. User mandate đã ghi nhận

### Decoupling boundary (kiến trúc tổng thể)

| Side | Trách nhiệm | KHÔNG được làm |
|---|---|---|
| **CMS — Control Plane** | Metadata CRUD + RBAC + Validate + Dispatch NATS + Read job status | ALTER DDL, cross-DB scan, drift compute, backfill row, sync data |
| **Worker — Data Plane** | Subscribe NATS cmd + Execute heavy + UPDATE state về `cdc_dw` + Publish `cdc.evt.X.completed` | Expose HTTP cho FE, validate user RBAC |

### 4 Pillar (chốt thứ tự, không hoán đổi)

| Pillar | Title | Effort | Risk |
|---|---|---|---|
| **P1** | Khởi tạo `domain/` + `app/` + interfaces (Repository, CommandBus, QueryBus, Publisher) | 2d | LOW |
| **P2** | Di chuyển logic Handler → `app/queries/` (Read paths trước, low risk) | 3d | LOW |
| **P3** | Chuyển Action Handler → `app/commands/` + NATS Command Dispatcher + move 2 INLINE | 5d | HIGH |
| **P4** | Chuẩn hóa `infra/persistence/` — xóa raw SQL khỏi tầng trên | 3d | MEDIUM |
| **Total** | | **13d** | |

---

## 2. Audit thực tế (2 Explore agent, KHÔNG đoán)

### 2.1 CMS hiện trạng

| Endpoint group | Đã dispatch NATS đúng | Còn INLINE | Companion evt |
|---|---|---|---|
| Reconciliation triggers (recon-check, heal, retry, debezium-signal/snapshot, backfill-source-ts) | 6/6 ✅ | — | ❌ thiếu 6 evt |
| Source object actions (standardize, scan-fields, detect-timestamp, create-default-columns) | 4/4 ✅ | — | ❌ thiếu 4 evt |
| Mapping rule (backfill, alter-column, batch-update) | 3/3 ✅ | — | ❌ thiếu 3 evt |
| Master (create, approve) | 2/2 ✅ | — | ⚠️ 1 phần qua provisioning step_completed |
| Master Swap | — | ❌ inline ALTER RENAME (master_registry_handler.go:630) | — |
| V2 SyncFromLegacy | — | ❌ inline UPSERT (registry_handler.go:148/268/316) | — |
| Transmute | 1/1 ✅ | — | ✅ `cdc.evt.transmute.completed` |
| **TỔNG** | **12/14** | **2 inline** | **12 thiếu, 2 đầy đủ** |

### 2.2 Worker đã có infrastructure

- 30+ subject subscriber (xem `worker_server.go` line 210-526)
- `command_handler.go` hub
- `event_bridge.go` poller pattern
- `service/job_monitor.go` close-loop pattern (đã làm cho `cdc.evt.transmute.completed`)
- DB connection `ControlPlane.DSN` → `cdc_dw` (config.go line 32-39)
- ĐÃ ghi 8+ table trong `cdc_system.*`: transmute_schedule, source_object_registry, shadow_binding, mapping_rule_v2, recon_runs, worker_registry, ...

### 2.3 NATS subjects (18 hiện tại)

17 cmd subject CMS publish + 2 evt subject worker publish back. Subject mới Phase 2 v2 sẽ thêm:
- `cdc.cmd.master-swap` (cmd) + `cdc.evt.master-swap.completed` (evt)
- `cdc.cmd.v2-sync` (cmd) + `cdc.evt.v2-sync.completed` (evt)
- 12 evt companion cho cmd existing — qua wildcard pattern `cdc.evt.*.completed`

### 2.4 State tracking phân mảnh (cần thống nhất)

5 column hiện có:
- `master_binding.schema_status`
- `transmute_schedule.last_status`
- `recon_runs.status`
- `failed_sync_logs.status`
- `source_object_registry.provisioning_state`

→ Quyết định: tạo MỚI `cdc_system.cdc_jobs` (job tracking generic) **bổ sung** chứ không thay thế per-domain column. Per-domain column = domain state machine; `cdc_jobs` = async execution tracking. Orthogonal concern.

---

## 3. Kiến trúc đích (CMS sau Phase 2 v2)

```
cdc-cms-service/
├── cmd/server/main.go               # bootstrap
├── internal/
│   ├── domain/                      # P1 — pure business
│   │   ├── mapping/ source/ master/ reconciliation/ job/
│   ├── app/
│   │   ├── ports/                   # P1 — interfaces (repo, command bus, query bus, publisher)
│   │   ├── queries/                 # P2 — read use case
│   │   └── commands/                # P3 — write use case
│   ├── infra/
│   │   ├── persistence/             # P4 — raw SQL CHỈ Ở ĐÂY
│   │   ├── messaging/               # NATS publisher + command bus impl
│   │   ├── http/                    # Kafka Connect REST client
│   │   └── cache/                   # Redis
│   └── api/                         # Fiber adapter ≤100 dòng/file
└── pkgs/                            # cross-cutting (logger, validator, pg_ident)
```

### Mô hình tương tác mới (Event-Driven Async)

```
[FE/Admin] ──HTTPS──▶ [CMS API]
                        │
                        ├─ Validate + RBAC
                        ├─ commandBus.Dispatch(cmd)
                        │     ├─ INSERT cdc_system.cdc_jobs (status='pending', job_id=uuid)
                        │     └─ NATS publish cdc.cmd.X (payload + job_id)
                        └─ Return 202 + {"job_id": "..."}

[NATS] ──cdc.cmd.X──▶ [Worker]
                        ├─ HandleX(msg) — execute heavy
                        └─ NATS publish cdc.evt.X.completed (job_id, status, result, error)

[NATS wildcard cdc.evt.*.completed] ──▶ [JobMonitor]
                                          └─ UPDATE cdc_system.cdc_jobs
                                             SET status, result, finished_at
                                             WHERE id=job_id AND status IN ('pending','running')

[CMS API] GET /api/jobs/:id ──▶ [JobRepo.GetByID] ──▶ JSON status response
```

---

## 4. Files đã tạo / cập nhật

### File mới (Phase 2 v2 — decoupling)

| Path | Action | Size |
|---|---|---|
| `01_requirements_phase2_decoupling.md` | NEW | 12,781 bytes |
| `02_plan_phase2_decoupling.md` | NEW | 23,813 bytes |
| `08_tasks_phase2_decoupling.md` | NEW | 12,656 bytes |
| `09_tasks_solution_phase2_decoupling.md` | NEW | 20,415 bytes |
| `report_phase2_cms_decoupling_2026-05-05.md` | NEW (file này) | — |
| `05_progress.md` | APPEND 2 row (không edit row cũ) | 4,945 bytes |

### File cũ (Phase 2 v1) — superseded nhưng GIỮ NGUYÊN

| Path | Status |
|---|---|
| `01_requirements_phase2_cms_refactor.md` | superseded (giữ làm lịch sử) |
| `02_plan_phase2_cms_refactor.md` | superseded |
| `08_tasks_phase2_cms_refactor.md` | superseded |
| `09_tasks_solution_phase2_cms_refactor.md` | superseded |
| `report_phase2_cms_refactor_plan_2026-05-05.md` | superseded |

→ CLAUDE.md §11 APPEND-only memory: KHÔNG xóa file v1, chỉ thêm v2.

### Source code

KHÔNG touch file `.go`/`.sql`/`.ts`/`.js` nào. Đây thuần plan output.

---

## 5. Service health snapshot (curl/lsof thực tế)

### Pre-write (trước khi viết 4 file plan v2)
| Service | Port | Endpoint | Result |
|---|---|---|---|
| cms-service | 8083 | `/health` | 200 ✅ |
| cms-service | 8083 | `/ready` | 200 ✅ |
| auth-service | 8081 | `/health` | 200 ✅ |
| worker | 8082 | TCP listen | PID 23565 alive ✅ |

### Post-write (sau khi viết 4 file plan v2 + report này)
| Service | Port | Endpoint | Result |
|---|---|---|---|
| cms-service | 8083 | `/health` | 200 ✅ — không bị disturbed |
| cms-service | 8083 | `/ready` | 200 ✅ |
| auth-service | 8081 | `/health` | 200 ✅ |
| worker | 8082 | TCP listen | PID 23565 vẫn alive ✅ |

→ **Không có regression** từ thao tác planning (chỉ ghi memory, không touch source).

---

## 6. Lessons đã re-read trước plan v2

| Lesson | Áp dụng |
|---|---|
| #160 Simplicity First | Reuse JobMonitor pattern hiện có; KHÔNG xây Saga framework. |
| #258 No Cross-Domain CQRS Handler | Query handler ≠ command handler; tách commands/ vs queries/. |
| #475 Forgotten Field Assignment | Patch command phải gán mọi field. |
| #1240 Schema rename ↔ search_path | Qualify `cdc_system.tab` trong mọi SQL infra/persistence. |
| #1253 GORM Raw().Scan no nested | Domain entity flat, repo dùng row struct → toDomain mapping. |
| #1264 PASS exercise-driven | 8 endpoint smoke + A1-A3 action smoke (POST → 202 → GET status). |
| #1277 User rule literal | 4 pillar order P1→P2→P3→P4 không hoán đổi. |
| **#1292 Fire-and-forget cmd cần companion event** | **CORE pattern**: 12/14 cmd hiện thiếu evt → P3 bổ sung wildcard `cdc.evt.*.completed`. |
| **#1399 Event-Driven Cascade Liability** | JobMonitor KHÔNG fire cmd khác trong handler completion. |
| 2026-04-29 line 828-852 NATS async + service boundary | Precedent 12 ADR-015 violation; v2 đi tiếp đường lối. |
| 2026-05-05 line 1817 Cross-repo decoupling | CMS không import worker code; chỉ qua NATS subject contract. |

---

## 7. Decision log (chính)

### Q1: `cdc_jobs` thống nhất hay reuse per-domain status column?
→ **Tạo MỚI `cdc_system.cdc_jobs`**. Per-domain column giữ nguyên (domain state machine). Job table = async execution tracking. Orthogonal.

### Q2: Move INLINE Master Swap — TX cross-DB cần không?
→ **KHÔNG**. Worker chạy ALTER RENAME 1 DB. CMS ghi `cdc_jobs.status='pending'` trước publish. Fail → JobMonitor flag, retry qua POST mới.

### Q3: Pillar P2 (Read) trước P3 (Write) — bắt buộc sequential?
→ **Sequential**. User mandate explicit. Read trước → ít rủi ro state.

### Q4: 12 evt subject — subscribe riêng hay wildcard?
→ **Wildcard `cdc.evt.*.completed`**. Single handler `HandleCompleted` route theo `msg.Subject`.

### Q5: `internal/service/` tồn tại sau P4?
→ **KHÔNG**. Migrate hết: use case → app/commands/ hoặc app/queries/; DB query → infra/persistence/; external call → infra/http/ hoặc infra/messaging/.

### Q6: Phase 2 v1 có execute không?
→ **KHÔNG**. v1 superseded. File giữ nguyên làm lịch sử (CLAUDE.md §11). v1 P5 (probe split) integrate vào v2 P4; v1 P6 (V2 sync atomicity) giải quyết bởi v2 P3.

---

## 8. Đề xuất bước kế tiếp (chờ user approval)

### Khuyến nghị: Bắt đầu **P1 (T1.1-T1.5)** — Setup interface skeleton

Lý do:
- Risk thấp nhất (LOW), không thay đổi behavior endpoint nào.
- Effort 2d.
- Là foundation cho P2-P4.
- Build PASS gate đảm bảo không phá hệ thống.

### Trình tự đề xuất:
1. **P1 (T1.1-T1.5)**: 2d — interface skeleton
2. **P2 (T2.1-T2.8)**: 3d — Read migration (15 endpoint)
3. **P3 (T3.1-T3.11)**: 5d — Write + Bus + 2 INLINE move + worker change + JobMonitor extend
4. **P4 (T4.1-T4.9)**: 3d — Persistence cleanup + delete service/

### User cần quyết định:

- ✅ Approve plan v2 + bắt đầu P1 ngay?
- ⏸ Hoặc xem lại 4 file plan + report, comment, rồi mới approve?
- 🔄 Hoặc cần điều chỉnh thêm pillar / boundary?
- ⚠️ Coordinate migration `cdc_system.cdc_jobs` với DB ops trước P3 (T3.1).

---

## 9. Nơi tham chiếu cho user check

```bash
# 4 file plan v2 (xem theo thứ tự):
cat agent/memory/workspaces/feature-cdc-system-refactor/01_requirements_phase2_decoupling.md
cat agent/memory/workspaces/feature-cdc-system-refactor/02_plan_phase2_decoupling.md
cat agent/memory/workspaces/feature-cdc-system-refactor/08_tasks_phase2_decoupling.md
cat agent/memory/workspaces/feature-cdc-system-refactor/09_tasks_solution_phase2_decoupling.md

# Audit log (APPEND only — đã thêm 2 row Phase 2 v1 + v2 superseded):
cat agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md

# Report v2 (file này):
cat agent/memory/workspaces/feature-cdc-system-refactor/report_phase2_cms_decoupling_2026-05-05.md

# Report v1 (superseded — vẫn giữ):
cat agent/memory/workspaces/feature-cdc-system-refactor/report_phase2_cms_refactor_plan_2026-05-05.md
```

---

## 10. Compliance checklist (CLAUDE.md §14 — Pre-flight)

- [x] §0 Trả lời tiếng Việt — ✅ (file này + chat)
- [x] §1 Brain plan / Muscle execute — Plan ghi vào workspace doc, code chưa touch.
- [x] §3 Verify trước khi báo done — service health verify trước+sau, không regression.
- [x] §3 Plan & Verify Deep Execution — 4 pillar ≥3 step, mỗi pillar có DoD verify command.
- [x] §7 Đọc lessons trước — 11 lesson re-read (đã liệt kê §6).
- [x] §7 Workspace governance — file đủ prefix (01, 02, 08, 09, 05_progress, report_).
- [x] §7 Full Doc Set — 4 file phase2_decoupling đầy đủ + report.
- [x] §11 APPEND-only memory — `05_progress.md` chỉ thêm 2 row (Phase 2 v1 + v2 supersede note), không edit row cũ.
- [x] §12 Brain Code Prohibition — KHÔNG touch `.go`/`.sql`/`.ts`/`.js`.
- [x] §13 Lesson Writing Standard — sẽ ghi lesson "Plan boundary first, structure second" sau khi user approve + execute (tránh lesson rỗng).
- [x] User requirement: report_*.md để user review — ✅ đây.
- [x] User requirement: kiểm tra service trước khi báo done — ✅ pre+post snapshot.
- [x] User requirement: không report láo — ✅ data từ 2 Explore agent + curl/lsof thực tế, line:file cụ thể.

**Status**: PLAN v2 READY — awaiting user approval to proceed with P1 (T1.1).
