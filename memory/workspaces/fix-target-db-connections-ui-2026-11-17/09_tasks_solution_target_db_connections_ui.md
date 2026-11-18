# 09 Tasks Solution — Tái Cấu Trúc Bảng UI Quản Lý Kết Nối Đích (`Target Database Connections`)

## 1. Kết Quả Điều Tra Mã Nguồn (`S2`) & Phản Biện Tiền Thiết Kế (`S2.5 / DG0`)

### 1.1. Các phát hiện đã xác thực (`VERIFIED`)
- **`F1` — Cấu trúc 7 cột rời hiện tại gây chật chiều ngang Drawer (`VERIFIED`):**
  - Tại `cdc-cms-web/src/pages/MasterRegistry.tsx:1553-1683`, Drawer `Quản Lý Kết Nối Đích (Target Database Connections)` (`width={840}`) đang render `<Table<MasterConnectionItem>>` gồm **7 cột riêng biệt**:
    1. `Mã kết nối` (`lines 1586-1595`): `<Tag color={code === 'default' || code === 'default_master' ? 'purple' : 'blue'}>{code}</Tag>`.
    2. `Engine` (`lines 1596-1608`): `<Tag color={meta.tagColor}>{meta.shortLabel}</Tag>`.
    3. `Tên hiển thị` (`lines 1609-1613`): `dataIndex: 'display_name'`.
    4. `Host : Port` (`lines 1614-1621`): `<Text code>{r.host ? `${r.host}:${r.port || defaultPort}` : '(mặc định hệ thống)'}</Text>`.
    5. `Database / Schema` (`lines 1622-1631`).
    6. `Trạng thái` (`lines 1632-1641`).
    7. `Thao tác` (`lines 1642-1680`): `Test`, `Edit`, `Popconfirm` vô hiệu hoá.
- **`F2` — Hệ thống đã có sẵn `MasterEngineIcon` và bảng màu ngữ nghĩa chuẩn (`VERIFIED`):**
  - Tại `cdc-cms-web/src/components/master/MasterGroupedCollapse.tsx:75-110`, component `MasterEngineIcon` đã hỗ trợ đầy đủ SVG icon từ `meta.deviconSlug` cho cả 4 engine đích (`postgresql`, `mongodb`, `mysql`, `clickhouse` khai báo tại `masterEngineConfig.ts:22-95`), kèm fallback tự động sang `ThunderboltOutlined` / `DatabaseOutlined` theo `meta.brandColor` khi mạng nội bộ không tải được CDN.
  - Tại `cdc-cms-web/src/index.css:48-51`, class `.top-aligned-table .ant-table-cell` đã có sẵn để căn đỉnh (`vertical-align: top !important`) cho các ô bảng nhiều dòng.
- **`F3` — Edge-case dữ liệu `display_name` và `host:port` (`VERIFIED` qua `DG0` — `06_laya_summary_dg0_20261117T154926.json`):**
  - **Edge-case 1 (`display_name` trùng `connection_code`):** Tại `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:105-108`, khi tạo kết nối mới mà bỏ trống `DisplayName`, backend gán `displayName = connCode`. Do đó ở Hàng 2 của cột gộp `Mã & Tên hiển thị`, phải luôn hiển thị `(r.display_name || '').trim() || '—'` (không ẩn khi trùng `connection_code`) để đảm bảo đúng cấu trúc 2 hàng (`hàng 1: mã kết nối`, `hàng 2: tên hiển thị`).
  - **Edge-case 2 (Multi-host hoặc `host` đã kèm `:port`):** Tại `cdc-cms-service/internal/domain/master/connection_dsn.go:67-85`, MongoDB cho phép chuỗi `host` nhiều node phân tách bởi dấu phẩy hoặc đã có `:port` (`node1:27017,node2:27017`). Biểu thức cũ `${r.host}:${r.port || defaultPort}` tại `MasterRegistry.tsx:1619` sẽ nối thừa `:port` thành `node1:27017,node2:27017:27017`. Hàm thuần `formatMasterConnectionHostPort` cần kiểm tra từng phân đoạn dấu phẩy: nếu đã kết thúc bằng `:<số>` thì giữ nguyên, ngược lại nối `:${resolvedPort}`; nếu `host` rỗng thì trả về `'(mặc định hệ thống)'`.
