# Spec: Rà soát & Bổ sung chi tiết Tracing cho Tiến trình Đối soát (Reconcile)

## 1. Yêu cầu chi tiết
- **Mục tiêu**: Bổ sung các span con (child spans) chi tiết cho tiến trình đối soát `cdc.recon.check` (hoặc các job đối soát chạy trên worker `centralized-data-service`).
- **Hiện trạng**: Trace hiện tại chỉ có các span cha (parent spans) ở mức thông báo khởi chạy job (ví dụ: `nats.HandleReconCheck`, `cdc.recon.check`). Tiến trình chạy mất nhiều thời gian (e.g. 46.79s) nhưng không hiển thị được các bước nhỏ bên trong (ví dụ: truy vấn database nguồn, shadow, so sánh hash, ghi report, trigger heal, v.v.), gây khó khăn cho việc debug và tối ưu hóa hiệu năng.
- **Yêu cầu cụ thể**:
  - Rà soát các hàm xử lý đối soát (đặc biệt là `ReconCheck`, `recon_tier_a`, `recon_hash`, `BucketCounts`, v.v.) trong `centralized-data-service`.
  - Tích hợp thêm OpenTelemetry Tracing (OTel Span) cho các giai đoạn quan trọng của quá trình đối soát:
    - Quét dữ liệu/so sánh hash (Hash comparison phase).
    - Lấy thông tin record detail (Detail check phase).
    - Tạo các bản ghi report và cập nhật trạng thái.
  - Đảm bảo các span con này kế thừa chính xác Context (TraceID/SpanID) từ span cha để hiển thị đúng phân cấp trên giao diện SigNoz/Jaeger.
