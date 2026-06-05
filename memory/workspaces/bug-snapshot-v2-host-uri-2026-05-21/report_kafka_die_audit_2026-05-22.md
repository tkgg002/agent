# Report — Audit: "Kafka die giữa snapshot.v2 6tr row (đã chạy >1tr)"

> **Date**: 2026-05-22
> **Author**: Brain (claude-opus-4-7) — audit-only, KHÔNG sửa code/DB/config
> **Workspace**: `bug-snapshot-v2-host-uri-2026-05-21` (workspace ACTIVE liên quan snapshot.v2)
> **Scope**: Trả lời câu hỏi "Snapshot đang chạy ~1tr/6tr row → Kafka die → tiến trình ra sao?"
> **Loại**: Read-only audit (đọc source + migration + lessons), không thay đổi DB, không thay đổi config, không sửa code.

---

## 1. TL;DR (Kết luận nhanh)

**Snapshot.v2 (Path B) KHÔNG phụ thuộc Kafka.** Khi Kafka die, snapshot vẫn tiếp tục chạy bình thường từ ~1tr → 6tr row. Tiến trình KHÔNG bị dừng do Kafka outage.

| Câu hỏi | Trả lời |
|---|---|
| Snapshot có dừng khi Kafka die không? | ❌ KHÔNG dừng. Hoàn toàn độc lập Kafka. |
| Có mất dữ liệu đã ghi (1tr row) không? | ❌ KHÔNG. PG shadow upsert đã commit. |
| Cần redispatch không? | ❌ KHÔNG. Trừ khi worker process cũng die kèm theo. |
| Có rủi ro real-time CDC gap không? | ✅ CÓ — nhưng nằm NGOÀI scope snapshot. |
| Có cần action ngay không? | ✅ Kiểm tra worker process còn alive; theo dõi `snapshot_progress.updated_at` advance. |

---

## 2. Bằng chứng kỹ thuật (cross-verify source)

### 2.1 Snapshot.v2 pipeline KHÔNG chạm Kafka

File: `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go`

| Bước | Hành động | Phụ thuộc |
|---|---|---|
| L97-L143 `Handle` | Nhận NATS message subject `cdc.cmd.snapshot.v2`, queue group `cdc-snapshot-runner`, detach goroutine | **NATS** (1 lần khi trigger) |
| L165-L216 | Resolve source_object + connection, lấy Mongo URI | **PG cdc-metadata** |
| L218-L242 `claimProgress` | INSERT/UPDATE `cdc_system.snapshot_progress`, idempotent claim, zombie reclaim sau 10 min | **PG cdc-metadata** |
| L245-L257 `mongo.Connect` | Kết nối Mongo source (SecondaryPreferred) | **Mongo source** |
| L264-L325 cursor loop | `coll.Find()` per batch (default 5000, max 10000), `_id > last_seen` filter, SORT `_id ASC` | **Mongo source** |
| L295-L309 per-doc | `buildSnapshotEnvelope` → `eventHandler.HandleRaw` | _local function call_ |
| L312-L318 checkpoint | UPDATE `snapshot_progress` SET `last_seen_id`, `rows_processed`, `updated_at` | **PG cdc-metadata** |
| L327-L330 `markProgressDone` | UPDATE status='done' | **PG cdc-metadata** |

**Grep verify (không có Kafka producer/writer)**:
```
$ grep -n "kafkaProducer\|sarama.Producer\|kafka.Writer\|writer\.\|signalClient\.Publish" \
    snapshot_runner_handler.go event_handler.go batch_buffer.go
→ ZERO match
```

### 2.2 Downstream chain (HandleRaw → PG)

File: `internal/handler/event_handler.go:59` → `processEvent` (L77-L161):
- `dynamicMapper.MapData` — in-memory transform, no I/O ngoài đọc cache registry.
- `batchBuffer.WriteRecordSync(record)` — **sync upsert PG** (xem `batch_buffer.go:73-111`).

File: `internal/handler/batch_buffer.go:99-110` `WriteRecordSync`:
```go
res := db.Exec(query, values...)   // ← PG, không Kafka
```

Comment chính thức tại L69-L72:
> "It deliberately does NOT write into failed_sync_logs — KafkaConsumer.writeDLQ is the single owner of DLQ persistence so we don't end up with duplicate rows for the same failure."

