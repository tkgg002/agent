# TODO: Refactoring and Relocating Handlers

- [x] Lập kế hoạch và thu thập phản hồi từ User về cấu trúc phân rã domain.
- [x] Di chuyển các file handler misplaced sang thư mục đích (`source`, `shadow`, `recon`, `orchestration`).
- [x] Tách file `provisioning_step_handlers.go` thành hai file riêng biệt đặt tại `shadow/` và `orchestration/`.
- [x] Di chuyển các hàm helper định dạng và ép kiểu SQL (`NormalizeMappingRuleDataType`, `BuildCastExpr`) sang `shadow/mapping_utils.go`.
- [x] Tái cấu trúc `base.BaseHandler` để loại bỏ metadata dependencies.
- [x] Di chuyển `ResolveTargetSchema` sang package `metadata` dưới dạng package utility.
- [x] Khai báo explicitly dependency `metadataRegistry` trên các struct handler cần thiết và thêm setter.
- [x] Cập nhật toàn bộ các import và khởi tạo DI trong `worker_server_init.go`.
- [x] Chạy build biên dịch (`go build ./...`) và chạy test suite (`go test ./...`) để xác minh hệ thống hoạt động ổn định.
- [x] Cập nhật tiến độ `05_progress.md` và đóng task.
