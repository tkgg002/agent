---
tier: T2
mode: standard
workspace: audit-consumer-retry-and-smoke-recon-2026-11-05
suffix: 2026_11_05_consumer_miss_smoke_recon
---

# 09. Tasks Solution & Audit Findings — Cơ chế Retry Consumer/DLQ & Smoke Recon 2h (`2026_11_05_consumer_miss_smoke_recon`)

## 1. Tóm tắt Kết luận Điều tra (`F1` – `F3`)

### Phát hiện F1 — Vì sao Smoke Recon báo "Đủ" (`diff = 0`, `status = "ok"`) sau khi qua 2 giờ (Báo bậy / False-OK)
- **Bằng chứng vật lý (`file:line`):**
  - `centralized-data-service/internal/service/recon/recon_smoke.go:154-385` (`RunTotalOnlyA`)
  - `centralized-data-service/internal/service/recon/recon_smoke_guard.go:37-164` (`verifySegmentAAtFreezeCutoff`) và `recon_smoke_guard.go:171-244` (`verifySegmentBAtFreezeCutoff`)
  - `centralized-data-service/internal/service/recon/recon_engine.go:309-321` (`effectiveLookback` mặc định ở `RunMode = "hot"` là `HotWindowLookback = 2 * time.Hour`).
- **Cơ chế gây báo bậy:**
  1. Trong `RunTotalOnlyA` (`recon_smoke.go:240-355`), hệ thống lấy tổng số bản ghi ước lượng trên Source (`srcEst` tại `hi = now - 5m`) trừ đi tổng số bản ghi active trên Shadow (`dstActive` tại `hi`), ra độ lệch tổng `rawDiff = srcEst - dstActive` (ví dụ `rawDiff = 11` bản ghi bị miss của bảng `trans_his`).
  2. Tuy nhiên, tại dòng `recon_smoke.go:355`, nó gọi `diff, statusStr := rc.verifySegmentAAtFreezeCutoff(ctx, entry, lo, hi, srcEst-dstActive)` với cửa sổ `[lo, hi]` chỉ dài **2 giờ** (`lo = hi - 2h`).
  3. Trong `verifySegmentAAtFreezeCutoff` (`recon_smoke_guard.go:46-52`), nếu `abs(diff) <= CountDriftThreshold` (ngưỡng mặc định `100` hoặc `500`, mà `11 <= 100`), hàm **không** báo `"drift"` ngay mà chạy `HashWindow` / `HashObjectIDWindow` chỉ trên phạm vi 2 giờ `[lo, hi]` (`recon_smoke_guard.go:84-110, 129-154`).
  4. Khi thời điểm xảy ra 11 bản ghi bị miss đã trôi qua mốc **2 giờ** (nằm trước `lo`), trong cửa sổ 2 giờ gần nhất `[lo, hi]` Source và Shadow khớp nhau 100% (`srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash`).
  5. Lúc này, `recon_smoke_guard.go:99-110` và `144-154` ngộ nhận độ lệch `11` dòng chỉ là sai số metadata của `EstimatedDocumentCount()`, nên **ép `return 0, "ok"`**, và `recon_smoke.go:379-381` ghi đè `srcTotalPtr = &dstActive`, dẫn tới UI hiển thị Smoke Recon khớp (`diff = 0`, `status = "ok"`) dù thực tế toàn bảng vẫn thiếu 11 bản ghi (như Report `#125` quét 24h vừa chứng minh: `Source = 10,980`, `Shadow = 10,969`, `Missing = 11`).

### Phát hiện F2 — Bug Nghiêm trọng ở Luồng `KafkaConsumer` / `BatchBuffer` → `DLQStateMachine` (Thông báo DLQ xong không chạy lại được / Nuốt lỗi)
- **Bằng chứng vật lý (`file:line`):**
  - `centralized-data-service/internal/handler/shadow/kafka_consumer.go:438-526`
  - `centralized-data-service/internal/handler/shadow/dlq_helper.go:19-111`
  - `centralized-data-service/internal/handler/shadow/batch_buffer.go:680-760`
  - `centralized-data-service/internal/handler/shadow/batch_buffer_logs.go:8-19`
  - `centralized-data-service/internal/handler/shadow/watermark_tracker.go:104-110`
  - `centralized-data-service/internal/handler/dlq/dlq_state_machine.go:94-266`
  - `centralized-data-service/internal/server/server_setup.go:198-199, 304-311`
- **Cơ chế gây mất khả năng tự chạy lại (Silent DLQ Blackhole):**
  1. **Khi lỗi ở tầng `BatchBuffer` (`batch_buffer.go:707, 746`):**
     - Hàm `writeFailedSyncLog` gọi `buildFailedSyncLog` (`batch_buffer_logs.go:8-19`) lưu vào bảng `cdc_system.failed_sync_logs` với `status = "failed"`, nhưng **bỏ trống `KafkaTopic = ""`**, và `RawJSON` chỉ lưu `record.RawData` (JSON phẳng của bản ghi, không có envelope).
     - Đồng thời, `batch_buffer.go:711, 751` gán kết quả record là `StatusDLQ`. Trong `watermark_tracker.go:107`, `StatusDLQ` được coi là `allSuccess = true` (không tạo barrier giữ offset), khiến **Kafka offset bị commit vượt qua luôn bản ghi lỗi**!
  2. **Khi lỗi ở tầng `KafkaConsumer.processMessage` (`kafka_consumer.go:500-526`):**
     - Ghi 1 dòng lỗi vào `cdc_activity_log` (`kafka_consumer.go:447-453`) và gọi `kc.writeDLQ(ctx, msg, procErr)` (`dlq_helper.go:19-55`) lưu vào `failed_sync_logs` với `KafkaTopic = msg.Topic` (ví dụ topic Kafka `mongo.core-trans-proxy-history-service.trans-his`) và `RawJSON` bị bọc thành `{"encoding":"base64","raw_base64":"..."}` nếu payload là binary Avro (`dlq_helper.go:101-109`).
     - Mặc dù tại `kafka_consumer.go:522` không gọi `commitOffsets` cho message lỗi đó, nhưng ở lần `flushBatch` kế tiếp, các message thành công có offset lớn hơn sẽ đẩy offset Kafka hoặc nếu lỗi nằm ở `BatchBuffer` thì `StatusDLQ` đã commit vượt qua.
  3. **Khi `DLQStateMachine` chạy chu kỳ 5 phút (`dlq_state_machine.go:94-266`):**
     - `DLQStateMachine` (khởi chạy tại `server_setup.go:198-199`, chu kỳ `5m`, **không ghi `cdc_activity_log`**) lấy các bản ghi `status IN ('pending','failed','retrying')` ra để replay (`replayOne`, dòng `210-266`).
     - Tại dòng `221-235`, nó bắn `sm.nats.Conn.PublishMsg(msg)` lên NATS Core với `Subject = row.KafkaTopic` (hoặc `"cdc.dlq"` nếu `KafkaTopic == ""`).
     - ** Điểm chết:** `ConsumerPool` của Shadow (`server_setup.go:304`) chỉ lắng nghe NATS JetStream subject `cdc.goopay.>`. Nó **không lắng nghe `"cdc.dlq"`** và cũng **không lắng nghe tên Kafka topic (`mongo...` / `debezium...`)** trên NATS!
     - Vì `nats.Conn.PublishMsg` là Core NATS fire-and-forget nên lệnh publish vẫn trả về `err == nil` dù không có subscriber nào nhận. Ngay lập tức, `dlq_state_machine.go:244-250` cập nhật `status = "resolved"` trong `failed_sync_logs`! Kết quả: bản ghi DLQ bị đánh dấu `"resolved"` giả tạo nhưng thực tế bị bắn vào hư không và không bao giờ được ghi vào Shadow DB.

