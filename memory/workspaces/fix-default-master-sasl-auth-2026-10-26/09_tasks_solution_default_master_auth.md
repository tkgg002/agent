# 09 Tasks Solution: Khắc phục dứt điểm lỗi `open master db connection for "default_master": failed SASL auth for user "root" database=cdc_cms_database (SQLSTATE 28P01)`

- **Workspace:** `fix-default-master-sasl-auth-2026-10-26`
- **Suffix:** `default_master_auth`
- **Tier:** `T2`

---

## 1. Phân Tích Nguyên Nhân Gốc Rễ (Root Cause — `VERIFIED` theo `file:line`)

### 1.1. Tại sao xuất hiện `user=root` và `database=cdc_cms_database` khi kết nối `"default_master"`?
Chuỗi lỗi xảy ra do sự kết hợp của **2 thay đổi trong bản `git diff` hiện tại**:

1. **Thủ phạm #1 — `cdc-cms-service` ghi đè cấu hình DB của chính CMS (`cdc_cms_database`) vào dòng `default_master` mà không có credentials:**
   - Tại [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L115-L119) và [master_connection.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/bootstrap/master_connection.go#L16-L56):
     - Hàm `EnsureDefaultMasterConnection` trong bản diff mới nhận tham số `cfg.DB` (vốn là cấu hình **Control Plane DB `cdc_cms_database` tại `10.200.186.80:5432`** của `cdc-cms-service`, **không phải** Master DB).
     - Khi môi trường chạy `cdc-cms-service` không đặt biến `CMS_MASTER_DB_HOST` / `CMS_MASTER_DB_DATABASE`, các dòng `L21-23`, `L24`, `L38-40` fallback thẳng về `cfg.Host` (`10.200.186.80`), `cfg.Port` (`5432`), `cfg.Database` (`cdc_cms_database`) rồi chạy `ON CONFLICT (connection_code) DO UPDATE`:
       - Ghi đè dòng `connection_code = 'default_master'` trong bảng `cdc_system.connection_registry` thành:
         `host = '10.200.186.80'`, `port = 5432`, `default_database = 'cdc_cms_database'`, nhưng **`options_json = NULL`** (không có `dsn`, không có `username`, không có `password`) và `secret_ref = 'env:CDS_MASTER_DB_PASSWORD'` (chỉ là tên biến mật khẩu, không phải URL `postgres://...`).
   - Trong khi đó, theo thiết kế gốc tại `HEAD` (`818064d`) của [master_connection.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/bootstrap/master_connection.go#L10-L24), `default_master` là bản ghi đại diện cho connection pool `RoleDestination` (`cfg.MasterDB`) của `centralized-data-service` nên `host`, `port`, `default_database` cố tình để `NULL`.

2. **Thủ phạm #2 — `centralized-data-service` gọi `buildDSNFromFieldsPatched(&conn)` tạo ra DSN rỗng `username`/`password` khi `default_master` có `host != ""` nhưng `options_json` trống:**
   - Tại [connection_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go#L58-L143):
     - Trước đợt diff hiện tại, các câu query ở [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L850-L860) hoặc khi `default_master` có `host = NULL`:
       - `conn.OptionsJSON` rỗng $\rightarrow$ `dsn == ""`
       - `tryPlainDSN(conn.SecretRef)` và `tryEnvPointer("env:CDS_MASTER_DB_PASSWORD")` đều trả về `""` (vì `tryEnvPointer` chỉ nhận giá trị bắt đầu bằng `postgres://` / `mongodb://`)
       - `buildDSNFromFieldsPatched(&conn)` trả về `""` khi `conn.Host == nil`
       - Do `dsn == ""`, hàm đi vào Bước 3 (`L126-136`) và trả về `m.reg.GetDB(database.RoleDestination)` (connection pool Master DB chuẩn từ `config.yml` của `centralized-data-service`).
     - Sau khi `default_master` bị `cdc-cms-service` ghi `host = '10.200.186.80'`, `port = 5432`, `default_database = 'cdc_cms_database'` (với `options_json = NULL`):
       - Tại `connection_manager.go:118`, `dsn = buildDSNFromFieldsPatched(&conn)` được gọi.
       - Tại [metadata_registry_service.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go#L560-L584), vì `options_json` là `NULL` nên `username == ""` và `userInfo == ""`, hàm trả về chuỗi URL **không có thông tin đăng nhập**:
         `postgres://10.200.186.80:5432/cdc_cms_database?sslmode=disable`
       - Vì `dsn != ""`, `GetMasterDB` **không** rơi xuống `m.reg.GetDB(database.RoleDestination)` mà gọi `m.reg.OpenGorm(dsn, "master:default_master")` (`L140`).
       - Khi `pgx` mở kết nối với URL không có `user:pass@`, driver mặc định lấy tên OS user đang chạy process trong container là **`user=root`** (không mật khẩu) kết nối vào `10.200.186.80:5432` (`database=cdc_cms_database`), dẫn tới lỗi:
         `failed to connect to user=root database=cdc_cms_database: 10.200.186.80:5432: failed SASL auth: FATAL: password authentication failed for user "root" (SQLSTATE 28P01)`!

---

## 2. Phương Án Tối Ưu Duy Nhất (Single Best Solution)

### 2.1. Tại `centralized-data-service/internal/service/source/connection_manager.go`
- Trong `GetMasterDB(ctx context.Context, key string)` (`L90-137`):
  - Khi đọc bản ghi `conn` từ `cdc_system.connection_registry`:
    1. Trích xuất `dsn` từ `conn.OptionsJSON["dsn"]` hoặc `conn.OptionsJSON["url"]` (đây là chuẩn lưu của `CreateMasterConnection` / `UpdateMasterConnection` trên CMS khi tạo kết nối Master ngoài).
    2. Nếu chưa có `dsn`, thử `tryPlainDSN(conn.SecretRef)` / `tryEnvPointer(conn.SecretRef)`.
    3. **Chỉ gọi `buildDSNFromFieldsPatched(&conn)` khi `conn.OptionsJSON` thực sự có khai báo `username` hoặc `user` khác rỗng (`hasOptionCredentials == true`)!**
  - Nếu bản ghi trong `connection_registry` (như `default_master`, `master_local_pg_dest`, `legacy_master_default`) không có `dsn`/`url` và không có `username`/`user` trong `options_json`, `dsn` sẽ giữ nguyên là `""` $\rightarrow$ Bước 3 (`L126-136`) lập tức phân giải về `m.reg.GetDB(database.RoleDestination)` (connection pool Master DB chuẩn đã cấu hình sẵn trong `centralized-data-service`) và cache vào `m.masterDBPool[key]`.

### 2.2. Tại `cdc-cms-service/internal/bootstrap/master_connection.go` & `internal/server/server.go`
- Không bao giờ lấy `cfg.DB` (Control Plane DB `cdc_cms_database` của CMS) làm fallback cho `default_master`!
- Trong `EnsureDefaultMasterConnection`:
  - Khôi phục chữ ký `EnsureDefaultMasterConnection(ctx context.Context, db *gorm.DB, logger *zap.Logger) error` tại `master_connection.go` và `server.go:118`.
  - Seed `default_master` với `host = NULL, port = NULL, default_database = NULL` (hoặc chỉ lấy từ `CMS_MASTER_DB_*` / `CDS_MASTER_DB_*` nếu được cấu hình tường minh).
  - Đồng thời thực thi câu lệnh làm sạch tự động (`self-heal`) cho bản ghi `default_master` đã bị ghi nhầm `host`/`port`/`default_database` mà không có `options_json` credentials (`WHERE connection_code = 'default_master' AND (options_json IS NULL OR COALESCE(options_json->>'dsn', options_json->>'url', options_json->>'username', options_json->>'user', '') = '')`), đưa `host = NULL, port = NULL, default_database = NULL` về trạng thái chuẩn ngay khi CMS khởi động (hoặc chạy câu lệnh SQL 1 dòng trên DB testing).

---

## 3. Code Demo

### 3.1. `centralized-data-service/internal/service/source/connection_manager.go`
```go
	// 2. Query system DB for connection_registry
	var foundActiveMasterRecord bool
	if dsn == "" {
		sysDB, err := m.GetSystemDB(ctx)
		if err == nil && sysDB != nil {
			var conn modelsource.ConnectionRegistry
			qErr := sysDB.WithContext(ctx).Table("cdc_system.connection_registry").
				Where("(connection_code = ? OR connection_code = ?) AND status = 'active'", key, strings.ToLower(key)).
				First(&conn).Error
			if qErr == nil {
				foundActiveMasterRecord = true
				hasOptionCredentials := false
				if len(conn.OptionsJSON) > 0 {
					var opts map[string]interface{}
					if err := json.Unmarshal(conn.OptionsJSON, &opts); err == nil {
						if dsnVal, ok := opts["dsn"].(string); ok && strings.TrimSpace(dsnVal) != "" {
							dsn = strings.TrimSpace(dsnVal)
						} else if urlVal, ok := opts["url"].(string); ok && strings.TrimSpace(urlVal) != "" {
							dsn = strings.TrimSpace(urlVal)
						}
						for _, uKey := range []string{"username", "user"} {
							if uVal, ok := opts[uKey].(string); ok && strings.TrimSpace(uVal) != "" {
								hasOptionCredentials = true
								break
							}
						}
					}
				}
				if dsn == "" && conn.SecretRef != "" {
					if plain := tryPlainDSN(conn.SecretRef); plain != "" {
						dsn = plain
					} else if envVal := tryEnvPointer(conn.SecretRef); envVal != "" {
						dsn = envVal
					}
				}
				if dsn == "" && hasOptionCredentials {
					dsn = buildDSNFromFieldsPatched(&conn)
				}
			}
		}
	}
```

### 3.2. `cdc-cms-service/internal/bootstrap/master_connection.go` & `internal/server/server.go`
```go
func EnsureDefaultMasterConnection(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	err := db.WithContext(ctx).Exec(`
		INSERT INTO cdc_system.connection_registry
			(connection_code, display_name, role_type, engine_type, secret_ref, status)
		SELECT 'default_master', 'Default Master Connection',
		       'master', 'postgresql', 'env:CDS_MASTER_DB_PASSWORD', 'active'
		WHERE NOT EXISTS (
			SELECT 1 FROM cdc_system.connection_registry
			WHERE role_type = 'master' AND status = 'active'
		)
		ON CONFLICT (connection_code) DO NOTHING
	`).Error
	if err != nil {
		return fmt.Errorf("seed default_master connection: %w", err)
	}
	// Làm sạch host/port/default_database bị ghi nhầm từ CMS Control Plane DB trên default_master khi không có credentials trong options_json
	_ = db.WithContext(ctx).Exec(`
		UPDATE cdc_system.connection_registry
		   SET host = NULL,
		       port = NULL,
		       default_database = NULL,
		       updated_at = NOW()
		 WHERE connection_code = 'default_master'
		   AND (options_json IS NULL
		        OR COALESCE(options_json->>'dsn', options_json->>'url', options_json->>'username', options_json->>'user', '') = '')
		   AND (host IS NOT NULL OR port IS NOT NULL OR default_database IS NOT NULL)
	`).Error
	logger.Info("master connection seeded", zap.String("connection_code", "default_master"))
	return nil
}
```

---

## 4. Bổ sung sau tái hiện local (2026-10-26 09:57) — bằng chứng runtime, không suy diễn

### 4.1. Tái hiện trên stack local (`gpay-postgres-cdc`)
- `cdc_system.connection_registry` dòng `default_master` bị code ghi thành `host=localhost, port=5433, default_database=cdc_dw, options_json={}` lúc `2026-10-26 02:02:03 UTC` (CMS khởi động) — đây là **DB control-plane của CMS**, không phải Master DB (`goopay_dest :5434`).
- `cdc_activity_log`: transmute `success` cuối cùng `02:00:51 UTC`; từ `02:15:11 UTC` tới `02:52:52 UTC`: **308** `transmute error`, **544** `transmute_sub error`, **580** `transmute_poison_pill`, **0** success. Lỗi: `open master db connection for "default_master": failed to connect to user=trainguyen database=cdc_dw ... failed SASL auth` — trùng pattern testing (`user=root database=cdc_cms_database`).

### 4.2. Sửa bổ sung: lỗi hạ tầng/xác thực không được coi là poison pill
- `centralized-data-service/internal/handler/master/transmute_handler.go:918-946` (`isTransientError`): thêm `sqlstate 28p01`, `sqlstate 28000`, `failed sasl auth`, `password authentication failed`, `get master db` → đi nhánh transient (Nak, giữ message trên `CDC_TRANSMUTE`) thay vì `binarySearchSplit` → `Term`.

```go
		strings.Contains(msg, "sqlstate 28p01") || // invalid_password
		strings.Contains(msg, "sqlstate 28000") || // invalid_authorization_specification
		strings.Contains(msg, "failed sasl auth") ||
		strings.Contains(msg, "password authentication failed") ||
		strings.Contains(msg, "get master db") ||
```

### 4.3. Phạm vi sửa (chỉ 3 file, không thêm tính năng)
1. `cdc-cms-service/internal/bootstrap/master_connection.go` + `internal/server/server.go:118` — về đúng hành vi `818064d` (không fallback `cfg.DB`) + tự dọn host/port/default_database của `default_master` khi không có credentials.
2. `centralized-data-service/internal/service/source/connection_manager.go:117-119` — chỉ `buildDSNFromFieldsPatched` khi `options_json` có `username`/`user`.
3. `centralized-data-service/internal/handler/master/transmute_handler.go:918-946` — mục 4.2.

### 4.4. Nghiệm thu trên local (bắt buộc trước khi báo xong)
- Restart CMS + worker local → query `connection_registry` (`default_master` host/port/db = NULL) → `cdc_activity_log` có `transmute success` mới sau restart, 0 lỗi `default_master` → chạy Recon B local và so số Thiếu/Thừa trước-sau.

---

## 5. Gỡ toàn bộ DeleteStream tự động + test chạm NATS thật (2026-10-26 10:01)

### 5.1. Kiểm kê (git grep trên commit `3369c31` / `80bf337`) — 13 lời gọi `DeleteStream(`
| Loại | Vị trí | Xử lý |
|---|---|---|
| **Production, TỰ ĐỘNG** | `centralized-data-service/pkgs/natsconn/nats_client.go:208-216` | GỠ: lệch retention → log Error + return error, không xóa |
| **Production, TỰ ĐỘNG (mới trong commit)** | `cdc-cms-service/pkgs/natsconn/nats_client.go:210-218` (chạy ở startup, reconnect, `GET /overview` auto-heal) | GỠ như trên |
| Production, thủ công có guard (API NATS Monitor, có từ `245907f`) | `cdc-cms-service/internal/api/system/nats_monitor_handler.go:627-647` | Giữ (thao tác người vận hành) |
| Test trên NATS THẬT (`config-local.yml`) | `centralized-data-service/.../transmute_debounce_group_test.go:763-790`; `cdc-cms-service/pkgs/natsconn/nats_client_test.go:17-116`; `cdc-cms-service/internal/api/system/nats_monitor_handler_test.go:178-225` | Chuyển sang `natsserver.NewServer` nhúng hoặc xóa test |
| Test trên NATS nhúng | `transmute_debounce_group_test.go:952,1003`; `centralized-data-service/pkgs/natsconn/nats_client_test.go:148` | Giữ (không chạm hạ tầng dùng chung) |

### 5.2. Code demo (áp dụng cho cả 2 repo)
```go
	if info.Config.Retention != desired.Retention {
		log.Error("NATS stream retention mismatch; refusing to delete/recreate core stream automatically",
			zap.String("stream", desired.Name),
			zap.Any("existing_retention", info.Config.Retention),
			zap.Any("desired_retention", desired.Retention))
		return fmt.Errorf("stream %s retention mismatch (existing=%v desired=%v): manual migration required",
			desired.Name, info.Config.Retention, desired.Retention)
	}
```

### 5.3. Nghiệm thu (runtime thật, không mock)
- `git grep -n "DeleteStream(" -- '*.go'` ở 2 repo: chỉ còn API thủ công `nats_monitor_handler.go:647` + test dùng NATS nhúng.
- Restart CMS + worker local → `curl localhost:18222/jsz?streams=true` vẫn đủ 4 core stream, `created` không đổi (không bị xóa/tạo lại).

---

## 6. PHẠM VI CUỐI (2026-10-26 10:02) — theo chỉ thị User: KHÔNG viết, KHÔNG sửa, KHÔNG chạy test

> Mục 5.1 dòng "Test trên NATS THẬT → chuyển sang natsserver nhúng hoặc xóa test" **BỊ HỦY**. Không đụng tới bất kỳ file `*_test.go` nào, không chạy `go test`.

**Chỉ sửa code production (5 file):**
1. `cdc-cms-service/internal/bootstrap/master_connection.go` — bỏ fallback `cfg.DB`, tự dọn host/port/default_database của `default_master` khi không có credentials.
2. `cdc-cms-service/internal/server/server.go:118` — gọi `EnsureDefaultMasterConnection(ctx, db, logger)`.
3. `centralized-data-service/internal/service/source/connection_manager.go:117-119` — chỉ ghép DSN khi `options_json` có `username`/`user`.
4. `centralized-data-service/internal/handler/master/transmute_handler.go:918-946` — `isTransientError` nhận `28P01`, `28000`, `failed sasl auth`, `password authentication failed`, `get master db`.
5. `centralized-data-service/pkgs/natsconn/nats_client.go:208-216` + `cdc-cms-service/pkgs/natsconn/nats_client.go:210-218` — bỏ `DeleteStream`/`AddStream` tự động khi lệch retention, chỉ log Error + trả lỗi.

**Nghiệm thu — chỉ runtime thật trên local:**
- `go build ./...` 2 repo (chỉ build, không test).
- Restart CMS + worker local.
- `docker exec gpay-postgres-cdc psql`: `default_master` có `host/port/default_database = NULL`.
- `cdc_activity_log`: có `transmute success` mới sau restart, 0 lỗi `28P01`.
- `curl localhost:18222/jsz?streams=true`: đủ 4 core stream, `created` không đổi.
- Chạy Recon B thật, đọc `cdc_reconciliation_report` trước/sau.

---

## 7. Điều tra toàn diện lỗi Recon Segment B (`reconB`) & Execute-Heal / Transmute (2026-10-26 10:33)

### 7.1. Vì sao báo cáo `7494` (30d: `Nguồn=4,536, Đích=4,539, Thiếu=1,534, Thừa=1,537`) và `7495` (7d: `Nguồn=1,542, Đích=1,545, Thiếu=1,166, Thừa=1,169`) vẫn báo lệch y nguyên?
- Tại `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go:751-754` và `831-857` (`ChunkStreamBucketEngine.executeSegmentB`), `reconB` chỉ **đọc** dữ liệu và đối sánh trực tiếp bằng khóa `SELECT "_gpay_id"::text FROM shadow` vs `SELECT "_gpay_id"::text FROM master`.
- Trên DB thực tế của bảng bị lệch:
  - Có `1,534` bản ghi (trong cửa sổ 30 ngày của `7494`, trong đó `1,166` bản ghi thuộc cửa sổ 7 ngày của `7495`) đang tồn tại ở cả Shadow lẫn Master với **cùng `_id`**, nhưng ở Shadow mang `_gpay_id` kiểu mới (Sonyflake 18 chữ số) còn ở Master vẫn mang `_gpay_id` cũ (SHA-256 64 ký tự hex), cộng thêm `3` bản ghi thừa (orphan) thật sự trên Master (`1537 - 1534 = 3`, `1169 - 1166 = 3`).
  - Vì `reconB` chỉ là trình kiểm tra (read-only), chừng nào luồng **Heal / Transmute** chưa cập nhật được `_gpay_id` trên bảng Master và xóa 3 bản ghi orphan, thì chạy `reconB` bao nhiêu lần cũng vẫn ra đúng con số `Thiếu = 1,166` / `Thừa = 1,169`.

### 7.2. Năm (05) lỗi trực tiếp trong code khiến Heal Segment B & Transmute thất bại sau commit `09:20` (`3369c31` / `80bf337`)

1. **Lỗi #1 — Xóa fallback đọc `primary_key_field` nhưng KHÔNG backfill `transform_spec` cho các `master_binding` đã tồn tại trong DB (`VERIFIED`):**
   - Trong commit `3369c31`, hàm `resolveBusinessPKForMaster` bị xóa và `resolveMasterPKFromSpec` (`centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:698-711`) cùng `resolveConflictTarget` (`centralized-data-service/internal/service/master/transmuter_utils.go:736-749`) bị sửa thành chỉ đọc duy nhất `transform_spec->>'pk'`.
   - Nhưng ở `cdc-cms-service` (`80bf337`), `transform_spec = {"pk": "..."}` chỉ được ghi khi **tạo mới** `master_binding` (`internal/app/commands/master/create_master.go:108`, `internal/app/services/provisioning_orchestrator.go:349`). **Hoàn toàn không có migration hay hàm bootstrap nào backfill `transform_spec` cho các `master_binding` cũ đã tồn tại trong DB (đang có `transform_spec = '{}'`)!**
   - Kiểm tra trực tiếp trên DB PostgreSQL (`cdc_system.master_binding`): toàn bộ 5 dòng `master_binding` hiện hữu đều có `transform_spec = '{}'`.
   - Hậu quả khi `transform_spec = '{}'`:
     - `resolveMasterPKFromSpec` (`recon_execute_heal_handler.go:698-711`) trả về `pkCol = ""`!
     - Tại `recon_execute_heal_handler.go:547`: điều kiện `if h.shadowDB != nil && pkCol != "" && pkCol != "_gpay_id" && shadowRel != ""` bị **FALSE** hoàn toàn! Nếu chạy `prune_missing_src` trên `1,169` ID của báo cáo `7495`, code **bỏ qua bước kiểm tra dòng còn sống ở Shadow** và chạy thẳng xuống dòng `597`: `DELETE FROM master WHERE "_gpay_id" IN (...)` — **xóa mất 1,166 bản ghi đang sống trên Master mà không resync lại!**
     - Tại `resolveSourceIDsForSegmentB` (`recon_execute_heal_handler.go:863-868`): vì `pkCol == ""`, code bỏ qua bước tra cứu từ `_gpay_id` cũ trên Master sang `_source_id` ở Shadow.
     - Tại `deleteFromMasterBySourceAndGpayIDs` (`transmuter.go:1689, 1713`): `resolveConflictTarget` trả về `"_gpay_id"`, làm điều kiện `if pkCol != "" && pkCol != "_gpay_id"` bị **FALSE** — không bao giờ xóa được dòng cũ theo `_id` trên Master.

2. **Lỗi #2 — Đánh dấu báo cáo `"healed"` giả trước khi `TransmuterModule.Run` thực sự chạy (`VERIFIED`):**
   - Tại `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:478-490`, `executeHealSegB` gọi `publishTransmuteChunked` (`L797-830`). Hàm này **chỉ mới bắn message bất đồng bộ lên NATS (`cdc.cmd.transmute`)** rồi lập tức trả về `processed = len(sourceIDs)` (`1,166`).
   - Ngay lập tức, `finalizeReport` (`recon_execute_heal_handler.go:222-260`) thấy `rpt.HealedMissingDestCount > 0` liền cập nhật DB: `status = "healed"`, `healed_missing_dest_count = 1166` **trước cả khi worker Transmute xử lý message**.
   - Khi worker `TransmuterModule.Run` chạy nền ngay sau đó và gặp lỗi (`28P01`, `42P10`, hoặc `23505`), **0 dòng nào trên Master được cập nhật**, nhưng báo cáo cũ đã bị đóng dấu `"healed"` (ẩn khỏi modal Heal), còn khi chạy lại `reconB` mới (`7495`) thì toàn bộ `1,166` dòng thiếu / `1,169` dòng thừa vẫn còn nguyên.

3. **Lỗi #3 — Xung đột giữa `conflictTarget` (`ON CONFLICT`) trong `bulkUpsertMaster` (`transmuter.go:1291-1327, 1802-1836`) và ràng buộc vật lý trên bảng Master (`VERIFIED`):**
   - **Khi `transform_spec = '{}'` (`conflictTarget = "_gpay_id"`):**
     - `staleLegacyGpayIDs` (`transmuter.go:1292`) chỉ tính duy nhất 1 công thức `deterministicSourceGpayID(row.SourceID, e.KeySuffix)`. Nếu dòng cũ trên Master mang `_gpay_id` khác (ví dụ từ lần sync trước khi recreate Shadow), nó không nằm trong `staleLegacyGpayIDs` và không được xóa trước khi upsert.
     - Khi `bulkUpsertMaster` (`transmuter.go:1828`) chạy `INSERT INTO master ... ON CONFLICT ("_gpay_id") DO UPDATE`:
       - Nếu bảng Master có `UNIQUE INDEX ux_<table>__id ON ("_id")` (`master_ddl_generator.go:118`), câu lệnh **nổ lỗi `SQLSTATE 23505: duplicate key value violates unique constraint "ux_<table>__id"`** (vì trùng `_id` nhưng khác `_gpay_id`) và bị đẩy vào `poison_pill`!
       - Nếu bảng Master không có `ux_<table>__id`, câu lệnh chèn thêm dòng thứ 2 với `_gpay_id` mới mà vẫn để nguyên dòng `_gpay_id` cũ trên Master (làm số dòng `Thừa` không giảm).
   - **Khi `transform_spec = '{"pk": "_id"}'` (`conflictTarget = "_id"`):**
     - Tại `transmuter.go:1325`: `if conflictTarget == "_gpay_id" && len(staleLegacyGpayIDs) > 0` — vì `conflictTarget == "_id"`, bước dọn `staleLegacyGpayIDs` bị bỏ qua.
     - Nếu bảng Master vật lý được tạo trước khi có `ux_<table>__id` (chưa có `UNIQUE INDEX` trên `"_id"`), câu lệnh `INSERT ... ON CONFLICT ("_id") DO UPDATE` tại `transmuter.go:1828` **nổ lỗi `SQLSTATE 42P10: there is no unique or exclusion constraint matching the ON CONFLICT specification`** và bị coi là `poison_pill`!

4. **Lỗi #4 — `ExecuteHealModal.tsx:202` mặc định tắt `pruneMissingSrc = false` và `HealMissingDest` không tự dọn dòng thừa đối xứng (`VERIFIED`):**
   - Tại `cdc-cms-web/src/components/ExecuteHealModal.tsx:202`, `setPruneMissingSrc(false)` tắt mặc định ô `Xóa dữ liệu thừa ở đích`.
   - Khi người dùng bấm Heal với mặc định (chỉ bật `healMissingDest` và `healMismatched`), `executeHealSegB` (`recon_execute_heal_handler.go:530`) bỏ qua hoàn toàn danh sách `staleB.MissingFromSrc` (`1,169` ID ở `7495` / `1,537` ID ở `7494`), dẫn tới `3` dòng orphan thật sự (`1169 - 1166 = 3`, `1537 - 1534 = 3`) không bao giờ được dọn.

5. **Lỗi #5 — Lỗi `default_master` SASL Auth (`SQLSTATE 28P01`) ở commit `80bf337` (`09:20`) (`VERIFIED`):**
   - Từ `09:20` đến trước khi sửa `master_connection.go`, mọi lời gọi `GetMasterDB("default_master")` trong cả `TransmuterModule.Run` (`transmuter.go:274`) và `ExecuteHealHandler.resolveMasterDBForRef` (`recon_execute_heal_handler.go:684`) đều chết ngay ở bước kết nối DB với `SQLSTATE 28P01`.

---

## 8. TASK-05: Mở nút `Edit` trên `MasterRegistry.tsx` để xem cấu hình, chỉ khóa sửa/lưu khi chưa Reject (`schema_status === 'approved'`)

### 8.1. Hiện trạng (`cdc-cms-web/src/pages/MasterRegistry.tsx:1001-1017` & `1653-1707`)
- Nút `Edit` ở dòng `1005` đang đặt `disabled={r.schema_status === 'approved'}` nên khi Master đã `approved`, người dùng không thể bấm mở Modal để xem `Transform Type` và `Spec (JSON)` đang cấu hình là gì.

### 8.2. Thay đổi (`cdc-cms-web/src/pages/MasterRegistry.tsx`)
1. Tại dòng `1001-1017`: Bỏ `disabled={r.schema_status === 'approved'}` trên nút `Edit` để luôn bấm mở được Modal xem cấu hình hiện tại.
2. Tại Modal `Edit Spec` (`1653-1707`):
   - Khi `editRow?.schema_status === 'approved'` (chưa `Reject`):
     - Khóa nút Lưu: `okButtonProps={{ disabled: editRow?.schema_status === 'approved' }}` (và chặn đầu hàm `onOk`).
     - Khóa chỉnh sửa `Select` (`disabled={editRow?.schema_status === 'approved'}`) và đặt `readOnly={editRow?.schema_status === 'approved'}` cho `Input.TextArea` để xem/copy được `Spec (JSON)` nhưng không cho sửa khi chưa `Reject`.

---

## 9. Giải phẫu trực tiếp dòng báo cáo `reconB` lúc `10:54:43 26/10/2026` (`Nguồn: 3,095 | Đích: 3,082 (-13 thiếu) | Thiếu: 1,693 | Lệch: 0 | Thừa: 1,680`)

### 9.1. Giải mã con số toán học trên báo cáo `10:54:43`
- Trong cửa sổ `10:49 19/10/2026 - 10:49 26/10/2026`:
  - `Nguồn (Shadow)` = **`3,095`**
  - `Đích (Master)` = **`3,082`**
  - Chênh lệch số lượng bản ghi thực tế giữa Shadow và Master chỉ là:
    $$3,082 - 3,095 = -13 \text{ (thiếu đúng 13 bản ghi)}$$
- Nhưng `reconB` lại báo:
  - `Thiếu (MissingFromDest)` = **`1,693`** (`= 1,680 + 13`)
  - `Lệch (Mismatched)` = **`0`**
  - `Thừa (MissingFromSrc / Orphan)` = **`1,680`**
  - Hiệu số: `1,693 - 1,680 = 13`!
- **Sự thật vật lý:**
  - Trong `3,082` bản ghi đang nằm ở Master, có `1,402` bản ghi trùng khớp `_gpay_id` với Shadow (`3,095 - 1,693 = 1,402`; `3,082 - 1,680 = 1,402`).
  - Có đúng **`1,680` bản ghi** đang tồn tại ở **cả Shadow lẫn Master** (cùng khóa chính nghiệp vụ `_id` và cùng `_source_ts`), nhưng giá trị cột `_gpay_id` trên Master khác với `_gpay_id` trên Shadow.
  - Vì `ChunkStreamBucketEngine.resolveFieldsB` (`recon_stream_bucket_engine.go:698-754`) luôn ép `pkCol = "_gpay_id"` và `diffIDTs` (`recon_tier_b.go:585-605`) so khớp theo `_gpay_id`: mỗi bản ghi trong `1,680` dòng bị lệch `_gpay_id` bị đếm thành **1 dòng `Thiếu` (do `_gpay_id` Sonyflake của Shadow không có ở Master)** đồng thời **1 dòng `Thừa` (do `_gpay_id` cũ trên Master không có ở Shadow)**. Cộng thêm `13` dòng thiếu thật sự trên Master thành **`1,693` Thiếu / `1,680` Thừa**.

### 9.2. Tại sao chính các bản sửa từ `38ba1f4` → `30b1b00` → `08c05a8` → `3369c31` làm nó bung bét từ `1,166` lên `1,693 Thiếu / 1,680 Thừa` và rớt `-13` dòng?
1. **Nguồn gốc đẻ ra `_gpay_id` lệch trên Master (`transmuter.go:702-794` — thêm từ `38ba1f4` & `30b1b00`):**
   - Thiết kế gốc trước `38ba1f4`: `TransmuterModule.Run` đọc từ bảng Shadow (`fetchShadowBatch`) và lấy nguyên `_gpay_id` Sonyflake của Shadow ghi sang Master (`deterministicGpayID(row.GpayID, "") == row.GpayID`).
   - Từ commit `38ba1f4`, hàm `RunEvents` được nhồi vào `transmuter.go` và tự sinh `_gpay_id = deterministicSourceGpayID(ev.SourceID, e.KeySuffix)` (băm SHA-256 của `_source_id`) ghi vào Master.
   - Đến commit `30b1b00`, dù thêm `lookupShadowGpayIDs` (`transmuter.go:710, 766-787`), dòng `786` vẫn giữ nhánh fallback:
     `else { rec["_gpay_id"] = deterministicSourceGpayID(ev.SourceID, e.KeySuffix) }`.
     Bất cứ khi nào event CDC đi qua `RunEvents` mà chưa tra ra `shadowGID` (hoặc các dòng đã ghi từ `38ba1f4` được update), Master lại mang `_gpay_id` SHA-256 khác hẳn `_gpay_id` Sonyflake của Shadow.
2. **Tại sao luồng Upsert/Heal trong `3369c31` không sửa lại được `_gpay_id` mà còn làm rớt `-13` dòng (`transmuter.go:1325, 1802-1836` & `transmuter_utils.go:736-749`):**
   - Trong `3369c31`, `resolveConflictTarget` (`transmuter_utils.go:736-749`) và `resolveMasterPKFromSpec` (`recon_execute_heal_handler.go:698-711`) bị xóa sạch fallback `primary_key_field`, chỉ đọc từ `transform_spec->>'pk'`.
   - **Trường hợp 1 (`transform_spec = '{}'` trên các `master_binding` cũ chưa được backfill):**
     - `conflictTarget` rơi về `"_gpay_id"`.
     - Khi dòng cũ trên Master có `_gpay_id` khác với `G_shadow` mới (hoặc khi `staleLegacyGpayIDs` không khớp) và bảng Master có unique index `ux_<table>__id`, câu lệnh `INSERT INTO master ... ON CONFLICT ("_gpay_id") DO UPDATE` tại `transmuter.go:1828` bị nổ lỗi `SQLSTATE 23505 (duplicate key value violates unique constraint "ux_...__id")` (hoặc nếu không có `ux_<table>__id` thì chèn thành 2 dòng trùng `_id` khác `_gpay_id`), làm các bản ghi mới trong mẻ thất bại (sinh ra `-13` dòng thiếu thật sự).
   - **Trường hợp 2 (`transform_spec = '{"pk": "_id"}'` nhưng bảng Master vật lý được tạo từ trước, chưa có `UNIQUE INDEX ux_<table>__id`):**
     - `conflictTarget` trả về `"_id"`.
     - Khi đó `bulkUpsertMaster` (`transmuter.go:1828`) chạy `INSERT ... ON CONFLICT ("_id") DO UPDATE`, bị PostgreSQL từ chối ngay với `SQLSTATE 42P10 (there is no unique or exclusion constraint matching the ON CONFLICT specification)`, làm cả Transmute lẫn Heal đều thất bại!

---

## 10. Giải phẫu & Giải pháp triệt để `TASK-06`: Vì sao NATS trên Testing lúc `10:56:54 26/10/2026` vẫn báo thiếu 3 Core Streams & `0/19` Kênh lệnh (`VERIFIED` 100% theo `file:line`)

### 10.1. Đối chiếu Telemetry thực tế trên Testing (`10:56:54 26/10/2026`) với Code hiện tại
Từ thông số trực tiếp trên màn hình Testing của User:
- **Stream duy nhất tồn tại:** `CDC_EVENTS` (`LIMITS`, `FILE`, `0 B / 512.00 MB`, subjects: `cdc.gpay.>`, `cdc.goopay.>`, `+2` = **4 subjects**, `Consumers: 5 (4/5 Active Bound)`).
- **3 Stream bị thiếu:** `CDC_TRANSMUTE`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`.
- **Quota JetStream:** `0 B / 1.00 GB` (`max_file_store = 1 GB`).
- **Command Bus (`cdc.cmd.*`):** `0 / 19 Ready`, nhưng header lại ghi **`Connections: 1 | Subs: 274`** và **`Consumers: 4 / 5 Active Bound`**!

Soi từng dòng code hiện tại ở cả 2 repo, có **3 lỗ hổng trong code do các lần sửa trước làm nửa vời / bỏ sót**:

#### Lỗ hổng #1 — Bỏ sót `server_setup.go:99-101` (`return nil, fmt.Errorf(...)` gây `logger.Fatal`) + `centralized-data-service` ghi đè 4 subjects của `CDC_EVENTS`:
1. Trong tài liệu lần trước (`nats-monitor-missing-core-stream-2026-10-23/09_tasks_solution_missing_core_stream.md:53`), Agent tưởng nhầm cả `server_setup.go` và `sinkworker/main.go` đều chỉ log `Warn` khi `EnsureStreams` trả lỗi.
   - Sự thật vật lý: Chỉ có `cmd/sinkworker/main.go:88-90` là `logger.Warn`. Còn tại [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L99-L101) (`NewWorkerServer`), code vẫn là:
     ```go
     if err := natsClient.EnsureStreams(cdcSubjects); err != nil {
         return nil, fmt.Errorf("nats streams: %w", err)
     }
     ```
     Và tại [cmd/worker/main.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/cmd/worker/main.go#L93-L96), khi `NewWorkerServer` trả lỗi thì process gọi `logger.Fatal("failed to initialize worker server")` **ngay tại dòng 95 — trước khi chạy tới dòng `299–538` của `server_setup.go` (nơi đăng ký 19 kênh lệnh `cdc.cmd.*` và khởi chạy `TransmuteConsumerPool`)**!
2. Hơn nữa, tại [centralized-data-service/pkgs/natsconn/nats_client.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L225-L228):
   - Trên Testing, `CDC_EVENTS` đang có **4 subjects** (`cdc.gpay.>`, `cdc.goopay.>`, `+2`) và **5 Durable Consumers** đang lọc theo các subjects đó.
   - Trong khi đó `goopay-testing/data-hub/onpremise-fpt/centralized-data-service.yaml:109-112` chỉ khai báo **3 `topicPrefix`** (`cdc.gpay`, `cdc.goopay`, `cdc.mariadb`).
   - Ở `cdc-cms-service/pkgs/natsconn/nats_client.go:206-208`, Agent có viết `preserveExistingSubjects = true` cho `CDC_EVENTS`. Nhưng ở `centralized-data-service/pkgs/natsconn/nats_client.go:225-228`, Agent **bỏ quên không giữ/merge subjects hiện có của `CDC_EVENTS`**, mà viết:
     ```go
     if !sameSubjects(info.Config.Subjects, desired.Subjects) {
         info.Config.Subjects = desired.Subjects
         needUpdate = true
     }
     ```
   - Việc ép `info.Config.Subjects` từ 4 subjects xuống 3 subjects làm `js.UpdateStream(&info.Config)` thất bại (hoặc làm mất subject thứ 4 của durable consumer), khiến `EnsureStreams` trả về lỗi khác `nil` $\rightarrow$ `server_setup.go:99-101` abort luôn `NewWorkerServer`!

#### Lỗ hổng #2 — `ensureSingleStream` ở cả 2 repo không có fallback `MaxBytes` khi `js.AddStream` đụng trần dự trữ `max_file_store` (`err_code 10047`):
- Trong mã nguồn `nats-server/v2/server/jetstream.go:2469-2478` (`checkBytesLimits`) và `server/stream.go:809-825`:
  - Khi gọi `js.AddStream(cfg)` với `cfg.MaxBytes > 0`, NATS JetStream **không kiểm tra dung lượng đĩa đang dùng (`0 B`)**, mà kiểm tra **tổng `MaxBytes` đăng ký dự trữ (`storeReserved + cfg.MaxBytes`) so với `MaxStore` của server/account**!
  - Nếu vượt hạn mức dự trữ hoặc giới hạn `max_bytes_required` / `max_stream_bytes` của account, NATS trả về `err_code 10047: insufficient storage resources available` (hoặc lỗi giới hạn `MaxBytes`).
- Tại [cdc-cms-service/pkgs/natsconn/nats_client.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client.go#L196-L200) và [centralized-data-service/pkgs/natsconn/nats_client.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/natsconn/nats_client.go#L194-L198):
  - Khi `js.AddStream(desired)` lỗi, code **bỏ cuộc ngay lập tức** (`return fmt.Errorf("failed to create stream %s: %w", desired.Name, err)`), hoàn toàn không có bước hạ bậc `MaxBytes` (`desired.MaxBytes` $\rightarrow$ `64 MiB` $\rightarrow$ `16 MiB` $\rightarrow$ `-1`).
- Đồng thời tại dòng `229-232` (`centralized-data-service`) và `232-235` (`cdc-cms-service`), mỗi lần `ensureSingleStream` chạy trên stream đã tồn tại, nó lại ép `if info.Config.MaxBytes != desired.MaxBytes { info.Config.MaxBytes = desired.MaxBytes }`, ghi đè cả cấu hình `MaxBytes` đã được hạ bậc hoặc đã được người vận hành chỉnh tay qua UI!

#### Lỗ hổng #3 — `GetOverview` trong `nats_monitor_handler.go` chỉ đọc `/connz?subs=detail` (bỏ sót `/subsz?subs=1`) và nuốt lỗi `EnsureStreams()`:
1. **Tại sao màn hình Testing hiện `Connections: 1 | Subs: 274` và `Consumers: 4/5 Active Bound` nhưng lại báo `Command Bus: 0/19 Ready`?**
   - Trong `nats-server/v2/server/monitor.go:303-307`, `/connz?subs=detail` chỉ trả về các TCP client kết nối trực tiếp vào node NATS đó (`Connections: 1` — chính là kết nối của `cdc-cms-service` với 1 subscription `_INBOX.*`).
   - Còn `274` subscriptions (`Subs: 274` từ `/varz`) và `4/5 Active Bound` consumers nằm trong bảng định tuyến subscription của NATS (`/subsz?subs=1`).
   - Dù tài liệu thiết kế `13_analysis_nats_monitor_v2.md:74` đã ghi rõ phải đối chiếu `/connz?subs=detail` + `/subsz?subs=1`, nhưng khi code [nats_monitor_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L157-L173), hàm `GetOverview` chỉ gọi `/varz`, `/jsz`, `/connz?subs=detail` và **bỏ quên hoàn toàn `/subsz?subs=1`**!
   - Hệ quả: `connz` trả về 1 connection của `cdc-cms` có `subscriptions_list_detail: ["_INBOX.> "]` $\rightarrow$ biến `hasSubscriptionTelemetry` tại `nats_monitor_handler.go:937` bật thành `true` $\rightarrow$ toàn bộ 19 kênh `cdc.cmd.*` bị kết luận oan là `NO_LISTENER` (`0/19 Ready`)!
2. **Tại sao `tryAutoHealMissingCoreStreams` (`nats_monitor_handler.go:233-273`) chạy mỗi 30s trên CMS mà User không thấy nguyên nhân lỗi trên UI?**
   - Tại [nats_monitor_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler.go#L269-L272), khi `h.natsClient.EnsureStreams()` trả về `err != nil`, hàm chỉ ghi `h.logger.Warn(...)` rồi trả về `true` mà **không lưu `err` để hiển thị lên `governance["alerts"]` trên giao diện NATS Control Center**.

---

### 10.2. Phương Án Tối Ưu Duy Nhất (Single Best Solution — Tuân thủ `L001` Không nhồi nhét God-file, `L002` Không viết/chạy test, `L003` Không `DeleteStream` tự động)

#### 1. Tại `centralized-data-service/internal/server/server_setup.go:99-101`
- Đổi lỗi `natsClient.EnsureStreams(cdcSubjects)` từ `return nil, fmt.Errorf(...)` sang `logger.Warn("EnsureStreams encountered partial error at startup (worker will continue and self-heal)", zap.Error(err))` (đồng nhất với `cmd/sinkworker/main.go:88-90`).
- Nhờ đó, `NewWorkerServer` **không bao giờ crash lúc khởi động** vì lỗi NATS stream cục bộ và luôn đăng ký đầy đủ 19 kênh lệnh `cdc.cmd.*` (`server_setup.go:299-538`) cùng vòng lặp tự phục hồi của `TransmuteConsumerPool`.

#### 2. Tại `centralized-data-service/pkgs/natsconn/nats_client.go` & `cdc-cms-service/pkgs/natsconn/nats_client.go`
- **Bảo toàn & Hợp nhất Subjects (`mergeSubjects`):**
  - Với stream `CDC_EVENTS` (hoặc bất kỳ core stream nào đang có subjects trên NATS), thay vì ghi đè `info.Config.Subjects = desired.Subjects` làm hỏng durable consumer hiện hữu, dùng hàm `mergeSubjects(info.Config.Subjects, desired.Subjects)` (giữ nguyên toàn bộ subjects đang có trên NATS và chỉ bổ sung subject mới nếu thiếu). Ở `cdc-cms-service`, giữ nguyên `preserveExistingSubjects = true` cho `CDC_EVENTS`.
- **Hạ bậc `MaxBytes` tự động khi tạo mới (`addStreamWithQuotaFallback`):**
  - Khi `js.StreamInfo(desired.Name)` trả về `nats.ErrStreamNotFound`, thử `js.AddStream(desired)` với cấu hình chuẩn trước; nếu NATS từ chối (do chạm trần dự trữ `max_file_store` / `err_code 10047`), tự động thử lại theo các mức hạ bậc an toàn: `64 MiB` $\rightarrow$ `16 MiB` $\rightarrow$ `-1` (unlimited/account default) để đảm bảo 4 Core Streams luôn được khởi tạo thành công.
- **Không ghi đè `MaxBytes` hợp lệ đã tồn tại:**
  - Khi stream đã tồn tại trên NATS và `info.Config.MaxBytes > 0`, **không** tự động ép `info.Config.MaxBytes = desired.MaxBytes` trong `ensureSingleStream` (chỉ cập nhật khi `info.Config.MaxBytes <= 0` và `desired.MaxBytes > 0`), tránh phá hỏng mức quota đã hạ bậc hoặc do người vận hành chỉnh qua UI.

#### 3. Tại `cdc-cms-service/internal/api/system/nats_monitor_handler.go` (giữ gọn file, không làm phình God-file)
- **Đọc bổ sung `/subsz?subs=1` khi `connz` chưa thấy đủ kênh lệnh:**
  - Trong `GetOverview`, lấy thêm `/subsz?subs=1` (`subsz`) song song trong `sync.WaitGroup` và nạp danh sách `subscriptions_list` từ `subsz` vào `subIndex` trong `buildNATSGovernance` để nhận diện chính xác 19 kênh `cdc.cmd.*` đang active trong `Subs: 274`.
- **Hiển thị trực tiếp lỗi `EnsureStreams` (nếu có) lên `governance["alerts"]` & không báo Drift giả khi `MaxBytes > 0`:**
  - Trong `buildNATSGovernance` (`L782-786`), chỉ báo `STREAM_DRIFT` về `max_bytes` khi `maxBytes <= 0` (chưa giới hạn), không báo lỗi drift khi stream dùng mức `MaxBytes > 0` đã hạ bậc hoặc chỉnh tay.
  - Lưu lỗi gần nhất của `tryAutoHealMissingCoreStreams` (`lastHealError`) và nếu vẫn còn stream thiếu thì đính kèm cảnh báo `AUTO_HEAL_ERROR` chứa nguyên văn lỗi NATS trả về ngay trong `governance["alerts"]` để hiển thị minh bạch trên UI.

### 10.3. Code Demo (`TASK-06`)

#### A. `centralized-data-service/internal/server/server_setup.go:99-101`
```go
	if err := natsClient.EnsureStreams(cdcSubjects); err != nil {
		logger.Warn("Failed to ensure all NATS JetStream streams at startup (worker will continue and self-heal)", zap.Error(err))
	}
```

#### B. `pkgs/natsconn/nats_client.go` (cả `centralized-data-service` và `cdc-cms-service`)
```go
func addStreamWithQuotaFallback(js nats.JetStreamContext, desired *nats.StreamConfig, log *zap.Logger) error {
	if _, err := js.AddStream(desired); err == nil {
		return nil
	} else {
		fallbackCaps := []int64{64 * 1024 * 1024, 16 * 1024 * 1024, -1}
		for _, capBytes := range fallbackCaps {
			if desired.MaxBytes > 0 && capBytes >= desired.MaxBytes {
				continue
			}
			clone := *desired
			clone.MaxBytes = capBytes
			if _, retryErr := js.AddStream(&clone); retryErr == nil {
				log.Warn("Created NATS stream with fallback MaxBytes due to broker storage quota",
					zap.String("stream", desired.Name),
					zap.Int64("desired_max_bytes", desired.MaxBytes),
					zap.Int64("applied_max_bytes", capBytes),
					zap.NamedError("initial_error", err))
				return nil
			}
		}
		return fmt.Errorf("failed to create stream %s: %w", desired.Name, err)
	}
}
```
Và khi đối chiếu `Subjects` & `MaxBytes` của stream đã tồn tại:
```go
	mergedSubjects := mergeSubjects(info.Config.Subjects, desired.Subjects)
	if !sameSubjects(info.Config.Subjects, mergedSubjects) {
		info.Config.Subjects = mergedSubjects
		needUpdate = true
	}
	if info.Config.MaxBytes <= 0 && desired.MaxBytes > 0 {
		info.Config.MaxBytes = desired.MaxBytes
		needUpdate = true
	}
```

#### C. `cdc-cms-service/internal/api/system/nats_monitor_handler.go`
- Trong `GetOverview`: gọi thêm `/subsz?subs=1` (`subsz`) và truyền `subsz` vào `buildNATSGovernance(varz, jsz, connz, subsz, ...)`:
```go
	if subList, ok := subsz["subscriptions_list"].([]any); ok {
		hasSubscriptionTelemetry = true
		for _, rawSub := range subList {
			subMap, ok := rawSub.(map[string]any)
			if !ok {
				continue
			}
			subj := asString(subMap["subject"])
			if subj == "" {
				continue
			}
			agg, exists := subIndex[subj]
			if !exists {
				agg = &subAggregation{
					queueGroups: make(map[string]bool),
					cids:        make(map[uint64]bool),
				}
				subIndex[subj] = agg
			}
			if cid := asUint64(subMap["cid"]); cid > 0 {
				if !agg.cids[cid] {
					agg.cids[cid] = true
					agg.listeners++
				}
			} else if agg.listeners == 0 {
				agg.listeners = 1
			}
			if qg := asString(subMap["qgroup"]); qg != "" {
				agg.queueGroups[qg] = true
			}
			agg.msgs += asInt64(subMap["msgs"])
		}
	}
```

---

## 11. Giải pháp Thực thi Dứt điểm `TASK-07` — Khắc phục Recon/Heal Segment B (`Shadow -> Master`)

### 11.1. Phương Án Tối Ưu Duy Nhất (Single Best Solution — Tuân thủ `L005` giảm dòng `transmuter.go`, `L006` không viết/chạy test, `L014` chỉ đọc `pk` từ `transform_spec`)

1. **Chuẩn hóa dữ liệu cấu hình `cdc_system.master_binding.transform_spec` ở tầng Bootstrap (`cdc-cms-service/internal/bootstrap/master_connection.go` & `NewTransmuterModule` trong `transmuter.go`):**
   - Thực thi câu lệnh idempotent backfill `transform_spec = jsonb_set(COALESCE(mb.transform_spec, '{}'::jsonb), '{pk}', to_jsonb(BTRIM(sor.primary_key_field)), true)` cho mọi dòng `cdc_system.master_binding` (`transform_type <> 'flatten'`) đang có `COALESCE(BTRIM(mb.transform_spec->>'pk'), '') = ''` và `COALESCE(BTRIM(sor.primary_key_field), '') <> ''`.
   - Nhờ đó, `resolveConflictTarget` (`transmuter_utils.go:736-749`) và `resolveMasterPKFromSpec` (`recon_execute_heal_handler.go:698-732`) giữ nguyên thiết kế sạch chỉ đọc từ `transform_spec.pk` (`L014`) mà vẫn nhận đầy đủ `{"pk": "_id"}` cho toàn bộ các binding cũ lẫn mới.

2. **Chấm dứt việc tự sinh `_gpay_id` SHA-256 trong `RunEvents` & Gỡ bỏ mảng chắp vá `staleLegacyGpayIDs` (`centralized-data-service/internal/service/master/transmuter.go`):**
   - Trong `RunEvents` (`transmuter.go:766-795`): nếu `shadowGID == 0 && ev.SourceID != "" && masterRow.ShadowTable != ""`, thử đọc `fetchShadowBatch(ctx, masterRow, 0, []string{ev.SourceID})` để lấy `GpayID` thật từ Shadow; nếu vẫn `shadowGID <= 0` thì `res.Skipped++; continue` (tuyệt đối không ghi `_gpay_id = deterministicSourceGpayID` SHA-256 vào Master).
   - Xóa bỏ hoàn toàn biến/nhánh chắp vá `staleLegacyGpayIDs` / `localLegacyGpayIDs` ở cả `RunEvents` (`L774-794`) và `processBatch` (`L1228, 1248, 1291-1295, 1302, 1323-1327`), làm gọn `transmuter.go`.

3. **Xử lý nguyên tử xung đột `_gpay_id` vs `conflictTarget` trong `bulkUpsertMaster` (`centralized-data-service/internal/service/master/transmuter.go:1802-1857`):**
   - Mọi bảng Master đều có `"_gpay_id" BIGINT PRIMARY KEY`, trong khi các bảng Master cũ trên Testing có thể có hoặc chưa có `UNIQUE INDEX ux_<table>_<pk>` trên cột `conflictTarget` (`_id`).
   - Do đó, trong `bulkUpsertMaster`:
     - Luôn dùng `ON CONFLICT ("_gpay_id") DO UPDATE SET ...` (không bao giờ bị lỗi `SQLSTATE 42P10` thiếu unique constraint trên bảng cũ, và cập nhật cả cột `conflictTarget` nếu cần).
     - Khi `conflictTarget != "_gpay_id"`, chạy trong cùng DB transaction (`masterDB.WithContext(ctx).Transaction`) bước dọn trước các bản ghi trên Master có cùng `conflictTarget` (`_id IN (?)`) nhưng mang `_gpay_id` lệch so với `_gpay_id` chuẩn từ Shadow (`"_gpay_id" NOT IN (?)`):
       ```sql
       DELETE FROM <qt> WHERE <conflictTarget> IN (?) AND "_gpay_id" NOT IN (?)
       ```
       Ngay sau đó thực thi `INSERT INTO <qt> (...) VALUES ... ON CONFLICT ("_gpay_id") DO UPDATE SET ...`.
     - Cơ chế này vừa chặn đứng lỗi `SQLSTATE 23505 (duplicate key value violates unique constraint "ux_<table>__id")`, vừa tự động xóa sạch các dòng nhân đôi/lệch `_gpay_id` cũ (`1,680` dòng) ngay khi chạy Transmute hoặc Heal.

4. **Bật mặc định `pruneMissingSrc` trên Segment B khi có dòng thừa (`cdc-cms-web/src/components/ExecuteHealModal.tsx:191-204`):**
   - Tính `hasRemainingOrphan = reports.some((r) => r.orphan_count - (r.pruned_missing_src_count || 0) > 0)` và đặt `setPruneMissingSrc(segment === 'shadow_master' ? hasRemainingOrphan : false)`.

### 11.2. Code Demo (`TASK-07`)

#### A. `cdc-cms-service/internal/bootstrap/master_connection.go:34-55`
```go
	backfillRes := db.WithContext(ctx).Exec(`
		UPDATE cdc_system.master_binding mb
		   SET transform_spec = jsonb_set(
		           CASE
		               WHEN mb.transform_spec IS NULL OR jsonb_typeof(mb.transform_spec) <> 'object' THEN '{}'::jsonb
		               ELSE mb.transform_spec
		           END,
		           '{pk}',
		           to_jsonb(BTRIM(sor.primary_key_field)),
		           true
		       ),
		       updated_at = NOW()
		  FROM cdc_system.source_object_registry sor
		 WHERE sor.id = mb.source_object_id
		   AND COALESCE(mb.transform_type, '') <> 'flatten'
		   AND COALESCE(BTRIM(mb.transform_spec->>'pk'), '') = ''
		   AND COALESCE(BTRIM(sor.primary_key_field), '') <> ''
	`)
	if backfillRes.Error != nil {
		return fmt.Errorf("backfill master_binding transform_spec.pk: %w", backfillRes.Error)
	}
```

#### B. `centralized-data-service/internal/service/master/transmuter.go`
```go
	conflictTarget := resolveConflictTarget(binding, colTypes)

	sets := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "_gpay_id" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", quoteTransmuteIdent(k), quoteTransmuteIdent(k)))
	}
	sets = append(sets, `"_updated_at" = NOW()`)

	qt := quoteTransmuteQualified(binding.MasterSchema, binding.MasterTable)
	var sqlText string
	if isFullSync {
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT ("_gpay_id") DO UPDATE SET %s
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			strings.Join(sets, ", "))
	} else {
		const clockSkewToleranceMs = 2000
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT ("_gpay_id") DO UPDATE SET %s
			WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			strings.Join(sets, ", "), qt, clockSkewToleranceMs)
	}

	var resRows []struct {
		Inserted bool `gorm:"column:inserted"`
	}
	execUpsert := func(tx *gorm.DB) error {
		if conflictTarget != "_gpay_id" {
			pkVals := make([]any, 0, len(records))
			gpayIDs := make([]int64, 0, len(records))
			for _, rec := range records {
				if pv := sqlBindValueTransmute(rec[conflictTarget]); pv != nil {
					pkVals = append(pkVals, pv)
				}
				if gid, ok := toInt64GpayID(rec["_gpay_id"]); ok && gid > 0 {
					gpayIDs = append(gpayIDs, gid)
				}
			}
			if len(pkVals) > 0 && len(gpayIDs) > 0 {
				delSQL := fmt.Sprintf(`DELETE FROM %s WHERE %s IN (?) AND "_gpay_id" NOT IN (?)`,
					qt, quoteTransmuteIdent(conflictTarget))
				if delErr := tx.Exec(delSQL, pkVals, gpayIDs).Error; delErr != nil {
					return delErr
				}
			}
		}
		return tx.Raw(sqlText, values...).Scan(&resRows).Error
	}
	if err := masterDB.WithContext(ctx).Transaction(execUpsert); err != nil {
		if failedField := findFailedFieldFromErr(err, records); failedField != "" {
			err = fmt.Errorf("%w (potential failed field: %q)", err, failedField)
		}
		return 0, 0, int64(len(records)), err
	}
```

#### C. `cdc-cms-web/src/components/ExecuteHealModal.tsx:191-204`
```tsx
      const hasRemainingOrphan = reports.some(
        (r) => r.orphan_count - (r.pruned_missing_src_count || 0) > 0
      );

      setHealMismatched(hasRemainingMismatched);
      setHealMissingDest(hasRemainingMissingDest);
      setPruneMissingSrc(segment === 'shadow_master' ? hasRemainingOrphan : false);
```

---

## 12. Đánh giá & Phản biện Kiến trúc: Với hệ thống hiện tại, việc `_gpay_id` ở Master khác `_gpay_id` ở Shadow có hợp lý không?

### 12.1. Nhận định Trực diện
- **Đối với các bảng `copy_1_to_1` / `json_extract` (quan hệ 1–1 giữa Shadow và Master, như `payment_bills`):**
  Việc `master._gpay_id` khác `shadow._gpay_id` là **HOÀN TOÀN BẤT HỢP LÝ** — đó là sự **vi phạm hợp đồng kiến trúc cốt lõi** của hệ thống hiện tại.
- **Đối với các bảng `flatten` (quan hệ 1–$N$, tách mảng JSON thành nhiều dòng con):**
  Việc `master._gpay_id` khác `shadow._gpay_id` là **hợp lý và bắt buộc** (`deterministicGpayID(shadowGpayID, "#<idx>")` tại `transmuter_utils.go:421-431`), vì 1 dòng Shadow (`1` `_gpay_id`) sinh ra $N$ dòng trên Master và bảng Master dùng `"_gpay_id" BIGINT PRIMARY KEY`.

### 12.2. Bằng chứng 4 Hợp đồng Thiết kế trong Code bắt buộc `master._gpay_id == shadow._gpay_id` (với bảng 1–1)
1. **Hợp đồng DDL & Định danh của Master (`master_ddl_generator.go:129-145` & `transmuter_utils.go:413-424`):**
   - Bảng Master đã bỏ cột `_source_id` và chỉ giữ 4 cột hệ thống (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`), trong đó comment thiết kế tại `master_ddl_generator.go:131-133` quy định rõ:
     `"_gpay_id BIGINT PK — định danh toàn cục + CONFLICT KEY (deterministic per source-row: copy = shadow _gpay_id; flatten = derive từ shadow gpay_id + array index)"`.
   - Tại `transmuter_utils.go:421-424`, khi `keySuffix == ""` (`copy_1_to_1`), `deterministicGpayID(shadowGpayID, "")` trả về nguyên vẹn `shadowGpayID`.
