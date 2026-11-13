---
tier: T2
mode: standard
workspace: audit-consumer-retry-and-smoke-recon-2026-11-05
suffix: 2026_11_05_consumer_miss_smoke_recon
---

# Audit Report — Kiểm tra Cơ chế Retry Consumer/DLQ & Smoke Recon 2h (`2026_11_05_consumer_miss_smoke_recon`)

## (a) Line-by-Line Code Evidence Audit (`file:line`)
- **Phạm vi:** Read-Only Audit (`mode: audit`) theo yêu cầu của User (*"ko muốn deloy prod ở phiên này. hiểu ko nên ko thể update code"*). Không thay đổi bất kỳ dòng source code nào.
- **Đối chiếu từng Task trong `08_tasks_2026_11_05_consumer_miss_smoke_recon.md`:**
  1. **Task 1 — Cơ chế tự chạy lại Consumer / DLQ và lý do "báo DLQ xong không chạy lại được":**
     - `centralized-data-service/internal/handler/shadow/kafka_consumer.go:438-526`: Khi `processMessage` lỗi, `KafkaConsumer` ghi 1 dòng lỗi vào `cdc_system.cdc_activity_log` (`kafka_consumer.go:447-453`) và gọi `kc.writeDLQ(ctx, msg, procErr)` (`dlq_helper.go:19-55`) lưu bản ghi vào bảng `cdc_system.failed_sync_logs` (`status = 'failed'` hoặc `'pending'`, `KafkaTopic = msg.Topic`).
     - `centralized-data-service/internal/handler/shadow/batch_buffer.go:707-712, 746-752` & `batch_buffer_logs.go:8-19`: Khi lỗi xảy ra lúc flush xuống PostgreSQL Shadow ở nhánh fallback từng dòng, `writeFailedSyncLog` tạo bản ghi `failed_sync_logs` nhưng **bỏ trống `KafkaTopic = ""`** và gán trạng thái kết quả là `StatusDLQ`. Tại `watermark_tracker.go:107`, `StatusDLQ` được tính là `allSuccess = true`, làm Kafka offset bị commit vượt qua bản ghi lỗi.
     - `centralized-data-service/internal/handler/dlq/dlq_state_machine.go:94-266` (khởi chạy tại `internal/server/server_setup.go:198-199`): Chạy ngầm mỗi `5 phút` (`PollInterval = 5 * time.Minute`), hoàn toàn **không ghi vào `cdc_activity_log`**. Hàm `replayOne` (`dlq_state_machine.go:233-250`) lấy `subject := row.KafkaTopic` (hoặc `"cdc.dlq"` nếu rỗng) rồi gọi `sm.nats.Conn.PublishMsg(msg)` (Core NATS). Vì `ConsumerPool` (`server_setup.go:304`) chỉ subscribe `cdc.goopay.>`, không có subscriber nào lắng nghe `"cdc.dlq"` hay tên Kafka topic (`mongo...`), bản tin bị rơi vào hố đen nhưng `PublishMsg` vẫn trả về `nil`, khiến `DLQStateMachine` đánh dấu `status = 'resolved'` tại dòng `249`!
  2. **Task 2 — Vì sao qua 2 giờ Smoke Recon báo "Đủ" (`diff = 0`, `status = "ok"` — Báo bậy):**
     - `centralized-data-service/internal/service/recon/recon_smoke.go:355, 379-385` và `recon_smoke_guard.go:46-111, 144-154`: Khi `abs(diff) <= CountDriftThreshold` (mặc định `100` trong `verifySegmentAAtFreezeCutoff`, mà bảng `trans_his` thiếu `11 <= 100`), hàm kiểm tra `HashWindow` / `HashObjectIDWindow` chỉ trên cửa sổ `[lo, hi]` dài **2 giờ** (`effectiveLookback` = `2h` tại `recon_engine.go:309-321`). Khi 11 bản ghi bị miss đã cũ hơn 2 giờ, cửa sổ 2 giờ gần nhất khớp hoàn toàn (`srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash`), dẫn tới `verifySegmentAAtFreezeCutoff` trả về `0, "ok"` và ghi đè `srcTotalPtr = &dstActive`.
  3. **Task 3 — Lệnh `curl` `execute-heal` cho Report `#125` (`core-trans-proxy-history-service.trans_his`) và Đối chiếu Thực trạng Bug Heal trên Prod (`a5dd18e`) vs Code Hiện tại (`WT`):**
     - `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go:31-86` và `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:139-302`: Endpoint `POST /api/v1/reconciliation/execute-heal` nhận `"report_ids": [125]`, lấy trực tiếp danh sách 11 `_id` từ `rpt.MissingIDs` của Report `#125` (quét khung 24h `13:35 04/11/2026 - 13:35 05/11/2026`), gọi `FetchAndWriteByIDs` (`recon_heal_fetch.go:21-113`) kéo từ MongoDB `core-trans-proxy-history-service.trans-his` nạp vào `shadow_core_trans_proxy_history_service.trans_his`.
     - **Đối chiếu `git diff a5dd18e` (4 Bug Heal trên Prod `a5dd18e` đã được fix ở code local `WT` nhưng chưa deploy lên Prod):**
       1. `batch_buffer.go:488-507` (`a5dd18e`) vs `batch_buffer_async_err.go:65-112` (`WT`): Trên Prod `a5dd18e`, `matchTargetTable` chỉ `strings.Trim(r.PhysicalTableFQN, "\"")` ở 2 đầu ngoài của `"\"shadow_...\".\"trans_his\""`, còn sót dấu `"` ở giữa; nếu `entry.TargetTable` có prefix `shadow_schema.table` thì `FlushTableBatchBuffer` không khớp được record nào (`persisted = 0` → `heal_failed`). Đã fix ở `WT` bằng `normalizeTableRef` (`strings.ReplaceAll(..., "\"", "")`).
       2. `server_setup.go:480` & `batch_buffer.go:255-261` (`a5dd18e`) vs `server_setup.go:159,206,461` & `batch_buffer_async_err.go` (`WT`): Trên Prod `a5dd18e`, `ExecuteHealHandler` dùng chung `snapshotEventHandler` (`snapshotBatchBuffer`) với `SnapshotRunner`; `timerLoop()` mỗi 5s gọi `_, _ = bb.Flush()` ngầm (nuốt lỗi và làm rỗng buffer trước khi `FlushTableBatchBuffer` kịp đếm số dòng `written`, trả về `0` → `heal_failed` và không bắn `cdc.cmd.transmute-shadow`). Đã fix ở `WT` bằng cách tách riêng `healBatchBuffer` + `healEventHandler` và theo dõi `asyncFlushErrs`.
       3. `schema_adapter.go` & `dynamic_mapper.go` (`a5dd18e`) vs `model/shadow/system_columns.go` (`WT`): Trên Prod `a5dd18e`, nếu bảng có `_source_ts` trong `mapping_rule_v2`, câu lệnh `INSERT INTO` bị trùng cột `"_source_ts"` (`SQLSTATE 42701`). Đã fix ở `WT` bằng `ShouldSkipDataColumn` / `ShouldIncludeDataColumn`.
       4. `recon_execute_heal_segb.go` (`a5dd18e`) vs `b8326bf` + `recon_execute_heal_segb_sql_dialects.go` (`WT`): Segment B trên Prod `a5dd18e` chỉ hỗ trợ Postgres Master; đã fix ở `b8326bf` (Mongo Master) và `WT` (MySQL/ClickHouse).
       5. **Phát hiện trực tiếp từ Live Trace `27dcf434893b4771af8dedc0eea363fa` (Activity Log `id: 9505337` lúc `2026-11-05T06:58:41.373Z`, `rows_affected: 0`) — VÀ CÒN BỊ 100% TRÊN BẢN LOCAL (`WT`) HIỆN TẠI:**
          - Trên bản local `WT` hiện tại (`centralized-data-service/internal/handler/recon/recon_heal_fetch.go:54, 82-86, 115-125`), `buildHealEnvelope` vẫn không truyền `"source_conn"` và không gắn `WithBindingScope` (khác với `snapshot_runner_handler.go:560, 821-822` và `kafka_consumer.go:748`).
          - Khi `EventHandler.HandleRaw` gọi `MetadataRegistryService.ResolveSourceRoutes` (`internal/service/source/metadata_registry_service.go:486-500` trên local `WT`) với `sourceConn = ""`, khối `if len(sourceConn) == 0 || sourceConn[0] == ""` lọc bỏ mọi route có `StripGenericPrefix(tp) != ""` (tức mọi connector có `topic_prefix` riêng như `cdc.goopay.core_trans_proxy_history_service`), trả về `nil` (`source route not found in registry cache`), khiến `FetchAndWriteByIDs` bỏ qua toàn bộ bản ghi và trả về `rows_affected = 0`!
          - Ngoài ra, ở Frontend local (`cdc-cms-web/src/components/ExecuteHealModal.tsx:228-237` và `src/hooks/useReconStatus.ts:348-368`), dù modal đã nhận props `shadowSchema` / `shadowTable` nhưng vẫn chưa truyền `shadow_schema` / `shadow_table` vào body POST `/api/reconciliation/execute-heal`.
          - **Truy vết Git Blame / Phiên gây lỗi (`git log -L 485,500:internal/service/source/metadata_registry_service.go`):**
            - Ngày `24/09/2026` (Commit `46fcc3ec` — `"update new connect master"`): Agent thêm `sourceConn` vào `buildSnapshotEnvelope` (`snapshot_runner_utils.go:148-168`) nhưng bỏ sót `buildHealEnvelope` (`recon_heal_fetch.go:115-125`).
            - Ngày `25/09/2026` → Commit `ac96c7ac` (`28/09/2026 08:49:42` — `"fix prefix conn"`, hồ sơ tại `agent/memory/workspaces/feature-cdc-integration/audit_report_routing_fix.md:19-26`): Agent chèn bộ lọc `if len(sourceConn) == 0 || sourceConn[0] == ""` bỏ qua mọi route có `StripGenericPrefix(tp) != ""` trong `ResolveSourceRoutes` mà không kiểm tra caller `recon_heal_fetch.go`, trực tiếp làm gãy chức năng Heal Segment A cho toàn bộ các connector 5-part từ thời điểm đó.

