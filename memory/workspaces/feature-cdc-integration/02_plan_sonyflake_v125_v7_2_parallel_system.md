# Master Plan v1.25 — v7.2 (Parallel System Focused)

> **Date**: 2026-04-21
> **Author**: User (prescription) + Brain (transcription + technical review notes)
> **Supersedes**: v7.1 (dual-write complexity eliminated)
> **Core pivot**: Airbyte `public` = legacy, để nó tự bơi. `cdc_internal` = thế giới mới sạch tuyệt đối via Debezium + Go Worker. **Không dọn rác, không dual-write.**

---

## 0. Architectural Shift

```
┌─────────────────────────────────────────────────────────────┐
│ LEGACY (để Airbyte tự bơi)                                  │
│   MongoDB → Airbyte → public.<table>                        │
│   (_airbyte_* metadata + schema flattened)                  │
│   Brain/Worker KHÔNG touch                                  │
└─────────────────────────────────────────────────────────────┘
                         ×  (no bridge)
┌─────────────────────────────────────────────────────────────┐
│ NEW CLEAN (cdc_internal)                                    │
│   MongoDB → Debezium → Kafka → Go Worker → cdc_internal.*   │
│   9 fields hệ thống + business fields từ payload           │
│   Sonyflake identity, fencing trigger, schema-on-read       │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. Phase 0: Foundation (Identity)

Muscle thiết lập "luật chơi" tại Postgres trước khi Worker nổ máy.

- **T0.1 MachineID Registry**: `cdc_internal.worker_registry` + SEQUENCE `machine_id_seq` + `fencing_token_seq` + `claim_machine_id()` + `heartbeat_machine_id()`
- **T0.2 BEFORE Trigger**: `tg_fencing_guard()` function — chặn CẢ INSERT + UPDATE nếu session token không khớp. Attach per-table ở T1.3.
- **T0.3 System Registry**: Bảng catalog theo dõi trạng thái sync per table (`pending_data` | `syncing` | `active`)

Function ready, chưa attach trigger nào (attach theo bảng khi T1.3 create shadow).

---

## 2. Phase 1: The New Pipeline (Debezium Sink)

Go Worker tiêu thụ Kafka, tự tạo/alter shadow table.

- **T1.1 Capture All**: Worker lấy 100% payload Debezium → `_raw_data` JSONB. Không quan tâm shadow schema — cứ JSONB nguyên bản tống vào.
- **T1.2 9-Field Enforcement**: Mọi record vào shadow phải có đủ:
  - `_gpay_id` (Sonyflake)
  - `_raw_data` (JSONB payload)
  - `_source` ('debezium')
  - `_synced_at`
  - `_version`
  - `_hash`
  - `_deleted`
  - `_created_at`
  - `_updated_at`
- **T1.3 Shadow Auto-Migration**: Worker phát hiện schema từ Kafka → `CREATE TABLE` hoặc `ALTER TABLE` trong `cdc_internal` + attach `tg_fencing_guard` mỗi khi create mới.

---

## 3. Phase 2: Backfill & Recon (hệ thống mới)

Chỉ đổ từ **nguồn Mongo** → `cdc_internal`. KHÔNG đọc `public.*` (Airbyte legacy).

- **T2.1 Debezium Incremental Snapshot**: dùng Debezium Signal collection (`execute-snapshot`) yêu cầu connector re-emit toàn bộ Mongo docs → Kafka → Worker consume bình thường → cdc_internal.
  - (Lưu ý Brain review: user đề xuất `ctid` scan là **KHÔNG apply** cho Mongo source; ctid là PG-to-PG. Bản này fix sang Debezium Signal.)
- **T2.2 Parallel Mapping**: Backfill chạy đồng thời Streaming. `ON CONFLICT DO NOTHING` WHERE `_gpay_source_ts` cũ hơn.

---

## 4. Inbound Policy (Luật nạp dữ liệu)

- **Policy 1 Strict Sonyflake**: Record không có ID hợp lệ = Reject DLQ
- **Policy 2 Payload Integrity**: `_raw_data` khớp `_hash` (Worker-computed deterministic). Mismatch = DLQ
- **Policy 3 Financial Override**: Bảng financial (regex field name) → admin verify schema thủ công trước khi bật Sink

---

## 5. Systematic Rollout (by data characteristic)

| Nhóm | Đặc điểm | Chiến lược |
|:-----|:---------|:-----------|
| **High-Stake (Financial)** | Có trường `amount`, `balance` | Inline Fencing + Admin review schema |
| **High-Volume (Event/Logs)** | 1M+ rows, payload ổn định | Parallel Backfill via Debezium Signal |
| **Empty/Seed Only** | 0 rows hoặc < 1000 | Đánh dấu `pending_data`, chỉ bật streaming, bỏ qua backfill |

---

## 6. Go Worker — Systematic Dynamic Mapping (user spec)

### 6.1 Quy trình 6 bước

1. **Phân rã payload Debezium**: bóc `after` thành `map[string]interface{}` — mọi field source (kể cả field mới) đều có trong map
2. **Hợp nhất**: merge 2 nguồn:
   - **System** (9 fields Worker tính)
   - **Business** (toàn bộ key-value Debezium after)
3. **Thực thi SQL**: GORM Map-based Insert đẩy toàn bộ vào DB

### 6.2 Code skeleton (user prescription)

```go
func (w *SinkWorker) HandleMessage(ctx context.Context, msg KafkaMessage) error {
    // 1. Đóng gói 100% Payload nguyên bản (Source of Truth)
    rawJSON := msg.Value
    
    // 2. Parse payload nghiệp vụ thành map động
    businessData := make(map[string]any)
    if err := json.Unmarshal(rawJSON, &businessData); err != nil {
        return err
    }

    // 3. Khởi tạo Final Record với 9 System Default Fields
    finalRecord := map[string]any{
        "_gpay_id":    w.idProvider.NextID(), // Sonyflake
        "_raw_data":   rawJSON,                // Bảo toàn cột D
        "_hash":       w.hasher.Compute(rawJSON),
        "_version":    w.extractVersion(msg),
        "_source":     "debezium",
        "_synced_at":  time.Now(),
        "_deleted":    w.isDeleted(msg),
        "_created_at": time.Now(),
        "_updated_at": time.Now(),
    }

    // 4. SYSTEMATIC MERGE: Hợp nhất toàn bộ cột nghiệp vụ vào record
    // Không quan tâm tên cột là gì (a, b, c, d hay 100 cột khác)
    for key, value := range businessData {
        // Tránh ghi đè các cột hệ thống
        if !strings.HasPrefix(key, "_gpay_") {
            finalRecord[key] = value
        }
    }

    // 5. Schema-on-read: Tự động điều chỉnh Shadow Table
    // Nếu trong finalRecord có key 'd' mà DB chưa có, thực hiện ALTER TABLE
    if err := w.schemaManager.EnsureFields(ctx, msg.TableName, finalRecord); err != nil {
        return err
    }

    // 6. Insert/Save Map
    return w.db.Table(msg.TableName).Save(finalRecord).Error
}
```

---

## 7. Technical Review Notes (Brain honest flag)

Brain reviews user code + v7.2 sections, flag 6 specific technical gaps **không rewrite, chỉ document để Muscle aware trước khi implement**:

### 7.1 `_source_ts` missing from finalRecord

v7.1 Migration 009 OCC guard dùng `_source_ts BIGINT` (Debezium `source.ts_ms`). Code trong 6.2 không set `_source_ts` → OCC WHERE clause không work. **Fix**:
```go
"_source_ts": w.extractSourceTsMs(msg),  // from payload.source.ts_ms
```
Gom chung với 9 fields hệ thống (thành 10 nếu tách riêng). HOẶC document `_source_ts` ≠ "system default field" mà là "business-extracted anchored field".

### 7.2 `_gpay_source_id` missing — anchor cho UPSERT

Shadow table có `UNIQUE INDEX (_gpay_source_id) WHERE NOT _gpay_deleted` (v7.1 Section 2). Nếu không extract `_gpay_source_id` từ `after._id` (Mongo ObjectID) → mỗi message = insert row mới = duplicate. **Fix**:
```go
"_gpay_source_id": extractMongoID(businessData),  // from after._id or key
```

### 7.3 `rawJSON` scope ambiguous

Section 6.1 step 1 nói "bóc phần `after`" nhưng code `businessData := json.Unmarshal(rawJSON)` treat entire Debezium envelope. `_raw_data = rawJSON` store **full envelope** (before, after, source, op, ts_ms) hay **after only**? Hash computed over which scope?

**Decision needed**:
- **Option A**: `rawJSON = after` only → `_raw_data` = business payload only. Simpler queries.
- **Option B**: `rawJSON = full envelope` → audit trail preserves op/source/ts_ms. More data.

Pick one + document. Hash deterministic theo scope chosen.

### 7.4 `_gpay_*` prefix guard vs v7.1 `_gpay_id` collision

Code line: `if !strings.HasPrefix(key, "_gpay_") { finalRecord[key] = value }`. Nhưng `finalRecord["_gpay_id"]` đã set ở step 3 — businessData có trường nào prefix `_gpay_*` (unlikely from Mongo but possible) sẽ bị skip. Correct behavior. Đánh dấu trong comment: "System fields namespace reserved".

Wider guard recommendation: skip ALL system fields prefix `_` (bao gồm `_id` Mongo, `_hash`, `_version`, etc.) để Mongo `_id` không override Worker-computed `_gpay_id`. Hoặc document mapping `_id` → `_gpay_source_id` explicit.

### 7.5 GORM `Save()` không phải UPSERT

GORM `Save()` với map-based insert mặc định là INSERT, không ON CONFLICT. Re-consume message 2 lần = duplicate rows. **Fix**:
```go
return w.db.Table(msg.TableName).
    Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "_gpay_source_id"}},
        DoUpdates: clause.AssignmentColumns([]string{"_raw_data", "_hash", "_source_ts", "_updated_at", ...}),
        Where: clause.Where{Exprs: []clause.Expression{
            // OCC guard
            clause.Expr{SQL: "EXCLUDED._gpay_source_ts > ?.tableName._gpay_source_ts"},
        }},
    }).
    Create(&finalRecord).Error
