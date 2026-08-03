# Optimization Analysis Report: Indexing & SQL Refactoring for `listLatestPrimary`

## 1. Phân tích Tối ưu DB Indexes (Database Optimization)

Nguyên nhân làm cho câu SQL chạy tốn **341ms** là do PostgreSQL phải thực hiện **Sequential Scan + In-Memory Sort** trên toàn bộ bảng `cdc_recon_smoke_result` và `shadow_binding`.

### 🌟 Chỉ mục 1: Expression Index cho `cdc_recon_smoke_result`
Tạo Expression Index khớp 100% với mệnh đề `DISTINCT ON` & `ORDER BY`:
```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cdc_smoke_result_latest_distinct 
ON cdc_system.cdc_recon_smoke_result (
    COALESCE(shadow_schema, ''), 
    shadow_table, 
    COALESCE(NULLIF(master_schema, ''), ''), 
    COALESCE(NULLIF(master_table, ''), ''), 
    COALESCE(segment, 'source_shadow'), 
    checked_at DESC
);
```
👉 **Tác dụng**: Giúp Postgres đọc trực tiếp bản ghi mới nhất (`checked_at DESC`) từ B-Tree Index mà không cần tốn CPU/RAM để Sort lại toàn bộ bảng.

### 🌟 Chỉ mục 2: Partial Index cho `shadow_binding`
```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shadow_binding_active_rn 
ON cdc_system.shadow_binding (shadow_schema, shadow_table, updated_at DESC, id DESC) 
WHERE is_active = TRUE;
```
👉 **Tác dụng**: Tăng tốc tối đa cho hàm cửa sổ `ROW_NUMBER() OVER (PARTITION BY shadow_schema, shadow_table ORDER BY updated_at DESC, id DESC)` trong CTE `active_bindings`.

---

## 2. Tối ưu SQL Code (`recon_read_repo_gorm.go`)

### 💡 Cải tiến Mệnh đề `LEFT JOIN` (Loại bỏ `OR`)
Chuyển đổi điều kiện JOIN có `OR` nhiều nhánh sang so sánh chuẩn bằng `COALESCE`:
```sql
LEFT JOIN active_bindings sb 
  ON sb.shadow_table = s.shadow_table 
  AND COALESCE(s.shadow_schema, '') = COALESCE(sb.shadow_schema, '') 
  AND sb.rn = 1
```
👉 **Tác dụng**: Cho phép Postgres Planner chuyển từ `Nested Loop Join` sang `Hash Join` / `Index Join` với tốc độ cao hơn rất nhiều.
