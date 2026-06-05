# Report — Check Connection Feature (Phase `check_connection`)

> **Date**: 2026-05-26
> **Workspace**: `feature-connector-check-connection-2026-05-26`
> **Author (Brain plan)**: claude-opus-4-7 (Antigravity Brain)
> **Author (Muscle exec)**: `<filled-by-muscle>`
> **Status**: 🟡 PLAN-ONLY (chờ user verb để Muscle thực thi)

---

## 1. Executive summary

### 1.1 Tóm tắt yêu cầu user

User propose UX mới thay thế approach cũ "FE-only hint":
- Form New Connector: nhập `MongoDB Connection URL` + `Database`
- Click "Check Connection" → progress bar chạy → scan collections trên BE
- PASS → render list collections dạng multi-select, default = all selected
- User uncheck những collection KHÔNG muốn CDC
- Nút "Create" chỉ hiện/enable khi check PASS
- Pattern mở rộng được cho MySQL/Postgres sau

### 1.2 Brain conclusion sau audit

- **~70% infra đã tồn tại**: worker service + BE endpoint + NATS subjects (Mongo).
- **Gap chính**: (1) worker handler đang nhận host+port build URI inline → drop auth; (2) FE chưa có hook + Collections vẫn là text input.
- **Strategy**: EXTEND không refactor. Add POST body endpoint + FE multi-select. Giữ GET legacy (R12).
- **Effort**: ~6h30m, 9 milestone M0..M8.
- **Risk**: LOW-MEDIUM (URI leak mitigated by ADR-002 POST body).

---

## 2. Files dự kiến thay đổi (P0)

| # | Repo | Path | Action | LOC ước tính |
|---|---|---|---|---|
| 1 | centralized-data-service | `internal/handler/command_handler.go` | Edit (extend 2 handler) | +70 |
| 2 | centralized-data-service | `internal/handler/command_handler_test.go` | Edit (add 7 unit) | +80 |
| 3 | cdc-cms-service | `internal/api/introspection_handler.go` | Edit (add 2 POST handler) | +100 |
| 4 | cdc-cms-service | `internal/api/introspection_handler_test.go` | Edit (add 6 unit) | +60 |
| 5 | cdc-cms-service | `internal/router/router.go` | Edit (register 2 POST) | +4 |
| 6 | cdc-cms-web | `src/services/connectorCheck.ts` | **NEW** | +60 |
| 7 | cdc-cms-web | `src/hooks/useConnectorCheck.ts` | **NEW** | +50 |
| 8 | cdc-cms-web | `src/pages/SourceConnectors.tsx` | Edit (state + UI + gate) | +135 |
| 9 | cdc-cms-web | `src/pages/SourceConnectors.test.tsx` | Edit (add 9 test) | +90 |

**Total**: ~649 LOC (code + test).

> Khi Muscle thực thi xong, REPLACE LOC ước tính bằng LOC thực tế từ `git diff --shortstat` + ghi commit hash.

---

## 3. Doc set workspace (đã tạo trong plan-only phase)

| File | Mục đích | Size |
|---|---|---|
| `00_context_check_connection.md` | Bối cảnh + audit findings + scope | 8.4KB |
| `01_requirements_check_connection.md` | R1-R12 + N1-N9 + DoD + Risk + Inverse | 9.3KB |
| `02_plan_check_connection.md` | M0..M8 roadmap + decision tree | 14.3KB |
| `03_implementation_check_connection.md` | Data flow + API contract + state machine | 18.6KB |
| `04_decisions_check_connection.md` | ADR-001..009 | 9.8KB |
| `05_progress.md` | Audit log (APPEND-ONLY) | 3.5KB |
| `06_test_cases_check_connection.md` | Unit + Integration + E2E + Security | 10.9KB |
| `08_tasks_check_connection.md` | Task checklist M0.1..M8.7 | 13.9KB |
| `09_tasks_solution_check_connection.md` | 14+ Edit demos full code | 24.3KB |
| `10_gap_analysis_check_connection.md` | Gap matrix per layer + baseline | (this phase) |
| `report_check_connection_2026-05-26.md` | File này (template + final) | (this file) |

---

## 4. Verify gates (Muscle phải fill sau khi thực thi)

### M0 — Pre-flight
- [ ] `git status` clean
- [ ] Mongo test instance UP
- [ ] BE + worker + FE local accessible
- **Result**: `<PASS|FAIL>` — note: `<...>`