- **`F4` — Ràng buộc kiến trúc `forbidden_ifelse_files` & `noUnusedLocals` (`VERIFIED`):**
  - Tại `cdc-cms-web/.agent-project.json:9,23`, `god_file_loc` là `600` và `src/pages/MasterRegistry.tsx` (hiện `1,895` dòng) nằm trong `forbidden_ifelse_files`. Theo `INV-4` (`review_profile.md:16`), bắt buộc tách Drawer bảng kết nối đích ra component riêng `src/components/master/TargetConnectionsDrawer.tsx` để **không thêm bất kỳ `if/else` nào vào `MasterRegistry.tsx`** và giảm kích thước `MasterRegistry.tsx` xuống `~1,774` dòng (`NOW < HEAD`).
  - Tại `cdc-cms-web/tsconfig.app.json:19-21`, `"verbatimModuleSyntax": true` và `"noUnusedLocals": true` đang bật cứng; khi chuyển Drawer sang `TargetConnectionsDrawer.tsx`, bắt buộc gỡ bỏ các import không còn dùng trong `MasterRegistry.tsx` (`Drawer`, `Popconfirm` tại dòng `4`; `DeleteOutlined` tại dòng `10`) theo bài học `L095`.

---

## 2. Bản Mẫu Trực Quan Bố Cục & Hệ Màu Ngữ Nghĩa (`L094` Visual UI Preview)

### 2.1. Đối chiếu cấu trúc cột (7 cột rời → 5 cột tinh gọn trong Drawer `840px`)

| Thứ tự | Bố cục CŨ (7 cột) | Bố cục MỚI (5 cột) | Quy cách hiển thị chi tiết & Màu ngữ nghĩa (`L094`, `L098`) |
| :---: | :--- | :--- | :--- |
| **1** | `Mã kết nối` *(cột 1)* + `Tên hiển thị` *(cột 3 — bỏ cột rời)* | **`Mã & Tên hiển thị`** | **Hàng 1:** `<Tag color={isDefault ? 'purple' : 'blue'}>{connection_code}</Tag>` (in đậm `fontWeight: 600`, không dùng `ellipsis`).<br>**Hàng 2:** `<Text type="secondary" style={{ fontSize: 12 }}>{display_name || '—'}</Text>`. |
| **2** | `Engine` *(cột 2 — bỏ cột rời)* + `Host : Port` *(cột 4)* | **`Engine & Host : Port`** | **Hàng 1:** `<MasterEngineIcon size={16} />` + `<Tag color={meta.tagColor}>{meta.shortLabel}</Tag>` (`POSTGRESQL` = `geekblue`, `MONGODB` = `green`, `MYSQL` = `orange`, `CLICKHOUSE` = `purple`).<br>**Hàng 2:** `<Text code style={{ fontSize: 12 }}>{host:port}</Text>` (hoặc `(mặc định hệ thống)` khi `host` trống). |
| **3** | `Database / Schema` *(cột 5)* | **`Database / Schema`** | Giữ nguyên: `{default_database || '-'}` + `({default_schema})` màu secondary. |
| **4** | `Trạng thái` *(cột 6)* | **`Trạng thái`** | Giữ nguyên: `<Tag color={status === 'active' ? 'success' : 'default'}>{status}</Tag>`. |
| **5** | `Thao tác` *(cột 7)* | **`Thao tác`** | Giữ nguyên 100% CRUD/Action: nút `Test` (ping kết nối), nút `Edit` (chỉnh sửa), nút `Delete` kèm `Popconfirm` (ẩn với `default` / `default_master`). |

### 2.2. Minh hoạ trực quan từng dòng trên bảng mới

