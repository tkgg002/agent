# Kế hoạch triển khai chi tiết: Rà soát & Bổ sung chi tiết Tracing cho Tiến trình Đối soát (Reconcile)

Mục tiêu của kế hoạch này là bổ sung OTel child spans chi tiết cho các tiến trình đối soát Tier-2 (`RunHashWindowCheck`, `RunHashWindowCheckB`) và Tier-3 (`RunDeepCheck`, `RunDeepCheckB`) thuộc Segment A và Segment B trong `centralized-data-service`.

## 1. Các thay đổi đề xuất

### A. Tích hợp `ContextWithoutSkipTrace` vào `pkgs/observability/trace_helpers.go`
- Bổ sung helper `ContextWithoutSkipTrace(ctx context.Context) context.Context` để cho phép tắt cờ `skipTraceKey` khi gặp window/bucket bị lệch (drifted). Điều này giúp kích hoạt lại tracing chi tiết cho các DB query con bên dưới (như `ListIDTsInWindow`) mà không gây quá tải trace (trace explosion) cho các window sạch.

### B. Bọc các khối thực thi quan trọng vào child spans & tắt skip trace trên window drifted
- **Trong `internal/service/recon/recon_tier_a.go` (`RunHashWindowCheck`)**:
  - Bọc lời gọi `rc.pickScanRangeWithLag` vào child span `cdc.recon.pick_scan_range` (đã có span con, nhưng cần đảm bảo lan truyền chính xác).
  - Đối với từng window bị lệch trong loop: tắt skip trace bằng `observability.ContextWithoutSkipTrace(ctxLoop)` trước khi tạo `cdc.recon.drift_drill_down` span và gọi `ListIDTsInWindow`.
  - Đảm bảo bắt lỗi qua `defer observability.EndSpan(span, &err)` cho span gốc.
- **Trong `internal/service/recon/recon_tier_b.go` (`RunHashWindowCheckB`)**:
  - Đối với từng bucket bị lệch: tắt skip trace bằng `observability.ContextWithoutSkipTrace(ctxLoop)` trước khi tạo `cdc.recon.drift_drill_down_b` span và gọi `ListIDTsInWindow`.
- **Trong `internal/service/recon/recon_tier_b.go` (`RunDeepCheckB`)**:
  - Đối với từng bucket bị lệch: tắt skip trace bằng `observability.ContextWithoutSkipTrace(ctxLoop)` trước khi tạo `cdc.recon.drift_drill_down_b` span và gọi `ListIDTsInWindow`.

## 2. Kế hoạch kiểm thử (Verification Plan)
- **Kiểm thử tự động**: Chạy các unit test liên quan đến `recon_tier_a_test.go` và `recon_tier_b_test.go` để đảm bảo code biên dịch thành công và các test case logic không bị regression.
- **Kiểm thử tích hợp**: Chạy job đối soát thực tế và theo dõi log hoặc trace output (nếu môi trường local có SigNoz/Jaeger) để xác nhận phân cấp span hoạt động như mong đợi.
