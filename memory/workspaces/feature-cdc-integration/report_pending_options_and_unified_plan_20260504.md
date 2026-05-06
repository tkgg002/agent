# Báo cáo: Tổng kết Option treo + Re-scan hệ thống + Plan tổng hợp

**Date**: 2026-05-04 (session resumption sau khi off, có AI khác làm phụ vào 2026-05-01 → 2026-05-02)
**Trigger**: User yêu cầu — "tổng kết option đang treo → plan thiếu sót → quét lại hệ thống → plan tổng hợp"
**Phương pháp**: Exercise-driven verification — query DB thực tế, đọc code thực tế, đọc Kafka Connect REST thực tế. KHÔNG tin báo cáo cũ.
**Verdict**: Code-level B4/B5/B6 ĐÃ FIX (verified in source). Track E vẫn chưa ingest được vì **B3 (include list) + B8 (MariaDB plugin) còn nguyên**. 4 gap mới phát sinh sau session 2026-05-02.

---

## 1. Tóm tắt option đang treo (recap từ các session trước)

### 1.1 Đã đóng trong session 2026-05-02 (verified by code grep)

| Blocker | File sửa | Verified | Note |
|---------|----------|----------|------|
| B4 schema_drift validator | `internal/service/schema_validator.go:118-130, 282-286` | ✅ Permissive-Additive — `// return fmt.Errorf("%w: unknown_field=%s",...)` đã comment, thay bằng `Warn` | Chỉ log warning, không reject |
| B5 DLQ UTF8 0x00 redelivery | `internal/handler/kafka_consumer.go:284, 774-781` | ✅ Base64 encode payload binary trước khi ghi `failed_sync_logs.raw_json` | Field thay đổi sang `{"raw_base64": ..., "encoding": "base64"}` |
| B6 transmute hardcode `_gpay_id` | `internal/service/transmuter.go:82, 203-228, 321-360` | ✅ Dynamic PK detection: query `information_schema.columns` để biết shadow có `_gpay_id` hay không, fallback PK = `id`, cast non-`_gpay_id` PK sang BIGINT cho cursor | Vẫn hardcode `_gpay_id` ở write-side (record map line 357-360, switch line 449/496) — **acceptable** vì đó là master-side convention |

### 1.2 Vẫn còn treo (chưa được động vào trong session 2026-05-02)

| Blocker | Lý do còn treo | Impact |
|---------|----------------|--------|
| **B3 — Debezium connector include list KHÔNG có addtest tables** | Cần infra change (PUT /connectors/.../config) | Track E shadow `*_addtest` = 0 rows mãi mãi |
| **B8 — MariaDB connector plugin chưa cài** | `report_final_resolution_20260502.md` claim "RESOLVED" nhưng `/connector-plugins` runtime KHÔNG list `MySqlConnector` → claim sai | Không có topic `cdc.mariadb.*`, ingest legacy_orders impossible |
| B7 orders_fact PK collision | Không thấy lỗi gần đây trong logs | Có thể đã hết va chạm (orders_fact = 25 rows ổn định), low priority |
| Cleanup test rows source | Phụ thuộc Track E unblock trước | source `public.orders` có 59 rows trong đó có dấu test |

### 1.3 Giả định trong báo cáo cũ NHƯNG sai khi verify

- ❌ `report_final_resolution_20260502.md` line 27: "B8 RESOLVED ... compose updates to auto-install connector" — Container `gpay-kafka-connect` UP 35h (started ~2026-05-02 22:00), nhưng `/connector-plugins` chỉ trả về Postgres + Mongo + Mirror. Plugin **KHÔNG** được install. Có 2 khả năng:
  - File compose đã update nhưng container chưa được `down && up` (binding mount mới chưa load)
  - Hoặc compose update method (`confluent-hub install at init`) fail silently
- ❌ `report_final_resolution_20260502.md` line 35: "Shadow Table Count: shadow_goopay_source.orders now contains 5 records" — VERIFIED ĐÚNG (5 rows). Nhưng các shadow `*_addtest` đều = 0 rows → claim "ingest data from PostgreSQL sources via Debezium" chỉ đúng cho V2 main path (orders), KHÔNG đúng cho Track E addtest.

---

## 2. Re-scan hệ thống hiện tại (real-time verification 2026-05-04)

