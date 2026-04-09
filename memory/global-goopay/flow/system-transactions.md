### **Danh sách các luồng giao dịch chính của hệ thống**

1.  **Luồng Thanh toán hóa đơn (Payment Bill Flow)**
    *   **Mô tả:** Khách hàng hoặc merchant tạo một hóa đơn thanh toán, sau đó người dùng cuối thực hiện thanh toán qua nhiều kênh (ví, thẻ ngân hàng).
    *   **Các service liên quan:** `payment-gateway`, `payment-bill-service`, `payment-service`, `core-trans-proxy-service`, `bank-transfer-service`, `wallet-service`.

2.  **Luồng Chuyển tiền trong ví (Wallet to Wallet Transfer)**
    *   **Mô tả:** Người dùng chuyển tiền từ ví của mình sang ví của người dùng khác.
    *   **Các service liên quan:** `wallet-trans-service` (sử dụng Saga Pattern), `wallet-service`, `notification-service`.

3.  **Luồng Nạp tiền vào ví (Cash-in Flow)**
    *   **Mô tả:** Người dùng nạp tiền vào ví GooPay từ tài khoản ngân hàng.
    *   **Các service liên quan:** `bank-transfer-service` (tạo tài khoản ảo), `[bank]-connector-service` (VD: `bidv-connector-service`), `core-trans-proxy-service`, `wallet-service`.

4.  **Luồng Rút tiền từ ví (Cash-out / Withdrawal Flow)**
    *   **Mô tả:** Người dùng rút tiền từ ví về tài khoản ngân hàng.
    *   **Các service liên quan:** `wallet-service`, `disbursement-service` (có thể), `bank-handler-service`.

5.  **Luồng Đặt vé xe (Booking Ticket Flow - Futa)**
    *   **Mô tả:** Người dùng tìm kiếm, đặt và thanh toán vé xe Futa, bao gồm cả việc mua bảo hiểm đi kèm.
    *   **Các service liên quan:** `booking-ticket-service`, `futa-connector-service`, `payment-service`, `insurance-service`.

6.  **Luồng Chi hộ (Disbursement Flow)**
    *   **Mô tả:** Một đơn vị (merchant/doanh nghiệp) tạo một "phiếu chi" để chi tiền hàng loạt cho nhiều người nhận (vào ví GooPay hoặc tài khoản ngân hàng).
    *   **Các service liên quan:** `disbursement-service` (Go), `wallet-service`, `external-adapter-service`.

7.  **Luồng Hoàn tiền (Refund Flow)**
    *   **Mô tả:** Xử lý các yêu cầu hoàn tiền cho các giao dịch đã thực hiện, có thể tự động hoặc thủ công qua admin portal.
    *   **Các service liên quan:** `payment-bill-service`, `booking-ticket-service`, `wallet-service`.

8.  **Luồng Đối soát (Reconciliation Flow)**
    *   **Mô tả:** Quy trình chạy định kỳ (cronjob) để đối chiếu dữ liệu giao dịch giữa hệ thống GooPay và các đối tác (ngân hàng, Futa).
    *   **Các service liên quan:** `reconcile-service` (Go), `scheduler-service`.

9.  **Luồng Xác thực định danh điện tử (eKYC Flow)**
    *   **Mô tả:** Người dùng tải lên giấy tờ tùy thân để xác thực tài khoản.
    *   **Các service liên quan:** `vmg-ekyc-connector-service`, `user-service`, `profile-service`.

10. **Luồng Quản lý Hỗ trợ khách hàng (Ticket Hub Flow)**
    *   **Mô tả:** Người dùng hoặc admin tạo các "phiếu hỗ trợ" (ticket) để giải quyết các vấn đề phát sinh.
    *   **Các service liên quan:** `ticket-service` (sử dụng CQRS).
