# 09_tasks_solution_frontend — Technical Solutions (Frontend)

> Bản patch cuối cùng + commands + expected output cho từng task T-FE-*.
> File này LÀ artifact plan; code chỉ là demo — Muscle FE responsable cho actual edits.

---

## T-FE-01 — Types

**File mới**: `cdc-cms-web/src/types/dashboard.ts` (toàn bộ Section 1 của `03_implementation_frontend.md`).

**Commands**: `npm run typecheck`.

---

## T-FE-02 — Service

**File mới**: `cdc-cms-web/src/services/dashboard.ts` (Section 2 impl). Reuse `cmsApi` từ `./api`.

**Commands**: `npm run lint && npm run typecheck`.

---

## T-FE-03 — Hook

**File mới**: `cdc-cms-web/src/hooks/useDashboard.ts` (Section 3 impl).

**Verify**: 
```
npm run typecheck
# import 5 hook trong 1 component test, MSW mock 5 endpoint → 5 hook resolve OK
```

---

## T-FE-04 — Route + menu

**Patch** `src/App.tsx` (Muscle locate router định nghĩa):

```tsx
import DashboardV2 from './pages/DashboardV2'
// ... existing routes
<Route path="/dashboard-v2" element={<DashboardV2 />} />
```

Sider menu (Muscle locate menu items — pattern current `Dashboard`):
```tsx
{
  key: '/dashboard-v2',
  icon: <DashboardOutlined />,
  label: <Link to="/dashboard-v2">Dashboard V2</Link>,
},
```

**Verify**: `npm run dev` → click menu navigate to `/dashboard-v2`.

---

## T-FE-05 — Skeleton container

**File mới**: `src/pages/DashboardV2.tsx`. Stub minimal:
```tsx
import { Tabs, Typography } from 'antd'
import { useSearchParams } from 'react-router-dom'
const { Title } = Typography
export default function DashboardV2() {
  const [params, setParams] = useSearchParams()
  const tab = params.get('tab') || 'snapshot'
  return (
    <>
      <Title level={3}>CDC Dashboard V2</Title>
      <Tabs
        activeKey={tab}
        onChange={(k) => setParams({ tab: k })}
        items={[
          { key: 'snapshot',  label: 'Snapshot Commander',  children: <div>TODO</div> },
          { key: 'streaming', label: 'Streaming Real-time', children: <div>TODO</div> },
          { key: 'dlq',       label: 'DLQ & Schema Drift',  children: <div>TODO</div> },
        ]}
      />
    </>
  )
}
```

**Verify**: 3 tab clickable; F5 với `?tab=streaming` giữ tab.

---

## T-FE-06 — Env config

**Patch** `.env.example`:
```
VITE_SIGNOZ_BASE_URL=http://localhost:3301
```

**File mới** `src/utils/signoz.ts`:
```ts
const SIGNOZ_BASE = import.meta.env.VITE_SIGNOZ_BASE_URL || 'http://localhost:3301'
export function buildSignozUrl(traceID?: string | null): string | null {
  if (!traceID) return null
  return `${SIGNOZ_BASE.replace(/\/$/, '')}/trace/${encodeURIComponent(traceID)}`
}
```

**Unit test** `src/utils/__tests__/signoz.test.ts`:
```ts
import { describe, it, expect } from 'vitest'
import { buildSignozUrl } from '../signoz'
describe('buildSignozUrl', () => {
  it('returns null when traceID falsy', () => {
    expect(buildSignozUrl(null)).toBeNull()
    expect(buildSignozUrl(undefined)).toBeNull()
    expect(buildSignozUrl('')).toBeNull()
  })
  it('builds URL with default base', () => {
    expect(buildSignozUrl('abc123')).toMatch(/\/trace\/abc123$/)
  })
})
```

---

## T-FE-07..11 — Tab 1 Snapshot

**File mới**: `src/components/dashboard/SnapshotCommanderTab.tsx` (Section 7 impl).

