# Plan: Multi-Destination Support

> Date: 2026-04-14
> Phase: multi_destination

## Hiện trạng
- Code `ListConnections` đã lấy TẤT CẢ connections (không hardcode 1 destination)
- Registry có `airbyte_connection_id` — phân biệt connection
- Reconciliation scan tất cả connections
- **Gần sẵn sàng** — chỉ cần bổ sung metadata + FE

## Tasks

### T1: Model — thêm destination metadata vào registry
- `airbyte_destination_id VARCHAR`
- `airbyte_destination_name VARCHAR` (VD: "Postgres", "BigQuery")

### T2: Reconciliation — populate destination metadata khi auto-register
- Lấy destination info từ connection → lưu vào registry

### T3: CMS API — endpoint destinations
- `GET /api/airbyte/destinations` — đã có, verify
- Registry list: thêm filter theo destination

### T4: FE — hiển thị destination
- Registry page: hiện cột destination name
- Filter theo destination
- Group hoặc tag theo destination

### T5: Activity Log — thêm destination info

### T6: Build + Swagger + Test
