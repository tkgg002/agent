# Status

- Workspace: `feature-cms-fe-overhaul`
- Phase: Phase 17 cdc_system Namespace Bridge
- Current Status: Completed
- Completed in this phase:
  - audit namespace drift cho `Source` và `WizardSession`
  - sửa runtime model/repo sang `cdc_system`
  - verify grep + backend tests
  - CMS go test pass
- Completed in previous phase:
  - dựng dual-view V2-native screen đầu tiên cho `Sources & Connectors`
- Next Recommended Step:
  - Phase 18: audit API và dựng plan cho màn V2-native tiếp theo:
    - `source_object_registry`
    - `shadow_binding`
    - `master_binding`
    ưu tiên xem có thể dựng dual-view practical screen nào tiếp theo từ API đã có

## Update [2026-04-27] Phase 18 Source Objects Read Path V2

- Current Status: Completed
- Completed in this phase:
  - audit dependency của `TableRegistry` vào legacy registry
  - thêm `GET /api/v1/source-objects`
  - chuyển `TableRegistry` sang read path V2
  - giữ operator actions legacy qua `registry_id` bridge
  - verify backend tests + frontend build
  - verify swagger annotations đã cập nhật; generated docs chưa regen do thiếu `swag`
- Next Recommended Step:
  - Phase 19: audit và dựng màn/contract V2-native tiếp theo cho `shadow_binding`
  - ưu tiên giảm tiếp phụ thuộc read-only vào `/api/registry`

## Update [2026-04-27] Phase 19 Shadow Bindings Dual View

- Current Status: Completed
- Completed in this phase:
  - thêm `GET /api/v1/shadow-bindings`
  - nâng `TableRegistry` thành dual-view practical screen
  - verify backend tests + frontend build
  - verify swagger annotations đã cập nhật; generated docs chưa regen do thiếu `swag`
- Next Recommended Step:
  - Phase 20: audit và cắt tiếp dependency của `MappingFieldsPage` vào `/api/registry`
  - ưu tiên tìm một read-model/detail API V2 để page mappings không còn phải fetch toàn bộ `/api/registry`

## Update [2026-04-27] Phase 20 Mapping Context Read Model

- Current Status: Completed
- Completed in this phase:
  - thêm `GET /api/v1/source-objects/registry/:registry_id`
  - chuyển `MappingFieldsPage` sang read-model mới
  - giữ compatibility action legacy cho `create-default-columns`
  - verify backend tests + frontend build
  - verify swagger annotations đã cập nhật; generated docs chưa regen do thiếu `swag`
- Next Recommended Step:
  - Phase 21: audit `ReDetectButton` / timestamp detection flow và các action còn neo trực tiếp vào `registry_id`
  - ưu tiên xem action nào thực sự cần giữ legacy, action nào nên được bọc lại qua read-model V2

## Update [2026-04-27] Phase 21 Registry Bridge Action Facade

- Current Status: Completed
- Completed in this phase:
  - audit nhóm action còn cần `registry_id`
  - thêm facade endpoints dưới `/api/v1/source-objects/registry/:id/...`
  - chuyển FE call-sites sang facade mới
  - verify backend tests + frontend build
  - verify swagger annotations đã cập nhật; generated docs chưa regen do thiếu `swag`
- Next Recommended Step:
  - Phase 22: audit `transform-status` và các status/action còn sót dưới `/api/registry`
  - ưu tiên chốt rõ phần nào sẽ giữ bridge lâu dài, phần nào nên tiếp tục được facad hóa hoặc loại bỏ

## Update [2026-04-27] Phase 22 Transform Status Facade

- Current Status: Completed
- Completed in this phase:
  - audit call-sites còn trỏ `/api/registry`
  - thêm facade `transform-status`
  - chuyển `TableRegistry` sang endpoint mới
  - verify backend tests + frontend build
  - verify swagger annotations đã cập nhật; generated docs chưa regen do thiếu `swag`
- Next Recommended Step:
  - Phase 23: review lại `PATCH /api/registry/:id`, `POST /api/registry/batch`, `GET /api/registry/stats`
  - chốt rõ endpoint nào sẽ tiếp tục là compatibility shell dài hạn, endpoint nào nên bị thay thế hoặc loại khỏi FE

## Update [2026-04-27] Phase 23 Dashboard + ActivityManager V2 Reads

- Current Status: Completed
- Completed in this phase:
  - audit compatibility shell read surfaces còn sót
  - chốt rằng `Dashboard` và `ActivityManager` có thể rời `/api/registry` trước
  - thêm `GET /api/v1/source-objects/stats`
  - chuyển `Dashboard` và `ActivityManager` sang read surface V2
  - verify backend tests + frontend build
  - note trạng thái swagger generation
- Next Recommended Step:
  - sau khi verify pass, Phase 24 sẽ tập trung review mutation shell còn lại:
    - `PATCH /api/registry/:id`
    - `POST /api/registry/batch`
    - `POST /api/registry`

## Update [2026-04-27] Phase 24 Registry Mutation Facade

- Current Status: Completed
- Completed in this phase:
  - audit semantics của 3 mutation compatibility shell còn lại
  - thêm 3 facade mutation dưới namespace `/api/v1/source-objects`
  - chuyển `TableRegistry.tsx` sang namespace mới
  - verify backend tests + frontend build
  - grep lại FE runtime call-sites
  - note trạng thái swagger generation
- Next Recommended Step:
  - sau khi verify pass, Phase 25 nên review xem còn cần giữ route legacy `/api/registry` ở mức backend bao lâu, và có route nào không còn FE/runtime caller nữa

