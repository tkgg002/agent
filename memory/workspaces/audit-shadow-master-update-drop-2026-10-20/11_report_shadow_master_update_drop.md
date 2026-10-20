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

## 2. Danh sách File Thay đổi (`git show --stat 30b1b00`)

- Commit Restore-Point (Local, nhánh `recon-heal`, không push): `30b1b00` (`fix(master,shadow,recon): resolve F1-F5 shadow-to-master update/delete drop and dynamic pk conflict target`)
  1. `internal/handler/master/transmute_handler.go` (`+123 / -29`)
  2. `internal/handler/master/transmute_handler_test.go` (`+24`)
  3. `internal/handler/recon/recon_execute_heal_handler.go` (`+154 / -22`)
  4. `internal/handler/shadow/batch_buffer.go` (`+16 / -5`)
  5. `internal/handler/shadow/batch_buffer_test.go` (`+58`)
  6. `internal/handler/shadow/batch_buffer_utils.go` (`+29 / -2`)
  7. `internal/service/master/transmuter.go` (`+324 / -65`)
  8. `internal/service/master/transmuter_f1_f5_integration_test.go` (`+292`, file test mới)
  9. `internal/service/master/transmuter_utils.go` (`+28`)
  10. `internal/service/recon/recon_engine_segment_b.go` (`+2`)

---

## 3. Kết quả Đối soát Dữ liệu Thực tế trên PostgreSQL Trước & Sau khi Thực thi

| Master Binding | Bảng Master (`transform_spec`) | Trước khi sửa & chạy Transmute | Sau khi sửa & chạy Transmute |
|---|---|---|---|
| **Binding 46** | `master_payment_bill_service.payment_bills` (`{"pk": "_id"}`) | `driftedIDs = 54,725` (`droppedDelete = 3`, `droppedUpdate = 54,719`, `gpayMismatch = 1,514`) | `activeShadow = 54,722`, `masterCount = 54,722`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 47** | `master_payment_bill_service.payment_bills_1` (`{"pk": "_id"}`) | `driftedIDs = 1,862` (`droppedDelete = 3`, `droppedUpdate = 344`, `gpayMismatch = 1,514`) | `activeShadow = 54,722`, `masterCount = 54,722`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
| **Binding 48** | `master_centrallized_export_service.export_jobs` (`{"pk": "_id"}`) | `driftedIDs = 523` (`droppedDelete = 1`, `droppedUpdate = 0`, `gpayMismatch = 521`) | `activeShadow = 540`, `masterCount = 540`, **`droppedDelete = 0`**, **`droppedUpdate = 0`**, **`gpayMismatch = 0`** |
