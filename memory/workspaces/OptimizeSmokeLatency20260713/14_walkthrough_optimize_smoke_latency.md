# Walkthrough - Tối ưu hóa Latency phát hiện Drift trong Smoke Check

Công việc tối ưu hóa độ trễ 11 giây khi xảy ra drift trong luồng `Smoke Check` đã được triển khai và xác minh thành công.

## Tóm tắt các thay đổi đã thực hiện

### 1. Sửa đổi cấu hình lookback động (`effectiveLookback`)
*   **Tệp tin:** [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
*   **Chi tiết:** Sửa lỗi logic trong `effectiveLookback` để trả về cửa sổ thời gian lookback quét chính xác theo `RunMode` (Hot mode: mặc định 2 giờ thông qua `HotWindowLookback`, Cold mode: mặc định 7 ngày thông qua `WindowLookback`).

### 2. Tối ưu hóa Segment B lookback check (`runLookbackCheckB`)
*   **Tệp tin:** [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
*   **Chi tiết:**
    *   Thay thế giá trị hardcode 7 ngày bằng `effectiveLookback(ctx)` động.
    *   Tự động phân giải và truy xuất trường timestamp nghiệp vụ từ registry thay vì hardcode `_source_ts`.
    *   Chuyển đổi các cuộc gọi `BucketCounts` trên Shadow và Master DB sang chạy song song bằng `sync.WaitGroup` để triệt tiêu thời gian RTT tuần tự.

### 3. Song song hóa Segment A lookback check (`runLookbackCheckA`)
*   **Tệp tin:** [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
*   **Chi tiết:** Chuyển đổi các cuộc gọi `BucketCounts` trên Source (MongoDB/Postgres) và Shadow DB sang chạy song song bằng `sync.WaitGroup`.

### 4. Bổ dung Unit Tests
*   **Tệp tin:** [recon_smoke_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke_test.go)
*   **Chi tiết:** Viết mới test suite `TestReconCore_EffectiveLookback` để kiểm thử hành vi của hàm `effectiveLookback` trong cả Hot mode, Cold mode và các tham số tùy chỉnh khác.

---

## Kết quả Kiểm thử & Đánh giá

### 1. Kết quả Unit Test
Tất cả các bài test trong package `internal/service/recon` đều vượt qua thành công:
```bash
go test -v ./internal/service/recon/...
```
*   `TestReconCore_EffectiveLookback` (5 sub-tests): **PASS**
*   Tổng số test suite trong package `recon`: **PASS**

### 2. Kết quả Process Linter
Script kiểm soát quy trình `verify_governance.py` chạy thành công 100%:
*   **Trạng thái:** `GOVERNANCE AUDIT PASSED 🟢`
