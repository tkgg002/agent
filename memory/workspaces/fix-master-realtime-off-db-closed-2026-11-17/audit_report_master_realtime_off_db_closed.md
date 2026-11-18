# Audit Report (Tier T2 Consolidated Validation + Report + S5 Audit) — `master_realtime_off_db_closed`

- **Workspace:** `fix-master-realtime-off-db-closed-2026-11-17`
- **Tier:** `T2` | **Mode:** `standard`
- **Ngày hoàn tất:** `2026-11-17`

---

## 0. Bảng Tổng Hợp Bằng Chứng Thực Thi (`06_evidence/`)

Toàn bộ 11 biên lai dưới đây đều được thực thi sau lần sửa code cuối cùng của từng repository và khớp `src_fingerprint` hiện tại trên đĩa (`69eecbe21e22` cho `centralized-data-service`, `de4b6aa80f20` cho `cdc-cms-service`, `a48767a29eae` cho `cdc-cms-web`):

| Biên lai (`EV-...`) | Phân loại (`kind`) | Repo | Lệnh thực thi | Kết quả |
| :--- | :--- | :--- | :--- | :--- |
| `EV-20261117T175118-cca1` | `integration` | `centralized-data-service` | `go test -race -v -count=1 ./internal/handler/master/...` | `PASS` chạy thật trên PostgreSQL 15 + NATS JetStream thật với cờ `-race`, kiểm chứng `1 Shadow -> 2 Masters` tắt lẻ 1 Master, tắt toàn bộ Master khi bảng vật lý bị xóa + xả sạch stream `CDC_TRANSMUTE` về `0` msg, chặn debouncer in-flight, và bảo toàn `bridge-oplog` / `isTransientError` (`EV-20261117T175118-cca1`) |
| `EV-20261117T175147-9096` | `unit` | `centralized-data-service` | `go test -race -v -count=1 -run TestBatchBuffer_FlushPostIngestCacheAndInvalidateConnectionAdapter ./internal/handler/shadow/...` | `PASS` kiểm chứng `FlushPostIngestCache`, `InvalidateConnectionAdapter("postgrestestingpaymentv2")` và trần `5s` của `postIngestEntryOnLookupError` (`EV-20261117T175147-9096`) |
| `EV-20261117T175335-12a1` | `integration` | `centralized-data-service` | `go test -race -v -count=1 -run TestConnectionManager_InvalidatePool_HotSwapsDatabaseWithoutClosedError_RealPostgres ./test/internal/service/... ./internal/sinkworker/... ./internal/repository/master/...` | `PASS` kiểm chứng không gây hồi quy trên `ConnectionManager` hot-swap PostgreSQL 15 `-race`, `SinkWorker` và `MasterBindingRepo` (`EV-20261117T175335-12a1`) |
| `EV-20261117T175212-4c82` | `runtime` | `centralized-data-service` | `go build ./...` | `PASS` (`EV-20261117T175212-4c82`) |
| `EV-20261117T175237-301c` | `unit` | `centralized-data-service` | `go vet ./...` | `PASS` (`EV-20261117T175237-301c`) |
| `EV-20261117T175345-0ed6` | `unit` | `cdc-cms-service` | `go test -race -v -count=1 ./internal/api/scheduler/... ./internal/app/commands/scheduler/... ./internal/infra/persistence/scheduler/... ./internal/infra/persistence/master/...` | `PASS` kiểm chứng `master_binding_id > 0` chấp nhận `master_schema` có dấu `-` (`payment-service-v2`) và phát đủ tín hiệu NATS `"transmute.schedule.changed"` trên `Create`, `Toggle`, `Delete` (`EV-20261117T175345-0ed6`) |
| `EV-20261117T175306-2359` | `runtime` | `cdc-cms-service` | `go build ./...` | `PASS` (`EV-20261117T175306-2359`) |
| `EV-20261117T175311-b14c` | `unit` | `cdc-cms-service` | `go vet ./...` | `PASS` (`EV-20261117T175311-b14c`) |
| `EV-20261117T175349-82ea` | `unit` | `cdc-cms-web` | `npx tsc -b --noEmit` | `PASS` (`EV-20261117T175349-82ea`) |
| `EV-20261117T175417-5f82` | `unit` | `cdc-cms-web` | `npx eslint src/components/master/masterScheduleUtils.ts` | `PASS` kiểm tra ESLint trên helper mới với `0` lỗi và `0` cảnh báo (`EV-20261117T175417-5f82`) |
| `EV-20261117T175400-75a4` | `runtime` | `cdc-cms-web` | `npm run build` | `PASS` đóng gói bundle production Vite (`EV-20261117T175400-75a4`) |

