# Airbyte Dynamic Catalog Synchronization Plan & Merge Schema Arch

Khắc phục triệt để lỗi "mất stream (schema) trên Airbyte do tắt Active" đồng thời bảo toàn toàn bộ cấu hình nâng cao của Airbyte (Sync mode, Cursor field).

## RCA & Architectural Problem
Việc kết nối vào API `/v1/sources/discover_schema` sẽ luôn trả về 100% Raw Schema nhưng **SẼ BỊ XÓA DỮ LIỆU CẤU HÌNH** nếu ghi đè thẳng qua UpdateConnection.
Vì Airbyte API không theo chuẩn RESTful (chỉ có DiscoverSchema full, hoặc GetConnection bị Drop inactive tables), chúng ta buộc phải dùng mô hình **Merge State** trong Client.

## Proposed Changes - Merge Config Mode

### CDC CMS API (Sync Logic)
**File**: `internal/api/registry_handler.go`
Thay đổi logic hàm `syncRegistryStateToAirbyte`:
1. **Lấy Current Settings**: Fetch lại `GetConnection` để cache toàn bộ cấu hình `sync_mode`, `cursor_field`, `primary_key` đang có của các Stream.
2. **Lấy Full Source**: Chạy `discoverCatalog, err := h.airbyteClient.DiscoverSchema()`.
3. **Lấy Active Tables từ CMS**: Get `db.Where("airbyte_connection_id = ? AND is_active = true", connID)`.
4. **Hợp nhất (Merge)**:
   - Loop qua toàn bộ Streams trong `discoverCatalog`.
   - Nếu Stream có trong cache (GetConnection) -> Chép đè cấu hình `config` của cache sang `discoverCatalog` để bảo vệ Incremental Sync.
   - Trạng thái `selected = true` nếu nằm trong `Active Tables` từ DB, ngược lại `false`.
5. Thực thi `UpdateConnection(conn.ID, newStatus, discoverCatalog.Streams)`.

### Mã Giả (Logic)
```go
// Giữ nguyên config cũ thay vì dùng default của discover
for i, stream := range discoverCatalog.Streams {
    if oldConfig, exists := current[stream.Stream.Name]; exists {
        discoverCatalog.Streams[i].Config.SyncMode = oldConfig.SyncMode
        discoverCatalog.Streams[i].Config.DestinationSyncMode = oldConfig.DestinationSyncMode
        discoverCatalog.Streams[i].Config.CursorField = oldConfig.CursorField
    }
}
```

## Verification Plan
1. Restart CMS (`make run`).
2. Kích hoạt thử một Data Stream đang off. Theo dõi Log Airbyte để xác nhận Incremental Mode vẫn đang hiển thị trên Stream gốc lẫn Stream mới bật.
3. Không gây crash backend do Go Context hoặc Nil references.
