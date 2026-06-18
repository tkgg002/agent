# 05_progress.md — Audit log (APPEND-ONLY)

> CLAUDE.md §11: TUYỆT ĐỐI cấm overwrite. Chỉ APPEND.

## 2026-05-19 — Workspace init + plan refactor cấu trúc cdc-cms-service

### Bước 0 — User request

- **User input**: "đang rất khó chịu về patten & cấu trúc của cdc-cms-service. ko trực quan để dev debug, theo dõi và quản lý tiến trình. nó chả có con mẹ gì. quản lý mấy cái table, thêm xoa sửa xong gọi triger, có thêm mấy cái connect để show, health service. mày lên 1 plan để sắp xếp cấu trúc. ko làm 1 dòng code nào, task lớn cẩn review."

### Bước 1 — Đọc bắt buộc (theo user note + CLAUDE.md §7)

| File | Status |
|------|--------|
| `agent/GEMINI.md` | ✅ Đọc full (14 rule + workflow list) |
| `agent/memory/global/project_context.md` | ✅ Đọc full — profile cms 76 .go, PID 13653 (cms-server local) |
| `agent/memory/global/tech_stack.md` | ✅ Đọc full — Fiber v2, CQRS light, Go 1.26.1 |
| `agent/memory/global/active_plans.md` | ✅ Đọc full — phát hiện Task #19 closed 2026-05-07 by x2 với hexagonal đã tồn tại |
| `agent/memory/global/lessons.md` | 🟡 File 325KB > giới hạn 256KB Read tool. Dùng grep với keywords (AutoMigrate, mapping, cms-service, refactor) → 30+ lesson liên quan |

### Bước 2 — Explore agent map cấu trúc cdc-cms-service

- Spawn `Explore` agent với thoroughness="very thorough".
- Output: 6 nhóm finding với file:line evidence.
- Bash verify thêm: confirm số LOC (provisioning_orchestrator 729, alert_manager 446, …) + đếm import violation (17 file api/, 4 file commands/, 7 file api/ → natsconn).

### Bước 3 — Tạo workspace + full doc set

**Workspace**: `agent/memory/workspaces/feature-cdc-cms-service-restructure-2026-05-19/`

Files đã tạo (Brain scope, KHÔNG sửa code §12):

| # | File | LOC (gần đúng) | Vai trò |
|---|------|----------------|---------|
| 1 | `00_context.md` | ~85 | Bối cảnh + scope + non-goals + 4 ràng buộc user |
| 2 | `10_gap_analysis.md` | ~230 | 14 mismatch / pain point + 4 bảng + root cause |
| 3 | `01_requirements.md` | ~115 | 10 FR + 10 NFR + 10 AC + DoD + risk register |
| 4 | `09_tasks_solution.md` | ~195 | 4 hướng đi + so sánh + khuyến nghị (chọn Hướng 3) + 6 câu hỏi pending |
| 5 | `02_plan.md` | ~325 | Cấu trúc đích + 11 phase rollout + Gantt + rollback strategy |
| 6 | `03_implementation.md` | ~580 | Code demo Go đầy đủ cho module `mapping/` (8 file canonical) |
| 7 | `04_decisions.md` | ~210 | 8 ADR (Vertical Slice, no anemic, no model/, ports inline, router thin, eventbus, pure structural, phase gate) |
| 8 | `08_tasks.md` | ~270 | Task break-down 11 phase × ~80 task, risk matrix |
| 9 | `05_progress.md` | (file này) | Audit log |
| 10 | `07_status.md` | (sắp tạo) | Trạng thái + checkpoint |
| 11 | `report_*.md` | (sắp tạo) | File report tổng theo user request |

### Bước 4 — Phát hiện chính

#### 4.1 Root cause: hexagonal đúng layer, sai granularity

- 4 layer (api / app / domain / infra) flat → 52/32/24/27 file mặt phẳng.
- Mental model user A/B/C/D không có chỗ đứng trong cây thư mục.
- Dev phải nhảy 4 folder × ~7 file để hiểu 1 feature.

#### 4.2 Số liệu critical

- File > 500 LOC: 4 file (`provisioning_orchestrator` 729, `source_object_actions` 550, `source_object_v2_sync` 495, `alert_manager` 446).
- Package > 25 file: 3 (`api/` 52, `commands/` 32, `persistence/` 27).
- File sai chỗ: 6 (provisioning_*, approval, alert_manager, alerts_handler nhận concrete type, provisioning_handler nhận concrete type).
- Import violation: 17 + 4 + 7 = **28 file vi phạm DIP/cross-layer rule**.
- Domain anemic: 7/7 entity (chỉ `job.New()` factory).

#### 4.3 Naming chồng chéo

- `internal/model/` (11 file GORM) + `internal/domain/` (7 file clean) → cùng concept "source" tồn tại ở 3 nơi.

