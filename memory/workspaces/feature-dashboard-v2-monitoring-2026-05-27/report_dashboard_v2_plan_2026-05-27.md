# Báo cáo: Plan Dashboard V2 (Monitoring + FE) — 2026-05-27

> **Workspace**: `agent/memory/workspaces/feature-dashboard-v2-monitoring-2026-05-27`
> **Status**: 🟡 **Plan-only Active** (chờ user verb để Muscle thực thi)
> **Authored by**: Brain (Antigravity:claude-opus-4-7)
> **Code Touched (Brain)**: KHÔNG (tuân thủ §12 — Brain Code Prohibition)

---

## TL;DR

Đã hoàn thành **bộ 11 tài liệu Workspace** để khoá scope việc bổ sung monitoring backend + xây Dashboard V2 frontend, **tách thành 2 plan độc lập có thể chạy song song**. Tổng effort ước tính **~32.5h Muscle** (16.5h BE + 16h FE).

**Cốt lõi**:
- Backend đáp ứng **4 diagnostic block** + lấp **7 gap** từ audit `2026-05-26`.
- Frontend triển khai **3 tab Dashboard V2** với 4 widget đặc biệt (TTC "Vạch Nguy Hiểm", Stream Expiry pair, Unified Crosshair chart, JSON Payload viewer).
- **9 ADR** đã ký để khoá decision (avoid bikeshedding).
- **Smoke gate CI** để ngăn lesson `metric-defined-but-never-set` tái phát.

---

## Files đã tạo

| # | File | Mục đích |
|---|------|---------|
| 1 | `00_context.md` | Scope, 4 block, 7 gap mapping, dependencies workspace cũ |
| 2 | `01_requirements_backend.md` | 15 requirement R-BE-1..15 (metric/probe/API/migration) |
| 3 | `01_requirements_frontend.md` | 13 requirement R-FE-1..13 (page/tab/widget/hook) |
| 4 | `02_plan_backend.md` | Roadmap 5 phase B1..B5 + dependencies + verify gate |
| 5 | `02_plan_frontend.md` | Roadmap 5 phase F1..F5 + dependencies + verify gate |
| 6 | `03_implementation_backend.md` | Technical design 10 section + **code demo Go** (~600 dòng demo) |
| 7 | `03_implementation_frontend.md` | Technical design 13 section + **code demo TSX** (~400 dòng demo) |
| 8 | `04_decisions.md` | 9 ADR (worker-in-place, aggregator vị trí, polling vs WS, trace col, TTC FE, recharts syncId, smoke gate, V2 endpoint) |
| 9 | `05_progress.md` | Audit log APPEND-only (đã ghi 11 dòng cho session khởi tạo) |
| 10 | `08_tasks_backend.md` | Checklist 22 task T-BE-* (mỗi task có file:line + DoD) |
| 11 | `08_tasks_frontend.md` | Checklist 22 task T-FE-* (mỗi task có file + DoD) |
| 12 | `09_tasks_solution_backend.md` | Solution patches sẵn sàng Muscle apply (commands + expected output) |
| 13 | `09_tasks_solution_frontend.md` | Solution patches FE + i18n VN list |
| 14 | `report_dashboard_v2_plan_2026-05-27.md` | (file này) |

Tổng: **14 file** (gồm 1 report, 13 doc theo prefix § 7 GEMINI.md).

---

## Mapping Spec → Backend → Frontend

