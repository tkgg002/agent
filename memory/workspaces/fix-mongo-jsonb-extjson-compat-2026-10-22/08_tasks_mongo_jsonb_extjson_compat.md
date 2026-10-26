# 08 Tasks: Giải quyết xung đột định dạng MongoDB Extended JSON trong cột JSONB cho các Service cũ (`mongo_jsonb_extjson_compat`)

## Nhóm 1: Khảo sát & Truy vết Toàn diện Luồng JSONB MongoDB (`S2 Investigate`)
- [x] **TASK-J01 — Truy vết tầng Shadow Ingestion & Coercion (`centralized-data-service`):**
  - **Context (`file:line`):**
    - `internal/handler/shadow/kafka_consumer.go:599-619, 921-935` (`toCanonicalMongoExtJSON`)
    - `internal/sinkworker/utils.go:35-86` (`decodeAfter`, `toCanonicalMongoExtJSON`)
    - `internal/handler/orchestration/snapshot_runner_handler.go:671-676` (`bson.MarshalExtJSON(doc, true, false)`)
    - `internal/handler/recon/recon_heal_fetch.go:73-77` (`bson.MarshalExtJSON(doc, true, false)`)
    - `internal/service/shadow/schema_adapter_coerce.go:54-140` (`CoerceValue`, `normalizeMongoExtendedJSON`)
    - `internal/service/shadow/dynamic_mapper.go:342-385` (`unwrapMongoTypes`)
  - **Mô tả:** Làm rõ cách dữ liệu MongoDB được chuyển sang Canonical Extended JSON v2 tại Shadow (`_raw_data`) và lý do không được phá vỡ cấu trúc `_raw_data` ở tầng Shadow.
  - **Definition of Done:** Xác nhận 100% vị trí `file:line` và đối chiếu với hồ sơ lịch sử `AnalysisRawDataDebeziumVsSnapshot20260929` & `canonical-mongo-extjson-scan`.

