# Kế hoạch sửa lỗi biên dịch Recon Tier B

Sửa đổi file `internal/service/recon/recon_tier_b.go` để khắc phục triệt để các lỗi biên dịch liên quan đến khai báo trùng lặp `stampB`, các trường không tồn tại trên `MasterBindingRef` / `ReconciliationReport` và phương thức `RunSegmentB` bị thiếu trên `ReconCore`.

## User Review Required

> [!IMPORTANT]
> Sửa đổi này chỉ sửa lỗi biên dịch, không làm thay đổi logic nghiệp vụ cốt lõi hay schema database hiện tại. Sau khi áp dụng, chương trình sẽ biên dịch thành công.

## Open Questions

Không có câu hỏi mở nào. Yêu cầu sửa lỗi biên dịch rất rõ ràng.

## Proposed Changes

### centralized-data-service

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)

- **Xóa method `stampB` trùng lặp:** Xóa khai báo `func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef)` tại dòng 631-641 vì nó đã được định nghĩa đúng và đầy đủ trong `recon_engine_segment_b.go`.
- **Sửa lỗi `errorReportB`:** Trong hàm `errorReportB` (dòng 643-657), cập nhật việc gán trường `SourceDB` từ `ref.SourceDB` thành `""` do `MasterBindingRef` không có trường này (Segment B là đối soát giữa Shadow và Master, không có Source DB trực tiếp).
- **Thêm định nghĩa `RunSegmentB`:** Định nghĩa thêm phương thức `RunSegmentB` trên `ReconCore` để định tuyến giữa `RunDeepCheckB` và `RunHashWindowCheckB` dựa trên tham số `deep bool`, giải quyết lỗi thiếu phương thức khi gọi ở `RunSegmentBFor` và `CheckAllSegmentB`.

## Verification Plan

### Automated Tests
- Chạy biên dịch dự án:
  `go build ./internal/service/recon/...`
- Chạy linter kiểm toán quy trình:
  `python3 agent/tooling/verify_governance.py`
