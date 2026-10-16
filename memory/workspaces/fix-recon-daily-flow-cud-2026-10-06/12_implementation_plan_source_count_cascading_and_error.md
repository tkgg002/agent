# KẾ HOẠCH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PLAN)
## SỬA 3 LỖI SOURCE COUNT, CASCADING METADATA & ERROR TOOLTIP

### BƯỚC 1: BACKEND CMS SERVICE (`cdc-cms-service`)
1. Mở rộng `master_repo_gorm.go`:
   - Bổ sung hàm `ListAllConnections(ctx context.Context, roleFilter string) ([]master.MasterConnectionItem, error)`:
     - Nếu `roleFilter == ""` hoặc `roleFilter == "all"`: query `status = 'active'`.
     - Nếu có `roleFilter`: query `status = 'active' AND role_type = ?`.
   - Giữ nguyên `ListMasterConnections` để không ảnh hưởng các caller hiện tại, hoặc cho `ListMasterConnections` gọi hàm này với `role_type = 'master'`.
2. Mở rộng `master_registry_handler_connections.go`:
   - Trong `ListConnections`: đọc query param `role := c.Query("role")` hoặc `roleType := c.Query("role_type")`.
   - Nếu `role == "all"` hoặc route được gọi là `/api/v1/connections`: trả về toàn bộ connection active trong `cdc_system.connection_registry`.
3. Đăng ký route trong `router.go`:
   - Đăng ký `GET /v1/connections` và `/connections` trỏ về `h.Master.Registry.ListConnections`.
4. Compile & Verify:
   - `go build -o server cmd/server/main.go` -> PASS.

---

### BƯỚC 2: WORKER SERVER (`centralized-data-service`)
1. Cập nhật `internal/server/server_scheduler.go`:
   - Trong khối `defer func() { if r := recover(); r != nil { ... } }`:
     - Bổ sung câu lệnh cập nhật `last_error = fmt.Sprintf("panic: %v", r)` vào database bảng `cdc_system.cdc_worker_schedule` cho `sched.ID`.
2. Compile & Test:
   - `go test -v ./internal/server/...` -> PASS.
   - `go build ./cmd/...` -> PASS.

---

### BƯỚC 3: FRONTEND CMS WEB (`cdc-cms-web`)
1. Cập nhật query nạp connections trong `ActivityManager.tsx`:
   - Gọi `cmsApi.get('/api/v1/connections')` hoặc `/api/v1/master-connections?role=all`.
2. Cập nhật `CascadingSourceSelector` trong `ActivityManager.tsx`:
   - `connectionOptions`: Lọc chính xác theo `selectedLayer`:
     - `source`: `c.role_type === 'source'`
     - `shadow`: `c.role_type === 'shadow'`
     - `master`: `c.role_type === 'master'`
   - `namespaceOptions`:
     - Tìm `currentConn` từ `rawConnections` theo `selectedConn`.
     - Với `master`: Thêm `currentConn.default_schema` (vd `public`), `currentConn.default_database` (vd `goopay_master_2`), và các `m.master_schema` từ `rawMasterBindings` có `m.master_connection_code === selectedConn || m.master_connection_id === currentConn.id`.
     - Với `source`: Thêm `currentConn.default_database` và các `s.source_db` từ `sourceObjects` có `s.source_connection_code === selectedConn`.
     - Với `shadow`: Thêm `currentConn.default_database` (`cdc_shadow`) và các `s.shadow_schema` từ `rawShadowBindings`.
   - `objectOptions`:
     - Với `master`: Lọc theo đúng `selectedNamespace` và các binding thuộc `selectedConn`.
3. Nâng cấp hiển thị cột `last_error` trong bảng `ActivityManager.tsx`:
   - Khi có lỗi: Hiển thị `<Tag icon={<CloseCircleOutlined />} color="error" style={{ cursor: 'pointer' }}>Có lỗi</Tag>`.
   - Bọc trong `Tooltip` chuyên dụng với style rõ ràng, tiêu đề "Chi tiết lỗi thực thi:", hiển thị trọn vẹn thông báo lỗi, có nút Copy to clipboard thông báo lỗi.
4. Compile & Build:
   - `npm run build` -> PASS.

---

### BƯỚC 4: AUDIT & BÁO CÁO NGHIỆM THU
- Kiểm tra Gate G1–G8 theo Hiến pháp Rule #14.
- Append log vào `05_progress.md`.
- Lập báo cáo chi tiết tại `11_report_source_count_cascading_and_error.md`.
