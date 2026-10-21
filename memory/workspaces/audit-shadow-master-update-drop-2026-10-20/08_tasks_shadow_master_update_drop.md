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

## Chặng Tiếp Theo: S4 → S6 (Đã Thực thi & Hoàn tất sau lệnh `APPROVE` của User)

- [x] **Task 4 (S4.1 — Giải pháp khắc phục Sơ hở F1: Xóa trên Master theo `transform_spec.pk`, Bọc Retry & Chia đôi Poison Pill cấp độ `Events`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L702-720`, `L766-784`, `L1185-1208`, `L1551-1684`): Loại bỏ câu lệnh hardcode `DELETE FROM %s WHERE _source_id = ?`. Bổ sung `deleteFromMasterBySourceAndGpayIDs` sử dụng trực tiếp cột `conflictTarget := resolveConflictTarget(masterRow, colTypes)` đọc động từ `masterRow.TransformSpec` (`parsePKFromSpec` — bảng có `{"pk": "_id"}` thì dùng `"_id"`, bảng có `{"pk": "id"}` thì dùng `"id"`, không có `"pk"` thì dùng `"_gpay_id"`). Bọc retry `isRetryableDBError` cho cả `DELETE` và `bulkUpsertMaster` trong `RunEvents()`.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go` (`L690-795`, `L866-899`): Khi `len(batch) == 1` nhưng `len(task.Req.Events) > 1` hoặc `len(task.Req.SourceIDs) > 1`, chia đôi đệ quy mảng `Events`/`SourceIDs` (`splitSingleTaskPayload`) để chỉ cô lập đúng 1 event lỗi duy nhất và cứu sống 100% các bản ghi `UPDATE` hợp lệ còn lại. Đồng bộ `isTransientError` với `isRetryableDBError`.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/handler/master/...` (`PASS`)
  - **Definition of Done:** Không còn lỗi `SQLSTATE 42703`, không còn hiện tượng 1 event lỗi làm rớt cả mẻ `Events` trong cùng NATS message.

- [x] **Task 5 (S4.2 — Giải pháp khắc phục Sơ hở F2: Đồng nhất `_gpay_id` giữa Shadow và Master & Khôi phục Recon/Heal Segment B theo `transform_spec.pk`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L690-696`, `L752-764`, `L1425-1471`, `L1734-1745`): Bổ sung `lookupShadowGpayIDs` để `RunEvents()` lấy đúng `_gpay_id` Sonyflake thực tế từ bảng Shadow truyền vào `deterministicGpayID`. Trong `bulkUpsertMaster()`, khi `conflictTarget != "_gpay_id"`, bỏ điều kiện chặn `_gpay_id` để câu lệnh `ON CONFLICT (<conflictTarget>) DO UPDATE` tự động cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` đồng bộ với Shadow.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` (`L307-369`, `L444-539`, `L680-755`): Trong `resolveSourceIDsForSegmentB()` và `executeHealSegB()`, nếu tra cứu `_gpay_id` trên Shadow chưa khớp hết (do dữ liệu lịch sử trên Master mang SHA-256), đọc động cột `"pk"` từ `master_binding.transform_spec` để truy vấn cột `<pk>` từ bảng Master làm `sourceIDs` đẩy vào transmute.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/handler/recon/...` (`PASS`)
  - **Definition of Done:** `_gpay_id` đồng nhất 100% giữa Shadow và Master; đã chữa lành thành công toàn bộ 1,514 dòng lệch trên `payment_bills` / `payment_bills_1` và 521 dòng trên `export_jobs` (`gpayMismatch = 0`).

- [x] **Task 6 (S4.3 — Giải pháp khắc phục Sơ hở F3: Quét Incremental bắt cả bản ghi cũ vừa `UPDATE` & Bắn NATS Transmute sau khi Heal Segment A):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L389-396`, `L1475-1546`): Khi chạy incremental (`runType != "full" && len(chunkIDs) == 0 && initialCursor > 0`), ngoài các dòng mới `_gpay_id > lastGpayID`, quét bổ sung các dòng cũ (`_gpay_id <= initialCursor`) vừa được cập nhật trên Shadow (`_updated_at >= cutoff` hoặc `_deleted = true`) để transmute sang Master.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` (`L280-305`, `L592-624`): Sau khi `HealMismatched` / `HealMissingDest` upsert thành công vào Shadow (`written > 0`), tự động bắn NATS `cdc.cmd.transmute-shadow` chứa `source_ids` vừa heal sang Master.
  - **Logs/Test:** `go test ./internal/service/master/... ./internal/handler/recon/...` (`PASS`)
  - **Definition of Done:** Toàn bộ các dòng rớt `UPDATE` và `DELETE` trên `payment_bills`, `payment_bills_1` và `export_jobs` được đồng bộ đầy đủ sang Master (`droppedUpdate = 0, droppedDelete = 0`).

- [x] **Task 7 (S4.4 — Giải pháp khắc phục Sơ hở F4: Gộp cột & Khử trùng lặp theo `PrimaryKeyValue` trước khi chia nhóm `getSignature` trong `BatchBuffer`):**
  - **Mô tả:** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go` (`L724-755`) và `batch_buffer_utils.go` (`L90-136`), gọi `dedupByConflictKey(sc.Records)` (có merge các cột giữa các sự kiện cùng PK) **trước khi** chia vào `groups[sig]`, và duyệt theo mảng `sigOrder` xác định thay vì duyệt Go `map` ngẫu nhiên.
  - **Logs/Test:** `go test ./internal/handler/shadow/...` (`PASS`)
  - **Definition of Done:** Không còn nguy cơ 2 sự kiện cùng PK trong 1 cửa sổ flush bị tách vào 2 nhóm `sig` và chạy ngược thứ tự.

