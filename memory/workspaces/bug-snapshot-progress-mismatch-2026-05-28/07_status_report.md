# 07_status_report — Bug Snapshot Progress Mismatch

**Workspace**: `agent/memory/workspaces/bug-snapshot-progress-mismatch-2026-05-28/`
**Status**: PLAN READY — chờ User verb
**Date**: 2026-05-28
**Owner**: Brain (plan-only) → Muscle (execute)

---

## TL;DR
- 3 root cause holistic patch:
  - **A**: `len(batch) < BatchSize` early-exit (line 553-555).
  - **B**: pause `break` fall-through `markProgressDone` (line 352-357 + 569).
  - **C**: `markProgressDone` không guard completeness (line 712-721).
- Defense-in-depth: invariant enforce ở `markProgressDone` (terminal transition) — bottleneck duy nhất sang status=done.
- **Effort**: ~5.5h Muscle (Phase 1 Core 2h + Phase 2 Metric 1h + Phase 3 Test 2h + Phase 4 Verify 30min).
- **LOC ước tính**: +23 production / +130 test = ~+153 NET, **3 file**.

---

## Workspace files (12 file vật lý)

```
bug-snapshot-progress-mismatch-2026-05-28/
├── 00_context.md                              — Trigger + 3 root cause + math
├── 01_requirements.md                         — 5 FR + 4 NFR + 7 DoD
├── 02_plan.md                                 — 4 phase sequencing + LOC table
├── 03_implementation.md                       — Code demo 5 patch S1-S5 + O1 + T1-T3
├── 04_decisions.md                            — 7 ADR
├── 05_progress.md                             — Audit log APPEND-only
├── 06_validation.md                           — Verify matrix + AC + smoke test
├── 07_status_report.md                        — File này
├── 08_tasks.md                                — Checklist Muscle
├── 09_tasks_solution.md                       — Rationale + alternative rejected
├── 10_gap_analysis.md                         — Gap A/B/C → Fix → Verify map
└── report_bug_snapshot_progress_mismatch_2026-05-28.md   — Bảng files thay đổi + LOC delta
```

---

## Governance compliance check

| Quy tắc | Status | Evidence |
|---|---|---|
| §1 Brain plan-only | ✓ | Brain không touch source code |
| §6 Simplicity First | ✓ | Patch tối thiểu, không re-architect |
| §7 Full Doc Set | ✓ | 12 file vật lý |
| §11 APPEND-only Memory | ✓ | `05_progress.md` chỉ APPEND |
| §12 Brain Code Prohibition | ✓ | Code chỉ demo trong markdown |
| §13 Lesson abstract | ✓ | Lesson global pattern A/B/X/Y (sẽ append `lessons.md`) |
| §14 Pre-flight | ✓ | File count verify |

---

## Cross-reference

| Lesson/Workspace | Quan hệ |
|---|---|
| `lessons.md` 2026-05-26 line 3417-3421 "Define DoD at destination" | Lesson cũ; bug hôm nay là regression — chứng minh DoD destination CHƯA đủ. |
| `snapshot-zero-records-2026-05-27/` | Fix layer Flush + counter PG RowsAffected. Vẫn còn 3 root cause khác (workspace hôm nay). |
| `bug-first-snapshot-no-write-2026-05-26/` | Fix layer 1 HandleRaw. |
| **Lesson MỚI 2026-05-28** | "Enforce invariant ở edge (terminal transition), không intermediate" — chống whack-a-mole. |

---

## Verb chờ User

| Verb | Hành động |
|---|---|
| `execute` | Muscle apply Phase 1+2+3+4 (Core + Metric + Test + Verify) |
| `execute core` | Chỉ Phase 1 (production fix, defer test) — KHÔNG khuyến nghị, để DoD đầy đủ |
| `revise` | User chỉ định patch nào cần plan lại |
| `defer` | Tạm hoãn, lưu trạng thái |

---

## Risk register
| Risk | Mitigation |
|---|---|
| Completeness threshold 0.99 quá strict trên dataset write-heavy | Configurable via constant; có thể tune sau monitoring |
| Pause `return nil` thay đổi semantics — user resume cần CMS re-publish | Đã có flow CMS publish lại snapshot.v2 với resume_from |
| Test coverage chưa cover replication lag thực | Bù bằng metric + alert production |
