# 02 Plan — Disable Debezium Source-DB Write

> **Phase**: `_disable_source_write` (extends `DebeziumSignalKafkaMigration`)
> **Status**: AWAIT BOSS VERB

## Root cause (Re-confirm, 2026-05-21)

Debezium MongoDB connector — bất kể `signal.enabled.channels` là `kafka` thuần hay `source,kafka` — KHI nhận `execute-snapshot` signal sẽ:

1. Đọc trigger từ channel `kafka` (nếu enable kafka).
2. Gọi `MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen` để bắt đầu chunk.
3. `emitWindowOpen` GHI document `{_id: '<uuid>-open', type: 'snapshot-window-open'}` vào collection chỉ ra bởi **`signal.data.collection`**.
4. Sau chunk done → ghi document `'-close'` tương tự.
5. Mục đích: DBLog watermark coordination — phân biệt event nào đến từ live change-stream vs snapshot replay (gap-detection).

Đây KHÔNG phải bug — là **design intent** của Debezium DBLog (Netflix paper). Bất kỳ Debezium version ≥1.7 nào có incremental snapshot đều yêu cầu watermark write vào source connection.

→ **Kết luận**: chừng nào còn dùng `signal.data.collection` + còn trigger `execute-snapshot` signal → Debezium còn ghi source DB. Channel `kafka` chỉ điều khiển nơi NHẬN signal, KHÔNG kiểm soát nơi GHI watermark.

## Tham chiếu lessons.md
- Line 53-59: "Incremental snapshot signal yêu cầu `signal.data.collection`" — thiếu sẽ silent-fail.
- Line 3466-3469: Source DB read-only → mọi pattern ghi (kể cả ghost collection) infeasible.
- Line 3433-3445: Kafka signal key routing fix (đã làm — Bug A đóng).

## Options matrix

| Option | Effort | Source DB write | Incremental snapshot khả dụng | Prod-ready | Risk |
|---|---|---|---|---|---|
| **A. Disable signal config** | 30 min | ❌ STOP | ❌ Mất | ✅ | Low: FE snapshot button thành no-op nếu Boss không patch FE state |
| **B. Custom snapshot worker** | 1-2 ngày | ❌ STOP | ✅ (qua worker, bypass Debezium) | ✅ | Medium: cần envelope schema tương thích + checkpoint table |
| C. Ghost collection trên source | — | ✅ ghi 1 lần | ✅ | ❌ (đã loại — lesson 3466) | High |
| D. Bump Debezium 2.7+ | 4h | ✅ vẫn ghi | ✅ | ❌ | Đã thử fail |
| E. PUT `signal.data.collection=null` thủ công | 5 min | ❌ STOP | ❌ | ❌ (cheat config, FE/BE vẫn inject lại) | High: drift |

## Recommendation

**Path đề xuất: A trước (immediate), B làm follow-up**.

- A giải quyết NGAY complaint của Boss (stop ghi source DB), accept tạm thời mất snapshot button.
- B là long-term fix khôi phục snapshot bằng worker tự control — KHÔNG đụng source.

## Path A — Concrete code demo

### A.1. BE — `cdc-cms-service/internal/api/system_connectors_handler.go`

**Trước** (line 438-461, hiện hành):
```go
func (h *SystemConnectorsHandler) injectDebeziumSignalDefaults(connectorName string, cfg map[string]string) {
    if h.signalBootstrap == "" {
        return
    }
    if !strings.HasPrefix(cfg["connector.class"], "io.debezium.") {
        return
    }
    overrides := map[string]string{
        "signal.enabled.channels":        "source,kafka",         // ← chứa "source"
        "signal.kafka.topic":             h.signalTopic,
        "signal.kafka.bootstrap.servers": h.signalBootstrap,
        "signal.kafka.group.id":          "debezium-signal-" + connectorName,
    }
    // …
}
```

**Sau** (proposal):
```go
func (h *SystemConnectorsHandler) injectDebeziumSignalDefaults(connectorName string, cfg map[string]string) {
    if h.signalBootstrap == "" {
        return
    }
    if !strings.HasPrefix(cfg["connector.class"], "io.debezium.") {
        return
    }
    overrides := map[string]string{
        "signal.enabled.channels":        "kafka",                // ← chỉ kafka, bỏ source
        "signal.kafka.topic":             h.signalTopic,
        "signal.kafka.bootstrap.servers": h.signalBootstrap,
        "signal.kafka.group.id":          "debezium-signal-" + connectorName,
    }
    // Force-DELETE signal.data.collection nếu FE/operator có lỡ gửi —
    // Debezium chỉ ghi watermark vào source nếu key này set. Bỏ key này
    // = không còn ghi source, đổi lại incremental snapshot Debezium
    // silent-fail (đã có lesson line 53-59; FE phải patch trạng thái).
    delete(cfg, "signal.data.collection")

    for k, v := range overrides {
        if old, set := cfg[k]; set && old != v {
            h.logger.Warn("overwriting signal.* key on debezium connector",
                zap.String("connector", connectorName),
                zap.String("key", k),
                zap.String("from", old),
                zap.String("to", v))
        }
        cfg[k] = v
    }
}
```

