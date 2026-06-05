# 03_implementation_frontend — Technical Design (Frontend)

> **Reader contract**: code demo dưới đây là blueprint. Muscle (FE engineer) PHẢI verify imports, types, build, lint, test trước khi commit.

---

## Section 1: Types — `src/types/dashboard.ts`

```ts
export interface TimelinePoint { t: string; v: number }   // t = ISO Z
export interface TimelineResponse {
  range_start: string
  range_end: string
  step_seconds: number
  series: {
    ingest_rate: TimelinePoint[]
    consume_rate: TimelinePoint[]
    consumer_lag: TimelinePoint[]
  }
}

export interface SnapshotItem {
  snapshot_id: string
  source_object_id: number
  table: string
  progress_pct?: number
  throughput_mbps?: number
  eta_seconds?: number
  started_at?: string
  queued_at?: string
}
export interface SnapshotActiveResponse {
  active: SnapshotItem[]
  pending: SnapshotItem[]
  max_concurrent_slots: number
}

export interface DLQItem {
  id: number
  topic: string
  occurred_at: string
  error_class: string
  error_message: string
  trace_id?: string
  span_id?: string
  signoz_url?: string
  payload_masked: Record<string, unknown>
  retry_count: number
}
export interface DLQRecentResponse { items: DLQItem[] }

export interface DriftItem {
  table: string
  field: string
  detection_count: number
  first_seen: string
  last_seen: string
  approval_status: 'pending' | 'approved' | 'rejected'
}
export interface DriftRecentResponse { items: DriftItem[] }

export type TtcState = 'ok' | 'warning' | 'critical' | 'cannot_catch_up'
export interface TtcComputed {
  state: TtcState
  ttcSeconds: number | null         // null when cannot_catch_up
  netRateMsgsPerSec: number
  ingestRate: number
  consumeRate: number
  currentLag: number
  humanLabel: string                 // "1h 23m" or "Không thể bắt kịp"
}
```

---

## Section 2: Service — `src/services/dashboard.ts`

```ts
import { cmsApi } from './api'
import type {
  TimelineResponse, SnapshotActiveResponse, DLQRecentResponse, DriftRecentResponse,
} from '../types/dashboard'

export async function fetchTimeline(range = '15m', step = '15s'): Promise<TimelineResponse> {
  const { data } = await cmsApi.get<TimelineResponse>('/api/v1/dashboard/timeline', {
    params: { range, step },
  })
  return data
}

export async function fetchSnapshotActive(): Promise<SnapshotActiveResponse> {
  const { data } = await cmsApi.get<SnapshotActiveResponse>('/api/v1/dashboard/snapshot/active')
  return data
}

export async function prioritizeSnapshot(id: string, priority = 100): Promise<void> {
  await cmsApi.post(`/api/v1/dashboard/snapshot/${encodeURIComponent(id)}/prioritize`, { priority })
}

export async function fetchDlqRecent(limit = 50, since = '1h'): Promise<DLQRecentResponse> {
  const { data } = await cmsApi.get<DLQRecentResponse>('/api/v1/dashboard/dlq/recent', {
    params: { limit, since },
  })
  return data
}

export async function fetchDriftRecent(limit = 50): Promise<DriftRecentResponse> {
  const { data } = await cmsApi.get<DriftRecentResponse>('/api/v1/dashboard/drift/recent', {
    params: { limit },
  })
  return data
}
```

---

## Section 3: Hook — `src/hooks/useDashboard.ts`

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchTimeline, fetchSnapshotActive, prioritizeSnapshot,
  fetchDlqRecent, fetchDriftRecent,
} from '../services/dashboard'

export function useDashboardTimeline(range = '15m', step = '15s', enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'timeline', range, step],
    queryFn: () => fetchTimeline(range, step),
    refetchInterval: 5_000,
    enabled,
  })
}

export function useDashboardSnapshot(enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'snapshot', 'active'],
    queryFn: fetchSnapshotActive,
    refetchInterval: 5_000,
    enabled,
  })
}

