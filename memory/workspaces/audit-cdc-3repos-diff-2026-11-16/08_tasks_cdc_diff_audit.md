# 08. TASKS & CRITICAL DIFF AUDIT FINDINGS — 3 REPO CDC

## 1. Danh sách hạng mục đã kiểm tra
- [x] **T1:** Kiểm tra `git status`, `git diff` (cả working tree và commit trên nhánh `recon-heal`) và file untracked của 3 repo: `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`.
- [x] **T2:** Chạy `go vet ./...` và `npx tsc --noEmit` xác nhận không có lỗi cú pháp/type.
- [x] **T3:** Phân tích phản biện từng cụm thay đổi để tìm lỗi mức **CRITICAL** và **HIGH**.

---

## 2. Chi tiết các vấn đề CRITICAL phát hiện trong Diff

### 🔴 [CRITICAL-1] Silent False-Negative trong Recon Row-Diff Segment B cho MongoDB Master (`shouldSkipMongoRowDiffRule`)
- **File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_multi_shadow_b.go:154-186` (đặc biệt dòng `180-184`), gọi tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go:711-713`.
- **Chi tiết:** Khi cột trên MongoDB Master bị thiếu hoặc `null` (`resolveMasterRowDiffVal(mrow, col) == nil`) nhưng bên Shadow có giá trị (`expected != nil`), vòng lặp:
  ```go
  for _, mv := range mrow {
      if subMap, ok := mv.(map[string]any); ok && len(subMap) > 0 {
          return true
      }
  }
  ```
  chỉ kiểm tra xem document `mrow` có chứa bất kỳ sub-document `map[string]any` nào không mà không kiểm tra `targetCol` có thuộc sub-document đó không.
- **Hậu quả:** Trên mọi bảng MongoDB Master có chứa ít nhất 1 nested object, mọi cột nghiệp vụ top-level bị mất dữ liệu hoặc `null` trên Master (`transId`, `value`, `status`, `billCode`...) đều bị bỏ qua 100% khi chạy `RunRowDiffB`.

### 🔴 [CRITICAL-2] Heuristic `isValueSubsetCarrierMap` xóa nhầm Nested Sub-document nghiệp vụ có chứa `_id` khi Transmute sang MongoDB
- **File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_subset_array_utils.go:274-286`, `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:712-746`, và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_bson_restorer.go:521-531`.
- **Chi tiết:**
  1. `extractValueSubsetMetadata` (`mongo_extjson_unwrapper.go:733`) luôn gọi `isValueSubsetCarrierMap(doc, k, rawVal)`,而 hàm này chỉ kiểm tra `len(m) >= 2` và `_, hasID := m["_id"]; hasID`. Các nested sub-document nghiệp vụ trong MongoDB (như `sender`, `receiver`, `customer` từ Mongoose) thường có `_id` riêng nên sẽ bị nhận nhầm là cột metadata `@value_subset`, bị `delete(doc, k)` xóa mất object cha và bung phẳng các trường con ra ngoài root.
  2. `promoteMongoDocumentID` (`mongo_bson_restorer.go:521-531`) duyệt `for k, rawID := range doc` trên `bson.M` (thứ tự map ngẫu nhiên) để lấy bất kỳ cột `*_id`/`*Id` nào có định dạng 24-hex ObjectId làm `doc["_id"]`, gây non-deterministic `_id` khi có nhiều cột ngoại khóa ObjectId (`original_trans_his_id`, `sender_customer_id`, `transaction_history_id`).

### 🔴 [CRITICAL-3] Vòng lặp Auto-Resume vô tận (Poison-Pill Crash Loop), Race đa Worker và ghi sai `cdc_activity_log` trong `SnapshotRunner`
- **File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_lifecycle.go:225-308` và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go:166-175,933-937`.
- **Chi tiết:**
  1. `recoverOrphanedSnapshots` không có giới hạn `MaxOrphanAutoResumeAttempts` (luôn xóa `error_msg = NULL`), dẫn tới nếu một bảng Snapshot bị OOM/crash sau mẻ đầu tiên (`last_seen_id <> ''`), Worker sẽ tự động resume lặp lại vô tận mỗi 30–45s suốt 24 giờ.
  2. Câu `SELECT` trong `recoverOrphanedSnapshots` (`snapshot_runner_lifecycle.go:238-258`) thiếu `FOR UPDATE SKIP LOCKED` và câu `UPDATE ... SET status = 'paused' WHERE id = ? AND status IN ('running', 'paused', 'error')` vẫn trả `RowsAffected = 1` cho nhiều Worker cùng lúc; Worker thua cuộc ở `claimProgress` sẽ ghi oan `status = 'error'` vào `cdc_activity_log`. Ngược lại, khi Worker shutdown (`r.stopping.Load() == true`), `runSnapshot` trả về `nil` làm `defer` ghi nhầm `status = 'success'` vào `cdc_activity_log` cho một tiến trình mới chỉ `paused`.
  3. Nhánh `final flush` (`snapshot_runner_handler.go:933-937`) thiếu kiểm tra `if r.stopping.Load()`, làm tiến trình bị đánh dấu `error` thay vì `paused` nếu hết 2s shutdown đúng lúc flush mẻ cuối.

### 🔴 [CRITICAL-4] Xung đột FE vs BE làm mất `transform_fn` của cột ClickHouse `String` & Lỗi SQL Migration `025_create_daily_source_counts.sql`
- **File FE:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/clickhouseDataTypes.ts:238-246`.
- **File BE:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/update_column_metadata.go:68-71,94-97` và `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/025_create_daily_source_counts.sql:20-51`.
- **Chi tiết:**
  1. Khi đổi kiểu cột sang `String` trên ClickHouse Master, FE chủ đích giữ `transform_fn` (`extjson_*` / `jsonb_passthrough`) bằng cách không gửi `transform_fn: ""` (`cmd.TransformFn == nil`). Tuy nhiên BE trong `update_column_metadata.go:68` kiểm tra `!isJSONOrJSONBType(newDT)` (chỉ cho phép `JSON`/`JSONB`, không có `STRING`) nên vẫn tự động xóa `updates["transform_fn"] = nil`.
  2. Trong `025_create_daily_source_counts.sql`, `CREATE TABLE IF NOT EXISTS cdc_system.daily_source_counts` nằm trong `IF sched_rel IS NOT NULL THEN`, nhưng 2 lệnh `CREATE INDEX` ở dòng `46-50` lại nằm ngoài `DO $$`, sẽ gây lỗi `relation "cdc_system.daily_source_counts" does not exist` nếu `sched_rel IS NULL`.
