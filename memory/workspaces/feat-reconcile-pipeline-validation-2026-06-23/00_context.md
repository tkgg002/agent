# Context: Reconcile Pipeline Validation

## Yêu cầu từ User
- `reconcile. chỉ chạy trên những pipeline đã đủ source, shadow, master`
- Chỉ chạy tính năng Reconcile trên những pipeline (hoặc binding) có đầy đủ cấu hình:
  - Source Connection (Nguồn dữ liệu nguồn)
  - Shadow Connection (Nguồn dữ liệu shadow)
  - Master Connection (Nguồn dữ liệu master)

## Bối cảnh & Hệ thống liên quan
- Module: `centralized-data-service`
- Service: `recon` (Reconciliation Engine)
- Các file liên quan (dự kiến):
  - `internal/service/recon/recon_engine.go` hoặc `recon_engine_run.go` hoặc `recon_tier_b.go` / `recon_tier_a.go`
  - Các cấu trúc định nghĩa pipeline hoặc database connection bindings.

## Mục tiêu (DoD - Definition of Done)
1. Xác định nơi Reconcile Engine khởi chạy một pipeline / binding.
2. Kiểm tra xem pipeline / binding đó có đầy đủ cấu hình `source`, `shadow`, và `master` hay không.
3. Nếu không đủ cấu hình, bỏ qua việc chạy Reconcile cho pipeline đó (hoặc log cảnh báo, không thực thi).
4. Viết unit test hoặc verify cơ chế này hoạt động chính xác.
