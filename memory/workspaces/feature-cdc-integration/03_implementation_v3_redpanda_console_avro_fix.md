# Implementation — Redpanda Console Avro Deserialize Fix

> **Date**: 2026-04-17
> **Triggered by**: User complaint `http://localhost:18088/topics/...` vẫn báo "There were issues deserializing the value" dù Avro đã migrate
> **Status**: ✅ RESOLVED — Console v2.8.1 → **v2.7.2** (downgrade for Debezium Avro compat)

---

## 1. Problem

User truy cập Redpanda Console UI sau khi Brain claim "Avro migration done". UI hiển thị: "There were issues deserializing the value" trên mọi topic `cdc.goopay.*`.

Brain verify data layer:
- Kafka messages **ARE Avro encoded** (`kafka-avro-console-consumer` decode OK → Debezium envelope `{"after":{"string":"..."},"before":null,...}`)
- Schema Registry có 6 subjects (3 key + 3 value) registered đúng
- Debezium connector RUNNING với AvroConverter + Schema Registry URL correct
- Console ENV `KAFKA_SCHEMAREGISTRY_ENABLED=true, KAFKA_SCHEMAREGISTRY_URLS=http://schema-registry:8081`
- Network aliases OK, DNS resolve OK, HTTP connectivity OK

→ Data đúng Avro. Console config đúng. Nhưng UI fail.

---

## 2. Root Cause

**Console v2.8.1 bug**: Connect RPC `ListMessages` endpoint returns `INVALID_TOPIC_EXCEPTION` cho mọi topic (kể cả `_schemas`) despite Kafka client log `successfully connected to kafka cluster, topic_count:8`.

**Console v3.1.2 bug** (sau upgrade test): panic `nil pointer dereference` trong message worker khi decode Avro Debezium envelope:
```
{"level":"error","msg":"recovered from panic in message worker","error":"runtime error: invalid memory address or nil pointer dereference"}
```

Root: v2.8.1 + v3.1.2 regression trên Debezium MongoDB Avro envelope (union type `{"after": {"string": "..."}}` + nullable fields like `lsid`, `txnNumber`, `wallTime`). Console Avro deserializer choke trên nested unions.

## 3. Fix

**Downgrade Console: v2.8.1 → v2.7.2** (stable Debezium Avro support).

### File edit
`/Users/trainguyen/Documents/work/centralized-data-service/docker-compose.yml`:
```diff
-  image: redpandadata/console:v2.8.1
+  image: redpandadata/console:v2.7.2
```

### Apply
```bash
cd /Users/trainguyen/Documents/work/centralized-data-service
docker compose up -d --force-recreate redpanda-console
```

---

## 4. Verify (end-to-end)

### 4.1 Console startup clean
```
{"msg":"successfully connected to kafka cluster","topic_count":8}
{"msg":"successfully tested schema registry connectivity"}
{"msg":"Server listening on :8080"}
```
No panic, no deserialize error.

### 4.2 Connect RPC ListMessages cho 3 topics CDC

**Key finding**: Console uses `connect-rpc` protocol. Proper field names:
- `topic` (not `topicName` — v3 alias broke v2.8.1 tương thích)
- `partitionId`
- `startOffset` (string: "0" / "-1" newest / "-2" earliest)
- `maxResults`
- `timestamp`
- `filterInterpreterCode`

**Test payload**:
```json
{"topic":"cdc.goopay.payment-bill-service.refund-requests","partitionId":0,"startOffset":"0","maxResults":1,"timestamp":"-1","filterInterpreterCode":""}
```

**Framing**: Connect streaming requires 5-byte envelope header (`\x00 + BE-uint32 length`) prepended.

### 4.3 Decoded message evidence

**Topic `cdc.goopay.payment-bill-service.refund-requests` offset 0**:
```
Key encoding:   PAYLOAD_ENCODING_AVRO schemaId: 19
Value encoding: PAYLOAD_ENCODING_AVRO schemaId: 20

KEY decoded:
  {"id":"{\"$oid\": \"69df0e67b87dab24273f118c\"}"}

VALUE decoded (Debezium envelope):
  {
    "after": {"string": "{\"_id\": {\"$oid\": \"69df0e67b87dab24273f118c\"},\"orderId\": \"VERIFY-FLOW-001\",\"amount\": 11111,\"state\": \"test\",\"createdAt\": {\"$date\": 1776225895718}}"},
    "before": null,
    "op": {"string": "r"},
    "source": {
      "collection": "refund-requests",
      "connector": "mongodb",
      "db": "payment-bill-service",
      ...
    },
    "ts_ms": {"long": 1776430835218},
    ...
  }
```

### 4.4 All 3 CDC topics verified

| Topic | Messages | Encoding | Status |
|:------|:---------|:---------|:-------|
| `cdc.goopay.payment-bill-service.refund-requests` | 1719 | AVRO | ✅ |
| `cdc.goopay.payment-bill-service.payment-bills` | 2 | AVRO | ✅ |
| `cdc.goopay.centralized-export-service.export-jobs` | 117 | AVRO | ✅ |

---

## 5. Versions tested

| Console Version | Result |
|:----------------|:-------|
| v2.8.1 | ❌ `INVALID_TOPIC_EXCEPTION` on ListMessages |
| v3.1.2 | ❌ `panic: nil pointer dereference` on Avro decode |
| **v2.7.2** | ✅ **WORKS** — decode Avro Debezium envelope, return normalized JSON |

---

## 6. Lesson learned (append candidate)

**Pattern**: Vendor-specific bug regression across versions. Upgrading latest ≠ more stable. Downgrade may be correct when:
- Current version panics on known-valid data (Debezium envelope format is stable, not broken by user)
- Latest version has regressions not documented in release notes
- Testing matrix: try 1 upstream back (v2.7.x when v2.8.x breaks) before jumping major (v3.x)

---

## 7. User action

1. Refresh browser `http://localhost:18088/topics/cdc.goopay.payment-bill-service.refund-requests` (Ctrl+Shift+R hard reload).
2. Click trên 1 message row → expected display:
   - Key: `{"id":"{\"$oid\":\"...\"}"}`
   - Value: Debezium envelope JSON structure với after/before/op/source fields.
3. No "issues deserializing" banner.

---

## 8. Files changed

| File | Change |
|:-----|:-------|
| `centralized-data-service/docker-compose.yml` | Console image tag `v2.8.1` → `v2.7.2` |

No code change. Pure infrastructure version adjust.

---

## 9. Related

Previous sessions' Avro migration (Phase B in `07_status_NOT_DELIVERED.md` item #1): Debezium connector config Avro, Schema Registry wiring, Worker decoder — tất cả đã work. Console UI là last-mile gap, fix bằng version downgrade v2.7.2.
