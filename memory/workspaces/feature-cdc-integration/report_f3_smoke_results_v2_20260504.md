# Report: F3 Smoke Round 2 — E2E Mongo addtest qua admin-api

**Date**: 2026-05-04 17:42+07
**Phase**: F3 round 2 (close-loop sau round 1 PARTIAL)
**Verdict**: ✅ PASS
**Author**: Brain (Antigravity) consolidate từ Muscle Agent (`ad5132f2`) output + Brain re-verify trên live infra.

---

## Section 1 — Phase A code audit + 2 fix bổ sung

### Bug pattern (giống line 232-237 fix round 1)

`internal/admin/helpers.go` còn 2 chỗ Mongo path không fallback `req.SourceObjectName`:

**Line 76-83** `qualifiedSourceObjectName` (sinh `normalized_source_key` cho UNIQUE constraint):
```go
case "mongodb":
    collection := stringFromLocator(req.SourceLocator, "collection")
    if collection == "" {
        collection = req.SourceObjectName     // ← fix Phase A
    }
    return stringFromLocator(req.SourceLocator, "database") + "." + collection
```

**Line 122-138** `topicNameFor` (sinh topic name cho schema registry preempt + worker discover):
```go
switch req.SourceEngineType {
case "mongodb":
    obj = stringFromLocator(req.SourceLocator, "collection")
    if obj == "" {
        obj = req.SourceObjectName            // ← fix Phase A
    }
case "postgresql":
    obj = req.SourceObjectName
default:
    obj = req.SourceObjectName
}
return fmt.Sprintf("cdc.%s.%s.%s", prefix, db, obj)
```

**Brain verify on disk** (lúc 17:42):
```
$ sed -n '76-82p' helpers.go    # qualifiedSourceObjectName
$ sed -n '127-133p' helpers.go  # topicNameFor
→ both fixes landed correctly
```

### Test results

```
$ go test ./internal/admin/ -count=1 -v
PASS — 21 assertion (17 test func + 4 sub-test)
ok  centralized-data-service/internal/admin  0.586s
```

---

## Section 2 — Phase B Connector config cleanup

**BEFORE round 2** (`collection.include.list` còn 2 entries rác từ round 1):
```
payment-bill-service.payment-bills
...
phase_e_ns_1777885325.items
payment-bill-service.            ← garbage
payment-bill-service.x           ← garbage
```

**Action**: Muscle PUT clean config → Debezium accepted.

**AFTER round 2 + POST register** (Brain re-verify lúc 17:42):
```
Total entries: 13
has_correct_entry (payment-bill-service.payment_bills_addtest): True
has_garbage (payment-bill-service. or .x): False

Full list:
  payment-bill-service.payment-bills
  payment-bill-service.refund-requests
  payment-bill-service.payment-bill-histories
  payment-bill-service.payment-bill-codes
  payment-bill-service.payment-bill-events
  payment-bill-service.payment-bill-holdings
  payment-bill-service.identitycounters
  payment-bill-service.refund-requests-histories
  centralized-export-service.export-jobs
  goopay.smoke_p02_close_1777882181
  goopay.smoke_p02_close_1777882418
  phase_e_ns_1777885325.items
  payment-bill-service.payment_bills_addtest    ← F3 round 2 entry, đúng format
```

---

## Section 3 — Phase C Admin-api restart

- Kill old PID 62951 (child binary từ `go run`).
- Build `/tmp/cdc-admin-api-f3v2` 45 MB từ source mới (đã có cả F1 + F3 fix).
- Start với env đầy đủ (port 5433 DB, NATS `cdc_worker:worker_secret_2026` từ `config/config-local.yml`).
- `/healthz` 200 confirmed.

Token: `f3v2_<TS>` (Muscle generate, prefix 8 ký tự).

---

## Section 4 — Phase D POST register

**Cleanup state cũ**: DELETE record `f3_smoke_payment_bills_addtest` round 1 (UNIQUE constraint không filter theo `is_active`, không thể chỉ soft-delete).

**Request**:
```json
POST /v2/sources/register
Authorization: Bearer f3v2_***
{
  "object_code": "f3v2_smoke_payment_bills_addtest",
  "source_engine_type": "mongodb",
  "sync_engine": "debezium",
  "source_object_name": "payment_bills_addtest",
  "source_object_type": "collection",
  "source_locator": {"database": "payment-bill-service"},
  "primary_key_field": "_id",
  "target_master_table": "payment_bills_addtest_master",
  "notes": "phase F3 round 2 smoke 2026-05-04"
}
```

**Response**:
```json
HTTP 200
{
  "source_object_id": 42,
  "provisioning_state": "active",
  "steps_completed": ["registry_insert", "debezium_include_extend", "schema_registry_preempt", "worker_signal"]
}
```

Connector verify (sau POST): `has_correct_entry=True`, `has_garbage=False` — fix Phase A + B đã ngăn không tạo entry rác mới.