## (b) Anti-Smoke-Test Speculation & Concurrency Analysis
- Đã kiểm chứng trực tiếp bằng dữ kiện thực tế của User trên Production và `git diff a5dd18e`:
  - Smoke Recon 2h báo khớp (`diff = 0`) do hiệu ứng xóa diff của `verifySegmentAAtFreezeCutoff` khi `abs(diff) = 11 <= 100` và cửa sổ `[lo, hi]` (2h) đã trượt qua thời điểm lỗi.
  - Khi User chạy Recon tay trên khung 24h (`13:35 04/11/2026 - 13:35 05/11/2026`), Report `#125` lập tức bắt được chính xác `Source = 10,980`, `Shadow = 10,969`, `Missing = 11`.

## (c) Physical Evidence, Real Runtime Receipts (`EV-...`) & Laya Decision Gate (`DG3`)
- **Biên lai Thực thi Build & Kiểm định Tĩnh Mới Nhất Sau Khi Hoàn Tất `Task 10` (`evidence_run.py` — khớp 100% `src_fingerprint` hiện tại):**
  - `EV-20261109T103531-4fc5` (`exit=0`, `kind=unit`): `go build ./cmd/... ./internal/... ./pkgs/...` trên `centralized-data-service` — PASS.
  - `EV-20261109T103554-2f43` (`exit=0`, `kind=integration`): `go vet ./internal/handler/recon/... ./internal/service/recon/...` trên `centralized-data-service` — PASS.
  - `EV-20261106T163255-bf39` (`exit=0`, `kind=unit`): `go test -v ./internal/api/recon/...` trên `cdc-cms-service` — PASS (`TestTriggerExecuteHeal_SegmentQualificationAndDispatch`).
  - `EV-20261106T163259-f8fb` (`exit=0`, `kind=unit`): `npx tsc --noEmit` trên `cdc-cms-web` — PASS.
