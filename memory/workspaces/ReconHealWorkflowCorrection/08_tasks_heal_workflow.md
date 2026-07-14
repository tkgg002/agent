# Danh sách Task - Sửa đổi Luồng và Trạng thái Chữa lành Đối soát

- `[x]` **Phase 1: Lập kế hoạch & Thiết kế**
  - `[x]` Tạo tài liệu `12_implementation_plan_heal_workflow.md` và artifact `implementation_plan.md`
  - `[x]` Đệ trình User phê duyệt kế hoạch
- `[x]` **Phase 2: Triển khai Backend (`cdc-cms-service`)**
  - `[x]` Cập nhật `ReleaseHealClaim` trong `reconciliation_report_repo.go` để tránh kẹt trạng thái
  - `[x]` Cập nhật `finalizeReport` trong `recon_execute_heal_handler.go` chỉ set `healed_at` khi hoàn thành 100% các loại lỗi
- `[x]` **Phase 3: Triển khai Frontend (`cdc-cms-web`)**
  - `[x]` Sửa đổi cột hiển thị (Thiếu, Lệch, Thừa) hiển thị remaining count dạng `Remaining / Original` trong `ExecuteHealModal.tsx`
  - `[x]` Vô hiệu hóa checkbox cho các loại lỗi đã được chữa lành hoàn toàn (Remaining = 0)
  - `[x]` Cập nhật `useEffect` gán checkbox mặc định động
- `[/]` **Phase 4: Kiểm thử và Xác nhận**
  - `[x]` Biên dịch và chạy thử hệ thống
  - `[ ]` Tạo walkthrough.md và đồng bộ vào workspace
