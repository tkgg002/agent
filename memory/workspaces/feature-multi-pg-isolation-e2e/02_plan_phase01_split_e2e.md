# 02 — Plan — Phase 01 Split E2E

## Tóm tắt
Tách 1 PG container hiện tại thành 4 (auth / cdc / dest / source) + nâng services lên multi-DSN routing + chạy E2E auto-pipeline test.

## Kiến trúc target (sau khi xong)

```
                           ┌──────────────────────┐
                           │  gpay-postgres-source│  port 5435
                           │  (sample source data)│  schema: public
                           └──────────┬───────────┘
                                      │ logical decoding (Debezium)
                                      ▼
                           ┌──────────────────────┐
                           │   gpay-kafka-connect │
                           │   topic cdc.<src>.*  │
                           └──────────┬───────────┘
                                      │ kafka consume
                                      ▼
   ┌──────────────────────┐    ┌──────────────────────┐    ┌──────────────────────┐
   │  cdc-auth-service    │    │  centralized-data    │    │  cdc-cms-service     │
   │  - auth_users only   │    │  /sinkworker (V2)    │    │  - control ops       │
   └──────────┬───────────┘    │  /worker             │    │  - master DDL        │
              │                └────────┬─────────────┘    └──────┬───────┬───────┘
              ▼                         │                         │       │
   ┌──────────────────────┐    ┌──────────────────────┐    ┌──────▼───┐ ┌─▼────────┐
   │   gpay-postgres      │    │   gpay-postgres-cdc  │    │  ...     │ │ ...      │
   │  port 5432           │    │  port 5433           │◀───┘          │ │          │
   │  schema:             │    │  schema:             │               │ │          │
   │   cdc_auth_service   │    │   cdc_system         │   ┌───────────▼─▼────────┐
   └──────────────────────┘    │   shadow_<src>       │   │  gpay-postgres-dest  │
                                └──────────────────────┘   │  port 5434           │
                                          │ transmute      │  schema:             │
                                          └───────────────▶│   public (master)    │
                                                           │   dw_<binding>       │
                                                           └──────────────────────┘
```

## Phase breakdown

### Phase A — Infrastructure (Docker Compose)
**A1**. Add 3 PG services vào `docker-compose.yml`:
   - `postgres-cdc` (image postgres:15-alpine, port 5433, vol pg_cdc_data, env user/password/db=cdc_dw)
   - `postgres-dest` (port 5434, vol pg_dest_data, db=goopay_dest)
   - `postgres-source` (port 5435, vol pg_source_data, db=goopay_source, configured `wal_level=logical` + `max_wal_senders=10` + `max_replication_slots=10` cho Debezium)
**A2**. Healthcheck cho cả 4 PG containers.
**A3**. Update env `DB_SINK_URL` trong worker service.

### Phase B — DB schema split
**B1**. Tách `wipe_cdc_runtime_v2.sql` → `wipe_auth.sql` + `wipe_cdc.sql` + `wipe_dest.sql`.
**B2**. Tách migrations:
   - `cdc-auth-service/migrations/` (đã có `001_auth_users.sql`) → giữ nguyên, chỉ thay target host.
   - `centralized-data-service/migrations/` (44 migrations) → split thành 2 set:
     - `migrations/cdc/` cho `-cdc` host: 001-039 + 040 + 041 + 042 + 043 + 044 + 028 (tất cả CDC system + sonyflake foundation + shadow helpers)
     - `migrations/dest/` cho `-dest` host: chỉ tạo `master_<binding>` parent partitioned tables + `dw_<binding>` schemas (mới — ít migrations)
   - HOẶC giữ nguyên 1 set + chia bằng `--target=cdc|dest` flag trong Makefile.
   - **Khuyến nghị**: chia thư mục `migrations/cdc/` + `migrations/dest/` (rõ ràng hơn).
**B3**. Bootstrap V2 seed scripts split:
   - `bootstrap_cdc_local.sql` (-cdc): 6 connections (1 trỏ source mới), 0 source_objects (chờ Wizard register), 0 bindings.
   - `bootstrap_dest_local.sql` (-dest): chỉ tạo schema `dw_default` empty, không seed rows.
**B4**. Source DB seed:
   - `cdc-source-test/sql/init_source_local.sql` — tạo schema public + 3 bảng (orders, users, payments) + 10 rows mỗi bảng + đảm bảo có Replication Identity FULL hoặc PRIMARY KEY.

