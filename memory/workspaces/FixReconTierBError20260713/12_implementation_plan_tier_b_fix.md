# Kế hoạch triển khai chi tiết (AI Implementation Plan) - Sửa lỗi biên dịch Recon Tier B

## 1. Yêu cầu & Bối cảnh
Do tái cấu trúc hoặc code chưa hoàn thiện, file `internal/service/recon/recon_tier_b.go` gặp các lỗi biên dịch sau:
- Khai báo trùng lặp `stampB` (đã có ở `recon_engine_segment_b.go`).
- Truy cập trường `SourceDB` trên `MasterBindingRef` (struct này không có trường `SourceDB`).
- Truy cập trường `TargetSchema` trên `ReconciliationReport` (struct này không có trường `TargetSchema`).
- Gọi phương thức `rc.RunSegmentB` (chưa được định nghĩa trên `ReconCore`).

## 2. Kế hoạch chi tiết
- **Bước 1:** Xóa hàm `stampB` trùng lặp trong `recon_tier_b.go` (dòng 631-641).
- **Bước 2:** Cập nhật hàm `errorReportB` trong `recon_tier_b.go` (dòng 651) để gán `SourceDB: ""` thay vì `ref.SourceDB`.
- **Bước 3:** Thêm phương thức `RunSegmentB` vào `recon_tier_b.go` ngay trước `RunSegmentBFor`:
  ```go
  func (rc *ReconCore) RunSegmentB(ctx context.Context, ref MasterBindingRef, deep bool) *recon.ReconciliationReport {
  	if deep {
  		return rc.RunDeepCheckB(ctx, ref)
  	}
  	return rc.RunHashWindowCheckB(ctx, ref)
  }
  ```
- **Bước 4:** Biên dịch kiểm tra với `go build ./internal/service/recon/...`.
- **Bước 5:** Chạy quy trình kiểm toán `verify_governance.py`.

## 3. Vai trò
- Brain: Lập kế hoạch, thiết kế và giám sát chất lượng.
- Muscle: Thực thi sửa code, chạy compile và kiểm tra linter.
