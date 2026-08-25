# Báo cáo thay đổi chi tiết (Audit & Report) - Tích hợp SFTP Source Connector

Báo cáo ghi nhận chi tiết các tệp tin đã chỉnh sửa, số dòng thay đổi và tổng quan các khối chức năng được bổ sung trong đợt triển khai này.

## 1. Tổng quan tệp tin thay đổi

| Project / Service | File Path | Action | Lines Added/Modified | Description |
| :--- | :--- | :---: | :---: | :--- |
| **cdc-cms-service** | `internal/api/source/system_connectors_handler.go` | `MODIFY` | ~25 lines | Cập nhật hàm `parseFingerprint` và `extractCredentialsAsOptions` để hỗ trợ parse class `SftpSourceConnector`, trích xuất host, port, path, pattern và credentials `sftp.username`, `sftp.password`. |
| **centralized-data-service** | `internal/handler/shadow/sftp_adapter.go` | `NEW` | ~30 lines | Tạo cấu trúc `SFTPEventAdapter` và hàm `ConvertToCDCEvent` convert flat JSON thành `CDCEvent`. |
| **centralized-data-service** | `internal/handler/shadow/event_handler.go` | `MODIFY` | ~20 lines | Chỉnh sửa `HandleRaw` nhận diện topic `sftp.`, tự động convert qua adapter và trích xuất db, table cho sftp. |
| **centralized-data-service** | `internal/handler/shadow/sftp_adapter_test.go` | `NEW` | ~66 lines | Tạo bộ unit test phủ toàn bộ logic convert thành công và handle JSON lỗi của `SFTPEventAdapter`. |

---

## 2. Kết quả kiểm thử tự động (Verification)

Bộ unit test chạy thành công trong shadow handler:
```bash
go test -v ./internal/handler/shadow/...
```
**Kết quả Output:**
```text
=== RUN   TestSFTPEventAdapter_ConvertToCDCEvent
--- PASS: TestSFTPEventAdapter_ConvertToCDCEvent (0.00s)
=== RUN   TestSFTPEventAdapter_ConvertToCDCEvent_InvalidJSON
--- PASS: TestSFTPEventAdapter_ConvertToCDCEvent_InvalidJSON (0.00s)
PASS
ok  	centralized-data-service/internal/handler/shadow	0.823s
```
Tất cả các test case đều **PASS** 100%, bảo đảm không có regression cho các module shadow handler khác.
