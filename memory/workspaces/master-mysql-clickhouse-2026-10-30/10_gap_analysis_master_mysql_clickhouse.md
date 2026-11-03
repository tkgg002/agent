# 10. Adversarial Plan Audit & Line-by-Line Pre-Implementation Critique — Master MySQL & ClickHouse

- **Workspace:** `master-mysql-clickhouse-2026-10-30`
- **Suffix:** `master_mysql_clickhouse`
- **Tier:** `T2`
- **Stage:** `S3 Pre-Execution Adversarial Plan Audit`

---

## 1. Mục tiêu Phản biện (Adversarial Objectives Critique)

Thực hiện rà soát phản biện ngược toàn bộ bản kế hoạch ban đầu đối chiếu 1–1 với mã nguồn vật lý trên cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) theo 4 tiêu chí cứng của User:
1. **Đúng Layer & Single-Responsibility Principle (SRP):** Tuyệt đối không nhồi nhét hàm xử lý dialect MySQL / ClickHouse vào các file điều phối lớn (`transmuter.go` `1835` dòng, `master_ddl_generator.go` `743` dòng, `recon_dest_query.go` `708` dòng, `recon_execute_heal_segb.go` `927` dòng, `MasterRegistry.tsx` `1942` dòng). Mỗi file hiện hữu chỉ được phép thêm các nhánh điều phối mỏng (3–10 dòng switch/if delegate).
2. **Không Sót Call-Site (Zero Hidden Postgres Assumptions):** Quét toàn bộ nơi gọi `IsMongoDestination`, `IsMongoMasterConnection`, `IsMongo()`, `GetMasterDB`, `engine_type`, `master_schema` để phát hiện mọi chỗ đang ngầm giả định `"không phải MongoDB thì chắc chắn là PostgreSQL"`.
3. **Line-by-Line Update Verification:** Kiểm tra từng dòng dự kiến sửa trong các file hiện hữu để đảm bảo không làm gãy syntax SQL (`::text`, `"schema"."table"`, `SET LOCAL lock_timeout`, `SET TRANSACTION READ ONLY`, `connect_timeout=5`).
4. **Tương thích ngược 100% (Backward Compatibility):** Giữ nguyên chữ ký hàm public, giữ nguyên hành vi mặc định `engine_type = "postgresql"` khi rỗng/default, và giữ nguyên 100% nhánh `mongodb` hiện hữu.

---

## 2. Phát hiện 8 Khe Hở Kỹ Thuật Từ Phản Biện Ngược (`G1` → `G8`) & Cách Khắc Phục Triệt Để

### G1. `MasterDDLGenerator` (`internal/service/master/master_ddl_generator.go`) còn 5 điểm phụ thuộc cú pháp PostgreSQL
- **Bằng chứng `file:line`:**
  - `Generate` (`master_ddl_generator.go:89-335`): Hiện tại luôn sinh câu lệnh PostgreSQL (`CREATE SCHEMA IF NOT EXISTS ...`, `"_gpay_id" BIGINT PRIMARY KEY`, `"_updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()`).
  - `Apply` (`master_ddl_generator.go:352-442`): Sau nhánh `if isMongo`, dòng `366-370` gọi `SET LOCAL lock_timeout = '5s'` và `SET LOCAL statement_timeout = '30s'`, dòng `395` gọi `DROP COLUMN IF EXISTS` (MySQL 8.0 không hỗ trợ `IF EXISTS` trong `DROP COLUMN`), dòng `414` gọi `ENABLE ROW LEVEL SECURITY`.
  - `ReconcileColumn` (`master_ddl_generator.go:538-614`): Dòng `581` gọi `ALTER COLUMN ... TYPE ... USING col::text::type` và dòng `592` gọi `SET LOCAL lock_timeout`.
  - `DropColumn` (`master_ddl_generator.go:620-653`): Dòng `637` gọi `SET LOCAL lock_timeout` và dòng `643` gọi `DROP COLUMN IF EXISTS`.
  - `Swap` (`master_ddl_generator.go:657-720`) & `DropTable` (`master_ddl_generator.go:722-743`): Dòng `676` gọi `SET LOCAL lock_timeout`, dòng `736` gọi `DROP TABLE IF EXISTS ... CASCADE` (MySQL và ClickHouse không hỗ trợ `CASCADE` ở `DROP TABLE`).