```

### 7.6 Fencing session vars missing

Để `tg_fencing_guard` trigger work, Worker phải `SET LOCAL app.fencing_machine_id/token` per transaction. Code hiện không có. **Fix** (wrap in tx):
```go
return w.db.Transaction(func(tx *gorm.DB) error {
    tx.Exec("SET LOCAL app.fencing_machine_id = ?; SET LOCAL app.fencing_token = ?",
        w.machineID, w.fencingToken)
    tx.Exec("SET LOCAL app.fencing_machine_id = ?", w.machineID)
    tx.Exec("SET LOCAL app.fencing_token = ?", w.fencingToken)
    return tx.Table(msg.TableName).Clauses(...).Create(&finalRecord).Error
})
```

### 7.7 `schemaManager.EnsureFields` — ALTER TABLE safety

Auto-ALTER on new fields có 3 risk (lesson #71 whack-a-mole):
1. **Rate limit**: MongoDB schema-less → Debezium push field mới liên tục → ALTER dồn dập. Cap 10 ALTER/table/day.
2. **Type inference**: value is string in msg 1, number in msg 2 → ALTER race. First-seen wins, later mismatch → DLQ.
3. **Financial audit**: financial table (is_financial=true) → block auto-ALTER, require admin approve.

Document scope trong `schemaManager` comment.

---

## 8. Muscle Sequencing (atomic per task)

Brain recommend execution order:

1. **T0.1 + T0.2** (migration 018): Foundation — worker_registry, sequences, claim/heartbeat, `tg_fencing_guard` function (not attached yet). ~4h
2. **T0.3** (migration 019): System Registry table (pending_data tracking). ~1h
3. **Self-test Phase 0**: psql verify claim/heartbeat/fencing edge cases. ~2h
4. **T1.1 SinkWorker skeleton**: HandleMessage function theo 7.x fixes (source_ts, source_id, upsert, fencing). ~6h
5. **T1.2 Field enforcement + SchemaManager**: EnsureFields với rate limit + financial audit. ~6h
6. **T1.3 Shadow auto-migration + trigger attach**: CREATE TABLE template + attach `tg_fencing_guard` mỗi table. ~4h
7. **Verify Phase 1**: ingest sample Debezium message, verify shadow row đủ fields. ~3h
8. **T2.1 Debezium Signal**: trigger incremental snapshot, verify Kafka flood handled. ~3h
9. **T2.2 Backfill parallel test**: 1 table scale test. ~4h
10. **Rollout batch 1**: High-Stake (financial) tables, admin schema review. ~4h
11. **Rollout batch 2**: High-Volume tables, Debezium Signal parallel. ~4h
12. **Rollout batch 3**: Empty/Seed streaming-only. ~2h

Total ~43h (bare v7.2, không có v7.1 typed extraction Phase which was descoped).

---

## 9. User's "Proof of Integrity" requirement

Muscle sau T1.1-T1.3 nộp bằng chứng:
> Một record từ Shadow Table của bảng `payment_bills` có đầy đủ 9 field hệ thống và **cột D nằm trong `_raw_data`**.

Deliverable cụ thể:
```sql
SELECT 
  _gpay_id, _gpay_source_id, _source, _synced_at, _source_ts,
  _version, _hash, _deleted, _created_at, _updated_at,
  _raw_data  -- cột D hoặc bất kỳ field nào extracted từ Mongo inside here
