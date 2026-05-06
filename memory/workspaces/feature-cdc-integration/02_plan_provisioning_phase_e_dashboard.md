# Phase E — Provisioning Live Dashboard (Design)

**Workspace**: feature-cdc-integration
**Phase**: provisioning_mode / E (Manager UI)
**Status**: DESIGN — chờ Phase D done + Architect duyệt UX scope.
**Goal**: Manager nhìn State Machine của tất cả sources nhảy LIVE, không F5 thủ công.

## 1. User stories

- **US-E1**: Manager mở 1 màn hình duy nhất, thấy danh sách sources với state hiện tại (color-coded), mode (auto/manual), thời gian đã ở state đó.
- **US-E2**: Click 1 source → drawer mở ra, vẽ state machine 13 nodes, highlight node hiện tại + edges đã đi qua + animate edge đang chạy (pending state).
- **US-E3**: Step log timeline reverse-chronological, hiển thị từng entry với from→to, actor, correlation_id, error nếu có.
- **US-E4**: Manager click button Advance/Pause/Resume/Retry/Archive ngay trong drawer (chỉ visible nếu role=ops-admin).
- **US-E5**: Khi worker emit `step_completed`, dashboard tự update trong < 2s (không F5).
- **US-E6**: Filter: state, mode, search source_name, range thời gian.
- **US-E7**: Aggregate widget: số source ở mỗi state, biểu đồ cột.

## 2. Architecture (3-tier)

```
┌─────────────────────────────────────────────────────┐
│ Browser (FE)                                        │
│   ProvisioningDashboard.tsx                         │
│   ┌─ Sources table (TanStack)                       │
│   ├─ State machine SVG (Cytoscape.js / D3)          │
│   ├─ Timeline (step_log)                            │
│   └─ SSE EventSource → tự refresh                   │
└──────────────────┬──────────────────────────────────┘
                   │ HTTPS
                   │  GET  /api/v1/cms/provisioning/dashboard?state=...&mode=...
                   │  GET  /api/v1/cms/sources/:id/provisioning      (Phase C đã có)
                   │  GET  /api/v1/cms/provisioning/stream  (SSE)    (Phase E NEW)
                   ▼
┌─────────────────────────────────────────────────────┐
│ CMS (cdc-cms-service)                               │
│   ProvisioningDashboardHandler (NEW)                │
│   ├─ ListSources(filter) — paginate + aggregate     │
│   ├─ Stream(SSE) — relay NATS events                │
│   └─ subscribe `cdc.evt.provisioning.>` 1 lần boot  │
└──────────────────┬──────────────────────────────────┘
                   │ NATS
                   │  cdc.evt.provisioning.step_completed   (Phase D)
                   │  cdc.evt.provisioning.state_changed    (Phase E NEW)
                   ▼
┌─────────────────────────────────────────────────────┐
│ Worker + CMS Orchestrator                           │
│   Phase B/D đã emit step_completed.                 │
│   Phase E thêm: orchestrator emit state_changed     │
│   sau mỗi CAS thành công (Advance/Pause/Resume/...).│
└─────────────────────────────────────────────────────┘
```

## 3. Backend API (CMS — Phase E NEW)

### 3.1 GET `/api/v1/cms/provisioning/dashboard`
Aggregate + list combined.
```json
{
  "summary": {
    "total": 142,
    "by_state": { "draft": 5, "running": 100, "paused": 8, "failed": 12, ... },
    "by_mode": { "auto": 90, "manual": 52 }
  },
  "rows": [
    {
      "source_id": 18,
      "source_object_name": "phaseC_curl_test",
      "state": "running",
      "mode": "manual",
      "last_step_error": null,
      "updated_at": "2026-04-29T03:42:31Z",
      "time_in_state_seconds": 1245
    }
  ],
  "page": { "limit": 50, "offset": 0, "total": 142 }
}
```