---

## 1. Phản biện từng dòng git diff & từng task (Line-by-Line Critique — Hoàn tất `F1..F4`)

Đã chạy `git diff` và `wc -l` thực tế sau lần sửa cuối cùng trên cả 3 repository — mọi file lớn đều đạt `NOW <= HEAD`, tuân thủ chặt chẽ kiến trúc phân tầng và `0` `if/else` mới trong `forbidden_ifelse_files`:

1. **`centralized-data-service/internal/handler/master/transmute_realtime_gate.go` (`97` dòng, mới), `transmute_handler.go` (`793 -> 778` dòng, giảm `15` dòng) & `internal/repository/master/master_binding_repo.go` (`176 -> 218` dòng) — Xử lý `F1`:**
   - Giữ nguyên hành vi của `ListMasterTargetsByShadowIdentity` và `ListMasterTargetsByShadowTable` cho các luồng `bridge-oplog`, `heal`, `orphan_prune`, `batch-transform` và test hiện hữu (`EV-20261117T175118-cca1`).
   - Bổ sung `FilterPostIngestEnabledIDs` và `IsPostIngestExplicitlyDisabled` hỗ trợ cả ID âm của fixture test thực tế `!= 0` / `== 0`, được phát hiện và sửa ngay qua vòng Red -> Green (`EV-20261117T175118-cca1`).
   - Tại `handleTransmuteShadowWithOutcome`: khi `isRealtimePostIngestTrigger(req.TriggeredBy)` khớp luồng post-ingest (`batch-buffer`, `sinkworker`, `snapshot`, `kafka-consumer-hook`, `snapshot-hook`), lọc `masterTargets` chỉ giữ binding có `post_ingest` đang bật; nếu rỗng trả `nil` để `TransmuteConsumerPool` gọi `msg.Ack()` dập tắt bão phát lại JetStream `CDC_TRANSMUTE` (`EV-20261117T175118-cca1`).
   - Tại `runDebouncedGroup`: kiểm tra `shouldSkipDisabledRealtimeBatch` trước khi mở `ActivityLog` hay gọi `GetMasterDB`, chặn sạch các task đang gom dở trong `TableDebouncer` (`100ms–1s`) ngay khi người dùng gạt tắt Realtime (`EV-20261117T175118-cca1`).
2. **`centralized-data-service/internal/handler/shadow/batch_buffer_lifecycle.go` (`126 -> 153` dòng), `batch_buffer_fanout_cache.go` (`33` dòng), `internal/server/server_setup_buffers.go` (`87 -> 103` dòng), `server_config_reload.go` (`56 -> 61` dòng) & `internal/sinkworker/worker.go` (`488 -> 513` dòng) — Xử lý `F2` & `F4`:**
   - `FlushPostIngestCache()` xóa tức thì `bb.piCache` trên cả 3 `BatchBuffer` (`cdc`, `snapshot`, `heal`) và `SinkWorker` khi nhận tín hiệu NATS `"transmute.schedule.changed"` hoặc `"schema.config.reload"` (`EV-20261117T175147-9096`, `EV-20261117T175212-4c82`).
   - `postIngestEntryOnLookupError` giới hạn thời gian giữ stale cache khi query lỗi ở mức `postIngestFailOpenTTL` (`5s`) thay vì gia hạn lại nguyên `30s` (`EV-20261117T175147-9096`).
   - `InvalidateConnectionAdapter(connKey)` xóa `"master:" + connKey` và `"shadow:" + connKey` khỏi `bb.adapters` ngay trong `"connection.config.changed"`, không để sót `*SchemaAdapter` giữ con trỏ `*gorm.DB` đã đóng (`EV-20261117T175147-9096`).
   - Đồng bộ nhánh join `multi_shadow` (`transform_spec->'shadow_sources'`) vào `SinkWorker.resolvePostIngestSchedule` khớp hoàn toàn với `BatchBuffer` (`EV-20261117T175212-4c82`, `EV-20261117T175237-301c`).
