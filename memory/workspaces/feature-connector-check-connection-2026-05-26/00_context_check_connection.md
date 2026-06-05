# 00_context_check_connection — Bối cảnh

> **Workspace**: `feature-connector-check-connection-2026-05-26`
> **Phase**: `check_connection`
> **Date**: 2026-05-26
> **Owner Brain**: Antigravity (Chairman) — Plan only theo §1, §12
> **Owner Muscle**: claude-sonnet-4-6 / claude-opus-4-7 (TBD bởi user)
> **Governance**: CLAUDE.md §0..§14 + GEMINI.md (đã đọc đầu phiên)
> **Lessons đọc trước**: lessons.md L-2026-05-19 "Mongo Scan-Fields Pattern" (5-case branching) + L-1276 "Wizard session" + L-3070 "Báo cáo láo: caller không gọi resolver" + L-3275 "Sanitize DSN".

---

## 1. Bối cảnh nghiệp vụ

User propose pattern UX mới cho form **New Connector** (`cdc-cms-web` SourceConnectors.tsx):

> Khi user nhập **MongoDB Connection URL** + **Database** → bấm nút **Check Connection** → tiến trình (progress bar) chạy → nếu OK: trả về danh sách collection của DB đó → render dạng **multi-select** (default chọn HẾT) + enable nút **Create**. Tương tự cho MySQL/Postgres (tương lai). Trước khi check pass → nút Create disabled.

**Lý do giải pháp này tốt hơn workspace cũ** `feature-connector-default-collections-2026-05-25`:
- Workspace cũ chỉ thêm hint text "để trống = CDC tất cả", UX implicit, dễ sai khi user gõ collection tên sai.
- Workspace mới explicit: user thấy danh sách thật của Mongo, chọn sub-set hoặc giữ all → KHÔNG còn cơ hội nhập typo, KHÔNG còn ambiguity "không filter vs intentional all".
- Bonus: catch connection error sớm (URL sai, DB không tồn tại, credentials sai) NGAY trước khi tạo connector — không tạo ra connector lỗi rồi mới phát hiện.

Workspace `feature-connector-default-collections-2026-05-25` được đánh dấu **SUPERSEDED** trong `active_plans.md`.

## 2. Audit findings (đã verify qua subagent Explore)

| Layer | Trạng thái | Evidence |
|---|---|---|
| BE endpoint list databases | ✅ Tồn tại `GET /api/introspection/mongo/databases?host=&port=` | `cdc-cms-service/internal/api/introspection_handler.go:25` + `router.go:331` |
| BE endpoint list collections | ✅ Tồn tại `GET /api/introspection/mongo/:db/collections?host=&port=` | `introspection_handler.go:77` + `router.go:332` |
| Worker service layer | ✅ `DiscoverDatabases(uri)` + `DiscoverCollections(uri, dbName)` nhận **full URI** | `centralized-data-service/internal/service/mongo_introspection.go:63,85` |
| Worker command handler | ⚠️ `HandleDiscoverMongoDatabases/Collections` chỉ nhận `host+port` rồi BUILD `mongodb://host:port` inline → BỎ qua auth / replicaSet | `command_handler.go:1164-1218` |
| 5-case diagnosis | ✅ `IntrospectDiagnosis` (`ok / cluster_err / db_missing / coll_missing / empty / no_fields`) | `mongo_introspection.go:45,149` |
| FE check-connection hook | ❌ KHÔNG TỒN TẠI | grep `useCheckConnection` = 0 hits |
| FE Collections field UI | ⚠️ Hiện là `<Input placeholder="users,orders,payments" />` (text tự nhập) | `SourceConnectors.tsx:966-969` |
| FE Modal pattern | ✅ Antd Modal đơn (không phải Drawer/Steps) | `SourceConnectors.tsx:878-1026` |
| Antd version | ✅ v6 (`^6.3.5`) → `<Select mode="multiple">` available | `cdc-cms-web/package.json` |
| Wizard session pattern | ✅ Tồn tại nhưng overkill cho UC này (chỉ cần sync request, không cần state machine multi-step) | `cdc-cms-service/internal/model/wizard_session.go:7-22` + router 348/349/394/395 |
| MySQL/PG introspect | ❌ Chưa có service / handler / NATS subject | grep `PostgresIntrospectionService` = 0 hits |
| Cross-DB abstraction | ❌ Không có interface chung | grep `SourceDriver` = 0 hits |

**Kết luận audit**: ~70% infrastructure đã có. Gap thật sự = (1) extend BE handler accept full URI + (2) FE thêm hook + UI flow. MySQL/PG defer phase sau.

## 3. Scope phase này