### Bước 5 — Solution chốt

⭐ **Hướng 3 — Vertical Slice / Modular Monolith** (xem `09_tasks_solution.md`).

Cấu trúc đích:
```
internal/
├── modules/         (9-10 module: mapping, registry, provisioning, alerts, reconciliation, connectors, sources, health, jobs, schedules)
├── platform/        (bus, db, http, observability, middleware, eventbus)
├── server/          (wire.go DI)
└── router/          (≤ 50 LOC)
```

11 phase × review gate sau mỗi phase. Tổng ~9 ngày Muscle execution.

### Bước 6 — Pending (user approve)

1. OK với Hướng 3 (Vertical Slice)?
2. `internal/model/` GORM: di chuyển vào module hay giữ `platform/db/gormmodel/` shared?
3. `platform/eventbus/`: tạo ngay P0 hay defer?
4. Naming `internal/modules/` hay `internal/feature/` hay `internal/bc/`?
5. Migration giữ `migrations/` root hay vào `platform/db/migrations/`?
6. Review gate: pause-then-resume sau mỗi phase, hay continuous với 3 checkpoint?

→ Sau khi user trả lời 6 câu này, Muscle có đủ context thực thi.

## Skills sử dụng (Brain — round 1 round 2)

| Skill | Lần dùng | Mục đích |
|-------|----------|----------|
| **Read** | 6 | `GEMINI.md` + 3 file global memory + `rule.go` + `mapping_rule_dto.go` + `mapping_rule_handler_create.go` |
| **Bash** | 5 | `ls workspace` + `wc -l` verify + `grep import` verify + `ls /tmp/cms-server` health check |
| **Agent Explore** | 1 | Map structure cdc-cms-service với thoroughness="very thorough" — output ground truth report |
| **Write** | 11 | 11 doc file trong workspace |
| **TaskCreate / TaskUpdate** | 12 / nhiều | Tracking 12 task |
| **Workspace governance §7** | - | Full Doc Set tuân thủ prefix 00-10 |
| **Brain Code Prohibition §12** | - | 0 dòng code Go chạm vào `cdc-cms-service/` |
| **Plan & Verify §3** | - | 4 hướng → chọn 1 + lý do + alternative |
| **Bug Fixing Tự chủ §2** | - | KHÔNG hỏi ngược user trong gap analysis — Brain tự research |
| **Simplicity First §6** | - | Pure structural — KHÔNG refactor logic / SQL / NATS |
| **Pre-flight check §14** | - | Verify 11 file vật lý trước khi báo done |

## 2026-05-19 (cont) — Verify service work + đính chính project_context.md

- **User request reminder**: "Khi kết thúc luôn kiểm tra các service work mới báo done".
- **Verify đã chạy thực tế**:
  - `lsof -iTCP -sTCP:LISTEN` → cms-server PID 8675 listening `*:8083`.
  - `ps -p 8675` → binary `~/Library/Caches/go-build/38/386dace0.../main`, etime ~48 phút.
  - `curl localhost:8083/health` → `{"service":"cdc-cms","status":"ok"}` ✅
  - `curl localhost:8083/ready` → `{"status":"ready"}` ✅
  - `curl localhost:8083/api/system/health` → `{"overall":"healthy", ...}` ✅
- **Đính chính**: `project_context.md` ghi binary `/tmp/cdc-cms-service-postJ` PID 52079 — **outdated**. Thực tế:
  - PID hiện tại: **8675** (rebuild ~48 phút trước).
  - Port: **8083** (per `config-local.yml`, không phải 8081 — 8081 là cdc-auth).
  - Binary: go-build cache, không phải `/tmp/...`.
- **Brain commitment**: KHÔNG sửa `project_context.md` ngay (append-only §11 + chờ user approve plan trước). Lesson update sẽ ghép vào round Muscle P10.
- **Source code**: 0 file `.go` bị Brain đụng trong round này. Toàn bộ ~127 KB doc nằm trong workspace.

## Skills sử dụng (round verify)

- **Bash** `lsof` + `curl` + `ps` — verify service runtime.
- **Edit** (`report_*.md`, `05_progress.md`) — đính chính + append truth.
- **No-Lie principle** — user yêu cầu "không report láo". Báo cáo đúng port thực tế (8083) thay vì 8081 trong project_context outdated.

## 2026-06-16 — Current-turn reconcile + dirty tree note

- **User request**: "làm đi em" sau khi review plan.
- **Reconcile thực tế**:
  - Repo `cdc-cms-service` đang có sẵn 1 file modified trước đó: `internal/api/master_mapping_rule_handler.go`.
  - Turn này **không** chạm vào source code; không `Edit/Write` bất kỳ file `.go` nào trong repo.
  - Workspace plan/report hiện tại đã đủ để user review, không cần bung code trước khi có approve.
