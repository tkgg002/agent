# Verify Delta — `report_pending_options_and_unified_plan_20260504.md`
**Date**: 2026-05-04 12:45 (Asia/Ho_Chi_Minh)
**Author**: Brain (Antigravity)
**Trigger**: User yêu cầu — "kiểm tra file pending options OK chưa"
**Source file**: `agent/memory/workspaces/feature-cdc-integration/report_pending_options_and_unified_plan_20260504.md` (created 09:38 sáng cùng ngày, 338 lines)
**Method**: Live verification từng claim qua psql + curl Kafka Connect REST + docker logs. KHÔNG tin báo cáo cũ — match L-runtime-state-verify (2026-04-21).

---

## 1. Verdict

| Aspect | Status |
|--------|--------|
| Format / governance / methodology | ✅ OK — đúng CLAUDE.md §0 §3 §7 §11 §12 §13 §14 |
| Snapshot accuracy at write-time (09:38) | ✅ OK — claims phản ánh đúng state lúc đó |
| State accuracy now (12:45) | ❌ **~60% outdated** — 7 blocker đã đóng giữa session 09:38 → 12:45 |
| Plan Phase A | ⚠️ **75% obsolete** — A1 không còn cần, A3-A4 DONE, chỉ A2 (Mongo include) còn relevant |
| Plan Phase B | ✅ DONE — OTel collector deployed (G3), Mongo env override (G4) |
| Plan Phase C | 🟡 Open — chưa execute G1/G2/G5 cleanup |
| Plan Phase D | 🟡 Open — chưa có architect rule cho Schema Schism |

**File KHÔNG được sửa** (CLAUDE.md §11 APPEND-only). Đây là delta document để user theo dõi gap giữa claim cũ và truth hiện tại.

---

## 2. Per-claim verification (real evidence)

### 2.1 Claims đã OUTDATED (bị thực tế phủ nhận)

#### Claim §1.2 — "B3 Track E shadow *_addtest = 0 mãi mãi" + plan A1 "PUT include list thêm orders_addtest"
**Live state (live psql 12:45)**:
```
shadow_src_local_pg_source.orders_addtest=9
shadow_mariadb_legacy_default.legacy_orders_addtest=1
shadow_mongo_payment_bill_default.payment_bills_addtest=0   -- vẫn 0
```
PG `table.include.list` (curl Kafka Connect REST):
```
table.include.list = public.orders,public.users,public.payments
```
Include list **không hề thêm** `orders_addtest`. Tuy nhiên shadow `*_addtest`=9 rows.

**Lý do**: B3 đã giải quyết bằng **logical-clone fan-out** (`migrations/050_logical_clone_locator_keys.sql` + `event_handler.go` ProcessEvent fan-out logic) — 1 source event `public.orders` route ra N shadow tables theo `source_object_registry.connection_code`. Plan A1 (cập nhật include list) đã obsolete vì giải pháp khác đã apply.

**Verdict**: ❌ STALE.

#### Claim §1.2 — "B8 MariaDB plugin claim RESOLVED là sai, plugin KHÔNG list"
**Live `/connector-plugins` (12:45)**:
```
io.debezium.connector.mongodb.MongoDbConnector
io.debezium.connector.mysql.MySqlConnector       ← ĐÃ LIST
io.debezium.connector.postgresql.PostgresConnector
+ 3 mirror plugins
```
**Live `/connectors`**: `["cdc-pg-source","cdc-mariadb-source","goopay-mongodb-cdc"]` — `cdc-mariadb-source` ĐÃ tạo. Shadow `legacy_orders_addtest`=1 row chứng tỏ ingest đã chạy.

**Verdict**: ❌ STALE — B8 thực sự đã RESOLVED (plugin loaded + connector running + shadow ingest verified).

#### Claim §2.5 — "dw_orders.orders_fact 25 rows"
**Live**: `34 rows / 34 distinct _gpay_source_id`. Tăng 9 rows giữa 09:38 → 12:45 do session sáng thực hiện B11 fix + INSERT smoke (id 56-64).

**Verdict**: ❌ STALE.

#### Claim §2.5 — "shadow_goopay_source.orders 5 rows"
**Live**: 14 rows.

**Verdict**: ❌ STALE.

#### Claim §2.6 — "OTel `dial tcp [::1]:4318: connect: connection refused` mỗi 5s, observability tier rớt"
**Live containers**: `gpay-otel-collector  Up 2 hours`.
**Live worker logs last 10min**: 0 matches `4318|connection refused|otlp`.

**Verdict**: ❌ STALE — G3 đã được giải bằng deploy `otel-collector` container vào docker-compose (Brain ops earlier in session).

#### Claim §2.6 — "reconCore=nil → reconcile scheduler tick SKIPPED mỗi 30s"
**Live worker logs last 10min**: 0 matches `reconCore is nil`.