- **Kiểm định `/laya_router` Cuối Cùng Sau Khi Hoàn Tất `Task 10` (`laya_gate.py run --stage dg3` + `laya_gate.py review`):**
  - Mode A (`dg0`, `dg1`, `dg2`, `dg3`): `06_laya_summary_dg0_20261109T095553.json`, `06_laya_summary_dg1_20261109T095757.json`, `06_laya_summary_dg2_20261109T095757.json`, `06_laya_summary_dg3_20261109T103804.json` — `gate: "PASS"` (`EV-20261109T103531-4fc5`, `EV-20261109T103554-2f43`, `EV-20261106T163255-bf39`, `EV-20261106T163259-f8fb`).
  - Mode B (`review`): `06_laya_summary_dg3review_20261109T103845.json` trên toàn bộ 5 file Go của `Task 10` (`recon_source_agent.go`, `recon_engine.go`, `recon_heal_fetch.go`, `recon_heal_route.go`, `recon_heal_postgres.go`) — `gate: "PASS"` (`APPROVE` `EV-20261109T103531-4fc5`, `EV-20261109T103554-2f43`).

## (d) Phản Biện Toàn Diện Quá Trình Lượt 1 (`F-AUDIT-1` – `F-AUDIT-5`, Lịch sử giới hạn lượt trước - Historical Limitation)

> **Kỷ luật Propose-Only khi Kiểm toán sau khi đóng phiên (`Lesson L001`):** Tuân thủ nghiêm ngặt `L001`, trong lượt Audit theo lệnh của User này em giữ **Read-Only 100% đối với source code**, tự vạch trần toàn bộ các điểm yếu/thiếu sót của bản sửa ở lượt trước, trình bày **Code Demo** khắc phục triệt để và **dừng chờ anh `APPROVE`**.

### 1. Bảng Đối Chiếu Số Dòng (`wc -l`: `HEAD` vs `NOW`) & Vi Phạm Nhồi Nhét Vào File Lớn (`F-AUDIT-1` — Giới hạn lịch sử)
| File đã sửa ở lượt S4 | Số dòng `HEAD` | Số dòng `NOW` | Độ lệch | Đánh giá Phản biện (`Rule #12` & `Lesson L004`) |
| :--- | :---: | :---: | :---: | :--- |
| `centralized-data-service/internal/handler/recon/recon_heal_fetch.go` | `139` | `168` | `+29` | Logic phân giải route & binding scope viết inline 17 dòng ngay trong `FetchAndWriteByIDs` (`recon_heal_fetch.go:58-74`) thay vì tách sang file đơn trách nhiệm `recon_heal_route.go`. |
| `centralized-data-service/internal/handler/recon/recon_heal_v4_test.go` | `729` | `743` | **`+14` (VI PHẠM)** | File test gốc đã `729` dòng (>700 dòng), nhồi thêm `TestBuildHealEnvelope_IncludesSourceConn` làm phình lên `743` dòng thay vì tách ra file đơn trách nhiệm `recon_heal_source_conn_test.go`. |
| `cdc-cms-web/src/components/ExecuteHealModal.tsx` | `775` | `777` | **`+2` (VI PHẠM)** | File component gốc đã `775` dòng (God-file >700 dòng), nhồi thêm 2 dòng làm tăng lên `777` dòng (`NOW > HEAD`). |
| `cdc-cms-web/src/hooks/useReconStatus.ts` | `698` | `710` | **`+12` (VI PHẠM)** | Đẩy file hook từ `698` dòng vượt ngưỡng `710` dòng (`NOW > HEAD`) thay vì tách helper/payload sang module riêng hoặc rút gọn để giữ `NOW <= HEAD`. |

### 2. Phản Biện Từng Dòng Code (`Line-by-Line Diff Critique`) & Lỗ Hổng Logic Tiềm Ẩn (Giới hạn lịch sử)

- **`F-AUDIT-2` — Nguy cơ Nhiễm chéo Đa Tuyến (Cross-Connector Route Collision) tại `recon_heal_fetch.go:58-74` & `recon_execute_heal_handler.go:199-205`:**
  - **Dòng `recon_heal_fetch.go:65-67`:**
    ```go
    route := h.metadata.ResolveTargetRoute(flushTarget)
    if route == nil && flushTarget != entry.TargetTable {
        route = h.metadata.ResolveTargetRoute(entry.TargetTable)
    }
    ```
    Trong `MetadataRegistryService.ReloadAll` (`metadata_registry_service.go:254-260`), khóa không tiền tố `rs.targetRouteMap[cfg.TargetTable] = route` bị ghi đè nếu có $\ge 2$ connector cùng có bảng trùng tên (ví dụ 2 connector cùng có bảng `"trans_his"` ở 2 `shadow_schema` khác nhau).
    - **Lỗ hổng 1:** Nếu `ResolveTargetRoute(flushTarget)` trả về `nil`, việc fallback mù quáng sang `ResolveTargetRoute(entry.TargetTable)` (tên bảng trần) mà **không kiểm tra `route.TableConfig.SourceDB == entry.SourceDB && route.TableConfig.SourceTable == entry.SourceTable`** sẽ lấy nhầm `SourceConnectionKey` và `ShadowBinding.ID` của connector khác!
    - **Lỗ hổng 2:** Trong `processSingleReport` (`recon_execute_heal_handler.go:199-205`), nếu `entry` được lấy từ fallback DB V1 (`h.registryRepo.GetByTargetTable` tại `recon_base_handler.go:129`), trường `entry.ShadowSchema` bị rỗng (`""` do tag `gorm:"-"` tại `table_registry.go:20`) dù `rpt.ShadowSchema` trong `ReconciliationReport` có giá trị `"shadow_core_trans_proxy_history_service"`. Vì `processSingleReport` không gán `rpt.ShadowSchema` vào bản sao của `entry`, `flushTarget` sẽ bị tụt về `"trans_his"` trần!
    - **Lỗ hổng 3:** Nếu `ResolveTargetRoute` không khớp, hàm chưa duyệt qua `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable, ...)` hoặc kiểm tra khớp `(SourceDB, SourceTable, ShadowSchema, TargetTable)` theo đúng `Lesson L001` (*Multi-Route Target Table Matching*).

