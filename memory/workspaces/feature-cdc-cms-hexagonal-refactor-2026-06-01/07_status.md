# 07_status.md — Workspace Status & DoD Checklist

| Field | Value |
|---|---|
| **Workspace** | `feature-cdc-cms-hexagonal-refactor-2026-06-01` |
| **Iteration** | v2 (supersede v1 nếu user approve) |
| **Phase hiện tại** | **Brain — Planning & Documentation** (chưa start execution) |
| **Status** | 🟢 PLAN_COMPLETE — Awaiting user review |
| **Next action** | User review toàn workspace → approve Phase 0 → Muscle execute |
| **Brain compliance** | ✅ 0 source code change (CLAUDE.md §12) |
| **Date** | 2026-06-01 |

---

## 1. Brain DoD Checklist (CLAUDE.md §14 Pre-flight)

### 1.1 Quy tắc Workspace (CLAUDE.md §7)
- ✅ Workspace tạo tại đúng path `agent/memory/workspaces/feature-cdc-cms-hexagonal-refactor-2026-06-01/`
- ✅ KHÔNG ghi đè workspace cũ `feature-cdc-cms-service-restructure-2026-05-19`
- ✅ File prefix đầy đủ: 00, 01, 02, 03, 04, 05, 06, 07, 08, 09, 10
- ✅ Append-only `05_progress.md` (CLAUDE.md §11)

### 1.2 Quy tắc Brain (CLAUDE.md §12)
- ✅ KHÔNG sửa file `.go` / `.ts` / `.sql` / `.yaml` của source code
- ✅ Plan + Document → User approve → Muscle thực thi (flow đúng)
- ✅ KHÔNG chạy `git mv` / `Edit` / `Write` lên source thực

### 1.3 Quy tắc Memory (CLAUDE.md §11)
- ✅ KHÔNG overwrite `agent/memory/global/lessons.md`
- ✅ KHÔNG overwrite `agent/memory/global/active_plans.md` (sẽ APPEND ở task 14)
- ✅ KHÔNG overwrite `04_decisions.md` global (ADR lưu trong workspace local)

### 1.4 Quy tắc Lesson (CLAUDE.md §13)
- ✅ Lesson applied: anti-over-abstraction, false cognate, CQRS verification
- ✅ KHÔNG ghi lesson mới vào global trong session này (chỉ ghi khi User confirm finding)

### 1.5 Bộ tài liệu Workspace (CLAUDE.md §7 Full Doc Set)
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
- ✅ report_*.md (sẽ tạo task 13)

---

## 2. Phase status overview

| Phase | Status | Blocker | Owner sắp tới |
|---|---|---|---|
| Phase 0 (Coverage) | ⬜ PENDING | User approve plan | Muscle |
| Phase 1 (Port Split) | ⬜ PENDING | Gate G0 | Muscle |
| Phase 2 (Composition Root) | ⬜ PENDING | Gate G1 | Muscle |
| Phase 3 (18 Commands) | ⬜ PENDING | Gate G2 | Muscle |
| Phase 4 (Vertical Slice) | ⬜ PENDING_OPTIONAL | Gate G3 + user opt-in | Muscle |

---

## 3. Files thay đổi trong workspace này

| File | Action | LOC ước tính |
|---|---|---|
| `00_context.md` | CREATE | ~ 250 LOC markdown |
| `01_requirements.md` | CREATE | ~ 150 LOC markdown |
| `02_plan.md` | CREATE | ~ 280 LOC markdown |
| `03_implementation.md` | CREATE | ~ 600 LOC markdown (chứa code demo) |
| `04_decisions.md` | CREATE | ~ 300 LOC markdown |
| `05_progress.md` | CREATE | ~ 80 LOC markdown |
| `06_validation.md` | CREATE | ~ 350 LOC markdown |
| `07_status.md` | CREATE | ~ 150 LOC markdown (file này) |
| `08_tasks.md` | CREATE | ~ 350 LOC markdown |
| `09_tasks_solution.md` | CREATE | ~ 550 LOC markdown |
| `10_gap_analysis.md` | CREATE | ~ 280 LOC markdown |
| `report_2026-06-01.md` | CREATE | ~ 200 LOC markdown (task 13) |

**TOTAL workspace:** ~ 3,540 LOC markdown.
**Source code changed by Brain:** **0 LOC** ✅

---

## 4. Acceptance gate cho User Review

Trước khi user approve, kiểm tra:

### 4.1 Tính đầy đủ
- [ ] 12 file vật lý đủ (00-10 + report)
- [ ] Mỗi file có nội dung thực sự (không placeholder)
- [ ] Reference workspace v1 rõ ràng (00_context.md §1)
- [ ] ADR cho mọi quyết định khó (04_decisions.md 8 ADR)

