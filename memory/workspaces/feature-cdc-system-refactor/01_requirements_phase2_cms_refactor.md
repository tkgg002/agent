# Phase 2 — cdc-cms-service Refactor — Requirements

> **Date**: 2026-05-05
> **Owner**: Brain (planning) → Muscle (execution per pillar)
> **Workspace**: `feature-cdc-system-refactor` (Phase 1 = centralized-data-service ✅ Done)

## Bối cảnh

cdc-cms-service hiện là control-plane CRUD cho operator, đã trải qua 39 phase trong workspace `feature-cms-fe-overhaul` (chủ yếu phase 7, 21-39 chạm BE: API surface pruning + V2 schema migrate + dual-write V1↔V2 + schema consolidation). Sau những phase đó, code BASE STABLE nhưng tích tụ bloat:

- 4 godfile handler 600-900 dòng (raw SQL + business logic + activity log + NATS publish chen chúc).
- 0 repository abstraction cho schema `cdc_system.*` (V2 metadata) → unable-to-unit-test handler.
- ActivityLog write logic inline 8+ vị trí trong `registry_handler.go` (copy-paste).
- `ReconciliationService` đã retire (Airbyte→Debezium phase F) nhưng vẫn ngồi trong dependency graph.
- `system_health_collector.go` monolith 781 dòng (7 probe + 3 query + alert + serialize trộn 1 file).
- V2 sync (`SourceObjectV2SyncService.SyncFromLegacy`) silent-fail (log + swallow) → V1/V2 silent diverge.
- Test coverage tổng 13% (10/76 file). `internal/model`, `internal/repository`, `internal/router`, `internal/server`, `pkgs/*` = 0 test.

## Mục tiêu

Refactor monolithic-handler architecture sang clean layered (handler thin / service rich / repository abstracted) **without breaking any existing endpoint** và **without changing any external contract** (HTTP path, JSON shape, NATS subject, DB schema).

## Definition of Done

| # | Criterion | Verify |
|---|---|---|
| 1 | Mỗi godfile handler ≤200 dòng (extract business → service, query → repo, helper → shared util) | `wc -l internal/api/*.go` mọi file ≤200 |
| 2 | Mọi raw SQL truy cập `cdc_system.*` đều đi qua `internal/repository/v2_*.go` | grep `h.db.Raw` trong handler = 0 |
| 3 | ActivityLog ghi qua `service.ActivityLogger.Log(ctx, op)` — handler KHÔNG còn `model.ActivityLog{}` inline | grep `model.ActivityLog{` trong handler = 0 |
| 4 | `ReconciliationService` no-op + dependency removed khỏi `server.go` | `grep ReconciliationService server.go` = 0 |
| 5 | `system_health_collector.go` ≤300 dòng — 7 probe tách `internal/service/health/probes/*.go` | `wc -l` xác nhận |
| 6 | V2 handler KHÔNG call vào `RegistryHandler` methods (V1 facade) — both call shared service | `grep "h.registry\." source_object_actions_handler.go` = 0 |
| 7 | V2 sync atomic với V1 write (cùng `gorm.Tx`) hoặc outbox-table với worker drainer | code review TX boundary |
| 8 | Test coverage `internal/service/` + `internal/repository/` ≥ 35% | `go test -cover ./internal/service/... ./internal/repository/...` |
| 9 | Zero regression: 8 endpoint smoke PASS sau mỗi pillar | curl thực tế 8 path (xem §Verification) |
| 10 | Mỗi pillar chạy `/security-agent` trước commit | report log |

## Verification — 8 endpoint smoke (exercise-driven, không health-driven, theo Lesson #1264)

