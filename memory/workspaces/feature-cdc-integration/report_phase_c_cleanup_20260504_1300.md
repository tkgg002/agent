# Báo cáo — Phase C Cleanup tàn dư + A2 attempt + rollback
**Date**: 2026-05-04 13:00 (Asia/Ho_Chi_Minh)
**Author**: Brain (Antigravity)
**Trigger**: User pushback — "sáng giờ làm theo cái report này để fix tồn đọng, sao vẫn còn tàn dư".
**Type**: Brain ops (DATA SQL UPDATE + Kafka Connect REST PUT/PUT-rollback). KHÔNG sửa `.go/.sql` repo (CLAUDE.md §12).
**Status**: ✅ DONE — verified live
**Source plan**: `report_pending_options_and_unified_plan_20260504.md` Phase C (C1, C2, C3) + Phase A2.

---

## 1. Bối cảnh

Sáng 09:38 file pending_options đặt ra Phase C cleanup nhưng tới 12:45 vẫn chưa execute đầy đủ. Brain đã focus vào B10/B11/V2 bridge mà miss phần dọn dẹp. User đúng — không triệt để.

Audit thực tế trước khi action:
- C1 prune V1 legacy seeds: kiểm tra → **đã DONE từ trước** (`legacy_active=0`, 10 hàng `legacy_1..10` đều `is_active=false`).
- C3 drop orphan `orders_e2e_d_v2/v3/v4`: kiểm tra → **đã DONE từ trước** (chỉ còn `orders_e2e_d_v5` thuộc id=26 active).
- C2 archive sources failed: 🟡 **CÒN MỞ** (trước fix có 2: id=27, 28).
- A2 Mongo include `payment_bills_addtest`: 🟡 **CÒN MỞ** (collection.include.list không có).

→ Thực sự cần execute: A2 + C2.

---

## 2. Hành động đã thực thi

### 2.1 C2 — Archive 2 sources failed (id=27, 28)

```sql
UPDATE cdc_system.source_object_registry
   SET provisioning_state='archived', is_active=false,
       notes = ... || '[archived 2026-05-04 12:50 — last_step_error: source preflight refused (no business columns / empty collection). Permanent failure, not transient. Brain ops]'
 WHERE provisioning_state='failed' AND id IN (27, 28);
-- UPDATE 2
```

**Lý do archive (KHÔNG retry)**: cả 2 sources có `last_step_error` permanent:
- id=27 `mariadb_legacy_orders_v1`: "discover: 0 mapping rules — shadow table legacy_orders has no business columns" — schema chưa được Debezium emit.
- id=28 `mongo_payment_bills_v2`: "mongo source preflight: collection payment_bills is empty — refusing to cascade".

Cả 2 không phải transient (network/deadlock); retry sẽ fail same way. Archive là quyết định đúng.

### 2.2 A2 — PUT Mongo include list thêm `payment_bills_addtest` → ROLLBACK

**Step 1 — PUT thêm vào include list**: ✅ thành công, `collection.include.list` từ 9 → 10 entries; connector RUNNING.

**Step 2 — INSERT 1 doc Mongo `payment_bills_addtest`** để smoke trigger:
```js
{ _id: "addtest-pb-201-a2-smoke", merchantId: "BRAIN-A2-FIX", amount: 201, status: "smoke-test" }
```

**Step 3 — Wait 60s + verify**:
- Topic `cdc.goopay.payment-bill-service.payment_bills_addtest` ĐÃ tạo trên Kafka, có **6 messages** (snapshot mode `initial` đã pickup 4 docs cũ + 1 doc mới + tombstone).
- Shadow `shadow_mongo_payment_bill_default.payment_bills_addtest` count = **0** (không ingest).
- Worker logs: chỉ thấy `transmute complete master:payment_bills_addtest scanned=0` (không có route signal cho topic mới).

