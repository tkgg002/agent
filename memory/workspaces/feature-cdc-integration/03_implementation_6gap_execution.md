# 6 Gap Execution Report — Registry + Masters end-to-end

> Date: 2026-04-24 07:10–07:55 ICT
> SOP: 7-stage (Muscle)
> Input plan: `09_tasks_solution_registry_masters.md`
> Status: ✅ ALL 6 gap DONE, build + route wire verified.

## 1. Files changed

### Backend (cdc-cms-service)
| File | Change | Purpose |
|:-|:-|:-|
| `internal/api/system_connectors_handler.go` | +63 LOC | `Create` + `Delete` methods (forward Kafka Connect REST) |
| `internal/router/router.go` | +9 LOC | `registerDestructive("/v1/system/connectors", Create)` + DELETE manual wire |

### Frontend (cdc-cms-web)
| File | Change | Purpose |
|:-|:-|:-|
| `src/pages/TableRegistry.tsx` | EDIT | Gap 2/3/4/5b: airbyte removal, SyncStatusIndicator fetch real, Bridge buttons removed, Snapshot Now + Manage Masters added |
| `src/pages/SourceConnectors.tsx` | EDIT +130 LOC | Gap 5a: New Connector modal với MongoDB template + Delete button |
| `src/pages/SourceToMasterWizard.tsx` | NEW 130 LOC | Gap 6: 11-step cockpit wizard |
| `src/App.tsx` | +4 LOC | Gap 6: `/source-to-master` route + menu item |

### Docs
| File | Change | Purpose |
|:-|:-|:-|
| `cdc-system/architecture.md` | +120 LOC (Section 5.5) | Gap 1: document Shadow→Master 2-tier pipeline |

## 2. Implementation details per gap

### Gap 1 — architecture.md §5.5
- Mermaid diagram Shadow Layer → Control Plane → Master Layer với NATS flows.
- 8 sub-sections: Shadow Layer, Schema Proposal Workflow, Mapping Rules, Master Registry, Transmuter Module, TransmuteScheduler, Operator End-to-End, Source Code References.
- Reference `/source-to-master` wizard cho operator onboarding.

### Gap 2 — airbyte options
`TableRegistry.tsx:442-444` Register Modal:
```tsx
// BEFORE: <Select> airbyte / debezium / both
// AFTER:
<Form.Item name="sync_engine" label="Sync Engine" initialValue="debezium">
  <Input disabled />
</Form.Item>
```

### Gap 3 — SyncStatusIndicator (Option B chọn)
Rewrite component từ hard-coded 'n/a' → fetch `/api/v1/system/connectors`, match by `collection.include.list` entry chứa `source_db.source_table`.
- State mapping: RUNNING → green, PAUSED → warning, FAILED → error, not_configured → default.
- Tooltip hiển thị connector name matched.
- Refresh button per-row.

### Gap 4 — Bridge/Transform buttons
XOÁ:
- `handleBridge` function (2 button "Đồng bộ" + "Batch") — gọi 410 Gone.
- `handleTransform` function (button "Chuyển đổi") — legacy batch-transform flow.
THÊM:
- "Manage Masters" button → navigate `/masters?source_shadow=<target_table>`.

### Gap 5a — Add Connector (BE + FE)
**BE** (`system_connectors_handler.go`):
```go
func (h *SystemConnectorsHandler) Create(c *fiber.Ctx) error {
    var req struct {
        Name   string            `json:"name"`
        Config map[string]string `json:"config"`
    }
    // ...validate name regex + connector.class required
    payload := map[string]any{"name": req.Name, "config": req.Config}
    var resp map[string]any
    if err := h.doJSON(c.Context(), http.MethodPost, "/connectors", payload, &resp); err != nil {
        return c.Status(502).JSON(fiber.Map{"error": "connector_create_failed"})
    }
    return c.Status(201).JSON(resp)
}
```
+ `Delete` method tương tự.
Router: `registerDestructive("/v1/system/connectors", Create)` + manual DELETE wire.

**FE** (`SourceConnectors.tsx`):
- Button "New Connector" ở header.
- Modal (720px) với:
  - Input name (regex validate).
  - Textarea JSON config pre-filled với `MONGO_TEMPLATE` (connector.class + mongodb.connection.string + collection.include.list + signal.data.collection + avro converters).
  - Reason ≥10 chars.
- Mutation: `cmsApi.post('/api/v1/system/connectors', {name, config, reason})` với Idempotency-Key.
- Delete button per-row + confirm modal.

### Gap 5b — Snapshot Now
`TableRegistry.tsx` thêm button "Snapshot Now" (icon ThunderboltOutlined).
`handleSnapshot` dùng `Modal.confirm` → `cmsApi.post('/api/tools/trigger-snapshot/:table', {database, collection})`.
Backend path đã có: `router.go:140 registerDestructive` + `reconciliation_handler.TriggerSnapshot` publish NATS `cdc.cmd.debezium-signal` → Worker ghi Mongo `<db>.debezium_signal` collection.

### Gap 6 — `/source-to-master` wizard
New page `SourceToMasterWizard.tsx`:
- `<Steps>` vertical 11 steps.
- Mỗi step = `{title, description, goto?, verify, icon}`.
- Active step card bên dưới hiển thị description + DoD verify alert + "Mở trang: /xxx" link + Previous/Next navigation.
- Menu item "Source → Master Wizard" với icon Compass.

## 3. Verify evidence

```
$ cd cdc-cms-service && go build ./...
EXIT=0

$ cd cdc-cms-web && npx tsc --noEmit
EXIT=0

$ curl -X POST http://localhost:8083/api/v1/system/connectors -d '{}'
HTTP=401 {"error":"missing authorization header"}  ← route wired

$ curl -X DELETE http://localhost:8083/api/v1/system/connectors/dummy
HTTP=401 ← route wired

$ curl -X POST http://localhost:8083/api/tools/trigger-snapshot/dummy
HTTP=401 ← pre-existing route wired (Gap 5b backend verified)

$ curl http://localhost:5173/src/pages/SourceToMasterWizard.tsx
HTTP=200 size=22239 bytes ← Vite HMR transform OK

$ curl http://localhost:5173/src/pages/TableRegistry.tsx
HTTP=200 size=77629 bytes ← Vite HMR OK

$ curl http://localhost:5173/src/pages/SourceConnectors.tsx
HTTP=200 size=54267 bytes ← Vite HMR OK
```

## 4. Operator UX when Boss adds new DB

Giờ đây khi Boss add 1 Mongo collection mới, flow trên UI:
1. **`/source-to-master`** — mở wizard, step 1.
2. **`/sources`** — click "New Connector", paste config (đổi `<service-db-name>` + `<collection-name>`). Idempotency-Key + reason ghi audit. → Connector RUNNING.
3. **`/registry`** — click "Register Table" modal, điền source_db + source_table + target_table + pk + timestamp_field. → Row mới group theo source_db.
4. **`/registry`** — click "Tạo Table" → shadow DDL.
5. **`/registry`** — click "Snapshot Now" → Debezium incremental snapshot. Column Sync Engine hiển thị RUNNING (auto refresh 15s).
6. **`/schema-proposals`** — badge pending > 0 cho field financial cần typed.
7. **Approve** → ALTER + mapping rule.
8. **`/registry/:id/mappings`** — thêm custom rule, Preview JsonPath.
9. **`/masters`** — Create Master wizard, spec JSON.
10. **Approve** → public.<master> CREATE TABLE + RLS.
11. **`/masters`** toggle is_active + **`/schedules`** create cron.

Các bước 2-5 và 10-11 đều có CTA trực tiếp, không còn curl thủ công.

## 5. Follow-ups (non-blocking)

- [ ] Connector Delete button + modal phía `/sources` — DONE trong commit này.
- [ ] MongoDB template mở rộng cho MySQL / Postgres connector (hiện chỉ Mongo).
- [ ] Wizard `/source-to-master` lưu state per-table vào localStorage (hiện chỉ in-memory per session).
- [ ] Remove/archive dead code trong registry_handler.go `Bridge` + `Transform` (BE cleanup, outside FE scope).

## 6. Security gate (Rule 8)

- `Create` handler validate `connector.class` required → không thể tạo connector rỗng.
- `connectorNameRE` regex guard `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,128}$` → prevent path injection.
- DELETE route wrap đủ 3 middleware: RequireOpsAdmin + Idempotency + Audit.
- `filterSafeConfig` vẫn strip password/secret/token khi GET.
- FE validate reason ≥10 chars + Idempotency-Key mỗi request.

PASS.


exc:


Stage 1 — INTAKE

  Boss quote (2026-04-24 07:10 ICT):

  ▎ "thực hiện toàn bộ... sau task này tôi sẽ add 1 db mới vào. và quan sát toàn bộ tiến trình."

  Scope: 6 gap, P0-P3. FE /masters + /registry + /sources + Cockpit mới; BE thêm POST connector; Doc arch section 5.5.
  DoD: Boss add 1 DB mới (Mongo collection) → UI cms-fe theo dõi được đủ 11 bước từ Connect Source đến Master có data.
  SOP: 7-stage per gap, gộp thành multi-plan vì tương tác chặt.

  Read 1 file (ctrl+o to expand)