### Phát hiện F3 — Lệnh `curl` Chuẩn xác để Heal Report `#125` (`core-trans-proxy-history-service.trans-his`) và Đối chiếu Thực trạng Bug Heal trên Prod (`a5dd18e`) vs Code Hiện tại (`WT`)
- **Bằng chứng vật lý (`file:line` đối chiếu `git diff a5dd18e` trên `centralized-data-service`):**
  - `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go:31-86` (`TriggerExecuteHeal` — `POST /api/v1/reconciliation/execute-heal`)
  - `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:139-302` (`executeHeal` → `processSingleReport` → `executeHealSegA`)
  - `centralized-data-service/internal/handler/recon/recon_heal_fetch.go:21-113` (`FetchAndWriteByIDs`)
- **Đối chiếu chính xác 4 Bug Heal trên Prod (`a5dd18e` = `786db91`) ĐÃ ĐƯỢC FIX ở code local hiện tại (`WT` trên nhánh `recon-heal`):**
  1. **Bug Heal #1 — `matchTargetTable` trượt so khớp chuỗi FQN có ngoặc kép `"\"shadow_...\".\"trans_his\""` khi `FlushTableBatchBuffer` (`batch_buffer.go:488-507` tại `a5dd18e` vs `batch_buffer_async_err.go:65-112` tại `WT`):**
     - Trong `FetchAndWriteByIDs` (`recon_heal_fetch.go:91, 104`), sau khi đẩy từng bản ghi qua `h.eventHandler.HandleRaw(ctx, raw)` vào `BatchBuffer`, hàm gọi `h.eventHandler.FlushTableBatchBuffer(ctx, entry.TargetTable)`.
     - Trên bản Prod `a5dd18e` (`batch_buffer.go:488-507`), `matchTargetTable` chỉ gọi `strings.Trim(r.PhysicalTableFQN, "\"")` ở 2 đầu ngoài của chuỗi `"\"shadow_core_trans_proxy_history_service\".\"trans_his\""`, làm còn sót dấu ngoặc kép ở giữa (`shadow_core_trans_proxy_history_service"."trans_his`). Nếu `entry.TargetTable` trong registry mang định dạng `shadow_schema.table` (`shadow_core_trans_proxy_history_service.trans_his`), cả `r.TableName == targetClean` (`"trans_his" == "shadow_...trans_his"` -> `false`) lẫn `rFQNClean == targetClean` đều trả về `false`! Kết quả: `FlushTableBatchBuffer` lọc ra `len(toFlush) == 0`, trả về `persisted = 0` và báo `status = "heal_failed"`.
     - **Đã fix ở code hiện tại (`WT`):** `batch_buffer_async_err.go:65-112` (`normalizeTableRef` dùng `strings.ReplaceAll(..., "\"", "")` tách chuẩn `(schema, table)`).
  2. **Bug Heal #2 — `ExecuteHealHandler` dùng chung `snapshotEventHandler` (`snapshotBatchBuffer`) với `SnapshotRunner` và bị `timerLoop()` (5s) tranh chấp flush ngầm / nuốt lỗi (`server_setup.go:480` & `batch_buffer.go:255-261` tại `a5dd18e` vs `server_setup.go:159,206,461` & `batch_buffer_async_err.go` tại `WT`):**
     - Trên Prod `a5dd18e` (`server_setup.go:480`), `ExecuteHealHandler` dùng chung `snapshotEventHandler` với `SnapshotRunner`. Trong `batch_buffer.go:255-261` (`a5dd18e`), goroutine `timerLoop()` gọi `_, _ = bb.Flush()` mỗi 5 giây (bỏ qua lỗi trả về). Nếu `timerLoop()` kịp flush các bản ghi đang nằm trong buffer trước khi `FetchAndWriteByIDs` gọi `FlushTableBatchBuffer` tại dòng `104`, hoặc nếu batch auto-flush bị lỗi SQL ngầm, thì `FlushTableBatchBuffer` ở cuối hàm thấy buffer trống và trả về `persisted = 0` → `finalizeReport` (`recon_execute_heal_handler.go:248-257`) đánh dấu `heal_failed` và không bắn NATS `cdc.cmd.transmute-shadow` (`if written > 0` tại dòng `296`).
     - **Đã fix ở code hiện tại (`WT`):** Tách riêng `healBatchBuffer` + `healEventHandler` (`server_setup.go:159, 206, 461`) và lưu sổ `asyncFlushErrs` theo từng bảng (`batch_buffer_async_err.go:18-61`).
  3. **Bug Heal #3 — Nhân đôi cột hệ thống `_source_ts` (`SQLSTATE 42701`) khi bảng có `_source_ts` trong `mapping_rule_v2` (`schema_adapter.go` & `dynamic_mapper.go` tại `a5dd18e` vs `internal/model/shadow/system_columns.go` & `internal/service/shadow/system_columns.go` tại `WT`):**
     - Trên Prod `a5dd18e`, nếu `mapping_rule_v2` chứa `_source_ts`, câu lệnh `INSERT INTO` khi heal/upsert bị lặp 2 lần cột `"_source_ts"` gây lỗi `SQLSTATE 42701` (`persisted = 0`).
     - **Đã fix ở code hiện tại (`WT`):** Chặn tại `modelshadow.ShouldSkipDataColumn` / `ShouldIncludeDataColumn`.
  4. **Bug Heal #4 — Segment B (`shadow_master`) trên Prod `a5dd18e` chỉ hỗ trợ Postgres Master (`recon_execute_heal_segb.go` tại `a5dd18e` vs `b8326bf` + `recon_execute_heal_segb_sql_dialects.go` tại `WT`):**
     - Đã fix ở commit `b8326bf` (hỗ trợ Mongo Master) và `WT` (hỗ trợ MySQL/ClickHouse Master).
  5. **Bug Heal #5 (Bắt được trực tiếp từ Live Trace `27dcf434893b4771af8dedc0eea363fa` — Activity Log `id: 9505337` lúc `2026-11-05T06:58:41.373Z`) — `buildHealEnvelope` trong `recon_heal_fetch.go:114-124` thiếu `"source_conn"`, bị `ResolveSourceRoutes` (`metadata_registry_service.go:605-619`) chặn hết route của connector có `topic_prefix` riêng:**
     - **Bằng chứng Runtime trên Prod:**
       - Request `POST /api/reconciliation/execute-heal` (`trace_id: 27dcf434893b4771af8dedc0eea363fa`) ghi nhận Activity Log `id: 9505337` (`status: "success"`, nhưng **`rows_affected: 0`**; tương tự các lần bấm trước `id: 9488484`, `9488767`, `9490395` đều `rows_affected: 0`).
       - Truy vấn `GET /api/v1/sources` trên Prod xác nhận connector `core-trans-proxy-history-service` (`id: 53`) có `"topic_prefix": "cdc.goopay.core_trans_proxy_history_service"`, nên `StripGenericPrefix(tp) = "core_trans_proxy_history_service" != ""`.
     - **Chuỗi gọi từ Nút "Heal" trên UI (`cdc-cms-web`) xuống Backend (`centralized-data-service`):**
       - Tại `cdc-cms-web/src/pages/DataIntegrity.tsx:554, 612-625` và `src/components/recon/ReconPipelineGrid.tsx`, nút **Heal** trên giao diện mở `<ExecuteHealModal />`.
       - Khi bấm xác nhận trên `<ExecuteHealModal />` (`cdc-cms-web/src/components/ExecuteHealModal.tsx:223-237`), hàm `executeHeal` gọi `useExecuteHealMutation` (`cdc-cms-web/src/hooks/useReconStatus.ts:345-380`) bắn **`POST /api/reconciliation/execute-heal`** với body `{"table":"trans_his","segment":"source_shadow","report_ids":[125],"heal_mismatched":false,"heal_missing_dest":true,"prune_missing_src":false,"force_heal":false}` (lưu ý `ExecuteHealModal.tsx:228-237` nhận prop `shadowSchema` và `shadowTable` nhưng không gửi 2 trường này lên body API; còn endpoint `/api/v1/reconciliation/heal/:table` ở backend hoàn toàn không được nút Heal trên UI gọi tới).
     - **Chuỗi lỗi vật lý tại Backend (`file:line`):**
       - `FetchAndWriteByIDs` (`recon_heal_fetch.go:53, 81-82, 114-124`) tạo `subject = "cdc.snapshot.core-trans-proxy-history-service.trans-his"` (4 phần, không có `source_conn`) và `buildHealEnvelope` không đính kèm `"source_conn"` (khác với `buildSnapshotEnvelope` tại `snapshot_runner_utils.go:148-159`).
       - `EventHandler.HandleRaw` (`event_handler.go:232-279`) gọi `h.registrySvc.ResolveSourceRoutes("core-trans-proxy-history-service", "trans-his", "")` với `sourceConn = ""`.
       - Tại `metadata_registry_service.go:605-619`, khi `sourceConn == ""`, vòng lặp kiểm tra `tp := rs.connCodeToTopicPrefix[r.SourceConnectionKey]`; vì `StripGenericPrefix(tp) != ""`, nó `continue` bỏ qua luôn route của `trans-his` và trả về `nil` (`source route not found in registry cache`).
       - `FetchAndWriteByIDs` (`recon_heal_fetch.go:82-86`) chỉ log `Warn("[heal-fetch] HandleRaw failed")` rồi `continue` qua cả 11 bản ghi, trả về `actualPersisted = 0`! (Lưu ý: Bug thiếu `"source_conn"` trong `buildHealEnvelope` vẫn còn tồn tại ngay trên nhánh local `WT` hiện tại).
     - **Workaround có sẵn trên Prod (không cần deploy code):** API `POST /api/v1/system/connectors/bridge-oplog` (`bridge_handler.go:203, 263, 516-595`) truyền thẳng `payload.ConnectorName` (`"core-trans-proxy-history-service"`) vào `ResolveSourceRoutes(coll.SourceDB, coll.SourceTable, conn)`, tự override `conn_string` bằng `tc.SourceURL` từ Registry (`bridge_handler.go:287-289`), đọc Oplog / fallback query theo khoảng thời gian (`bridge_mongo.go:53-160`), ghi thẳng vào `shadowDB` và tự bắn NATS `cdc.cmd.transmute-shadow`.

