---
tier: T2
workspace: fix-masters-cms-grouping-2026-11-10
suffix: masters_cms_grouping_20261110
date: 2026-11-10
status: S3_SOLUTION
---

# 09. Technical Solution — Gom nhóm hiển thị trang `/masters` trên CMS theo DB Master -> Schema/Database -> Table/Collection kèm Icon Nhóm

## 1. Bảng Tổng Hợp Các Phát Hiện (`F1..F3`) & Ánh Xạ Giải Pháp 1-1

| Mã | Mức độ | Thành phần (`file:line`) | Hiện trạng & Nguyên nhân gốc rễ (Root Cause) | Giải pháp khắc phục triệt để (1 Phương án tối ưu duy nhất) |
| :-: | :-: | :--- | :--- | :--- |
| **F1** | **HIGH** | `cdc-cms-web/src/pages/MasterRegistry.tsx:1163-1171` & `1292-1334` | `groupedMasters` hiện chỉ gom phẳng 1 tầng duy nhất theo `item.master_schema \|\| '(no schema)'`. Khi hệ thống có nhiều kết nối DB Master đích (`default_master`, `pg_analytics_01`, `mysql_master_01`, `ch_analytics_01`, `mongo_master_01`), các schema/database thuộc các DB Master khác nhau bị trộn lẫn ngang hàng, không biết nhóm nào thuộc DB Master nào trước khi mở bảng. | Tái cấu trúc thành cây phân cấp **3 tầng (`DB Master` → `Schema / Database` → `Table / Collection`)**: <br/>- **Tầng 1 (Nhóm DB Master):** Gom theo `master_connection_code \|\| 'default_master'`, kết hợp metadata từ `targetMasterConnections` (`display_name`, `engine_type`, `default_database`, `host`, `port`).<br/>- **Tầng 2 (Nhóm Schema / Database):** Gom các bảng bên trong từng DB Master theo `master_schema \|\| master_database \|\| '(default)'`.<br/>- **Tầng 3 (Table / Collection):** Hiển thị bảng `Table<MasterRow>` chứa các `master_name` thuộc đúng Schema/Database đó. |
| **F2** | **HIGH** | `cdc-cms-web/src/pages/MasterRegistry.tsx:990-1003`, `1301-1308` & `src/components/master/masterEngineConfig.ts:1-86` | (1) Header nhóm tại `MasterRegistry.tsx:1302` chỉ dùng cứng `<DatabaseOutlined />` và nhãn `'Database Master (MongoDB):'` vs `'Schema Master:'` (khiến MySQL và ClickHouse bị gọi sai là `Schema Master:`).<br/>(2) Chưa có icon đặc trưng cho từng loại DB Engine (`PostgreSQL`, `MySQL`, `ClickHouse`, `MongoDB`), chưa có icon phân biệt tầng Namespace (`Schema` vs `Database`) và tầng Object (`Table` vs `Collection`). | (1) Mở rộng `masterEngineConfig.ts` thêm metadata ngữ nghĩa theo từng Engine (`namespaceLabel`, `objectLabel`, `objectPluralLabel`, `deviconSlug`, `brandColor`).<br/>(2) Tạo component `MasterEngineIcon` (hỗ trợ logo Devicon + inline SVG chuẩn cho ClickHouse/PG/MySQL/MongoDB + fallback icon) và gắn icon cho cả 3 tầng:<br/>  • **Tầng 1 (DB Master):** Logo Engine (`PostgreSQL`, `MySQL`, `ClickHouse`, `MongoDB`) + badge màu chuẩn.<br/>  • **Tầng 2 (Schema / Database):** Icon `<PartitionOutlined />` cho `Schema` (PostgreSQL) và `<HddOutlined />` cho `Database` (MySQL / ClickHouse / MongoDB).<br/>  • **Tầng 3 (Table / Collection):** Icon `<TableOutlined />` cho `Table` (PostgreSQL / MySQL / ClickHouse) và `<AppstoreOutlined />` cho `Collection` (MongoDB). |
| **F3** | **MEDIUM** | `cdc-cms-web/src/pages/MasterRegistry.tsx:1661,1835,1903` | (1) `MasterRegistry.tsx` đang dài `1,952` dòng; nếu nhồi thêm nested Collapse và icon vào cùng file sẽ vi phạm nguyên tắc Single-Responsibility.<br/>(2) Có 3 lỗi TypeScript (`meta.tagText`, `meta.displayPlaceholder`, `meta.userPlaceholder`) làm `npx tsc -b --noEmit` báo lỗi exit code 2. | (1) Tách phần hiển thị phân cấp 3 tầng và icon sang component độc lập `src/components/master/MasterGroupedCollapse.tsx`.<br/>(2) Đồng bộ tên trường `meta.shortLabel`, `meta.displayNamePlaceholder`, `meta.defaultUserPlaceholder` tại `MasterRegistry.tsx:1661,1835,1903` để `tsc -b` pass sạch. |

---

## 2. Bản Mẫu Trực Quan UI/UX (Visual Prototype & Semantic Palette — Lesson L080)

### 2.1. Quy ước Ngữ nghĩa & Icon theo từng loại DB Master (`PostgreSQL`, `MySQL`, `ClickHouse`, `MongoDB`)

| Loại Engine (`engine_type`) | Tầng 1: Nhóm DB Master (Connection / Instance) | Tầng 2: Nhóm Namespace bên trong DB Master | Tầng 3: Vật chứa dữ liệu (Object) | Màu ngữ nghĩa (`tagColor` / `brandColor`) |
| :--- | :--- | :--- | :--- | :--- |
| **`postgresql`** | 🐘 `<MasterEngineIcon engine="postgresql" />` **`DB Master:`** `<connection_code>` (`POSTGRESQL` • `db: <default_database>`) | `<PartitionOutlined />` **`Schema:`** `<master_schema>` (`N tables`) | `<TableOutlined />` **`Table:`** `<master_name>` | `geekblue` (`#2f54eb`) |
| **`mysql`** (SQL / MariaDB) | 🐬 `<MasterEngineIcon engine="mysql" />` **`DB Master:`** `<connection_code>` (`MYSQL`) | `<HddOutlined />` **`Database (Schema):`** `<master_schema>` (`N tables`) | `<TableOutlined />` **`Table:`** `<master_name>` | `orange` (`#fa8c16`) |
| **`clickhouse`** | ⚡ `<MasterEngineIcon engine="clickhouse" />` **`DB Master:`** `<connection_code>` (`CLICKHOUSE`) | `<HddOutlined />` **`Database:`** `<master_schema>` (`N tables`) | `<TableOutlined />` **`Table:`** `<master_name>` | `purple` (`#722ed1`) |
| **`mongodb`** | 🍃 `<MasterEngineIcon engine="mongodb" />` **`DB Master:`** `<connection_code>` (`MONGODB`) | `<HddOutlined />` **`Database:`** `<master_schema>` (`N collections`) | `<AppstoreOutlined />` **`Collection:`** `<master_name>` | `green` (`#52c41a`) |

