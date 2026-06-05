# 05_progress — APPEND-ONLY

---

## [2026-05-26 18:30] [Agent:claude-opus-4-7] Brain plan complete

**Actor**: Brain (Chairman, CLAUDE.md §12 — không sửa code).
**Trigger**: User verb (paraphrase): "snapshot chỉ là 1 ví dụ. tất cả các flow trong hệ thống đều phải gom lại theo span cha."

**Pre-work**:
- ✅ Đọc `agent/memory/global/lessons.md` (L-2026-05-26-trace + 4177 dòng).
- ✅ Đọc `agent/GEMINI.md` (14 quy tắc).
- ✅ Đọc workspace phase 2 (`feature-trace-span-attrs-2026-05-26/*`).
- ✅ Survey toàn repo qua Explore subagent: 35 NATS subscribers, 2 HTTP servers, 2 Kafka consumers, 12 timer/cron loops, 7 background goroutines, 66 `context.Background()` sites, 18 goroutine spawn sites, 5 existing spans.

**Files tạo (workspace, KHÔNG đụng source code)**:
- `00_context.md` — bối cảnh, audit, scope, risks (135 dòng).
- `01_requirements.md` — R1-R10, N1-N7, DoD A1-A11, NEG-1..6 (140 dòng).
- `02_plan.md` — M0-M12 với code demo chi tiết per milestone, risk matrix, pre-execution checklist (~400 dòng).
- `04_decisions.md` — ADR-A01..A20 (22 row).
- `08_tasks.md` — T0.1..T12.7 chi tiết per milestone.
- `09_tasks_solution.md` — Reference snippet cho mỗi task (~500 dòng code Go + SQL).
- `10_gap_analysis.md` — Baseline vs target + risk gaps + effort breakdown.

**Key decisions**:
- Mỗi flow = 1 root span name `<subsystem>.<verb>` (ADR-A01).
- W3C TraceContext qua NATS + Kafka header (ADR-A02, ADR-A03).
- Background worker root span per **tick**, không per goroutine (ADR-A04).
- Snapshot chunked traces mỗi 100 batches (ADR-A05, ADR-A17).
- BatchBuffer dùng Span Link (KHÔNG fake parent) cho fan-in (ADR-A06, ADR-A07).
- Migration tạo file, KHÔNG apply (ADR-A10).
- SinkWorker defer phase sau (ADR-A12).
- KHÔNG đụng existing `trace_id` app-level column (ADR-A11).

**Effort estimate**: ~12 giờ Muscle execution, có thể chia 2-3 phiên.

**Status**: ✅ Plan-only. Chờ user verb để Muscle thực thi.

**Next verbs chờ user**:
- `thực hiện` / `execute` / `go` → Muscle chạy M0 → M12.
- `subset M1+M2+M5` → chỉ làm helper + NATS sub + BG workers.
- `revise <section>` → re-plan section cụ thể.
- `defer` → archive plan.
- `apply migration M10` → riêng cho DDL.

---
