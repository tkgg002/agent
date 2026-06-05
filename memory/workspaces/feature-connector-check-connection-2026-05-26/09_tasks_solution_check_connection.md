# 09_tasks_solution_check_connection — Code Demo chi tiết

> **Phase**: `check_connection`
> ⚠️ **Brain Code Prohibition (CLAUDE.md §12)**: File này là GIẢI PHÁP MẪU. Brain TUYỆT ĐỐI KHÔNG tự apply. Muscle nhận lệnh "execute" mới được edit.
> ⚠️ **L-3070**: Mỗi edit dưới đây phải verify ACTUAL line trong file qua Read TRƯỚC khi Edit. Có thể line đã shift do refactor khác.

---

## Edit #1 — Worker `HandleDiscoverMongoDatabases` extend URI

**File**: `centralized-data-service/internal/handler/command_handler.go`
**Location**: ~line 1162-1200 (verify chính xác qua T0.2)
**Risk**: LOW

### Before (giả định theo audit)

```go
func (h *CommandHandler) HandleDiscoverMongoDatabases(msg *nats.Msg) {
    var req struct {
        Host string `json:"host"`
        Port string `json:"port"`
    }
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        h.replyError(msg, fmt.Errorf("invalid payload: %w", err))
        return
    }
    uri := fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port)
    dbs, err := h.mongoIntrospect.DiscoverDatabases(uri)
    if err != nil {
        h.replyError(msg, err)
        return
    }
    h.replyJSON(msg, map[string]interface{}{
        "databases": dbs,
    })
}
```

### After (proposed)

```go
func (h *CommandHandler) HandleDiscoverMongoDatabases(msg *nats.Msg) {
    var req struct {
        Uri  string `json:"uri"`
        Host string `json:"host"`
        Port string `json:"port"`
    }
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        h.replyError(msg, fmt.Errorf("invalid payload: %w", err))
        return
    }

    uri := req.Uri
    if uri == "" {
        if req.Host == "" || req.Port == "" {
            h.replyError(msg, fmt.Errorf("missing connection: provide either uri or host+port"))
            return
        }
        uri = fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port)
    }

    dbs, err := h.mongoIntrospect.DiscoverDatabases(uri)
    sanitized := service.SanitizeMongoDSN(uri)
    if err != nil {
        h.replyJSON(msg, map[string]interface{}{
            "status":        "cluster_err",
            "error":         err.Error(),
            "sanitized_dsn": sanitized,
        })
        return
    }
    h.replyJSON(msg, map[string]interface{}{
        "status":        "ok",
        "databases":     dbs,
        "sanitized_dsn": sanitized,
    })
}
```

### Note

- `service.SanitizeMongoDSN` — confirm tồn tại trong `mongo_introspection.go`. Nếu chưa export → export, hoặc thêm helper local.
- KHÔNG log raw `req.Uri`. Chỉ log `sanitized`.

---

## Edit #2 — Worker `HandleDiscoverMongoCollections` extend URI + 5-case

**File**: cùng `command_handler.go`
**Location**: ~line 1204-1240
**Risk**: LOW

### After (proposed)

