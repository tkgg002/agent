# Gap Analysis

## Khoảng cách giữa FE hiện tại và kiến trúc V2

1. FE vẫn expose `cdc_internal` như một concept first-class
2. FE vẫn bám mạnh vào `/api/registry` và `target_table`
3. FE route tree chưa tổ chức theo lifecycle vận hành V2
4. Nhiều page trùng trách nhiệm hoặc chồng lên nhau

## Blocker lớn nhất nếu refactor ngay

1. Một số API backend vẫn còn transitional/legacy
2. `TableRegistry` hiện đang ôm nhiều semantics cũ
3. Chưa có màn V2-native cho `connection_registry`, `source_object_registry`, `shadow_binding`, `master_binding`

## Cách xử

1. bỏ page trái kiến trúc trước
2. nhóm lại menu
3. thay từng page dữ liệu theo V2 model

## Residual Gap sau Phase 2

1. `TableRegistry` mới đổi nhãn, chưa chuyển data contract sang `source_object_registry` thật.
2. `MasterRegistry` vẫn dùng field/API legacy như `source_shadow`.
3. `MappingFieldsPage` chưa được đổi tên và chưa bind sang `mapping_rule_v2`.
4. `QueueMonitoring` và `ActivityManager` vẫn còn overlap trách nhiệm vận hành.
5. File `CDCInternalRegistry.tsx` còn nằm trong codebase như artifact legacy, dù đã bị rút khỏi runtime navigation.

## Residual Gap sau Phase 3

1. `TableRegistry` đã hiển thị shadow namespace đúng hơn, nhưng data source vẫn đến từ `/api/registry` legacy.
2. `MasterRegistry` đã có contextual UX tốt hơn, nhưng submit path `/api/v1/masters` vẫn ghi vào `cdc_internal.master_table_registry`.
3. `MappingFieldsPage` vẫn đang dùng `registry.target_table` làm trục chính của mọi preview/introspection action.
4. `ActivityManager`, `DataIntegrity`, `ActivityLog` vẫn dùng vocabulary `target_table` gần như nguyên trạng.
5. FE hiện nói đúng hơn về V2, nhưng backend CMS chưa được cải tổ đồng bộ theo metadata V2 thật.

## Residual Gap sau Phase 4

1. `MappingFieldsPage` đã đổi mental model cho operator, nhưng network calls vẫn dựa vào:
   - `/api/mapping-rules`
   - `/api/registry/:id/create-default-columns`
   - `/api/introspection/scan[-raw]/:target_table`
2. `ActivityManager` vẫn quản lý schedule theo `target_table` string, chưa có source/shadow/master context.
3. `DataIntegrity` và `ActivityLog` vẫn phản ánh strongly-legacy model ở level bảng đích.
4. `MasterRegistry` frontend đã nói rõ contract legacy, nhưng backend `master_registry_handler` vẫn còn gắn chặt `cdc_internal.master_table_registry`.

## Residual Gap sau Phase 5

1. `ActivityManager` frontend đã enrich scope, nhưng backend `/api/worker-schedule` vẫn chỉ lưu `target_table`.
2. `DataIntegrity` frontend đã enrich scope, nhưng mutation endpoints `/api/reconciliation/*` và `/api/failed-sync-logs/:id/retry` vẫn nhận table string legacy.
3. `ActivityLog` và `QueueMonitoring` chưa được practical uplift tương tự.
4. `MasterRegistry`, `MappingFieldsPage`, `ActivityManager` vẫn còn bị chặn bởi backend CMS chưa migrate sang metadata V2 thật.

## Residual Gap sau Phase 6

1. FE đã prune phần thừa chính, nhưng backend router vẫn còn mount route legacy như:
   - `/api/registry/:id/bridge`
   - `/api/v1/tables`
2. `ActivityLog` filter đã được rút gọn, nhưng dữ liệu log backend vẫn có thể chứa operation legacy.
3. `QueueMonitoring.tsx` và `CDCInternalRegistry.tsx` vẫn còn file vật lý trong repo, dù không còn là runtime navigation chính.
4. Muốn thật sự "không dư thừa API" thì phase tiếp theo phải dọn backend CMS, không chỉ FE.

## Residual Gap sau Phase 7

1. Router đã gọn hơn, nhưng handler/file legacy vẫn còn vật lý:
   - `cdc_internal_registry_handler.go`
   - `QueueMonitoring.tsx`
   - `CDCInternalRegistry.tsx`
2. `worker-schedule`, `masters`, `mapping-rules` vẫn còn contract legacy ở cấp dữ liệu.
3. Swagger vẫn còn nhiều mô tả nói theo `CDC table` / `registry` semantics cũ, dù route đã bớt thừa hơn trước.

## Residual Gap sau Phase 8

