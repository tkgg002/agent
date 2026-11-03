# 08. TASKS — `dlq_188_missing_oct30_20261030`

## Task T1: Điều tra & Khắc phục dứt điểm `DLQ / Lỗi = 188` ngày `2026-10-30`, lỗi MongoDB `$numberInt`, và lệch cột `Master Sync` (`[x] COMPLETED`)
- **Mô tả & Context:**
  1. Unwrap Extended JSON (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`) trong `mongo_type_converter.go:310-379` và bọc `bson.M{"$literal": v}` trong `transmuter_mongo.go:245-261`.
  2. Ghi đầy đủ nguyên nhân lỗi DB kèm `source_ids` vào `error_message` của `transmute_poison_pill` tại `transmute_handler.go:682-728` và bổ sung `run_type` tại `transmute_handler.go:401`.
  3. Khử trùng lặp `transmute_poison_pill` theo `source_ids->>0` khi tính `dlq_count` và tách chuẩn `snapshot_master` (`triggered_by IN ('snapshot', 'snapshot-hook', 'admin', 'manual', 're-sync', 'resync')`) tại `activity_log_read_repo_gorm.go:428-470`.
  4. Hiển thị `transmute_poison_pill` và `_source_id` trên `ReconPipelineGrid.tsx:486, 1192-1255`, `ActivityLog.tsx:48-55, 107-112, 169, 246-260`, và truyền `status='failed'` trong `useReconStatus.ts:163-176` / `DataIntegrity.tsx:95`.
- **Logs/Test:** Biên lai thực thi thật `EV-20261030T150146-cd4d`, `EV-20261030T150221-e982`, `EV-20261030T150611-9846`, `EV-20261030T150702-86fd`, `EV-20261030T150820-d5b7`.
- **Definition of Done:** Pass toàn bộ unit/integration/live DB test, khởi chạy lại `cds-worker` (`PID 72187`), pass `DG3` sau lần sửa cuối (`06_laya_summary_dg3_20261030T151148.json`), và pass `verify_governance.py`.
