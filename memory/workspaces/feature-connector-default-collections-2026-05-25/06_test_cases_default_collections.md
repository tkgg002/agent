# 06_test_cases_default_collections — Test Plan

> **Phase**: `default_collections`
> **Test strategy**: 3 layers — Unit (FE) + Integration (API) + E2E smoke (CDC pipeline)

---

## 1. Unit test matrix (FE)

| TC | Layer | Scenario | Input | Expected Output |
|---|---|---|---|---|
| TC-U-01 | FE Form | User để trống Collections, submit | `collectionNames = ''` | Payload BE không có key `collection.include.list` |
| TC-U-02 | FE Form | User nhập "users,orders" | `collectionNames = 'users,orders'` | Payload BE có `collection.include.list: 'users,orders'` |
| TC-U-03 | FE Form | User nhập whitespace `"   "` | `collectionNames = '   '` | (Behavior chốt: trim → empty → CDC all; HOẶC trim → empty → CDC all). Confirm với compactConfig hiện tại |
| TC-U-04 | FE List | Render row với `collection.include.list = undefined` | Config object không có key | Cell hiển thị `(All collections)` italic gray |
| TC-U-05 | FE List | Render row với `collection.include.list = ''` | empty string | Cell hiển thị `(All collections)` italic gray |
| TC-U-06 | FE List | Render row với `collection.include.list = 'users,orders'` | explicit list | Cell hiển thị `users,orders` plain |
| TC-U-07 | FE Form | Hint text visible | Render form | `<extra>` text visible bên dưới input, có ARIA association |
| TC-U-08 | FE Form | A11y: screen reader đọc hint | NVDA / VoiceOver | Hint text được read sau label |

**Tooling**: Jest + React Testing Library nếu project đã có test setup. Nếu không có → skip unit FE, dựa vào smoke M4.

## 2. Integration test matrix (API)

| TC | Layer | Scenario | Steps | Pass criteria |
|---|---|---|---|---|
| TC-I-01 | BE handler | POST connector body không có `collection.include.list` | `curl POST /api/system-connectors -d '{"name":"test","config":{"connector.class":"io.debezium.connector.mongodb.MongoDbConnector","mongodb.hosts":"...","database.include.list":"goopay_pbs"}}'` | 200 OK, connector created |
| TC-I-02 | BE handler | POST connector body có `collection.include.list: ''` empty string | Same body + `"collection.include.list":""` | 200 OK, forward as-is HOẶC drop tùy implementation hiện tại |
| TC-I-03 | BE handler | POST connector với explicit list | Body có `"collection.include.list":"goopay_pbs.users"` | 200 OK, Kafka Connect nhận đúng |
| TC-I-04 | Kafka Connect REST | GET config sau khi tạo (TC-I-01) | `curl GET /connectors/<name>/config` | JSON không có key `collection.include.list` |

**Tooling**: Postman / curl + `jq`. Verify qua Kafka Connect REST `http://<connect>:8083`.

## 3. E2E smoke matrix (CDC pipeline)

| TC | Layer | Scenario | Steps | Pass criteria |
|---|---|---|---|---|
| TC-E-01 | UI → BE → KC → Debezium → Kafka | Tạo connector empty Collections qua UI form | 1. FE dev server up. 2. Open Create Connector page. 3. Chọn kiểu Mongo, fill required fields, để TRỐNG Collections. 4. Submit. | Toast success, connector xuất hiện trong list, list view hiển thị `(All collections)` |
| TC-E-02 | Mongo → Debezium → Kafka | CDC all collections | 1. Sau TC-E-01. 2. Mongo insert doc vào collection `brand_new_test_coll` chưa từng được khai báo. 3. Kafkacat consume topic. | Topic `cdc.<server>.<db>.brand_new_test_coll` có event payload đúng |
| TC-E-03 | UI → BE → KC | Explicit list backward compat | 1. Tạo connector thứ 2 với `Collections = users,orders`. 2. Verify list view. 3. Mongo insert vào `brand_new_test_coll` cho connector này (notice: DB khác hoặc connector khác). | Connector này KHÔNG capture `brand_new_test_coll` (chỉ capture users + orders). List view hiển thị `users,orders` plain |
| TC-E-04 | UI render | Hint visible cả create form và edit form | Open Create + Open Edit existing | Cả 2 form đều hiển thị `extra` text |
| TC-E-05 | UI render | List view sort / filter không break | Click sort column Collections | Sort hoạt động bình thường (so sánh string, `(All collections)` xếp đầu hoặc cuối tùy implementation) |
| TC-E-06 | Edge case | Update existing connector từ explicit → empty | 1. Edit connector TC-E-03. 2. Clear Collections field. 3. Save. | Connector update thành công, list view chuyển sang `(All collections)`, Kafka Connect config update không còn key |
| TC-E-07 | Edge case | Update existing connector từ empty → explicit | 1. Edit connector TC-E-01. 2. Nhập `users`. 3. Save. | Connector update, chỉ capture users từ giờ |

