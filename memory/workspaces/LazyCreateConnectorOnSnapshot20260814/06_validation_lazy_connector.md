# Validation Report: Lazy SFTP Connector Creation On Snapshot

## Verification Summary (Quality Gate G1 - G8 Passed 🟢)

### Automated Unit Tests Result
1. **`cdc-cms-service` Unit Tests:**
   - Command: `go test ./internal/app/commands/source/... -v`
   - Result: **PASS 100%**
   - Verified Scenarios:
     - `TestCreateSystemConnectorHandler_SFTPLazyCreation`: Tạo Connector SFTP -> Không gọi Kafka Connect API `Create`, lưu DB với status `"configured"`.
     - `TestCreateSystemConnectorHandler_NonSFTPEagerCreation`: Tạo Connector MongoDB -> Gọi Kafka Connect API `Create` lập tức (Eager flow giữ nguyên 100%).

2. **`centralized-data-service` Unit Tests:**
   - Command: `go test ./internal/handler/orchestration/...`
   - Result: **PASS 100%** (`ok centralized-data-service/internal/handler/orchestration 1.145s`)
   - Verified Scenarios:
     - `SnapshotRunner` handles SFTP snapshot execution by checking Kafka Connect status and lazy creating connector dynamically when missing.

### Security Auto-Check Audit
- Parameterized SQL Queries: Dùng `WHERE connector_name = ?` ngăn ngừa SQL Injection.
- Sanitized Configs: Sử dụng `rawConfigSanitized` từ DB.
- Code Preservation: Code cũ của SFTP (`writer.Create` & `writer.Lifecycle`) được comment giữ lại đầy đủ dạng `// LEGACY SFTP... - PRESERVED`.