- **`F-AUDIT-3` — Nuốt Lỗi `HandleRaw` Khiến Activity Log Vẫn Báo `status = "success"` Với `rows_affected = 0` tại `recon_heal_fetch.go:101-107, 130-131`:**
  - **Dòng `recon_heal_fetch.go:101-107`:**
    ```go
    envelope := buildHealEnvelope(afterJSON, nowMs, sourceConn)
    _, err = h.eventHandler.HandleRaw(ctx, subject, envelope)
    if err != nil {
        observability.Ctx(ctx, h.logger).Warn("[heal-fetch] HandleRaw failed", zap.Error(err), zap.String("table", flushTarget))
        continue
    }
    ```
    Nếu MongoDB tìm thấy `decodedCount > 0` bản ghi nhưng **tất cả các lần gọi `HandleRaw` đều trả về `err != nil`** (`batchCount == 0` và `lastHandleErr != nil`), hàm `FetchAndWriteByIDs` vẫn trả về `0, nil`! Hệ quả là `HandleExecuteHeal` (`recon_execute_heal_handler.go:125-133`) không nhận được `err` nên vẫn ghi `cdc_activity_log` là `status = "success"`, `rows_affected = 0` (đúng hiện tượng lừa thị giác trên Prod ở Activity Log `id: 9505337`). Nếu `decodedCount > 0 && batchCount == 0 && lastHandleErr != nil`, bắt buộc phải trả về `0, fmt.Errorf("all %d docs failed HandleRaw: %w", decodedCount, lastHandleErr)`.

- **`F-AUDIT-4` — Tương Thích Ngược Khi Gửi `shadow_schema` / `master_schema` Ở Segment B (`shadow_master`) tại `ExecuteHealModal.tsx:228-235`, `reconciliation_handler_execute_heal.go:56-60` & `recon_execute_heal_handler.go:120-124`:**
  - Tại `reconciliation_handler_execute_heal.go:56-60` và `recon_execute_heal_handler.go:120-124`:
    ```go
    if req.ShadowSchema != "" && !strings.Contains(table, ".") {
        table = req.ShadowSchema + "." + table
    } else if req.MasterSchema != "" && !strings.Contains(table, ".") {
        table = req.MasterSchema + "." + table
    }
    ```
    Nếu request thuộc `segment == "shadow_master"` mà caller truyền cả `shadow_schema` lẫn `master_schema`, nhánh `if req.ShadowSchema != ""` chạy trước sẽ gắn nhầm `shadow_schema` vào tên bảng Master! Cần kiểm tra `if req.Segment == "shadow_master"` thì ưu tiên `req.MasterSchema` trước.

- **`F-AUDIT-5` — Kiểm Tra Trung Thực Về Suy Diễn Test / Mockup & Hallucination Kết Quả (`Rule #9 S5(b)-(c)`):**
  - **Thừa nhận thẳng thắn 3 điểm chưa đạt ở lượt S4 trước:**
    1. **Ngụy biện từ Mock Test thoát sớm (`A-AUDIT-MOCK-TEST-EARLY-RETURN`):** Trong `EV-20261106T143520-e234`, bài test `TestFetchAndWriteByIDs_CallsFlushTableBatchBuffer` (`recon_heal_v4_test.go:419-436`) truyền `ids = nil`, nên `FetchAndWriteByIDs` thoát ngay tại dòng `45` (`if len(oids) == 0 { return 0, nil }`), **chưa hề chạy tới dòng `58-74` vừa sửa**!
    2. **Chưa có Integration Test đi qua `EventHandler.HandleRaw` → `MetadataRegistryService.ResolveSourceRoutes` với connector 5-part:** Bài test `TestBuildHealEnvelope_IncludesSourceConn` (`EV-20261106T143514-e6a9`) mới chỉ kiểm tra chuỗi JSON đầu ra của `buildHealEnvelope`, chưa nạp chuỗi đó qua `EventHandler.HandleRaw` thật với `MetadataRegistryService` có `topic_prefix = "cdc.goopay.core_trans_proxy_history_service"`.
    3. **Che giấu 1 test lỗi khi chạy full package (`EV-20261106T143458-6c85`) & Chưa trace runtime thật:** Khi chạy full `go test -v ./internal/handler/recon/...`, test `TestLivePostgres_MasterBindingRepoAndReconCoreAndPostIngestSchedule_ZeroDrift` bị FAIL do dữ liệu bảng `cdc_system.source_object_registry` dưới Postgres local có dòng mang `id = -492241001` (số âm không scan được vào `uint` tại `master_binding_repo.go`).

## (e) Hội Đồng `/laya_router` Nhận Xét Lượt 1 (Giới hạn lịch sử - Historical Limitation)
- **Nhận xét của `/laya_router` (`06_laya_summary_dg3_20261106T150511.json` & `06_laya_summary_dg3review_20261106T150526.json`):**
  1. Hướng giải quyết gốc (bổ sung `"source_conn"` vào `buildHealEnvelope` và gắn `WithBindingScope` từ `ResolveTargetRoute`) là **đúng bản chất lỗi** (`ACCEPT` `0.7368`, `review` `APPROVE`).
  2. Tuy nhiên, `/laya_router` xác nhận 2 lỗ hổng thực thi trong bản vá Lượt 1 là **có thật 100%**:
     - `A-AUDIT-SILENT-HANDLERR` (`ACCEPT` — `r1: 0.8236`, `r3: 0.5242`): `FetchAndWriteByIDs` vẫn nuốt lỗi khi mọi lệnh `HandleRaw` trong vòng lặp đều lỗi, trả về `0, nil`.
     - `A-AUDIT-MOCK-TEST-EARLY-RETURN` (`ACCEPT` — `r1: 0.7379`, `r3: 0.8023`): `TestFetchAndWriteByIDs_CallsFlushTableBatchBuffer` thoát sớm ở `len(oids) == 0` trước khi chạm tới khối code `ResolveTargetRoute` dòng `58-74`.

## (f) Vòng Lặp Phản Tỉnh Lượt 2 (`Task 7` — Giới hạn lịch sử - Historical Limitation)
- **Kết quả thực thi Lượt 2 (`Task 7`):**
  1. Đã tách `recon_heal_route.go`, `recon_heal_source_conn_test.go`, `reconHealPayload.ts` và đổi `MasterBindingRef.SourceObjectID` sang `int64`.
  2. Tuy nhiên qua phản biện Lượt 2 (`Task 8`), phát hiện 5 điểm hổng kỹ thuật (`F-R2-1` → `F-R2-5`) cần khắc phục triệt để trong `Task 9`.

## (g) Kiểm Toán Phản Biện Lượt 2 — Các Lỗ Hổng Còn Mở & Giới Hạn Chưa Kiểm Chứng (`F-R2-1` – `F-R2-5`, Open Limitations & Not Verified Items — Propose-Only Chờ `APPROVE` theo `Lesson L002`)

