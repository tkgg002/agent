---
tier: T2
workspace: fix-debezium-config-masked-password-2026-11-10
suffix: debezium_config_masked_password
---

# 09. Technical Solution & Code Demo — Xử lý lỗi đẩy Password / URI bị Mask (`***` / `*******` / `__KEEP__`) khi Update Config trên `/sources`

## 1. Hiện trạng Runtime trên Kafka Connect (`10.200.186.203:8083`) & Giải thích Triệu chứng

Qua kiểm tra trực tiếp trên Kafka Connect cluster (`http://10.200.186.203:8083/connectors?expand=info`) và CMS API (`http://localhost:8083/api/v1/sources`) ngay sau thao tác cập nhật connector `testcoretrans` lúc `2026-11-10T13:41:17Z`:

1. **Trạng thái thật của `testcoretrans` và `devcoretrans` trên Kafka Connect (`10.200.186.203:8083`):**
   - Cả 12 connector trên Kafka Connect hiện vẫn giữ nguyên secret thật (`testcoretrans` và `devcoretrans` có `database.password` dài 32 ký tự, `has_star=False`, `has_keep=False`; các connector MongoDB giữ nguyên `mongodb.connection.string` dài 155–169 ký tự không chứa `*`).
   - Lý do `testcoretrans` không bị mất password ở lần bấm Update vừa rồi: Trên form Edit PostgreSQL (`SourceConnectors.tsx:275`), khi anh để trống ô Password (`Leave blank to keep current`), frontend gửi `"database.password": "__KEEP__"` và hàm `UpdateSystemConnectorConfigHandler.Handle` (`debezium_connector.go:431-435`) bỏ qua key có giá trị `"__KEEP__"`, giữ lại `current["database.password"]`.
2. **Tại sao sau khi bấm Update anh lại thấy `"***"` / `*******` / `••••••••••••`:**
   - **Trên HTTP Response của `PATCH /api/v1/system/connectors/:name/config` và `GET /api/v1/system/connectors`:** Hàm `KafkaConnectClient.UpdateConfig` (`kafka_connect.go:177`) và `ListConnectors` (`list_connectors.go:98`) đều gọi `infrahttp.FilterSafeConfig` (`kafka_connect.go:243-264`), thay thế mọi key chứa `password`/`secret`/`token`/`credentials`/`ssl.key` thành `"***"` và mask `mongodb.connection.string` / `fs.uris` thành `***:***`.
   - **Trong DB `cdc_system.sources.raw_config_sanitized` (`system_connectors_handler.go:224`):** Lưu JSON đã qua `FilterSafeConfig` nên `"database.password"` là `"***"`.
   - **Trên giao diện `/sources` (`SourceConnectors.tsx:1024-1026`) và Kafka UI:** Bảng chi tiết config render `SENSITIVE_CONFIG_KEYS` thành `••••••••••••` (hoặc `******` / `*******` trên Kafka UI).

---

## 2. Năm Lỗi Thực Tế (`F1..F5`) Khiến Password / URI Bị Mask (`***` / `*******` / `__KEEP__`) Bị Đẩy Lên Debezium Hoặc PostgreSQL

Mặc dù trường hợp Edit PostgreSQL để trống ô Password gửi `"__KEEP__"`, hệ thống hiện có **5 lỗ hổng thực tế (`F1..F5`)** làm đẩy giá trị mask (`***`, `*******`, `***:***`, `__KEEP__`) thẳng lên Debezium, đẩy vào PostgreSQL nguồn, hoặc ghi đè hỏng DB `cdc_system`:

### `F1` — `UpdateSystemConnectorConfigHandler.Handle` chỉ chặn đúng chuỗi `"__KEEP__"`, KHÔNG chặn giá trị mask (`***`, `*******`, `•••`) và KHÔNG khôi phục `user:pass` bị mask (`***:***`) trong URI (`mongodb.connection.string`, `fs.uris`)
- **Bằng chứng (`cdc-cms-service/internal/app/commands/source/debezium_connector.go:427-436`):**
  ```go
  merged := make(map[string]string, len(current)+len(cmd.Config))
  for k, v := range current {
      merged[k] = v
  }
  for k, v := range cmd.Config {
      if v == KeepSecretSentinel {
          continue
      }
      merged[k] = v
  }
  ```
- **Hậu quả:**
  1. **SFTP (`fs.uris`):** `parseConnectionSeed` (`SourceConnectors.tsx:403`) nạp `sftpUri` từ `cfg['fs.uris']` (luôn là `sftp://***:***@host:port/path` do `FilterSafeConfig`). Khi bấm **Update**, `buildConnectorConfig` (`SourceConnectors.tsx:311`) gửi `'fs.uris': 'sftp://***:***@...'` và `UpdateSystemConnectorConfigHandler.Handle` **ghi đè thẳng `sftp://***:***@...` lên Debezium / Kafka Connect**!
  2. **MongoDB (`mongodb.connection.string`):** Khi mở Edit từ tab Connectors (`openEditFromConnector` tại `SourceConnectors.tsx:891`) hoặc khi `source.server_address` rỗng/đã bị mask (`parseConnectionSeed` tại `SourceConnectors.tsx:382-383`), ô `Connection URL` chứa `mongodb://***:***@...`. Khi bấm **Update**, `UpdateSystemConnectorConfigHandler.Handle` **ghi đè thẳng `mongodb://***:***@...` lên Debezium**!
  3. **Custom `kafkaConfig` (`producer.*` / `consumer.*`) & Password gửi dạng `***` / `*******`:** `extractKafkaConfig` (`SourceConnectors.tsx:201-209`) lấy các key `producer.*`/`consumer.*` đã bị `FilterSafeConfig` mask thành `"***"` nạp vào form Edit; hoặc khi client/autofill gửi `"***"` / `"*******"` vào `database.password`, `UpdateSystemConnectorConfigHandler.Handle` ghi đè thẳng `"***"` / `"*******"` lên Debezium.

