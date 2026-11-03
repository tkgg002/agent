# 08 Tasks: Lên đồng thời 2 Feature (Event Date Split & Batching + Master MongoDB Destination) trên Codebase `2026-10-28`

- **Workspace**: `audit-cdc-3repos-master-mongo-2026-10-28`
- **Suffix**: `cdc_3repos_master_mongo`
- **Tier**: `T2` (Re-Plan, Stash Audit & Unified Dual-Feature Implementation)

---

- [x] **TASK-AUDIT-01 — Thu thập trạng thái Git (`status`, `diff`, `stash`, `log`) trên 3 repo CDC**
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/data-hub/{centralized-data-service,cdc-cms-service,cdc-cms-web}`.
  - **DoD**: Xác định chính xác 100% các file đang thay đổi trên working tree và trong `stash@{0..2}`.

- [x] **TASK-AUDIT-02 — Bóc tách & Audit 3 Stash (`stash@{0..2}` ngày 22/10) đối chiếu với Codebase `2026-10-28` (`HEAD 786db91`)**
  - **Context (`file:line`)**: So sánh `stash@{0..2}` (base `43e5da7`) với `HEAD 786db91` và working tree hiện tại (`transmuter_events_batch.go`, `transmuter_identity.go`, `recon_stream_bucket_engine.go`, `recon_execute_heal_segb.go`, `recon_dest_query.go`, `recon_tier_b.go`, `source_count_runner.go`).
  - **DoD**: Chỉ rõ toàn bộ 10 lỗi/điểm outdated (`OB-01` → `OB-10`) khiến việc `git stash apply` trực tiếp sẽ gây bug/panic/regression, và lập thiết kế giải pháp + Code Demo hợp nhất trong `09_tasks_solution_cdc_3repos_master_mongo.md`.

- [x] **TASK-M01 — CMS Backend & Web UI: Lưu & hiển thị đầy đủ `auth_source`, `replica_set` cho Master MongoDB Connection (Fix `OB-09`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:604-625, 648-654, 710-725`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx:402-419`
  - **Mô tả**: Bổ sung `auth_source` và `replica_set` vào `options_json` khi `CreateConnection` và `UpdateConnection`; trích xuất lại vào `MasterConnectionRow` trong `ListConnectionsWithFilter`; nạp lại vào state `connForm` khi mở modal `openEditConnection` trên `MasterRegistry.tsx`. Giữ nguyên cải tiến `activity_log_read_repo_gorm.go` đang có trên working tree của `cdc-cms-service`.
  - **Test/Verify**: `go test ./internal/infra/persistence/master/...` và `npm run build` / `tsc --noEmit` trên `cdc-cms-web`.
  - **DoD**: Tạo/Sửa/Liệt kê Master MongoDB connection giữ nguyên 100% `auth_source` và `replica_set`.

- [x] **TASK-M02 — Data-Hub `ConnectionManager`: Quản lý Pool `*mongo.Client` cho Master MongoDB**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:22-56, 218-395`
  - **Mô tả**: Thêm `masterMongoPool`, `masterEngineMap`, `MasterMongoHandle`, `IsMongoMasterConnection(ctx, key)`, `GetMasterMongoClient(ctx, key)`, `buildMasterMongoURI(row)`, cập nhật `InvalidateConnection` và `Close()` đóng cả `masterMongoPool`. Trong `GetMasterDB(ctx, key)`, chặn sớm khi `engine_type == "mongodb"` và trả lỗi tường minh.
  - **Test/Verify**: `go test -v ./internal/service/source/...`
  - **DoD**: `ConnectionManager` phân biệt chính xác PostgreSQL vs MongoDB Master connection, cache thread-safe và hỗ trợ `InvalidateConnection`.