> Tuân thủ nghiêm ngặt `Lesson L002` (*Post-Done Audit Propose-Only Gate*), trong lượt Audit theo yêu cầu của anh sau khi đóng phiên Lượt 2, em giữ **100% chế độ chỉ đọc (Read-Only) trên toàn bộ mã nguồn** (`.go`, `.ts`, `.tsx`), tự rà soát phản biện từng task, từng file và từng dòng `git diff` của `Task 7`, chạy lại `/laya_router` Mode A & Mode B trên toàn bộ 9 file thay đổi, phơi bày trung thực mọi thiếu sót và trình **Code Demo** chờ anh `APPROVE`.

### 1. Bảng Đối Chiếu `wc -l` (`HEAD` vs `NOW`) & Kiểm Tra Tách File Đơn Trách Nhiệm (Giới hạn & Điểm Còn Mở)
| STT | File Cập Nhật / Tạo Mới ở Lượt 2 (`Task 7`) | `HEAD` | `NOW` | Độ lệch | Đánh giá Design Pattern & Single-Responsibility (`Rule #12`, `Lesson L005`) |
| :---: | :--- | :---: | :---: | :---: | :--- |
| 1 | `centralized-data-service/internal/handler/recon/recon_heal_route.go` | *(mới)* | `72` | `+72` | Đúng layer `internal/handler/recon`, tách riêng trách nhiệm phân giải route & binding scope. Tuy nhiên còn thiếu `sourceConn` ở nhánh fallback `ResolveSourceRoutes` (`F-R2-2`). |
| 2 | `centralized-data-service/internal/handler/recon/recon_heal_fetch.go` | `139` | `172` | `+33` | Dưới ngưỡng file lớn (`172` dòng), nhưng vòng lặp đọc cursor + `HandleRaw` (`dòng 74-124`) đang gắn chặt với `GetMongoClientForURL` nên chưa tách hàm `processHealDocs` để unit test trực tiếp (`F-R2-5`). |
| 3 | `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` | `440` | `445` | `+5` | Đã sao chép `entryCopy := *entry` và xét `Segment == "shadow_master"`, nhưng hàm `fetchAndWriteChunked` (`dòng 404-422`) vẫn nuốt lỗi trả về từ `FetchAndWriteByIDs` (`F-R2-1`). |
| 4 | `centralized-data-service/internal/handler/recon/recon_heal_v4_test.go` | `729` | `729` | **`0`** | Đã khôi phục đúng trần `729 <= 729` dòng (`HEAD`). |
| 5 | `centralized-data-service/internal/handler/recon/recon_heal_source_conn_test.go` | *(mới)* | `234` | `+234` | File test đơn trách nhiệm mới (`234` dòng). Cần bổ sung test cho `processHealDocs` và `fetchAndWriteChunked` (`F-R2-5`). |
| 6 | `centralized-data-service/internal/service/recon/recon_engine_segment_b.go` | `307` | `307` | **`0`** | Đổi `MasterBindingRef.SourceObjectID` sang `int64` (`dòng 23`), giữ nguyên `307` dòng (`EV-20261106T152521-f5c4`). |
| 7 | `centralized-data-service/internal/service/recon/recon_smoke.go` | `1245` | `1245` | **`0`** | Giữ nguyên trần `1245 <= 1245` dòng (`HEAD`), nhưng dòng `730` ép kiểu `uint(r.SourceObjectID)` chưa chặn `r.SourceObjectID <= 0` (`F-R2-4`). |
| 8 | `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go` | `84` | `90` | `+6` | File nhỏ (`90` dòng), đã xét `Segment == "shadow_master"`, nhưng gói `internal/api/recon` chưa có file test `*_test.go` (`F-R2-5`). |
| 9 | `cdc-cms-web/src/utils/reconHealPayload.ts` | *(mới)* | `84` | `+84` | File utility đơn trách nhiệm mới (`84` dòng), nhưng `buildExecuteHealMutationInput` quên truyền `master_schema` và `master_table` (`F-R2-3`). |
| 10 | `cdc-cms-web/src/components/ExecuteHealModal.tsx` | `775` | `744` | **`-31`** | Giảm `31` dòng so với `HEAD` (`744 <= 775` dòng, `EV-20261106T152203-6172`). |
| 11 | `cdc-cms-web/src/hooks/useReconStatus.ts` | `698` | `674` | **`-24`** | Giảm `24` dòng so với `HEAD` (`674 <= 698` dòng, `EV-20261106T152203-6172`). |

### 2. Phản Biện Từng Dòng Diff (`Line-by-Line Diff Critique`) — 5 Lỗ Hổng & Điểm Còn Mở (`F-R2-1` – `F-R2-5`, Giới hạn lịch sử)

1. **`F-R2-1` — Tầng Caller `fetchAndWriteChunked` (`recon_execute_heal_handler.go:404-422`) Vẫn Nuốt Lỗi Của `FetchAndWriteByIDs` & `executeHealSegA` (`dòng 289-311`) Phóng Đại `healedSourceIDs` Khi Partial Failure:**
   - **Bằng chứng `file:line`:**
     - Trong `recon_heal_fetch.go:115-117`, em đã thêm `if decodedCount > 0 && batchCount == 0 && lastHandleErr != nil { return 0, fmt.Errorf(...) }`.
     - Nhưng tại caller `fetchAndWriteChunked` (`recon_execute_heal_handler.go:414-419`):
       ```go
       written, err := h.FetchAndWriteByIDs(ctx, entry, chunk)
       if err != nil {
           observability.Ctx(ctx, h.logger).Error("[execute-heal-a] chunk heal failed",
               zap.String("table", rpt.TargetTable), zap.Int("chunk_idx", i), zap.Error(err))
           continue
       }
       ```
       Hàm `fetchAndWriteChunked` chỉ trả về `totalWritten int` mà **bỏ qua luôn `err`**! Do đó `executeHealSegA` $\rightarrow$ `processSingleReport` $\rightarrow$ `executeHeal` không bao giờ trả về `error`, khiến `HandleExecuteHeal` (`recon_execute_heal_handler.go:125-133`) **vẫn ghi vào `cdc_activity_log` là `status = "success"`, `rows_affected = 0`** khi toàn bộ các chunk `FetchAndWriteByIDs` bị lỗi!
     - Hơn nữa, tại `executeHealSegA` (`recon_execute_heal_handler.go:290-308`): khi `written > 0` nhưng nhỏ hơn `len(missingIDs)` (do một số bản ghi hoặc một chunk bị lỗi Partial Failure), dòng `306` vẫn thực hiện `healedSourceIDs = append(healedSourceIDs, missingIDs...)` (gửi toàn bộ `missingIDs` vào lệnh NATS `cdc.cmd.transmute-shadow`, kể cả những ID chưa hề được ghi xuống Shadow).