## 2. Kiểm chứng Trực tiếp trên Code Local (`WT` hiện tại) & Code Demo Sửa Triệt Để (Propose-Only — Chờ `APPROVE`)

### 2.1. Kết luận Kiểm tra Bản Local (`WT` nhánh `recon-heal`) & Truy vết Lịch sử Git (`git blame` / `git log -L`) — Phiên Nào Đã Phá Chức Năng Này
- **CÒN BỊ 100% TRÊN BẢN LOCAL (`WT`) HIỆN TẠI!**
- **Truy vết chính xác thời điểm & phiên làm việc đã phá hỏng chức năng Heal (`git log -L`):**
  1. **Mốc 1 — Ngày `24/09/2026 08:41:03` (Commit `46fcc3eca1bb6029d14d1158b1667ecb0193bc47` — `"update new connect master"` trên `centralized-data-service` & Commit `3a84cb49` trên `cdc-cms-web`):**
     - Ở commit `46fcc3ec`, Agent sửa `buildSnapshotEnvelope` (`internal/handler/orchestration/snapshot_runner_utils.go:148-168`) để thêm tham số `sourceConn ...string` (`"source_conn":"..."`) cho `SnapshotRunner`, nhưng **bỏ quên không cập nhật hàm song sinh `buildHealEnvelope` trong `internal/handler/recon/recon_heal_fetch.go:115-125`**.
     - Cùng lúc ở commit `3a84cb49` (`24/09/2026 08:40:03` trên `cdc-cms-web`), Agent thêm prop `shadowTable?: string` vào `ExecuteHealModal.tsx` nhưng cũng **bỏ quên không gửi `shadow_schema` / `shadow_table` trong `heal.mutateAsync` (`ExecuteHealModal.tsx:228-237`)**.
  2. **Mốc 2 (Thủ phạm trực tiếp làm chết hẳn nút Heal) — Phiên ngày `25/09/2026` (Commit `ac96c7ac96ef8a20195a49bff0e73d8d3e54f6fb` lúc `28/09/2026 08:49:42` — `"fix prefix conn"`, lưu biên bản tại `agent/memory/workspaces/feature-cdc-integration/audit_report_routing_fix.md:19-26`):**
     - Trong phiên `feature-cdc-integration` (tác vụ *"Routing Fix & Cross-Contamination Resolution"* xử lý vụ 2 connector `traitestctphs` 4-part và `traitestmongodevct` 5-part dùng chung bảng `trans_his`), Agent đã chèn đoạn lọc sau vào `MetadataRegistryService.ResolveSourceRoutes` (`internal/service/source/metadata_registry_service.go:486-500`):
       ```go
       if len(sourceConn) == 0 || sourceConn[0] == "" {
           var filtered []*metadata.ResolvedSourceRoute
           for _, r := range masterRoutes {
               tp := rs.connCodeToTopicPrefix[r.SourceConnectionKey]
               if tp != "" && StripGenericPrefix(tp) != "" {
                   continue
               }
               filtered = append(filtered, r)
           }
           masterRoutes = filtered
       }
       ```
     - Khi viết đoạn lọc trên, Agent ở phiên `feature-cdc-integration` đã suy diễn rằng *"message 5-part vốn đã có `sourceConn` cụ thể nên không bị ảnh hưởng"* (`audit_report_routing_fix.md:25`) mà **không kiểm tra toàn bộ các caller nội bộ gọi `EventHandler.HandleRaw` → `ResolveSourceRoutes`**! Vì `recon_heal_fetch.go:54, 82-83, 115-125` gọi `HandleRaw` với `sourceConn == ""`, nên ngay từ commit `ac96c7ac` (`28/09/2026`), mọi lệnh Heal Segment A cho các connector 5-part (có `topic_prefix` dạng `cdc.goopay.<conn>`) đều bị lọc mất route và trả về `rows_affected = 0`!