1. `worker-schedule` đã giàu context hơn cho operator-flow, nhưng persistence layer vẫn còn lưu `target_table` legacy trong `cdc_worker_schedule`.
2. `v1/masters` backend vẫn ghi/đọc trực tiếp `cdc_internal.master_table_registry`, chưa chuyển sang `cdc_system.master_binding`.
3. `mapping-rules` backend vẫn lấy `source_table` làm identity lọc/chạy reload/backfill.
4. `activity-log` API vẫn lọc chủ yếu theo `target_table`, chưa expose filter/source scope theo V2 metadata.
5. File legacy vật lý vẫn còn trong repo và cần purge sau khi contract-level refactor hoàn tất.

## Residual Gap sau Phase 9

1. `v1/masters` đã chuyển sang `cdc_system.master_binding`, nhưng approve/toggle/swap vẫn còn assume `master_table` là unique string ở cấp route.
2. `mapping-rules` backend vẫn lấy `source_table` làm identity chính cho list/reload/backfill.
3. `activity-log` API vẫn lọc theo `target_table`, chưa phản ánh source/shadow/master scope theo V2.
4. `DataIntegrity` mutation endpoints vẫn còn nhận `:table` string legacy.
5. File legacy vật lý vẫn còn trong repo và cần purge sau khi các API còn lại được refactor xong.

## Residual Gap sau Phase 10

1. `mapping-rules` đã chuyển sang `mapping_rule_v2`, nhưng command payload downstream vẫn còn dùng `target_table` string legacy vì worker side chưa được refactor hết.
2. `activity-log` API vẫn lọc theo `target_table`, chưa expose source/shadow/master-aware filtering.
3. `DataIntegrity` mutation endpoints (`check/:table`, `heal/:table`) vẫn còn nhận table string legacy.
4. `useAsyncDispatch` và một số hook FE vẫn assume `target_table` là tham số trung tâm.
5. File legacy vật lý vẫn còn trong repo và cần purge sau khi operational APIs còn lại được refactor xong.

## Residual Gap sau Phase 11

1. `activity-log` đã enrich scope V2, nhưng nguồn dữ liệu vật lý vẫn là `cdc_activity_log` với cột `target_table` legacy.
2. `DataIntegrity` mutation endpoints (`check/:table`, `heal/:table`) vẫn còn nhận table string legacy.
3. `useReconStatus` và một số flow reconciliation vẫn dùng `target_table` làm identity trung tâm.
4. `reconciliation/report` và `failed-sync-logs` chưa được enrich contract đầy đủ như `activity-log`.
5. File legacy vật lý vẫn còn trong repo và cần purge sau khi operational APIs còn lại được refactor xong.

## Residual Gap sau Phase 12

1. Reconciliation API đã scope-aware hơn, nhưng persistence/reporting vật lý vẫn còn xoay quanh `target_table` ở một số bảng legacy.
2. `failed-sync-logs/:id/retry` vẫn là action theo log ID, chưa mang scope-aware contract rõ ràng như check/heal.
3. Downstream worker/NATS payload ở một số chỗ vẫn target-table-centric, nên operator API hiện vẫn cần compatibility fallback.
4. Một số route path legacy (`check/:table`, `heal/:table`) vẫn còn tồn tại để giữ cutover an toàn.
5. File FE/BE legacy vật lý vẫn chưa purge hết sau khi contract-level refactor đã hoàn tất phần chính.

## Residual Gap sau Phase 13

1. Artifact FE/BE legacy chính đã bị purge, nhưng persistence layer của một số flow vẫn còn dùng `target_table` làm cột trung tâm.
2. `failed-sync-logs/:id/retry` vẫn chưa được enrich contract theo source/shadow scope, dù action theo ID đã bớt mơ hồ hơn check/heal.
3. `/api/registry` vẫn còn là compatibility surface lớn và chưa được thay thế hoàn toàn bằng các màn V2-native (`connections`, `source objects`, `bindings`).
4. Một số route path legacy vẫn đang được giữ vì cutover safety, chưa phải end-state tối giản cuối cùng.

## Residual Gap sau Phase 14

1. `/api/registry` đã gọn hơn, nhưng vẫn là compatibility surface lớn chứ chưa phải V2-native API cuối cùng.
2. Persistence/reporting của nhiều flow vẫn còn xoay quanh `target_table`, nên operator APIs vẫn phải carry fallback legacy.
3. `failed-sync-logs/:id/retry` chưa có contract scope-aware rõ như check/heal.
4. FE vẫn chưa có màn quản trị V2-native riêng cho `connection_registry`, `source_object_registry`, `shadow_binding`, `master_binding`.

## Residual Gap sau Phase 15

