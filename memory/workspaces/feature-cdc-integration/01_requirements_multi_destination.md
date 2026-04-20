# Requirements: Multi-Destination Support

> Date: 2026-04-14
> Phase: multi_destination
> Answer: Thêm destinations mới trong Airbyte (VD: MongoDB → BigQuery, MongoDB → S3...)

## Bối cảnh

Hiện tại: 1 Airbyte connection "MongoDb → Postgres" với 1 destinationId.
Mong muốn: Nhiều connections với nhiều destinations khác nhau. VD:
- MongoDb → Postgres (hiện tại)
- MongoDb → BigQuery
- MongoDb → S3
- MySQL → Postgres
- ...

## Yêu cầu

### R1: CMS phải quản lý được nhiều Airbyte destinations
- Hiển thị danh sách destinations từ Airbyte
- Mỗi destination: tên, loại (Postgres/BigQuery/S3...), status

### R2: CMS phải quản lý được nhiều connections
- Hiển thị danh sách connections từ Airbyte
- Mỗi connection: source → destination, schedule, streams count, status
- 1 source có thể map tới nhiều destinations

### R3: Registry hỗ trợ multi-destination
- Hiện tại registry chỉ có `airbyte_connection_id` — đã link tới 1 connection
- Cần: cùng 1 source stream có thể có nhiều registry entries (1 per destination)
- Hoặc: thêm `destination_id` / `destination_name` vào registry

### R4: FE hiển thị theo destination/connection
- Filter/group streams theo connection hoặc destination
- Nhìn được: stream X đang sync tới destinations nào

### R5: Activity Log phân biệt destination
- Log ghi rõ data từ connection/destination nào

### R6: Reconciliation scan tất cả connections
- Hiện tại scan 1 connection → cần scan tất cả connections từ tất cả workspaces/destinations