### 3.2 GET `/api/v1/cms/provisioning/stream` (SSE)
Server-Sent Events. Mỗi event = 1 JSON line:
```
event: state_changed
data: {"source_id":18,"from":"draft","to":"shadow_pending","actor":"phaseC-test","correlation_id":"prov-18-shadow_bind-...","ts":"2026-04-29T03:37:51Z"}

event: step_completed
data: {"source_id":18,"step":"shadow_bind","success":true,"correlation_id":"...","ts":"..."}
```

CMS subscribe NATS `cdc.evt.provisioning.>` ở boot, fan out tới tất cả SSE connections. Pattern giống AlertsHandler (đã có Phase 6).

### 3.3 (Tận dụng) GET `/api/v1/cms/sources/:id/provisioning` — Phase C đã ship.

### 3.4 Auth
- Tất cả endpoints behind `JWTAuth` + `RequireRole("ops-admin", "admin", "viewer")`. Viewer được xem dashboard (read-only), nhưng action button bị hide ở FE (FE check role; backend vẫn enforce RequireOpsAdmin riêng cho 6 action endpoints Phase C).
- SSE endpoint: cùng auth. JWT pass qua `?token=` query param hoặc Authorization header.

## 4. Frontend (cdc-cms-fe React)

### 4.1 Component tree
```
<ProvisioningDashboardPage>
  <FilterBar />
  <SummaryCards />
  <SourcesTable rows={rows} onRowClick={setSelected} />
  <SourceDrawer source={selected}>
    <StateMachineGraph current={state} traversed={pastEdges} />
    <ActionButtons disabled={role !== 'ops-admin'} />
    <StepLogTimeline entries={stepLog} />
  </SourceDrawer>
</ProvisioningDashboardPage>
```

### 4.2 State machine viz (US-E2)
- Library: **Cytoscape.js** (graph layout đỡ tự code SVG; có sẵn animate-edge plugin).
- Nodes: 13 state (color theo group: draft/pending=gray, active=green, terminal=blue, failed=red, archived=dark).
- Edges: 4 forward (advance), 4 finalize (step_completed), pause/resume/retry/archive (side edges).
- Animate: edge đang chạy (state=PENDING) blink 1s/loop.

### 4.3 SSE wire (US-E5)
```ts
const sse = new EventSource(`/api/v1/cms/provisioning/stream?token=${jwt}`);
sse.addEventListener('state_changed', (e) => {
  const ev = JSON.parse(e.data);
  queryClient.setQueryData(['source', ev.source_id], (prev) => ({
    ...prev, provisioning_state: ev.to, updated_at: ev.ts
  }));
  queryClient.invalidateQueries(['provisioning-dashboard']);
});
```
Reconnect logic: `EventSource` auto-reconnect; chỉ thêm exponential backoff 1s→30s nếu liên tục fail.

### 4.4 Action button UX
- 6 actions: button confirmation modal (textarea bắt buộc nhập `reason ≥10 chars` cho destructive chain Phase C).
- Idempotency-Key: FE generate UUID v4 khi user mở modal, gửi cùng request. Nếu user re-click → cùng UUID → backend reject duplicate (Idempotency middleware đã có Phase 4).
- Hiển thị HTTP error rõ ràng: 422 → toast "Trạng thái hiện tại không cho phép action này", 409 → toast "Có người vừa thao tác cùng lúc, refresh và thử lại", 403 → toast "Cần quyền ops-admin".

## 5. Backend tasks chi tiết

### E-1. NEW emit `state_changed` từ orchestrator
File: `cdc-cms-service/internal/service/provisioning_orchestrator.go` (CMS port) + worker copy.

Sau mỗi `casUpdateState` thành công, publish:
```go
h.nats.Publish("cdc.evt.provisioning.state_changed", payloadJSON)
```
Payload: `{source_id, from, to, actor, correlation_id, ts, trace_id?, span_id?}`.

### E-2. NEW `ProvisioningDashboardHandler` (CMS)
File: `cdc-cms-service/internal/api/provisioning_dashboard_handler.go`

```go
type ProvisioningDashboardHandler struct {
    db     *gorm.DB
    nats   *nats.Conn
    logger *zap.Logger
    sseHub *SSEHub  // fan-out 1 NATS subscription → N HTTP SSE conns
}

func (h *Handler) ListDashboard(c *fiber.Ctx) error { ... }
func (h *Handler) Stream(c *fiber.Ctx) error { /* SSE */ }
```

