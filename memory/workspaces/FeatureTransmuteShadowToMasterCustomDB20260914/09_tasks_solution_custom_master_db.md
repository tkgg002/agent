# 09_tasks_solution_custom_master_db.md - Hồ sơ giải pháp kỹ thuật

## 1. Kiến trúc tổng thể & Luồng dữ liệu

```
[UI: MasterRegistry.tsx]
      │
      ├─ Chọn Target Connection (code: "pg-analytics", "pg-reporting", v.v.)
      │  (Dữ liệu lấy từ API GET /api/v1/master-connections)
      │
      ▼
[CMS Backend: CreateMasterHandler]
      │
      ├─ Nhận cmd.MasterConnectionCode
      ├─ Truy vấn cdc_system.connection_registry WHERE code = ? AND is_active = true
      ├─ Lưu master_connection_id vào cdc_system.master_binding
      │
      ▼
[CDS Engine: Master DDL & Transmuter]
      │
      ├─ 1. DDL Flow (Master Create/Approve):
      │     - DDL Generator sinh schema và index.
      │     - Executor kết nối đến Master DB được chỉ định để chạy DDL.
      │
      ├─ 2. Transmuter Flow:
      │     - Transmuter lấy Target DB qua ConnectionManager.GetMasterDB(ctx, key)
      │     - Dynamic Resolve: Tra cứu pool cache masterDBPool[key].
      │       Nếu chưa có, query connection_registry, build DSN, mở GORM pool và cache.
      │     - DDL Safety Gate:
      │       Kiểm tra information_schema.tables WHERE table_schema = ? AND table_name = ?
      │       Nếu chưa tồn tại -> Báo lỗi SchemaNotReady, KHÔNG upsert.
      │     - Batch Upsert: Ghi dữ liệu an toàn từ Shadow sang Master.
```

## 2. Chi tiết Implementation từng tầng

### 2.1 CDS Engine: `centralized-data-service/internal/service/source/connection_manager.go`
- Thêm field vào struct `ConnectionManager`:
  ```go
  masterPoolMu sync.RWMutex
  masterDBPool map[string]*gorm.DB
  ```
- Khởi tạo trong `NewConnectionManager`:
  ```go
  masterDBPool: make(map[string]*gorm.DB),
  ```
- Cập nhật hàm `GetMasterDB(ctx context.Context, key string) (*gorm.DB, error)`:
  - Nếu `key == ""` hoặc `key == "default"`, fallback về `m.reg.GetDB(database.RoleDestination)`.
  - Kiểm tra `masterPoolMu.RLock()`, nếu đã có trong `masterDBPool[key]`, trả về ngay.
  - Nếu chưa có: `masterPoolMu.Lock()`, double check cache.
  - Tìm DSN:
    + Kiểm tra `m.connectionOverrides[key]`
    + Nếu không có trong override: query DB hệ thống (`m.reg.GetDB(database.RoleSystem)`) bảng `cdc_system.connection_registry`:
      `SELECT dsn, options_json, host, port, username, password_encrypted, database_name FROM cdc_system.connection_registry WHERE code = ? AND is_active = true`
    + Khởi tạo kết nối qua `gorm.Open(postgres.Open(dsn), &gorm.Config{...})`
    + Set connection pool settings (MaxOpenConns, MaxIdleConns, ConnMaxLifetime).
    + Lưu vào `m.masterDBPool[key]`, release Lock và return `*gorm.DB`.

### 2.2 CDS Engine: `centralized-data-service/internal/service/master/transmuter.go`
- Tại đầu hàm `TransmuterModule.Run(ctx, req)`:
  - Lấy `masterDB, err := m.connMgr.GetMasterDB(ctx, req.TargetConnectionKey)`
  - Thực hiện DDL Safety Gate check:
    ```go
    var tableExists bool
    checkSQL := `SELECT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = ? AND table_name = ?
    )`
    if err := masterDB.WithContext(ctx).Raw(checkSQL, targetSchema, targetTable).Scan(&tableExists).Error; err != nil {
        return fmt.Errorf("failed to verify master table existence: %w", err)
    }
    if !tableExists {
        return fmt.Errorf("master table %s.%s does not exist on target database (DDL not created or pending approval)", targetSchema, targetTable)
    }
    ```

### 2.3 CMS Backend: `cdc-cms-service/internal/app/commands/master/create_master.go`
- Cập nhật logic:
  ```go
  masterConnectionCode := cmd.MasterConnectionCode
  if masterConnectionCode == "" {
      masterConnectionCode = h.defaultMasterConnectionCode
  }
  // Tra cứu master_connection_id từ cdc_system.connection_registry
  connID, err := h.resolveConnectionID(ctx, masterConnectionCode)
  ...
  ```

### 2.4 CMS Backend: Endpoint `GET /api/v1/master-connections`
- Thêm query/handler trả về danh sách các connection có `type = 'postgres'` và `is_active = true` từ `cdc_system.connection_registry`.

### 2.5 CMS Web: `cdc-cms-web/src/pages/MasterRegistry.tsx`
- Tải danh sách connections từ `/api/v1/master-connections`.
- Trong Modal tạo Master Table: Thêm field `Target Connection` (Select dropdown), default là "default" hoặc connection đầu tiên.
- Table columns: Hiển thị tag connection name/code.
- Nút Transmute: Kiểm tra điều kiện `record.schema_status === 'approved'`. Nếu chưa approved thì disable kèm tooltip giải thích.