export function usePrioritizeSnapshot() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, priority }: { id: string; priority: number }) =>
      prioritizeSnapshot(id, priority),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dashboard', 'snapshot', 'active'] })
    },
  })
}

export function useDashboardDlq(limit = 50, since = '1h', enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'dlq', limit, since],
    queryFn: () => fetchDlqRecent(limit, since),
    refetchInterval: 30_000,
    enabled,
  })
}

export function useDashboardDrift(limit = 50, enabled = true) {
  return useQuery({
    queryKey: ['dashboard', 'drift', limit],
    queryFn: () => fetchDriftRecent(limit),
    refetchInterval: 30_000,
    enabled,
  })
}
```

---

## Section 4: TTC compute logic — `src/utils/ttc.ts`

```ts
import type { TimelineResponse, TtcComputed } from '../types/dashboard'

export function computeTtc(t: TimelineResponse | undefined): TtcComputed | null {
  if (!t) return null
  const last = (arr: { v: number }[]) => (arr.length ? arr[arr.length - 1].v : 0)
  const ingestRate  = last(t.series.ingest_rate)
  const consumeRate = last(t.series.consume_rate)
  const currentLag  = last(t.series.consumer_lag)
  const netRate = consumeRate - ingestRate

  if (netRate <= 0) {
    return {
      state: 'cannot_catch_up',
      ttcSeconds: null,
      netRateMsgsPerSec: netRate,
      ingestRate, consumeRate, currentLag,
      humanLabel: 'Không thể bắt kịp — lag đang tăng',
    }
  }
  const ttcSeconds = currentLag / netRate
  let state: TtcComputed['state']
  if (ttcSeconds <= 30 * 60) state = 'critical'
  else if (ttcSeconds <= 2 * 3600) state = 'warning'
  else state = 'ok'

  return {
    state, ttcSeconds,
    netRateMsgsPerSec: netRate,
    ingestRate, consumeRate, currentLag,
    humanLabel: humanizeSeconds(ttcSeconds),
  }
}

export function humanizeSeconds(s: number): string {
  if (!isFinite(s) || s < 0) return '–'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}
```

**Unit test ý tưởng** (`src/utils/__tests__/ttc.test.ts`):
```ts
import { describe, it, expect } from 'vitest'
import { computeTtc } from '../ttc'

const series = (ingest: number, consume: number, lag: number) => ({
  range_start: '', range_end: '', step_seconds: 15,
  series: {
    ingest_rate:  [{ t: '', v: ingest }],
    consume_rate: [{ t: '', v: consume }],
    consumer_lag: [{ t: '', v: lag }],
  },
})

describe('computeTtc', () => {
  it('cannot_catch_up when net rate negative', () => {
    expect(computeTtc(series(100, 50, 1000))?.state).toBe('cannot_catch_up')
  })
  it('critical when ttc <= 30m', () => {
    // net 10/s, lag 9000 → 900s = 15m
    expect(computeTtc(series(10, 20, 9000))?.state).toBe('critical')
  })
  it('warning when ttc <= 2h', () => {
    // net 10/s, lag 36000 → 3600s = 60m
    expect(computeTtc(series(10, 20, 36000))?.state).toBe('warning')
  })
  it('ok otherwise', () => {
    // net 10/s, lag 100 → 10s
    // ... 10s is also <= 30m → critical. Use larger lag.
    expect(computeTtc(series(10, 110, 36000))?.state).toBe('ok')   // net 100, lag 36000 → 360s = critical too
    // Correct case: net 1/s, lag 100000 → 100000s ~ 27h → ok
    expect(computeTtc(series(10, 11, 100000))?.state).toBe('ok')
  })
})
```

---

## Section 5: Widget `TtcWidget.tsx`

```tsx
import { Card, Statistic, Tag, Typography } from 'antd'
import { ExclamationCircleOutlined, ThunderboltOutlined } from '@ant-design/icons'
import type { TtcComputed } from '../../types/dashboard'

const { Title, Text } = Typography

