# 12_implementation_plan_smoke_trace_and_trans_his_drift.md
## Kế Hoạch Triển Khai: Chuẩn Hóa Span Traces Recon Smoke & Xử Lý Lệch trans_his

### 1. Mục Tiêu Triển Khai
1. Khắc phục triệt để cấu trúc cây Span Traces trên SigNoz: Mỗi pipeline có span cha riêng, chứa đúng các child spans của chính nó.
2. Khắc phục tranh chấp khóa Advisory Lock giữa các connector có cùng bảng `trans_his`.
3. Giải quyết dứt điểm độ lệch Master `+458` bằng cơ chế dọn dẹp tombstone `_deleted = true` từ Shadow sang Master.
4. Minh bạch hóa cơ chế đếm MongoDB `EstimatedDocumentCount()` vs Exact Count.

---

### 2. Chi Tiết Kế Hoạch Triển Khai

#### Phase 1: Tái Cấu Trúc Tracing & Pipeline Execution trong `recon_smoke.go`
- **Tệp tin:** `centralized-data-service/internal/service/recon/recon_smoke.go`
- **Các bước cụ thể:**
  1. Cấu trúc lại danh sách Pipeline Execution:
     - Tạo danh sách `PipelineItem` đại diện cho từng tuyến dữ liệu độc lập.
     - Với các bảng có nhiều Master Binding (như `export_jobs`), mỗi binding là 1 `PipelineItem` riêng biệt (`export_jobs`, `export_jobs_1`, `export_jobs_2`).
  2. Per-Pipeline Lifecycle:
     - Khởi tạo Span `pipeline: [<Source>] -> [<Shadow>] -> [<Master>]` cho từng pipeline khi pipeline đó bắt đầu chạy.
     - Chạy `prefetch` và `source_shadow` (Segment A) dưới context của pipeline đó.
     - Chạy `shadow_master` (Segment B) dưới context của pipeline đó.
     - Đóng Span của pipeline ngay khi hoàn thành kiểm tra pipeline đó.
  3. Sử dụng Cache `loadCountSnapshot` để các pipeline có cùng bảng Shadow (hoặc cùng Master) không bị COUNT(*) lặp lại.
  4. Cập nhật `withTableLock`:
     - Segment A: `rc.withTableLock(ctx, entry.QualifiedTarget())` thay vì `entry.TargetTable`.
     - Segment B: `rc.withTableLock(ctx, "reconB_"+ref.ShadowRel()+"_"+ref.MasterRel())` thay vì `ref.runName()`.

#### Phase 2: Khắc Phục Lệch Tombstone Master (+458)
- **Tệp tin:** `centralized-data-service/internal/service/master/transmuter.go` & `centralized-data-service/internal/service/recon/recon_execute_heal_handler.go`
- **Các bước cụ thể:**
  1. Thêm tác vụ đồng bộ Tombstone: Quét các dòng `_deleted = true` từ Shadow và thực hiện hard-delete tương ứng trên Master theo `_gpay_id`.
  2. Đảm bảo khi các dòng `_deleted = true` được dọn sạch khỏi Master, số lượng active Master khớp chính xác `234,756` của Shadow.

#### Phase 3: Biên Dịch & Nghiệm Thu (DoD Gate G1–G8)
- Biên dịch service: `go build ./cmd/worker` trong `centralized-data-service`.
- Chạy integration test xác nhận không có span collision và cây phân cấp đúng chuẩn.
- Báo cáo chi tiết diff mã nguồn cho User.
