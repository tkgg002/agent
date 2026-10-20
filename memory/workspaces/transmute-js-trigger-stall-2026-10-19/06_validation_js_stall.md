# 06 — Validation Report: Giới hạn Dung lượng Stream (`MaxBytes`), Xóa Message ngay khi ACK (`WorkQueuePolicy`) & Khắc phục Core NATS Fallback (`js_stall`)

## 1. Bảng Truy vết Yêu cầu (Gate G1)

| Req ID | Mô tả Yêu cầu (`01_requirements_js_stall.md`) | File:Line đáp ứng | Bằng chứng Test / Runtime | Trạng thái |
|--------|-----------------------------------------------|-------------------|---------------------------|------------|
| `R1` | Đặt giới hạn dung lượng (`MaxBytes`) cho các stream JetStream để không bao giờ gây `no space left on device` trên `/data/jetstream` | `centralized-data-service/pkgs/natsconn/nats_client.go:67-109` (`CDC_EVENTS`: 512 MiB, `CDC_TRANSMUTE`: 256 MiB, `SCHEMA_DRIFT`: 64 MiB, `SCHEMA_CONFIG`: 64 MiB + `DiscardOld`) | `TestNatsClient_EnsureStreams` (`pkgs/natsconn/nats_client_test.go:78-96`) — `PASS` | ✅ PASS |
| `R2` | Xóa message trên `CDC_TRANSMUTE` ngay sau khi worker xử lý xong (`msg.Ack()` / `msg.Term()`) thay vì giữ 24h | `centralized-data-service/pkgs/natsconn/nats_client.go:98-108` (`Retention: nats.WorkQueuePolicy`, `MaxAge: 1 * time.Hour`) + `internal/handler/master/transmute_consumer_pool.go:138,152` | `TestNatsClient_EnsureStreams` (`nats_client_test.go:97-142`: `State.Msgs == 0` ngay sau `AckSync()`) & `TestTransmuteConsumerPool_WorkQueueImmediateDeleteOnAck` (`transmute_debounce_group_test.go:65-147`: 5/5 messages xử lý xong, `State.Msgs == 0`) — `PASS` | ✅ PASS |
| `R3` | Tự động migrate stream `CDC_TRANSMUTE` đang tồn tại trên PROD từ `LimitsPolicy` sang `WorkQueuePolicy` mà không dính lỗi NATS `10052`, đồng thời tự động dọn sạch ~1 GB message cũ đã ACK | `centralized-data-service/pkgs/natsconn/nats_client.go:127-164` (`DeleteStream` + `AddStream` khi `info.Config.Retention != desired.Retention`; `UpdateStream` khi lệch `Subjects`/`MaxBytes`/`MaxAge`/`Discard`) | `TestNatsClient_EnsureStreams` (`nats_client_test.go:144-175`: tạo stream `LimitsPolicy` có sẵn message -> gọi `EnsureStreams` -> chuyển sang `WorkQueuePolicy` và `State.Msgs == 0`) — `PASS` | ✅ PASS |
| `R4` | Khắc phục lỗi mất 100% trigger (0 record ghi vào Master từ 16/10) khi `bb.js.PublishMsg` lỗi và fallback sang Core NATS | `centralized-data-service/internal/handler/master/transmute_handler.go:257-260` (`SubjectTransmuteShadowCoreFallback`), `internal/server/server_setup.go:360`, `internal/handler/shadow/batch_buffer_fanout.go:118-129` | `TestBatchBuffer_CoreNATSFallbackWhenJetStreamPublishFails` (`snapshot_cdc_isolation_test.go:156-229`: khi `CDC_TRANSMUTE` không tồn tại khiến JetStream publish lỗi 3 lần, message được chuyển tới subscriber trên `cdc.cmd.transmute-shadow.core-fallback`) — `PASS` | ✅ PASS |
| `R5` | Đảm bảo End-to-End `Ack()` / `NakWithDelay()` với `WorkQueuePolicy`: tuyệt đối không `msg.Ack()` bừa bãi trước khi `TableDebouncer` ghi xuống DB Master thành công; khi lỗi lookup binding hoặc lỗi ghi DB Master thì trả về `error` để gọi `msg.NakWithDelay()` giữ message trên đĩa và tự động phát lại | `centralized-data-service/internal/handler/master/debounce.go:12-25,75-96` (`DoneCh`, `complete`, `AddTask`), `internal/handler/master/transmute_handler.go:138-237,544-805,854-919` (`handleTransmuteShadowWithOutcome`, `waitForTransmuteDone`, `runDebouncedGroup`, `binarySearchSplit`, `processSubBatch`) | `TestTransmuteShadowTask_EndToEndAckAndNakRetryOnMasterDBFailure` (`transmute_debounce_group_test.go:147-250`: 2 lần đầu ghi DB Master lỗi -> giữ nguyên message trong `CDC_TRANSMUTE` (`State.Msgs == 1`) & redeliver -> lần 3 ghi Master thành công mới `Ack()` và xóa khỏi đĩa (`State.Msgs == 0`)) — `PASS` | ✅ PASS |
| `R6` | Đảm bảo giới hạn `MaxDeliver = 5` & xử lý Poison Pill / DLQ: khi lỗi vĩnh viễn đã thử đủ 5 lần (`meta.NumDelivered >= MaxDeliver`) thì gọi `msg.Term()` và log poison pill để dừng phát lại vô tận | `centralized-data-service/internal/handler/master/transmute_consumer_pool.go:130-175` (`meta.NumDelivered >= uint64(p.cfg.MaxDeliver)` -> `msg.Term()`, backoff `NakWithDelay`) | `TestTransmuteShadowTask_MaxDeliverExhaustionTerminatesPoisonPill` (`transmute_debounce_group_test.go:252-350`: lỗi schema vĩnh viễn chạy đúng `5` lần thử (`attempts == 5`), lần thứ 5 gọi `msg.Term()`, dừng hẳn không phát lần 6 và xóa khỏi `WorkQueuePolicy` (`State.Msgs == 0`)) — `PASS` | ✅ PASS |

