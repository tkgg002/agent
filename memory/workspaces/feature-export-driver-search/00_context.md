# Context: Driver Info & Approximate Search for Exports

**Feature**: 
1. Bổ sung thông tin driver (Driver ID, Driver Name, Driver Phone) cho `PaymentBillExport` và `PaymentHistoryExport`.
2. Tìm kiếm data gần đúng (Require 3 ký tự trở lên) cho các field:
   - Mã đơn hàng
   - Mã merchant
   - Tài khoản Merchant (Merchant email)
   - Mã Payer
   - Mã Payee
   - Mã GD Merchant
   - Mã GD Đối tác

**Scope**:
- `centralized-export-service`
  - `PaymentBillExport` logic, pure functions, handler
  - `PaymentHistoryExport` logic, pure functions, handler
- Có thể liên quan đến query và data trả về từ `payment-service` hoặc DB structure.
