---
tier: T2
workspace: fix-clickhouse-master-data-types-2026-11-09
suffix: clickhouse_master_data_types_20261109
date: 2026-11-09
status: S3_SOLUTION
---

# 09. Technical Solution — Chuẩn hóa Master Data Type cho ClickHouse

## 1. Kết luận Điều tra (`S2`) & Phản biện `DG0` (`S2.5`)

Nhận định của anh **hoàn toàn chính xác 100%**: Hiện tại ở màn hình **Master Mapping Rules** (`cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`) cũng như tầng kiểm tra hợp lệ ở Backend (`cdc-cms-service`) và Worker (`centralized-data-service`), cột **Master Data Type** của bảng Master thuộc **ClickHouse** đang hiển thị và khóa cứng theo bộ kiểu dữ liệu của **PostgreSQL (`POSTGRES_DATA_TYPES`)**, dẫn tới **4 điểm bất cập (`F1..F4`)**:

| Mã | Thành phần & Vị trí (`file:line`) | Hiện trạng (`VERIFIED`) | Hậu quả thực tế |
| :--- | :--- | :--- | :--- |
| **F1** | **Frontend UI (`cdc-cms-web`)**<br/>- `src/pages/MasterMappingFieldsPage.tsx:10,49-62,762-841,1158-1165`<br/>- `src/constants/postgresDataTypes.ts:1-25` | 1. `interface MasterBinding` (`:49-62`) chưa khai báo trường `master_engine_type` (dù API `GET /api/v1/masters` tại `list_masters.go:22` đã trả về `master_engine_type: "clickhouse"`).<br/>2. Cột `Master Data Type` (`:767`) và Modal tạo Nested Mapping (`:1163`) gán cứng `const types = POSTGRES_DATA_TYPES` (`VARCHAR`, `TEXT`, `TIMESTAMPTZ`, `BIGINT`, `JSONB`, `DOUBLE PRECISION`, `BOOLEAN`...).<br/>3. Khi rule được đồng bộ từ Shadow (`mapping_rule_v2`), `record.data_type` mang kiểu PostgreSQL của Shadow (`VARCHAR`, `TEXT`, `TIMESTAMPTZ`, `BIGINT`...) và hiển thị nguyên xi kiểu PostgreSQL trên cột `Master Data Type` của ClickHouse. | Trên bảng Master ClickHouse, người vận hành chỉ thấy và chỉ chọn được kiểu PostgreSQL (`TEXT`, `VARCHAR`, `TIMESTAMPTZ`, `JSONB`...) thay vì bộ kiểu chuẩn của ClickHouse (`String`, `FixedString`, `Int64`, `Int32`, `Int16`, `Int8`, `UInt64`, `UInt32`, `UInt16`, `UInt8`, `Float64`, `Float32`, `Decimal(18, 6)`, `Decimal(38, 10)`, `Bool`, `DateTime64(3, 'UTC')`, `DateTime`, `Date32`, `Date`, `UUID`, `JSON`). |
| **F2** | **Backend CMS Validation (`cdc-cms-service`)**<br/>- `pkgs/utils/dbutil/validate.go:7-36`<br/>- `internal/api/master/master_mapping_rule_handler.go:115-120,206-212` | 1. `dbutil.IsValidDataType` (`validate.go:28-35`) chỉ cho phép tập kiểu PostgreSQL; đồng thời vòng lặp kiểm tra tham số trong ngoặc `(...)` (`validate.go:22-26`) chỉ cho phép chữ số và dấu phẩy nên chặn đứng `DateTime64(3, 'UTC')`.<br/>2. `UpdateDataType` (`:207`) ép `strings.ToUpper` làm biến dạng kiểu ClickHouse (`Int64` → `INT64`, `DateTime64(3, 'UTC')` → `DATETIME64(3, 'UTC')`). | Nếu gửi kiểu chuẩn ClickHouse (`String`, `FixedString(255)`, `Int64`, `UInt8`, `Float64`, `DateTime64(3, 'UTC')`...) lên `PUT /api/v1/master-mapping-rules/:id/data-type`, API trả về lỗi `400 Bad Request: "data_type không hợp lệ"`. |
| **F3** | **Worker Type Whitelist & ClickHouse DDL (`centralized-data-service`)**<br/>- `internal/service/shadow/type_resolver.go:41`<br/>- `internal/service/master/master_ddl_generator_clickhouse.go:47-74,104,226`<br/>- `internal/service/master/transmuter.go:946` | 1. `reTypeWhitelist` (`type_resolver.go:41`) chỉ khớp các kiểu PostgreSQL viết hoa. Nếu `r.DataType` mang kiểu ClickHouse (`String`, `Int64`, `UInt64`, `Float64`, `DateTime64(3, 'UTC')`...), `shadow.IsTypeWhitelisted` và `t.typeRes.Validate` trả về `false`, khiến `generateClickHouseDDL` (`:104`) bỏ qua cột, `reconcileClickHouseColumn` (`:226`) báo lỗi, và `TransmuterModule.loadRules` (`:946`) drop rule.<br/>2. `mapToClickHouseColumnType` (`master_ddl_generator_clickhouse.go:47-74`) chỉ viết `switch` cho tên kiểu PostgreSQL (`BIGINT`, `INTEGER`, `SMALLINT`, `TIMESTAMPTZ`...); nếu nhận trực tiếp tên kiểu ClickHouse (`Int64`, `Int32`, `UInt8`, `Float64`, `DateTime64(3, 'UTC')`, `FixedString(255)`, `UUID`...) thì rơi vào `default` và bị biến hết thành `String`! Ngoài ra `DECIMAL(18,6)` bị ép cứng thành `Decimal(38, 10)`. | Không thể lưu và chạy DDL / Transmute với tên kiểu native của ClickHouse. |
| **F4** | **Worker Runtime Coercion (`centralized-data-service`)**<br/>- `internal/service/master/transmuter_utils.go:391-400,493-496`<br/>- `internal/service/shadow/type_resolver.go:73-128` | `isTimestampColumnType` (`:391`), `isTimestampOrDateColumnType` (`:493`) chỉ kiểm tra `strings.HasPrefix(dt, "timestamp") \|\| dt == "datetime" \|\| dt == "date"`, bỏ sót `datetime64(...)` và `date32`. `isTextColumnType` (`:397`) chỉ kiểm tra `text`, `varchar`, `char`, bỏ sót `string` và `fixedstring`. | Khi rule lưu kiểu native ClickHouse (`DateTime64(3, 'UTC')`, `Date32`, `String`, `FixedString(255)`), `extractColumns` và `coerceForColumn` không nhận diện được cột thời gian/chuỗi để parse epoch/ISO-8601 sang `time.Time` hoặc ép chuỗi. |