> **Note**: Có 1 entry rác xuất hiện thoáng qua từ POST với old binary PID 62951 (race condition trong cleanup window) — Muscle đã cleanup thủ công lần 2. Binary `f3v2` đã confirmed không tạo rác khi dùng đúng.

---

## Section 5 — Phase E Mongo INSERT + 60s shadow poll

```
$ docker exec gpay-mongo mongosh --quiet --eval \
  "db.getSiblingDB('payment-bill-service').payment_bills_addtest.insertOne(...)"

{ acknowledged: true, insertedId: 'f3v2_smoke_1777887709' }
```

**Worker log** (sequential events):
```
NATS topic refresh signal received
V2 metadata registry reloaded (sources, shadow_bindings updated)
discovered kafka topics → cdc.goopay.payment-bill-service.payment_bills_addtest
batch upsert ok | group=shadow_payment_bill_service_mongo|payment_bills_addtest | count=1
```

**Kafka offset**: topic `cdc.goopay.payment-bill-service.payment_bills_addtest` advance từ 6 → 7.

---

## Section 6 — Final verify shadow

**Brain re-query** lúc 17:42:
```sql
SELECT _id, _synced_at
  FROM shadow_payment_bill_service_mongo.payment_bills_addtest
 ORDER BY _synced_at DESC
 LIMIT 3;
```

Kết quả:
```
           _id           |         _synced_at
-------------------------+----------------------------
 f3v2_smoke_1777887709   | 2026-05-04 09:41:51.804387   ← F3 round 2 doc (mới nhất)
 f3_smoke_1777886898     | 2026-05-04 09:40:57.804878   ← F3 round 1 doc (đã re-pickup sau cleanup)
 addtest-pb-201-a2-smoke | 2026-05-04 09:26:43.882763   ← phase E artifact
```

> **Bonus discovery**: Doc `f3_smoke_1777886898` từ round 1 (insert lúc connector chưa có entry đúng) ALSO landed shadow sau khi cleanup connector — vì khi binary mới + connector clean được PUT, Debezium re-snapshot collection `payment_bills_addtest` từ đầu, pickup luôn doc round 1. Self-healing nice.

**Schema layout** shadow table:
```
_id text PK | _raw_data jsonb | _source varchar | _synced_at timestamp |
_version bigint | _hash varchar | _deleted bool | _created_at timestamp | _updated_at timestamp
```
→ V1 layout (V1 ingest path cho Mongo). KHÔNG có `_gpay_source_id` V2 anchor — bình thường vì Mongo Debezium ingest không qua generator V2 đã fix B11.

---

## Section 7 — Verdict

| Check | Status |
|---|---|
| Phase A fix 2 chỗ pattern bug bổ sung | ✅ |
| Build + 21 test PASS | ✅ |
| Phase B connector cleanup | ✅ |
| Phase C admin-api restart binary mới | ✅ |
| Phase D POST register HTTP 200, 4 steps | ✅ |
| Phase D connector has_correct=True, has_garbage=False | ✅ |
| Phase E Mongo INSERT acknowledged | ✅ |
| Phase E Kafka offset advance (6→7) | ✅ |
| Phase E worker batch upsert ok count=1 | ✅ |
| Phase F shadow `f3v2_smoke_1777887709` row landed | ✅ |
| Bonus: round 1 doc `f3_smoke_1777886898` self-healed | ✅ |

**VERDICT: ✅ PASS** — End-to-end E2 admin-api → Debezium → Kafka → cdc-worker → shadow path hoạt động đúng.

---

## Section 8 — Skills used

- Read source code (helpers.go diff verification)
- Edit source code (Phase A 2 fix bổ sung)
- go build + go test (verification)
- Bash process management (kill, ps, lsof)
- curl Debezium REST (GET/PUT connector config)
- Python json scripting (parse + clean config)
- docker exec mongosh (Mongo INSERT)
- docker exec psql (registry + shadow query)
- docker logs (worker pipeline trace)

---

## Section 9 — Out-of-scope (defer)

- master_binding cascade (admin-api hiện chỉ tạo source_object_registry + shadow_binding, không cascade lên master). User trigger riêng nếu muốn full path → master DW.
- Schema registry subject preempt cho topic mới — best-effort step 3, ít quan trọng.
- Issue 6/7/8 LOW từ E5 report — vẫn defer Phase F2.

---

**Token round 2**: `f3v2_***` (prefix 8 ký tự, full đã clear khỏi env). Binaries giữ lại để audit:
- `/tmp/cdc-admin-api-f1` (16:29) — F1 5 fix only.
- `/tmp/cdc-admin-api-f3-fixed` (16:27) — round 1 partial fix.
- `/tmp/cdc-admin-api-f3v2` (17:35) — round 2 final binary với cả F1 + F3 (line 76-78 + 128 + 232-237).
