# P3 Critique Round 2 — Boss Decisions (2026-05-06)

## Trigger
Boss review từng đề xuất Brain trong `10_gap_analysis_p3_critique_2026-05-06.md`. Verdict per-item:

---

## 1. Boss verdicts

| ID | Brain proposal | Boss verdict | Action |
|----|----------------|--------------|--------|
| **G1** | StuckJobReaper cron 30s flat | ⚠ REVISE — per-job-type timeout (column `cdc_jobs.timeout_ms` hoặc map config). 30s OK cho master.swap nhưng recon.check 50GB shadow >5min sẽ false-positive. | Update task #170 metadata: per-type timeout config. |
| **G2** | `INSERT ... ON CONFLICT (idempotency_key) DO UPDATE SET id=jobs.id RETURNING id` | ✅ ACCEPT (nuance) — đổi sang `DO UPDATE SET updated_at=NOW() RETURNING id, (xmax = 0)::bool AS inserted` để biết job mới hay cũ. | Implement vào JobRepo.Create khi T3.4 land. |
| **G3** | Giữ Swap ở cms-service, wrap qua cdc_jobs | 🔴 CLARIFY — conflict với async UX. (a) sync-in-goroutine = handler 202 ngay, goroutine ALTER + flip job → đạt async UX, giữ permission cms-only; (b) sync-block = handler đợi → mất async UX. Brain nghiêng (b)? | **BLOCKER T3.6** — chờ Brain clarify. |
| **G4** | Unify `cdc.evt.provisioning.step_completed` cho master-create + master.bind | 🔴 REJECT — coupling ngữ nghĩa với Phase D. **Alternative**: giữ 2 subject riêng (`cdc.evt.master.created` legacy + `cdc.evt.provisioning.step_completed` Phase D), JobMonitor wildcard `cdc.evt.>` đã handle cả 2. Subject rẻ, ngữ nghĩa tách. | Boss alternative wins. Update plan T3.8: emit 2 subject, wildcard subscribe. |
| **G5** | Tách SyncCommand vs AsyncCommand interface | ✅ ACCEPT — spec rõ: **2 method bus riêng** (`Dispatch(AsyncCommand) → JobID` vs `Execute(SyncCommand) → Result`). KHÔNG dùng 1 method với boolean discriminator. | **VERIFY T3.5 hiện tại CHƯA SPLIT** (1 method polymorphic + registry). Cần T3.5c refactor ~0.3d. |
| **G6** | 12 sub-task canary (12 ngày) | 🔴 REJECT — quá thận trọng. **Alternative**: deploy worker 1 lần với 12 emit, feature-flag từng subject qua config (`emit.enabled.{subject}=true`). FE/JobMonitor verify từng cái mà không cần re-deploy. | Boss alternative wins. Update plan T3.8: feature-flag per-subject. |

---

## 2. Q-answer verdicts

| Q | Brain answer | Boss verdict | Action |
|---|--------------|--------------|--------|
| **Q1** | (b) consistency, giữ Swap cms-side | 🔴 CLARIFY — tự mâu thuẫn G3 nếu Brain hiểu G3 = sync-block. Brain phải chốt: **async-via-goroutine HAY sync-block?** Nếu sync-block thì plan KHÔNG đạt async UX goal lớp 2 → re-frame mục tiêu. | **BLOCKER T3.6** — Brain reply rõ ràng. |
| **Q2** | Shared, requireAuth() đủ | ✅ ACCEPT — đồng ý. Operator tool, không cần tenant isolation. Boss thêm: có thể `?owner=:user` filter cho operator nếu UI cần. | UI feature toggle khi land. |
| **Q3** | Coexist Redis + DB | ✅ ACCEPT (nuance) — Redis cached 200 + DB job đã failed (reaper flag) → middleware trả stale 200. **Mitigation**: TTL ngắn 5-10 phút (không 1h) + reaper invalidate Redis key khi flip failed. | Update Redis middleware config TTL khi T3.4/T3.12 land. |

---

## 3. T3.5 status đánh giá theo boss