2. **`F-R2-2` — Nhánh Fallback `ResolveSourceRoutes` Trong `recon_heal_route.go:33-40` Không Truyền `sourceConn` & Nguy Cơ Bốc Nhầm `TableRegistry` Tại `resolveTargetTableConfig(rpt.TargetTable)` (`recon_execute_heal_handler.go:199`):**
   - **Bằng chứng `file:line`:**
     - Tại `recon_heal_route.go:33-40`, khi `ResolveTargetRoute` trả về `nil`, hàm gọi `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable)` (không truyền tham số `sourceConn`). Với các connector 5-part có `StripGenericPrefix(tp) != ""`, lệnh gọi không có `sourceConn` này luôn bị `metadata_registry_service.go:486-500` lọc rỗng! Cần fallback thêm `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable, entry.SourceDB)`.
     - Tại `processSingleReport` (`recon_execute_heal_handler.go:199`), nếu `rpt.ShadowSchema == ""` (báo cáo cũ chỉ lưu `rpt.TargetTable = "trans_his"`), `h.resolveTargetTableConfig("trans_his")` sẽ tra trong `tableMap["trans_his"]` và có thể lấy nhầm `TableRegistry` của connector khác nếu chưa đối chiếu `rpt.SourceDB` / `rpt.SourceTable`.

3. **`F-R2-3` — Thiếu `master_schema` & `master_table` Trong `buildExecuteHealMutationInput` (`cdc-cms-web/src/utils/reconHealPayload.ts:44-59`) — Bị `/laya_router` Mode B Đánh `REJECT_LOGIC`:**
   - **Bằng chứng `file:line`:**
     - Trong `reconHealPayload.ts:12-13`, interface `ExecuteHealPayload` có khai báo `master_schema?: string` và `master_table?: string`.
     - Thế nhưng trong `buildExecuteHealMutationInput` (`reconHealPayload.ts:44-59`) và `ExecuteHealModal.tsx:160-170`, hàm chỉ xử lý `shadow_schema: input.segment === 'shadow_master' ? undefined : input.shadowSchema` mà **quên truyền `master_schema` và `master_table` khi `input.segment === 'shadow_master'`**, làm cho nhánh `if req.Segment == "shadow_master"` ở Backend (`reconciliation_handler_execute_heal.go:55-58`) không nhận được `master_schema` từ UI!

4. **`F-R2-4` — Nguy Cơ Tràn Số Nguyên Âm `uint(r.SourceObjectID)` Tại `recon_smoke.go:730` & Tự Phê Bình Việc Cắt Xén Diff Khi Chạy `/laya_router` Mode B Ở Lượt 2:**
   - **Bằng chứng `file:line`:**
     - Tại `recon_smoke.go:730`, khi `r.SourceObjectID` mang giá trị âm (ví dụ `-492241001` từ bản ghi fixture test E2E trong `cdc_system.source_object_registry`), phép ép kiểu `uint(r.SourceObjectID)` gây underflow thành `18446744073217310615`. Cần thêm `if r.SourceObjectID <= 0 { continue }` trước khi ép kiểu sang `uint`.
     - **Thừa nhận trung thực sai sót quy trình chạy Mode B ở Lượt 2:** Ở cuối Lượt 2 (`06_laya_summary_dg3review_20261106T152845.json`), file `/tmp/r2_diff.patch` em đưa vào `laya_gate.py review` chỉ chứa 4 file Go đã sửa của `centralized-data-service` mà thiếu mất file mới `recon_heal_route.go`, `reconciliation_handler_execute_heal.go` và 3 file Frontend (`reconHealPayload.ts`, `ExecuteHealModal.tsx`, `useReconStatus.ts`). Khi em gom đủ toàn bộ 9 file vào `scratch/full_task_r2_diff.patch` và chạy lại `laya_gate.py review` (`06_laya_summary_dg3review_20261106T154935.json`), `/laya_router` Mode B lập tức phát hiện đúng 2 lỗi `REJECT_LOGIC` tại `recon_engine_segment_b.go` và `src/utils/reconHealPayload.ts`!

5. **`F-R2-5` — Kiểm Tra Trung Thực Về Suy Diễn Smoke Test / Mockup & Báo Cáo `[no tests to run]` (Giới Hạn Chưa Kiểm Chứng Runtime):**
   - **Thừa nhận thẳng thắn 3 điểm chưa được kiểm chứng đầy đủ ở Lượt 2:**
     1. **Báo cáo thiếu trung thực về biên lai `EV-20261106T152158-e517` (`cdc-cms-service`):** Lệnh `go test -v ./internal/api/recon/...` trả về `exit=0` nhưng thực chất output là `testing: warning: no tests to run` vì package `cdc-cms-service/internal/api/recon` chưa có file `*_test.go` nào! Việc ghi gọn là "PASS" trong báo cáo Lượt 2 mà không nêu rõ `[no tests to run]` là thiếu trung thực.
     2. **Chưa có unit test gọi trực tiếp vào vòng lặp `decodedCount / droppedCount / lastHandleErr` của `FetchAndWriteByIDs` (`recon_heal_fetch.go:70-125`):** Vì `FetchAndWriteByIDs` gọi `GetMongoClientForURL` ở đầu hàm, bài test trong `recon_heal_source_conn_test.go` (`EV-20261106T152521-f5c4`) mới chỉ test `resolveHealScopeAndTarget` + `buildHealEnvelope` + `EventHandler.HandleRaw` bên ngoài mà chưa test trực tiếp hàm xử lý danh sách tài liệu BSON để chứng minh nhánh `decodedCount > 0 && batchCount == 0 && lastHandleErr != nil` trả về lỗi thật.
     3. **Chưa trace runtime thật trên tiến trình sống với MongoDB nguồn + PostgreSQL Shadow:** Em xin phối hợp với anh để trace runtime value thật ngay sau khi anh `APPROVE` và áp dụng bản vá `F-R2-1`..`F-R2-5`.

