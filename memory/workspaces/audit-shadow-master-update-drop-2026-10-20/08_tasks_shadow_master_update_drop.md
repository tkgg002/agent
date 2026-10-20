# 08. Tasks Checklist — Audit & Fix Shadow -> Master Update Drop

## Chặng Hiện Tại: S1 → S3 (Audit & Thiết kế Giải pháp — Propose-Only, Chờ `APPROVE`)

- [x] **Task 1 (S0/S1 — Intake & Workspace Setup):**
  - **Mô tả:** Đọc `GEMINI.md`, `lessons/INDEX.md`, khởi tạo bộ hồ sơ S1 (`01_requirements`, `05_progress`, `08_tasks`) cho workspace `audit-shadow-master-update-drop-2026-10-20`.
  - **Context & Logs:** User yêu cầu audit hiện tượng sync Mongo -> Postgres (~15k records/ngày) bị rớt ~3-4 record UPDATE ở chặng Shadow -> Master.
  - **Definition of Done:** Đủ 3 file S1 đúng chuẩn `verify_governance.py`.

- [x] **Task 2 (S2 — Read-Only Deep Investigation):**
  - **Mô tả:** Truy vết toàn trình luồng dữ liệu UPDATE từ MongoDB CDC -> Shadow DB -> NATS JetStream / Debouncer / Catch-up Scan -> Transmute Engine -> Master DB.
  - **Context & Logs:** Kiểm tra `centralized-data-service/internal/service/master/transmuter.go:670-755`, `internal/handler/master/transmute_handler.go:690-745`, `internal/handler/shadow/batch_buffer.go:728-753`, `internal/service/recon/recon_heal.go:548-737`, `internal/handler/recon/recon_execute_heal_handler.go:653-734` và đối soát trực tiếp dữ liệu runtime trên `gpay-postgres-cdc`, `gpay-postgres-shadow`, `gpay-postgres-dest`.
  - **Definition of Done:** Đã xác định đầy đủ 5 lỗ hổng (`F1..F5`) có bằng chứng `file:line` (`VERIFIED`) và bằng chứng DB runtime (`14` dòng rớt UPDATE + `2` dòng rớt DELETE trên `payment_bills`, `344` dòng rớt UPDATE trên `payment_bills_1`, `1,482` dòng lệch `_gpay_id`).

- [x] **Task 3 (S3 & DG1/DG2/DG3 — Analysis, Single Best Solution & Laya Verification):**
  - **Mô tả:** Lập báo cáo phân tích chuyên sâu `13_analysis_shadow_master_update_drop.md` và hồ sơ giải pháp `09_tasks_solution_shadow_master_update_drop.md`, chạy `laya_gate.py` cho `DG3`, `DG1`, `DG2`.
  - **Context & Logs:** Đính kèm claims có `REF: <path>:<start>-<end>` và control claims.
  - **Definition of Done:** `DG1`, `DG2`, `DG3` đều đạt `PASS`, `verify_governance.py --stage S3` đạt `PASSED`, trình User và dừng chờ lệnh `APPROVE`.

---

## Chặng Tiếp Theo: S4 → S6 (Thực thi sau khi có lệnh `APPROVE` của User)