### `F2` — `validatePostgresConnectorPrereqs` quên chặn `"__KEEP__"` và chạy TRƯỚC khi merge password hiện tại, dẫn tới việc mang chuỗi `"__KEEP__"` đi đăng nhập PostgreSQL nguồn
- **Bằng chứng (`cdc-cms-service/internal/api/source/system_connectors_handler.go:215` & `system_connectors_credentials.go:40-66`):**
  - Tại `system_connectors_handler.go:215`, `UpdateConfig` gọi `h.validatePostgresConnectorPrereqs(c.UserContext(), req.Config)` khi `req.Config["database.password"]` đang là `"__KEEP__"`.
  - Tại `system_connectors_credentials.go:48`:
    ```go
    if host == "" || user == "" || database == "" || pass == "" || pass == "***" {
        return nil
    }
    ```
    Hàm quên kiểm tra `pass == sourceCmd.KeepSecretSentinel` (`"__KEEP__"`) hoặc `isKeepOrMaskedSecret(pass)`! Do đó mỗi lần bấm Update trên `/sources` (bao gồm lần cập nhật `testcoretrans` lúc `13:41:17`), backend tạo DSN với `pass = "__KEEP__"` và mở kết nối tới PostgreSQL nguồn (`10.200.186.110:5000`) với mật khẩu là chuỗi `"__KEEP__"`, vừa gây log lỗi `FATAL: password authentication failed` trên DB nguồn, vừa làm skip toàn bộ bước kiểm tra `wal_level`, `rolreplication`, `schema.include.list` (do `if err != nil { return nil }` ở dòng 60).

### `F3` — `UpdateConfig`, `syncSourceAfterConnectorUpsert` và `extractCredentialsAsOptions` ghi đè URI/username bị mask (`***`) vào `cdc_system.sources` và `cdc_system.connection_registry`
- **Bằng chứng (`cdc-cms-service/internal/api/source/system_connectors_credentials.go:24-30, 164-185` & `system_connectors_handler.go:220-237, 480-526`):**
  1. `isKeepOrMaskedSecret` (`system_connectors_credentials.go:24-30`) chỉ kiểm tra `s == "__KEEP__" || s == "***"`, không bắt chuỗi nhiều dấu sao (`"******"`, `"*******"`) hoặc dấu chấm (`"••••••••••••"`).
  2. Trong `extractCredentialsAsOptions` (`system_connectors_credentials.go:167-169`), khi parse `fs.uris` (`sftp://***:***@...`), code gán `opts["username"] = parts[0]` mà **không kiểm tra `!isKeepOrMaskedSecret(parts[0])`**, làm ghi đè `username: "***"` vào `cdc_system.connection_registry.options_json`.
  3. Trong `UpdateConfig` (`system_connectors_handler.go:223-237`), `syncSourceAfterConnectorUpsert` được gọi với `req.Config` (chưa merge) thay vì config đã resolve secret thật; nếu `req.Config["mongodb.connection.string"]` là `mongodb://***:***@...`, nó ghi đè `cdc_system.sources.server_address`, `cdc_system.connection_registry.host`, và `options_json["uri"]` bằng chuỗi `mongodb://***:***@...`!

### `F4` — Luồng `Recover` (`POST /api/v1/system/connectors/recover`) làm mất `database.password` và không khôi phục secret đã lưu
- **Bằng chứng (`cdc-cms-web/src/pages/SourceConnectors.tsx:250, 275, 852-880` & `cdc-cms-service/internal/api/source/system_connectors_handler.go:444-475`):**
  1. Trong `buildConnectorConfig(values, mode)` (`SourceConnectors.tsx:275`):
     `'database.password': values.password || (mode === 'edit' ? KEEP_SECRET_SENTINEL : '')`
     Khi `mode === 'recover'` và user để trống ô Password, `'database.password'` thành `''` và bị `compactConfig` (`SourceConnectors.tsx:231-237`) xóa mất hoàn toàn!
  2. Trong `SystemConnectorsHandler.Recover` (`system_connectors_handler.go:444-475`), backend đẩy thẳng `req.Config` vào `h.createH.Handle` mà không khôi phục `database.password` / `mongodb.connection.string` / `fs.uris` từ `h.client.GetConfig` hoặc `cdc_system.connection_registry` (`options_json`) / `cdc_system.sources`.

