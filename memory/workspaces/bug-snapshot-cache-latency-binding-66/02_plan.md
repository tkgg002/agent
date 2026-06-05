# Kế hoạch khắc phục lỗi trễ cache khi Snapshot (Snapshot Cache Latency Resolution Plan)

Tài liệu này phác thảo kế hoạch giải quyết triệt để vấn đề trễ cache (stale cache) bằng việc loại bỏ hoàn toàn cache cục bộ trong `MaskingService` và chuyển `MetadataRegistryService` thành Single Source of Truth cho cấu hình masking.

## 1. User Review Required / Các điểm cần Lưu ý

> [!IMPORTANT]
> **Loại bỏ hoàn toàn cache cục bộ và database query trong `MaskingService`**:
> - Sử dụng `MetadataRegistryService` để quản lý in-memory cả mapping rules và global sensitive fields.
> - Mỗi lần reload registry, dựng lại toàn bộ `maskMapCache` in-memory.
> - `MaskingService` đọc realtime từ in-memory của `MetadataRegistryService`, đảm bảo zero cache latency.

---

## 2. Proposed Changes / Các thay đổi đề xuất

### 2.1. centralized-data-service

#### [NEW] [sensitive_field.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/sensitive_field.go)
- Tạo model `SensitiveField` cho bảng `cdc_system.sensitive_fields`.

#### [MODIFY] [mapping_rule_v2_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/mapping_rule_v2_repo.go)
- Thêm method `ListGlobalSensitiveFields(ctx context.Context) ([]model.SensitiveField, error)`.

#### [MODIFY] [metadata_registry_service.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/metadata_registry_service.go)
- Định nghĩa interface `MetadataRegistry` (hoặc struct) hỗ trợ phương thức `GetMaskMap(bindingID int64) map[string]string`.
- Thêm thuộc tính cache `globalSensitiveFields map[string]string` và `maskMapCache map[int64]map[string]string`.
- Trong `ReloadAll`, truy vấn `ListGlobalSensitiveFields` và build `maskMapCache` cho từng active binding bằng cách gộp mapping_rule_v2 local với global sensitive fields.

#### [MODIFY] [masking_service.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/masking_service.go)
- Bỏ `sensitiveFields sync.Map`.
- Bỏ các logic truy vấn DB trong `resolveMaskMap`.
- Nhận `registrySvc MetadataRegistry` thông qua `SetMetadataRegistry`.
- `resolveMaskMap(bindingID)` gọi trực tiếp sang `registrySvc.GetMaskMap(bindingID)`.

#### [MODIFY] [worker_server.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/worker_server.go)
- Tiêm `registrySvc` vào `maskingSvc` tại luồng khởi tạo server.

---

## 3. Verification Plan / Kế hoạch xác minh

### Automated Tests / Kiểm thử tự động
- Cập nhật các mock hoặc test cases trong `masking_service_test.go` và `metadata_registry_service_test.go`.
- Chạy unit tests:
  ```bash
  go test -v ./internal/service/...
  go test -v ./internal/handler/...
  ```

### Manual Verification / Xác minh thủ công
- Chạy worker bằng `make run`, thay đổi cấu hình masking trên DB, reload cấu hình và chạy snapshot v2 để xác minh dữ liệu được mask tức thời theo cấu hình mới.
