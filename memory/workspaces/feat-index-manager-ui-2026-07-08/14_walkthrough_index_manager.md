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

---

## 3. Bản cập nhật & Khắc phục Lỗi Khuyến nghị Index (Timestamp Field) - 14/07/2026

### A. Mô tả Sự cố & Nguyên nhân
- **Hiện tượng**: Không hiển thị các khuyến nghị tạo index cho trường `Timestamp Field` của các Shadow Table trong CMS UI.
- **Nguyên nhân**:
  1. Hàm `GetRecommendations` thực hiện truy vấn schema metadata trên `masterDB` (context-mapped sang database của shadow table) thay vì database hệ thống `cdc_system` (`systemDB`).
  2. Cách truy vấn lookup registry để xác định `source_object_id` và giải quyết `target_column` bị sai lệch.
  3. Thiếu cơ chế tạo tự động index cho `Timestamp Field` khi khởi tạo mới / provision shadow table.

### B. Các Giải pháp đã Triển khai
1. **Sửa đổi backend `GetRecommendations` (`index_manager.go`)**:
   - Chuyển toàn bộ các truy vấn metadata hệ thống sang kết nối `systemDB`.
   - Cập nhật logic registry lookup bằng cách thực hiện `JOIN` giữa `cdc_system.mapping_rule_v2` và `cdc_system.source_object_registry` để ánh xạ chính xác từ `source_field` của source table sang `target_column` của shadow table.
2. **Cập nhật `IndexHandler`**:
   - Sử dụng hàm `GetSystemDB()` thay cho `GetMasterDB()` để cung cấp đúng context connection cho `GetRecommendations`.
3. **Thêm cơ chế Auto-create Index khi Provision Shadow Table (`schema_ddl_handler.go`)**:
   - Bổ sung hàm `CreateIndex` vào `SchemaAdapter`.
   - Trong `HandleCreateDefaultColumns`, sau khi cập nhật trạng thái registry, tiến hành truy vấn trường `Timestamp Field` của bảng đích.
   - Tìm kiếm `target_column` tương ứng qua `mapping_rule_v2`.
   - Thực hiện tạo index ngay lập tức trên cột đó bằng phương thức `CreateIndex`.
4. **Cập nhật Frontend `TableIndexManager.tsx`**:
   - Đảm bảo hiển thị đúng và đầy đủ các khuyến nghị nhận được từ API backend.
   - Nút "Tạo Index ngay" được kích hoạt và truyền đúng payload lên backend.

### C. Kết quả Xác minh & Kiểm thử Thực tế
- **Tự động hóa**: Đã bổ sung unit test kiểm tra logic đề xuất và mapping (`TestIndexManager_GetRecommendations_TimestampField`). Test chạy PASS.
- **Thử nghiệm trên CMS UI**:
  - Giao diện "Quản lý Indexes (Shadow Table)" hiển thị cảnh báo khuyến nghị:
    `Khuyến nghị: idx_schedules_last_updated_at: Tối ưu hóa MaxWindowTs: Tạo index trên cột lastUpdatedAt...` cùng nút **Tạo Index ngay**.
  - Khi bấm nút **Tạo Index ngay**: backend thực hiện thành công câu lệnh DDL tạo index.
  - Cảnh báo khuyến nghị biến mất hoàn toàn trên UI, và index `idx_schedules_lastUpdatedAt` xuất hiện trong danh sách index bên dưới với kích thước `16 kB` và scan count `0`.

---

## 4. Bản cập nhật & Khắc phục Lỗi Cột không tồn tại (Column Does Not Exist) - 14/07/2026

### A. Mô tả Sự cố & Nguyên nhân
- **Hiện tượng**: Lỗi SQL `ERROR: column "lastUpdatedAt" does not exist (SQLSTATE 42703)` xảy ra khi cố gắng tạo index trên shadow table cho các bảng chưa được provision các trường nghiệp vụ (business columns) hoặc cấu hình registry `timestamp_field` bị lệch so với cột thực tế trong DB.
- **Nguyên nhân**:
  - Giao diện khuyến nghị hoặc tiến trình provisioning tự động tìm thấy cấu hình `timestamp_field` từ registry và đề xuất/tạo index mà không kiểm tra xem cột đó có tồn tại vật lý trong bảng đích hay không.
  - Khi cột không tồn tại, PostgreSQL trả về lỗi `SQLSTATE 42703` làm dừng/báo lỗi các luồng nghiệp vụ.

### B. Các Giải pháp đã Triển khai
1. **Kiểm tra sự tồn tại của cột trong `GetRecommendations` (`index_manager.go`)**:
   - Trước khi đề xuất index cho trường timestamp, backend truy vấn `information_schema.columns` để lấy danh sách cột thực tế của bảng.
   - Hỗ trợ so khớp cột linh hoạt: Khớp chính xác tên cột (`lastUpdatedAt`), khớp theo kiểu snake_case (`last_updated_at`), hoặc khớp không phân biệt chữ hoa chữ thường.
   - Chỉ đưa ra khuyến nghị nếu cột thực sự tồn tại trong DB.
2. **Kiểm tra sự tồn tại của cột trong tự động hóa provisioning (`schema_ddl_handler.go`)**:
   - Trong `HandleCreateDefaultColumns`, trước khi tự động gọi tạo index mặc định cho timestamp field, kiểm tra thông qua `schemaAdapter` xem cột đó có tồn tại không. Nếu không tồn tại, ghi log warning và bỏ qua an toàn thay vì thực thi DDL lỗi.
3. **Cải tiến `CreateIndexConcurrently`**:
   - Kiểm tra danh sách cột đầu vào và trả về lỗi chi tiết, rõ ràng (ví dụ: `column "lastUpdatedAt" does not exist in table shadow_testss.payment_settlement_terms`) thay vì chạy truy vấn SQL lỗi trực tiếp.
4. **Bổ sung Unit Test**:
   - Thêm `TestIndexManager_NonExistentColumn` để đảm bảo hệ thống chặn chính xác việc tạo index trên cột không tồn tại.
   - Cập nhật `TestIndexManager_GetRecommendations_TimestampField` để tạo bảng mock hoàn chỉnh chứa cột cần khuyến nghị.

### C. Kết quả Xác minh & Kiểm thử Thực tế
- Chạy toàn bộ test suite của `governance` đạt **PASS 100%**.
- Việc introspect và đề xuất index cho các bảng như `payment_settlement_terms` (chỉ có system columns) diễn ra an toàn, không còn đề xuất cột không tồn tại, ngăn chặn hoàn toàn lỗi DDL index.