---

## 2. Kiểm tra 5 Bước Thiết kế Chuẩn (`system_overview_and_design_pattern.md` Mục 5)

1. **Functional Group & Layer Routing:**
   - `cdc-cms-web`: `src/constants/postgresDataTypes.ts` + `src/pages/MasterMappingFieldsPage.tsx`.
   - `cdc-cms-service`: `pkgs/utils/dbutil/validate.go` + `internal/api/master/master_mapping_rule_handler.go`.
   - `centralized-data-service`: `internal/service/shadow/type_resolver.go` + `internal/service/master/master_ddl_generator_clickhouse.go` + `internal/service/master/transmuter_utils.go`.
2. **Canonical Pattern Reference:**
   - Giữ nguyên pattern `POSTGRES_DATA_TYPES` tại `src/constants/postgresDataTypes.ts:1-25` và bổ sung `CLICKHOUSE_DATA_TYPES` + hàm `normalizeClickHouseDataType`.
   - Giữ nguyên pattern `mapToClickHouseColumnType` tại `master_ddl_generator_clickhouse.go:47-74`, mở rộng hỗ trợ song song cả tên kiểu native ClickHouse lẫn tên kiểu PostgreSQL kế thừa từ Shadow.
3. **Blast-Radius = 0 (Tương thích ngược 100% trên 5 Shared Chokepoints & 3D Polymorphism):**
   - Các bảng Master `postgresql`, `mongodb`, `mysql` giữ nguyên 100% hành vi hiện hữu (`!isClickHouseMaster` đi nguyên nhánh cũ).
   - Các bảng Master `clickhouse` đã tạo trước đây (đang có `m.data_type IS NULL` kế thừa kiểu Postgres từ `v2.data_type` hoặc đã lưu kiểu Postgres) vẫn hoạt động bình thường 100% vì `normalizeClickHouseDataType` trên UI tự động hiển thị kiểu ClickHouse tương ứng (`VARCHAR`/`TEXT` → `String`, `BIGINT` → `Int64`, `TIMESTAMPTZ` → `DateTime64(3, 'UTC')`...) và `mapToClickHouseColumnType` ở Worker hỗ trợ cả 2 chiều (Postgres → ClickHouse & ClickHouse native → ClickHouse).
4. **Cross-Service Contract Check:**
   - `GET /api/v1/masters` (`list_masters.go:22`) đã trả sẵn `"master_engine_type": "clickhouse"`.
   - `PUT /api/v1/master-mapping-rules/:id/data-type` và `POST /api/v1/master-mapping-rules` chuẩn hóa qua `dbutil.NormalizeMasterDataType` & `dbutil.IsValidDataType`.
5. **Bảng Đấu nối Điểm Gọi Thực thi (`Caller Wiring` — Lesson L026):**

| Hàm mới / cập nhật | Điểm gọi (`caller file:line`) | Thời điểm kích hoạt (`Lifecycle Stage`) |
| :--- | :--- | :--- |
| `CLICKHOUSE_DATA_TYPES` & `normalizeClickHouseDataType` (`postgresDataTypes.ts`) | `MasterMappingFieldsPage.tsx:767,1163` | Khi render cột `Master Data Type` và Modal `Create Master Mapping Rule` cho bảng Master có `master_engine_type === 'clickhouse'` |
| `dbutil.IsValidDataType` & `dbutil.NormalizeMasterDataType` (`validate.go:7`) | `master_mapping_rule_handler.go:109,207` (`Save`, `UpdateDataType`) | Khi người dùng tạo mới hoặc đổi `Master Data Type` trên UI (`POST /api/v1/master-mapping-rules`, `PUT /api/v1/master-mapping-rules/:id/data-type`) |
| `shadow.IsTypeWhitelisted` & `TypeResolver.Validate` / `ValidateValue` (`type_resolver.go:41,67`) | `master_ddl_generator_clickhouse.go:104,226`, `transmuter.go:946,1817` | Khi sinh DDL (`cdc.cmd.master-create`), duyệt đổi kiểu cột (`cdc.cmd.master-alter-column`), nạp rule (`loadRules`) và kiểm tra giá trị (`extractColumns`) |
| `mapToClickHouseColumnType` (`master_ddl_generator_clickhouse.go:47`) | `master_ddl_generator_clickhouse.go:108,230` (`generateClickHouseDDL`, `reconcileClickHouseColumn`) | Khi tạo bảng ClickHouse hoặc `ALTER TABLE ... MODIFY COLUMN` trên ClickHouse |
| `isTimestampColumnType`, `isTimestampOrDateColumnType`, `isTextColumnType` (`transmuter_utils.go:391,397,493`) | `transmuter.go:1551,1810,1837` & `transmuter_utils.go:352,376` | Khi Transmuter trích xuất và ép kiểu dữ liệu từ `_raw_data` ghi sang Master |

---

## 3. Phương án Tối ưu Duy nhất & Code Demo 1-1 (Verbatim từ HEAD)

### 3.1. File 1: `cdc-cms-web/src/constants/postgresDataTypes.ts` (Xử lý F1)
Bổ sung `CLICKHOUSE_DATA_TYPES` và hàm `normalizeClickHouseDataType` để vừa cung cấp danh sách kiểu dữ liệu chuẩn ClickHouse, vừa tự động ánh xạ các kiểu PostgreSQL kế thừa từ Shadow (`VARCHAR`, `TEXT`, `BIGINT`, `TIMESTAMPTZ`...) sang đúng kiểu ClickHouse hiển thị trên UI:

```typescript
<<<<<<< TARGET (cdc-cms-web/src/constants/postgresDataTypes.ts:1-25)
export const POSTGRES_DATA_TYPES = [
  'VARCHAR',
  'VARCHAR(50)',
  'VARCHAR(100)',
  'VARCHAR(255)',
  'CHAR',
  'TEXT',
  'INTEGER',
  'BIGINT',
  'SMALLINT',
  'NUMERIC',
  'DECIMAL(18,6)',
  'DOUBLE PRECISION',
  'REAL',
  'BOOLEAN',
  'TIMESTAMP',
  'TIMESTAMPTZ',
  'DATE',
  'JSONB',
  'JSON',
  'BYTEA',
  'OBJECTID',
] as const;

export type PostgresDataType = typeof POSTGRES_DATA_TYPES[number];
======= REPLACEMENT
export const POSTGRES_DATA_TYPES = [
  'VARCHAR',
  'VARCHAR(50)',
  'VARCHAR(100)',
  'VARCHAR(255)',
  'CHAR',
  'TEXT',
  'INTEGER',
  'BIGINT',
  'SMALLINT',
  'NUMERIC',
  'DECIMAL(18,6)',
  'DOUBLE PRECISION',
  'REAL',
  'BOOLEAN',
  'TIMESTAMP',
  'TIMESTAMPTZ',
  'DATE',
  'JSONB',
  'JSON',
  'BYTEA',
  'OBJECTID',
] as const;

export type PostgresDataType = typeof POSTGRES_DATA_TYPES[number];

export const CLICKHOUSE_DATA_TYPES = [
  'String',
  'FixedString',
  'FixedString(50)',
  'FixedString(100)',
  'FixedString(255)',
  'Int64',
  'Int32',
  'Int16',
  'Int8',
  'UInt64',
  'UInt32',
  'UInt16',
  'UInt8',
  'Float64',
  'Float32',
  'Decimal(18, 6)',
  'Decimal(38, 10)',
  'Bool',
  "DateTime64(3, 'UTC')",
  'DateTime',
  'Date32',
  'Date',
  'UUID',
  'JSON',
] as const;

export type ClickHouseDataType = typeof CLICKHOUSE_DATA_TYPES[number];

export function normalizeClickHouseDataType(rawType?: string | null): {
  baseType: string;
  length?: number;
} {
  const trimmed = (rawType || '').trim();
  if (!trimmed) {
    return { baseType: 'String' };
  }
  const upper = trimmed.toUpperCase();
  const fixedMatch = upper.match(/^FIXEDSTRING\((\d+)\)$/);
  if (fixedMatch) {
    return { baseType: 'FixedString', length: Number(fixedMatch[1]) };
  }
  if (upper === 'FIXEDSTRING') {
    return { baseType: 'FixedString', length: 255 };
  }
  if (upper === 'BIGINT' || upper === 'INT64') {
    return { baseType: 'Int64' };
  }
  if (upper === 'INTEGER' || upper === 'INT' || upper === 'INT4' || upper === 'INT32') {
    return { baseType: 'Int32' };
  }
  if (upper === 'SMALLINT' || upper === 'INT2' || upper === 'INT16') {
    return { baseType: 'Int16' };
  }
  if (upper === 'INT8') {
    return { baseType: 'Int8' };
  }
  if (upper === 'UINT64') {
    return { baseType: 'UInt64' };
  }
  if (upper === 'UINT32') {
    return { baseType: 'UInt32' };
  }
  if (upper === 'UINT16') {
    return { baseType: 'UInt16' };
  }
  if (upper === 'UINT8' || upper === 'BOOLEAN') {
    return { baseType: 'UInt8' };
  }
  if (upper === 'BOOL') {
    return { baseType: 'Bool' };
  }
  if (upper === 'DOUBLE PRECISION' || upper === 'FLOAT8' || upper === 'FLOAT64') {
    return { baseType: 'Float64' };
  }
  if (upper === 'REAL' || upper === 'FLOAT4' || upper === 'FLOAT32') {
    return { baseType: 'Float32' };
  }
  const decMatch = upper.match(/^(?:DECIMAL|NUMERIC)\(\s*(\d+)\s*,\s*(\d+)\s*\)$/);
  if (decMatch) {
    return { baseType: `Decimal(${decMatch[1]}, ${decMatch[2]})` };
  }
  if (upper === 'NUMERIC' || upper === 'DECIMAL') {
    return { baseType: 'Decimal(38, 10)' };
  }
  if (
    upper === 'TIMESTAMPTZ' ||
    upper === 'TIMESTAMP WITH TIME ZONE' ||
    upper === 'TIMESTAMP' ||
    upper.startsWith('DATETIME64')
  ) {
    return { baseType: "DateTime64(3, 'UTC')" };
  }
  if (upper === 'DATETIME') {
    return { baseType: 'DateTime' };
  }
  if (upper === 'DATE32') {
    return { baseType: 'Date32' };
  }
  if (upper === 'DATE') {
    return { baseType: trimmed === 'Date' ? 'Date' : 'Date32' };
  }
  if (upper === 'UUID') {
    return { baseType: 'UUID' };
  }
  if (upper === 'JSON') {
    return { baseType: 'JSON' };
  }
  return { baseType: 'String' };
}
>>>>>>>
```

### 3.2. File 2: `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx` (Xử lý F1)
Cập nhật `MasterMappingFieldsPage.tsx` để nhận diện `isClickHouseMaster` từ `binding.master_engine_type` và hiển thị đúng bộ kiểu ClickHouse ở cả bảng chính lẫn Modal tạo rule:

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:10-16)
import { POSTGRES_DATA_TYPES } from '../constants/postgresDataTypes';
import {
  ShadowSourceBadge,
  buildShadowSourceColumnFilter,
  loadAllBindingShadowColumns,
  loadAllBindingShadowRules,
} from '../components/master/ShadowSourceBadge';
======= REPLACEMENT
import {
  POSTGRES_DATA_TYPES,
  CLICKHOUSE_DATA_TYPES,
  normalizeClickHouseDataType,
} from '../constants/postgresDataTypes';
import {
  ShadowSourceBadge,
  buildShadowSourceColumnFilter,
  loadAllBindingShadowColumns,
  loadAllBindingShadowRules,
} from '../components/master/ShadowSourceBadge';
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:49-63)
interface MasterBinding {
  id: number;
  binding_code: string;
  master_name: string;
  master_schema: string;
  master_connection_code?: string;
  shadow_schema?: string | null;
  shadow_table?: string | null;
  shadow_binding_id?: number | null;
  transform_type: string;
  spec?: Record<string, unknown> | null;
  is_active: boolean;
  schema_status: 'pending_review' | 'approved' | 'rejected' | 'failed';
}
======= REPLACEMENT
interface MasterBinding {
  id: number;
  binding_code: string;
  master_name: string;
  master_schema: string;
  master_connection_code?: string;
  master_engine_type?: string | null;
  shadow_schema?: string | null;
  shadow_table?: string | null;
  shadow_binding_id?: number | null;
  transform_type: string;
  spec?: Record<string, unknown> | null;
  is_active: boolean;
  schema_status: 'pending_review' | 'approved' | 'rejected' | 'failed';
}
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:75-77)
  const [binding, setBinding] = useState<MasterBinding | null>(null);
  const [rules, setRules] = useState<MasterRule[]>([]);