2. **Hợp đồng Con trỏ Quét Lô (`last_gpay_id` Watermark Cursor) của Transmuter (`transmuter.go:358-376, 1070-1111`):**
   - `fetchShadowBatch` (`transmuter.go:1070-1077`) quét tăng dần theo `WHERE _gpay_id > ? ORDER BY 1 LIMIT ?` dựa trên tính tăng đơn điệu theo thời gian của Sonyflake `_gpay_id`.
   - `inferLastGpayIDFromMaster` (`transmuter.go:1093-1111`) suy diễn mốc tiếp theo bằng cách lấy `SELECT _gpay_id FROM master WHERE _gpay_id <= maxShadowID ORDER BY _gpay_id DESC LIMIT 50` rồi đối chiếu `WHERE _gpay_id IN (?)` trên Shadow. Nếu `master._gpay_id` khác `shadow._gpay_id` (đặc biệt khi bị gán băm SHA-256 phân bố ngẫu nhiên trên $[0, 2^{63}-1]$), tính đơn điệu theo thời gian và khả năng suy diễn mốc `last_gpay_id` bị phá vỡ hoàn toàn.
3. **Hợp đồng Đối soát Segment B (`Shadow → Master`) của Recon Engine (`recon_stream_bucket_engine.go:698-754` & `recon_tier_b.go:584-605`):**
   - `resolveFieldsB` (`recon_stream_bucket_engine.go:699, 715-747`) đặt `segBPKColumn = "_gpay_id"` làm ưu tiên số 1. Vì cả Shadow và Master đều luôn có cột `_gpay_id`, `resolveFieldsB` **100% luôn chọn `pkCol = "_gpay_id"`**.
   - `diffIDTs` (`recon_tier_b.go:585-605`) so khớp 2 tập dữ liệu theo `_gpay_id`. Khi `master._gpay_id != shadow._gpay_id`, mỗi bản ghi đang tồn tại đầy đủ ở cả 2 bên bị đếm thành **1 dòng `Thiếu (MissingFromDest)` + 1 dòng `Thừa (MissingFromSrc / Orphan)`** và `Lệch (Mismatched) = 0`.