| # | Endpoint | Expect |
|---|---|---|
| 1 | `GET /health` | 200 `{"status":"ok"}` |
| 2 | `GET /api/system/health` (auth) | 200, JSON snapshot, `overall.status` field exists |
| 3 | `GET /api/sync/health` (auth) | 200, `total_registered ≥ 0` |
| 4 | `GET /api/v1/source-objects` (auth) | 200, `data: []` (length tùy state) |
| 5 | `GET /api/mapping-rules` (auth) | 200, mảng object có `id,target_table,source_field,status` |
| 6 | `GET /api/v1/system/connectors` (auth) | 200 hoặc 502 (Kafka Connect down OK miễn không 500) |
| 7 | `GET /api/reconciliation/report` (auth) | 200, mảng (có thể rỗng) |
| 8 | `GET /api/v1/masters` (auth) | 200, mảng |

## Constraints (must hold)

- **No API contract change**: path / request / response shape giữ nguyên — FE đang dùng (`cdc-cms-web` 22 .tsx, ~7634 LOC).
- **No NATS subject change**: 18 subject publish hiện tại giữ nguyên — worker đã subscribe (`centralized-data-service`).
- **No DB schema change**: schema đã settled qua phase 38/39 cms-fe-overhaul.
- **Per-pillar commit**: mỗi pillar = standalone PR/commit, có thể revert riêng. KHÔNG mix 2 pillar trong 1 commit.
- **Per-pillar gate**: build PASS + unit test PASS + ≥1 endpoint smoke PASS + `/security-agent` PASS trước khi commit.
- **APPEND-only memory** (CLAUDE.md §11): mọi update vào `05_progress.md` là append, không edit cũ.
- **Real verification** (CLAUDE.md §3 + Lesson #1264): exercise endpoint thực tế, không chỉ build pass.

## Out of scope (không làm Phase 2)

- FE changes (`cdc-cms-web`).
- Worker changes (`centralized-data-service`).
- Auth service changes (`cdc-auth-service`).
- DB migration mới (cdc_system schema settled).
- Deploy automation / k8s manifest.
- New features (chỉ refactor architecture).

## Reference lessons (đã đọc trước plan)

- **#160** Simplicity First — không over-engineer base stable code, chỉ minimal-impact change.
- **#258** No Cross-Domain Model in CQRS Handler — handler không trực tiếp cross-domain.
- **#475** Forgotten Field Assignment — patch handler phải gán mọi field.
- **#1240** Schema rename ↔ search_path — qualify SQL hoặc set search_path khi move schema.
- **#1253** GORM Raw().Scan no nested struct — flat scan struct + manual transpose.
- **#1264** PASS exercise-driven — không health-driven; bao phủ TẤT CẢ flow (operator + auto + cli).
- **#1277** Tuân thủ rule user literal — không lý sự exception.
- **#1292** Fire-and-forget cmd cần companion event — 3-actor pattern (publisher + handler + monitor).
- **Recent: Validation BEFORE fallback** — config pipeline order matters (đã ghi 2026-05-05).



===>>
Nếu để cdc-cms-service (API) vừa quản lý metadata vừa thực thi các tác vụ nặng (như reconciliation, backfill, sync) thì sẽ vi phạm nguyên tắc Separation of Powers và làm hệ thống rất khó scale.

Để đạt được sự chuyên nghiệp như bạn mong muốn, chúng ta cần xác định lại ranh giới đỏ (boundary) giữa hai bên. Dưới đây là kiến trúc tách rời (Decoupling) mà chúng ta sẽ hướng tới:

1. Phân định trách nhiệm (Core Responsibilities)
A. CDC-CMS-SERVICE (The Brain - Control Plane)
Con này chỉ tập trung vào tương tác với database cdc_dw (gpay-postgres-cdc) và quản lý trạng thái hệ thống.

Quản lý Metadata: CRUD cấu hình (Mapping rules, Source objects, Master bindings).

Trạng thái (State Management): Ghi nhận trạng thái của các worker, lịch trình (Schedules).

Cổng giao tiếp (Gateway): Tiếp nhận yêu cầu từ FE/Admin, validate nghiệp vụ, kiểm tra quyền (RBAC).

Ra lệnh (Command Dispatcher): Thay vì tự thực hiện action, nó sẽ publish message (qua NATS) để ra lệnh cho Worker.

B. CDC-WORKER (The Muscle - Data Plane)
Con này mới là đứa "tay lấm chân bùn" xử lý mọi thứ liên quan đến luồng dữ liệu và database đích.

Thực thi Action: Nhận lệnh từ API qua NATS (ví dụ: cdc.cmd.reconcile.start) và bắt đầu chạy.

Tương tác DB: Đọc dữ liệu từ Source (Mongo/Postgres), so khớp, và ghi vào Destination.

Báo cáo (Feedback Loop): Trong quá trình chạy, nó sẽ update trạng thái tiến độ (Progress) hoặc kết quả vào DB cdc_dw để API có dữ liệu hiển thị.

2. Mô hình tương tác (Inter-service Flow)
Thay vì gọi trực tiếp logic, chúng ta sẽ chuyển sang mô hình Event-Driven:

User nhấn nút "Start Reconcile" trên UI.

API nhận request -> Ghi một record vào bảng reconciliation_reports với status PENDING.

API bắn một message sang NATS: {"action": "START_RECON", "report_id": "uuid-123", "target": "user_table"}.

Worker đang subscribe NATS nhận được message -> Nhảy vào xử lý.

API trả về 202 Accepted cho User ngay lập tức.

Worker làm xong -> Update status SUCCESS vào DB cdc_dw.

Nếu để cdc-cms-service (API) vừa quản lý metadata vừa thực thi các tác vụ nặng (như reconciliation, backfill, sync) thì sẽ vi phạm nguyên tắc Separation of Powers và làm hệ thống rất khó scale.

Để đạt được sự chuyên nghiệp như bạn mong muốn, chúng ta cần xác định lại ranh giới đỏ (boundary) giữa hai bên. Dưới đây là kiến trúc tách rời (Decoupling) mà chúng ta sẽ hướng tới:

1. Phân định trách nhiệm (Core Responsibilities)
A. CDC-CMS-SERVICE (The Brain - Control Plane)
Con này chỉ tập trung vào tương tác với database cdc_dw (gpay-postgres-cdc) và quản lý trạng thái hệ thống.

Quản lý Metadata: CRUD cấu hình (Mapping rules, Source objects, Master bindings).

Trạng thái (State Management): Ghi nhận trạng thái của các worker, lịch trình (Schedules).

Cổng giao tiếp (Gateway): Tiếp nhận yêu cầu từ FE/Admin, validate nghiệp vụ, kiểm tra quyền (RBAC).

Ra lệnh (Command Dispatcher): Thay vì tự thực hiện action, nó sẽ publish message (qua NATS) để ra lệnh cho Worker.

B. CDC-WORKER (The Muscle - Data Plane)
Con này mới là đứa "tay lấm chân bùn" xử lý mọi thứ liên quan đến luồng dữ liệu và database đích.

Thực thi Action: Nhận lệnh từ API qua NATS (ví dụ: cdc.cmd.reconcile.start) và bắt đầu chạy.

Tương tác DB: Đọc dữ liệu từ Source (Mongo/Postgres), so khớp, và ghi vào Destination.

Báo cáo (Feedback Loop): Trong quá trình chạy, nó sẽ update trạng thái tiến độ (Progress) hoặc kết quả vào DB cdc_dw để API có dữ liệu hiển thị.

2. Mô hình tương tác (Inter-service Flow)
Thay vì gọi trực tiếp logic, chúng ta sẽ chuyển sang mô hình Event-Driven:

User nhấn nút "Start Reconcile" trên UI.

API nhận request -> Ghi một record vào bảng reconciliation_reports với status PENDING.

API bắn một message sang NATS: {"action": "START_RECON", "report_id": "uuid-123", "target": "user_table"}.

Worker đang subscribe NATS nhận được message -> Nhảy vào xử lý.

API trả về 202 Accepted cho User ngay lập tức.

Worker làm xong -> Update status SUCCESS vào DB cdc_dw.