## Update [2026-04-27] Phase 25 Registry Route Prune

- Current Status: Completed
- Completed in this phase:
  - audit caller nội bộ cho nhóm route `/api/registry...`
  - thêm facade `transform`
  - gỡ các route legacy đã có replacement khỏi router
- Next Recommended Step:
  - sau khi verify pass, Phase 26 nên review có cần xóa luôn các handler/swagger comments legacy tương ứng hay giữ làm delegate nội bộ

## Update [2026-04-27] Phase 26 Legacy Swagger Cleanup

- Current Status: Completed
- Completed in this phase:
  - audit legacy annotations còn sót trong `registry_handler.go`
  - dọn `@Router /api/registry...` cũ khỏi source comments
-  verify grep/test
-  note trạng thái `make swagger`
- Next Recommended Step:
  - Phase 27: bắt đầu cắt vào write-model V2 thật cho source objects thay vì chỉ facade/delegate trên `cdc_table_registry`

## Update [2026-04-27] Phase 27 V2 Write Sync

- Current Status: Completed
- Completed in this phase:
  - audit write-path hiện tại
  - thêm service sync V2
  - cắm sync vào register/update/bulk register
-  verify backend tests
-  đánh giá gap còn lại của write-model V2
- Next Recommended Step:
  - Phase 28: đưa trạng thái V2-sync lên read model/UI để operator biết row nào đã có metadata đầy đủ trong `cdc_system`, và bắt đầu giảm các cảnh báo “legacy bridge only”

## Update [2026-04-27] Phase 28 V2 Status Visibility

- Current Status: Completed
- Completed in this phase:
  - audit read model source objects
  - thêm `metadata_status` + `bridge_status`
  - update UI `Source Objects`
-  verify backend tests + frontend build
- Next Recommended Step:
  - Phase 29: bắt đầu kéo `update` path khỏi phụ thuộc `registry_id` bằng cách cho row V2-only có mutation path trực tiếp theo `source_object_id`

## Update [2026-04-27] Phase 29 V2 Direct Update

- Current Status: Completed
- Completed in this phase:
  - audit update path
  - thêm route patch trực tiếp theo `source_object_id`
  - chuyển FE sang chọn endpoint theo bridge status
-  verify backend tests + frontend build
- Next Recommended Step:
  - Phase 30: tiếp tục giảm bridge dependence ở operator actions còn lại, ưu tiên timestamp / mapping related actions
## Phase 30 — Completed

### Done
- `detect-timestamp-field` đã có V2 direct path theo `source_object_id`
- `dispatch-status` cho action này đã có V2 direct path tương ứng
- `DataIntegrity` đã dùng được direct path khi row có `source_object_id`

### Still transitional
- `create-default-columns`
- `scan-fields`
- `standardize`
- `transform-status`

### Reason
- Các action transitional trên vẫn còn cần bridge metadata hoặc worker path legacy sâu hơn, chưa nên direct-V2 hóa giả.
## Phase 31 — Completed

### Direct V2 now available
- `detect-timestamp-field`
- `dispatch-status` cho re-detect
- `scan-fields`
- `transform-status`

### Still bridge/transitional
- `create-default-columns`
- `standardize`
- một phần mapping/operator actions dùng `registry_id`

## Phase 32 — Completed

### Direct V2 now available
- `standardize`

### Still bridge/transitional
- `create-default-columns`
- một phần mapping/operator actions dùng `registry_id`

## Phase 33 — Completed

### Direct V2 now available
- `create-default-columns`
- schema-aware `standardize`

### Still transitional
- một phần mapping/operator actions dùng `registry_id`
- legacy `is_table_created` vẫn còn được cập nhật song song để compatibility không gãy

## Phase 34 — Completed

### Runtime hardening delivered
- thêm `ResolveTargetRoute(target_table)` vào metadata registry
- `SchemaValidator` đã introspect theo `shadow_schema`
- `batch-transform` schema-aware
- `scan-raw-data` schema-aware
- `periodic-scan` schema-aware
- `drop-gin-index` schema-aware
- `scan-fields` sample path schema-aware

### Still transitional
- `discover` / `backfill` legacy paths chưa được schema-aware hóa toàn bộ
- một phần helper compatibility vẫn còn fallback `public`
- dual-write `is_table_created` vẫn còn cần cho cutover safety

## Phase 35 — Completed

### Runtime hardening delivered
- `PendingFieldRepo` hỗ trợ lookup column theo schema
- `SchemaInspector` resolve schema từ metadata V2
- `HandleDiscover` schema-aware
- `HandleBackfill` schema-aware

### Still transitional
- helper compatibility phía CMS như `registry_repo.GetDBColumns()` vẫn còn `public`-centric
- dual-write `is_table_created` vẫn còn cần cho bridge cutover
- một số path legacy ít dùng hơn trong worker vẫn chưa được audit sâu

## Phase 36 — Completed

### Runtime / compatibility hardening delivered
- `event_bridge` poll path schema-aware
- `transform_service` schema-aware nếu được dùng lại
- `cms registry_repo` có helper wrappers:
  - `ScanRawKeysInSchema`
  - `PerformBackfillInSchema`
  - `GetDBColumnsInSchema`

### Still transitional
- các wrappers mới ở CMS repo chưa được kéo lên caller V2-native cụ thể
- dual-write `is_table_created` vẫn còn tồn tại
- còn vài path legacy ít dùng chưa có giá trị refactor ngay
