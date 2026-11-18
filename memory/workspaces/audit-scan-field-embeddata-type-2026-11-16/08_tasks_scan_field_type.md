# 08 Tasks — scan_field_type

- [x] T1: Trace luồng Scan Fields (CMS `cdc.cmd.scan-fields` -> worker `ScanFieldsDebezium`).
- [x] T2: Đối chiếu data thật (mapping_rule_v2 id 688/743, shadow_traitestpbs.payment_bills `_raw_data->embedData`).
- [x] T3: Viết `09_tasks_solution_scan_field_type.md` (1 phương án + Code Demo).
- [x] T4: Chờ User APPROVE -> S4 sửa `mongo_introspection.go`, `discover_handler_mongo.go`, `discover_handler_utils.go` + unit test Red->Green.
- DoD: field có giá trị object ở bất kỳ doc mẫu nào -> JSONB; giá trị null không được bỏ phiếu TEXT; test cover mẫu {null..., object}.
