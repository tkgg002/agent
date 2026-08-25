# 00_context_sftp_internal_worker.md — Phạm vi & Thành phần

## Workspace
**Tên:** FeatureSftpCdcIntegration  
**Phase:** SFTP Internal Polling Worker (Hướng B)  
**Ngày tạo:** 2026-08-11  

---

## Bối cảnh

Sau khi phân tích kiến trúc, phát hiện ra luồng SFTP → Kafka bị ĐỨT ĐOẠN do plugin `io.confluent.connect.sftp.SftpSourceConnector` KHÔNG tồn tại trong hạ tầng Kafka Connect của hệ thống. Toàn bộ code phía consumer (sftp_adapter.go, event_handler.go, topic_helper.go) đã đúng nhưng thiếu publisher đầu vào.

Giải pháp: Triển khai `SFTPPollingWorker` — goroutine nội bộ trong `centralized-data-service`, tự động poll SFTP server → parse CSV → push Kafka topic → pipeline tiếp tục như bình thường.

---

## Scope (In-scope)

- Implement `SFTPPollingWorker` trong package `internal/handler/shadow/`.
- Thêm `SFTPWorkerConfig` vào `config/config.go`.
- Cập nhật `config/config-local.yml` với thông số SFTP Docker Host.
- Wire vào `internal/server/server_setup.go`.
- Unit test `sftp_worker_test.go`.

## Out-of-scope

- Kafka Connect SFTP plugin (Confluent, thương mại).
- n8n integration (sẽ làm sau).
- Multi-SFTP source (chỉ một source: reconcile_final).

---

## Thành phần liên quan

| Service | File chính | Vai trò |
|:---|:---|:---|
| `centralized-data-service` | `internal/handler/shadow/sftp_worker.go` | [NEW] SFTP Polling goroutine |
| `centralized-data-service` | `config/config.go` | [MODIFY] Thêm SFTPWorkerConfig |
| `centralized-data-service` | `config/config-local.yml` | [MODIFY] Thêm sftpWorker config block |
| `centralized-data-service` | `internal/server/server_setup.go` | [MODIFY] Wire SFTPPollingWorker |
| `data-hub/docker` | `docker-compose.yml` | [EXIST] SFTP Host Docker container (port 2022) |