- **Khắc phục SRP:**
  - Tại `master_ddl_generator.go`, chỉ thêm đúng các dòng delegate mỏng (`if g.isMySQLBinding(ctx, reg)` và `if g.isClickHouseBinding(ctx, reg)`) ở đầu `Generate`, `Apply`, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable`.
  - Toàn bộ logic DDL của MySQL nằm trong `internal/service/master/master_ddl_generator_mysql.go` và của ClickHouse nằm trong `internal/service/master/master_ddl_generator_clickhouse.go`.

### G2. `TransmuterModule` (`internal/service/master/transmuter.go`) có 6 Call-Site giả định SQL = PostgreSQL
- **Bằng chứng `file:line`:**
  1. `Run` DDL Safety Gate (`transmuter.go:277-322`) & `RunEvents` DDL Safety Gate (`transmuter.go:692-729`): Khi `masterRow.MasterSchema` rỗng hoặc `"public"`, truy vấn `information_schema.tables WHERE table_schema = 'public'` sẽ luôn trả về `false` trên MySQL/ClickHouse! Ngoài ra trên ClickHouse cần kiểm tra `system.tables WHERE database = ? AND name = ?`.
  2. `inferLastGpayIDFromMaster` (`transmuter.go:374-378` & `1003-1079`): Dùng `quoteTransmuteQualified` (`"schema"."table"`) và ép kiểu `::text` (`dòng 1030, 1049`) — gây lỗi cú pháp trên cả MySQL lẫn ClickHouse.
  3. Orphan Cleanup trong `Run` (`transmuter.go:524-565`): Dùng `"schema"."table"` và `DELETE FROM %s WHERE _gpay_id IN (?)` — gây lỗi trên MySQL (do dấu nháy kép) và trên ClickHouse (do `MergeTree` yêu cầu `ALTER TABLE ... DELETE WHERE _gpay_id IN (?)`).
  4. `deleteFromMasterBySourceAndGpayIDs` (`transmuter.go:1449-1567`): Dùng `"schema"."table"`, `pkExpr + "::text"` (`dòng 1509`), và `DELETE FROM`.
  5. `bulkUpsertMaster` (`transmuter.go:1588-1620`): Dùng placeholder `$1, $2` (`$%d::jsonb`) và `ON CONFLICT (...) DO UPDATE`.
- **Khắc phục SRP:**
  - Tách toàn bộ hàm thực thi sang `internal/service/master/transmuter_mysql.go` (`ensureMySQLMasterTableExists`, `inferLastGpayIDFromMySQLMaster`, `deleteOrphanGpayIDsFromMySQLMaster`, `deleteFromMySQLMasterBySourceAndGpayIDs`, `bulkUpsertMySQLMaster`) và `internal/service/master/transmuter_clickhouse.go` (`ensureClickHouseMasterTableExists`, `inferLastGpayIDFromClickHouseMaster`, `deleteOrphanGpayIDsFromClickHouseMaster`, `deleteFromClickHouseMasterBySourceAndGpayIDs`, `bulkUpsertClickHouseMaster`).
  - Trong `transmuter.go`, mỗi call-site chỉ thêm 4 dòng `if/else if` delegate, giữ nguyên 100% code PostgreSQL và MongoDB.

### G3. `ConnectionManager` & `MetadataRegistryDSN` (`internal/service/source/`) thiếu nhận diện DSN của MySQL & ClickHouse
- **Bằng chứng `file:line`:**
  - `tryPlainDSN` (`metadata_registry_utils.go:15-30`): Chưa nhận diện prefix `"clickhouse://"`, `"tcp://"`, hoặc chuỗi MySQL DSN dạng `user:pass@tcp(host:port)/db`.
  - `buildDSNFromFieldsPatched` (`metadata_registry_dsn.go:113-128`): Chỉ có `case "mongodb"`, `"sftp"`, `"postgres"/"postgresql"`, trả về `""` cho `"mysql"`, `"mariadb"`, `"clickhouse"`.
  - `GetMasterDB` (`connection_manager.go:340-406`): Dòng `400` gọi `m.reg.OpenGorm(dsn, "master:"+key)` thay vì mở theo `engineType`.
- **Khắc phục SRP:**
  - Tách `pkgs/database/multi_dialector.go` (`OpenGormWithEngine`, `ResolveGormDialector`) và `internal/service/source/connection_manager_sql_engines.go` (`IsMySQLMasterConnection`, `IsClickHouseMasterConnection`, `ResolveMasterSQLDatabase`, `NormalizeMySQLMasterDSN`, `BuildMasterSQLDSNFromConn`).
  - Trong `connection_manager.go:400`, đổi 1 dòng `m.reg.OpenGorm(dsn, "master:"+key)` thành `m.reg.OpenGormWithEngine(dsn, "master:"+key, resolvedEng)`.

### G4. Reconciliation (`internal/service/recon/` & `internal/handler/recon/`) gọi `SET TRANSACTION READ ONLY`, `pg_class`, `hashtextextended`, `::text`
- **Bằng chứng `file:line`:**
  - Đường dẫn package thực tế là `internal/service/recon` (không phải `internal/service/reconciliation`).
  - `ReconDestAgent.readOnlyDB` (`recon_dest_agent.go:121-125`): Luôn mở `Begin()` và chạy `SET TRANSACTION READ ONLY` — ClickHouse không hỗ trợ câu lệnh này.
  - `recon_dest_query.go` (`CountRows:37`, `CountDeletedRows:89`, `GetRealColumnName:123`, `EstimatedCountRows:217`, `CountInWindow:258`, `CountRecentDeletedRows:325`, `BucketCounts:396`, `ListIDTsInWindow:505`, `MaxWindowTs:658`), `recon_dest_hash.go` (`HashWindow:62`, `BucketHash:228`), `recon_window_index_probe.go` (`HasColumnIndex:64`): Dùng `quoteRelation` (`"schema"."table"`), `::text`, `pg_class`, `pg_indexes`, `bit_xor(hashtextextended(...))`.
  - `recon_tier_b.go` (`RunRowDiffB:679-680`, `TimeBoundedDiffMissingFromMaster:848-856`) và `recon_execute_heal_segb.go` (`ExecuteSegmentBHeal:103-210`, `resolveSourceIDsFromMaster:877`): Khi `!isMongoMaster`, truy vấn trực tiếp `masterDB` bằng cú pháp Postgres (`"schema"."table"`, `::text`, `DELETE FROM`).
- **Khắc phục SRP:**
  - Tách `internal/service/recon/recon_dest_sql_dialects.go` xử lý trọn vẹn các hàm đọc đối soát cho `mysql` (backtick + `CAST(... AS CHAR)` + Go-side XOR bucketing giống `mongoBucketCounts`) và `clickhouse` (`FINAL` + `toString(...)` + không mở transaction `SET TRANSACTION READ ONLY`).
  - Tách `internal/handler/recon/recon_execute_heal_segb_sql_dialects.go` xử lý Prune orphan và Resolve PK cho MySQL/ClickHouse khi chạy Segment B Heal.

### G5. `SourceCountRunner` (`internal/service/sourcecount/source_count_runner.go:165-181, 290-300`)
- **Bằng chứng `file:line`:**
  - Dòng `166-180`: Ở `case "master":`, nếu `params.EngineType` rỗng (bị gán mặc định `"postgresql"` ở dòng `106`), `countSQL` (dòng `291`) sẽ dùng dấu nháy kép `"schema"."table"` khiến MySQL lỗi cú pháp và ClickHouse thiếu `FINAL`.
- **Khắc phục:**
  - Nếu `params.ConnectionCode != ""` và `r.connMgr != nil`, đồng bộ `resolvedEng := r.connMgr.ResolveMasterEngineType(queryCtx, params.ConnectionCode)` trước khi gọi `r.countSQL`, và trong `countSQL` hỗ trợ `FINAL` khi `eng == "clickhouse"`.

### G6. `cdc-cms-service` sót 3 Call-Site chuẩn hóa Database-Scoped Master & lỗi `connect_timeout=5` khi Test Connection
- **Bằng chứng `file:line`:**
  1. `internal/app/commands/master/create_master.go:176-199`: Đang chỉ kiểm tra `isMongoConn` để覆蓋 `masterSchema = default_database` và `masterDatabase = &default_database`. Với MySQL và ClickHouse (cũng là 1 kết nối = 1 database, không có schema `public`), nếu không áp dụng quy tắc này thì khi tạo Master Table từ UI sẽ lưu `master_schema = "public"` hoặc tên schema Postgres cũ!
  2. `internal/infra/persistence/master/master_read_repo_gorm.go:34-42` & `master_connection_repo_gorm.go:350`: Chỉ mới kiểm tra `engine_type = 'mongodb'` khi đồng bộ `default_database` sang `master_schema`.
  3. `internal/api/master/master_registry_handler_connections_probe.go:184-192`: Khi bấm nút **"Test"** trên một dòng kết nối đã lưu (truyền `id` hoặc `connection_code`), code hiện tại nối thêm `&connect_timeout=5` vào mọi DSN không phải `mongodb://`! Với `go-sql-driver/mysql`, tham số `connect_timeout=5` bị hiểu là biến hệ thống MySQL và gây lỗi `Error 1193: Unknown system variable 'connect_timeout'`!
