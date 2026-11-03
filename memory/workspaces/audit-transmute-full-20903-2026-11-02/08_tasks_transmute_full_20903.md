# 08. Tasks Breakdown — `transmute_full_20903`

## Danh sách Task Điều tra & Giải pháp

### [x] TASK-01: Truy vết End-to-End luồng "Chạy ngay toàn bộ" và hiển thị `Synced (20,903 / 20,903)` cho `binding_id=76` (`transaction_history_2`)
- **Mô tả & Context `file:line`:**
  - `cdc-cms-web/src/pages/MasterRegistry.tsx:166-200` & `326-346`: Nút "Chạy ngay (toàn bộ)" gửi `POST /api/v1/masters/transaction_history_2/transmute?id=76` với `{ full_sync: true }`, sau đó hiển thị `Synced (${processed_rows} / ${total_rows})` từ `GET /api/v1/master-bindings/76/transmute-job-status`.
  - `centralized-data-service/internal/service/master/transmuter.go:214` & `1026-1037`: `TransmuterModule.Run` gọi `t.countShadowRows(ctx, masterRow, 0)` thực thi `SELECT COUNT(*) FROM "shadow_testcoretrans"."transaction_history" WHERE ("_gpay_id")::bigint > 0`.
  - Bằng chứng Runtime (`GET /api/v1/reconciliation/report` `id=538783`, `id=538785`, `id=538782`): Bảng nguồn `source_object_id=217` có `250,820` dòng, nhưng `binding_id=76` (`transaction_history_2`) đang trỏ vào cặp Shadow chưa backfill lịch sử là `shadow_testcoretrans.transaction_history` (`shadow_binding_id=222`, chỉ có `20,903` -> `20,909` dòng, thiếu `229,911` dòng) + `shadow_testcoretrans.transaction_history_payload` (`shadow_binding_id=237`). Trong khi đó `binding_id=75` (`transaction_history_1`) trỏ vào cặp Shadow đã heal đủ `250,820 / 250,820` dòng là `shadow_testcoretrans.transaction_history_1` (`shadow_binding_id=231`) + `shadow_testcoretrans.cdc_transaction_history_payload_1` (`shadow_binding_id=233`).
- **Logs / Test:**
  - Đã kiểm chứng qua `GET http://localhost:8083/api/v1/masters`, `GET /api/v1/reconciliation/report`, `GET /api/v1/activity-log` (`id=142943`, `id=142964`), `GET /api/v1/master-mapping-rules?master_binding_id=76`.
- **Definition of Done (DoD):**
  - Xác định chính xác 100% nguyên nhân gốc rễ kèm `file:line` và số liệu runtime thật.

### [x] TASK-02: Sửa lỗi tạo rule PK (`_id`) rỗng `source_field=""` / `data_type=""` khi clone Master Mapping Rules (`F1`)
- **Mô tả & Context `file:line`:**
  - Tại `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:67-80` (`cloneAllShadowRulesTx`), khi `pkCol` (ví dụ `"_id"` trên MongoDB Master `binding_id=76`) khác với `v2.target_column` (`"id"` trên PostgreSQL Primary Shadow `shadow_binding_id=222`), câu lệnh `INSERT INTO cdc_system.mapping_rule_master` đang chèn `mapping_v2_id = NULL` mà không điền `source_field` và `data_type`, sinh ra rule mồ côi `id=1862` (`target_column="_id"`, `source_field=""`, `data_type=""`).
  - Cập nhật `cloneAllShadowRulesTx` để tự động điền `source_field` (từ PK của Primary Shadow / `source_object_registry.primary_key_field`) và `data_type` (từ `mapping_rule_v2.data_type` của PK hoặc mặc định `'TEXT'`) cho rule PK self-contained (`mapping_v2_id = NULL`), đồng thời backfill nếu row PK đã tồn tại nhưng đang rỗng `source_field`/`data_type`.
- **Logs / Test:**
  - Đã chạy `go test -v -count=1 ./internal/infra/persistence/master/...` trong `cdc-cms-service` (`EV-20261102T131335-daf5`).
- **Definition of Done (DoD):**
  - Rule PK (`_id`) sinh ra trong `mapping_rule_master` luôn có đầy đủ `source_field="id"` và `data_type="TEXT"`, hiển thị đúng trên trang `/masters/transaction_history_2/mappings?binding_id=76` và không va chạm unique constraint khi chạy `syncRulesFromAllShadowsTx`.

### [x] TASK-03: Bổ sung cơ chế fallback `source_field` / `data_type` cho rule PK trong `TransmuterModule.loadRules` (`F2`)
- **Mô tả & Context `file:line`:**
  - Tại `centralized-data-service/internal/service/master/transmuter.go:918-955` (`loadRules`), khi đọc `cdc_system.mapping_rule_master` cho các bản ghi PK đã tồn tại có `mapping_v2_id IS NULL` và `source_field`/`data_type` rỗng (như `id=1862` của `binding_id=76`), `t.typeRes.Validate(r.DataType)` trả về `false` khiến rule `_id` bị drop (`invalid_data_type`), làm `resolveConflictTarget` (`transmuter_identity.go:85-98`) rơi về `_gpay_id` và gây insert trùng trên MongoDB khi nhận sự kiện CDC `UPDATE` (`transmuter_events_batch.go:169`).
  - Bổ sung chuẩn hóa trong `loadRules` (`transmuter.go:919-945`) và `normalizeMasterPKRule` (`transmuter_identity.go:130-155`): dùng `NULLIF(TRIM(m.source_field), '')` / `NULLIF(TRIM(m.data_type), '')` trong câu SQL và nếu rule khớp với PK (`strings.EqualFold(r.TargetColumn, parsePKFromSpec(row.TransformSpec))`) mà `r.SourceField == ""` hoặc `r.DataType == ""`, tự động fallback `r.SourceField` về `strings.TrimSpace(row.SourcePK)` (hoặc `r.TargetColumn`) và `r.DataType` về `"TEXT"`.
- **Logs / Test:**
  - Đã chạy unit test (`EV-20261102T131339-20f3`) và live integration test trên `binding_id=76` (`TestLiveBinding76_LoadRulesIncludesNormalizedMongoPK`, `EV-20261102T131346-b58c`).
- **Definition of Done (DoD):**
  - `loadRules` không bao giờ drop rule PK (`_id`) của `binding_id=76`, `resolveConflictTarget` trả về đúng `"_id"`, ngăn chặn triệt để lỗi nhân bản document (`diff = -7`) khi realtime CDC update.