```text
┌─────────────────────────────┬──────────────────────────────┬─────────────────────┬────────────┬──────────────────┐
│ Mã & Tên hiển thị           │ Engine & Host : Port         │ Database / Schema   │ Trạng thái │ Thao tác         │
├─────────────────────────────┼──────────────────────────────┼─────────────────────┼────────────┼──────────────────┤
│ [default_master] (purple)   │ 🐘 [POSTGRESQL] (geekblue)   │ cdc_master (public) │ [active]   │ [↻ Test] [✎]     │
│ Master PostgreSQL Mặc Định  │ `(mặc định hệ thống)`        │                     │            │                  │
├─────────────────────────────┼──────────────────────────────┼─────────────────────┼────────────┼──────────────────┤
│ [postgrestestingpaymentv2]  │ 🐘 [POSTGRESQL] (geekblue)   │ payment_v2 (public) │ [active]   │ [↻ Test] [✎] [🗑] │
│ Payment V2 Target DB        │ `10.200.186.210:5000`        │                     │            │                  │
├─────────────────────────────┼──────────────────────────────┼─────────────────────┼────────────┼──────────────────┤
│ [mongo_master_01] (blue)    │ 🍃 [MONGODB] (green)         │ analytics_master    │ [active]   │ [↻ Test] [✎] [🗑] │
│ MongoDB Master Cluster      │ `10.200.186.211:27017`       │ (analytics_master)  │            │                  │
└─────────────────────────────┴──────────────────────────────┴─────────────────────┴────────────┴──────────────────┘
```

---

## 3. Phương Án Thiết Kế Duy Nhất & Code Demo (`F1..F4`)

### 3.1. Bổ sung hàm thuần `formatMasterConnectionHostPort` tại `src/components/master/masterConnectionForm.ts`
- Mở rộng file `src/components/master/masterConnectionForm.ts` (từ `73` dòng lên `~90` dòng, tuân thủ `INV-2` Open-Closed, giữ nguyên 100% hàm và interface cũ):

```ts
// File: cdc-cms-web/src/components/master/masterConnectionForm.ts
export function formatMasterConnectionHostPort(
  item: Pick<MasterConnectionItem, 'host' | 'port' | 'engine_type'>,
): string {
  const rawHost = (item.host || '').trim();
  if (!rawHost) {
    return '(mặc định hệ thống)';
  }
  const defaultPort = getMasterEngineMeta(item.engine_type).defaultPort;
  const resolvedPort = item.port && item.port > 0 ? item.port : defaultPort;
  const segments = rawHost
    .split(',')
    .map((seg) => seg.trim())
    .filter(Boolean)
    .map((seg) => (/:\d+$/.test(seg) ? seg : `${seg}:${resolvedPort}`));
  return segments.length > 0 ? segments.join(',') : '(mặc định hệ thống)';
}
```

### 3.2. Tách Component đơn trách nhiệm `src/components/master/TargetConnectionsDrawer.tsx` (`INV-4`, `L095`, `L098`)
- Khai báo `useMemo` ở top-level của component (`L095`), gắn `className="top-aligned-table"` (`index.css:49`), tuyệt đối không dùng `ellipsis` trên tên thực thể kỹ thuật (`L098`):