```go
func (h *CommandHandler) HandleDiscoverMongoCollections(msg *nats.Msg) {
    var req struct {
        Uri      string `json:"uri"`
        Host     string `json:"host"`
        Port     string `json:"port"`
        Database string `json:"database"`
    }
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        h.replyError(msg, fmt.Errorf("invalid payload: %w", err))
        return
    }
    if req.Database == "" {
        h.replyError(msg, fmt.Errorf("missing database"))
        return
    }

    uri := req.Uri
    if uri == "" {
        if req.Host == "" || req.Port == "" {
            h.replyError(msg, fmt.Errorf("missing connection: provide either uri or host+port"))
            return
        }
        uri = fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port)
    }

    sanitized := service.SanitizeMongoDSN(uri)
    h.log.Info("introspect.mongo.collections.start",
        zap.String("sanitized_dsn", sanitized),
        zap.String("database", req.Database))

    // Probe diagnosis trước
    diag, err := h.mongoIntrospect.ProbeDatabase(uri, req.Database) // existing helper hoặc inline below
    if err != nil {
        // cluster_err
        h.replyJSON(msg, map[string]interface{}{
            "status":        "cluster_err",
            "error":         err.Error(),
            "sanitized_dsn": sanitized,
        })
        return
    }
    if diag.Status == "db_missing" {
        h.replyJSON(msg, map[string]interface{}{
            "status":              "db_missing",
            "error":               fmt.Sprintf("database '%s' not found", req.Database),
            "available_databases": diag.AvailableDBs,
            "sanitized_dsn":       sanitized,
        })
        return
    }

    cols, err := h.mongoIntrospect.DiscoverCollections(uri, req.Database)
    if err != nil {
        h.replyJSON(msg, map[string]interface{}{
            "status":        "cluster_err",
            "error":         err.Error(),
            "sanitized_dsn": sanitized,
        })
        return
    }
    if len(cols) == 0 {
        h.replyJSON(msg, map[string]interface{}{
            "status":        "empty",
            "collections":   []string{},
            "database":      req.Database,
            "sanitized_dsn": sanitized,
        })
        return
    }

    h.log.Info("introspect.mongo.collections.ok",
        zap.String("sanitized_dsn", sanitized),
        zap.String("database", req.Database),
        zap.Int("count", len(cols)))

    h.replyJSON(msg, map[string]interface{}{
        "status":        "ok",
        "collections":   cols,
        "database":      req.Database,
        "sanitized_dsn": sanitized,
    })
}
```

### Note

- Nếu `ProbeDatabase` chưa tồn tại trong service: tạo helper, hoặc inline `ListDatabaseNames` rồi check membership. Reference `IntrospectCollectionDiagnose` pattern (mongo_introspection.go:149).
- KHÔNG dùng `replyError` cho 5-case business — dùng `replyJSON` với `status` field để FE phân biệt rõ.

---

## Edit #3 — BE relay `DiscoverMongoDatabases` thêm POST

**File**: `cdc-cms-service/internal/api/introspection_handler.go`
**Location**: ~line 25-75
**Risk**: LOW

### Add new method

```go
// POST /api/introspection/mongo/databases
// Body: {"uri": "mongodb://..."}
func (h *IntrospectionHandler) DiscoverMongoDatabasesPost(c *gin.Context) {
    var body struct {
        Uri  string `json:"uri"`
        Host string `json:"host"`
        Port string `json:"port"`
    }
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
        return
    }
    if body.Uri == "" && body.Host == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing uri or host"})
        return
    }

    payload, _ := json.Marshal(map[string]interface{}{
        "uri":  body.Uri,
        "host": body.Host,
        "port": body.Port,
    })

    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    reply, err := h.natsConn.RequestWithContext(ctx, "cdc.cmd.introspect.mongo.databases", payload)
    if err != nil {
        c.JSON(http.StatusGatewayTimeout, gin.H{"status": "timeout", "error": err.Error()})
        return
    }

    var out map[string]interface{}
    _ = json.Unmarshal(reply.Data, &out)
    c.JSON(http.StatusOK, out)
}
```

### Note

- Giữ method `DiscoverMongoDatabases` (GET cũ) nguyên cho backward compat.
- Có thể refactor share parse+nats logic — defer.

---

## Edit #4 — BE relay `DiscoverMongoCollections` thêm POST

**File**: cùng `introspection_handler.go`
**Location**: ~line 77-150

```go
// POST /api/introspection/mongo/collections
// Body: {"uri": "...", "database": "..."}
func (h *IntrospectionHandler) DiscoverMongoCollectionsPost(c *gin.Context) {
    var body struct {
        Uri      string `json:"uri"`
        Host     string `json:"host"`
        Port     string `json:"port"`
        Database string `json:"database"`
    }
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
        return
    }
    if body.Database == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing database"})
        return
    }
    if body.Uri == "" && body.Host == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing uri or host"})
        return
    }

    payload, _ := json.Marshal(body)

    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    reply, err := h.natsConn.RequestWithContext(ctx, "cdc.cmd.introspect.mongo.collections", payload)
    if err != nil {
        c.JSON(http.StatusGatewayTimeout, gin.H{"status": "timeout", "error": err.Error()})
        return
    }

    var out map[string]interface{}
    _ = json.Unmarshal(reply.Data, &out)
    c.JSON(http.StatusOK, out)
}
```