### Phase C — Service config (multi-DSN)
**C1**. `cdc-auth-service/config-local.yml`: host = gpay-postgres (không đổi tên), port 5432, db `auth_dw` (rename) hoặc giữ `goopay_dw` nếu user muốn keep tên cũ. Chỉ chứa 1 schema `cdc_auth_service`.
**C2**. `cdc-cms-service/config-local.yml`: thêm 2 block:
   ```yaml
   database:
     control_plane:    # cdc_system, shadow
       host: localhost
       port: 5433
       database: cdc_dw
     destination:      # master, dw_<binding>
       host: localhost
       port: 5434
       database: goopay_dest
   ```
**C3**. `centralized-data-service/config-local.yml`: tương tự, 2 block control_plane + destination.
**C4**. Update `pkgs/database` của 3 service để hỗ trợ named connections (`db.Get("control_plane")`, `db.Get("destination")`).
**C5**. Audit grep code để đảm bảo các raw SQL có schema-qualify đúng.

### Phase D — E2E auto-pipeline
**D1**. Update `connection_registry` rows: 1 row trỏ `gpay-postgres-source:5432` (engine=postgres, debezium-compatible).
**D2**. Tạo Kafka Connect connector deploy script: `deployments/connect/register_pg_source.sh` POST tới Kafka Connect REST `/connectors` với config `connector.class=io.debezium.connector.postgresql.PostgresConnector`, `database.hostname=postgres-source`, etc.
**D3**. Sinkworker subject pattern: hiện đang `^cdc\.goopay\..*` → cần verify với topic mới `cdc.goopay_source.public.orders` (hoặc tùy theo Debezium topic.prefix config).
**D4**. CMS Wizard endpoint nhận POST → tạo source_object_registry row + shadow_binding + gọi EnsureShadowTable + (nếu chưa có connector) trigger Connect REST register.

### Phase E — Verification + DoD
**E1**. Auto-test bash script: `scripts/e2e_test_split.sh` chạy:
   - Verify 4 PG containers up.
   - Login auth → JWT.
   - Wizard register source `orders` table.
   - Wait Debezium connector READY (poll Connect REST `/connectors/.../status`).
   - INSERT 1 row vào `gpay-postgres-source.public.orders`.
   - Wait ≤3s → verify `shadow_goopay_source.orders` có row mới trên `-cdc`.
   - Trigger transmute → verify `dw_<binding>.orders` có row trên `-dest`.
   - Print 10-point PASS report.
**E2**. Document verify pack vào `06_validation_phase01_split_e2e.md`.

## Risk register
- **R1**: Sequence cross-DB không share — `machine_id_seq`, `fencing_token_seq` chỉ ở `-cdc`. Worker khi connect cả 2 DSN phải nhớ fencing context lấy từ `-cdc`. Mitigation: worker dùng connection `control_plane` cho fencing INIT, connection `destination` chỉ cho writes.
- **R2**: Debezium connector cần `wal_level=logical` trên source PG. Mitigation: set ngay từ docker-compose env (`POSTGRES_INITDB_ARGS` hoặc `command: postgres -c wal_level=logical`).
- **R3**: Topic naming Debezium → sinkworker subscribe pattern phải khớp. Mitigation: dùng env `KAFKA_TOPIC_PATTERN` đồng nhất giữa Connect config + sinkworker.
- **R4**: Migration 044 + 010 partition logic — cần tái áp dụng cho `-cdc` mà không bị orphan. Mitigation: viết lại migration 010 để CREATE TABLE schema-qualified `cdc_system.<partition>`.
- **R5**: Volume migration data cũ — cần wipe `pg_data` volume gốc HOẶC giữ làm auth-only. Mitigation: rename volume `pg_data` → `pg_auth_data` để rõ vai trò.

## Estimated effort
- Phase A: 1h (compose + healthchecks)
- Phase B: 3h (split migrations + bootstrap + source seed)
- Phase C: 2h (multi-DSN code path)
- Phase D: 2h (Debezium register + topic mapping + Wizard hook)
- Phase E: 1h (test script + docs)
- **Tổng**: ~9h muscle work.

## Order of execution
A → B → C → D → E. KHÔNG được skip. Mỗi phase end PHẢI verify riêng trước khi sang phase kế.
