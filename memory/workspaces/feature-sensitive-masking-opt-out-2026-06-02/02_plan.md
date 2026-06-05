# 02_plan.md - Kế hoạch Fix Column-level JSON Masking trong DynamicMapper

## 1. Goal / Mục tiêu
Sửa lỗi cấu hình chiến lược `json_mask` ở cấp cột không hoạt động trong `DynamicMapper`. Khi một cột có rule cấu hình chiến lược `json_mask`, `DynamicMapper` đi qua `MaskByStrategy` nhưng chiến lược này không có thông tin về bảng shadow đích để phân giải danh sách các trường con nhạy cảm cần mã hóa.

Giải pháp:
- Expose phương thức `MaskJSONFields` từ `MaskingService` để cho phép `DynamicMapper` gọi trực tiếp khi phát hiện strategy là `json_mask`.
- Điều chỉnh signature của `maybeMaskColumn` trong `DynamicMapper` để nhận thêm `targetTable string`.
- Điều chỉnh signature của `MapColumnsFromElement` để truyền nhận `targetTable string`.
- Cập nhật các call site trong `dynamic_mapper.go` và `child_explode.go` để truyền đúng tên bảng shadow đích.

---

## 2. Proposed Changes / Các thay đổi đề xuất

### A. Component: centralized-data-service

#### [MODIFY] [masking_service.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/masking_service.go)
Expose public method `MaskJSONFields`:
```go
func (s *MaskingService) MaskJSONFields(table string, value interface{}) interface{} {
    s.mu.RLock()
    maskMap, ok := s.tableMaskMaps[table]
    s.mu.RUnlock()
    if !ok {
        // If no table-specific configuration, fallback to global or resolve dynamic
        maskMap = s.ResolveMaskMap(table)
    }
    return s.maskJSONFields(value, maskMap)
}
```

#### [MODIFY] [dynamic_mapper.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/dynamic_mapper.go)
- Sửa đổi phương thức `maybeMaskColumn`:
```go
func (dm *DynamicMapper) maybeMaskColumn(targetTable string, rule model.MappingRule, value interface{}) interface{} {
    // ...
    if rule.MaskStrategy == MaskStrategyJSONMask {
        return dm.masking.MaskJSONFields(targetTable, value)
    }
    return dm.masking.MaskByStrategy(rule.MaskStrategy, value)
}
```
- Sửa đổi signature của `maybeMaskColumn` và `MapColumnsFromElement` để nhận thêm tham số `targetTable string`.
- Cập nhật các lời gọi tới `maybeMaskColumn` trong `MapData` để truyền `targetTable`.

#### [MODIFY] [child_explode.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/child_explode.go)
- Cập nhật cuộc gọi tới `MapColumnsFromElement` để truyền `child.ShadowTable` thay vì signature cũ.

#### [MODIFY] [masking_service_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/masking_service_test.go)
- Viết thêm bài test xác thực việc column-level JSON masking chạy chính xác qua `DynamicMapper`.

---

## 3. Verification Plan / Kế hoạch kiểm thử

### Automated Tests
- Chạy unit test trong `centralized-data-service`:
  ```bash
  go test -v ./internal/service/...
  ```
- Kiểm tra các test case liên quan đến dynamic mapping và masking.
