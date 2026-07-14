import { useMemo, useState } from 'react';
import { Table, Tag, Drawer, Card, Space, Typography, Tooltip, Spin, Empty, Badge, Row, Col, Button } from 'antd';
import { ArrowRightOutlined, DatabaseOutlined, InfoCircleOutlined, RightOutlined, DownOutlined, SyncOutlined, MedicineBoxOutlined, DeleteOutlined, ThunderboltOutlined } from '@ant-design/icons';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Legend, ResponsiveContainer,
  Tooltip as RechartsTooltip,
} from 'recharts';
import { useTableHistory, type ReconReport } from '../hooks/useReconStatus';
import { useQuery } from '@tanstack/react-query';
import { cmsApi } from '../services/api';

const { Text, Title } = Typography;

// ============================================================
// Recon V4 — Master Pipeline Grid (trục data-lineage)
// Tầng 1: mỗi dòng = 1 pipeline Source → Shadow → Master, ghép từ
//   report Segment A (source↔shadow) + Segment B (shadow↔master).
// Tầng 2: Drawer drill-down = Flow map + Convergence chart + Nhật ký recon.
// Thuần FE — dữ liệu từ /api/reconciliation/report (per-segment) +
// /api/reconciliation/report/:table (TableHistory, có sẵn).
// ============================================================

interface PipelineRow {
  key: string;
  sourceName: string;        // db.table nguồn
  sourceDb: string;          // source DB name (dùng cho group header)
  sourceConnector: string | null; // connector code (shadow_binding.source_connection_code)
  shadowName: string;        // schema.table shadow
  masterName: string | null; // master FQN schema.table (null = chưa có binding B)
  sourceCount: number | null; // A.source_count (7d window)
  shadowCount: number | null; // A.dest_count (window); fallback B.source_count
  masterCount: number | null; // B.dest_count (window)
  driftAB: number | null;     // shadow − source (A.diff đảo dấu hiển thị master-centric)
  driftBC: number | null;     // master − shadow (−B.diff)
  ingestLagMs: number | null;
  transmuteLagMs: number | null;
  lastReconAt: string;
  statusA?: string;
  statusB?: string;
  missingB: number;
  staleB: number;
  rowA?: ReconReport;
  rowB?: ReconReport;
}

const LAGGING_MS = 30 * 60 * 1000; // >30' = Lagging

const splitFQN = (fqn: string | null) => {
  if (!fqn) return { schema: '', table: '—' };
  const parts = fqn.split('.');
  if (parts.length > 1) {
    return { schema: parts[0], table: parts[parts.length - 1] };
  }
  return { schema: '', table: fqn };
};

const STATUS_COLOR: Record<string, string> = {
  ok: 'green',
  ok_empty: 'default',
  warning: 'gold',
  drift: 'red',
  dest_missing: 'red',
  source_missing_or_stale: 'orange',
  error: 'red',
};

const STATUS_LABEL_VI: Record<string, string> = {
  ok: 'Khớp',
  ok_empty: 'Khớp (trống)',
  warning: 'Cảnh báo',
  drift: 'Lệch',
  dest_missing: 'Thiếu dest',
  source_missing_or_stale: 'Nguồn trống/stale',
  error: 'Lỗi',
};

function overallStatus(p: PipelineRow): { label: string; color: string } {
  if (p.statusA === 'error' || p.statusB === 'error') return { label: 'Lỗi', color: 'volcano' };
  if (p.statusA === 'drift' || p.statusB === 'drift' || (p.driftBC ?? 0) !== 0) return { label: 'Lệch', color: 'red' };
  if (p.statusA === 'warning' || p.statusB === 'warning') return { label: 'Cảnh báo', color: 'gold' };
  if ((p.ingestLagMs ?? 0) > LAGGING_MS || (p.transmuteLagMs ?? 0) > LAGGING_MS) return { label: 'Lagging', color: 'gold' };
  return { label: 'Khớp', color: 'green' };
}

function fmtNum(v: number | null | undefined) {
  return v == null ? '—' : v.toLocaleString();
}

function fmtLag(ms: number | null | undefined) {
  if (ms == null) return '—';
  if (ms === 0) return '0';
  const m = ms / 60000;
  if (m < 60) return `${m.toFixed(0)}m`;
  if (m < 60 * 24) return `${(m / 60).toFixed(1)}h`;
  return `${(m / 60 / 24).toFixed(1)}d`;
}

