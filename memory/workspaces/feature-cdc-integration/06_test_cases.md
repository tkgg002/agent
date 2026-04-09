# Required Test Cases: Feature CDC Integration

Để hoàn thiện Phase 1, chúng ta cần verify 5 kịch bản sau:

### TC1: Airbyte Source Discovery (Unit/Integration)
- **Input**: Gọi lệnh `cdc.cmd.introspect` cho 1 bảng MongoDB.
- **Expect**: Worker gọi Airbyte API thành công, nhận được danh sách fields, so sánh với `mapping_rules` hiện tại và ghi đúng các field mới vào `pending_fields` với status `pending`.

### TC2: CMS Approval Transaction (API Test)
- **Input**: Gọi `POST /api/schema-changes/:id/approve` trên CMS.
- **Expect**:
    - Bảng đích trong Postgres được `ALTER TABLE ADD COLUMN`.
    - 1 bản ghi mới xuất hiện trong `mapping_rules`.
    - Message `schema.config.reload` được đẩy lên NATS.
    - Trigger lệnh Discover Schema ngược lại phía Airbyte thành công.

### TC3: Worker Live Reload (System Test) - **ĐANG THIẾU LOGIC**
- **Input**: Đẩy message `schema.config.reload` lên NATS.
- **Expect**: Worker nhận được message, xóa cache mapping cũ trong memory và nạp lại Mapping Rules mới từ DB mà không cần khởi động lại.

### TC4: Raw Data Extraction Flow (E2E Test)
- **Input**: Insert 1 bản ghi vào MongoDB có field mới (ví dụ: `discount_code`).
- **Expect**:
    - Airbyte sync bản ghi đó vào cột `_raw_data` của Postgres.
    - Sau khi Approve (TC2), Worker chạy scan và trích xuất thành công `discount_code` từ JSONB vào cột mới tạo.

### TC5: Error Handling & Rollback (Security/Robustness)
- **Input**: Cố tình Approve 1 field với kiểu dữ liệu sai (ví dụ: gán String cho cột INT).
- **Expect**: CMS báo lỗi, giao dịch `ALTER TABLE` rollback, trạng thái `pending_fields` quay về `pending` hoặc chuyển `failed`, `schema_change_logs` ghi nhận đúng lỗi.