FROM cdc_internal.payment_bills 
LIMIT 1;
```

Row output phải có:
- 9 system fields non-null
- `_raw_data` JSONB chứa `{"_id": ..., "amount": ..., "d": ..., ...}` (cột D preserved inside)
- `_gpay_source_id` = Mongo ObjectID từ `_raw_data._id`
- `_source = 'debezium'`

---

## 10. Parallel Independence Contract

Muscle commit KHÔNG touch:
- ❌ `public.<table>` (Airbyte legacy)
- ❌ `_airbyte_*` metadata columns
- ❌ Airbyte connections / sync schedules
- ❌ SQL bridge worker paths (bridge_batch.go stays as-is cho legacy use)

Muscle CHỈ touch:
- ✅ `cdc_internal.*` schema
- ✅ Debezium Kafka consumer path (`kafka_consumer.go`, `event_handler.go`)
- ✅ New SinkWorker + SchemaManager files
- ✅ Migrations 018+ under `centralized-data-service/migrations/`

---

## 11. Decisions still needed (before Muscle T0.1 execute)

User confirm answers để Brain delegate Muscle:

1. **`_raw_data` scope** (7.3): Option A (after only) hay Option B (full envelope)?
2. **`_source_ts` classification** (7.1): include in 9-field list (thành 10) hay separate as "business-anchored"?
3. **Empty stream handling** (v7.2 Section 5): Airbyte empty tables — defer với `pending_data` là `public.*` (Airbyte) hay `cdc_internal.*` (new shadow)?
4. **Financial admin review tooling**: manual psql inspect OR build CMS UI để admin approve per table schema trước enable Sink?
5. **Debezium current connector**: reuse existing `goopay-mongodb-cdc` connector hay tạo new connector dành cho v1.25 pipeline?

---

## 12. Appendix — lessons applied

- #65 Per-entity band-aid (v7.2 chọn systematic rollout by data characteristic)
- #67 Reconstruction honest (v7.2 IS clean rebuild, không patch legacy)
- #69 Scope-cut hèn nhát (v7.2 không cut — legacy explicitly "để tự bơi", not hidden)
- #70 Proven > novel (Debezium Signal + GORM Map Insert = textbook)
- #71 Whack-a-mole (Section 7 flags 6 gaps TRƯỚC Muscle execute, không discover after fail)
- #72 Distributed primitives (fencing + session vars + Signal incremental snapshot)
- #73 SQL clause scope (BEFORE trigger mandatory, WHERE EXISTS was wrong)

---

## 13. File history

- v1 band-aid → v2 vocab-lie → v3 scope-cut → v4 trigger-hell → v5 queue+regex → v6 missing primitives → v7 distributed primitives → v7.1 fencing+outbox+profiling+ctid → **v7.2 parallel system (current)** — Airbyte public discarded, cdc_internal fresh via Debezium
