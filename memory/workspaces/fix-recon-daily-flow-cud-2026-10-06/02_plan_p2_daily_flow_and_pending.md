# 02 — Plan: Phase P2 & Pending Issues Resolution

Ngày lập: 2026-10-12
Tôn chỉ: **The Single Best Approach** (Phương án tối ưu duy nhất, không đưa nhiều option bắt user chọn).

## Kiến trúc Tổng thể (High-level Architecture)

```
[ Browser / CMS UI ]
       │
       │ GET /api/reconciliation/daily-flow?shadow_schema=...&shadow_table=...&days=14&tz_offset=420
       ▼
[ cdc-cms-service (Fiber API) ]
       │
       ▼ ActivityLogReadRepo.GetDailyFlowAggregation(...)
[ PostgreSQL cdc_dw (Partitioned cdc_activity_log + cdc_reconciliation_report) ]
       │
       ├─ SQL Aggregation (Filter created_at >= NOW() - 14d -> Partition Pruning)
       ├─ Subquery 1: kafka-consumer & snapshot logs (Sum C/U/D, Ingest, Written, Filtered errors)
       ├─ Subquery 2: transmute & transmute_sub logs (Sum Inserted, Updated, Skipped, Filtered poison pills)
       ├─ Subquery 3: cdc_reconciliation_report (Daily Drift Metrics: Missing, Stale, Healed, Total Runs)
       ▼
[ Aggregated DailyFlowRecord[] ] -> Trả về JSON gọn gàng (< 2KB thay vì kéo 400 raw JSON rows)
```

## Các giai đoạn triển khai (Execution Stages)

### Giai đoạn 1: Fix Core Safety & Logging (CDS + CMS Router)
1. **CMS Router**: Sửa `func chain(...)` tại `internal/router/router.go` trả về `out[:len(out):len(out)]` để triệt tiêu lỗi aliasing memory.
2. **CDS Transmute Handler**: Bổ sung `"split_retry": true` vào `details` khi mẻ transmute gặp lỗi và phải đệ quy chia nhỏ (`len(subBatch) > 1`). Khi cô lập thành công (`len(batch) == 1`), đánh dấu rõ `"poison_pill": true`.

### Giai đoạn 2: Xây dựng Server-side Aggregation Query & API (CMS Backend)
1. **Query DTO**: Khởi tạo `DailyFlowParams` và `DailyFlowRow` trong `internal/app/queries/system/` (hoặc `queries/recon/`).
2. **Repository Aggregation**: Viết hàm `GetDailyFlow(ctx, params)` trong `activity_log_read_repo_gorm.go`:
   - Pruning partition `created_at >= NOW() - INTERVAL 'X days'`.
   - LATERAL join hoặc FQN match chính xác theo pipeline đã chọn.
   - Nhóm theo `DATE(started_at AT TIME ZONE 'UTC' + (tz_offset * INTERVAL '1 minute'))`.
   - Tách bạch rõ ràng giữa Snapshot và Stream CDC.
3. **API Handler & Route**:
   - Thêm handler `GetDailyFlow` tại `internal/api/system/activity_log_handler.go` (hoặc `api/recon/`).
   - Đăng ký route `GET /api/reconciliation/daily-flow` (và `/v1/reconciliation/daily-flow`).

### Giai đoạn 3: Tích hợp Frontend CMS Web
1. **Hook**: Thêm hook `useDailyFlow(params)` trong `src/hooks/useReconStatus.ts`. Tự động tính `tz_offset = -new Date().getTimezoneOffset()`.
2. **Grid Component**:
   - Thay thế việc gọi 2 hook `usePipelineActivityLog` (limit 200) và client reducer trong `ReconPipelineGrid.tsx` bằng dữ liệu từ `useDailyFlow`.
   - Xóa bỏ hoàn toàn cảnh báo `LOG_FETCH_LIMIT = 200` và tình trạng mất ngày cũ.
   - Hiển thị chính xác các chỉ số: Khối 1 (Snapshot), Khối 2 (Kafka Ingest), Khối 3 (Shadow Applied), Khối 4 (Master Transmuted), Khối 5 (Recon & DLQ thật).

### Giai đoạn 4: Script Bảo trì Dữ liệu cũ (Legacy Logs) & Verification
1. Viết script SQL `enrich_legacy_activity_logs.sql`: Backfill `shadow_schema` cho các log cũ thuộc các bảng có duy nhất 1 binding active.
2. Kiểm thử end-to-end trên Docker PostgreSQL thật (`cdc_dw`), verify EXPLAIN partition pruning và độ chính xác của bảng Daily Flow.
