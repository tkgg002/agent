# Implementation: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## 1. File sửa đổi (Modified Files)
- `internal/handler/recon/recon_heal_v4.go`

## 2. File thêm mới (New Files)
- `internal/handler/recon/recon_heal_v4_test.go`

## 3. Chi tiết thực hiện

### 3.1. Định nghĩa hằng số Max Age
```go
const (
    // ...
    healReportMaxAge = 5 * time.Minute
)
```

### 3.2. Cập nhật logic `healSegmentB`
- Kiểm tra báo cáo mới nhất:
```go
reportPtr, err := h.reportRepo.GetLatestByTable(ctx, table, "shadow_master")
isStale := reportPtr != nil && time.Since(reportPtr.CheckedAt) > healReportMaxAge
if err != nil || reportPtr == nil || reportPtr.HealedAt != nil || isStale {
    // Nếu stale, log thông tin và chạy lại:
    newReport := h.reconCore.RunSegmentBFor(ctx, table, true)
    // ...
}
```

### 3.3. Cập nhật logic `healSegmentA`
- Xác định FQN table và tìm report:
```go
targetFQN := entry.QualifiedTarget()
reportPtr, err := h.reportRepo.GetLatestByTable(ctx, targetFQN, "source_shadow")
isStale := reportPtr != nil && time.Since(reportPtr.CheckedAt) > healReportMaxAge
if err == gorm.ErrRecordNotFound || reportPtr == nil || reportPtr.HealedAt != nil || isStale {
    newReport := h.reconCore.RunTier2(ctx, *entry)
    // ...
}
```

### 3.4. Triển khai Unit Test
- Khởi tạo mock NATS Server nhúng và sqlmock cho GORM.
- Mock các query tìm table registry config và report tương ứng.
- Đảm bảo trả về các cột `checked_at` hợp lệ để tránh lỗi zero-time trong GORM.
- Verify rằng khi có stale report hoặc không có report, hệ thống tự động gọi hàm đối soát sâu để xác minh trạng thái dữ liệu tươi mới.