3. **`cdc-cms-service/internal/app/commands/scheduler/{create_schedule,toggle_schedule,delete_schedule}.go` (`82`, `79`, `64` dòng), `internal/app/commands/master/toggle_master_active.go` (`75` dòng), `internal/server/server.go` (`497` dòng) & `internal/api/scheduler/transmute_schedule_handler.go` (`249` dòng) — Xử lý `F2` & `F3`:**
   - Giữ nguyên chữ ký hàm tạo `New*Handler(...)`, nối thêm `.WithPublisher(natsPublisher)` để phát NATS `"transmute.schedule.changed"` ngay sau khi lưu DB (`EV-20261117T175345-0ed6`, `EV-20261117T175306-2359`).
   - Miễn kiểm tra regex `schedNameRe` cho `masterSchema` khi `req.MasterBindingID > 0`, cho phép cập nhật schedule bình thường với tên database chứa dấu `-` như `payment-service-v2` (`EV-20261117T175345-0ed6`).
4. **`cdc-cms-web/src/components/master/masterScheduleUtils.ts` (`30` dòng, mới), `src/pages/MasterRegistry.tsx` (`1775 -> 1774` dòng) & `src/components/ReconPipelineGrid.tsx` (`1730 -> 1728` dòng) — Xử lý `F3`:**
   - Ưu tiên khớp tuyệt đối `s.master_binding_id === bindingId` khi đọc trạng thái công tắc `Realtime` / `Hẹn giờ` / `Immediate`, thêm `0` câu lệnh `if/else` mới vào `MasterRegistry.tsx` và giữ `NOW <= HEAD` trên cả hai file lớn (`EV-20261117T175349-82ea`, `EV-20261117T175400-75a4`, `EV-20261117T175417-5f82`).

---

## 2. Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

- **Zero Mock Pollution:** Không thêm bất kỳ trường closure giả `*Fn`, `sqlmock` hay SQLite in-memory nào vào code production hay bài test tích hợp.
- **Tái hiện Red -> Green thực tế trên PostgreSQL 15 + NATS JetStream thật (`EV-20261117T175118-cca1`):**
  - Dựng mô hình thật `1 Shadow -> 2 Masters` trên PostgreSQL 15 + NATS Server JetStream nhúng (`Stream: CDC_TRANSMUTE`):
    1. Pha 1 (`Master 1` bật `post_ingest`, `Master 2` tắt `post_ingest`): bắn sự kiện `"batch-buffer"` vào `CDC_TRANSMUTE` $\rightarrow$ xác nhận `Master 1` nhận `1` bản ghi, `Master 2` giữ nguyên `0` bản ghi, stream `CDC_TRANSMUTE` về `0` message (`EV-20261117T175118-cca1`).
    2. Pha 2 (Tắt nốt `Master 1` + `DROP TABLE` vật lý bảng `Master 2` để mô phỏng đích lỗi/đóng): bắn cả sự kiện JetStream `"batch-buffer"` lẫn đẩy trực tiếp batch `"kafka-consumer-hook"` vào `TableDebouncer` $\rightarrow$ xác nhận cả hai đều bỏ qua an toàn với `0` lỗi DB, stream `CDC_TRANSMUTE` được `Ack()` sạch về `0` message (`EV-20261117T175118-cca1`).
    3. Pha 3 (Kiểm chứng tương thích ngược `bridge-oplog` khi `post_ingest` vẫn đang tắt): tạo lại bảng `Master 2` và bắn sự kiện `"bridge-oplog"` $\rightarrow$ cả `Master 1` (`count=2`) và `Master 2` (`count=1`) đều đồng bộ đầy đủ (`EV-20261117T175118-cca1`).

