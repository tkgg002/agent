# 01_requirements_frontend — Dashboard V2 Frontend (cdc-cms-web)

> **Phase**: frontend (task riêng — chạy SAU/SONG SONG backend)
> **Track**: `cdc-cms-web` (React + TypeScript + Vite + AntD)
> **Output**: 1 page mới `DashboardV2.tsx` + 3 tab component + 4 widget component + 1 hook + 1 service.

---

## R-FE-0. Phạm vi & nguyên tắc UX

- Page nằm dưới menu hiện có; route `/dashboard-v2` (tách khỏi `/dashboard` cũ — backward compatibility).
- 3 tab (`Snapshot Commander` / `Streaming Real-time` / `DLQ & Schema Drift`) — chuyển tab KHÔNG mất state.
- Polling refresh:
  - Tab 1 (Snapshot): 5s
  - Tab 2 (Streaming): 5s (chart) + 5s (TTC widget)
  - Tab 3 (DLQ/Drift): 30s
- AntD ≥ v5, Recharts cho chart, dayjs cho time format. Hiện đã có (kiểm tra `package.json`).
- I18n: text VN-first, EN fallback (giữ pattern `SystemHealth.tsx`).
- Mobile: defer (Phase 2).

---

## R-FE-1. Container `DashboardV2.tsx`

**Mục đích**: layout + tab routing + shared header (auto-refresh toggle, last-updated timestamp).

**Spec**:
- AntD `<Tabs>` với 3 `<TabPane>`.
- Header: `<Statistic>` "Last updated: HH:mm:ss" + toggle "Auto refresh".
- Trên header: 1 banner critical alert nếu `TtcStatus === "red_blink"` (cross-cutting cảnh báo).
- URL state: `?tab=streaming` để deep-link.

**DoD**: chuyển tab → URL update, F5 → tab giữ nguyên.

---

## R-FE-2. Tab 1 `SnapshotCommanderTab.tsx`

**Bao gồm**:
1. **Stat row** (top): Active slots `X/4`, Pending queue `N`, Throughput total `MB/s`, Avg ETA.
2. **Active snapshots table** (AntD `<Table>`):
   - Cột: Snapshot ID (rút gọn 8 ký tự, tooltip full), Source table, Progress (`<Progress percent={...} />`), Throughput MB/s, ETA (human format), Started, Actions.
   - Actions: `<Button>Prioritize</Button>` (chỉ enable cho pending), `<Button>View Trace</Button>` (mở SigNoz).
3. **Pending queue** (collapse panel): list snapshot pending với badge `queued for Xs`.

**Data source**: `GET /api/v1/dashboard/snapshot/active` (R-BE-9), poll 5s.

**Prioritize action**: confirm modal → `POST /api/v1/dashboard/snapshot/:id/prioritize` body `{ priority: 100 }`.

**DoD**: 
- Kick 1 snapshot từ backend → xuất hiện trong table < 10s.
- Progress bar tăng dần.
- Prioritize button hiển thị correctly cho pending row.

---

## R-FE-3. Tab 2 `StreamingRealtimeTab.tsx`

**Bao gồm**:
1. **TTC Widget** (top, full width) — chi tiết R-FE-5.
2. **Stream Expiry Pair Widget** — chi tiết R-FE-6.
3. **Unified Crosshair Chart** (3 chart stacked) — chi tiết R-FE-7.
4. **Reconciliation status card**: cuối page, hiển thị `recon_last_success_timestamp` per table + drift count → tag color (green/yellow/red).

**Data source**:
- Widget TTC + Stream Expiry: poll 5s `GET /api/v1/dashboard/timeline?range=1m&step=15s` (lấy 4 điểm gần nhất tính rate).
- Chart: poll 5s `GET /api/v1/dashboard/timeline?range=15m&step=15s`.
- Recon: tận dụng endpoint hiện có `/api/system/health` section `reconciliation`.

