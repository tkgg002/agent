# 06_test_cases_check_connection — Test Plan

> **Phase**: `check_connection`
> **3 layers**: Unit (Go + TS) + Integration (HTTP) + E2E smoke

---

## 1. Unit test matrix — Worker (Go)

| TC | File | Function | Input | Expected |
|---|---|---|---|---|
| TC-WU-01 | `command_handler_test.go` (NEW) | `HandleDiscoverMongoDatabases` URI ưu tiên | payload `{uri:"mongodb://h1:27017",host:"h2",port:"27017"}` | service called với `uri=mongodb://h1:27017` (not h2) |
| TC-WU-02 | same | Fallback host+port | payload `{uri:"",host:"localhost",port:"27017"}` | service called với `uri=mongodb://localhost:27017` |
| TC-WU-03 | same | Both empty | payload `{uri:"",host:"",port:""}` | reply có `error="missing connection"` |
| TC-WU-04 | same | Only URI present | payload `{uri:"mongodb+srv://atlas.example.com"}` | service called với `uri=mongodb+srv://atlas.example.com` |
| TC-WU-05 | same | `HandleDiscoverMongoCollections` URI ưu tiên + database | payload `{uri:"...",database:"goopay_pbs"}` | service called với `(uri, "goopay_pbs")` |
| TC-WU-06 | same | Missing database | payload `{uri:"...",database:""}` | reply `error="missing database"` |
| TC-WU-07 | `mongo_introspection_test.go` (existing, KHÔNG đụng) | — | — | regression check: no failure |

## 2. Unit test matrix — BE relay (Go)

| TC | File | Function | Input | Expected |
|---|---|---|---|---|
| TC-BU-01 | `introspection_handler_test.go` (NEW or extend) | POST DiscoverMongoCollections body OK | body `{"uri":"mongodb://...","database":"goopay_pbs"}` | 200, NATS request sent với uri |
| TC-BU-02 | same | POST missing uri+host | body `{"database":"goopay_pbs"}` | 400 "missing uri or host" |
| TC-BU-03 | same | POST missing database | body `{"uri":"..."}` | 400 "missing database" |
| TC-BU-04 | same | GET legacy backward compat | `?host=localhost&port=27017&database=...` (or path :db) | 200, behavior unchanged |
| TC-BU-05 | same | NATS timeout simulation | mock NATS returns timeout | 504 với `status:"timeout"` |
| TC-BU-06 | same | Worker reply with cluster_err | mock NATS returns `{status:"cluster_err"}` | HTTP 200 với JSON status="cluster_err" + sanitized_dsn |

## 3. Unit test matrix — FE (TS, Jest + RTL nếu có)

| TC | File | Scenario | Input | Expected |
|---|---|---|---|---|
| TC-FU-01 | `useConnectorCheck.test.ts` (NEW) | Mutation success → state ok | mock axios returns `{status:"ok",collections:["a","b"]}` | hook state = `{status:"ok", collections:["a","b"]}` |
| TC-FU-02 | same | Mutation error 4xx | mock axios returns 400 | hook state = `{status:"unknown", error:"..."}` |
| TC-FU-03 | same | Reset | call `reset()` | hook state = null |
| TC-FU-04 | `SourceConnectors.test.tsx` (NEW or extend) | Check button click → spinner visible | render form, click [Check Connection] | Spin role=alert present |
| TC-FU-05 | same | Check OK → multi-select pre-filled all | mock check returns 3 collections | Form value `collectionNames=["a","b","c"]` |
| TC-FU-06 | same | Check OK → Create enabled | render after success | Create button not disabled |
| TC-FU-07 | same | Check FAIL → Create disabled | mock check returns cluster_err | Create button disabled |
| TC-FU-08 | same | Edit URI after check → reset | type new URI | checkResult null, multi-select disabled |
| TC-FU-09 | same | Multi-select uncheck → submit only checked | uncheck "b", click Create | Submit payload `collectionNames=["a","c"]` |

## 4. Integration test matrix (HTTP end-to-end, local stack)

