# 03_implementation_phase_ui — Admin Audit UI

## Backend (`cdc-cms-service`)

### Migration NEW: `migrations/0060_qa_gap_state.sql`
```sql
CREATE SCHEMA IF NOT EXISTS cdc_system;

CREATE TABLE IF NOT EXISTS cdc_system.qa_gap_state (
    gap_id          VARCHAR(8)   PRIMARY KEY,            -- 'G-1', 'G-2', ...
    title           TEXT         NOT NULL,
    description     TEXT         NOT NULL,
    priority        VARCHAR(2)   NOT NULL CHECK (priority IN ('P0','P1','P2')),
    category        TEXT         NOT NULL,               -- 'functional', 'stability', 'performance', 'resource', 'metric'
    status          VARCHAR(16)  NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_progress','closed','wontfix')),
    estimated_hours NUMERIC(4,1) NOT NULL DEFAULT 0,
    actual_hours    NUMERIC(4,1),
    closed_at       TIMESTAMPTZ,
    closed_by       TEXT,
    evidence_link   TEXT,                                -- link to PR/commit
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cdc_system.qa_criterion_rating (
    criterion_id    VARCHAR(8)   PRIMARY KEY,            -- '1.1', '1.2', ...
    group_name      VARCHAR(64)  NOT NULL,               -- 'Functional & Correctness'
    title           TEXT         NOT NULL,
    rating          VARCHAR(2)   NOT NULL CHECK (rating IN ('L0','L1','L2','L3','L4')),
    evidence        TEXT,                                -- file:line refs (JSON or markdown)
    last_audited_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed 16 gap (initial state = open, từ audit 2026-05-26)
INSERT INTO cdc_system.qa_gap_state (gap_id, title, description, priority, category, estimated_hours) VALUES
  ('G-1','cdc_kafka_consumer_lag metric DEAD','Metric defined nhưng không có .Set() call','P0','metric',0.5),
  ('G-2','OTel Collector exporter chỉ debug stdout','Traces production không persist','P0','metric',1.0),
  ('G-3','Prometheus production thiếu scrape','cdc-worker:9090 + kafka-exporter:9308 không scrape','P0','metric',1.0),
  ('G-4','Pipeline-level DLQ circuit breaker thiếu','DLQ rate spike không pause pipeline','P0','stability',4.0),
  ('G-5','Restart smoke test failover','Không có script verify zero-loss + zero-dup','P1','stability',4.0),
  ('G-6','WAL slot expire alert','Slot lag không proactive alert','P1','stability',4.0),
  ('G-7','pprof + goleak','pprof endpoint vắng, goleak imported không dùng','P1','resource',2.0),
  ('G-8','Event Ordering test','Không có test out-of-order','P1','functional',2.0),
  ('G-9','Schema Drift approve E2E test','Approve flow CMS không có integration test','P1','functional',8.0),
  ('G-10','Tier 3 off-peak config','Hardcoded 02-05h','P2','functional',1.0),
  ('G-11','cdc_batches_flushed_total counter','Thiếu per-batch counter','P2','performance',0.5),
  ('G-12','Burst mode adaptive batch','Không adaptive khi lag cao','P2','performance',2.0),
  ('G-13','Per-source connection pool','Pool global, 60 source contention','P2','resource',4.0),
  ('G-14','Runbook drift + WAL','Thiếu runbook escalation','P2','metric',2.0),
  ('G-15','Chaos test network flicker','Không test mất mạng 5-15 phút','P2','stability',4.0),
  ('G-16','Load test TPS script','Chưa benchmark TPS thực tế','P2','performance',2.5)
ON CONFLICT (gap_id) DO NOTHING;

-- Seed 16 criterion rating (snapshot 2026-05-26)
INSERT INTO cdc_system.qa_criterion_rating (criterion_id, group_name, title, rating, evidence) VALUES
  ('1.1','Functional & Correctness','Data Reconciliation','L4','recon_core.go:98-753; recon_hash_test.go:154-296'),
  ('1.2','Functional & Correctness','Schema Drift','L3','schema_inspector.go:85-276'),
  ('1.3','Functional & Correctness','Event Ordering','L2','schema_adapter.go:506-531'),
  ('2.1','Stability & Resilience','Failover & Self-Healing','L2','kafka_consumer.go:444-465'),
  ('2.2','Stability & Resilience','Network Flicker','L2','nats_client.go:19-28; recon_source_agent.go:831-897'),
  ('2.3','Stability & Resilience','LSN/Offset Expire','L1','pg-source-connector.json:13'),
  ('2.4','Stability & Resilience','DLQ','L3','dlq_handler.go:122-274; dlq_state_machine.go:37-238'),
  ('3.1','Performance & Scalability','Data Lag','L3','kafka_consumer.go:396-400; prometheus.go:135-139'),
  ('3.2','Performance & Scalability','Throughput / TPS','L3','batch_buffer.go:33-243'),
  ('3.3','Performance & Scalability','Backlog Catch-up','L1','kafka-exporter:9308 docker-compose.yml:169-183'),
  ('3.4','Performance & Scalability','Source DB Overhead','L2','pg-source-connector.json:18-19'),
  ('4.1','Resource Utilization','Memory Leak / Soak','L1','go.sum:260-261 goleak unused'),
  ('4.2','Resource Utilization','Concurrency / Throttling','L3','postgres.go:34-109; recon_source_agent.go:163'),
  ('5.1','Metric Monitor','Replication Lag Dashboard','L2','prom_client.go'),
  ('5.2','Metric Monitor','CPU / Mem','L1','No node_exporter'),
  ('5.3','Metric Monitor','Disk I/O / Network','L1','Demo Kafka JMX only'),
  ('5.4','Metric Monitor','OpenTelemetry','L3','otel.go:317-465')
ON CONFLICT (criterion_id) DO NOTHING;
```

