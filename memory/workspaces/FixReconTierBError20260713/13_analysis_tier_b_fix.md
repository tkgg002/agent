# Phân tích lỗi biên dịch (Root Cause Analysis) - Recon Tier B

## 1. Các lỗi biên dịch phát hiện
- **Lỗi 1 (Khai báo trùng lặp):** `ReconCore.stampB` đã được khai báo ở `recon_engine_segment_b.go:44:22`. Nguyên nhân là do cấu trúc file đối soát phân mảnh (giữa `recon_tier_b.go` và `recon_engine_segment_b.go`), dẫn đến khai báo hàm `stampB` ở cả hai file.
- **Lỗi 2 (Trường undefined trên struct):** `ref.SourceDB` undefined trên `MasterBindingRef` (struct này không có trường `SourceDB`). Thực chất Segment B đối soát Shadow vs Master nên không chứa database nguồn.
- **Lỗi 3 (Trường undefined trên struct):** `report.TargetSchema` undefined trên `ReconciliationReport` (struct này không có trường `TargetSchema`). Lỗi này do drift trong định nghĩa struct model.
- **Lỗi 4 (Phương thức undefined trên ReconCore):** `rc.RunSegmentB` undefined trên `ReconCore`. Phương thức này được gọi nhưng chưa được khai báo ở bất cứ đâu.

## 2. Giải pháp khắc phục
- Loại bỏ hoàn toàn khai báo `stampB` trong `recon_tier_b.go` để dùng định nghĩa ở `recon_engine_segment_b.go`.
- Gán `SourceDB: ""` trong `errorReportB` thay vì `ref.SourceDB`.
- Bổ sung định nghĩa cho phương thức `RunSegmentB` để định tuyến gọi `RunDeepCheckB` hoặc `RunHashWindowCheckB`.
