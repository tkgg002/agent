---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# 06 — Validation Report: Khắc Phục 5 Sơ Hở Rớt Bản Ghi UPDATE/DELETE Chặng Shadow -> Master (F1..F5)

## 1. Bảng Truy vết Yêu cầu & Sơ hở (G1)

| Req / Finding ID | Mô tả yêu cầu & sơ hở | File:Line đáp ứng | Bằng chứng Test & Runtime | Trạng thái |
|---|---|---|---|---|
| **F1 (R1/R2/R3)** | Sửa lỗi `DELETE FROM ... WHERE _source_id = ?` (`SQLSTATE 42703`) trên Master bằng cách đọc động khóa chính từ `master_binding.transform_spec` (`{"pk": "_id"}` $\rightarrow$ `"_id"`, `{"pk": "id"}` $\rightarrow$ `"id"`, mặc định `"_gpay_id"`), bọc retry lỗi DB tạm thời, và chia đôi đệ quy `Events`/`SourceIDs` trong `binarySearchSplit` để 1 event lỗi không làm rớt cả mẻ `Events`. | `internal/service/master/transmuter.go:702-720`, `766-784`, `1185-1208`, `1551-1684`; `internal/handler/master/transmute_handler.go:472-481`, `690-795`, `866-899` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestIsTransientError_CoversConnectionResetAndBadConnection` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedDelete=0` trên 46/47/48; phát hiện `AF-02` tại `splitSingleTaskPayload` `L790-794` cần vá tiếp) | **PARTIAL (`AF-02`)** |
| **F2 (R2/R3)** | Đồng nhất `_gpay_id` giữa Shadow và Master trong `RunEvents` (`lookupShadowGpayIDs`), cho phép cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` trong `bulkUpsertMaster` khi `conflictTarget != "_gpay_id"`, và đọc động khóa `"pk"` từ `transform_spec` trong `resolveSourceIDsForSegmentB` / `executeHealSegB`. | `internal/service/master/transmuter.go:690-696`, `752-764`, `1425-1471`, `1734-1745`; `internal/service/recon/recon_engine_segment_b.go:27,80`; `internal/handler/recon/recon_execute_heal_handler.go:307-337`, `444-539`, `595-621`, `680-755` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`gpayMismatch=0` cho 46, 47, 48; phát hiện `AF-05` chưa khử trùng lặp `out` tại `L747`) | **PARTIAL (`AF-05`)** |
| **F3 (R2/R3)** | Quét bổ sung các dòng cũ trên Shadow (`_gpay_id <= initialCursor`) vừa được cập nhật (`_updated_at >= cutoff`) hoặc bị đánh dấu xóa (`_deleted = true`) khi chạy Incremental Sync (`syncUpdatedShadowRowsBeforeCursor`), và tự động bắn NATS `cdc.cmd.transmute-shadow` sau khi `executeHealSegA` ghi dữ liệu lên Shadow. | `internal/service/master/transmuter.go:545-557`, `1475-1546`; `internal/handler/recon/recon_execute_heal_handler.go:280-305`, `339-381` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedUpdate=0` trên 46/47/48; phát hiện `AF-03` reset `LastCursorJSON` + cutoff 7 ngày và `AF-04` tại `publishTransmuteShadowForSegA` & `recon_heal.go`) | **PARTIAL (`AF-03`, `AF-04`)** |
| **F4 (R1/R3)** | Gộp `MappedData` và khử trùng lặp theo `PrimaryKeyValue` (`dedupByConflictKey`) **trước khi** chia nhóm `getSignature(r)` trong `BatchBuffer.batchUpsert`, và duyệt tuần tự theo slice `sigOrder` thay vì duyệt Go `map` ngẫu nhiên. | `internal/handler/shadow/batch_buffer_utils.go:90-136`; `internal/handler/shadow/batch_buffer.go:724-749` | `TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents` trong `adaptive_batcher_test.go:388-435` (`PASS`; phát hiện `AF-05` cần so sánh thêm `r.SourceTsMs`) | **PARTIAL (`AF-05`)** |
| **F5 (R2/R3)** | Hợp nhất (union) toàn bộ keys của mọi bản ghi trong batch (`unionSortedKeys(records)`) trong `bulkUpsertMaster` thay vì chỉ lấy `sortedKeysAny(records[0])`. | `internal/service/master/transmuter_utils.go:222-251`; `internal/service/master/transmuter.go:1710-1715` | `TestUnionSortedKeys_IncludesColumnsFromAllRecords` trong `transmuter_orphan_test.go:74-96` (`PASS`), `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`) | **PASS** |

---

## 2. Kết quả Test & Runtime Thực tế (G2, G3, G4, G5, G6)

### 2.1. Unit Tests & Package Regression Suite
- **Lệnh đã chạy:**
  ```bash
  go test -v ./internal/handler/shadow/... ./internal/handler/master/... ./internal/handler/recon/...
  ```