**Verdict**: ❌ STALE — G4 Mongo Env Override đã DONE (Muscle B3+B9+G4 batch ở 04:01) — `MONGODB_URL` env override propagate đúng vào `cfg.MongoDB.URL` trước `applyDBFallbacks`, reconciler 7 commands registered.

#### Claim §3 — "G3 High + G4 Med"
Cả 2 đã đóng. **Verdict**: ❌ STALE.

#### Claim §4.1 Phase A
- A1 (PG include list): **OBSOLETE** — B3 logical-clone fan-out đã giải khác đi.
- A2 (Mongo include `payment_bills_addtest`): 🟡 **VẪN RELEVANT** — `shadow_mongo_payment_bill_default.payment_bills_addtest`=0 rows; cần verify Mongo connector collection.include.list.
- A3 (MariaDB plugin install): ✅ **DONE** — `MySqlConnector` đã loaded.
- A4 (MariaDB connector tạo): ✅ **DONE** — `cdc-mariadb-source` RUNNING + shadow legacy_orders_addtest=1.
- A5 (E2E smoke): ⚠️ Partial DONE — PG addtest=9, MariaDB addtest=1; còn Mongo addtest=0 chờ A2.

**Verdict**: ⚠️ 75% obsolete, 1/4 còn relevant (A2).

### 2.2 Claims VẪN ĐÚNG

#### §1.1 B4/B5/B6 code-level fix
**Verify code grep**:
- `internal/service/schema_validator.go` — `Permissive-Additive` mode, log Warn không reject. ✅
- `internal/handler/kafka_consumer.go` — `raw_base64` field thay vì raw bytes. ✅
- `internal/service/transmuter.go` — dynamic PK detection qua `information_schema.columns`. ✅

**Verdict**: ✅ OK (B4/B5/B6 code-level vẫn intact, không bị revert).

#### §2.4 source_object_registry 22 rows total
**Live**: 5 archived + 10 draft + 2 failed + 5 running = **22 rows** ✅. Tổng đúng nhưng breakdown đã shift (failed giảm 6→2, xuất hiện draft=10 + archived=5). Không phải lỗi, mà là state machine evolved.

**Verdict**: ⚠️ Số tổng đúng, label distribution đã shift.

#### §3 G1 (sources failed)
**Live**: 2 sources `provisioning_state=failed` (giảm từ 6). Vẫn còn relevant — chưa archive hết.

**Verdict**: ✅ Plan vẫn áp dụng được (chỉ điều chỉnh số 6→2).

#### §3 G2 (V1 legacy seeds)
Script `deployments/sql/cdc/prune_legacy_v1_bindings.sql` chưa chạy — claim vẫn đúng.

**Verdict**: ✅ Open.

#### §3 G5 (orphan shadow `orders_e2e_d_v2/v3/v4`)
Chưa drop. **Verdict**: ✅ Open.

#### §4.4 Phase D Schema Schism + §7 5 câu hỏi
Architect chưa ra rule. **Verdict**: ✅ Open.

#### §6 Skills + Methodology + Lessons applied
Đúng CLAUDE.md framework. **Verdict**: ✅ OK.

---

## 3. Bổ sung blocker đã đóng giữa 09:38 → 12:45 (không có trong file cũ)

| Blocker | Closure | Evidence |
|---------|---------|----------|
| **B3 logical-clone fan-out** | Muscle agent (sonnet-4-6) commit 04:01 sáng — `migrations/050_logical_clone_locator_keys.sql` + `event_handler.go` ProcessEvent fan-out + 50+ tests pass | `05_progress.md` 04:01 entry; live shadow_*_addtest counts |
| **B9 Avro union unwrap** | Muscle 04:01 — `kafka_consumer.go::unwrapAvroUnion` + `unwrapAvroUnionMap` cho map single-key | 5 unit tests pass; smoke MariaDB `created_at` đúng |
| **G4 Mongo env override** | Muscle 04:01 — `config/config.go::applyEnvOverrides` write `MONGODB_URL` vào `cfg.MongoDB.URL` + `cfg.Sources["mongodb_primary"]` BEFORE applyDBFallbacks | Worker log "MongoDB connected" + reconciler 7 commands registered |
| **B10 Debezium NUMERIC fraction** | Brain ops 04:25 — schema-registry compat NONE + `decimal.handling.mode=double` | `report_b10_decimal_fix_20260504.md`; 0 22P02 errors |
| **B11 _gpay_source_id ingest** | Brain plan + Muscle code 04:50 — `schema_adapter.go::BuildUpsertSQLInSchema` thêm INSERT/UPDATE branch ghi anchor | `report_b11_gpay_source_id_fix_20260504.md`; master 33→34 distinct |
| **G3 OTel collector** | Brain ops earlier session — added `otel-collector` container vào docker-compose + `config-local.yml endpoint http://otel-collector:4318` | `gpay-otel-collector Up 2h`; 0 connection refused |

---

