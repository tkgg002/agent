# Tasks: Bridge Fix

> Date: 2026-04-14
> Phase: bridge_fix

## Checklist

- [x] T1: Tạo `ensureCDCColumns()` + `hasColumn()` + `tableExists()` helpers
- [x] T2: Gọi `ensureCDCColumns` trong `HandleAirbyteBridge` trước bridge
- [x] T3: `bridgeInPlace` đã OK (chạy sau ensureCDCColumns)
- [x] T4: `HandleBatchTransform` — check table + `_raw_data`, skip nếu chưa có
- [x] T5: `HandlePeriodicScan` + `HandleScanRawData` — check table + `_raw_data`
- [x] T6: Build OK (Worker + CMS)
- [x] T7: Swagger updated
- [ ] T8: Restart Worker → verify no errors (USER)
- [ ] T9: Activity Log hiện đúng status success/skipped (USER)
- [x] T10: `05_progress.md` updated