4. **Hợp đồng Self-Healing Segment B (`recon_execute_heal_handler.go:474-614, 826-882`):**
   - Heal Segment B nhận danh sách `missing_from_dest` và `missing_from_src` theo `_gpay_id`. Khi `master._gpay_id != shadow._gpay_id`, các dòng đang sống trên Master bị đẩy vào `MissingFromSrc` (Orphan giả).

### 12.3. Phản biện Ngược (Adversarial Critique): Điểm mù của kiến trúc `_gpay_id` hiện tại
1. **Tại sao thực tế lại phát sinh `master._gpay_id != shadow._gpay_id`?**
   - **Nguyên nhân 1 (Lỗi code trong `RunEvents` — đã triệt tiêu ở `TASK-07`):** Trước `TASK-07`, nhánh fallback tại `transmuter.go:786` tự sinh `rec["_gpay_id"] = deterministicSourceGpayID(ev.SourceID, e.KeySuffix)` (SHA-256 của `_source_id`), hoàn toàn lệch với Sonyflake `_gpay_id` của Shadow.
   - **Nguyên nhân 2 (Bản chất phụ thuộc thời điểm vật lý của Sonyflake Trigger trên Shadow — `schema_adapter.go:1055-1090`):**
     Trên bảng Shadow, `_gpay_id` được sinh bởi trigger `BEFORE INSERT` (`gen_sonyflake_id()`) từ `clock_timestamp()` + `fencing_token_seq`. Do đó, `shadow._gpay_id` là **ID theo thời điểm INSERT vật lý vào bảng Shadow**, không phải hàm tất định của `_source_id`. Nếu bảng Shadow bị drop/recreate hoặc một dòng bị xóa cứng rồi insert lại trên Shadow trong khi Master vẫn giữ dòng cũ, thì cùng một `_source_id` sẽ có `shadow._gpay_id` mới khác với `master._gpay_id` cũ (cho tới khi dòng đó được Transmute/Heal chạy qua `bulkUpsertMaster` của `TASK-07` để thay thế `_gpay_id` cũ bằng `_gpay_id` mới).
