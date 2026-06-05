# 02 — Plan: Path B — Custom Snapshot Worker

## Kiến trúc

```
CMS (POST /api/sources/:id/snapshot.v2)
  └─ bus.Dispatch(SnapshotV2Command{source_object_id, trace_id, batch_size})
       └─ NATS publish "cdc.cmd.snapshot.v2"  (header: Cdc-Job-Id, Cdc-Correlation-Id)

centralized-data-service worker:
  cdc.cmd.snapshot.v2 → SnapshotRunner.Handle(msg)
    ├─ jobLog.Open(trace_id, source_object_id)
    ├─ Repo: source_object_registry.GetByID
    ├─ Repo: connection_registry.GetByID → ConnectionCode + EngineType
    ├─ MetadataRegistryService.GetSourceDSN(ConnectionCode) → mongo URI
    ├─ mongo.Connect(URI, readPreference=secondaryPreferred)
    │  defer mongo.Disconnect
    ├─ Resume: SELECT last_seen_id FROM snapshot_progress WHERE source_object_id=?
    │  (NULL = first run)
    ├─ cursor := coll.Find(filter={_id: {$gt: last_seen_id}}, sort={_id:1}, batch=N)
    │  (KHÔNG insert / update / dropIndexes / createCollection — Find ONLY)
    └─ for batch in cursor:
         for each doc:
           build CDCEvent {
             specversion:"1.0", source:"snapshot:v2",
             data:{ op:"c", after:doc, source_ts_ms:nowMs }
           }
           subject := "cdc.snapshot.v2."+sourceDB+"."+sourceColl
           eventHandler.HandleRaw(ctx, subject, json)   // reuse pipeline
         UPDATE snapshot_progress SET last_seen_id=batchTail._id, rows_processed+=len(batch), updated_at=NOW()
         on err: UPDATE snapshot_progress SET status='error', error_msg=err.Error(), updated_at=NOW(); return

      on cursor exhaust:
         UPDATE snapshot_progress SET status='done', finished_at=NOW()
```

## Reuse map (đã verify đọc code)

| Component | Reuse | Tham chiếu |
|---|---|---|
| Mongo client | `mongoClientShared` đã có ở `worker_server.go:181` nhưng URI cố định MONGODB_URL | dùng riêng client cho snapshot DSN |
| Source URI resolver | `MetadataRegistryService.GetSourceDSN` | `metadata_registry_service.go:341` |
| Registry repos | `SourceObjectRegistryRepo.GetByID`, `ConnectionRegistryRepo.GetByID` | đã sẵn |
| Apply pipeline | `EventHandler.HandleRaw(ctx, subject, []byte)` | `event_handler.go:59` — subject parse `cdc.<owner>.<srcDB>.<coll>` ở line 222-234 → cần subject 4-part để extract đúng (db, table) |
| CDCEvent shape | `model.CDCEvent` + `CDCEventData{Op, After, SourceTsMs}` | `model/cdc_event.go` |
| Shadow upsert | `BatchBuffer.WriteRecordSync` đã được `HandleRaw` gọi | tự động |
| Trace propagation | NATS headers + ctx via `WithMetadata` (CMS bus) | `nats_command_bus.go` |

## Quyết định binary

### D1: Direct invoke `HandleRaw` vs publish to Kafka
- **Chọn**: Direct invoke. Lý do: tránh round-trip Kafka + serialize/deserialize,
  giảm latency, không cần Producer trên worker, không thay đổi Kafka topic policy.
  Trade-off: snapshot runner và Kafka consumer dùng chung BatchBuffer → cùng
  shadow DB → ok. Failure semantics: snapshot loop trả error → snapshot_progress
  ghi error_msg, không retry trong loop. CMS truy progress thấy `status='error'`
  → user re-trigger.
- **Loại**: Publish Kafka — tăng phức tạp, KHÔNG thêm benefit (snapshot không
  cần ordering guarantee mà Kafka cung cấp; resumable đã có nhờ checkpoint).

