# Kế hoạch thực thi (Phases 5, 6, 7)

## Mục tiêu
Hoàn thành tích hợp tính năng Reconciliation Smoke Test thông qua 3 phase còn lại của kế hoạch:

### Phase 5: Cập nhật Tầng Đọc Kết quả Smoke Test (Backend Service)
- [ ] Định nghĩa các model GORM mới: `SmokeResult` và `CycleSummary` (hoặc tương đương) khớp với cấu trúc bảng `cdc_recon_smoke_result`.
- [ ] Sửa đổi `recon_read_repo_gorm.go` trong `cdc-cms-service`:
  - Cập nhật hàm `ListLatest` và `GetTableHistory` để truy vấn từ bảng `cdc_recon_smoke_result`.
  - Thực hiện mapping chính xác sang API DTO tương thích với frontend.
- [ ] Đảm bảo backend compile và build thành công.

### Phase 6: Nâng cấp Giao diện hiển thị Data Integrity (Frontend Web)
- [ ] Sửa `useReconStatus.ts` trong `cdc-cms-web` để hỗ trợ các trường dữ liệu 3 tầng (Master, Shadow, Dest) mới.
- [ ] Cải tiến component `DataIntegrity.tsx`:
  - Chia giao diện thành 3 cột hoạt động song song rõ ràng.
  - Hiển thị thông tin Lag theo segment (ví dụ: Master vs Shadow, Shadow vs Dest, hoặc tương đương).
- [ ] Kiểm thử build frontend (`npm run build` hoặc tương đương) đảm bảo không có lỗi TypeScript/lint.

### Phase 7: Triển khai Prometheus Metrics dạng O(1) (Worker)
- [ ] Sửa `pkgs/metrics/prometheus.go` để định nghĩa các metric O(1) mới cho smoke test:
  - Ví dụ: `cdc_recon_smoke_latency`, `cdc_recon_smoke_status`, v.v.
- [ ] Sửa logic chạy smoke test trong `recon_smoke.go` để emit các metric này một cách an toàn (tránh memory leak, tránh ghi đè nhãn không hợp lệ).
- [ ] Đảm bảo `cdc-worker` compile và build thành công.

### Verification
- [ ] Biên dịch tất cả dịch vụ (Backend CMS, Web FE, Worker).
- [ ] Khởi chạy và kiểm tra logs khởi động không có lỗi SQL/Panic/Address-in-use.
