# Yêu cầu kỹ thuật: Tối ưu hoá SLOW SQL tại GetTableHistory & Composite Index Migration

## 1. Bối cảnh & Vấn đề (Context & Problem)
Endpoint lấy lịch sử đối soát theo bảng (`GetTableHistory`) bị chậm nghiêm trọng (độ trễ đo được 4639ms) khi truy vấn trên khối lượng bản ghi lớn của 2 bảng `cdc_system.cdc_reconciliation_report` và `cdc_system.cdc_recon_smoke_result`.

### Nguyên nhân gốc rễ (Root Cause):
1. **Clause WHERE phức tạp với toán tử OR cản trở Index Scan**:
   Mệnh đề `((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND master_table = ?))` khiến PostgreSQL không thể sử dụng hiệu quả index trên `(shadow_schema, shadow_table, checked_at DESC)` mà phải fallback sang Seq Scan hoặc Bitmap Index Scan tốn kém.
2. **Count Query bọc ngoài subquery 40 cột**:
   Cả nhánh `excludeSmoke = true` và `excludeSmoke = false` đều thực hiện `SELECT COUNT(*) FROM (SELECT id, run_id, ... 40 cột ...) AS subquery`, khiến PostgreSQL phải giải mã, chiếu cột, tính toán biểu thức `CASE WHEN` và `COALESCE` cho từng dòng trước khi đếm.
3. **Thiếu Composite Index hỗ trợ**:
   - `cdc_recon_smoke_result` chỉ có index trên `(shadow_table, checked_at DESC)`, thiếu `shadow_schema` dẫn đến filter `shadow_schema` phải recheck từng row.
   - `cdc_reconciliation_report` thiếu index trên `(master_table, checked_at DESC)`.

## 2. Mục tiêu kỹ thuật (Objectives)
1. **Sửa code Go tại `recon_read_repo_gorm.go`**:
   - Thay đổi điều kiện `where` và `args` khi có `shadowSchema`:
     ```go
     where := "shadow_table = ? OR master_table = ?"
     args := []interface{}{table, table}
     if shadowSchema != "" {
         where = "shadow_schema = ? AND shadow_table = ?"
         args = []interface{}{shadowSchema, table}
         if masterTable != "" {
             where += " AND (segment <> 'shadow_master' OR master_table = ?)"
             args = append(args, masterTable)
         }
     }
     ```
   - Tối ưu `countQuery` trong nhánh `excludeSmoke = true`:
     `SELECT COUNT(*) FROM cdc_system.cdc_reconciliation_report WHERE (%s) AND check_type NOT IN ('smoke', 'segment_b_smoke')` (loại bỏ subquery 40 cột).
   - Tối ưu `countQuery` trong nhánh `excludeSmoke = false`:
     `SELECT COUNT(*) FROM ((SELECT 1 FROM cdc_reconciliation_report WHERE ...) UNION ALL (SELECT 1 FROM cdc_recon_smoke_result WHERE ...)) AS unioned` với `unionArgs` chính xác.
2. **Xác thực và áp dụng Migration SQL 102**:
   - File `migrations/schema/recon_dlq/102_fix_slow_table_history.sql` tạo index `idx_smoke_schema_table_checked` và `idx_recon_master_table`.
   - Thực thi migration vào container Docker `gpay-postgres-cdc` (`cdc_dw`).
3. **Verification (DoD)**:
   - `git diff` sạch, đúng yêu cầu.
   - `go build` thành công 100% không lỗi cú pháp.
   - Báo cáo chi tiết Before vs After.