**DoD**: 
- TTC widget thay đổi màu khi rate đảo chiều.
- Hover chart → 3 chart đồng bộ cursor (vertical line) + tooltip cùng giá trị t.

---

## R-FE-4. Tab 3 `DlqDriftTab.tsx`

**Bao gồm**:
1. **Sub-tab inner** (AntD nested tab hoặc 2 Card):
   - Card "DLQ Recent" — list 50 DLQ message gần nhất.
   - Card "Schema Drift Recent" — list 50 pending_field gần nhất.
2. **DLQ row** click → mở `<Modal>` viewer hiển thị:
   - Trace ID + link SigNoz/Jaeger (configurable env `VITE_SIGNOZ_BASE_URL`).
   - Payload masked (JSON viewer — dùng `react-json-view` hoặc syntax-highlight code block).
   - Error stack.
   - Retry count + "Retry now" button (call existing retry endpoint, defer if not exists).
3. **Drift row** click → modal show: field name, sample values seen (mongo), proposal status, "Approve" deep-link to existing SchemaProposals page.

**Data source**: 
- `GET /api/v1/dashboard/dlq/recent?limit=50&since=1h` (R-BE-11)
- `GET /api/v1/dashboard/drift/recent?limit=50` (R-BE-12)

**DoD**: 1 DLQ row → modal show payload + trace link mở đúng SigNoz tab mới.

---

## R-FE-5. Widget `TtcWidget.tsx` — "Vạch Nguy Hiểm"

**Logic**:
```
ingestRate  = series.ingest_rate.last
consumeRate = series.consume_rate.last
currentLag  = series.consumer_lag.last
netRate     = consumeRate - ingestRate

if netRate <= 0:
    state = "cannot_catch_up"   // red + blink
    label = "Không thể bắt kịp — lag đang tăng"
else:
    ttcSeconds = currentLag / netRate
    if ttcSeconds <= 30*60:    state = "critical"  // red
    elif ttcSeconds <= 2*3600: state = "warning"   // yellow
    else:                       state = "ok"       // green
    label = humanize(ttcSeconds)  // "1h 23m" / "45m"
```

**Visual**:
- Card với background gradient theo state.
- Big number: TTC trong human format.
- Sub-line: `Net rate: -12.4 msgs/s` (đỏ nếu âm).
- Blink animation: CSS keyframes `@keyframes blink { 50% { opacity: 0.3 } }` chỉ khi `red_blink`.

**DoD**: mock 4 state khác nhau → render đúng màu + label.

---

## R-FE-6. Widget Stream Expiry Pair (composed component)

**Mục đích**: pair "Current Lag" vs "Stream Expiry" — operator thấy "có còn cứu được không".

**Logic**:
```
streamExpiry = healthPipeline.kafka_retention_ms / 1000   // s
currentLag   = series.consumer_lag.last (đổi sang time-distance qua ingest_rate)

remainingS = streamExpiry - currentLagTime
if remainingS <= 30*60: blink red
elif remainingS <= 2h:  yellow
else: green
```

**Visual**: 2 column statistic, divider giữa.

**DoD**: render đúng tag color theo threshold.

---

## R-FE-7. Component `UnifiedCrosshairChart.tsx` — **Block B3**

**Mục đích**: 3 chart stacked vertically (Ingest Rate / Consume Rate / Lag), share X-axis (time) và hover crosshair đồng bộ.

**Spec kỹ thuật**:
- Dùng Recharts `<LineChart>` × 3 trong 1 `<ResponsiveContainer>` stacked column.
- Shared state `activeIndex: number | null`. Mỗi chart bắn `onMouseMove` → set global → 3 chart render `<ReferenceLine x={...} />` cùng X.
- Tooltip duy nhất hiển thị giữa (tách `<Tooltip content={...} />` custom render).

**DoD**: hover chart 1 → chart 2 + 3 có line cùng vị trí + tooltip cùng timestamp.

---

## R-FE-8. Hook `useDashboard.ts`

**Mục đích**: gom 5 endpoint mới.

