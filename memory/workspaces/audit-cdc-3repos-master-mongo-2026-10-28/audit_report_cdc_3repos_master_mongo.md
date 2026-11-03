# Audit & Validation Report: Hợp Nhất Đồng Thời 2 Feature (Event Date Split & Batching + Master MongoDB Destination) trên 3 Repo CDC

- **Workspace**: `audit-cdc-3repos-master-mongo-2026-10-28`
- **Suffix**: `cdc_3repos_master_mongo`
- **Tier**: `T2`
- **Timestamp**: `2026-10-28T12:39:30+07:00`

---

## (a) Line-by-Line Diff Critique & Task Traceability (Gate G1, G7)

Đã chạy `git status -s` và `git diff` thực tế trên cả 3 repo (`/Users/trainguyen/Documents/work/data-hub/{centralized-data-service,cdc-cms-service,cdc-cms-web}`) và đối chiếu từng dòng thay đổi với `09_tasks_solution_cdc_3repos_master_mongo.md` & `08_tasks_cdc_3repos_master_mongo.md`:

1. **`cdc-cms-service` (`8 files changed, 452 insertions(+), 92 deletions(-)`) & `cdc-cms-web` (`1 file changed, 115 insertions(+), 36 deletions(-)`) — `TASK-M01` + Feature 1 (`activity_log_read_repo_gorm.go`):**
   - `internal/domain/master/binding.go:68-71`: Bổ sung `OptionsJSON`, `SSLMode`, `AuthSource`, `ReplicaSet` vào `MasterConnectionItem`.
   - `internal/api/master/master_registry_handler_connections.go:255-268, 376-389`: Gán `SSLMode`, `AuthSource`, `ReplicaSet` vào `MasterConnectionItem` khi `CreateConnection` và `UpdateConnection`.
   - `internal/infra/persistence/master/master_repo_gorm.go:753-774, 801-835, 868-895`: Bổ sung `populateConnectionItemOptions(item)` parse `options_json` ra `sslmode`, `auth_source`, `replica_set` trong `ListConnectionsWithFilter` và `GetMasterConnectionByID`; lưu đầy đủ `auth_source` và `replica_set` vào `options_json` khi tạo/cập nhật kết nối (`OB-09`).
   - `internal/infra/persistence/system/activity_log_read_repo_gorm.go`: Giữ nguyên 100% logic thống kê `event_date` và tách riêng `master_deleted` khỏi `master_updated` của Feature 1.
   - `cdc-cms-web/src/pages/MasterRegistry.tsx:402-419`: Khai báo `sslmode`, `auth_source`, `replica_set` trong `MasterConnectionItem` và nạp vào `connForm` khi mở `openEditConnection` (`OB-09`).