1. Retry flow đã rõ nghĩa hơn, nhưng persistence/reporting lõi của failed logs vẫn còn dựa nhiều vào `target_table`.
2. Compatibility surfaces như `/api/registry` và một phần `/api/v1/masters` vẫn còn cần thiết vì FE chưa có màn V2-native riêng.
3. FE chưa có IA/screen model riêng cho:
   - connections
   - source objects V2
   - shadow bindings
   - master bindings
4. Một số payload downstream trong worker path vẫn giữ `target_table` là trường trung tâm để đảm bảo cutover an toàn.

## Residual Gap sau Phase 16

1. FE đã có màn dual-view cho runtime connector và source fingerprint, nhưng `v1/sources` backend vẫn đang bám `cdc_internal.sources`.
2. Chưa có màn V2-native riêng cho:
   - `source_object_registry`
   - `shadow_binding`
   - `master_binding`
3. `/api/registry` vẫn là compatibility surface lớn cho source-object/shadow maintenance.
4. Persistence/reporting và một số worker payload vẫn còn target-table-centric.

## Residual Gap sau Phase 17

1. `Source` và `WizardSession` đã về `cdc_system`, nhưng CMS vẫn chưa có backing model/API riêng cho:
   - `connection_registry`
   - `source_object_registry`
   - `shadow_binding`
   - `master_binding` đầy đủ ngoài màn master hiện có
2. `/api/registry` vẫn là compatibility surface chính cho source-object/shadow maintenance.
3. Một số comments lịch sử trong code vẫn còn nhắc `cdc_internal` như legacy context, dù runtime path đã sạch hơn.
4. Worker payload/persistence của nhiều flow vẫn còn target-table-centric để giữ cutover an toàn.

## Residual Gap sau Phase 18

1. `TableRegistry` đã đọc từ V2 metadata, nhưng write path/operator actions vẫn còn phụ thuộc vào `/api/registry` và `registry_id` bridge.
2. Row V2-only chưa có bridge legacy hiện mới ở trạng thái monitorable + partially actionable, chưa có mutation path V2-native đầy đủ.
3. Generated Swagger docs chưa được regenerate vì môi trường hiện thiếu binary `swag`; hiện source-of-truth là annotations trong code.
4. Màn V2-native riêng cho `shadow_binding` và chi tiết `source_object_registry` vẫn chưa tồn tại, nên `/api/registry` còn phải gánh một phần operator workflow.

## Residual Gap sau Phase 19

1. `shadow_binding` đã có monitoring surface riêng, nhưng vẫn mới là read-only V2 surface; chưa có mutation path V2-native cho binding management.
2. `MappingFieldsPage` vẫn phải fetch `/api/registry` để resolve row theo `registry_id`, nên page mappings còn là một điểm phụ thuộc lớn vào compatibility shell.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Các operator actions cũ như `scan-fields`, `standardize`, `detect-timestamp-field` vẫn còn neo vào `registry_id` legacy.

## Residual Gap sau Phase 20

1. `MappingFieldsPage` đã bỏ fetch full registry, nhưng một số operator actions của page vẫn còn neo vào `registry_id` legacy.
2. `ReDetectButton` / timestamp detection và vài action tương tự vẫn chưa có lớp V2-aware riêng.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Vẫn chưa có mutation path V2-native cho `source objects` và `shadow bindings`; FE hiện chủ yếu đã sạch ở read path trước.

## Residual Gap sau Phase 21

1. FE-facing namespace của các action bridge đã sạch hơn, nhưng backend semantics của chúng vẫn còn dựa vào `registry_id`.
2. `transform-status` và một số status/action phụ vẫn còn nằm trực tiếp dưới `/api/registry`.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Mutation path V2-native thật cho `source objects` / `shadow bindings` vẫn chưa có; hiện mới chủ yếu là read-path cleanup + action facades.

## Residual Gap sau Phase 22

1. `transform-status` đã được facad hóa, nhưng các mutation compatibility shell như `PATCH /api/registry/:id` và `POST /api/registry/batch` vẫn còn sống thật.
2. `GET /api/registry/stats` và một số registry-level read tổng hợp vẫn chưa có V2-native replacement rõ ràng.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Mutation path V2-native thật cho `source objects` / `shadow bindings` vẫn chưa có; hiện chủ yếu mới dọn read/status surface và action facades.

## Residual Gap sau Phase 23

1. `Dashboard` và `ActivityManager` đã rời read path `/api/registry`, nhưng mutation shell như `POST /api/registry`, `PATCH /api/registry/:id`, `POST /api/registry/batch` vẫn còn sống thật.
2. `ActivityManager` vẫn tạo schedule bằng `target_table` fallback trong payload để giữ compatibility với storage hiện tại.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Mutation path V2-native thật cho `source objects` / `shadow bindings` vẫn chưa có; hiện cleanup chủ yếu mới bao phủ read/status surfaces.

## Residual Gap sau Phase 24