```tsx
// File: cdc-cms-web/src/components/master/TargetConnectionsDrawer.tsx
import React, { useMemo } from 'react';
import { Alert, Button, Drawer, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DatabaseOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { getMasterEngineMeta } from './masterEngineConfig';
import {
  formatMasterConnectionHostPort,
  type MasterConnectionItem,
} from './masterConnectionForm';
import { MasterEngineIcon } from './MasterGroupedCollapse';

const { Text } = Typography;

export interface TargetConnectionsTableProps {
  connections: MasterConnectionItem[];
  testingRowId: number | null;
  onOpenEdit: (item: MasterConnectionItem) => void;
  onTestRow: (item: MasterConnectionItem) => void;
  onDeleteRow: (id: number, connectionCode: string) => void;
}

export interface TargetConnectionsDrawerProps extends TargetConnectionsTableProps {
  open: boolean;
  onClose: () => void;
  onOpenCreate: () => void;
}

export const TargetConnectionsTable: React.FC<TargetConnectionsTableProps> = ({
  connections,
  testingRowId,
  onOpenEdit,
  onTestRow,
  onDeleteRow,
}) => {
  const columns = useMemo<ColumnsType<MasterConnectionItem>>(
    () => [
      {
        title: 'Mã & Tên hiển thị',
        key: 'connection_identity',
        render: (_: unknown, r: MasterConnectionItem) => {
          const isDefault = r.connection_code === 'default' || r.connection_code === 'default_master';
          const displayName = (r.display_name || '').trim() || '—';
          return (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 3, alignItems: 'flex-start' }}>
              <Tag color={isDefault ? 'purple' : 'blue'} style={{ marginInlineEnd: 0, fontWeight: 600 }}>
                {r.connection_code}
              </Tag>
              <Text type="secondary" style={{ fontSize: 12, wordBreak: 'break-word' }}>
                {displayName}
              </Text>
            </div>
          );
        },
      },
      {
        title: 'Engine & Host : Port',
        key: 'engine_host_port',
        render: (_: unknown, r: MasterConnectionItem) => {
          const meta = getMasterEngineMeta(r.engine_type);
          return (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 3, alignItems: 'flex-start' }}>
              <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                <MasterEngineIcon engine={r.engine_type} size={16} />
                <Tag color={meta.tagColor} style={{ marginInlineEnd: 0, fontWeight: 600 }}>
                  {meta.shortLabel}
                </Tag>
              </div>
              <Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
                {formatMasterConnectionHostPort(r)}
              </Text>
            </div>
          );
        },
      },
      {
        title: 'Database / Schema',
        key: 'db_schema',
        render: (_: unknown, r: MasterConnectionItem) => (
          <span>
            {r.default_database || '-'}{' '}
            {r.default_schema && <Text type="secondary">({r.default_schema})</Text>}
          </span>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => (
          <Tag color={status === 'active' ? 'success' : 'default'}>
            {status}
          </Tag>
        ),
      },
      {
        title: 'Thao tác',
        key: 'actions',
        render: (_: unknown, r: MasterConnectionItem) => (
          <Space size={4}>
            <Tooltip title="Kiểm tra ping kết nối">
              <Button
                size="small"
                icon={<SyncOutlined spin={testingRowId === r.id} />}
                loading={testingRowId === r.id}
                onClick={() => onTestRow(r)}
              >
                Test
              </Button>
            </Tooltip>
            <Tooltip title="Chỉnh sửa thông số">
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={() => onOpenEdit(r)}
              />
            </Tooltip>
            {r.connection_code !== 'default' && r.connection_code !== 'default_master' && (
              <Popconfirm
                title="Vô hiệu hoá kết nối này?"
                description="Hệ thống sẽ kiểm tra xem có bảng Master active nào đang sử dụng không trước khi vô hiệu hoá."
                okText="Vô hiệu hoá"
                cancelText="Hủy"
                okButtonProps={{ danger: true }}
                onConfirm={() => onDeleteRow(r.id, r.connection_code)}
              >
                <Tooltip title="Vô hiệu hoá kết nối">
                  <Button size="small" danger icon={<DeleteOutlined />} />
                </Tooltip>
              </Popconfirm>
            )}
          </Space>
        ),
      },
    ],
    [testingRowId, onTestRow, onOpenEdit, onDeleteRow],
  );

  return (
    <Table<MasterConnectionItem>
      className="top-aligned-table"
      size="small"
      rowKey="id"
      dataSource={connections}
      pagination={false}
      columns={columns}
    />
  );
};

export const TargetConnectionsDrawer: React.FC<TargetConnectionsDrawerProps> = ({
  open,
  connections,
  testingRowId,
  onClose,
  onOpenCreate,
  onOpenEdit,
  onTestRow,
  onDeleteRow,
}) => (
  <Drawer
    title={
      <Space>
        <DatabaseOutlined style={{ color: '#1677ff' }} />
        <span>Quản Lý Kết Nối Đích (Target Database Connections)</span>
      </Space>
    }
    size={840}
    open={open}
    onClose={onClose}
    extra={
      <Button type="primary" icon={<PlusOutlined />} onClick={onOpenCreate}>
        Thêm Kết Nối Mới
      </Button>
    }
  >
    <Alert
      type="info"
      showIcon
      message="Target Connection Management"
      description="Quản trị các kết nối đích (PostgreSQL / MongoDB) phục vụ vật lý hoá schema Master và chạy pipeline Transmute. Worker Engine tự động nhận diện kết nối mới mà không cần khởi động lại."
      style={{ marginBottom: 16 }}
    />
    <TargetConnectionsTable
      connections={connections}
      testingRowId={testingRowId}
      onOpenEdit={onOpenEdit}
      onTestRow={onTestRow}
      onDeleteRow={onDeleteRow}
    />
  </Drawer>
);
```

