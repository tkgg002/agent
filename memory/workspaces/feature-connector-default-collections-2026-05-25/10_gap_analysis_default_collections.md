# 10_gap_analysis_default_collections — Gap giữa yêu cầu & hiện trạng

> **Phase**: `default_collections`
> **Method**: Evidence-driven audit dựa trên direct file read (subagent #Explore).

---

## Bảng gap matrix

| Layer | Yêu cầu | Hiện trạng | Gap | Severity |
|---|---|---|---|---|
| **FE Form: hint text** | User hiểu rõ "để trống = CDC all" | KHÔNG có hint, placeholder `users,orders,payments` gợi ý required | UX confusion | 🟠 P1 (UX) |
| **FE Form: validation** | Field optional, accept empty | Field optional, no `rules: required` | — | 🟢 OK |
| **FE Form: `compactConfig`** | Drop empty value khỏi payload | `Object.entries(cfg).filter(([, value]) => value !== '')` (line 131-133) | — | 🟢 OK |
| **FE Form: `buildConnectorConfig`** | Không inject default khi empty | `if (collectionNames) cfg['collection.include.list'] = collectionNames;` (line 160-166) — conditional set | — | 🟢 OK |
| **BE handler: Create endpoint** | Accept config map as-is | `var req struct { Name string; Config map[string]string }` (line 168-171), forward to Kafka Connect không inject | — | 🟢 OK |
| **BE handler: Update endpoint** | Tương tự Create | Tương tự (assumption, verify ở T0.2) | — | 🟢 OK (giả định) |
| **Debezium Mongo connector** | Default CDC all khi missing `collection.include.list` | Per Debezium docs: missing key = no filter = CDC all | Cần verify version cụ thể trong T0.3 | 🟢 OK (verify needed) |
| **FE List view: hiển thị empty list** | Phân biệt "không filter" vs "đã filter" | Hiện hiển thị `-` hoặc empty (audit M1.2 confirm) | Visual confusion | 🟠 P1 (UX) |
| **Documentation** | Note default behavior trong README/handbook | Không có note | New onboarding miss | 🟡 P2 (defer) |
| **i18n setup** | Hint text đa ngôn ngữ | Chưa rõ (audit M1.4) | Có thể không có i18n → hardcode VN | 🟢 OK (conditional) |
| **A11y** | Hint text có ARIA association với input | Antd `Form.Item extra` tự sinh `aria-describedby` | — | 🟢 OK |

## Phân loại theo phase fix

### Trong scope phase `default_collections`

| Gap | Task | Solution ref |
|---|---|---|
| FE Form thiếu hint text | M2 T2.1 | Edit #1 |
| FE List view không phân biệt empty | M2 T2.3 | Edit #2 |
| i18n key (conditional) | M2 T2.2 | Edit #3 |
| Verify Debezium default behavior thực tế | M0 T0.3 + T0.5 | (Verify only, no edit) |

### Ngoài scope (defer)

| Gap | Lý do defer | Phase đề xuất |
|---|---|---|
| UI multi-select collection picker | Cần BE endpoint mới `GET /mongo/collections` | future phase `connector-collection-picker` |
| Validation format `db.collection,...` | Out of scope, không bắt buộc cho correctness | future phase `connector-filter-validate` |
| Documentation README update | Cosmetic, ít priority | future phase `docs-cms-handbook` |
| BE explicit inject `collection.include.list: "*.*"` | ADR-005 reject — implicit trust framework default OK | — |
| Apply pattern `(All X)` cho field `database.include.list` | Out of scope, có thể repeat pattern future | future phase `connector-default-display-unified` |

## Architectural debt liên quan

1. **FE compactConfig là implicit contract**: Phụ thuộc behavior của `Object.entries().filter()` để drop empty. Nếu future ai đó muốn gửi explicit empty value (`collection.include.list: ""`) — sẽ silent drop. Hiện tại đây là design intentional, nhưng cần document trong `03_implementation_default_collections.md` Section 1 (đã có).

2. **BE no-validation pass-through**: Handler chỉ forward map. Pro: linh hoạt. Con: không có check nào trên BE để catch user typo (vd `collection.includ.list` thiếu `e`). Phase này KHÔNG fix, defer governance future.

3. **Debezium version implicit dependency**: Default behavior phụ thuộc connector version. Nếu upgrade major (1.x → 2.x) có thể đổi semantics. Mitigate: M0.3 note version, smoke test trong M4 catch nếu đổi.

4. **Single-DB connector assumption**: Field `database.include.list` có thể chứa multi DB. Nếu user nhập 2 DB và để trống Collections → CDC all collections của CẢ 2 DB. Phase này KHÔNG specify behavior này, defer audit nếu user thắc mắc.

## Verification baseline (trước fix)

```bash
# 1. FE: render form, screenshot Collections field
# Expect baseline: placeholder "users,orders,payments", KHÔNG có extra text below

# 2. FE: tạo connector empty Collections, screenshot list view
# Expect baseline: cell hiển thị "-" hoặc empty

# 3. BE: curl create connector với body không có collection.include.list
curl -X POST http://localhost:<port>/api/system-connectors \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"test-empty-coll",
    "config":{
      "connector.class":"io.debezium.connector.mongodb.MongoDbConnector",
      "mongodb.hosts":"localhost:27017",
      "mongodb.name":"test_server",
      "database.include.list":"test_db"
    }
  }'
# Expect baseline: 200 OK

# 4. Kafka Connect: verify config
curl -s http://localhost:8083/connectors/test-empty-coll/config | jq
# Expect baseline: NO key "collection.include.list"

# 5. Debezium: insert doc new collection, kafkacat
mongosh "mongodb://localhost:27017/test_db" --eval 'db.brand_new.insertOne({_id:"baseline"})'
kafkacat -b localhost:9092 -t cdc.test_server.test_db.brand_new -C -e | head -3
# Expect baseline: HAS event → confirms Debezium default = CDC all
```

Nếu baseline #5 fail (NO event) → hypothesis sai → STOP, re-audit Debezium version.

Sau fix, expect:

| Verify | Expected |
|---|---|
| FE form Collections | Hint text visible below input |
| FE list view empty list | `(All collections)` italic gray |
| FE list view explicit list | Plain text, e.g., `users,orders` |
| BE behavior | KHÔNG đổi |
| Kafka Connect config | KHÔNG đổi |
| Debezium CDC | KHÔNG đổi (vẫn CDC all khi empty, vẫn filter khi explicit) |

## Risk gap (cần monitor sau release)

| Risk | Detection | Mitigation |
|---|---|---|
| Debezium upgrade đổi default | Smoke test trên environment trước prod | Note version trong report; tự động re-test trong CI nếu có |
| User upgrade tài liệu UI nhưng quên đồng bộ wording sang FE khác | Audit khi merge multi-repo | Centralize wording trong shared package (future phase) |
| Hint text quá dài làm vỡ layout horizontal form | UI review trước merge | Switch sang `tooltip` nếu chật |
| Existing connector có `collection.include.list = ""` empty string (vs null) | TC-U-05 cover | Fallback render handle cả 2 case |
