# Technical Solution Profile - Khôi phục Logic Scan Raw Data & Periodic Scan

Hồ sơ giải pháp kỹ thuật cụ thể đã được áp dụng trong quá trình sửa đổi logic của hàm `HandleScanRawData` và `HandlePeriodicScan`.

## 1. Chi tiết giải pháp `HandleScanRawData`
Đoạn code logic so khớp schema drift với rules hiện có được thiết kế như sau:

```go
// 1. Quét các keys/types từ _raw_data
rows, err := h.DB.Raw(fmt.Sprintf(`
    SELECT DISTINCT jsonb_object_keys(t._raw_data) as key, 
                    jsonb_typeof(t._raw_data->jsonb_object_keys(t._raw_data)) as type
    FROM (SELECT _raw_data FROM %s LIMIT 100) t`, tableNameQualified)).Rows()
...
// 2. Query mapping rules v2 hiện có
var existingRules []mastermodel.MappingRuleV2
err = h.DB.Where("binding_id = ?", binding.ID).Find(&existingRules).Error
...
// 3. So khớp để tìm các keys chưa có rule
for _, scanField := range scanFields {
    found := false
    for _, rule := range existingRules {
        if rule.SourcePath == scanField.Key {
            found = true
            break
        }
    }
    if !found {
        // Tự động tạo rule pending
        newRule := mastermodel.MappingRuleV2{
            BindingID:    binding.ID,
            SourcePath:   scanField.Key,
            SourceFormat: "raw",
            Status:       "pending",
            IsActive:     false,
        }
        h.DB.Create(&newRule)
    }
}
```

## 2. Chi tiết giải pháp `HandlePeriodicScan`
Sử dụng registry service để lấy cấu hình các bảng active:

```go
tableConfigs, err := h.metadataRegistry.ListTableConfigs()
...
for _, config := range tableConfigs {
    if !config.IsActive {
        continue
    }
    // Giả lập tin nhắn NATS để gọi HandleScanRawData
    msgPayload := fmt.Sprintf(`{"binding_id": "%s"}`, config.BindingID)
    h.HandleScanRawData(msgPayload)
}
```