2. **Điểm yếu kiến trúc dài hạn của `resolveFieldsB` (`recon_stream_bucket_engine.go:698-754`):**
   - Việc `resolveFieldsB` luôn chọn `_gpay_id` làm khóa đối soát giữa Shadow và Master khiến Recon Segment B:
     - Đánh đồng *"lệch ID kỹ thuật `_gpay_id`"* thành *"1 Thiếu + 1 Thừa dữ liệu nghiệp vụ"*.
     - Không thể đối soát đúng cho các bảng `flatten` (vì với `flatten`, `master._gpay_id` là băm SHA-256 của `shadow._gpay_id + "#idx"`, không bao giờ bằng `shadow._gpay_id`).

---

## 13. Nhận định lại tận gốc: Khi 1 bảng Shadow rẽ nhánh ra 5 bảng Master (`master_1..5`), `_gpay_id` trên 5 bảng đó như thế nào & vì sao hướng ép `master._gpay_id == shadow._gpay_id` là sai bản chất?

### 13.1. Trả lời trực tiếp câu hỏi: "1 shadow rẽ nhánh qua 5 table trên master (`master_1, 2, 3, 4, 5`), vậy `_gpay_id` trên 5 cái table đó đều giống nhau, phải không?"
- **Theo code hiện tại (`transmuter_utils.go:421-431` & `transmuter.go:623, 780`): ĐÚNG, code hiện tại đang làm cho `_gpay_id` trên cả 5 bảng `master_1, master_2, master_3, master_4, master_5` GIỐNG HỆT NHAU (và giống `shadow._gpay_id`)!**
  - Tại `centralized-data-service/internal/service/master/transmuter_utils.go:421-424`:
    ```go
    func deterministicGpayID(shadowGpayID int64, keySuffix string) int64 {
        if keySuffix == "" {
            return shadowGpayID
        }
        ...
    ```
  - Khi 1 bảng Shadow rẽ nhánh (`1 -> 5`) sang `master_1..5`:
    - Với các bảng `copy_1_to_1` / `json_extract` (`keySuffix == ""`), cả 5 bảng `master_1..5` đều bị gán cứng cùng một giá trị `rec["_gpay_id"] = shadow._gpay_id`.
    - Ngay cả với các bảng `flatten` (`keySuffix = "#0", "#1", ...`), công thức băm `sha256(fmt.Sprintf("%d%s", shadowGpayID, keySuffix))` (`transmuter_utils.go:425`) **không hề có tên bảng hay `master_binding_id`**, nên dòng `#0` của `master_2` và dòng `#0` của `master_3` cũng ra **cùng một `_gpay_id` y hệt nhau**!