### `F5` — Nút `"Check Connection & Quét Table"` (PostgreSQL) & `"Check Connection"` (MongoDB) trong chế độ `Edit` / `Recover` thất bại vì gửi password rỗng hoặc URI bị mask
- **Bằng chứng (`cdc-cms-web/src/pages/SourceConnectors.tsx:480-597` & `cdc-cms-service/internal/api/system/introspection_handler_postgres.go:70-103`, `introspection_handler.go:116-195`):**
  - Khi mở modal `Edit Config` cho PostgreSQL, ô Password để trống (`Leave blank to keep current`). Nếu user bấm **"Check Connection & Quét Table"** để quét thêm bảng mới, `runCheckPostgresConnection` gửi `password: ""` vào `POST /api/introspection/postgres/tables`, bị báo lỗi `auth_err` (`password authentication failed`).
  - Tương tự với MongoDB, khi `openEdit` tự động chạy `runCheckConnection` (`SourceConnectors.tsx:848`) mà `connectionUrl` đang là `mongodb://***:***@...`, `POST /api/introspection/mongo/databases` gửi thẳng URI bị mask sang worker và lỗi xác thực.

---

## 3. Giải pháp Kỹ thuật Tối ưu Duy nhất (Ánh xạ 1-1 `F1..F5` -> `S1..S5`)