======= REPLACEMENT
  const [binding, setBinding] = useState<MasterBinding | null>(null);
  const [rules, setRules] = useState<MasterRule[]>([]);
  const isClickHouseMaster = useMemo(() => {
    const eng = (binding?.master_engine_type || '').toLowerCase().trim();
    if (eng === 'clickhouse' || eng === 'ch') return true;
    return (binding?.master_connection_code || '').toLowerCase().includes('clickhouse');
  }, [binding]);
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:483-500)
        await cmsApi.post('/api/v1/master-mapping-rules', {
          master_binding_id: masterBindingID,
          mapping_v2_id: parentRule ? parentRule.id : 0,
          source_field: parentField,
          source_path: nested,
          data_type: nestedDataType || 'TEXT',
          target_column: target,
          status: 'pending',
          is_active: true,
        });
        message.success(`Đã tạo nested mapping ${nested} → ${target} (pending)`);
        setCreateOpen(false);
        setNestedPath('');
        setNestedDataType('TEXT');
        setTargetColumnName('');
        setSelectedMappingV2ID(undefined);
        fetchRules();
======= REPLACEMENT
        const defaultNestedType = isClickHouseMaster ? 'String' : 'TEXT';
        const resolvedNestedType =
          isClickHouseMaster && (!nestedDataType || nestedDataType === 'TEXT')
            ? 'String'
            : nestedDataType || defaultNestedType;
        await cmsApi.post('/api/v1/master-mapping-rules', {
          master_binding_id: masterBindingID,
          mapping_v2_id: parentRule ? parentRule.id : 0,
          source_field: parentField,
          source_path: nested,
          data_type: resolvedNestedType,
          target_column: target,
          status: 'pending',
          is_active: true,
        });
        message.success(`Đã tạo nested mapping ${nested} → ${target} (pending)`);
        setCreateOpen(false);
        setNestedPath('');
        setNestedDataType(defaultNestedType);
        setTargetColumnName('');
        setSelectedMappingV2ID(undefined);
        fetchRules();
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:530-538)
        await cmsApi.post('/api/v1/master-mapping-rules', {
          master_binding_id: masterBindingID,
          mapping_v2_id: 0,
          source_field: formattedPath,
          data_type: 'JSONB',
          target_column: target,
          status: 'pending',
          is_active: true,
        });