### 2.2. Sơ đồ Bố cục Phân cấp 3 Tầng trên Trang `/masters`

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ ▼ [🐘 Logo PG] DB Master: [default_master] (Default Master Connection) [POSTGRESQL] [db: postgres]       │
│                [2 schemas] [5 tables] [4 Approved] [3 Manual synced]                                     │
├──────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│   ┌────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│   │ ▼ [🔀 Icon Schema] Schema: [master_goopay]  [3 tables]  [2 Manual synced]                          │ │
│   ├────────────────────────────────────────────────────────────────────────────────────────────────────┤ │
│   │   Source         │ Shadow         │ Master Table / Collection    │ Transform │ Reviewed │ Status ...│ │
│   │   trans_his      │ shadow_goopay  │ [📅 Table] trans_his         │ 1-1 Copy  │ ...      │ Approved │ │
│   │                  │ .trans_his     │ Schema: master_goopay        │           │          │          │ │
│   └────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│   ┌────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│   │ ▶ [🔀 Icon Schema] Schema: [public]         [2 tables]  [1 Manual synced]                          │ │
│   └────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ ▼ [⚡ Logo CH] DB Master: [ch_analytics_01] (ClickHouse Analytics) [CLICKHOUSE]                          │
│                [1 database] [2 tables] [2 Approved] [2 Manual synced]                                    │
├──────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│   ┌────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│   │ ▼ [🖴 Icon DB] Database: [analytics_db]      [2 tables]  [2 Manual synced]                          │ │
│   ├────────────────────────────────────────────────────────────────────────────────────────────────────┤ │
│   │   Source         │ Shadow         │ Master Table / Collection    │ Transform │ Reviewed │ Status ...│ │
│   │   payment_log    │ shadow_pay     │ [📅 Table] payment_log       │ 1-1 Copy  │ ...      │ Approved │ │
│   │                  │ .payment_log   │ Database: analytics_db       │           │          │          │ │
│   └────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Code Demo Nguyên Văn 1-1 (Zero Placeholder `...`)

### 3.1. File 1: `cdc-cms-web/src/components/master/masterEngineConfig.ts` (Bổ sung metadata phân cấp & icon cho từng Engine)

#### Trước khi sửa (`cdc-cms-web/src/components/master/masterEngineConfig.ts:1-86`):
```ts
export type MasterEngineType = 'postgresql' | 'mongodb' | 'mysql' | 'clickhouse';

export interface MasterEngineMeta {
  value: MasterEngineType;
  label: string;
  shortLabel: string;
  tagColor: string;
  defaultPort: number;
  defaultDatabase: string;
  defaultUserPlaceholder: string;
  codePlaceholder: string;
  displayNamePlaceholder: string;
  isDatabaseScoped: boolean;
}

export const MASTER_ENGINE_CONFIGS: Record<MasterEngineType, MasterEngineMeta> = {
  postgresql: {
    value: 'postgresql',
    label: 'PostgreSQL (port 5432)',
    shortLabel: 'POSTGRESQL',
    tagColor: 'geekblue',
    defaultPort: 5432,
    defaultDatabase: 'postgres',
    defaultUserPlaceholder: 'ví dụ: postgres',
    codePlaceholder: 'ví dụ: pg_analytics_01',
    displayNamePlaceholder: 'ví dụ: PostgreSQL Analytics Cluster',
    isDatabaseScoped: false,
  },
  mongodb: {
    value: 'mongodb',
    label: 'MongoDB (port 27017)',
    shortLabel: 'MONGODB',
    tagColor: 'green',
    defaultPort: 27017,
    defaultDatabase: 'admin',
    defaultUserPlaceholder: 'ví dụ: mongo_admin',
    codePlaceholder: 'ví dụ: mongo_master_01',
    displayNamePlaceholder: 'ví dụ: MongoDB Master Cluster',
    isDatabaseScoped: true,
  },
  mysql: {
    value: 'mysql',
    label: 'MySQL / MariaDB (port 3306)',
    shortLabel: 'MYSQL',
    tagColor: 'orange',
    defaultPort: 3306,
    defaultDatabase: 'mysql',
    defaultUserPlaceholder: 'ví dụ: root',
    codePlaceholder: 'ví dụ: mysql_master_01',
    displayNamePlaceholder: 'ví dụ: MySQL Master Cluster',
    isDatabaseScoped: true,
  },
  clickhouse: {
    value: 'clickhouse',
    label: 'ClickHouse (port 9000)',
    shortLabel: 'CLICKHOUSE',
    tagColor: 'purple',
    defaultPort: 9000,
    defaultDatabase: 'default',
    defaultUserPlaceholder: 'ví dụ: default',
    codePlaceholder: 'ví dụ: ch_analytics_01',
    displayNamePlaceholder: 'ví dụ: ClickHouse Analytics Cluster',
    isDatabaseScoped: true,
  },
};

export const MASTER_ENGINE_OPTIONS = Object.values(MASTER_ENGINE_CONFIGS).map((cfg) => ({
  label: cfg.label,
  value: cfg.value,
}));

export function normalizeMasterEngineTypeUI(raw?: string | null): MasterEngineType {
  const lower = (raw || '').trim().toLowerCase();
  if (lower === 'mongodb' || lower === 'mongo') return 'mongodb';
  if (lower === 'mysql' || lower === 'mariadb') return 'mysql';
  if (lower === 'clickhouse' || lower === 'ch') return 'clickhouse';
  return 'postgresql';
}

export function getMasterEngineMeta(raw?: string | null): MasterEngineMeta {
  return MASTER_ENGINE_CONFIGS[normalizeMasterEngineTypeUI(raw)];
}

export function isDatabaseScopedMasterEngine(raw?: string | null): boolean {
  return getMasterEngineMeta(raw).isDatabaseScoped;
}
```

