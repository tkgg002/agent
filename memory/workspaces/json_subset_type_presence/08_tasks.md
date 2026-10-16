# 08 Tasks — JSON Subset type/presence

## Frontend (cdc-cms-web)
- [x] Thêm state `jsonSubsetType` ('value' | 'type_presence') vào MasterMappingFieldsPage.tsx
- [x] Thêm Radio.Group trong UI khi JSON Subset mode ON
- [x] Sửa submit handler: khi type_presence → source_field = `@type_presence:{field1,field2,...}`
- [x] Thêm Alert mô tả chế độ mới (động theo loại subset)
- [x] Reset state khi đóng modal
- [x] TypeScript compile: 0 errors ✅

## Backend Runtime (centralized-data-service)
- [x] Tìm module xử lý JSONB subset hiện tại → `transmuter.go:extractColumns` + gjson multipath
- [x] Thêm hàm `buildTypePresenceSubset(rawStr, fields)` vào `transmuter_utils.go`
- [x] Thêm hàm `detectBsonType(val)` vào `transmuter_utils.go`
- [x] Thêm hàm `parseTypePresenceFields(sourceField)` + `isTypePresenceRule(sourceField)` vào `transmuter_utils.go`
- [x] Hook vào `extractColumns`: nếu source_field bắt đầu `@type_presence:` → gọi buildTypePresenceSubset
- [x] Go build: 0 errors ✅
- [x] Go test transmute: 9/9 PASS ✅
