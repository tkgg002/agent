# 01 — Requirements — Phase 01 Split E2E

## Functional requirements

### FR-1: 4 PG containers độc lập
- `gpay-postgres` (giữ nguyên tên + port 5432): chỉ chứa schema `cdc_auth_service` và bảng `auth_users`. KHÔNG có bất kỳ schema CDC nào.
- `gpay-postgres-cdc` (port 5433): chứa schema `cdc_system` + tất cả `shadow_<src>` schemas. Sequence `machine_id_seq` / `fencing_token_seq` đặt ở đây vì worker fencing dùng.
- `gpay-postgres-dest` (port 5434): chứa master physical tables + `dw_<binding>` schemas (theo Phase 39 doc — destination layer cho Transmute output).
- `gpay-postgres-source` (port 5435): chứa sample source tables trong schema `public` (orders, users, payments, wallets, ...) — đóng vai trò "DB nghiệp vụ hạ nguồn cần CDC". Sample data ≥10 rows mỗi bảng.

### FR-2: Service routing 2 DSN
- `cdc-auth-service` → 1 DSN: `gpay-postgres` (cdc_auth_service).
- `cdc-cms-service` → 2 DSN: `gpay-postgres-cdc` (control plane operations) + `gpay-postgres-dest` (master tables management). Hoặc 1 DSN primary (`-cdc`) và lazy-resolve `-dest` khi cần.
- `centralized-data-service/worker` → 2 DSN: `-cdc` (control + shadow) + `-dest` (transmute writes master).
- `centralized-data-service/sinkworker` → 1 DSN: `-cdc` (write shadow_<src>).
- Auth verify (JWT decode) ở cms/worker → KHÔNG cần connect auth DB (chỉ verify signature).

### FR-3: E2E auto-pipeline khi register source mới
Input từ user:
- Source connection: `postgres://srcuser:srcpass@gpay-postgres-source:5432/srcdb`
- Table list: ví dụ `[orders, users]`
- (optional) primary key field, timestamp field

Auto flow:
1. Wizard register `source_object_registry` row trên `-cdc`.
2. CMS gọi `EnsureShadowTable(reg, shadowSchema)` → tạo `shadow_<src>.<table>` trên `-cdc`.
3. Kafka Connect deploy Debezium connector trỏ đến `gpay-postgres-source` → publish topic `cdc.<srcdb>.<table>`.
4. Sinkworker subscribe topic, write rows vào `shadow_<src>.<table>` trên `-cdc`.
5. Transmute scheduler/manual trigger đọc shadow → write master + `dw_<binding>.<table>` trên `-dest`.
6. Verification: row count ở source = ở shadow = ở dw (sau khi pipeline ổn định ≥30s).

### FR-4: Bootstrap + wipe scripts cho 3 vai trò
- `wipe_cdc_runtime_v2.sql` chia thành 3 file: `wipe_auth.sql`, `wipe_cdc.sql`, `wipe_dest.sql` (mỗi file target 1 container).
- Migration runner phải hỗ trợ multi-target.
- Source DB có script `seed_source_local.sql` riêng để load sample data.

## Non-functional requirements

### NFR-1: Idempotency
- Toàn bộ migrations + bootstrap phải chạy lại được mà không lỗi (sau khi rebuild containers).
- E2E test phải pass repeated runs.

### NFR-2: Network
- 4 PG containers cùng network `cdc_default`.
- Service connect bằng container name (Docker DNS) trong compose, bằng `localhost:<port>` từ host machine.

### NFR-3: Performance
- Chấp nhận ≤3s end-to-end latency cho 1 INSERT trên source → xuất hiện ở dw (Debezium poll interval mặc định ~1s).

## Definition of Done

| ID | Criterion | Verify |
|---|---|---|
| D1 | 4 PG containers up + healthy | `docker ps` shows all 4 |
| D2 | Auth login PASS sau split | curl `/api/auth/login` trả JWT |
| D3 | CMS smoke test PASS | curl `/api/v1/system/connectors` trả 200 |
| D4 | Worker + sinkworker up không error | log scan 60s không có ERROR/FATAL |
| D5 | cdc_internal=0 trên cả 3 DB | psql `\dn` ở mỗi container |
| D6 | public empty trên auth + cdc + dest | 0 user-tables ở public của 3 container CDC |
| D7 | Source DB có ≥3 sample tables × ≥10 rows | psql count |
| D8 | Register 1 source object qua API → shadow_<src>.<table> tạo trên `-cdc` | psql `\dt` |
| D9 | INSERT 1 row vào source → trong ≤3s xuất hiện ở `shadow_<src>.<table>` | psql count compare |
| D10 | Trigger transmute → row xuất hiện ở `dw_<binding>.<table>` trên `-dest` | psql count compare |
