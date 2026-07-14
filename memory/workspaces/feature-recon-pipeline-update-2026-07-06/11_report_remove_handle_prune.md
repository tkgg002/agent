# Báo cáo Thay đổi & Tái cấu trúc - Loại bỏ handlePrune & Tái cấu trúc Routing Phân đoạn (DRY)

Báo cáo chi tiết về kết quả thực hiện việc loại bỏ logic `handlePrune` dư thừa, cấu trúc lại định tuyến segment (`both`), và áp dụng cơ chế `executeGenericCheck` (DRY).

## 1. Danh sách File Thay đổi & Số lượng Dòng Code
Tổng cộng thay đổi trên **3 files chính**:
1. **[recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)**
   - **Tác vụ**: Tái cấu trúc bộ định tuyến `HandleReconCheck` bằng cách tách biệt trực tiếp luồng A, B và both. Áp dụng generic check wrapper để tái sử dụng logic check.
2. **[recon_engine_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_run.go)**
   - **Tác vụ**: Đổi tên hàm `CheckAll` thành `CheckAllSegmentA` để đảm bảo tính đối xứng và nhất quán với `CheckAllSegmentB`. Đồng thời triển khai hàm `ListActiveRegistries` công khai.
3. **[recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)**
   - **Tác vụ**: Tách hàm `RunSegmentB` thành các hàm tiered service (`RunSmokeCheckB`, `RunHashWindowCheckB`, `RunDeepCheckB`) tương thích với phân đoạn A. Đồng thời triển khai hàm `TimeBoundedDiffMissingFromMaster`.

## 2. Chi tiết các Thay đổi Kỹ thuật

### 2.1. Đồng nhất hóa Tên gọi Check All các Phân đoạn
- Đổi tên `CheckAll` thành `CheckAllSegmentA` để đối xứng và rõ nghĩa.
- Triển khai `ListActiveRegistries(ctx)` công khai để phục vụ định tuyến trong handler.

### 2.2. Áp dụng Generic Wrapper `executeGenericCheck`
- Gom cụm logic switch-case theo `payload.TypeRecon` vào một hàm dùng chung:
  ```go
  func (h *CheckHandler) executeGenericCheck(
      ctx context.Context,
      payload *reconCheckPayload,
      segmentType string,
      targetTable string,
      sourceDB string,
      fnSmoke func(context.Context) *recon.ReconciliationReport,
      fnDeep func(context.Context) *recon.ReconciliationReport,
      fnHash func(context.Context) *recon.ReconciliationReport,
      fnDiff func(context.Context, time.Time, time.Time) ([]string, int, int, error),
  ) *recon.ReconciliationReport
  ```
- Loại bỏ hoàn toàn sự lặp lại của logic so sánh thời gian và kiểm tra tham số của `FullDiff` cho cả Segment A và B.

### 2.3. Triển khai Đối soát Khác biệt Phân đoạn B theo Khoảng Thời gian
- Triển khai `TimeBoundedDiffMissingFromMaster` để đối chiếu hiệu quả các khóa chính của Shadow DB và Master DB trong một khoảng thời gian, tận dụng cấu trúc chỉ mục CDC hiệu suất cao.

## 3. Kết quả Kiểm thử & Đánh giá
- Biên dịch thành công 100% không lỗi cú pháp.
- Chạy unit test gói `service/recon` và `handler/recon` thành công 100% (Green).