- Dù bản local (`WT`) đã fix 4 lỗi trước đó (`normalizeTableRef` tại `batch_buffer_async_err.go:65-112`, tách riêng `healBatchBuffer` tại `server_setup.go:159,206,461`, chặn trùng `_source_ts` tại `system_columns.go`, và dialect Segment B), nhưng **riêng lỗi số 5 từ commit `ac96c7ac` + `46fcc3ec` khiến Nút "Heal" trả về `rows_affected = 0` vẫn đang nằm nguyên trên bản local (`WT`) ở cả Backend lẫn Frontend**:
  1. **Tại Backend (`centralized-data-service/internal/handler/recon/recon_heal_fetch.go:54, 82-86, 115-125`):**
     - Trong toàn bộ codebase (`grep -rn "HandleRaw"`), `kafka_consumer.go:748` và `snapshot_runner_handler.go:560, 821-822` đều đính kèm `"source_conn"` vào envelope (và `WithBindingScope(ctx, bindingID)`).
     - Duy nhất `recon_heal_fetch.go:54, 82-83, 115-125` trên bản local vẫn tạo `subject := fmt.Sprintf("cdc.snapshot.%s.%s", entry.SourceDB, entry.SourceTable)` (4 phần) và `buildHealEnvelope(afterJSON, nowMs)` **không có trường `"source_conn"`**.
     - Khi chạy tới `MetadataRegistryService.ResolveSourceRoutes` (`internal/service/source/metadata_registry_service.go:486-500` trên local `WT`), vì `sourceConn == ""`, khối:
       ```go
       if len(sourceConn) == 0 || sourceConn[0] == "" {
           var filtered []*metadata.ResolvedSourceRoute
           for _, r := range masterRoutes {
               tp := rs.connCodeToTopicPrefix[r.SourceConnectionKey]
               if tp != "" && StripGenericPrefix(tp) != "" {
                   continue
               }
               filtered = append(filtered, r)
           }
           masterRoutes = filtered
       }
       ```
       vẫn lọc bỏ toàn bộ các bảng thuộc connector có `topic_prefix` riêng (như `core-trans-proxy-history-service` có `topic_prefix = "cdc.goopay.core_trans_proxy_history_service"`), trả về `nil` (`source route not found in registry cache`) → `FetchAndWriteByIDs` nuốt lỗi ở dòng `84-86` và trả về `actualPersisted = 0`!
  2. **Tại Frontend (`cdc-cms-web/src/components/ExecuteHealModal.tsx:228-237` & `src/hooks/useReconStatus.ts:348-368`):**
     - Bản local của `ExecuteHealModal.tsx` đã nhận props `shadowSchema` và `shadowTable` (truyền từ `DataIntegrity.tsx:616-617`), nhưng hàm `executeHeal` (`ExecuteHealModal.tsx:228-237`) và `useExecuteHealMutation` (`useReconStatus.ts:348-368`) vẫn **bỏ quên không gửi `shadow_schema` và `shadow_table` lên body POST `/api/reconciliation/execute-heal`**, chỉ gửi `"table": "trans_his"` trần.

### 2.2. Code Demo Giải pháp Sửa Triệt Để Trên Bản Local (Chờ `APPROVE`)

#### (A) Backend — `centralized-data-service/internal/handler/recon/recon_heal_fetch.go`
Tra cứu `route := h.resolveTargetRoute(entry)` qua `h.metadata.ResolveTargetRoute(...)` để lấy `sourceConn = route.SourceConnectionKey`, gắn `handlershadow.WithBindingScope(ctx, route.ShadowBinding.ID)` và truyền `sourceConn` vào `buildHealEnvelope` + `flushTarget`:

```go
// Trong FetchAndWriteByIDs (recon_heal_fetch.go):
	var sourceConn string
	flushTarget := entry.TargetTable
	if entry.ShadowSchema != "" && !strings.Contains(flushTarget, ".") {
		flushTarget = entry.ShadowSchema + "." + flushTarget
	}
	if h.metadata != nil {
		route := h.metadata.ResolveTargetRoute(flushTarget)
		if route == nil {
			route = h.metadata.ResolveTargetRoute(entry.TargetTable)
		}
		if route != nil {
			sourceConn = strings.TrimSpace(route.SourceConnectionKey)
			if route.ShadowBinding != nil && route.ShadowBinding.ID > 0 {
				ctx = handlershadow.WithBindingScope(ctx, route.ShadowBinding.ID)
			}
		}
	}
...
		envelope := buildHealEnvelope(afterJSON, nowMs, sourceConn)
		_, err = h.eventHandler.HandleRaw(ctx, subject, envelope)
...
func buildHealEnvelope(afterJSON []byte, clusterTimeMs int64, sourceConn ...string) []byte {
	var conn string
	if len(sourceConn) > 0 {
		conn = strings.TrimSpace(sourceConn[0])
	}
	var sb strings.Builder
	sb.WriteString(`{"specversion":"1.0","source":"heal:full-diff"`)
	if conn != "" {
		sb.WriteString(`,"source_conn":"`)
		sb.WriteString(conn)
		sb.WriteString(`"`)
	}
	sb.WriteString(`,"type":"cdc.snapshot","time":"`)
	sb.WriteString(time.Now().UTC().Format(time.RFC3339))
	sb.WriteString(`","data":{"op":"c","source_ts_ms":`)
	sb.WriteString(fmt.Sprintf("%d", clusterTimeMs))
	sb.WriteString(`,"after":`)
	sb.Write(afterJSON)
	sb.WriteString(`}}`)
	return []byte(sb.String())
}
```

#### (B) Frontend — `cdc-cms-web/src/components/ExecuteHealModal.tsx` & `src/hooks/useReconStatus.ts`
Truyền đầy đủ `shadow_schema`, `shadow_table`, `master_schema`, `master_table` từ `ExecuteHealModal` xuống `useExecuteHealMutation`:

```tsx
// ExecuteHealModal.tsx:228-237
      await heal.mutateAsync({
        table,
        shadow_schema: shadowSchema,
        shadow_table: shadowTable,
        master_schema: masterSchema,
        master_table: masterTable,
        segment,
        report_ids: reportIds,
        heal_mismatched: healMismatched,
        heal_missing_dest: healMissingDest,
        prune_missing_src: pruneMissingSrc,
        force_heal: forceHeal,
        reason: trimmed,
      });
```

---

## 3. Giải Pháp Khắc Phục Sau Kiểm Toán Phản Biện (`F-AUDIT-1` – `F-AUDIT-5`) — Propose-Only Chờ `APPROVE` (`Lesson L001`)

### 3.1. Tách File Đơn Trách Nhiệm (`recon_heal_route.go` & `recon_heal_source_conn_test.go`) & Chống Collision Đa Tuyến (`F-AUDIT-1`, `F-AUDIT-2`, `F-AUDIT-3`)

1. **Tạo file mới `centralized-data-service/internal/handler/recon/recon_heal_route.go`:**
   - Chứa hàm `resolveHealScopeAndTarget(ctx, entry)` đơn trách nhiệm:
     - Chuẩn hóa `flushTarget = entry.ShadowSchema + "." + entry.TargetTable`.
     - Khi tra cứu `route := h.metadata.ResolveTargetRoute(...)`, kiểm tra chặt chẽ `routeMatchesEntry(route, entry)` (`SourceDB` và `SourceTable` khớp `entry.SourceDB`, `entry.SourceTable` và `ShadowSchema` khớp nếu có) trước khi lấy `sourceConn = route.SourceConnectionKey` và gắn `handlershadow.WithBindingScope(ctx, route.ShadowBinding.ID)` — chặn đứng nguy cơ bốc nhầm route khi 2 connector có bảng trùng tên `"trans_his"`.
   - Trong `processSingleReport` (`recon_execute_heal_handler.go:199-205`): Tạo bản sao `entryCopy := *entry` và nếu `entryCopy.ShadowSchema == ""` thì gán `entryCopy.ShadowSchema = rpt.ShadowSchema` để trường hợp fallback V1 DB vẫn giữ được `ShadowSchema`.
   - Trong `FetchAndWriteByIDs` (`recon_heal_fetch.go`): Đếm `decodedCount` và lưu `lastHandleErr`; nếu `decodedCount > 0 && batchCount == 0 && lastHandleErr != nil` thì trả về `0, fmt.Errorf("heal-fetch HandleRaw failed for all %d docs: %w", decodedCount, lastHandleErr)` thay vì nuốt lỗi trả về `0, nil`.
2. **Tách `TestBuildHealEnvelope_IncludesSourceConn` khỏi `recon_heal_v4_test.go` sang `recon_heal_source_conn_test.go`:**
   - Trả `recon_heal_v4_test.go` về đúng `729` dòng (`NOW <= HEAD`).
   - Trong `recon_heal_source_conn_test.go`, bổ sung bài test tích hợp thật gọi qua `EventHandler.HandleRaw` → `MetadataRegistryService.ResolveSourceRoutes` với connector 5-part (`topic_prefix = "cdc.goopay.core_trans_proxy_history_service"`).
3. **Giữ trần số dòng `NOW <= HEAD` cho `ExecuteHealModal.tsx` (`<= 775` dòng) và `useReconStatus.ts` (`<= 698` dòng) & Ưu tiên `MasterSchema` khi `segment == "shadow_master"` (`F-AUDIT-1`, `F-AUDIT-4`).**

```go
// Code Demo: centralized-data-service/internal/handler/recon/recon_heal_route.go
package recon

