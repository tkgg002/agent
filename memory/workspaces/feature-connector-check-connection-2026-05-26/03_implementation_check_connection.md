# 03_implementation_check_connection — Technical Design

> **Phase**: `check_connection`
> **Strategy**: ADR-001..006
> **Audience**: Muscle thực thi + reviewer.

---

## 1. High-level data flow

```
┌────────────────────────────────────────────────────────────────────────┐
│ FE cdc-cms-web SourceConnectors.tsx                                    │
│                                                                        │
│  1. User mở Modal Create Connector kiểu MongoDB                        │
│  2. User nhập:                                                         │
│     - Connection URL: mongodb://user:pass@host:port/?replicaSet=rs0    │
│     - Database: goopay_pbs                                             │
│  3. User click [Check Connection]                                      │
│                                                                        │
│  ─────────────────────  useConnectorCheck hook ─────────────────────   │
│  POST /api/introspection/mongo/collections                             │
│  Body: { "uri": "mongodb://...", "database": "goopay_pbs" }            │
└────────────────────────────────────────┬───────────────────────────────┘
                                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│ BE cdc-cms-service introspection_handler.go                            │
│                                                                        │
│  IntrospectionHandler.DiscoverMongoCollections (POST variant)          │
│   - parse body { uri, database }                                       │
│   - NATS request-reply subject "cdc.cmd.introspect.mongo.collections"  │
│   - payload: { "uri": "...", "database": "..." }                       │
│   - timeout 10s                                                        │
└────────────────────────────────────────┬───────────────────────────────┘
                                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│ Worker centralized-data-service command_handler.go                     │
│                                                                        │
│  HandleDiscoverMongoCollections                                        │
│   - parse payload                                                      │
│   - resolveURI:                                                        │
│       if payload.Uri != "" → uri = payload.Uri                         │
│       else → uri = fmt.Sprintf("mongodb://%s:%s", host, port)          │
│   - call mongoIntrospectionService.DiscoverCollections(uri, database)  │
│       → return ([]string, IntrospectDiagnosis, error)                  │
│   - reply NATS với JSON:                                               │
│       { collections, status, sanitized_dsn, available_dbs, error }     │
└────────────────────────────────────────┬───────────────────────────────┘
                                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│ Worker service mongo_introspection.go (KHÔNG đụng — đã đúng)           │
│                                                                        │
│  DiscoverCollections(uri, dbName)                                      │
│   - mongo.Connect(uri)                                                 │
│   - client.Database(dbName).ListCollectionNames()                      │
│   - probe 5-case → IntrospectDiagnosis                                 │
└────────────────────────────────────────────────────────────────────────┘
                                         ▲
                                         │ JSON response back up chain
                                         │
                                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│ FE SourceConnectors.tsx — render result                                │
│                                                                        │
│  if response.status === 'ok':                                          │
│    setCheckResult({ collections, status })                             │
│    form.setFieldValue('collectionNames', response.collections)         │
│    enable Create button                                                │
│  else:                                                                 │
│    setCheckResult({ status, error, availableDbs })                     │
│    show <Alert> with VN message                                        │
│    keep Create disabled                                                │
└────────────────────────────────────────────────────────────────────────┘
```

## 2. Code changes overview

| # | File | Change kind | LOC est |
|---|---|---|---|
| 1 | `centralized-data-service/internal/handler/command_handler.go` lines 1164-1230 | Extend DTO struct, add URI priority resolution | ~30 |
| 2 | `centralized-data-service/internal/handler/command_handler_test.go` (or new file) | Unit tests 4 cases | ~80 |
| 3 | `cdc-cms-service/internal/api/introspection_handler.go` lines 25-150 | Add POST handler variant, JSON body parse, NATS relay | ~60 |
| 4 | `cdc-cms-service/internal/router/router.go` lines 331-332 | Register POST routes | ~5 |
| 5 | `cdc-cms-web/src/services/connectorCheck.ts` (NEW) | Service functions axios | ~40 |
| 6 | `cdc-cms-web/src/hooks/useConnectorCheck.ts` (NEW) | React Query mutation hook | ~50 |
| 7 | `cdc-cms-web/src/pages/SourceConnectors.tsx` | Extend Modal: button, state, Select multiple, Alert, gate | ~100 |
| 8 | `cdc-cms-web/src/pages/SourceConnectors.tsx` (cont.) | Form watch listener invalidate | ~20 |
| 9 | `cdc-cms-web/src/utils/checkStatusVi.ts` (NEW, optional) | Helper map 5-case → VN message | ~30 |

**Total est**: ~415 LOC (mostly mechanical extends, low complexity).

## 3. Schema changes

**KHÔNG có**. Phase này stateless, không DB persist.

## 4. Migration changes

**KHÔNG có**.

## 5. API contract

### 5.1 POST `/api/introspection/mongo/databases`