1. FE runtime đã gần như sạch `/api/registry`, nhưng backend legacy routes vẫn còn tồn tại để cutover an toàn và cho caller cũ.
2. 3 mutation mới chỉ là facade namespace; backend write-model vẫn còn dựa vào `cdc_table_registry`.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Chưa có write-model V2 thật cho `source objects` / `shadow bindings`; phase này chỉ dọn FE-facing contract.

## Residual Gap sau Phase 25

1. Router đã sạch hơn, nhưng `RegistryHandler` và swagger comments legacy vẫn còn tồn tại như lớp delegate nội bộ.
2. Backend write-model cho source objects vẫn còn dựa vào `cdc_table_registry`.
3. Generated Swagger docs vẫn chưa regen được vì môi trường thiếu `swag`; annotations trong code là source-of-truth hiện tại.
4. Vẫn chưa có write-model V2 thật cho source objects / shadow bindings; mới dọn route surface và FE-facing contracts.

## Residual Gap sau Phase 26

1. Router và source annotations đã sạch hơn, nhưng implementation delegate legacy vẫn còn tồn tại trong `RegistryHandler`.
2. Backend write-model cho source objects vẫn còn dựa vào `cdc_table_registry`.
3. `make swagger` vẫn chưa chạy được trên máy hiện tại vì thiếu `swag`, nên chưa thể chứng minh generated docs cuối cùng.
4. Vẫn chưa có write-model V2 thật cho source objects / shadow bindings.
## 2026-04-27 — Gap Update After Phase 30

### Gap A
`detect-timestamp-field` đã direct-V2 hóa được, nhưng `create-default-columns` vẫn chưa thể direct hóa an toàn vì worker path còn cần metadata legacy/bridge nhiều hơn.

### Gap B
`MappingFieldsPage` và một phần operator actions vẫn còn neo vào `registry_id` cho create/sync style actions. Đây là debt thật, không còn chỉ là FE naming.

### Gap C
Generated swagger docs vẫn chưa regen được trên local vì thiếu binary `swag`; hiện source-of-truth là annotations trong code.
## 2026-04-27 — Gap Update After Phase 31

### Gap D
`create-default-columns` vẫn là debt bridge lớn nhất ở operator-flow source-object maintenance.

### Gap E
`standardize` hiện vẫn bridge-backed; dù payload nhìn đơn giản, cần audit thêm table/namespace assumptions trước khi direct hóa.

### Gap F
`MappingFieldsPage` vẫn còn một số hành động gắn với bridge registry trong luồng sync/create-default-columns.

## 2026-04-28 — Gap Update After Phase 32

### Gap G
`create-default-columns` vẫn là bridge debt lớn nhất còn lại ở `TableRegistry` vì ảnh hưởng tới DDL path + trạng thái `is_table_created`.

### Gap H
Nếu muốn direct-V2 hóa `create-default-columns`, cần audit và có thể sửa worker payload/path để không còn phụ thuộc `cdc_table_registry` khi cập nhật post-create state.

## 2026-04-28 — Gap Update After Phase 33

### Gap I
`create-default-columns` đã có direct path, nhưng worker vẫn dual-write trạng thái về legacy `is_table_created` để giữ compatibility. Chưa thể drop ngay lớp này.

### Gap J
`standardize` và `create-default-columns` đã schema-aware ở operator path mới, nhưng các SQL helper/migration legacy trong worker vẫn còn nhiều assumption `public`.

## 2026-04-28 — Gap Update After Phase 34

### Gap K
Các command/operator path chính trong worker đã schema-aware hơn, nhưng `discover`, `backfill` và vài helper compatibility vẫn chưa được kéo hết khỏi assumption `public`.

### Gap L
`shadow_binding.ddl_status` đang ngày càng trở thành source of truth tốt hơn, nhưng `is_table_created` vẫn còn bị dual-write để giữ compatibility với bridge cũ.

## 2026-04-28 — Gap Update After Phase 35

### Gap M
Worker đã schema-aware hơn ở `discover/backfill/schema-inspector`, nhưng compatibility shell phía CMS vẫn còn helper introspection/backfill cũ bám `public`.

### Gap N
Nếu muốn dọn tiếp sạch hơn, phase sau nên audit các path legacy ít dùng như `event_bridge`, `transform_service`, và CMS `registry_repo` để quyết định giữ, bọc, hay loại bỏ.

## 2026-04-28 — Gap Update After Phase 36

### Gap O
`event_bridge`, `transform_service` và `cms registry_repo` đã được harden, nhưng một phần trong đó vẫn đang là “prepared path” hơn là path có caller runtime dày đặc.

### Gap P
Nếu muốn tiếp tục tối giản hệ thống, phase sau nên quyết định dứt điểm path nào:
- giữ như compatibility reserve
- kéo lên caller thật
- hoặc xóa hẳn để giảm debt
