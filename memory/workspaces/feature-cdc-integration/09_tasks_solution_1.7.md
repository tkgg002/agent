# Technical Solution: Task 1.7 - NATS Reload & Indexing Fix

## 1. Problem Definition
- **Component**: `RegistryService` (Worker Service)
- **Bug**: Lỗi Index Mismatch khi lưu trữ Mapping Rules vào bộ nhớ (Cache).
- **Chi tiết**: 
    - `ReloadAll` đang lấy danh sách `MappingRule` (chứa `SourceTable`) và lưu vào cache với Key là `SourceTable`.
    - Tuy nhiên, `EventHandler` nhận dữ liệu đã được định danh theo `TargetTable` (ví dụ: `cdc_merchants`) và gọi hàm `GetMappingRules(targetTable)`.
    - Do `mappingCache` không chứa Key `targetTable`, kết quả trả về luôn là `nil`, dẫn đến việc bỏ qua các quy tắc chuẩn hóa dữ liệu.

## 2. Solution Design

### A. Quy trình Reload mới (RegistryService.ReloadAll)
1. **Load Bảng đăng ký**: Lấy tất cả `TableRegistry` đang hoạt động từ DB.
2. **Xây dựng Bản đồ Ánh xạ (Source to Target)**:
   ```go
   sourceToTarget := make(map[string]string)
   for _, entry := range entries {
       sourceToTarget[entry.SourceTable] = entry.TargetTable
   }
   ```
3. **Load Quy tắc Mapping**: Lấy tất cả `MappingRule` đang hoạt động.
4. **Lưu Cache theo TargetTable**:
   ```go
   rs.mappingCache = make(map[string][]model.MappingRule)
   for _, r := range rules {
       targetTable := sourceToTarget[r.SourceTable]
       if targetTable != "" {
           rs.mappingCache[targetTable] = append(rs.mappingCache[targetTable], r)
       }
   }
   ```

### B. Kiểm soát Độc quyền (Concurrency)
Sử dụng `rs.mu.Lock()` xuyên suốt quá trình từ khi tạo map mới cho đến khi ghi đè vào cache cũ để tránh tình trạng dữ liệu không nhất quán (Race Condition) khi worker đang xử lý sự kiện đồng thời.

## 3. Implementation Steps
1. Sửa file `internal/service/registry_service.go` theo logic trên.
2. Thêm Log `zap.Info` để báo cáo chi tiết: "Mapped [N] rules to target table [TableName]".
3. Kiểm tra NATS Subscriber trong `worker_server.go` để đảm bảo lệnh reload gọi đúng hàm này.

## 4. Verification
- Chạy unit test `registry_service_test.go` với dữ liệu giả lập.
- Sử dụng lệnh `nats pub schema.config.reload "{}"` để kiểm tra khả năng hot-reload thực tế.
