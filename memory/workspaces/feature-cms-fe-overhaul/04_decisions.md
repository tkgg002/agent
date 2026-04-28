# Decisions

## [2026-04-25] Cải tổ FE sẽ đi theo flow vận hành V2, không đi theo legacy menu

- Lý do:
  - `cdc_internal` đã bị loại khỏi runtime
  - control plane mới ở `cdc_system`
  - operator cần flow kiểu `source -> shadow -> master -> schedule -> health`

## [2026-04-25] Không refactor code FE ngay trong phase audit

- Phase hiện tại chỉ audit + plan
- Refactor code FE sẽ là phase sau, sau khi user chốt scope/menu mới

## [2026-04-27] `worker-schedule` là API operator-flow, không phải route phụ có thể làm nghèo contract

- Lý do:
  - `cms-fe` chịu trách nhiệm monitoring / backup / retry / reconcile
  - operator không được buộc phải suy luận scope chỉ từ `target_table`
  - giữ surface API ít không đồng nghĩa với làm API mù nghĩa
- Quyết định:
  - không thêm endpoint mới
  - enrich trực tiếp `GET/POST /api/worker-schedule`
  - giữ compatibility với payload legacy nhưng ưu tiên metadata V2 khi resolve scope

## [2026-04-27] `v1/masters` phải lấy `shadow_schema + shadow_table` làm identity operator-facing chính

- Lý do:
  - `source_shadow` kiểu string phẳng không đủ cho namespace V2
  - operator cần nhìn và gửi đúng master namespace + shadow namespace thật
  - vẫn cần giữ compatibility ngắn hạn cho flow cũ
- Quyết định:
  - backend `v1/masters` chuyển sang `cdc_system.master_binding`
  - FE submit `master_schema`, `shadow_schema`, `shadow_table`
  - `source_shadow` chỉ còn là fallback

## [2026-04-27] `mapping-rules` phải resolve theo source/shadow metadata thay vì chỉ `source_table`

- Lý do:
  - `mapping_rule_v2` đã tồn tại ở `cdc_system`
  - rule layer phục vụ cả auto-flow lẫn operator-flow
  - dispatch reload/backfill theo `source_table` đơn lẻ sẽ mơ hồ khi nhiều namespace trùng tên
- Quyết định:
  - backend `mapping-rules` chuyển sang `cdc_system.mapping_rule_v2`
  - FE gửi `source_database`, `source_table`, `shadow_schema`, `shadow_table`
  - `source_table`/`table` cũ chỉ còn là fallback

## [2026-04-27] `activity-log` phải là source-of-truth monitoring surface cho operator-flow

- Lý do:
  - operator dùng page này để nhìn lại retry/reconcile/transform thực tế
  - nếu log chỉ có `target_table`, khi namespace trùng tên sẽ gây thao tác sai
  - không cần thêm endpoint mới, nhưng phải enrich contract hiện có
- Quyết định:
  - enrich `activity-log` bằng source/shadow metadata V2
  - `useAsyncDispatch` mở rộng thành hook nhận `statusParams`
  - giữ `target_table` như compatibility fallback

## [2026-04-27] `reconciliation` phải giữ API surface gọn nhưng contract phải scope-aware

- Lý do:
  - `cms-fe` là luồng backup/monitoring/retry/reconcile, nên thao tác check/heal không được mù nghĩa
  - nếu mỗi action vẫn chỉ nhận `:table`, operator sẽ khó phân biệt khi nhiều namespace trùng tên
  - mở thêm quá nhiều endpoint mới sẽ làm surface API phình ra không cần thiết
- Quyết định:
  - reuse `POST /api/reconciliation/check` cho cả all-scope và single-scope
  - thêm `POST /api/reconciliation/heal` generic nhận body scope-aware, nhưng vẫn giữ path legacy
  - enrich `report` và `failed-sync-logs` bằng source/shadow metadata V2

## [2026-04-27] Artifact legacy chỉ được purge sau khi audit usage xác nhận đã chết ở runtime

- Lý do:
  - mục tiêu là giảm dư thừa feature/API thật, không phải xóa mù rồi tạo regression
  - nhiều file FE/BE legacy vẫn có thể tự đứng riêng nhưng không còn route/wiring
- Quyết định:
  - audit grep + route + server wiring trước
  - sau khi xác nhận dead runtime mới xóa vật lý

## [2026-04-27] `/api/registry` chưa phải surface dư hoàn toàn; chỉ prune phần dead trước

- Lý do:
  - FE/operator-flow vẫn đang dùng nhiều action thực chiến như `scan-fields`, `standardize`, `detect-timestamp-field`, `create-default-columns`
  - nhưng trong cùng cụm lại có nhiều route đã retired hoặc không còn caller