| Phát hiện | Giải pháp Kỹ thuật (1 Phương án Tối ưu Duy nhất) | File tác động |
| :--- | :--- | :--- |
| **`F1`** | **`S1`:** Viết bộ hàm chuẩn hóa & merge secret trong `debezium_connector.go`:<br>- `IsKeepOrMaskedSecret(v string) bool`: nhận diện `"__KEEP__"` và mọi chuỗi chỉ chứa `*` hoặc `•` (`"***"`, `"*******"`, `"••••••••••••"`).<br>- `IsSensitiveConnectorKey(k string) bool`: đồng bộ tập từ khóa nhạy cảm (`password`, `secret`, `token`, `credentials`, `ssl.key`).<br>- `RestoreMaskedURI(incomingURI, currentURI string) string`: nếu `incomingURI` chứa user/pass bị mask (`***` hoặc `__KEEP__`), tự động tách `userinfo` thật từ `currentURI` ghép vào authority của `incomingURI` (giữ nguyên mọi thay đổi host/port/path/query của user).<br>- `MergeConnectorConfigPreservingSecrets(current, incoming map[string]string) map[string]string`: dùng trong `UpdateSystemConnectorConfigHandler.Handle` để không bao giờ đẩy `***` / `*******` / `***:***` lên Debezium. | [debezium_connector.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go#L356-L450) |
| **`F2` & `F3` & `F4`** | **`S2 + S3 + S4`:** Trong `system_connectors_credentials.go` và `system_connectors_handler.go`:<br>- Thêm `resolveEffectiveConnectorConfig(ctx, name, incoming)` kết hợp config hiện tại từ Kafka Connect (`h.client.GetConfig`) và fallback từ `cdc_system.sources` + `cdc_system.connection_registry.options_json`, sau đó gọi `sourceCmd.MergeConnectorConfigPreservingSecrets`.<br>- Trong `UpdateConfig` và `Recover`: gọi `effectiveCfg := h.resolveEffectiveConnectorConfig(...)` **trước** `validatePostgresConnectorPrereqs` và `syncSourceAfterConnectorUpsert`.<br>- Trong `validatePostgresConnectorPrereqs`: dùng `isKeepOrMaskedSecret(pass)` thay vì `pass == "***"`.<br>- Trong `extractCredentialsAsOptions`: kiểm tra `!isKeepOrMaskedSecret(parts[0])` cho SFTP username và `!sourceCmd.IsMaskedURI(mongoURI)` cho MongoDB URI; trong `syncSourceAfterConnectorUpsert`: không ghi đè `server_address` / `connection_registry.host` bằng URI bị mask. | [system_connectors_credentials.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_credentials.go#L1-L187)<br>[system_connectors_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go#L198-L251) |
| **`F5`** | **`S5`:** Hỗ trợ kiểm tra kết nối & quét bảng/collection khi Edit/Recover mà không lộ password xuống trình duyệt:<br>- Trong `IntrospectionHandler` (`introspection_handler.go`, `introspection_handler_postgres.go`, `server.go`): thêm `SetConnectorConfigReader(kafkaConnectClient)`. Khi `DiscoverPostgresTablesPost`, `DiscoverMongoDatabasesPost`, `DiscoverMongoCollectionsPost` nhận `connector_name` mà `password` rỗng/mask hoặc `uri` bị mask (`***`), tự động lấy password/URI thật từ `connectorCfgReader.GetConfig(ctx, connectorName)`.<br>- Trong `SourceConnectors.tsx`:<br>  1. Ở `buildConnectorConfig`: dùng `(mode === 'edit' \|\| mode === 'recover') ? KEEP_SECRET_SENTINEL : ''` cho `database.password`, và chuyển các giá trị mask (`***` / toàn `*`) trong `kafkaConfig` sang `KEEP_SECRET_SENTINEL`.<br>  2. Truyền `connector_name: modal.connector?.name \|\| modal.source?.connector_name` vào `POST /api/introspection/postgres/tables`, `POST /api/introspection/mongo/databases`, `POST /api/introspection/mongo/collections`. | [introspection_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler.go#L15-L195)<br>[introspection_handler_postgres.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler_postgres.go#L16-L105)<br>[server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L330-L335)<br>[SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx#L250-L600) |

---

## 4. Code Demo Chi Tiết (Zero Truncation)

### Demo 1: `cdc-cms-service/internal/app/commands/source/debezium_connector.go` (`S1`)

```go
// ── Update Config ──────────────────────────────────────────────────────

const KeepSecretSentinel = "__KEEP__"

// IsKeepOrMaskedSecret returns true if v is the KeepSecretSentinel ("__KEEP__")
// or a non-empty string composed exclusively of masking characters ('*' or '•').
func IsKeepOrMaskedSecret(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	if s == KeepSecretSentinel {
		return true
	}
	for _, r := range s {
		if r != '*' && r != '•' {
			return false
		}
	}
	return true
}

// IsSensitiveConnectorKey matches the sensitive key criteria in infrahttp.FilterSafeConfig.
func IsSensitiveConnectorKey(k string) bool {
	lk := strings.ToLower(strings.TrimSpace(k))
	return strings.Contains(lk, "password") ||
		strings.Contains(lk, "secret") ||
		strings.Contains(lk, "token") ||
		strings.Contains(lk, "credentials") ||
		strings.Contains(lk, "ssl.key")
}

// splitURIAuthority splits a URI into (schemePrefix, userinfo, hostPort, pathAndQuery, hasUserInfo).
func splitURIAuthority(raw string) (string, string, string, string, bool) {
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd == -1 {
		return "", "", "", "", false
	}
	schemePrefix := raw[:schemeEnd+3]
	rest := raw[schemeEnd+3:]
	slashIdx := strings.IndexAny(rest, "/?")
	authority := rest
	pathAndQuery := ""
	if slashIdx != -1 {
		authority = rest[:slashIdx]
		pathAndQuery = rest[slashIdx:]
	}
	atIdx := strings.LastIndex(authority, "@")
	if atIdx == -1 {
		return schemePrefix, "", authority, pathAndQuery, false
	}
	return schemePrefix, authority[:atIdx], authority[atIdx+1:], pathAndQuery, true
}

// IsMaskedURI returns true if raw URI contains masked userinfo (*** or __KEEP__).
func IsMaskedURI(raw string) bool {
	_, userInfo, _, _, ok := splitURIAuthority(strings.TrimSpace(raw))
	if !ok {
		return IsKeepOrMaskedSecret(raw)
	}
	parts := strings.SplitN(userInfo, ":", 2)
	if IsKeepOrMaskedSecret(parts[0]) {
		return true
	}
	if len(parts) == 2 && IsKeepOrMaskedSecret(parts[1]) {
		return true
	}
	return false
}

// RestoreMaskedURI restores masked username/password in incomingURI from currentURI
// while preserving any host/port/path/query changes made in incomingURI.
func RestoreMaskedURI(incomingURI, currentURI string) string {
	inTrim := strings.TrimSpace(incomingURI)
	curTrim := strings.TrimSpace(currentURI)
	if inTrim == "" || curTrim == "" {
		return incomingURI
	}
	if IsKeepOrMaskedSecret(inTrim) {
		return currentURI
	}
	inScheme, inUserInfo, inHostPort, inPathQuery, inHasUser := splitURIAuthority(inTrim)
	_, curUserInfo, _, _, curHasUser := splitURIAuthority(curTrim)
	if !inHasUser || !curHasUser || curUserInfo == "" {
		return incomingURI
	}
	inParts := strings.SplitN(inUserInfo, ":", 2)
	curParts := strings.SplitN(curUserInfo, ":", 2)

	resolvedUser := inParts[0]
	if IsKeepOrMaskedSecret(resolvedUser) || resolvedUser == "" {
		resolvedUser = curParts[0]
	}
	if len(inParts) == 2 {
		resolvedPass := inParts[1]
		if IsKeepOrMaskedSecret(resolvedPass) && len(curParts) == 2 {
			resolvedPass = curParts[1]
		}
		return inScheme + resolvedUser + ":" + resolvedPass + "@" + inHostPort + inPathQuery
	}
	if IsKeepOrMaskedSecret(inParts[0]) && len(curParts) == 2 {
		return inScheme + resolvedUser + ":" + curParts[1] + "@" + inHostPort + inPathQuery
	}
	return inScheme + resolvedUser + "@" + inHostPort + inPathQuery
}

// MergeConnectorConfigPreservingSecrets merges incoming config into current config while
// preserving existing secrets whenever incoming contains __KEEP__, masked asterisks (***),
// empty sensitive values, or masked URIs (mongodb.connection.string, fs.uris).
func MergeConnectorConfigPreservingSecrets(current, incoming map[string]string) map[string]string {
	merged := make(map[string]string, len(current)+len(incoming))
	for k, v := range current {
		merged[k] = v
	}
	for k, v := range incoming {
		if IsKeepOrMaskedSecret(v) {
			continue
		}
		if strings.TrimSpace(v) == "" && IsSensitiveConnectorKey(k) {
			if curVal, ok := current[k]; ok && strings.TrimSpace(curVal) != "" {
				continue
			}
		}
		if k == "mongodb.connection.string" || k == "fs.uris" {
			if curURI, ok := current[k]; ok && strings.TrimSpace(curURI) != "" {
				v = RestoreMaskedURI(v, curURI)
			}
		}
		merged[k] = v
	}
	return merged
}
```

Và trong `UpdateSystemConnectorConfigHandler.Handle` (`debezium_connector.go:427-438`):
```go
	merged := MergeConnectorConfigPreservingSecrets(current, cmd.Config)

	resp, err := h.writer.UpdateConfig(ctx, cmd.Name, merged)
```

---

### Demo 2: `cdc-cms-service/internal/api/source/system_connectors_credentials.go` & `system_connectors_handler.go` (`S2 + S3 + S4`)

Trong `system_connectors_credentials.go`:
```go
func isKeepOrMaskedSecret(v string) bool {
	return sourceCmd.IsKeepOrMaskedSecret(v)
}

// resolveEffectiveConnectorConfig merges incoming config with live Kafka Connect config
// (or stored DB credentials fallback) so secret placeholders (__KEEP__, ***, masked URIs)
// are resolved before validation, recovery, and DB metadata sync.
func (h *SystemConnectorsHandler) resolveEffectiveConnectorConfig(
	ctx context.Context,
	name string,
	incoming map[string]string,
) map[string]string {
	baseline := make(map[string]string)
	if h.sourceRepo != nil && name != "" {
		if fp, err := h.sourceRepo.GetByConnectorName(ctx, name); err == nil && fp != nil {
			if len(fp.RawConfigSanitized) > 0 {
				var rawMap map[string]string
				if err := json.Unmarshal(fp.RawConfigSanitized, &rawMap); err == nil {
					for k, v := range rawMap {
						if !isKeepOrMaskedSecret(v) && !sourceCmd.IsMaskedURI(v) {
							baseline[k] = v
						}
					}
				}
			}
			if fp.ServerAddress != "" && !sourceCmd.IsMaskedURI(fp.ServerAddress) {
				if fp.SourceType == "mongodb" {
					baseline["mongodb.connection.string"] = fp.ServerAddress
				}
			}
		}
	}
	if h.client != nil && name != "" {
		if liveCfg, err := h.client.GetConfig(ctx, name); err == nil && len(liveCfg) > 0 {
			for k, v := range liveCfg {
				baseline[k] = v
			}
		}
	}
	return sourceCmd.MergeConnectorConfigPreservingSecrets(baseline, incoming)
}
```

Trong `validatePostgresConnectorPrereqs` (`system_connectors_credentials.go:48`) và `extractCredentialsAsOptions` (`system_connectors_credentials.go:164-185`):
```go
	if host == "" || user == "" || database == "" || pass == "" || isKeepOrMaskedSecret(pass) {
		return nil
	}
```
```go
	case "sftp":
		sftpURI := strings.TrimSpace(cfg["fs.uris"])
		if u, err := url.Parse(sftpURI); err == nil && u.User != nil {
			parts := strings.SplitN(u.User.String(), ":", 2)
			if parts[0] != "" && !isKeepOrMaskedSecret(parts[0]) {
				opts["username"] = parts[0]
			}
			if len(parts) == 2 && !isKeepOrMaskedSecret(parts[1]) {
				opts["password"] = parts[1]
			}
		}
	default:
		mongoURI := strings.TrimSpace(cfg["mongodb.connection.string"])
		if mongoURI != "" && !isKeepOrMaskedSecret(mongoURI) && !sourceCmd.IsMaskedURI(mongoURI) {
			opts["uri"] = mongoURI
		}
```

Trong `SystemConnectorsHandler.UpdateConfig` (`system_connectors_handler.go:213-246`) và `Recover` (`system_connectors_handler.go:459-470`):
```go
	effectiveCfg := h.resolveEffectiveConnectorConfig(c.UserContext(), name, req.Config)
	if err := h.validatePostgresConnectorPrereqs(c.UserContext(), effectiveCfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	updatedBy := resolveActor(c)

	var fp *model.Source
	if h.sourceRepo != nil {
		fp, _ = h.sourceRepo.GetByConnectorName(c.UserContext(), name)
		rawCfg, _ := json.Marshal(infrahttp.FilterSafeConfig(effectiveCfg))
		connClass := effectiveCfg["connector.class"]
		if fp == nil {
			fp = &model.Source{
				ConnectorName:      name,
				ConnectorClass:     connClass,
				SourceType:         ConnectorTypeFromClass(connClass),
				Status:             "running",
				RawConfigSanitized: rawCfg,
				UpdatedBy:          updatedBy,
			}
		} else {
			fp.RawConfigSanitized = rawCfg
			fp.UpdatedBy = updatedBy
		}
		h.syncSourceAfterConnectorUpsert(c.UserContext(), name, effectiveCfg, ConnectorTypeFromClass(connClass), rawCfg, updatedBy)
	}

	out, err := h.updateH.Handle(c.UserContext(), sourceCmd.UpdateSystemConnectorConfigCommand{
		Name:        name,
		Config:      effectiveCfg,
		Fingerprint: fp,
		UpdatedBy:   updatedBy,
	})
```

Và trong `SystemConnectorsHandler.Recover` (`system_connectors_handler.go:459-470`):
```go
	req.Config = h.resolveEffectiveConnectorConfig(c.UserContext(), req.Name, req.Config)
	if err := h.validatePostgresConnectorPrereqs(c.UserContext(), req.Config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
```

Và trong `syncSourceAfterConnectorUpsert` (`system_connectors_handler.go:483-516`):
```go
	host := cfg["database.hostname"]
	if host == "" {
		if mongoURI := cfg["mongodb.connection.string"]; !sourceCmd.IsMaskedURI(mongoURI) {
			host = mongoURI
		}
	}
```

---

### Demo 3: `cdc-cms-service/internal/api/system/introspection_handler.go`, `introspection_handler_postgres.go` & `SourceConnectors.tsx` (`S5`)

Trong `introspection_handler.go`:
```go
type ConnectorConfigReader interface {
	GetConfig(ctx context.Context, name string) (map[string]string, error)
}

type IntrospectionHandler struct {
	rpc                ports.RPCClient
	mappingRuleRepo    ports.MappingRuleRepo
	shadowReader       ports.ShadowSchemaReader
	masterRepo         ports.MasterRepo
	connectorCfgReader ConnectorConfigReader
}

func (h *IntrospectionHandler) SetConnectorConfigReader(r ConnectorConfigReader) {
	h.connectorCfgReader = r
}
```
Trong `DiscoverPostgresTablesPost` (`introspection_handler_postgres.go:16-85`):
```go
type DiscoverPostgresTablesRequest struct {
	ConnectorName string `json:"connector_name"`
	Host          string `json:"host"`
	Port          any    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Database      string `json:"database"`
	Schemas       string `json:"schemas"`
	SSLMode       string `json:"sslmode"`
}
```
Khi `pass == "" || sourceCmd.IsKeepOrMaskedSecret(pass)` và `body.ConnectorName != ""` và `h.connectorCfgReader != nil`:
```go
	if (pass == "" || sourceCmd.IsKeepOrMaskedSecret(pass)) && strings.TrimSpace(body.ConnectorName) != "" && h.connectorCfgReader != nil {
		if liveCfg, err := h.connectorCfgReader.GetConfig(c.UserContext(), strings.TrimSpace(body.ConnectorName)); err == nil {
			if livePass := liveCfg["database.password"]; livePass != "" && !sourceCmd.IsKeepOrMaskedSecret(livePass) {
				pass = livePass
			}
		}
	}
```
Tương tự trong `DiscoverMongoDatabasesPost` và `DiscoverMongoCollectionsPost` (`introspection_handler.go:116-195`): thêm `ConnectorName string` (`json:"connector_name"`) và khi `sourceCmd.IsMaskedURI(body.URI)` thì gọi `h.connectorCfgReader.GetConfig` để dùng `sourceCmd.RestoreMaskedURI(body.URI, liveCfg["mongodb.connection.string"])`.

Trong `cdc-cms-web/src/pages/SourceConnectors.tsx`:
1. Tại `buildConnectorConfig` (`SourceConnectors.tsx:250-290` & `321-333`):
```ts
  const keepExistingSecret = mode === 'edit' || mode === 'recover';
```
```ts
    const base: Record<string, string | undefined> = {
      'connector.class': connectorClass,
      'database.hostname': values.host,
      'database.port': String(values.port || 5432),
      'database.user': values.username,
      'database.password': values.password || (keepExistingSecret ? KEEP_SECRET_SENTINEL : ''),
```
```ts
  if (Array.isArray(values.kafkaConfig)) {
    for (const item of values.kafkaConfig) {
      if (item && item.key && item.value !== undefined && item.value !== '') {
        const trimmedVal = String(item.value).trim();
        const isAllStars = trimmedVal.length > 0 && /^[•*]+$/.test(trimmedVal);
        out[item.key.trim()] = (keepExistingSecret && isAllStars) ? KEEP_SECRET_SENTINEL : item.value;
      }
    }
  }
```
2. Tại `runCheckConnection`, `loadCollectionsForDb`, và `runCheckPostgresConnection` (`SourceConnectors.tsx:492-595`): gửi kèm `connector_name: modal.connector?.name || modal.source?.connector_name || v.name || ''`.

---

## 5. Cập nhật Theo 6 Ràng Buộc Kỹ Thuật Bắt Buộc Từ User `APPROVE` (`G1..G6`)

### 5.1. `G1` — Fix triệt để `Recover` (`options_json`), Lọc `liveCfg` Sạch, Chốt chặn Fail-Closed (`ValidateNoMaskedSecrets`) & Tránh gọi trùng `GetConfig`
1. **Đọc `connection_registry.options_json` trong `GetByConnectorName` (`system_connector_repo_gorm.go:125-135`):**
   - Trong `GetByConnectorName`, truy vấn thêm `SELECT COALESCE(options_json::text, '') FROM cdc_system.connection_registry WHERE connection_code = ? LIMIT 1` và gán vào `s.OptionsJSON`. Trường `OptionsJSON` có tag `gorm:"-" json:"-"` nên không bao giờ lộ ra JSON API nhưng cung cấp đầy đủ `username`, `password`, `sslmode`, `uri` khi `liveCfg` trả `404` (luồng `Recover` hoặc lazy SFTP).
2. **`BuildStoredConfigBaseline(fp *model.Source, liveCfg map[string]string, incoming map[string]string)` trong `debezium_connector_secrets.go`:**
   - Nạp các key không bị mask từ `fp.RawConfigSanitized` và `fp.ServerAddress`.
   - Giải mã `fp.OptionsJSON`: khôi phục `database.user`, `database.password`, `database.sslmode`, `mongodb.password`, `sftp.username`, `sftp.password`, `mongodb.connection.string` (từ `opts["uri"]` hoặc ghép `username:password` vào `incoming["mongodb.connection.string"]`), và `fs.uris` (ghép `username:password` vào `incoming["fs.uris"]`).
   - Khi merge `liveCfg` vào `baseline`: **bắt buộc lọc `!IsKeepOrMaskedSecret(v) && !IsMaskedURI(v)`** để giá trị bẩn (nếu có) không đè mất baseline sạch từ DB.
3. **Chốt chặn Fail-Closed `ValidateNoMaskedSecrets(cfg map[string]string) error`:**
   - Chặn cứng (trả lỗi `400 Bad Request` tại handler hoặc lỗi tại command handler trước khi gọi Kafka Connect) nếu bất kỳ key nào trong `cfg` còn chứa `IsKeepOrMaskedSecret(v)` hoặc `IsMaskedURI(v)`, hoặc nếu thiếu `database.password` (với Postgres/MySQL/SQLServer), `mongodb.connection.string` (với MongoDB), `fs.uris` (với `FsSourceConnector`).
4. **Tránh gọi trùng `KafkaConnectClient.GetConfig` 2 lần:**
   - Thêm cờ `PreResolved bool` vào `UpdateSystemConnectorConfigCommand`. Khi `SystemConnectorsHandler.UpdateConfig` đã gọi `resolveEffectiveConnectorConfig` (đã gọi `GetConfig` 1 lần và kiểm tra `ValidateNoMaskedSecrets`), nó đặt `PreResolved: true` (kèm `LiveMissing: !liveFound` cho nhánh lazy SFTP) để `UpdateSystemConnectorConfigHandler.Handle` không gọi `h.writer.GetConfig` lần thứ 2.

### 5.2. `G2` — Chống Lộ Secret Qua API `Introspection` Khi Đổi Host (Host-Bound Secret Resolution)
- Trong `DiscoverPostgresTablesPost` (`introspection_handler_postgres.go`):
  - Khi `pass == "" || sourceCmd.IsKeepOrMaskedSecret(pass)` và `body.ConnectorName != ""`:
    Đọc cấu hình hiện tại của connector qua `h.resolveStoredConnectorConfig(c.UserContext(), body.ConnectorName, nil)`.
    Chỉ cho phép lấy `storedCfg["database.password"]` khi `strings.EqualFold(host, storedHost) && port == storedPort && (storedUser == "" || user == storedUser)`. Nếu user đã đổi sang host/port/user khác, trả ngay `"status": "auth_err"`, `"error": "Host/Port/Username đã thay đổi so với cấu hình hiện tại của connector — vui lòng nhập lại Password để kiểm tra kết nối."` (không mang mật khẩu cũ kết nối sang host lạ).
- Trong `DiscoverMongoDatabasesPost` & `DiscoverMongoCollectionsPost` (`introspection_handler.go`):
  - Khi `sourceCmd.IsMaskedURI(body.URI)` và `body.ConnectorName != ""`:
    Kiểm tra `sourceCmd.SameURIHostAuthority(body.URI, storedURI)` (so sánh `scheme` và `host:port` của URI). Chỉ khi khớp tuyệt đối mới gọi `sourceCmd.RestoreMaskedURI(body.URI, storedURI)`; nếu khác host trả lỗi `400` (`"status": "auth_err"`).

### 5.3. `G3` — Sửa 3 Ca Biên Xử Lý URI & Credentials
1. **Hỗ trợ `fs.uris` chứa nhiều URI phân tách bằng dấu phẩy (`sftp://u:p@h1/d1,sftp://u:p@h2/d2`):**
   - Hàm `splitMultiURIList(raw string)` nhận diện danh sách nhiều URI phân tách bằng dấu phẩy (khi phần tử sau dấu phẩy chứa `://`, phân biệt với multi-host replicaSet của MongoDB chỉ có một `mongodb://` ở đầu).
   - `IsMaskedURI`, `RestoreMaskedURI`, `MaskURIUserInfo`, và `SameURIHostAuthority` đều xử lý từng URI trong danh sách phân tách bằng dấu phẩy.
   - Trong `MergeConnectorConfigPreservingSecrets`: áp dụng kiểm tra `IsMaskedURI(v)` cho **mọi key** trong map (không khóa cứng tên key).
2. **Xử lý `len(curParts) == 1` trong `RestoreMaskedURI`:**
   - Khi `len(inParts) == 2` và `IsKeepOrMaskedSecret(resolvedPass)`: nếu `len(curParts) == 2 && !IsKeepOrMaskedSecret(curParts[1])` thì lấy `curParts[1]`, ngược lại (URI gốc chỉ có `user@host` không có password) trả về `inScheme + resolvedUser + "@" + inHostPort + inPathQuery` (tuyệt đối không để sót `:***`).
3. **Dùng `u.User.Username()` và `u.User.Password()` trong `extractCredentialsAsOptions`:**
   - Parse URI đầu tiên trong `fs.uris` bằng `url.Parse` và dùng `u.User.Username()` + `u.User.Password()` để tự động giải mã `%E2%80%A2` về `•` (bị `isKeepOrMaskedSecret` chặn đứng) và giải mã đúng `%40`/`%3A` trong mật khẩu thật.

### 5.4. `G4` — Khép Kín Luồng `F5` Cho MongoDB & Frontend `SourceConnectors.tsx`
- Cài đặt đầy đủ `resolveMongoIntrospectionURI` cho cả `DiscoverMongoDatabasesPost` và `DiscoverMongoCollectionsPost` trong `introspection_handler.go`.
- Trong `SourceConnectors.tsx`, `useConnectorCheck.ts`, `connectorCheck.ts`:
  - Truyền `connector_name` khi `editorMode === 'edit' || editorMode === 'recover'` cho cả `checkHook.check` (MongoDB) và `pgCheckHook.check` (PostgreSQL).
  - Bổ sung `isMaskedSecretInput` kiểm tra `/^[•*]+$/` cho cả `values.password` và `kafkaConfig`, chuẩn hóa về `KEEP_SECRET_SENTINEL` khi `mode === 'edit' || mode === 'recover'`.

### 5.5. `G5` — Bảo Vệ Sanitization Ở Bảng `sources` & API `GET /api/v1/sources`
- `RawConfigSanitized` luôn được tạo từ `infrahttp.FilterSafeConfig(effectiveCfg)`.
- Trong `ListSourcesHandler.Handle`, `GetSourceHandler.Handle` (`list_sources.go`) và `SourcesHandler.Create` (`sources_handler.go`): gọi `sourceCmd.MaskURIUserInfo(s.ServerAddress)` trước khi trả kết quả về HTTP client để tuyệt đối không làm lộ plaintext password trong `server_address` trên DevTools Network, trong khi DB vẫn lưu đầy đủ thông tin tại `connection_registry.options_json`.

### 5.6. `G6` — Tuân Thủ Trần 600 Dòng (`wc -l`) & Unit Test Toàn Diện
- Tách toàn bộ hàm pure logic xử lý secret/URI sang file mới `cdc-cms-service/internal/app/commands/source/debezium_connector_secrets.go` kèm `debezium_connector_secrets_test.go`.
- Chuyển `parseFingerprint`, `extractSFTPCollectionName` và `resolveEffectiveConnectorConfig` sang `cdc-cms-service/internal/api/source/system_connectors_credentials.go`, kéo `system_connectors_handler.go` từ `597` dòng xuống `~515` dòng (an toàn dưới ngưỡng 600 dòng).

### 5.7. Bổ Sung Hậu Kiểm `V1..V4` — Khóa Host-Bound Secret Reuse Tại `MergeConnectorConfigPreservingSecrets` & Khử Trùng Lặp Cho Laya Mode B
- **`V3` (Host-Bound Secret Reuse trong `UpdateConfig` & `Recover`):**
  - Bổ sung hàm `SameConnectorEndpointIdentity(current, incoming map[string]string) bool` trong `debezium_connector_secrets.go`: so sánh `database.hostname`, `database.port` (mặc định `"5432"` cho Postgres, `"3306"` cho MySQL nếu một bên rỗng), `database.user`, `mongodb.user`, `sftp.host`, `sftp.port`, `sftp.username`. Nếu `current` có giá trị và `incoming` đổi sang host/port/user khác thì trả về `false`.
  - Trong `MergeConnectorConfigPreservingSecrets(current, incoming)`:
    - Khi gặp `IsKeepOrMaskedSecret(v)` hoặc chuỗi rỗng trên key nhạy cảm: chỉ khôi phục `current[k]` khi `SameConnectorEndpointIdentity(current, incoming) == true`. Nếu đổi host/port/user mà vẫn gửi `"__KEEP__"` / `"***"`, giữ nguyên `merged[k] = v` để `ValidateNoMaskedSecrets` chặn cứng với HTTP `400 Bad Request` trước khi gọi `validatePostgresConnectorPrereqs` hay Kafka Connect.
    - Khi gặp `IsMaskedURI(v)`: chỉ gọi `RestoreMaskedURI(v, curURI)` khi `SameURIHostAuthority(v, curURI) == true`. Nếu đổi host authority trong `mongodb.connection.string` hoặc `fs.uris` mà vẫn để `***:***`, giữ nguyên URI bị mask trong `merged[k]` để `ValidateNoMaskedSecrets` chặn cứng HTTP `400`.
  - Trong `BuildStoredConfigBaseline(fp, liveCfg, incoming)`: chỉ nạp secret dự phòng từ `fp.OptionsJSON` / `liveCfg` nếu khớp host/port/user và `SameURIHostAuthority`.
- **`V1` (Gom `MaskURIUserInfo` về `internal/model/source/source.go` để khử `REJECT_REDUNDANT` ở Laya Mode B):**
  - Chuyển hàm `MaskURIUserInfo` về `internal/model/source/source.go` để cả `internal/app/queries/source/list_sources.go`, `internal/api/source/sources_handler.go` và `internal/app/commands/source/debezium_connector_secrets.go` gọi chung mà không nhân bản code hay tạo phụ thuộc ngược `queries -> commands`.