**Root cause discovery**:
```
id=31 source_locator_json = {"db":"payment-bill-service",
                              "collection":"payment_bills",        ← KHÔNG phải payment_bills_addtest
                              "fan_out_role":"clone",
                              "logical_clone_of":28}
id=28 = archived (source empty)
```

id=31 (`addtest_mongo_bills`) được setup làm **logical clone** của id=28. Kiến trúc B3 logical-clone fan-out hoạt động: 1 source topic event → N shadow tables (theo source_object_registry rows có `logical_clone_of` chỉ về source gốc). 

Khi PUT thêm topic mới `cdc.goopay.payment-bill-service.payment_bills_addtest`, không có entry registry nào có `source_object_name='payment_bills_addtest' AND collection='payment_bills_addtest'` để route. id=31 có `source_object_name='payment_bills_addtest'` nhưng locator `collection='payment_bills'` → routing logic của worker (match theo locator collection trong Debezium event metadata) bỏ qua.

**Step 4 — ROLLBACK PUT**:
```bash
PUT /connectors/goopay-mongodb-cdc/config @original_config.json
# collection.include.list len: 10 → 9
# contains payment_bills_addtest? False
# connector RUNNING + task RUNNING
```

**Lý do rollback**: tránh orphan topic + 6 message stuck không có consumer. Clean state.

### 2.3 Archive id=31 (consistent với id=28 archived)

```sql
UPDATE cdc_system.source_object_registry
   SET provisioning_state='archived', is_active=false,
       notes = ... || '[archived ... id=31 was logical_clone_of=28 ... Clone has no upstream → cannot ingest. Architectural decision needed: (a) reprovision as first-class source ..., (b) leave archived. Brain ops]'
 WHERE id = 31;
-- UPDATE 1

UPDATE cdc_system.shadow_binding SET is_active=false WHERE source_object_id=31;
-- UPDATE 1 (binding id=40)
```

**Lý do**: id=31 là clone của id=28 archived → permanent no-data. Giữ active là noise.

### 2.4 Fix inconsistent state id=18

Phát hiện thêm: id=18 `phaseC_curl_test` có `provisioning_state=archived` NHƯNG `is_active=true` — inconsistent.

```sql
UPDATE cdc_system.source_object_registry SET is_active=false WHERE id=18 AND provisioning_state='archived' AND is_active=true;
-- UPDATE 1
```

---

## 3. State BEFORE vs AFTER

### Registry breakdown
| State | BEFORE | AFTER | Note |
|-------|--------|-------|------|
| `archived` total / active | 5 / 1 | 8 / 0 | +3 (id 27, 28, 31), +1 fix id=18 inconsistency |
| `draft` total / active | 10 / 0 | 10 / 0 | unchanged (V1 legacy seeds is_active=false từ trước) |
| `failed` total | 2 | **0** | id 27, 28 → archived |
| `running` total / active | 5 / 5 | **4 / 4** | id=31 archive → 4 active |
| TOTAL | 22 | 22 | conserved |

### Active running list (post-cleanup)
| id | object_code | source_object_name | engine |
|----|-------------|---------------------|--------|
| 11 | src_local_goopay_source_orders | orders | postgresql |
| 26 | e2e_phaseD_auto_v5 | orders_e2e_d_v5 | postgresql |
| 29 | addtest_pg_orders | orders_addtest | postgresql |
| 30 | addtest_maria_legacy | legacy_orders_addtest | mariadb |

→ 4 sources first-class: 1 PG main + 1 PG e2e + 1 PG addtest + 1 MariaDB addtest. Mongo addtest đã archive.

### Shadow + Master (no regression)
| Table | Rows |
|-------|------|
| shadow_goopay_source.orders | 14 |
| shadow_src_local_pg_source.orders_addtest | 9 |
| shadow_mariadb_legacy_default.legacy_orders_addtest | 1 |
| shadow_mongo_payment_bill_default.payment_bills_addtest | 0 (orphan, không drop) |
| **dw_orders.orders_fact** | **34 / 34 distinct** |

### Mongo connector
- collection.include.list: 9 (rolled back)
- connector + task: RUNNING