**Mount trong DashboardV2.tsx** — replace `<div>TODO</div>` cho `key: 'snapshot'` bằng `<SnapshotCommanderTab />`.

**Verify**:
- BE chưa có endpoint → empty + skeleton.
- BE có 1 active snapshot → table show progress bar advancing.
- Click "Prioritize" → modal → confirm → POST → list refetch.

---

## T-FE-12 — TtcWidget

**File mới**: `src/components/dashboard/TtcWidget.tsx` (Section 5 impl) + `TtcWidget.css`.

**File mới**: `src/utils/ttc.ts` (Section 4 impl).

**Unit test** `src/utils/__tests__/ttc.test.ts` (Section 4 unit test).

**Verify**: 
```
npm run test -- ttc
```
Expected: 4 case PASS.

---

## T-FE-13 — StreamExpiryPair

**File mới**: `src/components/dashboard/StreamExpiryPair.tsx`. Snippet:
```tsx
import { Card, Col, Row, Statistic, Tag } from 'antd'
import { humanizeSeconds } from '../../utils/ttc'

interface Props {
  health: any  // SystemHealthSnapshot from useSystemHealth
  ttc: any | null
}

export function StreamExpiryPair({ health, ttc }: Props) {
  const expirySec = Number(health?.sections?.pipeline?.data?.kafka_retention_seconds ?? 0)
  const lag = Number(ttc?.currentLag ?? 0)
  const ingest = Number(ttc?.ingestRate ?? 0)
  // Estimate lag-in-time = lag / ingest
  const lagSeconds = ingest > 0 ? lag / ingest : 0
  const remaining = expirySec - lagSeconds

  let tagColor = 'green'; let label = 'OK'
  if (remaining <= 30 * 60) { tagColor = 'red'; label = 'CRITICAL' }
  else if (remaining <= 2 * 3600) { tagColor = 'orange'; label = 'WARNING' }

  return (
    <Card title="Stream Expiry">
      <Row gutter={16}>
        <Col span={12}><Statistic title="Current Lag (time)" value={humanizeSeconds(lagSeconds)} /></Col>
        <Col span={12}><Statistic title="Remaining before expiry"
           value={humanizeSeconds(remaining)}
           suffix={<Tag color={tagColor}>{label}</Tag>} /></Col>
      </Row>
    </Card>
  )
}
```

---

## T-FE-14 — UnifiedCrosshairChart

**File mới**: `src/components/dashboard/UnifiedCrosshairChart.tsx` (Section 6 impl).

**PoC step**: 
1. Bật `syncId="dash-v2"` cho 3 chart.
2. Open Chrome → hover chart 1 → check chart 2 + 3 có line vertical sync KHÔNG.
3. Nếu OK: drop `hoverT` state lift code.
4. Nếu KHÔNG (do tooltip không sync): giữ state lift.

**Verify**: render test `UnifiedCrosshairChart.test.tsx`:
```tsx
import { render } from '@testing-library/react'
import { UnifiedCrosshairChart } from '../UnifiedCrosshairChart'

it('renders 3 charts', () => {
  const data = {
    range_start: '', range_end: '', step_seconds: 15,
    series: {
      ingest_rate:  [{ t: '2026-05-27T08:00:00Z', v: 10 }],
      consume_rate: [{ t: '2026-05-27T08:00:00Z', v: 11 }],
      consumer_lag: [{ t: '2026-05-27T08:00:00Z', v: 100 }],
    },
  }
  const { container } = render(<UnifiedCrosshairChart data={data} />)
  expect(container.querySelectorAll('.recharts-line').length).toBe(3)
})
```

---

## T-FE-15 — Reconciliation card

**Patch** `StreamingRealtimeTab.tsx` (Section 8 impl) — uncomment reconciliation block, render từ `health.sections.reconciliation.data`.

---

## T-FE-16 — DLQ + Drift list

**File mới**: `src/components/dashboard/DlqDriftTab.tsx` (Section 9 impl).

**Verify**: 5 mock DLQ row → 5 List.Item render; click → modal open.

---

## T-FE-17 — PayloadViewerModal

