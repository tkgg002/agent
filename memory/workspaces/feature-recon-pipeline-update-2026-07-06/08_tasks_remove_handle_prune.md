# Danh sách Task - Loại bỏ handlePrune & Tái cấu trúc Routing

- [x] Thực hiện chỉnh sửa `recon_check_handler.go` qua sub-agent/Muscle.
  - [x] Tái cấu trúc logic routing trong `HandleReconCheck`.
  - [x] Xóa phương thức `handlePrune`.
  - [x] Thêm phương thức `handleCheckAllSegmentB` và `executeCheckSegmentB`.
  - [x] Thay thế `executeStandardCheck` bằng `executeCheckSegmentA` (tích hợp `executeFullDiff` làm case trong switch).
- [x] Chạy build dự án để đảm bảo dự án compile thành công.
- [x] Triển khai hàm `ListActiveRegistries` trong `recon_engine_run.go` để lấy danh sách active registries cho Segment A.
- [x] Triển khai hàm `TimeBoundedDiffMissingFromMaster` trong `recon_tier_b.go` để đối soát sự khác biệt giữa shadow và master theo thời gian.
- [x] Biên dịch và chạy toàn bộ unit test thành công 100%.