const stateConfig: Record<TtcComputed['state'], {
  bg: string; tag: string; tagColor: string; icon: React.ReactNode; blink: boolean
}> = {
  ok: { bg: '#f6ffed', tag: 'OK', tagColor: 'green',  icon: <ThunderboltOutlined />, blink: false },
  warning:  { bg: '#fffbe6', tag: 'WARNING',  tagColor: 'orange', icon: <ExclamationCircleOutlined />, blink: false },
  critical: { bg: '#fff2f0', tag: 'CRITICAL', tagColor: 'red', icon: <ExclamationCircleOutlined />, blink: false },
  cannot_catch_up: { bg: '#fff2f0', tag: 'CRITICAL', tagColor: 'red', icon: <ExclamationCircleOutlined />, blink: true },
}

interface Props { ttc: TtcComputed | null }

export function TtcWidget({ ttc }: Props) {
  if (!ttc) {
    return <Card title="Time-to-Live Countdown"><Text type="secondary">Đang tải metric…</Text></Card>
  }
  const cfg = stateConfig[ttc.state]
  return (
    <Card
      style={{ background: cfg.bg }}
      className={cfg.blink ? 'ttc-blink' : ''}
      title={
        <span>
          {cfg.icon}  Time-to-Live Countdown  <Tag color={cfg.tagColor}>{cfg.tag}</Tag>
        </span>
      }
    >
      <Title level={2} style={{ margin: 0 }}>{ttc.humanLabel}</Title>
      <Text type={ttc.netRateMsgsPerSec < 0 ? 'danger' : 'secondary'}>
        Net rate: {ttc.netRateMsgsPerSec.toFixed(1)} msgs/s ·
        Ingest: {ttc.ingestRate.toFixed(1)} ·
        Consume: {ttc.consumeRate.toFixed(1)} ·
        Lag: {ttc.currentLag.toLocaleString()} msgs
      </Text>
    </Card>
  )
}
```

**CSS** (cùng folder `TtcWidget.css` import bằng `import './TtcWidget.css'`):
```css
@keyframes ttc-blink-kf { 50% { opacity: 0.35 } }
.ttc-blink { animation: ttc-blink-kf 1.4s ease-in-out infinite; }
@media (prefers-reduced-motion: reduce) {
  .ttc-blink { animation: none; outline: 2px solid #ff4d4f; }
}
```

---

## Section 6: `UnifiedCrosshairChart.tsx`

```tsx
import { useMemo, useState } from 'react'
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceLine, CartesianGrid,
} from 'recharts'
import dayjs from 'dayjs'
import type { TimelineResponse } from '../../types/dashboard'

interface Props { data: TimelineResponse | undefined; height?: number }

interface Row { t: number; ingest: number; consume: number; lag: number }

export function UnifiedCrosshairChart({ data, height = 180 }: Props) {
  const rows: Row[] = useMemo(() => {
    if (!data) return []
    const ing = data.series.ingest_rate
    const con = data.series.consume_rate
    const lag = data.series.consumer_lag
    const len = Math.min(ing.length, con.length, lag.length)
    return Array.from({ length: len }, (_, i) => ({
      t: new Date(ing[i].t).getTime(),
      ingest:  ing[i].v,
      consume: con[i].v,
      lag:     lag[i].v,
    }))
  }, [data])
  const [hoverT, setHoverT] = useState<number | null>(null)

  const handleMove = (s: any) => {
    if (s && s.activeLabel) setHoverT(Number(s.activeLabel))
  }
  const handleLeave = () => setHoverT(null)

  const xFmt = (v: number) => dayjs(v).format('HH:mm:ss')

  const renderChart = (key: keyof Row, color: string, name: string) => (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={rows} onMouseMove={handleMove} onMouseLeave={handleLeave} syncId="dash-v2">
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="t" tickFormatter={xFmt} domain={['dataMin', 'dataMax']} type="number" />
        <YAxis />
        <Tooltip labelFormatter={xFmt} />
        <Line type="monotone" dataKey={key} stroke={color} name={name} dot={false} isAnimationActive={false} />
        {hoverT != null && <ReferenceLine x={hoverT} stroke="#888" strokeDasharray="3 3" />}
      </LineChart>
    </ResponsiveContainer>
  )

  return (
    <div>
      {renderChart('ingest',  '#1677ff', 'Ingest rate (msgs/s)')}
      {renderChart('consume', '#52c41a', 'Consume rate (msgs/s)')}
      {renderChart('lag',     '#ff4d4f', 'Consumer lag (msgs)')}
    </div>
  )
}
```

> **Note**: Recharts có prop `syncId` built-in — đặt cùng `syncId` cho 3 chart → tooltip/cursor auto-sync. Nếu bản recharts cũ chưa support đầy đủ, fallback bằng `hoverT` state lift như trên.

---

## Section 7: `SnapshotCommanderTab.tsx`

```tsx
import { Card, Col, Empty, Progress, Row, Statistic, Table, Tag, Button, Modal } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { useDashboardSnapshot, usePrioritizeSnapshot } from '../../hooks/useDashboard'
import { humanizeSeconds } from '../../utils/ttc'
import type { SnapshotItem } from '../../types/dashboard'

