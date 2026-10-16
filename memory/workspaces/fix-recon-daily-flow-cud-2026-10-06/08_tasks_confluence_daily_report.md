# Danh Mục Task Triển Khai: Confluence Multi-Target Daily Report

- **Parent Target 1:** `Core Trans Proxy History Service` (ID: `463833042`)
- **Space Key:** `~712020a3dcebcbe93646fea463f119dd38b08c`

---

## Danh Sách Task Triển Khai (Checklist):

### Phase 1: Database & Model Layer
- [ ] **TASK-CF-01:** Tạo migration DDL `migrations/schema/recon_dlq/105_create_confluence_report_configs.sql` tạo bảng `cdc_system.confluence_report_configs` kèm index và seed data mẫu cho `Core Trans Proxy History Service`.
- [ ] **TASK-CF-02:** Tạo Go Model `internal/model/recon/confluence_config.go` định nghĩa struct `ConfluenceReportConfig` và `PipelineTarget`.
- [ ] **TASK-CF-03:** Tạo GORM Repository `internal/infra/persistence/recon/confluence_config_repo_gorm.go` hỗ trợ `FindAllEnabled`, `FindByName`, `UpdateStatus`.

### Phase 2: External Client & Storage Format Builder
- [ ] **TASK-CF-04:** Tạo Confluence REST API Client `internal/infra/external/confluence/client.go` hỗ trợ `FindChildPage`, `CreateChildPage`, `UpdatePage`.
- [ ] **TASK-CF-05:** Tạo XHTML Storage Format Builder `internal/app/commands/recon/confluence_report_builder.go` hỗ trợ lọc theo whitelist `pipelines` và render HTML bảng CUD + Đối soát.

### Phase 3: Service Layer, HTTP API & Scheduler
- [ ] **TASK-CF-06:** Tạo `ConfluenceReportService` trong `internal/app/commands/recon/confluence_report_service.go` điều phối việc lấy số liệu từ `DailyFlowQuery`, lọc theo pipeline, gọi client và cập nhật trạng thái `last_run_at`, `last_page_url`.
- [ ] **TASK-CF-07:** Tạo HTTP Handler `internal/api/recon/confluence_report_handler.go` và đăng ký route `POST /api/reconciliation/confluence/publish` trong `internal/router/router.go`.
- [ ] **TASK-CF-08:** Đăng ký cron job tự động chạy lúc `06:00:00` hàng ngày để chốt số liệu D-1.

### Phase 4: Verification & Quality Gate
- [ ] **TASK-CF-09:** Chạy migration trên Postgres docker `:5433`.
- [ ] **TASK-CF-10:** Chạy `go build ./cmd/...` và `go test ./...` đảm bảo không có lỗi biên dịch.
- [ ] **TASK-CF-11:** Test publish bài viết mẫu qua API và kiểm tra kết quả trên Confluence.
- [ ] **TASK-CF-12:** Cập nhật tài liệu Workspace và đóng Task.