- Quyết định:
  - xem `/api/registry` là compatibility surface có chọn lọc
  - chỉ gỡ route dead trước
  - update swagger/comment của phần còn sống sang semantics `Source Objects`

## [2026-04-27] `retry failed log` giữ ID làm identity chuẩn, chỉ enrich scope ở output/payload

- Lý do:
  - retry là thao tác lên một failed log cụ thể, không phải resolve target từ operator scope như check/heal
  - ép FE gửi thêm source/shadow input ở đây không tăng độ đúng, chỉ làm API nặng hơn
  - operator vẫn cần nhìn rõ scope để tránh mù nghĩa
- Quyết định:
  - giữ `POST /api/failed-sync-logs/{id}/retry`
  - enrich response và NATS payload bằng source/shadow metadata khi resolve được

## [2026-04-27] Màn V2-native đầu tiên của FE nên dựng từ API đã có, không bịa thêm surface mới

- Lý do:
  - hiện đã có đủ 2 API để tách runtime connector và source fingerprint:
    - `/api/v1/system/connectors`
    - `/api/v1/sources`
  - dùng API sẵn có giúp giảm rủi ro và cho phép tiến hóa IA trước khi phải dựng API mới
- Quyết định:
  - nâng `Sources & Connectors` thành dual-view screen
  - dùng page này làm bước đệm sang các màn V2-native sâu hơn

## [2026-04-27] Nếu migration end-state đã move bảng system sang `cdc_system`, CMS runtime model phải đi theo ngay

- Lý do:
  - để FE semantics mới mà backend model vẫn trỏ `cdc_internal` sẽ tạo drift khó thấy
  - `Source` và `WizardSession` đang là backing store sống cho UI hiện tại
- Quyết định:
  - sửa namespace runtime model/repo ngay khi audit phát hiện
  - không chờ tới khi có màn V2-native hoàn chỉnh mới sửa

## [2026-04-27] `TableRegistry` nên chuyển read path sang V2 trước, nhưng write path vẫn giữ compatibility shell

- Lý do:
  - page này đã đổi semantics sang `Source Objects`, nên tiếp tục đọc `/api/registry` sẽ giữ sai source-of-truth
  - nhiều action operator thực chiến vẫn còn buộc vào `registry_id` legacy
  - ép cutover cả read + write trong một phase sẽ dễ biến operator-flow thành “vỏ”
- Quyết định:
  - thêm `GET /api/v1/source-objects` đọc từ `cdc_system.source_object_registry + shadow_binding`
  - carry `registry_id` bridge nếu có
  - FE đọc từ endpoint mới
  - action legacy chỉ bật khi row có bridge thật

## [2026-04-27] `shadow_binding` nên được đưa lên cùng màn `Source Objects` thay vì mở thêm page mới

- Lý do:
  - operator cần nhìn binding layer thật, nhưng thêm page mới sẽ làm IA nở ra
  - `source object` và `shadow binding` là 2 mặt của cùng một flow vận hành
  - dual-view trong cùng page giữ FE gọn hơn mà vẫn practical
- Quyết định:
  - thêm `GET /api/v1/shadow-bindings`
  - nâng `TableRegistry` thành dual-view screen với tab `Shadow Bindings`

## [2026-04-27] `MappingFieldsPage` nên chuyển sang detail read-model theo `registry_id`, không fetch toàn bộ `/api/registry`

- Lý do:
  - page này chỉ cần một context duy nhất cho row hiện tại
  - tải cả registry list rồi tự `find()` vừa nặng vừa giữ dependency sai
  - vẫn cần bridge `registry_id` để giữ operator actions legacy
- Quyết định:
  - thêm `GET /api/v1/source-objects/registry/{registry_id}`
  - page mappings dùng endpoint này làm context header/read-model chính

## [2026-04-27] Với các action còn cần `registry_id`, nên dùng facade V2-aware thay vì tiếp tục để FE gọi thẳng `/api/registry`

- Lý do:
  - semantics backend của các action này chưa đổi thật sang V2
  - nhưng FE-facing namespace vẫn nên phản ánh đúng kiến trúc hiện tại
  - cách này giảm lộ compatibility shell mà không giả capability mới
- Quyết định:
  - thêm facade `/api/v1/source-objects/registry/:id/...`
  - backend delegate sang `RegistryHandler`
  - FE chuyển call-sites sang facade mới

## [2026-04-27] `transform-status` nên được facad hóa tiếp, nhưng `PATCH /api/registry/:id` và `POST /api/registry/batch` chưa nên bọc giả

- Lý do:
  - `transform-status` là read/status surface thuần, nên facad hóa thêm là sạch và rẻ
  - còn `PATCH /api/registry/:id` và `POST /api/registry/batch` vẫn là mutation compatibility shell có semantics legacy thật