### DTO NEW: `internal/api/dto/audit_dto.go`
```go
package dto

type QASummaryResponse struct {
    CompositeScore   float64                  `json:"composite_score"`     // 0-100
    TotalCriteria    int                      `json:"total_criteria"`       // 16
    RatingBreakdown  map[string]int           `json:"rating_breakdown"`     // {"L4":1,"L3":6,...}
    GapBreakdown     map[string]GapCount      `json:"gap_breakdown"`        // {"P0":{open:4,closed:0},...}
    LastAudited      string                   `json:"last_audited"`         // RFC3339
    Criteria         []CriterionRow           `json:"criteria"`
}

type GapCount struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    Closed     int `json:"closed"`
    Wontfix    int `json:"wontfix"`
}

type CriterionRow struct {
    ID       string `json:"id"`
    Group    string `json:"group"`
    Title    string `json:"title"`
    Rating   string `json:"rating"`
    Evidence string `json:"evidence"`
}

type GapRow struct {
    ID              string  `json:"id"`
    Title           string  `json:"title"`
    Description     string  `json:"description"`
    Priority        string  `json:"priority"`
    Category        string  `json:"category"`
    Status          string  `json:"status"`
    EstimatedHours  float64 `json:"estimated_hours"`
    ActualHours     *float64 `json:"actual_hours,omitempty"`
    ClosedAt        *string `json:"closed_at,omitempty"`
    EvidenceLink    string  `json:"evidence_link"`
}

type MetricHealthResponse struct {
    Metrics []MetricHealth `json:"metrics"`
}

type MetricHealth struct {
    Name        string  `json:"name"`              // 'consumer_lag', 'e2e_latency_p99', ...
    Value       float64 `json:"value"`
    Status      string  `json:"status"`            // 'healthy', 'warning', 'critical', 'unknown'
    Threshold   float64 `json:"threshold"`
    Unit        string  `json:"unit"`              // 'events', 'seconds', 'percent'
    LastUpdated string  `json:"last_updated"`
}
```