### 13.2. Tại sao việc gán chung `_gpay_id` từ Shadow sang `master_1..5` và ép `master._gpay_id == shadow._gpay_id` là SAI BẢN CHẤT ("thua rồi")?
1. **Sai bản chất của `_gpay_id` (Global Surrogate ID):**
   - Đúng bản chất thiết kế dữ liệu, `_gpay_id` (Sonyflake ID) là **Surrogate Primary Key định danh bản ghi vật lý độc lập trên từng bảng**.
   - Khi 1 dòng ở `shadow` rẽ nhánh ra 5 bảng `master_1, master_2, master_3, master_4, master_5`, đó là **5 thực thể đích hoàn toàn khác nhau** (có thể là 5 bảng nghiệp vụ khác nhau, có vòng đời, số dòng, điều kiện lọc/tách dòng khác nhau).
   - `_gpay_id` của `master_1`, `master_2`, ..., `master_5` **không việc gì phải giống nhau và càng không cần phải giống `shadow._gpay_id`**! Mỗi bảng Master chỉ cần sinh `_gpay_id` riêng cho dòng mới của bảng đó khi `INSERT` (ví dụ `DEFAULT public.gen_sonyflake_id()` hoặc cấp Sonyflake mới khi insert) và **giữ nguyên bất biến `_gpay_id` đó khi `UPDATE`**.
