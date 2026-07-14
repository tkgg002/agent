# Danh sách task chi tiết - Khắc phục Hồi quy Đối soát Smoke

- `[ ]` Khắc phục lỗi biên dịch/assertion trong `TestBuildCastExpr` ở `metadata_mapping_test.go`.
- `[ ]` Khắc phục lỗi làm tròn trong `TestHashWindowDriftDetection` ở `recon_hash_test.go`.
- `[ ]` Tái cấu trúc và tối ưu hóa `runLookbackCheckA` trong `recon_smoke.go` để tránh double query `pickScanRangeWithLag`.
- `[ ]` Tái kích hoạt (uncomment) các cuộc gọi `runLookbackCheckA` và `runLookbackCheckB` trong `recon_smoke.go`.
- `[ ]` Chạy và đảm bảo pass tất cả unit tests của service `recon`.
- `[ ]` Chạy script linter quy trình (`verify_governance.py`) để xác nhận tính tuân thủ.
