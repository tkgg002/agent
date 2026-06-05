# Tasks: Resolving Snapshot Masking Cache Latency

## Task: Move Masking Cache to Registry & Remove Caching in MaskingService
- **Phase**: GĐ4 (Observability) / GĐ2 (Safety Net)
- **Service Group**: Utilities
- **Service(s)**: centralized-data-service
- **Mô tả**: Bỏ hoàn toàn cache cục bộ (`sync.Map`) và các truy vấn DB của `MaskingService`, chuyển `MetadataRegistryService` thành Single Source of Truth cho cấu hình masking.
- **Trạng thái**: [ ] IN_PROGRESS

### [Context]
- **Current state**: Đã hoàn thành plan thiết kế mới và sẵn sàng thực thi.
- **Dependencies**: centralized-data-service (worker)
- **Logs/Error**: Người dùng yêu cầu "bỏ cha cái cahce vâosw vẩn đó đi ko đc à".

### [Definition of Done]
- [ ] 1. Tạo model `SensitiveField` cho bảng `cdc_system.sensitive_fields`.
- [ ] 2. Thêm method `ListGlobalSensitiveFields(ctx context.Context) ([]model.SensitiveField, error)` vào `MappingRuleV2Repo`.
- [ ] 3. Trong `MetadataRegistryService`, thêm thuộc tính cache `globalSensitiveFields` và `maskMapCache map[int64]map[string]string`.
- [ ] 4. Trong `ReloadAll`, load global sensitive fields và build in-memory `maskMapCache` cho từng shadow binding.
- [ ] 5. Cung cấp API `GetMaskMap(bindingID int64) map[string]string` trong `MetadataRegistryService`.
- [ ] 6. Thêm interface `MetadataRegistry` (hoặc method) tương ứng và tiêm vào `MaskingService`.
- [ ] 7. Xóa `sensitiveFields sync.Map` và phương thức `Invalidate` khỏi `MaskingService`.
- [ ] 8. Sửa đổi `MaskingService.resolveMaskMap` để đọc trực tiếp từ `registrySvc.GetMaskMap(bindingID)`.
- [ ] 9. Tiêm `registrySvc` vào `maskingSvc` trong `worker_server.go`.
- [ ] **[QA Gate]**: Toàn bộ unit tests pass: `go test -v ./internal/service/...` và `go test -v ./internal/handler/...`.
- [ ] **[Security Gate]**: Không có vi phạm bảo mật dữ liệu, cơ chế default/fallback của masking vẫn được đảm bảo khi bindingID <= 0.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md`.
