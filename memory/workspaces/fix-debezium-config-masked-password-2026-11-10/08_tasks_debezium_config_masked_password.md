# 08. Tasks — Kiểm tra & xử lý lỗi cập nhật config `/sources` đẩy password bị mask lên Debezium

## Task 1: Điều tra End-to-End luồng đọc & cập nhật Connector Config trên `/sources` (S2)
- **Mô tả & Context:**
  - Truy vết từ giao diện `/sources` (`cdc-cms-web/src/pages/SourceConnectors.tsx` và các component/hook liên quan) khi mở Modal Edit Config và khi bấm Save/Update.
  - Truy vết API đọc chi tiết connector (`GET /api/v1/system/connectors/:name`) và danh sách connector (`GET /api/v1/system/connectors`) trong `cdc-cms-service` xem cơ chế mask password/URI (`database.password`, `mongodb.password`, `mongodb.connection.string`, `fs.uris`, v.v.) đang hoạt động như thế nào.
  - Truy vết API cập nhật config (`PATCH /api/v1/system/connectors/:name/config` tại `system_connectors_handler.go` -> `UpdateSystemConnectorConfigCommand` -> `KafkaConnectClient.UpdateConnectorConfig`) và kiểm tra trạng thái thực tế của connector trên Kafka Connect / Debezium runtime.
- **Logs/Test:** Kiểm tra code thực tế kèm trạng thái config hiện tại trên Kafka Connect (`:8083` hoặc container/API tương ứng).
- **Definition of Done:** Xác định chính xác 100% `file:line` nơi password/URI bị mask khi trả về UI và bị ghi đè nguyên chuỗi `***`/`******` lên Kafka Connect khi cập nhật config.

## Task 2: Phản biện Pre-Plan DG0 & Thiết kế Giải pháp + Code Demo (S2.5 & S3)
- **Mô tả & Context:**
  - Chạy cổng phản biện `DG0` (`laya_gate.py run --stage dg0`).
  - Thiết kế giải pháp tối ưu duy nhất trong `09_tasks_solution_debezium_config_masked_password.md` kèm Code Demo đầy đủ và chạy `DG1` (`laya_gate.py run --stage dg1`).
- **Definition of Done:** `DG0` và `DG1` đạt `PASS`, hiển thị Code Demo trực tiếp trên chat và dừng chờ `APPROVE` từ User.

