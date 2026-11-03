# 08. TASKS — `master_row_count_mismatch_oct30_20261030`

## Task T1: Điều tra nguyên nhân lệch số thống kê giữa Shadow (`250,137 + 317`) và Master (`250,240 + 347`) trên dòng `2026-10-30` và tư vấn có cần Truncate & chạy lại hay không
- **Mô tả & Context:**
  1. Kiểm tra dòng log `cdc_activity_log` của `transaction_history_1`, `cdc_transaction_history_payload_1`, và `devtest.transaction_history_1` ngày `2026-10-30`, cùng số lượng bản ghi thực tế trong bảng PostgreSQL Shadow và MongoDB Master.
  2. Kiểm tra cơ chế `multi_shadow` (`handleTransmuteShadowWithOutcome` trong `centralized-data-service/internal/handler/master/transmute_handler.go:175-230` và `transmuter_mongo.go:290-335`) để làm rõ vì sao:
     - Lượt `Master Sync` (`triggered_by='admin'`) quét ra `250,240` dòng (`250,137` dòng Snapshot + `103` dòng CDC đã vào Shadow trước thời điểm chạy Re-sync).
     - Cột `4. GHI VÀO DB MASTER (Từ CDC Stream)` ghi nhận `347 Inserted` / `462 Updated` (`809 Ops / 386 dòng DB`) so với `317 Insert` / `418 Update` (`735 Ops / 326 dòng DB`) của riêng bảng `transaction_history_1`.
- **Logs/Test:** Kiểm tra qua API `:8083` và code `centralized-data-service` / `cdc-cms-service`; chạy test tích hợp thật trên PostgreSQL + MongoDB (`EV-20261030T162117-f3b4`, `EV-20261030T162239-bd28`, `EV-20261030T162345-7359`, `EV-20261030T162632-17a0`, `EV-20261030T162902-89a9`).
- **Definition of Done:** Giải thích rõ ràng bằng con số thực tế từ DB + `file:line`, trả lời dứt khoát câu hỏi "có cần truncate và chạy lại không", thực thi bản vá Fix 1 & Fix 2 sau `APPROVE`, dọn sạch bản ghi trùng trên `devtest.transaction_history_1` (`shadowActive = 250486 == mongoActive = 250486`, `drift_detected = 0`), khởi chạy lại `cds-worker` (`PID 89267`) và pass cổng `DG3` (`06_laya_summary_dg3_20261030T162950.json`).
- **Status:** `[x] COMPLETED`

## Task T2: Giải quyết Issue #1, #2, #3 — Chuẩn hóa `GetDailyFlow` (`recon_daily` lấy lần đối soát mới nhất + khử trùng `transmute_sub` redelivery) và kiểm chứng build `OpenGormWithEngine`
- **Mô tả & Context:**
  1. Kiểm chứng `pkgs/database/multi_dialector.go` và `internal/service/source/connection_manager.go:405` biên dịch & pass test sạch trên `centralized-data-service`.
  2. Tại `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:420-575`:
     - Khử trùng các dòng `transmute_sub` bị ghi lặp trong 5 vòng NATS redelivery của cùng 1 mẻ (`split_retry = true` có cùng `correlation_id`, `rows_affected`, `inserted`, `updated`).
     - Trong `recon_daily`, lấy `missing_count` và `stale_count` của lần đối soát cuối cùng trong ngày (`ARRAY_AGG(... ORDER BY r.checked_at DESC, r.id DESC)[1]`) thay vì `SUM(...)` cộng dồn các lần lệch cũ đã được heal; đồng thời kiểm tra trạng thái đối soát mới nhất từ `cdc_system.reconciliation_report` (`stage = 'shadow_master'`) để hiển thị đúng trạng thái thực tế.
- **Logs/Test:** `go test -v ./internal/infra/persistence/system/...` trên PostgreSQL thật (`:15432`).
- **Definition of Done:** Test tích hợp thật trên PostgreSQL PASS, số liệu `GetDailyFlow` ngày `2026-10-26` và `2026-10-30` chính xác, có biên lai `EV-...` (`EV-20261030T173100-2b4c`, `EV-20261030T172759-e71f`, `EV-20261030T172811-4388`).
- **Status:** `[x] COMPLETED`

## Task T3: Giải quyết Issue #4 — Tách 4 file vượt ngưỡng dòng (`transmute_handler.go`, `batch_buffer.go`, `mongo_bson_restorer_test.go`, `ReconPipelineGrid.tsx`) theo đúng trách nhiệm đơn (`SRP`)
- **Mô tả & Context:**
  1. Tách `binarySearchSplit`, `splitSingleTaskPayload`, `processSubBatch`, `isTransientError` khỏi `centralized-data-service/internal/handler/master/transmute_handler.go` (`1,074` dòng -> `796` dòng) sang `transmute_poison_pill_split.go` (`298` dòng).
  2. Tách `batchUpsertWithRetry`, `evacuateFailedChunkToDLQ`, `writeFailedSyncLog`, `isRetryableDBError` khỏi `centralized-data-service/internal/handler/shadow/batch_buffer.go` (`967` dòng -> `814` dòng) sang `batch_buffer_dlq.go` (`171` dòng).
  3. Tách `TestMongoOCCBulkModels_ExtendedJSONAndLiteralProtection` và `TestMongoOCCBulkModels_UUIDAndPrimaryOnlyNoDuplicates_LiveMongo` khỏi `centralized-data-service/internal/service/master/mongo_bson_restorer_test.go` (`832` dòng -> `501` dòng) sang `mongo_bson_uuid_live_test.go` (`349` dòng).
  4. Tách Modal thống kê 3 trạm theo ngày khỏi `cdc-cms-web/src/components/ReconPipelineGrid.tsx` (`1,961` dòng -> `1,730` dòng) sang `cdc-cms-web/src/components/ReconDailyFlowModal.tsx` (`296` dòng), và tách `ListActivity`/`Stats24h`/`ClearActivity` khỏi `activity_log_read_repo_gorm.go` (`644` dòng -> `421` dòng) sang `activity_log_list_clear_gorm.go` (`234` dòng).
- **Logs/Test:** Đo `wc -l` trước/sau, chạy `go test` trên các package liên quan (`EV-20261030T172213-3211`, `EV-20261030T173100-2b4c`), `npx tsc --noEmit` trên `cdc-cms-web` (`EV-20261030T172225-9683`), và runtime check (`EV-20261030T172858-46ce`).
- **Definition of Done:** Toàn bộ các file lớn giảm số dòng về `NOW <= HEAD` theo đúng SRP, toàn bộ test và typecheck PASS kèm biên lai `EV-...` và pass cổng `DG3` (`06_laya_summary_dg3_20261030T173300.json`).
- **Status:** `[x] COMPLETED`