### 2.1 Containers (14 running, 11 CDC + 3 some-* unrelated)

| Container | Status | Note |
|-----------|--------|------|
| gpay-cdc-worker | Up 34h | Worker live, transmute scheduler chạy mỗi 60s |
| gpay-kafka-connect | Up 35h (healthy) | port 18083→8083 |
| gpay-postgres-cdc/source/dest/postgres | Up 5d (healthy) | 4 PG isolated containers |
| gpay-mariadb | Up 4d (healthy) | port 13307→3306, đã có `legacy_orders_addtest` |
| gpay-mongo | Up 5d (healthy) | port 17017, đã có `payment_bills_addtest` |
| gpay-kafka, gpay-schema-registry, gpay-redis, gpay-nats | Up 5d | Hạ tầng |

### 2.2 Source DB physical objects (đã exist sẵn cho Track E)

| Engine | Database | Object | Note |
|--------|----------|--------|------|
| PG (postgres-source) | goopay_source.public | orders, users, payments, **orders_addtest** | 4 tables, có `orders_addtest` physical |
| MariaDB | goopay_legacy_maria | legacy_orders, **legacy_orders_addtest** | 2 tables |
| MongoDB | payment-bill-service | payment-bills, payment_bills, **payment_bills_addtest** | Có 3 collections |

→ Physical addtest objects ĐÃ TỒN TẠI trên source. Chỉ thiếu Debezium include list.

### 2.3 Debezium connector configs (live verification)

```
cdc-pg-source.table.include.list = "public.orders,public.users,public.payments"
                                    ↑ THIẾU public.orders_addtest

goopay-mongodb-cdc.collection.include.list = "...payment-bills,refund-requests,..."
                                              ↑ THIẾU payment_bills_addtest

cdc-mariadb-* = KHÔNG TỒN TẠI (B8)
```

### 2.4 cdc_system control-plane state

| Bảng | Số rows | Note |
|------|---------|------|
| source_object_registry | 22 | 11 V1 Airbyte (1-10) + 12 V2 Debezium |
| source_object_registry status="failed" | 6 | id=19/23/24/25/27/28 (e2e_d/legacy_orders/payment_bills) — stale |
| source_object_registry status="running" | 6 | id=11 (orders) + 26 (e2e_v5) + 29/30/31 (3 addtest) + 18 (curl_test archived) |
| master_binding | 9 | tất cả approved+is_active=t |
| transmute_schedule | 6 | tất cả last_status=success (cron tick mới nhất 2026-05-04 02:31 UTC) |
| failed_sync_logs (DLQ) | 0 | trống |
| cdc_activity_log (2h gần) | 94 | hoạt động đều |

### 2.5 Shadow + Master tables

```
shadow_goopay_source.orders                          5 rows  ✅
shadow_src_local_pg_source.orders                    0 rows
shadow_src_local_pg_source.orders_e2e_d_v5           0 rows
shadow_src_local_pg_source.orders_addtest            0 rows  ❌ (B3)
shadow_mariadb_legacy_default.legacy_orders          0 rows  ❌ (B8)
shadow_mariadb_legacy_default.legacy_orders_addtest  0 rows  ❌ (B8)
shadow_mongo_payment_bill_default.payment_bills      0 rows
shadow_mongo_payment_bill_default.payment_bills_addtest 0 rows ❌ (B3)
+ orders_e2e_d_v2/v3/v4 (3 stale empty tables)

dw_orders.orders_fact (postgres-dest)               25 rows  ✅
dw_src_local_pg_source.orders_addtest               EXISTS, 0 rows
dw_mariadb_legacy_default.legacy_orders_addtest     EXISTS, 0 rows
dw_mongo_payment_bill_default.payment_bills_addtest EXISTS, 0 rows
```

→ Master DDL đã được tạo cho cả 3 addtest (provisioning cascade chạy tới `master_active`). Chỉ chờ data từ shadow.

### 2.6 Worker live behavior (logs last 30m)