**File mới**: `src/components/dashboard/PayloadViewerModal.tsx` (Section 10 impl).

**Security note**: `JSON.stringify(item.payload_masked, null, 2)` → `<pre>` — chuỗi sẽ render as text, browser auto-escape. KHÔNG dùng `dangerouslySetInnerHTML`.

**Verify**: 1 DLQ row click → modal có button "View trace in SigNoz" → tab mới mở URL đúng format.

---

## T-FE-18 — Drift deep-link

**Patch** trong `DlqDriftTab.tsx`:
```tsx
import { useNavigate } from 'react-router-dom'
const navigate = useNavigate()
// in drift List.Item:
<List.Item onClick={() => navigate(`/schema-proposals?field=${encodeURIComponent(it.field)}`)} style={{cursor:'pointer'}}>
```

**SchemaProposals page enhancement**: defer — file riêng `09_tasks_solution_extras.md` nếu Muscle gặp blocker.

---

## T-FE-19 — Empty + Skeleton

**Patch** mọi nơi `useQuery` được dùng:
```tsx
const { data, isLoading, error } = useDashboardSnapshot()
if (isLoading) return <Skeleton active />
if (error) return <Alert type="error" message="Lỗi tải dữ liệu" />
if (!data?.active.length && !data?.pending.length) return <Empty description="Chưa có snapshot" />
```

---

## T-FE-20 — Reduced motion

**CSS** trong `TtcWidget.css`:
```css
@media (prefers-reduced-motion: reduce) {
  .ttc-blink {
    animation: none;
    outline: 2px solid #ff4d4f;
  }
}
```

**Verify**: macOS Settings → Accessibility → Display → Reduce motion ON → reload → blink stops, outline appears.

---

## T-FE-21 — I18n VN-first

Quét tất cả file dashboard/* — replace EN strings bằng VN. Pattern:
- "Active slots" → "Slot đang chạy"
- "Pending queue" → "Hàng đợi"
- "Throughput" → "Thông lượng"
- "ETA" → "Còn lại"
- "Prioritize" → "Ưu tiên"
- "View trace in SigNoz" → "Xem trace trên SigNoz"
- "Loading metrics…" → "Đang tải metric…"
- "No snapshot running" → "Không có snapshot đang chạy"
- "Cannot catch up" → "Không thể bắt kịp"

**Note**: nếu codebase đã có i18n library (`i18next` etc), thêm key vào file translation thay vì hardcode. Check `package.json`.

---

## T-FE-22 — Unit + render test

**Files**:
- `src/utils/__tests__/ttc.test.ts` (đã có ở T-FE-12)
- `src/utils/__tests__/signoz.test.ts` (đã có ở T-FE-06)
- `src/components/dashboard/__tests__/TtcWidget.test.tsx` — 4 snapshot render
- `src/components/dashboard/__tests__/UnifiedCrosshairChart.test.tsx` (đã có ở T-FE-14)
- `src/hooks/__tests__/useDashboard.test.ts` — MSW mock 5 endpoint

**Commands**:
```
npm run test                    # vitest
npm run test:coverage           # threshold > 70% dashboard/*
```

---

## Cross-cutting verification

```
npm run typecheck              # PASS
npm run lint                   # PASS
npm run build                  # PASS, bundle size delta
npm run test                   # PASS
```

Manual smoke (sau khi BE ready):
1. Mở Dashboard V2 → 3 tab navigable.
2. Kick snapshot từ BE (qua CMS endpoint) → Tab 1 update < 10s.
3. Tạo lag (chậm DB) → Tab 2 TTC chuyển vàng/đỏ.
4. Inject DLQ row → Tab 3 list update + payload modal mở + trace link work.

Sau khi tất cả task done:
1. APPEND `05_progress.md`.
2. Chạy `/security-agent` review (kiểm XSS qua JSON viewer + URL injection ở SigNoz link).
3. Bundle size check: nếu lazy import cần thiết, áp dụng `React.lazy(() => import('./PayloadViewerModal'))`.