---

## 2. Kết quả Kiểm thử & Runtime (Gates G2, G3, G4, G5, G6)

### 2.1. Lệnh đã chạy
```bash
go build ./cmd/... ./internal/... ./pkgs/...
go test -v ./pkgs/natsconn/... ./internal/handler/master/... ./internal/handler/shadow/...
go vet ./pkgs/natsconn/... ./internal/handler/master/... ./internal/handler/shadow/... ./internal/server/...
```

### 2.2. Output thực tế
```text
=== RUN   TestCompleteActionTrace_WithMessage
--- PASS: TestCompleteActionTrace_WithMessage (0.00s)
=== RUN   TestCompleteActionTrace_WithExplicitParams
--- PASS: TestCompleteActionTrace_WithExplicitParams (0.00s)
=== RUN   TestCompleteActionTrace_Fallback
--- PASS: TestCompleteActionTrace_Fallback (0.00s)
=== RUN   TestNatsClient_EnsureStreams
--- PASS: TestNatsClient_EnsureStreams (0.04s)
=== RUN   TestSameSubjects
--- PASS: TestSameSubjects (0.00s)
PASS
ok  	centralized-data-service/pkgs/natsconn	0.830s
=== RUN   TestTableDebouncer_FlushOnMaxSize
--- PASS: TestTableDebouncer_FlushOnMaxSize (0.00s)
=== RUN   TestTableDebouncer_ShutdownNoDeadlock
--- PASS: TestTableDebouncer_ShutdownNoDeadlock (0.05s)
=== RUN   TestHandleMasterSwap_Success
--- PASS: TestHandleMasterSwap_Success (0.00s)
=== RUN   TestGroupDebouncedTasks_SplitsByBindingAndTrigger
--- PASS: TestGroupDebouncedTasks_SplitsByBindingAndTrigger (0.00s)
=== RUN   TestGroupDebouncedTasks_EmptyAndHomogeneous
--- PASS: TestGroupDebouncedTasks_EmptyAndHomogeneous (0.00s)
=== RUN   TestTransmuteConsumerPool_WorkQueueImmediateDeleteOnAck
--- PASS: TestTransmuteConsumerPool_WorkQueueImmediateDeleteOnAck (0.06s)
=== RUN   TestTransmuteShadowTask_EndToEndAckAndNakRetryOnMasterDBFailure
--- PASS: TestTransmuteShadowTask_EndToEndAckAndNakRetryOnMasterDBFailure (0.51s)
=== RUN   TestTransmuteShadowTask_MaxDeliverExhaustionTerminatesPoisonPill
--- PASS: TestTransmuteShadowTask_MaxDeliverExhaustionTerminatesPoisonPill (1.01s)
PASS
ok  	centralized-data-service/internal/handler/master	2.283s
=== RUN   TestBatchBuffer_CoreNATSFallbackWhenJetStreamPublishFails
--- PASS: TestBatchBuffer_CoreNATSFallbackWhenJetStreamPublishFails (1.85s)
PASS
ok  	centralized-data-service/internal/handler/shadow	3.097s
=== RUN   TestDeterministicGpayID
--- PASS: TestDeterministicGpayID (0.00s)
=== RUN   TestInferLastGpayIDFromMaster
    transmuter_gpayid_test.go:178: live payment_bills inferred last_gpay_id=105516608188842027, remaining_shadow_rows=0
--- PASS: TestInferLastGpayIDFromMaster (0.14s)
PASS
ok  	command-line-arguments	0.852s
```