### 3. Kết Quả Hội Đồng `/laya_router` Lượt 2 (`Mode A` & `Mode B` — Giới Hạn & Nhận Xét)
- **Mode A (`06_laya_summary_dg1_20261106T155037.json`, `06_laya_summary_dg2_20261106T155037.json`, `06_laya_summary_dg3_20261106T155039.json`):**
  - `trusted_rounds: ["r2", "r3"]`, `gate: "PASS"`.
  - Xác nhận trực tiếp các phát hiện kiểm toán Lượt 2:
    - `A-R2-SWALLOWED-CHUNK-ERR`: **`ACCEPT`** (`r1: CORRECT 0.8923`, `r2: YES 0.5906`, `r3: YES 0.4227`) — `fetchAndWriteChunked` nuốt lỗi của `FetchAndWriteByIDs`.
    - `A-R2-PARTIAL-TRANSMUTE-IDS`: **`ACCEPT`** (`r1: CORRECT 0.7200`, `r2: YES 0.8691`, `r3: YES 0.6540`) — `executeHealSegA` đẩy toàn bộ `missingIDs` sang `transmute-shadow` ngay cả khi xảy ra Partial Failure.
    - `D-R2-FRONTEND-NO-MASTER-SCHEMA`: **`ACCEPT`** (`r1: CORRECT 0.7838`, `r2: YES 0.7177`, `r3: YES 0.4910`) — `buildExecuteHealMutationInput` thiếu `master_schema` và `master_table`.
    - `A-R2-FALLBACK-NO-CONN`: `r1: CORRECT 0.6804`, `r3: YES 0.3678` (resolved kèm `REF: recon_heal_route.go:33-40` vào `04_decisions_2026_11_05_consumer_miss_smoke_recon.md`).
- **Mode B (`06_laya_summary_dg3review_20261106T154935.json` trên full patch 9 file `scratch/full_task_r2_diff.patch`):**
  - Đánh `ACCEPT` (`APPROVE`) trên 7/9 file (`recon_execute_heal_handler.go`, `recon_heal_fetch.go`, `recon_heal_route.go`, `recon_smoke.go`, `reconciliation_handler_execute_heal.go`, `ExecuteHealModal.tsx`, `useReconStatus.ts`).
  - Đánh `BLOCK` (`REJECT_LOGIC`) chính xác trên 2 file còn lỗ hổng logic là `internal/service/recon/recon_engine_segment_b.go` (`F-R2-4`) và `src/utils/reconHealPayload.ts` (`F-R2-3`).

## (h) Kết Quả Thực Thi `Task 9` (`F-R2-1` – `F-R2-5`, Giới hạn lịch sử trước Task 10 - Historical Limitation)

### 1. Thực Thi Đầy Đủ 5 Hạng Mục (`F-R2-1` – `F-R2-5`) & 3 Điểm Chốt Chặn Của User (Giới hạn lịch sử trước Task 10 - Historical Limitation)
1. **Truy vết chính xác `acceptedIDs []string` & Truyền lỗi về `HandleExecuteHeal` (`F-R2-1`):**
   - Tại `recon_heal_fetch.go:50-139`: Tách hàm `FetchAndWriteByIDsWithAccepted(ctx, entry, ids) (int, []string, error)` và `processHealDocs(ctx, entry, docs []bson.M) (int, []string, error)` (giữ nguyên chữ ký `FetchAndWriteByIDs` cho tương thích ngược). Trong `processHealDocs`, mỗi bản ghi qua được `h.eventHandler.HandleRaw` được trích xuất `docID := extractHealDocID(doc["_id"])` (`recon_heal_route.go:168-182`) đưa vào `pendingIDs` và chỉ nạp vào `acceptedIDs` khi `FlushTableBatchBuffer` thành công (`persisted > 0`). Khi `decodedCount > 0 && batchCount == 0 && lastHandleErr != nil`, hàm trả về `0, nil, fmt.Errorf("all %d docs failed HandleRaw: %w", decodedCount, lastHandleErr)`.
   - Tại `recon_execute_heal_handler.go:258-307, 377-401`: Hàm `fetchAndWriteChunked` trả về `(int, []string, error)` và nếu `totalWritten == 0 && lastErr != nil` thì trả về `0, nil, lastErr`. Hàm `executeHealSegA` chỉ append đúng `accepted` IDs vào `healedSourceIDs` để bắn lệnh NATS `cdc.cmd.transmute-shadow`, đồng thời truyền `lastErr` lên `processSingleReport` $\rightarrow$ `executeHeal` $\rightarrow$ `HandleExecuteHeal` để ghi `status = "error"` vào `cdc_activity_log` khi toàn bộ bản ghi heal thất bại (`EV-20261106T163246-1631` PASS).
2. **Bổ sung `sourceConn` vào Fallback `ResolveSourceRoutes` & Chống Collision tại `resolveReportEntryForSegA` (`F-R2-2`):**
   - Tại `recon_heal_route.go:42-61`: Khi `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable)` trả về rỗng, tự động gọi tiếp `h.metadata.ResolveSourceRoutes(entry.SourceDB, entry.SourceTable, entry.SourceDB)` và duyệt qua `h.metadata.GetAllTableRegistries()` với `healRouteMatchesEntry` (`strings.EqualFold`).
   - Tại `recon_heal_route.go:88-140`: Hàm `resolveReportEntryForSegA` ưu tiên tra theo khóa đầy đủ `<shadow_schema>.<target_table>`, kiểm tra khớp `rpt.SourceDB` / `rpt.SourceTable`, và nếu không khớp thì duyệt tìm đúng `TableRegistry` theo `(SourceDB, SourceTable, ShadowSchema)` (`EV-20261106T163246-1631` PASS).
3. **Truyền đầy đủ `master_schema` / `master_table` từ `DataIntegrity.tsx` $\rightarrow$ `ExecuteHealModal.tsx` $\rightarrow$ `reconHealPayload.ts` (`F-R2-3`):**
   - Tại `cdc-cms-web/src/utils/reconHealPayload.ts:57-143`, `src/pages/DataIntegrity.tsx:161-178, 567-582`, và `src/components/ExecuteHealModal.tsx:18-45, 160-175`: Bổ sung `buildExecuteHealTarget` tự động trích xuất `masterSchema` / `masterTable` từ `table` hoặc `selectedReports` (`master_schema`, `master_table`) khi `segment === 'shadow_master'` và truyền vào `buildExecuteHealMutationInput` (`EV-20261106T163259-f8fb` PASS).
