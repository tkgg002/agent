# Kế hoạch triển khai - Range Counts

Chúng ta sẽ tối ưu hóa `recon_tier_b.go` bằng cách loại bỏ hoàn toàn các truy vấn đếm dòng toàn bảng (`TotalSourceCount` / `TotalDestCount`) và chỉnh sửa `SourceCount` / `DestCount` để phản ánh đúng số lượng bản ghi trong dải thời gian quét của Segment B.

## Proposed Changes

### centralized-data-service

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)

- Xóa bỏ định nghĩa và các lời gọi hàm `runCountCheckB`.
- Xóa bỏ logic tối ưu hóa `if count match && lag == 0` ở đầu cả 2 hàm check.
- Điều chỉnh việc gán các trường `SourceCount`, `DestCount`, `Diff` trong `ReconciliationReport` ở cả `RunHashWindowCheckB` và `RunDeepCheckB` để lấy dữ liệu từ `totalShadow` và `totalMaster`.
- Loại bỏ các trường `TotalSourceCount` và `TotalDestCount` khỏi báo cáo.

## Verification Plan

### Automated Tests
- Biên dịch lại dự án: `go build ./internal/service/recon/...`
- Chạy linter quy trình: `python3 agent/tooling/verify_governance.py`
