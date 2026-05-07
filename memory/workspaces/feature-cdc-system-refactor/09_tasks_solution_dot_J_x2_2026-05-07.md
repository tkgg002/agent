# Tasks Solution — Đợt J (x2 review + plan)

> **Author**: x2 (Muscle, cms-lane) | **Date**: 2026-05-07 ICT
> **Source plan**: `02_plan_dot_J_2026-05-07.md` (max), `08_tasks_dot_J_2026-05-07.md` (max)
> **Lane**: x2 lock `cdc-cms-service/`. max KHÔNG đụng cms.

## 1. Pre-flight kết quả thực

| Probe | Cmd | Result |
|---|---|---|
| HEAD cms | `git rev-parse HEAD` | `b4a3461` (đợt I) ✅ |
| Build baseline | `go build ./...` | EXIT 0 ✅ |
| Test baseline | `go test ./... -count=1` | all PASS (api, infra/{http,messaging,persistence}, middleware, service, service/health/probes) ✅ |
| DoD grep `service.{Collector,NewCollector,CollectorConfig,Snapshot}` | `grep -rEn ...` | **5 hit functional**: server.go × 3 (L37, L235, L236) + system_health_handler.go × 1 (L108 `service.Snapshot`) + 1 hit comment-only (L4 system_health_handler) |
| `internal/service/health/probes` import path | `grep -rln ...` | 3 file: `system_health_queries.go` + `system_health_collector.go` + `system_health_collector_test.go` (cluster sẽ tự move cùng) |
| `cmd/` ref | `grep -rEn ...` | 0 hit ✅ |
| `internal/infra/observability/` | `ls` | KHÔNG tồn tại — sẵn sàng tạo mới |
| `pkgs/observability/otel.go` | `ls` | tồn tại nhưng khác namespace (`pkgs/`), không conflict |

**Kết luận pre-flight**: Pass — sẵn sàng đợt J.

## 2. Review max plan

Đồng ý với max plan **Option B** (`infra/observability/{,probes/}`). Lý do:
1. Semantic fit (observability stack), không bị bias về protocol HTTP.
2. Cluster co-locate giảm cross-pkg noise.
3. 1 commit duy nhất đủ đóng Task #19.

**Deviation từ max plan** (x2 quyết):
- Không có deviation lớn. Theo đúng 9-step max đề xuất.
- Step 6 cosmetic clean `model/alert.go:12` comment: x2 sẽ làm cùng commit J (gộp).
- Step 9 (rebuild cms-server) — x2 sẽ thực hiện sau khi commit J + APPEND progress xong, KHÔNG skip.
- Sẽ tạo `report_dot_J_x2_2026-05-07.md` cho Boss check (theo Boss directive).

## 3. Plan x2 chốt — execution sequence

### Phase A — bulk move (8 phút dự kiến)

1. `mkdir -p internal/infra/observability/probes`
2. `cp` 7 file system_health_* → `internal/infra/observability/`
3. `cp` 14 file probes/* → `internal/infra/observability/probes/`
4. `sed -i '' 's/^package service$/package observability/'` cho 7 file system_health_*. Probes giữ `package probes`.
5. `sed -i '' 's|cdc-cms-service/internal/service/health/probes|cdc-cms-service/internal/infra/observability/probes|g'` cho 3 file system_health_{queries,collector,collector_test}.

### Phase B — caller update (5 phút)

6. `sed -i '' 's/service\.Collector/observability.Collector/g; s/service\.NewCollector/observability.NewCollector/g; s/service\.CollectorConfig/observability.CollectorConfig/g; s/service\.Snapshot/observability.Snapshot/g'` 2 file: `internal/server/server.go` + `internal/api/system_health_handler.go`.
7. `Edit` import block 2 file: thêm `"cdc-cms-service/internal/infra/observability"`. Bỏ `"cdc-cms-service/internal/service"` ở `system_health_handler.go` nếu không còn ref. `server.go` còn ref `service.X` khác — giữ import.
8. `Edit` cosmetic comment ở `internal/api/system_health_handler.go:4` (comment) + `internal/model/alert.go:12` (comment).

### Phase C — drop service/ + verify (4 phút)

9. `rm internal/service/system_health_*.go internal/service/system_health_*_test.go`
10. `rm -r internal/service/health`
11. `ls internal/service/` — nếu empty → `rmdir internal/service` (close Task #19).
12. `go build ./...` PASS.
13. `go vet ./...` clean.
14. `go test ./... -count=1` PASS toàn repo.
15. DoD grep:
    - `grep -rEn "service\.(Collector|NewCollector|CollectorConfig|Snapshot|StatusOK|StatusDegraded|StatusDown|StatusUnknown|StatusUp)" --include="*.go" .` → 0 hit functional.
    - `grep -rln "internal/service/health/probes" --include="*.go" .` → 0 hit.

### Phase D — commit + handover (3 phút)

16. `git status --short` review.
17. `git add internal/service/ internal/infra/observability/ internal/server/server.go internal/api/system_health_handler.go internal/model/alert.go`.
18. Commit subject `refactor(cms): Task #19 đợt J — drain system_health_* + probes/ to infra/observability (Task #19 closed)`.
19. APPEND `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` entry x2 đợt J.
20. APPEND `coordination_max_x2_2026-05-07.md` "Task #19 closed at cms <commit>".
21. Commit agent docs.

### Phase E — runtime verify (Q3, 5 phút)

22. `kill 33841` (cms-server PID đang chạy binary cũ `/tmp/cdc-cms-service-t27`).
23. `go build -o /tmp/cdc-cms-service-postJ ./cmd/server`.
24. `/tmp/cdc-cms-service-postJ &` (background).
25. Smoke `/health` 200 + 1-2 endpoint (registry list / system-health snapshot).
26. APPEND `05_progress.md` entry runtime verify.

### Phase F — report cho Boss (3 phút)

27. Tạo `cdc-cms-service/report_dot_J_x2_2026-05-07.md` (Boss directive: report_*.md cho mỗi changes).

## 4. Risk & mitigation

| Risk | Mitigation |
|---|---|
| Sed insert import block fragile (BSD sed multiline) | Dùng Edit tool thay vì sed — đã ghi rõ Phase B step 7 |
| `system_health_alerts.go` import `persistence.X` cross-pkg sau move sẽ không hợp lệ vì ngoài `service/` package | Khả năng cao OK: sau move, file đã ở `observability/`, `persistence.X` vẫn cross-pkg ref valid (giống cũ). Nếu fail → `goimports -w` |
| Rename detection < 98% nếu sed nhiều dòng | Sed chỉ thay 1 dòng (`^package service$`); body identical → expect ≥99% |
| `system_health_collector_test.go` reference cross-pkg helpers | Cùng package observability → no import needed |
| `gpay-postgres-cdc` health endpoint require live DB | Đã verify port 8083 + DB live ở task wizard |

## 5. DoD x2 (kế thừa max DoD)

- ✅ `internal/service/` empty hoặc removed.
- ✅ Build/test PASS.
- ✅ DoD grep stale = 0 hit functional.
- ✅ Commit subject "Task #19 closed".
- ✅ APPEND `05_progress.md`.
- ✅ Coordination file noted "Task #19 closed at cms `<commit>`".
- ✅ cms-server rebuild + restart smoke (Phase E).
- ✅ `report_dot_J_x2_2026-05-07.md` (Phase F) — Boss directive.

— x2