→ Snapshot path không produce Kafka, không tạo DLQ qua Kafka, không Add() vào async batch buffer (chỉ realtime kafka consumer dùng `Add`/`flush`).

### 2.3 Migration schema — checkpoint table

File: `cdc-cms-service/migrations/schema/core/058_v1_snapshot_progress.sql`:
- Table `cdc_system.snapshot_progress` (PG cdc-metadata 5433).
- Columns: `id, source_object_id, status, last_seen_id, rows_processed, trace_id, error_msg, started_at, updated_at, finished_at`.
- CHECK status IN (running, done, error, cancelled).
- Index `(source_object_id, status, started_at DESC)`.

**Checkpoint không qua Kafka**: trực tiếp PG UPDATE.

---

## 3. Phân tích kịch bản chi tiết "Kafka die ở ~1tr/6tr"

### 3.1 Case A — Worker process VẪN còn alive (Kafka die độc lập)

**Hành vi**:
1. Cursor loop tiếp tục: Mongo `coll.Find(_id > last_seen)` → trả về 5000 docs (batch_size) mỗi vòng.
2. Per-doc: `HandleRaw` → `WriteRecordSync` upsert PG shadow → no Kafka call.
3. Checkpoint `UPDATE snapshot_progress SET last_seen_id, rows_processed, updated_at = NOW()` mỗi cuối batch.
4. Loop kết thúc khi `len(batch) < batch_size`.

**Kết quả**: ~1tr → 6tr row tiếp tục bình thường. **Không có downtime.**

**Side-effect Kafka die LÀ riêng biệt** (không cản snapshot):
- `kafka_consumer.go` (real-time stream từ Debezium → Kafka) bị stall consumer group; oplog không apply.
- `DebeziumSignalClient.Publish` (Path A) fail; nhưng Path B snapshot không gọi method này.
- `ReconHealer` heal qua Kafka signal cũng fail — nhưng đây là tier-3 recon, chạy trên cron riêng, không ảnh hưởng snapshot v2.

### 3.2 Case B — Worker process die kèm Kafka (vd container restart vì healthcheck)

**Hành vi**:
1. Goroutine snapshot bị giết NGAY giữa batch (giả sử batch đang xử lý 1500/5000 docs).
2. `snapshot_progress.status` còn 'running', `last_seen_id` ở giá trị batch trước (chưa update batch hiện tại).
3. Worker khởi động lại → NATS subject `cdc.cmd.snapshot.v2` cần được dispatch LẠI để runner Handle().
4. `claimProgress` thấy row `running` + `time.Since(updated_at) >= 10min` → reclaim zombie:
   ```sql
   UPDATE cdc_system.snapshot_progress
   SET status='running', trace_id=?, error_msg=NULL,
       updated_at=NOW(), finished_at=NULL, started_at=NOW()
   WHERE id = ?
   ```
5. Cursor resume từ `_id > last_seen_id` cũ → re-process các doc của batch dở dang.
6. ON CONFLICT (pk) DO UPDATE — idempotent với cùng hash, không double-insert.

**Kết quả**: Tiến độ resume ổn, chỉ lặp lại ≤ 5000 doc cuối (1 batch). Tổng thời gian thêm: 10 min zombie + 1 batch.

**⚠️ Lưu ý sharp**:
- Nếu worker restart trong khoảng <10 min, claim mới sẽ thấy `existing.UpdatedAt < 10min` → **acquired=false**, skip. Snapshot bị KẸT cho đến khi qua 10 min hoặc operator can thiệp.
- File `snapshot_runner_handler.go:56` định nghĩa `snapshotV2ZombieAfter = 10 * time.Minute` — hằng số cố định.

### 3.3 Case C — PG cdc-metadata (5433) die

Đây là single point of failure THẬT SỰ (không phải Kafka). Nếu PG cdc-metadata die:
- `claimProgress`, `checkpoint`, `markProgressDone` đều fail → snapshot return error sớm.
- `markProgressError` cũng fail → status có thể kẹt 'running'.

Khác Kafka die: PG die thực sự dừng snapshot. Nhưng câu hỏi user là Kafka die nên không trong scope.

### 3.4 Cursor resume safety (idempotency check)