### Query layer NEW: `internal/app/queries/get_qa_summary.go`
```go
package queries

type GetQASummaryQuery struct{}

type GetQASummaryHandler struct {
    db *gorm.DB
}

func (h *GetQASummaryHandler) Handle(ctx context.Context, q GetQASummaryQuery) (*dto.QASummaryResponse, error) {
    var criteria []model.QACriterionRating
    if err := h.db.WithContext(ctx).Find(&criteria).Error; err != nil { return nil, err }

    var gaps []model.QAGapState
    if err := h.db.WithContext(ctx).Find(&gaps).Error; err != nil { return nil, err }

    // Compute composite
    weight := map[string]int{"L0":0, "L1":1, "L2":2, "L3":3, "L4":4}
    total := 0
    breakdown := map[string]int{}
    for _, c := range criteria {
        total += weight[c.Rating]
        breakdown[c.Rating]++
    }
    composite := float64(total) / float64(len(criteria)*4) * 100

    // Gap breakdown
    gapBreakdown := map[string]dto.GapCount{}
    for _, g := range gaps {
        gc := gapBreakdown[g.Priority]
        switch g.Status {
        case "open": gc.Open++
        case "in_progress": gc.InProgress++
        case "closed": gc.Closed++
        case "wontfix": gc.Wontfix++
        }
        gapBreakdown[g.Priority] = gc
    }

    return &dto.QASummaryResponse{
        CompositeScore: composite,
        TotalCriteria: len(criteria),
        RatingBreakdown: breakdown,
        GapBreakdown: gapBreakdown,
        LastAudited: criteria[0].LastAuditedAt.Format(time.RFC3339),
        Criteria: toRows(criteria),
    }, nil
}
```

### Handler NEW: `internal/api/audit_handler.go`
```go
package api

import "github.com/gofiber/fiber/v2"

type AuditHandler struct {
    qaQuery     *queries.GetQASummaryHandler
    gapsQuery   *queries.ListGapsHandler
    metricQuery *queries.GetMetricHealthHandler
    logger      *zap.Logger
}

func NewAuditHandler(qa *queries.GetQASummaryHandler, gaps *queries.ListGapsHandler, mh *queries.GetMetricHealthHandler, logger *zap.Logger) *AuditHandler {
    return &AuditHandler{qaQuery: qa, gapsQuery: gaps, metricQuery: mh, logger: logger}
}

func (h *AuditHandler) Summary(c *fiber.Ctx) error {
    resp, err := h.qaQuery.Handle(c.Context(), queries.GetQASummaryQuery{})
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(resp)
}

func (h *AuditHandler) Gaps(c *fiber.Ctx) error {
    priority := c.Query("priority") // P0|P1|P2|<empty>
    status := c.Query("status")
    resp, err := h.gapsQuery.Handle(c.Context(), queries.ListGapsQuery{Priority: priority, Status: status})
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(resp)
}

func (h *AuditHandler) MetricHealth(c *fiber.Ctx) error {
    resp, err := h.metricQuery.Handle(c.Context(), queries.GetMetricHealthQuery{})
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(resp)
}
```

### Metric Health query (`get_metric_health.go`) — gọi PromClient
```go
type GetMetricHealthHandler struct {
    prom *infrahttp.PromClient
}

func (h *GetMetricHealthHandler) Handle(ctx context.Context, q GetMetricHealthQuery) (*dto.MetricHealthResponse, error) {
    metrics := []dto.MetricHealth{}

    // 1. Consumer lag
    if v, err := h.prom.QueryGauge(ctx, "max(cdc_kafka_consumer_lag)"); err == nil {
        status := classify(v, 1000, 10000)
        metrics = append(metrics, dto.MetricHealth{
            Name: "consumer_lag", Value: v, Status: status, Threshold: 10000, Unit: "events",
            LastUpdated: time.Now().UTC().Format(time.RFC3339),
        })
    }

    // 2. E2E latency p99
    if v, err := h.prom.QueryQuantile(ctx, "cdc_e2e_latency_seconds", 0.99, "5m"); err == nil {
        status := classify(v, 1, 5)
        metrics = append(metrics, dto.MetricHealth{Name: "e2e_latency_p99", Value: v, Status: status, Threshold: 5, Unit: "seconds"})
    }

    // 3. DLQ rate
    if v, err := h.prom.QueryRate(ctx, "cdc_dlq_write_failures_total", "1m"); err == nil {
        status := classify(v, 1, 10)
        metrics = append(metrics, dto.MetricHealth{Name: "dlq_rate", Value: v, Status: status, Threshold: 10, Unit: "events/sec"})
    }

    // 4. Recon drift
    if v, err := h.prom.QueryGauge(ctx, "sum(cdc_recon_drift_count)"); err == nil {
        status := classify(v, 1, 100)
        metrics = append(metrics, dto.MetricHealth{Name: "recon_drift", Value: v, Status: status, Threshold: 100, Unit: "rows"})
    }

    return &dto.MetricHealthResponse{Metrics: metrics}, nil
}

func classify(v, warnT, critT float64) string {
    if v >= critT { return "critical" }
    if v >= warnT { return "warning" }
    return "healthy"
}
```