### A.2. FE — `cdc-cms-web/src/pages/SourceConnectors.tsx`

3 chỗ inject `signal.data.collection`: line 181 (Mongo create), 206 (MySQL create), 235 (PG create). Bỏ key này khỏi cả 3 và đổi `signal.enabled.channels` thành `'kafka'`.

**Diff Mongo (line 180-184)**:
```diff
-      'signal.enabled.channels': 'source,kafka',
-      'signal.data.collection': `${values.database}.debezium_signals`,
+      'signal.enabled.channels': 'kafka',
       'signal.kafka.topic': SIGNAL_KAFKA_TOPIC,
       'signal.kafka.bootstrap.servers': KAFKA_BOOTSTRAP_SERVERS,
```

(Tương tự cho MySQL & PG.)

### A.3. Backfill existing 3 connectors trên dev cluster

Áp dụng PUT `/connectors/<name>/config` với key `signal.enabled.channels=kafka` + remove `signal.data.collection`. Nhưng MUSCLE KHÔNG tự làm — Boss verb `apply backfill` cần thiết vì action chạm prod-like cluster (10.200.186.203).

```bash
# Demo (KHÔNG chạy cho đến khi Boss verb apply backfill):
for c in goopay-ps goopay-pbs goopay-ces; do
  CFG=$(curl -s "http://10.200.186.203:8083/connectors/$c/config" \
    | python3 -c "import json,sys; d=json.load(sys.stdin); d.pop('signal.data.collection',None); d['signal.enabled.channels']='kafka'; print(json.dumps(d))")
  curl -X PUT -H 'Content-Type: application/json' \
    "http://10.200.186.203:8083/connectors/$c/config" -d "$CFG"
done
```

### A.4. FE patch: snapshot button → "Not available"

Tìm component nút "Snapshot" hiện tại. Disable + tooltip "Snapshot via Debezium signal disabled — pending custom worker (Path B)".

(Cần survey thêm file FE — sẽ làm khi Boss verb `ship path A`.)

## Path B — Architecture (long-term, document only)

Đã viết trong `report_2026-05-20_debezium-bump-27-manual.md` line 84-113. Tóm tắt:

```
[UI snapshot button] → CMS → NATS cdc.cmd.snapshot
                                   ↓
   [cdc-worker:SnapshotRunner (NEW)]
      ↓ MongoClient read-only credential cho source
      ↓ cursor: find({_id:{$gt:lastSeenId}}).sort({_id:1}).limit(batchSize)
      ↓ transform → publish Kafka cdc.<topic-prefix>.<db>.<coll>
      ↓ checkpoint vào PG control-plane cdc_system.snapshot_progress
   [existing shadow-apply consumer — reuse]
```

Pros: không đụng source. Reuse downstream pipeline. Resume-able.
Cons: phải implement envelope tương thích Debezium + migration `snapshot_progress` table.
Estimate: 1-2 ngày Muscle work.

## Verb dictionary

| Verb | Action |
|---|---|
| `ship path A` | Muscle apply A.1 + A.2 + build + test + commit (KHÔNG backfill). |
| `apply backfill` | Sau ship A, PUT 3 connector trên 10.200.186.203 bỏ source channel. |
| `start path B` | Muscle plan B chi tiết hơn (tasks + migration + worker file layout). |
| `defer` | Document only, không sửa code. |

## Files dự kiến chạm (path A)

- `cdc-cms-service/internal/api/system_connectors_handler.go` (1 file, 2 edits)
- `cdc-cms-web/src/pages/SourceConnectors.tsx` (1 file, 3 spots)
- `cdc-cms-service/internal/api/system_connectors_handler_test.go` (verify mock expectations nếu có) — kiểm tra trước khi build.

## Verification plan (sau khi ship)

1. `cd cdc-cms-service && go build ./... && go test ./...` → PASS.
2. `cd cdc-cms-web && npx tsc --noEmit` → PASS.
3. Restart cmsapi (binary mới).
4. Re-create 1 connector dev qua FE → verify `signal.enabled.channels=kafka` only + key `signal.data.collection` absent.
5. Trigger snapshot button → verify FE báo lỗi rõ ràng (không silent), Mongo `db.debezium_signals.countDocuments()` KHÔNG tăng so với count trước trigger.
6. Verify live CDC streaming vẫn nhận event (insert 1 doc vào source → confirm Kafka topic `cdc.goopay.*` có message).