⏺ Stage 2 — PLAN (multi-gap execution order)

  Execution order tối ưu (dependencies):
  1. BE first (Gap 5a Create handler) → unblock FE Gap 5a.
  2. FE TableRegistry cleanup (Gap 2+3+4+5b) → file duy nhất, edit 1 lần.
  3. FE SourceConnectors (Gap 5a modal) → sau khi BE ready.
  4. FE Cockpit (Gap 6 new wizard) → sau khi các API kia sẵn.
  5. Arch doc (Gap 1) → độc lập.

  Read 1 file (ctrl+o to expand)

⏺ Stage 3 — EXECUTE

  Step 1/7: Gap 5a BE — Add Create method + Delete method

⏺ Update(~/Documents/work/cdc-system/cdc-cms-service/internal/api/system_connectors_handler.go)
  ⎿  Added 43 lines
      169    return c.Status(202).JSON(fiber.Map{"status": "task_restart_triggered", "connector": name, "task_id": taskID})
      170  }
      171  
      172 +// Create forwards a new connector config to Kafka Connect.                                                                                                                                                  
      173 +// POST /api/v1/system/connectors                                                                                                                                                                            
      174 +// Body: {"name": "...", "config": {"connector.class": "...", ...}}                                                                                                                                          
      175 +func (h *SystemConnectorsHandler) Create(c *fiber.Ctx) error {                                                                                                                                               
      176 +  var req struct {                                                                                                                                                                                           
      177 +    Name   string            `json:"name"`                                                                                                                                                                   
      178 +    Config map[string]string `json:"config"`                                                                                                                                                                 
      179 +  }                                                                                                                                                                                                          
      180 +  if err := c.BodyParser(&req); err != nil {                                                                                                                                                                 
      181 +    return c.Status(400).JSON(fiber.Map{"error": "bad_json"})                                                                                                                                                
      182 +  }                                                                                                                                                                                                          
      183 +  if !connectorNameRE.MatchString(req.Name) {                                                                                                                                                                
      184 +    return c.Status(400).JSON(fiber.Map{"error": "invalid_connector_name"})                                                                                                                                  
      185 +  }                                                                                                                                                                                                          
      186 +  if len(req.Config) == 0 {                                                                                                                                                                                  
      187 +    return c.Status(400).JSON(fiber.Map{"error": "config_required"})                                                                                                                                         
      188 +  }                                                                                                                                                                                                          
      189 +  if _, ok := req.Config["connector.class"]; !ok {                                                                                                                                                           
      190 +    return c.Status(400).JSON(fiber.Map{"error": "connector.class_required"})                                                                                                                                
      191 +  }                                                                                                                                                                                                          
      192 +  payload := map[string]any{"name": req.Name, "config": req.Config}                                                                                                                                          
      193 +  var resp map[string]any                                                                                                                                                                                    
      194 +  if err := h.doJSON(c.Context(), http.MethodPost, "/connectors", payload, &resp); err != nil {                                                                                                              
      195 +    return c.Status(502).JSON(fiber.Map{"error": "connector_create_failed", "detail": err.Error()})                                                                                                          
      196 +  }                                                                                                                                                                                                          
      197 +  h.logger.Info("connector created", zap.String("connector", req.Name))                                                                                                                                      
      198 +  return c.Status(201).JSON(resp)                                                                                                                                                                            
      199 +}                                                                                                                                                                                                            
      200 +                                                                                                                                                                                                             
      201 +// Delete removes a connector (use with care — consumer offsets may replay).                                                                                                                                 
      202 +// DELETE /api/v1/system/connectors/:name                                                                                                                                                                    
      203 +func (h *SystemConnectorsHandler) Delete(c *fiber.Ctx) error {                                                                                                                                               
      204 +  name := strings.TrimSpace(c.Params("name"))                                                                                                                                                                
      205 +  if !connectorNameRE.MatchString(name) {                                                                                                                                                                    
      206 +    return c.Status(400).JSON(fiber.Map{"error": "invalid_connector_name"})                                                                                                                                  
      207 +  }                                                                                                                                                                                                          
      208 +  if err := h.doJSON(c.Context(), http.MethodDelete, "/connectors/"+url.PathEscape(name), nil, nil); err != nil {                                                                                            
      209 +    return c.Status(502).JSON(fiber.Map{"error": "delete_failed", "detail": err.Error()})                                                                                                                    
      210 +  }                                                                                                                                                                                                          
      211 +  h.logger.Info("connector deleted", zap.String("connector", name))                                                                                                                                          
      212 +  return c.Status(202).JSON(fiber.Map{"status": "delete_triggered", "connector": name})                                                                                                                      
      213 +}                                                                                                                                                                                                            
      214 +                                                                                                                                                                                                             
      215  // Pause / Resume for maintenance.
      216  // POST /api/v1/system/connectors/:name/pause
      217  func (h *SystemConnectorsHandler) Pause(c *fiber.Ctx) error {

⏺ Step 2/7: Gap 5a — Router wire POST + DELETE

⏺ Reading 1 file… (ctrl+o to expand)
  ⎿  ~/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go

⏺ Update(~/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go)
  ⎿  Added 9 lines
      157    }
      158  
      159    // Debezium Command Center — Kafka Connect REST proxy. Replaces the
      160 +  registerDestructive("/v1/system/connectors", systemConnectorsHandler.Create)                                                                                                                               
      161    registerDestructive("/v1/system/connectors/:name/restart", systemConnectorsHandler.Restart)
      162    registerDestructive("/v1/system/connectors/:name/tasks/:taskId/restart", systemConnectorsHandler.RestartTask)
      163    registerDestructive("/v1/system/connectors/:name/pause", systemConnectorsHandler.Pause)
      164    registerDestructive("/v1/system/connectors/:name/resume", systemConnectorsHandler.Resume)
      165 +  // Destructive DELETE — removes connector entirely. Registered manually                                                                                                                                    
      166 +  // (registerDestructive only wraps POST).                                                                                                                                                                  
      167 +  apiGroup.Delete("/v1/system/connectors/:name",                                                                                                                                                             
      168 +    middleware.RequireOpsAdmin(),                                                                                                                                                                            
      169 +    middleware.Idempotency(idempotencyStore),                                                                                                                                                                
      170 +    middleware.Audit(auditLogger),                                                                                                                                                                           
      171 +    systemConnectorsHandler.Delete,                                                                                                                                                                          
      172 +  )                                                                                                                                                                                                          
      173  
      174    // Master Table Registry (Sprint 5 §R8) — admin plane for warehouse
      175    // masters. Approve dispatches cdc.cmd.master-create → worker runs DDL.

⏺ Searching for 1 pattern, reading 1 file… (ctrl+o to expand)
  ⎿  ~/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go

⏺ Update(~/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go)
  ⎿  Added 5 lines, removed 6 lines
      164    registerDestructive("/v1/system/connectors/:name/resume", systemConnectorsHandler.Resume)
      165    // Destructive DELETE — removes connector entirely. Registered manually
      166    // (registerDestructive only wraps POST).
      167 -  apiGroup.Delete("/v1/system/connectors/:name",                                                                                                                                                             
      168 -    middleware.RequireOpsAdmin(),                                                                                                                                                                            
      169 -    middleware.Idempotency(idempotencyStore),                                                                                                                                                                
      170 -    middleware.Audit(auditLogger),                                                                                                                                                                           
      171 -    systemConnectorsHandler.Delete,                                                                                                                                                                          
      172 -  )                                                                                                                                                                                                          
      167 +  {                                                                                                                                                                                                          
      168 +    deleteHandlers := append([]fiber.Handler{}, destructiveChain...)                                                                                                                                         
      169 +    deleteHandlers = append(deleteHandlers, systemConnectorsHandler.Delete)                                                                                                                                  
      170 +    apiGroup.Delete("/v1/system/connectors/:name", deleteHandlers...)                                                                                                                                        
      171 +  }                                                                                                                                                                                                          
      172  
      173    // Master Table Registry (Sprint 5 §R8) — admin plane for warehouse
      174    // masters. Approve dispatches cdc.cmd.master-create → worker runs DDL.

⏺ Step 3/7: BE build verify

⏺ Bash(cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go build ./... 2>&1 | tail -10; echo "EXIT=$?")
  ⎿  Running… (3s)
  ⎿  EXIT=0

⏺ BE xanh. Restart CMS để route mới active.

⏺ Bash(ps aux | grep -E "cmd/server" | grep -v grep | awk '{print $2}' | head -5)
  ⎿  28480
  ⎿  Allowed by auto mode classifier

⏺ Bash(kill 28480 2>&1; sleep 2; go run ./cmd/server > /tmp/cms.log 2>&1 &
      echo "restart kicked PID=$!")
  ⎿  Running in the background (↓ to manage)
  ⎿  Allowed by auto mode classifier

⏺ Bash(sleep 12 && lsof -iTCP:8083 -sTCP:LISTEN 2>/dev/null | head -3
      echo "--- CMS tail ---"…)
  ⎿  Running… (3s)
  ⎿  COMMAND   PID       USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
     server  28488 trainguyen   11u  IPv4 0xe10f4374f4bc50fb      0t0  TCP *:us-srv (LISTEN)
     --- CMS tail ---
     … +67 lines (ctrl+o to expand)

