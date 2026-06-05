# 04_decisions — Architectural Decision Records

> ADRs cho workspace `feature-dashboard-v2-monitoring-2026-05-27`. Mỗi ADR liệt kê alternative đã cân nhắc + trade-off.

---

## ADR-001 — Lấp G1..G4 ở worker process thay vì xây service mới

**Context**: Audit chỉ ra 4 gap (ConsumerLag dead, transient error chưa đếm, sinkworker không classify, broker probe thiếu) ở **worker process**. Có thể chọn: (A) sửa trực tiếp worker, (B) tách 1 sidecar process scrape Kafka rồi expose Prometheus, (C) chỉ scrape kafka-exporter (đã có).

**Decision**: Chọn **(A) sửa trực tiếp worker** + **giữ (C) kafka-exporter** làm nguồn cross-check ở CMS probe.

**Lý do**:
- Worker đã có code path đọc message + classifier → chi phí marginal thấp.
- Sidecar phải duplicate auth/config → tăng surface bảo mật.
- kafka-exporter cho topology view ngoài; worker metric cho per-handler view trong. **Hai nguồn truth cùng tồn tại có chủ đích** — sai khác → operator biết khi đang điều tra (signal vs noise).

**Trade-off**: thêm metric vào worker → +1 lock contention nhỏ. Mitigation: RateMeter có per-topic mutex riêng.

**Lesson tham chiếu**: `L-2026-05-26-metric-defined-but-never-set` mục "Definition near use".

---

## ADR-002 — Aggregator endpoint nằm ở `cdc-cms-service`, KHÔNG ở worker

**Context**: FE cần 5 endpoint Dashboard V2. Có thể đặt: (A) tất cả ở worker (`centralized-data-service/admin-api`), (B) tất cả ở cms-service, (C) split — timeline ở worker, metadata ở cms.

**Decision**: Chọn **(B) tất cả ở cms-service** — `internal/api/dashboard_handler.go`.

**Lý do**:
- cms-service đã là control plane (CRUD + dashboards). FE đã có axios client (`cmsApi`) wired ở `services/api.ts`.
- Worker không nên expose HTTP public — đang sau internal network với rate-limited admin-api riêng (Phase F1).
- Cms-service đã có auth middleware → reuse.
- Timeline endpoint chỉ proxy Prometheus query — không phụ thuộc state worker.

**Trade-off**: cms-service phải biết Prometheus URL + kafka-connect URL → thêm config. Mitigate: config có sẵn pattern (`probes/*.go` đã đọc env tương tự).

---

## ADR-003 — Snapshot active list đọc từ **Prometheus**, không gọi worker REST

**Context**: cần `GET /api/v1/dashboard/snapshot/active`. Hai nguồn:
- (A) Worker expose `/api/v1/snapshot/active` đọc in-memory state map.
- (B) cms query Prometheus với metric `cdc_snapshot_progress_percent`.

**Decision**: Bắt đầu với **(B)**, fallback (A) nếu cardinality vượt threshold.

**Lý do**:
- (B) zero new endpoint worker → giảm coupling.
- snapshot_id labels đã có per-progress metric → list = `group by snapshot_id`.
- Prometheus đã được scrape worker (giả định prod sẽ wire — audit P0-3).

**Trade-off**:
- "Pending queue" KHÔNG có metric (chưa run = chưa emit). Mitigate: thêm metric `cdc_snapshot_pending_count{snapshot_id, table}` ở worker khi enqueue.
- Latency Prom query ~50-200ms. OK với poll 5s.

**Re-evaluate trigger**: nếu thấy "active list" delay > 10s → switch sang (A).

---

## ADR-004 — Polling thay vì WebSocket / Server-Sent Events

**Context**: Dashboard real-time có 3 tab polling 5s. Có thể chọn:
- (A) HTTP polling (như hiện tại react-query refetchInterval).
- (B) WebSocket bidirectional.
- (C) SSE (one-way push từ server).

**Decision**: **(A) HTTP polling** ở Phase 1. Reserve (C) cho Phase 2 nếu cardinality user > 100.

**Lý do**:
- 50 operator × 12 req/min/operator × 5 endpoint = ~3000 req/min — trong khả năng cms-service (đã có CQRS read repo).
- WebSocket cần infra (sticky session, reconnect logic, heartbeat) → complexity ↑↑.
- Cache TTL 10s ở R-BE-8 đã giảm load Prometheus.
- SSE dễ thêm sau (tương thích với react-query polling).

**Trade-off**: chart cập nhật max trễ 5s. Chấp nhận được cho operator UX (TTC widget có thể "spike" → quan trọng là không miss alert ≤ 1 phút).

