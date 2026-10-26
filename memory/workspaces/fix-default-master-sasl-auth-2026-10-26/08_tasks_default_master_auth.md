# 08 Tasks: Khắc phục lỗi kết nối `default_master` (`user=root database=cdc_cms_database` SQLSTATE 28P01)

- **TASK-01**: Điều tra Root Cause tại `centralized-data-service/internal/service/source/connection_manager.go`, `centralized-data-service/internal/service/master/transmuter.go`, và `cdc-cms-service/internal/bootstrap/master_connection.go`.
  - **Context**: `connection_manager.go:60-180`, `transmuter.go:880-950`, `master_connection.go:15-65`
  - **Logs/Test**: Log lỗi `open master db connection for "default_master": gorm open: failed to connect to user=root database=cdc_cms_database: 10.200.186.80:5432`
  - **DoD**: Xác định chính xác vì sao `GetMasterDB(ctx, "default_master")` lại tự build DSN với `user=root` và `database=cdc_cms_database` thay vì sử dụng connection pool `database.RoleDestination` đã được cấu hình sẵn của `centralized-data-service`.
- **TASK-02**: Thiết kế giải pháp chuẩn xác trong `09_tasks_solution_default_master_auth.md` và kiểm định qua `/laya_router` (`DG0`, `DG1/DG2`).
  - **Context**: `cdc-cms-service/internal/bootstrap/master_connection.go` & `centralized-data-service/internal/service/source/connection_manager.go`
  - **Logs/Test**: `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg0` & `dg1`
  - **DoD**: Đạt `PASS` cổng `DG0` và `DG1/DG2`, trình User `APPROVE`.
- **TASK-03**: Transmute lỗi không được giữ lại trên NATS `CDC_TRANSMUTE` (User báo 2026-10-26 09:43, sau commit `3369c31 fix nat, recon & heal` lúc 09:20).
  - **Root cause (VERIFIED theo code):**
    1. Testing đang thiếu stream `CDC_TRANSMUTE` → `PublishMsg` lỗi `ErrNoResponders/ErrNoStreamResponse` → `break` ngay (`centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:123-127`, thuộc commit `3369c31`) → rơi xuống Core NATS `cdc.cmd.transmute-shadow.core-fallback` (`batch_buffer_fanout.go:134-145`). Core NATS không lưu, không redeliver → transmute lỗi là mất trigger.
    2. Kể cả khi stream còn: lỗi `failed SASL auth ... SQLSTATE 28P01` KHÔNG nằm trong `isTransientError` (`internal/handler/master/transmute_handler.go:918-946`) → bị coi là poison pill → `binarySearchSplit` → `complete(poisonErr)` (`transmute_handler.go:696-730`) → `finalizeJetStreamMessage` Nak tối đa `MaxDeliver=5` rồi `Term()` (`internal/handler/master/transmute_consumer_pool.go:349-359`) → xóa khỏi WorkQueue. Lỗi cấu hình/hạ tầng bị đối xử như lỗi dữ liệu.
  - **Dữ liệu:** Shadow vẫn ghi thành công (`kafka-consumer success written=1`), chỉ Master thiếu → backfill được bằng Recon/Heal Segment B sau khi sửa `default_master`.
  - **DoD:** Có giải pháp được duyệt trong `09_tasks_solution_default_master_auth.md` (sau khi User cung cấp jsz + log EnsureStreams testing), DG1/DG2 PASS, User APPROVE.
- **TASK-04**: Recon B report 7495 (testing, 09:34 26/10) vẫn Thiếu 1166 / Thừa 1169 trên 1542 vs 1545 dòng — cùng pattern với 7494 (23/10).
  - **VERIFIED (code):** Recon B so khớp theo `_gpay_id` trên cả Shadow lẫn Master (`centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go:698-753`, `segBPKColumn`). Thiếu ≈ Thừa ≈ 75% tổng → cùng bản ghi nhưng `_gpay_id` ở Master khác Shadow (nghi `_gpay_id` SHA-256 cũ vs Sonyflake).
  - **VERIFIED (code):** Bản sửa trong commit `3369c31` chỉ có tác dụng khi transmute GHI được vào Master: `bulkUpsertMaster` cập nhật khi lệch `_gpay_id` (`transmuter.go:1824-1836`), dọn `staleLegacyGpayIDs` (`transmuter.go:1322-1325`), heal B resync qua `publishTransmuteChunked` (`recon_execute_heal_handler.go:507-610`).
  - **HYPOTHESIS (chưa verify trên testing):** Trên testing mọi transmute từ sau deploy đều lỗi `default_master` 28P01 → không dòng nào được ghi lại → recon B y nguyên.
  - **Không truy cập được:** report 7495 không có trong DB local (`gpay-postgres-cdc` max id = 268) → nằm trên DB testing. Cần User chạy SQL read-only để xác nhận.
  - **DoD:** Có bằng chứng từ testing (activity log transmute sau 09:20 + mẫu `_gpay_id` Shadow vs Master theo cùng business PK) trước khi kết luận.