function fmtDrift(v: number | null) {
  if (v == null) return <Text type="secondary">—</Text>;
  if (v === 0) return <Text style={{ color: 'green' }}>0</Text>;
  // < 0: trạm sau THIẾU data (đỏ); > 0: trạm sau thừa (orphan — vàng)
  return v < 0
    ? <Text style={{ color: 'red' }}>{v.toLocaleString()} (thiếu)</Text>
    : <Text style={{ color: '#d4a017' }}>+{v.toLocaleString()} (thừa)</Text>;
}
// Ghép rows per-segment thành pipelines theo lineage.
function buildPipelines(rows: ReconReport[]): PipelineRow[] {
  // Loại bỏ các bản ghi trùng lặp (chỉ giữ bản ghi mới nhất cho mỗi bộ shadow_schema, target_table, segment)
  const uniqueMap = new Map<string, ReconReport>();
  rows.forEach((r) => {
    const schema = r.shadow_schema || '';
    const table = r.target_table;
    const segment = r.segment || '';
    const key = segment === 'shadow_master'
      ? `${schema}::${table}::${r.master_schema || ''}::${segment}`
      : `${schema}::${table}::${segment}`;
    const existing = uniqueMap.get(key);
    if (!existing || new Date(r.checked_at) > new Date(existing.checked_at)) {
      uniqueMap.set(key, r);
    }
  });
  const dedupedRows = Array.from(uniqueMap.values());

  const aRows = dedupedRows.filter((r) => r.segment !== 'shadow_master');
  const bRows = dedupedRows.filter((r) => r.segment === 'shadow_master');
  const claimedA = new Set<number>();
  const out: PipelineRow[] = [];

  for (const b of bRows) {
    // B.source_db = "shadow_schema.shadow_table" (RunSegmentB ghi shadow FQN)
    const shadowTable = b.shadow_table || (b.source_db || '').split('.').pop() || '';
    const a = aRows.find((r) => {
      const aSchema = r.shadow_schema || '';
      const aTable = r.shadow_table || r.target_table || '';
      const bSchema = b.shadow_schema || '';
      const bTable = b.shadow_table || '';
      if (aSchema && bSchema) {
        return aTable === bTable && aSchema === bSchema;
      }
      return r.target_table === shadowTable;
    });
    if (a) claimedA.add(a.id);
    // Counts = TỔNG record thật (migration 084, đo tại thời điểm recon).
    // Fallback window-count cho report cũ chưa có totals.
    // Row A status=error → KHÔNG đếm được nguồn (unreachable/timeout) →
    // sourceTotal=null hiện "—" thay vì số 0 giả (drift "+N thừa" rác).
    const aCountable = a && a.status !== 'error';
    const sourceTotal = aCountable ? (a.total_source_count ?? a.source_count ?? null) : null;
    // Shadow: dest_count = active (Go tier_a đã trừ tombstone). Fallback b.source_count.
    const shadowActive = (aCountable ? a.dest_count : null) ?? b.source_count ?? null;
    // Master: dest_count = active (Go tier_b đã trừ tombstone).
    // Fallback total_dest_count cho report cũ chưa có active logic.
    const masterActive = b.dest_count ?? b.total_dest_count ?? null;
    out.push({
      key: `b-${b.id}`,
      sourceName: a ? `${a.source_db}.${a.source_table || a.target_table}` : '—',
      sourceDb: a?.source_db ?? '',
      sourceConnector: a?.source_connection_code ?? b.source_connection_code ?? null,
      shadowName: b.shadow_schema && b.shadow_table ? `${b.shadow_schema}.${b.shadow_table}` : (b.source_db || shadowTable),
      // Master hiển thị FQN schema.table (master_schema enrich từ master_binding)
      masterName: b.master_schema ? `${b.master_schema}.${b.target_table}` : b.target_table,
      sourceCount: sourceTotal,
      shadowCount: shadowActive,
      masterCount: masterActive,
      driftAB: sourceTotal != null && shadowActive != null ? shadowActive - sourceTotal : null,
      // transmute drift = master_active − shadow_active (cùng phương pháp, đã trừ tombstone ở Go).
      driftBC: shadowActive != null && masterActive != null
        ? masterActive - shadowActive
        : -(b.diff ?? 0),
      ingestLagMs: a?.ingest_lag_ms ?? b.ingest_lag_ms ?? null,
      transmuteLagMs: b.transmute_lag_ms ?? null,
      lastReconAt: [a?.checked_at, b.checked_at].filter(Boolean).sort().pop() || b.checked_at,
      statusA: a?.status,
      statusB: b.status,
      missingB: b.missing_count || 0,
      staleB: b.stale_count || 0,
      rowA: a,
      rowB: b,
    });
  }
  // A chưa có master binding → vẫn hiện pipeline (master = —)
  for (const a of aRows) {
    if (claimedA.has(a.id)) continue;
    // status=error → nguồn không đếm được → null ("—"), không phải 0 giả.
    const aCountable = a.status !== 'error';
    const sourceTotal = aCountable ? (a.total_source_count ?? a.source_count ?? null) : null;
    // dest_count = active rows (exclude tombstones) — same logic as segment B path above.
    const shadowTotal = aCountable ? (a.dest_count ?? null) : null;
    out.push({
      key: `a-${a.id}`,
      sourceName: `${a.source_db}.${a.source_table || a.target_table}`,
      sourceDb: a.source_db ?? '',
      sourceConnector: a.source_connection_code ?? null,
      shadowName: a.shadow_schema ? `${a.shadow_schema}.${a.shadow_table}` : a.target_table,
      masterName: null,
      sourceCount: sourceTotal,
      shadowCount: shadowTotal,
      masterCount: null,
      driftAB: sourceTotal != null && shadowTotal != null ? shadowTotal - sourceTotal : null,
      driftBC: null,
      ingestLagMs: a.ingest_lag_ms ?? null,
      transmuteLagMs: null,
      lastReconAt: a.checked_at,
      statusA: a.status,
      missingB: 0,
      staleB: 0,
      rowA: a,
    });
  }
  return out.sort((x, y) => x.shadowName.localeCompare(y.shadowName));
}