## Task 3: Bổ sung `MergeConnectorConfigPreservingSecrets` & `RestoreMaskedURI` trong `debezium_connector.go` (`S1` — sau `APPROVE`)
- **Mô tả & Context:**
  - File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go:356-450`.
  - Hiện thực `IsKeepOrMaskedSecret`, `IsSensitiveConnectorKey`, `IsMaskedURI`, `RestoreMaskedURI`, và `MergeConnectorConfigPreservingSecrets`.
  - Cập nhật `UpdateSystemConnectorConfigHandler.Handle` gọi `MergeConnectorConfigPreservingSecrets(current, cmd.Config)`.
- **Logs/Test:** Viết unit test bao phủ các trường hợp `__KEEP__`, `***`, `*******`, `••••••••••••`, `mongodb.connection.string` bị mask (`***:***`) kèm đổi host/query, `fs.uris` bị mask (`***:***`), và sensitive key rỗng.
- **Definition of Done:** `go test ./internal/app/commands/source/...` PASS 100% qua `evidence_run.py`.

## Task 4: Chuẩn hóa `resolveEffectiveConnectorConfig` trong `system_connectors_credentials.go` & `system_connectors_handler.go` (`S2 + S3 + S4` — sau `APPROVE`)
- **Mô tả & Context:**
  - Files:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_credentials.go:24-187`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go:198-251, 444-526`
  - Gọi `resolveEffectiveConnectorConfig` trong `UpdateConfig` và `Recover` trước `validatePostgresConnectorPrereqs` và `syncSourceAfterConnectorUpsert`.
  - Chặn `isKeepOrMaskedSecret(parts[0])` cho SFTP username và `!sourceCmd.IsMaskedURI(mongoURI)` cho MongoDB URI.
- **Logs/Test:** Viết unit test trong `system_connectors_handler_test.go` kiểm chứng `UpdateConfig`, `Recover`, `validatePostgresConnectorPrereqs`, và `extractCredentialsAsOptions`.
- **Definition of Done:** `go test ./internal/api/source/...` PASS 100% qua `evidence_run.py`.

## Task 5: Hỗ trợ Resolve Secret khi Introspection (`introspection_handler*.go`) & Cập nhật `SourceConnectors.tsx` (`S5` — sau `APPROVE`)
- **Mô tả & Context:**
  - Files:
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler.go:15-195`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler_postgres.go:16-105`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go:330-335`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx:250-600`
- **Logs/Test:** Chạy `go test ./internal/api/system/...`, build `cdc-cms-service` + `cdc-cms-web` (`npm run build`), và verify runtime thật qua `evidence_run.py`.
- **Definition of Done:** Build + Test + Runtime verification PASS, hoàn tất `S5` + `DG3` + `audit_report_debezium_config_masked_password.md` + `verify_governance.py`.

## Task 6: Ràng buộc bổ sung `G1..G6` trong thực thi `S4 -> S5 -> DG3`
- **Mô tả & Context:**
  - `G1`: Tích hợp đọc `connection_registry.options_json` vào `GetByConnectorName` (`system_connector_repo_gorm.go:125-135`), lọc `!IsKeepOrMaskedSecret(v) && !IsMaskedURI(v)` khi merge `liveCfg` vào `baseline`, chặn Fail-Closed bằng `ValidateNoMaskedSecrets`, và tránh gọi trùng `GetConfig` qua `PreResolved: true`.
  - `G2`: Khóa điều kiện khớp host/port/user trong `DiscoverPostgresTablesPost`, `DiscoverMongoDatabasesPost`, `DiscoverMongoCollectionsPost` trước khi nạp secret lưu trữ theo `connector_name`.
  - `G3`: Hỗ trợ `fs.uris` đa URI phân tách bằng dấu phẩy, xử lý `len(curParts) == 1` trong `RestoreMaskedURI`, dùng `u.User.Username()` / `u.User.Password()` trong `extractCredentialsAsOptions`, áp dụng `IsMaskedURI` cho mọi key.
  - `G4`: Khép kín MongoDB Introspection và truyền `connector_name` + chuẩn hóa `values.password` trong `SourceConnectors.tsx`, `useConnectorCheck.ts`, `connectorCheck.ts`.
  - `G5`: Mask `server_address` trên response `GET /api/v1/sources` (`list_sources.go`) và `POST /api/v1/sources` (`sources_handler.go`).
  - `G6`: Tách `debezium_connector_secrets.go` + `debezium_connector_secrets_test.go` và chuyển `parseFingerprint`/`extractSFTPCollectionName`/`resolveEffectiveConnectorConfig` sang `system_connectors_credentials.go` để giữ `system_connectors_handler.go` dưới 600 dòng.
- **Logs/Test:** Table-driven unit tests cho `G1..G6`, đo `wc -l` tất cả các file chạm tới.
- **Definition of Done:** Toàn bộ unit test + build + runtime trace PASS, `wc -l < 600` cho mọi file backend.

## Task 7: Xử lý dứt điểm 4 điểm hậu kiểm `V1..V4` (`S5 -> S3 -> S4 -> S5`)
- **Mô tả & Context:**
  - `V1`: Giải thích nguyên nhân `laya_gate.py review` (Mode B) trả exit code `2` (do `list_sources.go` nhân bản hàm `maskURIUserInfo` dẫn tới `round_1: REJECT_REDUNDANT` và `system_connector_repo_gorm.go` có nhánh gán lặp), gom hàm dùng chung về `internal/model/source/source.go` (không vi phạm CQRS `queries -> commands`), rút gọn `system_connector_repo_gorm.go`, và chạy lại `laya_gate.py review` tới khi exit code `0` (`PASS`).
  - `V2`: Chạy `arch_check.py` và full `go test ./...` trên `cdc-cms-service` qua `evidence_run.py`; làm rõ cơ chế đánh số `L001..L095` theo vị trí dòng trong `build_lessons_index.py:222` vs `T1:L111` (line 111) và ghi bài học `L095` cuối file `lessons.md`.
  - `V3`: Khóa Host-Bound Secret Reuse (`SameConnectorEndpointIdentity` & `SameURIHostAuthority`) ngay trong `MergeConnectorConfigPreservingSecrets` và `BuildStoredConfigBaseline` (`debezium_connector_secrets.go:240-390`) để khi `UpdateConfig` / `Recover` đổi `database.hostname` / `port` / `user` hoặc đổi host authority trong URI mà vẫn gửi `__KEEP__` / `***` / `***:***` thì không nạp secret cũ và bị `ValidateNoMaskedSecrets` chặn ngay HTTP `400`.
  - `V4`: Build lại binary `cdc-cms-service` và khởi động lại tiến trình lắng nghe trên `:8083` (thay thế `PID 44455` cũ), sau đó chạy live smoke test qua `evidence_run.py --kind runtime` trên `GET /api/v1/sources` và `POST /api/introspection/postgres/tables`.
- **Logs/Test:** Biên lai `EV-...` cho `arch_check.py`, `go test ./...`, và live HTTP smoke tests trên `:8083`, cùng `06_laya_review_dg3_*.json` (`gate: PASS`, exit code `0`).
- **Definition of Done:** Tất cả 4 mục `V1..V4` hoàn tất 100% có biên lai `EV-...` và `verify_governance.py` đạt `PASSED` 🟢.
