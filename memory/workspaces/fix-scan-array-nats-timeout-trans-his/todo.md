# Workspace Todo: fix-scan-array-nats-timeout-trans-his

- [x] Điều tra root cause và trace mã nguồn `cdc-cms-service` và `centralized-data-service`.
- [x] Lập hồ sơ giải pháp kỹ thuật `09_tasks_solution_scan_array_timeout.md`.
- [x] Lập kế hoạch triển khai `12_implementation_plan_scan_array_timeout.md`.
- [x] Lập báo cáo phân tích `13_analysis_scan_array_timeout.md`.
- [ ] Báo cáo User và nhận `APPROVE` trước khi sửa source code (tuân thủ Rule #13 Brain Code Prohibition).
- [ ] Ủy quyền Muscle thực thi sửa code:
  - Cập nhật `ScanArrayFields` trong `centralized-data-service/internal/service/source/scan_service.go` với trần an toàn `LIMIT 2000` và DB context timeout 10s.
  - Cập nhật `ScanArray` và `ScanArrayAll` trong `cdc-cms-service/internal/api/system/introspection_handler.go` với mode `"sample"`.
- [ ] Kiểm thử và xác thực: Unit tests, build verification.
- [ ] Hướng dẫn kiểm tra trạng thái pod/service `cdc-worker` trên cụm server k8s.