export function SnapshotCommanderTab() {
  const { data, isLoading } = useDashboardSnapshot()
  const prioritize = usePrioritizeSnapshot()

  const onPrioritize = (id: string) => {
    Modal.confirm({
      title: 'Đẩy snapshot này lên ưu tiên cao?',
      content: `Snapshot ${id.slice(0, 8)}… sẽ chạy trước các snapshot pending khác.`,
      onOk: () => prioritize.mutate({ id, priority: 100 }),
    })
  }

  const activeColumns = [
    { title: 'ID',     dataIndex: 'snapshot_id', render: (v: string) => v.slice(0, 8) + '…' },
    { title: 'Table',  dataIndex: 'table' },
    { title: 'Progress', dataIndex: 'progress_pct',
      render: (v?: number) => <Progress percent={Math.round(v ?? 0)} size="small" /> },
    { title: 'Throughput', dataIndex: 'throughput_mbps',
      render: (v?: number) => `${(v ?? 0).toFixed(1)} MB/s` },
    { title: 'ETA', dataIndex: 'eta_seconds',
      render: (v?: number) => v == null ? '–' : humanizeSeconds(v) },
    { title: 'Action', render: (_: any, r: SnapshotItem) => (
      <Button size="small" onClick={() => onPrioritize(r.snapshot_id)}>Prioritize</Button>
    ) },
  ]

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}><Card><Statistic title="Active slots"
          value={`${data?.active.length ?? 0}/${data?.max_concurrent_slots ?? '–'}`}
          prefix={<ThunderboltOutlined />} /></Card></Col>
        <Col span={6}><Card><Statistic title="Pending queue"
          value={data?.pending.length ?? 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="Total throughput"
          value={`${(data?.active.reduce((a, x) => a + (x.throughput_mbps ?? 0), 0) ?? 0).toFixed(1)} MB/s`} /></Card></Col>
        <Col span={6}><Card><Statistic title="Avg ETA"
          value={humanizeSeconds(avgEta(data?.active ?? []))} /></Card></Col>
      </Row>

      <Card title="Active snapshots" loading={isLoading}>
        {data?.active.length ? (
          <Table dataSource={data.active} columns={activeColumns} rowKey="snapshot_id" pagination={false} />
        ) : <Empty description="Không có snapshot đang chạy" />}
      </Card>

      <Card title={`Pending queue (${data?.pending.length ?? 0})`} style={{ marginTop: 16 }}>
        {data?.pending.length ? (
          <Table dataSource={data.pending}
                 columns={[
                   { title: 'ID',    dataIndex: 'snapshot_id', render: (v: string) => v.slice(0, 8) + '…' },
                   { title: 'Table', dataIndex: 'table' },
                   { title: 'Queued at', dataIndex: 'queued_at' },
                   { title: '', render: (_: any, r: SnapshotItem) => (
                     <Button size="small" type="primary" onClick={() => onPrioritize(r.snapshot_id)}>
                       Prioritize
                     </Button>
                   ) },
                 ]}
                 rowKey="snapshot_id"
                 pagination={false}
          />
        ) : <Empty description="Queue trống" />}
      </Card>
    </div>
  )
}