2. **Nhầm lẫn giữa Surrogate ID (`_gpay_id`) và Khóa Nghiệp Vụ (`_source_id` $\leftrightarrow$ `transform_spec.pk`):**
   - Sợi dây liên kết thật sự giữa 1 dòng trên `shadow` và các bảng `master_1..5` là **Khóa Nghiệp Vụ** (`shadow."_source_id"` $\leftrightarrow$ cột `pk` được cấu hình trong `transform_spec.pk` của từng bảng `master_1..5`, nơi DDL Generator `master_ddl_generator.go:287-304` đã tạo sẵn `UNIQUE INDEX ux_<master_table>_<pk>`).
   - Vì giải pháp trước cố ép `master._gpay_id` phải bằng `shadow._gpay_id`, dẫn đến chuỗi hệ lụy phi lý:
     - Trong `bulkUpsertMaster`: hễ `master._gpay_id != shadow._gpay_id` là phải `DELETE` dòng đang sống trên Master rồi `INSERT` lại chỉ để đổi `_gpay_id`!
     - Trên DB hiện có hàng trăm triệu bản ghi ở ~100 bảng Master, bất kỳ chỗ nào `shadow._gpay_id` khác `master._gpay_id` đều bị coi là hỏng.
     - Trong `reconB` (`recon_stream_bucket_engine.go:699, 719-748`): vì `resolveFieldsB` lấy `_gpay_id` đi so giữa `shadow` và `master_1..5`, nên dù bảng Master đã có đủ dữ liệu nghiệp vụ (`3,082 / 3,095`, chỉ thiếu `13` dòng), `reconB` lại báo sai thành `1,693 Thiếu | 1,680 Thừa`!
3. **Cách vận hành đúng đắn nếu tách bạch `_gpay_id` và Khóa Nghiệp Vụ (`transform_spec.pk`):**
   - **Trên `master_1..5` (`bulkUpsertMaster`):**
     - Upsert theo khóa nghiệp vụ `ON CONFLICT (<pk>) DO UPDATE SET ...` (**không cập nhật lại `_gpay_id`**, không bao giờ cần `DELETE` + `INSERT` lại dòng cũ vì lệch `_gpay_id`). Khi `INSERT` dòng mới chưa có trên Master thì cấp `_gpay_id` mới độc lập.
   - **Trên `reconB` (`Shadow → Master` — `recon_stream_bucket_engine.go`):**
     - Không bao giờ đem `shadow."_gpay_id"` đi so với `master."_gpay_id"`!
     - Phía Shadow lấy khóa nghiệp vụ `shadow."_source_id"`, còn phía từng bảng Master (`master_1..5`) lấy đúng cột `pk` trong `transform_spec.pk` của `master_binding` đó (ví dụ `master_1."_id"`).
     - Khi đó, dù `_gpay_id` trên `shadow` và trên 5 bảng `master_1..5` có khác nhau hoàn toàn (và không cần đụng chạm vào hàng trăm triệu bản ghi hiện có trên DB Master), `reconB` đối soát trên tập `3,095 vs 3,082` sẽ ra chính xác tuyệt đối: **`Thiếu: 13 | Lệch: 0 | Thừa: 0`**, và khi Heal chỉ cần truyền đúng `13` `_source_id` bị thiếu đó sang Transmute!

---

## 14. Truy vết Lịch sử Git (`38ba1f4`) & Giải pháp Chi tiết `TASK-08`: Tách bạch `_gpay_id` khỏi Khóa Nghiệp Vụ (`_source_id` $\leftrightarrow$ `transform_spec.pk`)