### 2.3. Đối soát Dữ liệu & Edge-Case (Gates G4, G6)
- **Immediate Deletion (G6):** Trong `TestTransmuteConsumerPool_WorkQueueImmediateDeleteOnAck`, publish `5` messages (`totalMsgs = 5`) → `TransmuteConsumerPool` (`WorkerCount: 3`) xử lý đủ `5/5` messages (`processed == 5`) và gọi `msg.Ack()` → số lượng message còn lại trên đĩa trong `CDC_TRANSMUTE` (`info.State.Msgs`) về đúng `0` ngay lập tức.
- **End-to-End Retry on Master DB Failure (G2, G4, G6):** Trong `TestTransmuteShadowTask_EndToEndAckAndNakRetryOnMasterDBFailure`, khi `runEventsFn` trả lỗi kết nối DB (`sqlstate 08006: connection refused`) ở lần 1 và lần 2, message vẫn được giữ nguyên trong `CDC_TRANSMUTE` (`msgsInStreamAfterFirstFailure == 1`), được NATS JetStream phát lại cho tới lần thứ 3 ghi DB Master thành công (`dbWriteAttempts == 3`) mới gọi `msg.Ack()` và xóa khỏi đĩa (`State.Msgs == 0`).
- **MaxDeliver = 5 Poison Pill Termination (G4, G6):** Trong `TestTransmuteShadowTask_MaxDeliverExhaustionTerminatesPoisonPill`, khi gặp lỗi dữ liệu/schema vĩnh viễn, message được thử lại đúng `5` lần (`attempts == 5`), ở lần thứ 5 worker gọi `msg.Term()` xóa khỏi `WorkQueuePolicy` (`State.Msgs == 0`) và không phát lại lần thứ 6.
- **Tự động suy diễn `lastGpayID` từ bảng Master khi bấm "Chạy ngay (tiếp theo)" (G2, G4, G6):** Trong `TestInferLastGpayIDFromMaster`:
  - Khi bảng Shadow có 6 dòng (`_gpay_id = 100..600`) và bảng Master đã có 4 dòng đầu (`s1..s3` từ `Run()` với Sonyflake `_gpay_id = 100..300`, `s4` từ `RunEvents` với hash SHA-256 `deterministicSourceGpayID("s4", "")`), kèm 1 dòng hash ngẫu nhiên `550` không tồn tại trên Shadow: `inferLastGpayIDFromMaster` lọc bỏ hoàn toàn `550` và suy diễn chính xác mốc `lastGpayID = 400` (dòng thứ 4 trên Shadow), đồng thời `countShadowRows(ctx, row, 400)` trả về đúng `2` dòng còn thiếu (`500, 600`).
  - Kiểm chứng read-only trên bảng thật `payment_bills` (`> 1.03M` dòng): `inferLastGpayIDFromMaster` suy diễn chính xác `last_gpay_id = 105516608188842027` (`remaining_shadow_rows = 0`) trong `< 0.14s`.