### M1 — Worker extend
- [ ] `go vet ./internal/handler/...` — `<PASS|FAIL>`
- [ ] `go test ./internal/handler/... -run TestHandleDiscoverMongo` — `<N/N PASS>`
- [ ] Build worker binary `/tmp/cdc-worker-check-connection` — `<size, mtime>`
- [ ] Restart worker, log line `discover_mongo_databases uri=mongodb://***:***@host:port/?replicaSet=rs0` — `<evidence path>`

### M2 — BE relay
- [ ] `go vet ./internal/api/...` — `<PASS|FAIL>`
- [ ] `go test ./internal/api/... -run TestDiscoverMongo` — `<N/N PASS>`
- [ ] curl POST `/api/introspection/mongo/databases` → 200 — `<evidence>`
- [ ] curl POST `/api/introspection/mongo/collections` → 200 — `<evidence>`

### M3 — FE service + hook
- [ ] `npm run typecheck` — `<0 errors>`
- [ ] `npm run lint` — `<PASS>`
- [ ] Unit test `connectorCheck.test.ts` — `<N/N PASS>`

### M4 — FE UI integration
- [ ] `npm run build` — `<PASS>`
- [ ] Manual: Click Check button → Spin → render multi-select — `<screenshot path>`
- [ ] Manual: Uncheck collections → submit → BE nhận đúng `collection.include.list` — `<curl evidence>`

### M5 — Happy smoke E2E
- [ ] Create Mongo connector via UI với URI có auth — `<source_connection_id>`
- [ ] Verify SELECT cdc_source_connections WHERE id=<id> → collection.include.list = "db.a,db.b,db.c" — `<sql evidence>`
- [ ] Verify pipeline status RUNNING — `<status evidence>`

### M6 — Negative smoke
- [ ] Wrong URI → VN message "Không kết nối được..." — `<screenshot>`
- [ ] Wrong password → VN message "Sai thông tin xác thực" — `<screenshot>`
- [ ] DB không tồn tại → VN message "Database `<X>` không tồn tại. Database có sẵn: ..." — `<screenshot>`
- [ ] DB rỗng → VN message "Database `<X>` chưa có collection nào." — `<screenshot>`
- [ ] Worker timeout (kill worker) → VN message "Worker không phản hồi sau 10s..." — `<screenshot>`

### M7 — Security
- [ ] `/security-agent` chạy → `<output path>`
- [ ] grep `password=` / `mongodb://[^:]+:[^@]+@` trong log → `<0 matches>`
- [ ] DSN sanitize verified (L-3275) — `<evidence>`

### M8 — Report + memory
- [ ] File này được fill đầy đủ — `<commit hash>`
- [ ] `agent/memory/global/active_plans.md` updated → status Done
- [ ] Lesson nếu có → APPEND `agent/memory/global/lessons.md`

---

## 5. Behavior changes (BEFORE → AFTER)

| Scenario | BEFORE | AFTER |
|---|---|---|
| Add Mongo connector, fill URI có auth | Worker drop auth (bug) → fail | Worker dùng URI gốc → OK |
| Click "Check Connection" button | Button không tồn tại | Spin → render multi-select hoặc Alert VN |
| Collections field | `<Input>` text "users,orders" | `<Select mode="multiple">` với options từ scan |
| Submit khi chưa check (Create mode) | OK submit, có thể fail BE | Create disabled, ép check trước |
| Submit edit existing connector | Save string trực tiếp | Pre-fill multi-select, optional re-check |
| Wrong URI | BE 500 generic error | VN clear message 5-case |
| Empty Collections submit | Pass-through empty → CDC all (OK runtime) | Multi-select default all → submit explicit list |

---

## 6. Screenshots (Muscle fill sau)

> Placeholder — Muscle attach screenshot/asciinema sau khi smoke test pass.

- [ ] Form New Connector before fill
- [ ] After fill URI + DB, before click Check
- [ ] Spinning during check
- [ ] PASS → multi-select with all selected
- [ ] FAIL cluster_err → Alert
- [ ] FAIL db_missing với available list
- [ ] After uncheck 2 collections → Create button enable
- [ ] After Create → list view show new connector

---

## 7. Rollback plan

### Trigger rollback nếu
- Pipeline tạo từ UI mới fail → revert worker handler về host+port
- URI leak vào log → emergency stop, audit
- Performance regression check timeout > 30s consistent