---

## Edit #4.5 — Router register POST routes

**File**: `cdc-cms-service/internal/router/router.go:331-332`

### Before

```go
dualGet(shared, "/introspection/mongo/databases", introspectionHandler.DiscoverMongoDatabases)
dualGet(shared, "/introspection/mongo/:db/collections", introspectionHandler.DiscoverMongoCollections)
```

### After

```go
dualGet(shared, "/introspection/mongo/databases", introspectionHandler.DiscoverMongoDatabases)
dualGet(shared, "/introspection/mongo/:db/collections", introspectionHandler.DiscoverMongoCollections)
shared.POST("/introspection/mongo/databases", introspectionHandler.DiscoverMongoDatabasesPost)
shared.POST("/introspection/mongo/collections", introspectionHandler.DiscoverMongoCollectionsPost)
```

### Note

- `shared` group = `/api` (audit subagent đã confirm). Route final = `/api/introspection/mongo/{databases,collections}`.
- Có thể dùng `dualPost` nếu pattern đó tồn tại (cho v1 alias). Verify trong M2.

---

## Edit #5 — FE service `connectorCheck.ts`

**File**: `cdc-cms-web/src/services/connectorCheck.ts` (NEW)
**Risk**: LOW

```ts
import { cmsApi } from './api';

export type CheckStatus =
  | 'ok'
  | 'cluster_err'
  | 'auth_err'
  | 'db_missing'
  | 'empty'
  | 'timeout'
  | 'unknown';

export interface CheckCollectionsResponse {
  status: CheckStatus;
  collections?: string[];
  database?: string;
  available_databases?: string[];
  error?: string;
  sanitized_dsn?: string;
}

export interface CheckDatabasesResponse {
  status: CheckStatus;
  databases?: string[];
  error?: string;
  sanitized_dsn?: string;
}

export async function checkMongoDatabases(payload: { uri: string }): Promise<CheckDatabasesResponse> {
  const { data } = await cmsApi.post<CheckDatabasesResponse>(
    '/api/introspection/mongo/databases',
    payload,
  );
  return data;
}

export async function checkMongoCollections(payload: {
  uri: string;
  database: string;
}): Promise<CheckCollectionsResponse> {
  const { data } = await cmsApi.post<CheckCollectionsResponse>(
    '/api/introspection/mongo/collections',
    payload,
  );
  return data;
}
```

### Note

- `cmsApi` đã có sẵn trong `src/services/api.ts` (audit confirmed).
- KHÔNG cache vào React Query cache với key chứa URI (security).

---

## Edit #6 — FE hook `useConnectorCheck.ts`

**File**: `cdc-cms-web/src/hooks/useConnectorCheck.ts` (NEW)

```ts
import { useMutation } from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import {
  checkMongoCollections,
  CheckCollectionsResponse,
} from '../services/connectorCheck';

export interface UseCheckMongoConnectionReturn {
  result: CheckCollectionsResponse | null;
  isPending: boolean;
  check: (input: { uri: string; database: string }) => Promise<void>;
  reset: () => void;
}

export function useCheckMongoConnection(): UseCheckMongoConnectionReturn {
  const [result, setResult] = useState<CheckCollectionsResponse | null>(null);

  const mutation = useMutation({
    mutationFn: checkMongoCollections,
    onSuccess: (data) => {
      setResult(data);
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      setResult({ status: 'unknown', error: msg });
    },
  });

  const check = useCallback(
    async (input: { uri: string; database: string }) => {
      setResult(null);
      await mutation.mutateAsync(input);
    },
    [mutation],
  );

  const reset = useCallback(() => {
    setResult(null);
    mutation.reset();
  }, [mutation]);

  return {
    result,
    isPending: mutation.isPending,
    check,
    reset,
  };
}
```

---

## Edit #7 — FE state setup trong SourceConnectors.tsx

**File**: `cdc-cms-web/src/pages/SourceConnectors.tsx`
**Location**: ~line 286-330 (component state section)