### Router wire `internal/router/router.go`
```go
// Inside SetupRoutes() — admin group (router.go:366 area)
adminGroup := apiGroup.Group("", middleware.RequireRole("admin"))
dualGet(adminGroup, "/audit/qa-summary", auditHandler.Summary)
dualGet(adminGroup, "/audit/gaps", auditHandler.Gaps)
dualGet(adminGroup, "/audit/metric-health", auditHandler.MetricHealth)
```

### DI wire `internal/server/server.go`
```go
// Inside Server.New() — sau khi PromClient được khởi tạo
qaQueryHandler := queries.NewGetQASummaryHandler(db, logger)
gapsQueryHandler := queries.NewListGapsHandler(db, logger)
metricHealthHandler := queries.NewGetMetricHealthHandler(promClient, logger)
auditHandler := api.NewAuditHandler(qaQueryHandler, gapsQueryHandler, metricHealthHandler, logger)
// pass auditHandler vào SetupRoutes signature
```

### Test NEW: `internal/api/audit_handler_test.go`
```go
func TestAuditHandler_Summary(t *testing.T) {
    app := fiber.New()
    app.Use(func(c *fiber.Ctx) error {
        c.Locals("role", "admin"); return c.Next()
    })
    db := setupTestDB(t) // sqlite in-memory hoặc testcontainers PG
    seedCriteria(t, db)
    handler := api.NewAuditHandler(/* ... */)
    app.Get("/api/v1/audit/qa-summary", handler.Summary)

    resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/audit/qa-summary", nil), -1)
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)
    var body dto.QASummaryResponse
    json.NewDecoder(resp.Body).Decode(&body)
    require.InDelta(t, 54.7, body.CompositeScore, 1.0)
    require.Equal(t, 16, body.TotalCriteria)
}
```

---

## Frontend (`cdc-cms-web`)

### Type NEW: `src/types/audit.ts`
```typescript
export type Rating = 'L0' | 'L1' | 'L2' | 'L3' | 'L4';
export type Priority = 'P0' | 'P1' | 'P2';
export type GapStatus = 'open' | 'in_progress' | 'closed' | 'wontfix';
export type MetricStatus = 'healthy' | 'warning' | 'critical' | 'unknown';

export interface CriterionRow {
  id: string;
  group: string;
  title: string;
  rating: Rating;
  evidence: string;
}

export interface GapCount {
  open: number;
  in_progress: number;
  closed: number;
  wontfix: number;
}

export interface QASummary {
  composite_score: number;
  total_criteria: number;
  rating_breakdown: Record<Rating, number>;
  gap_breakdown: Record<Priority, GapCount>;
  last_audited: string;
  criteria: CriterionRow[];
}

export interface Gap {
  id: string;
  title: string;
  description: string;
  priority: Priority;
  category: string;
  status: GapStatus;
  estimated_hours: number;
  actual_hours?: number;
  closed_at?: string;
  evidence_link: string;
}

export interface MetricHealth {
  name: string;
  value: number;
  status: MetricStatus;
  threshold: number;
  unit: string;
  last_updated: string;
}

export interface MetricHealthResponse {
  metrics: MetricHealth[];
}
```

