---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# 11 — Implementation & Verification Report: Khắc Phục Triệt Để Hiện Tượng Rớt UPDATE/DELETE Chặng Shadow -> Master (F1..F5)

## 1. Tóm tắt Nguyên nhân Gốc rễ (`F1..F5`) & Giải pháp Đã Triển khai

1. **Giải pháp cho Sơ hở F1 (Lỗi câu lệnh `DELETE` thiếu cột `_source_id` trên bảng Master + Cô lập Poison Pill cả mẻ `Events` + Rớt kết nối DB tạm thời):**
   - Bổ sung `deleteFromMasterBySourceAndGpayIDs` (`internal/service/master/transmuter.go:1551-1684`) đọc động khóa chính `conflictTarget := resolveConflictTarget(binding, colTypes)` từ `master_binding.transform_spec` (`{"pk": "_id"}` $\rightarrow$ `"_id"`, `{"pk": "id"}` $\rightarrow$ `"id"`, mặc định `"_gpay_id"`) kết hợp xóa theo `_gpay_id`, có bọc retry `maxRetries = 3` (`isRetryableDBError`).
   - Bọc retry `maxRetries = 3` cho `bulkUpsertMaster` trong `RunEvents` (`transmuter.go:766-784`).
   - Nâng cấp `binarySearchSplit` và thêm `splitSingleTaskPayload` (`internal/handler/master/transmute_handler.go:690-795`) để chia đôi đệ quy mảng `Events`/`SourceIDs` bên trong 1 NATS message, cứu toàn bộ các bản ghi hợp lệ đi cùng trong 1 message NATS. Mở rộng `isTransientError` (`transmute_handler.go:866-899`).
2. **Giải pháp cho Sơ hở F2 (Lệch `_gpay_id` giữa Shadow và Master làm vô hiệu hóa Recon/Heal Segment B):**
   - Thêm `lookupShadowGpayIDs` (`transmuter.go:1425-1471`) để `RunEvents` tra cứu `_gpay_id` thực từ bảng Shadow và tính `deterministicGpayID(shadowGID, e.KeySuffix)`.
   - Sửa `bulkUpsertMaster` (`transmuter.go:1738-1744`) để khi `conflictTarget != "_gpay_id"` thì `ON CONFLICT (conflictTarget) DO UPDATE` tự động cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"`, chữa lành toàn bộ các dòng lịch sử từng bị ghi bằng SHA-256.
   - Bổ sung `TransformSpec` vào `MasterBindingRef` (`recon_engine_segment_b.go:27,80`) và cập nhật `resolveSourceIDsForSegmentB` / `executeHealSegB` (`recon_execute_heal_handler.go:307-369, 444-539, 680-755`) đọc động cột `"pk"` từ `transform_spec` để giải mã `sourceIDs` và xóa/heal chính xác trên cả Shadow lẫn Master.
3. **Giải pháp cho Sơ hở F3 (Incremental Catch-up Scan chỉ quét `_gpay_id > lastGpayID` và Heal Segment A không kích hoạt Transmute sang Master):**
   - Thêm `syncUpdatedShadowRowsBeforeCursor` (`transmuter.go:389-396, 1475-1546`) để khi chạy Incremental Sync (`runType != "full" && len(chunkIDs) == 0 && initialCursor > 0`), hệ thống quét bổ sung các dòng cũ (`_gpay_id <= initialCursor`) có `_deleted = true` hoặc `_updated_at >= cutoff`.
   - Thêm `publishTransmuteShadowForSegA` (`recon_execute_heal_handler.go:280-305, 592-624`) bắn NATS `cdc.cmd.transmute-shadow` ngay sau khi `executeHealSegA` ghi dữ liệu heal lên bảng Shadow.
4. **Giải pháp cho Sơ hở F4 (`BatchBuffer.batchUpsert` chia nhóm theo `getSignature` trước khi `dedupByConflictKey` và duyệt Go `map` ngẫu nhiên):**
   - Nâng cấp `dedupByConflictKey` (`internal/handler/shadow/batch_buffer_utils.go:90-136`) gộp `MappedData` theo thứ tự thời gian cho các bản ghi trùng `PrimaryKeyValue` và gọi `dedupByConflictKey(sc.Records)` **trước khi** chia nhóm `getSignature(r)` trong `batchUpsert` (`internal/handler/shadow/batch_buffer.go:724-755`), đồng thời duyệt tuần tự theo slice `sigOrder`.
5. **Giải pháp cho Sơ hở F5 (`bulkUpsertMaster` chỉ lấy danh sách cột từ `records[0]`):**
   - Thêm `unionSortedKeys(records)` (`internal/service/master/transmuter_utils.go:222-247`) và sử dụng trong `bulkUpsertMaster` (`transmuter.go:1711`) để lấy hợp (union) tất cả các cột của mọi bản ghi trong batch.

---

## 2. Danh sách File Thay đổi Thực tế (`30b1b00` & `08c05a8`)