| Spec Block | Backend deliverable | Frontend deliverable |
|------------|---------------------|----------------------|
| **B1. Time-to-Live Countdown** | Wire `cdc_kafka_consumer_lag.Set()` (G1) + 2 gauge mới `cdc_ingest_rate_msgs_per_sec`, `cdc_consume_rate_msgs_per_sec`. RateMeter rolling 10s. Counter `cdc_kafka_transient_errors_total` (G2). sinkworker classify (G3). | `TtcWidget.tsx` 4 trạng thái (ok/warning/critical/cannot_catch_up). CSS blink + `prefers-reduced-motion`. Banner alert ở header khi `cannot_catch_up`. |
| **B2. Snapshot Mode Module** | 4 gauge mới `cdc_snapshot_{active_slots,progress_percent,throughput_mbps,eta_seconds}`. API `GET /snapshot/active`, `POST /snapshot/:id/prioritize` → NATS `cdc.cmd.snapshot.priority`. | `SnapshotCommanderTab.tsx` — stat row + table (Progress + Throughput + ETA + Prioritize button + ViewTrace). Pending queue collapse. |
| **B3. Unified Timeline Crosshair** | API `/timeline?range=15m&step=15s` proxy Prometheus `query_range`. Cache TTL 10s. Trả 3 series time-aligned. | `UnifiedCrosshairChart.tsx` 3 chart stacked + Recharts `syncId="dash-v2"` (fallback state lift). |
| **B4. In-memory Queue Health** | Probe mới `probes/debezium_queue.go` scrape Kafka Connect REST `/connectors/{name}/metrics` (queue-size, queue-total-capacity, write-rate). Threshold > 80% degraded, > 95% down. | Surface qua `useSystemHealth().sections.pipeline.debezium_queue` ở Tab 2 reconciliation card. |

---

## Lấp Gap Audit 2026-05-26

| Gap | Severity | Fix trong plan |
|-----|----------|----------------|
| G1: `cdc_kafka_consumer_lag` define-only | P0 | T-BE-03 wire `.Set()` + ticker 5s từ `reader.Stats()` |
| G2: Transient err không có counter | P1 | T-BE-04 + T-BE-05 — `cdc_kafka_transient_errors_total{component,error_class}` |
| G3: sinkworker không classify | P1 | T-BE-06 reuse classifier mới |
| G4: Không có broker partition health | P1 | T-BE-09 probe `debezium_queue` (gián tiếp), kafka_lag.go đã có per-topic |
| G5: Không có signal snapshot đang chạy | P0 | T-BE-07 + T-BE-08 — 4 snapshot metric |
| G6: Drift không có trace_id | P2 | (defer — drift chưa critical; trace cho DLQ là đủ scope) |
| G7: DLQ payload không show UI | P0 | T-BE-18..20 (trace_id col + producer) + R-FE-17 (PayloadViewerModal) |

→ Plan đóng **6/7 gap audit P0+P1**, defer G6 vào Phase 2 (drift detection chỉ là detection — schema_proposal flow vẫn còn track).

---

## Effort estimate chi tiết

### Backend (Phase B1..B5, ~16.5h)
| Phase | Effort | Mục tiêu |
|-------|--------|----------|
| B1 | 5h | Streaming metric foundation (G1..G5 fix) |
| B2 | 4h | Snapshot + Debezium queue probe |
| B3 | 5h | 5 endpoint Dashboard aggregator + cache |
| B4 | 2h | Trace correlation (migration + DLQ producer) |
| B5 | 1.5h | Smoke gate CI |

### Frontend (Phase F1..F5, ~16h)
| Phase | Effort | Mục tiêu |
|-------|--------|----------|
| F1 | 3h | Foundation (types, service, hook, route) |
| F2 | 4h | Tab 1 Snapshot Commander |
| F3 | 5h | Tab 2 Streaming + TTC widget + Crosshair |
| F4 | 3h | Tab 3 DLQ + Drift + PayloadViewer |
| F5 | 1h | Polish (a11y, i18n VN, test) |

**Có thể parallel**: 2 dev (1 BE + 1 FE) → realistic 2 ngày (16h) wall-time. Một mình → ~5 ngày.

---

## ADR Highlights

