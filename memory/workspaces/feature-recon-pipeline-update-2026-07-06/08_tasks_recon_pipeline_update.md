# Danh sách Task - Cập nhật Reconciliation UI & API Pipeline

- [x] Task 1: Thiết kế và phân tích mã nguồn chi tiết (Tạo `12_implementation_plan_recon_pipeline_update.md`)
- [x] Task 2: Cập nhật Frontend (`ConfirmDestructiveModal.tsx`) để tự động điền Range 30 ngày cho Full Search/Deep Check
- [x] Task 3: Cập nhật Backend (`cdc-cms-service`) để tính toán và trả về trường `heal_needed` trong API `/api/reconciliation/report`
- [x] Task 4: Cập nhật Frontend (`useReconStatus.ts`, `DataIntegrity.tsx`, `ReconPipelineGrid.tsx`) để nhận diện trường `heal_needed` và điều chỉnh điều kiện hiển thị nút "Chữa lành"
- [x] Task 5: Kiểm thử và xác nhận hoạt động (chạy build, test, audit chất lượng đầu ra)