- [x] **Task 8 (S4.5 — Giải pháp khắc phục Sơ hở F5: Hợp nhất toàn bộ Keys trong `bulkUpsertMaster`):**
  - **Mô tả:** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go` (`L222-251`) và `transmuter.go` (`L1711`), lấy tập hợp (union) tất cả các key của mọi phần tử trong `records` (`unionSortedKeys(records)`) thay vì chỉ lấy `sortedKeysAny(records[0])`.
  - **Logs/Test:** `go test ./internal/service/master/...` (`PASS`)
  - **Definition of Done:** Không bị rớt cột khi các bản ghi trong cùng mẻ `records` có tập cột khác `nil` khác nhau.

---

## Chặng Bổ Sung: S5 → S3 Closed-Loop Re-Plan (Khắc phục 5 phát hiện Phản Biện `AF-01..AF-05` — Đã hoàn tất sau User `APPROVE`, Commit `08c05a8`)

- [x] **Task 9 (Khắc phục `AF-01` — Quét động toàn bộ 5 Active Bindings `46, 47, 48, 51, 52` & Phối hợp Restart 3 Daemon Runtime `:8081/:8082/:8083`):**
  - **Mô tả:** Cập nhật `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` tại `internal/service/master/transmuter_gpayid_test.go:403-574` để truy vấn động toàn bộ `id` có `is_active = true` từ `cdc_system.master_binding` (gồm `46, 47, 48, 51, 52`) và kết nối đúng `row.MasterConnectionKey` (`default_master` `:5434` và `master_2` `:5437`).
  - **Logs/Test:** `go test -v -count=1 ./internal/service/master/... -run TestLiveBindings` (`PASS`: cả 5 bindings `46, 47, 48, 51, 52` đều đạt `droppedDelete = 0, droppedUpdate = 0, gpayMismatch = 0`; `Binding 51` và `Binding 52` đã giảm từ `541` về `540/540` dòng khớp với Shadow).
  - **Definition of Done:** Cả 5 active bindings (`46, 47, 48, 51, 52`) đều đạt `droppedDelete = 0, droppedUpdate = 0, gpayMismatch = 0` (`Binding 51, 52` giảm từ `541` xuống `540` dòng khớp với Shadow).

- [x] **Task 10 (Khắc phục `AF-02` — Sửa lỗi nuốt lỗi 1 nửa trong `splitSingleTaskPayload` tại `internal/handler/master/transmute_handler.go:782-805`):**
  - **Mô tả:** Sửa điều kiện `if leftErr != nil && rightErr != nil` thành `if leftErr != nil || rightErr != nil` để bất kỳ nửa nào lỗi cũng trả lỗi về `task.complete(firstErr)` và gọi `h.replyErr(...)`, không để JetStream `Ack()` sớm mất cơ hội redeliver của nửa bị lỗi; khi cả hai nửa thành công thì gọi `h.reply(task.Msg, ...)` trước `task.complete(nil)`. Bổ sung unit test `TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails` trong `internal/handler/master/transmute_debounce_group_test.go:839-875`.
  - **Logs/Test:** `go test -v -count=1 ./internal/handler/master/...` (`PASS`)
  - **Definition of Done:** Khi 1 nửa payload lỗi, `task.complete` nhận khác `nil` để JetStream redeliver đúng chuẩn.

- [x] **Task 11 (Khắc phục `AF-03` — Bảo toàn `LastCursorJSON` khi Incremental Sync & Giới hạn `cutoff` theo `LastSuccessAt` trong `syncUpdatedShadowRowsBeforeCursor`):**
  - **Mô tả:** Tại `internal/service/master/transmuter.go:357-365, 558-563`, lưu checkpoint `last_gpay_id` khi `inferLastGpayIDFromMaster` khôi phục cursor và chỉ reset `item.LastCursorJSON = []byte("{}")` khi `runType == "full" && len(onlySourceIDs) == 0 && lastBatchErr == nil`. Tại `transmuter.go:1510-1518`, lấy `cutoff` từ `runtimeState.LastSuccessAt` (lùi 10 phút an toàn, fallback `24h`) thay vì hardcode `7 days`.
  - **Logs/Test:** `go test -v -count=1 ./internal/service/master/...` (`PASS`)
  - **Definition of Done:** `LastCursorJSON` giữ nguyên `{"last_gpay_id": ...}` sau khi chạy Incremental Sync; `syncUpdatedShadowRowsBeforeCursor` chỉ quét các dòng mới cập nhật kể từ `LastSuccessAt - 10m`.

- [x] **Task 12 (Khắc phục `AF-04` — Chia chunk & dùng đúng `shadow_connection_key` trong `publishTransmuteShadowForSegA` + Bổ sung publish Transmute cho `executeLegacyHeal`):**
  - **Mô tả:** Tại `internal/handler/recon/recon_execute_heal_handler.go:339-392`, khử trùng lặp `uniqueStrings(sourceIDs)`, truyền đúng `"shadow_connection_key": "default_shadow"` và chia chunk theo `healChunkSize = 200`. Tại `internal/handler/recon/recon_check_heal_handler.go:308-311` (`executeLegacyHeal`), bắn NATS `cdc.cmd.transmute-shadow` sau khi heal Shadow thành công.
  - **Logs/Test:** `go test -v -count=1 ./internal/handler/recon/...` (`PASS`)
  - **Definition of Done:** Cả `ExecuteHealHandler` và `executeLegacyHeal` đều bắn NATS transmute theo từng chunk `200` ID với đúng `shadow_connection_key`.

- [x] **Task 13 (Khắc phục `AF-05` — Khử trùng lặp `sourceIDs` trong `resolveSourceIDsForSegmentB` & So sánh `SourceTsMs` trong `dedupByConflictKey`):**
  - **Mô tả:** Tại `internal/handler/recon/recon_execute_heal_handler.go:732-780`, lọc `missingGpayIDs` chưa tìm thấy trên Shadow và bọc `uniqueStrings(out)` trước khi trả về. Tại `internal/handler/shadow/batch_buffer_utils.go:90-153`, so sánh `r.SourceTsMs` khi gộp `MappedData` và chọn bản ghi mới nhất cho cùng `PrimaryKeyValue`. Bổ sung unit test `TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder` tại `internal/handler/shadow/adaptive_batcher_test.go:134-174`.
  - **Logs/Test:** `go test -v -count=1 ./internal/handler/recon/... ./internal/handler/shadow/...` (`PASS`)
  - **Definition of Done:** Unit test kiểm chứng 2 sự kiện cùng PK bị đảo thứ tự `SourceTsMs` vẫn giữ giá trị của sự kiện có `SourceTsMs` lớn hơn; `resolveSourceIDsForSegmentB` không trả về ID trùng lặp.

---

## Chặng Round 3 (Đã Thực thi & Hoàn tất sau lệnh `APPROVE` của User — Commits `43e5da7`, `069c456`, `2b2fa59`): Tách Bạch `Deleted` Ở Tầng Master & Khắc Phục Tàng Hình `transmute_poison_pill` (`F6`, `F7`, `F8`)

- [x] **Task 14 (Khắc phục Sơ hở F6 — Tách riêng trường `Deleted` trong `TransmuterModule`, `TransmuteHandler` & `SyncRuntimeState` của `centralized-data-service`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L106-126`, `L425-455`, `L565-584`, `L708-726`, `L1400-1414`, `L1541-1548`, `L1638-1681`): Bổ sung `deleted int64` vào `batchOutcome` và `Deleted int64` (`json:"deleted"`) vào `TransmuteResult`; tăng `res.Deleted` / `out.deleted` khi xóa thành công trên Master thay vì cộng gộp vào `res.Updated` / `out.updated`; bổ sung fallback `_source_id` trong `deleteFromMasterBySourceAndGpayIDs` khi `conflictTarget == "_gpay_id"` và bảng Master có map cột `_source_id`.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go` (`L392-421`, `L445-453`, `L655-678`, `L879-898`) và `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_state.go` (`L15-26`): Ghi `"deleted": res.Deleted` vào `cdc_activity_log.details`, `cdc.evt.transmute.completed` và `StatsJSON`; truyền `res.Inserted + res.Updated + res.Deleted` vào `rows_affected`.
  - **Logs/Test:** `go test -v -count=1 ./internal/service/master/... ./internal/handler/master/...` (`PASS`: `TestTransmuter_OrphanMasterSoftDelete`, `TestTransmuter_FullSyncHardDeleteWhenShadowMarkedDeleted`, `TestTransmuter_RunEvents_TracksDeletedSeparatelyFromUpdated`, `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift`).
  - **Definition of Done:** Khi `Run` hoặc `RunEvents` xóa dòng trên Master, `TransmuteResult.Deleted` và `cdc_activity_log.details->>'deleted'` phản ánh đúng số lượng xóa, không cộng lẫn vào `Updated`.

- [x] **Task 15 (Khắc phục Sơ hở F7 & F8 ở Backend CMS — Tổng hợp `master_deleted` và `transmute_poison_pill` trong `cdc-cms-service`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/system/daily_flow.go` (`L36-43`, `L64-85`): Bổ sung `Deleted int64` (`json:"deleted"`) vào `DailyFlowMasterMetrics` và `MasterDeleted int64` (`json:"masterDeleted"`) vào `DailyFlowTotals`.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` (`L62`, `L347-348`, `L377`, `L421`, `L427-454`, `L533-576`): Mở rộng CTE `enriched` và `enrichmentFromClause` lấy cả `transmute_poison_pill`; bọc `COALESCE(..., false)` cho kiểm tra `snapshot` chống SQL 3-valued NULL; tổng hợp `master_deleted` từ `e.details->>'deleted'`; cộng `row.MasterDeleted` vào `rec.Master.Total` và `rec.Master.Written`; cảnh báo `WARNING` khi `rec.Shadow.Total > rec.Master.Total` (`isMasterIncomplete`).
  - **Logs/Test:** `go test -v -count=1 ./internal/infra/persistence/system/... ./internal/app/queries/system/...` (`PASS`: `TestGetDailyFlow_RealPostgres_ExportJobsPoisonPillAndMasterDeleted` trên PostgreSQL thật `cdc_dw`).
  - **Definition of Done:** API `/api/reconciliation/daily-flow` trả về chính xác `master.deleted`, `totals.masterDeleted`, đếm đúng `dlq` khi có `transmute_poison_pill` (`DLQ: 5, Status: DRIFT` cho ngày `2026-10-20` của `export_jobs`) và báo `WARNING` khi `Shadow.Total > Master.Total`.

- [x] **Task 16 (Khắc phục Sơ hở F7 ở Frontend UI — Hiển thị cột `Delete(d)` ở khối Master trên `cdc-cms-web`):**
  - **Mô tả:**
    1. Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts` (`L634`, `L654`): Bổ sung `deleted: number` vào `DailyFlowRecord['master']` và `masterDeleted: number` vào `DailyFlowTotals`.
    2. Tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx` (`L506`, `L841-843`, `L1357-1361`): Hiển thị `Del: ...` trên Card tổng hợp Master và bổ sung cột `Delete(d)` (màu đỏ `#ff4d4f` đồng bộ với Kafka & Shadow) vào khối `4. GHI VÀO DB MASTER`.
  - **Logs/Test:** `npx tsc --noEmit --skipLibCheck` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` (`ZERO_TS_ERRORS_IN_MODIFIED_FILES`).
  - **Definition of Done:** Bảng Daily Flow và Card tổng hợp hiển thị rõ cột `Delete(d)` của Master đối xứng 1:1 với Kafka & Shadow.

