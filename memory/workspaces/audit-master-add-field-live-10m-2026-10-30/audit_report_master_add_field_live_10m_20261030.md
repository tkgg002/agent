# AUDIT REPORT — `master_add_field_live_10m_20261030`

- **Workspace:** `audit-master-add-field-live-10m-2026-10-30`
- **tier:** T2
- **mode:** standard
- **Suffix:** `master_add_field_live_10m_20261030`
- **Trạng thái:** `PROPOSE-ONLY (Chờ APPROVE)`

---

## 1. Tóm tắt Kết quả Kiểm toán (Executive Summary)

Đối với bài toán **`source -> shadow -> master`** đang chạy **live luồng đọc trên `master`** và **live luồng ghi từ `source`** với quy mô **~10 triệu (`10,000,000`) bản ghi**, khi muốn thêm 1 field mới từ `shadow` sang `master`:

1. **Những phần hệ thống ĐÃ ĐÁP ỨNG TỐT (Zero-Downtime Ready):**
   - **Kéo field từ `shadow` sang `master`:** Đã có [`SyncRulesFromShadow`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/sync_from_shadow.go#L27-L42) tự động đồng bộ danh sách field mới từ `mapping_rule_v2` (Shadow) sang `mapping_rule_master` (Master).
   - **Mở cột mới trên `master` (nếu là cột thường, không tạo index mới):** Sử dụng `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (nullable, không có volatile default) tại [`master_ddl_generator.go:213-218`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L213-L218) kèm `SET LOCAL lock_timeout = '5s'` ([`master_ddl_generator.go:366`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L366)). Trên PostgreSQL 11+, thao tác này chỉ sửa catalog (`~1–5ms`). Trên MongoDB Master, không cần DDL mở cột.
   - **Tự động cập nhật cho dữ liệu Live mới:** Ngay sau khi `MasterDDLGenerator.Apply()` chạy xong, hàm gọi `g.cacheInvalidator(reg.ID, reg.MasterTable)` ([`master_ddl_generator.go:445-447`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L445-L447)) xóa cache rule của `TransmuterModule`, giúp luồng CDC live (`RunEvents`) tự động trích xuất field mới từ `_raw_data` và ghi vào `master` cho mọi bản ghi mới/cập nhật từ `source`.
   - **Đọc phân trang không dùng `OFFSET`:** Luồng `fetchShadowBatch` ([`transmuter.go:977-986`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L977-L986)) dùng Keyset Pagination `WHERE (_gpay_id)::bigint > ? ORDER BY 1 LIMIT 2000` nên quét 10 triệu dòng trên `shadow` theo B-Tree index rất nhẹ.

2. **4 Điểm Nghẽn Rủi Ro Cao (`F1`–`F4`) SẼ ẢNH HƯỞNG LUỒNG LIVE nếu chưa vá trước khi chạy trên 10 triệu bản ghi:**
   - **`F1` (CRITICAL — Ghi đè dữ liệu cũ lên dữ liệu Live mới khi chạy Full Resync 10M dòng):** Tại [`transmuter.go:451`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L451), khi chạy `run_type = "full"` (hoặc truyền `_source_ids`), `isFullSync` được gán `= true`, làm [`bulkUpsertMaster` (`transmuter.go:1636-1641`)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1636-L1641) và [`BuildMongoOCCBulkModels` (`transmuter_mongo.go:234`)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L234) **bỏ hoàn toàn điều kiện kiểm tra OCC (`_source_ts`)**. Nếu một bản ghi vừa được luồng Live CDC cập nhật lên phiên bản mới (`T2`) trong lúc lô Backfill 10M đang cầm bản ghi đó ở phiên bản cũ (`T1`), lô Backfill sẽ ghi đè bản ghi trên `master` lùi về `T1`.
   - **`F2` (CRITICAL — Đứt gãy luồng CDC Live với lỗi `SQLSTATE 42703 column does not exist`):** [`loadRules` (`transmuter.go:883-894`)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L883-L894) chỉ lọc `status = 'approved'` mà không kiểm tra `m.in_master = true`, trong khi CMS (`save_master_rule.go:65-74`, `batch_update_rules.go:42-53`) lưu `status = 'approved'` trước rồi mới gửi `cdc.cmd.master-create` dạng fire-and-forget (lại thiếu `master_binding_id`). Nếu DDL tạo cột trên PostgreSQL `master` chưa kịp xong hoặc bị trượt do `lock_timeout = 5s`, luồng Live CDC sẽ `INSERT` ngay vào cột chưa tồn tại và nổ lỗi `42703`.
   - **`F3` (HIGH — Giữ `AccessExclusiveLock` trong suốt lúc dựng Index trên 10M dòng):** [`MasterDDLGenerator.Apply` (`master_ddl_generator.go:361-400`)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L361-L400) chạy `ALTER TABLE ADD COLUMN` và `CREATE INDEX IF NOT EXISTS` (không `CONCURRENTLY`) trong cùng 1 transaction. Nếu field mới là cột tài chính (`amount`, `fee`, `balance`...) hoặc có index, `AccessExclusiveLock` của `ALTER TABLE` sẽ bị giữ chặt suốt quá trình quét 10 triệu dòng tạo index, chặn đứng toàn bộ luồng đọc/ghi Live trên `master` và dễ chạm `statement_timeout = 30s`.
   - **`F4` (HIGH — Bão I/O, MVCC Dead Tuples & Deadlock `40P01` khi Backfill 10M dòng không nghỉ):** Vòng lặp `Full Resync` ([`transmuter.go:401-581`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L401-L581)) chạy liên tục 5,000 batches (`2,000` dòng/batch) không có nhịp nghỉ (Zero Throttle) và không sort theo khóa `conflictTarget` trước khi `INSERT ... ON CONFLICT DO UPDATE`, gây bão WAL/Autovacuum làm tăng latency đọc Live và dễ đụng Deadlock `40P01` với batch Live CDC.

---

## 2. Kết quả Kiểm định Decision Gate (`/laya_router`)
- **DG0 (`Pre-Plan Critique`):** `PASS` (`06_laya_summary_dg0_20261030T152137.json`).
- **DG1 + DG2 + DG3 (`Solution, Code Demo & Audit Claims`):** Đã kiểm chứng toàn bộ `REF:` khớp tuyệt đối mã nguồn thực tế (`transmuter.go`, `transmuter_mongo.go`, `master_ddl_generator.go`).


---

## 3. KIỂM TOÁN PHẢN BIỆN PLAN V1 & NÂNG CẤP LÊN PLAN V2 (Adversarial Critique: Mục tiêu, File, Từng Dòng Update, SRP & Tương thích Ngược)

Thực hiện kiểm toán phản biện chéo Plan V1 với toàn bộ bộ test hiện hữu (`transmuter_orphan_test.go`, `transmuter_events_batch_test.go`, `transmuter_mongo_test.go`, `transmuter_gpayid_test.go`, `master_ddl_indexes_test.go`, `master_ddl_handler_test.go`) và kỷ luật số dòng `L008` (`NOW <= HEAD` trên file `> 500` dòng):

1. **Phản biện Mục tiêu #1 (`F2` — Bác bỏ việc sửa câu SQL `loadRules` thêm `AND (m.in_master = true OR ? = true)`):**
   - **Tử huyệt của V1:** Cột `in_master` có `DEFAULT FALSE` (migration `076`), nên các bảng `master` cũ và dữ liệu seed trong [`transmuter_events_batch_test.go:143-148`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_events_batch_test.go#L143-L148) có `in_master = false`. Lọc cứng `in_master = true` trong `loadRules` sẽ làm tê liệt toàn bộ bảng cũ và làm gãy 5 unit tests trong [`transmuter_orphan_test.go:97,288,449,596,726`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_orphan_test.go#L97).
   - **Quyết định V2:** Giữ nguyên 100% `loadRules`. Tách sang file mới đơn trách nhiệm `transmuter_batch_guard.go`: Hàm `bulkUpsertMasterWithSchemaGuard` chỉ kích hoạt `filterRecordsToExistingMasterColumns` khi PostgreSQL thực sự trả về lỗi `SQLSTATE 42703` (`undefined_column`), tạm thời loại cột chưa kịp `ALTER TABLE` khỏi bản sao `records` của mẻ hiện tại và retry ngay.

2. **Phản biện Mục tiêu #2 (`F1` — Bác bỏ việc sửa `BuildMongoOCCBulkModels` và xóa nhánh `if isFullSync` trong `bulkUpsertMaster`):**
   - **Tử huyệt của V1:** Sửa `BuildMongoOCCBulkModels` làm PANIC [`transmuter_mongo_test.go:194-207`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo_test.go#L194-L207); xóa `if isFullSync` trong `bulkUpsertMaster` làm hỏng luồng Recon Force-Heal theo danh sách ID tại [`transmuter_gpayid_test.go:527`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_gpayid_test.go#L527).
   - **Quyết định V2:** Giữ nguyên 100% `bulkUpsertMaster` và `BuildMongoOCCBulkModels`. Chỉ sửa đúng điểm gốc tại [`transmuter.go:451`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L451) từ `runType == "full" || len(chunkIDs) > 0` thành `runType == "full" && len(chunkIDs) > 0`. Khi quét toàn bảng 10M dòng (`len(chunkIDs) == 0`), `isFullSync = false` được truyền xuống giúp giữ nguyên khóa bảo vệ OCC (`_source_ts`) trên cả PostgreSQL và MongoDB Master!

3. **Phản biện Mục tiêu #3 (`F3` + `L008` — Tách file `master_ddl_generator_pg.go` & Loại bỏ `AccessExclusiveLock` thừa trên bảng 10M dòng):**
   - **Tử huyệt của V1:** `master_ddl_generator.go` đã dài `743` dòng (`> 500` dòng). Ngoài ra, 7 câu `ALTER TABLE ... DROP COLUMN IF EXISTS` (dòng `394-398`) cũng lấy `AccessExclusiveLock` trước khi kiểm tra `IF EXISTS`, và SQLite trong unit test không hỗ trợ `CREATE INDEX CONCURRENTLY`.
   - **Quyết định V2:** Tách toàn bộ thực thi DDL PostgreSQL sang file mới `master_ddl_generator_pg.go` (`~395` dòng, đối xứng với `master_ddl_generator_mongo.go`), giúp `master_ddl_generator.go` giảm từ `743` xuống `~360` dòng (`NOW < HEAD`). Khi bảng đã tồn tại (`len(existingCols) > 0`), chỉ chạy `ALTER TABLE ADD COLUMN` cho cột mới chưa có trong `existingCols` và chỉ `DROP COLUMN` nếu cột legacy còn tồn tại (commit trong `< 5ms`), sau đó chỉ chạy `CREATE INDEX CONCURRENTLY IF NOT EXISTS` ngoài transaction cho các index chưa có trong `pg_indexes`.

4. **Phản biện Mục tiêu #4 (`F4` + `L008` — Tách file `transmuter_batch_guard.go` & Giảm số dòng `transmuter.go`):**
   - **Quyết định V2:** Rút 68 dòng dedup inline tại [`transmuter.go:1202-1270`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1202-L1270) sang `dedupAndSortMasterBatchRecords` trong `transmuter_batch_guard.go` (vừa khử trùng theo `_source_ts` vừa sort theo `conflictTarget` chống Deadlock `40P01`), giúp `transmuter.go` giảm từ `1,835` xuống `~1,775` dòng (`NOW < HEAD`).

5. **Phản biện Mục tiêu #5 (`F5` — Phát hiện và chặn "Cú Nổ Ẩn" lệch 10 triệu dòng trên Báo cáo Đối soát 3 Trạm `GetDailyFlow`):**
   - **Bằng chứng (`file:line`):** [`transmute_handler.go:401`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L401) ghi `"run_type": "full"`, nhưng [`activity_log_read_repo_gorm.go:428,438,454-462`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go#L428-L462) chỉ lọc `= 'full_sync'` (thiếu `'full'`), khiến 10 triệu dòng Updated khi Backfill có nguy cơ bị cộng nhầm vào cột `4. GHI VÀO DB MASTER (Từ CDC Stream)` thay vì `1. SNAPSHOT / RE-SYNC`.
   - **Quyết định V2:** Sửa in-place 7 dòng tại `activity_log_read_repo_gorm.go:428, 438, 454-462` sang `IN ('full', 'full_sync')` / `NOT IN ('full', 'full_sync')` (giữ nguyên `599/599` dòng `NOW == HEAD`).


## 4. Kết quả Kiểm định `/laya_router` cho Plan V2 (`DG1`, `DG2`, `DG3` — `PASS`)
- **Claims File V2:** [`06_laya_claims_dg1_v2_master_add_field_live_10m_20261030.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-add-field-live-10m-2026-10-30/06_laya_claims_dg1_v2_master_add_field_live_10m_20261030.json)
- **Trusted Rounds:** `["r1", "r2", "r3"]` (Đạt chuẩn hiệu chuẩn 3/3 vòng với `CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`).
- **Kết quả chi tiết từng Claim trong Plan V2:**
  - `S-RC-1` (`F1` — `transmuter.go:450-452` truyền `runType == "full" || len(chunkIDs) > 0` làm tắt OCC khi Full Resync): **`ACCEPT`** (`r3 YES = 0.9047`).
  - `S-FIX-1` (`F4` — `transmuter.go:1257-1293` gán `dedupedRecords` vào `allRecords` và truyền thẳng vào `bulkUpsertMaster` chưa sort theo `conflictTarget`): **`ACCEPT_WITH_EVIDENCE`** (đã ghi ADR kèm `REF:` tại [`04_decisions_master_add_field_live_10m_20261030.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-add-field-live-10m-2026-10-30/04_decisions_master_add_field_live_10m_20261030.md)).
  - `S-IMPACT-1` (`F5` — `activity_log_read_repo_gorm.go:438-455` chỉ lọc `= 'full_sync'`, bỏ sót `'full'`): **`ACCEPT`** (`r3 YES = 0.5088`).
  - `S-PATTERN-1` (`F3` — `master_ddl_generator.go:372-399` chạy gộp `CreateSQL`, `AlterSQL`, `IndexSQL` và 7 lệnh `DROP COLUMN IF EXISTS` trong 1 transaction): **`ACCEPT`** (`r3 YES = 0.6554`).
  - `D-API-1` (`F2` — `master_ddl_publisher.go:124-136` gửi `cdc.cmd.master-create` thiếu `master_binding_id`): **`ACCEPT`** (`r1 CORRECT = 0.7526`, `r3 YES = 0.6964`).
- **Biên bản Summary V2:**
  - `DG1`: [`06_laya_summary_dg1_20261030T163646.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-add-field-live-10m-2026-10-30/06_laya_summary_dg1_20261030T163646.json) (`"gate": "PASS"`)
  - `DG2`: [`06_laya_summary_dg2_20261030T163646.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-add-field-live-10m-2026-10-30/06_laya_summary_dg2_20261030T163646.json) (`"gate": "PASS"`)
  - `DG3`: [`06_laya_summary_dg3_20261030T163733.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-add-field-live-10m-2026-10-30/06_laya_summary_dg3_20261030T163733.json) (`"gate": "PASS"`)
