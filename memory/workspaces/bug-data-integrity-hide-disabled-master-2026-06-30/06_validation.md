# Validation Plan: Hide Disabled Master Tables in Data Integrity

## Automated Tests
- Chạy build local frontend để đảm bảo TypeScript check và bundler không gặp lỗi:
  ```bash
  npm run build
  ```
  hoặc
  ```bash
  npm run check
  ```

## Manual Verification
- Truy cập vào dashboard đối soát `/data-integrity` trên localhost.
- Xác nhận bảng `payment_bills` (với master `master_payment_bill_service`) đã bị ẩn hoàn toàn khỏi UI (không xuất hiện ở tab Pipelines, tab Tổng quan và không được tính vào card số liệu thống kê ở đầu trang).