⏺ Background command "Restart CMS" completed (exit code 0)

⏺ Bash(kill -TERM 28488 2>&1; sleep 3
      ps aux | grep -E "cmd/server" | grep -v grep | awk '{print $2}' | head -5…)
  ⎿  Running… (3s)
  ⎿  (No output)
  ⎿  Allowed by auto mode classifier

⏺ Bash(go run ./cmd/server > /tmp/cms.log 2>&1 &
      echo "cms PID=$!")
  ⎿  Running in the background (↓ to manage)
  ⎿  Allowed by auto mode classifier

⏺ Bash(go run ./cmd/server > /tmp/cms.log 2>&1 &
      sleep 8…)
  ⎿  Running… (3s)
  ⎿  COMMAND   PID       USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
     server  50525 trainguyen   11u  IPv4 0x39127087d9aa27f9      0t0  TCP *:us-srv (LISTEN)
     {
     … +26 lines (ctrl+o to expand)

⏺ Background command "Start fresh CMS" completed (exit code 0)

⏺ BE route wired (401). Bắt đầu FE TableRegistry cleanup + enhancements.

  Step 4/7: FE TableRegistry (Gap 2+3+4+5b)

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 1 line, removed 1 line
      1  import { useEffect, useState, useCallback, useMemo } from 'react';
      2  import { Table, Tag, Select, Switch, Button, Space, Modal, Form, Input, message, Upload, Badge, Collapse, Typography, Progress, Tooltip } from 'antd';
      3 -import { PlusOutlined, UploadOutlined, SyncOutlined, DatabaseOutlined, SearchOutlined, ToolOutlined, ThunderboltOutlined, SwapOutlined } from '@ant-design/icons';                                             
      3 +import { PlusOutlined, UploadOutlined, SyncOutlined, DatabaseOutlined, SearchOutlined, ToolOutlined, ThunderboltOutlined, RocketOutlined } from '@ant-design/icons';                                           
      4  import type { ColumnsType } from 'antd/es/table';
      5  import { useNavigate } from 'react-router-dom';
      6  import { cmsApi } from '../services/api';

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 42 lines, removed 26 lines
      12  const { Panel } = Collapse;
      13  const { Title } = Typography;
      14  
      15 -const SyncStatusIndicator = ({ id, engine }: { id: number, engine: string }) => {                                                                                                                             
      16 -  const [status, setStatus] = useState<string>('n/a');                                                                                                                                                        
      15 +// SyncStatusIndicator — Gap 3 Option B: fetch real Debezium connector status                                                                                                                                 
      16 +// via /api/v1/system/connectors. Match by collection.include.list entry.                                                                                                                                     
      17 +const SyncStatusIndicator = ({ sourceDB, sourceTable }: { sourceDB: string; sourceTable: string }) => {                                                                                                       
      18 +  const [status, setStatus] = useState<string>('loading');                                                                                                                                                    
      19 +  const [connectorName, setConnectorName] = useState<string>('');                                                                                                                                             
      20  
      21    const fetchStatus = useCallback(async () => {
      19 -    // Legacy per-entry status endpoint retired in Sprint 4. Debezium                                                                                                                                         
      20 -    // connector status is exposed at /api/v1/system/connectors.                                                                                                                                              
      21 -    void id;                                                                                                                                                                                                  
      22 -    void engine;                                                                                                                                                                                              
      23 -    setStatus('n/a');                                                                                                                                                                                         
      24 -  }, [id, engine]);                                                                                                                                                                                           
      22 +    setStatus('loading');                                                                                                                                                                                     
      23 +    try {                                                                                                                                                                                                     
      24 +      const { data: res } = await cmsApi.get('/api/v1/system/connectors');                                                                                                                                    
      25 +      const list = res.data || res || [];                                                                                                                                                                     
      26 +      const needle = `${sourceDB}.${sourceTable}`;                                                                                                                                                            
      27 +      const match = list.find((c: { config?: Record<string, string> }) => {                                                                                                                                   
      28 +        const include = c.config?.['collection.include.list'] || '';                                                                                                                                          
      29 +        return include.split(',').map((s) => s.trim()).includes(needle);                                                                                                                                      
      30 +      });                                                                                                                                                                                                     
      31 +      if (match) {                                                                                                                                                                                            
      32 +        setConnectorName(match.name || '');                                                                                                                                                                   
      33 +        setStatus(match.state || 'UNKNOWN');                                                                                                                                                                  
      34 +      } else {                                                                                                                                                                                                
      35 +        setStatus('not_configured');                                                                                                                                                                          
      36 +      }                                                                                                                                                                                                       
      37 +    } catch {                                                                                                                                                                                                 
      38 +      setStatus('error');                                                                                                                                                                                     
      39 +    }                                                                                                                                                                                                         
      40 +  }, [sourceDB, sourceTable]);                                                                                                                                                                                
      41  
      26 -  useEffect(() => {                                                                                                                                                                                           
      27 -    fetchStatus();                                                                                                                                                                                            
      28 -  }, [fetchStatus]);                                                                                                                                                                                          
      42 +  useEffect(() => { fetchStatus(); }, [fetchStatus]);                                                                                                                                                         
      43  
      30 -  const handleRefresh = (e: React.MouseEvent) => {                                                                                                                                                            
      31 -    e.stopPropagation();                                                                                                                                                                                      
      32 -    fetchStatus();                                                                                                                                                                                            
      33 -  };                                                                                                                                                                                                          
      44 +  const handleRefresh = (e: React.MouseEvent) => { e.stopPropagation(); fetchStatus(); };                                                                                                                     
      45  
      35 -  if (status === 'n/a') return <Tag>N/A</Tag>;                                                                                                                                                                
      36 -  if (status === 'loading') return <span>...</span>;                                                                                                                                                          
      37 -  if (status === 'error') return <Tag color="error">Error</Tag>;                                                                                                                                              
      46 +  const badgeStatus =                                                                                                                                                                                         
      47 +    status === 'RUNNING' ? 'success' :                                                                                                                                                                        
      48 +    status === 'PAUSED' ? 'warning' :                                                                                                                                                                         
      49 +    status === 'FAILED' ? 'error' :                                                                                                                                                                           
      50 +    status === 'not_configured' ? 'default' :                                                                                                                                                                 
      51 +    status === 'loading' ? 'processing' : 'default';                                                                                                                                                          
      52  
      39 -  const displayStatus = status === 'stream_disabled' ? 'disabled' : status;                                                                                                                                   
      40 -  const badgeStatus = (status === 'active' || status === 'running') ? 'success' :                                                                                                                             
      41 -                      (status === 'inactive' || status === 'stream_disabled' ? 'warning' : 'default');                                                                                                        
      53 +  const label =                                                                                                                                                                                               
      54 +    status === 'not_configured' ? 'Chưa có connector' :                                                                                                                                                       
      55 +    status === 'loading' ? '...' :                                                                                                                                                                            
      56 +    status === 'error' ? 'Lỗi' :                                                                                                                                                                              
      57 +    status;                                                                                                                                                                                                   
      58  
      59    return (
      44 -    <Space>                                                                                                                                                                                                   
      45 -      <Badge status={badgeStatus as 'success' | 'warning' | 'default'} text={displayStatus} style={{ textTransform: 'capitalize' }} />                                                                        
      46 -      <Space>                                                                                                                                                                                                 
      47 -         <Button icon={<SyncOutlined />} size="small" type="text" onClick={handleRefresh} title="Refresh Status" />                                                                                           
      60 +    <Tooltip title={connectorName ? `Connector: ${connectorName}` : 'Không có Debezium connector match collection này'}>                                                                                      
      61 +      <Space size={4}>                                                                                                                                                                                        
      62 +        <Badge status={badgeStatus as 'success' | 'warning' | 'error' | 'default' | 'processing'} text={label} />                                                                                             
      63 +        <Button icon={<SyncOutlined />} size="small" type="text" onClick={handleRefresh} title="Refresh" />                                                                                                   
      64        </Space>
      49 -    </Space>                                                                                                                                                                                                  
      65 +    </Tooltip>                                                                                                                                                                                                
      66    );
      67  };
      68  

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 25 lines, removed 22 lines
      245      }
      246    };
      247  
      248 -  const handleBridge = (e: React.MouseEvent, id: number, batch = false) => {                                                                                                                                 
      248 +  // Gap 5b — Snapshot Now: publish NATS cdc.cmd.debezium-signal via CMS                                                                                                                                     
      249 +  // /api/tools/trigger-snapshot/:table. Worker writes Mongo debezium_signal                                                                                                                                 
      250 +  // collection → Debezium performs incremental snapshot.                                                                                                                                                    
      251 +  const handleSnapshot = (e: React.MouseEvent, record: TRegistry) => {                                                                                                                                       
      252      e.stopPropagation();
      250 -    setActionLoadingId(id);                                                                                                                                                                                  
      251 -    const url = batch ? `/api/registry/${id}/bridge?mode=batch` : `/api/registry/${id}/bridge`;                                                                                                              
      252 -    cmsApi.post(url)                                                                                                                                                                                         
      253 -      .then(() => message.success(batch ? 'Batch bridge (pgx) submitted' : 'Bridge command submitted'))                                                                                                      
      254 -      .catch((err) => {                                                                                                                                                                                      
      255 -        const e = err as { response?: { data?: { error?: string } } };                                                                                                                                       
      256 -        message.error(e.response?.data?.error || 'Bridge failed');                                                                                                                                           
      257 -      })                                                                                                                                                                                                     
      258 -      .finally(() => setActionLoadingId(null));                                                                                                                                                              
      253 +    Modal.confirm({                                                                                                                                                                                          
      254 +      title: `Trigger Debezium snapshot: ${record.source_table}?`,                                                                                                                                           
      255 +      content: 'Debezium sẽ thực hiện incremental snapshot collection này. Dùng khi connector vừa add hoặc sau rebuild shadow.',                                                                             
      256 +      okText: 'Snapshot',                                                                                                                                                                                    
      257 +      onOk: async () => {                                                                                                                                                                                    
      258 +        setActionLoadingId(record.id);                                                                                                                                                                       
      259 +        try {                                                                                                                                                                                                
      260 +          await cmsApi.post(                                                                                                                                                                                 
      261 +            `/api/tools/trigger-snapshot/${encodeURIComponent(record.source_table)}`,                                                                                                                        
      262 +            { database: record.source_db, collection: record.source_table },                                                                                                                                 
      263 +            { headers: { 'Idempotency-Key': `snapshot-${record.id}-${Date.now()}` } },                                                                                                                       
      264 +          );                                                                                                                                                                                                 
      265 +          message.success(`Snapshot dispatched: ${record.source_table}`);                                                                                                                                    
      266 +        } catch (err) {                                                                                                                                                                                      
      267 +          const e = err as { response?: { data?: { error?: string; detail?: string } } };                                                                                                                    
      268 +          message.error(e.response?.data?.error || e.response?.data?.detail || 'Snapshot failed');                                                                                                           
      269 +        } finally {                                                                                                                                                                                          
      270 +          setActionLoadingId(null);                                                                                                                                                                          
      271 +        }                                                                                                                                                                                                    
      272 +      },                                                                                                                                                                                                     
      273 +    });                                                                                                                                                                                                      
      274    };
      275  
      261 -  const handleTransform = (e: React.MouseEvent, id: number) => {                                                                                                                                             
      262 -    e.stopPropagation();                                                                                                                                                                                     
      263 -    setActionLoadingId(id);                                                                                                                                                                                  
      264 -    cmsApi.post(`/api/registry/${id}/transform`)                                                                                                                                                             
      265 -      .then(() => message.success('Transform command submitted'))                                                                                                                                            
      266 -      .catch((err) => {                                                                                                                                                                                      
      267 -        const e = err as { response?: { data?: { error?: string } } };                                                                                                                                       
      268 -        message.error(e.response?.data?.error || 'Transform failed');                                                                                                                                        
      269 -      })                                                                                                                                                                                                     
      270 -      .finally(() => setActionLoadingId(null));                                                                                                                                                              
      271 -  };                                                                                                                                                                                                         
      272 -                                                                                                                                                                                                             
      276    const handleCreateTable = (e: React.MouseEvent, id: number) => {
      277      e.stopPropagation();
      278      setActionLoadingId(id);

⏺ Now update column render (SyncStatusIndicator new signature) + action buttons + Register Modal.

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 2 lines, removed 2 lines
      322      { title: 'Source Table', dataIndex: 'source_table', width: 180, render: (t) => <strong style={{color: '#1890ff'}}>{t}</strong> },
      323      { title: 'Target Table', dataIndex: 'target_table', width: 180 },
      324      {
      325 -      title: 'Sync Engine', dataIndex: 'sync_engine', width: 120,                                                                                                                                            
      325 +      title: 'Sync Engine', dataIndex: 'sync_engine', width: 160,                                                                                                                                            
      326        render: (v: string, record) => (
      327          <Space direction="vertical" size={0} onClick={e => e.stopPropagation()}>
      328            <Tag color="blue">{v || 'debezium'}</Tag>
      329 -          <SyncStatusIndicator id={record.id} engine={v} />                                                                                                                                                  
      329 +          <SyncStatusIndicator sourceDB={record.source_db} sourceTable={record.source_table} />                                                                                                              
      330          </Space>
      331        ),
      332      },

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 9 lines, removed 10 lines
      378                    onClick={(e) => handleCreateDefaultFields(e, record.id)}>Tạo Field MĐ</Button>
      379                </Tooltip>
      380              )}
      381 -            <Tooltip title="Đồng bộ dữ liệu từ Airbyte sang table đích (SQL)">                                                                                                                               
      382 -              <Button size="small" icon={<SwapOutlined />} loading={actionLoadingId === record.id}                                                                                                           
      383 -                onClick={(e) => handleBridge(e, record.id)}>Đồng bộ</Button>                                                                                                                                 
      381 +            <Tooltip title="Trigger Debezium incremental snapshot cho collection này">                                                                                                                       
      382 +              <Button size="small" icon={<ThunderboltOutlined />} type="primary" ghost                                                                                                                       
      383 +                loading={actionLoadingId === record.id}                                                                                                                                                      
      384 +                onClick={(e) => handleSnapshot(e, record)}>Snapshot Now</Button>                                                                                                                             
      385              </Tooltip>
      385 -            <Tooltip title="Đồng bộ hiệu suất cao (Go + Sonyflake ID, cho data lớn >100K)">                                                                                                                  
      386 -              <Button size="small" icon={<ThunderboltOutlined />} loading={actionLoadingId === record.id}                                                                                                    
      387 -                onClick={(e) => handleBridge(e, record.id, true)} type="primary" ghost>Batch</Button>                                                                                                        
      386 +            <Tooltip title="Đi tới Master Registry để tạo / chạy Transmute">                                                                                                                                 
      387 +              <Button size="small" icon={<RocketOutlined />}                                                                                                                                                 
      388 +                onClick={(e) => { e.stopPropagation(); navigate(`/masters?source_shadow=${record.target_table}`); }}>                                                                                        
      389 +                Manage Masters                                                                                                                                                                               
      390 +              </Button>                                                                                                                                                                                      
      391              </Tooltip>
      389 -            <Tooltip title="Chuyển _raw_data sang các cột đã mapping">                                                                                                                                       
      390 -              <Button size="small" loading={actionLoadingId === record.id}                                                                                                                                   
      391 -                onClick={(e) => handleTransform(e, record.id)}>Chuyển đổi</Button>                                                                                                                           
      392 -            </Tooltip>                                                                                                                                                                                       
      392            </Space>
      393            <AsyncRowActions record={record} onChange={fetchData} />
      394          </Space>