- **Output thực tế:**
  ```text
  === RUN   TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents
  --- PASS: TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents (0.00s)
  ok  	github.com/gpay/centralized-data-service/internal/handler/shadow	0.923s
  === RUN   TestIsTransientError_CoversConnectionResetAndBadConnection
  --- PASS: TestIsTransientError_CoversConnectionResetAndBadConnection (0.00s)
  ok  	github.com/gpay/centralized-data-service/internal/handler/master	1.641s
  ok  	github.com/gpay/centralized-data-service/internal/handler/recon	0.517s
  ```

### 2.2. Integration Tests & Đối soát Dữ liệu Thực trên PostgreSQL (`127.0.0.1:5433` `cdc_dw`, `127.0.0.1:5436` `cdc_shadow`, `127.0.0.1:5434` `goopay_dest`, `127.0.0.1:5437` `goopay_master_2`)
- **Lệnh đã chạy:**
  ```bash
  go test -v -count=1 ./internal/service/master/... -run "TestDeterministicGpayID|TestInferLastGpayIDFromMaster|TestUnionSortedKeys|TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres|TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift"
  ```
- **Output thực tế (`internal/service/master/transmuter_gpayid_test.go` & `transmuter_orphan_test.go`):**
  ```text
  === RUN   TestUnionSortedKeys_IncludesColumnsFromAllRecords
  --- PASS: TestUnionSortedKeys_IncludesColumnsFromAllRecords (0.00s)
  === RUN   TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres
  --- PASS: TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres (0.05s)
  === RUN   TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift
      transmuter_gpayid_test.go:467: Binding 46 BEFORE: driftedIDs=54725, droppedDelete=3, droppedUpdate=54719, gpayMismatch=1514
      transmuter_gpayid_test.go:484: Binding 46 Run result: scanned=109446 inserted=0 updated=109444 skipped=0
      transmuter_gpayid_test.go:539: Binding 46 AFTER: activeShadow=54722, masterCount=54722, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:467: Binding 47 BEFORE: driftedIDs=1862, droppedDelete=3, droppedUpdate=344, gpayMismatch=1514
      transmuter_gpayid_test.go:484: Binding 47 Run result: scanned=55070 inserted=0 updated=55066 skipped=0
      transmuter_gpayid_test.go:539: Binding 47 AFTER: activeShadow=54722, masterCount=54722, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:467: Binding 48 BEFORE: driftedIDs=523, droppedDelete=1, droppedUpdate=0, gpayMismatch=521
      transmuter_gpayid_test.go:484: Binding 48 Run result: scanned=541 inserted=0 updated=540 skipped=0
      transmuter_gpayid_test.go:539: Binding 48 AFTER: activeShadow=540, masterCount=540, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
  --- PASS: TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift (9.50s)
  PASS
  ok  	github.com/gpay/centralized-data-service/internal/service/master	10.172s
  ```
- **Kiểm tra bổ sung 2 Active Bindings `51, 52` và Tiến trình Daemon Runtime (`AF-01`):**
  - `Binding 51` (`goopay_master_2.master_centrallized_export_service.export_jobs_2` trên `:5437`) và `Binding 52` (`goopay_dest.master_centrallized_export_service.export_jobs_1` trên `:5434`) đang có `541` dòng vs `540` active shadow (cần chạy đồng bộ bổ sung sau khi User `APPROVE` `Task 9`).
  - Ba tiến trình daemon `:8081` (`PID 34453`), `:8083` (`PID 50156`), `:8082` (`PID 50249`) vẫn đang chạy binary từ sáng (`08:55` và `10:30`), cần phối hợp với User để rebuild & restart sau khi hoàn tất `Task 9..13`.

---

## 3. Kết quả Decision Gate DG3, Security & Final Audit (G7)

- **DG1 Solution Gate (Re-Plan `AF-01..AF-05`):** `06_laya_summary_dg1_20261020T160744.json` (`trusted_rounds=["r3"]`) | Gate: `PASS`
- **DG2 Code Demo Gate (Re-Plan `AF-01..AF-05`):** `06_laya_summary_dg2_20261020T160746.json` (`trusted_rounds=["r3"]`) | Gate: `PASS`
- **DG3 Post-Implementation Adversarial Audit Gate (Mode A):** `06_laya_summary_dg3_20261020T160810.json` (`trusted_rounds=["r3"]`) | Gate: `PASS`
- **DG3 Agentic Debate Diff Review (Mode B):** `06_laya_summary_dg3review_20261020T155849.json` | Gate: `BLOCK` (`REJECT_LOGIC` trên `internal/handler/recon/recon_execute_heal_handler.go` — xác nhận độc lập `AF-04` & `AF-05`, kích hoạt `S5 -> S3 Re-Plan`).
- **Security Gate (`/security-agent`):** `CLEAN_DIFF` & `CLEAN_WORKSPACE` (0 hardcoded secrets, 0 PII leaks, 0 SQL injection).
- **Final Adversarial Audit Report:** `audit_report_shadow_master_update_drop.md`
