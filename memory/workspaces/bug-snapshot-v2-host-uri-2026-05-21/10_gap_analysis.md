# 10_gap_analysis — Gap giữa kiến trúc hiện tại và yêu cầu Strong Eventual Consistency

> **Phase**: `lww_guard`
> **Method**: Evidence-driven audit dựa trên 2 lần read trực tiếp source code (chat history) + cross-check 2 subagent report.

## Bảng gap matrix

| Layer | Yêu cầu | Hiện trạng | Gap | Severity |
|---|---|---|---|---|
| **Shadow V1 schema** | có column `_source_ts BIGINT` để OCC guard active | `cdcCols` (`schema_adapter.go:195-204`) **KHÔNG có** `_source_ts`; DDL `createShadowTableV1WithCols:314-325` cũng không | Column thiếu → `hasSourceTs=false` (`schema_adapter.go:408-411`) → guard skip → fall hash dedup → **không có thứ tự thời gian** | 🔴 P0 |
| **Shadow V2 schema** | tương tự | sinkworker package có `_source_ts` (audit subagent #2) | Đã có — chỉ áp dụng cho path V2 sinkworker, KHÔNG cho V1 path | 🟢 OK |
| **Snapshot envelope ts** | dùng oplog clusterTime của Mongo | `buildSnapshotEnvelope:498-511` dùng `now.UnixMilli()` (wall clock worker) | Wall clock có thể > oplog ts thật do clock skew / latency → snapshot clobber realtime | 🔴 P0 |
| **OCC guard ts** | có guard `<` (strict) + tiebreaker | `schema_adapter.go:514-518` dùng `<=` (cho phép bằng) | Cùng ms → record đến sau thắng vô điều kiện → non-deterministic | 🟠 P1 |
| **`_source` discriminator** | phân hoá snapshot vs realtime | Default `'airbyte'`, runtime set `'debezium-v125'` / `'cdc-legacy'` / `'debezium-transmute'`. Snapshot envelope set `source='snapshot:v2'` ở envelope-level, NHƯNG `_source` column ở write path chưa verify có nhận giá trị này | Có thể snapshot path đang set `_source='debezium-v125'` (giống realtime) → tiebreaker fail | 🟠 P1 (cần audit M4) |
| **`_version` Lamport** | monotonic logical clock | V1 update path: `tbl._version + 1` (`schema_adapter.go:490`); V2 sinkworker hardcode `1` | Không phải Lamport cross-process, không dùng làm guard | 🟡 P2 (không scope phase này) |
| **`_synced_at` / `_updated_at` / `_created_at`** | reflect oplog ts | Tất cả là `NOW()` của PG / `time.Now()` của worker | Wall clock, không phải oplog → không dùng được làm LWW | 🟡 P2 (info only) |
| **Sonyflake `_gpay_id`** | monotonic theo time | Sonyflake = 10ms epoch + machine ID, monotonic per worker | Multi-worker → race; không phản ánh oplog | 🟡 P2 (info only) |
| **Snapshot progress watermark** | lưu Mongo clusterTime để recovery | Migration 058 chỉ có `last_seen_id` (Mongo `_id` cursor), không có clusterTime | Không thể resume snapshot at consistent point; audit khó | 🟠 P1 (fix qua migration 061) |
| **Pipeline routing** | snapshot v2 và realtime cùng pipeline (HandleRaw) | ✅ Đã cùng pipeline `EventHandler.HandleRaw` (line 59) | — | 🟢 OK (Path B pattern correct) |

## Phân loại theo phase fix

### Trong scope phase `lww_guard`

| Gap | Task | Solution ref |
|---|---|---|
| Column `_source_ts` thiếu V1 | M1 (T1.2, T1.3) + M2 | Edit #1, #2 + Migration 060 |
| Snapshot dùng wall clock | M3 | Edit #4, #5 + Migration 061 |
| Guard `<=` không tiebreak | M1 (T1.4) | Edit #3 |
| `_source` discriminator chưa verify | M4 | Audit + Edit #6 |
| Snapshot progress watermark | M3 (T3.5) | Migration 061 |

### Ngoài scope (defer)

| Gap | Lý do defer | Phase đề xuất |
|---|---|---|
| `_version` Lamport | Hack-y, vi phạm SRP. ADR-001 đã reject phương án B | — |
| `_synced_at`/`_updated_at` semantic | Info only, không phục vụ LWW | — |
| Sonyflake multi-worker | Không liên quan LWW | — |
| V2 sinkworker buildUpsertSQLSnapshot DO NOTHING | Path khác, không trong scope. Có thể audit phase sau | future phase |
| Config reload storm 72 events/giây | Liên quan FE debounce + worker throttle, không liên quan LWW | future phase |
| Topic typo `centrallized` | Cosmetic, không ảnh hưởng correctness | future phase |
| Dual-tree `cdc-system/` archive | Governance task, không liên quan code change | future phase |

## Architectural debt liên quan

1. **V1 vs V2 schema divergence**: V1 dùng `cdcCols` map (8 field + sonyflake), V2 sinkworker dùng `schema_manager.go` (khác set field). 2 schema generator song song → maintenance hell. Phase `lww_guard` chỉ unify field `_source_ts` — đầy đủ unify cần dedicated refactor phase.

2. **`_deleted` vs `_gpay_deleted`**: V1 dùng `_deleted BOOLEAN`, V2 dùng `_gpay_deleted`. 2 field cùng semantic, khác tên. Snapshot path V2 không populate `_deleted` → bảng có cả 2 col sẽ có 1 col luôn NULL. Phase sau cần consolidate.

3. **`_source` column quá ngắn** (VARCHAR(20)). Giá trị mới `'snapshot:v2'` (11 chars) fit, nhưng `'snapshot:v2:retry-1'` (19 chars) gần ngưỡng. Phase sau có thể nới VARCHAR(64).

## Verification baseline (trước fix)

Để compare trước/sau:

```sql
-- 1. Bảng nào hiện đã có _source_ts (V2 path)?
SELECT n.nspname, c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
JOIN pg_attribute a ON a.attrelid=c.oid
WHERE c.relkind='r'
  AND (n.nspname='cdc_internal' OR n.nspname LIKE 'shadow_%')
  AND a.attname='_source_ts'
ORDER BY 1,2;
-- Expect baseline: <count> tables (chỉ V2-managed).

-- 2. Distribution của _source values hiện tại
SELECT _source, COUNT(*)
FROM cdc_internal.<sample_shadow_table>
GROUP BY _source;
-- Expect baseline: 'debezium-v125' chiếm phần lớn; 'snapshot:v2' = 0 (chưa có).

-- 3. snapshot_progress columns hiện tại
\d cdc_system.snapshot_progress
-- Expect baseline: KHÔNG có mongo_cluster_time_start_ms.
```

Sau fix, expect:
- Mọi cdc_internal.* + shadow_*.* table có `_source_ts` column.
- Distribution `_source` xuất hiện `snapshot:v2`.
- `snapshot_progress` có 2 column mới.