// ---- Tầng 2: Drawer drill-down -----------------------------------------

function FlowStation({ title, count, sub }: { title: string; count: number | null; sub?: string }) {
  return (
    <Card size="small" style={{ width: 180, textAlign: 'center', background: '#fafafa' }}>
      <Space direction="vertical" size={2}>
        <Space size={4}><DatabaseOutlined /><Text strong>{title}</Text></Space>
        <Title level={4} style={{ margin: 0 }}>{fmtNum(count)}</Title>
        {sub ? <Text type="secondary" style={{ fontSize: 11 }}>{sub}</Text> : null}
      </Space>
    </Card>
  );
}

function FlowEdge({ label, lag }: { label: string; lag: number | null }) {
  const ms = lag ?? 0;
  const color = lag == null ? '#999' : ms === 0 ? 'green' : ms > LAGGING_MS ? 'red' : '#d4a017';
  return (
    <Space direction="vertical" size={0} style={{ textAlign: 'center', minWidth: 130 }}>
      <ArrowRightOutlined style={{ fontSize: 20, color }} />
      <Text type="secondary" style={{ fontSize: 11 }}>{label}</Text>
      <Text style={{ fontSize: 12, color }}>Lag: {fmtLag(lag)}</Text>
    </Space>
  );
}

function DrillDown({
  pipeline,
  onCheckTable,
  onHeal,
  onExecuteHeal,
  onPrune,
}: {
  pipeline: PipelineRow;
  onCheckTable?: (record: ReconReport) => void;
  onHeal?: (record: ReconReport) => void;
  onExecuteHeal?: (record: ReconReport) => void;
  onPrune?: (record: ReconReport) => void;
}) {
  // Migration 085: history theo KHÓA pipeline (shadow_schema, shadow_table) — gộp ĐỦ
  // segment A+B + hết collision target_table bare (cũ: masterName→chỉ ra segment B, mất A).
  // shadow_table có ở cả rowA (seg A) lẫn rowB (seg B, stamp mới). Fallback target_table cũ.
  const historyTable = pipeline.rowA?.shadow_table || pipeline.rowB?.shadow_table || pipeline.rowA?.target_table || null;
  const historySchema = pipeline.rowA?.shadow_schema || pipeline.rowB?.shadow_schema || null;
  // 1 shadow fan-out nhiều master → scope segment B theo master của ĐÚNG pipeline này
  // (segment A source→shadow dùng chung). rowB.target_table = master table của pipeline.
  const historyMaster = pipeline.rowB?.target_table || null;
  const { data: history, isLoading } = useTableHistory(historyTable, historySchema, historyMaster);

  // Convergence chart: mỗi phiên recon = 1 điểm; 2 đường (trạm trước vs trạm sau).
  const chartData = useMemo(() => {
    const rows = (history?.data || []).slice().reverse();
    return rows.map((r) => ({
      time: new Date(r.checked_at).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' }),
      [r.segment === 'shadow_master' ? 'shadow' : 'source']: r.source_count,
      [r.segment === 'shadow_master' ? 'master' : 'shadow']: r.dest_count,
    }));
  }, [history]);

  const levelLabel = (r: ReconReport) => {
    if (r.segment === 'shadow_master') return r.field_diffs ? 'Segment B (Deep)' : 'Segment B';
    return `Level ${r.tier}`;
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }} size={20}>
      {/* A — Flow map: 3 trạm + 2 chặng */}
      <Card size="small" title="Sơ đồ luồng dữ liệu (tổng record thật tại thời điểm recon gần nhất)">
        <Space align="center" style={{ width: '100%', justifyContent: 'center' }} wrap>
          <FlowStation title="DB SOURCE" count={pipeline.sourceCount} sub={pipeline.sourceName} />
          <FlowEdge label="Debezium CDC" lag={pipeline.ingestLagMs} />
          <FlowStation title="DB SHADOW" count={pipeline.shadowCount} sub={pipeline.shadowName} />
          <FlowEdge label="Transmute Worker" lag={pipeline.transmuteLagMs} />
          <FlowStation
            title="DB MASTER"
            count={pipeline.masterCount}
            sub={pipeline.masterName
              ? `${pipeline.masterName}${pipeline.missingB ? ` • thiếu ${pipeline.missingB}` : ''}${pipeline.staleB ? ` • stale ${pipeline.staleB}` : ''}`
              : 'chưa có binding'}
          />
        </Space>
      </Card>

      {/* Thao tác vận hành */}
      <Card size="small" title="Thao tác vận hành">
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <div style={{ padding: '12px', border: '1px solid #f0f0f0', borderRadius: '8px', height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
              <div>
                <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8, color: '#262626' }}>Chặng Ingest (Source → Shadow)</div>
                <div style={{ marginBottom: 12 }}>
                  Trạng thái: <Tag color={pipeline.rowA ? (STATUS_COLOR[pipeline.rowA.status] || 'default') : 'default'}>
                    {pipeline.rowA ? (STATUS_LABEL_VI[pipeline.rowA.status] || pipeline.rowA.status) : 'Chưa có dữ liệu'}
                  </Tag>
                </div>
              </div>
              <Space wrap>
                {pipeline.rowA && onCheckTable && (
                  <Button
                    size="small"
                    icon={<SyncOutlined />}
                    onClick={() => onCheckTable(pipeline.rowA!)}
                  >
                    Kiểm tra (Tier 2)
                  </Button>
                )}
                {pipeline.rowA && onHeal && (
                  <Button
                    size="small"
                    type="primary"
                    danger
                    icon={<MedicineBoxOutlined />}
                    disabled={!(pipeline.rowA.status === 'drift' || pipeline.rowA.status === 'dest_missing' || pipeline.rowA.status === 'warning')}
                    onClick={() => onHeal(pipeline.rowA!)}
                  >
                    Chữa lành
                  </Button>
                )}
                {pipeline.rowA && onExecuteHeal && (
                  <Button
                    size="small"
                    type="primary"
                    icon={<ThunderboltOutlined />}
                    disabled={!(pipeline.rowA.status === 'drift' || pipeline.rowA.status === 'dest_missing' || pipeline.rowA.status === 'warning')}
                    onClick={() => onExecuteHeal(pipeline.rowA!)}
                  >
                    Thực thi chữa lành
                  </Button>
                )}
                {pipeline.rowA && onPrune && (
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    disabled={!(pipeline.rowA.status === 'drift' || pipeline.rowA.status === 'warning')}
                    onClick={() => onPrune(pipeline.rowA!)}
                    title="Soft-delete shadow row có _source_id không còn ở source"
                  >
                    Prune shadow
                  </Button>
                )}
              </Space>
            </div>
          </Col>
          <Col span={12}>
            <div style={{ padding: '12px', border: '1px solid #f0f0f0', borderRadius: '8px', height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
              <div>
                <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8, color: '#262626' }}>Chặng Transmute (Shadow → Master)</div>
                {pipeline.masterName ? (
                  <div style={{ marginBottom: 12 }}>
                    Trạng thái: <Tag color={pipeline.rowB ? (STATUS_COLOR[pipeline.rowB.status] || 'default') : 'default'}>
                      {pipeline.rowB ? (STATUS_LABEL_VI[pipeline.rowB.status] || pipeline.rowB.status) : 'Chưa có dữ liệu'}
                    </Tag>
                  </div>
                ) : (
                  <div style={{ marginBottom: 12 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>Chưa cấu hình master binding</Text>
                  </div>
                )}
              </div>
              {pipeline.masterName && (
                <Space wrap>
                  {pipeline.rowB && onCheckTable && (
                    <Button
                      size="small"
                      icon={<SyncOutlined />}
                      onClick={() => onCheckTable(pipeline.rowB!)}
                    >
                      Kiểm tra (Tier 2)
                    </Button>
                  )}
                  {pipeline.rowB && onHeal && (
                    <Button
                      size="small"
                      type="primary"
                      danger
                      icon={<MedicineBoxOutlined />}
                      disabled={!(pipeline.rowB.status === 'drift' || pipeline.rowB.status === 'dest_missing' || pipeline.rowB.status === 'warning')}
                      onClick={() => onHeal(pipeline.rowB!)}
                    >
                      Chữa lành
                    </Button>
                  )}
                  {pipeline.rowB && onExecuteHeal && (
                    <Button
                      size="small"
                      type="primary"
                      icon={<ThunderboltOutlined />}
                      disabled={!(pipeline.rowB.status === 'drift' || pipeline.rowB.status === 'dest_missing' || pipeline.rowB.status === 'warning')}
                      onClick={() => onExecuteHeal(pipeline.rowB!)}
                    >
                      Thực thi chữa lành
                    </Button>
                  )}
                  {pipeline.rowB && onPrune && (
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      disabled={!(pipeline.rowB.status === 'drift' || pipeline.rowB.status === 'warning')}
                      onClick={() => onPrune(pipeline.rowB!)}
                      title="Soft-delete master row có _shadow_id không còn ở shadow"
                    >
                      Prune master
                    </Button>
                  )}
                </Space>
              )}
            </div>
          </Col>
        </Row>
      </Card>

      {/* B — Convergence chart từ lịch sử phiên recon */}
      <Card size="small" title={
        <Space size={4}>
          Biến động số lượng theo phiên recon
          <Tooltip title="Các đường bám sát nhau = không lag. Đường trạm sau rớt xuống/đi ngang = tắc ở chặng đó.">
            <InfoCircleOutlined />
          </Tooltip>
        </Space>
      }>
        {isLoading ? <Spin /> : chartData.length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có lịch sử" /> : (
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" fontSize={11} />
              <YAxis fontSize={11} width={80} tickFormatter={(v: number) => v.toLocaleString()} />
              <RechartsTooltip />
              <Legend />
              <Line type="monotone" dataKey="source" stroke="#1677ff" dot={false} connectNulls name="Source" />
              <Line type="monotone" dataKey="shadow" stroke="#722ed1" dot={false} connectNulls name="Shadow" />
              <Line type="monotone" dataKey="master" stroke="#fa541c" dot={false} connectNulls name="Master" />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Card>

      {/* C — Nhật ký phiên recon của riêng pipeline này */}
      <Card size="small" title="Nhật ký đối soát (30 phiên gần nhất)">
        <Table<ReconReport>
          size="small"
          rowKey="id"
          loading={isLoading}
          dataSource={history?.data || []}
          pagination={false}
          scroll={{ y: 260 }}
          columns={[
            {
              title: 'Phiên lúc', dataIndex: 'checked_at', width: 150,
              render: (v: string) => v ? new Date(v).toLocaleString('vi-VN') : '—',
            },
            { title: 'Loại scan', width: 130, render: (_: unknown, r: ReconReport) => <Tag>{levelLabel(r)}</Tag> },
            {
              title: 'Kết quả', dataIndex: 'status', width: 90,
              render: (s: string) => <Tag color={s === 'ok' ? 'green' : s === 'drift' ? 'red' : 'volcano'}>{s === 'ok' ? 'KHỚP' : s === 'drift' ? 'LỆCH' : s.toUpperCase()}</Tag>,
            },
            {
              title: 'Chi tiết',
              render: (_: unknown, r: ReconReport) => (
                <Text style={{ fontSize: 12 }}>
                  {r.source_count?.toLocaleString() ?? '—'} → {r.dest_count?.toLocaleString() ?? '—'}
                  {r.missing_count ? ` • thiếu ${r.missing_count}` : ''}
                  {r.stale_count ? ` • stale ${r.stale_count}` : ''}
                  {(r as ReconReport & { healed_count?: number }).healed_count ? ` • đã heal ${(r as ReconReport & { healed_count?: number }).healed_count}` : ''}
                </Text>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}

// ---- Tầng 1: Master Grid -------------------------------------------------

interface PipelineTableDataType extends PipelineRow {
  isGroupHeader?: boolean;
  connector?: string | null;
  db?: string;
  tableCount?: number;
}

interface ReconPipelineGridProps {
  rows: ReconReport[];
  loading: boolean;
  onCheckTable?: (record: ReconReport) => void;
  onHeal?: (record: ReconReport) => void;
  onExecuteHeal?: (record: ReconReport) => void;
  onPrune?: (record: ReconReport) => void;
}

export default function ReconPipelineGrid({
  rows,
  loading,
  onCheckTable,
  onHeal,
  onExecuteHeal,
  onPrune,
}: ReconPipelineGridProps) {
  const [selected, setSelected] = useState<PipelineRow | null>(null);
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);
  const pipelines = useMemo(() => buildPipelines(rows), [rows]);

  const { data: sourceObjects } = useQuery({
    queryKey: ['source-objects-recon-grid'],
    queryFn: async () => {
      const { data } = await cmsApi.get<{ data: any[] }>('/api/v1/source-objects', { params: { page: 1, page_size: 500 } });
      return data.data || [];
    },
    refetchInterval: 30000,
  });

  const { data: masters } = useQuery({
    queryKey: ['masters-recon-grid'],
    queryFn: async () => {
      const { data } = await cmsApi.get<{ data: any[] }>('/api/v1/masters', { params: { page: 1, page_size: 500 } });
      return data.data || [];
    },
    refetchInterval: 30000,
  });

  const { data: schedules } = useQuery({
    queryKey: ['schedules-recon-grid'],
    queryFn: async () => {
      const { data } = await cmsApi.get<{ data: any[] }>('/api/v1/schedules');
      return data.data || [];
    },
    refetchInterval: 30000,
  });

  const flatData = useMemo<PipelineTableDataType[]>(() => {
    const groups: Record<string, { connector: string | null; db: string; rows: PipelineRow[] }> = {};

    pipelines.forEach(p => {
      const conn = p.sourceConnector || '';
      // Dùng sourceDb trực tiếp thay vì splitFQN — tránh lỗi khi sourceName='—'
      const db = p.sourceDb || splitFQN(p.sourceName).schema || '';
      const groupKey = `${conn}::${db}`;

      if (!groups[groupKey]) {
        groups[groupKey] = {
          connector: p.sourceConnector,
          db,
          rows: []
        };
      }
      groups[groupKey].rows.push(p);
    });

    const flatList: PipelineTableDataType[] = [];

    const groupList = Object.entries(groups).map(([groupKey, g]) => {
      const sortedRows = g.rows.sort((a, b) => a.shadowName.localeCompare(b.shadowName));
      return {
        groupKey,
        connector: g.connector,
        db: g.db,
        rows: sortedRows,
      };
    });

    // Sắp xếp các group theo connector và db
    groupList.sort((a, b) => {
      const connA = a.connector || '';
      const connB = b.connector || '';
      if (connA !== connB) return connA.localeCompare(connB);
      return (a.db || '').localeCompare(b.db || '');
    });

    groupList.forEach(({ groupKey, connector, db, rows }) => {
      const isExpanded = expandedKeys.includes(groupKey);

      flatList.push({
        key: `group-${groupKey}`,
        isGroupHeader: true,
        connector,
        db,
        tableCount: rows.length,
        sourceName: '',
        sourceDb: db,
        sourceConnector: connector,
        shadowName: '',
        masterName: null,
        sourceCount: null,
        shadowCount: null,
        masterCount: null,
        driftAB: null,
        driftBC: null,
        lastReconAt: rows[0]?.lastReconAt || '',
      } as PipelineTableDataType);

      if (isExpanded) {
        rows.forEach(r => {
          flatList.push({
            ...r,
            isGroupHeader: false,
          } as PipelineTableDataType);
        });
      }
    });

    return flatList;
  }, [pipelines, expandedKeys]);

  return (
    <>
      <Table<PipelineTableDataType>
        key={expandedKeys.length > 0 ? expandedKeys.join(',') : 'collapsed-all'}
        size="middle"
        loading={loading}
        dataSource={flatData}
        rowKey="key"
        pagination={false}
        onRow={(r) => ({
          onClick: () => {
            if (r.isGroupHeader) {
              const groupKey = `${r.connector || ''}::${r.db}`;
              setExpandedKeys(prev =>
                prev.includes(groupKey)
                  ? prev.filter(k => k !== groupKey)
                  : [...prev, groupKey]
              );
            } else {
              setSelected(r);
            }
          },
          style: { cursor: 'pointer' }
        })}
        columns={[
          {
            title: 'Group',
            key: 'group_connection',
            width: 0,
            render: (_: unknown, p) => {
              if (p.isGroupHeader) {
                const groupKey = `${p.connector || ''}::${p.db}`;
                const isExpanded = expandedKeys.includes(groupKey);
                return (
                  <Space size={8}>
                    {isExpanded ? (
                      <DownOutlined style={{ fontSize: 11, color: '#bfbfbf', cursor: 'pointer' }} />
                    ) : (
                      <RightOutlined style={{ fontSize: 11, color: '#bfbfbf', cursor: 'pointer' }} />
                    )}
                    <DatabaseOutlined style={{ fontSize: 14, color: '#1890ff' }} />
                    <Text strong style={{ fontSize: 13 }}>{p.db || '—'}</Text>
                    {p.connector && <Tag color="cyan" style={{ fontSize: 11, margin: 0 }}>{p.connector}</Tag>}
                    <Badge count={p.tableCount} style={{ backgroundColor: '#52c41a' }} />
                    <Text type="secondary" style={{ fontSize: 12, marginLeft: -4 }}>pipeline</Text>
                  </Space>
                );
              }
              return <div style={{ width: 16 }} />;
            },
            onCell: (r) => {
              if (r.isGroupHeader) {
                return { colSpan: 10 };
              }
              return {};
            }
          },
          {
            title: 'Source',
            key: 'source',
            width: 200,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => {
              // Fallback: segment A recon report chưa có → lookup từ sourceObjects
              if (p.sourceName === '—') {
                const shadowTable = (p.shadowName?.split('.') || []).pop();
                const srcObj = sourceObjects?.find((s: any) => {
                  const sFqn = `${s.shadow_schema || ''}.${s.target_table}`;
                  return sFqn === p.shadowName || s.target_table === shadowTable;
                });
                const srcTable = srcObj?.source_table || srcObj?.source_object_name;
                const srcDb = srcObj?.source_db;
                return (
                  <div style={{ lineHeight: 1.4 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: srcObj ? '#595959' : '#bfbfbf', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {srcTable || '—'}
                    </div>
                    <div style={{ fontSize: 11, color: '#8c8c8c', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {srcDb || '—'}
                    </div>
                  </div>
                );
              }
              const src = splitFQN(p.sourceName);
              return (
                <div style={{ lineHeight: 1.4 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: '#1f1f1f', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {src.table}
                  </div>
                  <div style={{ fontSize: 11, color: '#8c8c8c', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {src.schema || '—'}
                  </div>
                </div>
              );
            },
          },
          {
            title: 'Shadow',
            key: 'shadow',
            width: 180,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => {
              const shd = splitFQN(p.shadowName);
              // Tìm source object tương ứng
              const srcObj = sourceObjects?.find(s => {
                const sFqn = `${s.shadow_schema || ''}.${s.target_table}`;
                const pFqn = p.shadowName;
                return sFqn === pFqn || s.target_table === pFqn || s.target_table === (p.shadowName?.split('.').pop() || '');
              });
              const isOnstream = srcObj
                ? (srcObj.shadow_binding_id != null ? Boolean(srcObj.shadow_binding_is_active) : Boolean(srcObj.is_active))
                : false;
              return (
                <div style={{ lineHeight: 1.4 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'nowrap' }}>
                    <code style={{ fontSize: 12, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 120 }}>{shd.table}</code>
                    {isOnstream ? (
                      <Tag color="green" style={{ fontSize: 9, lineHeight: '14px', height: '16px', margin: 0, padding: '0 4px', flexShrink: 0 }}>on</Tag>
                    ) : (
                      <Tag color="default" style={{ fontSize: 9, lineHeight: '14px', height: '16px', margin: 0, padding: '0 4px', flexShrink: 0 }}>off</Tag>
                    )}
                  </div>
                  <div style={{ fontSize: 11, color: '#8c8c8c', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{shd.schema || '—'}</div>
                </div>
              );
            },
          },
          {
            title: 'Master',
            key: 'master',
            width: 180,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => {
              if (!p.masterName) {
                return <Text type="secondary" style={{ fontSize: 12 }}>—</Text>;
              }
              const mst = splitFQN(p.masterName);
              // Tìm master config tương ứng
              const mstObj = masters?.find(m => {
                const mFqn = `${m.master_schema || ''}.${m.master_name}`;
                const pFqn = p.masterName;
                return mFqn === pFqn || m.master_name === pFqn || m.master_name === p.masterName?.split('.').pop();
              });

              let syncLabel = 'Sync: Tắt';
              let syncColor = 'default';

              if (mstObj) {
                if (!mstObj.is_active) {
                  syncLabel = 'Sync: Tắt (Chưa duyệt)';
                  syncColor = 'default';
                } else {
                  const mstScheds = schedules?.filter(s => s.master_table === mstObj.master_name && s.is_enabled) || [];
                  const hasRealtime = mstScheds.some(s => s.mode === 'post_ingest');
                  const cronSched = mstScheds.find(s => s.mode === 'cron');

                  if (hasRealtime) {
                    syncLabel = 'Sync: Realtime';
                    syncColor = 'green';
                  } else if (cronSched) {
                    syncLabel = `Sync: Hẹn giờ (${cronSched.cron_expr || 'cron'})`;
                    syncColor = 'blue';
                  } else {
                    syncLabel = 'Sync: Manual';
                    syncColor = 'orange';
                  }
                }
              }

              return (
                <div style={{ lineHeight: 1.4 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'nowrap' }}>
                    <code style={{ fontSize: 12, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 110 }}>{mst.table}</code>
                    <Tag color={syncColor} style={{ fontSize: 9, lineHeight: '14px', height: '16px', margin: 0, padding: '0 4px', flexShrink: 0 }}>
                      {syncLabel}
                    </Tag>
                  </div>
                  <div style={{ fontSize: 11, color: '#8c8c8c', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{mst.schema || '—'}</div>
                </div>
              );
            },
          },
          {
            title: 'Source (recs)',
            width: 120,
            align: 'right' as const,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => fmtNum(p.sourceCount)
          },
          {
            title: 'Shadow (recs)',
            width: 120,
            align: 'right' as const,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => fmtNum(p.shadowCount)
          },
          {
            title: 'Master (recs)',
            width: 120,
            align: 'right' as const,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => fmtNum(p.masterCount)
          },
          {
            title: (
              <Space size={4}>Drift<Tooltip title="Trạm sau − trạm trước (trong window 7d). Âm = trạm sau thiếu; dương = thừa (orphan)."><InfoCircleOutlined /></Tooltip></Space>
            ),
            width: 150,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => (
              <div style={{ lineHeight: 1.6 }}>
                {p.driftAB != null ? <div style={{ fontSize: 12, whiteSpace: 'nowrap' }}>ingest: {fmtDrift(p.driftAB)}</div> : null}
                {p.driftBC != null ? <div style={{ fontSize: 12, whiteSpace: 'nowrap' }}>transmute: {fmtDrift(p.driftBC)}</div> : null}
              </div>
            ),
          },
          {
            title: 'Recon cuối lúc',
            dataIndex: 'lastReconAt',
            width: 160,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (v: string, p) => {
              if (p.isGroupHeader) return null;
              return <Text style={{ fontSize: 12 }}>{v ? new Date(v).toLocaleString('vi-VN') : '—'}</Text>;
            },
          },
          {
            title: 'Trạng thái',
            width: 110,
            onCell: (r) => r.isGroupHeader ? { colSpan: 0, style: { display: 'none' } } : {},
            render: (_: unknown, p) => {
              if (p.isGroupHeader) return null;
              const s = overallStatus(p);
              return <Tag color={s.color}>{s.label}</Tag>;
            },
          },
        ]}
      />
      <Drawer
        open={!!selected}
        onClose={() => setSelected(null)}
        width={860}
        title={selected ? `Pipeline: ${selected.shadowName} → ${selected.masterName || '(chưa có master)'}` : ''}
      >
        {selected ? (
          <DrillDown
            pipeline={selected}
            onCheckTable={onCheckTable}
            onHeal={onHeal}
            onExecuteHeal={onExecuteHeal}
            onPrune={onPrune}
          />
        ) : null}
      </Drawer>
    </>
  );
}