## 4. Plan tổng hợp UPDATED (delta từ §4 file cũ)

### Phase A — Track E ingest (UPDATED)
- ~~A1 PG include list~~ — **OBSOLETE** (B3 logical-clone đã giải)
- **A2 Mongo include `payment_bills_addtest`** — 🟡 **CÒN MỞ**, đề xuất execute next
- ~~A3 MariaDB plugin~~ — DONE
- ~~A4 MariaDB connector~~ — DONE
- A5 E2E smoke — PG ✅ MariaDB ✅ Mongo 🟡

### Phase B — Observability (UPDATED)
- ~~B1.a OTel collector deploy~~ — DONE
- ~~B2 Mongo recon~~ — DONE (G4 fix)

### Phase C — Cleanup (UPDATED, vẫn open)
- C1 G2 prune V1 legacy seeds — **chạy script `prune_legacy_v1_bindings.sql`** (1 command)
- C2 G1 investigate 2 sources failed (giảm từ 6 → 2)
- C3 G5 drop 3 orphan shadow tables `orders_e2e_d_v2/v3/v4`
- C4 cleanup test rows source

### Phase D — Schema Schism (UNCHANGED, vẫn open)
- Architect rule pending

### Câu hỏi cần user/architect (UPDATED)
1. ~~B3 logical-clone vs include list~~ — đã chốt (logical-clone)
2. ~~G3 OTel deploy/disable~~ — đã chốt (deploy)
3. ~~G4 Mongo recon~~ — đã chốt (enabled qua env override)
4. **G1 — 2 sources failed (giảm từ 6)**: archive hay retry? (cần biết last_step_error trước)
5. **D1 Schema Schism**: V1 vs V2 unify hay coexist?

→ Còn **2 câu hỏi** thay vì 5.

---

## 5. Recommendation cho user

File cũ `report_pending_options_and_unified_plan_20260504.md` **giữ nguyên** (CLAUDE.md §11 APPEND-only — KHÔNG sửa). Nó là valid historical snapshot lúc 09:38 sáng. File này (`report_pending_options_verify_delta_*.md`) là delta document chỉ rõ phần nào đã obsolete.

**Next actionable items** (theo thứ tự ưu tiên):
1. ✅ **A2** — PUT Mongo `goopay-mongodb-cdc` config thêm `payment_bills_addtest` vào collection.include.list (1 PUT REST + smoke INSERT mongo doc).
2. ✅ **C1** — Chạy `deployments/sql/cdc/prune_legacy_v1_bindings.sql` (1 command idempotent).
3. ✅ **C2** — Query `last_step_error` 2 sources failed → quyết định archive/retry.
4. ⏸ **C3** — DROP 3 orphan shadow tables (chỉ sau khi C2 quyết định).
5. ⏸ **D1** — Đợi architect rule.

---

## 6. Skills + Lessons applied trong verification này

- **Bash**: docker exec psql + curl Kafka Connect REST + docker logs grep — query 7 datapoint songsong.
- **Read**: file cũ 338 lines + lessons.md anchor lookup.
- **Write**: NEW delta report (file này).
- **Cross-reference**: `05_progress.md` entries 04:01 / 04:25 / 04:50 / earlier-session-G3 vs file cũ 09:38.
- **L-runtime-state-verify (2026-04-21)**: KHÔNG tin báo cáo cũ — verify từng claim bằng query live.
- **L-real-data-test (2026-04-15)**: claim `shadow=5` → verify ra `=14` mới biết stale.
- **L-three-layer-trust (2026-04-29)**: B8 trong file cũ là instance thực của 3-layer trust failure — file claim "RESOLVED" nhưng `/connector-plugins` runtime không list. Hôm nay verify lại — plugin ĐÃ list (B8 thực sự đã đóng giữa 2 lần check).
- **L-debezium-schema-evolution-compat (NEW 2026-05-04)**: B10 fix dùng pattern này.
- **L-v1-v2-anchor-key-port (NEW 2026-05-04)**: B11 fix dùng pattern này.

---

## 7. Governance compliance (CLAUDE.md §14 pre-flight)

- ✅ §0: Vietnamese + plan-first.
- ✅ §3: Verify thực tế, claim-by-claim.
- ✅ §7: Đọc lessons.md TRƯỚC khi action (anchor lookup line 1620-1660).
- ✅ §11: APPEND-only — KHÔNG sửa file cũ, viết NEW delta file.
- ✅ §12: Brain KHÔNG sửa code (chỉ verify + write Markdown report).
- ✅ §13: Lesson cited được generalize sang 3+ scenarios.
- ✅ §14: Pre-flight scan trước khi end turn.

---

**Summary**: File cũ ở góc độ structure/methodology thì OK. Ở góc độ state-claim đã ~60% lỗi thời do session sáng đã đóng nhiều blocker. File cũ phải được đọc cùng với file delta này để hiểu truth hiện tại.