## 4. Test data setup

### Mongo source (local stack)

```js
// Connect: mongodb://localhost:27017
use goopay_pbs

// Pre-existing collections (giả sử đã có)
db.users.insertOne({_id: "u1", name: "Alice"});
db.orders.insertOne({_id: "o1", amount: 100});

// Brand-new collection (chưa từng được listed)
db.brand_new_test_coll.insertOne({_id: "test1", marker: "default_collections_smoke_2026-05-25"});
```

### Kafka Connect verify

```bash
# List connectors
curl -s http://localhost:8083/connectors | jq

# Get config of specific connector
curl -s http://localhost:8083/connectors/<connector-name>/config | jq

# Get status
curl -s http://localhost:8083/connectors/<connector-name>/status | jq '.connector.state, .tasks[].state'
# Expect: RUNNING, RUNNING
```

### Kafka topic verify

```bash
# List topics matching connector
kafka-topics --bootstrap-server localhost:9092 --list | grep "^cdc\."

# Consume topic of brand_new_test_coll
kafkacat -b localhost:9092 -t cdc.<server>.goopay_pbs.brand_new_test_coll -C -e -o -1 | head -3
# Expect: JSON payload với "op":"c" hoặc snapshot event
```

## 5. Pass / Fail criteria tổng

| Criterion | Pass | Fail |
|---|---|---|
| FE build | exit 0 | exit non-zero |
| FE lint | exit 0 | new warnings |
| FE typecheck | exit 0 | new errors |
| Smoke TC-E-01 (UI create empty) | connector created, status RUNNING | error toast hoặc status FAILED |
| Smoke TC-E-02 (CDC all collections) | topic có event cho brand_new_test_coll | topic không tồn tại / không có event sau 60s wait |
| Smoke TC-E-03 (backward compat) | explicit connector KHÔNG capture collection không khai báo | regression |
| Hint visible TC-E-04 | text render đúng, ARIA OK | text missing / position lỗi |
| Security review | no HIGH/CRITICAL | có HIGH/CRITICAL |
| Pre-existing test regression | 0 new failure | bất kỳ test cũ chuyển từ PASS sang FAIL |

## 6. Test artifacts

Lưu vào `/tmp/default_collections_*.log`, attach evidence vào `report_default_collections_2026-05-25.md`:

- `/tmp/default_collections_build.log` — pnpm build output
- `/tmp/default_collections_lint.log` — pnpm lint output
- `/tmp/default_collections_tsc.log` — typecheck output
- `/tmp/default_collections_smoke_create.log` — TC-E-01 evidence (curl output + screenshot path)
- `/tmp/default_collections_smoke_cdc.log` — TC-E-02 evidence (kafkacat output)
- `/tmp/default_collections_security.log` — /security-agent output
- Screenshot: form Create với hint visible, list view với `(All collections)` cell, list view với explicit list cell

## 7. Negative tests (CẤM regression)

| Anti-pattern | Verify |
|---|---|
| Hint text bị inject HTML | Inspect DOM, không có `<script>` injectable |
| `extra` text che mất input | Form layout vertical OK; horizontal nếu chật → fallback `tooltip` |
| List view crash khi value null | TC-U-04 + TC-U-05 cover |
| Connector empty mà CDC sai collection (ví dụ collection của DB khác) | TC-E-02 kiểm tra topic name có đúng db name |
| User update empty → BE crash do thiếu key | TC-E-06 cover (cần BE PATCH/PUT handler chấp nhận missing key) |
