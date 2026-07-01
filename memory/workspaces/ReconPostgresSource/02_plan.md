# Plan: ReconPostgresSource (Fallback Probing Edition)

## Proposed Architectural Changes

### 1. Dò tìm và Phân giải Cột Timestamp hoạt động
Để giải quyết triệt để lỗi cột timestamp mặc định (như `updated_at`) không tồn tại trên PostgreSQL nguồn, ta định nghĩa cơ chế dò tìm cột:
- Thêm helper check lỗi cột không tồn tại trên DB:
  ```go
  func isColumnNotExistError(err error) bool {
      if err == nil {
          return false
      }
      msg := err.Error()
      return strings.Contains(msg, "SQLSTATE 42703") ||
          (strings.Contains(strings.ToLower(msg), "column") && strings.Contains(strings.ToLower(msg), "does not exist"))
  }
  ```
- Tạo hàm `resolveSourceTSField(ctx, entry)` trên `ReconCore` để dò tìm cột timestamp hợp lệ bằng cách truy vấn thử `MaxWindowTs`. Nếu cột đầu tiên lỗi `SQLSTATE 42703`, hệ thống tự động duyệt qua các candidates trong `entry.GetCandidates()` để tìm cột hoạt động.

### 2. Tích hợp cột đã phân giải vào luồng chạy đối soát (Tier 1, Tier 2, Tier 3)
- Thay đổi chữ ký của `pickScanRangeWithLag` để trả về thêm `resolvedTS string`.
- Cập nhật `RunTier1`: Lấy `resolvedTS` từ `pickScanRangeWithLag`, tiếp tục bỏ qua (`continue`) trong loop nếu `BucketCounts` gặp lỗi cột không tồn tại, để tự động thử candidate khác.
- Cập nhật `RunTier2` và `RunTier3`: Sử dụng `resolvedTS` cho các truy vấn `HashWindow`, `ListIDsInWindow` và `BucketHash`.

## Kế hoạch kiểm thử (Verification Plan)
- Viết unit test mô phỏng lỗi `SQLSTATE 42703` bằng `DATA-DOG/go-sqlmock` để đảm bảo cơ chế dò tìm hoạt động đúng.
- Chạy `go test -v ./internal/service/recon/...` để kiểm tra.

## Cập nhật v1.1: Tránh sử dụng Estimated Count cho PostgreSQL Nguồn
- **Vấn đề**: Trong `RunTier1`, `EstimatedCount` của PostgreSQL sử dụng dữ liệu từ `pg_class.reltuples`, vốn là ước tính không đồng bộ trực tiếp (cần ANALYZE). Điều này dẫn đến sự lệch pha nhẹ khi đối soát thời gian thực (ví dụ: source trả về 465 thay vì 467 thực tế), gây ra drift cảnh báo giả.
- **Giải pháp**: Nếu `isPostgres(entry.SourceURL)` là true, ta sử dụng hàm `CountDocuments` (thực hiện truy vấn SQL `COUNT(*)`) để đếm chính xác số lượng bản ghi của PostgreSQL nguồn.
- **Thay đổi chi tiết**:
  - Tại `internal/service/recon/recon_tier_a.go` (`RunTier1`), thay đổi đoạn gọi `EstimatedCount` thành câu lệnh rẽ nhánh rành mạch:
    ```go
    var srcEst int64
    var errE error
    if isPostgres(entry.SourceURL) {
        srcEst, errE = rc.sourceAgent.CountDocuments(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
    } else {
        srcEst, errE = rc.sourceAgent.EstimatedCount(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
    }
    ```