```tsx
import { useCheckMongoConnection } from '../hooks/useConnectorCheck';
import { Select, Alert, Button, Spin } from 'antd';

// ... inside component body, after existing `form` hook:

const checkHook = useCheckMongoConnection();
const connectionUrl = Form.useWatch('connectionUrl', form);
const database = Form.useWatch('database', form);

// Reset check result when URI or database changes
React.useEffect(() => {
  checkHook.reset();
  form.setFieldValue('collectionNames', undefined);
}, [connectionUrl, database]);  // eslint-disable-line react-hooks/exhaustive-deps
```

### Note

- React 19 + Antd v6 hỗ trợ `Form.useWatch`.
- ESLint disable cho `checkHook` ref (intentional — chỉ trigger trên URL/DB change).

---

## Edit #8 — FE Button Check Connection

**File**: cùng, trong Modal form render (~line 950)

```tsx
{dbKind === 'mongodb' && (
  <Form.Item label=" " colon={false}>
    <Button
      onClick={() => {
        if (!connectionUrl || !database) {
          message.warning('Cần nhập Connection URL và Database trước khi check');
          return;
        }
        checkHook.check({ uri: connectionUrl, database });
      }}
      loading={checkHook.isPending}
      disabled={!connectionUrl || !database}
    >
      {checkHook.isPending ? 'Đang kiểm tra...' : 'Check Connection'}
    </Button>
  </Form.Item>
)}

{checkHook.isPending && (
  <Form.Item label=" " colon={false}>
    <Spin size="small" /> <span style={{ marginLeft: 8 }}>Đang kết nối tới Mongo và liệt kê collections...</span>
  </Form.Item>
)}

{checkHook.result && checkHook.result.status !== 'ok' && (
  <Form.Item label=" " colon={false}>
    <Alert
      type="error"
      message={mapCheckStatusToVi(checkHook.result)}
      description={
        checkHook.result.status === 'db_missing' && checkHook.result.available_databases ? (
          <div>
            <div>Database có sẵn:</div>
            <ul style={{ marginTop: 4, marginBottom: 0 }}>
              {checkHook.result.available_databases.slice(0, 20).map((d) => (
                <li key={d}>{d}</li>
              ))}
            </ul>
          </div>
        ) : checkHook.result.sanitized_dsn ? (
          <small style={{ color: '#888' }}>DSN: {checkHook.result.sanitized_dsn}</small>
        ) : null
      }
      showIcon
    />
  </Form.Item>
)}

{checkHook.result && checkHook.result.status === 'ok' && (
  <Form.Item label=" " colon={false}>
    <Alert
      type="success"
      message={`Kết nối OK — tìm thấy ${checkHook.result.collections?.length ?? 0} collection.`}
      showIcon
    />
  </Form.Item>
)}
```

---

## Edit #9 — FE field Collections → multi-select

**File**: cùng, replace `<Form.Item name="collectionNames">...` (line ~966)

### Before

```tsx
<Form.Item name="collectionNames" label="Collections">
  <Input placeholder="users,orders,payments" />
</Form.Item>
```

### After

```tsx
<Form.Item
  name="collectionNames"
  label="Collections"
  rules={[
    {
      validator: (_, value) => {
        if (editorMode === 'create' && (!value || value.length === 0)) {
          return Promise.reject(new Error('Chọn ít nhất 1 collection'));
        }
        return Promise.resolve();
      },
    },
  ]}
  extra="Tất cả collection được chọn sẵn sau khi Check Connection PASS. Bỏ chọn collection không cần CDC."
>
  <Select
    mode="multiple"
    placeholder={
      checkHook.result?.status === 'ok'
        ? 'Bỏ chọn collection không cần CDC'
        : 'Bấm Check Connection để load danh sách'
    }
    options={(checkHook.result?.collections ?? []).map((c) => ({ value: c, label: c }))}
    disabled={editorMode === 'create' && checkHook.result?.status !== 'ok'}
    showSearch
    allowClear
    optionFilterProp="label"
  />
</Form.Item>
```

---

## Edit #10 — Auto-select-all on success

**File**: cùng, trong hook usage (~line 295 hoặc trong onSuccess handler)

Update `useCheckMongoConnection` hook hoặc handler trong page:

```tsx
const checkHook = useCheckMongoConnection();

// override behavior: on success, auto-select all
React.useEffect(() => {
  if (checkHook.result?.status === 'ok' && checkHook.result.collections) {
    form.setFieldValue('collectionNames', checkHook.result.collections);
  }
}, [checkHook.result, form]);
```

### Note

- Pattern này keep hook generic + handle UI-specific logic ở page.
- Alternative: pass `onSuccess` callback vào hook signature.

---

## Edit #11 — Gate Create button

**File**: cùng, Modal okButtonProps (~line 884)

### Before

```tsx
okText={editorMode === 'create' ? 'Create' : 'Update'}
confirmLoading={createMut.isPending || updateMut.isPending}
```

### After

```tsx
okText={editorMode === 'create' ? 'Create' : 'Update'}
confirmLoading={createMut.isPending || updateMut.isPending}
okButtonProps={{
  disabled:
    dbKind === 'mongodb' &&
    editorMode === 'create' &&
    (!checkHook.result || checkHook.result.status !== 'ok'),
}}
```

### Note

- Edit mode KHÔNG gate (user có thể save mà không re-check).
- Non-Mongo kind chưa có check (defer phase sau) → KHÔNG gate.

---

## Edit #12 — Helper `mapCheckStatusToVi`

**File**: `cdc-cms-web/src/utils/checkStatusVi.ts` (NEW, optional — hoặc inline trong page)

```ts
import { CheckCollectionsResponse } from '../services/connectorCheck';

export function mapCheckStatusToVi(r: CheckCollectionsResponse): string {
  switch (r.status) {
    case 'ok':
      return `Kết nối OK — ${r.collections?.length ?? 0} collection.`;
    case 'cluster_err':
      return `Không kết nối được tới Mongo: ${r.error ?? 'unknown error'}. Kiểm tra URL và network.`;
    case 'auth_err':
      return `Sai thông tin xác thực Mongo. Kiểm tra user/password trong URL.`;
    case 'db_missing':
      return `Database '${r.database ?? '?'}' không tồn tại.`;
    case 'empty':
      return `Database '${r.database ?? '?'}' chưa có collection nào. Tạo collection rồi check lại.`;
    case 'timeout':
      return `Worker không phản hồi sau 10s. Vui lòng thử lại.`;
    case 'unknown':
    default:
      return `Lỗi không xác định: ${r.error ?? 'unknown'}.`;
  }
}
```

### Note

- `auth_err` không phải status worker trả trực tiếp — parse từ `r.error` chứa "Authentication failed". Edit #12.5 (optional):

```ts
function refineAuthError(r: CheckCollectionsResponse): CheckCollectionsResponse {
  if (r.status === 'cluster_err' && r.error && /auth/i.test(r.error)) {
    return { ...r, status: 'auth_err' };
  }
  return r;
}
```

Apply ở hook `onSuccess`.

---

## Edit #13 — buildConnectorConfig: handle array collectionNames

**File**: `SourceConnectors.tsx:148-220`

### Before (line 168 area)

```ts
if (values.collectionNames) {
  cfg['collection.include.list'] = values.collectionNames;  // string
}
```

### After

```ts
if (values.collectionNames) {
  const list = Array.isArray(values.collectionNames)
    ? values.collectionNames
    : String(values.collectionNames).split(',').map((s) => s.trim()).filter(Boolean);
  // Debezium spec: collection.include.list = db.collection,db.collection
  const db = values.database ?? '';
  cfg['collection.include.list'] = list
    .map((c) => (c.includes('.') ? c : `${db}.${c}`))
    .join(',');
}
```

### Note

- Worker trả collection name không có DB prefix. FE PHẢI prepend `${db}.${c}` để match Debezium spec.
- Check format `c.includes('.')` để handle case user gõ tay sẵn dạng `db.col`.

---

## Edit #14 — Edit existing connector pre-fill

**File**: `SourceConnectors.tsx` (~line 410 trong onClickEdit handler)

### Before (giả định)

```ts
const onClickEdit = (record: ConnectorRecord) => {
  form.setFieldsValue({
    name: record.name,
    connectorClass: record.config['connector.class'],
    // ...
    collectionNames: record.config['collection.include.list'] ?? '',
  });
  setEditorOpen(true);
};
```

