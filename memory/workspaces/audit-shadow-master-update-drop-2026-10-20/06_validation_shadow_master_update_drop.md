---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# 06 — Validation Report: Khắc Phục 5 Sơ Hở Rớt Bản Ghi UPDATE/DELETE Chặng Shadow -> Master (F1..F5)

## 1. Bảng Truy vết Yêu cầu & Sơ hở (G1)

| Req / Finding ID | Mô tả yêu cầu & sơ hở | File:Line đáp ứng | Bằng chứng Test & Runtime | Trạng thái |
|---|---|---|---|---|
| **F1 (R1/R2/R3)** | Sửa lỗi `DELETE FROM ... WHERE _source_id = ?` (`SQLSTATE 42703`) trên Master bằng cách đọc động khóa chính từ `master_binding.transform_spec` (`{"pk": "_id"}` $\rightarrow$ `"_id"`, `{"pk": "id"}` $\rightarrow$ `"id"`, mặc định `"_gpay_id"`), bọc retry lỗi DB tạm thời, và chia đôi đệ quy `Events`/`SourceIDs` trong `binarySearchSplit` để 1 event lỗi không làm rớt cả mẻ `Events`. | `internal/service/master/transmuter.go:702-720`, `766-784`, `1185-1208`, `1551-1684`; `internal/handler/master/transmute_handler.go:472-481`, `690-795`, `866-899` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestIsTransientError_CoversConnectionResetAndBadConnection` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedDelete=0`) | **PASS** |
| **F2 (R2/R3)** | Đồng nhất `_gpay_id` giữa Shadow và Master trong `RunEvents` (`lookupShadowGpayIDs`), cho phép cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` trong `bulkUpsertMaster` khi `conflictTarget != "_gpay_id"`, và đọc động khóa `"pk"` từ `transform_spec` trong `resolveSourceIDsForSegmentB` / `executeHealSegB`. | `internal/service/master/transmuter.go:690-696`, `752-764`, `1425-1471`, `1734-1745`; `internal/service/recon/recon_engine_segment_b.go:27,80`; `internal/handler/recon/recon_execute_heal_handler.go:307-369`, `444-539`, `680-755` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`gpayMismatch=0` cho cả 3 bindings 46, 47, 48) | **PASS** |
| **F3 (R2/R3)** | Quét bổ sung các dòng cũ trên Shadow (`_gpay_id <= initialCursor`) vừa được cập nhật (`_updated_at >= cutoff`) hoặc bị đánh dấu xóa (`_deleted = true`) khi chạy Incremental Sync (`syncUpdatedShadowRowsBeforeCursor`), và tự động bắn NATS `cdc.cmd.transmute-shadow` sau khi `executeHealSegA` ghi dữ liệu lên Shadow. | `internal/service/master/transmuter.go:389-396`, `1475-1546`; `internal/handler/recon/recon_execute_heal_handler.go:280-305`, `592-624` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedUpdate=0` cho cả 3 bindings 46, 47, 48) | **PASS** |
| **F4 (R1/R3)** | Gộp `MappedData` và khử trùng lặp theo `PrimaryKeyValue` (`dedupByConflictKey`) **trước khi** chia nhóm `getSignature(r)` trong `BatchBuffer.batchUpsert`, và duyệt tuần tự theo slice `sigOrder` thay vì duyệt Go `map` ngẫu nhiên. | `internal/handler/shadow/batch_buffer_utils.go:90-136`; `internal/handler/shadow/batch_buffer.go:724-755` | `TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents` (`PASS`) + toàn bộ test suite `internal/handler/shadow/...` (`PASS`) | **PASS** |
| **F5 (R2/R3)** | Hợp nhất (union) toàn bộ keys của mọi bản ghi trong batch (`unionSortedKeys(records)`) trong `bulkUpsertMaster` thay vì chỉ lấy `sortedKeysAny(records[0])`. | `internal/service/master/transmuter_utils.go:222-247`; `internal/service/master/transmuter.go:1710-1715` | `TestUnionSortedKeys_IncludesColumnsFromAllRecords` (`PASS`), `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`) | **PASS** |

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

### 2.2. Integration Tests & Đối soát Dữ liệu Thực trên PostgreSQL (`gpay-postgres-shadow` vs `gpay-postgres-dest`)
- **Lệnh đã chạy:**
  ```bash
  go test -v -count=1 ./internal/service/master/... -run "TestDeterministicGpayID|TestInferLastGpayIDFromMaster|TestUnionSortedKeys|TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres|TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift"
  ```
- **Output thực tế:**
  ```text
  === RUN   TestUnionSortedKeys_IncludesColumnsFromAllRecords
  --- PASS: TestUnionSortedKeys_IncludesColumnsFromAllRecords (0.00s)
  === RUN   TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres
  --- PASS: TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres (0.05s)
  === RUN   TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift
      transmuter_f1_f5_integration_test.go:244: Binding 46 BEFORE: driftedIDs=54725, droppedDelete=3, droppedUpdate=54719, gpayMismatch=1514
      transmuter_f1_f5_integration_test.go:260: Binding 46 Run result: scanned=109446 inserted=0 updated=109444 skipped=0
      transmuter_f1_f5_integration_test.go:286: Binding 46 AFTER: activeShadow=54722, masterCount=54722, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_f1_f5_integration_test.go:244: Binding 47 BEFORE: driftedIDs=1862, droppedDelete=3, droppedUpdate=344, gpayMismatch=1514
      transmuter_f1_f5_integration_test.go:260: Binding 47 Run result: scanned=55070 inserted=0 updated=55066 skipped=0
      transmuter_f1_f5_integration_test.go:286: Binding 47 AFTER: activeShadow=54722, masterCount=54722, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_f1_f5_integration_test.go:244: Binding 48 BEFORE: driftedIDs=523, droppedDelete=1, droppedUpdate=0, gpayMismatch=521
      transmuter_f1_f5_integration_test.go:260: Binding 48 Run result: scanned=541 inserted=0 updated=540 skipped=0
      transmuter_f1_f5_integration_test.go:286: Binding 48 AFTER: activeShadow=540, masterCount=540, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
  --- PASS: TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift (9.50s)
  PASS
  ok  	github.com/gpay/centralized-data-service/internal/service/master	10.172s
  ```

---

## 3. Kết quả Decision Gate DG3, Security & Final Audit (G7)

- **DG1 Solution Gate:** `06_laya_summary_dg1_20261020T144218.json` | Gate: `PASS`
- **DG2 Code Demo Gate:** `06_laya_summary_dg2_20261020T144221.json` | Gate: `PASS`
- **DG3 Post-Implementation Audit Gate:** `06_laya_summary_dg3_20261020T152525.json` | Gate: `PASS`
- **Security Gate (`/security-agent`):** `CLEAN_DIFF` & `CLEAN_WORKSPACE` (0 hardcoded secrets, 0 PII leaks, 0 SQL injection — mọi định danh cột/bảng động đều đi qua `quoteTransmuteIdent` / `quoteTransmuteQualified` / `quoteIdent`).
- **Final Adversarial Audit Report:** `audit_report_shadow_master_update_drop.md`