- [x] **TASK-J02 — Truy vết tầng Master Transmuter, Transform Registry, CMS Backend & CMS Web UI (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`):**
  - **Context (`file:line`):**
    - `centralized-data-service/internal/service/master/transmuter.go:991-1017, 1914-1945` (`loadRules`, `extractColumns`, `isJSONColumnType`)
    - `centralized-data-service/internal/service/master/master_ddl_generator.go:110-124` (`Generate` SQL query)
    - `centralized-data-service/internal/service/master/transmuter_utils.go:38-130` (`unwrapMongoExtJSON`, `coerceForColumn`)
    - `centralized-data-service/internal/service/master/transform_registry.go:1-225` (`ApplyTransform`, `transformRegistry`)
    - `cdc-cms-service/migrations/schema/cdc_system_model/074_v2_mapping_rule_master.sql:13`, `075_redesign_mapping_rule_master.sql:9-23`, `077_master_self_contained_nested.sql:9-13`
    - `cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:87-102, 159-207, 421-432`
    - `cdc-cms-service/internal/app/commands/master/update_column_metadata.go:11-72` & `internal/api/master/master_mapping_rule_handler.go:70-186`
    - `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:20-34, 768-826` (`MasterRule`, cột `Master Data Type`)
  - **Mô tả:** Xác định toàn bộ vị trí của `transform_fn` từ DB -> Backend -> Worker -> UI và lý do trên trang `/masters/:id/mappings?binding_id=...` chưa chọn được `transform_fn` cho từng field `JSONB`/`JSON`.
  - **Definition of Done:** Xác nhận 100% `file:line` trên cả 3 repo.

## Nhóm 2: Thiết kế Giải pháp Kỹ thuật & Kiểm định Decision Gate (`S3 Solution & DG1/DG2`)
- [x] **TASK-J03 — Thiết kế Giải pháp Full-Stack 3 Repo (`09_tasks_solution_mongo_jsonb_extjson_compat.md`) & Kiểm định `DG1/DG2`:**
  - **Context (`file:line`):**
    - `cdc-cms-service/migrations/schema/cdc_system_model/081_add_transform_fn_to_mapping_rule_master.sql`
    - `cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:87-102, 159-207`
    - `cdc-cms-service/internal/app/commands/master/update_column_metadata.go:11-72`
    - `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go:163-186`
    - `centralized-data-service/internal/service/master/transmuter.go:991-1002`
    - `centralized-data-service/internal/service/master/transmuter_utils.go:38-130`
    - `centralized-data-service/internal/service/master/transform_registry.go:31-62`
    - `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:768-826`
  - **Mô tả:** Thiết kế giải pháp đồng bộ 3 repo: (1) Thêm cột `transform_fn` vào `cdc_system.mapping_rule_master` và mở API cập nhật; (2) Bổ sung hàm đệ quy `NormalizeMongoExtJSONDeep` và 4 `transform_fn` chuyên biệt cho `JSONB` trong `centralized-data-service` + đọc `COALESCE(m.transform_fn, v2.transform_fn)`; (3) Hiển thị `<Select>` chọn `transform_fn` ngay tại cột `Master Data Type` trong `MasterMappingFieldsPage.tsx` khi kiểu dữ liệu là `JSONB` hoặc `JSON`.
  - **Definition of Done:** Hoàn tất `09_tasks_solution_mongo_jsonb_extjson_compat.md` kèm Code Demo 3 repo và chạy `laya_gate.py run --stage dg1` đạt `PASS`, trình anh duyệt (`APPROVE`).

## Nhóm 3: Thực thi & Kiểm thử sau khi có `APPROVE` (`S4 Execute -> S5 Audit -> S6 Close`)
- [x] **TASK-J04 — [Repo 1: `cdc-cms-service`] Thêm cột `transform_fn` vào `mapping_rule_master` & API cập nhật (Chờ `APPROVE`):**
  - **Context (`file:line`):** `migrations/schema/cdc_system_model/081_add_transform_fn_to_mapping_rule_master.sql`, `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:87-207`, `internal/app/commands/master/update_column_metadata.go:11-72`, `internal/api/master/master_mapping_rule_handler.go:70-186`
  - **Mô tả:** Thêm migration `081`, cập nhật `COALESCE(m.transform_fn, v2.transform_fn) AS transform_fn`, cho phép lưu và cập nhật `transform_fn` trên từng rule của Master.
  - **Definition of Done:** Build & test `cdc-cms-service` PASS (kèm biên lai `EV-...`).

- [x] **TASK-J05 — [Repo 2: `centralized-data-service`] Hiện thực hóa `NormalizeMongoExtJSONDeep`, `transformRegistry` & `COALESCE(m.transform_fn, v2.transform_fn)` (Chờ `APPROVE`):**
  - **Context (`file:line`):** `internal/service/master/transmuter_utils.go`, `internal/service/master/transform_registry.go`, `internal/service/master/transmuter.go:960`, `internal/service/master/master_ddl_generator.go:112`
  - **Mô tả:** Cài đặt hàm bóc tách đệ quy và 4 `transform_fn` cho `JSONB` (`extjson_relaxed`, `extjson_relaxed_num_str`, `extjson_unwrap_plain`, `extjson_unwrap_num_str`), cập nhật `coerceForColumn` không double-encode chuỗi JSON hợp lệ cho cột `JSONB`, và cập nhật query `loadRules` lấy `COALESCE(m.transform_fn, v2.transform_fn)`.
  - **Definition of Done:** `go test ./internal/service/master/...` PASS (kèm biên lai `EV-...`).

- [x] **TASK-J06 — [Repo 3: `cdc-cms-web`] Hiển thị `<Select>` chọn `transform_fn` khi `Master Data Type` là `JSONB`/`JSON` trên `MasterMappingFieldsPage.tsx` (Chờ `APPROVE`):**
  - **Context (`file:line`):** `src/pages/MasterMappingFieldsPage.tsx:768-860`
  - **Mô tả:** Khi `baseType === 'JSONB' || baseType === 'JSON'`, render `<Select>` ngay tại cột `Master Data Type` trên từng dòng field để chọn `transform_fn` và gọi API lưu xuống `cdc-cms-service`.
  - **Definition of Done:** `npx vite build` PASS (kèm biên lai `EV-...`), qua `DG3`.

- [x] **TASK-J07 — [S5 -> S3 -> S4 Self-Healing] Hoàn thiện `gjsonValueToGo` (`UseNumber`), tách đệ quy `normalizeMongoExtJSONValue`, bổ sung `NULLIF` SQL, validate `Save` whitelist, tự dọn `transform_fn` khi rời `JSONB`, và test trực tiếp dữ liệu thật của `binding_id=59` (`shadow_walletservice.events`):**
  - **Context (`file:line`):** `centralized-data-service/internal/service/master/transmuter_utils.go:31-305`, `transmuter.go:960`, `master_ddl_generator.go:112`, `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go:82-115`, `internal/app/commands/master/update_column_metadata.go:54-85`, `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:100`, `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:778-795`
  - **Mô tả:** Khắc phục triệt để 4 phát hiện từ phản biện từng dòng `git diff` tại `S5` và kiểm thử trực tiếp trên dữ liệu thật của `binding_id=59` (`shadow_walletservice.events`, rule `id=1293` `payload JSONB`).
  - **Definition of Done:** Toàn bộ unit + integration test trên DB thật PASS (`EV-...`), qua `DG3` (`Mode A: run --stage dg3` + `Mode B: review`).