---

## 3. Kiểm tra Concurrency, Multi-Worker, Timeout-Heartbeat & Watermark Continuity

1. **Multi-Worker Broadcast Invalidation (`transmute.schedule.changed` & `connection.config.changed`):**
   - Sự kiện phát qua NATS Core `Publish` (không qua QueueGroup) nên toàn bộ các replica pod của `centralized-data-service` và `SinkWorker` đồng loạt xóa `piCache` và `SchemaAdapter` cũ trong vài mili-giây (`EV-20261117T175147-9096`, `EV-20261117T175345-0ed6`).
2. **JetStream `AckWait` vs `Heartbeat` & `TableDebouncer`:**
   - Khi toàn bộ target bị lọc hết do tắt `post_ingest`, `handleTransmuteShadowWithOutcome` trả `nil` để `finalizeJetStreamMessage` gọi `msg.Ack()` ngay lập tức, không tạo `shadowMsgTracker` mồ côi và không phát sinh redelivery vượt `AckWait` (`EV-20261117T175118-cca1`).
   - Ở tầng `TableDebouncer` (`concurrency = 1`), nhánh `shouldSkipDisabledRealtimeBatch` gọi `t.complete(nil)` giải phóng `shadowMsgTracker` đúng chuẩn, bảo toàn dòng chảy cho các task `bridge-oplog`/`heal` đi cùng debouncer (`EV-20261117T175118-cca1`).
3. **Watermark / Cursor (`last_gpay_id` & Kafka Offset) Continuity:**
   - Kafka Shadow Ingest vẫn ghi dữ liệu vào bảng Shadow và tiến con trỏ offset bình thường; bên Master con trỏ `sync_runtime_state.last_cursor_json` không bị trượt khi `post_ingest` đang tắt, sẵn sàng cho lần chạy `Cron` hoặc `Run Now` tiếp theo (`EV-20261117T175118-cca1`).
4. **Kiểm định Bảo mật (`/security-agent`):**
   - Quét toàn bộ `git diff` trên 3 repo và các file trong `memory/workspaces/fix-master-realtime-off-db-closed-2026-11-17/`: `0` mật khẩu/token/DSN nhạy cảm lộ ra ngoài, payload NATS `"transmute.schedule.changed"` chỉ chứa metadata định danh schedule (`status`, `master_binding_id`, `mode`, `is_enabled`).

---

## 4. Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty)

- Toàn bộ kết quả kiểm thử, biên dịch và `wc -l` đều được đo trực tiếp sau lần sửa code cuối cùng với 11 biên lai vật lý khớp `src_fingerprint` (`EV-20261117T175118-cca1`, `EV-20261117T175147-9096`, `EV-20261117T175212-4c82`, `EV-20261117T175237-301c`, `EV-20261117T175335-12a1`, `EV-20261117T175306-2359`, `EV-20261117T175311-b14c`, `EV-20261117T175345-0ed6`, `EV-20261117T175349-82ea`, `EV-20261117T175400-75a4`, `EV-20261117T175417-5f82`).
- Báo cáo trung thực về trạng thái tiến trình (`L065`):
  - Trên máy local của anh hiện tại, tiến trình `cdc-cms-service` (`PID 79300`, mở trên `ttys028` từ `10:56:18` sáng) và cụm Worker trên môi trường Testing cần được **deploy / khởi động lại** để nạp binary mới (vì log `transmute_poison_pill` lúc `15:40:35` xuất phát từ binary cũ trước `e82ed6e` còn giữ con trỏ `sql.DB` đã đóng trong RAM; em tuyệt đối không tự ý `kill` terminal `ttys028` đang chạy của anh).
  - Lệnh `npm run lint` toàn repo `cdc-cms-web` hiện trả mã lỗi `1` do các lỗi `@typescript-eslint/no-explicit-any` tồn tại sẵn ở những file cũ ngoài phạm vi như `TransmuteSchedules.tsx`, `api.ts` (`EV-20261117T175354-fa56`).
  - Riêng helper mới `masterScheduleUtils.ts` đạt `PASS` ESLint tuyệt đối (`EV-20261117T175417-5f82`), đồng thời toàn bộ dự án `cdc-cms-web` đạt `PASS` ở cả `npx tsc -b --noEmit` (`EV-20261117T175349-82ea`) và `npm run build` (`EV-20261117T175400-75a4`).