import (
	"context"
	"strings"

	handlershadow "centralized-data-service/internal/handler/shadow"
	source "centralized-data-service/internal/model/source"
	"centralized-data-service/internal/service/metadata"
)

func (h *ExecuteHealHandler) resolveHealScopeAndTarget(
	ctx context.Context,
	entry *source.TableRegistry,
) (context.Context, string, string) {
	flushTarget := entry.TargetTable
	if entry.ShadowSchema != "" && !strings.Contains(flushTarget, ".") {
		flushTarget = entry.ShadowSchema + "." + flushTarget
	}
	if h == nil || h.metadata == nil {
		return ctx, "", flushTarget
	}
	route := h.metadata.ResolveTargetRoute(flushTarget)
	if !healRouteMatchesEntry(route, entry) {
		route = nil
	}
	if route == nil && flushTarget != entry.TargetTable {
		if candidate := h.metadata.ResolveTargetRoute(entry.TargetTable); healRouteMatchesEntry(candidate, entry) {
			route = candidate
		}
	}
	if route == nil {
		return ctx, "", flushTarget
	}
	sourceConn := strings.TrimSpace(route.SourceConnectionKey)
	if route.ShadowBinding != nil && route.ShadowBinding.ID > 0 {
		ctx = handlershadow.WithBindingScope(ctx, route.ShadowBinding.ID)
	}
	return ctx, sourceConn, flushTarget
}