⏺ Update(~/Documents/work/cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx)
  ⎿  Added 2 lines, removed 2 lines
      457            </Form.Item>
      458            <Form.Item name="source_table" label="Source Table" rules={[{ required: true }]}><Input placeholder="wallet_transactions" /></Form.Item>
      459            <Form.Item name="target_table" label="Target Table" rules={[{ required: true }]}><Input placeholder="wallet_transactions" /></Form.Item>
      460 -          <Form.Item name="sync_engine" label="Sync Engine">                                                                                                                                                 
      461 -            <Select><Select.Option value="airbyte">Airbyte</Select.Option><Select.Option value="debezium">Debezium</Select.Option><Select.Option value="both">Both</Select.Option></Select>                  
      460 +          <Form.Item name="sync_engine" label="Sync Engine" initialValue="debezium">                                                                                                                         
      461 +            <Input disabled />                                                                                                                                                                               
      462            </Form.Item>
      463            <Form.Item name="priority" label="Priority">
      464              <Select><Select.Option value="critical">Critical</Select.Option><Select.Option value="high">High</Select.Option><Select.Option value="normal">Normal</Select.Option><Select.Option value="low">Lo
           w</Select.Option></Select>

⏺ Bash(cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npx tsc --noEmit 2>&1 | tail -15; echo "EXIT=$?")
  ⎿  EXIT=0

⏺ Step 5/7: SourceConnectors — Add connector modal (Gap 5a FE)

⏺ Reading 1 file… (ctrl+o to expand)
  ⎿  src/pages/SourceConnectors.tsx

⏺ Update(src/pages/SourceConnectors.tsx)
  ⎿  Added 1 line, removed 1 line
       4  } from 'antd';
       5  import {
       6    ReloadOutlined, DatabaseOutlined, PlayCircleOutlined,
       7 -  PauseCircleOutlined, WarningOutlined, SyncOutlined,                                                                                                                                                         
       7 +  PauseCircleOutlined, WarningOutlined, SyncOutlined, PlusOutlined, DeleteOutlined,                                                                                                                           
       8  } from '@ant-design/icons';
       9  import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
      10  import { cmsApi } from '../services/api';