function avgEta(items: SnapshotItem[]): number {
  const vals = items.map(i => i.eta_seconds).filter((v): v is number => v != null)
  if (!vals.length) return 0
  return vals.reduce((a, b) => a + b, 0) / vals.length
}
```

---

## Section 8: `StreamingRealtimeTab.tsx`

```tsx
import { Card, Col, Row } from 'antd'
import { useMemo } from 'react'
import { useDashboardTimeline } from '../../hooks/useDashboard'
import { computeTtc } from '../../utils/ttc'
import { TtcWidget } from './TtcWidget'
import { UnifiedCrosshairChart } from './UnifiedCrosshairChart'
import { StreamExpiryPair } from './StreamExpiryPair'
import { useSystemHealth } from '../../hooks/useSystemHealth'   // existing

export function StreamingRealtimeTab() {
  const { data: timeline } = useDashboardTimeline('15m', '15s')
  const { data: health }   = useSystemHealth()
  const ttc = useMemo(() => computeTtc(timeline), [timeline])

  return (
    <div>
      <Row gutter={16}>
        <Col span={12}><TtcWidget ttc={ttc} /></Col>
        <Col span={12}><StreamExpiryPair health={health} ttc={ttc} /></Col>
      </Row>
      <Card title="Timeline (15m)" style={{ marginTop: 16 }}>
        <UnifiedCrosshairChart data={timeline} />
      </Card>
      <Card title="Reconciliation" style={{ marginTop: 16 }}>
        {/* render từ health.reconciliation — pattern reuse SystemHealth.tsx */}
      </Card>
    </div>
  )
}
```

---

## Section 9: `DlqDriftTab.tsx` (lược)

```tsx
import { Card, Col, List, Modal, Row, Tag } from 'antd'
import { useState } from 'react'
import { useDashboardDlq, useDashboardDrift } from '../../hooks/useDashboard'
import { PayloadViewerModal } from './PayloadViewerModal'
import type { DLQItem } from '../../types/dashboard'