| TC | Layer | Scenario | Steps | Pass criteria |
|---|---|---|---|---|
| TC-I-01 | BE → Worker → Mongo | Happy path list collections via URI | `curl -X POST http://localhost:8001/api/introspection/mongo/collections -d '{"uri":"mongodb://localhost:27017","database":"goopay_pbs"}'` | 200 với `{status:"ok",collections:[...]}` |
| TC-I-02 | BE → Worker → Mongo | URI có replica set | `curl ... -d '{"uri":"mongodb://node1:27017,node2:27017/?replicaSet=rs0","database":"..."}'` | 200 với collections |
| TC-I-03 | BE → Worker → Mongo | URI có auth | `curl ... -d '{"uri":"mongodb://user:pass@host/?authSource=admin","database":"..."}'` | 200 với collections (verify worker dùng auth) |
| TC-I-04 | BE → Worker | DB không tồn tại | `database:"nonexistent_xyz"` | 200 với `status:"db_missing"` + `available_databases:[...]` |
| TC-I-05 | BE → Worker | Mongo cluster down | `uri:"mongodb://localhost:9999"` (port sai) | 200 với `status:"cluster_err"` (HOẶC 504) |
| TC-I-06 | BE | Backward compat GET | `curl 'http://localhost:8001/api/introspection/mongo/databases?host=localhost&port=27017'` | 200 với databases list (như cũ) |
| TC-I-07 | BE | Auth required | request without JWT | 401 |
| TC-I-08 | Security | Sanitize URI trong response | URI có `user:pass@` | response `sanitized_dsn` không chứa `pass` |
| TC-I-09 | Security | Sanitize URI trong worker log | check log file `/tmp/worker.log` sau TC-I-03 | `grep 'pass'` returns 0 |

## 5. E2E smoke matrix (UI + full stack)

| TC | Scenario | Steps | Pass criteria |
|---|---|---|---|
| TC-E-01 | Happy path Create | 1. Open Create Modal kiểu Mongo. 2. Nhập URI `mongodb://localhost:27017` + Database `goopay_pbs`. 3. Click [Check]. 4. Wait spin. 5. Verify multi-select hiện collections (compare với `mongosh ... db.getCollectionNames()`). 6. Click Create. 7. Verify toast + Kafka Connect config. | All steps OK, `collection.include.list` = explicit list |
| TC-E-02 | Uncheck before Create | Sau TC-E-01.5, uncheck 2 collections. Click Create. | Kafka Connect config = chỉ collection checked |
| TC-E-03 | Cluster err UX | URI sai port. Check. | Alert error VN "Không kết nối được...". Create disabled. Multi-select disabled. |
| TC-E-04 | DB missing UX | URI đúng + Database `nonexistent`. Check. | Alert "Database `nonexistent` không tồn tại. Có sẵn: ..." + danh sách hiện. |
| TC-E-05 | DB empty UX | Tạo `db_empty_test` rỗng trên Mongo. URI + DB này. Check. | Alert "Database chưa có collection nào." Create disabled. |
| TC-E-06 | Auth error UX | URI `mongodb://wrong:wrong@host/?authSource=admin`. Check. | Alert đề cập auth. Sanitized DSN trong error. |
| TC-E-07 | State invalidate | TC-E-01 step 5 → đổi URI thành khác → multi-select clear + Create disable + checkResult null. | UI state reset. |
| TC-E-08 | State invalidate 2 | TC-E-01 step 5 → đổi Database thành khác → tương tự. | UI state reset. |
| TC-E-09 | Spam click prevent | Click [Check] 10 lần trong 5s. | Button disabled while pending, max 1 NATS request inflight |
| TC-E-10 | Edit existing connector | Edit connector cũ có `collection.include.list = "a,b,c"`. Open Modal. | Multi-select pre-fill `[a,b,c]` (KHÔNG ép re-check). Create button = Update enabled. |
| TC-E-11 | Edit existing rồi optional re-check | Edit + click [Check] | Re-check trigger, kết quả overwrite multi-select (giữ overlap selected nếu match). |
| TC-E-12 | Large DB (>500 col) | URI tới DB với 500+ collections | Multi-select render OK (Antd virtual scroll), Create OK |
| TC-E-13 | Timeout | Worker offline (kill process) | After 10s, Alert "Worker không phản hồi" |

## 6. Negative / security tests