---

## ADR-005 — Trace ID propagation qua DB column, không qua header

**Context**: cần link DLQ row → OTel trace ở SigNoz. Có thể chọn:
- (A) Persist `_otel_trace_id` vào table `failed_sync_logs`.
- (B) Embed trace_id vào payload JSON.
- (C) Đứng riêng table `dlq_trace_map`.

**Decision**: **(A) ADD COLUMN**.

**Lý do**:
- Index được trên trace_id → query nhanh.
- Schema rõ ràng, type-safe.
- (B) phá vỡ PII-mask flow (trace_id không phải PII nhưng trộn vào payload phức tạp scrubber).
- (C) thêm 1 JOIN không cần thiết.

**Trade-off**: cần migration. Mitigate: `ADD COLUMN IF NOT EXISTS` + index CONCURRENTLY → không lock prod (PG ≥ 11).

**Lesson tham chiếu**: L-listing-join-misses-identity-tier-column — JOIN scope quan trọng, tránh thêm JOIN không cần.

---

## ADR-006 — Vạch Nguy Hiểm logic ở FE (compute từ raw series)

**Context**: TTC formula có thể tính ở BE hoặc FE.

**Decision**: Tính ở **FE** từ raw series (`computeTtc` trong `utils/ttc.ts`).

**Lý do**:
- FE đã polling timeline series → reuse data, không round-trip thêm.
- Logic đơn giản (8 dòng), test dễ với vitest.
- BE side luôn có thể thêm sau nếu cần (vd alert backend) — không khóa decision.

**Trade-off**: 2 client → 2 implementation nếu thêm CLI client. Chấp nhận vì hiện chỉ có 1 client (FE).

---

## ADR-007 — Recharts `syncId` + fallback state lift

**Context**: 3 chart cần shared crosshair.

**Decision**: Dùng `syncId="dash-v2"` prop của recharts làm primary; fallback state lift (`hoverT`) nếu version recharts đang dùng không sync hoàn hảo.

**Lý do**: `syncId` không cần custom code; nhưng `<ReferenceLine>` đôi khi không sync nếu chart khác type → state lift backup là an toàn.

**Trade-off**: redundant code path. Mitigate: PoC sớm ở T-FE-14, drop fallback nếu syncId work.

---

## ADR-008 — Smoke gate `make smoke-metrics` block CI

**Context**: Cách ngăn lesson `metric-defined-but-never-set` tái phát.

**Decision**: Smoke gate trong CI: chạy worker với fixture, đợi 30s, scrape `/metrics`, assert N metric ≠ 0.

**Lý do**: 
- Đáp ứng lesson "Smoke test metric" + "Cross-ref grep".
- Cheap (~40s CI).
- Catch silent drift ngay (vd PR rm `.Set()`).

**Trade-off**: tăng 40s CI. Mitigate: chạy parallel với unit test.

**Alternative đã loại**: static grep cho `.Set()` — fragile (false positive với indirect call).

---

## ADR-009 — Dashboard V2 = endpoint mới `/api/v1/dashboard/*`, KHÔNG breaking `/api/system/health`

**Context**: SystemHealth.tsx hiện gọi `/api/system/health`. Có thể:
- (A) Extend `/api/system/health` thêm field.
- (B) Endpoint mới `/api/v1/dashboard/*`.

**Decision**: **(B)** — endpoint mới.

**Lý do**:
- Cardinality response khác nhau (15m timeline khác monthly snapshot).
- Cache strategy khác (10s vs 30s).
- Backward compat: `/api/system/health` không bị breaking.
- Future deprecation path rõ ràng.

**Trade-off**: code duplicate aggregator. Chấp nhận — sau khi V2 stable có thể deprecate V1.

---

## Tổng kết

| ADR | Quyết định ngắn | Rủi ro chính |
|-----|-----------------|--------------|
| 001 | Sửa worker + giữ kafka-exporter | Lock contention metric |
| 002 | Aggregator ở cms-service | Config thêm |
| 003 | Snapshot list từ Prom (fallback REST) | Pending queue ko có metric → cần thêm metric mới |
| 004 | HTTP polling 5s | Tăng load ở scale > 100 user |
| 005 | DB column cho trace_id | Migration prod |
| 006 | TTC compute FE | Duplicate logic nếu thêm client |
| 007 | Recharts syncId + fallback | Bug version-specific |
| 008 | Smoke gate CI | +40s CI |
| 009 | Endpoint mới Dashboard V2 | Aggregator code duplicate tạm |