| ADR | Quyết định | Rủi ro |
|-----|-----------|--------|
| 001 | Sửa worker in-place (vs sidecar) | Lock contention nhỏ — mitigate per-topic mutex |
| 002 | Aggregator API ở cms-service (vs worker) | Thêm config Prom URL — pattern đã có |
| 003 | Snapshot list query từ Prometheus (vs worker REST) | Pending queue chưa có metric — cần thêm metric mới |
| 004 | HTTP polling 5s (vs WebSocket) | Defer SSE/WS sang Phase 2 nếu > 100 user |
| 005 | `_otel_trace_id` column trong `failed_sync_logs` (vs JSON embed) | Migration prod — `IF NOT EXISTS` + CONCURRENTLY index |
| 006 | TTC compute ở FE từ raw series | Duplicate logic nếu thêm client tương lai |
| 007 | Recharts `syncId` + fallback state lift | Bug version-specific — PoC sớm |
| 008 | Smoke gate `make smoke-metrics` block CI | +40s CI — chấp nhận |
| 009 | Endpoint mới `/api/v1/dashboard/*` (vs extend `/system/health`) | Duplicate aggregator tạm thời |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Prometheus HTTP API không có ở local dev | High | Medium | Fallback expose `/internal/metrics-json` qua `prometheus.Gatherer` |
| NATS subject `cdc.cmd.snapshot.priority` chưa có subscriber worker | Medium | High | Audit lesson L3100 conditional subscriber — log Warn + return 503 nếu sub count = 0 |
| Migration lock `failed_sync_logs` lâu | Low | Medium | `ADD COLUMN IF NOT EXISTS` (instant PG ≥ 11) + index CONCURRENTLY |
| Recharts `syncId` không sync ở version đang dùng | Medium | Low | PoC sớm ở T-FE-14, fallback state lift đã có |
| Polling load: 50 user × 12 req/min × 5 endpoint = 3000 req/min | Medium | Medium | Cache 10s ở R-BE-8 + react-query dedup |
| XSS qua DLQ payload JSON viewer | Medium | High | Render qua `<pre>` text content, KHÔNG `dangerouslySetInnerHTML`; `/security-agent` review T-FE-99 |
| Per-snapshot_id metric label cardinality vượt | Low | Low | Hard cap 4 concurrent + 100 pending; cleanup `DeleteLabelValues` khi done |

---

## Verify gate cho mỗi phase (Staff Engineer review)

Trước khi báo "Done" cho mỗi phase, Muscle PHẢI:

### Backend
- [ ] `go build ./...` PASS (worker + cms)
- [ ] `go vet ./...` PASS
- [ ] `go test ./...` PASS không regression
- [ ] `make smoke-metrics` PASS
- [ ] Manual curl 5 endpoint dashboard → 200 + JSON đúng schema
- [ ] Probe `debezium_queue` smoke local PASS (status=ok khi connector live)
- [ ] 1 DLQ row inject → trace_id chứa giá trị, signoz_url mở được
- [ ] Snapshot kick + tail metric → progress 0→100; cleanup OK
- [ ] `/security-agent` review PASS (theo §8)

### Frontend
- [ ] `npm run typecheck` PASS
- [ ] `npm run lint` PASS
- [ ] `npm run build` PASS, bundle delta < 100KB gz
- [ ] `npm run test` PASS coverage > 70% dashboard
- [ ] Manual 4 scenario smoke (xem `02_plan_frontend.md`)
- [ ] Lighthouse a11y > 90
- [ ] OS Reduce Motion ON → blink stops
- [ ] `/security-agent` review PASS (XSS payload + URL injection SigNoz)

---

## Quy trình Brain ↔ Muscle ↔ User (theo §1 + §12)

