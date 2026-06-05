# 07_status.md — Workspace Status & DoD

| Field | Value |
|---|---|
| **Workspace** | `bug-gpay-id-trigger-contract-2026-06-02` |
| **Severity** | 🔴 HIGH — prod CDC pipeline block |
| **Phase** | **Brain — Planning & Documentation** |
| **Status** | 🟢 BRAIN_PLAN_COMPLETE — Awaiting User review |
| **Next action** | User review → answer 5 pending Q → approve Muscle P0 |
| **Brain compliance** | ✅ 0 source code change (CLAUDE.md §12) |
| **Date** | 2026-06-02 |

---

## 1. Brain DoD Checklist (CLAUDE.md §14 Pre-flight)

### 1.1 Workspace rules (CLAUDE.md §7)
- ✅ Path đúng: `agent/memory/workspaces/bug-gpay-id-trigger-contract-2026-06-02/`
- ✅ KHÔNG ghi đè workspace khác
- ✅ Prefix 00 → 10 + report đầy đủ
- ✅ `05_progress.md` append-only (CLAUDE.md §11)

### 1.2 Brain rules (CLAUDE.md §12)
- ✅ KHÔNG sửa `.go` / `.sql` / `.yaml`
- ✅ Plan + Document → User approve → Muscle thực thi
- ✅ KHÔNG chạy `Edit`/`Write` lên source code thực

### 1.3 Memory rules (CLAUDE.md §11)
- ✅ KHÔNG overwrite `agent/memory/global/lessons.md`
- ✅ Sẽ APPEND `active_plans.md` (chưa overwrite)

### 1.4 Lesson rules (CLAUDE.md §13)
- ✅ Lesson candidate draft trong `09_tasks_solution.md` §9
- ✅ Format pattern abstract A/B/X — chờ User confirm để append global

### 1.5 Full Doc Set (CLAUDE.md §7)
- ✅ 00_context.md
- ✅ 01_requirements.md
- ✅ 02_plan.md
- ✅ 03_implementation.md
- ✅ 04_decisions.md
- ✅ 05_progress.md
- ✅ 06_validation.md
- ✅ 07_status.md (file này)
- ✅ 08_tasks.md
- ✅ 09_tasks_solution.md
- ✅ 10_gap_analysis.md
- ⏳ report_2026-06-02.md (sẽ tạo cuối)

---

## 2. Phase status

| Phase | Status | Blocker | Owner |
|---|---|---|---|
| P0 (Reproduce) | ⬜ PENDING | User approve plan | Muscle |
| P1 (Migration) | ⬜ PENDING | Gate G0 | Muscle |
| P2 (Go DDL) | ⬜ PENDING | Plan approval (parallel với P1) | Muscle |
| P3 (Comment) | ⬜ PENDING | Plan approval (parallel) | Muscle |
| P4 (Integration test) | ⬜ PENDING | Gates G1+G2+G3 | Muscle |
| P5 (Deploy prod) | ⬜ PENDING | Gate G4 + User confirm | Muscle + User |
| P6 (Verify 24h + lesson) | ⬜ PENDING | Gate G5 | Brain + Muscle |

---

## 3. Source code touched by Brain (this session)

**0 file** ✅ (CLAUDE.md §12 compliance)

| Path | Action |
|---|---|
| `data-hub/centralized-data-service/` | READ only — forensics |
| `data-hub/cdc-cms-service/migrations/` | READ only — forensics |

---

## 4. Workspace files (measured by `wc -l`)

Cập nhật sau pre-flight (xem `report_2026-06-02.md`).

| File | Status |
|---|---|
| 00_context.md | ✅ CREATED |
| 01_requirements.md | ✅ CREATED |
| 02_plan.md | ✅ CREATED |
| 03_implementation.md | ✅ CREATED |
| 04_decisions.md | ✅ CREATED |
| 05_progress.md | ✅ CREATED |
| 06_validation.md | ✅ CREATED |
| 07_status.md | ✅ CREATED (file này) |
| 08_tasks.md | ✅ CREATED |
| 09_tasks_solution.md | ✅ CREATED |
| 10_gap_analysis.md | ✅ CREATED |
| report_2026-06-02.md | ⏳ pending (next step) |

