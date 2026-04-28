# Phase 38 — Validation

## Status: ✅ Operator-flow PASS · ⚠️ Auto-flow GREEN-infra (data starved)

### Operator-flow (11 endpoints, JWT bearer admin/admin123)

| HTTP | Endpoint | Body (head 200B) |
|---:|---|---|
| 200 | `/api/schema-changes/pending?status=pending&page_size=1` | `{"data":[],"page":1,"total":0}` |
| 200 | `/api/v1/source-objects/stats` | `{"total":1,"by_source_db":{"goopay_payment":1},"by_sync_engine":{"debezium":1},"by_priority":{"critical":1},"tables_created":0}` |
| 200 | `/api/v1/source-objects?page=1&page_size=100` | data 1 row id=12 source_db=goopay_payment |
| 200 | `/api/v1/shadow-bindings?page=1&page_size=100` | data 1 row binding_code=sb_local_goopay_payment_payments |
| 200 | `/api/v1/schema-proposals?status=pending` | `{"count":0,"data":null}` |
| 200 | `/api/v1/schedules` | `{"count":1,"data":[{"id":1,"master_table":"payment_fact","mode":"post_ingest",...}]}` |
| 200 | `/api/activity-log?page=1&page_size=30` | data có id=16 op=reconcile … |
| 200 | `/api/activity-log/stats` | stats_24h gồm cmd-batch-transform/reconcile |
| 200 | `/api/failed-sync-logs?page_size=50` | `{"data":null,"page":1,"total":0}` |
| 200 | `/api/worker-schedule` | data id=5 op=airbyte-sync next_run_at=… |
| 200 | `/api/v1/source-objects?page_size=500` | data 1 row (cùng response như trên) |

### Auto-flow

- Debezium connector `goopay-mongodb-cdc`: `RUNNING` / tasks: `['RUNNING']`.
- Kafka topics khớp prefix `cdc.goopay.*`:
  - `cdc.goopay.centralized-export-service.export-jobs`
  - `cdc.goopay.payment-bill-service.debezium_signal`
  - `cdc.goopay.payment-bill-service.payment-bills`
  - `cdc.goopay.payment-bill-service.refund-requests`
- Worker (`/tmp/cdc-worker.log`): vòng lặp `discoverTopics` đều đặn 60s,
  `topics:[]` (vì registry filter), không panic.

### Đánh giá auto-flow

Mismatch dữ liệu giữa Debezium config và registry:

| Side | Config |
|---|---|
| Debezium `database.include.list` | `payment-bill-service, centralized-export-service` |
| Debezium `collection.include.list` | `payment-bills, refund-requests, payment-bill-histories, payment-bill-codes, payment-bill-events, payment-bill-holdings, identitycounters, refund-requests-histories, export-jobs` |
| Worker `GetDebeziumTables()` | `[payments]` (1 row, id=12 trong `cdc_system.source_object_registry`) |

Worker filter so khớp `parts[3]` (collection segment trong topic) với
`source_object_name` của registry. Hiện không entry nào trong registry trùng
với 9 collection được Debezium publish → 0 topic được consume. Đây là
**data linkage gap**, không phải bug code.

### Definition of Done check

- [x] 11/11 operator endpoints = 200.
- [x] Build pass cả 2 service Go.
- [x] Migration 039 áp dụng (`SHOW search_path` confirm `cdc_system, public`).
- [x] Phase 38 docs đầy đủ prefix.
- [x] Lesson tổng quát hóa Global Pattern.
- [ ] Auto-flow ingest end-to-end: GAP — cần một phase riêng để align registry
      với Debezium include list (xem `01_requirements_phase38_*`, mục Out of scope).

### Re-verify command pack

```bash
# 1) DB schemas
docker exec gpay-postgres psql -U user -d goopay_dw -c "SHOW search_path;"

# 2) Build
cd cdc-system/cdc-cms-service && go build ./...
cd cdc-system/centralized-data-service && go build ./...

# 3) Login + 11 endpoints (xem 03_implementation_phase38 để pull script)
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")

# 4) Auto-flow
curl -s http://localhost:18083/connectors/goopay-mongodb-cdc/status
docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --list \
  | grep '^cdc\.goopay\.'
tail -20 /tmp/cdc-worker.log
```
