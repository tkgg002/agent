# Kế hoạch triển khai: Loại bỏ thuật ngữ Tier khỏi log (12_implementation_plan_term_sync)

Nhiệm vụ này nhằm chuẩn hóa toàn bộ các câu log trong `recon_tier_a.go` và `recon_tier_b.go`, chuyển các nhãn `tierA`/`tierB` cũ thành đúng 1 format thống nhất: `[<typerecon>-<segment>] <message>`.

## Proposed Changes

### Component: `recon`

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
* Sửa đổi các hàm:
  - `resolveSourceAndDestTSFields(ctx, entry, checkType)`: Nhận thêm tham số `checkType`. Log sử dụng `fmt.Sprintf("[%s-A] ts_fields resolved", checkType)`.
  - `TimeBoundedDiffMissingFromShadow(ctx, entry, startTime, endTime, checkType)`: Nhận thêm tham số `checkType`, truyền tiếp vào `resolveSourceAndDestTSFields`, và log lỗi time params sử dụng `fmt.Sprintf("[%s-A] failed to resolve postgres time params...", checkType)`.
  - `RunTotalOnlyA`: Truyền `"smoke"` vào `resolveSourceAndDestTSFields`.
  - `RunHashWindowCheck`: 
    * Thay `tierA skipped — previous run ongoing` thành `[hash_window-A] skipped — previous run ongoing`.
    * Thay `[tierA] scan range resolved` thành `[hash_window-A] scan range resolved`.
    * Thay `[tierA] global hash match — no drift detected in range` thành `[hash_window-A] global hash match — no drift detected in range`.
    * Thay `[tierA] global blocks hash match — no drift detected in range` thành `[hash_window-A] global blocks hash match — no drift detected in range`.
    * Thay `tierA hash_window` (tổng kết) thành `[hash_window-A] check completed`.
  - `RunDeepCheck`: Truyền `"bucket_hash"` vào `resolveSourceAndDestTSFields`. Thay `tierA skipped — outside off-peak window` thành `[bucket_hash-A] skipped — outside off-peak window`. Thay `tierA budget exceeded` thành `[bucket_hash-A] budget exceeded`.
  - `RunBucketHash`: Thay `tierA bucket_hash` (tổng kết) thành `[bucket_hash-A] check completed`.

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
* Sửa đổi các hàm:
  - `measureAndResolveWatermarksB(ctx, ref, checkType)`: Nhận thêm tham số `checkType`, truyền tiếp vào `resolveSourceAndDestTSFields`.
  - `RunHashWindowCheckB`: Thay `[tierB] global hash match — no drift` thành `[hash_window-B] global hash match — no drift`. Thay `[tierB] global blocks hash match — no drift` thành `[hash_window-B] global blocks hash match — no drift`. Truyền `"hash_window"` vào `measureAndResolveWatermarksB`.
  - `RunDeepCheckB`: Truyền `"bucket_hash"` vào `measureAndResolveWatermarksB`.
  - `errorReportB`: Thay `tierB error` thành `fmt.Sprintf("[%s-B] error", checkType)`.
* Cập nhật các comment chứa `Tier B` hoặc `Tier A` thành `Segment B` hoặc `Segment A`.

---

## Verification Plan

### Automated Tests
* Chạy unit tests để verify compilation và tính đúng đắn:
  ```bash
  go test -v ./internal/service/recon/...
  ```