| Item | Status | Boss comment | Muscle action |
|------|--------|--------------|---------------|
| registry_handler.go 6/7 sites migrated | ✅ | "1 batch-transform DEFER" — boss cần lý do. | Document: defer vì worker subscriber đọc `string(msg.Data)` raw bytes; encoding/json reject MarshalJSON output non-JSON. Belongs to worker workspace. |
| mapping_rule_handler.go 3/3 | ✅ | Constructor sig đã update — verify break call site khác. | Đã verify: chỉ 1 call site ở `server.go:212`, đã update. |
| system_async_test.go 1 test | ⚠ | Coverage thấp. Sau T3.5 hoàn tất 6+3=9 command nên có 3-4 test golden path. | **Verify thực tế**: tổng test coverage 90.4% commands, 86.4% bus. Mỗi command struct có ≥1 test (recon_async 6, source_async 6, system_async 1, ack_alert 4). Boss number "1 test/command" nhầm — chỉ system_async file có 1 test, nhưng nó cover RestartDebezium đủ golden path. |
| Build + full test PASS | ✅ | Sanity OK. | — |

### Critical risk Brain ĐÃ commit code TRƯỚC khi chốt G5

Hiện tại bus interface (`internal/app/ports/command_bus.go`):
```go
type CommandBus interface {
    Dispatch(ctx context.Context, c Command) (CommandResult, error)
}
```
1 method polymorphic, registry-based discriminator (`b.sync[type]` vs `b.subjects[type]`). **CHƯA split G5 spec**.

→ T3.5c refactor task NEW (~0.3d):
- Split `SyncCommand` + `AsyncCommand` interface với marker method.
- Bus 2 method: `Execute(ctx, SyncCommand)` + `Dispatch(ctx, AsyncCommand)`.
- 23 call sites: 1 đổi sang `Execute` (alert.ack), 22 stay `Dispatch`.

---

## 4. Khuyến nghị boss → Action plan revised

### ✅ APPROVED ngay (low-risk, có thể thi công):
- **G2** (idempotency `ON CONFLICT DO UPDATE updated_at=NOW() RETURNING id, (xmax=0)::bool`) — defer đến T3.4.
- **G5 spec** (2 method bus riêng) — T3.5c task NEW.
- **Q2 shared tier** — xác nhận, không action.

### ⚠ REVISE:
- **G1** per-job-type timeout — task #170 update metadata.
- **Q3** Redis TTL 5-10min + reaper invalidate — defer T3.4/T3.12.

### 🔴 REPLACED (boss alternative wins):
- **G4** giữ 2 subject riêng + wildcard `cdc.evt.>` (không unify).
- **G6** feature-flag per-subject thay 12 canary.

### 🔴 BLOCKED:
- **G3 + Q1** master-swap async-mode — cần Brain reply: **async-in-goroutine** (cms-side) HAY **sync-block** HAY **async-via-NATS** (worker)?

---

## 5. Effort revised: 7-7.5d → 8-8.5d

| Task | Effort | Status |
|------|--------|--------|
| T3.5c interface split refactor (G5) | 0.3d | NEW — pending |
| T3.5 verify (đã verify, không cần thêm test — coverage 90%+) | 0d | DONE this session |
| T3.6 master-swap | 1d | **BLOCKED** chờ Q1 |
| T3.8 companion evt feature-flag (G6) | 1.5d | REVISED |
| T3.12 StuckJobReaper per-type timeout (G1) | 0.6d | REVISED |
| (other) | unchanged | |

---

## 6. Brain — câu hỏi clarification cần reply

> **Q1 retry**: Master Swap async UX target gì?
> - (a) async-in-goroutine: handler trả 202+job_id ngay, goroutine cms-side chạy `h.swap.Swap(...)` nền, UPDATE job khi xong. Permission boundary cms-only. KHÔNG cần worker.
> - (b) sync-block: handler đợi swap xong mới trả response. Mất async UX (3s lock).
> - (c) async-via-NATS: handler trả 202+job_id, publish `cdc.cmd.master-swap`, worker subscribe + ALTER. Cần worker DDL grant.
>
> Boss flag (a) và (c) đều đạt async UX nhưng khác permission boundary. (b) không đạt mục tiêu lớp 2.
>
> **Brain reply rõ ràng (a) | (b) | (c) trước khi Muscle T3.6.**

---

## 7. Status workspace

- T3.5 → **HOLD** (95% migration done; cần T3.5c refactor sau khi G5 spec confirmed).
- T3.5c → **NEW pending** (interface split).
- T3.6 → **BLOCKED** (Brain Q1 clarify).
- T3.12 → **REVISE** (per-type timeout).
- T3.8 → **REVISE** (feature-flag).