### Hook NEW: `src/hooks/useAuditStatus.ts`
```typescript
import { useQuery } from '@tanstack/react-query';
import { cmsApi } from '../services/api';
import type { QASummary, Gap, MetricHealthResponse, Priority, GapStatus } from '../types/audit';

const REFETCH_MS = 30_000;

export function useQASummary() {
  return useQuery<QASummary>({
    queryKey: ['audit', 'qa-summary'],
    queryFn: async () => (await cmsApi.get<QASummary>('/api/v1/audit/qa-summary')).data,
    refetchInterval: REFETCH_MS,
    staleTime: REFETCH_MS,
  });
}

export function useGaps(priority?: Priority, status?: GapStatus) {
  return useQuery<Gap[]>({
    queryKey: ['audit', 'gaps', priority, status],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (priority) params.set('priority', priority);
      if (status) params.set('status', status);
      const url = `/api/v1/audit/gaps${params.toString() ? `?${params}` : ''}`;
      return (await cmsApi.get<Gap[]>(url)).data;
    },
    refetchInterval: REFETCH_MS,
  });
}

export function useMetricHealth() {
  return useQuery<MetricHealthResponse>({
    queryKey: ['audit', 'metric-health'],
    queryFn: async () => (await cmsApi.get<MetricHealthResponse>('/api/v1/audit/metric-health')).data,
    refetchInterval: 15_000, // faster refresh cho health check
  });
}
```

### Component NEW: `src/components/audit/RatingMatrix.tsx`
```tsx
import { Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { CriterionRow, Rating } from '../../types/audit';

const RATING_COLOR: Record<Rating, string> = {
  L0: 'red', L1: 'volcano', L2: 'orange', L3: 'blue', L4: 'green',
};

const columns: ColumnsType<CriterionRow> = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: 'Nhóm', dataIndex: 'group', width: 200 },
  { title: 'Tiêu chí', dataIndex: 'title' },
  { title: 'Rating', dataIndex: 'rating', width: 80,
    render: (r: Rating) => <Tag color={RATING_COLOR[r]}>{r}</Tag> },
  { title: 'Evidence', dataIndex: 'evidence', ellipsis: true },
];

export function RatingMatrix({ rows }: { rows: CriterionRow[] }) {
  return <Table columns={columns} dataSource={rows} rowKey="id" pagination={false} size="small" />;
}
```

### Component NEW: `src/components/audit/GapList.tsx`
```tsx
import { Table, Tag, Progress, Tabs } from 'antd';
import type { Gap, Priority } from '../../types/audit';
import { useGaps } from '../../hooks/useAuditStatus';

const PRIORITY_COLOR: Record<Priority, string> = { P0: 'red', P1: 'orange', P2: 'blue' };
const STATUS_COLOR: Record<string, string> = {
  open: 'red', in_progress: 'orange', closed: 'green', wontfix: 'default',
};

export function GapList() {
  return (
    <Tabs items={[
      { key: 'P0', label: 'P0 Blocker', children: <GapTable priority="P0" /> },
      { key: 'P1', label: 'P1 Pre-release', children: <GapTable priority="P1" /> },
      { key: 'P2', label: 'P2 Backlog', children: <GapTable priority="P2" /> },
    ]} />
  );
}

function GapTable({ priority }: { priority: Priority }) {
  const { data, isLoading } = useGaps(priority);
  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Title', dataIndex: 'title' },
    { title: 'Priority', dataIndex: 'priority',
      render: (p: Priority) => <Tag color={PRIORITY_COLOR[p]}>{p}</Tag> },
    { title: 'Status', dataIndex: 'status',
      render: (s: string) => <Tag color={STATUS_COLOR[s]}>{s}</Tag> },
    { title: 'Estimated (h)', dataIndex: 'estimated_hours', width: 100 },
    { title: 'Evidence', dataIndex: 'evidence_link', ellipsis: true,
      render: (link: string) => link ? <a href={link} target="_blank" rel="noreferrer">View</a> : '-' },
  ];
  return <Table loading={isLoading} dataSource={data} columns={columns} rowKey="id" pagination={false} size="small" />;
}
```

### Component NEW: `src/components/audit/MetricHealthCards.tsx`
```tsx
import { Row, Col, Card, Statistic, Tag } from 'antd';
import { useMetricHealth } from '../../hooks/useAuditStatus';
import type { MetricStatus } from '../../types/audit';

const STATUS_COLOR: Record<MetricStatus, string> = {
  healthy: 'green', warning: 'orange', critical: 'red', unknown: 'default',
};

export function MetricHealthCards() {
  const { data, isLoading } = useMetricHealth();
  return (
    <Row gutter={[16, 16]}>
      {data?.metrics.map((m) => (
        <Col xs={24} sm={12} md={6} key={m.name}>
          <Card loading={isLoading}>
            <Statistic
              title={m.name.replace(/_/g, ' ').toUpperCase()}
              value={m.value}
              precision={2}
              suffix={m.unit}
            />
            <Tag color={STATUS_COLOR[m.status]} style={{ marginTop: 8 }}>{m.status}</Tag>
            <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>
              threshold: {m.threshold} {m.unit}
            </div>
          </Card>
        </Col>
      ))}
    </Row>
  );
}
```