- **Why this note exists**: để tránh report sai rằng working tree hoàn toàn sạch, trong khi git status thực tế đang dirty từ trước.

## 2026-06-16 — Implement `master_mapping_rule` slice

- **User request**: refactor `internal/api/master_mapping_rule_handler.go` trước, rồi mới duyệt.
- **What changed**
  - `internal/api/master_mapping_rule_handler.go` được bóc về thin HTTP adapter.
  - `internal/app/commands/master_mapping_rule.go` mới, giữ toàn bộ orchestration DB/NATS cho master mapping rules.
  - `internal/server/server.go` đổi wiring sang service mới.
- **Behavior preserved**
  - Route contracts và response shape chính vẫn giữ nguyên.
  - Mapping-rule validation ở HTTP layer vẫn chạy.
  - NATS publish/worker-trigger logic vẫn hoạt động, nhưng nằm ở app layer thay vì handler.
- **Verify thực tế**
  - `gofmt -w ...` chạy sạch.
  - `GOCACHE=/private/tmp/cdc-gocache go test ./internal/api ./internal/app/commands ./internal/server` ✅ PASS.
  - `GOCACHE=/private/tmp/cdc-gocache go test ./...` ✅ phần lớn PASS, chỉ fail ở 2 test `httptest`/listen port do sandbox chặn bind localhost:
    - `test/internal/infra/http.TestPercentileFallbackScrape`
    - `test/internal/infra/observability/probes.TestDebezium_RunningExtractsState`
  - Runtime service verify:
    - `lsof -nP -iTCP -sTCP:LISTEN` thấy PID `75385` listen `*:8083`.
    - `/health` → `{"service":"cdc-cms","status":"ok"}`
    - `/ready` → `{"status":"ready"}`
    - `/api/system/health` vẫn trả snapshot runtime, `overall":"critical"` do infra warnings hiện hữu chứ không phải do turn này.
- **Diff reality**
  - `internal/api/master_mapping_rule_handler.go`: `141` insertions, `477` deletions so với HEAD.
  - `internal/app/commands/master_mapping_rule.go`: file mới `580 LOC`.
- `internal/server/server.go`: `2` insertions, `1` deletion.

## 2026-06-16 — Verify after refactor

- **Compile verify**
  - `GOCACHE=/private/tmp/cdc-gocache go test ./internal/api ./internal/app/commands ./internal/server` ✅ PASS
  - `GOCACHE=/private/tmp/cdc-gocache go test ./...` ✅ phần lớn PASS, fail ở 2 test `httptest`/port bind do sandbox:
    - `test/internal/infra/http.TestPercentileFallbackScrape`
    - `test/internal/infra/observability/probes.TestDebezium_RunningExtractsState`
- **Runtime verify**
  - `lsof -nP -iTCP -sTCP:LISTEN` thấy `*:8083` đang do PID `75385` listen.
  - `curl 127.0.0.1:8083/health` → `{"service":"cdc-cms","status":"ok"}`
  - `curl 127.0.0.1:8083/ready` → `{"status":"ready"}`
  - `curl 127.0.0.1:8083/api/system/health` → snapshot vẫn trả về, `overall":"critical"` do infra warnings có sẵn trước turn này.
- **Ghi chú report**
  - `internal/api/master_mapping_rule_handler.go` đã được kéo xuống thin adapter.
  - `internal/app/commands/master_mapping_rule.go` là slice app-layer mới.
  - `internal/server/server.go` đã đổi wiring sang service mới.

## 2026-06-16 — Corrected to screaming/module-first + repository pattern

- **User correction**: anh yêu cầu trả logic về repository pattern và áp dụng screaming architecture, không giữ app-service trung gian.
- **What changed in this turn**
  - Xóa `internal/api/master_mapping_rule_handler.go`.
  - Xóa `internal/app/commands/master_mapping_rule.go`.
  - Thêm module `internal/modules/mastermapping/{module.go,routes.go,handler.go}`.
  - Đẩy logic persistence liên quan `master_mapping_rule` vào `internal/infra/persistence/master_mapping_rule_repo_gorm.go`.
  - Rewire `internal/router/router.go` và `internal/server/server.go` sang module mới.
- **Verify thực tế**
  - `go test ./...` ✅ PASS toàn repo.
  - `curl 127.0.0.1:8083/health` ✅ `{"service":"cdc-cms","status":"ok"}`
  - `curl 127.0.0.1:8083/ready` ✅ `{"status":"ready"}`
- **Why this matters**
  - Không để lại 2 kiến trúc song song cho cùng 1 feature.
  - HTTP adapter còn mỏng hơn, business/persistence đi đúng nơi của nó.
  - Tài liệu report đã được sync lại để phản ánh đúng trạng thái hiện tại, không kể nhầm nhánh cũ.
