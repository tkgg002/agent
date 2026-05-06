# Báo cáo verify V2 bridge end-to-end sau cron tick

**Date**: 2026-04-30 (loop dynamic mode)
**Trigger**: `/loop verify V2 bridge end-to-end after cron tick`
**Sources kiểm tra**: 29 (orders_addtest), 30 (legacy_orders_addtest), 31 (payment_bills_addtest)
**Kết luận**: ✅ **Control-plane thông** | ❌ **Data-plane bị 3 blocker chặn (B4/B5/B6)**

---

## 1. Trạng thái thực tế (kiểm chứng bằng query)

### 1.1 Provisioning state-machine — ĐÃ CASCADE XONG

```
SELECT id, source_object_name, provisioning_state, profile_status, last_step_error
FROM cdc_system.source_object_registry WHERE id IN (29,30,31);
```

| id | source_object_name    | provisioning_state | profile_status | last_step_error |
|----|-----------------------|--------------------|----------------|-----------------|
| 29 | orders_addtest        | running            | active         | (null)          |
| 30 | legacy_orders_addtest | running            | active         | (null)          |
| 31 | payment_bills_addtest | running            | active         | (null)          |

→ Cascade `draft → shadow_pending → shadow_active → master_pending → master_active →
mapping_pending → mapping_ready → schedule_pending → running` đã hoàn tất, không có
bước nào lỗi.

### 1.2 shadow_binding — DDL CREATED

| id | source_object_id | shadow_schema                       | shadow_table          | ddl_status |
|----|------------------|-------------------------------------|-----------------------|------------|
| 38 | 29               | shadow_src_local_pg_source          | orders_addtest        | created    |
| 39 | 30               | shadow_mariadb_legacy_default       | legacy_orders_addtest | created    |
| 40 | 31               | shadow_mongo_payment_bill_default   | payment_bills_addtest | created    |

→ 3 shadow tables vật lý đã tồn tại, đúng convention `shadow_<connection_code>`.

### 1.3 master_binding — APPROVED + ACTIVE

| id | source_object_id | master_schema                     | master_table          | schema_status | is_active |
|----|------------------|-----------------------------------|-----------------------|---------------|-----------|
| 28 | 29               | dw_src_local_pg_source            | orders_addtest        | approved      | t         |
| 29 | 30               | dw_mariadb_legacy_default         | legacy_orders_addtest | approved      | t         |
| 30 | 31               | dw_mongo_payment_bill_default     | payment_bills_addtest | approved      | t         |

→ Constraint `v2_master_active_requires_approved` thoả mãn — V2 bridge approve+activate
đúng pipeline.

### 1.4 mapping_rule_v2 — V2 BRIDGE DUAL-WRITE THÀNH CÔNG ✅

```sql
SELECT source_object_id, COUNT(*) FROM cdc_system.mapping_rule_v2
 WHERE source_object_id IN (29,30,31) GROUP BY source_object_id;
```

| source_object_id | rule_count |
|------------------|------------|
| 29               | 7          |
| 30               | 7          |
| 31               | 4          |

→ V2 bridge (V1 cdc_mapping_rules → V2 mapping_rule_v2 dual-write tại discover handler)
populate đầy đủ rules. Đây là **bằng chứng V2 bridge end-to-end ở control-plane đang
hoạt động đúng thiết kế**.

### 1.5 transmute_schedule — CRON ĐÃ FIRE NHƯNG TRANSMUTE LỖI ❌

| id | master_binding_id | last_status | last_error                                                                | last_run_at                  |
|----|-------------------|-------------|---------------------------------------------------------------------------|------------------------------|
| 13 | 28                | failed      | fetch shadow batch: ERROR: column "_gpay_id" does not exist (SQLSTATE 42703) | 2026-05-01 15:40:45.588537+00 |
| 14 | 29                | failed      | fetch shadow batch: ERROR: column "_gpay_id" does not exist (SQLSTATE 42703) | 2026-05-01 15:40:45.588537+00 |
| 15 | 30                | failed      | fetch shadow batch: ERROR: column "_gpay_id" does not exist (SQLSTATE 42703) | 2026-05-01 15:40:45.588537+00 |

→ Cron tick chạy đều, nhưng transmuter scanner query column `_gpay_id` không tồn tại
trong shadow → **Blocker B6**.

