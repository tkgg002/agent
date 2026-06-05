# 10_gap_analysis_check_connection — Gap Matrix

> **Phase**: `check_connection`
> **Baseline date**: 2026-05-26
> **Audit source**: subagent Explore (very thorough) trên 3 repo `cdc-cms-web` / `cdc-cms-service` / `centralized-data-service`.

---

## 1. Bird's-eye view

**Tổng kết**: ~70% hạ tầng cho feature ĐÃ TỒN TẠI. Gap chính dồn về 2 chỗ:
1. **Worker handler** đang build URI inline `mongodb://host:port` (drop auth/replicaSet).
2. **Frontend** chưa có hook check + Collections vẫn là text input.

Strategy: **EXTEND không refactor**. Add POST variant + FE component, giữ GET legacy (R12 backward compat).

---

## 2. Per-layer gap matrix

### 2.1 Layer: `centralized-data-service` (Go worker)

| Mục | Hiện trạng | Gap | Severity | In-scope phase này? |
|---|---|---|---|---|
| Service `DiscoverDatabases(uri)` | ✅ Tồn tại (mongo_introspection.go:63), nhận full URI | — | — | — |
| Service `DiscoverCollections(uri, db)` | ✅ Tồn tại (mongo_introspection.go:85), nhận full URI | — | — | — |
| `IntrospectCollectionDiagnose` 5-case | ✅ Tồn tại (mongo_introspection.go:149) | Chỉ dùng cho scan-fields, không reuse cho list-collections | LOW | NO (chỉ map status sang reply) |
| Handler `HandleDiscoverMongoDatabases` | ⚠️ Tồn tại (command_handler.go:1164) nhưng nhận `host+port`, build `mongodb://host:port` inline | **DROP auth/replicaSet/TLS** | HIGH | ✅ YES (Edit #1) |
| Handler `HandleDiscoverMongoCollections` | ⚠️ Tồn tại (command_handler.go:1218) tương tự | Tương tự + thiếu 5-case status reply | HIGH | ✅ YES (Edit #2) |
| NATS subject `cdc.cmd.introspect.mongo.databases` | ✅ Registered (worker_server.go:283) | — | — | — |
| NATS subject `cdc.cmd.introspect.mongo.collections` | ✅ Registered (worker_server.go:284) | — | — | — |
| DSN sanitization helper | ✅ Tồn tại (L-3275 pattern) | Cần áp dụng nhất quán trong log line mới | LOW | ✅ YES (Edit #1, #2) |
| MySQL introspection service | ❌ Không tồn tại | Full implementation needed | — | ❌ NO (ADR-001, defer phase sau) |
| Postgres introspection service | ❌ Không tồn tại | Tương tự | — | ❌ NO (ADR-001) |

**Gap workload**: ~70 LOC extend 2 handler.

### 2.2 Layer: `cdc-cms-service` (Go BE relay)

| Mục | Hiện trạng | Gap | Severity | In-scope phase này? |
|---|---|---|---|---|
| Endpoint `GET /api/introspection/mongo/databases` | ✅ Tồn tại (router.go:331) | URI passed as query param → leak risk | MEDIUM | KEEP legacy, ADD POST (ADR-002) |
| Endpoint `GET /api/introspection/mongo/:db/collections` | ✅ Tồn tại (router.go:332) | Tương tự | MEDIUM | KEEP legacy, ADD POST |
| Handler `DiscoverMongoDatabases` (GET) | ✅ Tồn tại (introspection_handler.go:25) | — | — | — (giữ nguyên) |
| Handler `DiscoverMongoCollections` (GET) | ✅ Tồn tại (introspection_handler.go:77) | — | — | — (giữ nguyên) |
| Handler `DiscoverMongoDatabasesPost` (NEW) | ❌ Chưa có | Cần POST nhận body `{uri}` | HIGH | ✅ YES (Edit #3) |
| Handler `DiscoverMongoCollectionsPost` (NEW) | ❌ Chưa có | Cần POST nhận body `{uri, database}` | HIGH | ✅ YES (Edit #4) |
| Router register POST routes | ❌ Chưa có | Add 2 dòng vào router.go | LOW | ✅ YES (Edit #4.5) |
| NATS request-reply timeout 10s | ✅ Tồn tại pattern | Reuse trong handler mới | — | ✅ YES (reuse) |
| Error envelope chuẩn | ✅ Tồn tại pattern | Reuse | — | ✅ YES (reuse) |
| Auth middleware cho endpoint mới | ✅ `shared` router group có auth | — | — | ✅ YES (đặt vào group `shared`) |

**Gap workload**: ~100 LOC handler mới + 2 dòng router.

### 2.3 Layer: `cdc-cms-web` (React FE)

| Mục | Hiện trạng | Gap | Severity | In-scope phase này? |
|---|---|---|---|---|
| Modal New Connector | ✅ Tồn tại (SourceConnectors.tsx:878-1026) | — | — | — |
| Form fields: connectionUrl + databaseName | ✅ Tồn tại | — | — | — |
| Field `collectionNames` | ⚠️ Tồn tại nhưng là `<Input>` plain text | Phải đổi sang `<Select mode="multiple">` | HIGH | ✅ YES (Edit #9) |
| Service `connectorCheck.ts` | ❌ Chưa có | Cần file mới với `checkMongoDatabases` + `checkMongoCollections` | HIGH | ✅ YES (Edit #5, file mới) |
| Hook `useConnectorCheck.ts` | ❌ Chưa có | Cần file mới dùng React Query `useMutation` | HIGH | ✅ YES (Edit #6, file mới) |
| State `checkResult` + `checkStatus` | ❌ Chưa có | Cần local state + reducer logic | MEDIUM | ✅ YES (Edit #7) |
| Reset on URI/DB change | ❌ Chưa có | Cần Form.useWatch + useEffect invalidate | MEDIUM | ✅ YES (Edit #7) |
| Check button + Alert + Spin UX | ❌ Chưa có | UI block mới giữa fields và buttons | HIGH | ✅ YES (Edit #8) |
| Auto-select-all sau check PASS | ❌ Chưa có | `setFieldValue('collectionNames', response.collections)` | MEDIUM | ✅ YES (Edit #10) |
| Gate Create button | ❌ Chưa có | Modal `okButtonProps.disabled` | HIGH | ✅ YES (Edit #11) |
| VN error message map (5-case) | ❌ Chưa có | Helper `mapCheckStatusToVi` | MEDIUM | ✅ YES (Edit #12) |
| `buildConnectorConfig` handle array | ⚠️ Hiện handle string → string | Phải support array, prefix `${db}.${c}` | MEDIUM | ✅ YES (Edit #13) |
| Edit existing connector pre-fill | ⚠️ Hiện gán string trực tiếp | Phải split `collection.include.list` thành array | LOW | ✅ YES (Edit #14) |
| List view rendering empty collections | ✅ Workspace cũ đã đề xuất, chưa thi công | OUT-OF-SCOPE phase này | — | ❌ NO (supersede ADR-009) |
| Antd `<Select mode="multiple">` | ✅ Antd v6.3.5 support | — | — | — |
| Loading UX = `<Spin>` | ✅ Antd có | — | — | ✅ YES (ADR-005) |

**Gap workload**: ~245 LOC FE (2 file mới + ~120 LOC edit SourceConnectors.tsx).

---

## 3. In-scope vs DEFER summary

### In-scope phase này
- Worker handler accept full URI (Edit #1, #2)
- BE POST relay handler (Edit #3, #4, #4.5)
- FE service + hook (Edit #5, #6)
- FE UI integration: Check button, multi-select, gate Create (Edit #7-14)
- 5-case error map VN
- POST-only cho route mới (giữ GET legacy)

### DEFER (phase sau)
- MySQL/Postgres introspection (ADR-001 → phase `connector-check-mysql`, `connector-check-pg`)
- Cross-DB driver interface (ADR-007)
- Wizard session persistence (ADR-008)
- Streaming progress (real `<Progress percent>`) qua SSE/WebSocket (ADR-005)
- List view fallback render (workspace cũ SUPERSEDED — ADR-009)

---

## 4. Architectural debt audit

| Debt | Severity | Mitigation phase này |
|---|---|---|
| Worker handler legacy host+port → drop auth | MEDIUM | Extend (additive field `uri`), giữ host+port fallback để backward compat (Edit #1) |
| GET legacy có thể leak URI nếu caller bypass POST | LOW | Document trong code comment; future phase audit caller |
| 2 path (GET + POST) cho cùng resource | LOW | OK trong P0; deprecate GET trong phase sau |
| Worker không reuse driver pool giữa 2 calls (databases → collections) | LOW | Acceptable cho UC sync. Future: connection cache |
| FE store URI trong `Form` state có password | MEDIUM | Acceptable trong scope React app (memory only); KHÔNG persist localStorage |

---

## 5. Verification baseline (BEFORE state)

Trước khi Muscle thực thi, capture baseline để compare sau:

### 5.1 Endpoint baseline

```bash
# A. GET legacy (đã tồn tại) — phải VẪN PASS sau khi ship
curl -sS -X GET "http://localhost:8080/api/introspection/mongo/databases?host=localhost&port=27017" \
  -H "Authorization: Bearer <token>" | jq .

# B. POST mới (chưa tồn tại) — phải 404 BEFORE, 200 AFTER
curl -sS -X POST "http://localhost:8080/api/introspection/mongo/databases" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"uri":"mongodb://localhost:27017"}' | jq .
# Expected BEFORE: 404 Not Found
# Expected AFTER: 200 {"databases":[...]}
```

### 5.2 Frontend baseline

```
1. Open /source-connectors → click "New connector"
2. Fill connectionUrl = mongodb://localhost:27017
3. Fill databaseName = mydb
4. BEFORE: thấy `<Input>` text cho Collections, không có "Check Connection" button
5. AFTER: thấy "Check Connection" button + collections render multi-select sau check PASS
```

### 5.3 Worker baseline

```bash
# Test NATS round-trip
nats req cdc.cmd.introspect.mongo.databases \
  '{"host":"localhost","port":27017}' --timeout=5s
# Expected BEFORE: OK với host+port

nats req cdc.cmd.introspect.mongo.databases \
  '{"uri":"mongodb://user:pass@localhost:27017/?replicaSet=rs0"}' --timeout=5s
# Expected BEFORE: build URI inline drop creds → có thể fail auth
# Expected AFTER: dùng URI gốc → PASS với auth/replicaSet
```

### 5.4 Database baseline

KHÔNG có migration nào trong phase này. SELECT `cdc_source_connections` không thay đổi.

```sql
-- Snapshot trước/sau (phải bằng nhau)
SELECT COUNT(*), MIN(id), MAX(id) FROM cdc_source_connections;
```

---

## 6. Acceptance gate per file changed

| File | Verify command | Expected |
|---|---|---|
| `centralized-data-service/internal/handler/command_handler.go` | `go vet ./internal/handler/...` | no new warnings |
| `centralized-data-service/internal/handler/command_handler_test.go` | `go test ./internal/handler/...` | 7/7 unit (mocked) PASS |
| `cdc-cms-service/internal/api/introspection_handler.go` | `go test ./internal/api/...` | 6/6 PASS |
| `cdc-cms-service/internal/router/router.go` | `go build ./...` | PASS |
| `cdc-cms-web/src/services/connectorCheck.ts` (NEW) | `npm run typecheck` | 0 errors |
| `cdc-cms-web/src/hooks/useConnectorCheck.ts` (NEW) | `npm run typecheck` | 0 errors |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | `npm run build` + `npm run lint` | PASS |

---

## 7. Risk → mitigation re-check (mirror 01_requirements)

| Risk | Likelihood | Impact | Mitigation in scope |
|---|---|---|---|
| URI leak vào log/access log | HIGH | HIGH | ADR-002 POST body + sanitization helper |
| Worker timeout > 10s | MEDIUM | MEDIUM | 10s NATS timeout + clear timeout VN message |
| User edit existing connector breaks | LOW | MEDIUM | Edit #14 split string→array; KHÔNG ép re-check |
| Antd v6 multi-select bug với 1000+ collections | LOW | LOW | Acceptable; thực tế DB ít khi >100 collections; có search trong Select |
| Backward-compat GET caller broken | LOW | HIGH | KHÔNG xóa GET, chỉ ADD POST |
| MySQL/PG user expect tương tự | MEDIUM | LOW | Document defer rõ trong release note |

---

## 8. Open questions (chờ user/Muscle xác nhận)

1. **Q1**: Auth header truyền vào NATS payload cho worker introspect? — A: Hiện tại NATS chỉ trust internal; BE đã authenticate user → KHÔNG cần re-auth ở worker.
2. **Q2**: Có rate-limit endpoint POST `/check` không? — A: Phase này KHÔNG thêm (acceptable cho internal tool). Future phase nếu expose ngoài → add rate limit.
3. **Q3**: Cache check result trong server-side? — A: KHÔNG. Ephemeral, mỗi click = call mới. Tránh stale data.

---

## 9. Pre-execution check list (Muscle phải xác nhận trước khi run M1)

- [ ] Đã `git status` clean ở cả 3 repo
- [ ] Branch `feature/connector-check-connection-2026-05-26` đã tạo (hoặc xác nhận branch cùng tên đã có)
- [ ] Mongo test instance available: `mongodb://localhost:27017`
- [ ] Có Mongo có auth để test 5-case (e.g. `mongodb://user:pass@localhost:27017`)
- [ ] BE local đang chạy (port 8080) hoặc sẵn sàng restart sau build
- [ ] FE dev server (5173 hoặc port khác) hoặc sẵn sàng `npm run dev`
- [ ] Worker local đang chạy + connect NATS

Nếu BẤT KỲ item nào ❌ → STOP, báo Brain re-plan.

---
