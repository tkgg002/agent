# Kế hoạch Triển khai: Sửa lỗi Quyền NATS & Timeout khi Chữa lành (NATS Permissions Fix Plan)

Kế hoạch này hướng dẫn cách sửa đổi cấu hình NATS ACL để giải quyết vấn đề Permissions Violation đối với inbox reply.

---

## 1. Thay đổi đề xuất (Proposed Changes)

### Cấu hình: `nats-server.conf`
Đường dẫn: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/deployments/nats/nats-server.conf`

#### Thay đổi 1: Thêm `_INBOX.>` vào `publish` permissions của các user
Cần bổ sung `_INBOX.>` vào danh sách `publish` của `cdc_worker`, `cms_service` và `debezium`:
```conf
authorization {
  users = [
    {
      user: cdc_worker
      password: worker_secret_2026
      permissions: {
        publish: [
          "cdc.>",
          "schema.>",
          "$JS.API.>",
          "$JS.ACK.>",
          "_INBOX.>"
        ]
        subscribe: [
          "cdc.>",
          "schema.>",
          "$JS.API.>",
          "$JS.ACK.>",
          "_INBOX.>"
        ]
      }
    }
    {
      user: cms_service
      password: cms_secret_2026
      permissions: {
        publish: [
          "cdc.>",
          "schema.config.reload",
          "$JS.API.>",
          "_INBOX.>"
        ]
        subscribe: [
          "cdc.>",
          "schema.config.reload",
          "$JS.API.>",
          "_INBOX.>"
        ]
      }
    }
    {
      user: debezium
      password: debezium_secret_2026
      permissions: {
        publish: [
          "cdc.>",
          "$JS.API.>",
          "_INBOX.>"
        ]
        subscribe: [
          "cdc.>",
          "$JS.API.>",
          "_INBOX.>"
        ]
      }
    }
    ...
```

---

## 2. Kế hoạch Kiểm tra (Verification Plan)
- Chạy linter verify_governance.py.
- Khởi động lại dịch vụ NATS để reload config.