#### Sau khi sửa (`cdc-cms-web/src/components/master/masterEngineConfig.ts`):
```ts
export type MasterEngineType = 'postgresql' | 'mongodb' | 'mysql' | 'clickhouse';

export interface MasterEngineMeta {
  value: MasterEngineType;
  label: string;
  shortLabel: string;
  tagColor: string;
  brandColor: string;
  deviconSlug: string;
  namespaceLabel: string;
  namespacePluralLabel: string;
  objectLabel: string;
  objectPluralLabel: string;
  defaultPort: number;
  defaultDatabase: string;
  defaultUserPlaceholder: string;
  codePlaceholder: string;
  displayNamePlaceholder: string;
  isDatabaseScoped: boolean;
}

export const MASTER_ENGINE_CONFIGS: Record<MasterEngineType, MasterEngineMeta> = {
  postgresql: {
    value: 'postgresql',
    label: 'PostgreSQL (port 5432)',
    shortLabel: 'POSTGRESQL',
    tagColor: 'geekblue',
    brandColor: '#2f54eb',
    deviconSlug: 'postgresql/postgresql-original',
    namespaceLabel: 'Schema',
    namespacePluralLabel: 'schemas',
    objectLabel: 'Table',
    objectPluralLabel: 'tables',
    defaultPort: 5432,
    defaultDatabase: 'postgres',
    defaultUserPlaceholder: 'ví dụ: postgres',
    codePlaceholder: 'ví dụ: pg_analytics_01',
    displayNamePlaceholder: 'ví dụ: PostgreSQL Analytics Cluster',
    isDatabaseScoped: false,
  },
  mongodb: {
    value: 'mongodb',
    label: 'MongoDB (port 27017)',
    shortLabel: 'MONGODB',
    tagColor: 'green',
    brandColor: '#52c41a',
    deviconSlug: 'mongodb/mongodb-original',
    namespaceLabel: 'Database',
    namespacePluralLabel: 'databases',
    objectLabel: 'Collection',
    objectPluralLabel: 'collections',
    defaultPort: 27017,
    defaultDatabase: 'admin',
    defaultUserPlaceholder: 'ví dụ: mongo_admin',
    codePlaceholder: 'ví dụ: mongo_master_01',
    displayNamePlaceholder: 'ví dụ: MongoDB Master Cluster',
    isDatabaseScoped: true,
  },
  mysql: {
    value: 'mysql',
    label: 'MySQL / MariaDB (port 3306)',
    shortLabel: 'MYSQL',
    tagColor: 'orange',
    brandColor: '#fa8c16',
    deviconSlug: 'mysql/mysql-original',
    namespaceLabel: 'Database (Schema)',
    namespacePluralLabel: 'databases',
    objectLabel: 'Table',
    objectPluralLabel: 'tables',
    defaultPort: 3306,
    defaultDatabase: 'mysql',
    defaultUserPlaceholder: 'ví dụ: root',
    codePlaceholder: 'ví dụ: mysql_master_01',
    displayNamePlaceholder: 'ví dụ: MySQL Master Cluster',
    isDatabaseScoped: true,
  },
  clickhouse: {
    value: 'clickhouse',
    label: 'ClickHouse (port 9000)',
    shortLabel: 'CLICKHOUSE',
    tagColor: 'purple',
    brandColor: '#722ed1',
    deviconSlug: 'clickhouse/clickhouse-original',
    namespaceLabel: 'Database',
    namespacePluralLabel: 'databases',
    objectLabel: 'Table',
    objectPluralLabel: 'tables',
    defaultPort: 9000,
    defaultDatabase: 'default',
    defaultUserPlaceholder: 'ví dụ: default',
    codePlaceholder: 'ví dụ: ch_analytics_01',
    displayNamePlaceholder: 'ví dụ: ClickHouse Analytics Cluster',
    isDatabaseScoped: true,
  },
};

export const MASTER_ENGINE_OPTIONS = Object.values(MASTER_ENGINE_CONFIGS).map((cfg) => ({
  label: cfg.label,
  value: cfg.value,
}));

export function normalizeMasterEngineTypeUI(raw?: string | null): MasterEngineType {
  const lower = (raw || '').trim().toLowerCase();
  if (lower === 'mongodb' || lower === 'mongo') return 'mongodb';
  if (lower === 'mysql' || lower === 'mariadb') return 'mysql';
  if (lower === 'clickhouse' || lower === 'ch') return 'clickhouse';
  return 'postgresql';
}

export function getMasterEngineMeta(raw?: string | null): MasterEngineMeta {
  return MASTER_ENGINE_CONFIGS[normalizeMasterEngineTypeUI(raw)];
}

export function isDatabaseScopedMasterEngine(raw?: string | null): boolean {
  return getMasterEngineMeta(raw).isDatabaseScoped;
}
```

---

### 3.2. File 2 (Tạo mới): `cdc-cms-web/src/components/master/MasterGroupedCollapse.tsx`

```tsx
import React, { useState } from 'react';
import { Collapse, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  DatabaseOutlined,
  HddOutlined,
  PartitionOutlined,
  TableOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  type MasterEngineType,
  getMasterEngineMeta,
  normalizeMasterEngineTypeUI,
} from './masterEngineConfig';
import type { MasterConnectionItem } from '../../pages/MasterRegistry';

const { Text } = Typography;

export interface MasterGroupableRow {
  id: number;
  binding_code: string;
  master_name: string;
  master_schema: string;
  master_database?: string | null;
  master_connection_code?: string | null;
  master_engine_type?: string | null;
  source_shadow: string;
  source_database?: string | null;
  source_schema?: string | null;
  source_namespace?: string | null;
  source_table?: string | null;
  source_connection_code?: string | null;
  source_engine_type?: string | null;
  shadow_binding_id?: number | null;
  shadow_schema?: string | null;
  shadow_table?: string | null;
  physical_table_fqn?: string | null;
  transform_type: string;
  spec: unknown;
  is_active: boolean;
  schema_status: 'pending_review' | 'approved' | 'rejected' | 'failed';
  schema_reviewed_by?: string | null;
  schema_reviewed_at?: string | null;
  rejection_reason?: string | null;
  created_by?: string | null;
  created_at: string;
  updated_at: string;
  last_transmute_status?: string | null;
  last_transmute_rows?: number | null;
  last_transmute_total_rows?: number | null;
  last_transmute_trace_id?: string | null;
  last_transmute_job_id?: string | null;
  last_transmute_error?: string | null;
}

export interface MasterNamespaceGroup<T extends MasterGroupableRow = MasterGroupableRow> {
  key: string;
  namespace: string;
  rows: T[];
}

export interface MasterDbGroup<T extends MasterGroupableRow = MasterGroupableRow> {
  key: string;
  connectionCode: string;
  displayName: string;
  engineType: MasterEngineType;
  databaseName: string;
  hostPort: string;
  namespaces: MasterNamespaceGroup<T>[];
  totalRows: number;
}

export const MasterEngineIcon: React.FC<{ engine?: string | null; size?: number }> = ({
  engine,
  size = 18,
}) => {
  const [imgErr, setImgErr] = useState(false);
  const meta = getMasterEngineMeta(engine);

  if (!imgErr && meta.deviconSlug) {
    return (
      <Tooltip title={meta.label}>
        <img
          src={`https://cdn.jsdelivr.net/gh/devicons/devicon/icons/${meta.deviconSlug}.svg`}
          alt={meta.shortLabel}
          width={size}
          height={size}
          style={{ verticalAlign: 'middle', display: 'inline-block' }}
          onError={() => setImgErr(true)}
        />
      </Tooltip>
    );
  }

  if (meta.value === 'clickhouse') {
    return (
      <Tooltip title={meta.label}>
        <ThunderboltOutlined style={{ color: meta.brandColor, fontSize: size }} />
      </Tooltip>
    );
  }

  return (
    <Tooltip title={meta.label}>
      <DatabaseOutlined style={{ color: meta.brandColor, fontSize: size }} />
    </Tooltip>
  );
};