Cursor query: `_id > last_seen_id`, sort `_id ASC`.
- Mongo `ObjectId` monotonic per machine (timestamp + counter) → KHÔNG có doc OID mới insert nằm "trước" cursor sau khi cursor đã pass.
- Trừ trường hợp source dùng custom `_id` (string/int sequence) — có thể có gap.
- `extractDocID` (L470) handle ObjectId hex + string + int → resume OK theo `_id`.
- Write-then-checkpoint order:
  ```
  for doc in batch:
      WriteRecordSync(doc)    ← PG commit immediate
  checkpoint(last_seen_id)    ← PG UPDATE
  ```
  → Crash GIỮA loop: doc đã ghi PG, checkpoint chưa update → resume replay → ON CONFLICT idempotent.
- Crash NGAY SAU checkpoint, TRƯỚC batch tiếp theo: resume bắt đầu từ doc mới → no duplicate.

---

## 4. Rủi ro phụ phát hiện trong audit (KHÔNG fix trong báo cáo này)

| # | Rủi ro | Mức | Trigger | Tác động |
|---|---|---|---|---|
| R1 | `claimProgress` không dùng `SELECT ... FOR UPDATE` | LOW | 2 worker cùng nhận NATS message zombie | NATS queue group đã dedup 1 consumer → mitigation tự nhiên, vẫn nên thêm FOR UPDATE để defense-in-depth |
| R2 | Zombie threshold 10 min cố định | LOW | Mongo cursor chậm bất thường | Batch 5000 doc thường <2s. 10 min đủ rộng. |
| R3 | `_source_ts` NULL trong snapshot envelope (BUG `lww_guard`) | MEDIUM | Snapshot ghi sau real-time CDC chạm cùng PK | OCC guard hiện `<=` → snapshot có thể overwrite ghi mới hơn. **Đã có plan `lww_guard` (workspace này) — chờ user verb `execute lww_guard`.** |
| R4 | Real-time CDC gap khi Kafka die | MEDIUM-HIGH | Kafka outage kéo dài | Doc mới insert vào source sau snapshot start, OID > last_seen → snapshot có thể pick. OID < last_seen (custom PK) → MISS. Cần tier-3 recon catch up. |
| R5 | NATS không persistent (no JetStream) | LOW | Worker crash giữa dispatch | Operator phải redispatch NATS sau worker restart. |

---

## 5. Hành động đề xuất (operator runbook)

### Khi Kafka die giữa snapshot run

**Bước 1 — Verify worker process còn alive**:
```bash
# K8s
kubectl -n data-hub get pods -l app=cdc-worker
# Local
ps aux | grep cmd/worker
```

**Bước 2 — Verify snapshot tiến triển** (chạy trên PG cdc-metadata 5433):
```sql
SELECT id, source_object_id, status, last_seen_id, rows_processed,
       updated_at, NOW() - updated_at AS idle_for
FROM cdc_system.snapshot_progress
WHERE status = 'running'
ORDER BY started_at DESC;
```
- `rows_processed` tăng đều mỗi ~vài giây → snapshot OK.
- `updated_at` không advance >2 min → worker stuck (không phải Kafka).
- `updated_at` không advance >10 min → vào zombie zone.

**Bước 3 — Nếu worker DIE**:
1. Khởi động lại worker (K8s rollout / process restart).
2. Chờ ≥10 min kể từ `updated_at` cuối (hoặc redispatch ngay sau zombie threshold):
   ```bash
   # Redispatch NATS (operator command)
   nats pub cdc.cmd.snapshot.v2 \
     '{"source_object_id":<ID>,"trace_id":"resume-after-kafka-die","batch_size":5000}'
   ```
3. Verify `claimProgress` reclaim zombie → log `snapshot.v2 started ... resume_from=<last_seen>` xuất hiện.

**Bước 4 — Sau khi Kafka phục hồi**:
- Real-time CDC backlog: Kafka Connect tự catch up trong Mongo oplog retention window. Nếu vượt window → cần recon tier-3 healing.
- Verify: `SELECT COUNT(*) FROM cdc_internal.<table>` so sánh với `db.<coll>.estimatedDocumentCount()` Mongo. Diff sẽ về 0 khi Debezium replay xong.