func healRouteMatchesEntry(route *metadata.ResolvedSourceRoute, entry *source.TableRegistry) bool {
	if route == nil || entry == nil {
		return false
	}
	if route.TableConfig != nil {
		if entry.SourceDB != "" && route.TableConfig.SourceDB != "" && route.TableConfig.SourceDB != entry.SourceDB {
			return false
		}
		if entry.SourceTable != "" && route.TableConfig.SourceTable != "" && route.TableConfig.SourceTable != entry.SourceTable {
			return false
		}
	}
	if entry.ShadowSchema != "" && route.ShadowBinding != nil &&
		route.ShadowBinding.ShadowSchema != "" && route.ShadowBinding.ShadowSchema != entry.ShadowSchema {
		return false
	}
	return true
}
```

---

## 4. Giải Pháp Khắc Phục Sau Kiểm Toán Phản Biện Lượt 2 (`F-R2-1` – `F-R2-5`) — Propose-Only Chờ `APPROVE` (`Lesson L002`)

### 4.1. Danh Sách 5 Phát Hiện Phản Biện Lượt 2 (`F-R2-1` – `F-R2-5`)

1. **`F-R2-1` — Lỗ hổng Nuốt Lỗi ở Tầng Caller `fetchAndWriteChunked` (`recon_execute_heal_handler.go:404-422`) & Phóng Đại `healedSourceIDs` khi Partial Failure (`recon_execute_heal_handler.go:289-311`):**
   - **Bằng chứng `file:line` & `/laya_router` Mode A (`A-R2-SWALLOWED-CHUNK-ERR`: `ACCEPT 0.8923`, `A-R2-PARTIAL-TRANSMUTE-IDS`: `ACCEPT 0.7200`):**
     - Ở Lượt 2, trong `FetchAndWriteByIDs` (`recon_heal_fetch.go:115-117`), em đã thêm `if decodedCount > 0 && batchCount == 0 && lastHandleErr != nil { return 0, fmt.Errorf(...) }` với mục đích để `HandleExecuteHeal` không ghi khống `status = "success"`, `rows_affected = 0` vào `cdc_activity_log`.
     - **Tuy nhiên**, em đã không truy vết ngược lên caller `fetchAndWriteChunked` (`recon_execute_heal_handler.go:404-422`)! Hàm `fetchAndWriteChunked` gọi `written, err := h.FetchAndWriteByIDs(...)`, nhưng khi `err != nil` nó chỉ ghi log `Error("[execute-heal-a] chunk heal failed", zap.Error(err))` rồi **nuốt trọn `err`**, chỉ trả về `totalWritten int` (không trả về `error`). Hệ quả là `executeHealSegA` $\rightarrow$ `processSingleReport` $\rightarrow$ `executeHeal` luôn trả về `err = nil`, khiến `HandleExecuteHeal` (`recon_execute_heal_handler.go:125-133`) **vẫn ghi `status = "success"`, `rows_affected = 0` vào `cdc_activity_log` ngay cả khi 100% các chunk `FetchAndWriteByIDs` đều trả về lỗi**!
     - Đồng thời, tại `executeHealSegA` (`recon_execute_heal_handler.go:289-311`), khi xảy ra Partial Failure (ví dụ `len(missingIDs) == 10` nhưng chỉ `written == 8` bản ghi xuống Shadow, `2` bản ghi bị drop/chunk lỗi), dòng `306` vẫn thực hiện `healedSourceIDs = append(healedSourceIDs, missingIDs...)` (nạp đủ 10 ID) và bắn lệnh NATS `cdc.cmd.transmute-shadow` cho cả 2 ID chưa hề có mặt dưới Shadow!

2. **`F-R2-2` — Lỗ hổng Fallback `ResolveSourceRoutes` Thiếu `sourceConn` trong `recon_heal_route.go:33-40` & Nguy cơ Bốc Nhầm `TableRegistry` tại `resolveTargetTableConfig(rpt.TargetTable)` (`recon_execute_heal_handler.go:199`):**
   - **Bằng chứng `file:line` & `/laya_router` Mode A (`A-R2-FALLBACK-NO-CONN`: `r1: CORRECT 0.6804`, `r3: YES 0.3678`):**
     - Trong `recon_heal_route.go:33-40`, nhánh fallback gọi `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable)` mà không truyền `sourceConn`. Như chính bài test `recon_heal_source_conn_test.go:192` đã chứng minh, `ResolveSourceRoutes` khi không có `sourceConn` sẽ lọc bỏ mọi connector 5-part (`StripGenericPrefix(tp) != ""`)! Do đó, nếu `ResolveSourceRoutes(entry.SourceDB, entry.SourceTable)` trả về rỗng, cần fallback thêm `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable, entry.SourceDB)`.
     - Ngoài ra, tại `processSingleReport` (`recon_execute_heal_handler.go:199`), `entry := h.resolveTargetTableConfig(rpt.TargetTable)` tra cứu theo `rpt.TargetTable`. Nếu report cũ có `rpt.ShadowSchema == ""` (`rpt.TargetTable = "trans_his"`) trong khi có 2 connector cùng có bảng `"trans_his"`, `GetTableConfig("trans_his")` có thể trả về `entry` của connector kia. Cần bổ sung helper `resolveReportTableConfig(rpt)` (trong `recon_heal_route.go`) ưu tiên tra theo `rpt.ShadowSchema + "." + rpt.TargetTable` và kiểm tra khớp `rpt.SourceDB` / `rpt.SourceTable`.

3. **`F-R2-3` — Lỗ hổng Thiếu `master_schema` / `master_table` ở Frontend (`reconHealPayload.ts:44-59`, `ExecuteHealModal.tsx`) — Bị `/laya_router` Mode B Bắt Trúng (`REJECT_LOGIC`):**
   - **Bằng chứng `file:line` & `/laya_router` Mode A (`D-R2-FRONTEND-NO-MASTER-SCHEMA`: `ACCEPT 0.7838`) + Mode B (`06_laya_summary_dg3review_20261106T154935.json`):**
     - Ở Lượt 2, dù em đã thêm `master_schema?: string` và `master_table?: string` vào `ExecuteHealPayload` (`reconHealPayload.ts:12-13`) và thêm nhánh `if req.Segment == "shadow_master"` ở Backend (`reconciliation_handler_execute_heal.go:55-63`), nhưng trong hàm `buildExecuteHealMutationInput` (`reconHealPayload.ts:44-59`), em chỉ gán `shadow_schema: input.segment === 'shadow_master' ? undefined : input.shadowSchema` mà **hoàn toàn không truyền `master_schema` và `master_table` vào object trả về**! Khi chạy `/laya_router` Mode B (`review`) trên đầy đủ 9 file, Laya đã lập tức đánh `REJECT_LOGIC` trên `src/utils/reconHealPayload.ts`.

4. **`F-R2-4` — Lỗ hổng Ép Kiểu Số Âm `uint(r.SourceObjectID)` gây Underflow trong `recon_smoke.go:730` & Tự Phê Bình Việc Cắt Xén Diff Khi Chạy `/laya_router` Mode B ở Lượt 2:**
   - **Bằng chứng `file:line` & `/laya_router` Mode B (`internal/service/recon/recon_engine_segment_b.go`: `REJECT_LOGIC` trong `06_laya_summary_dg3review_20261106T154935.json`):**
     - **Tự phê bình trung thực về việc cắt xén diff Mode B ở Lượt 2:** Trong lần chạy `laya_gate.py review` ở cuối Lượt 2 (`06_laya_summary_dg3review_20261106T152845.json`), file `/tmp/r2_diff.patch` em truyền vào chỉ gồm `git diff HEAD` của 4 file Go đã sửa trong `centralized-data-service`, **bỏ sót** file mới `recon_heal_route.go`, `recon_heal_source_conn_test.go`, file `reconciliation_handler_execute_heal.go` bên `cdc-cms-service` và 3 file Frontend bên `cdc-cms-web`! Khi em tạo lại patch đầy đủ tất cả các file (`scratch/full_task_r2_diff.patch`) và chạy lại `laya_gate.py review` (`06_laya_summary_dg3review_20261106T154935.json`), `/laya_router` Mode B đã bắt trúng 2 lỗi `REJECT_LOGIC` tại `recon_engine_segment_b.go` và `src/utils/reconHealPayload.ts`!
     - **Bản chất lỗi tại `recon_engine_segment_b.go:23, 243` & `recon_smoke.go:730`:** Khi `r.SourceObjectID` mang giá trị âm (như `-492241001` từ dữ liệu test E2E trong `cdc_system.source_object_registry`), phép ép kiểu `uint(r.SourceObjectID)` tại `recon_smoke.go:730` gây tràn số nguyên âm sang số dương cực lớn (`18446744073217310615`), và câu truy vấn `ListActiveMasterBindings` (`recon_engine_segment_b.go:243`) vẫn nạp cả các bản ghi fixture test có `sor.id <= 0` vào luồng đối soát thực tế. Cần thêm rào chắn `if r.SourceObjectID <= 0 { continue }` tại `recon_smoke.go:730` và điều kiện `AND sor.id > 0` (hoặc bỏ qua ID âm khi lookup) để tránh tràn số `uint`.

5. **`F-R2-5` — Kiểm Tra Trung Thực Về Test Coverage, `[no tests to run]` & Suy Diễn Kết Quả (`Rule #9 S5(b)-(c)`):**
   - **Thừa nhận thẳng thắn 3 khoảng trống kiểm thử ở Lượt 2:**
     1. **Vòng lặp xử lý cursor `decodedCount / droppedCount / lastHandleErr` trong `FetchAndWriteByIDs` (`recon_heal_fetch.go:70-125`) chưa được unit test gọi trực tiếp:** Vì `FetchAndWriteByIDs` gọi `h.reconCore.GetMongoClientForURL` ngay ở dòng `52` trước khi vào vòng lặp đọc cursor, bài test `TestHealFlow_FivePartTopicPrefix_EndToEndRouteResolution` trong `recon_heal_source_conn_test.go` chỉ gọi `resolveHealScopeAndTarget` + `buildHealEnvelope` + `eventHandler.HandleRaw` rời bên ngoài chứ chưa chạy qua hàm xử lý batch tài liệu của `FetchAndWriteByIDs` để kiểm chứng nhánh `decodedCount > 0 && batchCount == 0 && lastHandleErr != nil` trả về lỗi thật và nhánh partial drop ghi log `doc_id`.
     2. **Biên lai `EV-20261106T152158-e517` (`cdc-cms-service` `go test -v ./internal/api/recon/...`) thực chất là `[no tests to run]`:** Ở báo cáo Lượt 2, em ghi `go test -v ./internal/api/recon/... PASS (EV-20261106T152158-e517)`. Kiểm tra lại `stdout_tail` của biên lai `EV-20261106T152158-e517`, thực tế thư mục `cdc-cms-service/internal/api/recon` **chưa có bất kỳ file `*_test.go` nào** (`testing: warning: no tests to run`), nên việc báo PASS mà không nói rõ `[no tests to run]` là thiếu trung thực!
     3. **Chưa chạy Live Runtime Trace với MongoDB nguồn + PostgreSQL Shadow thật:** Cần phối hợp với anh để trace runtime thật sau khi anh duyệt bản vá.

---

### 4.2. Code Demo Khắc Phục Triệt Để `F-R2-1` – `F-R2-5` (Propose-Only — Chờ Anh `APPROVE`)

#### (A) Tách hàm `processHealDocs` trong `recon_heal_fetch.go` & Trả về danh sách `healedIDs` + `error` cho `fetchAndWriteChunked` (`F-R2-1`, `F-R2-5`)