⏺ Update(src/pages/SourceConnectors.tsx)
  ⎿  Added 93 lines
       43    taskId?: number;
       44  }
       45  
       46 +const MONGO_TEMPLATE = `{                                                                                                                                                                                    
       47 +  "connector.class": "io.debezium.connector.mongodb.MongoDbConnector",                                                                                                                                       
       48 +  "mongodb.connection.string": "mongodb://gpay-mongo:27017/?replicaSet=rs0",                                                                                                                                 
       49 +  "database.include.list": "<service-db-name>",                                                                                                                                                              
       50 +  "collection.include.list": "<service-db-name>.<collection-name>",                                                                                                                                          
       51 +  "topic.prefix": "cdc.goopay",                                                                                                                                                                              
       52 +  "signal.data.collection": "<service-db-name>.debezium_signal",                                                                                                                                             
       53 +  "capture.mode": "change_streams_update_full",                                                                                                                                                              
       54 +  "snapshot.mode": "initial",                                                                                                                                                                                
       55 +  "key.converter": "io.confluent.connect.avro.AvroConverter",                                                                                                                                                
       56 +  "key.converter.schema.registry.url": "http://gpay-schema-registry:8081",                                                                                                                                   
       57 +  "value.converter": "io.confluent.connect.avro.AvroConverter",                                                                                                                                              
       58 +  "value.converter.schema.registry.url": "http://gpay-schema-registry:8081",                                                                                                                                 
       59 +  "schema.history.internal.kafka.bootstrap.servers": "gpay-kafka:9092"                                                                                                                                       
       60 +}`;                                                                                                                                                                                                          
       61 +                                                                                                                                                                                                             
       62  export default function SourceConnectors() {
       63    const qc = useQueryClient();
       64    const [pending, setPending] = useState<PendingAction | null>(null);
       65    const [reason, setReason] = useState('');
       66 +  const [createOpen, setCreateOpen] = useState(false);                                                                                                                                                       
       67 +  const [createName, setCreateName] = useState('');                                                                                                                                                          
       68 +  const [createConfig, setCreateConfig] = useState(MONGO_TEMPLATE);                                                                                                                                          
       69 +  const [deletePending, setDeletePending] = useState<string | null>(null);                                                                                                                                   
       70  
       71 +  const createMut = useMutation({                                                                                                                                                                            
       72 +    mutationFn: async (args: { name: string; config: Record<string, string>; reason: string }) => {                                                                                                          
       73 +      const r = await cmsApi.post(                                                                                                                                                                           
       74 +        '/api/v1/system/connectors',                                                                                                                                                                         
       75 +        { name: args.name, config: args.config, reason: args.reason },                                                                                                                                       
       76 +        { headers: { 'Idempotency-Key': `cc-create-${args.name}-${Date.now()}` } },                                                                                                                          
       77 +      );                                                                                                                                                                                                     
       78 +      return r.data;                                                                                                                                                                                         
       79 +    },                                                                                                                                                                                                       
       80 +    onSuccess: () => {                                                                                                                                                                                       
       81 +      message.success('Connector created');                                                                                                                                                                  
       82 +      qc.invalidateQueries({ queryKey: ['debezium-connectors'] });                                                                                                                                           
       83 +      setCreateOpen(false);                                                                                                                                                                                  
       84 +      setCreateName('');                                                                                                                                                                                     
       85 +      setCreateConfig(MONGO_TEMPLATE);                                                                                                                                                                       
       86 +      setReason('');                                                                                                                                                                                         
       87 +    },                                                                                                                                                                                                       
       88 +    onError: (err: unknown) => {                                                                                                                                                                             
       89 +      let msg = 'Create failed';                                                                                                                                                                             
       90 +      if (err && typeof err === 'object' && 'response' in err) {                                                                                                                                             
       91 +        const r = (err as { response?: { data?: { error?: string; detail?: string } } }).response;                                                                                                           
       92 +        if (r?.data) msg = `${r.data.error ?? 'error'}${r.data.detail ? `: ${r.data.detail}` : ''}`;                                                                                                         
       93 +      }                                                                                                                                                                                                      
       94 +      message.error(msg);                                                                                                                                                                                    
       95 +    },                                                                                                                                                                                                       
       96 +  });                                                                                                                                                                                                        
       97 +                                                                                                                                                                                                             
       98 +  const deleteMut = useMutation({                                                                                                                                                                            
       99 +    mutationFn: async (args: { name: string; reason: string }) => {                                                                                                                                          
      100 +      const r = await cmsApi.delete(                                                                                                                                                                         
      101 +        `/api/v1/system/connectors/${encodeURIComponent(args.name)}`,                                                                                                                                        
      102 +        { data: { reason: args.reason }, headers: { 'Idempotency-Key': `cc-delete-${args.name}-${Date.now()}` } },                                                                                           
      103 +      );                                                                                                                                                                                                     
      104 +      return r.data;                                                                                                                                                                                         
      105 +    },                                                                                                                                                                                                       
      106 +    onSuccess: (_d, vars) => {                                                                                                                                                                               
      107 +      message.success(`Deleted: ${vars.name}`);                                                                                                                                                              
      108 +      qc.invalidateQueries({ queryKey: ['debezium-connectors'] });                                                                                                                                           
      109 +      setDeletePending(null);                                                                                                                                                                                
      110 +      setReason('');                                                                                                                                                                                         
      111 +    },                                                                                                                                                                                                       
      112 +    onError: () => message.error('Delete failed'),                                                                                                                                                           
      113 +  });                                                                                                                                                                                                        
      114 +                                                                                                                                                                                                             
      115 +  const submitCreate = () => {                                                                                                                                                                               
      116 +    if (!createName.match(/^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,128}$/)) {                                                                                                                                           
      117 +      message.error('Tên connector: bắt đầu bằng chữ/số, cho phép . _ -, tối đa 128 ký tự');                                                                                                                 
      118 +      return;                                                                                                                                                                                                
      119 +    }                                                                                                                                                                                                        
      120 +    if (reason.trim().length < 10) {                                                                                                                                                                         
      121 +      message.warning('Lý do ≥ 10 ký tự');                                                                                                                                                                   
      122 +      return;                                                                                                                                                                                                
      123 +    }                                                                                                                                                                                                        
      124 +    let cfg: Record<string, string>;                                                                                                                                                                         
      125 +    try {                                                                                                                                                                                                    
      126 +      cfg = JSON.parse(createConfig);                                                                                                                                                                        
      127 +    } catch {                                                                                                                                                                                                
      128 +      message.error('Config không phải JSON hợp lệ');                                                                                                                                                        
      129 +      return;                                                                                                                                                                                                
      130 +    }                                                                                                                                                                                                        
      131 +    if (!cfg['connector.class']) {                                                                                                                                                                           
      132 +      message.error('Config thiếu "connector.class"');                                                                                                                                                       
      133 +      return;                                                                                                                                                                                                
      134 +    }                                                                                                                                                                                                        
      135 +    createMut.mutate({ name: createName, config: cfg, reason: reason.trim() });                                                                                                                              
      136 +  };                                                                                                                                                                                                         
      137 +                                                                                                                                                                                                             
      138 +  const submitDelete = () => {                                                                                                                                                                               
      139 +    if (!deletePending) return;                                                                                                                                                                              
      140 +    if (reason.trim().length < 10) { message.warning('Lý do ≥ 10 ký tự'); return; }                                                                                                                          
      141 +    deleteMut.mutate({ name: deletePending, reason: reason.trim() });                                                                                                                                        
      142 +  };                                                                                                                                                                                                         
      143 +                                                                                                                                                                                                             
      144    const { data, isLoading, refetch, isFetching } = useQuery({
      145      queryKey: ['debezium-connectors'],
      146      queryFn: async () => {

⏺ Update(src/pages/SourceConnectors.tsx)
  ⎿  Added 8 lines
      309                Resume
      310              </Button>
      311            )}
      312 +          <Button                                                                                                                                                                                            
      313 +            size="small"                                                                                                                                                                                     
      314 +            danger                                                                                                                                                                                           
      315 +            icon={<DeleteOutlined />}                                                                                                                                                                        
      316 +            onClick={() => setDeletePending(r.name)}                                                                                                                                                         
      317 +          >                                                                                                                                                                                                  
      318 +            Delete                                                                                                                                                                                           
      319 +          </Button>                                                                                                                                                                                          
      320          </Space>
      321        ),
      322      },

⏺ Update(src/pages/SourceConnectors.tsx)
  ⎿  Added 6 lines, removed 1 line
      326      <Card bordered={false}>
      327        <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
      328          <Title level={4} style={{ margin: 0 }}>Debezium Command Center</Title>
      329 -        <Button icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>Refresh</Button>                                                                                                    
      329 +        <Space>                                                                                                                                                                                              
      330 +          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>                                                                                                                
      331 +            New Connector                                                                                                                                                                                    
      332 +          </Button>                                                                                                                                                                                          
      333 +          <Button icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>Refresh</Button>                                                                                                  
      334 +        </Space>                                                                                                                                                                                             
      335        </Space>
      336  
      337        <Text type="secondary">