- **Khắc phục SRP:**
  - Tách `internal/infra/persistence/master/master_connection_engine_dsn.go` và `internal/api/master/master_registry_handler_connections_sql_dialects.go`.
  - Chỉ nối `connect_timeout=5` khi `engineType == "postgresql"`; với MySQL dùng `timeout=5s&readTimeout=10s&writeTimeout=10s` và mở bằng `mysql.Open(cleanDSN)`; với ClickHouse dùng `dial_timeout=5s&read_timeout=10s` và mở bằng `clickhouse.Open(dsn)`.

### G7. `cdc-cms-web` đối chiếu đúng đường dẫn file vật lý (`MasterRegistry.tsx` & `CreateMasterModal.tsx`)
- **Bằng chứng `file:line`:**
  - Form tạo Master nằm tại `src/components/master/CreateMasterModal.tsx` (`316` dòng), còn Modal Tạo/Sửa Connection nằm trong `src/pages/MasterRegistry.tsx` (`dòng 1733-1940`).
- **Khắc phục SRP:**
  - Tách `src/components/master/masterEngineConfig.ts` chứa toàn bộ cấu hình 4 engine (`postgresql`, `mongodb`, `mysql`, `clickhouse`): port mặc định (`5432`, `27017`, `3306`, `9000`), màu Tag, nhãn hiển thị, placeholder, và hàm `isDatabaseScopedMasterEngine(engine)`.
  - Cập nhật mỏng tại `MasterRegistry.tsx` và `CreateMasterModal.tsx`.

### G8. Đảm bảo Tương Thích Ngược 100% (Zero Regression)
- Mọi kết nối có `engine_type` rỗng, `"postgres"`, `"postgresql"` hoặc `connection_code = "default"` / `"default_master"` đi vào đúng nhánh `default` (PostgreSQL) như cũ 100%.
- Mọi kết nối `engine_type = "mongodb"` / `"mongo"` đi vào đúng nhánh MongoDB như cũ 100%.
- Giữ nguyên toàn bộ chữ ký hàm hiện có (`OpenGorm`, `GetMasterDB`, `IsMongoMasterConnection`, `IsMongoDestination`, `NewReconDestAgentWithConfig`).