```go
// Trong centralized-data-service/internal/handler/recon/recon_heal_fetch.go:
// Tách phần xử lý danh sách raw BSON docs ra hàm processHealDocs để vừa giữ FetchAndWriteByIDs gọn nhẹ,
// vừa test trực tiếp được 100% nhánh Partial Failure / 100% Failure mà không cần mock MongoClient:
func (h *ExecuteHealHandler) processHealDocs(
	ctx context.Context,
	entry *source.TableRegistry,
	rawDocs []bson.Raw,
) (int, error) {
	ctx, sourceConn, flushTarget := h.resolveHealScopeAndTarget(ctx, entry)
	subject := fmt.Sprintf("cdc.snapshot.%s.%s", entry.SourceDB, entry.SourceTable)
	nowMs := time.Now().UnixMilli()
	var batchCount, decodedCount, droppedCount int
	var lastHandleErr error

	for _, rawDoc := range rawDocs {
		docID := extractBSONDocID(rawDoc)
		afterJSON, err := convertBSONDocToCleanJSON(rawDoc)
		if err != nil {
			droppedCount++
			lastHandleErr = err
			observability.Ctx(ctx, h.logger).Warn("[heal-fetch] BSON to JSON failed",
				zap.String("table", flushTarget), zap.String("doc_id", docID), zap.Error(err))
			continue
		}
		decodedCount++
		envelope := buildHealEnvelope(afterJSON, nowMs, sourceConn)
		if _, err = h.eventHandler.HandleRaw(ctx, subject, envelope); err != nil {
			droppedCount++
			lastHandleErr = err
			observability.Ctx(ctx, h.logger).Warn("[heal-fetch] HandleRaw failed",
				zap.String("table", flushTarget), zap.String("doc_id", docID), zap.Error(err))
			continue
		}
		batchCount++
	}
	if droppedCount > 0 {
		observability.Ctx(ctx, h.logger).Warn("[heal-fetch] partial or full drop during HandleRaw loop",
			zap.String("table", flushTarget), zap.Int("decoded", decodedCount),
			zap.Int("accepted", batchCount), zap.Int("dropped", droppedCount), zap.Error(lastHandleErr))
	}
	if decodedCount > 0 && batchCount == 0 && lastHandleErr != nil {
		return 0, fmt.Errorf("heal-fetch HandleRaw failed for all %d decoded docs on %s: %w", decodedCount, flushTarget, lastHandleErr)
	}
	if batchCount == 0 {
		return 0, nil
	}
	return h.flushHealBatch(ctx, entry, flushTarget, batchCount)
}
```

Và tại `recon_execute_heal_handler.go` (`fetchAndWriteChunked` & `executeHealSegA`):
- `fetchAndWriteChunked` trả về `(totalWritten int, healedIDs []string, lastErr error)`: chỉ những chunk ghi thành công `written == len(chunk)` mới nối `chunk` vào `healedIDs` để bắn `transmute-shadow`, và nếu `totalWritten == 0 && lastErr != nil` thì truyền `lastErr` lên `executeHealSegA` $\rightarrow$ `processSingleReport` để `HandleExecuteHeal` ghi đúng `status = "error"` vào `cdc_activity_log`!

#### (B) Bổ sung `sourceConn` cho Fallback `ResolveSourceRoutes` và `resolveReportTableConfig` trong `recon_heal_route.go` (`F-R2-2`)

```go
// Trong centralized-data-service/internal/handler/recon/recon_heal_route.go:
	if route == nil && entry.SourceDB != "" && entry.SourceTable != "" {
		candidates := h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable)
		if len(candidates) == 0 {
			candidates = h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable, entry.SourceDB)
		}
		for _, candidate := range candidates {
			if healRouteMatchesEntry(candidate, entry) {
				route = candidate
				break
			}
		}
	}
```

#### (C) Truyền đầy đủ `master_schema` & `master_table` khi `segment === 'shadow_master'` trong `cdc-cms-web/src/utils/reconHealPayload.ts` & `ExecuteHealModal.tsx` (`F-R2-3`)

```ts
// Trong cdc-cms-web/src/utils/reconHealPayload.ts:
export interface BuildExecuteHealMutationInput {
  table: string;
  shadowSchema?: string;
  shadowTable?: string;
  masterSchema?: string;
  masterTable?: string;
  segment: 'source_shadow' | 'shadow_master';
  reportIds: number[];
  healMismatched: boolean;
  healMissingDest: boolean;
  pruneMissingSrc: boolean;
  forceHeal: boolean;
  reason: string;
}

export function buildExecuteHealMutationInput(
  input: BuildExecuteHealMutationInput,
): ExecuteHealPayload & { reason?: string } {
  const isSegB = input.segment === 'shadow_master';
  return {
    table: input.table,
    shadow_schema: isSegB ? undefined : input.shadowSchema,
    shadow_table: isSegB ? undefined : input.shadowTable,
    master_schema: isSegB ? input.masterSchema : undefined,
    master_table: isSegB ? input.masterTable : undefined,
    segment: input.segment,
    report_ids: input.reportIds,
    heal_mismatched: input.healMismatched,
    heal_missing_dest: input.healMissingDest,
    prune_missing_src: input.pruneMissingSrc,
    force_heal: input.forceHeal,
    reason: input.reason.trim(),
  };
}
```

#### (D) Chặn Underflow Số Âm `r.SourceObjectID <= 0` tại `recon_smoke.go:730` & `recon_engine_segment_b.go:243` (`F-R2-4`)

```go
// Trong centralized-data-service/internal/service/recon/recon_smoke.go:728-733:
	for _, r := range refs {
		if r.SourceObjectID <= 0 {
			continue
		}
		entry, ok := entryMap[uint(r.SourceObjectID)]
		if !ok {
			continue
		}
```

#### (E) Bổ sung Unit Test cho `TriggerExecuteHeal` tại `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal_test.go` & Test `processHealDocs` tại `recon_heal_source_conn_test.go` (`F-R2-5`)
- Tạo file test `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal_test.go` kiểm chứng cả 2 nhánh `segment == "source_shadow"` (qualify `shadow_schema.table`) và `segment == "shadow_master"` (qualify `master_schema.table`, không bị `shadow_schema` ghi đè), xóa bỏ hoàn toàn tình trạng `[no tests to run]`.
- Bổ sung `TestProcessHealDocs_AllFailedReturnsErrorAndLogsDocID` và `TestProcessHealDocs_PartialDropLogsWarning` vào `recon_heal_source_conn_test.go`.

---

## 11. Khắc Phục Lỗi `execute-heal` Cho Nguồn PostgreSQL (`shadow_testcoretrans.transaction_history` — `F-PG-1` → `F-PG-3`, Propose-Only Chờ `APPROVE`)

### 11.1. Nguyên nhân Gốc rễ (`file:line`) & Tự Kiểm Điểm Vì Sao Unit Test Tự Viết Đã Bypass Lỗi Này (`Lesson L001`)
1. **Vì sao em bị lọt lỗi này dù báo cáo test xanh:**
   - Trong `Task 9`, em tách `processHealDocs(ctx, entry, docs []bson.M)` ra khỏi `FetchAndWriteByIDsWithAccepted` và viết unit test gọi thẳng vào `processHealDocs(...)` với dữ liệu `bson.M` dựng sẵn trong bộ nhớ — tức là **tự ý nhảy cóc (bypass) qua dòng `recon_heal_fetch.go:50` (`h.reconCore.GetMongoClientForURL(ctx, entry.SourceURL)`)**!
   - Trước `Task 9`, `fetchAndWriteChunked` nuốt sạch lỗi của `FetchAndWriteByIDs` nên UI luôn báo `status = "success"`, `rows_affected = 0`. Sau khi `Task 9` truyền lỗi thật ra ngoài, khi anh chạy `execute-heal` trên bảng `shadow_testcoretrans.transaction_history` (có nguồn là PostgreSQL `postgres://...`, `source_object_id = 214`, `shadow_binding_id = 219`), lỗi thật bị che giấu bấy lâu lập tức nổ ra:
     `get mongo client: connect to source postgres://<redacted>: error parsing uri: scheme must be "mongodb" or "mongodb+srv"`.
