# 09 Tasks Solution: Audit & Tối ưu Detect DDL cho `scan-fields` PostgreSQL (`scan_fields_postgres`)

- **Workspace**: `audit-scan-fields-postgres-2026-10-28`
- **Suffix**: `scan_fields_postgres`
- **Tier**: `T2`
- **Date**: `2026-10-28`

---

## 1. Bảng Ánh xạ Điểm Gọi Thực thi & Vòng đời Kích hoạt (`L002` & `L018` Callchain Tracing)

| Bước | Thành phần | Điểm gọi (`caller file:line`) | Hàm đích (`callee file:line`) | Sự kiện / Thời điểm kích hoạt (`Lifecycle Trigger`) |
| :--- | :--- | :--- | :--- | :--- |
| **1** | **CMS API (V2 / Legacy)** | `cdc-cms-service/internal/api/source/source_object_actions_handler.go:352-407` (`ScanFieldsV2`) & `registry_handler_tools_scan.go:51-91` (`ScanFields`) | `h.bus.Dispatch(ctx, source.ScanFieldsCommand{...})` → NATS subject `cdc.cmd.scan-fields` | User bấm nút **Scan Fields** trên trang Table Registry (`POST /api/v1/source-objects/:id/scan-fields`). |
| **2** | **Worker NATS Consumer** | `centralized-data-service/internal/server/server_setup.go:314` | `(*DiscoverHandler).HandleScanFields` (`internal/handler/source/discover_handler.go:246-328`) | Nhận message từ NATS queue group `cdc-worker-group` trên subject `cdc.cmd.scan-fields`. |
| **3** | **Điều phối Scan Fields** | `centralized-data-service/internal/handler/source/discover_handler.go:280` | `(*DiscoverHandler).ScanFieldsDebezium` (`internal/handler/source/discover_handler.go:195-244`) | Phân loại theo `sourceType` (`postgresql`, `mysql`, `mongodb`, `sftp`...). |
| **4** | **Detect DDL SQL Source** | `centralized-data-service/internal/handler/source/discover_handler.go:210` (mới — rẽ nhánh thẳng) | `(*DiscoverHandler).scanFieldsSQLSource` (`internal/handler/source/discover_handler_sql.go:13-94`) | Ngay khi xác định `isSQLSource(sourceType)` (`postgres`/`postgresql`/`pg`/`mysql`/`mariadb`) — **bỏ qua hoàn toàn việc quét mẫu `_raw_data`**. |
| **5** | **Introspect `information_schema.columns`** | `centralized-data-service/internal/handler/source/discover_handler_sql.go:38` | `(*SourceInferrer).InferSourceColumnsWithError` → `(*SourceInferrer).inferPGCols` (`internal/handler/source/discovery_utils.go:112-240`) | Kết nối tới PostgreSQL nguồn qua DSN từ `MetadataRegistryService.GetSourceDSN` (`metadata_registry_service.go:473-586`) và đọc trực tiếp DDL từ `information_schema.columns`. |
| **6** | **Lưu Credentials khi Tạo/Sửa Connector** | `cdc-cms-service/internal/api/source/system_connectors_handler.go:182, 245, 652` | `extractCredentialsAsOptions` (`system_connectors_handler.go:567-612`) → `systemConnectorRepoGorm.Upsert` (`system_connector_repo_gorm.go:70`) | Khi tạo mới (`POST /api/v1/system/connectors`) hoặc cập nhật (`PUT /api/v1/system/connectors/:name`) connector trên CMS. |
| **7** | **CMS UI Tạo Master (MongoDB vs PostgreSQL)** | `cdc-cms-web/src/pages/MasterRegistry.tsx:1295-1456` (`Create Master` Modal) & `809-826` (`submitCreate`) | `createMut.mutate(...)` → `POST /api/v1/masters` (`MasterRegistry.tsx:627-647`) | User bấm nút **Create Master** trên trang Master Registry, chọn kết nối đích (`Target Database Connection`) và bấm **Submit for review**. |
| **8** | **CMS API Tạo Master Binding** | `cdc-cms-service/internal/api/master/master_registry_handler_create.go:16-96` (`Create`) | `(*CreateMasterHandler).Handle` (`internal/app/commands/master/create_master.go:111-228`) → `(*masterRepoGorm).CreateMasterBinding` (`internal/infra/persistence/master/master_repo_gorm.go:423-438`) | Khi thực thi `POST /api/v1/masters` — phân giải `ResolveMasterConnection` và lưu vào bảng `cdc_system.master_binding`. |
| **9** | **Worker Khởi tạo Collection & Index MongoDB** | `centralized-data-service/internal/service/master/master_ddl_generator.go:365-382` (`Apply`) | `(*MasterDDLGenerator).applyMongoCollectionAndIndexes` (`internal/service/master/master_ddl_generator.go:480-582`) | Khi User bấm **Approve** trên Master Binding có kết nối đích `engine_type == 'mongodb'`. |

---

## 2. Phân tích Gốc rễ (`F1` → `F4`) & Ánh xạ Giải pháp 1-1 (`L020`)