export const MasterNamespaceIcon: React.FC<{ engine?: string | null }> = ({ engine }) => {
  const meta = getMasterEngineMeta(engine);
  if (meta.isDatabaseScoped) {
    return <HddOutlined style={{ color: meta.brandColor, fontSize: 15 }} />;
  }
  return <PartitionOutlined style={{ color: meta.brandColor, fontSize: 15 }} />;
};

export const MasterObjectIcon: React.FC<{ engine?: string | null }> = ({ engine }) => {
  const normalized = normalizeMasterEngineTypeUI(engine);
  const meta = getMasterEngineMeta(normalized);
  if (normalized === 'mongodb') {
    return <AppstoreOutlined style={{ color: meta.brandColor, fontSize: 14 }} />;
  }
  return <TableOutlined style={{ color: meta.brandColor, fontSize: 14 }} />;
};

export function buildMasterHierarchyGroups<T extends MasterGroupableRow>(
  rows: T[],
  connections?: MasterConnectionItem[],
): MasterDbGroup<T>[] {
  const connMap = new Map<string, MasterConnectionItem>();
  (connections || []).forEach((c) => {
    if (c.connection_code) {
      connMap.set(c.connection_code, c);
    }
  });

  const dbGroupsMap = new Map<
    string,
    {
      connectionCode: string;
      displayName: string;
      engineType: MasterEngineType;
      databaseName: string;
      hostPort: string;
      namespaceMap: Map<string, T[]>;
      totalRows: number;
    }
  >();

  rows.forEach((row) => {
    const connCode = (row.master_connection_code || '').trim() || 'default_master';
    const connInfo = connMap.get(connCode);
    const engineType = normalizeMasterEngineTypeUI(
      row.master_engine_type || connInfo?.engine_type || 'postgresql',
    );
    const meta = getMasterEngineMeta(engineType);

    let dbGroup = dbGroupsMap.get(connCode);
    if (!dbGroup) {
      const displayName = (connInfo?.display_name || '').trim();
      const databaseName = (
        row.master_database ||
        connInfo?.default_database ||
        (meta.isDatabaseScoped ? row.master_schema : '') ||
        ''
      ).trim();
      const hostPort = connInfo?.host
        ? `${connInfo.host}:${connInfo.port || meta.defaultPort}`
        : '';

      dbGroup = {
        connectionCode: connCode,
        displayName: displayName && displayName !== connCode ? displayName : '',
        engineType,
        databaseName,
        hostPort,
        namespaceMap: new Map<string, T[]>(),
        totalRows: 0,
      };
      dbGroupsMap.set(connCode, dbGroup);
    }

    const nsName =
      (row.master_schema || '').trim() ||
      (row.master_database || '').trim() ||
      (meta.isDatabaseScoped ? '(default database)' : 'public');

    const existingNsRows = dbGroup.namespaceMap.get(nsName) || [];
    existingNsRows.push(row);
    dbGroup.namespaceMap.set(nsName, existingNsRows);
    dbGroup.totalRows += 1;
  });

  const result: MasterDbGroup<T>[] = [];
  dbGroupsMap.forEach((grp) => {
    const namespaces: MasterNamespaceGroup<T>[] = [];
    grp.namespaceMap.forEach((nsRows, nsName) => {
      const sortedRows = [...nsRows].sort((a, b) =>
        (a.master_name || '').localeCompare(b.master_name || ''),
      );
      namespaces.push({
        key: `${grp.connectionCode}::${nsName}`,
        namespace: nsName,
        rows: sortedRows,
      });
    });
    namespaces.sort((a, b) => a.namespace.localeCompare(b.namespace));

    result.push({
      key: `db::${grp.connectionCode}`,
      connectionCode: grp.connectionCode,
      displayName: grp.displayName,
      engineType: grp.engineType,
      databaseName: grp.databaseName,
      hostPort: grp.hostPort,
      namespaces,
      totalRows: grp.totalRows,
    });
  });

  result.sort((a, b) => {
    if (a.connectionCode === 'default_master') return -1;
    if (b.connectionCode === 'default_master') return 1;
    return a.connectionCode.localeCompare(b.connectionCode);
  });

  return result;
}

interface MasterGroupedCollapseProps<T extends MasterGroupableRow> {
  groups: MasterDbGroup<T>[];
  activeDbKeys: string[];
  activeNamespaceKeys: string[];
  onChangeDbKeys: (keys: string[]) => void;
  onChangeNamespaceKeys: (keys: string[]) => void;
  columns: ColumnsType<T>;
}

