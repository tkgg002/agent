# Yêu cầu Chi tiết: Sửa lỗi Quyền NATS (Permissions Violation) & Timeout khi Chữa lành

## 1. Bối cảnh
Khi thực hiện hành động Chữa lành đối soát (Heal), CDC Worker gửi yêu cầu `Request` qua NATS tới topic `cdc.cmd.transmute`. Khi xử lý xong, CDC Worker (trong vai trò Responder) thực hiện gửi phản hồi (Reply) về inbox của yêu cầu (chứa subject dạng `_INBOX.xxx`). 
Tuy nhiên, cấu hình ACL của NATS (`nats-server.conf`) hiện tại không cho phép user `cdc_worker` (và các user khác) được `publish` tới các topic `_INBOX.>`. Điều này dẫn tới lỗi:
`nats: permissions violation: Permissions Violation for Publish to "_INBOX.xxx"`
Và hệ quả là phía yêu cầu bị timeout (`nats: timeout`) do không nhận được phản hồi.

## 2. Phạm vi điều chỉnh (Scope)
- **Tệp điều chỉnh:** `deployments/nats/nats-server.conf`
- **Logic điều chỉnh:** Thêm `_INBOX.>` vào danh sách quyền `publish` cho các user:
  - `cdc_worker`
  - `cms_service`
  - `debezium`
- **Xác minh:** Restart service nats hoặc chạy lại docker-compose để áp dụng cấu hình mới.
