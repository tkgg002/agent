# 07_status_report — Bug Snapshot Orphan No-Resume

**Workspace**: `agent/memory/workspaces/bug-snapshot-orphan-no-resume-2026-05-28/`
**Status**: **PLAN READY — chờ User verb `execute`**
**Date**: 2026-05-28
**Owner**: Brain (plan-only) → Muscle (execute)

---

## TL;DR
- 2 layer bug: BE worker không reclaim orphan khi boot + FE không có Resume cho `running` stale.
- Fix HOLISTIC 1 PR: BE goroutine boot reclaim + FE Force Resume button + 3 unit test.
- **Effort**: ~5h Muscle (Phase 1 BE 2h + Phase 2 FE 1h + Phase 3 Test 1.5h + Phase 4-5 verify+report 30min).
- **LOC ước tính**: ~+261 LOC trên **4 file** (2 BE + 1 FE + 1 test mới hoặc append).

---

## Workspace files (12 file)

```
bug-snapshot-orphan-no-resume-2026-05-28/
├── 00_context.md
├── 01_requirements.md
├── 02_plan.md
├── 03_implementation.md
├── 04_decisions.md
├── 05_progress.md
├── 06_validation.md
├── 07_status_report.md     ← file này
├── 08_tasks.md
├── 09_tasks_solution.md
├── 10_gap_analysis.md
└── report_bug_snapshot_orphan_no_resume_2026-05-28.md
```

---

## Patch list ước tính

| Patch | File | Type | LOC |
|---|---|---|---|
| B1+B2+B4 | `snapshot_runner_handler.go` | APPEND ReclaimOrphans + publishResumeMessage + const | +90 |
| B3 | `internal/server/worker_server.go` | APPEND goroutine boot reclaim | +12 |
| F1+F2+F3 | `cdc-cms-web/src/pages/SnapshotMonitor.tsx` | isStaleRunning + Force Resume + modal warning | +29 |
| T1+T2+T3 | `internal/handler/snapshot_runner_handler_test.go` | 3 unit test | +130 |

---

## Governance compliance

| Quy tắc | Status |
|---|---|
| §1 Brain plan-only | ✓ KHÔNG touch source code |
| §6 Simplicity First | ✓ Patch tactical, không refactor NATS sang JetStream |
| §7 Full Doc Set | ✓ 12 file vật lý |
| §11 Memory APPEND-only | ✓ |
| §12 Brain Code Prohibition | ✓ Code chỉ ở markdown demo |
| §13 Lesson abstract | ✓ Global Pattern (sẽ append `lessons.md`) |
| §14 Pre-flight 12 file | ✓ |

---

## Cross-reference

| Reference | Quan hệ |
|---|---|
| `snapshot-zero-records-2026-05-27/` | Fix Flush chain — chưa đủ. |
| `bug-snapshot-progress-mismatch-2026-05-28/` (vừa apply) | Fix cursor/pause/markDone. Bug hôm nay là vector orphan-recovery, không phải data correctness. |
| Lesson `L-2026-05-28-mark-done-without-completeness-guard` | Bug terminal transition. Lesson hôm nay khác: process death without graceful handoff. |
| **Lesson MỚI** `L-2026-05-28-boot-reclaim-missing-for-message-driven-runner` | Sẽ append sau Muscle apply. |

---

## Verb chờ User

| Verb | Hành động |
|---|---|
| `execute` | Muscle apply Phase 1+2+3+4+5 toàn bộ |
| `execute be` | Chỉ Phase 1 BE (defer FE) |
| `execute fe` | Chỉ Phase 2 FE (yêu cầu BE done) |
| `revise <patch_id>` | Plan lại patch cụ thể |
| `defer` | Lưu trạng thái, hoãn |
