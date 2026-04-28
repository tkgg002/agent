| Timestamp | Operator | Model | Action / Status |
| --- | --- | --- | --- |
| 2026-04-25 00:00 | Codex | gpt-5 | Khởi tạo workspace `feature-cms-fe-overhaul` cho task audit và lập kế hoạch cải tổ `cdc-cms-web`. |
| 2026-04-25 00:00 | Codex | gpt-5 | Đã đọc route thật trong `App.tsx`, liệt kê toàn bộ page hiện có và rà API usage của từng page. |
| 2026-04-25 00:00 | Codex | gpt-5 | Đã đối chiếu FE routes với backend routes ở `cdc-cms-service` để phân loại page theo Keep / Merge / Remove. |
| 2026-04-25 00:00 | Codex | gpt-5 | Đã viết plan cải tổ FE theo flow vận hành V2 và xác định `CDCInternalRegistry` là page phải loại bỏ đầu tiên. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 2 FE refactor: sửa `App.tsx`, nhóm lại navigation thành `Setup / Operate / Advanced`, và bỏ `CDCInternalRegistry` khỏi menu runtime. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã đổi các label/operator text chính sang vocabulary V2 ở `SourceToMasterWizard`, `TableRegistry`, `SourceConnectors`, `MasterRegistry`, `ActivityManager`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Khi build thật đã phát hiện bug Ant Design `Steps.Step` trong `SourceToMasterWizard`; đã vá sang `items` API và build pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 3 FE semantics uplift cho `TableRegistry`: audit backend contract của `registry` và `masters` để giữ compatibility trong khi đổi UX sang `Source Objects`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm hiển thị `Shadow Target` theo convention `shadow_<source_db>.<table>` và đổi action copy/register modal cho operator hiểu đúng source object + shadow target. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã nối context từ `TableRegistry` sang `MasterRegistry` bằng query params và thêm compatibility notice vì backend vẫn nhận `source_shadow` theo contract legacy. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 4 FE semantics uplift cho `MappingFieldsPage` và `AddMappingModal`, tập trung đổi mental model từ `target_table` đơn lẻ sang `source object + shadow target`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm context alert, shadow schema/shadow target info card, và đổi toàn bộ copy chính của page sang `Mapping Rules`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật modal tạo rule để nói rõ backend legacy vẫn nhận `source_table` làm identity chính; build FE tiếp tục pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 5 practical uplift cho `ActivityManager` và `DataIntegrity`, tập trung làm rõ scope thao tác theo source object / shadow target thay vì chỉ target_table trần. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã enrich `ActivityManager` bằng metadata từ `/api/registry` để các schedule hiển thị `source_db.source_table -> shadow_<source_db>.<target_table>`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã enrich `DataIntegrity` overview và failed logs bằng source/shadow context từ recon report; build FE tiếp tục pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 6 Debezium-only pruning sau khi audit route/page/API surface để xác nhận phần nào đã lỗi thời với operating model hiện tại. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã loại `QueueMonitoring` khỏi primary navigation, redirect route cũ về `SystemHealth`, và cắt operation `bridge` / `airbyte-sync` khỏi UI Operations. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã bỏ các nhắc Airbyte/bridge first-class khỏi `SystemHealth`, `DataIntegrity`, `MappingFieldsPage`, `ActivityLog`; build FE pass sau khi vá regression `Typography.Text`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 7 backend API pruning sau khi audit call site FE và router CMS service để tìm route legacy không còn dùng. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã gỡ khỏi router/server các route `GET /api/v1/tables`, `PATCH /api/v1/tables/:name`, `POST /api/registry/:id/bridge`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật swagger/comment stale cho `refresh-catalog` và sửa retirement messages không còn trỏ về `/api/v1/tables`; `go test ./...` + `npm run build` đều pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 8 sau khi chốt lại rule 2 luồng: auto-flow là luồng chính, còn `cms-fe` giữ operator-flow cho monitoring / backup / retry / reconcile. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã audit `worker-schedule` API và xác nhận contract cũ quá nghèo: list chỉ trả `target_table`, create không resolve scope V2, khiến FE phải tự đắp nghĩa từ `/api/registry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `schedule_handler.go` để enrich `GET /api/worker-schedule` bằng source/shadow metadata từ `cdc_system`, đồng thời cho `POST /api/worker-schedule` nhận scope giàu hơn nhưng vẫn tương thích `target_table`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật swagger/comment cho `worker-schedule` list/create/update và refactor `ActivityManager` để dùng scope từ API cho list view. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 8 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 9 sau khi audit `master_registry_handler.go`, `MasterRegistry.tsx`, `TableRegistry.tsx` và xác nhận `v1/masters` vẫn là contract lệch kiến trúc lớn nhất. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã chuyển backend `v1/masters` sang `cdc_system.master_binding`, join `shadow_binding`, `source_object_registry`, `connection_registry` để trả master/source/shadow context đúng V2. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật swagger/comment cho `v1/masters` list/create/approve/reject/toggle/swap và giữ `source_shadow` như compatibility fallback. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `MasterRegistry.tsx` để operator nhập `master_schema`, `shadow_schema`, `shadow_table`; `TableRegistry` cũng deep-link thêm `shadow_schema` và `shadow_table`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 9 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 10 sau khi audit `mapping_rule_handler.go`, `MappingFieldsPage.tsx`, `AddMappingModal.tsx`, type definitions và xác nhận contract còn buộc vào `cdc_mapping_rules.source_table`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã chuyển backend `mapping-rules` sang `cdc_system.mapping_rule_v2`, join `source_object_registry` và `shadow_binding`, đồng thời giữ fallback `source_table` / `table` cho giai đoạn cutover. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật swagger/comment cho `mapping-rules` list/create/reload/update/backfill và đổi dispatch reload/backfill/batch update sang resolve theo `shadow_table`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `MappingFieldsPage`, `AddMappingModal`, `types/index.ts` để FE gửi source/shadow metadata thực khi list/create/reload. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 10 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 11 sau khi audit `activity_log_handler.go`, `ActivityLog.tsx`, `useAsyncDispatch.ts` và xác nhận monitoring surface vẫn còn nhìn system qua `target_table` legacy. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor backend `activity-log` để enrich source/shadow scope bằng join `cdc_system.shadow_binding` + `source_object_registry`, đồng thời thêm filter source/shadow metadata. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật swagger/comment cho `GET /api/activity-log` và `GET /api/activity-log/stats`; `recent_errors` giờ cũng trả enriched scope V2. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `useAsyncDispatch` để support `statusParams` và refactor `ActivityLog.tsx` render scope V2 + filter source DB. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 11 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 12 sau khi audit `reconciliation_handler.go`, `useReconStatus.ts`, `DataIntegrity.tsx` và xác nhận check/heal/report vẫn còn target-table-centric. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor backend reconciliation để enrich `report` và `failed-sync-logs` bằng source/shadow metadata V2, đồng thời thêm resolve scope từ `cdc_system.shadow_binding` + `source_object_registry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã reuse `POST /api/reconciliation/check` cho scoped single-check khi body có metadata, đồng thời thêm `POST /api/reconciliation/heal` generic nhưng vẫn giữ route path legacy. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `useReconStatus` và `DataIntegrity.tsx` để check/heal gửi `source_database`, `source_table`, `shadow_schema`, `shadow_table` và render scope `source/shadow` giàu nghĩa hơn. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 12 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 13 bằng audit usage của `QueueMonitoring`, `CDCInternalRegistry`, `CDCInternalRegistryHandler` để xác nhận chúng đã chết hoàn toàn ở runtime. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã xác nhận `CDCInternalRegistry.tsx` chỉ còn tự gọi các API đã bị remove, `QueueMonitoring.tsx` không còn route/menu, và backend handler không còn server wiring. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã purge vật lý ba artifact legacy ở FE và backend để giảm nhiễu kiến trúc và gỡ hẳn dấu vết runtime `cdc_internal` cũ. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 13 pass: grep không còn reference runtime sống; `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 14 bằng audit `/api/registry` theo call site thật ở FE, router backend và handler methods để phân loại live/transitional/dead. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã xác nhận `/api/registry` chưa thể xóa nguyên cụm vì vẫn đang gánh operator-flow thực chiến; dead group chủ yếu là `status`, `sync`, `jobs`, `scan-source`, `discover`, `drop-gin-index`, `sync/reconciliation`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã gỡ route dead khỏi router, xóa dead handler khỏi `registry_handler.go`, và đổi swagger/comment của nhóm còn sống sang semantics `Source Objects`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 14 pass: grep route dead không còn match; `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 15 bằng audit `failed-sync-logs/:id/retry` và downstream consumer `cdc.cmd.retry-failed` để xác định có cần đổi input contract sang scope-aware hay không. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã chốt rằng retry flow nên giữ `failed_log_id` làm identity chuẩn; thay vào đó enrich response và payload downstream bằng source/shadow metadata khi resolve được. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã cập nhật `reconciliation_handler.go` với swagger annotation + scope enrichment cho retry endpoint, và refactor `DataIntegrity` để failed logs ưu tiên render metadata trực tiếp từ bản ghi lỗi. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 15 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 16 bằng audit `SourceConnectors.tsx`, `system_connectors_handler.go`, `sources_handler.go` để tìm màn V2-native nào có thể dựng ngay từ API hiện có. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã xác nhận `GET /api/v1/system/connectors` + `GET /api/v1/sources` đủ để dựng dual-view screen cho runtime connector và source fingerprint mà chưa cần API mới. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `SourceConnectors.tsx` thành màn có summary cards, mismatch alert, tab `Connectors` và `Source Fingerprints`, trong khi vẫn giữ nguyên các destructive runtime actions. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 16 pass: `npm run build` ở `cdc-cms-web` thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 17 bằng audit namespace drift trong CMS backend sau khi phát hiện `Source` và `WizardSession` vẫn còn trỏ `cdc_internal` dù migration end-state đã move bảng sang `cdc_system`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã sửa `Source.TableName()`, `WizardSession.TableName()` và `WizardRepo.AppendProgress()` sang `cdc_system.*`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 17 pass: grep không còn `cdc_internal.sources` / `cdc_internal.cdc_wizard_sessions` trong `internal/`; `go test ./...` ở `cdc-cms-service` thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 18 bằng audit read dependencies của `TableRegistry` và xác nhận điểm nghẽn chính là list view vẫn lấy từ `/api/registry` dù metadata V2 thật đã nằm ở `cdc_system`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm `GET /api/v1/source-objects` đọc từ `cdc_system.source_object_registry` + `shadow_binding`, enrich reconciliation status và carry `registry_id` bridge legacy khi có. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `TableRegistry.tsx` sang read path V2, đồng thời disable minh bạch các action legacy khi row chưa có `registry_id` bridge. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 18 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử `make swagger`; annotations đã có nhưng generated docs chưa regenerate được do môi trường thiếu binary `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 19 bằng audit `shadow_binding` surface và chốt rằng nên dựng dual-view trong chính `Source Objects` thay vì mở thêm page mới. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm `GET /api/v1/shadow-bindings` đọc từ `cdc_system.shadow_binding` + `source_object_registry`, enrich drift và `registry_id` bridge nếu có. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `TableRegistry.tsx` thành dual-view screen với tab `Source Objects` và `Shadow Bindings`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 19 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 20 bằng audit `MappingFieldsPage` và xác nhận dependency xấu nhất là `GET /api/registry` để tải cả danh sách rồi tự tìm row. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm `GET /api/v1/source-objects/registry/{registry_id}` làm bridge-aware read model cho page mappings. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `MappingFieldsPage.tsx` sang endpoint mới và giữ `create-default-columns` trên legacy path để không gãy operator-flow. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 20 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 21 bằng audit `detect-timestamp-field`, `scan-fields`, `standardize`, `create-default-columns` và xác nhận chúng vẫn thật sự cần `registry_id` ở worker/backend semantics. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm facade handler `source_object_actions_handler.go` với namespace `/api/v1/source-objects/registry/:id/...` để FE không còn gọi trực tiếp `/api/registry` cho các action này. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã refactor `useRegistry`, `ReDetectButton`, `TableRegistry`, `MappingFieldsPage` sang facade endpoints mới. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 21 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 22 bằng audit call-sites còn trỏ trực tiếp `/api/registry` và chốt rằng `transform-status` là phần thực chiến còn sót đáng facad hóa tiếp. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm facade `GET /api/v1/source-objects/registry/:id/transform-status` và chuyển `TableRegistry` sang endpoint mới. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 22 pass: `go test ./...` ở `cdc-cms-service` và `npm run build` ở `cdc-cms-web` đều thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 23 bằng audit các compatibility shell còn sót; xác nhận `Dashboard` và `ActivityManager` là 2 read surfaces nên có thể rời `/api/registry` trước một cách an toàn. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm `GET /api/v1/source-objects/stats` đọc từ `cdc_system.source_object_registry` + `shadow_binding` + bridge priority/created fallback. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã chuyển `Dashboard.tsx` sang stats endpoint V2 và đổi copy từ `Registered Tables` sang `Source Objects`, `Tables Created` sang `Shadow Ready`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã chuyển `ActivityManager.tsx` sang `GET /api/v1/source-objects` để render/create schedule scope bằng source/shadow metadata V2 thay vì fetch `/api/registry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 23 pass: `go test ./...` ở `cdc-cms-service` pass sau khi rerun ngoài sandbox vì Go build cache bị chặn; `npm run build` ở `cdc-cms-web` pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã grep lại call-sites: `Dashboard.tsx` không còn `/api/registry/stats`, `ActivityManager.tsx` không còn fetch `/api/registry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`, nhưng source annotations đã được cập nhật cùng phase. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 24 bằng audit `POST /api/registry`, `PATCH /api/registry/:id`, `POST /api/registry/batch`; xác nhận đây vẫn là compatibility mutations thật, nhưng FE-facing namespace nên được dọn tiếp. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm 3 facade mutation dưới `/api/v1/source-objects` và chuyển `TableRegistry.tsx` sang namespace mới cho single register, bulk import, update bridged row. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 24 pass: `go test ./...` ở `cdc-cms-service` pass, `npm run build` ở `cdc-cms-web` pass, grep FE runtime không còn `/api/registry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thử lại `make swagger`; generated docs vẫn chưa regen được do thiếu binary `swag`, nhưng source annotations cho 3 mutation facade đã được cập nhật. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 25 bằng audit router `/api/registry...`; xác nhận FE/runtime nội bộ không còn caller nào tới nhóm route legacy này. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm facade `POST /api/v1/source-objects/registry/:id/transform` và gỡ toàn bộ route `/api/registry...` đã có replacement khỏi router. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 26 bằng audit `registry_handler.go`; xác nhận source comments vẫn còn nhiều `@Router /api/registry...` dù router đã bị prune. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã đổi các swagger blocks legacy trong `RegistryHandler` thành comment delegate nội bộ để source annotations khớp router mới. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 25 pass: `go test ./...` ở `cdc-cms-service` pass, `npm run build` ở `cdc-cms-web` pass, grep router legacy `/registry...` rỗng. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 26 pass: grep `@Router /api/registry` trong `registry_handler.go` rỗng; `go test ./...` pass; `make swagger` vẫn fail do thiếu `swag`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 27 bằng audit write-path `Register/Update/BulkRegister` và schema V2; chốt hướng `legacy gatekeeper, V2 sync follower`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm `SourceObjectV2SyncService` và cắm sync vào `RegistryHandler.Register/Update/BulkRegister` để upsert `source_object_registry` + `shadow_binding` sau write thành công. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 27 pass: `go test ./...` ở `cdc-cms-service` pass sau khi sửa blocker `gorm.io/datatypes` bằng cách dùng `[]byte` JSON payload. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 28 bằng audit `GET /api/v1/source-objects`; chốt rằng cần hiển thị rõ `metadata_status` + `bridge_status` để operator phân biệt `V2 Ready` với `No Bridge`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã bổ sung status vào read model và render badge mới trong `TableRegistry`. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 28 pass: `go test ./...` ở `cdc-cms-service` pass; `npm run build` ở `cdc-cms-web` pass. |
| 2026-04-27 00:00 | Codex | gpt-5 | Bắt đầu Phase 29 bằng audit update-path; chốt rằng row `V2-only` cần mutation trực tiếp theo `source_object_id`, nhưng chỉ cho field có V2 home rõ ràng. |
| 2026-04-27 00:00 | Codex | gpt-5 | Đã thêm direct patch route `PATCH /api/v1/source-objects/:id` và chuyển `TableRegistry` sang chọn endpoint theo trạng thái bridge. |
| 2026-04-27 00:00 | Codex | gpt-5 | Verify Phase 29 pass: `go test ./...` ở `cdc-cms-service` pass; `npm run build` ở `cdc-cms-web` pass. |
## 2026-04-27 — Phase 30 V2 Direct Re-detect
- Audit xong `detect-timestamp-field` và xác nhận worker hiện support fallback theo `target_table`, nên có thể tách action này khỏi `registry_id` trước.
- Thêm direct CMS routes:
  - `GET /api/v1/source-objects/{id}/dispatch-status`
  - `POST /api/v1/source-objects/{id}/detect-timestamp-field`
- Enrich `reconciliation report` với `source_object_id` để `DataIntegrity` gọi direct route mới.
- Refactor `ReDetectButton` ưu tiên `source_object_id`, fallback `registry_id`.
- Verify:
  - `go test ./...` pass ở `cdc-cms-service`
  - `npm run build` pass ở `cdc-cms-web`
- Note: có một lần chạy sai `gofmt` lên file `.ts/.tsx`; không ảnh hưởng code, đã được chặn lại bằng verify build thật.
## 2026-04-27 — Phase 31 Direct Scan Fields & Transform Status
- Audit xác nhận:
  - `transform-status` direct V2 hóa được
  - `scan-fields` direct V2 hóa được
  - `create-default-columns` chưa đủ an toàn để direct hóa
- Thêm direct CMS routes:
  - `POST /api/v1/source-objects/{id}/scan-fields`
  - `GET /api/v1/source-objects/{id}/transform-status`
- Refactor FE:
  - `useRegistry.ts` ưu tiên direct scan
  - `TableRegistry.tsx` ưu tiên direct transform status
- Verify:
  - `go test ./...` pass
  - `npm run build` pass
- Regression đã bắt và sửa:
  - thiếu import `fmt`
  - biến TS không còn dùng
## 2026-04-28 — Phase 32 Direct Standardize
- Audit xác nhận `standardize` direct-V2 hóa an toàn vì worker path chỉ cần `target_table`.
- Thêm route:
  - `POST /api/v1/source-objects/{id}/standardize`
- Refactor `TableRegistry` để `Tạo Field MĐ` ưu tiên direct V2, fallback bridge.
- Verify:
  - `go test ./...` pass
  - `npm run build` pass
- Kết luận: `create-default-columns` vẫn chưa nên direct hóa trong phase này.
## 2026-04-28 — Phase 33 Schema-aware Create Default Columns
- Audit phát hiện root cause lớn: worker `create-default-columns` và một phần `standardize` đang assume `public.<target_table>`.
- Đã mở direct route:
  - `POST /api/v1/source-objects/{id}/create-default-columns`
- Đã sửa worker:
  - nhận `shadow_schema`
  - chạy schema-aware path
  - update `cdc_system.shadow_binding.ddl_status='created'`
- FE:
  - `TableRegistry` dùng direct route cho row V2-only
  - `MappingFieldsPage` dùng direct route khi có `source_object_id`
- Verify pass:
  - CMS tests
  - worker tests
  - FE build
## 2026-04-28 — Phase 34 Shadow Schema Runtime Hardening
- Audit phát hiện `SchemaValidator`, `batch-transform`, `scan-raw-data`, `periodic-scan`, `drop-gin-index`, `scan-fields` vẫn còn assumption `public`.
- Đã thêm lookup `ResolveTargetRoute(target_table)` để worker resolve `shadow_schema` từ V2 metadata.
- Đã sửa worker:
  - `SchemaValidator` introspect theo schema đã resolve
  - `CommandHandler` qualify SQL theo `schema.table` cho các operator path chính
  - `DROP INDEX` qualify theo shadow schema thay vì search path mặc định
- Verify pass:
  - `gofmt`
  - `go test ./internal/service ./internal/handler ./internal/server`
## 2026-04-28 — Phase 35 Runtime Schema Follow-through
- Audit phát hiện `HandleDiscover`, `HandleBackfill` và `SchemaInspector` vẫn còn đuôi assumption `public`.
- Đã thêm `GetTableColumnsInSchema()` cho `PendingFieldRepo`.
- Đã inject metadata vào `SchemaInspector` để resolve cache/schema theo `shadow_schema`.
- Đã sửa worker:
  - `HandleDiscover` introspect theo schema đã resolve
  - `HandleBackfill` update theo `schema.table`
- Verify pass:
  - `gofmt`
  - `go test ./internal/service ./internal/handler ./internal/server ./internal/repository`
## 2026-04-28 — Phase 36 Schema Tail Cleanup
- Audit xác nhận:
  - `event_bridge` có poll path nên nên harden schema-aware
  - `transform_service` hiện chưa có caller runtime nhưng vẫn nên chuẩn bị đường resolve schema
  - `cms registry_repo` cần helper `...InSchema()` để compatibility shell không khóa cứng `public`
- Đã sửa:
  - `event_bridge` query theo `schema.table`
  - `transform_service` hỗ trợ metadata-aware schema resolution
  - `cms registry_repo` có wrappers schema-aware cho scan/backfill/get-columns
- Verify pass:
  - `gofmt`
  - worker go tests
  - CMS go tests
## 2026-04-28 — Phase 37 Dead-code Prune & Deprecate (PARTIAL — session bị block)
- Quyết định dứt điểm thay vì thêm lớp đệm:
  - `TransformService`: dead code, không có caller → đã prune (xóa `transform_service.go`, -112 dòng).
  - Helper raw SQL trong CMS `registry_repo` không có caller → đã prune (-49 dòng).
  - `EventBridge`: giữ làm compatibility reserve (còn test + giá trị nếu poller quay lại) → đóng dấu rõ không thuộc runtime chính (+11/-3).
- Tổng: 3 files changed, +11/-164.
- ⚠️ Session bị hit usage limit lúc 8:51 AM trước khi kịp tạo bộ docs `01..09_phase37_*`. Đề xuất Phase 38 tạo hồi tố.

## 2026-04-28 — Consolidated Status Report (Phase 16 → Phase 37)
- Đã tổng hợp toàn bộ tiến trình từ `cdc-system/Untitled-2.ini` (2092 dòng) thành 1 report duy nhất:
  - File: `07_status_consolidated_phase16_to_phase37.md`
- Nội dung report bao gồm: bảng tổng hợp 22 phase, chi tiết từng phase, files-changed stats, chiến lược kỹ thuật xuyên suốt, risks/gaps, đề xuất Phase 38+, verification matrix.
- Operator: Muscle (CC CLI) — Claude Opus 4.7.

## 2026-04-28 — Phase 37 Docs hồi tố + Re-verify
- Đã tạo bộ docs Phase 37 đầy đủ (hồi tố do session gốc bị block):
  - `01_requirements_phase37_dead_code_prune.md`
  - `02_plan_phase37_dead_code_prune.md`
  - `03_implementation_phase37_dead_code_prune.md`
  - `06_validation_phase37_dead_code_prune.md`
  - `08_tasks_phase37_dead_code_prune.md`
  - `09_tasks_solution_phase37_dead_code_prune.md`
- Re-verify code:
  - `transform_service.go` đã xóa hoàn toàn ✓
  - Methods legacy `ScanRawKeys`/`PerformBackfill`/`GetDBColumns` không còn trong `registry_repo.go` ✓
  - `grep TransformService` trong centralized-data-service/internal: 0 hits ✓
  - `go build ./...` trong cdc-cms-service: pass ✓
  - `go build ./...` trong centralized-data-service: pass ✓
- Phase 37 chính thức đóng kín.
- Operator: Muscle (CC CLI) — Claude Opus 4.7.

## 2026-04-28 — Start 4 service + Fix Kafka empty-topic panic
- Audit trạng thái: cdc-auth-service (8081) và cdc-cms-web (5173) đã chạy sẵn; cần start cdc-cms-service (8083) và centralized-data-service (8082).
- Verify infra docker đầy đủ: postgres/nats/redis/kafka/mongo.
- Started:
  - `cdc-cms-service` (port 8083) → log `/tmp/cdc-cms.log`. LISTEN ✓ health=ok.
  - `centralized-data-service` worker (port 8082) → log `/tmp/cdc-worker.log`.
- **Bug bắt được khi start worker lần 1**:
  - `kafka_consumer.go:118` log "no kafka topics found, will retry periodically" nhưng vẫn fall-through xuống `kafka.NewReader` → panic `either Topic or GroupTopics must be specified with GroupID`.
  - Root cause: Log claim không khớp behavior.
  - Fix minimal-impact: thêm retry loop `time.Ticker(60s)` + `ctx.Done()` cancel; chỉ tạo reader khi `len(topics) > 0`.
  - Đã append lesson vào `agent/memory/global/lessons.md`.
- Verify final:
  - `lsof -iTCP:8081/8082/8083/5173 -sTCP:LISTEN` → 4/4 LISTEN ✓
  - `curl /health`: 8081 cdc-auth=ok, 8082 cdc-worker=ok, 8083 cdc-cms=ok, 5173 vite=200 ✓
- Files changed:
  - `cdc-system/centralized-data-service/internal/handler/kafka_consumer.go` (+13/-0)
- Operator: Muscle (CC CLI) — Claude Opus 4.7.


## 2026-04-28 — Phase 38 closed (Muscle audit re-run)

- Search_path migration 039 áp dụng (`ALTER ROLE "user" SET search_path = cdc_system, public;`).
- 4 file CMS được patch:
  - `internal/api/schema_proposal_handler.go` — 7× `cdc_internal.schema_proposal` → `cdc_system.schema_proposal`.
  - `internal/api/transmute_schedule_handler.go` — 4× rename schema + List/RunNow rewrite Raw JOIN `master_binding`.
  - `internal/api/source_objects_handler.go` — xóa 2 ref `tr.sync_status` + 3 ref `tr.recon_drift`.
  - `internal/api/schedule_handler.go` — flat scan `workerScheduleScanRow` (22 field) + transpose ra `WorkerScheduleResponse{Scope: …}`.
- Build cả 2 service `go build ./...` → exit 0.
- Auth login đã verify: `POST /api/auth/login` body `{"username":"admin","password":"admin123"}` (KHÔNG dùng email).
- Operator-flow: 11/11 endpoints curl thành công (200) — full body snapshot ở `06_validation_phase38_*.md`.
- Auto-flow: Debezium `goopay-mongodb-cdc` RUNNING, 4 topic `cdc.goopay.*`, worker discovery loop healthy KHÔNG panic. Mismatch dữ liệu giữa `collection.include.list` (9 collection) và `source_object_registry` (1 row debezium tên `payments`) → 0 topic được consume hiện tại — DEBT, không phải bug code.
- Phase 38 docs đầy đủ: 01_requirements / 02_plan / 03_implementation / 06_validation / 08_tasks / 09_tasks_solution. Đúng prefix CLAUDE.md.

---

## 2026-04-28 — Phase 39 START (Schema Consolidation)

**Trigger từ user**:
1. "kiểm tra lại rule các table. toàn bộ table của hệ thống ở cdc_system, ko 1 table nào nằm ngoài schema này"
2. Pointer runbook: `cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md`
3. "auth_users, admin_actions ..." — yêu cầu phân loại từng public table
4. **Reprimand** mid-session: Brain đề xuất giữ `auth_users` ở public với lý lẽ "non-CDC bounded context" → user phẫn nộ "use_auth ko phải để quản lý à. mày ngu mà thích nói chuyện lý lẽ à". Brain ghi lesson + chuyển hướng.
5. Quyết định cuối: tạo schema riêng `cdc_auth_service` cho auth-service (vẫn microservice riêng). public phải rỗng hoàn toàn.

**Brain output (Phase 39 START)**:
- Workspace docs: 01/02/03/06/08/09 prefix `phase39_schema_consolidation`
- 3 migration drafts: `040_admin_actions_in_cdc_system.sql`, `041_cdc_alerts_in_cdc_system.sql`, `042_search_path_with_auth.sql`
- 1 migration rewrite: `cdc-auth-service/migrations/001_auth_users.sql` (CREATE SCHEMA cdc_auth_service)
- 3 code patches: user.go, alert.go, audit.go (qualify schema)
- 1 wipe script update: `wipe_cdc_runtime_v2.sql` → DROP SCHEMA public CASCADE
- Lesson đã append: Brain cãi rule user bằng "lý lẽ exception" (mid-session correction)

**Status**: ⏳ Awaiting user approval trước Muscle exec wipe (irreversible).

