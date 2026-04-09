# Phase 1.7 Task: NATS Reload & Index Fix

## Context
Hiện tại, CDC Worker đang gặp lỗi Index Mismatch:
- Registry lưu cache theo `SourceTable` (e.g., `merchants`).
- EventHandler tìm kiếm theo `TargetTable` (e.g., `cdc_merchants`).
- Hệ quả: Mapping rules không bao giờ được áp dụng, làm tê liệt khả năng đồng bộ dữ liệu chuẩn hóa.

## Task Checklist

### 1. Registry Service (Muscle Work)
- [ ] **Fix `ReloadAll()`**: Cập nhật logic để `mappingCache` sử dụng `TargetTable` làm key.
  - Cần lấy `sourceToTargetMap` từ `TableRegistry` trước khi xử lý `MappingRules`.
- [ ] **Thread Safety**: Đảm bảo `mappingLock` (sync.RWMutex) bao quát toàn bộ quá trình reload cache.
- [ ] **Logging**: Ghi log số lượng rules đã load theo từng `TargetTable`.

### 2. NATS Handler (Muscle Work)
- [ ] **Verify Subscriber**: Kiểm tra `worker_server.go` xem subscriber đã nối đúng channel `schema.config.reload` chưa.
- [ ] **Audit Payload**: Đảm bảo handler log lại metadata (user_id, source) của lệnh reload để phục vụ Audit Log.

### 3. Verification (Brain Check)
- [ ] **Unit Test**: Viết test case giả lập MappingRule và TableRegistry để kiểm tra hàm `GetMappingRules`.
- [ ] **Integration Log**: Chạy service và gởi lệnh reload qua NATS để quan sát log.

## Next Step
- Sau khi hoàn thành 1.7, CMS và Worker sẽ được kết nối hoàn chỉnh (Close the Loop).
- Tiếp theo sẽ là Phase 1.8: Hoàn thiện CMS FE (React).