2. **`centralized-data-service` (`18 tracked files changed + 12 new files`) — Feature 1 + Feature 2 (`TASK-M02` → `TASK-M10`):**
   - **Feature 1 (`internal/handler/master/transmute_event_date_split.go`, `internal/service/master/transmuter_events_batch.go`, `internal/handler/shadow/batch_buffer_event_date.go`, `internal/model/shadow/cdc_event_date.go`):** Bảo toàn nguyên vẹn 100% cơ chế gom batch sự kiện theo `(masterBindingID, triggeredBy, eventDate)` giờ Hồ Chí Minh (`UTC+7`) và tách batch qua mốc nửa đêm mà không cần `git stash apply`.
   - **`TASK-M02` (`internal/service/source/connection_manager.go:66-165, 251-300, 415-437`):** Bổ sung `masterMongoPool`, `masterMongoDBs`, `masterEngines` cache map, `IsMongoMasterConnection`, `GetMasterMongoClient`, `ResolveMasterMongoDatabase`, `InvalidatePool` xóa sạch `masterEngines`, và chốt chặn sớm trong `GetMasterDB` từ chối kết nối MongoDB Master với thông báo tường minh.
   - **`TASK-M03` (`internal/service/master/master_ddl_generator.go:380-415, 485-583, 815-915`):** Tạo collection và bộ index chuẩn (`ux_<table>_gpay_id` unique, `ix_<table>_source_id`, `ix_<table>_source_ts`, `ix_<table>_deleted`, index động trên `transform_spec.pk` `ix_<table>_pk_<pk>`, và financial columns), đồng thời thêm `AND status = 'approved'` khi đánh dấu `in_master = true` (`OB-08`) và hỗ trợ `ReconcileColumn` (`$rename`), `DropColumn` (`$unset`), `DropTable` (`Collection.Drop`).
   - **`TASK-M04` (`internal/service/master/mongo_type_converter.go:1-214`):** Chuyển đổi chính xác UUID 16-byte BinData sang chuỗi `8-4-4-4-12`, `ObjectId` 24-hex sang `primitive.ObjectID`, `DECIMAL/NUMERIC` sang `primitive.Decimal128`, `TIMESTAMPTZ` sang `primitive.DateTime`, và `JSONB` sang `bson.M`/`bson.A`.
   - **`TASK-M05` & `TASK-M06` (`internal/service/master/transmuter_mongo.go:1-440`, `transmuter.go`, `transmuter_events_batch.go`):**
     - Hợp nhất `IsMongoDestination` vào cả `Run`, `RunEvents` (`runEventBatches`), `processBatch`, `deleteFromMasterBySourceAndGpayIDs`, `bulkUpsertMaster`.
     - Trong `BuildMongoOCCBulkModels` (`transmuter_mongo.go:246-280`), dùng Aggregation Pipeline Update `[]bson.M{{"$set": setStage}}` với `$cond` trên `$_source_ts <= incomingTs + 2000` (khắc phục triệt để lỗi `E11000 duplicate key` khi upsert loạn thứ tự), và khi `targetKey != "_gpay_id"` thì dùng `"$ifNull": ["$_gpay_id", v]` (chế độ OCC) hoặc `"$setOnInsert"` (chế độ full-sync) để không bao giờ ghi đè `_gpay_id` hiện có trên Master (`OB-07`).
     - Khôi phục thứ tự chuẩn của `inferLastGpayIDFromMaster` (`transmuter.go:997-1075`) kết hợp tiền tố liên tục (Contiguous Prefix) và Gap-Safe Count Guard.
   - **`TASK-M07` (`internal/service/recon/recon_dest_mongo.go`, `recon_dest_agent.go`, `recon_dest_query.go`, `recon_dest_hash.go`, `recon_engine.go`, `recon_stream_bucket_engine.go`, `recon_tier_b.go`, `internal/service/sourcecount/source_count_runner.go`):**
     - Cập nhật cả `ReconCore.GetMasterAgent` lẫn `ChunkStreamBucketEngine.GetMasterAgent` khởi tạo `NewReconDestMongoAgent` (`OB-01`).
     - Bổ sung đầy đủ các hàm MongoDB trên `ReconDestAgent` (`ColumnExists`, `GetRealColumnName`, `BucketCounts`, `mongoFetchRowsByPK`, `CountInWindow` lọc `_deleted: {$ne: true}`, `buildMongoWindowFilter` hỗ trợ cả BSON `DateTime` lẫn `int64` epoch-ms) (`OB-02`, `OB-04`, `OB-05`, `OB-10`).
   - **`TASK-M08` (`internal/handler/recon/recon_execute_heal_segb.go:100-112, 350-685, 876-885`):**
     - Bổ sung `isMongoMasterRef`, `pruneMongoMasterMissingFromSrc` (kiểm tra `activeShadowIDs` trên Shadow trước khi xóa để resync các bản ghi còn sống qua `publishTransmuteChunked` thay vì xóa nhầm, và xóa cứng orphan thật sự bằng `DeleteMany`), cùng `resolveSourceIDsFromMongoMasterByGpayIDs` trong `resolveSourceIDsForSegmentB` (`OB-03`).

---

## (b) Anti-Smoke-Test Speculation & Concurrency/Edge-Case Critique (Gate G4)

1. **Tương tác Đa Worker (`WorkerCount > 1`) & Loạn thứ tự CDC (Out-of-Order Events):**
   - Trên MongoDB Master, nếu dùng filter `{"_gpay_id": id, "$or": [{"_source_ts": {"$lte": ts}}, ...]}` với `SetUpsert(true)` như bản cũ chưa chuẩn, khi một worker mang event cũ đến sau, filter không khớp document hiện có nên MongoDB cố `INSERT` mới và nổ lỗi `E11000 duplicate key error`.
   - Giải pháp đã triển khai trong `BuildMongoOCCBulkModels` (`transmuter_mongo.go:235-260`) đặt filter chỉ chứa duy nhất `{targetKey: keyVal}` và chuyển điều kiện OCC `$_source_ts <= incomingTs + 2000` vào biểu thức `$cond` bên trong Aggregation Pipeline `[]bson.M{{"$set": setStage}}`. Nhờ đó lệnh `UpdateOne` luôn tìm thấy document theo khóa duy nhất, cập nhật nguyên tử (atomic) trên cấp độ document của WiredTiger mà không bao giờ sinh lỗi `E11000`, đồng thời giữ nguyên giá trị mới hơn khi event cũ đến trễ.
   - Đã kiểm chứng thực tế với 8 goroutine chạy song song trên 50 bản ghi (`TestTransmuteMongoMaster_MultiWorkerConcurrentOCC` trong `transmuter_mongo_test.go`).