======= REPLACEMENT
        await cmsApi.post('/api/v1/master-mapping-rules', {
          master_binding_id: masterBindingID,
          mapping_v2_id: 0,
          source_field: formattedPath,
          data_type: isClickHouseMaster ? 'String' : 'JSONB',
          target_column: target,
          status: 'pending',
          is_active: true,
        });
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:762-844)
      title: 'Master Data Type',
      dataIndex: 'data_type',
      width: 280,
      render: (t: string, record) => {
        if (record.id < 0) return <Tag color="orange">OBJECT</Tag>;
        const types = POSTGRES_DATA_TYPES;
        const matched = (t || '').toUpperCase().match(/^([A-Z]+)(?:\((\d+)\))?$/);
        const baseType = matched ? matched[1] : (t || '').toUpperCase();
        const length = matched && matched[2] ? Number(matched[2]) : undefined;

        const handleTypeChange = async (newBaseType: string, newLength?: number) => {
          let updatedType = newBaseType;
          if ((newBaseType === 'VARCHAR' || newBaseType === 'CHAR') && newLength) {
            updatedType = `${newBaseType}(${newLength})`;
          }
          const payload: { data_type: string; transform_fn?: string } = { data_type: updatedType };
          if (
            newBaseType !== 'JSONB' &&
            newBaseType !== 'JSON' &&
            record.transform_fn &&
            (record.transform_fn.startsWith('extjson_') || record.transform_fn === 'jsonb_passthrough')
          ) {
            payload.transform_fn = '';
          }
          try {
            await cmsApi.put(`/api/v1/master-mapping-rules/${record.id}/data-type`, payload);
            message.success(`Cập nhật kiểu dữ liệu thành ${updatedType}`);
            fetchRules();
          } catch (err) {
            message.error(humanizeApiError(err, 'Cập nhật kiểu dữ liệu thất bại'));
          }
        };

        const handleTransformChange = async (newTransformFn: string) => {
          try {
            await cmsApi.put(`/api/v1/master-mapping-rules/${record.id}/data-type`, {
              transform_fn: newTransformFn,
            });
            message.success(
              newTransformFn
                ? `Đã gắn transform_fn "${newTransformFn}" cho cột ${record.target_column}`
                : `Đã về mặc định (Canonical ExtJSON) cho cột ${record.target_column}`
            );
            fetchRules();
          } catch (err) {
            message.error(humanizeApiError(err, 'Cập nhật transform_fn thất bại'));
          }
        };

        return (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Space>
              <Select
                size="small"
                value={baseType || 'TEXT'}
                style={{ width: 250 }}
                onChange={(val) => {
                  if (val === 'VARCHAR' || val === 'CHAR') {
                    handleTypeChange(val, length || 255);
                  } else {
                    handleTypeChange(val);
                  }
                }}
                options={types.map(type => ({ label: type, value: type }))}
              />
              {(baseType === 'VARCHAR' || baseType === 'CHAR') && (
                <InputNumber
                  size="small"
                  style={{ width: 60 }}
                  min={1}
                  max={65535}
                  value={length || 255}
                  onChange={(val) => {
                    if (val) {
                      handleTypeChange(baseType, val);
                    }
                  }}
                />
              )}
            </Space>
            {(baseType === 'JSONB' || baseType === 'JSON') && (
======= REPLACEMENT
      title: 'Master Data Type',
      dataIndex: 'data_type',
      width: 280,
      render: (t: string, record) => {
        if (record.id < 0) return <Tag color="orange">OBJECT</Tag>;
        const types: readonly string[] = isClickHouseMaster ? CLICKHOUSE_DATA_TYPES : POSTGRES_DATA_TYPES;
        const chNorm = isClickHouseMaster ? normalizeClickHouseDataType(t) : null;
        const matched = !isClickHouseMaster ? (t || '').toUpperCase().match(/^([A-Z]+)(?:\((\d+)\))?$/) : null;
        const baseType = isClickHouseMaster
          ? chNorm!.baseType
          : matched
            ? matched[1]
            : (t || '').toUpperCase();
        const length = isClickHouseMaster
          ? chNorm!.length
          : matched && matched[2]
            ? Number(matched[2])
            : undefined;
        const hasLengthInput = isClickHouseMaster
          ? baseType === 'FixedString'
          : baseType === 'VARCHAR' || baseType === 'CHAR';

        const handleTypeChange = async (newBaseType: string, newLength?: number) => {
          let updatedType = newBaseType;
          if ((newBaseType === 'VARCHAR' || newBaseType === 'CHAR' || newBaseType === 'FixedString') && newLength) {
            updatedType = `${newBaseType}(${newLength})`;
          }
          const payload: { data_type: string; transform_fn?: string } = { data_type: updatedType };
          if (
            newBaseType !== 'JSONB' &&
            newBaseType !== 'JSON' &&
            !(isClickHouseMaster && newBaseType === 'String') &&
            record.transform_fn &&
            (record.transform_fn.startsWith('extjson_') || record.transform_fn === 'jsonb_passthrough')
          ) {
            payload.transform_fn = '';
          }
          try {
            await cmsApi.put(`/api/v1/master-mapping-rules/${record.id}/data-type`, payload);
            message.success(`Cập nhật kiểu dữ liệu thành ${updatedType}`);
            fetchRules();
          } catch (err) {
            message.error(humanizeApiError(err, 'Cập nhật kiểu dữ liệu thất bại'));
          }
        };

        const handleTransformChange = async (newTransformFn: string) => {
          try {
            await cmsApi.put(`/api/v1/master-mapping-rules/${record.id}/data-type`, {
              transform_fn: newTransformFn,
            });
            message.success(
              newTransformFn
                ? `Đã gắn transform_fn "${newTransformFn}" cho cột ${record.target_column}`
                : `Đã về mặc định (Canonical ExtJSON) cho cột ${record.target_column}`
            );
            fetchRules();
          } catch (err) {
            message.error(humanizeApiError(err, 'Cập nhật transform_fn thất bại'));
          }
        };

        const selectOptions = types.includes(baseType)
          ? types.map(type => ({ label: type, value: type }))
          : [{ label: baseType, value: baseType }, ...types.map(type => ({ label: type, value: type }))];

        return (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Space>
              <Select
                size="small"
                value={baseType || (isClickHouseMaster ? 'String' : 'TEXT')}
                style={{ width: 250 }}
                onChange={(val) => {
                  if (val === 'VARCHAR' || val === 'CHAR' || val === 'FixedString') {
                    handleTypeChange(val, length || 255);
                  } else {
                    handleTypeChange(val);
                  }
                }}
                options={selectOptions}
              />
              {hasLengthInput && (
                <InputNumber
                  size="small"
                  style={{ width: 60 }}
                  min={1}
                  max={65535}
                  value={length || 255}
                  onChange={(val) => {
                    if (val) {
                      handleTypeChange(baseType, val);
                    }
                  }}
                />
              )}
            </Space>
            {(baseType === 'JSONB' || baseType === 'JSON') && (
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:1100-1109)
        onCancel={() => {
          setCreateOpen(false);
          setSelectedMappingV2ID(undefined);
          setTargetColumnName('');
          setNestedPath('');
          setNestedDataType('TEXT');
          setJsonSubsetMode(false);
          setJsonSubsetFields([]);
          setJsonSubsetType('value');
        }}
======= REPLACEMENT
        onCancel={() => {
          setCreateOpen(false);
          setSelectedMappingV2ID(undefined);
          setTargetColumnName('');
          setNestedPath('');
          setNestedDataType(isClickHouseMaster ? 'String' : 'TEXT');
          setJsonSubsetMode(false);
          setJsonSubsetFields([]);
          setJsonSubsetType('value');
        }}
>>>>>>>
```

```typescript
<<<<<<< TARGET (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:1158-1165)
                <span style={{ display: 'inline-block', width: 140, fontWeight: 'bold' }}>Data type (nested):</span>
                <Select
                  style={{ width: 'calc(100% - 140px)' }}
                  value={nestedDataType}
                  onChange={setNestedDataType}
                  options={POSTGRES_DATA_TYPES.map((t) => ({ label: t, value: t }))}
                />
======= REPLACEMENT
                <span style={{ display: 'inline-block', width: 140, fontWeight: 'bold' }}>Data type (nested):</span>
                <Select
                  style={{ width: 'calc(100% - 140px)' }}
                  value={isClickHouseMaster && nestedDataType === 'TEXT' ? 'String' : nestedDataType}
                  onChange={setNestedDataType}
                  options={(isClickHouseMaster ? CLICKHOUSE_DATA_TYPES : POSTGRES_DATA_TYPES).map((t) => ({ label: t, value: t }))}
                />
>>>>>>>
```

---

### 3.3. File 3: `cdc-cms-service/pkgs/utils/dbutil/validate.go` & `internal/api/master/master_mapping_rule_handler.go` (Xử lý F2)

1. Mở rộng `dbutil.IsValidDataType` và thêm `dbutil.NormalizeMasterDataType` trong `cdc-cms-service/pkgs/utils/dbutil/validate.go`:

```go
<<<<<<< TARGET (cdc-cms-service/pkgs/utils/dbutil/validate.go:7-36)
func IsValidDataType(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || len(s) > 50 {
		return false
	}
	base := s
	if i := strings.IndexByte(s, '('); i >= 0 {
		if !strings.HasSuffix(s, ")") {
			return false
		}
		base = strings.TrimSpace(s[:i])
		inner := s[i+1 : len(s)-1]
		if inner == "" {
			return false
		}
		for _, r := range inner {
			if !((r >= '0' && r <= '9') || r == ',' || r == ' ') {
				return false
			}
		}
	}
	allowed := map[string]bool{
		"VARCHAR": true, "CHAR": true, "TEXT": true, "INTEGER": true, "INT": true,
		"BIGINT": true, "SMALLINT": true, "NUMERIC": true, "DECIMAL": true, "REAL": true,
		"DOUBLE PRECISION": true, "BOOLEAN": true, "BOOL": true, "DATE": true,
		"TIMESTAMP": true, "TIMESTAMPTZ": true, "JSONB": true, "JSON": true, "UUID": true,
		"OBJECTID": true, "OID": true,
	}
	return allowed[base]
}
======= REPLACEMENT
var canonicalClickHouseBaseTypes = map[string]string{
	"STRING":      "String",
	"FIXEDSTRING": "FixedString",
	"INT8":        "Int8",
	"INT16":       "Int16",
	"INT32":       "Int32",
	"INT64":       "Int64",
	"UINT8":       "UInt8",
	"UINT16":      "UInt16",
	"UINT32":      "UInt32",
	"UINT64":      "UInt64",
	"FLOAT32":     "Float32",
	"FLOAT64":     "Float64",
	"DATE32":      "Date32",
	"DATETIME":    "DateTime",
	"DATETIME64":  "DateTime64",
}

func NormalizeMasterDataType(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	upper := strings.ToUpper(trimmed)
	if upper == "BOOL" && trimmed == "Bool" {
		return "Bool"
	}
	if upper == "DATE" && trimmed == "Date" {
		return "Date"
	}
	if i := strings.IndexByte(trimmed, '('); i >= 0 && strings.HasSuffix(trimmed, ")") {
		baseUpper := strings.ToUpper(strings.TrimSpace(trimmed[:i]))
		inner := strings.TrimSpace(trimmed[i+1 : len(trimmed)-1])
		if canon, ok := canonicalClickHouseBaseTypes[baseUpper]; ok {
			if baseUpper == "DATETIME64" && strings.Contains(strings.ToUpper(inner), "UTC") {
				return "DateTime64(3, 'UTC')"
			}
			return canon + "(" + inner + ")"
		}
		if baseUpper == "DECIMAL" && strings.HasPrefix(trimmed, "Decimal") {
			return "Decimal(" + inner + ")"
		}
	}
	if canon, ok := canonicalClickHouseBaseTypes[upper]; ok {
		return canon
	}
	return upper
}

func IsValidDataType(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || len(s) > 50 {
		return false
	}
	base := s
	if i := strings.IndexByte(s, '('); i >= 0 {
		if !strings.HasSuffix(s, ")") {
			return false
		}
		base = strings.TrimSpace(s[:i])
		inner := strings.TrimSpace(s[i+1 : len(s)-1])
		if inner == "" {
			return false
		}
		if base == "DATETIME64" && (inner == "3, 'UTC'" || inner == "3,'UTC'" || inner == "6, 'UTC'" || inner == "6,'UTC'") {
			return true
		}
		for _, r := range inner {
			if !((r >= '0' && r <= '9') || r == ',' || r == ' ') {
				return false
			}
		}
	}
	allowed := map[string]bool{
		"VARCHAR": true, "CHAR": true, "TEXT": true, "INTEGER": true, "INT": true,
		"BIGINT": true, "SMALLINT": true, "NUMERIC": true, "DECIMAL": true, "REAL": true,
		"DOUBLE PRECISION": true, "BOOLEAN": true, "BOOL": true, "DATE": true,
		"TIMESTAMP": true, "TIMESTAMPTZ": true, "JSONB": true, "JSON": true, "UUID": true,
		"OBJECTID": true, "OID": true, "BYTEA": true,
		"STRING": true, "FIXEDSTRING": true,
		"INT8": true, "INT16": true, "INT32": true, "INT64": true,
		"UINT8": true, "UINT16": true, "UINT32": true, "UINT64": true,
		"FLOAT32": true, "FLOAT64": true,
		"DATE32": true, "DATETIME": true, "DATETIME64": true,
	}
	return allowed[base]
}
>>>>>>>
```

2. Gọi `dbutil.NormalizeMasterDataType` tại `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go`:

```go
<<<<<<< TARGET (cdc-cms-service/internal/api/master/master_mapping_rule_handler.go:205-213)
	cmd := masterCmd.UpdateColumnMetadataCommand{ID: id}
	if req.DataType != nil {
		dt := strings.ToUpper(strings.TrimSpace(*req.DataType))
		if !dbutil.IsValidDataType(dt) {
			return c.Status(400).JSON(fiber.Map{"error": "data_type không hợp lệ"})
		}
		cmd.DataType = &dt
	}
======= REPLACEMENT
	cmd := masterCmd.UpdateColumnMetadataCommand{ID: id}
	if req.DataType != nil {
		dt := dbutil.NormalizeMasterDataType(*req.DataType)
		if !dbutil.IsValidDataType(dt) {
			return c.Status(400).JSON(fiber.Map{"error": "data_type không hợp lệ"})
		}
		cmd.DataType = &dt
	}
>>>>>>>
```

---

### 3.4. File 4: `centralized-data-service/internal/service/shadow/type_resolver.go`, `internal/service/master/master_ddl_generator_clickhouse.go`, `internal/service/master/transmuter_utils.go` (Xử lý F3 & F4)

1. Cập nhật `type_resolver.go` để chấp nhận cả bộ kiểu PostgreSQL lẫn bộ kiểu ClickHouse trong `reTypeWhitelist`, `reCharLen`, `reNumericPrc`, và `ValidateValue`:

```go
<<<<<<< TARGET (centralized-data-service/internal/service/shadow/type_resolver.go:40-46)
var (
	reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|BIGINT|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|OBJECTID|OID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]{0,3},[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)

	reCharLen    = regexp.MustCompile(`^(?:VAR)?CHAR\((\d+)\)$`)
	reNumericPrc = regexp.MustCompile(`^(?:NUMERIC|DECIMAL)\((\d+),(\d+)\)$`)
	reEnumRef    = regexp.MustCompile(`^ENUM:([a-z_][a-z0-9_]{0,62})$`)
)
======= REPLACEMENT
var (
	reTypeWhitelist = regexp.MustCompile(`^(SMALLINT|INTEGER|INT|BIGINT|REAL|DOUBLE PRECISION|NUMERIC|DECIMAL|BOOLEAN|BOOL|Bool|DATE|Date|DATE32|Date32|TIME|TIMESTAMP|TIMESTAMPTZ|DATETIME|DateTime|DATETIME64\([0-9](?:,\s*'UTC')?\)|DateTime64\([0-9](?:,\s*'UTC')?\)|INTERVAL|JSON|JSONB|UUID|OBJECTID|OID|INET|CIDR|MACADDR|BYTEA|TEXT|VARCHAR|CHAR|STRING|String|FIXEDSTRING|FixedString|INT8|Int8|INT16|Int16|INT32|Int32|INT64|Int64|UINT8|UInt8|UINT16|UInt16|UINT32|UInt32|UINT64|UInt64|FLOAT32|Float32|FLOAT64|Float64|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|(?:FIXEDSTRING|FixedString)\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]{0,3},\s*[0-9][0-9]{0,3}\)|(?:DECIMAL|Decimal)\([1-9][0-9]{0,3},\s*[0-9][0-9]{0,3}\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$`)

	reCharLen    = regexp.MustCompile(`^(?:(?:VAR)?CHAR|FIXEDSTRING|FixedString)\((\d+)\)$`)
	reNumericPrc = regexp.MustCompile(`^(?:NUMERIC|DECIMAL|Decimal)\((\d+),\s*(\d+)\)$`)
	reEnumRef    = regexp.MustCompile(`^ENUM:([a-z_][a-z0-9_]{0,62})$`)
)
>>>>>>>
```

```go
<<<<<<< TARGET (centralized-data-service/internal/service/shadow/type_resolver.go:109-112)
	// Catch plain numeric types without precision bounds
	specUpper := strings.ToUpper(spec)
	if specUpper == "SMALLINT" || specUpper == "INTEGER" || specUpper == "BIGINT" || specUpper == "REAL" || specUpper == "DOUBLE PRECISION" || specUpper == "NUMERIC" || specUpper == "DECIMAL" {
======= REPLACEMENT
	// Catch plain numeric types without precision bounds
	specUpper := strings.ToUpper(spec)
	if specUpper == "SMALLINT" || specUpper == "INTEGER" || specUpper == "INT" || specUpper == "BIGINT" || specUpper == "REAL" || specUpper == "DOUBLE PRECISION" || specUpper == "NUMERIC" || specUpper == "DECIMAL" || specUpper == "INT8" || specUpper == "INT16" || specUpper == "INT32" || specUpper == "INT64" || specUpper == "UINT8" || specUpper == "UINT16" || specUpper == "UINT32" || specUpper == "UINT64" || specUpper == "FLOAT32" || specUpper == "FLOAT64" {
>>>>>>>
```

2. Cập nhật `mapToClickHouseColumnType` trong `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go` để hỗ trợ đầy đủ cả kiểu native ClickHouse lẫn kiểu PostgreSQL:

```go
<<<<<<< TARGET (centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:47-74)
func mapToClickHouseColumnType(pgType string, isNullable bool) string {
	upper := strings.ToUpper(strings.TrimSpace(pgType))
	var baseType string
	switch {
	case upper == "BIGINT" || upper == "INT8":
		baseType = "Int64"
	case upper == "INTEGER" || upper == "INT" || upper == "INT4":
		baseType = "Int32"
	case upper == "SMALLINT" || upper == "INT2":
		baseType = "Int16"
	case upper == "BOOLEAN" || upper == "BOOL":
		baseType = "UInt8"
	case upper == "DOUBLE PRECISION" || upper == "FLOAT8" || upper == "REAL" || upper == "FLOAT4":
		baseType = "Float64"
	case strings.HasPrefix(upper, "NUMERIC") || strings.HasPrefix(upper, "DECIMAL"):
		baseType = "Decimal(38, 10)"
	case upper == "TIMESTAMPTZ" || upper == "TIMESTAMP WITH TIME ZONE" || upper == "TIMESTAMP":
		baseType = "DateTime64(3, 'UTC')"
	case upper == "DATE":
		baseType = "Date32"
	default:
		baseType = "String"
	}
	if isNullable {
		return fmt.Sprintf("Nullable(%s)", baseType)
	}
	return baseType
}
======= REPLACEMENT
func mapToClickHouseColumnType(pgType string, isNullable bool) string {
	trimmed := strings.TrimSpace(pgType)
	upper := strings.ToUpper(trimmed)
	var baseType string
	switch {
	case upper == "BIGINT" || upper == "INT64" || (upper == "INT8" && trimmed == "INT8"):
		baseType = "Int64"
	case upper == "INT8":
		baseType = "Int8"
	case upper == "INTEGER" || upper == "INT" || upper == "INT4" || upper == "INT32":
		baseType = "Int32"
	case upper == "SMALLINT" || upper == "INT2" || upper == "INT16":
		baseType = "Int16"
	case upper == "UINT64":
		baseType = "UInt64"
	case upper == "UINT32":
		baseType = "UInt32"
	case upper == "UINT16":
		baseType = "UInt16"
	case upper == "UINT8" || upper == "BOOLEAN" || upper == "BOOL":
		baseType = "UInt8"
	case upper == "DOUBLE PRECISION" || upper == "FLOAT8" || upper == "FLOAT64":
		baseType = "Float64"
	case upper == "FLOAT32":
		baseType = "Float32"
	case upper == "REAL" || upper == "FLOAT4":
		baseType = "Float64"
	case strings.HasPrefix(upper, "NUMERIC") || strings.HasPrefix(upper, "DECIMAL"):
		if i := strings.IndexByte(trimmed, '('); i >= 0 && strings.HasSuffix(trimmed, ")") {
			baseType = "Decimal" + trimmed[i:]
		} else {
			baseType = "Decimal(38, 10)"
		}
	case upper == "TIMESTAMPTZ" || upper == "TIMESTAMP WITH TIME ZONE" || upper == "TIMESTAMP" || strings.HasPrefix(upper, "DATETIME64"):
		baseType = "DateTime64(3, 'UTC')"
	case upper == "DATETIME":
		baseType = "DateTime"
	case upper == "DATE32":
		baseType = "Date32"
	case upper == "DATE":
		if trimmed == "Date" {
			baseType = "Date"
		} else {
			baseType = "Date32"
		}
	case upper == "UUID":
		baseType = "UUID"
	case strings.HasPrefix(upper, "FIXEDSTRING(") && strings.HasSuffix(upper, ")"):
		baseType = "FixedString" + trimmed[len("FixedString"):]
	default:
		baseType = "String"
	}
	if isNullable {
		return fmt.Sprintf("Nullable(%s)", baseType)
	}
	return baseType
}
>>>>>>>
```

3. Cập nhật `transmuter_utils.go` để nhận diện đúng các kiểu thời gian và chuỗi của ClickHouse:

```go
<<<<<<< TARGET (centralized-data-service/internal/service/master/transmuter_utils.go:391-400)
func isTimestampColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return strings.HasPrefix(dt, "timestamp") || dt == "datetime"
}

// isTextColumnType reports whether the mapping rule targets a text or varchar column
func isTextColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return dt == "text" || strings.HasPrefix(dt, "varchar") || strings.HasPrefix(dt, "char")
}
======= REPLACEMENT
func isTimestampColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return strings.HasPrefix(dt, "timestamp") || strings.HasPrefix(dt, "datetime")
}

// isTextColumnType reports whether the mapping rule targets a text or varchar column
func isTextColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return dt == "text" || dt == "string" || strings.HasPrefix(dt, "varchar") || strings.HasPrefix(dt, "char") || strings.HasPrefix(dt, "fixedstring")
}
>>>>>>>
```

```go
<<<<<<< TARGET (centralized-data-service/internal/service/master/transmuter_utils.go:493-496)
func isTimestampOrDateColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return strings.HasPrefix(dt, "timestamp") || dt == "datetime" || dt == "date"
}
======= REPLACEMENT
func isTimestampOrDateColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return strings.HasPrefix(dt, "timestamp") || strings.HasPrefix(dt, "datetime") || dt == "date" || dt == "date32"
}
>>>>>>>
```

---

## 4. Vòng Lặp Phản Tỉnh `S5 → S3 → S4` (Adversarial Audit Findings `G1..G3` & Khắc phục Triệt để)

Sau khi rà soát phản biện từng dòng (`git diff`) và kiểm tra luồng runtime từ `_raw_data` → `extractColumns` → `ValidateValue` → `bulkUpsertClickHouseMaster`, phát hiện **3 điểm vi phạm / lỗi tiềm ẩn (`G1..G3`)** trong bản sửa đầu tiên cần khắc phục triệt để:

| Mã | Vị trí (`file:line`) | Phát hiện Phản biện (`VERIFIED`) | Phương án Khắc phục Chuẩn Single-Responsibility |
| :--- | :--- | :--- | :--- |
| **G1** | **Vi phạm Single-Responsibility & Ngân sách Dòng God-File**<br/>- `cdc-cms-web/src/constants/postgresDataTypes.ts` (`25` → `136` dòng)<br/>- `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx` (`1,388` → `1,420` dòng)<br/>- `cdc-cms-service/pkgs/utils/dbutil/validate.go` (`66` → `123` dòng) | 1. Nhồi `CLICKHOUSE_DATA_TYPES` và `normalizeClickHouseDataType` (`+111` dòng) vào `postgresDataTypes.ts` làm sai lệch trách nhiệm đơn của file hằng số PostgreSQL.<br/>2. Làm phình God-file `MasterMappingFieldsPage.tsx` thêm `+32` dòng (`1,420 > 1,388` dòng ở HEAD, vi phạm Lesson `L012` / `L031`), đồng thời chưa truyền `engineType="clickhouse"` xuống `<TableIndexManager>` (`:1115`).<br/>3. Nhồi `canonicalClickHouseBaseTypes` và `NormalizeMasterDataType` vào `dbutil/validate.go`. | 1. Tách riêng module đơn trách nhiệm `cdc-cms-web/src/constants/clickhouseDataTypes.ts` chứa `CLICKHOUSE_DATA_TYPES`, `normalizeClickHouseDataType`, `isClickHouseMasterBinding`, `resolveMasterDataTypeView`, `buildMasterDataTypePayload`; trả `postgresDataTypes.ts` về đúng trách nhiệm PostgreSQL (`<= 30` dòng, chỉ re-export để tương thích ngược).<br/>2. Rút gọn `MasterMappingFieldsPage.tsx` xuống `<= 1,388` dòng (`NOW <= HEAD`) bằng cách gọi helper từ `clickhouseDataTypes.ts` và truyền đúng `engineType` cho `<TableIndexManager>`.<br/>3. Tách `canonicalClickHouseBaseTypes` và `NormalizeMasterDataType` sang file riêng `cdc-cms-service/pkgs/utils/dbutil/clickhouse_types.go`. |
| **G2** | **Lỗi Runtime Validation & ClickHouse Upsert với cột `UInt8` chứa giá trị Boolean**<br/>- `centralized-data-service/internal/service/shadow/type_resolver.go:111-128`<br/>- `centralized-data-service/internal/service/master/transmuter_clickhouse.go:324-331` | 1. Trên ClickHouse (`goopay_clickhouse.transaction_history`), các cột boolean (`is_delete`, `sys_trans`) có kiểu vật lý `Nullable(UInt8)` và hiển thị trên UI là `UInt8`. Khi rule lưu `data_type = "UInt8"` nhưng `_raw_data` chứa JSON boolean (`true`/`false`), `gjsonValueToGo` trả về kiểu Go `bool`. Tại `type_resolver.go:111-128`, nhánh `specUpper == "UINT8"` chỉ chấp nhận `string`, `float64`, `int64`, `int` nên trả lỗi `"expected numeric for UInt8, got bool"`, khiến `extractColumns` (`transmuter.go:1817-1835`) đánh trượt hoặc xóa `nil` cột `is_delete`/`sys_trans`!<br/>2. Tại `bulkUpsertClickHouseMaster` (`transmuter_clickhouse.go:324-331`), chỉ riêng cột hệ thống `rec["_deleted"]` được đổi từ `bool` sang `uint8(1)/uint8(0)`, còn các cột nghiệp vụ (`is_delete`, `sys_trans`) mang giá trị `bool` truyền thẳng vào `sqlBindValueTransmute(rec[k])` khi bind vào cột `Nullable(UInt8)` của ClickHouse. | 1. Tại `type_resolver.go:111-128`, cho phép `case bool:` khi `specUpper == "UINT8"` (và bổ sung đầy đủ các kiểu số nguyên/thực chuẩn của Go `int8..int64`, `uint8..uint64`, `float32..float64`).<br/>2. Tại `transmuter_clickhouse.go:324-331`, tách hàm helper `coerceClickHouseBindValue(val any)` để chuyển `bool` (`true`/`false`) sang `uint8(1)`/`uint8(0)` cho toàn bộ các cột trong `bulkUpsertClickHouseMaster`. |
| **G3** | **Thiếu chuẩn hóa `DataType` ở `POST /api/v1/master-mapping-rules` (`Save`) & Bóc `Nullable(...)` ở `mapToClickHouseColumnType`**<br/>- `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go:108-120`<br/>- `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:47` | 1. API `Save` (`POST /api/v1/master-mapping-rules`) lưu trực tiếp `req.DataType` mà chưa gọi `dbutil.NormalizeMasterDataType` và `dbutil.IsValidDataType`.<br/>2. `mapToClickHouseColumnType` (`master_ddl_generator_clickhouse.go:47`) chưa bóc vỏ `Nullable(...)` nếu `pgType` truyền vào đã có dạng `Nullable(Int64)`, khiến rơi vào `default: String`. | 1. Gọi `dbutil.NormalizeMasterDataType` và kiểm tra `dbutil.IsValidDataType` ngay trong `Save` (`master_mapping_rule_handler.go`).<br/>2. Bóc lớp vỏ `Nullable(...)` ở đầu hàm `mapToClickHouseColumnType` trước khi đối chiếu kiểu cơ sở. |

