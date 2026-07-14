# Kế hoạch Triển khai Kỹ thuật - Khắc phục Hồi quy Đối soát Smoke

Kế hoạch này chi tiết hóa phương án sửa lỗi kiểm thử và tối ưu/tái kích hoạt lookback check cho đối soát smoke.

## Phân tích & Giải pháp Chi tiết

### 1. Sửa đổi `test/internal/service/metadata_mapping_test.go`
Lớp kiểm thử `TestBuildCastExpr` đang kiểm tra xem biểu thức sinh ra bởi `BuildCastExpr` cho các kiểu dữ liệu có chứa chuỗi băm cứng nhất định hay không. Tuy nhiên, logic của `BuildCastExpr` đã được thay đổi để bọc giá trị bằng `NULLIF(val, '')` và chuyển đổi sang `NUMERIC` trước khi chuyển sang `INTEGER`/`BOOLEAN`.
- **Giải pháp:** Sửa đổi mảng `contains` trong struct test case để kiểm tra sự tồn tại của hai phần độc lập: cột dữ liệu thô (`_raw_data->>'my_field'`) và kiểu dữ liệu đích (`::INTEGER` hoặc `::BOOLEAN`).

### 2. Sửa đổi `test/internal/service/recon_hash_test.go`
Hàm băm `hashIDPlusTsMs` thực hiện làm tròn timestamp về hàng giây (`epoch_ms / 1000 * 1000`) nhằm đảm bảo tính toàn vẹn dữ liệu khi so sánh giữa các nguồn dữ liệu khác nhau. Vì vậy, sự sai lệch `1ms` (như trong `TestHashWindowDriftDetection`) sẽ bị triệt tiêu sau khi làm tròn, làm cho giá trị băm của nguồn và đích giống nhau.
- **Giải pháp:** Tăng độ lệch thời gian trong test từ `1ms` lên `1000ms` (1 giây) để đảm bảo sau khi làm tròn, giá trị băm sẽ khác nhau và kích hoạt tính năng phát hiện sai lệch.

### 3. Tái cấu trúc & Kích hoạt lookback check (`internal/service/recon/recon_smoke.go`)
- **Tối ưu hóa `runLookbackCheckA`:**
  - Nhận trực tiếp `lo, hi time.Time, srcTS, dstTS string` làm đối số truyền vào.
  - Loại bỏ lệnh gọi `pickScanRangeWithLag(ctx, entry)`.
- **RunTotalOnlyA:**
  - Lấy đầy đủ kết quả từ `pickScanRangeWithLag`:
    `lo, hi, ingestLagMs, srcTS, dstTS, err := rc.pickScanRangeWithLag(ctx, entry)`
  - Uncomment và truyền tham số:
    ```go
    driftTimes := rc.runLookbackCheckA(fastCtx, entry, lo, hi, srcTS, dstTS)
    if len(driftTimes) > 0 {
        diffTimeJSON, _ = json.Marshal(driftTimes)
    }
    ```
- **RunTotalOnlyB:**
  - Uncomment và gọi:
    ```go
    driftTimes := rc.runLookbackCheckB(fastCtx, ref)
    if len(driftTimes) > 0 {
        diffTimeJSON, _ = json.Marshal(driftTimes)
    }
    ```
