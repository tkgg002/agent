---
tier: T2
suffix: debezium_config_masked_password
workspace: fix-debezium-config-masked-password-2026-11-10
created_at: 2026-11-10T13:44:00+07:00
---

# 01. Requirements — Kiểm tra & xử lý lỗi cập nhật config `/sources` đẩy password bị mask (`***` / `*******`) lên Debezium

## 1. Yêu cầu của User
- Khi ở trang `/sources` (Source Connectors) cập nhật config của connector, password bị đẩy lên Debezium dưới dạng chuỗi đã mask (`*******` / `***`).
- Yêu cầu: Kiểm tra nguyên nhân gốc rễ (Frontend → CMS Backend → Kafka Connect / Debezium) và thiết kế giải pháp xử lý triệt để.

## 2. Ràng buộc Thực thi (Technical Constraints từ Global Lessons)
- **L025 (Frontend Call-Site Tracing):** Bắt buộc truy vết trực tiếp từ component trang `/sources` (`SourceConnectors.tsx` hoặc tương đương) qua modal chỉnh sửa config đến hàm gọi API (`services/api.ts`) và handler phía `cdc-cms-service` (`system_connectors_handler.go` / `UpdateSystemConnectorConfigCommand`).
- **L067 (End-to-End Secret Masking & RFC 3986 URI Handling):**
  - API `GET` trả về config đã mask mật khẩu/URI (`***` hoặc `******`) để bảo mật trên UI và Audit Log.
  - Khi `PATCH` / `PUT` cập nhật config lên Debezium, nếu người dùng không nhập mật khẩu mới (giữ nguyên chuỗi mask `***`/`******` hoặc URI có `:***` từ payload GET), hệ thống **tuyệt đối không được gửi chuỗi mask đó thẳng sang Kafka Connect REST API**, mà phải khôi phục mật khẩu/URI thật từ cấu hình hiện hữu trên Kafka Connect hoặc `cdc_system.sources` trước khi đẩy lên Debezium.
  - Xử lý URI theo chuẩn RFC 3986 dùng `strings.LastIndex(authority, "@")` và hỗ trợ chuỗi multi-host Replica Set.
- **L009 & L017 (Verbatim Disk-Source Parity & 1-to-1 Code Demo):** Đọc trực tiếp file nguồn trên đĩa, trích dẫn chính xác `file:line`, trình bày Code Demo khớp 1-1 không dùng dấu `...` rút gọn, và giữ chế độ **Propose-Only** dừng chờ User `APPROVE`.
- **L020 (Respect Live Production Baseline):** Tuân thủ kiến trúc Hexagonal/CQRS của `cdc-cms-service` và Feature/Hook-Driven của `cdc-cms-web`, bảo toàn 100% tương thích ngược.

## 3. Sáu Ràng buộc Kỹ thuật Bắt buộc từ User `APPROVE` (`G1..G6`)
- **`G1` (Fix triệt để Recover & Chốt chặn Fail-Closed):** Đọc credentials từ `connection_registry.options_json` (`username`, `password`, `sslmode`, `uri`) qua `GetByConnectorName` khi `liveCfg` trả 404 hoặc bị mask; lọc `!IsKeepOrMaskedSecret(v) && !IsMaskedURI(v)` khi merge `liveCfg` vào `baseline`; bổ sung `ValidateNoMaskedSecrets(cfg)` chặn cứng (Fail-Closed HTTP 400) trước khi gọi Kafka Connect; không gọi trùng `GetConfig` 2 lần trong cùng 1 request `UpdateConfig`.
- **`G2` (Chống lộ Secret qua Introspection khi đổi Host):** Chỉ cho phép nạp secret lưu trữ theo `connector_name` tại `DiscoverPostgresTablesPost`, `DiscoverMongoDatabasesPost`, `DiscoverMongoCollectionsPost` khi `host` (kèm `port`/`user` với Postgres, hoặc host authority với Mongo URI) khớp với cấu hình hiện tại của connector.
- **`G3` (Sửa 3 ca biên URI & Credentials):** Hỗ trợ `fs.uris` chứa nhiều URI phân tách bằng dấu phẩy; trong `RestoreMaskedURI` khi `len(curParts) == 1` trả về `scheme://resolvedUser@host...` không dư `:***`; dùng `u.User.Username()` và `u.User.Password()` trong `extractCredentialsAsOptions`; áp dụng `IsMaskedURI` cho mọi key trong `MergeConnectorConfigPreservingSecrets`.
- **`G4` (Khép kín luồng `F5` cho MongoDB & Frontend):** Cài đặt khôi phục URI bị mask cho `DiscoverMongoDatabasesPost` & `DiscoverMongoCollectionsPost`; tại `SourceConnectors.tsx` truyền `connector_name` khi `mode === 'edit' || mode === 'recover'`, không chặn client-side khi password trống ở `edit`/`recover`, chuẩn hóa cả `values.password` toàn `*`/`•` về `KEEP_SECRET_SENTINEL`.
- **`G5` (Bảo vệ Sanitization ở bảng `sources`):** Đảm bảo `raw_config_sanitized` luôn qua `FilterSafeConfig` và mask `server_address` trên response `GET /api/v1/sources` (`ListSourcesHandler` / `GetSourceHandler` / `SourcesHandler.Create`) để không lộ plaintext password trong URI ra DevTools.
- **`G6` (Tuân thủ trần 600 dòng `wc -l` & Unit Test):** Giữ `system_connectors_handler.go` dưới 600 dòng bằng cách chuyển logic sang `system_connectors_credentials.go`; tách bộ hàm xử lý secret/URI sang `internal/app/commands/source/debezium_connector_secrets.go` kèm `debezium_connector_secrets_test.go`.