- [ ] **Task 4 (S4.1 — Giải pháp khắc phục Sơ hở F1: Xóa trên Master theo `transform_spec.pk`, Bọc Retry & Chia đôi Poison Pill cấp độ `Events`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L424-440`, `L672-688`, `L726`): Loại bỏ câu lệnh hardcode `DELETE FROM %s WHERE _source_id = ?`. Bổ sung `deleteFromMasterBySourceIDs` sử dụng trực tiếp cột `conflictTarget := resolveConflictTarget(masterRow, colTypes)` đọc động từ `masterRow.TransformSpec` (`parsePKFromSpec` — bảng có `{"pk": "_id"}` thì dùng `"_id"`, bảng có `{"pk": "id"}` thì dùng `"id"`, không có `"pk"` thì dùng `"_gpay_id"`). Bọc retry `isRetryableDBError` cho cả `DELETE` và `bulkUpsertMaster` trong `RunEvents()`.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go` (`L690-720`, `L826-845`): Khi `len(batch) == 1` nhưng `len(task.Req.Events) > 1` hoặc `len(task.Req.SourceIDs) > 1`, chia đôi đệ quy mảng `Events`/`SourceIDs` (`splitSingleTaskPayload`) để chỉ cô lập đúng 1 event lỗi duy nhất và cứu sống 100% các bản ghi `UPDATE` hợp lệ còn lại. Đồng bộ `isTransientError` với `isRetryableDBError`.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/handler/master/...`
  - **Definition of Done:** Không còn lỗi `SQLSTATE 42703`, không còn hiện tượng 1 event lỗi làm rớt cả mẻ `Events` trong cùng NATS message.

- [ ] **Task 5 (S4.2 — Giải pháp khắc phục Sơ hở F2: Đồng nhất `_gpay_id` giữa Shadow và Master & Khôi phục Recon/Heal Segment B theo `transform_spec.pk`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L718-724`, `L1438-1443`): Bổ sung `lookupShadowGpayIDs` để `RunEvents()` lấy đúng `_gpay_id` Sonyflake thực tế từ bảng Shadow truyền vào `deterministicGpayID`. Trong `bulkUpsertMaster()`, khi `conflictTarget != "_gpay_id"`, bỏ điều kiện chặn `_gpay_id` để câu lệnh `ON CONFLICT (<conflictTarget>) DO UPDATE` tự động cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` đồng bộ với Shadow.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` (`L653-734`): Trong `resolveSourceIDsForSegmentB()`, nếu tra cứu `_gpay_id` trên Shadow chưa khớp hết (do dữ liệu lịch sử trên Master mang SHA-256), đọc động cột `"pk"` từ `master_binding.transform_spec` để truy vấn cột `<pk>` từ bảng Master làm `sourceIDs` đẩy vào transmute.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/handler/recon/...`
  - **Definition of Done:** `_gpay_id` đồng nhất 100% giữa Shadow và Master; Heal Segment B chữa lành thành công toàn bộ 1,482 dòng lệch trên `payment_bills` và 520 dòng trên `export_jobs`.

- [ ] **Task 6 (S4.3 — Giải pháp khắc phục Sơ hở F3: Quét Incremental bắt cả bản ghi cũ vừa `UPDATE` & Bắn NATS Transmute sau khi Heal Segment A):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L330-370`): Khi chạy incremental (`!req.FullResync && len(req.OnlySourceIDs) == 0`), ngoài các dòng mới `_gpay_id > lastGpayID`, quét bổ sung các dòng cũ (`_gpay_id <= lastGpayID`) vừa được cập nhật trên Shadow (`_synced_at >= lookbackWindow`) để transmute sang Master.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go` (`L548-570`, `L720-737`): Sau khi `HealMissingIDs` / `HealWindow` upsert thành công vào Shadow, tự động bắn NATS `cdc.cmd.transmute` chứa `source_ids` vừa heal sang Master.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/service/recon/...`
  - **Definition of Done:** Toàn bộ 14 dòng rớt `UPDATE` trên `payment_bills` và 344 dòng trên `payment_bills_1` được đồng bộ đầy đủ sang Master.

- [ ] **Task 7 (S4.4 — Giải pháp khắc phục Sơ hở F4: Gộp cột & Khử trùng lặp theo `PrimaryKeyValue` trước khi chia nhóm `getSignature` trong `BatchBuffer`):**
  - **Mô tả:** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go` (`L728-753`) và `batch_buffer_utils.go` (`L26-42`), gọi `dedupByConflictKey(sc.Records)` (có merge các cột không `nil` giữa các sự kiện cùng PK) **trước khi** chia vào `groups[sig]`, và duyệt theo mảng `sigOrder` xác định thay vì duyệt Go `map` ngẫu nhiên.
  - **Logs/Test:** `go test ./internal/handler/shadow/...`
  - **Definition of Done:** Không còn nguy cơ 2 sự kiện cùng PK trong 1 cửa sổ flush bị tách vào 2 nhóm `sig` và chạy ngược thứ tự.

- [ ] **Task 8 (S4.5 — Giải pháp khắc phục Sơ hở F5: Hợp nhất toàn bộ Keys trong `bulkUpsertMaster`):**
  - **Mô tả:** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L1412`), lấy tập hợp (union) tất cả các key của mọi phần tử trong `records` thay vì chỉ lấy `sortedKeysAny(records[0])`.
  - **Logs/Test:** `go test ./internal/service/master/...`
  - **Definition of Done:** Không bị rớt cột khi các bản ghi trong cùng mẻ `records` có tập cột khác `nil` khác nhau.