- **TASK-05**: Mở chức năng `Edit` trong `cdc-cms-web/src/pages/MasterRegistry.tsx` để xem được đang config gì, chỉ khóa sửa/lưu khi chưa Reject (`schema_status === 'approved'`).
  - **Context**: `cdc-cms-web/src/pages/MasterRegistry.tsx:1001-1017` và `1653-1707`
  - **Logs/Test**: Build frontend (`npm run build`) không lỗi, nút `Edit` luôn bấm mở được Modal hiển thị `transform_type` và `spec`, khi `schema_status === 'approved'` thì khóa `Select`, `Input.TextArea` (`readOnly`) và khóa nút `Lưu spec`.
  - **DoD**: Xem được config ở mọi trạng thái, chỉ cho phép chỉnh sửa & lưu khi chưa `approved` (đã `Reject` hoặc đang `pending_review`/`failed`).
- **TASK-06**: Xử lý triệt để lỗi NATS Control Center trên Testing vẫn báo `MISSING_CORE_STREAM` (3 streams: `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) và `MISSING_COMMAND_LISTENER` (`0/19 Ready`).
  - **Context**:
    - `centralized-data-service/internal/server/server_setup.go:99-101`
    - `centralized-data-service/pkgs/natsconn/nats_client.go:178-284`
    - `cdc-cms-service/pkgs/natsconn/nats_client.go:180-287`
    - `cdc-cms-service/internal/api/system/nats_monitor_handler.go:150-273, 782-792, 920-1018`
    - `cdc-cms-service/internal/router/router.go:97-105`
  - **Logs/Test**: Kiểm chứng `go build ./...` ở 2 repo Go + runtime thật trên NATS local `:18222` (không viết/chạy `go test`, không gọi `DeleteStream` tự động).
  - **DoD**:
    1. `NewWorkerServer` (`server_setup.go:99-101`) không `logger.Fatal` khi `EnsureStreams` gặp lỗi cục bộ mà luôn khởi chạy để đăng ký đủ 19 kênh lệnh `cdc.cmd.*` và vòng lặp self-heal định kỳ.
    2. `EnsureStreams` ở cả `centralized-data-service` và `cdc-cms-service` bảo toàn/hợp nhất (`mergeSubjects`) các subjects hiện có của `CDC_EVENTS` (tránh lỗi `UpdateStream` khi `CDC_EVENTS` trên Testing có 4 subjects & 5 durable consumers) và tự động hạ bậc `MaxBytes` (`64 MiB -> 16 MiB -> -1`) khi `js.AddStream` bị từ chối do giới hạn quota dự trữ (`max_file_store: 1.00 GB`, `err_code 10047`).
    3. `GetOverview` đọc cả `/subsz?subs=1` lẫn `/connz?subs=detail` để nhận diện chính xác 19 kênh lệnh `cdc.cmd.*` trong số `Subs: 274` khi kết nối qua routing/cluster, đồng thời hiển thị rõ lỗi `EnsureStreams` (nếu có) trong `governance.alerts` và cung cấp API `POST /api/v1/system/nats/streams/reconcile`.
- **TASK-07**: Khắc phục triệt để lỗi Recon/Heal Segment B (`Shadow -> Master`) báo `Thiếu 1,693 / Thừa 1,680` trên chênh lệch thực tế `-13` (`3,095 vs 3,082`).
  - **Context**:
    - `cdc-cms-service/internal/bootstrap/master_connection.go:18-55`
    - `centralized-data-service/internal/service/master/transmuter.go:130-151, 766-805, 1225-1330, 1800-1857`
    - `cdc-cms-web/src/components/ExecuteHealModal.tsx:191-204`
  - **Logs/Test**: Kiểm chứng `go build ./...` ở 2 repo Go, `npm run build` ở `cdc-cms-web`, và trace runtime value thật trên PostgreSQL (`gpay-postgres-cdc`, `gpay-postgres-shadow`, `gpay-postgres-dest`) — tuyệt đối không viết/chạy test mockup.
  - **DoD**:
    1. Tự động backfill `transform_spec` (`{"pk": "<primary_key_field>"}`) từ `cdc_system.source_object_registry` cho các dòng `cdc_system.master_binding` (`transform_type <> 'flatten'`) đang có `transform_spec->>'pk'` rỗng (tại `EnsureDefaultMasterConnection` và `NewTransmuterModule`), giữ nguyên `resolveConflictTarget` và `resolveMasterPKFromSpec` chỉ đọc từ `transform_spec.pk` (`L014`).
    2. Trong `RunEvents` (`transmuter.go:766-795`), bỏ nhánh tự bịa `_gpay_id = deterministicSourceGpayID` (SHA-256) khi `shadowGID == 0` và xóa bỏ mảng chắp vá `staleLegacyGpayIDs` ở cả `RunEvents` lẫn `processBatch` (giảm độ phình `transmuter.go`).
    3. Trong `bulkUpsertMaster` (`transmuter.go:1802-1857`), luôn dùng `ON CONFLICT ("_gpay_id") DO UPDATE` (tránh `SQLSTATE 42P10`) và khi `conflictTarget != "_gpay_id"` thì thực thi trong cùng transaction lệnh `DELETE FROM <qt> WHERE <conflictTarget> IN (?) AND "_gpay_id" NOT IN (?)` trước khi `INSERT` (tránh `SQLSTATE 23505` và tự dọn sạch mọi dòng trùng `_id` nhưng lệch `_gpay_id`).
    4. Trong `ExecuteHealModal.tsx:191-204`, bật mặc định `pruneMissingSrc` cho Segment B (`segment === 'shadow_master'`) khi báo cáo còn `orphan_count > 0`.
- **TASK-08** `[DONE]`: Tách bạch hoàn toàn `_gpay_id` (Surrogate ID nội bộ của từng bảng sinh bằng Sonyflake `idgen.NextID()`) khỏi Khóa Nghiệp Vụ (`shadow."_source_id"` $\leftrightarrow$ `master.<transform_spec.pk>`) trên toàn bộ luồng Transmute, Recon Segment B và Heal Segment B.
  - **Context**:
    - `centralized-data-service/internal/service/master/transmuter_utils.go:416-447`
    - `centralized-data-service/internal/service/master/transmuter.go:448-452, 516-521, 717-800, 1079-1138, 1230-1280, 1586-1820`
    - `centralized-data-service/internal/service/recon/recon_engine_segment_b.go:39-65`
    - `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go:592-643, 696-1055`
    - `centralized-data-service/internal/service/recon/recon_tier_b.go:71-278, 364-465, 645-705, 791-840`
    - `centralized-data-service/internal/service/recon/recon_smoke.go:1233-1255`
    - `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:504-685, 785-925`
  - **Logs/Test**: Kiểm chứng `go build ./cmd/... ./internal/... ./pkgs/...` (`EV-20261026T152858-fd8f` `exit=0`) và trace runtime value thật trên PostgreSQL (`EV-20261026T152908-ce2d` `exit=0`) — tuyệt đối không viết/chạy test mockup (`L006`), không nhồi nhét God-file (`L009`), không quét/update toàn bộ DB Master hay chạy `SELECT COUNT(*)` toàn bảng (`L001`), chỉ đọc `pk` từ `transform_spec.pk` (`L015`).
  - **DoD**:
    1. **Sinh `_gpay_id` trên Master bằng Sonyflake (`idgen.NextID()`):** Thêm `nextMasterSonyflakeGpayID()` tại `transmuter_utils.go:416-430` (và `deterministicMasterFlattenGpayID` riêng cho `flatten` tại `L433-447`). Khi 1 bảng Shadow rẽ nhánh ra nhiều bảng Master (`payment_bills`, `payment_bills_1`, ...), mỗi bảng Master tự sinh `_gpay_id` Sonyflake độc lập khi `INSERT` dòng mới và giữ nguyên `_gpay_id` hiện hữu khi `ON CONFLICT (<pk>) DO UPDATE`. Trong `RunEvents` (`transmuter.go:717-800`), bỏ hẳn `lookupShadowGpayIDs`.
    2. **Upsert & Delete trên Master theo Khóa Nghiệp Vụ `conflictTarget` (`transform_spec.pk`):** Trong `bulkUpsertMaster` (`transmuter.go:1730-1820`), dùng trực tiếp `ON CONFLICT (<conflictTarget>) DO UPDATE SET ...` (bỏ qua `_gpay_id` và `conflictTarget` trong `sets`; nếu bảng cũ thiếu unique index trả lỗi `42P10` thì tự tạo `CREATE UNIQUE INDEX IF NOT EXISTS ux_<table>_<conflictTarget>` rồi retry 1 lần), xóa bỏ hoàn toàn khối Transaction `DELETE FROM <master> WHERE <pk> IN (?) AND "_gpay_id" NOT IN (?)`.
    3. **Recon Segment B (`Shadow → Master`) đối soát theo `shadow."_source_id"` $\leftrightarrow$ `master.<transform_spec.pk>`:** Thêm `ResolveSegmentBPKPair` tại `recon_engine_segment_b.go:39-65`, bổ sung select `transform_spec` vào `lookupMasterRef`/`lookupMasterRefExact` (`recon_stream_bucket_engine.go:598-638`), và cập nhật `recon_stream_bucket_engine.go`, `recon_tier_b.go`, `recon_smoke.go` để bên Shadow lấy `_source_id` còn bên Master lấy `transform_spec.pk` (gán `IDType: "id"` trong `StaleIDsPayload`).
    4. **Heal Segment B (`recon_execute_heal_handler.go`) & Cursor `inferLastGpayIDFromMaster` (`transmuter.go:1079-1138`):** Ưu tiên xử lý theo khóa nghiệp vụ `transform_spec.pk` / `_source_id` (tương thích ngược với báo cáo cũ lưu `_gpay_id`), bật ghi đè khi Heal có đích danh `len(chunkIDs) > 0` (`transmuter.go:451`), và bỏ 2 câu `SELECT COUNT(*)` toàn bảng trong `inferLastGpayIDFromMaster`.
- **TASK-09** `[DONE]`: Tách file/module theo đúng trách nhiệm đơn (Single-Responsibility), chuẩn hóa ranh giới Layer (`NewTransmuterModule`) và khôi phục đếm `int(res.RowsAffected)` khi Prune Segment B từ vòng lặp Phản Tỉnh S5 (`S5 → S3 → S4 → S5`).
  - **Context**:
    - `centralized-data-service/internal/service/master/transmuter_identity.go:1-126` (tách từ `transmuter_utils.go`, giảm `776 -> 695` dòng)
    - `centralized-data-service/internal/service/master/transmuter.go:130-145` (giảm `2010 -> 1904` dòng)
    - `centralized-data-service/internal/handler/recon/recon_execute_heal_segb.go:1-576` (tách từ `recon_execute_heal_handler.go`, giảm `932 -> 438` dòng)
    - `cdc-cms-service/internal/api/system/nats_monitor_governance.go:1-549` (tách từ `nats_monitor_handler.go`, giảm `1149 -> 677` dòng)
    - `centralized-data-service/pkgs/natsconn/nats_stream_provision.go:1-194` (tách từ `nats_client.go`, giảm `284 -> 145` dòng)
    - `cdc-cms-service/pkgs/natsconn/nats_stream_provision.go:1-191` (tách từ `nats_client.go`, giảm `287 -> 152` dòng)
  - **Logs/Test**: Kiểm chứng `wc -l` tất cả các file lớn đều giảm dòng so với baseline (`-81`, `-106`, `-494`, `-472`, `-139`, `-135` dòng), `go build ./cmd/... ./internal/... ./pkgs/...` ở cả 2 repo Go (`EV-20261026T160217-0f71`, `EV-20261026T160224-28ce` `exit=0`), `npm run build` ở `cdc-cms-web` (`EV-20261026T160226-0a07` `exit=0`), kiểm chứng runtime PostgreSQL + NATS + Process state (`EV-20261026T160540-7245` `exit=0`), chạy `laya_gate.py` (`DG1` `06_laya_summary_dg1_20261026T155124.json` + `DG3` Mode A `06_laya_summary_dg3_20261026T161006.json` & Mode B `06_laya_summary_dg3review_20261026T161006.json` `PASS`).
  - **DoD**:
    1. Tách `transmuter_identity.go` (`125` dòng) khỏi `transmuter_utils.go` (giảm `transmuter_utils.go` từ `776` xuống `695` dòng, `-81` dòng).
    2. Tách `recon_execute_heal_segb.go` (`575` dòng) khỏi `recon_execute_heal_handler.go` (giảm `recon_execute_heal_handler.go` từ `932` xuống `438` dòng, `-494` dòng) và khôi phục `pruned += int(res.RowsAffected)` ở cả 3 nhánh `DELETE` (`recon_execute_heal_segb.go:223, 296, 310`).
    3. Tách `nats_monitor_governance.go` (`548` dòng) khỏi `nats_monitor_handler.go` (giảm `nats_monitor_handler.go` từ `1149` xuống `677` dòng, `-472` dòng).
    4. Tách `nats_stream_provision.go` khỏi `pkgs/natsconn/nats_client.go` trên cả 2 repo (giảm `nats_client.go` từ `284 -> 145` dòng ở CDS và `287 -> 152` dòng ở CMS).
    5. Gỡ bỏ SQL migration `UPDATE cdc_system.master_binding` khỏi constructor `NewTransmuterModule` (`transmuter.go:130-145`), giữ đúng ranh giới tầng `service` vs `bootstrap` (`cdc-cms-service/internal/bootstrap/master_connection.go:50-77`).

