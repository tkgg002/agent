# 12_implementation_plan_custom_master_db.md

## Kế hoạch triển khai chi tiết của Muscle (Chief Engineer)

### 1. Phân bổ công việc kỹ thuật
- **Phase 1: CDS Engine (`centralized-data-service`)**
  - Mở rộng `ConnectionManager` hỗ trợ `masterDBPool map[string]*gorm.DB` và `masterPoolMu sync.RWMutex`.
  - Cài đặt hàm `GetMasterDB(ctx, key)`: tra cứu `cfg.ConnectionOverrides` -> truy vấn `cdc_system.connection_registry` -> fallback destination pool -> mở GORM pool chuẩn và lưu cache.
  - Thêm DDL Safety Gate check trước khi bulk upsert trong `TransmuterModule.Run()`.
  - Cập nhật unit test trong `test/internal/service/connection_manager_test.go`.

- **Phase 2: CMS Backend (`cdc-cms-service`)**
  - Mở rộng `CreateMasterCommand` và `CreateMasterHandler` để ưu tiên `cmd.MasterConnectionCode`, resolve sang connection ID từ `cdc_system.connection_registry`.
  - Mở rộng `MasterRepo` và `masterRepoGorm` với hàm `ListMasterConnections`.
  - Thêm handler `GET /api/v1/master-connections` trong `MasterRegistryHandler` và đăng ký trong `internal/router/router.go`.

- **Phase 3: CMS Web Frontend (`cdc-cms-web`)**
  - Tích hợp `useQuery` gọi `GET /api/v1/master-connections`.
  - Bổ sung dropdown "Target Database Connection" vào Modal tạo Master table mới.
  - Hiển thị thông tin Target Connection trên bảng danh sách Master (`DB Master` column).
  - Ép điều kiện kiểm tra DDL: Nút `Sync` và `Chạy ngay` chỉ active khi `schema_status === 'approved'`.
  - Kiểm tra tính phân lập: Không đặt nút Transmute trên hàng Shadow table ở `TableRegistry.tsx`.

- **Phase 4: Governance & Audit**
  - Cập nhật `05_progress.md` (append-only) theo đúng timestamp và format chuẩn.
  - Hoàn tất toàn bộ checklist trong `08_tasks.md`.
  - Xuất báo cáo thay đổi trong `11_report_transmute_custom_master_db.md`.