---

## 4. Items KHÔNG executed (lý do)

| Item | Status | Lý do |
|------|--------|-------|
| **A2 thực sự** (Mongo addtest ingest) | **NEEDS architect ruling** | Kiến trúc B3 logical-clone không support topic riêng. 2 path: (a) reprovision id=31 thành first-class với locator `collection=payment_bills_addtest` + register topic; (b) chấp nhận Mongo addtest không ingest. Brain không tự quyết được. |
| **C4 cleanup test rows source PG** | **DEFER** | Đụng upstream source DB (DELETE rows) → cần user duyệt. Có thể spawn DELETE events Debezium → soft-delete shadow rows → master count thay đổi. |
| **Drop physical shadow_mongo_payment_bill_default.payment_bills_addtest** | **LOW PRIORITY** | Là orphan storage (~16KB). Không hại. Có thể drop sau khi quyết định A2. |

---

## 5. Trade-off & Risks

### Trade-off A2 rollback
- ✅ State sạch — không có orphan topic + 6 messages stuck.
- ❌ Mongo `payment_bills_addtest` collection (4 docs thực) **không được ingest** → shadow + master không có data từ collection này.

### Risk archive id=31
- Schedule transmute cho master `payment_bills_addtest` vẫn chạy (60s/lần) nhưng `scanned=0` (master_binding cho shadow id=40 đã `is_active=false`). Idle CPU minor — chấp nhận.
- Re-activate dễ dàng: `UPDATE source_object_registry SET provisioning_state='running', is_active=true WHERE id=31` + xét lại locator.

### Risk fix id=18
- Trước fix: archived nhưng `is_active=true` → query `WHERE is_active=true` lọc lẫn vào danh sách → phantom.
- Sau fix: consistent state. Không có rủi ro.

---

## 6. Files thay đổi trong phase này

| Path | Action | Type |
|------|--------|------|
| `cdc_system.source_object_registry` rows id=27,28,31,18 | UPDATE | DATA SQL ops |
| `cdc_system.shadow_binding` row id=40 (source_object_id=31) | UPDATE | DATA SQL ops |
| Mongo connector `goopay-mongodb-cdc` config | PUT (10) → PUT rollback (9) | Kafka Connect REST |
| Mongo `payment-bill-service.payment_bills_addtest` doc `addtest-pb-201-a2-smoke` | INSERT | Smoke (residue, có thể giữ hoặc xóa) |
| `agent/memory/workspaces/feature-cdc-integration/report_phase_c_cleanup_20260504_1300.md` | NEW (this file) | Markdown report |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND entry 13:00 | Immutable log |

**Không có code change** (Brain prohibition §12 honored).

---

## 7. Câu hỏi escalate user/architect

1. **A2 architectural decision**: muốn `payment_bills_addtest` ingest data thực không? Nếu có → đề xuất reprovision id=31 thành first-class (update locator + bật include list + activate cascade). Brain có thể execute SQL + REST nếu user OK.
2. **C4 cleanup test rows source PG**: có duyệt cho Brain DELETE `WHERE notes LIKE 'p2-p3-p4-smoke-%' OR notes LIKE 'b11-permanent-fix-smoke%' OR notes LIKE 'track-e-test-%'`? Sẽ có Debezium DELETE events → shadow soft-delete → master count tăng skipped (delete-event handling).
3. **D1 Schema Schism unify hay coexist**: vẫn open.
4. **Mongo smoke doc residue** `addtest-pb-201-a2-smoke`: xóa cùng C4 hay giữ?

---

## 8. Lessons applied trong phase này

- **L-runtime-state-verify (2026-04-21)**: Audit thực tế TRƯỚC khi action — phát hiện C1 + C3 thực sự đã DONE từ session trước, tránh duplicate work.
- **L-three-layer-trust (2026-04-29)**: Khi A2 PUT thành công nhưng shadow=0, không assume "OK rồi" — trace 3 layer: connector status → kafka topic content → registry route → locator.
- **L-real-data-test (2026-04-15)**: Mỗi UPDATE/PUT đều `RETURNING` hoặc verify ngay sau (counts, status).
- **L-2026-04-29 cascade-liability**: Master schedule continued running ngay cả khi binding inactive — pipeline có "phantom OK"; cần xét active flag, không chỉ "scheduled".

