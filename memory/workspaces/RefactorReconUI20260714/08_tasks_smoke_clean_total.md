# Danh sách Task: Cập nhật biến Total/Active sạch và mốc CheckedAt trong Smoke Recon Segment A

- [x] Lập kế hoạch chi tiết thiết kế thay đổi trong `recon_smoke.go`
- [x] Chỉnh sửa `RunTotalOnlyA` để tính toán `dstTotalClean` và gán vào `SmokeResult` cùng với `fromTime`
- [x] Build & Verify `centralized-data-service` để đảm bảo không bị lỗi biên dịch
- [x] Chạy linter kiểm tra quy trình `verify_governance.py`
- [x] Chỉnh sửa `RunTotalOnlyB` để tính toán `shadowTotalClean` và `masterTotalClean` và gán vào `SmokeResult` cùng với `fromTime`
- [x] Cập nhật test `TestReconCore_RunTotalOnlyB_Normal` để assert các chỉ số làm sạch

