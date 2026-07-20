# Kế hoạch chi tiết (08_tasks_term_sync)

- [x] Sửa `recon_tier_a.go`:
  - [x] Thêm tham số `checkType` vào `resolveSourceAndDestTSFields` và format log.
  - [x] Thêm tham số `checkType` vào `TimeBoundedDiffMissingFromShadow` và format log.
  - [x] Cập nhật các nơi gọi `resolveSourceAndDestTSFields` trong `recon_tier_a.go`.
  - [x] Sửa đổi nhãn log `tierA`/`[tierA]` của `RunHashWindowCheck` và `RunDeepCheck`/`RunBucketHash`.
- [x] Sửa `recon_tier_b.go`:
  - [x] Thêm tham số `checkType` vào `measureAndResolveWatermarksB` và truyền tiếp vào `resolveSourceAndDestTSFields`.
  - [x] Cập nhật các nơi gọi `measureAndResolveWatermarksB`.
  - [x] Sửa đổi nhãn log `[tierB]` thành `[hash_window-B]` trong `RunHashWindowCheckB`.
  - [x] Sửa đổi nhãn log `tierB error` thành `[<checkType>-B] error` trong `errorReportB`.
  - [x] Cập nhật các comment chứa từ khóa `Tier A` / `Tier B`.
- [x] Chạy kiểm thử tự động để đảm bảo tính đúng đắn và không có regression.