**Request**:
```json
{
  "uri": "mongodb://user:pass@host:27017/?replicaSet=rs0&authSource=admin"
}
```

Backward compat (GET cũ vẫn giữ):
```
GET /api/introspection/mongo/databases?host=localhost&port=27017
```

**Response (200)**:
```json
{
  "status": "ok",
  "databases": ["goopay_pbs", "goopay_core", "admin", "config", "local"],
  "sanitized_dsn": "mongodb://***@host:27017/?replicaSet=rs0"
}
```

**Response (error)**:
```json
{
  "status": "cluster_err",
  "error": "context deadline exceeded after 10s",
  "sanitized_dsn": "mongodb://***@host:9999"
}
```

### 5.2 POST `/api/introspection/mongo/collections`

**Request**:
```json
{
  "uri": "mongodb://...",
  "database": "goopay_pbs"
}
```

Backward compat:
```
GET /api/introspection/mongo/:db/collections?host=&port=
```

**Response (ok)**:
```json
{
  "status": "ok",
  "collections": ["users", "orders", "payments", "refunds"],
  "sanitized_dsn": "mongodb://***@host/goopay_pbs",
  "database": "goopay_pbs"
}
```

**Response (db_missing)**:
```json
{
  "status": "db_missing",
  "error": "database 'goopay_pbs_typo' not found",
  "available_databases": ["goopay_pbs", "goopay_core", "admin"],
  "sanitized_dsn": "mongodb://***@host"
}
```

**Response (empty)**:
```json
{
  "status": "empty",
  "collections": [],
  "database": "db_empty_test",
  "sanitized_dsn": "..."
}
```

### 5.3 NATS subject contract (worker)

**Subject**: `cdc.cmd.introspect.mongo.collections`
**Request payload** (extended):
```json
{
  "uri": "mongodb://...",
  "host": "...",   // legacy fallback
  "port": "...",   // legacy fallback
  "database": "goopay_pbs"
}
```
**Reply payload**: shape giống Section 5.2.

## 6. Component design (FE)

### 6.1 Modal layout (proposed)

```
┌────────────────────────── Create Connector ──────────────────────────┐
│  Name:          [ ____________________________ ]                      │
│  Connector Class: [ MongoDB ▼ ]                                       │
│  Connection URL: [ mongodb://... ____________ ]                       │
│  Database:      [ ___________________________ ] [Check Connection]    │
│                                                                        │
│  Status: ⏳ Checking...                                                │
│    └─ (Spin + step label) "Đang kết nối tới Mongo..."                 │
│                                                                        │
│  Collections (chọn collection muốn CDC):                              │
│  ┌──────────────────────────────────────────┐                         │
│  │ ✓ users           ✓ orders                │                         │
│  │ ✓ payments        ✓ refunds               │ (Antd Select multi)    │
│  │ ✓ users_audit     ☐ legacy_tmp            │                         │
│  └──────────────────────────────────────────┘                         │
│                                                                        │
│  [Cancel]              [Create] (disabled until check PASS)            │
└────────────────────────────────────────────────────────────────────────┘
```

### 6.2 State machine FE

```
┌────────┐   user types URI    ┌────────────┐
│ idle   │ ─────────────────→  │ form-dirty │
└────────┘                     └──────┬─────┘
   ▲                                  │ click [Check]
   │                                  ▼
   │                           ┌────────────┐
   │                           │ checking   │  ← Spin visible, Create disabled
   │                           └──────┬─────┘
   │                                  │
   │             ┌────────────────────┼────────────────────┐
   │             ▼                    ▼                    ▼
   │     ┌────────────┐         ┌──────────┐        ┌────────────┐
   │     │ check_ok   │         │check_fail│        │check_empty │
   │     └─────┬──────┘         └────┬─────┘        └─────┬──────┘
   │           │                     │                    │
   │           │ collection list     │ <Alert error>      │ <Alert empty>
   │           │ + multi-select      │                    │
   │           │ + Create enabled    │                    │
   │           │                     │                    │
   │           ▼                     ▼                    ▼
   │     ┌────────────┐         ┌──────────────────────────┐
   │     │ submit OK  │         │ user edits URI/DB → idle │
   │     └────────────┘         └──────────────────────────┘
   │                                  │
   └──── on URI/DB change ────────────┘
```

### 6.3 Hook signature

```ts
// src/hooks/useConnectorCheck.ts
type CheckStatus = 'ok' | 'cluster_err' | 'auth_err' | 'db_missing' | 'empty' | 'timeout' | 'unknown';

interface CheckResult {
  status: CheckStatus;
  collections: string[];
  availableDbs?: string[];
  error?: string;
  sanitizedDsn?: string;
}

interface UseConnectorCheckReturn {
  result: CheckResult | null;
  isPending: boolean;
  check: (input: { uri: string; database: string }) => Promise<void>;
  reset: () => void;
}

export function useCheckMongoConnection(): UseConnectorCheckReturn;
```

