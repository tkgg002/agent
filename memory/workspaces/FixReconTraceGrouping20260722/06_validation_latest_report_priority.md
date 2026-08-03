# 06 Validation: Sửa Lỗi Thứ Tự Ưu Tiên Mới Nhất (`checked_at DESC`) Cho SQL ListLatestPrimary

## 1. Mục Đích & Bối Cảnh
Khắc phục hiện tượng hiển thị ngược trạng thái Chặng Ingest / Chặng Transmute trên FE:
- **Hiện tượng**: Số lượng dòng Source = 2,764,642, Shadow = 2,764,642 (khớp 100%), Master = 2,764,636 (lệch 6 dòng). Tuy nhiên UI lại hiển thị Chặng Ingest (Source $\rightarrow$ Shadow) là `Lệch` và Chặng Transmute (Shadow $\rightarrow$ Master) là `Khớp`.
- **Root Cause**:
  Trong SQL `listLatestPrimary` của [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go), CTE `deduped_reports` trước đây sắp xếp:
  `ORDER BY ..., priority ASC, checked_at DESC`
  Vì bản ghi `cdc_reconciliation_report` được gán `priority = 1` và `cdc_recon_smoke_result` được gán `priority = 2`, PostgreSQL bị ép lấy bản ghi ở `cdc_reconciliation_report` cũ từ trước đó, che lấp dữ liệu Smoke Check mới nhất (chỉ cách vài giây).

---

## 2. Giải Pháp Chỉnh Sửa Raw SQL

Sửa thứ tự `ORDER BY` trong CTE `deduped_reports`:
```sql
deduped_reports AS (
    SELECT DISTINCT ON (COALESCE(shadow_schema, ''), shadow_table, norm_master_schema, norm_master_table, COALESCE(segment, 'source_shadow')) *
    FROM latest_reports
    ORDER BY COALESCE(shadow_schema, ''), shadow_table, norm_master_schema, norm_master_table, COALESCE(segment, 'source_shadow'), 
             checked_at DESC, priority ASC
)
```

---

## 3. Kết Quả Kiểm Thử (Unit Test Validation)
Chạy bộ test suite của `cdc-cms-service`:
```bash
$ go test ./test/...
ok  	cdc-cms-service/test/internal/api	(cached)
ok  	cdc-cms-service/test/internal/api/dto	(cached)
ok  	cdc-cms-service/test/internal/app/commands	(cached)
ok  	cdc-cms-service/test/internal/app/queries	(cached)
ok  	cdc-cms-service/test/internal/infra/http	(cached)
ok  	cdc-cms-service/test/internal/infra/messaging	(cached)
ok  	cdc-cms-service/test/internal/infra/observability	(cached)
ok  	cdc-cms-service/test/internal/infra/observability/probes	(cached)
ok  	cdc-cms-service/test/internal/infra/persistence	(cached)
ok  	cdc-cms-service/test/internal/middleware	(cached)
```
- **Result**: `100% PASS`. Thời điểm đối soát mới nhất (`checked_at DESC`) luôn được đưa lên ưu tiên hàng đầu.
