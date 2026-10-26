# 13 Analysis: Báo Cáo Kiểm Tra & Phân Loại `git diff` Trên 3 Repo CDC (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`)

- **Workspace:** `audit-cdc-3repos-diff-2026-10-26`
- **Suffix:** `cdc_3repos_diff`
- **Tier:** `T1` (Read-Only Diff Audit)
- **Thời gian kiểm tra:** `2026-10-26T09:11:00+07:00`

---

## 1. Tổng Quan Trạng Thái Git Trên 3 Repo CDC

Cả 3 repository của CDC trong `/Users/trainguyen/Documents/work/data-hub` đều đang ở nhánh `recon-heal` (đồng bộ với `origin/recon-heal`, không có staged diff, toàn bộ thay đổi nằm ở Working Tree + 2 file test untracked):

| # | Repository | Nhánh (`HEAD`) | Số file thay đổi | Thống kê dòng (`git diff --stat`) |
| :-: | :--- | :--- | :---: | :--- |
| 1 | `centralized-data-service` | `recon-heal` (`778dbad`) | `18` Modified + `1` Untracked (`19` files) | `+1,490` insertions, `-285` deletions (+ `95` dòng file untracked) |
| 2 | `cdc-cms-service` | `recon-heal` (`818064d`) | `13` Modified + `1` Untracked (`14` files) | `+704` insertions, `-147` deletions (+ `106` dòng file untracked) |
| 3 | `cdc-cms-web` | `recon-heal` (`e2964e3`) | `3` Modified (`3` files) | `+117` insertions, `-42` deletions |
| **Tổng** | **3 Repos CDC** | **`recon-heal`** | **`34` Modified + `2` Untracked (`36` files)** | **`+2,311` insertions, `-474` deletions** |

*(Lưu ý phụ: Repo `cdc-auth-service` ở nhánh `develop` chỉ có 1 diff nhỏ `+6/-1` tại `internal/server/server.go` cấu hình CORS `AllowOrigins` không thuộc 3 repo lõi CDC).*

---

## 2. Phân Nhóm Các Feature Đang Nằm Trong `git diff` Của 3 Repo

Toàn bộ `36` files đang diff trên 3 repo chia thành **2 Feature / Đợt Fix chính** (tương ứng với 2 workspace ngày `2026-10-23`: `nats-monitor-missing-core-stream-2026-10-23` và `audit-recon-heal-delete-id-impact-2026-10-23`):

