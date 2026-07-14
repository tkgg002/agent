# Walkthrough: Index Management Feature Integration

Dưới đây là tổng hợp kết quả tích hợp tính năng Index Management trên toàn bộ hệ thống Data Hub.

## 1. Các File đã Thay đổi & Khởi tạo

### A. Worker Service (`centralized-data-service`)
- **[NEW]** `internal/service/governance/index_manager.go`: Core logic xử lý `CREATE/DROP INDEX CONCURRENTLY` & liệt kê index trực tiếp từ catalog.
- **[NEW]** `internal/service/governance/index_manager_test.go`: Test-suite kiểm tra chu trình tạo - kiểm tra - xóa index thực tế trên DB Postgres.
- **[NEW]** `internal/handler/governance/index_handler.go`: NATS RPC handler tiếp nhận và chuyển tiếp request từ CMS.
- **[MODIFY]** `internal/server/server_setup.go`: Đăng ký NATS handlers mới vào vòng đời start-up của Worker.

### B. CMS Control Plane (`cdc-cms-service`)
- **[MODIFY]** `internal/api/system/introspection_handler.go`: Định nghĩa các API endpoint proxy NATS: `ListIndexes`, `CreateIndex`, `DropIndex`.
- **[MODIFY]** `internal/router/router.go`: Đăng ký routes HTTP tương ứng.

### C. CMS Frontend (`cdc-cms-web`)
- **[NEW]** `src/components/TableIndexManager.tsx`: Giao diện quản lý index đẹp mắt, tích hợp Auto Recommendations cho `_deleted`, `_source_ts`, và `_source_id`.
- **[MODIFY]** `src/pages/MappingFieldsPage.tsx`: Nhúng `TableIndexManager` cho Shadow DB.
- **[MODIFY]** `src/pages/MasterMappingFieldsPage.tsx`: Nhúng `TableIndexManager` cho Master DB.
- **[MODIFY]** `src/pages/SourceConnectors.tsx`: Thay đổi URL / Host sang Connector / Topic hiển thị tên connector thay vì server_address/URL để tránh rò rỉ và lỗi hiển thị placeholder.

---

## 2. Kết quả Kiểm thử & Biên dịch

### A. Kiểm thử Integration (Worker)
Chạy test-suite thành công trên môi trường DB thực tế:
```bash
go test -v ./internal/service/governance/ -run TestIndexManager_Lifecycle
```
**Kết quả**:
```
=== RUN   TestIndexManager_Lifecycle
--- PASS: TestIndexManager_Lifecycle (0.93s)
PASS
ok  	centralized-data-service/internal/service/governance	2.005s
```

### B. Biên dịch Frontend
Chạy lệnh đóng gói production thành công:
```bash
npm run build
```
**Kết quả**: Build thành công toàn bộ bundle (bao gồm chunk `TableIndexManager-B3AQWqUx.js`), không lỗi TS lint.