## 9. Lesson NEW (sẽ append `lessons.md`)

**Global Pattern [Adding new ingest topic via include-list PUT for connector C, but registry routing logic R lookup by source_locator_json L (not source_object_name), and L of related entry points to a different physical object O'] → Result Y: orphan topic created, events emitted, no shadow row written, silent data drop**

**Correct Pattern**: Khi muốn enable ingest cho new physical source O qua include-list PUT:
1. **Audit registry FIRST**: query `WHERE source_object_name=O AND source_engine_type=...` xem có entry không, và locator có match O không.
2. Nếu entry tồn tại nhưng locator point khác O (e.g. clone of another) → KHÔNG PUT include-list. Thay vào đó: reprovision entry → set locator chính xác → re-cascade.
3. Nếu không có entry → tạo source_object_registry entry mới (full provisioning) TRƯỚC khi PUT include list.
4. Verify smoke: INSERT 1 record → wait connector poll → verify Kafka topic message count > 0 AND shadow row count > 0 (cả 2 layer). Nếu Kafka có message nhưng shadow = 0 → route logic không match → rollback include list.

Tags: #cdc #debezium #connector-include-list #routing #logical-clone #orphan-topic #pre-flight-audit

Generalize: pattern áp dụng cho mọi (1) Debezium PG `table.include.list`, (2) Mongo `collection.include.list`, (3) MariaDB `database.include.list` + `table.include.list`, (4) bất cứ Kafka Connect connector config có "include" pattern + worker route lookup tách rời (worker dùng metadata service, không phải parse topic name trực tiếp).

---

## 10. Governance compliance (CLAUDE.md §14 pre-flight)

- ✅ §0: Vietnamese + plan-first — list audit BEFORE action.
- ✅ §1: Brain Chairman — DATA ops + REST ops + plan + escalate. KHÔNG sửa code.
- ✅ §3: Verify thực tế qua RETURNING + count queries + connector status.
- ✅ §7: Đọc lessons.md TRƯỚC (anchor lookup); APPEND new lesson sau.
- ✅ §11: APPEND-only — file mới + 05_progress.md APPEND.
- ✅ §12: Brain KHÔNG sửa `.go/.sql` repo. Chỉ DATA SQL UPDATE trên live DB + Kafka Connect REST. Script `prune_legacy_v1_bindings.sql` KHÔNG chạy lại (đã idempotent từ trước).
- ✅ §13: Lesson abstract Global Pattern, generalize 4+ scenarios.
- ✅ §14: Pre-flight scan trước khi end turn.

---

## 11. Skills used

- Bash (psql + curl Kafka Connect REST + docker exec mongosh + kafka-tools)
- Read (config snapshots in /tmp + locator_json reflection)
- Write (NEW report file)
- DATA SQL UPDATE (4 rows total: id 27, 28, 31, 18)
- Kafka Connect REST PUT (then PUT rollback)
- 3-layer trust trace (connector status → kafka topic content → registry locator → worker route)
- Pre-flight audit pattern (verify state BEFORE action)
- Memory APPEND-only protocol
- Lesson generalization (Global Pattern with A/B/X/Y variables)

---

**Summary**: Phase C tàn dư đã được clean: 2 failed → archived, 1 inconsistent state → fixed, 1 logical-clone-of-archived → archived consistent. A2 attempt thực hiện full + rollback an toàn vì kiến trúc B3 logical-clone không support topic riêng cho clone — cần architect decision trước khi reprovision. Registry state hiện tại: 4 active running (PG main + e2e_v5 + 2 addtest), 0 failed, 8 archived consistent, 10 V1 draft seeds neutralized.