```
[every 60s] scheduler tick dispatched count=6
[every 60s] transmute complete master=orders_fact scanned=5 inserted=5 ...
[every 60s] transmute complete master=orders_addtest scanned=0 ...
[every 60s] transmute complete master=legacy_orders_addtest scanned=0 ...
[every 60s] transmute complete master=payment_bills_addtest scanned=0 ...
[every 5s]  Post "http://localhost:4318/v1/logs": dial tcp [::1]:4318: connect: connection refused
[every 5s]  failed to upload metrics: ... 4318 connection refused
[every 30s] reconcile schedule tick SKIPPED — reconCore is nil
            "MongoDB not configured or Mongo connection failed at startup"
```

→ 2 gap observability mới phát hiện (G3, G4 dưới).

---

## 3. Gap mới phát sinh (sau session 2026-05-02)

| Gap | Mức độ | Mô tả | File evidence |
|-----|--------|-------|---------------|
| **G1** | Med | 6 sources `provisioning_state=failed` (id 19/23/24/25/27/28) lưu lại stale state machine, không retry/archive | `source_object_registry` |
| **G2** | Low | 10 V1 Airbyte legacy seeds (id 1-10) `is_active=false` nhưng vẫn chiếm slot trong registry — đã có script `deployments/sql/cdc/prune_legacy_v1_bindings.sql` chưa chạy | Migration 035 |
| **G3** | High | OTel collector không deployed → worker spam log `dial tcp [::1]:4318: connect: connection refused` mỗi 5s. Toàn bộ tier observability (logs + metrics) bị rớt | `config/config-local.yml:79 endpoint: http://localhost:4318` |
| **G4** | Med | ReconCore=nil → reconcile scheduler tick SKIPPED mỗi 30s. Recon T1/T2/T3 không chạy → không phát hiện được drift giữa source vs master | Worker startup log "MongoDB connection failed" |
| **G5** | Low | 3 stale shadow tables `orders_e2e_d_v2/v3/v4` tồn tại nhưng source object 23/24/25 đã `failed` — orphan storage | Schema list |

### 4 lessons applied during this verification

- **L-2026-04-29 cascade-liability**: state-machine có thể cascade tới `running` với pipeline rỗng — đúng case G1 (6 sources failed nhưng vẫn ngồi trong registry)
- **L-2026-04-29 three-layer-trust-failure**: B8 là 1 ví dụ — file compose update KHÔNG đảm bảo container đã reload plugin. Cần verify ở runtime (`/connector-plugins` REST), không tin compose
- **L-fire-and-forget DDL generator**: master tables addtest đã được tạo (DDL apply succeeded) ngay cả khi data chưa flow vào — same pattern, additive idempotent re-runs
- **L-schema-from-real-DB**: query `\d` trực tiếp confirm `failed_sync_logs` không có column `topic` mà là `kafka_topic`, không có `offset` mà là `kafka_offset` — không tin schema từ memory

---

## 4. Plan tổng hợp (ưu tiên + sequencing)

> **Brain prohibition §12**: Plan dưới chỉ là **đề xuất** — Brain KHÔNG sửa code. Cần Muscle thực thi sau khi user approve.

### 4.1 Phase A — Unblock Track E ingest (mở data-plane)

#### A1 [INFRA] Update PG connector include list
**Action**: PUT `localhost:8083/connectors/cdc-pg-source/config` thêm `public.orders_addtest` vào `table.include.list`.
**Verify**: `docker exec gpay-kafka-connect curl -s localhost:8083/connectors/cdc-pg-source/config` → contains `orders_addtest`.

#### A2 [INFRA] Update Mongo connector include list
**Action**: PUT thêm `payment-bill-service.payment_bills_addtest` vào `collection.include.list`.
**Verify**: `/connectors/goopay-mongodb-cdc/config` lists collection.

#### A3 [INFRA] Re-install MariaDB connector plugin (fix B8 thực sự)
**Action**:
```bash
# Verify root cause first
docker exec gpay-kafka-connect ls /usr/share/confluent-hub-components/ | grep -i mysql
# Nếu trống → install:
docker exec gpay-kafka-connect bash -c \
  "confluent-hub install --no-prompt debezium/debezium-connector-mysql:latest"
docker restart gpay-kafka-connect
# Verify:
curl -s localhost:18083/connector-plugins | python3 -c "import sys,json; print('\n'.join([p['class'] for p in json.load(sys.stdin)]))" | grep -i mysql
```
**Verify**: `/connector-plugins` lists `MySqlConnector` (Debezium dùng MySQL plugin cho MariaDB).

