# Specification: Compare Disbursement Trans His Export

## 01_spec.md - Đặc tả yêu cầu

### Đối tượng so sánh
1. **Source of Truth (Gốc)**:
   - Logic truy vấn: `Disbursement.export-trans-his` (NATS request).
   - Mapping dữ liệu: Xem tại code adapter Go và `info.md` (Goopay service).
2. **Target (Mới)**:
   - Centralized Export Service.
   - Domain: `disbursement`.
   - Feature: `disbursement-trans-his-export`.

### Yêu cầu chi tiết
- So sánh các tham số đầu vào (Params) và bộ lọc (Filters).
- So sánh danh sách các cột (Columns) trong file xuất ra.
- Kiểm tra các logic logic formatting (Status map, Expense Type map, Date format).
- Đảm bảo không mất dữ liệu hoặc sai lệch logic khi chuyển sang service tập trung.