- [x] **TASK-M03 — Data-Hub `MasterDDLGenerator`: Tạo Collection & Indexes chuẩn trên MongoDB Master (Fix `OB-08`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:380-405, 480-560`
  - **Mô tả**:
    - Trong `ExecutePendingDDL` và `EnsureDynamicIndexes`: rẽ nhánh `if g.connMgr.IsMongoMasterConnection(ctx, reg.MasterConnectionKey)` sang `applyMongoCollectionAndIndexes(ctx, reg)`.
    - Tạo collection (idempotent) và tạo các index chuẩn: `idx_master_gpay_id_uniq` (`_gpay_id` unique), `idx_master_source_id` (`_source_id`), `idx_master_source_ts` (`_source_ts`), `idx_master_deleted` (`_deleted`), cộng thêm **index trên `conflict_target`** (khi `conflict_target != ""` và `conflict_target != "_gpay_id"` và `conflict_target != "_id"`) để tránh `COLLSCAN` khi `BulkWrite` upsert theo business PK (`OB-08`).
    - Khi đánh dấu `in_master = true`, bắt buộc giữ điều kiện `WHERE master_binding_id = ? AND is_active = true AND status = 'approved'` (`OB-08`).
  - **Test/Verify**: `go test -v ./internal/service/master/... -run TestMasterDDL`
  - **DoD**: Không bao giờ chạy SQL `CREATE TABLE` lên MongoDB Master; tạo đủ index chuẩn + `conflict_target` index; chỉ đánh dấu `in_master = true` cho rule `approved`.

- [x] **TASK-M04 — Data-Hub `mongo_type_converter.go`: Chuyển đổi kiểu dữ liệu chuẩn BSON cho MongoDB Master**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_type_converter.go` (file mới)
  - **Mô tả**: Xây dựng bộ chuyển đổi `ConvertValueForMongoMaster(val, targetType)` hỗ trợ đầy đủ `objectid`, `decimal128` (`primitive.ParseDecimal128`), `date`/`datetime`/`timestamp`/`timestamptz` (`time.Time` -> BSON Date), `int`/`bigint`/`int64`, `double`/`float64`, `bool`/`boolean`, `object`/`json`/`jsonb`/`map`, `array`, `string`/`text`.
  - **Test/Verify**: Unit test đầy đủ các kiểu dữ liệu BSON trong `transmuter_mongo_test.go`.
  - **DoD**: Convert chính xác 100% kiểu dữ liệu từ Shadow sang BSON, không làm mất độ chính xác số học (`Decimal128`) hay kiểu `Date`.

- [x] **TASK-M05 & TASK-M06 — Data-Hub `TransmuterModule`: Hợp nhất Feature 1 (`runEventBatches` + `transmuter_identity.go`) với Feature 2 (`transmuter_mongo.go`) & Fix `OB-06`, `OB-07`**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:77-96, 265-360, 468-525, 692-725, 1080-1120, 1430-1580`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_events_batch.go:1-326` (giữ nguyên Feature 1)
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_identity.go:1-125` (giữ nguyên Sonyflake & `deterministicMasterFlattenGpayID`)
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go` (file mới)
  - **Mô tả**:
    - **Bảo toàn 100% Feature 1:** Giữ nguyên `transmuter_events_batch.go` và `transmute_event_date_split.go`, giữ `RunEvents` gọi `t.runEventBatches(...)` (`transmuter.go:701`).
    - **Bảo toàn `transmuter_identity.go` (`OB-06`):** Giữ nguyên `deterministicMasterFlattenGpayID(masterRow.ID, row.SourceID, row.GpayID, suffix)` ở orphan cleanup (`transmuter.go:296`) và `nextMasterSonyflakeGpayID()`.
    - **Bảo toàn `_gpay_id` trên MongoDB Master khi update (`OB-07`):** Trong `BuildMongoOCCBulkModels` (`transmuter_mongo.go`), khi `conflictTarget != "_gpay_id"`, trường `_gpay_id` trong pipeline update dùng `bson.M{"$ifNull": bson.A{"$_gpay_id", v}}` (chỉ gán khi insert mới, không ghi đè `_gpay_id` cũ khi update), và ở nhánh non-OCC (`isFullSync || incomingTs == 0`) tách `_gpay_id` sang `"$setOnInsert": bson.M{"_gpay_id": gpayIDVal}`!
    - **Gap-Safe Guard (`AF-DH-05`):** Không advance `watermark` khi `totalSourceRows > 0 && totalUpserted == 0` hoặc khi có chunk lỗi; trong `runMasterPipeline` (`transmuter.go:468-520`), với MongoDB Master lấy `existingCount` bằng `t.countMongoMasterActiveDocs(ctx, masterRow)` để bật `isFullSync` khi collection đang rỗng (`existingCount == 0`).
  - **Test/Verify**: `go test -v ./internal/service/master/... ./internal/handler/master/...`
  - **DoD**: Cả 2 luồng `RunByTable` (batch) và `RunEvents` (`runEventBatches` date-split) đều ghi xuống PostgreSQL Master và MongoDB Master chuẩn xác, chống loạn thứ tự bằng OCC pipeline update (`$_source_ts <= incomingTs`) và giữ nguyên `_gpay_id` của row đã tồn tại.

- [x] **TASK-M07 — Data-Hub `ReconDestAgent` & `ChunkStreamBucketEngine` & `ReconCore`: Hỗ trợ đầy đủ MongoDB Master trên mọi luồng Recon Segment B (Fix `OB-01`, `OB-02`, `OB-04`, `OB-05`, `OB-10`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go:16-95`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_mongo.go` (file mới)
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go:16-680`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go:24-280`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go:174-225`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go:134-176`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go:612-676, 774-848`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go:165-176`
  - **Mô tả**:
    - Bổ sung `NewReconDestMongoAgent` và `da.IsMongo()`.
    - Cập nhật **CẢ HAI** `ReconCore.GetMasterAgent` (`recon_engine.go:174`) VÀ `ChunkStreamBucketEngine.GetMasterAgent` (`recon_stream_bucket_engine.go:134`) để kiểm tra `connMgr.IsMongoMasterConnection(ctx, connectionKey)` và khởi tạo `NewReconDestMongoAgent` (`OB-01`).
    - Bổ sung nhánh `if da.IsMongo()` cho **TẤT CẢ** các hàm của `ReconDestAgent`: `CountRows`, `CountDeletedRows`, `ColumnExists`, `GetRealColumnName`, `EstimatedCountRows`, `CountInWindow`, `CountRecentDeletedRows`, `BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs`, `HashWindow`, `BucketHash` (`OB-01`, `OB-02`).
    - Trong `mongoCountInWindow` (`recon_dest_mongo.go`), truyền `bson.M{"_deleted": bson.M{"$ne": true}}` để không đếm nhầm document đã xóa mềm (`OB-04`), và hỗ trợ linh hoạt BSON Date / int64 epoch ms trong `buildMongoWindowFilter` & `extractMongoDocIDAndTsMs` (`OB-05`).
    - Trong `RunRowDiffB` và `TimeBoundedDiffMissingFromMaster` (`recon_tier_b.go:612, 774`) và `SourceCountRunner` (`source_count_runner.go:165`), bổ sung nhánh MongoDB Master (`OB-02`, `OB-10`).
  - **Test/Verify**: `go test -v ./internal/service/recon/... ./internal/service/sourcecount/...`
  - **DoD**: Smoke Recon, ChunkStreamBucketEngine (`2h/7d/30d`), DeepCheckB (`BucketCounts`), RowDiffB và Daily Source Count đều chạy mượt trên MongoDB Master, không bao giờ gọi nhầm SQL trên `da.replica == nil`.

- [x] **TASK-M08 — Data-Hub `ReconExecuteHealHandler`: Tự động Heal Segment B (`executeHealSegB` & `resolveSourceIDsForSegmentB`) cho MongoDB Master (Fix `OB-03`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb.go:95-150, 340-360, 520-555`
  - **Mô tả**:
    - Đặt `isMongoMasterRef(ctx, ref)` và `pruneMongoMasterMissingFromSrc(ctx, rpt, ref, shadowRel, staleB.MissingFromSrc, idType, masterBindingID)` đúng vào kiến trúc tách file mới (`recon_execute_heal_segb.go`).
    - Khi prune `MissingFromSrc` trong `executeHealSegB` (`recon_execute_heal_segb.go:100`), nếu `h.isMongoMasterRef(ctx, ref)` thì gọi `h.pruneMongoMasterMissingFromSrc(...)` (kèm bước kiểm tra `activeShadowIDs` để resync qua `publishTransmuteChunked` thay vì xóa nhầm bản ghi đang sống).
    - Trong `resolveSourceIDsForSegmentB` (`recon_execute_heal_segb.go:528-546`, khi `effectiveIDType == "gpay"` và row đã bị xóa cứng khỏi Shadow), nếu `h.isMongoMasterRef(ctx, ref)` thì truy vấn MongoDB Master theo `_gpay_id` để phân giải business PK (`pkCol`), tuyệt đối không gọi `resolveMasterDBForRef` (PostgreSQL).
  - **Test/Verify**: `go test -v ./internal/handler/recon/...`
  - **DoD**: Heal Segment B (`MissingFromDest`, `Mismatched`, `MissingFromSrc`) hoạt động chính xác 100% cho cả PostgreSQL Master và MongoDB Master.

- [x] **TASK-M09 & TASK-M10 — Bộ Test Tích hợp & Concurrency (`transmuter_mongo_test.go`, `recon_dest_mongo_test.go`) + Kiểm định S5 (`audit_report_*.md`, `DG3`, `verify_governance.py`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo_test.go`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_mongo_test.go`
  - **Mô tả**: Viết bộ test kiểm chứng: (1) Type conversion BSON; (2) OCC Aggregation Pipeline chống ghi đè lùi và bảo toàn `_gpay_id` khi update theo business PK; (3) Soft delete `_deleted = true`; (4) Recon hash/bucket/window/column-exists trên MongoDB Master; (5) Heal Segment B và Event Batching (`runEventBatches`) với MongoDB Master.
  - **Test/Verify**: Chạy qua `evidence_run.py` lấy biên lai `EV-...` thật, chạy `laya_gate.py run --stage dg3` và `verify_governance.py`.
  - **DoD**: Toàn bộ test suite PASS 100%, không regression sang Feature 1 hay PostgreSQL Master.

- [x] **TASK-M11 — Fix Single-Host Master MongoDB ReplicaSet Discovery (`directConnection=true` Normalization) ở `cdc-cms-service` và `centralized-data-service`**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_connections.go:37-78, 139-231`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:892-928`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go:128-249`
  - **Mô tả**:
    - Container `gpay-mongo` (`0.0.0.0:17017->27017/tcp`) cấu hình ReplicaSet `rs0` với member internal hostname là `gpay-mongo:27017`. Khi `go.mongodb.org/mongo-driver v1.17.9` kết nối tới URI single-host `mongodb://localhost:17017/devtest?connectTimeoutMS=5000` (hoặc bất kỳ single-host `mongodb://` nào không có `directConnection=true`), theo chuẩn SDAM driver tự động nhận diện `setName: "rs0"` từ phản hồi `hello`, chuyển topology sang `ReplicaSetNoPrimary`, loại bỏ seed `localhost:17017` và dial sang `gpay-mongo:27017` → nổ lỗi DNS `lookup gpay-mongo: no such host` trên máy host.
    - Bổ sung `normalizeMasterMongoURI` và cập nhật `buildMasterConnectionDSN` tại `cdc-cms-service` + `resolveMasterMongoURI` tại `centralized-data-service`: tự động gắn `directConnection=true` cho mọi URI `mongodb://` đơn host (`!strings.Contains(host, ",")`) khi chưa chỉ định `directConnection` (đồng thời hỗ trợ format multi-host `h1:p1,h2:p2` không gắn `directConnection=true`), giúp cả tạo mới lẫn test/sử dụng bản ghi đã lưu (`id=406`, `mongo_local`) hoạt động tức thì.
  - **Test/Verify**: Unit test cho `buildMasterConnectionDSN`, `normalizeMasterMongoURI` trên cả `cdc-cms-service` và `centralized-data-service` + kiểm tra ping thực tế tới `localhost:17017`.
  - **DoD**: `POST /api/v1/master-connections/test` và `GetMasterMongoClient` kết nối thành công tới `localhost:17017` (`mongo_local`) với latency < 50ms, trả về `MongoDB 6.0.x`, không còn lỗi `lookup gpay-mongo: no such host`.

- [x] **TASK-M12 — Xử lý lỗi thiếu quyền `REPLICATION` trên PostgreSQL Source Connector (`devcoretrans`) & Bổ sung Preflight Check tại `cdc-cms-service`**
  - **Context (`file:line`)**:
    - PostgreSQL Server `10.200.187.110:5432`, DB `core-trans-proxy-history-service`, bảng `pg_roles` (`rolname = 'core-trans-proxy-history-user'`, `rolreplication = false`, `rolsuper = false`).
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go:150-153, 209-211, 437-456`
  - **Mô tả**:
    - Trên DB `10.200.187.110:5432`, `wal_level` đã là `logical` và `core-trans-proxy-history-user` là owner (`datdba`) của DB `core-trans-proxy-history-service` cùng 4 bảng trong schema `public`, nhưng trong `pg_roles` thì `core-trans-proxy-history-user` đang có `rolreplication = f` (`false`). Do đó khi Debezium (`PostgresConnectorTask.tryToCreateSlot`) mở kết nối replication để tạo slot `devcoretrans_slot`, PostgreSQL từ chối với lỗi `FATAL: permission denied to start WAL sender Detail: Only roles with the REPLICATION attribute may start a WAL sender process`.
    - Hướng xử lý hạ tầng DB: Chạy lệnh `ALTER ROLE "core-trans-proxy-history-user" WITH REPLICATION;` bằng tài khoản superuser `postgres` trên `10.200.187.110:5432` rồi restart task của connector `devcoretrans` (hoặc tạo sẵn slot/cấp quyền cho role `replicator` đang có `rolreplication = t`).
    - Hướng xử lý code tại `cdc-cms-service`: Bổ sung `validatePostgresConnectorPrereqs(ctx, req.Config)` trong `system_connectors_handler.go` kiểm tra nhanh `rolreplication`/`rolsuper` và `wal_level = 'logical'` khi tạo/cập nhật PostgreSQL Source Connector để cảnh báo/trả về thông báo lỗi kèm câu lệnh SQL khắc phục rõ ràng thay vì để task crash ngầm trên Kafka Connect.
  - **Test/Verify**: `go test -v ./internal/api/source/...`
  - **DoD**: Kiểm tra rõ ràng điều kiện `REPLICATION` + `wal_level=logical` và connector `devcoretrans` chuyển sang `RUNNING` sau khi role được cấp `REPLICATION`.

- [x] **TASK-M13 — Bổ sung Check Connection (Kiểm tra điều kiện Debezium) & Quét danh sách Table theo Schema cho PostgreSQL Connector trong Modal New Connect (`cdc-cms-service` & `cdc-cms-web`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler_postgres.go` (file mới tách đơn trách nhiệm trong `package system`)
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go:343-347`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go:470-512`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/services/connectorCheck.ts:1-74`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useConnectorCheck.ts:1-52`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx:463-540, 1423-1432, 1727-1754`
  - **Mô tả**:
    1. **Backend (`cdc-cms-service`):**
       - Tạo file mới `internal/api/system/introspection_handler_postgres.go` cài đặt `DiscoverPostgresTablesPost` (`POST /api/introspection/postgres/tables` & `/api/v1/introspection/postgres/tables`), nhận `{ host, port, username, password, database, schemas }`.
       - Thực hiện kiểm tra tuần tự:
         - Kết nối & xác thực PostgreSQL (`connect_timeout=5`), phân loại lỗi chính xác: `auth_err` (`28P01`/`password authentication failed`), `db_missing` (`3D000`/`does not exist`), `cluster_err` (lỗi mạng/dial).
         - Kiểm tra `SHOW wal_level`: nếu `wal_level != 'logical'` trả về `status: "wal_level_err"` kèm câu lệnh `ALTER SYSTEM SET wal_level = 'logical';`.
         - Kiểm tra `SELECT rolsuper, rolreplication FROM pg_roles WHERE rolname = current_user LIMIT 1`: nếu `!rolsuper && !rolreplication` trả về `status: "replication_err"` kèm câu lệnh `ALTER ROLE "<user>" WITH REPLICATION;`.
         - Kiểm tra danh sách schema (`schemas`, mặc định `public`): nếu schema không tồn tại trong `information_schema.schemata`, trả về `status: "schema_missing"` kèm `available_schemas`.
         - Quét danh sách bảng (`relkind IN ('r', 'p') AND NOT relispartition`) trong các schema đã chọn từ `pg_catalog.pg_class` + `pg_catalog.pg_namespace`, trả về mảng `tables: ["schema.table1", ...]` sắp xếp tăng dần (`status: "ok"` hoặc `"empty"` nếu chưa có bảng).
       - Cập nhật `validatePostgresConnectorPrereqs` tại `internal/api/source/system_connectors_handler.go` kiểm tra thêm `SHOW wal_level` (`wal_level == 'logical'`) bên cạnh `rolsuper || rolreplication`.
    2. **Frontend (`cdc-cms-web`):**
       - Mở rộng `src/services/connectorCheck.ts` (`CheckPostgresTablesResponse`, `checkPostgresTables`, `mapPostgresCheckStatusToVi`) và `src/hooks/useConnectorCheck.ts` (`useCheckPostgresConnection`).
       - Trong `src/pages/SourceConnectors.tsx`:
         - Khai báo `Form.useWatch` cho `host`, `port`, `username`, `password`, `schemaIncludeList` và `useCheckPostgresConnection()` ở top-level component (tuân thủ `L042`).
         - Tự động chuyển `port` mặc định sang `5432` khi đổi `dbKind` sang `'postgresql'`.
         - Trong khối `dbKind === 'postgresql'`: hiển thị `Schema Include List` (mặc định `public`), nút **Check Connection & Quét Table**, `<Alert>` hiển thị kết quả kiểm tra Debezium (`wal_level=logical`, `REPLICATION` privilege, số bảng tìm thấy hoặc câu lệnh SQL fix lỗi copyable), và chuyển `Table Include List` sang `<Select mode="multiple">` tự động chọn tất cả bảng `schema.table` khi Check PASS.
         - Khóa nút `Create` ở chế độ tạo mới (`editorMode === 'create' && dbKind === 'postgresql'`) cho tới khi `pgCheckHook.result?.status === 'ok'`.
  - **Test/Verify**: `go test -v ./internal/api/system/... ./internal/api/source/...` và `npm run build` trên `cdc-cms-web`.
  - **DoD**: Nút Check Connection cho PostgreSQL kiểm tra chính xác điều kiện Debezium + quét danh sách `schema.table` theo schema vừa nhập, build FE/BE PASS 100%.