2. **3 Lỗi Kỹ Thuật Cốt Lõi Khi Heal Segment A Từ Nguồn PostgreSQL (`F-PG-1` → `F-PG-3`):**
   - **`F-PG-1` (`recon_heal_fetch.go:50-53`):** `FetchAndWriteByIDsWithAccepted` gọi cứng `h.reconCore.GetMongoClientForURL(ctx, entry.SourceURL)` cho mọi bảng mà không kiểm tra `isPostgresSourceEntry(entry)` (`entry.SourceType == "postgresql"` / `"postgres"` hoặc `strings.HasPrefix(entry.SourceURL, "postgres")`). Trong khi đó, `ReconSourceAgent` (`recon_source_agent.go:58-106`) đã có sẵn `isPostgres` và `getPGClient(ctx, sourceURL)` quản lý kết nối `*gorm.DB` tới PostgreSQL nguồn!
   - **`F-PG-2` (`recon_heal_fetch.go:81, 89, 140-160` & `event_handler.go:154-159, 203-212, 500-525`):**
     - Với nguồn PostgreSQL, `synthesizeLegacyTableRegistry` (`metadata_registry_utils.go:99-102`) gán `entry.SourceTable = "public.transaction_history"` (có dấu chấm `.`). Nếu `buildHealEnvelope` không đóng gói `"source_db"` và `"source_table"`, `EventHandler.HandleRaw` (`event_handler.go:204-207`) sẽ cắt `subject = "cdc.snapshot.<db>.public.transaction_history"` từ phải sang trái và gán nhầm `db = "public"`, `table = "transaction_history"`, làm `ResolveSourceRoutes` trượt 100%!
     - Hơn nữa, nếu đưa bản ghi PostgreSQL qua `bson.MarshalExtJSON(doc, true, false)`, các trường số bị bọc thành `{"$numberDouble": "..."}`, khiến `extractPrimaryKey` (`event_handler.go:515-525`) cho nguồn `postgresql` đọc ra chuỗi `"map[$numberDouble:...]"` thay vì giá trị khóa chính! Giống hệt `SnapshotRunner` (`snapshot_runner_handler.go:735-745`), nguồn PostgreSQL phải đọc bằng `SELECT row_to_json(t)::text` và truyền thẳng raw JSON bytes vào `buildHealEnvelope`.
   - **`F-PG-3` (`recon_heal_fetch.go:88`):** `processHealDocs` đang chỉ lấy `docID := extractHealDocID(doc["_id"])`. Với nguồn PostgreSQL, khóa chính trong bản ghi JSON là `entry.PrimaryKeyField` (ví dụ `"id"`), không phải `"_id"`. Nếu không lấy từ `doc[entry.PrimaryKeyField]` (fallback `"_id"` và `"id"`), `acceptedIDs` sẽ rỗng và không kích hoạt `cdc.cmd.transmute-shadow`.

### 11.2. Code Demo Khắc Phục Triệt Để (`Propose-Only` — Chờ Anh `APPROVE`)

#### (1) Mở rộng `GetPGClient` & `GetPGClientForURL` tại `recon_source_agent.go` & `recon_engine.go`
```go
// Trong centralized-data-service/internal/service/recon/recon_source_agent.go:
func (sa *ReconSourceAgent) GetPGClient(ctx context.Context, sourceURL string) (*gorm.DB, error) {
	return sa.getPGClient(ctx, sourceURL)
}

// Trong centralized-data-service/internal/service/recon/recon_engine.go:
func (rc *ReconCore) GetPGClientForURL(ctx context.Context, sourceURL string) (*gorm.DB, error) {
	if rc.sourceAgent == nil {
		return nil, fmt.Errorf("reconCore.sourceAgent not initialized")
	}
	return rc.sourceAgent.GetPGClient(ctx, sourceURL)
}
```

#### (2) Phân nhánh PostgreSQL Source trong `recon_heal_fetch.go` & Tạo mới `recon_heal_postgres.go` (SRP)
```go
// Trong centralized-data-service/internal/handler/recon/recon_heal_fetch.go:
func (h *ExecuteHealHandler) FetchAndWriteByIDsWithAccepted(ctx context.Context, entry *source.TableRegistry, ids []string) (int, []string, error) {
	if h.eventHandler == nil {
		return 0, nil, fmt.Errorf("eventHandler not wired — call WithEventHandler first")
	}
	if h.reconCore == nil {
		return 0, nil, fmt.Errorf("reconCore not wired")
	}
	if isPostgresSourceEntry(entry) {
		return h.fetchAndWritePostgresByIDs(ctx, entry, ids)
	}
	// ... giữ nguyên nhánh MongoDB cho nguồn mongodb:// ...
}
```

```go
// File mới: centralized-data-service/internal/handler/recon/recon_heal_postgres.go
package recon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	source "centralized-data-service/internal/model/source"
	"centralized-data-service/pkgs/observability"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func isPostgresSourceEntry(entry *source.TableRegistry) bool {
	if entry == nil {
		return false
	}
	st := strings.ToLower(strings.TrimSpace(entry.SourceType))
	if st == "postgres" || st == "postgresql" {
		return true
	}
	u := strings.ToLower(strings.TrimSpace(entry.SourceURL))
	return strings.HasPrefix(u, "postgres://") || strings.HasPrefix(u, "postgresql://")
}

func (h *ExecuteHealHandler) fetchAndWritePostgresByIDs(ctx context.Context, entry *source.TableRegistry, ids []string) (int, []string, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}
	db, err := h.reconCore.GetPGClientForURL(ctx, entry.SourceURL)
	if err != nil {
		return 0, nil, fmt.Errorf("get postgres client: %w", err)
	}
	schemaName, bareTable := splitPGSchemaAndTable(entry.SourceTable)
	pkCol := strings.TrimSpace(entry.PrimaryKeyField)
	if pkCol == "" || pkCol == "_id" {
		pkCol = resolvePGPrimaryKeyCol(ctx, db, schemaName, bareTable)
	}
	relation := quoteIdent(schemaName) + "." + quoteIdent(bareTable)
	filterArg := buildPGPrimaryKeyFilterArg(ctx, db, schemaName, bareTable, pkCol, entry.PrimaryKeyType, ids)

	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	sqlQuery := fmt.Sprintf(`SELECT row_to_json(t)::text FROM (SELECT * FROM %s WHERE %s IN (?)) t`, relation, quoteIdent(pkCol))
	rows, err := db.WithContext(queryCtx).Raw(sqlQuery, filterArg).Rows()
	if err != nil {
		return 0, nil, fmt.Errorf("postgres find by ids on %s: %w", relation, err)
	}
	defer rows.Close()

	var rawRows [][]byte
	for rows.Next() {
		var jsonStr string
		if sErr := rows.Scan(&jsonStr); sErr != nil {
			observability.Ctx(ctx, h.logger).Warn("[heal-fetch-pg] scan row failed", zap.Error(sErr))
			continue
		}
		if strings.TrimSpace(jsonStr) != "" {
			rawRows = append(rawRows, []byte(jsonStr))
		}
	}
	if rErr := rows.Err(); rErr != nil {
		observability.Ctx(ctx, h.logger).Warn("[heal-fetch-pg] rows error", zap.Error(rErr))
	}
	return h.processHealRawJSONRows(ctx, entry, pkCol, rawRows)
}
```
Và trong `buildHealEnvelope` / `processHealRawJSONRows`:
- Đóng gói `"source_db": entry.SourceDB` và `"source_table": entry.SourceTable` trực tiếp vào JSON `CDCEvent` để `EventHandler.HandleRaw` (`event_handler.go:154-159`) nhận diện chính xác `db = entry.SourceDB` và `table = "public.transaction_history"` mà không bị dấu chấm trong `public.transaction_history` làm lệch `subject`.
- Giải mã khóa chính bằng `json.NewDecoder(bytes.NewReader(rawJSON))` với `UseNumber()` trên trường `pkCol` (fallback `"id"` / `"_id"`) để lấy chính xác `docID` đưa vào `acceptedIDs`.
