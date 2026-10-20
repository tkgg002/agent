---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# 06 — Validation Report: Khắc Phục 5 Sơ Hở Rớt Bản Ghi UPDATE/DELETE Chặng Shadow -> Master (F1..F5)

## 1. Bảng Truy vết Yêu cầu & Sơ hở (G1)

| Req / Finding ID | Mô tả yêu cầu & sơ hở | File:Line đáp ứng | Bằng chứng Test & Runtime | Trạng thái |
|---|---|---|---|---|
| **F1 (R1/R2/R3) + AF-02** | Sửa lỗi `DELETE FROM ... WHERE _source_id = ?` (`SQLSTATE 42703`) trên Master bằng cách đọc động khóa chính từ `master_binding.transform_spec` (`{"pk": "_id"}` $\rightarrow$ `"_id"`, `{"pk": "id"}` $\rightarrow$ `"id"`, mặc định `"_gpay_id"`), bọc retry lỗi DB tạm thời, và chia đôi đệ quy `Events`/`SourceIDs` trong `binarySearchSplit` / `splitSingleTaskPayload` (`leftErr != nil || rightErr != nil` + `h.replyErr` / `h.reply`). | `internal/service/master/transmuter.go:702-720`, `766-784`, `1185-1208`, `1557-1690`; `internal/handler/master/transmute_handler.go:472-481`, `690-805`, `911-944` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedDelete=0` trên cả 5 bindings `46, 47, 48, 51, 52`) | **PASS** |
| **F2 (R2/R3) + AF-01 + AF-05** | Đồng nhất `_gpay_id` giữa Shadow và Master trong `RunEvents` (`lookupShadowGpayIDs`), cho phép cập nhật `"_gpay_id" = EXCLUDED."_gpay_id"` trong `bulkUpsertMaster` khi `conflictTarget != "_gpay_id"`, đọc động khóa `"pk"` từ `transform_spec` và chỉ truy vấn `missingGpayIDs` kèm `uniqueStrings(out)` trong `resolveSourceIDsForSegmentB` / `executeHealSegB`. | `internal/service/master/transmuter.go:690-696`, `752-764`, `1431-1477`, `1740-1751`; `internal/service/recon/recon_engine_segment_b.go:27,80`; `internal/handler/recon/recon_execute_heal_handler.go:307-337`, `444-548`, `732-780`, `829` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`gpayMismatch=0` cho cả 5 bindings `46, 47, 48, 51, 52`) | **PASS** |
| **F3 (R2/R3) + AF-03 + AF-04** | Quét bổ sung các dòng cũ trên Shadow (`_gpay_id <= initialCursor`) theo `cutoff = LastSuccessAt - 10m` khi chạy Incremental Sync (`syncUpdatedShadowRowsBeforeCursor`), bảo toàn `LastCursorJSON` (chỉ reset `{}` khi `runType == "full"`), và tự động bắn NATS `cdc.cmd.transmute-shadow` (chia chunk `healChunkSize = 200`, `"shadow_connection_key": "default_shadow"`) trong cả `executeHealSegA` và `executeLegacyHeal`. | `internal/service/master/transmuter.go:357-365`, `551-563`, `1510-1552`; `internal/handler/recon/recon_execute_heal_handler.go:280-305`, `339-392`; `internal/handler/recon/recon_check_heal_handler.go:308-311` | `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`), `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`droppedUpdate=0` và bảo toàn `last_cursor_json` trên cả 5 bindings `46, 47, 48, 51, 52`) | **PASS** |
| **F4 (R1/R3) + AF-05** | Gộp `MappedData` và khử trùng lặp theo `PrimaryKeyValue` (`dedupByConflictKey`) có đối chiếu thứ tự `r.SourceTsMs` **trước khi** chia nhóm `getSignature(r)` trong `BatchBuffer.batchUpsert`, và duyệt tuần tự theo slice `sigOrder`. | `internal/handler/shadow/batch_buffer_utils.go:90-153`; `internal/handler/shadow/batch_buffer.go:724-749` | `TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents` (`PASS`), `TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder` (`PASS`) | **PASS** |
| **F5 (R2/R3)** | Hợp nhất (union) toàn bộ keys của mọi bản ghi trong batch (`unionSortedKeys(records)`) trong `bulkUpsertMaster` thay vì chỉ lấy `sortedKeysAny(records[0])`. | `internal/service/master/transmuter_utils.go:222-251`; `internal/service/master/transmuter.go:1716-1721` | `TestUnionSortedKeys` (`PASS`), `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`PASS`) | **PASS** |

---

## 2. Kết quả Test & Runtime Thực tế (G2, G3, G4, G5, G6)

### 2.1. Unit Tests & Package Regression Suite (Round 2 — Commit `08c05a8`)
- **Lệnh đã chạy:**
  ```bash
  go test -v -count=1 ./internal/handler/shadow/... ./internal/handler/master/... ./internal/handler/recon/...
  ```
- **Output thực tế:**
  ```text
  === RUN   TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents
  --- PASS: TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents (0.00s)
  === RUN   TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder
  --- PASS: TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder (0.00s)
  ok  	centralized-data-service/internal/handler/shadow	0.912s
  === RUN   TestIsTransientError_CoversConnectionResetAndBadConnection
  --- PASS: TestIsTransientError_CoversConnectionResetAndBadConnection (0.00s)
  === RUN   TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails
  --- PASS: TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails (0.06s)
  ok  	centralized-data-service/internal/handler/master	3.741s
  ok  	centralized-data-service/internal/handler/recon	2.066s
  ```

### 2.2. Integration Tests & Đối soát Dữ liệu Thực trên PostgreSQL cho CẢ 5 Active Bindings (`46, 47, 48, 51, 52`)
- **Lệnh đã chạy:**
  ```bash
  go test -v -count=1 ./internal/service/master/... -run "TestDeterministicGpayID|TestInferLastGpayIDFromMaster|TestUnionSortedKeys|TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres|TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift"
  ```
- **Output thực tế (Heal Run + Idempotency Verification):**
  ```text
  === RUN   TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift
      transmuter_gpayid_test.go:412: Discovered active master_binding IDs: [46 47 48 51 52]
      transmuter_gpayid_test.go:517: [BEFORE] Binding 46 (conn=default_master, master_payment_bill_service.payment_bills): driftedIDs=0, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:535: [INCREMENTAL CHECK] Binding 46: scanned=25 updated=5 cursor={"last_gpay_id": 105963391826264116}
      transmuter_gpayid_test.go:569: [AFTER] Binding 46 (conn=default_master, master_payment_bill_service.payment_bills): activeShadow=54755, masterCount=54755, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:517: [BEFORE] Binding 47 (conn=default_master, master_payment_bill_service.payment_bills_1): driftedIDs=24, droppedDelete=0, droppedUpdate=0, gpayMismatch=24
      transmuter_gpayid_test.go:523: [HEAL RUN] Binding 47 (conn=default_master, master_payment_bill_service.payment_bills_1): scanned=24 inserted=0 updated=24 skipped=0
      transmuter_gpayid_test.go:535: [INCREMENTAL CHECK] Binding 47: scanned=25 updated=5 cursor={"last_gpay_id": 105963391822069811}
      transmuter_gpayid_test.go:569: [AFTER] Binding 47 (conn=default_master, master_payment_bill_service.payment_bills_1): activeShadow=54755, masterCount=54755, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:517: [BEFORE] Binding 48 (conn=default_master, master_centrallized_export_service.export_jobs): driftedIDs=0, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:535: [INCREMENTAL CHECK] Binding 48: scanned=3 updated=2 cursor={"last_gpay_id": 105937024405471269}
      transmuter_gpayid_test.go:569: [AFTER] Binding 48 (conn=default_master, master_centrallized_export_service.export_jobs): activeShadow=540, masterCount=540, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:517: [BEFORE] Binding 51 (conn=master_2, master_centrallized_export_service.export_jobs_2): driftedIDs=523, droppedDelete=1, droppedUpdate=0, gpayMismatch=521
      transmuter_gpayid_test.go:523: [HEAL RUN] Binding 51 (conn=master_2, master_centrallized_export_service.export_jobs_2): scanned=522 inserted=0 updated=522 skipped=0
      transmuter_gpayid_test.go:535: [INCREMENTAL CHECK] Binding 51: scanned=1 updated=0 cursor={"last_gpay_id": 105937024405471269}
      transmuter_gpayid_test.go:569: [AFTER] Binding 51 (conn=master_2, master_centrallized_export_service.export_jobs_2): activeShadow=540, masterCount=540, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
      transmuter_gpayid_test.go:517: [BEFORE] Binding 52 (conn=default_master, master_centrallized_export_service.export_jobs_1): driftedIDs=4, droppedDelete=1, droppedUpdate=0, gpayMismatch=2
      transmuter_gpayid_test.go:523: [HEAL RUN] Binding 52 (conn=default_master, master_centrallized_export_service.export_jobs_1): scanned=3 inserted=0 updated=3 skipped=0
      transmuter_gpayid_test.go:535: [INCREMENTAL CHECK] Binding 52: scanned=1 updated=0 cursor={"last_gpay_id": 105937024405471269}
      transmuter_gpayid_test.go:569: [AFTER] Binding 52 (conn=default_master, master_centrallized_export_service.export_jobs_1): activeShadow=540, masterCount=540, droppedDelete=0, droppedUpdate=0, gpayMismatch=0
  --- PASS: TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift (1.43s)
  PASS
  ok  	centralized-data-service/internal/service/master	2.294s
  ```

---

## 3. Kết quả Decision Gate DG3, Security & Final Audit (G7)

- **DG1 Solution Gate (Re-Plan `AF-01..AF-05`):** `06_laya_summary_dg1_20261020T160744.json` (`trusted_rounds=["r3"]`) | Gate: `PASS`
- **DG2 Code Demo Gate (Re-Plan `AF-01..AF-05`):** `06_laya_summary_dg2_20261020T160746.json` (`trusted_rounds=["r3"]`) | Gate: `PASS`
- **DG3 Post-Implementation Final Gate (Round 2 — sau commit `08c05a8`):** `06_laya_summary_dg3_20261020T164916.json` (`trusted_rounds=["r2", "r3"]`) | Gate: `PASS`
- **Security Gate (`/security-agent`):** `CLEAN_DIFF` & `CLEAN_WORKSPACE` (0 hardcoded secrets, 0 PII leaks, 0 SQL injection).
- **Final Adversarial Audit Report:** `audit_report_shadow_master_update_drop.md`