### 3.3. Kết nối vào `src/pages/MasterRegistry.tsx` & Dọn Unused Imports (`NOW <= HEAD`, `0` `if/else`)
- Tại `src/pages/MasterRegistry.tsx:2-11` và `1552-1683`:

```tsx
// File: cdc-cms-web/src/pages/MasterRegistry.tsx (Before -> After)
// [1] Imports (lines 2-11 & 30): gỡ Drawer, Popconfirm, DeleteOutlined không còn dùng tại file này; import TargetConnectionsDrawer
import {
  Table, Card, Typography, Space, Button, Tag, Modal, Input, Select, message, Alert,
  Switch, Tooltip, Badge, Progress,
} from 'antd';
import type { ShadowBindingRow } from '../types';
import {
  ReloadOutlined, PlusOutlined, CheckCircleOutlined, CloseCircleOutlined,
  DatabaseOutlined, SettingOutlined, SyncOutlined, InfoCircleOutlined, EditOutlined,
  LoadingOutlined, StopOutlined, CopyOutlined,
} from '@ant-design/icons';
// ...
import { TargetConnectionsDrawer } from '../components/master/TargetConnectionsDrawer';

// [2] Thay khối <Drawer> (lines 1552-1683, 132 dòng) bằng component đơn trách nhiệm:
      {/* Target Database Connection Management Drawer */}
      <TargetConnectionsDrawer
        open={connectionDrawerOpen}
        connections={targetMasterConnections}
        testingRowId={testingRowId}
        onClose={() => setConnectionDrawerOpen(false)}
        onOpenCreate={() => openCreateConnection()}
        onOpenEdit={openEditConnection}
        onTestRow={handleTestRowConnection}
        onDeleteRow={handleDeleteConnection}
      />
```

---

## 4. Kế Hoạch Kiểm Chứng Sau Khi Nhận `APPROVE` (`S4` & `S5`)
1. Tạo git restore-point local trên repo `cdc-cms-web` trước khi sửa code (Rule #19).
2. Cài đặt 3 file đúng phạm vi thiết kế trên (`masterConnectionForm.ts`, `TargetConnectionsDrawer.tsx`, `MasterRegistry.tsx`), đảm bảo `MasterRegistry.tsx` giảm từ `1,895` xuống `~1,774` dòng (`NOW < HEAD`) và không phát sinh `if/else` trong `MasterRegistry.tsx`.
3. Chạy kiểm chứng thực tế qua `$T/evidence_run.py` theo `.agent-project.json`:
   - `npx tsc -b --noEmit`
   - `npm run lint`
   - `npm run build`
   - Kiểm chứng render logic của `formatMasterConnectionHostPort` và cấu trúc cột của `TargetConnectionsDrawer`.
4. Thực hiện `S5 Final Adversarial Audit`: kiểm tra `git diff` từng dòng + `wc -l`, chạy `/laya_router` (`DG3` + `review`), quét bảo mật `/security-agent`, lưu vào `audit_report_target_db_connections_ui.md` và chạy `verify_governance.py`.
