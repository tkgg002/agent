# 00_context: Bối Cảnh & Phạm Vi Đề Án Chuẩn Hóa SFTP Kafka Connect Reconcile

## 1. Bối Cảnh Hệ Thống (System Context)
Hệ thống Đối soát (Reconciliation Engine) thuộc nền tảng Goopay chịu trách nhiệm đối soát giao dịch hàng ngày giữa Hệ thống Ví/Cổng thanh toán nội bộ và các Ngân hàng / Cổng trung gian thanh toán đối tác.
Dữ liệu đối soát từ phía đối tác được truyền thông qua giao thức mạng **SFTP** dưới dạng các tệp tin CSV (ví dụ: `reconcile_20260814_01.csv`).

Để tự động hóa quá trình đưa dữ liệu này vào đường ống dữ liệu CDC / Event Streaming, đội ngũ phát triển sử dụng plugin mã nguồn mở **`kafka-connect-fs`** (tác giả *mmolimar*) chạy trên hạ tầng **Kafka Connect**.

## 2. Phạm Vi Đề Án (Scope)
- **Đối tượng rà soát:** Cấu hình JSON Connector, hạ tầng SFTP Server, đường ống Kafka Topic, và quy trình xử lý dữ liệu đằng sau Consumer.
- **Phạm vi chuyển đổi:** Chuyển dịch toàn bộ cấu hình từ thử nghiệm cá nhân (Local Docker Desktop) sang tiêu chuẩn **Production-Ready Enterprise**.
- **Mục tiêu:** Tháo gỡ 100% rủi ro bảo mật (Tripwire 1), môi trường ảo (Tripwire 2), quá tải server (Tripwire 3), mất dữ liệu dở dang (Tripwire 4), sai lệch cấu hình header (Tripwire 5), xáo trộn thứ tự giao dịch (Tripwire 6), nghẽn directory listing (Dọn rác), đồng thời bổ sung 3 Cổng An toàn Enterprise (DLQ, Decimal Precision, Partitioning Key).