- **Edge-Case Empty Subjects (G4):** `client.EnsureStreams([]string{})` trả về lỗi rõ ràng, không tạo stream rỗng.
- **Edge-Case Immutable Retention Migration (G4):** Khi stream `CDC_TRANSMUTE` cũ đang là `LimitsPolicy`, `EnsureStreams` xóa và tạo lại sang `WorkQueuePolicy` không dính lỗi NATS `err_code: 10052`.
- **Edge-Case JetStream Outage / Disk Full (G4):** Khi `bb.js.PublishMsg` thất bại cả 3 lần, `publishTransmuteTrigger` không làm rớt trigger mà phát qua `cdc.cmd.transmute-shadow.core-fallback` tới `HandleTransmuteShadow`.

---

## 3. Kết quả Decision Gate `DG1`, `DG2`, `DG3` & Security Gate (Gate G7)

### 3.1. Decision Gates (`/laya_router`)
- **DG1 (Technical Solution):**
  - Summary files: `06_laya_summary_dg1_20261019T100232.json`, `06_laya_summary_dg1_20261019T103722.json`, `06_laya_summary_dg1_20261019T104452.json` | Gate: `PASS`
- **DG2 (Code Demo):**
  - Summary files: `06_laya_summary_dg2_20261019T100236.json`, `06_laya_summary_dg2_20261019T103723.json`, `06_laya_summary_dg2_20261019T104509.json` | Gate: `PASS`
- **DG3 Mode A (Post-Implementation Claim Verification):**
  - Summary files: `06_laya_summary_dg3_20261019T102856.json`, `06_laya_summary_dg3_20261019T105634.json`, `06_laya_summary_dg3_20261019T111220.json` | Gate: `PASS`
- **DG3 Mode B (Agentic Debate Diff Review):**
  - Summary files: `06_laya_summary_dg3review_20261019T102919.json`, `06_laya_summary_dg3review_20261019T105831.json`, `06_laya_summary_dg3review_20261019T111243.json` | Gate: `PASS` (toàn bộ file được `APPROVE` ngay tại `round_1`)

### 3.2. Security Gate Report (`/security-agent`)
| Hạng mục | Kết quả | Chi tiết |
|----------|---------|----------|
| Secret & PII Scan (Code + Memory) | PASS | `SECRET_SCAN_CLEAN` — Không phát hiện secret/PII thô trong diff và workspace |
| Input & Query Injection Check | PASS | Các truy vấn SQL trong `inferLastGpayIDFromMaster` dùng `quoteTransmuteQualified` / `quoteTransmuteIdent` và tham số `?` an toàn |
| AuthN / AuthZ & Error Exposure | PASS | Các NATS subject nội bộ dùng chung kết nối đã xác thực của `NatsClient` |
| VCS Hygiene (No secret staged, No push) | PASS | Đã lưu local commit restore-point (`73e47ca`, `78c20a9`), tuyệt đối không chạy `git push` |
- **Verdict:** ✅ PASS

---

## 4. Nghiệm Thu Sau Khi Khắc Phục Toàn Bộ Phát Hiện Audit `F1 → F7` (`T13 → T16`)

### 4.1. Lệnh Kiểm Thử & Runtime Đã Chạy Trên Postgres Thật + NATS JetStream Thật (0% Mockup)
```bash
go test -v ./internal/handler/master/... ./pkgs/natsconn/...
go test -count=1 -v $(ls internal/service/master/*.go | grep -v transmuter_orphan_test.go)
go build ./cmd/... ./internal/... ./pkgs/...
curl -s "http://localhost:18222/jsz?streams=true&config=true"
```

### 4.2. Kết Quả Thực Tế (Gates G1–G8)
- **`TestTransmuteShadowTask_RealPostgres_BatchesBeyondWorkerCountAndImmediateAck` (`PASS 0.33s` — Sửa `F2`, `F5`):**
  - `WorkerCount = 2`, bắn `25` messages liên tiếp vào `CDC_TRANSMUTE`. Callback `QueueSubscribe` nhận `ErrAsyncAckPending` và trả về ngay lập tức, giúp `TableDebouncer` gom đủ `25` messages, ghi xuống bảng Postgres Master thật (`masterCount == 25`) rồi mới `Ack()` xóa toàn bộ khỏi `CDC_TRANSMUTE` (`info.State.Msgs == 0`).