- **Commit Restore-Point Round 1:** `30b1b0061ac4d6150196076d28d7732916ea1b9a` (`fix(master,shadow,recon): resolve F1-F5 shadow-to-master update/delete drop and dynamic pk conflict target` — `11 files changed, 1089 insertions(+), 153 deletions(-)`)
- **Commit Restore-Point Round 2 (Khắc phục `AF-01..AF-05`):** `08c05a8` (`fix(cdc): resolve audit findings AF-01..AF-05 for shadow-master sync and heal` — `8 files changed, 287 insertions(+), 79 deletions(-)`):
  1. `internal/handler/master/transmute_handler.go` (sửa `splitSingleTaskPayload` dùng `leftErr != nil || rightErr != nil` + `h.replyErr` / `h.reply` — `AF-02`)
  2. `internal/handler/master/transmute_debounce_group_test.go` (bổ sung `TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails` — `AF-02`)
  3. `internal/service/master/transmuter.go` (persist `inferred last_gpay_id`, chỉ reset `LastCursorJSON = "{}"` khi `runType == "full"`, giới hạn `cutoff` trong `syncUpdatedShadowRowsBeforeCursor` theo `LastSuccessAt - 10m` — `AF-03`)
  4. `internal/service/master/transmuter_gpayid_test.go` (quét động cả 5 Active Bindings `46, 47, 48, 51, 52`, lấy DB theo `ShadowConnectionKey`/`MasterConnectionKey`, kiểm tra bảo toàn `last_cursor_json` — `AF-01`, `AF-03`)
  5. `internal/handler/recon/recon_execute_heal_handler.go` (chia chunk `healChunkSize = 200`, `"shadow_connection_key": "default_shadow"`, lọc `missingGpayIDs` và `uniqueStrings(out)` — `AF-04`, `AF-05`)
  6. `internal/handler/recon/recon_check_heal_handler.go` (bổ sung gọi `publishTransmuteShadowForSegA` trong `executeLegacyHeal` — `AF-04`)
  7. `internal/handler/shadow/batch_buffer_utils.go` (so sánh `r.SourceTsMs` trong `dedupByConflictKey` — `AF-05`)
  8. `internal/handler/shadow/adaptive_batcher_test.go` (bổ sung `TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder` — `AF-05`)

---

## 3. Kết quả Đối soát Dữ liệu Thực tế trên PostgreSQL Trước & Sau khi Thực thi (Cả 5 Active Bindings)

| Master Binding | Cụm DB & Bảng Master (`transform_spec`) | Trước khi chạy Transmute | Sau khi chạy Transmute (Round 2 — `08c05a8`) |
|---|---|---|---|
| **Binding 46** | `goopay_dest:5434` (`default_master`) — `master_payment_bill_service.payment_bills` (`{"pk": "_id"}`) | `driftedIDs = 54,725` (`droppedDelete = 3`, `droppedUpdate = 54,719`, `gpayMismatch = 1,514`) | `activeShadow = 54,755`, `masterCount = 54,755`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 47** | `goopay_dest:5434` (`default_master`) — `master_payment_bill_service.payment_bills_1` (`{"pk": "_id"}`) | `driftedIDs = 1,862` (`droppedDelete = 3`, `droppedUpdate = 344`, `gpayMismatch = 1,514`) | `activeShadow = 54,755`, `masterCount = 54,755`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 48** | `goopay_dest:5434` (`default_master`) — `master_centrallized_export_service.export_jobs` (`{"pk": "_id"}`) | `driftedIDs = 523` (`droppedDelete = 1`, `droppedUpdate = 0`, `gpayMismatch = 521`) | `activeShadow = 540`, `masterCount = 540`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 51** | `goopay_master_2:5437` (`master_2`) — `master_centrallized_export_service.export_jobs_2` (`{"pk": "_id"}`) | `driftedIDs = 523` (`droppedDelete = 1`, `droppedUpdate = 0`, `gpayMismatch = 521`, `masterCount = 541`) | `activeShadow = 540`, `masterCount = 540`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 52** | `goopay_dest:5434` (`default_master`) — `master_centrallized_export_service.export_jobs_1` (`{"pk": "_id"}`) | `driftedIDs = 4` (`droppedDelete = 1`, `droppedUpdate = 0`, `gpayMismatch = 2`, `masterCount = 541`) | `activeShadow = 540`, `masterCount = 540`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |

---

## 4. Kết quả Phản Biện Cuối Workflow & Đóng Vòng Lặp `AF-01..AF-05` (`06_laya_summary_dg3_20261020T164916.json`)

Toàn bộ 5 phát hiện phản biện (`AF-01..AF-05`) từ đợt Audit trước đã được khắc phục triệt để trong commit `08c05a8`, vượt qua toàn bộ unit/integration tests trên PostgreSQL thật và đạt `gate: "PASS"` tại cổng `/laya_router` **DG3** (`06_laya_summary_dg3_20261020T164916.json`, `trusted_rounds: ["r2", "r3"]`).
