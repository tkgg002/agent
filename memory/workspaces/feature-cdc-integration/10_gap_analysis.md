# Gap Analysis: Code vs Documentation

> **Date**: 2026-03-30
> **Sources**: Code vs `update-sytem-design.md` + `09_tasks_solution.md`

---

## 0. Architecture Gap — Service Topology

### Design (theo `update-sytem-design.md`)

**4 services riêng biệt:**

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Auth Service │     │  CMS API     │     │  CDC Worker  │     │   CMS FE     │
│   (Go)       │     │  (Go/Fiber)  │     │   (Go)       │     │  (React)     │
│              │     │              │     │              │     │              │
│ Login        │     │ Schema CRUD  │     │ NATS consumer│     │ UI Dashboard │
│ Register     │◄────│ Registry CRUD│────►│ Schema Inspect│     │ Approve/Reject│
│ Issue JWT    │     │ Approve flow │     │ Batch upsert │     │ Registry mgmt│
│ RBAC         │     │ Airbyte API  │     │ (no auth)    │     │              │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
       ▲                    │ ▲                   ▲                    │
       │                    │ │                   │                    │
       │ REST (Login)       │ │ NATS (reload)     │ NATS (events)     │ REST + JWT
       │                    │ │                   │                    │
       └────────────────────┘ └───────────────────┘                   │
              FE → Auth          API → Worker                  FE → CMS API