### After

```ts
const onClickEdit = (record: ConnectorRecord) => {
  const raw = record.config['collection.include.list'] ?? '';
  const list = raw
    ? raw.split(',').map((s) => s.trim().split('.').pop()!).filter(Boolean)
    : [];
  form.setFieldsValue({
    name: record.name,
    connectorClass: record.config['connector.class'],
    // ...
    collectionNames: list,  // array, not string
  });
  // For edit mode, optionally seed checkHook to allow non-blocking edit
  if (list.length > 0) {
    // Not calling check; let Edit mode bypass gate
  }
  setEditorOpen(true);
};
```

### Note

- Strip DB prefix `.split('.').pop()` để hiển thị tên collection ngắn trong Select.
- Edit mode bypass gate (ADR-006) → KHÔNG cần call check.

---

## NON-edit references — KHÔNG sửa

| Module | Lý do giữ |
|---|---|
| `mongo_introspection.go` (service) | Đã đúng, nhận full URI. |
| `worker_server.go` NATS subscription | KHÔNG đổi subject name. |
| `compactConfig` (FE) | Giữ — vẫn drop empty key nếu user submit edit với multi-select rỗng. |
| `cdc_wizard_sessions` table + handler | KHÔNG dùng cho UC này (ADR-008). |

---

## Test sketches

### Unit test worker (TC-WU-01..06)

```go
// command_handler_test.go
func TestHandleDiscoverMongoCollections_UriPriority(t *testing.T) {
    mockSvc := &mockMongoIntrospect{}
    h := &CommandHandler{mongoIntrospect: mockSvc, /*...*/}

    payload := []byte(`{"uri":"mongodb://h1:27017","host":"h2","port":"27017","database":"db1"}`)
    msg := &nats.Msg{Data: payload, Reply: "reply"}
    h.HandleDiscoverMongoCollections(msg)

    require.Equal(t, "mongodb://h1:27017", mockSvc.gotUri)
    require.Equal(t, "db1", mockSvc.gotDB)
}

func TestHandleDiscoverMongoCollections_HostPortFallback(t *testing.T) {
    payload := []byte(`{"host":"localhost","port":"27017","database":"db1"}`)
    // ... expect uri = "mongodb://localhost:27017"
}

func TestHandleDiscoverMongoCollections_MissingDatabase(t *testing.T) {
    payload := []byte(`{"uri":"mongodb://h:27017"}`)
    // ... expect reply error "missing database"
}
```

### Unit test FE hook (TC-FU-01..03)

```ts
// useConnectorCheck.test.tsx
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useCheckMongoConnection } from './useConnectorCheck';
import * as service from '../services/connectorCheck';

jest.mock('../services/connectorCheck');

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const qc = new QueryClient();
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
};

test('check success sets result', async () => {
  (service.checkMongoCollections as jest.Mock).mockResolvedValue({
    status: 'ok',
    collections: ['users', 'orders'],
  });
  const { result } = renderHook(() => useCheckMongoConnection(), { wrapper });
  await act(() => result.current.check({ uri: 'mongodb://localhost', database: 'd' }));
  expect(result.current.result?.status).toBe('ok');
  expect(result.current.result?.collections).toEqual(['users', 'orders']);
});
```

---

## Checklist trước khi commit

- [ ] Read file đầy đủ trước Edit, verify exact line + indent (L-3070 warning).
- [ ] Sau Edit, Read lại file để verify diff đúng.
- [ ] Worker `go build ./... && go vet ./... && go test ./...` PASS.
- [ ] BE `go build ./... && go vet ./... && go test ./...` PASS.
- [ ] FE `pnpm tsc --noEmit && pnpm lint && pnpm build` PASS.
- [ ] Manual smoke trên local stack (M5 + M6).
- [ ] APPEND `05_progress.md` cho mỗi milestone (KHÔNG batch).
- [ ] `/security-agent` PASS (M7).
- [ ] KHÔNG `git commit --no-verify`.
- [ ] KHÔNG `git push --force`.
- [ ] KHÔNG cheat DB / config.
- [ ] KHÔNG log raw URI có password.