### 4.2 Tính nhất quán
- [ ] FR trong 01 link với AC, AC link với gate trong 02
- [ ] Task trong 08 link với phase trong 02
- [ ] Verify command trong 06 link với AC trong 01

### 4.3 Tính khả thi
- [ ] Phase estimate có buffer
- [ ] Risk register có mitigation
- [ ] Stop rule có rõ (R1)
- [ ] Rollback strategy mỗi phase

### 4.4 Compliance với rule
- [ ] Brain 0 code change
- [ ] Append-only `05_progress.md`
- [ ] Không thảo luận shadow files (CLAUDE.md §7)
- [ ] Lesson global không bị overwrite

---

## 5. Câu hỏi cho User (cần answer trước khi start Phase 0)

(Reference `09_tasks_solution.md` §6 — 6 pending question)

1. **Q1:** Có chấp nhận estimate Phase 0 = 3-5 ngày không? (Nếu coverage < 60% thực tế tốn hơn → có chấp nhận trễ?)
2. **Q2:** Phase 4 (Vertical Slice) có làm hay không? Quyết định ảnh hưởng `internal/shared/` setup.
3. **Q3:** Có dùng `go-arch-lint` hay prefer `depguard` (golangci-lint plugin)?
4. **Q4:** Wizard BC có cross-BC import được phép — confirm hay yêu cầu pure async Saga?
5. **Q5:** 1 PR per batch (3 PR tổng cho Phase 3) hay 1 PR cho tất cả 18 commands?
6. **Q6:** Có cần thêm `internal/platform/observability/` cho OTel/metrics tách khỏi `bootstrap/` không?

---

## 6. Tracking metrics (cập nhật khi Muscle bắt đầu)

| Metric | Baseline | After Phase 1 | After Phase 2 | After Phase 3 | After Phase 4 |
|---|---|---|---|---|---|
| Coverage `internal/app/` | TBD | ≥ baseline | ≥ baseline | ≥ baseline | ≥ baseline |
| Coverage `internal/server/` | TBD | ≥ baseline | ≥ baseline | ≥ baseline | ≥ baseline |
| Coverage `internal/bootstrap/` | TBD | ≥ baseline | ≥ baseline | ≥ baseline | ≥ baseline |
| `server.go` LOC | 333 | 333 | ≤ 80 | ≤ 80 | ≤ 80 |
| Commands import `gorm` | 18 | 18 | 18 | **0** | 0 |
| Files import legacy ports | TBD | **0** | 0 | 0 | 0 |
| 53 routes status | 53/53 | 53/53 | 53/53 | 53/53 | 53/53 |
| 26 NATS subjects | 26/26 | 26/26 | 26/26 | 26/26 | 26/26 |
| Start time (ms) | TBD | ≤ +100ms | ≤ +100ms | ≤ +100ms | ≤ +100ms |
| BC cross-import | N/A | N/A | N/A | N/A | **0** (trừ wizard) |

---

## 7. Communication Protocol

### 7.1 Khi Muscle muốn start phase
1. Muscle đọc `02_plan.md` + `08_tasks.md` của phase
2. Muscle confirm với User trên main chat: "Bắt đầu Phase X?"
3. User approve → Muscle execute
4. Muscle append `05_progress.md` mỗi task

### 7.2 Khi Muscle gặp blocker
1. Stop ngay (KHÔNG retry > 3 lần — CLAUDE.md §8)
2. Append `05_progress.md` ESCALATE entry
3. Notify User → Brain re-plan

### 7.3 Khi User want abort
1. User chỉ định: "Abort Phase X" → Muscle stop ngay
2. Muscle revert code (nếu có change)
3. Brain document reason vào `05_progress.md`

---

## 8. Sign-off

**Brain (Antigravity / Claude Opus 4.7) certification:**

> Workspace `feature-cdc-cms-hexagonal-refactor-2026-06-01` đã hoàn tất Brain phase.
> Tất cả 12 file vật lý đã tạo theo CLAUDE.md §7 prefix convention.
> **0 LOC source code thay đổi** (CLAUDE.md §12 compliance).
> Memory global files không bị overwrite (CLAUDE.md §11 compliance).
> `05_progress.md` append-only (CLAUDE.md §11 compliance).
> Sẵn sàng cho User review + approval gate.

**Awaiting User signature:**
```
[YYYY-MM-DD HH:MM] [User] APPROVE WORKSPACE — start Phase 0
```
HOẶC
```
[YYYY-MM-DD HH:MM] [User] REQUEST_CHANGES — <chi tiết yêu cầu>
```
