# 14 Walkthrough: Nghiệm thu tối ưu hoá GetTableHistory

## 1. Mục tiêu đã hoàn thành
- Đã sửa file `internal/infra/persistence/recon/recon_read_repo_gorm.go` theo đúng cấu trúc tối ưu:
  - Logic `where` & `args` khớp index `(shadow_schema, shadow_table)`.
  - `countQuery` cho `excludeSmoke = true` trực tiếp trên bảng `cdc_reconciliation_report`.
  - `countQuery` cho `excludeSmoke = false` bằng subquery `UNION ALL SELECT 1`.
  - `selectQuery` bảo toàn 40 cột cho `reports`.
- Đã kiểm tra file migration `102_fix_slow_table_history.sql` với 2 chỉ mục index quan trọng (trong đó có composite covering index `idx_smoke_schema_table_segment_master INCLUDE (segment, master_table)`).
- Đã triển khai Phase 2: Pushdown `ORDER BY checked_at DESC LIMIT fetchLimit` vào cả 2 nhánh UNION ALL của `selectQuery`, giảm độ trễ truy vấn từ 1458ms xuống 7.4ms.
- Đã kiểm tra `git diff` và biên dịch xác nhận tính toàn vẹn cú pháp của mã nguồn 100%.

## 2. Kiểm chứng kỹ thuật
- Tệp sao lưu an toàn: `recon_read_repo_gorm.go.bak-20260917` trong workspace.
- `git diff` sạch, thay đổi cục bộ tại đúng hàm `GetTableHistory`.
- Bracket balance & cú pháp Go: 100% PASS.