#### A4 [INFRA] Tạo MariaDB connector
**Action**: POST `cdc-mariadb-source` config:
```json
{
  "connector.class": "io.debezium.connector.mysql.MySqlConnector",
  "database.hostname": "mariadb",
  "database.user": "<user>",
  "database.password": "<pass>",
  "database.server.id": "184054",
  "topic.prefix": "cdc.mariadb",
  "database.include.list": "goopay_legacy_maria",
  "table.include.list": "goopay_legacy_maria.legacy_orders,goopay_legacy_maria.legacy_orders_addtest",
  "schema.history.internal.kafka.bootstrap.servers": "gpay-kafka:9092",
  "schema.history.internal.kafka.topic": "schema-history.mariadb",
  "value.converter": "io.confluent.connect.avro.AvroConverter",
  "value.converter.schema.registry.url": "http://gpay-schema-registry:8081",
  "key.converter": "io.confluent.connect.avro.AvroConverter",
  "key.converter.schema.registry.url": "http://gpay-schema-registry:8081"
}
```
**Verify**: `/connectors/cdc-mariadb-source/status` → state=RUNNING, task[0].state=RUNNING.

#### A5 [VERIFY] End-to-end smoke test sau A1–A4
- INSERT test row vào `public.orders_addtest` source → wait 30s → query `shadow_src_local_pg_source.orders_addtest` count > 0
- INSERT MariaDB legacy_orders → wait 30s → shadow_mariadb_legacy_default.legacy_orders count > 0
- Insert Mongo `payment_bills_addtest` doc → shadow count > 0
- Wait 60s (cron tick) → transmute_schedule.last_status=success VÀ master DW table có data

**Definition of Done Phase A**: 3 shadow `*_addtest` tables có >0 rows VÀ 3 master DW `*_addtest` tables có >0 rows.

---

### 4.2 Phase B — Restore observability (G3, G4)

#### B1 [INFRA] Deploy OTel collector hoặc disable Otel exporter
**Option B1.a (deploy)**: Add `otel-collector` container vào docker-compose, expose 4318 OTLP HTTP, configure exporters (logs → file/loki, metrics → Prometheus).
**Option B1.b (disable)**: Set `otel.enabled: false` trong `config-local.yml` để worker không spam connection refused.
**Recommendation**: B1.a (deploy) — observability cần có để phát hiện drift sớm.

#### B2 [CONFIG] Enable Mongo recon hoặc disable cleanly
**Action**: Verify Mongo URL config cho worker. Nếu Mongo recon không cần thiết → set `recon.mongo.enabled=false` thay vì để code raise nil-pointer skip mỗi 30s.

---

### 4.3 Phase C — Cleanup stale state (G1, G2, G5)

#### C1 [SQL] Prune V1 legacy seeds (G2)
**Action**: chạy script đã có sẵn:
```bash
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  < deployments/sql/cdc/prune_legacy_v1_bindings.sql
```
**Verify**: count(*) source_object_registry WHERE object_code LIKE 'legacy_%' AND is_active=true → 0.

#### C2 [SQL] Investigate failed sources (G1)
**Action**: query `last_step_error` của 6 sources `failed` (id 19/23/24/25/27/28). Cho mỗi source quyết định:
- Retry: nếu lỗi do transient (network, deadlock) → re-trigger via NATS `cdc.cmd.provisioning.retry`
- Archive: nếu lỗi do permanent (source dropped, schema invalid) → set provisioning_state='archived'

#### C3 [SQL] Drop orphan shadow tables (G5)
**Action**: nếu C2 quyết định archive id=23/24/25 → drop:
```sql
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v2 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v3 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v4 CASCADE;
```
(KHÔNG drop v5 vì id=26 đang `running`.)

#### C4 [SQL] Cleanup test rows source
**Action**: sau khi Phase A unblock + smoke test xong:
```sql
DELETE FROM public.orders WHERE notes LIKE 'track-e-test-%' OR notes LIKE 'p2-p3-p4-smoke-%';
```

---

### 4.4 Phase D — Long-term Schema Schism resolution (architect rule needed)

