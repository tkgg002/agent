# Tasks — Transmute Hard Delete & Toast Trace ID

- [x] **Task 1: Transmuter Master Hard Delete on Oplog Delete**
  - Tách luồng trong `processBatch` của `transmuter.go`: `row.Deleted == true` gom vào `allGpayIDsToDelete`.
  - Tạo hàm `hardDeleteMasterByGpayIDs` thực thi `DELETE FROM master WHERE _gpay_id IN (?)`.
  - Không đưa các bản ghi `_deleted == true` vào `allRecords` của `bulkUpsertMaster`.
  - Áp dụng đồng bộ cho cả `copy_1_to_1` và `flatten`.

- [x] **Task 2: Segment A Orphan Prune Cascade Trigger to Master**
  - Cập nhật `executeHealSegA` trong `recon_execute_heal_handler.go`: Bắn NATS `cdc.cmd.transmute-shadow` mang `_source_ids` khi `pruned > 0`.
  - Cập nhật `RunOrphanPrune` trong `recon_tier_a.go`: Bắn NATS `cdc.cmd.transmute-shadow` khi `pruned > 0`.
  - Wire NATS publisher vào `ReconCore` (`recon_engine.go` & `server_setup.go`).

- [x] **Task 3: Action Toast Trace ID & Job ID on Frontend**
  - Cập nhật `actionToast.tsx` hiển thị đồng thời cả `Job ID` và `Trace ID`.
  - Cập nhật `useReconStatus.ts` nhận `traceId` và gắn header `X-Correlation-Id`.
  - Cập nhật `DataIntegrity.tsx` tạo `createActionTrace` và truyền vào `showActionToast`.

- [x] **Task 4: Build Verification**
  - Build Go worker binary (`go build ./cmd/worker`) -> PASS.
  - Build Vite bundle (`npm run build`) -> PASS.
