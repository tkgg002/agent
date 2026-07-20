# Walkthrough (14_walkthrough_term_sync)

## Thay đổi đã thực hiện

### 1. File `internal/service/recon/recon_tier_a.go`
- Cập nhật hàm `resolveSourceAndDestTSFields` để nhận `checkType string` làm tham số động phục vụ format prefix log.
- Cập nhật hàm `TimeBoundedDiffMissingFromShadow` để nhận `checkType string` làm tham số động.
- Cập nhật `pickScanRange` và `pickScanRangeWithLag` để nhận `checkType string`.
- Loại bỏ hoàn toàn các tiền tố log tĩnh `tierA`/`[tierA]` và thay thế bằng định dạng động `[<checkType>-A]` (ví dụ: `[hash_window-A]`, `[bucket_hash-A]`).

### 2. File `internal/service/recon/recon_tier_b.go`
- Cập nhật hàm `measureAndResolveWatermarksB` để nhận `checkType string` làm tham số và truyền tiếp đến `resolveSourceAndDestTSFields`.
- Cập nhật hàm `TimeBoundedDiffMissingFromMaster` để nhận `checkType string` làm tham số.
- Thay thế các tiền tố log tĩnh `[tierB]` thành `[hash_window-B]` trong `RunHashWindowCheckB`.
- Thay thế log lỗi `tierB error` thành `[<checkType>-B] error` trong hàm `errorReportB`.
- Dọn dẹp các comment mang tính chất legacy chứa thuật ngữ `Tier A`, `Tier B` thành `Segment A`, `Segment B`.

### 3. File `internal/service/recon/recon_smoke.go`
- Cập nhật các cuộc gọi `resolveSourceAndDestTSFields` và `pickScanRangeWithLag` để truyền tham số `"smoke"`.

### 4. File `internal/handler/recon/recon_check_handler.go`
- Cập nhật cuộc gọi tới `TimeBoundedDiffMissingFromShadow` và `TimeBoundedDiffMissingFromMaster` truyền động `payload.TypeRecon`.

### 5. File `internal/service/recon/recon_fallback_test.go`
- Cập nhật cuộc gọi tới `resolveSourceAndDestTSFields` truyền tham số `"test"`.

## Kết quả kiểm thử (Verification)

Đã chạy thành công toàn bộ test suite của package `recon`:
```bash
go test -v ./internal/service/recon/...
```
Kết quả:
- **PASS**: Tất cả unit tests và integration tests đều hoạt động chính xác với thiết kế signature mới, không có lỗi regression.