#### D1 [ARCH] Quyết định convention chuẩn
**Question cho architect**:
- Shadow tier dùng PK `id` (TEXT, V1 convention) hay `_gpay_id` (BIGINT, V2 convention)?
- Hay cho phép coexistence — có flag/column `pk_type` để mỗi shadow tự khai báo?

#### D2 [MIGRATION] Theo decision D1
- Nếu unify về V2: ALTER 11 shadow tables thêm `_gpay_id BIGINT GENERATED ALWAYS AS IDENTITY`, regenerate master DDL, transmuter detect mode (đã làm rồi B6).
- Nếu coexistence: document convention trong `agent/memory/global/conventions.md`, transmuter detect logic là final.

---

### 4.5 Verification end-to-end (sau khi Phase A+B+C complete)

```bash
# 1. Smoke test ingest end-to-end
docker exec -i gpay-postgres-source psql -U src_user -d goopay_source <<SQL
INSERT INTO public.orders_addtest (id, ...) VALUES ('test-1', ...);
INSERT INTO public.orders (user_id, amount, status, notes)
  SELECT 9000+i, 100+i, 'pending', 'phase-a-smoke-'||i FROM generate_series(1,3) i;
SQL

# 2. Wait Debezium snapshot 30s
sleep 30

# 3. Verify Worker ingest
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM shadow_src_local_pg_source.orders_addtest;
   SELECT count(*) FROM shadow_src_local_pg_source.orders;"
# Expect: orders_addtest >= 1, orders >= 3

# 4. Wait cron tick 60s
sleep 60

# 5. Verify Master
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
  "SELECT count(*) FROM dw_src_local_pg_source.orders_addtest;
   SELECT count(*) FROM dw_orders.orders_fact;"
# Expect: orders_addtest >= 1, orders_fact >= 28

# 6. Verify DLQ vẫn trống (no schema_drift)
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM cdc_system.failed_sync_logs;"
# Expect: 0

# 7. Verify OTel logs sạch (sau Phase B)
docker logs gpay-cdc-worker --tail 100 | grep -c 'connection refused'
# Expect: 0

# 8. Verify Mongo recon (sau Phase B)
docker logs gpay-cdc-worker --since 5m | grep -c 'reconCore is nil'
# Expect: 0
```

---

## 5. Files thay đổi trong session này

| Path | Action |
|------|--------|
| `agent/memory/workspaces/feature-cdc-integration/report_pending_options_and_unified_plan_20260504.md` | NEW (this file) |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND — entry cho session 2026-05-04 |

**Không có code change** (Brain prohibition §12 honored).

---

## 6. Skills used (CLAUDE.md §0)

- `Bash` — psql/docker exec/curl Kafka Connect REST/grep code
- `Read` — đọc 4 report files delta + lessons.md (recent 200 lines) + active_plans.md
- `Write` — sinh report file vật lý
- `TaskCreate` / `TaskUpdate` — track 5 tasks (verify → map → rescan → plan → write)
- Governance: CLAUDE.md §0 (vietnamese), §3 (Plan & Verify), §7 (memory retention — đọc lessons trước), §11 (APPEND-only progress), §12 (Brain prohibition — chỉ đề xuất plan), §14 (pre-flight check)
- Lessons applied:
  - L-2026-04-29-cascade-liability — gate quality không chỉ "step ran"
  - L-2026-04-29-three-layer-trust-failure — verify mỗi layer riêng (B8 compose update ≠ runtime plugin loaded)
  - L-schema-from-real-DB — `\d` trước khi viết SQL
  - L-fire-and-forget-DDL-generator — additive idempotent

---

## 7. Câu hỏi cần user/architect quyết

1. **B3 logical-clone**: Cập nhật include list (option B) thay vì giữ logical-clone fan-out (option A) — đồng ý chứ?
2. **G3 OTel**: Deploy collector hay disable? (xài monitor chính cho production hay tạm tắt?)
3. **G4 Mongo recon**: Có dùng recon T1/T2/T3 cho Mongo không? Hay chỉ PG/MariaDB?
4. **G1 sources failed**: 6 sources retry hết hay archive hết? Có cần debug từng cái?
5. **D1 Schema Schism**: V1 vs V2 unify hay coexist?

Nếu user approve thì em delegate cho Muscle thực thi Phase A → B → C theo thứ tự (mỗi phase là 1 PR riêng, blast radius rõ ràng).