---

## 5. Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- Đã áp dụng nghiêm ngặt các bài học **`L015`** (đối chiếu nguyên bản mã nguồn đĩa), **`L049`** (tách file SRP giữ `NOW <= HEAD` trên mọi file lớn: `transmute_handler.go` `793 -> 778`, `MasterRegistry.tsx` `1775 -> 1774`, `ReconPipelineGrid.tsx` `1730 -> 1728`), **`L065` & `L100`** (kiểm tra trung thực PID tiến trình và khóa khớp `src_fingerprint`), **`L074`** (bảo vệ secret/PII), **`L087` & `L088`** (xả êm JetStream `Ack()` chống bão redelivery & poison pill giả), và **`L101`** (sửa sạch luồng core hiện tại và tương lai, không nhồi band-aid) (`EV-20261117T175118-cca1`, `EV-20261117T175345-0ed6`). Không phát sinh vi phạm lesson nào trong phiên.

---

## 6. Kết quả QC & nhận xét từ /laya_router (laya_gate DG3 Mode A & Mode B)

- **File kết quả Mode A:** `06_laya_summary_dg3_20261117T175605.json`
- **File kết quả Mode B:** `06_laya_summary_dg3review_20261117T175646.json` (`centralized-data-service`) & `06_laya_summary_dg3review_20261117T175709.json` (`cdc-cms-service`)

```text
Mode A (dg3) — 06_laya_summary_dg3_20261117T175605.json:
stage: dg3 | gate: PASS | trusted_rounds: [] (UNVALIDATED — verified directly via git diff + RealPostgres 15 EV-20261117T175118-cca1 & EV-20261117T175345-0ed6)
A-F1-GATE     (transmute_realtime_gate.go:14-97)     -> r1=CORRECT (0.3690), r2=CORRECT, r3=YES (0.6868)
A-F1-HANDLER  (transmute_handler.go:176-198)         -> r1=CORRECT (0.3005), r2=CORRECT, r3=YES (0.6590)
A-F2-F4-CACHE (batch_buffer_lifecycle.go:44-80)      -> r1=CORRECT (0.6941), r2=CORRECT, r3=YES (0.6900)
A-F2-RELOAD   (server_config_reload.go:32-58)        -> r1=CORRECT (0.0604), r2=CORRECT, r3=YES (0.5423)
A-F3-CMS      (transmute_schedule_handler.go:94-99)  -> r1=CORRECT (0.0431), r2=CORRECT, r3=YES (0.7203)
A-F3-WEB      (masterScheduleUtils.ts:11-30)         -> r1=CORRECT (0.2104), r2=CORRECT, r3=YES (0.6536)

Mode B (dg3review) — centralized-data-service (06_laya_summary_dg3review_20261117T175646.json):
gate: PASS (all touched Go files ACCEPT / APPROVE)

Mode B (dg3review) — cdc-cms-service (06_laya_summary_dg3review_20261117T175709.json):
gate: PASS (all touched Go files ACCEPT / APPROVE)
```