2. **Gom lô (`batch/debouncer`) & Tách Ngày Sự Kiện (`event_date`):**
   - Trong `transmute_event_date_split.go` và `transmuter_events_batch.go`, khi một mẻ debounce có các sự kiện vắt qua mốc `00:00:00` giờ Hồ Chí Minh (`UTC+7`), mẻ được tách thành các nhóm con đồng nhất theo `(masterBindingID, triggeredBy, eventDate)` nhưng giữ một `taskAckTracker` dùng chung (`atomic.Int32` `remaining` + `failed`). Tin nhắn NATS chỉ được `Ack()` khi **tất cả** các nhóm con của nó đã hoàn tất thành công; nếu bất kỳ nhóm con nào lỗi thì `NakWithDelay` một lần duy nhất, ngăn chặn mất dữ liệu khi crash giữa chừng.
3. **Tính liên tục của `Watermark/Cursor` sau sự cố Outage (`inferLastGpayIDFromMaster` & `inferLastGpayIDFromMongoMaster`):**
   - Khi hệ thống vừa hồi phục sau Outage, một vài event realtime mới (`s9, s10`) có thể được ghi vào Master qua `RunEvents` trước khi job `Run` (incremental) chạy. Nếu chỉ lấy `MAX(_gpay_id)` theo `_source_ts DESC`, cursor sẽ nhảy vọt qua vùng trũng Outage (`s4..s8`) gây mất dữ liệu vĩnh viễn.
   - Cả `inferLastGpayIDFromMaster` (`transmuter.go:997-1075`) và `inferLastGpayIDFromMongoMaster` (`transmuter_mongo.go:64-135`) đều áp dụng **Gap-Safe Count Guard**: chỉ nâng cursor lên `bySourceID` khi chứng minh 100% các dòng trong khoảng `(inferred, bySourceID]` đã tồn tại trên Master và `remAfterCand >= (activeShadowRows - masterCount)`; nếu phát hiện vùng trũng Outage thì dừng lại ở mốc liên tục cuối cùng trước vùng trũng (`300` trong `TestInferLastGpayIDFromMaster`).

---

## (c) Zero-Mockup & Real Runtime Verification Receipts (`EV-...`, Gate G2, G3, G5, G6, G8)

Toàn bộ lệnh kiểm định và biên dịch đã được thực thi qua `evidence_run.py` sau lần sửa code cuối cùng:

1. **Biên lai `EV-20261028T123458-6b46` (`exit=0`, `kind=unit`, repo `centralized-data-service`):**
   - Lệnh: `go test -p 1 -race -v ./internal/service/master/... ./internal/service/recon/... ./internal/handler/master/... ./internal/handler/recon/... ./internal/service/sourcecount/...`
   - Kết quả: `PASS` 100% các test suite (bao gồm cả PostgreSQL thực tế trong `TestTransmuter_RunEvents_RealPostgres_DedupWrittenAndAccurateDeleteCount`, `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres`, `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift`, `TestLivePostgres_MasterBindingRepoAndReconCoreAndPostIngestSchedule_ZeroDrift`, `TestTransmuteMongoMaster_MultiWorkerConcurrentOCC`, `TestBuildMongoOCCBulkModels_PreservesGpayIDWhenConflictTargetIsBusinessPK`, `TestBuildMongoWindowFilter_SupportsBothDateAndEpochMs`).
2. **Biên lai `EV-20261028T123542-4542` (`exit=0`, `kind=unit`, repo `centralized-data-service`):**
   - Lệnh: `go test -race -v ./internal/service/source -run TestConnectionManager`
   - Kết quả: `PASS` (`TestConnectionManager_IsMongoMasterConnection_AndGetMasterDBRejectsMongo`).
