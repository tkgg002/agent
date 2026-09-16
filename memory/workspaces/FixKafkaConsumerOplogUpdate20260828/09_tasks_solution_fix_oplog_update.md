# Technical Solution: Fix Kafka Consumer Oplog UPDATE Event Processing

## 1. Problem Statement
Sự kiện `UPDATE` (`op == "u"`) từ Debezium MongoDB CDC bị bỏ sót khi Kafka Consumer xử lý vì 3 nguyên nhân:

1. **Connector Configuration**: Connector được tạo với `"capture.mode": "change_streams"` → Debezium Mongo Connector không đưa full document vào trường `after` đối với sự kiện UPDATE (`op: "u"`).
2. **Kafka Consumer Safeguard**: `internal/handler/shadow/kafka_consumer.go:596-605` kiểm tra:
   ```go
   if afterData == nil && opStr != "d" {
       // DROPS EVENT
       return 0, nil
   }
   ```
   Khi `afterData` là `nil` và `opStr` là `"u"`, tin nhắn bị coi là invalid non-delete và bị DROP.
3. **Event Handler Safeguard**: `internal/handler/shadow/event_handler.go:308-313` kiểm tra:
   ```go
   if data == nil {
       return 0, fmt.Errorf("no 'after' data in event for table %s", sourceTable)
   }
   ```

## 2. Proposed Changes

### Step 1: Fix Debezium Connector Configuration
Trong `cdc_system.sources` (và khi tạo/recover Debezium MongoDB Connectors), đổi cấu hình `"capture.mode"`:
```json
"capture.mode": "change_streams_with_full_update"
```
Điều này đảm bảo MongoDB Change Streams luôn phát đi bản ghi đầy đủ trong trường `after` khi có bất kỳ thay đổi UPDATE nào.

### Step 2: Update `kafka_consumer.go`
Sửa nhánh kiểm tra `nil_after_data` trong `kafka_consumer.go`:
```go
if afterData == nil && opStr != "d" {
    // Nếu là 'u' và có 'patch' hoặc 'updateDescription', cố gắng parse hoặc log chi tiết thay vì lặng lẽ drop.
    // Nếu capture.mode là change_streams_with_full_update, afterData sẽ luôn được cung cấp.
}
```

### Step 3: Update `event_handler.go`
Bổ sung xử lý khi `op == "u"` và `after` rỗng, đảm bảo tin nhắn không rớt im lặng.

## 3. Verification Plan
- Chạy `go test ./internal/handler/shadow/...`
- Đối soát DB Shadow (`cdc_shadow`) kiểm tra số lượng câu UPDATE (`_updated_at > _created_at`).

## 4. Recon Timestamp Misalignment Issue & Solution
### Problem:
Recon resolve `dstTS` bị rơi về `_source_ts` hoặc `_updated_at` làm trôi cửa sổ thời gian (Time Window Shift) khi so sánh với MongoDB `lastUpdatedAt`.

### Exact Code Locations:
1. `internal/service/recon/recon_tier_a.go:201-236`: `resolveSourceAndDestTSFields` fallback về `_source_ts` khi `ColumnExists` trả về `false`.
2. `internal/service/recon/recon_stream_bucket_engine.go:185-205 & 584-590`: Fallback `tsCol` về `_source_ts` / `updated_at`.
3. `internal/service/recon/recon_dest_query.go:86-109`: `ColumnExists` check `information_schema.columns` với `column_name = ?` nhạy cảm hoa thường.

### Proposed Fix:
1. Sửa `ColumnExists` trong `recon_dest_query.go` sử dụng `LOWER(column_name) = LOWER(?)` để hỗ trợ cả camelCase và snake_case.
2. Sửa fallback trong `recon_tier_a.go` và `recon_stream_bucket_engine.go` ưu tiên giữ đúng cột timestamp nghiệp vụ của Mongo thay vì fallback rớt sang `_updated_at` / `_source_ts`.

