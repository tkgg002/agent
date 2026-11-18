# 08 Tasks — scan_nested_id
- [x] T1: Bỏ `"id"`, `"_id"` khỏi `extraNestedScanSkipFields` (scan_field_filter.go:17-24). Context: scan_service.go:382-400 target nested = `basePath_relPath` (vd `merchantInfo_id`) nên không đụng PK.
- [x] T1b: Nhánh shadow scan_service.go:401 thêm `shouldSkipTopLevelScanField(relPath)` để không đè PK.
- [x] T2: Thêm test `TestScanFieldFilter_AllowsNestedIDWhileBlockingTopLevelIDAndExplodeCols`: shouldSkipNestedScanField("id")==false, ("_id")==false; ("_parent_source_id")/("_array_index")==true; top-level "id" vẫn true (`EV-20261117T095810-88b6`).
- DoD: `go test ./internal/service/source/ -run ScanFieldFilter` PASS (EV-...), scan lại merchantInfo hiện `merchantInfo_id`.
- [x] T3 (F2): migration `cdc-cms-service/migrations/schema/cdc_system_model/083_mapping_rule_master_source_field_text.sql` ALTER COLUMN source_field TYPE TEXT (committed 5a8bac7).
