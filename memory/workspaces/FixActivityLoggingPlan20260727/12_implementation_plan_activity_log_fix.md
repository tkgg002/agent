# 12_implementation_plan_activity_log_fix.md

## Kế hoạch Triển khai Kỹ thuật Tối giản (Implementation Plan)

### Mục tiêu
Sửa duy nhất 1 điểm lỗi gốc rễ làm trượt SQL Read Join ở Log #28232 (truyền `tableName` thay vì `targetFQN` ở `batch_buffer.go`), làm sạch `details` JSON của Log #28233, giữ nguyên 100% service `cdc-cms-service`.

### Target Files
1. **[MODIFY] `centralized-data-service/internal/handler/shadow/batch_buffer.go`**
   - Đổi `targetFQN` sang `tableName` khi gọi `act.Start("kafka-consumer", tableName, "kafka-consumer")`.
2. **[MODIFY] `centralized-data-service/internal/sinkworker/worker.go`**
   - Đổi `targetFQN` sang `tableName` khi gọi `w.activity.Start("sink-upsert", tableName, "kafka-consumer")`.
3. **[MODIFY] `centralized-data-service/internal/handler/master/transmute_handler.go`**
   - Thêm `correlation_id` vào `details` JSON.
   - Xóa bỏ `duration_ms` dư thừa khỏi `details` JSON.

### Files KHÔNG SỬA
- `cdc-cms-service`: **GIỮ NGUYÊN 100%**.