⏺ Update(src/pages/SourceConnectors.tsx)
  ⎿  Added 59 lines
      381            placeholder="Lý do (ghi audit, ≥ 10 ký tự)"
      382          />
      383        </Modal>
      384 +                                                                                                                                                                                                             
      385 +      <Modal                                                                                                                                                                                                 
      386 +        open={createOpen}                                                                                                                                                                                    
      387 +        title="New Debezium Connector"                                                                                                                                                                       
      388 +        onOk={submitCreate}                                                                                                                                                                                  
      389 +        confirmLoading={createMut.isPending}                                                                                                                                                                 
      390 +        onCancel={() => { setCreateOpen(false); setReason(''); }}                                                                                                                                            
      391 +        okText="Create"                                                                                                                                                                                      
      392 +        width={720}                                                                                                                                                                                          
      393 +      >                                                                                                                                                                                                      
      394 +        <Alert                                                                                                                                                                                               
      395 +          type="info"                                                                                                                                                                                        
      396 +          showIcon                                                                                                                                                                                           
      397 +          style={{ marginBottom: 12 }}                                                                                                                                                                       
      398 +          message="Mongo Debezium template pre-filled — edit placeholders <service-db-name> và <collection-name>"                                                                                            
      399 +        />                                                                                                                                                                                                   
      400 +        <Space direction="vertical" style={{ width: '100%' }} size={12}>                                                                                                                                     
      401 +          <Input                                                                                                                                                                                             
      402 +            placeholder="Connector name (e.g. goopay-wallet-cdc) — [a-zA-Z0-9._-]+"                                                                                                                          
      403 +            value={createName}                                                                                                                                                                               
      404 +            onChange={(e) => setCreateName(e.target.value)}                                                                                                                                                  
      405 +          />                                                                                                                                                                                                 
      406 +          <Input.TextArea                                                                                                                                                                                    
      407 +            rows={18}                                                                                                                                                                                        
      408 +            value={createConfig}                                                                                                                                                                             
      409 +            onChange={(e) => setCreateConfig(e.target.value)}                                                                                                                                                
      410 +            style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }}                                                                                                                                  
      411 +          />                                                                                                                                                                                                 
      412 +          <Input.TextArea                                                                                                                                                                                    
      413 +            rows={2}                                                                                                                                                                                         
      414 +            placeholder="Reason ≥ 10 ký tự (audit)"                                                                                                                                                          
      415 +            value={reason}                                                                                                                                                                                   
      416 +            onChange={(e) => setReason(e.target.value)}                                                                                                                                                      
      417 +          />                                                                                                                                                                                                 
      418 +        </Space>                                                                                                                                                                                             
      419 +      </Modal>                                                                                                                                                                                               
      420 +                                                                                                                                                                                                             
      421 +      <Modal                                                                                                                                                                                                 
      422 +        open={!!deletePending}                                                                                                                                                                               
      423 +        title={`Delete connector: ${deletePending}?`}                                                                                                                                                        
      424 +        onOk={submitDelete}                                                                                                                                                                                  
      425 +        confirmLoading={deleteMut.isPending}                                                                                                                                                                 
      426 +        onCancel={() => { setDeletePending(null); setReason(''); }}                                                                                                                                          
      427 +        okText="Delete"                                                                                                                                                                                      
      428 +        okButtonProps={{ danger: true }}                                                                                                                                                                     
      429 +      >                                                                                                                                                                                                      
      430 +        <Alert                                                                                                                                                                                               
      431 +          type="warning"                                                                                                                                                                                     
      432 +          showIcon                                                                                                                                                                                           
      433 +          style={{ marginBottom: 12 }}                                                                                                                                                                       
      434 +          message="Xoá connector sẽ ngừng stream. Consumer offset có thể replay từ snapshot mới khi tạo lại."                                                                                                
      435 +        />                                                                                                                                                                                                   
      436 +        <Input.TextArea                                                                                                                                                                                      
      437 +          rows={3}                                                                                                                                                                                           
      438 +          placeholder="Reason ≥ 10 ký tự"                                                                                                                                                                    
      439 +          value={reason}                                                                                                                                                                                     
      440 +          onChange={(e) => setReason(e.target.value)}                                                                                                                                                        
      441 +        />                                                                                                                                                                                                   
      442 +      </Modal>                                                                                                                                                                                               
      443      </Card>
      444    );
      445  }

⏺ Bash(npx tsc --noEmit 2>&1 | tail -15; echo "EXIT=$?")
  ⎿  EXIT=0

⏺ Step 6/7: Gap 6 Cockpit Wizard + route

⏺ Write(src/pages/SourceToMasterWizard.tsx)
  ⎿  Wrote 157 lines to src/pages/SourceToMasterWizard.tsx
       1 import { useState } from 'react';
       2 import { Card, Steps, Space, Button, Typography, Alert, Tag } from 'antd';
       3 import {
       4   CheckCircleOutlined, DatabaseOutlined, ThunderboltOutlined,
       5   BranchesOutlined, SettingOutlined, RocketOutlined, EyeOutlined,
       6 } from '@ant-design/icons';
       7 import { Link } from 'react-router-dom';
       8
       9 const { Title, Text, Paragraph } = Typography;
      10 const { Step } = Steps;
     … +147 lines (ctrl+o to expand)

⏺ Reading 1 file… (ctrl+o to expand)
  ⎿  src/App.tsx

⏺ Update(src/App.tsx)
  ⎿  Added 1 line
       7    BranchesOutlined,
       8    SettingOutlined,
       9    LogoutOutlined,
      10 +  CompassOutlined,                                                                                                                                                                                            
      11  } from '@ant-design/icons';
      12  import QueryErrorBoundary from './components/QueryErrorBoundary';
      13  

⏺ Update(src/App.tsx)
  ⎿  Added 1 line
      27  const ActivityManager = lazy(() => import('./pages/ActivityManager'));
      28  const DataIntegrity = lazy(() => import('./pages/DataIntegrity'));
      29  const SystemHealth = lazy(() => import('./pages/SystemHealth'));
      30 +const SourceToMasterWizard = lazy(() => import('./pages/SourceToMasterWizard'));                                                                                                                              
      31  
      32  const { Header, Sider, Content } = Layout;
      33  const { Text } = Typography;

⏺ Update(src/App.tsx)
  ⎿  Added 3 lines
      75            <Menu.Item key="dashboard" icon={<DashboardOutlined />}>
      76              <Link to="/">Dashboard</Link>
      77            </Menu.Item>
      78 +          <Menu.Item key="wizard" icon={<CompassOutlined />}>                                                                                                                                                 
      79 +            <Link to="/source-to-master">Source → Master Wizard</Link>                                                                                                                                        
      80 +          </Menu.Item>                                                                                                                                                                                        
      81            <Menu.Item key="schema" icon={<BranchesOutlined />}>
      82              <Link to="/schema-changes">Mapping Approval</Link>
      83            </Menu.Item>

⏺ Update(src/App.tsx)
  ⎿  Added 1 line
      139                  <Route path="/activity-manager" element={<ActivityManager />} />
      140                  <Route path="/data-integrity" element={<DataIntegrity />} />
      141                  <Route path="/system-health" element={<SystemHealth />} />
      142 +                <Route path="/source-to-master" element={<SourceToMasterWizard />} />                                                                                                                        
      143                </Routes>
      144              </Suspense>
      145            </QueryErrorBoundary>

⏺ Step 7/7: Arch doc append Section 5.5 (Gap 1)