### Steps
1. `git revert <commit>` ở repo có issue
2. Rebuild + redeploy
3. FE: revert `SourceConnectors.tsx` về `<Input>` text input (force resubmit UI)
4. KHÔNG cần migration rollback (phase này không có schema change)
5. APPEND `05_progress.md` với entry rollback + lý do

### Recovery time estimate: ~15 phút (3 service rebuild + redeploy)

---

## 8. Lessons learned (Muscle fill sau M8)

> Tóm tắt bài học rút ra trong phiên này. Cần generic hóa thành Global Pattern (CLAUDE.md §13).

### Candidate patterns sau audit
- **L-check-connection-extend-not-refactor**: Khi audit cho thấy >70% infra đã có (X), strategy đúng là EXTEND endpoint additive (POST mới) thay vì refactor (Y). Pattern global: `A reuses B by adding C variant, keeping legacy D` → giảm regression risk.
- **L-uri-in-post-body**: DSN/URL chứa secret KHÔNG được pass qua GET query (leak access log). Pattern: `Secret S transferred via POST body B, never query string Q`.

(Muscle bổ sung nếu phát hiện thêm).

---

## 9. Open items / Future phases

| Item | Why deferred | Future phase |
|---|---|---|
| MySQL introspection check | ADR-001 (0% infra) | `connector-check-mysql` |
| Postgres introspection check | ADR-001 (0% infra) | `connector-check-pg` |
| Cross-DB driver interface | ADR-007 (YAGNI với 1 impl) | `connector-driver-abstraction` |
| Streaming progress (SSE/WS) | ADR-005 (over-engineer P0) | `connector-streaming-introspect` |
| List view fallback `(All collections)` | Workspace cũ SUPERSEDED (ADR-009) | Out-of-scope; new approach explicit list |
| Wizard session persistence | ADR-008 (ephemeral đủ) | N/A trừ khi UX flow đổi |
| Cache check result server-side | Tránh stale | N/A |

---

## 10. Pre-flight Check §14 (CLAUDE.md)

Brain tự kiểm tra trước khi báo plan-only done:

- [x] §0 — Trả lời tiếng Việt: ✅ (doc viết VN)
- [x] §1 — Brain chỉ làm Chairman, không sửa code: ✅ (chỉ doc, Muscle thực thi)
- [x] §2 — Lệnh delegate có Mô tả + Data + DoD: ✅ (`08_tasks` + `09_tasks_solution` chi tiết)
- [x] §3 — Plan & Verify: ✅ (M0..M8 + per-milestone exit gate)
- [x] §6 — Simplicity First, no over-engineer: ✅ (defer cross-DB interface, defer wizard session)
- [x] §7 — Workspace + Full Doc Set: ✅ (11 file: 00, 01, 02, 03, 04, 05, 06, 08, 09, 10, report)
- [x] §11 — APPEND-ONLY memory: ✅ (05_progress chỉ có 1 entry init, không overwrite)
- [x] §12 — Brain Code Prohibition: ✅ (KHÔNG `.go/.ts/.js/.py/.sql` được sửa trong phase này)
- [x] §13 — Lesson abstract thành Pattern: ✅ (template trong §8 này)
- [x] §14 — Pre-flight: ✅ (đang chạy)

---

## 11. Verification commit list (Muscle fill sau)

| Repo | Commit hash | Subject |
|---|---|---|
| centralized-data-service | `<hash>` | `feat(worker): extend mongo introspect handlers accept full URI` |
| cdc-cms-service | `<hash>` | `feat(cms): add POST introspection endpoints (no URI in query)` |
| cdc-cms-web | `<hash>` | `feat(cms-web): add check-connection + collections multi-select` |

---

## 12. Sign-off

| Role | Name | Date | Status |
|---|---|---|---|
| Brain (plan) | claude-opus-4-7 | 2026-05-26 | ✅ Plan-only complete |
| Muscle (exec) | `<filled>` | `<filled>` | `<pending>` |
| User (review) | admin@homeproxy.vn | `<filled>` | `<pending>` |

---

**Verb chờ user** để Muscle thực thi:
- `execute` / `muscle thực thi` / `go` → giao Muscle chạy M0 → M8
- `revise` / `đổi plan` → Brain re-plan
- `defer` → archive plan
- `model: opus` / `model: sonnet` → chọn muscle model