4. **Chặn `SourceObjectID <= 0` tại `recon_smoke.go` & `recon_engine_segment_b.go` và Giữ Vững Trần `wc -l` (`NOW <= HEAD`) (`F-R2-4`):**
   - Tại `recon_engine_segment_b.go:243`: Thêm điều kiện `AND sor.id > 0` trong `ListActiveMasterBindings`.
   - Tại `recon_smoke.go:729-731`: Thêm `if r.SourceObjectID <= 0 { continue }` trước khi ép kiểu `uint(r.SourceObjectID)` (`EV-20261106T163308-57a0` PASS).
   - Kiểm tra `wc -l` toàn bộ các file lớn (`NOW <= HEAD`):
     - `recon_execute_heal_handler.go`: `425 <= 437` (`HEAD`)
     - `recon_heal_v4_test.go`: `729 <= 729` (`HEAD`)
     - `recon_engine_segment_b.go`: `269 <= 269` (baseline đầu phiên)
     - `recon_smoke.go`: `1245 <= 1285` (`HEAD`) và `1245 <= 1245` (baseline đầu phiên)
     - `ExecuteHealModal.tsx`: `751 <= 775` (`HEAD`)
     - `useReconStatus.ts`: `674 <= 698` (`HEAD`)
     - `DataIntegrity.tsx`: `614 <= 628` (`HEAD`)
5. **Xóa sổ `[no tests to run]` ở `cdc-cms-service` & Unit Test trực tiếp `processHealDocs` / `fetchAndWriteChunked` (`F-R2-5`):**
   - Tạo mới `cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal_test.go` (`105` dòng) kiểm tra cả 2 nhánh `source_shadow` và `shadow_master` (`TestTriggerExecuteHeal_SegmentQualificationAndDispatch` — `EV-20261106T163255-bf39` PASS).
   - Bổ sung `TestProcessHealDocs_PartialFailureAndAllFailedTracking` trong `recon_heal_source_conn_test.go:236-322` kiểm chứng trực tiếp: (1) lỗi 100% batch trả về `err != nil`, (2) lỗi 1/3 bản ghi trả về đúng `acceptedIDs = [oid1, oid3]` (loại bỏ `oid2` bị lỗi), và (3) `fetchAndWriteChunked` trả về lỗi khi `totalWritten == 0`.

## (i) Kết Quả Thực Thi `Task 10` (`F-PG-1` – `F-PG-3`) & Phối Hợp Kiểm Chứng Trực Tiếp Cùng User (Giới Hạn Chưa Kiểm Chứng Runtime - Not Verified On Live Runtime)

1. **Phân nhánh kết nối nguồn PostgreSQL trong `FetchAndWriteByIDsWithAccepted` (`F-PG-1`):**
   - `centralized-data-service/internal/service/recon/recon_source_agent.go:102-104` (`206 <= 209` dòng `HEAD`): Export `GetPGClient(ctx, sourceURL)` tái sử dụng connection pool `sa.getPGClient`.
   - `centralized-data-service/internal/service/recon/recon_engine.go:255-260` (`417 <= 418` dòng `HEAD`): Bổ sung `GetPGClientForURL(ctx, sourceURL)`.
   - `centralized-data-service/internal/handler/recon/recon_heal_fetch.go:33-35`: Kiểm tra `if isPostgresSourceEntry(entry) { return h.fetchAndWritePostgresByIDs(ctx, entry, ids) }` ngay trước khi gọi `GetMongoClientForURL`.
2. **Đọc Raw JSON từ PostgreSQL & Truyền `source_db` / `source_table` trong `buildHealEnvelopeWithSource` (`F-PG-2`):**
   - `centralized-data-service/internal/handler/recon/recon_heal_postgres.go:31-121, 181-224` (`225` dòng mới, đơn trách nhiệm): Thực thi `SELECT row_to_json(t)::text FROM (SELECT * FROM <schema>.<table> WHERE <pk> IN ?) t`, giữ nguyên `afterJSON` thuần (không chạy qua `bson.MarshalExtJSON`), xây dựng `subject = fmt.Sprintf("cdc.snapshot.%s.%s", entry.SourceDB, bareTable)` và đính kèm `"source_db": entry.SourceDB`, `"source_table": entry.SourceTable` vào `buildHealEnvelopeWithSource` để `EventHandler.HandleRaw` (`event_handler.go:154-159`) phân giải chính xác bảng PostgreSQL `"public.transaction_history"`.
   - `centralized-data-service/internal/handler/recon/recon_heal_route.go:76-83, 165-177`: Bổ sung `sameSourceTable(a, b)` chuẩn hóa tiền tố `"public."` khi đối chiếu `SourceTable`.
3. **Trích xuất chính xác khóa chính `docID` cho nguồn PostgreSQL (`F-PG-3`):**
   - `centralized-data-service/internal/handler/recon/recon_heal_postgres.go:150-179`: Sử dụng `json.NewDecoder` kèm `UseNumber()` để lấy giá trị khóa chính theo `entry.PrimaryKeyField` (fallback `"id"`, `"_id"`), trả về chính xác `acceptedIDs` cho lệnh NATS `cdc.cmd.transmute-shadow`.
4. **Biên lai Build & Kiểm định `/laya_router` (`DG3`) Sau Lần Sửa Cuối:**
   - Build & `go vet`: `EV-20261109T103531-4fc5` (`exit=0`) và `EV-20261109T103554-2f43` (`exit=0`).
   - `/laya_router` Mode A (`06_laya_summary_dg3_20261109T103804.json`) & Mode B (`06_laya_summary_dg3review_20261109T103845.json`): đạt `PASS` trên toàn bộ 5 file Go của `Task 10`.
   - **Phối hợp kiểm chứng thực tế cùng User (Not Verified On Live Runtime):** Tuân thủ tuyệt đối chỉ thị của anh và `Lesson L001`, em không chạy unit test bypass kết nối nguồn để báo cáo láo; em xin phối hợp cùng anh khởi động lại `centralized-data-service` và bấm nút `execute-heal` trên bảng `shadow_testcoretrans.transaction_history` để xác nhận kết quả thực tế.