SSEHub pattern (single NATS subscriber, multi HTTP fanout) giống AlertsHandler đã có.

### E-3. Subscribe NATS once ở boot
`server.go`: thêm `cdc.evt.provisioning.>` subscribe ở `New()` → push events vào SSEHub channel.

### E-4. SQL query cho dashboard
```sql
SELECT id, source_object_name, provisioning_state, provisioning_mode,
       last_step_error, updated_at,
       EXTRACT(EPOCH FROM (NOW() - updated_at))::int AS time_in_state
  FROM cdc_system.source_object_registry
 WHERE deleted_at IS NULL
   AND ($1::text IS NULL OR provisioning_state = $1)
   AND ($2::text IS NULL OR provisioning_mode = $2)
 ORDER BY updated_at DESC
 LIMIT $3 OFFSET $4
```
Aggregate 1 query riêng `GROUP BY provisioning_state, provisioning_mode` — cache Redis 5s để tránh đập DB khi nhiều dashboard mở cùng lúc.

## 6. Verification checklist

- [ ] E2E: open dashboard → see 142 sources → mở drawer → graph render đúng → kick `/advance` → drawer state nhảy < 2s không F5.
- [ ] Stress test: 50 dashboard tabs cùng mở, worker emit 100 events/min → mỗi tab vẫn nhận events, p99 latency < 3s.
- [ ] Auth: viewer JWT → list OK, action button disabled FE + backend reject 403.
- [ ] SSE reconnect: disconnect network 30s, reconnect → resume stream, miss < 5 events (acceptable, FE re-fetch dashboard sau reconnect).
- [ ] Browser memory: 1h dashboard mở → < 200MB heap (clear step_log entries cũ hơn 100).

## 7. Out of scope (defer)

- Multi-tenant filter (chỉ ops-admin xem all, không split theo team) — Phase F.
- Historical replay (thấy state machine nhảy 1 ngày trước) — cần TimescaleDB; Phase G.
- Mobile responsive — desktop only Phase E.
- Cytoscape.js có thể swap sang reactflow nếu team đã quen.

## 8. Risk register

| Risk | Mitigation |
|---|---|
| SSE qua reverse proxy buffering (nginx) | Set `proxy_buffering off` + `Cache-Control: no-cache` + `X-Accel-Buffering: no` |
| 100+ dashboard concurrent → CMS bottleneck NATS subscribe | SSEHub single subscriber, fan-out trong process; CMS scale horizontal nếu cần |
| Cytoscape rerender lag với 13 nodes + animate | Memoize layout, chỉ rerender khi state đổi |
| FE/BE state desync sau SSE drop | Sau reconnect, FE invalidate React Query → re-fetch dashboard fresh |

## 9. Files modified/created

| Path | Action |
|---|---|
| `cdc-cms-service/internal/api/provisioning_dashboard_handler.go` | NEW |
| `cdc-cms-service/internal/api/sse_hub.go` | NEW (hoặc reuse existing pattern) |
| `cdc-cms-service/internal/service/provisioning_orchestrator.go` | EDIT (emit state_changed) |
| `cdc-cms-service/internal/server/server.go` | EDIT (subscribe + handler init) |
| `cdc-cms-service/internal/router/router.go` | EDIT (2 routes mới + auth) |
| `cdc-cms-fe/src/pages/ProvisioningDashboard/*` | NEW (5 components) |
| `cdc-cms-fe/src/lib/sse-client.ts` | NEW (EventSource wrapper) |
| Worker `provisioning_orchestrator.go` | EDIT (emit state_changed song song) |

## 10. Execution order

Phase E khởi động SAU khi Phase D ship + smoke E2E auto-loop chạy ổn 24h.
1. E-1 (emit state_changed) — backend foundation.
2. E-2 + E-3 + E-4 (CMS dashboard handler + SSE).
3. FE shell + table.
4. FE state machine graph (Cytoscape).
5. FE SSE wire + action buttons.
6. Stress test + auth test.
