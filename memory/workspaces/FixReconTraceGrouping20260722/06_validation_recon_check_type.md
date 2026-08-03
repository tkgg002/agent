# 06 Validation: Chuẩn Hoá CheckType Động (reconA / reconB) Cho CMS Reconciliation Report

## 1. Mục Đích & Bối Cảnh
Thay thế giá trị hardcode `CheckType = "chunk_stream_bucket"` trong [recon_job_worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go) thành phân loại động theo đúng segment của Pipeline:
- **`reconA`**: Áp dụng cho Chặng A (`source_shadow` — Nguồn Mongo/Source $\rightarrow$ Shadow Postgres).
- **`reconB`**: Áp dụng cho Chặng B (`shadow_master` — Shadow Postgres $\rightarrow$ Master Postgres).

---

## 2. GORM / Raw SQL Dry-Run Statement
```sql
-- DDL & SQL Statement tương ứng sinh ra khi ReconJobWorker lưu report cho Chặng A hoặc Chặng B:
INSERT INTO cdc_system.cdc_reconciliation_report (
    run_id, shadow_schema, shadow_table, master_schema, master_table, 
    source_db, source_table, source_type, source_host, source_count, dest_count, 
    segment, check_type, status, diff, missing_count, stale_count, stale_ids, 
    duration_ms, recon_start_time, recon_end_time, checked_at
)
VALUES (
    'job_12345', 'shadow_testpbs', 'payment_bills', 'public', 'payment_bills',
    '', '', '', '', 5507, 5507,
    'shadow_master', 'reconB', 'ok', 0, 0, 0, '{}', 399,
    '2026-07-22 09:25:00', '2026-07-22 09:25:00', NOW()
);
```

---

## 3. Kết Quả Kiểm Thử Tự Động (Unit Test Validation)
Chạy bộ test suite cô lập `recon_job_worker_test.go`:
```bash
$ go test ./internal/service/recon/...
ok  	centralized-data-service/internal/service/recon	0.720s
```
- **Test 1**: Verify `CheckType == "reconA"` khi `Segment == "source_shadow"` $\rightarrow$ **PASS**.
- **Test 2**: Verify `CheckType == "reconB"` khi `Segment == "shadow_master"` $\rightarrow$ **PASS**.

---

## 4. Đồng Bộ Hiển Thị Trên FE (`cdc-cms-web`)
Cập nhật hàm `levelLabel` trong [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx):
- `case 'reconA'`: Trả về nhãn `'reconA'`.
- `case 'reconB'`: Trả về nhãn `'reconB'`.