### D2: Subject format đưa vào HandleRaw
- Phải khớp regex `cdc.<owner>.<srcDB>.<coll>` (4 part) để
  `extractSourceAndTable` trả về (sourceDB, sourceColl) đúng → routes resolve.
- **Chọn**: `cdc.snapshot.v2.<srcDB>.<coll>` (5 part) → parts[2]="v2", parts[3]=srcDB sai.
- **Sửa**: `cdc.snapshot.<srcDB>.<coll>` (4 part) → parts[2]=srcDB, parts[3]=coll ✓.

### D3: Mongo readPreference + DB-level checks
- `readPreference=secondaryPreferred` — đi secondary để giảm tải primary.
- Cấm hàm tạo collection mới: dùng `Client.Database(name).Collection(name)`
  → MongoDB tự lazy-init, nhưng tuyệt đối KHÔNG gọi `coll.InsertOne/Many`,
  `coll.UpdateMany`, `coll.Indexes().CreateOne`, hay `db.RunCommand` mutating.
- Defense-in-depth: comment `// READ-ONLY: any mutation here violates CDC golden rule`
  ngay đầu hàm `runSnapshot`.

### D4: Resume key
- **Chọn**: `_id` ObjectId monotonic timestamp prefix. Sort `{_id: 1}`.
- Filter resume: `{_id: {$gt: ObjectIdFromHex(last_seen_id)}}`.
- Bsd cho cả Mongo `_id` ObjectId và custom string `_id` (cmp lexicographic
  string ok với ObjectId hex và uuid). Nếu PK custom là number → cast safely.
  MVP: chấp nhận ObjectId only; non-ObjectId → log warn + reset từ đầu (re-upsert idempotent).

### D5: Batch size
- Default 1000 (Mongo cursor default). CMS có thể override 100-5000.
- Quá lớn → memory spike. Quá nhỏ → checkpoint quá nhiều round-trip PG.

### D6: Concurrency
- 1 snapshot/source_object_id tại 1 thời điểm. NATS subscribe **queue group**
  để khi scale worker nhiều instance, 1 message → 1 worker xử lý.
- Idempotent guard: SELECT snapshot_progress.status WHERE source_object_id=?
  AND status='running' → nếu có row đang chạy < 5 phút → reject (msg ACK).
  > 5 phút coi như zombie → claim mới.

### D7: Trace/audit
- CMS dispatch ghi cdc_jobs row sẵn.
- Worker NATS header `Cdc-Job-Id` + `Cdc-Correlation-Id` → ghi vào
  `snapshot_progress.trace_id`. Đủ truy ngược.
- Log dòng đầu + dòng cuối có `zap.String("trace_id", trace)` để grep.

## Verb dictionary (cho cuộc nói)
- "Path B" = custom snapshot runner bypass Debezium signal.
- "snapshot.v2" = command type / NATS subject hậu tố.
- "snapshot_progress" = control-plane checkpoint table.
- "READ-ONLY guard" = comment + lack of any mutating Mongo call.

## Risks
- R1: Doc rất lớn (>16MB BSON limit?) — Mongo Find ok; HandleRaw parse JSON
  okay nếu doc < ~10MB; lớn hơn sẽ fail json.Unmarshal → snapshot_progress
  ghi error. Acceptable cho MVP (Boss biết trước).
- R2: Snapshot chạy lúc Debezium realtime cũng đang stream → cùng row có thể
  vừa được Debezium upsert vừa được snapshot upsert. UPSERT idempotent
  (`_gpay_source_id` PK) → ok, không duplicate. Order: ai ghi sau win
  (CDC realtime có `_source_ts` newer → OCC guard skip nếu snapshot ghi sau).
- R3: Mongo URI từ secret_ref empty (legacy v1:<name> pointer). `buildDSNFromFields`
  dùng host/port → URI dạng `mongodb://host:port/` không có auth. Nếu Mongo
  bật auth → Find fail. Mitigation: log error rõ, không hard-fail boot.