```
┌─────────────────────────────────────────────────────────────┐
│ Step 1: USER chốt verb                                       │
│   Options:                                                   │
│     • execute backend      → Muscle chạy Phase B1..B5        │
│     • execute frontend     → Muscle chạy Phase F1..F5 (mock) │
│     • execute both         → 2 Muscle parallel               │
│     • revise <section>     → Brain điều chỉnh plan trước     │
│     • defer                → Workspace ⏸ Paused              │
│     • model: opus|sonnet   → set Muscle model                │
├─────────────────────────────────────────────────────────────┤
│ Step 2: BRAIN delegate qua /muscle-execute hoặc /brain-      │
│         delegate, hand off `08_tasks_*.md` + `09_tasks_      │
│         solution_*.md` làm input.                            │
├─────────────────────────────────────────────────────────────┤
│ Step 3: MUSCLE chạy full-loop (§2):                          │
│   - Đọc workspace                                            │
│   - Apply patch theo solution doc                            │
│   - Verify gate per phase                                    │
│   - APPEND `05_progress.md` per task                         │
│   - Chạy `/security-agent` cuối                              │
├─────────────────────────────────────────────────────────────┤
│ Step 4: USER smoke test (E2E manual).                        │
│         Đồng ý → Brain cập nhật `active_plans.md` ✅ Done.    │
│         Không OK → Brain re-plan, append `05_progress.md`.   │
└─────────────────────────────────────────────────────────────┘
```

---

## Lesson references đã áp dụng

- **L-2026-05-26-metric-defined-but-never-set** → drive ADR-008 (smoke gate) + R-BE-15 (smoke binary).
- **L-2026-05-21 Kafka transient error LoadBalancer** → T-BE-01 extract classifier (đã có ở `kafka_consumer.go:1116`, gen hóa).
- **L-2026-05-21 RowsAffected accuracy** → T-BE-04 wire consume rate từ batch_buffer thay vì raw Kafka count.
- **L-2026-05-21 Log noise reduction** → log levels: Warn cho transient, Error cho fatal.
- **L-listing-join-misses-identity-tier-column** → R-BE-11 dashboard handler join scope rõ (failed_sync_logs đã có connection scope qua source label).
- **L-CDC-route-empty-silent-skip-2026-05-26** → đảm bảo Snapshot mode metric không silent-skip khi `routes==nil`.

---

## Verb chờ user

Sau khi đọc plan, user vui lòng chọn 1 trong các verb:

| Verb | Hành động |
|------|-----------|
| `execute backend` | Brain delegate Muscle chạy Phase B1..B5 (16.5h) |
| `execute frontend` | Brain delegate Muscle chạy Phase F1..F5 (16h, MSW mock layer) |
| `execute both` | 2 Muscle parallel, tổng wall-time ~16h |
| `revise backend <section>` | Brain điều chỉnh section yêu cầu (vd `revise backend snapshot-metrics` để hạ scope) |
| `revise frontend <section>` | Tương tự FE |
| `defer` | Workspace ⏸ Paused, append 05_progress |
| `model: opus | sonnet | haiku` | Set Muscle model trước khi execute |

---

## Governance compliance check (§14 Pre-flight)

- [x] §3 Plan & Verify — 9 ADR + verify gate cho mỗi phase.
- [x] §7 Workspace doc set — 13 file theo prefix bắt buộc.
- [x] §7 Mandatory Full Doc Set — phase backend + frontend mỗi cái có đủ 01/02/03/08/09.
- [x] §11 Memory file protection — KHÔNG overwrite lessons.md (chưa append; sẽ append sau khi thực thi nếu có lesson mới).
- [x] §12 Brain Code Prohibition — Brain KHÔNG edit `.go`/`.ts`. Tất cả code chỉ ở docs làm blueprint.
- [x] §13 Lesson abstract — không tạo lesson mới ở plan-only phase; sẽ ghi sau execute (nếu có).
- [x] §14 Pre-flight Check — quét lại 14 file → tất cả tồn tại vật lý ở `agent/memory/workspaces/feature-dashboard-v2-monitoring-2026-05-27/`.

---

## Liên hệ workspace cũ (cập nhật `active_plans.md`)

Sẽ APPEND row mới vào `agent/memory/global/active_plans.md`:
```
| feature-dashboard-v2-monitoring-2026-05-27 | Dashboard V2 monitoring + FE (4 block + 7 gap audit) | 🟡 Plan-only Active | 2026-05-27 |
```