### KHÔNG làm (theo nguyên tắc "không cheat DB"):
- ❌ KHÔNG UPDATE manual `snapshot_progress.status` = 'done' để bypass zombie.
- ❌ KHÔNG TRUNCATE `snapshot_progress` để force re-snapshot — sẽ tạo full 6tr re-read từ Mongo.
- ❌ KHÔNG hard-restart Kafka mid-run nếu worker đang ổn — Kafka die không cản snapshot, chỉ ảnh hưởng real-time stream.

---

## 6. Files đã đọc (không có file thay đổi)

| File | Mục đích |
|---|---|
| `agent/GEMINI.md` | Confirm role Brain + 14 rules |
| `agent/memory/global/project_context.md` | Architecture overview |
| `agent/memory/global/active_plans.md` | Workspace registry |
| `agent/memory/global/tech_stack.md` | Tech stack (Kafka 19092, NATS 14222, PG 5433) |
| `agent/memory/global/lessons.md` lines 3670-3867 | Lessons CDC golden rule, Path B pattern, dual-tree drift |
| `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go` | Snapshot v2 implementation (569 LOC) |
| `data-hub/centralized-data-service/internal/handler/event_handler.go` | HandleRaw + processEvent (293 LOC) |
| `data-hub/centralized-data-service/internal/handler/batch_buffer.go` | WriteRecordSync semantics (L1-L264) |
| `data-hub/centralized-data-service/internal/server/worker_server.go` L400-L520 | NATS subscribe registration |
| `data-hub/cdc-cms-service/migrations/schema/core/058_v1_snapshot_progress.sql` | Checkpoint table schema |
| `agent/memory/workspaces/bug-snapshot-v2-host-uri-2026-05-21/03_implementation_lww_guard.md` | LWW guard plan (chờ execute) |

## 7. Files mới tạo trong audit này

| File | Loại |
|---|---|
| `agent/memory/workspaces/bug-snapshot-v2-host-uri-2026-05-21/report_kafka_die_audit_2026-05-22.md` | **NEW** — báo cáo này |
| `agent/memory/workspaces/bug-snapshot-v2-host-uri-2026-05-21/05_progress.md` | **APPEND** — entry mới timestamp 2026-05-22 |

## 8. Verify gate

| Gate | Trạng thái |
|---|---|
| `go build ./internal/handler/... ./internal/server/...` (data-hub) | ✅ EXIT 0 |
| Grep Kafka producer trong snapshot path | ✅ ZERO match |
| Source code cross-verified với migration 058 | ✅ Khớp |
| Audit-only (no code/DB/config change) | ✅ Confirmed |
| Tuân thủ §11 (memory file append-only) | ✅ Append progress, không overwrite |
| Tuân thủ §12 (Brain không sửa source code) | ✅ Read-only |

---

## 9. Definition of Done

- [x] Đã đọc lessons.md (đặc biệt L-CDC-golden-rule, L-Path-B-pattern)
- [x] Đã đọc GEMINI.md để xác nhận Brain role
- [x] Đã đọc project_context.md, active_plans.md, tech_stack.md
- [x] Đã cross-verify source code thực tế (snapshot_runner_handler.go + event_handler.go + batch_buffer.go)
- [x] Đã verify `go build` PASS
- [x] Đã tạo file report vật lý trong workspace
- [x] Sẽ APPEND `05_progress.md` (không overwrite)
- [x] Không cheat DB, không sửa config, không sửa code
- [x] Báo cáo dựa trên evidence thực tế (file:line) — không láo

[2026-05-22] [Brain:claude-opus-4-7] Audit report written.

---

## 10. Follow-up audit (2026-05-25): "Connector name KHÔNG có dạng `goopay-*` thì có chạy không?"

### Câu hỏi
User: "kiểm tra coi connector name ko có dạng goopayxxx thì có chạy ko"

### Kết luận
✅ **Snapshot.v2 chạy bình thường với BẤT KỲ connector/connection_code không phải `goopay-*`** (ví dụ `mybank-pbs`, `acme_mongo`, `tenant1.cdc`, `prod-source-001`).
⚠️ Realtime Kafka CDC (Debezium → kafka_consumer) PHỤ THUỘC config `kafka.topicPrefix` của worker — nếu connector tạo topic với prefix mới mà worker config chưa include → realtime sẽ MISS (KHÔNG liên quan snapshot.v2).

### Evidence kỹ thuật