### 1.6 Shadow row counts — 0 ROWS

```
shadow_src_local_pg_source.orders_addtest          → 0
shadow_mariadb_legacy_default.legacy_orders_addtest → 0
shadow_mongo_payment_bill_default.payment_bills_addtest → 0
```

Source `goopay_source.public.orders` có 58 rows. Worker không ingest được vào shadow →
xác nhận **Blocker B4 (schema_drift) + B5 (DLQ binary loop)** vẫn chặn ingest.

### 1.7 Master DW — TABLE CHƯA TỒN TẠI

```
ERROR: relation "dw_src_local_pg_source.orders_addtest" does not exist
```

→ Transmuter chưa CREATE master table vì fail ngay ở `fetch shadow batch` step trước
DDL.

### 1.8 Schema thực tế của shadow — XÁC NHẬN ROOT CAUSE B6

```
\d shadow_src_local_pg_source.orders_addtest
```

| Column      | Type                  | Note                |
|-------------|-----------------------|---------------------|
| id          | text                  | **PK = id**, không phải `_gpay_id` |
| _raw_data   | jsonb                 | CDC meta cols       |
| _source     | varchar(20)           |                     |
| _synced_at  | timestamp             |                     |
| _version    | bigint                |                     |
| _hash       | varchar(64)           |                     |
| _deleted    | boolean               |                     |
| _created_at | timestamp             |                     |
| _updated_at | timestamp             |                     |
| user_id     | bigint                | business col        |
| amount      | numeric               | business col        |
| status      | text                  | business col        |
| notes       | text                  | business col        |
| created_at  | timestamptz           | business col        |
| updated_at  | timestamptz           | business col        |

Indexes: `orders_addtest_id_cdc_unique` UNIQUE (id)

→ Convention shadow tier mới: PK = `id text` + 8 V1 CDC meta cols + business cols.
Transmuter hardcode `_gpay_id` (master-table convention) → mismatch.

---

## 2. Ma trận end-to-end V2 bridge

```
Source DB ──INSERT──► Debezium ──Kafka──► Worker ──Shadow──► Transmute ──► Master DW
   ✅            ✅           ✅          ❌          ❌            ❌            ❌
                                       (B4/B5)              (B6)
```

| Tầng                                    | Status | Bằng chứng                                                       |
|-----------------------------------------|--------|------------------------------------------------------------------|
| Source DB rows                          | ✅      | 58 orders                                                         |
| Debezium connectors                     | ✅      | `cdc-pg-source` + `goopay-mongodb-cdc` running                   |
| Kafka topics                            | ✅      | `cdc.gpay.public.orders` accepting writes                        |
| **Provisioning V2 cascade**             | ✅      | state=running, 4-step transitions clean                          |
| **V2 bridge dual-write**                | ✅      | mapping_rule_v2: 7/7/4 rows                                      |
| **master_binding approve+activate**     | ✅      | schema_status=approved, is_active=t                              |
| **transmute_schedule cron firing**      | ✅      | last_run_at moved to 2026-05-01 15:40:45                         |
| Worker → Shadow ingest                  | ❌      | 0 shadow rows; B4 schema_drift, B5 DLQ UTF8 0x00 redelivery loop |
| Transmute fetch_shadow batch            | ❌      | B6 — `_gpay_id` column không tồn tại                              |
| Master DW write                         | ❌      | Table chưa được tạo do transmute fail trước DDL                   |

**Verdict**:
- **Control-plane (V2 bridge)**: ✅ FULLY VERIFIED — 100% pass.
- **Data-plane**: ❌ BLOCKED — 3 bug code-level (B4/B5/B6) cần fix mới chạy được.

---

## 3. Khác biệt so với gap analysis 10_gap_analysis_track_e.md

File gap đã liệt kê 8 blocker (B1–B8). Verify hôm nay xác nhận:

| Blocker | Status sau verify  |
|---------|--------------------|
| B1 (profile_status='draft' gate) | ✅ ĐÃ FIX (state=running, profile=active) |
| B2 (shadow_binding ddl_status='pending') | ✅ ĐÃ FIX (ddl_status=created) |
| B3 (logical-clone locator) | ⚠️ Chưa verify ở loop này — cần architect rule |
| B4 (schema_drift validator) | ❌ VẪN CHẶN — shadow 0 rows |
| B5 (DLQ UTF8 0x00 loop) | ❌ VẪN CHẶN — Avro bytes vào TEXT column |
| B6 (transmute hardcode `_gpay_id`) | ❌ VẪN CHẶN — confirmed by `\d` shadow table |
| B7 (orders_fact PK collision) | ⚠️ Không liên quan addtest, không verify ở loop này |
| B8 (MariaDB connector chưa cài) | ❌ VẪN CHẶN — chỉ có PG + Mongo connector |

**Tiến triển từ 2026-04-29 → 2026-04-30**: B1+B2 đã được rolled-forward (an toàn).
B4/B5/B6/B7/B8 chưa được động tới.

---

## 4. Khuyến nghị (chỉ liệt kê, KHÔNG implement — Brain prohibition §12)

### 4.1 Ưu tiên cao (unblock data-plane)

1. **B6 — transmute scanner**: Sửa query `fetch shadow batch` đọc PK column từ
   `shadow_binding.shadow_pk_column` (live-resolved) thay vì hardcode `_gpay_id`.
   Blast radius: `internal/service/transmuter.go` `fetchShadowBatch()` function.

2. **B4 — schema validator**: Cache Avro schema theo schema_id (đã có) nhưng cho phép
   *additive* fields (worker shadow tier theo V1 conventions vốn append-only). Hoặc
   chuyển từ hard-fail sang soft-warn + auto-evolve (architect rule cần).

3. **B5 — DLQ binary**: Đổi cột `failed_sync_logs.raw_json` từ TEXT/JSONB sang `bytea`,
   hoặc base64-encode Avro bytes trước INSERT. Hiện tại 3 message offset 55/56/57
   stuck redelivery loop, burns CPU.

### 4.2 Ưu tiên trung (Track E hoàn chỉnh)

4. **B8 — MariaDB connector**: Drop `debezium-connector-mysql-2.x.jar` vào
   `gpay-kafka-connect:/usr/share/confluent-hub-components/`, restart Connect, tạo
   `cdc.mariadb.legacy_orders` connector → mở topic.

5. **B3 — Mongo logical-vs-physical addtest**: Cần architect rule trước khi triển khai.
   Hiện locator id=31 trỏ `payment_bills` nhưng physical `payment_bills_addtest` cũng
   tồn tại 5 docs. Nếu giữ logical-clone: 1 Debezium event fan-out 2 shadow.

### 4.3 Ưu tiên thấp (cleanup)

6. **3 test rows source**: `DELETE FROM public.orders WHERE notes LIKE 'track-e-test-%'`
   — sau khi B4/B5 fix xong và DLQ drained.

7. **Stale failed source objects** (theo report cũ ghi 6 entries `failed`): pruning sau
   khi unblock data-plane.

---

## 5. Files thay đổi trong loop này

| Path | Action |
|------|--------|
| `agent/memory/workspaces/feature-cdc-integration/report_v2_bridge_e2e_verification.md` | NEW (this file) |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND (verification entry) |

**Không có code change**. Brain prohibition §12 honored — chỉ verify + document.

---

## 6. Tại sao kết thúc loop (không ScheduleWakeup)

Loop dynamic mode hỏi "verify V2 bridge end-to-end after cron tick". Kết quả:
- ✅ V2 bridge control-plane PASS.
- ❌ Data-plane FAIL — 3 blocker code-level (B4/B5/B6).

Code-level blocker → wait-and-retry không thay đổi gì. Cần:
- Hoặc Muscle commit fix → user explicit re-trigger verify.
- Hoặc architect rule đổi convention.

→ Kết thúc loop. Không schedule wakeup.

---

## 7. Skills used (CLAUDE.md §0 — liệt kê cuối câu trả lời)

- `Bash` — psql cli (docker exec) để query state machine + binding tables + shadow data
- `Read` — đọc gap analysis + reset script + plan file (Track D Hardening) để cross-check
- `Write` — sinh report file mới (vật lý hóa, không shadow)
- `Skill` — `/loop` dynamic mode (single-iteration verify)
- Governance: CLAUDE.md §3 (Plan & Verify), §11 (APPEND-only), §12 (Brain prohibition),
  §14 (pre-flight check)
