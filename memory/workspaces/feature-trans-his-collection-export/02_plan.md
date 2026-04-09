# Implementation Plan: Export TransHisCollection

**Mục tiêu (Objectives)**:
- Hỗ trợ loại giao dịch nội bộ mới: Khai báo `INTERNAL_BANK_TRANSFER`.
- Cấu hình file excel xuất ra: Thêm đủ các field Ngân hàng gửi/nhận, Tài khoản gửi/nhận, bỏ field Phí...
- Điều chỉnh query filter `sysTrans`.

**Brain Delegation Plan**:
Sử dụng Muscle qua lệnh `/muscle-execute` để thực thi:
1. `utils/constants.ts`: Thêm `INTERNAL_BANK_TRANSFER`.
2. `trans-his-collection-export.pure.ts`: Sửa logic filter, sửa danh sách column `getConfig()`, cập nhật `transformRow()`.
3. Test và Verify.