**1. Validation `connection_code` ở schema & API:**
- DB constraint `cdc-cms-service/migrations/schema/cdc_system_model/029_v2_connection_registry.sql:30`:
  ```sql
  connection_code VARCHAR(100) NOT NULL UNIQUE
  ```
  → CHỈ unique + length 100, KHÔNG có CHECK regex `goopay.*`.
- API regex `cdc-cms-service/internal/api/system_connectors_handler.go:87`:
  ```go
  var connectorNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,128}$`)
  ```
  → Chấp nhận mọi alphanumeric + `_.-`, KHÔNG ép prefix `goopay`.

**2. Snapshot.v2 KHÔNG hardcode `goopay` ở runtime path:**
- File `snapshot_runner_handler.go` — grep `goopay` → ZERO match.
- Subject build (L262):
  ```go
  subject := fmt.Sprintf("cdc.snapshot.%s.%s", srcDB, srcColl)
  ```
  Dùng `srcDB` (từ `so.SourceDatabase`) + `srcColl` (từ `so.SourceObjectName`), **không động đến `connection_code`**.
- `conn.ConnectionCode` chỉ dùng để:
  - `registrySvc.GetSourceDSN(ctx, conn.ConnectionCode)` — lookup DSN theo unique key (không pattern match) — `snapshot_runner_handler.go:218`.
  - Log + activity_log details (`L243, L271, L684`).

**3. Subject parser `extractSourceAndTable` (event_handler.go:224-236):**
```go
parts := strings.Split(subject, ".")
if len(parts) >= 4 {
    return parts[2], parts[3]
}
```
Parse `cdc.snapshot.<srcDB>.<srcColl>` → parts[2]=srcDB, parts[3]=srcColl. **Không lệ thuộc literal `goopay`.**
⚠️ Comment L225 `// subject format: cdc.goopay.{source_db}.{table_name}` là **documentation cũ outdated** — code thực tế chỉ index theo position parts[2]/[3].

**4. ResolveSourceRoutes (downstream):**
- `event_handler.go:80` — `routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)` — lookup theo `(sourceDB, sourceTable)` trong registry. Nếu source_object đã được register với `source_database` + `source_object_name` đúng → route resolve OK bất kể connection_code dạng gì.

**5. Realtime Kafka path (NGOÀI scope snapshot, nhưng cảnh báo):**
- `kafka_consumer.go:52` config field `TopicPrefix []string` (mapstructure).
- `config/config_test.go:14-16` — config có thể là scalar/list, không hardcode "goopay".
- Worker discovery (`kafka_consumer.go:709 filterMatchingTopics`) — chỉ match topic theo `strings.HasPrefix(topic, pre)` với từng `pre` trong `config.TopicPrefix`.
- ⚠️ Default deployment trong codebase test có example `cdc.goopay`, `cdc.gpay`, `cdc.mariadb` — nếu vận hành cluster mới với connector `mybank` tạo topic `cdc.mybank.*` mà KHÔNG update `kafka.topicPrefix` config worker → topic không được discover. Đây là realtime CDC issue, KHÔNG ảnh hưởng snapshot.v2 (snapshot đọc trực tiếp Mongo, không cần Kafka topic).

### Constraints cần đảm bảo khi đặt connector name mới
| Constraint | Source | Quy tắc |
|---|---|---|
| Length ≤ 100 | DB `029_v2_connection_registry.sql:30` | `VARCHAR(100) NOT NULL UNIQUE` |
| Char set | `system_connectors_handler.go:87` regex | `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,128}$` (length 1-129 ở API, intersect DB → 1-100) |
| Uniqueness | DB UNIQUE constraint | Không trùng bất kỳ connection_code khác |

→ Examples hợp lệ: `mybank-pbs`, `acme_mongo`, `tenant1.cdc`, `prod-source-001`, `cms-mongo-001`.
→ Examples KHÔNG hợp lệ: bắt đầu bằng `-`/`_`/`.`, chứa space/`@`/`/`/`:`/Unicode.

### Files đã đọc trong follow-up này
- `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go` (recheck L218, L262)
- `data-hub/centralized-data-service/internal/handler/event_handler.go:222-236` (subject parser)
- `data-hub/centralized-data-service/internal/handler/kafka_consumer.go:47-731` (TopicPrefix logic)
- `data-hub/cdc-cms-service/internal/api/system_connectors_handler.go:87,200,261,349,359` (regex + topic.prefix fingerprint)
- `data-hub/cdc-cms-service/migrations/schema/cdc_system_model/029_v2_connection_registry.sql:30` (schema constraint)
- `data-hub/centralized-data-service/internal/admin/helpers.go:102-110` (connection_code lookup, no pattern match)

