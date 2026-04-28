| Timestamp | Operator | Model | Action / Status |
| --- | --- | --- | --- |
| 2026-04-24 00:00 | Codex | gpt-5 | Khởi tạo workspace `feature-cdc-schema-design-v2` cho task thiết kế schema V2 của CDC system. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã đọc memory bắt buộc và rà soát code thực tế tại `cdc-system/centralized-data-service` để xác định root cause hiện trạng "đổ đống". |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã viết bộ requirements, plan, implementation notes, decisions, validation, status, tasks và solution proposal cho schema design V2. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt đầu Phase 1 implementation: scaffold migration V2, model V2 và repository V2 trong `centralized-data-service`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm migrations `029`-`035` cho `cdc_system` cùng backfill bootstrap từ registry legacy. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm model/repository V2 cho connection, source object, shadow binding, master binding, mapping rule v2, sync runtime state. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã chạy `gofmt` và verify package-level bằng `go test ./internal/model ./internal/repository` thành công. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã chạy `go test ./...`; quan sát được nhiều package pass, chưa thấy fail nhưng session chưa kết thúc dứt khoát trong cửa sổ chờ hiện tại. |
| 2026-04-24 00:00 | Codex | gpt-5 | Mở rộng config để support `CDC_SYSTEM_DB_URL` dạng single và `CDC_SHADOW_DB_URLS`/`CDC_MASTER_DB_URLS` dạng multi-key, đồng thời fix bug `DB_SINK_URL` trước đây chưa thực sự được dùng làm DSN. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm `internal/service/connection_manager.go` để cache system/shadow/master DB connections theo key, dùng trực tiếp từ config multi-db mới. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 2: thêm `MetadataRegistryService` đọc V2 source/shadow tables, giữ compatibility mapping rules legacy, và chuyển `EventHandler`/`DynamicMapper` sang metadata interface. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 2 pass: `go test ./internal/service ./internal/handler ./internal/server`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 3: nâng write path để record/batch/schema adapter hiểu `connection key + schema + table`, đồng thời nối `ConnectionManager` vào ingest path. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt được regression compatibility ở `SchemaAdapter` cache sau refactor schema-aware và đã vá bằng dual-key cache (`schema.table` + `table`). |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 3 pass: `go test ./internal/service ./internal/handler ./internal/server`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 4: chuyển transmuter/master DDL runtime sang V2 bindings + ConnectionManager, giữ compatibility NATS payload hiện tại. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 4 pass: `go test ./internal/service ./internal/handler ./internal/server`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 5: master path auto-ensure destination trước khi transmute, đồng thời ghi `cdc_system.sync_runtime_state` cho master DDL + transmute runtime. |
| 2026-04-24 00:00 | Codex | gpt-5 | Mở rộng payload `cdc.cmd.transmute-shadow` với `shadow_schema` + `shadow_connection_key`, và cập nhật handler lookup theo identity-aware route trước khi fallback theo `shadow_table`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 5 pass: `go test ./internal/service ./internal/handler ./internal/server` và `go test ./internal/sinkworker`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 6: dời transmute scheduler metadata sang `cdc_system.transmute_schedule` và gắn trực tiếp với `master_binding_id`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm migration `036_v2_transmute_schedule.sql`, model/repository schedule V2 và đổi `TransmuteScheduler` sang poll bảng mới với guard `master_binding` active/approved. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 6 pass: `go test ./internal/service ./internal/server ./internal/repository ./internal/model`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Refactor phase 7: nối `MetadataRegistryService` vào `command/recon/backfill/full-count/schema validation/recon core` để runtime ưu tiên metadata V2 thay cho lookup trực tiếp từ `cdc_table_registry`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 7 pass: `go test ./internal/service ./internal/handler ./internal/server`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã viết checklist cutover `wipe & bootstrap` và gap analysis cho trạng thái sẵn sàng chuyển sang V2 metadata. |
| 2026-04-24 00:00 | Codex | gpt-5 | Audit phase 9 xác nhận vẫn còn system table references rải rác ngoài `cdc_system`; user chốt rule mới: chỉ shadow/master physical tables được phép nằm ngoài `cdc_system`, shadow naming dùng `shadow_<source_db>`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã dời runtime function calls từ `cdc_internal` sang `cdc_system` cho sinkworker fencing/heartbeat/claim và master RLS helper. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã cập nhật integration tests sang `cdc_system.*` cho activity log, failed sync logs và table registry. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm migration `038_finalize_cdc_system_namespace.sql` để move sequence/function cuối cùng sang `cdc_system` và drop schema `cdc_internal`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã cập nhật grant trong `005_pg_users.sql` để các role runtime có `USAGE/EXECUTE` trên `cdc_system`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Verify phase 9 pass: audit runtime không còn `cdc_internal.*`; `go test ./internal/service ./internal/handler ./internal/server`, `go test ./internal/sinkworker`, `go test ./internal/service ./internal/server ./internal/repository ./internal/model` đều pass. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt đầu phase 10: tạo bootstrap seed SQL riêng cho `cdc_system`, phục vụ đợt wipe & bootstrap mà không đưa dữ liệu môi trường vào migration chain. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm `deployments/sql/bootstrap_cdc_system_v2_template.sql` với flow mẫu đầy đủ: connection -> source object -> shadow binding -> master binding -> mapping_rule_v2 -> transmute_schedule. |
| 2026-04-24 00:00 | Codex | gpt-5 | Tự rà lại seed template và vá conflict handling của `mapping_rule_v2` sang `ON CONFLICT DO NOTHING` để tránh mismatch với expression unique index hiện tại. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt đầu phase 11: tạo runbook wipe/bootstrap và vá Makefile migrate để không còn chỉ chạy `001_init_schema.sql`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã cập nhật `Makefile` với target `migrate` chạy toàn bộ `migrations/*.sql` theo lexical order và thêm `migrate-bootstrap` để seed template V2 sau migrate. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm runbook `deployments/runbooks/wipe_bootstrap_v2.md` mô tả thứ tự stop, backup, wipe, migrate, seed, restart và checklist verify sau cutover. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt đầu phase 12: tạo wipe SQL script riêng cho V2 để drop master/shadow đúng thứ tự rồi mới truncate control-plane metadata. |
| 2026-04-24 00:00 | Codex | gpt-5 | Đã thêm `deployments/sql/wipe_cdc_runtime_v2.sql` với flow wipe: drop master physical tables, drop schema `shadow_%`, cleanup legacy public system tables, truncate `cdc_system`, drop `cdc_internal`. |
| 2026-04-24 00:00 | Codex | gpt-5 | Bắt được mismatch trong runbook verify (`transmute_status` không tồn tại trong `sync_runtime_state`) và đã sửa query về đúng schema thật. |