3. **Biên lai `EV-20261028T123544-723a` (`exit=0`, `kind=runtime`, repo `centralized-data-service`):**
   - Lệnh: `go build ./cmd/... ./internal/...`
   - Kết quả: Build thành công toàn bộ binary và internal packages (`exit=0`).
4. **Biên lai `EV-20261028T123419-09b8` (`exit=0`, `kind=unit`, repo `cdc-cms-service`):**
   - Lệnh: `go test -v ./internal/api/master/... ./internal/infra/persistence/master/... ./internal/infra/persistence/system/...`
   - Kết quả: `PASS` (`TestPopulateConnectionItemOptions_ParsesMongoAndPostgresOptions` và `TestGetDailyFlow_RealPostgres_ExportJobsPoisonPillAndMasterDeleted` trên PostgreSQL thật).
5. **Biên lai `EV-20261028T123422-e956` (`exit=0`, `kind=runtime`, repo `cdc-cms-service`):**
   - Lệnh: `go build ./cmd/... ./internal/...`
   - Kết quả: Build thành công toàn bộ binary và internal packages (`exit=0`).
6. **Biên lai `EV-20261028T123426-4dcc` (`exit=0`, `kind=runtime`, repo `cdc-cms-web`):**
   - Lệnh: `npx tsc --noEmit`
   - Kết quả: Type-check TypeScript toàn bộ frontend `cdc-cms-web` sạch lỗi (`exit=0`).
7. **Biên lai `EV-20261028T154023-1768` & `EV-20261028T154111-860b` (`exit=0`, `kind=integration`, repo `cdc-cms-service` — `TASK-M11` & `TASK-M12`):**
   - Lệnh: `go test -v ./internal/api/master/... ./internal/api/source/... ./internal/infra/persistence/master/...`
   - Kết quả: `TestBuildMasterConnectionDSN_AndNormalizeMongoURI` (kèm kết nối thực và `client.Ping(ctx, readpref.Primary())` tới `gpay-mongo` trên `localhost:17017/devtest`), `TestValidatePostgresConnectorPrereqs` và `TestPopulateConnectionItemOptions_ParsesMongoAndPostgresOptions` đều `PASS` (`0.11s` / `0.02s`).
8. **Biên lai `EV-20261028T154046-78ea` (`exit=0`, `kind=unit`, repo `centralized-data-service` — `TASK-M11`):**
   - Lệnh: `go test -v -run TestConnectionManager ./internal/service/source/...`
   - Kết quả: `TestConnectionManager_IsMongoMasterConnection_AndGetMasterDBRejectsMongo` kiểm chứng `normalizeMasterMongoURI` tự động gắn `directConnection=true` cho single-host và giữ nguyên multi-host (`PASS`).
9. **Biên lai `EV-20261028T154136-2586` (`exit=0`, `kind=runtime`):**
   - Kiểm tra tiến trình đang lắng nghe trên máy (`:8090` / `go-build` procesos).
10. **Biên lai `EV-20261028T163005-2e22` & `EV-20261028T163025-adf6` (`exit=0`, `kind=unit`):**
    - `go test -v -run TestDiscoverPostgresTablesPost ./internal/api/system/...` & `go test -v -run TestParsePostgresSchemaList ./internal/api/system/...` (`cdc-cms-service`): PASS 100% (xác thực `DiscoverPostgresTablesPost`, phân loại lỗi `auth_err`, `db_missing`, `cluster_err`, kiểm tra `SHOW wal_level` trên PostgreSQL thực tế `localhost:5433`, kiểm tra `pg_roles` và quét bảng `schema.table`).
11. **Biên lai `EV-20261028T163011-cbf1` (`exit=0`, `kind=unit`):**
    - `go test -v ./internal/api/source/...` (`cdc-cms-service`): `TestValidatePostgresConnectorPrereqs` PASS 100% (kiểm tra cả `wal_level = logical` và `rolsuper || rolreplication`).
12. **Biên lai `EV-20261028T163030-d6c2` (`exit=0`, `kind=runtime`):**
    - `go build ./cmd/... ./internal/...` (`cdc-cms-service`): PASS 100%.
13. **Biên lai `EV-20261028T162854-4740` (`exit=0`, `kind=unit`):**
    - `npm run build` (`cdc-cms-web`): `tsc -b && vite build` PASS (`built in 584ms`, bundle `SourceConnectors-MiqBH-Ju.js` `45.65 kB`, không lỗi TypeScript hay React Rules of Hooks).

