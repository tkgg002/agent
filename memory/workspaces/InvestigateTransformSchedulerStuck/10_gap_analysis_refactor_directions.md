# 10_gap_analysis — Refactor Directions

**Created:** 2026-05-14
**Status:** EXPLORATION (chưa có user approval, chưa khởi tạo phase workspace)
**Input:** Tổng hợp findings từ `report_2026-05-13_*` → `report_2026-05-14_1408.md` (3 loại trigger A/B/C + 3 channel CMS→Worker + 11 gap)

---

## 1. Root issue (kỹ thuật gốc)

Hệ thống có **3 channel CMS→Worker không đồng nhất** + **state-machine nội bộ publish thẳng**, không có **single contract** cho "1 lệnh = 1 audit row + 1 activity row + propagation". Hệ quả:

| Channel | Cơ chế | Audit gap |
|---|---|---|
| Shared-DB (loại A) | Poll `cdc_worker_schedule` 60s | 5 cron không log ActivityLog (transmute_scheduler, full_count_aggregator, partition_dropper, dlq_worker, airbyte-sync orphan) |
| NATS `cdc.cmd.*` (loại B) | CommandBus / Publish thẳng | 6 subject bypass log + 4 path bypass `cdc_jobs` audit + provisioning publish thẳng |
| Kafka `cdc.<engine>.*` (loại C) | Subscribe Debezium topic | 1 sinkworker invisible (không log ActivityLog) |
| NATS `schema.config.reload` | Publish reload | Payload thiếu `version`/`sender` (no contract) |

**Tổng gap quan sát: 11 entry-point không log + 4 bypass audit + 1 payload thiếu contract.**

---

## 2. 5 hướng refactor (sort theo ROI)

### Hướng 1 — Activity-log middleware (Decorator pattern)
- **Mô tả:** Bọc tất cả handler (cron tick + NATS subscriber + Kafka consumer + sinkworker) qua 1 decorator log `start/done/error` thống nhất.
- **Effort:** S (1-2 tuần)
- **Impact:** Fix 11 gap, observability hoàn chỉnh, không cần đụng architecture.
- **Tradeoff:**
  - Risk: log explosion nếu không sample (cân nhắc rate-limit cho high-freq cron như `partition-check`).
  - Phải refactor entry point từng worker — risk regression nếu test coverage thấp.
- **Precondition:** Không có. Có thể start ngay.

### Hướng 2 — CommandBus mandatory (Mediator pattern)
- **Mô tả:** Xóa 4 path bypass + provisioning publish-thẳng. Ép tất cả NATS command đi qua `CommandBus`. Bus tự ghi `cdc_jobs` + ActivityLog (kết hợp với hướng 1).
- **Effort:** M (3-4 tuần)
- **Impact:** Audit complete, dễ test (mock 1 interface), unified error handling.
- **Tradeoff:**
  - Provisioning state machine phải refactor lớn (state transition hiện hard-code publish).
  - Breaking change cho 6 subject — cần migration plan.
- **Precondition:** Hướng 1 (cần ActivityLog đầy đủ trước để verify).

### Hướng 3 — Trigger DAG schema
- **Mô tả:** Thêm `trigger_id` (root) + `parent_id` cho sub-event vào `cdc_activity_log`. Phân biệt rõ root trigger vs sub (vd `recon-healer` hiện confuse vì TriggeredBy khác root). UI render tree thay vì flat list.
- **Effort:** M (2-3 tuần)
- **Impact:** Truy vết causality chain, không còn nhầm "loại trigger" với "audit phase".
- **Tradeoff:**
  - Migration DB live (ALTER TABLE 600+ row hiện tại, backfill `trigger_id` cho row cũ).
  - Rewrite UI list view → tree view.
- **Precondition:** Hướng 1 (cần đủ data để xác minh DAG đúng).

### Hướng 4 — Schedule push thay poll
- **Mô tả:** Bỏ poller 60s. Dùng NATS JetStream delayed-delivery hoặc 1 "tickler service" publish `cdc.cron.<job>` đúng giờ. Worker chỉ subscribe (giống loại B).
- **Effort:** M-L (4-6 tuần)
- **Impact:**
  - Loại orphan (airbyte-sync no-op): nếu không có subscriber, message DLQ → alarm rõ ràng thay vì silent.
  - Latency: 60s → <1s.
  - Unified với loại B (giảm 1 channel).
- **Tradeoff:**
  - Phải tự build/maintain tickler hoặc phụ thuộc JetStream delayed-delivery feature.
  - At-least-once delivery → worker handler phải idempotent (hiện tại có thể chưa).
- **Precondition:** Hướng 1 + 2 (cần audit/bus stable).

### Hướng 5 — Standardized envelope + OTel tracing
- **Mô tả:** Mọi message header chuẩn `{message_id, correlation_id, causation_id, version, sender, trace_id}`. W3C TraceContext propagate qua NATS/Kafka header. Span tree CMS click → CommandBus → Worker handler → Sub-event → Sink.
- **Effort:** L (1-2 tháng)
- **Impact:**
  - Truy vết end-to-end trên 1 trace ID (Jaeger/Tempo).
  - Fix payload thiếu `version`/`sender` (vd PublishReload).
- **Tradeoff:**
  - Cần infra OTel collector + backend (Jaeger/Tempo/Honeycomb).
  - Cost lưu trace (10-100% sampling tùy traffic).
- **Precondition:** Hướng 1-3 (tracing chỉ giá trị khi data flow đã chuẩn).

---

## 3. Phase recommendation

| Phase | Hướng | Priority | Lý do |
|---|---|---|---|
| **Phase 1** | #1 (middleware) | P0 | Quick win — 11 gap biến mất, không đụng architecture, precondition cho mọi phase sau |
| **Phase 2** | #2 + #3 song song | P1 | Khi có audit đầy đủ → standardize command flow + DAG schema |
| **Phase 3** | #4 (push scheduler) | P2 | Bỏ poller — cần phase 1-2 stable trước |
| **Phase 4** | #5 (tracing) | P3 | "Nice to have" — chỉ giá trị khi 1-3 xong |

---

## 4. Decision cần từ user

1. **Ưu tiên:** Fix observability (Phase 1) trước hay architecture (Phase 2) trước?
2. **DB migration:** Có rào cản nào với việc ALTER `cdc_activity_log` thêm `trigger_id`/`parent_id` trên live DB?
3. **OTel infra:** Team có Jaeger/Tempo sẵn chưa? Nếu chưa → Phase 4 deferred.
4. **Idempotency:** Khi push scheduler (Phase 3) cần handler idempotent. Có audit nào về tình trạng idempotency hiện tại của worker handlers chưa?

---

## 5. Next step (nếu user approve)

- Tạo workspace mới: `agent/memory/workspaces/CDC-RefactorPhase1-Middleware/` với full doc set:
  - `01_requirements_phase1.md`
  - `02_plan_phase1.md`
  - `03_implementation_phase1.md`
  - `08_tasks_phase1.md`
  - `09_tasks_solution_phase1.md`
- Brain (Antigravity) plan, Muscle (CC) execute theo CLAUDE.md §1.

---

## 6. Constraint compliance

- Brain Code Prohibition (§12): file này KHÔNG sửa source code, chỉ propose.
- No Shadow Files (§7): file vật lý lưu tại đường dẫn workspace.
- Plan & Verify (§3): mọi gap đã verify trên DB live + grep code (xem `report_2026-05-14_1408.md`).
