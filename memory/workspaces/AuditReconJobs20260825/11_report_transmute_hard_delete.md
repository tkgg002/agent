# Report — Transmute Hard Delete & Toast Trace ID Implementation

## 1. Summary of Changes
| File | Lines Changed | Description |
| :--- | :---: | :--- |
| `centralized-data-service/internal/service/master/transmuter.go` | ~75 dòng | Tách riêng dòng `_deleted=true`, không bulk upsert, gọi `hardDeleteMasterByGpayIDs` thực thi `DELETE FROM master WHERE _gpay_id IN (?)`. |
| `centralized-data-service/internal/service/master/transmuter_orphan_test.go` | ~130 dòng | Bổ sung unit test `TestTransmuter_FullSyncHardDeleteWhenShadowMarkedDeleted`. |
| `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go` | ~40 dòng | Bổ sung NATS publish `cdc.cmd.transmute-shadow` khi prune orphan ở `executeHealSegA`. |
| `centralized-data-service/internal/service/recon/recon_engine.go` | ~15 dòng | Thêm `natsPub NatsPublisher` vào `ReconCore` và method `SetNatsPublisher`. |
| `centralized-data-service/internal/service/recon/recon_tier_a.go` | ~30 dòng | Bổ sung NATS publish `cdc.cmd.transmute-shadow` khi prune orphan ở `RunOrphanPrune`. |
| `centralized-data-service/internal/server/server_setup.go` | ~4 dòng | Wire `natsClient.Conn` vào `reconCore.SetNatsPublisher`. |
| `cdc-cms-web/src/utils/actionToast.tsx` | ~25 dòng | Format hiển thị đồng thời cả `Job ID` và `Trace ID` (copyable). |
| `cdc-cms-web/src/hooks/useReconStatus.ts` | ~10 dòng | Truyền `traceId` và gắn header `X-Correlation-Id`. |
| `cdc-cms-web/src/pages/DataIntegrity.tsx` | ~15 dòng | Gọi `createActionTrace` và truyền `traceId` vào `showActionToast`. |

## 2. Total Diff Metrics
- Backend (Go): ~294 lines added/modified across 6 files.
- Frontend (TSX/TS): ~50 lines added/modified across 3 files.