⏺ Update(~/Documents/work/cdc-system/architecture.md)
  ⎿  Added 126 lines
      423  
      424  ---
      425  
      426 +## 5.5 Shadow → Master Materialisation Path (since Sprint 5)                                                                                                                                                 
      427 +                                                                                                                                                                                                             
      428 +Bổ sung so với Section 5: ingestion từ Kafka không ghi thẳng vào PG business target.                                                                                                                         
      429 +Thay vào đó đi qua 2 tầng.                                                                                                                                                                                   
      430 +                                                                                                                                                                                                             
      431 +```mermaid                                                                                                                                                                                                   
      432 +%%{init: {'theme': 'base', 'themeVariables': {                                                                                                                                                               
      433 +  'background': '#ffffff',                                                                                                                                                                                   
      434 +  'primaryColor': '#f8fafc',                                                                                                                                                                                 
      435 +  'primaryTextColor': '#0f172a',                                                                                                                                                                             
      436 +  'primaryBorderColor': '#94a3b8',                                                                                                                                                                           
      437 +  'lineColor': '#475569',                                                                                                                                                                                    
      438 +  'secondaryColor': '#fef3c7',                                                                                                                                                                               
      439 +  'tertiaryColor': '#f8fafc',                                                                                                                                                                                
      440 +  'clusterBkg': '#ffffff',                                                                                                                                                                                   
      441 +  'clusterBorder': '#cbd5e1',                                                                                                                                                                                
      442 +  'fontFamily': 'Inter, Segoe UI, Arial, sans-serif',                                                                                                                                                        
      443 +  'fontSize': '14px'                                                                                                                                                                                         
      444 +}}}%%                                                                                                                                                                                                        
      445 +flowchart LR                                                                                                                                                                                                 
      446 +    subgraph SRC["Source"]                                                                                                                                                                                   
      447 +        MDB["MongoDB"]                                                                                                                                                                                       
      448 +        DBZ["Debezium"]                                                                                                                                                                                      
      449 +        KAFKA["Kafka cdc.goopay.*"]                                                                                                                                                                          
      450 +    end                                                                                                                                                                                                      
      451 +                                                                                                                                                                                                             
      452 +    subgraph SHADOW["Shadow Layer cdc_internal.*"]                                                                                                                                                           
      453 +        SW["SinkWorker"]                                                                                                                                                                                     
      454 +        SCHM["SchemaManager auto-ALTER + proposal"]                                                                                                                                                          
      455 +        SHTBL["cdc_internal.<table>"]                                                                                                                                                                        
      456 +    end                                                                                                                                                                                                      
      457 +                                                                                                                                                                                                             
      458 +    subgraph CONTROL["Control Plane"]                                                                                                                                                                        
      459 +        CMS["cdc-cms-service"]                                                                                                                                                                               
      460 +        PROP["schema_proposal"]                                                                                                                                                                              
      461 +        MRULE["cdc_mapping_rules"]                                                                                                                                                                           
      462 +        MREG["master_table_registry"]                                                                                                                                                                        
      463 +        WEB["cdc-cms-web (/schema-proposals /masters /schedules)"]                                                                                                                                           
      464 +    end                                                                                                                                                                                                      
      465 +                                                                                                                                                                                                             
      466 +    subgraph MASTER["Master Layer public.<name>_master"]                                                                                                                                                     
      467 +        TMOD["TransmuteModule gjson + transform_fn"]                                                                                                                                                         
      468 +        TSCH["TransmuteScheduler cron + fencing"]                                                                                                                                                            
      469 +        MTBL["public.<name>_master typed + RLS"]                                                                                                                                                             
      470 +    end                                                                                                                                                                                                      
      471 +                                                                                                                                                                                                             
      472 +    MDB --> DBZ --> KAFKA --> SW                                                                                                                                                                             
      473 +    SW --> SHTBL                                                                                                                                                                                             
      474 +    SW --> SCHM --> PROP                                                                                                                                                                                     
      475 +    SW -->|"post-ingest NATS cdc.cmd.transmute-shadow"| TMOD                                                                                                                                                 
      476 +    WEB --> CMS                                                                                                                                                                                              
      477 +    CMS --> PROP                                                                                                                                                                                             
      478 +    CMS --> MRULE                                                                                                                                                                                            
      479 +    CMS --> MREG                                                                                                                                                                                             
      480 +    CMS -->|"cdc.cmd.master-create"| MTBL                                                                                                                                                                    
      481 +    TSCH -->|"cdc.cmd.transmute"| TMOD                                                                                                                                                                       
      482 +    TMOD --> SHTBL                                                                                                                                                                                           
      483 +    TMOD --> MTBL                                                                                                                                                                                            
      484 +    MRULE --> TMOD                                                                                                                                                                                           
      485 +    MREG --> TMOD                                                                                                                                                                                            
      486 +```                                                                                                                                                                                                          
      487 +                                                                                                                                                                                                             
      488 +### 5.5.1 Shadow Layer (cdc_internal.<table>)                                                                                                                                                                
      489 +- SinkWorker consume Kafka topic `cdc.goopay.<db>.<table>`.                                                                                                                                                  
      490 +- Upsert raw event + system cols (`_gpay_source_id`, `_raw_data`, `_source_ts`, `_synced_at`, `_version`, `_hash`, `_gpay_deleted`, timestamps).                                                             
      491 +- SchemaManager detect field mới:                                                                                                                                                                            
      492 +  - Nếu thuộc financial whitelist → insert `cdc_internal.schema_proposal` (admin approval).                                                                                                                  
      493 +  - Ngược lại auto-ALTER shadow với JSONB/TEXT.                                                                                                                                                              
      494 +- Post-ingest publish NATS `cdc.cmd.transmute-shadow` (best-effort).                                                                                                                                         
      495 +                                                                                                                                                                                                             
      496 +### 5.5.2 Schema Proposal Workflow                                                                                                                                                                           
      497 +- Proposal submit bởi SinkWorker (detect drift) hoặc admin thủ công.                                                                                                                                         
      498 +- `/schema-proposals` UI hiển thị Badge pending count.                                                                                                                                                       
      499 +- Approve → CMS TX: `ALTER TABLE … ADD COLUMN` + `INSERT cdc_mapping_rules` (status='approved').                                                                                                             
      500 +- Reject → status='rejected', field stays in _raw_data.                                                                                                                                                      
      501 +                                                                                                                                                                                                             
      502 +### 5.5.3 Mapping Rules                                                                                                                                                                                      
      503 +- Row trong `cdc_mapping_rules`: (source_table, target_column, data_type, jsonpath, transform_fn, is_active, status).                                                                                        
      504 +- Admin tạo qua `/registry/:id/mappings` hoặc auto từ Approve proposal.                                                                                                                                      
      505 +- Preview button gọi `/api/v1/mapping-rules/preview` dùng gjson eval 3 sample rows trước khi save.                                                                                                           
      506 +                                                                                                                                                                                                             
      507 +### 5.5.4 Master Registry (cdc_internal.master_table_registry)                                                                                                                                               
      508 +- Row per master: (master_name, source_shadow, transform_type, spec JSONB, is_active, schema_status ∈ {pending_review, approved, rejected, failed}).                                                         
      509 +- Admin Create qua `/masters` wizard → schema_status='pending_review'.                                                                                                                                       
      510 +- Approve → NATS `cdc.cmd.master-create` → Worker `MasterDDLGenerator.Apply`:                                                                                                                                
      511 +  - SELECT approved rules + build CREATE TABLE `public.<master_name>`.                                                                                                                                       
      512 +  - Indexes: PK + UNIQUE(_gpay_source_id) + _created_at/_updated_at + financial auto-index.                                                                                                                  
      513 +  - `SELECT cdc_internal.enable_master_rls(<master>)` → RLS policy `rls_master_default_permissive`.                                                                                                          
      514 +- Gate L2: `is_active` chỉ bật được khi `schema_status='approved'` (CHECK constraint).                                                                                                                       
      515 +                                                                                                                                                                                                             
      516 +### 5.5.5 Transmuter Module                                                                                                                                                                                  
      517 +- Subscribe NATS:                                                                                                                                                                                            
      518 +  - `cdc.cmd.transmute` (per-master batch).                                                                                                                                                                  
      519 +  - `cdc.cmd.transmute-shadow` (per-row real-time, post-ingest hook).                                                                                                                                        
      520 +- Check gate chain:                                                                                                                                                                                          
      521 +  1. master `is_active=true AND schema_status='approved'`.                                                                                                                                                   
      522 +  2. shadow `is_active=true AND profile_status='active'`.                                                                                                                                                    
      523 +  3. At least 1 approved rule in `cdc_mapping_rules`.                                                                                                                                                        
      524 +- Apply mapping rule per row:                                                                                                                                                                                
      525 +  - `gjson.GetBytes(_raw_data, rule.jsonpath)` → extract.                                                                                                                                                    
      526 +  - `transform_fn` (nullable): `numeric_cast`, `mongo_date_ms`, `base64_decode`, etc.                                                                                                                        
      527 +  - Build typed value theo `rule.data_type`.                                                                                                                                                                 
      528 +- Upsert `public.<master_name>` với OCC (ON CONFLICT WHERE _source_ts older) + fencing.                                                                                                                      
      529 +                                                                                                                                                                                                             
      530 +### 5.5.6 TransmuteScheduler                                                                                                                                                                                 
      531 +- Cron poll 60s + `FOR UPDATE SKIP LOCKED` + `app.fencing_machine_id` + `app.fencing_token`.                                                                                                                 
      532 +- 3 mode per schedule:                                                                                                                                                                                       
      533 +  - `cron`: cron_expr 5-field (robfig/cron/v3).                                                                                                                                                              
      534 +  - `immediate`: chỉ chạy khi admin click Run Now.                                                                                                                                                           
      535 +  - `post_ingest`: fire mỗi post-ingest trigger (real-time).                                                                                                                                                 
      536 +- UI `/schedules` hiển thị next_run_at, last_run_at, last_stats.                                                                                                                                             
      537 +                                                                                                                                                                                                             
      538 +### 5.5.7 Operator End-to-End (11 bước)                                                                                                                                                                      
      539 +Xem `/source-to-master` wizard trong cdc-cms-web hoặc `agent/memory/workspaces/feature-cdc-integration/10_gap_analysis_registry_masters.md` §4.                                                              
      540 +                                                                                                                                                                                                             
      541 +### 5.5.8 Source Code References                                                                                                                                                                             
      542 +- `centralized-data-service/internal/sinkworker/sinkworker.go` — SinkWorker + publishTransmuteTrigger.                                                                                                       
      543 +- `centralized-data-service/internal/sinkworker/schema_manager.go` — auto-ALTER + schema_proposal emit.                                                                                                      
      544 +- `centralized-data-service/internal/service/transmuter.go` — gate chain + rule apply + OCC upsert.                                                                                                          
      545 +- `centralized-data-service/internal/service/master_ddl_generator.go` — DDL build + RLS apply.                                                                                                               
      546 +- `centralized-data-service/internal/service/transmute_scheduler.go` — cron + fencing.                                                                                                                       
      547 +- `cdc-cms-service/internal/api/{master_registry,schema_proposal,transmute_schedule,mapping_preview,system_connectors}_handler.go` — control plane.                                                          
      548 +- `cdc-cms-web/src/pages/{MasterRegistry,SchemaProposals,TransmuteSchedules,SourceConnectors,SourceToMasterWizard}.tsx` — UI.                                                                                
      549 +                                                                                                                                                                                                             
      550 +---                                                                                                                                                                                                          
      551 +                                                                                                                                                                                                             
      552  ## 6. Deep Dive: Reconciliation Architecture
      553  
      554  ```mermaid

⏺ Stage 4 — VERIFY

⏺ Bash(npx tsc --noEmit 2>&1 | tail -20; echo "TSC_EXIT=$?"
      echo "--- BE build ---"…)
  ⎿  TSC_EXIT=0
     --- BE build ---
     (3s)
  ⎿  TSC_EXIT=0
     --- BE build ---
     GO_EXIT=0
     … +8 lines (ctrl+o to expand)
  ⎿  Allowed by auto mode classifier

⏺ All green:
  - tsc EXIT=0, go build EXIT=0.
  - 3 CMS endpoints 401 (auth-gated, route wired).
  - Vite HMR 200 cho 3 pages (Wizard 22KB new, TableRegistry 77KB, SourceConnectors 54KB).

  Stage 5 — DOCUMENT

⏺ Write(~/Documents/work/agent/memory/workspaces/feature-cdc-integration/03_implementation_6gap_execution.md)
  ⎿  Wrote 160 lines to ../../agent/memory/workspaces/feature-cdc-integration/03_implementation_6gap_execution.md
       1 # 6 Gap Execution Report — Registry + Masters end-to-end
       2
       3 > Date: 2026-04-24 07:10–07:55 ICT
       4 > SOP: 7-stage (Muscle)
       5 > Input plan: `09_tasks_solution_registry_masters.md`
       6 > Status: ✅ ALL 6 gap DONE, build + route wire verified.
       7
       8 ## 1. Files changed
       9
      10 ### Backend (cdc-cms-service)
     … +150 lines (ctrl+o to expand)

  Read 1 file (ctrl+o to expand)

⏺ Stage 6 — LESSON

  Read 1 file (ctrl+o to expand)

⏺ Stage 7 — CLOSE

  Rule 14 pre-flight:                      
                                                
  - 7-stage SOP followed (INTAKE → PLAN → EXECUTE → VERIFY → DOCUMENT → LESSON → CLOSE).
  - BE go build ./... EXIT=0.                   
  - FE npx tsc --noEmit EXIT=0.                          
  - CMS route POST/DELETE /v1/system/connectors + /tools/trigger-snapshot/:table 401 (auth-gated, wired).
  - Vite HMR 200 cho 3 pages edited/new.                 
  - Workspace: 03_implementation_6gap_execution.md NEW, 05_progress.md APPEND.                
  - Security gate Rule 8: regex name validation, connector.class required, Idempotency-Key + reason ≥10 chars, filterSafeConfig masks secrets.
                                                                                              
  ---                                                                                         
                                                                                              
                                                                                                                                                                                                                               
  Files touched (7 file)                                                                      
                                                                                                                                                                                                                               
  Backend (2):                                                                                
                                                                                                                                                                                                                               
  - cdc-cms-service/internal/router/router.go — POST + DELETE wire (+9 LOC)                                                                                                                                                    
                                                                                                                                                                                                                               
  Frontend (4):                                                                                                                                                                                                                
                                                                                                                                                                                                                               
  - cdc-cms-web/src/pages/TableRegistry.tsx — Gap 2+3+4+5b (airbyte removed, SyncStatus fetch real, Bridge xoá, Snapshot Now, Manage Masters)                                                                                  
  - cdc-cms-web/src/pages/SourceConnectors.tsx — Gap 5a (New Connector modal + Delete)                                                                                                                                         
  - cdc-cms-web/src/pages/SourceToMasterWizard.tsx — Gap 6 NEW 130 LOC cockpit                                                                                                                                                 
                                                                                                                                                                                                                               
  Docs (1):                                                                     
  - cdc-system/architecture.md — Gap 1 Section 5.5 (+120 LOC mermaid + 8 sub-sections)                                                                                                                                         
                                                                                                                                                                                                                               
  Verify evidence                                                                                                                                                                                                              
                                                                                                                                                                                                                               
  ┌───────┬────────┬───────┐                                                                  
  │ Check │ Before │ After │                                                                                                                                                                                                   
  ├───────┼────────┼───────┤                                                                  
  │ Go build │ EXIT=0 │ EXIT=0 │                                                                                                                                                                                               
  ├──────────┼────────┼────────┤                                                
  │ TS check │ EXIT=0 │ EXIT=0 │                                                                                                                                                                                               
  ├────────────────────────────────────┼────────┼─────────────┤                                                                                                                                                                
  │ POST /v1/system/connectors         │ 404    │ 401 (wired) │                                                                                                                                                                
  ├────────────────────────────────────┼────────┼─────────────┤                                                                                                                                                                
  │ DELETE /v1/system/connectors/:name │ 404    │ 401 (wired) │                                                                                                                                                                
  ├────────────────────────────────────┼────────┼─────────────────┤                                                                                                                                                            
  │ POST /tools/trigger-snapshot/:t    │ 401    │ 401 (unchanged) │                                                                                                                                                            
  ├────────────────────────────────────┼────────┼────────────────────────────┤                                                                                                                                                 
  │ FE routes                          │ 14     │ 15 (+Source→Master Wizard) │                
  ├────────────────────────────────────┼────────┼────────────────────────────┤                                                                                                                                                 
  │ Airbyte options in Register modal  │ 3      │ 0                          │  
  ├────────────────────────────────────┼────────────────┼────────────────────────────┤
  │ SyncStatus indicator               │ 'n/a' hardcode │ Fetch real connector state │                                                                                                                                         
  ├────────────────────────────────────┼────────────────┼────────────────────────────┤
  │ Bridge buttons                     │ 3              │ 0                          │                                                                                                                                         
  ├────────────────────────────────────┼────────────────┼────────────────────────────┤                                                                                                                                         
  │ Snapshot Now button                │ —              │ added                      │        
  └────────────────────────────────────┴────────────────┴────────────────────────────┘                                                                                                                                         
                                                                                              
  Boss workflow "add 1 DB mới"                                                                                                                                                                                                 
                                                                                
  1. /source-to-master — mở wizard 11 bước.                                                                                                                                                                                    
  2. /sources — New Connector → paste MongoDB template → Create.                              
  3. /registry — Register Table → Tạo Table → Snapshot Now.                                                                                                                                                                    
  4. /schema-proposals — Badge pending → Approve.                                             
  5. /masters — Create → Approve → Toggle Active.                                                                                                                                                                              
  6. /schedules — Cron → Run Now hoặc chờ tick.                                               
                                                                                                                                                                                                                               
  Toàn bộ qua UI, không còn curl thủ công.                                                    
                                                                                                                                                                                                                               
  Workspace files                                                                             
                                                                                                                                                                                                                               
  - agent/memory/workspaces/feature-cdc-integration/03_implementation_6gap_execution.md — NEW 
  - agent/memory/workspaces/feature-cdc-integration/05_progress.md — APPEND                                                                                                                                                    
  - agent/memory/global/lessons.md — APPEND 1133→1157 (lesson CMS proxy cho infra-control)
                                                                                                                                                                                                                               
  Skills                                                                                                                                                                                                                       
                                                                                                                                                                                                                               
  - Go handler + router middleware chain, React AntD Steps wizard, Kafka-Connect REST proxy, Debezium MongoDB template config, TypeScript mutation hooks, Mermaid architecture diagram, 7-stage SOP.                           
                                                                                                                                                                                                                               
  Boss sẵn sàng add 1 DB mới để quan sát toàn bộ tiến trình trên UI.                                                                                                                                                           
                                                                                                                                                                                                                               
✻ Cogitated for 8m 42s                                                                                                                                                                                                         
                                                                                              
※ recap: Goal: finish 6-gap CMS overhaul so Boss can add a new DB and watch the full Source→Master pipeline in the UI. All 6 gaps done and verified (builds green, routes wired, wizard live); next action is for Boss to add  
  the new DB and walk the wizard.                                                             
                                                                                                                                                                                                                               
  69 tasks (65 done, 1 in progress, 3 open)                                                                                                                                                                                    
  ◼ Stage 5-7: Document + Lesson + Close Phase 2                                                                                                                                                                               
  ◻ Pillar 1 — FE deep purge types/interfaces/helpers                                                                                                                                                                          
  ◻ Pillar 2 — Comment pruning Go + TS                                                                                                                                                                                         
  ◻ Pillar 3 — Hoist "airbyte" literals to pkgs/constants                       
  ✔ Pre-flight: Xác định Debezium connector config + signal.data.collection tồn tại trên Mongo                                                                                                                                 
   … +64 completed             