### Nhóm Feature 1: Tự Phục Hồi NATS JetStream Core Streams, Consumer Watchdog & Fast-Fallback (`missing_core_stream`)
> **Mục tiêu:** Chống mất mát 4 Core Streams (`CDC_EVENTS`, `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) và tự động phục hồi durable consumer `transmute-shadow-consumer-v2` khi NATS khởi động lại hoặc stream bị xóa/tạo lại trên môi trường Multi-Pod, đồng thời tránh phạt độ trễ `300ms` khi fallback sang Core NATS.

1. **Single Source of Truth cho 4 Core Streams & Non-Fail-Fast `EnsureStreams()` + Reconnect Auto-Heal:**
   - `centralized-data-service/pkgs/natsconn/nats_client.go:17-264` & `pkgs/natsconn/nats_client_test.go:1-145`
   - `cdc-cms-service/pkgs/natsconn/nats_client.go:18-286` & `pkgs/natsconn/nats_client_test.go:1-106` (untracked)
   - `cdc-cms-service/internal/server/server.go:126-128`
   - **Chi tiết:** Định nghĩa tập trung `DefaultCDCEventsStreamConfig`, `DefaultSchemaDriftStreamConfig`, `DefaultSchemaConfigStreamConfig`, `DefaultTransmuteStreamConfig`; bảo vệ `Conn`/`JS` bằng `sync.RWMutex`; tự động gọi `EnsureStreams()` bất đồng bộ (`reconciling atomic.Bool`) trong `ReconnectHandler`; gom lỗi bằng `errors.Join` thay vì fail-fast ở stream đầu tiên; giữ nguyên `Subjects` của worker đang chạy (`preserveExistingSubjects`).
2. **Watchdog Tự Phục Hồi Push Consumer & Chống Zombie `_INBOX` Multi-Pod (`TransmuteConsumerPool`):**
   - `centralized-data-service/internal/handler/master/transmute_consumer_pool.go:55-400`
   - `centralized-data-service/internal/handler/master/transmute_debounce_group_test.go:196-377`
   - **Chi tiết:** Thêm `watchdogLoop` (chu kỳ `10s`) kiểm tra `ConsumerInfo` bên ngoài mutex; phát hiện cả trường hợp Stream/Consumer bị xóa (`ErrStreamNotFound`/`ErrConsumerNotFound`) lẫn trường hợp Pod khác đã tạo lại consumer mới (`!info.PushBound`, lệch `boundDeliverSubject` hoặc `boundConsumerCreated`); chỉ `Drain()/Unsubscribe()` `p.subs` cũ sau khi đảm bảo stream/consumer trên NATS đã sẵn sàng.
3. **Fast-Fallback `< 50ms` sang Core NATS khi Stream `CDC_TRANSMUTE` vắng mặt:**
   - `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:123-127`
   - `centralized-data-service/internal/handler/shadow/batch_buffer_fanout_test.go:1-95` (untracked)
   - **Chi tiết:** Trong `publishTransmuteTrigger`, nếu `bb.js.PublishMsg` trả về `nats.ErrNoStreamResponse` hoặc `nats.ErrStreamNotFound`, lập tức `break` khỏi vòng retry (không ngủ chờ `300ms`) để bắn ngay sang Core NATS `cdc.cmd.transmute`.
4. **Auto-Heal Missing Core Streams có Cooldown `30s` trên trang NATS Monitor (`/nats-monitor`):**
   - `cdc-cms-service/internal/api/system/nats_monitor_handler.go:29-275`
   - `cdc-cms-service/internal/api/system/nats_monitor_handler_test.go:258-397`
   - **Chi tiết:** Đồng bộ `canonicalCoreStreams` từ `natsconn.Default*StreamConfig()` và thêm `tryAutoHealMissingCoreStreams` có cổng chặn `lastHealAttemptUnix atomic.Int64` (`autoHealCooldownSec = 30s`) trong `GET /api/system/nats/overview`.

---

### Nhóm Feature 2: Chuẩn Hóa Khóa Chính Master Từ `transform_spec.pk`, Định Danh Kết Nối Thật (`Anti-Cheat`) & Sửa Lỗi Đồng Bộ / Đối Soát (`recon_heal_delete_id`)
> **Mục tiêu:** Loại bỏ hoàn toàn cơ chế đoán mò tên cột PK (`_id`, `id`, `_source_id`) và chuỗi kết nối giả (`"default"`, `"default_shadow"`), bắt buộc lấy chính xác từ `transform_spec` (`{"pk":"..."}`) và `connection_registry`, đồng thời sửa lỗi mất/lệch dữ liệu giữa Shadow và Master trong Transmute & Recon Self-Healing (Segment B).

1. **Quản trị & Tự động đồng bộ `transform_spec.pk` từ `source_object_registry.primary_key_field` (Xuyên suốt UI $\rightarrow$ CMS $\rightarrow$ DB):**
   - `cdc-cms-web/src/pages/MasterRegistry.tsx:290, 568-609, 787-800, 1334-1360, 1406-1470, 1661-1675` & `cdc-cms-web/src/types/index.ts:119-121`:
     - Tự động điền `"pk"` vào ô `Spec (JSON)` dựa trên `primary_key_field` của Shadow Binding khi chọn bảng hoặc đổi `transform_type`.
     - Chặn submit trên UI nếu `Spec (JSON)` thiếu trường `"pk"` hoặc `"pk"` rỗng (khi Tạo mới và khi Sửa Spec).
     - Đổi mã kết nối Master mặc định trên form từ `'default'` sang `'default_master'`.
   - `cdc-cms-service/internal/app/commands/master/create_master.go:63-85, 126-151` & `internal/app/commands/master/update_master_spec.go:42-78`:
     - Bắt buộc validate `transform_spec` có `"pk"` hợp lệ (`invalid_spec_pk_required`). Khi tạo Master Binding, nếu `spec.pk` chưa truyền thì tự động tra cứu `GetSourcePrimaryKeyField(ctx, sourceObjectID)` từ `source_object_registry` để điền vào `transform_spec`.
   - `cdc-cms-service/internal/infra/persistence/governance/provisioning_orchestrator.go:374-485`:
     - Khi auto-provision `master_binding` (`seedMasterBindingForAdvance`), đọc `sor.primary_key_field` để ghi `transform_spec = {"pk": "<primary_key_field>"}` thay vì `'{}'::jsonb`, và tự backfill nếu bản ghi cũ đang có `transform_spec->>'pk'` rỗng.
   - `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:200-275, 406, 464-523, 930-949` (+ `internal/app/ports/repository.go:107`, `internal/app/commands/master/update_column_metadata_test.go:257-289`):
     - `CloneMappingRules` gọi `resolveMasterPKTargetColumn` ưu tiên đọc `transform_spec->>'pk'`; `UpdateSpec` thêm điều kiện `AND schema_status <> 'approved'`; bổ sung method `GetSourcePrimaryKeyField`.
2. **Truy Vấn Đúng `shadow_connection_key` & `master_connection_key` Thật Từ `connection_registry` (Xoá Bỏ Hardcode `"default"` / `"default_shadow"`):**
   - `cdc-cms-web/src/pages/ActivityManager.tsx:628-634, 700-760, 861-924`:
     - Bỏ hardcode `'default_shadow'` / `'default_master'` / `'default'` trong `ConfluenceBranchSelector` và `normalizeSavedBranches`, hiển thị đúng `shadow_connection_code`, `source_connection_code`, `master_connection_code` từ backend.
   - `cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go:414-415, 472-474` & `internal/app/queries/source/source_objects_read_models.go:118-120`:
     - `ListShadowBindings` JOIN thêm `cdc_system.connection_registry shd_cn` để trả về `shadow_connection_id`, `shadow_connection_code`, `primary_key_field`.
   - `cdc-cms-service/internal/bootstrap/master_connection.go:18-65`, `internal/server/server.go:118`, `internal/infra/persistence/master/master_repo_gorm.go:553-567`:
     - `EnsureDefaultMasterConnection` seed đầy đủ `host`, `port`, `default_database` cho `default_master` từ env/config DB; `ResolveMasterConnection` chỉ fallback trong nhóm `role_type IN ('master', 'destination', 'dest', 'mixed')` (bỏ `OR engine_type IN ('postgres', 'postgresql')`).
   - `centralized-data-service`:
     - `internal/handler/shadow/batch_buffer.go:72-76`, `internal/handler/shadow/batch_buffer_fanout.go:27-90, 151-199`, `internal/handler/shadow/batch_transform_handler.go:591-641`, `internal/sinkworker/worker.go:26-33, 372-465`, `internal/service/recon/recon_tier_a.go:580-588`, `internal/service/recon/recon_engine_segment_b.go:25-75, 117-126`: Khi publish trigger transmute từ Kafka BatchBuffer, BatchTransform, SinkWorker hoặc Recon Tier A, đều truy vấn `shadow_binding_code`, `shadow_schema`, `shadow_connection_key` thật từ `cdc_system.shadow_binding LEFT JOIN cdc_system.connection_registry` (bỏ hẳn `"shadow_connection_key": "default"` / `"default_shadow"`).
     - `internal/repository/master/master_binding_repo.go:82-112` & `internal/service/master/master_ddl_generator.go:478`: Bỏ `COALESCE(..., 'default')`, so khớp trực tiếp `@shadow_connection_key = '' OR scr.connection_code = @shadow_connection_key`.
     - `internal/service/source/connection_manager.go:89-138`: Cache `destDB` vào `masterDBPool[key]` khi `connection_registry` có bản ghi active của destination mặc định.
3. **Sửa Lỗi Logic Đồng Bộ Transmute (`Shadow -> Master`) & Đối Soát Tự Chữa Lành (`ExecuteHealSegB`):**
   - `centralized-data-service/internal/service/master/transmuter.go:895-915, 974-976, 1116, 1216-1330, 1516-1525, 1689-1715, 1824-1836` & `internal/service/master/transmuter_utils.go:750-776` (+ test `internal/service/master/transmuter_orphan_test.go:265-399`):
     - Bổ sung `isMasterPKMappedFromSourcePK` kiểm chứng cột `pkCol` trong `transform_spec.pk` thực sự được map từ khóa chính nguồn trong `mapping_rule_master` trước khi dùng để xóa hoặc suy diễn watermark.
     - Trong `processBatch`: Nếu trong cùng batch có cả dòng `_deleted = true` cũ và dòng active (`_deleted = false`) mới hơn của cùng `SourceID`, chặn việc đưa `SourceID` vào `allDeleteSourceIDs` (tránh xóa mất dòng vừa upsert).
     - Tự động thu thập `staleLegacyGpayIDs` (các dòng mang `_gpay_id` băm SHA-256 kiểu cũ khác với Sonyflake `_gpay_id` chuẩn) để dọn trước khi upsert khi `conflictTarget == "_gpay_id"`.
     - Trong `bulkUpsertMaster`: Cho phép ghi đè cập nhật khi `_gpay_id` trên Master lệch với `_gpay_id` chuẩn từ Shadow (`OR COALESCE(%s._gpay_id, 0) <> COALESCE(EXCLUDED._gpay_id, 0)`).
     - Trong `lookupShadowGpayIDs`: Sắp xếp `ORDER BY COALESCE(_deleted, false) ASC, _gpay_id DESC` để luôn lấy `_gpay_id` của dòng đang active.
   - `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:358-375, 507-730, 862-905` (+ test `internal/handler/recon/recon_heal_v4_test.go:240-539`):
     - Thay thế hàm đoán mò `resolvePKColumnAndType` bằng `resolveMasterPKFromSpec` (chỉ đọc nghiêm ngặt từ `transform_spec.pk`, lấy kiểu dữ liệu từ `mapping_rule_master` hoặc `information_schema.columns`).
     - Trong `executeHealSegB`: Khi prune theo `_gpay_id`, kiểm tra đối chiếu khóa nghiệp vụ `pkCol` với bảng Shadow (`WHERE "_source_id" IN (?) AND NOT COALESCE("_deleted", false)`). Nếu bản ghi thực chất vẫn còn sống trên Shadow (chỉ bị lệch `_gpay_id` SHA-256 cũ vs Sonyflake mới), **không xóa khỏi Master** mà gom vào `resyncSourceIDs` đẩy qua `publishTransmuteChunked` để cập nhật lại `_gpay_id`.
     - Trong `resolveSourceIDsForSegmentB`: Truy vấn trực tiếp cột `"_source_id"` (`AND NOT COALESCE("_deleted", false)`) trên bảng Shadow và trả về `nil` (thay vì trả nguyên mảng `inputIDs` là `_gpay_id`) khi `effectiveIDType == "gpay"` không tìm thấy bản ghi tương ứng.

---

## 3. Bảng Tham Chiếu Chi Tiết Từng File Diff Theo Repo

### 3.1. Repo 1: `centralized-data-service` (`18` Modified + `1` Untracked)
| # | File | Trạng thái | Thuộc Feature | Tóm tắt thay đổi chính (`file:line`) |
| :-: | :--- | :---: | :--- | :--- |
| 1 | `pkgs/natsconn/nats_client.go` | `M` (`+280/-60`) | Feature 1 (NATS Self-Heal) | Chuẩn hóa `Default*StreamConfig()` cho 4 Core Streams, khóa `mu`, auto-reconcile khi `ReconnectHandler` (`L17-264`). |
| 2 | `pkgs/natsconn/nats_client_test.go` | `M` (`+48/-0`) | Feature 1 (NATS Self-Heal) | Unit/Integration test kiểm chứng `EnsureStreams` và `preserveExistingSubjects` (`L98-145`). |
| 3 | `internal/handler/master/transmute_consumer_pool.go` | `M` (`+187/-15`) | Feature 1 (NATS Self-Heal) | Thêm `watchdogLoop` tự phục hồi stream/consumer, kiểm tra `PushBound`/`boundDeliverSubject`/`boundConsumerCreated` chống Zombie `_INBOX` (`L55-400`). |
| 4 | `internal/handler/master/transmute_debounce_group_test.go` | `M` (`+182/-0`) | Feature 1 (NATS Self-Heal) | Test watchdog tự khôi phục khi mất stream `CDC_TRANSMUTE` và khi Pod khác recreate consumer (`L196-377`). |
| 5 | `internal/handler/shadow/batch_buffer_fanout_test.go` | `??` (`+95`) | Feature 1 (NATS Self-Heal) | Test `publishTransmuteTrigger` fast-fallback `< 50ms` sang Core NATS khi mất stream (`L1-95`). |
| 6 | `internal/handler/shadow/batch_buffer.go` | `M` (`+6/-1`) | Feature 2 (Anti-Cheat Conn) | Mở rộng `postIngestCacheEntry` lưu `bindingCode`, `shadowSchema`, `shadowConnKey` (`L72-76`). |
| 7 | `internal/handler/shadow/batch_buffer_fanout.go` | `M` (`+52/-10`) | Feature 1 & 2 | Fast-break trên `ErrNoStreamResponse` (`L123-127`) + truy vấn `shadow_binding_code` & `shadow_connection_key` thật từ DB (`L27-90, 151-199`). |
| 8 | `internal/handler/shadow/batch_transform_handler.go` | `M` (`+27/-3`) | Feature 2 (Anti-Cheat Conn) | Truy vấn `binding_code`, `shadow_schema`, `shadow_connection_key` thật khi bắn trigger sau `batch-transform` (`L591-641`). |
| 9 | `internal/sinkworker/worker.go` | `M` (`+47/-12`) | Feature 2 (Anti-Cheat Conn) | Mở rộng `piCacheEntry` và `resolvePostIngestSchedule` lấy `binding_code`, `shadow_connection_key` thật (`L26-33, 372-465`). |
| 10 | `internal/repository/master/master_binding_repo.go` | `M` (`+3/-3`) | Feature 2 (Anti-Cheat Conn) | Bỏ `COALESCE(..., 'default')`, khớp chính xác `scr.connection_code = @shadow_connection_key` (`L82-112`). |
| 11 | `internal/service/master/master_ddl_generator.go` | `M` (`+1/-1`) | Feature 2 (Anti-Cheat Conn) | Bỏ `COALESCE(cr.connection_code, 'default')` trong `loadBinding` (`L478`). |
| 12 | `internal/service/source/connection_manager.go` | `M` (`+11/-3`) | Feature 2 (Anti-Cheat Conn) | Cache `destDB` cho kết nối master active mặc định trong `GetMasterDB` (`L89-138`). |
| 13 | `internal/service/master/transmuter.go` | `M` (`+68/-23`) | Feature 2 (Transmute/PK Fix) | Kiểm tra `isMasterPKMappedFromSourcePK`, chống xóa đè dòng active cùng batch, dọn `_gpay_id` SHA-256 cũ, cập nhật khi lệch `_gpay_id` (`L895-1836`). |
| 14 | `internal/service/master/transmuter_utils.go` | `M` (`+27/-0`) | Feature 2 (Transmute/PK Fix) | Thêm hàm `isMasterPKMappedFromSourcePK` đối chiếu `transform_spec.pk` với `mapping_rule_master` (`L750-776`). |
| 15 | `internal/service/master/transmuter_orphan_test.go` | `M` (`+135/-1`) | Feature 2 (Transmute/PK Fix) | Unit tests kiểm chứng dọn dòng SHA-256 cũ, chống xóa đè dòng active và kiểm tra `transform_spec.pk` (`L265-399`). |
| 16 | `internal/service/recon/recon_engine_segment_b.go` | `M` (`+40/-3`) | Feature 2 (Recon/Heal Fix) | Thêm `SystemDB()`, `ResolveShadowBindingIdentity()` và lấy `shadow_connection_key` thật trong `ListActiveMasterBindings` (`L25-126`). |
| 17 | `internal/service/recon/recon_tier_a.go` | `M` (`+4/-5`) | Feature 2 (Recon/Heal Fix) | Dùng `ResolveShadowBindingIdentity` lấy `shadow_binding_code` & `shadow_connection_key` thật khi `RunOrphanPrune` (`L580-588`). |
| 18 | `internal/handler/recon/recon_execute_heal_handler.go` | `M` (`+194/-98`) | Feature 2 (Recon/Heal Fix) | Dùng `resolveMasterPKFromSpec` (chỉ đọc từ `transform_spec.pk`), bảo vệ bản ghi còn sống trên Shadow khi prune `_gpay_id` bằng cách resync sang Transmute (`L358-905`). |
| 19 | `internal/handler/recon/recon_heal_v4_test.go` | `M` (`+300/-0`) | Feature 2 (Recon/Heal Fix) | Bộ test kiểm chứng `executeHealSegB` resync dòng sống lệch `_gpay_id` và `resolveMasterPKFromSpec` (`L240-539`). |

### 3.2. Repo 2: `cdc-cms-service` (`13` Modified + `1` Untracked)
| # | File | Trạng thái | Thuộc Feature | Tóm tắt thay đổi chính (`file:line`) |
| :-: | :--- | :---: | :--- | :--- |
| 1 | `pkgs/natsconn/nats_client.go` | `M` (`+215/-26`) | Feature 1 (NATS Self-Heal) | Chuẩn hóa 4 hàm `Default*StreamConfig()`, đồng bộ `EnsureStreams` cả 4 core streams và auto-heal khi reconnect (`L18-286`). |
| 2 | `pkgs/natsconn/nats_client_test.go` | `??` (`+106`) | Feature 1 (NATS Self-Heal) | Test `EnsureStreams` và `preserveExistingSubjects` trên embedded NATS server cô lập (`L1-106`). |
| 3 | `internal/api/system/nats_monitor_handler.go` | `M` (`+95/-34`) | Feature 1 (NATS Self-Heal) | Dùng `natsconn.Default*StreamConfig()` và thêm `tryAutoHealMissingCoreStreams` có cooldown `30s` (`L29-275`). |
| 4 | `internal/api/system/nats_monitor_handler_test.go` | `M` (`+140/-0`) | Feature 1 (NATS Self-Heal) | Test `GetOverview` tự động khôi phục core streams bị thiếu trên embedded NATS (`L258-397`). |
| 5 | `internal/server/server.go` | `M` (`+4/-1`) | Feature 1 & 2 | Truyền `cfg.DB` vào `EnsureDefaultMasterConnection` (`L118`) và gọi `natsClient.EnsureStreams()` khi khởi động (`L126-128`). |
| 6 | `internal/bootstrap/master_connection.go` | `M` (`+51/-8`) | Feature 2 (Anti-Cheat Conn) | Seed `default_master` kèm `host`, `port`, `default_database` thật từ env/config DB (`L18-65`). |
| 7 | `internal/app/commands/master/create_master.go` | `M` (`+41/-9`) | Feature 2 (Strict Spec PK) | Validate `spec.pk` và tự động backfill `spec["pk"]` từ `GetSourcePrimaryKeyField` khi tạo Master Binding (`L63-151`). |
| 8 | `internal/app/commands/master/update_master_spec.go` | `M` (`+21/-14`) | Feature 2 (Strict Spec PK) | Bắt buộc `transform_spec` khi update phải có trường `"pk"` không rỗng (`L42-78`). |
| 9 | `internal/app/ports/repository.go` | `M` (`+1/-0`) | Feature 2 (Strict Spec PK) | Thêm `GetSourcePrimaryKeyField(ctx, sourceObjectID)` vào `MasterRepo` interface (`L107`). |
| 10 | `internal/infra/persistence/governance/provisioning_orchestrator.go` | `M` (`+15/-2`) | Feature 2 (Strict Spec PK) | Khởi tạo `transform_spec = {"pk": "<primary_key_field>"}` khi auto-seed `master_binding` (`L374-485`). |
| 11 | `internal/infra/persistence/master/master_repo_gorm.go` | `M` (`+92/-41`) | Feature 2 (Strict Spec PK & Conn) | `resolveMasterPKTargetColumn` ưu tiên đọc `transform_spec->>'pk'`, `GetSourcePrimaryKeyField`, chặn `UpdateSpec` khi `approved`, bỏ fallback `engine_type = postgres` (`L200-949`). |
| 12 | `internal/app/commands/master/update_column_metadata_test.go` | `M` (`+33/-0`) | Feature 2 (Strict Spec PK) | Unit test kiểm chứng `CreateMasterHandler` và `UpdateMasterSpecCommand` cưỡng chế `spec.pk` (`L257-289`). |
| 13 | `internal/app/queries/source/source_objects_read_models.go` | `M` (`+3/-0`) | Feature 2 (Anti-Cheat Conn) | Thêm `ShadowConnectionID`, `ShadowConnectionCode`, `PrimaryKeyField` vào `ShadowBindingListItem` (`L118-120`). |
| 14 | `internal/infra/persistence/source/source_object_read_repo_gorm.go` | `M` (`+5/-0`) | Feature 2 (Anti-Cheat Conn) | JOIN `connection_registry shd_cn` trả về `shadow_connection_code` và `primary_key_field` trong `ListShadowBindings` (`L414-474`). |

### 3.3. Repo 3: `cdc-cms-web` (`3` Modified)
| # | File | Trạng thái | Thuộc Feature | Tóm tắt thay đổi chính (`file:line`) |
| :-: | :--- | :---: | :--- | :--- |
| 1 | `src/types/index.ts` | `M` (`+3/-0`) | Feature 2 (Strict Spec PK & Conn) | Bổ sung `shadow_connection_id`, `shadow_connection_code`, `primary_key_field` vào interface `ShadowBindingRow` (`L119-121`). |
| 2 | `src/pages/MasterRegistry.tsx` | `M` (`+106/-33`) | Feature 2 (Strict Spec PK & Conn) | Tự động điền `{"pk":"..."}` theo `primary_key_field` của bảng Shadow; bắt buộc validate `parsedSpec.pk` khi Create/Edit Master Binding; dùng `'default_master'` thay cho `'default'` (`L290, 568-609, 787-800, 1334-1470, 1661-1675`). |
| 3 | `src/pages/ActivityManager.tsx` | `M` (`+8/-9`) | Feature 2 (Anti-Cheat Conn) | Bỏ hardcode `'default_shadow'` / `'default_master'` / `'default'`, hiển thị đúng mã connection từ API trên bộ chọn nhánh dữ liệu (`L628-924`). |

---

## 4. Kết Quả Kiểm Định `/laya_router` (`DG3`)

- **Lệnh thực thi:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --workspace audit-cdc-3repos-diff-2026-10-26 --stage dg3 --claims claims_dg3.json`
- **Biên lai Summary:** `06_laya_summary_dg3_20261026T091600.json` (`gate: PASS`)
- **Lưu ý Calibration (Rule #10):** `trusted_rounds: []` (`UNVALIDATED` trên Laya local do control calibration), toàn bộ `5` claims (`A-DIFF-1..A-DIFF-5`) đã được đối chiếu và xác minh trực tiếp bằng ground truth `REF: <path>:<start>-<end>` từ mã nguồn thực tế trên đĩa (`evidence_error: ""`).