### Page NEW: `src/pages/AuditPage.tsx`
```tsx
import { Typography, Card, Row, Col, Statistic, Skeleton, Alert, Divider } from 'antd';
import { useQASummary } from '../hooks/useAuditStatus';
import { RatingMatrix } from '../components/audit/RatingMatrix';
import { GapList } from '../components/audit/GapList';
import { MetricHealthCards } from '../components/audit/MetricHealthCards';

const { Title, Text } = Typography;

function scoreColor(score: number): string {
  if (score >= 80) return '#52c41a';
  if (score >= 60) return '#faad14';
  return '#f5222d';
}

export default function AuditPage() {
  const { data, isLoading, error } = useQASummary();

  if (isLoading) return <Skeleton active />;
  if (error || !data) return <Alert type="error" message="Không thể tải audit summary" />;

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>QA Audit Dashboard</Title>
      <Text type="secondary">Last audited: {data.last_audited}</Text>

      <Divider orientation="left">Composite Score</Divider>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="Composite Score"
              value={data.composite_score}
              precision={1}
              suffix="%"
              valueStyle={{ color: scoreColor(data.composite_score) }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Total Criteria" value={data.total_criteria} />
            <div style={{ marginTop: 8, fontSize: 12 }}>
              L4: {data.rating_breakdown.L4 || 0} | L3: {data.rating_breakdown.L3 || 0} |
              L2: {data.rating_breakdown.L2 || 0} | L1: {data.rating_breakdown.L1 || 0} |
              L0: {data.rating_breakdown.L0 || 0}
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="P0 Open" value={data.gap_breakdown.P0?.open || 0}
              valueStyle={{ color: data.gap_breakdown.P0?.open ? '#f5222d' : '#52c41a' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="P1 Open" value={data.gap_breakdown.P1?.open || 0} />
          </Card>
        </Col>
      </Row>

      <Divider orientation="left">Live Metric Health</Divider>
      <MetricHealthCards />

      <Divider orientation="left">Rating Matrix</Divider>
      <RatingMatrix rows={data.criteria} />

      <Divider orientation="left">Gap Status</Divider>
      <GapList />
    </div>
  );
}
```

### Route + Menu wire `src/App.tsx`
**Vị trí 1** — lazy import (App.tsx:17-30 pattern):
```tsx
const AuditPage = lazy(() => import('./pages/AuditPage'));
```

**Vị trí 2** — menu item trong group `operate` (App.tsx:104-140), thêm cuối group:
```tsx
{
  key: '/audit',
  icon: <AuditOutlined />, // import từ '@ant-design/icons'
  label: <Link to="/audit">QA Audit</Link>,
}
```

**Vị trí 3** — Route trong `<Routes>` (App.tsx:187-206):
```tsx
<Route path="/audit" element={<AuditPage />} />
```

### Acceptance
- `npm run dev` → http://localhost:5173/audit → page render với data từ backend.
- 4 panel: Composite Score gauge + Metric Health Cards (4) + Rating Matrix (16 row) + Gap List (3 tab P0/P1/P2).
- Auto-refresh 30s.
- Auth guard: chỉ admin role thấy menu item (currently FE chỉ check `isLoggedIn`, role-check ở BE — BE return 403 nếu không phải admin → FE show error).

---

## Composite score change (UI done)
- KHÔNG thay đổi composite score (UI là visibility tool, không vá gap).
- Nhưng tăng **operational maturity**: admin có dashboard realtime → giảm time-to-detect khi gap re-open.

## Effort recap
- BE: 8h (handler + query + DTO + migration + test).
- FE: 8h (page + hook + 3 component + types + route).
- **Tổng UI**: 16h Muscle work.