| In scope (P0) | Out of scope (defer) |
|---|---|
| ✅ MongoDB Check Connection happy path qua full URI | ❌ MySQL/Postgres Check Connection (cần implement service mới) |
| ✅ FE: nút Check + Spin/Steps + multi-select + gate Create | ❌ Cross-DB driver interface refactor |
| ✅ BE: extend `introspection_handler` + worker `command_handler` accept `uri` field | ❌ Wizard session table persistence (UC này stateless, không cần) |
| ✅ 5-case error UX: `cluster_err`, `db_missing`, `coll_missing`, `empty`, `no_fields` mapped sang i18n-friendly message | ❌ Save URL credentials encrypted (đã có flow riêng, không phải scope phase này) |
| ✅ Sanitize URI trước log/error (L-3275) | ❌ Real-time progress streaming (sync 10s là đủ — UX dùng `<Spin>` indeterminate) |
| ✅ Smoke test end-to-end với local Mongo | ❌ Auto-detect Mongo connector class theo URI scheme (mongodb:// vs mongodb+srv://) |

## 4. Constraints (ràng buộc)

1. **§12 Brain Code Prohibition**: Brain CHỈ tạo plan + tasks + solution demo. KHÔNG sửa code. Muscle thực thi sau user verb.
2. **§7 Full Doc Set**: Bộ doc 00..10 + report với suffix `_check_connection`.
3. **§11 APPEND-ONLY** cho `05_progress.md`.
4. **§3 Verify before Done**: Mọi gate phải có evidence thực tế. Smoke E2E PASS mới được report DONE.
5. **§6 Simplicity & Elegance**: Tận dụng tối đa code đã có. Worker service layer đã nhận full URI — chỉ cần fix handler relay. KHÔNG refactor service.
6. **L-3070 (báo cáo láo)**: Sau khi sửa, PHẢI verify end-to-end call chain — KHÔNG chỉ test handler isolated rồi tuyên bố done. CHECK trên environment thực mới đếm.
7. **L-3275 (sanitize DSN)**: TUYỆT ĐỐI không log raw URI có password. Dùng helper `SanitizeMongoDSN` đã có (xác nhận tồn tại trong `mongo_introspection.go`).
8. **User directive**: "không cheat db hay thay đổi các config để đạt đc kêt quả" → KHÔNG patch DB / KHÔNG hack k8s config. Mọi thay đổi qua source code đường chính thống.

## 5. Files liên quan (READ-only audit; Edit sẽ ở M2-M3)

### Backend (cdc-cms-service)

| Path | Mục đích | Action phase này |
|---|---|---|
| `internal/api/introspection_handler.go:25-150` | DiscoverMongoDatabases + DiscoverMongoCollections handler | EXTEND request DTO accept optional `uri` |
| `internal/router/router.go:331-332` | Route registration | KHÔNG đụng (path không đổi) |

### Worker (centralized-data-service)

| Path | Mục đích | Action phase này |
|---|---|---|
| `internal/handler/command_handler.go:1164-1218` | HandleDiscoverMongo{Databases,Collections} | EXTEND DTO accept `uri`, ưu tiên uri nếu có |
| `internal/service/mongo_introspection.go:63-200` | Service layer | KHÔNG đụng (đã đúng) |
| `internal/worker_server.go:283-284` | NATS subject subscription | KHÔNG đụng |

### Frontend (cdc-cms-web)

| Path | Mục đích | Action phase này |
|---|---|---|
| `src/pages/SourceConnectors.tsx:107` | MONGO_URL_RE regex | KHÔNG đụng |
| `src/pages/SourceConnectors.tsx:286-295` | Modal state + form hook | EXTEND: thêm state `checkResult` |
| `src/pages/SourceConnectors.tsx:878-1026` | Modal render | EDIT: thêm button Check + multi-select |
| `src/services/api.ts` | axios instances | EXTEND: thêm service function `checkMongoConnection` |
| `src/hooks/useRegistry.ts` (hoặc file mới `useConnectorCheck.ts`) | Hook layer | CREATE: `useCheckMongoConnection` |

## 6. Stakeholders

| Vai trò | Người | Vai phase này |
|---|---|---|
| Product / Ops | trainguyen | Quyết định UX wording + approve plan |
| Brain | Antigravity | Plan + design + verify gate review |
| Muscle | CC CLI (sonnet-4-6 hoặc opus-4-7) | Thực thi sau approve |
| Security | `/security-agent` | Review trước Done (M7) |

## 7. References

- Lessons cross-reference: `agent/memory/global/lessons.md`
  - L-2026-05-19 Mongo Scan-Fields 5-case Pattern (line 3275+)
  - L-2026-04 báo cáo láo caller không gọi resolver (line 3070+)
  - L-1276 Wizard session draft-vs-execute split
  - L-2026-05-18 GetSourceDSN multi-scheme (line 27+)
- Tech stack: `agent/memory/global/tech_stack.md` — React 19, Antd v6, mongo-driver v1.17.9, Go 1.22
- Project context: `agent/memory/global/project_context.md`
- Workspace SUPERSEDED: `feature-connector-default-collections-2026-05-25` (UX hint-only approach)
