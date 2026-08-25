# 01_requirements_sftp_internal_worker.md — Yêu cầu Chi tiết

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## Yêu cầu Nghiệp vụ (Business Requirements)

| ID | Yêu cầu |
|:---|:---|
| BR-01 | Khi file CSV mới được upload lên SFTP server (Docker, port 2022), hệ thống phải tự động phát hiện và đọc file. |
| BR-02 | Mỗi row trong file CSV phải được chuyển thành một event trong shadow DB (bảng shadow tương ứng với `reconcile_final`). |
| BR-03 | Sau khi đọc xong, file CSV phải được move sang thư mục `processed/` để tránh đọc lại. |
| BR-04 | Nếu lỗi khi xử lý file, file phải được move sang thư mục `error/`. |
| BR-05 | Dữ liệu sau khi vào shadow DB phải được transmute sang Master DB theo đúng pipeline hiện có. |

---

## Yêu cầu Kỹ thuật (Technical Requirements)

| ID | Yêu cầu |
|:---|:---|
| TR-01 | `SFTPPollingWorker` phải chạy như một goroutine nội bộ trong `centralized-data-service`, không phụ thuộc Kafka Connect plugin nào. |
| TR-02 | Worker poll SFTP server định kỳ theo `pollInterval` (default 30s), configurable qua `config.yml`. |
| TR-03 | Parse CSV: dòng đầu là header, mỗi dòng còn lại là 1 record. Convert thành flat JSON `map[string]string`. |
| TR-04 | Mỗi row flat JSON được push vào Kafka topic `sftp.reconcile.final` (hoặc `{topicPrefix}`) dưới dạng message value. |
| TR-05 | Sử dụng thư viện `github.com/pkg/sftp` (SSH-based) để kết nối SFTP. Thư viện `golang.org/x/crypto/ssh` làm SSH transport. |
| TR-06 | Kafka Producer sử dụng `github.com/segmentio/kafka-go` (đã có sẵn trong go.mod). |
| TR-07 | Worker hỗ trợ graceful shutdown qua context cancellation. |
| TR-08 | `SFTPWorkerConfig` được expose trong `config.go` với fields: `Enabled`, `Host`, `Port`, `Username`, `Password`, `InputPath`, `FilePattern` (regex), `ProcessedPath`, `ErrorPath`, `TopicPrefix`, `PollInterval`. |
| TR-09 | FilePattern dùng `regexp.MatchString` để lọc file cần đọc (ví dụ: `^reconcile_final_.*\.csv$`). |
| TR-10 | Logging đầy đủ: file detected, rows pushed, file moved, errors. |
| TR-11 | Đăng ký vào WorkerServer thông qua `RegisterOnStart` và `RegisterOnStop`. |

---

## Definition of Done (DoD)

- [ ] `sftp_worker.go` được tạo và compile thành công.
- [ ] `config.go` có `SFTPWorkerConfig`, `config-local.yml` có section `sftpWorker`.
- [ ] `server_setup.go` wire SFTPPollingWorker vào WorkerServer.
- [ ] Unit test `sftp_worker_test.go` pass.
- [ ] End-to-end test: upload CSV → shadow DB có data → Master DB sau transmute.
- [ ] `go build ./...` pass (0 errors).