export function MasterGroupedCollapse<T extends MasterGroupableRow>({
  groups,
  activeDbKeys,
  activeNamespaceKeys,
  onChangeDbKeys,
  onChangeNamespaceKeys,
  columns,
}: MasterGroupedCollapseProps<T>) {
  const handleDbCollapseChange = (keys: string[] | string) => {
    onChangeDbKeys(Array.isArray(keys) ? keys : [keys]);
  };

  return (
    <Collapse
      activeKey={activeDbKeys}
      onChange={handleDbCollapseChange}
      ghost
      expandIconPlacement="end"
      items={groups.map((dbGroup) => {
        const meta = getMasterEngineMeta(dbGroup.engineType);
        const allDbRows = dbGroup.namespaces.flatMap((ns) => ns.rows);
        const approvedCount = allDbRows.filter((r) => r.schema_status === 'approved').length;
        const syncedCount = allDbRows.filter((r) => r.last_transmute_status === 'COMPLETED').length;
        const failedCount = allDbRows.filter((r) => r.last_transmute_status === 'FAILED').length;

        const dbNamespaceKeySet = new Set(dbGroup.namespaces.map((ns) => ns.key));
        const currentDbActiveNsKeys = activeNamespaceKeys.filter((k) => dbNamespaceKeySet.has(k));

        const handleNsCollapseChange = (nextKeys: string[] | string) => {
          const nextArray = Array.isArray(nextKeys) ? nextKeys : [nextKeys];
          const otherDbNsKeys = activeNamespaceKeys.filter((k) => !dbNamespaceKeySet.has(k));
          onChangeNamespaceKeys([...otherDbNsKeys, ...nextArray]);
        };

        return {
          key: dbGroup.key,
          style: {
            marginBottom: 16,
            border: '1px solid #d9d9d9',
            borderRadius: 10,
            background: '#f5f7fa',
          },
          label: (
            <Space wrap size={8}>
              <MasterEngineIcon engine={dbGroup.engineType} size={18} />
              <span style={{ fontWeight: 600 }}>DB Master:</span>
              <Tag color={dbGroup.connectionCode === 'default_master' ? 'purple' : 'cyan'} style={{ marginInlineEnd: 0, fontWeight: 600 }}>
                {dbGroup.connectionCode}
              </Tag>
              {dbGroup.displayName && (
                <Text type="secondary" style={{ fontSize: 13 }}>
                  ({dbGroup.displayName})
                </Text>
              )}
              <Tag color={meta.tagColor} style={{ marginInlineEnd: 0 }}>
                {meta.shortLabel}
              </Tag>
              {!meta.isDatabaseScoped && dbGroup.databaseName && (
                <Tag color="default" style={{ marginInlineEnd: 0 }}>
                  db: {dbGroup.databaseName}
                </Tag>
              )}
              {dbGroup.hostPort && (
                <Text type="secondary" code style={{ fontSize: 11 }}>
                  {dbGroup.hostPort}
                </Text>
              )}
              <Tag color="geekblue" style={{ marginInlineEnd: 0 }}>
                {dbGroup.namespaces.length} {dbGroup.namespaces.length === 1 ? meta.namespaceLabel.toLowerCase() : meta.namespacePluralLabel}
              </Tag>
              <Tag color="blue" style={{ marginInlineEnd: 0 }}>
                {dbGroup.totalRows} {dbGroup.totalRows === 1 ? meta.objectLabel.toLowerCase() : meta.objectPluralLabel}
              </Tag>
              {approvedCount > 0 && (
                <Tag color="green" style={{ marginInlineEnd: 0 }}>
                  {approvedCount} Approved
                </Tag>
              )}
              {syncedCount > 0 && (
                <Tag color="success" style={{ marginInlineEnd: 0 }}>
                  {syncedCount} Manual synced
                </Tag>
              )}
              {failedCount > 0 && (
                <Tag color="red" style={{ marginInlineEnd: 0 }}>
                  {failedCount} Manual sync failed
                </Tag>
              )}
            </Space>
          ),
          children: (
            <div style={{ padding: '4px 8px 8px 8px' }}>
              <Collapse
                activeKey={currentDbActiveNsKeys}
                onChange={handleNsCollapseChange}
                ghost
                expandIconPlacement="end"
                items={dbGroup.namespaces.map((nsGroup) => {
                  const nsSynced = nsGroup.rows.filter((r) => r.last_transmute_status === 'COMPLETED').length;
                  const nsFailed = nsGroup.rows.filter((r) => r.last_transmute_status === 'FAILED').length;

                  return {
                    key: nsGroup.key,
                    style: {
                      marginBottom: 10,
                      border: '1px solid #e8e8e8',
                      borderRadius: 8,
                      background: '#ffffff',
                    },
                    label: (
                      <Space wrap size={6}>
                        <MasterNamespaceIcon engine={dbGroup.engineType} />
                        <span style={{ fontWeight: 600 }}>{meta.namespaceLabel}:</span>
                        <Tag color={meta.tagColor} style={{ marginInlineEnd: 0, fontWeight: 500 }}>
                          {nsGroup.namespace}
                        </Tag>
                        <Tag color="cyan" style={{ marginInlineEnd: 0 }}>
                          {nsGroup.rows.length} {nsGroup.rows.length === 1 ? meta.objectLabel.toLowerCase() : meta.objectPluralLabel}
                        </Tag>
                        {nsSynced > 0 && (
                          <Tag color="green" style={{ marginInlineEnd: 0 }}>
                            {nsSynced} Manual synced
                          </Tag>
                        )}
                        {nsFailed > 0 && (
                          <Tag color="red" style={{ marginInlineEnd: 0 }}>
                            {nsFailed} Manual sync failed
                          </Tag>
                        )}
                      </Space>
                    ),
                    children: (
                      <Table<T>
                        className="top-aligned-table"
                        size="small"
                        dataSource={nsGroup.rows}
                        rowKey="id"
                        columns={columns}
                        pagination={false}
                        scroll={{ x: 1100 }}
                      />
                    ),
                  };
                })}
              />
            </div>
          ),
        };
      })}
    />
  );
}
```

---

### 3.3. File 3: `cdc-cms-web/src/pages/MasterRegistry.tsx`

#### (A) Khối Import (`cdc-cms-web/src/pages/MasterRegistry.tsx:1-30`)

**Trước khi sửa (`MasterRegistry.tsx:1-30`):**
```tsx
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Table, Card, Typography, Space, Button, Tag, Modal, Input, Select, message, Alert,
  Switch, Tooltip, Collapse, Badge, Progress, Drawer, Popconfirm,
} from 'antd';
import type { ShadowBindingRow } from '../types';
import {
  ReloadOutlined, PlusOutlined, CheckCircleOutlined, CloseCircleOutlined,
  DatabaseOutlined, SettingOutlined, SyncOutlined, InfoCircleOutlined, EditOutlined,
  LoadingOutlined, StopOutlined, CopyOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { cmsApi } from '../services/api';
import { useAsyncJob } from '../hooks/useAsyncJob';
import { humanizeApiError } from '../utils/apiError';
import ConfirmDestructiveModal from '../components/ConfirmDestructiveModal';
import { CreateMasterModal } from '../components/master/CreateMasterModal';
import {
  type MasterEngineType,
  MASTER_ENGINE_OPTIONS,
  getMasterEngineMeta,
  isDatabaseScopedMasterEngine,
  normalizeMasterEngineTypeUI,
} from '../components/master/masterEngineConfig';
import { MultiShadowPicker, MultiShadowSummaryCell, formatTransformTypeLabel } from '../components/master/MultiShadowPicker';
import { MasterSpecFieldsControl, applyTransformTypeToSpec } from '../components/master/MasterSpecFieldsControl';
import { showActionToast } from '../utils/actionToast';

const { Title, Text } = Typography;
```

**Sau khi sửa (`MasterRegistry.tsx:1-35`):**
```tsx
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Table, Card, Typography, Space, Button, Tag, Modal, Input, Select, message, Alert,
  Switch, Tooltip, Badge, Progress, Drawer, Popconfirm,
} from 'antd';
import type { ShadowBindingRow } from '../types';
import {
  ReloadOutlined, PlusOutlined, CheckCircleOutlined, CloseCircleOutlined,
  DatabaseOutlined, SettingOutlined, SyncOutlined, InfoCircleOutlined, EditOutlined,
  LoadingOutlined, StopOutlined, CopyOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { cmsApi } from '../services/api';
import { useAsyncJob } from '../hooks/useAsyncJob';
import { humanizeApiError } from '../utils/apiError';
import ConfirmDestructiveModal from '../components/ConfirmDestructiveModal';
import { CreateMasterModal } from '../components/master/CreateMasterModal';
import {
  type MasterEngineType,
  MASTER_ENGINE_OPTIONS,
  getMasterEngineMeta,
  isDatabaseScopedMasterEngine,
  normalizeMasterEngineTypeUI,
} from '../components/master/masterEngineConfig';
import {
  MasterGroupedCollapse,
  MasterObjectIcon,
  buildMasterHierarchyGroups,
} from '../components/master/MasterGroupedCollapse';
import { MultiShadowPicker, MultiShadowSummaryCell, formatTransformTypeLabel } from '../components/master/MultiShadowPicker';
import { MasterSpecFieldsControl, applyTransformTypeToSpec } from '../components/master/MasterSpecFieldsControl';
import { showActionToast } from '../utils/actionToast';

const { Title, Text } = Typography;
```

#### (B) Cột hiển thị Table/Collection bên trong nhóm (`cdc-cms-web/src/pages/MasterRegistry.tsx:990-1003`)

**Trước khi sửa (`MasterRegistry.tsx:990-1003`):**
```tsx
    {
      title: 'DB Master',
      width: 220,
      render: (_: unknown, r: MasterRow) => (
        <Space direction="vertical" size={2}>
          <Space><Text strong>{r.master_name}</Text></Space>
          {r.master_schema && <Text type="secondary" style={{ fontSize: 12 }}>{r.master_schema}</Text>}
          {r.master_connection_code && (
            <Tag color="cyan" style={{ marginInlineEnd: 0, fontSize: 11 }}>
              <DatabaseOutlined /> {r.master_connection_code}
            </Tag>
          )}
        </Space>
      ),
    },
```

**Sau khi sửa (`MasterRegistry.tsx`):**
```tsx
    {
      title: 'Table / Collection',
      width: 230,
      render: (_: unknown, r: MasterRow) => {
        const meta = getMasterEngineMeta(r.master_engine_type);
        return (
          <Space direction="vertical" size={2}>
            <Space size={6}>
              <MasterObjectIcon engine={r.master_engine_type} />
              <Text strong>{r.master_name}</Text>
            </Space>
            {r.master_schema && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {meta.namespaceLabel}: {r.master_schema}
              </Text>
            )}
          </Space>
        );
      },
    },
