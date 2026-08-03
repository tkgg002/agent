# Yêu cầu Tối ưu hóa SLOW SQL System Health Queries (FixSystemHealthQueriesSlowSql)

## 1. Bối cảnh & Hiện trạng
Hệ thống ghi nhận 1 câu truy vấn SQL bị SLOW SQL (>= 200ms) tại file `internal/infra/observability/system_health_queries.go`:

- **Slow Query:** 205.653ms
  - File: `internal/infra/observability/system_health_queries.go:54` (Hàm `queryReconciliation` thuộc System Health Metrics Collector)
  - SQL:
    ```sql
    SELECT DISTINCT ON (CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END)
           id, run_id, segment, shadow_schema, shadow_table, ...
    FROM cdc_reconciliation_report
    ORDER BY CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END, checked_at DESC
    ```
  - Nguyên nhân: Hàm `queryReconciliation` dùng `DISTINCT ON` với biểu thức `CASE WHEN ...` trên bảng `cdc_reconciliation_report` mà **không khoanh vùng thời gian `checked_at`**. Postgres buộc phải Seq Scan toàn bộ dữ liệu lịch sử đối soát và Sort trên đĩa/RAM.

## 2. Mục tiêu (Definition of Done)
- [ ] Bổ sung cờ time-window pruning `WHERE checked_at >= NOW() - INTERVAL '7 days'` (hoặc `24 hours`) trong câu SQL của `queryReconciliation`.
- [ ] Đạt mục tiêu thời gian truy vấn giảm từ **205ms xuống < 10ms**.
- [ ] Đảm bảo 100% không ảnh hưởng đến dữ liệu System Health Metrics thu thập được.
