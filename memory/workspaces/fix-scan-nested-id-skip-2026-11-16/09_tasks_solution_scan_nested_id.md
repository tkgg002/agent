# 09 Solution — scan_nested_id

## F1 Root cause (VERIFIED)
- `scan_service.go:382-385`: mọi relPath sau flatten đi qua `shouldSkipNestedScanField`.
- `scan_field_filter.go:17-22`: nested skip list chứa `"id"` và `"_id"` → `merchantInfo.id` (relPath=`id`) bị `continue`, không bao giờ tạo rule.
- Skip này vô lý cho nested: target column = `strings.ReplaceAll(basePath+"."+relPath, ".", "_")` (scan_service.go:388) = `merchantInfo_id` → không trùng PK `id`/`_id` của bảng. Bug có từ map cũ, refactor 9d41783 bê nguyên sang.

## Lưu ý phản biện (S2.5)
- Nhánh shadow (scan_service.go:401-425, khi không có master binding) tạo rule `TargetColumn = relPath` KHÔNG prefix → nếu thả `id` ở nhánh này sẽ đè PK `id`. Vì vậy skip `id/_id` phải giữ ĐÚNG ở nhánh không-prefix, bỏ ở nhánh có prefix.

## Fix (1 phương án)
```go
// scan_field_filter.go — nested chỉ lọc cột hệ thống
var extraNestedScanSkipFields = map[string]struct{}{
	"_parent_source_id": {},
	"_array_index":      {},
}

// scan_service.go:401 — nhánh shadow ghi cột top-level không prefix → áp luật top-level
} else if binding.ID > 0 && !isArrayExplode && !strings.Contains(relPath, ".") {
	if shouldSkipTopLevelScanField(relPath) || existingV2[relPath] {
		continue
	}
```
+ test regression trong scan_field_filter_test.go.

## Impact
- Nhánh master (object/array-explode): `merchantInfo.id` → cột `merchantInfo_id` hiện ra. Nhánh shadow không đổi hành vi (vẫn chặn `id/_id`).

## F2 — source_field -> TEXT
- Hiện trạng: 077_master_self_contained_nested.sql:10 tạo `source_field VARCHAR(255)`; 074 (bảng cũ) VARCHAR(255). Không có GORM `size` tag/validation 255 trong code.
```sql
BEGIN;
ALTER TABLE cdc_system.mapping_rule_master ALTER COLUMN source_field TYPE TEXT;
COMMIT;
```
## F3 — Hoàn thiện thực thi T1 + T2 (2026-11-17, sau User APPROVE)
- Áp dụng nốt thay đổi tại `internal/service/source/scan_field_filter.go:17-22` (xóa `"_id"`, `"id"` khỏi `extraNestedScanSkipFields`, chỉ giữ `_parent_source_id` và `_array_index`).
- Bổ sung `TestScanFieldFilter_AllowsNestedIDWhileBlockingTopLevelIDAndExplodeCols` vào `internal/service/source/scan_field_filter_test.go` để kiểm chứng đủ 3 chiều: cho phép nested `id`/`_id`, chặn nested `_parent_source_id`/`_array_index`, và chặn top-level `id`/`_id`.