**Export**:
```ts
export function useDashboardTimeline(range: string, step: string) { ... }
export function useDashboardSnapshot() { ... }
export function usePrioritizeSnapshot() { ... }   // mutation
export function useDashboardDlq(limit: number, since: string) { ... }
export function useDashboardDrift(limit: number) { ... }
```

Mỗi hook dùng `useQuery`/`useMutation` từ `@tanstack/react-query` (đã có).

**DoD**: unit test mỗi hook với MSW mock.

---

## R-FE-9. Service `services/dashboard.ts`

**Mục đích**: API client wrap axios cmsApi.

**Spec**: 5 function tương ứng 5 endpoint. Type-safe response types (interface trong `src/types/dashboard.ts`).

**DoD**: TS compile pass; type check không có `any`.

---

## R-FE-10. Routing + Menu

**File**: `src/App.tsx` (hoặc router config tương đương).

**Spec**:
- Thêm `<Route path="/dashboard-v2" element={<DashboardV2 />} />`.
- Menu item "Dashboard V2" (icon `DashboardOutlined`) — đặt ngay dưới "Dashboard" cũ.
- Highlight tab active qua URL.

**DoD**: click menu → navigate; URL deep-link `?tab=streaming` mở đúng tab.

---

## R-FE-11. Env config

**Spec**:
- Thêm `VITE_SIGNOZ_BASE_URL` vào `.env.example`. Default `http://localhost:3301`.
- Function `buildSignozUrl(traceID)` trong utils.

**DoD**: button "View Trace" mở đúng URL `<base>/trace/<id>`.

---

## R-FE-12. Error handling + Empty states

**Spec**:
- Mỗi data fetch: dùng `<QueryErrorBoundary>` (đã có).
- Empty: AntD `<Empty>` với text "Chưa có dữ liệu — đợi worker emit metric đầu tiên".
- Loading: AntD `<Skeleton>` thay vì spinner full-page.

**DoD**: tắt backend → page render empty + retry button, KHÔNG crash.

---

## R-FE-13. Accessibility & UX guards

- Blink animation: tôn trọng `prefers-reduced-motion` (CSS media query).
- Color không phải tín hiệu duy nhất: kèm icon (`<ExclamationCircleOutlined>` đỏ) + text.
- Modal payload: max-height 70vh, scroll inside.

**DoD**: bật `Reduce Motion` OS → animation tắt; screen-reader phát text trạng thái.

---

## Tóm tắt deliverable Frontend

| ID | Type | File | Effort |
|----|------|------|--------|
| R-FE-1 | container page | `pages/DashboardV2.tsx` | 1.5h |
| R-FE-2..4 | 3 tab component | `components/dashboard/{Snapshot,Streaming,DlqDrift}Tab.tsx` | 6h |
| R-FE-5..7 | 3 widget | `components/dashboard/{TtcWidget,StreamExpiryPair,UnifiedCrosshairChart}.tsx` | 5h |
| R-FE-8 | hook | `hooks/useDashboard.ts` | 1h |
| R-FE-9 | service | `services/dashboard.ts` + types | 1h |
| R-FE-10..11 | route + env | `App.tsx`, `.env.example`, utils | 0.5h |
| R-FE-12..13 | polish | inline | 1h |
| **Total** | | | **~16h** |

---

## Dependencies + risk

- **Phụ thuộc backend**: tất cả endpoint phải sẵn sàng trước khi merge FE. Mock layer (`msw`) dùng để dev song song.
- **Risk-1**: Recharts không có built-in synced cursor — phải tự implement state lift. Mitigate: viết unit test riêng cho `UnifiedCrosshairChart`.
- **Risk-2**: Polling 5s × 3 tab = 18 req/min/user. Với 50 operator → 900 req/min. Cache layer (R-BE-8 in-memory 10s) phải có.
- **Risk-3**: SigNoz URL format khác giữa local và prod. Mitigate: env config + test cả 2.