### Phát hiện `F1` (`VERIFIED`): `ScanFieldsDebezium` đang coi nguồn SQL (`PostgreSQL`/`MySQL`) như nguồn Schemaless (`MongoDB`), đi quét 100 dòng `_raw_data` thay vì detect thẳng DDL
- **Bằng chứng (`file:line`):**
  - Tại [`discover_handler.go:207-243`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler.go#L207-L243):
    - Với `isFileOrStreamSource(sourceType)`, hàm rẽ nhánh ngay ở dòng 207.
    - Nhưng với `isSQLSource(sourceType)` (`postgres`, `postgresql`, `pg`, `mysql`, `mariadb`), hàm **không rẽ nhánh ngay** mà gọi `GetShadowSampleRows(ctx, schemaName, targetTable, 100)` (dòng 222) để lấy 100 dòng `_raw_data` rồi chạy `processDiscoveryRows` (dòng 243) đoán kiểu dữ liệu từ JSON. Chỉ khi bảng Shadow có `0` dòng (`len(rows) == 0`, dòng 227) nó mới chịu gọi `scanFieldsSQLSource` (dòng 231).
    - Hệ quả: Khi bảng Shadow đã có dữ liệu (ví dụ sau khi chạy `snapshot.v2` có `4,683` dòng trong `shadow_devcoretrans.transaction_history`), bấm `scan-fields` sẽ quét JSON `_raw_data` của Debezium (nơi `TIMESTAMPTZ` bị serialize thành số integer microsecond hoặc string, `NUMERIC` thành string/float) làm lệch kiểu dữ liệu so với DDL gốc.
- **Giải pháp 1-1 cho `F1`:**
  - Trong `ScanFieldsDebezium` (`discover_handler.go:207-215`): Nếu `sourceType == ""` thì tra cứu `source_engine_type` từ `source_object_registry`. Ngay khi `isSQLSource(sourceType)` trả về `true`, **lập tức gọi `return h.scanFieldsSQLSource(...)` ngay đầu hàm** (trước khi kiểm tra hay đọc `_raw_data`).

### Phát hiện `F2` (`VERIFIED`): Tại sao lúc `16:30:15` bị lỗi `"SQL source returned 0 columns or connection failed"`? — Bug ghi đè mật khẩu thành `"__KEEP__"` trong `connection_registry.options_json`
- **Bằng chứng (`file:line` & DB Runtime):**
  - Tại [`system_connectors_handler.go:245, 567-612`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go#L567-L612): Khi User mở modal Edit Connector `devcoretrans` và bấm Save mà không nhập lại mật khẩu, Frontend gửi `"database.password": "__KEEP__"` (`KeepSecretSentinel`).
  - Hàm `extractCredentialsAsOptions(req.Config)` không lọc bỏ `"__KEEP__"` (và `"***"`), nên đóng gói `{"password": "__KEEP__", "username": "core-trans-proxy-history-user"}` vào `Fingerprint.OptionsJSON`.
  - Khi `systemConnectorRepoGorm.Upsert` ([`system_connector_repo_gorm.go:70`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/system_connector_repo_gorm.go#L70)) thực thi `options_json = COALESCE(cdc_system.connection_registry.options_json, '{}'::jsonb) || EXCLUDED.options_json`, nó **ghi đè mật khẩu thật trong `connection_registry` bằng chuỗi `"__KEEP__"`**!
  - Khi `SourceInferrer.inferPGCols` (`discovery_utils.go:209`) lấy DSN từ `connection_registry` (với password `"__KEEP__"`) để kết nối tới `10.200.187.110:5432`, PostgreSQL trả lỗi `28P01 password authentication failed`, khiến `InferSourceColumns` trả về `nil` (0 cột) và `scanFieldsSQLSource` (`discover_handler_sql.go:40`) báo lỗi `"SQL source returned 0 columns or connection failed"`.
- **Giải pháp 1-1 cho `F2`:**
  - Trong `extractCredentialsAsOptions` (`cdc-cms-service/internal/api/source/system_connectors_handler.go:567-612`): Bổ sung hàm `isKeepOrMaskedSecret(v string) bool` để bỏ qua không ghi khóa `"password"` vào `opts` khi giá trị là `""`, `"__KEEP__"` (`sourceCmd.KeepSecretSentinel`) hoặc `"***"`. Nhờ toán tử JSONB merge `||` trong PostgreSQL, khóa `"password"` thật đã lưu trước đó trong `cdc_system.connection_registry.options_json` sẽ được giữ nguyên 100%.

### Phát hiện `F3` (`VERIFIED`): Khiếm khuyết khi detect DDL PostgreSQL trong `SourceInferrer` (`discovery_utils.go`) & `scanFieldsSQLSource` (`discover_handler_sql.go`)
- **Bằng chứng (`file:line`):**
  1. Tại [`discovery_utils.go:159-163, 206-240`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discovery_utils.go#L206-L240):
     - Nếu `row.Schema` bị cấu hình nhầm thành tên Database (như ở bản ghi `id=212` lúc `16:30:15`: `core-trans-proxy-history-service.transaction_history`) hoặc `row.Obj` có dạng `public.transaction_history`, câu truy vấn `WHERE table_schema = $1 AND table_name = $2` trả về `0` dòng mà không biết tách `schema.table` hay fallback về `table_schema = 'public'`.
     - Câu SQL chỉ lấy `SELECT column_name, data_type, is_nullable` mà thiếu `udt_name`. Khi gặp cột mảng (`text[]`, `int4[]`), `data_type` là `"ARRAY"`, hàm `pgSafeType` trả về `"ARRAY"` (vi phạm `base.IsSafeType` khi chạy `alter-column`).
     - `InferSourceColumns` nuốt hoàn toàn `error` (`return nil`), không trả nguyên nhân lỗi kết nối/SQL về cho `scanFieldsSQLSource`.
  2. Tại [`metadata_registry_service.go:560-585`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go#L560-L585):
     - `buildDSNFromFieldsPatched` ghép chuỗi `username + ":" + password + "@"` thủ công mà không dùng `url.UserPassword(username, password).String()`, và nếu `password == "__KEEP__"` hoặc `"***"` vẫn tạo DSN hỏng.
- **Giải pháp 1-1 cho `F3`:**
  - Cập nhật `inferPGCols` (`discovery_utils.go`):
    1. Tách tiền tố `schema.table` nếu `table` có dấu chấm.
    2. Đọc `SELECT column_name, data_type, COALESCE(udt_name, ''), is_nullable FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`.
    3. Nếu kết quả rỗng (`len(out) == 0`) và `schema != "public"`, tự động fallback truy vấn lại với `table_schema = 'public'`.
    4. Chuẩn hóa kiểu dữ liệu bằng `pgSafeTypeWithUDT(dataType, udtName)` hỗ trợ đầy đủ kiểu mảng (`_text` → `TEXT[]`, `_int4` → `INTEGER[]`, `_int8` → `BIGINT[]`, `_uuid` → `UUID[]`, `_int2` → `SMALLINT[]`) và chuẩn ANSI SQL (`L031`).
  - Bổ sung `InferSourceColumnsWithError(ctx, sourceID, pkColumn) ([]shadow.BusinessColumn, error)` vào `SourceColumnsInferrer` & `SourceInferrer` để `scanFieldsSQLSource` nhận được lỗi gốc rõ ràng, đồng thời trong `scanFieldsSQLSource` bổ sung fallback đọc DDL các cột business từ `information_schema.columns` của bảng Shadow nếu nguồn tạm thời không kết nối được.
  - Trong `buildDSNFromFieldsPatched` (`metadata_registry_service.go:560-585`): Dùng `url.UserPassword` để encode an toàn thông tin xác thực.

### Phát hiện `F4` (`VERIFIED`): Phân giải `v2ObjectID` sai trong `ScanFieldsDebezium` (`L032`)
- **Bằng chứng (`file:line`):**
  - Tại [`discover_handler.go:196-205`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler.go#L196-L205): `_, err := h.mappingV2Repo.ListBySourceObject(ctx, v2ObjectID)` sử dụng GORM `.Find()`, luôn trả `err == nil` khi 0 dòng khớp, làm cho nhánh `GetByLegacyRegistryID` bên dưới không bao giờ được gọi.
- **Giải pháp 1-1 cho `F4`:**
  - Tra cứu `h.sourceObjectRegistryRepo.GetByID(ctx, v2ObjectID)` trước; nếu không tìm thấy mới fallback sang `h.sourceObjectRegistryRepo.GetByLegacyRegistryID(ctx, registryID)`, đồng thời điền luôn `sourceType = reg.SourceEngineType` nếu `sourceType` trong payload bị rỗng.

### Phát hiện `F5` (`VERIFIED`): Modal `Create Master Table` (`MasterRegistry.tsx:1295-1456`) đang hardcode nhãn PostgreSQL (`Master Schema`, `Master Table Name`) và tự điền `master_<source_db>` kể cả khi chọn Kết Nối Đích là MongoDB
- **Bằng chứng (`file:line` & DB Runtime):**
  - Tại [`MasterRegistry.tsx:1355-1450`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx#L1355-L1450):
    1. Khi User chọn `Shadow Table` (`dòng 1380`), UI tự động gán `master_schema: prev.master_schema || normalizeMasterSchema(b?.source_db || prev.shadow_schema || '')` → sinh ra `master_core_trans_proxy_history_service` (quy ước Schema của PostgreSQL).
    2. Khi User chọn `Target Database Connection` là `mongo_local` (`engine_type = 'mongodb'`, `default_database = 'devtest'`), UI **không đổi label** sang chuẩn MongoDB (`Database / Collection`), **không tự động gán `form.master_schema = 'devtest'`**, và **vẫn cho phép sửa tự do** ô `Master Schema`.
    3. Ô `Target Database Connection` (`dòng 1412-1451`) đặt ở dưới cùng sau `Master Schema` & `Master Table Name`, và hardcode chữ `"Chọn database connection đích (PostgreSQL)"`, `"Chỉ định instance PostgreSQL lưu trữ bảng Master"`.
- **Giải pháp 1-1 cho `F5` (Đúng chuẩn thiết kế của User: `1 Kết nối Master Mongo = 1 Database`):**
  1. Khai báo `selectedMasterConn`, `isMongoMasterTarget`, `mongoMasterDatabase` ở top-level của component `MasterRegistry` (tuân thủ nghiêm ngặt `L043` React Rules of Hooks).
  2. Đưa khối chọn **`Target Database Connection *`** lên ngay trên khối **`Master Database / Master Schema`** để luồng thao tác tự nhiên: Chọn Kết Nối Đích → hệ thống biết ngay là PostgreSQL (`Database -> Schema -> Table`) hay MongoDB (`Database -> Collection`).
  3. Khi `isMongoMasterTarget === true` (`engine_type === 'mongodb'`):
     - Đổi tiêu đề modal thành **`Create Master Collection (MongoDB)`**.
     - Đổi nhãn `Master Schema` → **`Master Database (MongoDB)`**, tự động gán `form.master_schema = selectedMasterConn.default_database` (ví dụ `devtest`) và **khóa cứng (`disabled={true}`)** không cho nhập tay.
     - Đổi nhãn `Master Table Name` → **`Master Collection Name`**.
     - Khi User đổi `Shadow Table` trong lúc đang chọn kết nối MongoDB, giữ nguyên `master_schema = mongoMasterDatabase` (tuyệt đối không ghi đè bằng `normalizeMasterSchema(b?.source_db)`). Khi chuyển ngược về kết nối PostgreSQL, tự động mở khóa (`disabled={false}`) và điền lại schema gợi ý `master_<source_db>`.

### Phát hiện `F6` (`VERIFIED`): `CreateMasterHandler.Handle` (`create_master.go:155-183`) bỏ qua `default_database` của kết nối MongoDB và lưu `master_database = NULL`, `master_schema = 'master_core_trans_proxy_history_service'`
- **Bằng chứng (`file:line` & DB Runtime):**
  - Tại [`create_master.go:155-183`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go#L155-L183):
    - `connID, connCode, _, defaultSchema, err := h.masterRepo.ResolveMasterConnection(...)` bỏ qua giá trị trả về thứ 3 (`defaultDatabase`), không kiểm tra `engine_type` của kết nối đích, và truyền `nil` vào tham số `masterDatabase` khi gọi `CreateMasterBinding` (dòng 176).
    - Thực tế trong DB (`cdc_system.master_binding` `id = 60`): `master_connection_id = 406` (`mongo_local`, `default_database = 'devtest'`), nhưng `master_database` bị `NULL` còn `master_schema` lưu `'master_core_trans_proxy_history_service'`.
- **Giải pháp 1-1 cho `F6`:**
  1. Trong `CreateMasterHandler.Handle` (`create_master.go:155-183`): Lấy `defaultDB` từ `ResolveMasterConnection` và kiểm tra `connItem, _ := h.masterRepo.GetMasterConnectionByID(ctx, connID)`. Nếu kết nối đích có `engine_type == "mongodb"`, **cưỡng chế `masterSchema = strings.TrimSpace(*defaultDB)` và `masterDatabase = &masterSchema`** (nếu kết nối MongoDB chưa có `default_database` thì báo lỗi `invalid_mongo_master_database_required`). Với PostgreSQL, vẫn giữ `masterSchema` theo schema được chọn và gán `masterDatabase = defaultDB`.
  2. Mở rộng regex `namespaceName` / `masterNameRe` (`master_registry_handler.go:28-29`, `create_master.go:44-45`) cho phép dấu gạch ngang `-` (`^[a-z_][a-z0-9_-]{0,62}$`) để tương thích với tên database/collection MongoDB có dấu `-`.
  3. Trong `ListEnriched` (`master_read_repo_gorm.go:33-36` & `list_masters.go:20-22`): Trả thêm `mc.engine_type AS master_engine_type`, và đối với dòng `mc.engine_type = 'mongodb'` thì ưu tiên hiển thị `COALESCE(NULLIF(mc.default_database, ''), mb.master_schema) AS master_schema` để UI luôn hiển thị đúng Database MongoDB của kết nối.
  4. Trong `UpdateMasterConnection` (`master_repo_gorm.go:750-852`): Khi cập nhật `default_database` của kết nối Master MongoDB, đồng bộ luôn `master_database`, `master_schema`, `physical_table_fqn` cho các dòng `cdc_system.master_binding` đang trỏ tới kết nối đó.

### Phát hiện `F7` (`VERIFIED`): Tại sao ở MongoDB không tạo database `master_core_trans_proxy_history_service`? — Do `MasterDDLGenerator.applyMongoCollectionAndIndexes` (`master_ddl_generator.go:492-501`) tạo collection trong `default_database` (`devtest`) của kết nối!
- **Bằng chứng (`file:line` & MongoDB Runtime):**
  - Tại [`master_ddl_generator.go:492-501`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L492-L501):
    ```go
    dbName := g.connMgr.ResolveMasterMongoDatabase(ctx, reg.MasterConnectionKey, reg.DefaultDatabase)
    if dbName == "" {
        dbName = reg.MasterSchema
    }
    ```
    Vì kết nối `mongo_local` (`id=406`) có `default_database = 'devtest'`, hàm `ResolveMasterMongoDatabase` trả về `dbName = "devtest"`. Do đó, khi Approve bản ghi `id=60`, Worker đã tạo collection `transaction_history` trong database **`devtest`** trên MongoDB (`devtest: ["transaction_history", "devtest"]`), chứ không tạo database `master_core_trans_proxy_history_service` như chữ hiển thị lệch trên UI!
- **Giải pháp 1-1 cho `F7`:**
  - Khi đã áp dụng quy tắc **`1 kết nối Master Mongo = 1 Database (`default_database`)`** ở `F5` & `F6`, cả UI, `cdc_system.master_binding` và `MasterDDLGenerator` đều cùng trỏ về duy nhất `default_database` (`devtest`).
  - Đồng thời trong `applyMongoCollectionAndIndexes` (`master_ddl_generator.go:571-577`), sau khi tạo collection & index trên `dbName`, tự động đồng bộ lại `master_database = dbName`, `master_schema = dbName`, `physical_table_fqn = dbName + "." + reg.MasterTable` trên `cdc_system.master_binding` (tự động sửa luôn bản ghi `id=60` hiện hữu từ `master_core_trans_proxy_history_service` về đúng `devtest`).

---

## 3. Code Demo Chi Tiết (Trước & Sau)

### 3.1. `cdc-cms-service/internal/api/source/system_connectors_handler.go` (Fix `F2`)
```go
// TRƯỚC (lines 567-586):
func extractCredentialsAsOptions(cfg map[string]string) []byte {
	opts := make(map[string]string)
	// Postgres / Mysql / SQL Server
	if user, ok := cfg["database.user"]; ok {
		opts["username"] = user
	}
	if pass, ok := cfg["database.password"]; ok {
		opts["password"] = pass
	}
	if sslmode, ok := cfg["database.sslmode"]; ok {
		opts["sslmode"] = sslmode
	}
	// MongoDB
	if user, ok := cfg["mongodb.user"]; ok {
		opts["username"] = user
	}
	if pass, ok := cfg["mongodb.password"]; ok {
		opts["password"] = pass
	}

// SAU:
func isKeepOrMaskedSecret(v string) bool {
	s := strings.TrimSpace(v)
	return s == "" || s == sourceCmd.KeepSecretSentinel || s == "***"
}

func extractCredentialsAsOptions(cfg map[string]string) []byte {
	opts := make(map[string]string)
	// Postgres / Mysql / SQL Server
	if user, ok := cfg["database.user"]; ok && strings.TrimSpace(user) != "" {
		opts["username"] = strings.TrimSpace(user)
	}
	if pass, ok := cfg["database.password"]; ok && !isKeepOrMaskedSecret(pass) {
		opts["password"] = pass
	}
	if sslmode, ok := cfg["database.sslmode"]; ok && strings.TrimSpace(sslmode) != "" {
		opts["sslmode"] = strings.TrimSpace(sslmode)
	}
	// MongoDB
	if user, ok := cfg["mongodb.user"]; ok && strings.TrimSpace(user) != "" {
		opts["username"] = strings.TrimSpace(user)
	}
	if pass, ok := cfg["mongodb.password"]; ok && !isKeepOrMaskedSecret(pass) {
		opts["password"] = pass
	}
```
*(Tương tự cho nhánh `sftp.password` / `fs.uris` ở dòng 595-606: kiểm tra `!isKeepOrMaskedSecret(...)` trước khi gán `opts["password"]`).*

### 3.2. `centralized-data-service/internal/handler/source/discover_handler.go` (Fix `F1` & `F4`)
```go
// TRƯỚC (lines 195-220):
func (h *DiscoverHandler) ScanFieldsDebezium(ctx context.Context, registryID uint, shadowBindingID int64, targetTable, sourceTable, sourceType string, autoApprove bool) (int, int, error) {
	v2ObjectID := int64(registryID)
	if h.mappingV2Repo != nil && v2ObjectID > 0 {
		_, err := h.mappingV2Repo.ListBySourceObject(ctx, v2ObjectID)
		if err != nil {
			reg, err := h.sourceObjectRegistryRepo.GetByLegacyRegistryID(ctx, registryID)
			if err == nil && reg != nil && reg.ID > 0 {
				v2ObjectID = reg.ID
			}
		}
	}

	if isFileOrStreamSource(sourceType) {
		return h.scanFieldsFileSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
	}

	schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, targetTable)
	if !h.discoverSvc.TableExistsInShadow(ctx, schemaName, targetTable) || !h.discoverSvc.HasColumnInShadow(ctx, schemaName, targetTable, "_raw_data") {
		if sourceType == "mongodb" {
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		if isSQLSource(sourceType) {
			return h.scanFieldsSQLSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		return 0, 0, fmt.Errorf("table %s has no _raw_data column in shadow db", targetTable)
	}

// SAU:
func (h *DiscoverHandler) ScanFieldsDebezium(ctx context.Context, registryID uint, shadowBindingID int64, targetTable, sourceTable, sourceType string, autoApprove bool) (int, int, error) {
	v2ObjectID := int64(registryID)
	if h.sourceObjectRegistryRepo != nil && v2ObjectID > 0 {
		reg, err := h.sourceObjectRegistryRepo.GetByID(ctx, v2ObjectID)
		if err != nil || reg == nil {
			if legacyReg, lErr := h.sourceObjectRegistryRepo.GetByLegacyRegistryID(ctx, registryID); lErr == nil && legacyReg != nil && legacyReg.ID > 0 {
				reg = legacyReg
				v2ObjectID = legacyReg.ID
			}
		}
		if reg != nil && strings.TrimSpace(sourceType) == "" {
			sourceType = reg.SourceEngineType
		}
	}

	if isFileOrStreamSource(sourceType) {
		return h.scanFieldsFileSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
	}
	if isSQLSource(sourceType) {
		return h.scanFieldsSQLSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
	}
```

### 3.3. `centralized-data-service/internal/handler/source/discover_handler_sql.go` & `discovery_utils.go` (Fix `F3`)
```go
// Trong centralize-data-service/internal/handler/source/discovery_utils.go:
func (si *SourceInferrer) InferSourceColumnsWithError(
	ctx context.Context, sourceID int64, pkColumn string,
) ([]shadow.BusinessColumn, error) {
	// ... (lookup source_object_registry + connection_registry)
	// Với postgres/postgresql/pg:
	dsn := si.pickSourceDSN(ctx, row.ConnectionID, row.ConnectionCode, "SOURCE_PG_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("no DSN configured for connection %q (id=%d)", row.ConnectionCode, row.ConnectionID)
	}
	schema := strings.TrimSpace(row.Schema)
	if schema == "" {
		schema = "public"
	}
	cols, err := si.inferPGCols(ctx, dsn, schema, row.Obj, pkColumn)
	if err != nil {
		return nil, err
	}
	return cols, nil
}

func (si *SourceInferrer) inferPGCols(
	ctx context.Context, dsn, schema, table, pkColumn string,
) ([]shadow.BusinessColumn, error) {
	table = strings.TrimSpace(table)
	if parts := strings.SplitN(table, ".", 2); len(parts) == 2 {
		if strings.TrimSpace(parts[0]) != "" {
			schema = strings.TrimSpace(parts[0])
		}
		table = strings.TrimSpace(parts[1])
	}
	if strings.TrimSpace(schema) == "" {
		schema = "public"
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("dial source pg: %w", err)
	}
	defer conn.Close(ctx)

	queryCols := func(targetSchema string) ([]shadow.BusinessColumn, error) {
		rows, qErr := conn.Query(ctx,
			`SELECT column_name, data_type, COALESCE(udt_name, ''), is_nullable
			   FROM information_schema.columns
			  WHERE table_schema = $1 AND table_name = $2
			  ORDER BY ordinal_position`,
			targetSchema, table)
		if qErr != nil {
			return nil, fmt.Errorf("query information_schema (%s.%s): %w", targetSchema, table, qErr)
		}
		defer rows.Close()
		var out []shadow.BusinessColumn
		for rows.Next() {
			var name, dataType, udtName, isNullable string
			if err := rows.Scan(&name, &dataType, &udtName, &isNullable); err != nil {
				return nil, err
			}
			out = append(out, shadow.BusinessColumn{
				Name:     name,
				DataType: pgSafeTypeWithUDT(dataType, udtName),
				Nullable: strings.EqualFold(isNullable, "YES"),
			})
		}
		return out, rows.Err()
	}

	out, err := queryCols(schema)
	if err != nil {
		return nil, err
	}
	// Fallback sang schema 'public' nếu source_schema bị lưu nhầm bằng tên Database
	if len(out) == 0 && !strings.EqualFold(schema, "public") {
		out, err = queryCols("public")
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
```
Và trong `discover_handler_sql.go`:
```go
	var cols []shadow.BusinessColumn
	var inferErr error
	if errInferrer, ok := h.sourceInferrer.(interface {
		InferSourceColumnsWithError(ctx context.Context, sourceID int64, pkColumn string) ([]shadow.BusinessColumn, error)
	}); ok {
		cols, inferErr = errInferrer.InferSourceColumnsWithError(ctx, registry.ID, pkColumn)
	} else {
		cols = h.sourceInferrer.InferSourceColumns(ctx, registry.ID, pkColumn)
	}
	if len(cols) == 0 {
		if inferErr != nil {
			return 0, 0, fmt.Errorf("SQL source DDL detection failed: %w", inferErr)
		}
		return 0, 0, fmt.Errorf("SQL source returned 0 columns or connection failed")
	}
```

### 3.4. `cdc-cms-web/src/pages/MasterRegistry.tsx` (Fix `F5` — Đổi label `Database / Collection`, tự chọn & khóa Database theo kết nối MongoDB)
```tsx
// 1. Khai báo ở top-level của MasterRegistry component (tuân thủ L043 Rules of Hooks):
  const selectedMasterConn = useMemo(() => {
    const code = form.master_connection_code || 'default_master';
    return (targetMasterConnections || []).find((c) => c.connection_code === code) || null;
  }, [targetMasterConnections, form.master_connection_code]);

  const isMongoMasterTarget = (selectedMasterConn?.engine_type || '').toLowerCase() === 'mongodb';
  const mongoMasterDatabase = (selectedMasterConn?.default_database || '').trim();

  // Tự động đồng bộ form.master_schema = mongoMasterDatabase khi chọn kết nối MongoDB,
  // và khôi phục gợi ý master_<source_db> khi chuyển về kết nối PostgreSQL:
  useEffect(() => {
    if (isMongoMasterTarget) {
      if (mongoMasterDatabase && form.master_schema !== mongoMasterDatabase) {
        setForm((prev) => ({ ...prev, master_schema: mongoMasterDatabase }));
      }
      return;
    }
    // Khi chuyển từ kết nối MongoDB về PostgreSQL: nếu master_schema đang giữ tên DB Mongo cũ thì gợi ý lại master_<source_db>
    const isStaleMongoDB = (targetMasterConnections || []).some(
      (c) => (c.engine_type || '').toLowerCase() === 'mongodb' && c.default_database === form.master_schema,
    );
    if (isStaleMongoDB || !form.master_schema) {
      const b = (shadowBindings || []).find(
        (x) => x.shadow_schema === form.shadow_schema && x.shadow_table === form.shadow_table,
      );
      const suggested = (b?.source_db || sourceDB || form.shadow_schema)
        ? normalizeMasterSchema(b?.source_db || sourceDB || form.shadow_schema || '')
        : '';
      if (suggested && form.master_schema !== suggested) {
        setForm((prev) => ({ ...prev, master_schema: suggested }));
      }
    }
  }, [isMongoMasterTarget, mongoMasterDatabase, targetMasterConnections, shadowBindings, form.shadow_schema, form.shadow_table, sourceDB, form.master_schema]);

// 2. Trong Modal Create Master (lines 1295-1455):
      <Modal
        open={createOpen}
        title={isMongoMasterTarget ? 'Create Master Collection (MongoDB)' : 'Create Master Table'}
        ...
      >
        ...
          {/* 3) Target Database Connection — đặt trước để xác định PostgreSQL (Schema/Table) hay MongoDB (Database/Collection) */}
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
              <Text strong>
                Target Database Connection <Text type="danger">*</Text>
              </Text>
              <Button type="link" size="small" icon={<PlusOutlined />} style={{ padding: 0 }} onClick={() => openCreateConnection()}>
                Thêm kết nối mới
              </Button>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Select
                showSearch
                style={{ flex: 1 }}
                placeholder="Chọn database connection đích (PostgreSQL / MongoDB)"
                value={form.master_connection_code || 'default_master'}
                options={
                  targetMasterConnections && targetMasterConnections.length > 0
                    ? targetMasterConnections.map((c) => {
                        const isMongo = (c.engine_type || '').toLowerCase() === 'mongodb';
                        const dbHint = c.default_database ? ` • db: ${c.default_database}` : '';
                        return {
                          label: `${c.connection_code} (${isMongo ? 'MongoDB' : 'PostgreSQL'}${dbHint})`,
                          value: c.connection_code,
                        };
                      })
                    : [{ label: 'default_master (Default Master DB)', value: 'default_master' }]
                }
                optionFilterProp="label"
                onChange={(v) => {
                  const nextConn = (targetMasterConnections || []).find((c) => c.connection_code === v);
                  const nextIsMongo = (nextConn?.engine_type || '').toLowerCase() === 'mongodb';
                  const b = (shadowBindings || []).find(
                    (x) => x.shadow_schema === form.shadow_schema && x.shadow_table === form.shadow_table,
                  );
                  const pgSuggested = normalizeMasterSchema(b?.source_db || sourceDB || form.shadow_schema || '');
                  setForm((prev) => ({
                    ...prev,
                    master_connection_code: v,
                    master_schema: nextIsMongo ? (nextConn?.default_database || '').trim() : (prev.master_schema && !isMongoMasterTarget ? prev.master_schema : pgSuggested),
                  }));
                }}
              />
              <Tooltip title="Thêm mới kết nối đích (PostgreSQL / MongoDB)">
                <Button icon={<PlusOutlined />} onClick={() => openCreateConnection()} />
              </Tooltip>
            </div>
            <Text type="secondary" style={{ fontSize: 11 }}>
              {isMongoMasterTarget
                ? `Kết nối MongoDB [${selectedMasterConn?.connection_code}] gắn cố định với Database "${mongoMasterDatabase || '(chưa cấu hình)'}" (1 kết nối = 1 database).`
                : 'Chỉ định instance PostgreSQL / MongoDB lưu trữ dữ liệu Master.'}
            </Text>
          </div>

          {/* 4) Master Database (MongoDB - locked) hoặc Master Schema (PostgreSQL - editable) */}
          <div>
            <Text strong style={{ display: 'block', marginBottom: 4 }}>
              {isMongoMasterTarget ? 'Master Database (MongoDB)' : 'Master Schema'} <Text type="danger">*</Text>
            </Text>
            {isMongoMasterTarget ? (
              <>
                <Input
                  value={mongoMasterDatabase || form.master_schema}
                  disabled
                  placeholder="Tự động chọn theo Database của kết nối MongoDB"
                />
                <Text type="secondary" style={{ fontSize: 11 }}>
                  Tự động chọn theo Database của kết nối MongoDB ({mongoMasterDatabase || 'N/A'}) — không cho phép nhập tay.
                </Text>
              </>
            ) : (
              <AutoComplete
                style={{ width: '100%' }}
                placeholder="Chọn master schema đang có hoặc nhập mới"
                value={form.master_schema}
                options={masterSchemaOptions}
                filterOption={(input, opt) => (opt?.value ?? '').toLowerCase().includes(input.toLowerCase())}
                onChange={(v) => setForm({ ...form, master_schema: v })}
              />
            )}
          </div>

          {/* 5) Master Collection Name (MongoDB) hoặc Master Table Name (PostgreSQL) */}
          <div>
            <Text strong style={{ display: 'block', marginBottom: 4 }}>
              {isMongoMasterTarget ? 'Master Collection Name' : 'Master Table Name'} <Text type="danger">*</Text>
            </Text>
            <Input
              placeholder={isMongoMasterTarget ? 'đề xuất theo tên shadow table (tên collection đích)' : 'đề xuất theo tên shadow table'}
              value={form.master_name}
              onChange={(e) => setForm({ ...form, master_name: e.target.value })}
            />
          </div>
```

### 3.5. `cdc-cms-service/internal/app/commands/master/create_master.go` & `master_read_repo_gorm.go` (Fix `F6`)
```go
// Trong cdc-cms-service/internal/app/commands/master/create_master.go (lines 150-183):
	connID, connCode, defaultDB, defaultSchema, err := h.masterRepo.ResolveMasterConnection(ctx, strings.TrimSpace(masterConnectionCode))
	if err != nil {
		if strings.Contains(err.Error(), "not found") || errors.Is(err, ports.ErrRecordNotFound) {
			return nil, ErrMasterConnectionNotFound
		}
		return nil, err
	}

	var masterDatabase *string
	connItem, _ := h.masterRepo.GetMasterConnectionByID(ctx, connID)
	isMongoConn := connItem != nil && strings.EqualFold(strings.TrimSpace(connItem.EngineType), "mongodb")

	if isMongoConn {
		// Quy tắc 1 Kết nối Master Mongo = 1 Database: bắt buộc dùng default_database của kết nối
		mongoDB := ""
		if defaultDB != nil && strings.TrimSpace(*defaultDB) != "" {
			mongoDB = strings.TrimSpace(*defaultDB)
		} else if connItem.DefaultDatabase != nil && strings.TrimSpace(*connItem.DefaultDatabase) != "" {
			mongoDB = strings.TrimSpace(*connItem.DefaultDatabase)
		}
		if mongoDB == "" {
			return nil, errors.New("invalid_mongo_master_database_required")
		}
		masterSchema = mongoDB
		masterDatabase = &mongoDB
	} else {
		if masterSchema == "public" && defaultSchema != nil && strings.TrimSpace(*defaultSchema) != "" {
			masterSchema = strings.TrimSpace(*defaultSchema)
		}
		if defaultDB != nil && strings.TrimSpace(*defaultDB) != "" {
			dbTrimmed := strings.TrimSpace(*defaultDB)
			masterDatabase = &dbTrimmed
		}
	}

	bindingCode := normalizeBindingCodeCmd("mb", masterSchema, cmd.MasterName, fmt.Sprintf("%d", time.Now().UTC().Unix()))
	physicalTableFQN := masterSchema + "." + cmd.MasterName

	masterBindingID, err := h.masterRepo.CreateMasterBinding(
		ctx,
		bindingCode,
		sourceObjectID,
		shadowBindingID,
		connID,
		masterDatabase,
		masterSchema,
		cmd.MasterName,
		physicalTableFQN,
		cmd.TransformType,
		string(spec),
		cmd.UpdatedBy,
	)
```

### 3.6. `centralized-data-service/internal/service/master/master_ddl_generator.go` (Fix `F7`)
```go
// Trong applyMongoCollectionAndIndexes (lines 571-578):
	if g.systemDB != nil {
		_ = g.systemDB.WithContext(ctx).Exec(
			`UPDATE cdc_system.mapping_rule_master SET in_master = true WHERE master_binding_id = ? AND is_active = true AND status = 'approved'`,
			reg.ID,
		).Error
		// Đồng bộ master_database, master_schema, physical_table_fqn trên cdc_system.master_binding khớp 100% với dbName của kết nối MongoDB
		_ = g.systemDB.WithContext(ctx).Exec(
			`UPDATE cdc_system.master_binding
			    SET master_database = ?,
			        master_schema = ?,
			        physical_table_fqn = ?,
			        updated_at = NOW()
			  WHERE id = ? AND (COALESCE(master_database, '') <> ? OR COALESCE(master_schema, '') <> ?)`,
			dbName, dbName, dbName+"."+reg.MasterTable, reg.ID, dbName, dbName,
		).Error
	}
```

---

## 4. Bổ sung Phản tỉnh & Tự Chữa lành (`S5 -> S3 -> S4 Self-Healing Loop` — `F8`, `F9`, `F10`)

### Phát hiện `F8` (`VERIFIED` — Vi phạm `L006` Single-Responsibility & Line-Budget Guard): Nhồi nhét thêm dòng vào 6 file lớn khiến `NOW > HEAD`
- **Bằng chứng (`wc -l` so với `HEAD`):**
  1. `cdc-cms-web/src/pages/MasterRegistry.tsx`: `HEAD = 2019` → `NOW = 2177` (`+158` dòng).
  2. `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go`: `HEAD = 956` → `NOW = 1135` (`+179` dòng).
  3. `cdc-cms-service/internal/api/source/system_connectors_handler.go`: `HEAD = 701` → `NOW = 773` (`+72` dòng).
  4. `centralized-data-service/internal/service/master/master_ddl_generator.go`: `HEAD = 750` → `NOW = 935` (`+185` dòng).
  5. `centralized-data-service/internal/handler/source/discovery_utils.go`: `HEAD = 446` → `NOW = 488` (`+42` dòng).
  6. `centralized-data-service/internal/service/source/metadata_registry_service.go`: `HEAD = 735` → `NOW = 739` (`+4` dòng).
- **Giải pháp 1-1 cho `F8` (`TASK-BUG-09`):**
  1. Tách Modal Tạo Master từ `MasterRegistry.tsx` ra component đơn trách nhiệm `cdc-cms-web/src/components/master/CreateMasterModal.tsx` → đưa `MasterRegistry.tsx` xuống `< 1930` dòng (`NOW <= HEAD`).
  2. Tách toàn bộ nhóm hàm CRUD Kết nối Master & `syncMongoMasterBindingsDatabase` từ `master_repo_gorm.go` sang `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go` → đưa `master_repo_gorm.go` xuống `~535` dòng (`NOW <= HEAD`).
  3. Tách nhóm hàm kiểm tra tiền điều kiện Debezium và trích xuất thông tin xác thực (`verifyPostgresReplicationPreflight`, `injectDebeziumSignalDefaults`, `joinHostPort`, `isKeepOrMaskedSecret`, `extractCredentialsAsOptions`) từ `system_connectors_handler.go` sang `cdc-cms-service/internal/api/source/system_connectors_credentials.go` → đưa `system_connectors_handler.go` xuống `~590` dòng (`NOW <= HEAD`).
  4. Tách `applyMongoCollectionAndIndexes` từ `master_ddl_generator.go` sang `centralized-data-service/internal/service/master/master_ddl_generator_mongo.go` → đưa `master_ddl_generator.go` xuống `< 750` dòng (`NOW <= HEAD`).
  5. Tách nhóm hàm suy diễn DDL SQL (`inferPGCols`, `inferMySQLCols`, `pgSafeType`, `pgSafeTypeWithUDT`, `mysqlToPGType`) từ `discovery_utils.go` sang `centralized-data-service/internal/handler/source/discovery_utils_sql.go` → đưa `discovery_utils.go` xuống `~255` dòng (`NOW <= HEAD`).
  6. Tách nhóm hàm phân giải DSN nguồn (`GetSourceDSN`, `resolveSourceURIFromConn`, `buildDSNFromFieldsPatched`) từ `metadata_registry_service.go` sang `centralized-data-service/internal/service/source/metadata_registry_dsn.go` → đưa `metadata_registry_service.go` xuống `~620` dòng (`NOW <= HEAD`).

### Phát hiện `F9` (`VERIFIED` — Vi phạm Phân tầng Kiến trúc): Truy vấn SQL `UPDATE cdc_system.master_binding` đặt trực tiếp ở tầng Service (`MasterDDLGenerator`) thay vì tầng Repository
- **Bằng chứng (`file:line`):**
  - Tại `centralized-data-service/internal/service/master/master_ddl_generator.go:576-584`, `MasterDDLGenerator` thuộc tầng `service/master` nhưng gọi trực tiếp `g.systemDB.WithContext(ctx).Exec("UPDATE cdc_system.master_binding ...")`.
- **Giải pháp 1-1 cho `F9` (`TASK-BUG-10`):**
  - Bổ sung phương thức `SyncMasterBindingDatabase(ctx context.Context, masterBindingID int64, dbName, masterTable string) error` vào `SyncRuntimeStateRepo` tại `centralized-data-service/internal/repository/master/sync_runtime_state_repo.go`.
  - Trong `applyMongoCollectionAndIndexes` (`master_ddl_generator_mongo.go`), gọi `g.runtimeStateRepo.SyncMasterBindingDatabase(ctx, reg.ID, dbName, reg.MasterTable)` (fallback an toàn nếu `runtimeStateRepo == nil`).

### Phát hiện `F10` (`VERIFIED` — Vi phạm `L016` Zero-Mockup & `L019` Honest Process State): Dùng struct giả `fakeMasterRepoForCreate` và chưa kiểm thử tích hợp trực tiếp luồng `ScanFieldsDebezium` trên PostgreSQL nguồn `10.200.187.110:5432`
- **Bằng chứng (`file:line` & Runtime):**
  1. Tại `cdc-cms-service/internal/app/commands/master/update_column_metadata_test.go:162-238`, bài test `TestCreateMasterHandler_EnforcesMongoDefaultDatabase` dùng struct giả `fakeMasterRepoForCreate` thay vì kết nối tới PostgreSQL thật (`:5433` `cdc_dw`).
  2. Luồng `ScanFieldsDebezium` cho `core-trans-proxy-history-service.transaction_history` (`source_object_id = 214`, `shadow_binding_id = 219`, kết nối `id = 407` tới `10.200.187.110:5432`) chưa có integration test chạy thật trên DB thật mà mới chỉ chạy qua `sqlmock` trong `discover_handler_test.go`.
  3. Tiến trình `cdc-cms-service` (`PID 7591`, `:8083`) và `centralized-data-service` (`PID 9181`, `:8082`) được khởi chạy từ `Wed Oct 28 16:35` (trước khi sửa code lúc `Oct 29 09:30`), cần báo cáo trung thực trong `audit_report` và phối hợp cùng User nếu cần restart/trace trên daemon đang chạy.
- **Giải pháp 1-1 cho `F10` (`TASK-BUG-11`):**
  1. Xóa hoàn toàn `fakeMasterRepoForCreate` khỏi `update_column_metadata_test.go`; tạo `cdc-cms-service/internal/app/commands/master/create_master_mongo_real_test.go` chạy trực tiếp trên PostgreSQL `:5433` (`cdc_dw`) trong transaction rollback để kiểm chứng đầu-cuối `CreateMasterHandler.Handle` + `masterRepoGorm` + `masterReadRepoGorm.ListEnriched`.
  2. Tạo `centralized-data-service/internal/handler/source/discover_handler_live_pg_test.go` chạy trực tiếp trên PostgreSQL `:5433` (`cdc_dw`) và nguồn PostgreSQL `10.200.187.110:5432` (`source_object_id = 214`, `shadow_binding_id = 219`) để kiểm chứng `InferSourceColumnsWithError` và `ScanFieldsDebezium` lấy đúng danh sách cột DDL thật.

### Code Demo Bổ sung (`F8`, `F9`, `F10`)
```go
// centralize-data-service/internal/repository/master/sync_runtime_state_repo.go (Fix F9):
func (r *SyncRuntimeStateRepo) SyncMasterBindingDatabase(ctx context.Context, masterBindingID int64, dbName, masterTable string) error {
	if r == nil || r.db == nil || masterBindingID <= 0 || dbName == "" || masterTable == "" {
		return nil
	}
	return r.db.WithContext(ctx).Exec(
		`UPDATE cdc_system.master_binding
		    SET master_database = ?,
		        master_schema = ?,
		        physical_table_fqn = ?,
		        updated_at = NOW()
		  WHERE id = ? AND (COALESCE(master_database, '') <> ? OR COALESCE(master_schema, '') <> ?)`,
		dbName, dbName, dbName+"."+masterTable, masterBindingID, dbName, dbName,
	).Error
}
```
```go
// centralized-data-service/internal/service/master/master_ddl_generator_mongo.go (Fix F8 & F9):
	if g.runtimeStateRepo != nil {
		_ = g.runtimeStateRepo.SyncMasterBindingDatabase(ctx, reg.ID, dbName, reg.MasterTable)
	}
```