```

**Luồng giao tiếp:**
| Giao tiếp | Phương thức | Nội dung |
|-----------|-------------|----------|
| FE → Auth Service | REST | Login / Refresh Token |
| FE → CMS API | REST + JWT | Quản lý Registry, Approve Schema |
| CMS API → Worker | NATS (Event) | Reload Config + Metadata người thực hiện |
| Worker → Postgres | SQL | Lưu dữ liệu + Audit Log (ai làm, lúc nào) |

### Current Code

**Chỉ có 2 projects:**

| Project | Maps to | Status |
|---------|---------|--------|
| `centralized-data-service` | CDC Worker | ✅ Done |
| `cdc-cms-service` | CMS API | ✅ Done |
| Auth Service | ❌ **MISSING** | Chưa tạo |
| CMS FE (React) | ❌ **MISSING** | Chưa tạo |

### Gap Detail

| # | Service | Status | Impact |
|---|---------|--------|--------|
| **Auth Service** | ❌ Missing | CMS API hiện verify JWT bằng shared secret. Không có Login/Register API. FE không có cách lấy token. |
| **CMS FE** | ❌ Missing | 08_tasks.md có task CDC-F1 (React + Ant Design) nhưng chưa bắt đầu. Cần project riêng. |
| **CMS API ↔ Auth** | ❌ Missing | CMS API cần AuthClient để verify token với Auth Service (gRPC hoặc internal HTTP). Hiện tại chỉ verify local. |

---

## 1. Auth Service Gaps

### 1.1 Login / Register API ❌

**Design**: `POST /api/auth/login` → issue JWT chứa `user_id`, `role`, `permissions`

**Code**: Không có. CMS API middleware chỉ parse JWT, không issue.

### 1.2 RBAC (Role-Based Access Control) ❌

**Design**:
- Role Admin: approve mọi thứ, sửa workflow
- Role Operator: chỉ xem + approve AI suggestions (>95% confidence)
- System User: cho AI Agent actions

**Code**: JWT middleware lấy `role` từ claims nhưng không enforce anywhere.

### 1.3 AuthClient (CMS API → Auth Service) ❌

**Design**: CMS API dùng AuthClient (inject qua Wire/DI) để verify token hoặc lấy user info từ Auth Service.

**Code**: Không có AuthClient. JWT verify trực tiếp bằng shared secret.

---

## 2. CMS API Gaps

### 2.1 Context Propagation qua NATS ❌

**Design**: Khi CMS approve, đính kèm `user_id` vào NATS payload `schema.config.reload`. Worker log "User X thay đổi Schema Y".

**Code hiện tại**:
```go
// approval_service.go — chỉ gửi table name
s.natsClient.Conn.Publish("schema.config.reload", []byte(pf.TblName))
```

**Cần sửa**: Gửi JSON `{table, user_id, action, timestamp}`

### 2.2 Role Enforcement ❌

**Code hiện tại**: Mọi authenticated user đều gọi được approve/reject/registry.

**Cần thêm**: Middleware check role per route group (admin-only routes vs operator routes).

---

## 3. CDC Worker Gaps

### 3.1 Dynamic Mapper Stub ❌ MISSING

File `dynamic_mapper.go` bị xoá khi tách CMS. Cần tạo lại trong `centralized-data-service`.

### 3.2 Audit Log từ NATS Reload ❌

**Design**: Worker nhận reload event → ghi "User X thay đổi Schema Y" vào `schema_changes_log`.

**Code**: Worker nhận reload → chỉ reload registry + clear cache. Không log user context.

### 3.3 NATS Credentials ❌

**Design**: Worker dùng Username/Password để connect NATS.

**Code**: `nats.Connect(url)` — không có auth. Config có `User`/`Pass` fields nhưng không dùng.

### 3.4 Batch Upsert ⚠️ PARTIAL

Single upsert loop thay vì true batch INSERT ... VALUES.

### 3.5 Unit Tests ❌ NOT STARTED

### 3.6 Prometheus Metrics ❌ NOT STARTED

---

## 4. CMS FE Gaps

### 4.1 Project chưa tồn tại ❌

08_tasks.md có CDC-F1 (React + Ant Design) nhưng chưa tạo project.

Theo design, CMS FE là project riêng, giao tiếp:
- FE → Auth Service: Login, get JWT
- FE → CMS API: REST + JWT header

---

## 5. Infrastructure Gaps

### 5.1 NATS Permissions/ACL ❌

**Design**: Chỉ Airbyte/Debezium publish `cdc.goopay.>`. Worker chỉ subscribe.

**Code**: NATS không có auth/permissions.

### 5.2 PostgreSQL User Separation ❌

**Design**:
- `cdc_worker`: chỉ INSERT/UPDATE
- `cms_service`: có DDL (ALTER TABLE, CREATE TABLE)

**Code**: Docker compose dùng chung user `user`.

### 5.3 NATS Auth cho Worker ❌

**Design**: Worker dùng credentials connect NATS.

**Code**: Config có fields nhưng `NewNatsClient()` chỉ dùng khi `User`/`Pass` non-empty → hiện tại rỗng.

---

## 6. Summary — 4 Service Mapping

| Service | Project | Status | Priority Fixes |
|---------|---------|--------|----------------|
| **Auth Service** | ❌ chưa tạo | Missing | P1 — cần cho CMS FE login |
| **CDC Worker** | `centralized-data-service` | ✅ core done | P1: dynamic_mapper stub, metrics, tests, context log |
| **CMS API** | `cdc-cms-service` | ✅ core done | P1: context propagation, RBAC, connect Auth Service |
| **CMS FE** | ❌ chưa tạo | Missing | P1 — task CDC-F1 trong 08_tasks.md |

---

## 7. Recommended Action Plan

### Phase 1A (Immediate — fix code gaps)
1. Tạo `dynamic_mapper.go` stub trong Worker
2. Fix context propagation (NATS payload chứa user_id)
3. Add RBAC role check trong CMS API handlers
4. Unit tests cho Worker + CMS API
5. Prometheus metrics integration

### Phase 1B (New services)
6. Tạo Auth Service project (Go/Fiber) — Login, Register, Issue JWT, RBAC
7. Tạo CMS FE project (React + Ant Design) — dashboard, approve/reject UI, registry manager
8. Connect CMS API → Auth Service (AuthClient)

### Phase 1C (Infrastructure)
9. NATS Permissions/ACL config
10. PostgreSQL user separation
11. NATS auth credentials cho Worker

### Phase 2
12. Dynamic Mapper full implementation
13. True batch upsert
14. Debezium activation
15. AI Auto-Reconcile integration
