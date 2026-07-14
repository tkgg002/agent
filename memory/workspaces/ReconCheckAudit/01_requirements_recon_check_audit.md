# Yêu cầu - Audit luồng đối soát hash_window và lỗi timeout

Phân tích và báo cáo chi tiết luồng xử lý của yêu cầu POST `/api/reconciliation/check?type_recon=hash_window`.
- Các hàm/phương thức đi qua ở cả `cdc-cms-service` và `centralized-data-service`.
- Nguyên nhân gây ra lỗi `dst hash window ...: timeout: context deadline exceeded`.
- Cấu trúc Trace OTEL của hành động này trên Signoz.