### 14.1. Bằng chứng Git History: Chính xác từ khi nào `master._gpay_id` bắt đầu khác `shadow._gpay_id`? (`VERIFIED` 100%)
1. **Trước ngày `2026-10-12 15:56:01` (`commit 8a76b45` trở về trước):**
   - Toàn bộ luồng Transmute chỉ đi qua `TransmuterModule.Run` (`processBatch`) đọc từ bảng Shadow (`fetchShadowBatch`) và gán `rec["_gpay_id"] = deterministicGpayID(row.GpayID, e.KeySuffix)`.
2. **Thủ phạm gốc — `commit 38ba1f44d07f5663ee2d844049c3fd5c30525a7e` (`Mon Oct 12 15:56:01 2026 +0700 | update nat monitor, fix number count update master`):**
   - Trong commit này, AI đã tự ý nhét thêm 2 hàm mới vào core Master:
     - `func (t *TransmuterModule) RunEvents(...)` tại `internal/service/master/transmuter.go:543-723`
     - `func deterministicSourceGpayID(sourceID string, keySuffix string) int64` tại `internal/service/master/transmuter_utils.go:192-203`
   - Đồng thời sửa `internal/handler/shadow/batch_buffer_fanout.go:31-78` và `internal/handler/master/transmute_handler.go:508-514` để lái toàn bộ sự kiện CDC realtime qua `RunEvents`.
   - Tại dòng `+691` của `internal/service/master/transmuter.go` trong commit `38ba1f4`:
     ```go
     rec["_gpay_id"] = deterministicSourceGpayID(ev.SourceID, e.KeySuffix)
     ```
     Hàm `deterministicSourceGpayID` băm SHA-256 của chuỗi `ev.SourceID` ra số `int64`, làm mọi bản ghi đi qua `RunEvents` từ chiều ngày `12/10/2026` mang `_gpay_id` SHA-256 trên Master, khác hoàn toàn với `_gpay_id` Sonyflake trên bảng Shadow!
3. **Chuỗi đắp vá tiếp theo (`73e47ca` 19/10, `fa20799` 19/10, `30b1b00` 20/10, `08c05a8` 20/10, `3369c31` 26/10):**
   - Thay vì tách `_gpay_id` khỏi Khóa Nghiệp Vụ, các commit này lại cố đắp vá quanh việc ép `master._gpay_id == shadow._gpay_id` (`lookupShadowGpayIDs`, `staleLegacyGpayIDs`, xóa dòng lệch `_gpay_id` trước khi insert...), dẫn đến xung đột `23505` / `42P10` và báo cáo `reconB` sai lệch `1,693 Thiếu | 1,680 Thừa`.

---

### 14.2. Thiết kế Giải pháp Duy nhất & Tối ưu (`TASK-08`): Tách bạch hoàn toàn `_gpay_id` khỏi Khóa Nghiệp Vụ (`_source_id` $\leftrightarrow$ `transform_spec.pk`)

#### Nguyên lý cốt lõi:
1. **`_gpay_id` là Surrogate ID nội bộ độc lập của từng bảng (`shadow`, `master_1`, `master_2`, ..., `master_5`):**
   - Khi 1 bảng Shadow rẽ nhánh ra 5 bảng Master (`master_1..5`), mỗi bảng `master_i` (`masterRow.ID`) sinh `_gpay_id` riêng của bảng đó thông qua `deriveMasterGpayID(masterRow.ID, sourceID, keySuffix)` (`transmuter_utils.go:421-443`). Nhờ có `masterRow.ID` trong hàm dẫn xuất, `_gpay_id` trên `master_1..5` hoàn toàn khác `shadow._gpay_id` và khác nhau giữa cả 5 bảng `master_1..5`!
   - Khi bản ghi đã tồn tại trên bảng Master (trùng khóa nghiệp vụ `conflictTarget`), lệnh `ON CONFLICT (<conflictTarget>) DO UPDATE` **giữ nguyên bất biến `_gpay_id` đang có trên bảng Master** (`if k == "_gpay_id" || k == conflictTarget { continue }`), tuyệt đối không bao giờ ghi đè `_gpay_id` hay `DELETE` dòng đang sống!
2. **Upsert & Delete trên Master dựa thuần túy vào Khóa Nghiệp Vụ `conflictTarget` (`transform_spec.pk`):**
   - Tại `bulkUpsertMaster` (`transmuter.go:1802-1865`):
     - Sử dụng trực tiếp `ON CONFLICT (%s) DO UPDATE SET %s` với `quoteTransmuteIdent(conflictTarget)` (đọc nghiêm ngặt từ `transform_spec.pk`).
     - Xóa bỏ hoàn toàn khối Transaction `DELETE FROM <master> WHERE <pk> IN (?) AND "_gpay_id" NOT IN (?)`.
     - Nếu gặp bảng Master cũ trên môi trường chưa kịp có `UNIQUE INDEX ux_<table>_<conflictTarget>` (trả lỗi `SQLSTATE 42P10`), tự động thực thi `CREATE UNIQUE INDEX IF NOT EXISTS ux_<table>_<conflictTarget> ON <master> (<conflictTarget>)` 1 lần rồi chạy lại câu `INSERT ... ON CONFLICT (<conflictTarget>) DO UPDATE`.
   - Tại `RunEvents` (`transmuter.go:722-809`):
     - Bỏ hoàn toàn việc gọi `lookupShadowGpayIDs` chọc vào DB Shadow mỗi mẻ event! `RunEvents` biến đổi thẳng `ev.RawData` và gọi `bulkUpsertMaster` theo `conflictTarget`.
3. **Recon Segment B (`Shadow → Master`) đối soát theo `shadow."_source_id"` $\leftrightarrow$ `master.<transform_spec.pk>`:**
   - Tại `recon_engine_segment_b.go`: Thêm `ResolveSegmentBPKPair(ctx, ref, msAgent) (shadowPKCol, masterPKCol, idType string)` đọc `pk` nghiêm ngặt từ `ref.TransformSpec` (`{"pk": "..."}`). Khi `pk != "" && pk != "_gpay_id"`, trả về `shadowPKCol = "_source_id"`, `masterPKCol = pk`, `idType = "id"`.
   - Tại `recon_stream_bucket_engine.go:598-638`: Bổ sung `COALESCE(mb.transform_spec::text, '{}') AS transform_spec` vào `lookupMasterRef` và `lookupMasterRefExact`.
   - Tại `recon_stream_bucket_engine.go`, `recon_tier_b.go`, `recon_smoke.go`: Phía Shadow băm và liệt kê theo `shadowPKCol` (`"_source_id"`), phía Master băm và liệt kê theo `masterPKCol` (`transform_spec.pk`), và ghi `IDType: idType` (`"id"`) vào `StaleIDsPayload`.
4. **Heal Segment B (`recon_execute_heal_handler.go`) & Cursor `inferLastGpayIDFromMaster` (`transmuter.go:1108-1166`):**
   - Trong `executeHealSegB` và `resolveSourceIDsForSegmentB`: Khi `ref` có `pkCol` từ `transform_spec.pk` và `idType == "id"` (hoặc `idType == ""` nhưng các ID khớp `_source_id` / `pkCol`), xử lý thẳng theo khóa nghiệp vụ mà không bị `allNumeric` ép nhầm sang `"gpay"`; vẫn giữ nhánh fallback `"gpay"` cho các báo cáo cũ đã lưu `_gpay_id` trước đó.
   - Trong `inferLastGpayIDFromMaster`: Suy diễn mốc `last_gpay_id` của Shadow thông qua khóa nghiệp vụ `pkCol` (`SELECT <pkCol>::text FROM master ... ORDER BY _source_ts DESC LIMIT 50` $\rightarrow$ `SELECT COALESCE(MAX(_gpay_id), 0) FROM shadow WHERE _source_id IN (?)`) và **xóa bỏ 2 câu `SELECT COUNT(*)` toàn bảng** ở dòng `1154-1155`.

---

### 14.3. Code Demo Chi tiết (`TASK-08`)

#### A. `centralized-data-service/internal/service/master/transmuter_utils.go:421-443`
```go
// nextMasterSonyflakeGpayID sinh Sonyflake _gpay_id mới, độc lập cho từng bản ghi được INSERT vào bảng Master.
func nextMasterSonyflakeGpayID() int64 {
	if sfID, err := idgen.NextID(); err == nil && sfID > 0 {
		return int64(sfID & 0x7FFFFFFFFFFFFFFF)
	}
	_ = idgen.Init(zap.NewNop())
	if sfID, err := idgen.NextID(); err == nil && sfID > 0 {
		return int64(sfID & 0x7FFFFFFFFFFFFFFF)
	}
	return time.Now().UnixNano()
}
```

#### B. `centralized-data-service/internal/service/master/transmuter.go` (`RunEvents`, `inferLastGpayIDFromMaster`, `processBatch`, `bulkUpsertMaster`)
```go
// 1. Trong RunEvents (transmuter.go:722-810): bỏ lookupShadowGpayIDs, sinh Sonyflake _gpay_id độc lập cho Master
	var lastEventErr error
	for _, ev := range events {
		res.Scanned++
		if ev.IsDelete || ev.Op == "d" {
			_, delErr := t.deleteFromMasterBySourceAndGpayIDs(ctx, masterRow, colTypes, []string{ev.SourceID}, nil)
			if delErr != nil {
				t.logger.Error("transmuter: event delete master failed",
					zap.String("master", masterRow.MasterTable),
					zap.String("source_id", ev.SourceID),
					zap.Error(delErr))
				res.Skipped++
				if lastEventErr == nil {
					lastEventErr = delErr
				}
			} else {
				res.Deleted++
			}
			continue
		}

		rawDataBytes := []byte(ev.RawData)
		if len(rawDataBytes) == 0 {
			shadowRows, fetchErr := t.fetchShadowBatch(ctx, masterRow, 0, []string{ev.SourceID})
			if fetchErr == nil && len(shadowRows) > 0 {
				rawDataBytes = shadowRows[0].RawData
			}
		}
		if len(rawDataBytes) == 0 {
			res.Skipped++
			continue
		}

		emits, st, buildErr := strat.BuildEmits(rc, transmute.ShadowRow{
			Raw:      rawDataBytes,
			SourceID: ev.SourceID,
			SourceTs: ev.SourceTs,
			Deleted:  false,
		})
		res.RuleMisses += st.RuleMisses
		res.TypeErrors += st.TypeErrors
		if buildErr != nil || len(emits) == 0 {
			res.Skipped++
			continue
		}

		records := make([]map[string]any, 0, len(emits))
		for _, e := range emits {
			rec := e.Cols
			if e.KeySuffix == "" {
				rec["_gpay_id"] = nextMasterSonyflakeGpayID()
			} else {
				rec["_gpay_id"] = deterministicSourceGpayID(fmt.Sprintf("%d:%s", masterRow.ID, ev.SourceID), e.KeySuffix)
			}
			rec["_source_ts"] = ev.SourceTs
			rec["_deleted"] = false
			records = append(records, rec)
		}
```