- Quyết định:
  - thêm `GET /api/v1/source-objects/registry/:id/transform-status`
  - chưa bọc giả các mutation compatibility shell còn lại ở phase này

## [2026-04-27] `Dashboard` và `ActivityManager` nên rời `/api/registry` ở read path trước, còn mutation shell để lại có chủ đích

- Lý do:
  - `GET /api/registry/stats` và list registry trong `ActivityManager` chỉ là read/enrichment surface
  - hai chỗ này đã có đủ semantics V2 để thay thế mà không làm gãy operator-flow
  - ngược lại `PATCH /api/registry/:id` và `POST /api/registry/batch` vẫn là mutation legacy thật, chưa nên “facade hóa cho đẹp”
- Quyết định:
  - thêm `GET /api/v1/source-objects/stats`
  - chuyển `Dashboard` và `ActivityManager` sang read surface V2
  - tiếp tục giữ mutation compatibility shell cũ cho tới khi có write-model V2 thật

## [2026-04-27] 3 mutation còn lại của `/api/registry` nên được facad hóa cho FE, nhưng backend write semantics vẫn giữ legacy

- Lý do:
  - FE runtime gọi trực tiếp `/api/registry` làm surface còn lệch kiến trúc
  - nhưng `Register`, `Update`, `BulkRegister` vẫn đang ghi vào compatibility write-model thật, chưa nên giả vờ đã chuyển sang V2
  - operator-flow vẫn cần các thao tác này cho monitoring + maintenance hàng ngày
- Quyết định:
  - thêm facade mutation dưới `/api/v1/source-objects`
  - chuyển FE call-sites sang namespace mới
  - giữ `RegistryHandler` làm lớp delegate tạm thời

## [2026-04-27] Route `/api/registry...` không còn caller nội bộ thì nên gỡ khỏi router sau khi có replacement V2

- Lý do:
  - giữ route cũ khi FE/runtime nội bộ đã bỏ hoàn toàn chỉ làm surface API phình ra
  - mục tiêu hiện tại là CMS FE/API gọn, đúng Debezium-only operator-flow
  - nếu capability vẫn cần, replacement V2 phải có trước rồi mới gỡ route cũ
- Quyết định:
  - thêm `POST /api/v1/source-objects/registry/:id/transform`
  - gỡ các route `/api/registry...` đã được thay thế khỏi router

## [2026-04-27] Sau khi prune router, phải dọn luôn swagger annotations legacy ở source code

- Lý do:
  - router sạch mà `@Router` cũ còn nguyên thì lần regen sau sẽ sinh spec sai
  - source-level annotations hiện là source-of-truth cho tương lai khi `swag` có mặt
- Quyết định:
  - đổi các godoc blocks `/api/registry...` trong `RegistryHandler` thành comment delegate nội bộ

## [2026-04-27] Write-path nên đi theo hướng legacy-gatekeeper, V2-sync-follower trước khi thay write-model hoàn toàn

- Lý do:
  - operator-flow hiện vẫn còn bridge sâu với `cdc_table_registry`
  - thay write-model hoàn toàn ngay có rủi ro gãy luồng đang chạy
  - sync song song sau write thành công là bước chuyển tiếp an toàn nhất
- Quyết định:
  - thêm service sync từ `TableRegistry` row sang `cdc_system.source_object_registry` + `cdc_system.shadow_binding`
  - cắm vào `Register/Update/BulkRegister`

## [2026-04-27] Sau khi dual-write sang V2, UI phải hiển thị trạng thái migration rõ ràng thay vì chỉ cảnh báo bridge

- Lý do:
  - operator cần biết row đã `V2 Ready` chưa, không chỉ biết “thiếu bridge”
  - warning một chiều dễ làm họ hiểu sai rằng row chưa usable, dù thực tế metadata V2 có thể đã ổn
- Quyết định:
  - thêm `metadata_status` và `bridge_status` vào read model
  - render thành badge rõ nghĩa trong `TableRegistry`

## [2026-04-27] Row `V2-only` nên được update trực tiếp theo `source_object_id`, nhưng chỉ cho field đã có V2 home rõ ràng

- Lý do:
  - chặn toàn bộ update cho row `V2-only` làm operator-flow bị cụt
  - nhưng `priority`/`sync_interval` hiện chưa có nơi chứa V2 rõ ràng, nên không thể giả vờ hỗ trợ
- Quyết định:
  - thêm `PATCH /api/v1/source-objects/:id`
  - cho phép update `is_active`, `timestamp_field`, `notes`
  - giữ `priority` là bridge-only cho tới phase write-model sâu hơn