---

## (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 → S3 → S4 → S5`)

Trong quá trình chạy kiểm định thực tế ở bước S4/S5, đã phát hiện và xử lý dứt điểm 3 điểm giao thoa tinh vi giữa commit `HEAD 786db91` và bộ tính năng mới:
1. Khi `conflictTarget == "_gpay_id"` (bảng Master `copy_1_to_1` không khai báo business PK trong `transform_spec.pk`), nếu sinh `nextMasterSonyflakeGpayID()` mới cho mỗi lần chạy batch thì `ON CONFLICT ("_gpay_id")` sẽ chèn dòng trùng thay vì cập nhật dòng cũ. Đã sửa tại `transmuter.go:1189-1193` để giữ `row.GpayID` khi `conflictTarget == "_gpay_id" && row.GpayID > 0`, và chỉ sinh `nextMasterSonyflakeGpayID()` khi `conflictTarget != "_gpay_id"`.
2. Trong `deleteFromMasterBySourceAndGpayIDs` (`transmuter.go:1505-1514`), tránh gọi lại `t.loadRules(ctx, binding)` xuống DB khi `colTypes` đã xác nhận `pkCol` nằm trong mapping rules và khớp `SourcePK`.
3. Trong `inferLastGpayIDFromMaster` (`transmuter.go:997-1075`), khôi phục bước suy diễn `inferred` từ `candGpayIDs` trước khi kiểm tra `bySourceID` kèm tiền tố liên tục (Contiguous Prefix), đảm bảo không bao giờ trả về `0` sai lệch khi có vùng trũng Outage.
4. Tại Bước 1 & Bước 2 (`TASK-M11` & `TASK-M12`), đã bổ sung `normalizeMasterMongoURI` + `formatMongoHostList` (`master_registry_handler_connections.go:37-115`, `master_repo_gorm.go:914-946`, `connection_manager.go:248-269`) và `validatePostgresConnectorPrereqs` (`system_connectors_handler.go:470-512`).
5. Tại `TASK-M13`, đã bổ sung `DiscoverPostgresTablesPost` (`internal/api/system/introspection_handler_postgres.go:1-247`), đăng ký route `POST /api/introspection/postgres/tables` (`internal/router/router.go:347`), nâng cấp `validatePostgresConnectorPrereqs` (`internal/api/source/system_connectors_handler.go:497-506`), và hoàn thiện giao diện Check Connection + Quét danh sách bảng theo Schema cho PostgreSQL Connector trên `cdc-cms-web` (`src/services/connectorCheck.ts:75-143`, `src/hooks/useConnectorCheck.ts:62-108`, `src/pages/SourceConnectors.tsx:468-595, 1784-1904`).

Không vi phạm bất kỳ lesson nào trong `lessons/INDEX.md`.

---

## (e) Hội đồng `/laya_router` (`DG3`) sau lần sửa code cuối

- **File Claims:** `/Users/trainguyen/.gemini/antigravity/brain/29987e6f-a7d0-4106-816a-c6d84041eeba/scratch/dg3_m13.json`
- **File Raw Output:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-cdc-3repos-master-mongo-2026-10-28/06_laya_raw_dg3_20261028T163127.json`
- **File Summary:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-cdc-3repos-master-mongo-2026-10-28/06_laya_summary_dg3_20261028T163127.json`
- **Kết quả Gate:** `PASS` (`trusted_rounds: ["r1", "r3"]`, các audit claims `A-M13-BE`, `A-M13-ROUTER`, `A-M13-FE` đều đạt `ACCEPT` dựa trên bằng chứng `REF:` vật lý sau lần sửa code cuối).

---

## (f) Tổng kết Trạng thái Sẵn sàng (Sign-off)

Cả 2 feature (**Feature 1: Event Date Split & Batching** và **Feature 2: Master MongoDB Destination + 10 bản vá `OB-01` → `OB-10`**) cùng các nâng cấp `TASK-M11`, `TASK-M12`, `TASK-M13` đã được hợp nhất đồng thời trên codebase `2026-10-28` của cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) mà không dùng `git stash apply` và không dùng `git push`. Toàn bộ 8 Quality Gates (`G1–G8`) và Decision Gates (`DG0`, `DG1`, `DG2`, `DG3`) đều đạt `PASS`.