```

#### (C) Logic Gom nhóm 3 tầng `DB Master -> Schema/Database -> Table` & Render Collapse (`cdc-cms-web/src/pages/MasterRegistry.tsx:1144-1336`)

**Trước khi sửa (`MasterRegistry.tsx:1144-1336`):**
```tsx
  // Bộ lọc: search (master/shadow/schema/binding) + status + transform + active.
  const filteredData = useMemo(() => {
    const txt = filterText.trim().toLowerCase();
    return (data || []).filter((m) => {
      if (filterStatus && m.schema_status !== filterStatus) return false;
      if (filterTransform && m.transform_type !== filterTransform) return false;
      if (filterActive === 'active' && !m.is_active) return false;
      if (filterActive === 'inactive' && m.is_active) return false;
      if (txt) {
        const hay = [m.master_name, m.master_schema, m.source_shadow, m.shadow_schema, m.shadow_table, m.binding_code,
        m.source_connection_code, m.source_table, m.source_database, m.source_engine_type]
          .map((x) => (x || '').toString().toLowerCase()).join(' ');
        if (!hay.includes(txt)) return false;
      }
      return true;
    });
  }, [data, filterText, filterStatus, filterTransform, filterActive]);

  // Gom master binding theo schema master (nhóm cha = schemas master), mirror trang /shadow.
  const groupedMasters = useMemo(() => {
    const groups: Record<string, MasterRow[]> = {};
    (filteredData || []).forEach((item) => {
      const key = item.master_schema || '(no schema)';
      if (!groups[key]) groups[key] = [];
      groups[key].push(item);
    });
    return groups;
  }, [filteredData]);

  const [activeKeys, setActiveKeys] = useState<string[]>([]);

  // Khởi tạo activeKeys từ localStorage, mặc định ĐÓNG TẤT CẢ ([]). Xóa mảng full-keys cũ nếu dính từ phiên trước.
  useEffect(() => {
    const saved = localStorage.getItem('master-registry-active-keys');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed) && parsed.length > 0 && parsed.length >= Object.keys(groupedMasters).length) {
          localStorage.removeItem('master-registry-active-keys');
          setActiveKeys([]);
        } else if (Array.isArray(parsed)) {
          setActiveKeys(parsed);
        }
      } catch {
        setActiveKeys([]);
      }
    }
  }, [groupedMasters]);

  const handleCollapseChange = (keys: string[] | string) => {
    const keysArray = Array.isArray(keys) ? keys : [keys];
    setActiveKeys(keysArray);
    localStorage.setItem('master-registry-active-keys', JSON.stringify(keysArray));
  };


  return (
    <Card variant="borderless">
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Master Registry</Title>
        <Space>
          <Button
            onClick={() => {
              if (activeKeys.length > 0) {
                 setActiveKeys([]);
                 localStorage.removeItem('master-registry-active-keys');
              } else {
                const keys = Object.keys(groupedMasters);
                setActiveKeys(keys);
                localStorage.setItem('master-registry-active-keys', JSON.stringify(keys));
              }
            }}
          >
            {activeKeys.length > 0 ? 'Thu gọn tất cả' : 'Mở tất cả'}
          </Button>
          <Button icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>Refresh</Button>
          <Button icon={<DatabaseOutlined />} onClick={() => setConnectionDrawerOpen(true)}>
            Quản Lý Kết Nối Đích
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            Create Master
          </Button>
        </Space>
      </Space>

      {/* Bộ lọc */}
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          allowClear
          placeholder="Tìm master / shadow / schema / binding..."
          style={{ width: 260 }}
          value={filterText}
          onChange={(e) => setFilterText(e.target.value)}
        />
        <Select
          allowClear
          placeholder="Status"
          style={{ width: 160 }}
          value={filterStatus}
          onChange={(v) => setFilterStatus(v)}
          options={[
            { value: 'pending_review', label: 'Pending review' },
            { value: 'approved', label: 'Approved' },
            { value: 'rejected', label: 'Rejected' },
            { value: 'failed', label: 'Failed' },
          ]}
        />
        <Select
          allowClear
          placeholder="Transform"
          style={{ width: 150 }}
          value={filterTransform}
          onChange={(v) => setFilterTransform(v)}
          options={TRANSFORM_TYPES.map((t) => ({ value: t, label: t }))}
        />
        <Select
          allowClear
          placeholder="Active"
          style={{ width: 120 }}
          value={filterActive}
          onChange={(v) => setFilterActive(v)}
          options={[
            { value: 'active', label: 'Active' },
            { value: 'inactive', label: 'Inactive' },
          ]}
        />
        <Text type="secondary">{filteredData.length}/{(data || []).length} master</Text>
      </Space>
      {sourceLabel && (
        <Alert
          style={{ marginTop: 16 }}
          type="info"
          showIcon
          title="Source object context"
          description={
            <span>
              Shadow hiện hành: <Text code>{sourceLabel}</Text>
              {sourceDB && sourceTable ? <> từ source <Text code>{sourceDB}.{sourceTable}</Text></> : null}.
              API hiện tại sẽ ưu tiên resolve theo <Text code>shadow_schema</Text> + <Text code>shadow_table</Text>; <Text code>source_shadow</Text> chỉ còn là compatibility fallback.
            </span>
          }
        />
      )}

      <div style={{ marginTop: 16 }}>
        {isLoading ? (
          <Table loading columns={columns} dataSource={[]} rowKey="id" pagination={false} />
        ) : (
          <Collapse
            activeKey={activeKeys}
            onChange={handleCollapseChange}
            ghost
            expandIconPlacement="end"
            items={Object.entries(groupedMasters).map(([schema, rows]) => ({
              key: schema,
              style: { marginBottom: 16, border: '1px solid #f0f0f0', borderRadius: 8, background: '#fafafa' },
              label: (
                <Space wrap>
                  <DatabaseOutlined style={{ color: '#1890ff' }} />
                  <span style={{ fontWeight: 600 }}>
                    {rows.length > 0 && rows.every((r) => (r.master_engine_type || '').toLowerCase() === 'mongodb')
                      ? 'Database Master (MongoDB):'
                      : 'Schema Master:'}
                  </span>
                  <Tag color="geekblue">{schema}</Tag>
                  <Tag color="cyan">{rows.length} bindings</Tag>
                  {(() => {
                    const syncedCount = rows.filter((r) => r.last_transmute_status === 'COMPLETED').length;
                    const failedCount = rows.filter((r) => r.last_transmute_status === 'FAILED').length;
                    return (
                      <>
                        {syncedCount > 0 && <Tag color="green">{syncedCount} Manual synced</Tag>}
                        {failedCount > 0 && <Tag color="red">{failedCount} Manual sync failed</Tag>}
                      </>
                    );
                  })()}
                </Space>
              ),
              children: (
                <Table
                  className="top-aligned-table"
                  size="small"
                  dataSource={rows}
                  rowKey="id"
                  columns={columns}
                  pagination={false}
                  scroll={{ x: 1100 }}
                />
              ),
            }))}
          />
        )}
      </div>
```

**Sau khi sửa (`MasterRegistry.tsx`):**
```tsx
  // Bộ lọc: search (master/shadow/schema/binding/connection) + status + transform + active.
  const filteredData = useMemo(() => {
    const txt = filterText.trim().toLowerCase();
    return (data || []).filter((m) => {
      if (filterStatus && m.schema_status !== filterStatus) return false;
      if (filterTransform && m.transform_type !== filterTransform) return false;
      if (filterActive === 'active' && !m.is_active) return false;
      if (filterActive === 'inactive' && m.is_active) return false;
      if (txt) {
        const hay = [
          m.master_name,
          m.master_schema,
          m.master_database,
          m.master_connection_code,
          m.master_engine_type,
          m.source_shadow,
          m.shadow_schema,
          m.shadow_table,
          m.binding_code,
          m.source_connection_code,
          m.source_table,
          m.source_database,
          m.source_engine_type,
        ]
          .map((x) => (x || '').toString().toLowerCase())
          .join(' ');
        if (!hay.includes(txt)) return false;
      }
      return true;
    });
  }, [data, filterText, filterStatus, filterTransform, filterActive]);

  // Gom master binding theo 3 tầng: DB Master (connection) -> Schema/Database -> Table/Collection.
  const masterHierarchyGroups = useMemo(
    () => buildMasterHierarchyGroups(filteredData, targetMasterConnections),
    [filteredData, targetMasterConnections],
  );

  const [activeDbKeys, setActiveDbKeys] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('master-registry-active-db-keys');
      const parsed = saved ? JSON.parse(saved) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  });

  const [activeNamespaceKeys, setActiveNamespaceKeys] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('master-registry-active-ns-keys');
      const parsed = saved ? JSON.parse(saved) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  });

  const handleDbKeysChange = (keys: string[]) => {
    setActiveDbKeys(keys);
    localStorage.setItem('master-registry-active-db-keys', JSON.stringify(keys));
  };

  const handleNamespaceKeysChange = (keys: string[]) => {
    setActiveNamespaceKeys(keys);
    localStorage.setItem('master-registry-active-ns-keys', JSON.stringify(keys));
  };

  const isAnyGroupExpanded = activeDbKeys.length > 0 || activeNamespaceKeys.length > 0;

  return (
    <Card variant="borderless">
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Master Registry</Title>
        <Space>
          <Button
            onClick={() => {
              if (isAnyGroupExpanded) {
                setActiveDbKeys([]);
                setActiveNamespaceKeys([]);
                localStorage.removeItem('master-registry-active-db-keys');
                localStorage.removeItem('master-registry-active-ns-keys');
              } else {
                const dbKeys = masterHierarchyGroups.map((g) => g.key);
                const nsKeys = masterHierarchyGroups.flatMap((g) => g.namespaces.map((ns) => ns.key));
                handleDbKeysChange(dbKeys);
                handleNamespaceKeysChange(nsKeys);
              }
            }}
          >
            {isAnyGroupExpanded ? 'Thu gọn tất cả' : 'Mở tất cả'}
          </Button>
          <Button icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>Refresh</Button>
          <Button icon={<DatabaseOutlined />} onClick={() => setConnectionDrawerOpen(true)}>
            Quản Lý Kết Nối Đích
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            Create Master
          </Button>
        </Space>
      </Space>

      {/* Bộ lọc */}
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          allowClear
          placeholder="Tìm DB master / schema / table / shadow..."
          style={{ width: 280 }}
          value={filterText}
          onChange={(e) => setFilterText(e.target.value)}
        />
        <Select
          allowClear
          placeholder="Status"
          style={{ width: 160 }}
          value={filterStatus}
          onChange={(v) => setFilterStatus(v)}
          options={[
            { value: 'pending_review', label: 'Pending review' },
            { value: 'approved', label: 'Approved' },
            { value: 'rejected', label: 'Rejected' },
            { value: 'failed', label: 'Failed' },
          ]}
        />
        <Select
          allowClear
          placeholder="Transform"
          style={{ width: 150 }}
          value={filterTransform}
          onChange={(v) => setFilterTransform(v)}
          options={TRANSFORM_TYPES.map((t) => ({ value: t, label: t }))}
        />
        <Select
          allowClear
          placeholder="Active"
          style={{ width: 120 }}
          value={filterActive}
          onChange={(v) => setFilterActive(v)}
          options={[
            { value: 'active', label: 'Active' },
            { value: 'inactive', label: 'Inactive' },
          ]}
        />
        <Text type="secondary">
          {filteredData.length}/{(data || []).length} master • {masterHierarchyGroups.length} DB Master
        </Text>
      </Space>
      {sourceLabel && (
        <Alert
          style={{ marginTop: 16 }}
          type="info"
          showIcon
          title="Source object context"
          description={
            <span>
              Shadow hiện hành: <Text code>{sourceLabel}</Text>
              {sourceDB && sourceTable ? <> từ source <Text code>{sourceDB}.{sourceTable}</Text></> : null}.
              API hiện tại sẽ ưu tiên resolve theo <Text code>shadow_schema</Text> + <Text code>shadow_table</Text>; <Text code>source_shadow</Text> chỉ còn là compatibility fallback.
            </span>
          }
        />
      )}

      <div style={{ marginTop: 16 }}>
        {isLoading ? (
          <Table loading columns={columns} dataSource={[]} rowKey="id" pagination={false} />
        ) : (
          <MasterGroupedCollapse
            groups={masterHierarchyGroups}
            activeDbKeys={activeDbKeys}
            activeNamespaceKeys={activeNamespaceKeys}
            onChangeDbKeys={handleDbKeysChange}
            onChangeNamespaceKeys={handleNamespaceKeysChange}
            columns={columns}
          />
        )}
      </div>
```

#### (D) Đồng bộ 3 trường `MasterEngineMeta` bị lệch tên trong `MasterRegistry.tsx:1654-1665`, `1830-1839`, `1897-1907`

**Trước khi sửa (`MasterRegistry.tsx:1654-1665`):**
```tsx
            {
              title: 'Engine',
              dataIndex: 'engine_type',
              key: 'engine_type',
              render: (engine: string) => {
                const meta = getMasterEngineMeta(engine);
                return (
                  <Tag color={meta.tagColor}>
                    {meta.tagText}
                  </Tag>
                );
              },
            },
```

**Sau khi sửa (`MasterRegistry.tsx:1654-1665`):**
```tsx
            {
              title: 'Engine',
              dataIndex: 'engine_type',
              key: 'engine_type',
              render: (engine: string) => {
                const meta = getMasterEngineMeta(engine);
                return (
                  <Tag color={meta.tagColor}>
                    {meta.shortLabel}
                  </Tag>
                );
              },
            },
```

**Trước khi sửa (`MasterRegistry.tsx:1830-1839`):**
```tsx
          <div>
            <Text strong style={{ display: 'block', marginBottom: 4 }}>
              Tên Hiển Thị (Display Name)
            </Text>
            <Input
              placeholder={getMasterEngineMeta(connForm.engine_type).displayPlaceholder}
              value={connForm.display_name}
              onChange={(e) => setConnForm({ ...connForm, display_name: e.target.value })}
            />
          </div>
```

**Sau khi sửa (`MasterRegistry.tsx:1830-1839`):**
```tsx
          <div>
            <Text strong style={{ display: 'block', marginBottom: 4 }}>
              Tên Hiển Thị (Display Name)
            </Text>
            <Input
              placeholder={getMasterEngineMeta(connForm.engine_type).displayNamePlaceholder}
              value={connForm.display_name}
              onChange={(e) => setConnForm({ ...connForm, display_name: e.target.value })}
            />
          </div>
```

**Trước khi sửa (`MasterRegistry.tsx:1897-1907`):**
```tsx
          <div style={{ display: 'flex', gap: 12 }}>
            <div style={{ flex: 1 }}>
              <Text strong style={{ display: 'block', marginBottom: 4 }}>
                Username {connForm.engine_type !== 'mongodb' && connForm.engine_type !== 'clickhouse' && <Text type="danger">*</Text>}
              </Text>
              <Input
                placeholder={getMasterEngineMeta(connForm.engine_type).userPlaceholder}
                value={connForm.username}
                onChange={(e) => setConnForm({ ...connForm, username: e.target.value })}
              />
            </div>
```

**Sau khi sửa (`MasterRegistry.tsx:1897-1907`):**
```tsx
          <div style={{ display: 'flex', gap: 12 }}>
            <div style={{ flex: 1 }}>
              <Text strong style={{ display: 'block', marginBottom: 4 }}>
                Username {connForm.engine_type !== 'mongodb' && connForm.engine_type !== 'clickhouse' && <Text type="danger">*</Text>}
              </Text>
              <Input
                placeholder={getMasterEngineMeta(connForm.engine_type).defaultUserPlaceholder}
                value={connForm.username}
                onChange={(e) => setConnForm({ ...connForm, username: e.target.value })}
              />
            </div>
```