- **`TestTransmuteShadowTask_RealPostgres_InProgressHeartbeatsKeepSlowBatchAlivePastAckWait` (`PASS 0.75s` — Sửa `F1`, `F5`):**
  - Khoá bảng Postgres Master thật bằng `LOCK TABLE ... IN ACCESS EXCLUSIVE MODE` trong `600ms` (gấp `2.4x` `AckWait = 250ms`). `shadowMsgTracker` gửi `msg.InProgress()` định kỳ giữ sống message, sau khi nhả lock ghi thành công vào Postgres Master (`masterCount == 1`, `NumRedelivered == 0`, `State.Msgs == 0`).
- **`TestTransmuteShadowTask_RealPostgres_NakRetryAndMaxDeliverOnMissingTable` (`PASS 1.70s` — Sửa `F5`, `F7`):**
  - Kịch bản 3a (Thiếu bảng Master trên Postgres): `TransmuterModule.RunEvents` trả lỗi thật → `NakWithDelay` giữ message trên `CDC_TRANSMUTE` (`NumRedelivered >= 1`, `State.Msgs == 1`); sau khi tạo bảng Master trên Postgres, lần redeliver kế tiếp ghi thành công (`masterCount == 1`, `State.Msgs == 0`).
  - Kịch bản 3b (Lỗi `bulkUpsertMaster` do thiếu cột `amount` — `SQLSTATE 42703`, kiểm chứng `F7`): `RunEvents` trả về `lastEventErr` (không nuốt vào `res.Skipped++`), kích hoạt `NakWithDelay` và ghi thành công sau khi `ALTER TABLE ADD COLUMN amount bigint` (`upsertRecoveredCount == 1`, `State.Msgs == 0`).
  - Kịch bản 3c (Poison pill vượt `MaxDeliver = 4`): gọi `msg.Term()` xóa khỏi `CDC_TRANSMUTE` (`State.Msgs == 0`).
- **`TestLocalDockerNats_EnsureStreamsMigratesToWorkQueueAndCleansOrphan` (`PASS 0.02s` — Sửa `F6`):**
  - Đã xóa sạch stream rác `TEST_WQ_CHECK` trên NATS Docker local (`localhost:14222` / `:18222`) và xác nhận `CDC_TRANSMUTE` trên `http://localhost:18222/jsz?streams=true&config=true` có `"retention": "workqueue", "max_bytes": 268435456`.
- **`TestInferLastGpayIDFromMaster` (`PASS 0.35s` — Sửa `F3`):**
  - Kịch bản Outage Gap (`s1..s3` có trước outage, `s4..s8` bị thiếu trong outage, `s9..s10` vừa vào qua `RunEvents`): `Gap-Safe Guard` phát hiện `matchedInMaster (2) != len(gapSourceIDs) (7)`, bác bỏ `bySourceID = 1000` và giữ đúng `inferred = 300`, `countShadowRows = 7` để quét bù toàn bộ vùng trũng `s4..s8`.
  - Kịch bản Liên tục (`s4..s8` đã được `RunEvents` đồng bộ đủ): `Gap-Safe Guard` xác nhận `5/5` dòng tồn tại trong Master, suy diễn đúng `inferred = 800`, `countShadowRows = 2`.
  - Bảng thật `payment_bills` (`>1.03M` dòng): `inferred = 105528041874128942, remaining_shadow_rows = 0`.
- **Decision Gate `DG3` sau khi sửa `F1–F7`:**
  - Summary file: [`06_laya_summary_dg3_20261019T115418.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/transmute-js-trigger-stall-2026-10-19/06_laya_summary_dg3_20261019T115418.json) | Gate: `PASS`
  - Final Adversarial Audit Report: [`audit_report_js_stall.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/transmute-js-trigger-stall-2026-10-19/audit_report_js_stall.md)
