# 08_tasks.md - Checklist triển khai Transmute Shadow sang Master với Custom Master DB Connection

## Phase 1: CDS Engine (centralized-data-service)
- [x] Task 1.1: Bổ sung `masterDBPool map[string]*gorm.DB` và `masterPoolMu sync.RWMutex` vào struct `ConnectionManager` (`internal/service/source/connection_manager.go`).
- [x] Task 1.2: Triển khai phương thức resolve target DB connection trong `ConnectionManager.GetMasterDB(ctx, key)`: tra cứu override DSN hoặc query `cdc_system.connection_registry`, khởi tạo GORM connection pool theo standard settings (MaxOpenConns, MaxIdleConns), cache vào `masterDBPool`.
- [x] Task 1.3: Thêm DDL Safety Gate vào `TransmuterModule.Run()` (`internal/service/master/transmuter.go`): kiểm tra sự tồn tại của bảng master trên target DB (`information_schema.tables`) trước khi thực hiện batch upsert. Trả về lỗi rõ ràng nếu bảng chưa có DDL.
- [x] Task 1.4: Cập nhật unit test `test/internal/service/connection_manager_test.go` và chạy test xác nhận pass.

## Phase 2: CMS Backend (cdc-cms-service)
- [x] Task 2.1: Cập nhật `CreateMasterHandler` (`internal/app/commands/master/create_master.go`): lấy `cmd.MasterConnectionCode`, resolve sang connection ID từ `cdc_system.connection_registry` thay vì hardcode default.
- [x] Task 2.2: Bổ sung endpoint API `GET /api/v1/master-connections` trong CMS Service để UI lấy danh sách các active master connections. Đăng ký route trong `internal/router/router.go`.
- [x] Task 2.3: Build & verify API backend không có compiler/linter error.

## Phase 3: CMS Web Frontend (cdc-cms-web)
- [x] Task 3.1: Cập nhật `MasterRegistry.tsx`: Thêm dropdown chọn Master Database Connection khi tạo Master Table mới. Hiển thị Target Connection trên bảng danh sách.
- [x] Task 3.2: Kiểm tra nút Transmute tại `MasterRegistry.tsx`: Đảm bảo chỉ active khi Master Table có `schema_status === 'approved'`.
- [x] Task 3.3: Kiểm tra `TableRegistry.tsx`: Giữ nguyên tính toàn vẹn (không đặt nút Transmute mù quáng trên hàng Shadow table chính; chỉ hiển thị nút Transmute ở sub-table Master Binding con khi binding đó đã Approved).
- [x] Task 3.4: Build frontend (`npm run build`) để kiểm tra compile errors.

## Phase 4: Verification & Governance
- [x] Task 4.1: Chạy test suites Go trên cả 2 backend services.
- [x] Task 4.2: Security review code changes (Rule #3, #15).
- [x] Task 4.3: Append audit log vào `05_progress.md` và báo cáo hoàn thành.
