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

---

## 5. Kết quả Triển khai & Kiểm định Round 3 (`F6, F7, F8` — Commits `43e5da7`, `069c456`, `2b2fa59`)

1. **Giải pháp cho Sơ hở F6 (`centralized-data-service` — Commit `43e5da7`, `4 files changed, 200 insertions(+), 36 deletions(-)`):**
   - Bổ sung `deleted int64` vào `batchOutcome` và `Deleted int64` (`json:"deleted"`) vào `TransmuteResult` (`internal/service/master/transmuter.go:106-127`).
   - Khi xóa dòng trên Master trong `Run` (`L429, 451`), `RunEvents` (`L726`), `processBatch` (`L1411`) và `syncUpdatedShadowRowsBeforeCursor` (`L1547`), tăng `res.Deleted` / `out.deleted` độc lập với `res.Updated` (không còn cộng nhầm `2 Delete` vào `6 Update` thành `8 Updated`).
   - Ghi `"deleted": res.Deleted` và `rows_affected = res.Inserted + res.Updated + res.Deleted` vào `cdc_activity_log.details`, `cdc.evt.transmute.completed` và `StatsJSON` (`internal/handler/master/transmute_handler.go:395, 408-412, 439, 659-678, 885-888` & `internal/service/master/transmuter_state.go:21`).
2. **Giải pháp cho Sơ hở F7 & F8 (`cdc-cms-service` — Commit `069c456`, `3 files changed, 198 insertions(+), 14 deletions(-)`):**
   - Bổ sung `Deleted int64` (`json:"deleted"`) vào `DailyFlowMasterMetrics` và `MasterDeleted int64` (`json:"masterDeleted"`) vào `DailyFlowTotals` (`internal/app/queries/system/daily_flow.go:41, 81`).
   - Mở rộng `enrichmentFromClause`, `ClearActivity` và `GetDailyFlow` (`internal/infra/persistence/system/activity_log_read_repo_gorm.go:62, 347-348, 421`) bao gồm `'transmute_poison_pill'` để không bỏ sót lỗi Poison Pill ở tầng Master (`DLQ` của ngày `2026-10-20` trên `export_jobs` hiện đúng `5` và `Status: DRIFT` thay vì `0 / BALANCED`).
   - Tổng hợp `master_deleted` từ `COALESCE((e.details->>'deleted')::bigint, 0)` (`L449-450`), bọc `COALESCE(..., false)` cho kiểm tra `snapshot` (`L427-444`), và đặt `Status = "WARNING"` khi `isMasterIncomplete` (`rec.Shadow.Total > rec.Master.Total`, `L547-553`).
3. **Giải pháp cho Sơ hở F7 trên Frontend UI (`cdc-cms-web` — Commit `2b2fa59`, `2 files changed, 5 insertions(+), 3 deletions(-)`):**
   - Bổ sung `deleted: number` và `masterDeleted: number` trong `src/hooks/useReconStatus.ts:634, 654`.
   - Hiển thị `Del: ...` trên Card tổng hợp Master (`src/components/ReconPipelineGrid.tsx:842`) và bổ sung cột `Delete(d)` (`dataIndex: ['master', 'deleted']`, màu `#ff4d4f` đồng bộ với Kafka & Shadow) tại khối **4. GHI VÀO DB MASTER** (`L1359`).
4. **Kết quả `/laya_router` DG3 Final Gate:**
   - Mode A: `06_laya_summary_dg3_20261021T094856.json` (`gate: "PASS"`).
   - Mode B: `06_laya_summary_dg3review_20261021T094858.json` & `06_laya_summary_dg3review_20261021T094930.json` (`gate: "PASS"`).