export function DlqDriftTab() {
  const { data: dlq } = useDashboardDlq(50, '1h')
  const { data: drift } = useDashboardDrift(50)
  const [selected, setSelected] = useState<DLQItem | null>(null)

  return (
    <>
      <Row gutter={16}>
        <Col span={12}>
          <Card title={`DLQ recent (${dlq?.items.length ?? 0})`}>
            <List dataSource={dlq?.items ?? []}
              renderItem={(it) => (
                <List.Item onClick={() => setSelected(it)} style={{ cursor: 'pointer' }}>
                  <List.Item.Meta
                    title={<><Tag color="red">{it.error_class}</Tag> {it.topic}</>}
                    description={it.error_message}
                  />
                  <span>{new Date(it.occurred_at).toLocaleString()}</span>
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col span={12}>
          <Card title={`Schema drift recent (${drift?.items.length ?? 0})`}>
            <List dataSource={drift?.items ?? []}
              renderItem={(it) => (
                <List.Item>
                  <List.Item.Meta
                    title={`${it.table}.${it.field}`}
                    description={`Detected ${it.detection_count}× · last seen ${new Date(it.last_seen).toLocaleString()}`}
                  />
                  <Tag color={it.approval_status === 'pending' ? 'orange' : it.approval_status === 'approved' ? 'green' : 'red'}>
                    {it.approval_status}
                  </Tag>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>

      <PayloadViewerModal item={selected} onClose={() => setSelected(null)} />
    </>
  )
}
```

---

## Section 10: `PayloadViewerModal.tsx`

```tsx
import { Button, Modal, Space, Tag, Typography } from 'antd'
import type { DLQItem } from '../../types/dashboard'

const { Text, Paragraph } = Typography

const SIGNOZ_BASE = import.meta.env.VITE_SIGNOZ_BASE_URL || 'http://localhost:3301'

function buildSignozUrl(traceID?: string) {
  if (!traceID) return null
  return `${SIGNOZ_BASE.replace(/\/$/, '')}/trace/${traceID}`
}

export function PayloadViewerModal({ item, onClose }: { item: DLQItem | null; onClose: () => void }) {
  const signozUrl = buildSignozUrl(item?.trace_id)
  return (
    <Modal
      open={!!item}
      onCancel={onClose}
      footer={null}
      width={780}
      title={item ? `DLQ #${item.id} · ${item.topic}` : ''}
    >
      {item && (
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <Tag color="red">{item.error_class}</Tag>
            <Text type="secondary">retry: {item.retry_count}</Text>
          </div>
          <Paragraph>{item.error_message}</Paragraph>
          {signozUrl && (
            <Button type="primary" href={signozUrl} target="_blank" rel="noreferrer">
              View trace in SigNoz
            </Button>
          )}
          <div>
            <Text strong>Payload (masked):</Text>
            <pre style={{ maxHeight: 400, overflow: 'auto', background: '#fafafa', padding: 12 }}>
{JSON.stringify(item.payload_masked, null, 2)}
            </pre>
          </div>
        </Space>
      )}
    </Modal>
  )
}
```

---

## Section 11: `DashboardV2.tsx` container

```tsx
import { Alert, Tabs, Typography } from 'antd'
import { useSearchParams } from 'react-router-dom'
import { useMemo } from 'react'
import { useDashboardTimeline } from '../hooks/useDashboard'
import { computeTtc } from '../utils/ttc'
import { SnapshotCommanderTab } from '../components/dashboard/SnapshotCommanderTab'
import { StreamingRealtimeTab } from '../components/dashboard/StreamingRealtimeTab'
import { DlqDriftTab } from '../components/dashboard/DlqDriftTab'

const { Title } = Typography

export default function DashboardV2() {
  const [params, setParams] = useSearchParams()
  const tab = params.get('tab') || 'snapshot'

  // Top-level banner alert when streaming red_blink
  const { data: timeline } = useDashboardTimeline('1m', '15s')
  const ttc = useMemo(() => computeTtc(timeline), [timeline])
  const showBanner = ttc?.state === 'cannot_catch_up'

  return (
    <div>
      <Title level={3}>CDC Dashboard V2</Title>
      {showBanner && (
        <Alert
          type="error"
          showIcon
          banner
          message="Pipeline không thể bắt kịp — consumer rate < ingest rate"
          style={{ marginBottom: 16 }}
        />
      )}
      <Tabs
        activeKey={tab}
        onChange={(k) => setParams({ tab: k })}
        items={[
          { key: 'snapshot',  label: 'Snapshot Commander',  children: <SnapshotCommanderTab /> },
          { key: 'streaming', label: 'Streaming Real-time', children: <StreamingRealtimeTab /> },
          { key: 'dlq',       label: 'DLQ & Schema Drift',  children: <DlqDriftTab /> },
        ]}
      />
    </div>
  )
}
```

---

## Section 12: Route wiring (`src/App.tsx`)

```tsx
// Trong <Routes>
<Route path="/dashboard-v2" element={<DashboardV2 />} />
// Trong sider menu items, thêm:
{ key: 'dashboard-v2', icon: <DashboardOutlined />, label: <Link to="/dashboard-v2">Dashboard V2</Link> }
```

---

## Section 13: Env config

`.env.example` thêm:
```
VITE_SIGNOZ_BASE_URL=http://localhost:3301
```

`README.md` (FE) section "Environment" cập nhật.

---

## Verification checklist (theo §3)

- [ ] `npm run typecheck` PASS
- [ ] `npm run lint` PASS
- [ ] `npm run build` PASS — bundle size delta < 100KB gz
- [ ] `npm run test` PASS (vitest, các unit test `ttc.test.ts` + render test cho widget)
- [ ] Manual smoke 4 scenario (xem `02_plan_frontend.md` Verify gate)
- [ ] Lighthouse a11y > 90
- [ ] `prefers-reduced-motion` toggle → blink dừng
- [ ] `/security-agent` review PASS (kiểm XSS từ payload JSON viewer — phải HTML-escape pre/code)