| TC | Scenario | Pass criteria |
|---|---|---|
| TC-S-01 | URI password log leak | `grep -E 'mongodb://[^/]+:[^@]+@' /tmp/worker.log /tmp/cdc-cms.log` | 0 hits |
| TC-S-02 | URI password trong network response | DevTools Network → response payload | Không chứa raw password, chỉ `***` |
| TC-S-03 | XSS in collection name | Mongo create collection tên `<script>alert(1)</script>` | Antd Select escape, render text plain |
| TC-S-04 | SQL/NoSQL injection in DB name | `database: "db'; DROP TABLE--"` | Treated as literal string, escape OK |
| TC-S-05 | JWT missing | Request without auth header | 401, no NATS call |
| TC-S-06 | CSRF (nếu cookie auth) | POST without CSRF token | 403 (nếu middleware có) |

## 7. Test data setup

### Mongo local seed

```js
// mongosh "mongodb://localhost:27017"

use goopay_pbs
db.users.insertOne({ _id: "u1", name: "Alice" })
db.orders.insertOne({ _id: "o1", amount: 100 })
db.payments.insertOne({ _id: "p1", method: "card" })

use db_empty_test
// (no insert — DB chỉ xuất hiện sau khi có >= 1 collection,
// nên tạo dummy rồi drop)
db.tmp.insertOne({_:1})
db.tmp.drop()
// → DB sẽ tự xóa nếu không còn collection nào. Test "empty" thực chất khó.
// Alternative: tạo collection rỗng:
db.createCollection("placeholder_will_drop")
// rồi smoke test sẽ thấy 1 collection "placeholder_will_drop"
// → đây không phải "empty" strict; refine TC-E-05:
//   - Tạo db_with_one_coll → collection rỗng → multi-select hiện 1 option
//   - "empty" thực sự = drop hết → DB tự disappear → fall vào db_missing case
// NOTE: Mongo semantic: DB không tồn tại implicit cho đến khi có collection.
//   → 'empty' chỉ tồn tại trong 1 race window rất ngắn.
//   → Trong UC này, sẽ rare. ADR-003 vẫn map ‘empty’ để complete.
```

### Verify commands

```bash
# Compare với BE response
mongosh "mongodb://localhost:27017/goopay_pbs" --eval 'db.getCollectionNames()' --json relaxed

# Curl test BE
curl -X POST http://localhost:8001/api/introspection/mongo/collections \
  -H 'Authorization: Bearer <jwt>' \
  -H 'Content-Type: application/json' \
  -d '{"uri":"mongodb://localhost:27017","database":"goopay_pbs"}' | jq

# Kafka Connect config verify
curl -s http://localhost:8083/connectors/<name>/config | jq '.["collection.include.list"]'
```

## 8. Pass / Fail criteria tổng

| Criterion | Pass | Fail |
|---|---|---|
| Worker unit tests | 7/7 PASS | < 7/7 |
| BE unit tests | 6/6 PASS | < 6/6 |
| FE unit tests | 9/9 PASS (nếu test setup tồn tại; nếu không có Jest → SKIP, ghi rõ trong report) | < 9/9 |
| Integration tests | 9/9 PASS | < 9/9 |
| E2E smoke happy | TC-E-01, E-02 MUST PASS | bất kỳ MUST fail |
| E2E smoke negative | TC-E-03, E-04, E-05, E-07, E-08 MUST PASS | bất kỳ MUST fail |
| Security tests | TC-S-01, S-02 MUST PASS | bất kỳ MUST fail |
| Build/Vet/Lint | All exit 0 | non-zero |
| Pre-existing test regression | 0 new failure | bất kỳ test cũ PASS → giờ fail |

## 9. Test artifacts

Lưu vào `/tmp/check_connection_*.log`:
- `/tmp/check_connection_worker_test.log`
- `/tmp/check_connection_be_test.log`
- `/tmp/check_connection_fe_test.log`
- `/tmp/check_connection_build_worker.log`
- `/tmp/check_connection_build_be.log`
- `/tmp/check_connection_build_fe.log`
- `/tmp/check_connection_smoke_happy.log`
- `/tmp/check_connection_smoke_negative.log`
- `/tmp/check_connection_security.log`
- `/tmp/check_connection_log_grep_password.log` (verify 0 hits)
- `/tmp/check_connection_screens/*.png` (screenshots)