---

## 5. Pending Questions cho User (BLOCKER cho Muscle P0)

(Tham chiếu `09_tasks_solution.md` §8)

1. **Q1:** DB-side DEFAULT `sf_nextval()` với random 8-bit seq — chấp nhận hay yêu cầu monotonic per-machine sequence?
2. **Q2:** Migration `019` áp dụng cho **mọi schema** có cột `_gpay_id` — có muốn whitelist schema cụ thể?
3. **Q3:** Deploy window prod — live deploy (migration metadata-only) hay maintenance window?
4. **Q4:** Có service nào khác đang dùng V2 shadow cần backport migration không?
5. **Q5:** Lesson global ghi vào `agent/memory/global/lessons.md` — User confirm format pattern abstract?

---

## 6. Tracking metrics (cập nhật khi Muscle execute)

| Metric | Baseline (trước fix) | After P5 (deploy) | After P6 (24h) |
|---|---|---|---|
| Batch upsert error rate `_gpay_id` | > 0 | TBD | **0** |
| Latency p99 batch upsert | TBD (đo P0) | ≤ +5% | ≤ +5% |
| Throughput rows/sec | TBD | ≥ baseline -5% | ≥ baseline -5% |
| V2 shadow tables with DEFAULT | 0 / N | **N / N** | N / N |
| Coverage `internal/handler/` | TBD | ≥ baseline | ≥ baseline |

---

## 7. Acceptance gate cho User Review

### 7.1 Completeness
- [ ] 12 file vật lý (00-10 + report)
- [ ] Mỗi file có nội dung thực
- [ ] Reference giữa file consistent (FR ↔ AC ↔ ADR ↔ task)

### 7.2 Correctness
- [ ] Root cause đúng (3-layer Contract Drift)
- [ ] Solution single source of truth (DB DEFAULT)
- [ ] ADR đầy đủ cho mỗi decision khó
- [ ] Rollback procedure rõ

### 7.3 Compliance
- [ ] Brain 0 source change
- [ ] Append-only `05_progress.md`
- [ ] 0 memory global overwrite
- [ ] Out-of-scope rõ (không creep)

---

## 8. Communication Protocol

### 8.1 User Decision Tree

**Branch A — User APPROVE full plan:**
- Append `05_progress.md`: `[User] APPROVE WORKSPACE — start P0`
- Muscle start T0.1

**Branch B — User REQUEST changes:**
- Append `05_progress.md`: `[User] REQUEST_CHANGES — <detail>`
- Brain CREATE `XX_revised_2026-06-02.md` (KHÔNG overwrite)
- Re-review loop

**Branch C — User REJECT solution (chọn approach khác):**
- Append `05_progress.md`: `[User] REJECT — <reason>`
- Brain RE-PLAN với approach mới (vd: Go-side fill, hoặc trigger)

### 8.2 Khi Muscle stuck (CLAUDE.md §8)
1. STOP sau 3 fail
2. Append `05_progress.md` ESCALATE
3. Notify Brain → re-plan

### 8.3 Khi prod incident sau deploy
1. Muscle execute rollback procedure (`03_implementation.md` §6)
2. Append `05_progress.md` ROLLBACK
3. Brain post-mortem

---

## 9. Sign-off

**Brain (Antigravity / Claude Opus 4.7):**

> Workspace `bug-gpay-id-trigger-contract-2026-06-02` Brain phase complete.
> 12 file vật lý đã tạo theo CLAUDE.md §7 prefix.
> **0 LOC source code change** (CLAUDE.md §12).
> 0 memory global overwrite (CLAUDE.md §11).
> Append-only `05_progress.md` (CLAUDE.md §11).
> Sẵn sàng User review + 5 pending question.

**Awaiting User:**
```
[YYYY-MM-DD HH:MM] [User] APPROVE WORKSPACE — start P0
```
HOẶC
```
[YYYY-MM-DD HH:MM] [User] REQUEST_CHANGES — <detail>
```
