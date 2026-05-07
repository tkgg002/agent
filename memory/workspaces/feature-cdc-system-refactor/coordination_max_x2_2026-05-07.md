# Coordination — max ↔ x2 (2026-05-07)

> **Boss directive (initial)**: "max sẽ làm cùng thằng x2 ở workspace `feature-cdc-system-refactor/`"
> **Boss directive (revised, 2026-05-07 ICT)**: "Lane phân chia: max làm tài liệu tổng thể, phân chia task, lock centralized-data-service/ (worker), x2 lock cdc-cms-service/ (cms)"
> **Auto mode**: ON. Hai agent song song, chia lane theo phạm vi đụng file.

## Lane Lock (REVISED 2026-05-07 ICT — role swap effective from commit `b4a3461`)

| Agent | Owns | Touches | Forbidden |
|---|---|---|---|
| **max** (Opus 4.7) | Tài liệu tổng thể, phân chia task, worker code | `centralized-data-service/internal/...`, workspace docs (`feature-cdc-system-refactor`, `feature-cdc-integration`, `feature-multi-pg-isolation-e2e`), `agent/memory/global/{lessons,project_context,active_plans}.md`, migrations cdc | `cdc-cms-service/internal/...`, FE `cdc-cms-web/`, live runtime restart, push remote |
| **x2** (other CLI) | CMS code refactor (Task #19 đợt J + tail), CMS test/build | `cdc-cms-service/internal/{service,infra,api,server,middleware,router,model}/`, `cdc-cms-service/cmd/`, CMS workspace progress entries (APPEND) | `centralized-data-service/`, FE, runtime restart, push remote |

**Lịch sử lane (trước swap)**: Đợt G (`3424764`) + H (`ff16e38`) + I (`b4a3461`) do max thi công khi CMS còn nằm dưới max-lane. Sau commit I, x2 nhận quyền code lên CMS.

## Shared resources

| File | Rule |
|---|---|
| `agent/memory/global/lessons.md` | APPEND only (CLAUDE.md §11). Mỗi agent stamp `[max]` hoặc `[x2]` ở header lesson + timestamp ICT để truy vết. |
| `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` | APPEND only. Mỗi entry stamp agent + timestamp + commit hash khi có. |
| Git working tree (`cdc-system/`) | max chỉ stage/commit file ở `cdc-cms-service/`. x2 chỉ stage/commit file ở `centralized-data-service/`. Không ai dùng `git add .` / `git add -A`. |
| Git working tree (`agent/`) | Atomic commit per agent — APPEND tiếp với `--message` rõ chủ thể. |

## Live state snapshot (max ghi 2026-05-07 ICT post-/loop verify by x2)

- cms-server PID `33841` `/tmp/cdc-cms-service-t27` — chạy binary build trước Đợt G/H. **Pause Q3 chờ Boss confirm rebuild + restart.**
- cdc-worker PID `23565` `/tmp/cdc-worker-host` — x2 đã verify V2 bridge end-to-end DONE post-cron-tick (per `feature-multi-pg-isolation-e2e/05_progress.md` 2026-05-07 entry).
- Track D Hardening (P1+P2+P3+P4 + 045 + 046 model drift) — DONE per x2.
- Wizard tier-classification re-verify post-compact (port 8083) — DONE per x2 (`report_wizard_tier_reverify_20260507.md`).
- cdc-cms-service `internal/service/` còn: `alert_manager`, `approval_service`, `source_object_v2_sync` (Bucket A); `system_health_*` cluster + `health/probes/` (Bucket A*/C). max sẽ drain Plan A đợt I (Bucket A) → đợt J (Bucket A*/C).

## Handshake protocol

- Trước khi commit, agent grep `git log --oneline -5` của repo liên quan để xác nhận không bị diverge.
- Nếu phát hiện file ở lane đối tác bị dirty / staged trên working tree, KHÔNG `git add` → để cho owner xử lý.
- Khi report mới được tạo, tag rõ `Author: max` hoặc `Author: x2` ở đầu file.

## Open question for Boss (vẫn pause)

- Q1 Plan A vs Plan B → **default Plan A** (Boss đã rõ "1-2 commit cuối, đóng Task #19"); đợt I done, đợt J nằm ở x2.
- Q2 `infra/external/probes/` mới hay reuse `infra/http/probes/` → recommend **reuse `infra/http/probes/`** (pattern đã dùng cho `prom_client.go`). Final call → x2 quyết khi thi công đợt J vì lock đã chuyển.
- Q3 rebuild + restart cms-server → x2 sẽ verify runtime sau đợt J (cms-lane). max KHÔNG động.

## Task spec cho x2 (Đợt J — cluster cuối Task #19)

### Mục tiêu
Drain 7 file `internal/service/` còn lại + cluster `internal/service/health/probes/` ra `infra/` để đóng Task #19.

### Files & bucket (trích từ `report_session_audit_2026-05-07.md` §4-5)

| File | Bucket | Đề xuất destination |
|---|---|---|
| `system_health_alerts.go` (+ test) | A* (pure-fn `*Collector` method) | co-locate với collector |
| `system_health_compute.go` (+ test) | A* (pure-fn) | co-locate với collector |
| `system_health_queries.go` | A* (Collector method, DB) | co-locate với collector |
| `system_health_collector.go` (+ test) | C (HTTP client gọi worker) | `internal/infra/external/health/` (mới) **HOẶC** reuse `internal/infra/http/` (rec.) |
| `service/health/probes/{debezium,deps,kafka_connect,kafka_lag,nats,postgres,redis,worker}.go` (+ tests) | C | `internal/infra/http/probes/` (rec. — reuse pattern `prom_client.go` đã ở `infra/http/`) |

### Pattern thi công (đã proven Đợt G/H/I)
1. `cp + sed -i '' 's/^package service$/package <newpkg>/'` — byte-equivalent move (≥98% rename detection).
2. Bulk sed cross-file thay refs `service.X` → `<newpkg>.X` ở tất cả callers.
3. Fix import block: bỏ unused `cdc-cms-service/internal/service`, thêm `cdc-cms-service/internal/infra/<newpkg>`.
4. Build verify (`go build ./...`) + test verify (`go test ./... -count=1`).
5. DoD grep `service\.<symbol>` = 0 hit functional.
6. Commit subject: `refactor(cms): Task #19 đợt J — ...`. Co-Authored-By: x2.

### Caller hotspots cần check trước khi sed
```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service
grep -rn -E "service\.(NewCollector|Collector|CollectorConfig|StatusOK|StatusDegraded|StatusDown|StatusUnknown|Snapshot|FireRequest|Fingerprint)" --include="*.go"
grep -rn "cdc-cms-service/internal/service/health/probes" --include="*.go"
```
- `internal/server/server.go` đã có `service.NewCollector(...)` (line 235), `service.CollectorConfig{...}` (line 236), `service.Collector` (line 37). Đây là caller chính.
- `internal/api/system_health_handler.go` (nếu có) — recheck.
- 2 file system_health_* hiện tại đang qualify `*persistence.AlertManager` ở line 39, 103 — sau khi system_health_* tự move sang infra, package mới sẽ tự nhiên ref `persistence.AlertManager` (cross-package), KHÔNG đụng.

### Rủi ro đã biết (cảnh báo cho x2)
- `service/health/probes/postgres.go` ping DB qua `*gorm.DB` direct — không thay đổi semantics khi move sang `infra/http/probes/` (tên thư mục `http` hơi misleading nhưng không ảnh hưởng build). Nếu x2 muốn rename dir thành `infra/probes/` đứng độc lập cũng ok — quyền x2 quyết.
- `system_health_collector.go` đã import `cdc-cms-service/internal/service/health/probes` ở line 28. Khi move probes/ → `internal/infra/http/probes/`, đường import phải sed cùng lúc → caller chính là chính file collector.
- Comment cosmetic `// natural key used by service.AlertManager` ở `internal/model/alert.go:12` — x2 có thể clean luôn khi đụng.

### DoD đợt J
- Build PASS toàn repo.
- Test PASS — đặc biệt `internal/service/`, `internal/api/`, `internal/server/`, package mới.
- DoD grep stale `service.<symbol>` cho mọi symbol Bucket A*/C → 0 hit.
- `internal/service/` còn rỗng (không file `.go` ngoài subdir đã move) HOẶC chỉ còn empty package marker (recommend rỗng — drop `service/` luôn).
- APPEND vào `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` entry "Đợt J — closed Task #19" với commit hash.
- Cms-server runtime verify (Q3) → x2 quyết khi nào rebuild + restart sau đợt J.

— max

---

## 🔔 UPDATE 2026-05-07 ICT (max-Brain hand-off, agent commit `dd21443`)

Plan + tasks chính thức cho Đợt J đã được vật lý hóa (CLAUDE.md §7 Full Doc Set):

- `02_plan_dot_J_2026-05-07.md` — Option B (`infra/observability/{,probes/}`), audit fact base (cluster A* 4 source + 4 test, cluster C 8 source + 6 test), only 2 cross-package callers (server.go 7 sites + api/system_health_handler.go), 9-step execution sequence, risk table, DoD final.
- `08_tasks_dot_J_2026-05-07.md` — checklist J.1-J.10 với acceptance criteria.

x2 nên dùng 2 file mới làm **source of truth** cho thi công (chi tiết hơn spec inline ở §"Task spec cho x2" phía trên — section đó vẫn giữ làm overview).

`active_plans.md` đã APPEND entry `feature-cdc-system-refactor` (agent commit `24fbe26`).

— max

---

## 🎉 Task #19 CLOSED at cms commit `b453d36` (x2 đợt J — 2026-05-07 ICT)

**Status**: cms-lane unlock back to shared. `internal/service/` removed entirely (10 đợt drainage A→J). Build/test PASS toàn repo. Cms-server runtime verify (Q3) — x2 sẽ thực hiện Phase E (rebuild + restart `/tmp/cdc-cms-service-postJ` + smoke).

**Hand-back**: max có thể resume worker-lane (fix sub-issues + Track E plan).

— x2
