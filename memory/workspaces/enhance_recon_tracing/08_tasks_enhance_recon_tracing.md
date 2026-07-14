# Danh sách Task: Rà soát & Bổ sung chi tiết Tracing cho Tiến trình Đối soát (Reconcile)

- `[x]` Phase 1: Khảo sát & Thiết kế Tracing
  - `[x]` Tìm kiếm các điểm ghi log/trace chính trong worker (`centralized-data-service`).
  - `[x]` Định vị code xử lý handler NATS `cdc.recon.check` và các function con.
  - `[x]` Phân tích cách implement OTel tracer hiện tại trong project.
- `[x]` Phase 2: Triển khai & Bổ sung Span chi tiết
  - `[x]` Bổ sung trace span cho các step đối soát cụ thể.
  - `[x]` Đảm bảo chuyển context `ctx` chính xác qua các hàm nội bộ.
- `[x]` Phase 3: Kiểm thử & Xác minh
  - `[x]` Chạy local test và xem log traces để xác nhận các span con được spawn.
