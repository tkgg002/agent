# Danh sách Task: Khắc phục lỗi lệch pha đối soát cho các bảng có dữ liệu ghi liên tục

## Phase 1: Phân tích & Lập Kế hoạch
- [ ] Phân tích code hiện tại của các API đối soát và các hàm stream/hash.
- [ ] Thiết kế phương án tích hợp mốc chặn thời gian `upper` vào `FullIDDiffMissingFromShadow`, `RunOrphanPrune` và `BucketHash`.
- [ ] Xác minh ảnh hưởng hiệu năng của việc lọc thời gian trên MongoDB.

## Phase 2: Triển khai Kỹ thuật (Muscle)
- [ ] Sửa đổi `FullIDDiffMissingFromShadow` trong `recon_tier_a.go`.
- [ ] Sửa đổi `RunOrphanPrune` trong `recon_tier_a.go`.
- [ ] Cập nhật signature và logic của `BucketHash` trong `ReconSourceAgent` (`recon_hash.go`) để nhận `upper time.Time` (hoặc cấu hình tương đương).
- [ ] Cập nhật signature và logic của `BucketHash` trong `ReconDestAgent` (`recon_dest_hash.go`) để nhận `upper time.Time` (hoặc cấu hình tương đương).
- [ ] Cập nhật `RunDeepCheck` trong `recon_tier_a.go` để truyền mốc chặn trên thời gian cho `BucketHash`.

## Phase 3: Kiểm thử & Xác minh
- [ ] Viết bổ sung hoặc cập nhật các test case trong `recon_lag_test.go`, `recon_hash_test.go`, `recon_core_test.go`.
- [ ] Chạy toàn bộ test suite của recon để đảm bảo không lỗi regression.
