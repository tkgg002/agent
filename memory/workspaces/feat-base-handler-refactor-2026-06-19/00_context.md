# Workspace Context: Refactor Common Handler to Base Handler

## Objective
Tái cấu trúc file `internal/handler/common/common.go` thành `internal/base/base_handler.go` (package `base`), gỡ bỏ Tight Coupling qua interface `RegistryResolver`, bổ sung context cho các hàm GORM để phục vụ Graceful Shutdown, và khắc phục tình trạng spam request trong cơ chế retry của Kafka Connect.

## Scope
- Tạo package mới `internal/base` và file `base_handler.go` với mã nguồn mới do User cung cấp.
- Gỡ bỏ file cũ `internal/handler/common/common.go`.
- Cập nhật toàn bộ các file import và references từ `centralized-data-service/internal/handler/common` sang `centralized-data-service/internal/base`.
- Cập nhật cách truyền dependency `RegistryRepo` dưới dạng interface `RegistryResolver` cho `BaseHandler`.
- Thay đổi tham số truyền context (`ctx context.Context`) vào các hàm `PublishResult`, `WriteActivity`, `TableExists`, `TableExistsInSchema`, `HasColumn`, `HasColumnInSchema`.
- Chạy biên dịch dự án (`go build ./...`) và chạy toàn bộ unit tests để xác minh tính đúng đắn.

## Governance Compliance
- Trạng thái vi phạm: Không vi phạm.
- Gốc rễ lỗi vi phạm quy trình Governance trước đó: Không có (N/A).