## 7. Backward compatibility

| Aspect | Status |
|---|---|
| Existing GET `?host=&port=` callers | ✅ Giữ route GET cũ, behavior unchanged |
| Existing connector list view | ✅ KHÔNG đụng |
| Worker old NATS payload `{host, port}` | ✅ Fallback path khi `uri == ""` |
| FE old buildConnectorConfig signature | ✅ Cải tiến field `collectionNames` từ string → string\[\] khi multi-select. Cần update mapping logic trong M4.3 (split-join compat) |
| Existing `<Input placeholder="users,orders">` UX | ⚠️ REPLACED. User cũ quen text input giờ thấy multi-select → release note + screenshot trong report |

**Breaking change concern**: FE form value type của `collectionNames` từ `string` → `string[]`. Cần verify edit-existing-connector path (load existing config → parse `collection.include.list` → preset multi-select selected). Xem ADR-006.

## 8. Performance

| Aspect | Impact | Note |
|---|---|---|
| Worker NATS request | +1 RTT (~ms) per check | Acceptable, sync UX |
| Mongo `ListCollectionNames` | ~10ms cho DB <100 col, ~500ms cho DB >1000 col | Within 10s timeout |
| FE bundle size | +~5KB cho hook + service | Negligible |
| Network payload | ~50 bytes request, ~few KB response (vài chục col) | OK |

## 9. Observability

Log lines mới (sanitized):

```
INFO  worker.introspect.mongo.collections.start    sanitized_dsn=mongodb://***@host/db database=goopay_pbs
INFO  worker.introspect.mongo.collections.ok       sanitized_dsn=... database=goopay_pbs count=12 elapsed_ms=234
WARN  worker.introspect.mongo.collections.db_missing sanitized_dsn=... database=typo available_count=5
ERROR worker.introspect.mongo.collections.cluster_err sanitized_dsn=mongodb://***@host:9999 err="connection refused"

INFO  cms.introspect.mongo.collections.request  remote_addr=...
INFO  cms.introspect.mongo.collections.response status=ok count=12
```

Metrics (nếu có Prometheus exporter):
- `cdc_introspect_collections_total{status="ok|cluster_err|db_missing|empty|timeout"}` counter
- `cdc_introspect_collections_duration_seconds` histogram

## 10. Security

- **L-3275**: TUYỆT ĐỐI sanitize URI. Áp dụng `SanitizeMongoDSN` ở:
  - Log statement worker (3 vị trí)
  - Log statement BE (2 vị trí)
  - Error response payload (`sanitized_dsn` field)
- POST body chứa URI raw → đảm bảo HTTPS in prod, JWT auth on endpoint (router middleware đã có).
- KHÔNG cache URI ở FE local storage / session storage.
- KHÔNG store URI trong React Query cache với key chứa URI (dùng key `['connectorCheck', hash(uri)]` HOẶC không cache — mutation thay vì query).
- Rate limit: nếu chưa có generic rate limit middleware, M7 verify spam click không gây flood NATS.
- `/security-agent` review BẮT BUỘC ở M7.

## 11. Failure modes

| Failure | Detection | UX response |
|---|---|---|
| Mongo cluster down | Driver timeout 10s | `cluster_err` Alert "Không kết nối được tới Mongo. Kiểm tra URL và network." |
| Mongo auth fail | Driver `MongoServerError: Authentication failed` | `auth_err` Alert "Sai thông tin xác thực Mongo." |
| Database name typo | `ListDatabaseNames` không chứa target | `db_missing` Alert "Database không tồn tại. Có sẵn: <list>" |
| Database empty | `ListCollectionNames` returns 0 | `empty` Alert "Database chưa có collection nào." |
| NATS worker down | BE timeout 10s waiting reply | `timeout` Alert "Worker không phản hồi. Liên hệ ops." |
| BE crash | FE axios error | Generic Alert "Lỗi hệ thống. Thử lại sau." |
| URI format invalid | Driver `URI parse error` | `cluster_err` variant với hint format chuẩn |

## 12. Future work (defer)

- MySQL Check: tạo `MysqlIntrospectionService` + handler + NATS subject + endpoint. Pattern y hệt Mongo (5-case branching).
- Postgres Check: tương tự.
- Cross-DB driver interface: `SourceDriver` interface với `Probe(uri) (Diagnosis, error)` + `ListEntities(uri, ns) ([]string, error)` → mỗi DB implement.
- Auto-detect connector class theo URI scheme (`mongodb://`, `mysql://`, `postgres://`) → pre-fill `dbKind` field.
- WebSocket progress streaming nếu introspect lớn (>1000 collections).
- Cache result short-lived (60s) trong FE để tránh re-check khi user mở lại modal.
- Save URI encrypted nếu user check OK rồi muốn lưu (out of scope).