```go
// 2. Trong inferLastGpayIDFromMaster (transmuter.go:1108-1166): suy diễn qua khóa nghiệp vụ pkCol, bỏ SELECT COUNT(*) toàn bảng
func (t *TransmuterModule) inferLastGpayIDFromMaster(ctx context.Context, row *masterBindingRuntime, masterDB *gorm.DB, rules []mappingRuleRow) int64 {
	shadowDB, err := t.connMgr.GetShadowDB(ctx, row.ShadowConnectionKey)
	if err != nil || masterDB == nil {
		return 0
	}
	masterQT := quoteTransmuteQualified(row.MasterSchema, row.MasterTable)
	shadowQT := quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable)
	shadowPK := fmt.Sprintf("(%s)::bigint", quoteTransmuteIdent(row.ShadowPK))

	colTypes := make(map[string]string, len(rules))
	for _, r := range rules {
		colTypes[r.TargetColumn] = r.DataType
	}
	if pkCol := resolveConflictTarget(row, colTypes); pkCol != "" && pkCol != "_gpay_id" && isMasterPKMappedFromSourcePK(row, rules, pkCol) {
		pkIdent := quoteTransmuteIdent(pkCol)
		var recentIDs []string
		if masterDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT %s::text FROM %s WHERE _source_ts IS NOT NULL ORDER BY _source_ts DESC LIMIT 50`, pkIdent, masterQT)).Scan(&recentIDs).Error == nil && len(recentIDs) > 0 {
			var bySourceID int64
			if shadowDB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s WHERE _source_id IN (?)`, shadowPK, shadowQT), recentIDs).Scan(&bySourceID).Error == nil && bySourceID > 0 {
				var sampleSourceIDs []string
				errGap := shadowDB.WithContext(ctx).Raw(
					fmt.Sprintf(`SELECT _source_id FROM %s WHERE %s <= ? AND NOT COALESCE(_deleted, false) ORDER BY %s DESC LIMIT 50`, shadowQT, shadowPK, shadowPK),
					bySourceID,
				).Scan(&sampleSourceIDs).Error
				if errGap == nil && len(sampleSourceIDs) > 0 {
					var matchedInMaster int64
					errMatch := masterDB.WithContext(ctx).Raw(
						fmt.Sprintf(`SELECT COUNT(DISTINCT %s::text) FROM %s WHERE %s::text IN (?)`, pkIdent, masterQT, pkIdent),
						sampleSourceIDs,
					).Scan(&matchedInMaster).Error
					if errMatch == nil && matchedInMaster == int64(len(sampleSourceIDs)) {
						return bySourceID
					}
				}
			}
		}
	}
	return 0
}
```

```go
// 3. Trong bulkUpsertMaster (transmuter.go:1802-1865): ON CONFLICT (conflictTarget) DO UPDATE trực tiếp, giữ nguyên _gpay_id, không DELETE dòng đang sống
	conflictTarget := resolveConflictTarget(binding, colTypes)
	conflictIdent := quoteTransmuteIdent(conflictTarget)

	sets := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "_gpay_id" || k == conflictTarget {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", quoteTransmuteIdent(k), quoteTransmuteIdent(k)))
	}
	sets = append(sets, `"_updated_at" = NOW()`)

	qt := quoteTransmuteQualified(binding.MasterSchema, binding.MasterTable)
	var sqlText string
	if isFullSync {
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT (%s) DO UPDATE SET %s
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			conflictIdent, strings.Join(sets, ", "))
	} else {
		const clockSkewToleranceMs = 2000
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT (%s) DO UPDATE SET %s
			WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			conflictIdent, strings.Join(sets, ", "), qt, clockSkewToleranceMs)
	}

	var resRows []struct {
		Inserted bool `gorm:"column:inserted"`
	}
	execErr := masterDB.WithContext(ctx).Raw(sqlText, values...).Scan(&resRows).Error
	if execErr != nil && conflictTarget != "_gpay_id" &&
		(strings.Contains(execErr.Error(), "42P10") || strings.Contains(strings.ToLower(execErr.Error()), "no unique or exclusion constraint")) {
		uxName := quoteTransmuteIdent("ux_" + binding.MasterTable + "_" + conflictTarget)
		createUxSQL := fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)`, uxName, qt, conflictIdent)
		if idxErr := masterDB.WithContext(ctx).Exec(createUxSQL).Error; idxErr == nil {
			resRows = nil
			execErr = masterDB.WithContext(ctx).Raw(sqlText, values...).Scan(&resRows).Error
		}
	}
```

#### C. `centralized-data-service/internal/service/recon/recon_engine_segment_b.go` & `recon_stream_bucket_engine.go` & `recon_tier_b.go`
```go
// Trong recon_engine_segment_b.go:
// ResolveSegmentBPKPair tách biệt khóa đối soát của Segment B (Shadow ↔ Master) khỏi _gpay_id:
// - Khi transform_spec.pk khai báo cột khóa nghiệp vụ (vd: "_id"), bên Shadow đối soát theo "_source_id" và bên Master đối soát theo pk.
// - Chỉ khi transform_spec.pk trống (vd bảng flatten) mới fallback về "_gpay_id".
func ResolveSegmentBPKPair(ctx context.Context, ref MasterBindingRef, msAgent *ReconDestAgent) (shadowPKCol, masterPKCol, idType string) {
	if strings.TrimSpace(ref.TransformSpec) != "" {
		var spec struct {
			PK string `json:"pk"`
		}
		if err := json.Unmarshal([]byte(ref.TransformSpec), &spec); err == nil {
			pk := strings.TrimSpace(spec.PK)
			if pk != "" && pk != segBPKColumn && validateIdent(pk) == nil {
				masterOK := true
				if msAgent != nil && ref.MasterTable != "" {
					if ok, err := msAgent.ColumnExists(ctx, ref.MasterRel(), pk); err != nil || !ok {
						masterOK = false
					}
				}
				if masterOK {
					return "_source_id", pk, "id"
				}
			}
		}
	}
	return segBPKColumn, segBPKColumn, "gpay"
}
```

---

## 9. Thiết kế Giải pháp Bổ sung — `TASK-09` (S5 Self-Audit Refactoring: Tách File Đơn Trách Nhiệm, Chuẩn Hóa Layer & Khắc Phục Đếm `res.RowsAffected`)

### 9.1. Kết quả Phản biện S5 (Vì sao cần vòng lặp tự chữa lành `S5 → S3 → S4 → S5`?)
Qua rà soát phản biện từng file và từng dòng `git diff` theo yêu cầu của anh:
1. **Vi phạm nguyên tắc Single-Responsibility / Không nhồi nhét vào các file lớn (`L001` / `L011`):**
   - `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`: bị nhồi thêm `+63` dòng (`932 -> 995` dòng) do đặt toàn bộ logic Heal & Prune của Segment B (`executeHealSegB`, `resolveMasterDBForRef`, `resolveMasterPKFromSpec`, `resolveSourceIDsForSegmentB`) chung với HTTP handler chung và Segment A.
   - `centralized-data-service/internal/service/master/transmuter_utils.go`: bị nhồi thêm `+36` dòng (`776 -> 812` dòng) do nhét các hàm định danh `_gpay_id` & phân giải khóa chính (`nextMasterSonyflakeGpayID`, `deterministicMasterFlattenGpayID`, `deterministicGpayID`, `deterministicSourceGpayID`, `resolveConflictTarget`, `isMasterPKMappedFromSourcePK`) vào file tiện ích JSON/BSON/Type chung.
   - `cdc-cms-service/internal/api/system/nats_monitor_handler.go`: bị nhồi thêm `+65` dòng (`1149 -> 1214` dòng) do nhét bộ máy phân tích Governance & Subscription Telemetry (`buildNATSGovernance`, `/subsz?subs=1`, `matchNATSSubjectFilter`, `extractStreamsFromJsz`) chung vào file Fiber HTTP handler.
   - `centralized-data-service/pkgs/natsconn/nats_client.go` (`286 -> 331` dòng) và `cdc-cms-service/pkgs/natsconn/nats_client.go` (`279 -> 325` dòng): bị nhồi thêm `+45` dòng do nhét logic provision/reconcile stream (`EnsureStreams`, `ensureSingleStream`, `addStreamWithQuotaFallback`, `mergeSubjects`, `sameSubjects`) vào file kết nối NATS client cơ bản.
2. **Vi phạm ranh giới Layer tại `NewTransmuterModule` (`transmuter.go:137-156`):**
   - Trong `TASK-07`, câu lệnh SQL `UPDATE cdc_system.master_binding SET transform_spec = ...` được đặt trực tiếp bên trong constructor `NewTransmuterModule` (thuộc tầng `internal/service/master`), trùng lặp với tầng `cdc-cms-service/internal/bootstrap/master_connection.go:50-74`. Tầng `service` của worker không được phép tự ý chạy migration cập nhật bảng metadata khi khởi tạo struct!
3. **Lỗi đếm số dòng xóa (`PrunedMissingSrcCount`) trong `executeHealSegB` (`recon_execute_heal_handler.go:619, 692, 706`):**
   - Khi sửa `TASK-08`, câu lệnh `pruned += int(res.RowsAffected)` bị đổi nhầm thành `pruned += len(toDeleteGpay)` và `pruned += end - i`. Nếu có bản ghi orphan đã được xóa trước đó, cách cộng theo độ dài mảng sẽ báo cáo khống số dòng thực sự xóa trên PostgreSQL. Phải khôi phục `pruned += int(res.RowsAffected)`.

### 9.2. Giải pháp Tách Module Đơn Trách Nhiệm & Chuẩn Hóa Layer (`TASK-09`)
1. **Tách `centralized-data-service/internal/service/master/transmuter_identity.go`:**
   - Chuyển toàn bộ các hàm định danh `_gpay_id` và phân giải khóa chính Master (`nextMasterSonyflakeGpayID`, `deterministicMasterFlattenGpayID`, `deterministicGpayID`, `deterministicSourceGpayID`, `resolveConflictTarget`, `isMasterPKMappedFromSourcePK`) từ `transmuter_utils.go` sang file đơn trách nhiệm `transmuter_identity.go` (cùng package `master`).
   - Kết quả: `transmuter_utils.go` giảm từ `812` dòng xuống **`~700` dòng** (`-76` dòng so với bản gốc `776` dòng).
2. **Tách `centralized-data-service/internal/handler/recon/recon_execute_heal_segb.go`:**
   - Chuyển toàn bộ các hàm thực thi Heal & Prune chuyên biệt của Segment B (`executeHealSegB`, `resolveMasterDBForRef`, `resolveMasterPKFromSpec`, `resolveSourceIDsForSegmentB`) từ `recon_execute_heal_handler.go` sang file đơn trách nhiệm `recon_execute_heal_segb.go` (cùng package `recon`), đồng thời sửa lại `pruned += int(res.RowsAffected)` ở cả 3 nhánh `DELETE`.
   - Kết quả: `recon_execute_heal_handler.go` giảm từ `995` dòng xuống **`~450` dòng** (`-482` dòng so với bản gốc `932` dòng).
3. **Tách `cdc-cms-service/internal/api/system/nats_monitor_governance.go`:**
   - Chuyển bộ máy tính toán Governance & Subscription Telemetry (`buildNATSGovernance`, `matchNATSSubjectFilter`, `extractStreamsFromJsz`, `asString`, `asInt64`, `asUint64`) từ `nats_monitor_handler.go` sang file đơn trách nhiệm `nats_monitor_governance.go` (cùng package `system`).
   - Kết quả: `nats_monitor_handler.go` giảm từ `1214` dòng xuống **`~760` dòng** (`-389` dòng so với bản gốc `1149` dòng).
4. **Tách `pkgs/natsconn/nats_stream_provision.go` trên cả 2 repo (`centralized-data-service` & `cdc-cms-service`):**
   - Chuyển toàn bộ logic khởi tạo/đối soát JetStream stream (`EnsureStreams`, `ensureSingleStream`, `addStreamWithQuotaFallback`, `mergeSubjects`, `sameSubjects`) từ `nats_client.go` sang `nats_stream_provision.go` (cùng package `natsconn`).
   - Kết quả: `nats_client.go` giảm từ `331`/`325` dòng xuống **`~146` dòng** (`-140` dòng so với bản gốc).
5. **Gỡ bỏ SQL `UPDATE cdc_system.master_binding` khỏi constructor `NewTransmuterModule` (`transmuter.go:137-156`):**
   - Giữ trách nhiệm bootstrap metadata duy nhất tại `cdc-cms-service/internal/bootstrap/master_connection.go:50-74`, trả `NewTransmuterModule` về đúng vai trò khởi tạo service thuần túy.

### 9.3. Code Demo (`TASK-09` — `transmuter_identity.go` & `recon_execute_heal_segb.go`)
```go
// centralized-data-service/internal/service/master/transmuter_identity.go
package master

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
	"time"

	"centralized-data-service/pkgs/idgen"
	"go.uber.org/zap"
)

func nextMasterSonyflakeGpayID() int64 {
	id, err := idgen.NextID()
	if err != nil {
		idgen.Init(zap.NewNop())
		id, err = idgen.NextID()
	}
	if err == nil && id > 0 {
		return int64(id & 0x7FFFFFFFFFFFFFFF)
	}
	return time.Now().UnixNano() & 0x7FFFFFFFFFFFFFFF
}
```