### Files thay đổi
- **APPEND** `report_kafka_die_audit_2026-05-22.md` Section 10 (entry này).
- **APPEND** `05_progress.md` Followup #9.

KHÔNG sửa source code, KHÔNG sửa DB, KHÔNG sửa config. Audit-only.

### Recommend (optional, KHÔNG fix lần này)
- R6 (cosmetic): Update comment `event_handler.go:225` từ `cdc.goopay.{source_db}.{table_name}` → `cdc.{prefix}.{source_db}.{table_name}` để tránh hiểu nhầm "prefix bị fix là goopay". Một dòng comment, không thay đổi behavior.

[2026-05-25] [Brain:claude-opus-4-7] Follow-up audit (connector naming) completed.

---

## §11. Hotfix — `_source` VARCHAR(20) overflow (2026-05-22)

**Severity**: P0 — toàn bộ realtime CDC fail SQLSTATE 22001, 0 row ghi.

**Root cause**: `kafka_consumer.go:585` build envelope với
`"source": "/kafka/" + msg.Topic` (41 ký tự cho topic `cdc.goopay.auth-service.user-auths`).
`event_handler.go:126-129` chuyển nguyên chuỗi này vào `record.Source` → cột `_source VARCHAR(20)` reject.

Đồng thời path-form còn vi phạm LWW tiebreaker (`schema_adapter.go:521-528`)
yêu cầu literal `_source = 'snapshot:v2'`.

**Fix** (1 file, 1 edit):
`kafka_consumer.go:584-595` — `"source": "debezium"` thay vì path.
Subject (msg.Topic) đã đủ cho `extractSourceAndTable`.

**Verification**:
- `go build ./...` → OK
- `go test ./internal/handler/... -run "Kafka|Event"` → PASS
- `grep -rn "/kafka/" --include="*.go"` → empty
- Snapshot.v2 path không đổi (`snapshot_runner_handler.go:657`).

**Tác động lên `lww_guard`**: KHÔNG cần migration ALTER COLUMN VARCHAR(20)→VARCHAR(255)
cho bảng legacy. Cả `debezium` (8 chars) và `snapshot:v2` (11 chars) đều fit
trong VARCHAR(20). Phase `lww_guard` schema generator update vẫn giữ
VARCHAR(255) cho bảng mới để dư dải an toàn.

**Files modified**:
- `data-hub/centralized-data-service/internal/handler/kafka_consumer.go` (+6/-1)

---

## §12. Snapshot.v2 Circuit Breaker (2026-05-22)

**Why this exists**: Followup #10 fix VARCHAR overflow xong, user phát hiện non-strict
snapshot mode (default) sẽ "chạy điên" — every doc fail nhưng vẫn `continue` →
6M rows fail liên tiếp, fill DLQ, ẩn root cause.

**Trip conditions** (any):
- 100 lỗi liên tiếp (consecutive)
- Batch error ratio ≥ 50% với ≥ 10 lỗi

**Khi trip**: Flush DLQ → `snapshot_progress.status='error'` → log ERROR → exit loop → activity_log status='error'.

**Resume**: Operator fix root cause → re-dispatch (Overwrite=false) → resume từ checkpoint cuối.
Nếu chưa fix → CB lại trip trong 100 doc → không bao giờ flood.

**Files**:
- `centralized-data-service/internal/handler/snapshot_runner_handler.go` (+~80 LOC, refactor inner loop sang closures + add constants)

**Verify**: build OK, vet OK, handler tests PASS.

**Operator runbook bổ sung**:
```sql
-- Khi snapshot CB trip, kiểm tra root cause:
SELECT id, source_object_id, status, rows_processed, error_msg, finished_at
FROM cdc_system.snapshot_progress
WHERE status = 'error' ORDER BY finished_at DESC LIMIT 10;

-- Sample DLQ rows để xác định pattern lỗi:
SELECT progress_id, document_id, error_msg, created_at
FROM cdc_system.snapshot_dlq
WHERE progress_id = <ID_FROM_ABOVE>
ORDER BY created_at DESC LIMIT 20;
```
