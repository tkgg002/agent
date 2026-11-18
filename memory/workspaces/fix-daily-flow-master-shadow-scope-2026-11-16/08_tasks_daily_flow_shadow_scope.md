# Tasks — daily_flow_shadow_scope
- [x] S2 Điều tra: SQL GetDailyFlow (cdc-cms-service activity_log_read_repo_gorm.go:197-200) lọc transmute chỉ theo master_table trần.
- [x] S2 Điều tra: transmute log (centralized-data-service transmute_handler.go:752-766, transmute_poison_pill_split.go) không ghi shadow identity; multi_shadow fan-in qua transform_spec.shadow_sources (master_binding_repo.go:109-119).
- [x] S4 (sau APPROVE) T1: thêm shadow scope vào TransmuteRequest + debounceRunKey + details log (`transmute_shadow_scope.go`, `transmute_handler.go`, `transmute_event_date_split.go`, `transmute_poison_pill_split.go`).
- [x] S4 T2: GetDailyFlow + ListActivity + ClearActivity lọc transmute theo shadow_schema/shadow_table (`activity_log_read_repo_gorm.go`, `activity_log_list_clear_gorm.go`), có fallback an toàn cho orphan binding khi bảng shadow là duy nhất.
- [x] S4 T3: test Postgres thật 2 shadow cùng tên bảng (`activity_log_daily_flow_shadow_scope_test.go`) + unit test `TestGroupDebouncedTasks_SeparatesSameNameShadowsOnSharedBinding` + đồng bộ bộ đếm 3-tier trong `verify_governance.py`; go build/test 2 service PASS.
- DoD: xem 01_requirements.
