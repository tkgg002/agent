# [Phase 1.5] CDC Mapping Visualization & Introspection

Hệ thống hiện tại (Phase 1) đã có Registry Table để theo dõi các bảng được sync (chủ yếu qua Airbyte), nhưng việc mapping các field từ Source vào Destination vẫn còn nằm trong "hộp đen" `_raw_data` (JSONB) và chưa có giao diện trực quan để quản lý. Phase 1.5 tập trung vào việc hiển thị, khám phá (discovery) và quản lý các mapping rules một cách dễ dàng trước khi tiến lên Phase 2 (Dynamic Mapping Engine hoàn chỉnh).

## Proposed Changes

---

### 1. Backend: `cdc-cms-service` (Go)

Hệ thống đã có API basic nhưng cần bổ sung các chức năng quản lý sâu.

#### [MODIFY] [mapping_rule_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/mapping_rule_handler.go)
- Bổ sung `Update` và `Delete` mapping rules (GORM).
- **Endpoint `POST /api/mapping-rules/approve`**: Phê duyệt các field mới từ `pending_fields`, kích hoạt column creation và trigger backfill qua NATS.

#### [NEW] [introspection_handler.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/introspection_handler.go)
- **Endpoint `GET /api/introspection/scan/:table`**: Sample dữ liệu từ `_raw_data`.
- Phân tích JSON keys để tìm các field "ẩn" trong backup nhưng chưa có mapping chính thức.

---

### 2. Frontend: `cdc-cms-web` (React)

Đây là phần quan trọng nhất để giải quyết yêu cầu "vẫn chưa rõ table đang đc map những field nào".

#### [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/TableRegistry.tsx)
- Thêm cơ chế **Expandable Row**: Khi click vào một bảng, gọi API `/api/mapping-rules?table=...` để hiển thị danh sách mapping hiện tại ngay lập tức.
- Hiển thị Badge "Drift Detected" nếu bảng có dữ liệu trong `_raw_data` nhưng mapping chưa đầy đủ.

#### [NEW] [MappingEditor.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/MappingEditor.tsx)
- Giao diện trực quan hiển thị song song: `Source Field` (trong `_raw_data`) <---> `Target Column`.
- Cho phép chỉnh sửa Mapping Rule và chọn "Sync/Backfill" để đổ dữ liệu cũ vào cột mới.

---

### 3. Service: `centralized-data-service` (Go Worker)

#### [MODIFY] [internal/service/dynamic_mapper.go](file:///Users/trainguyen/Documents/work/centralized-data-service/internal/service/dynamic_mapper.go)
- Triển khai logic **Backfill**: Quét `_raw_data` của các bản ghi hiện có và update các cột tương ứng dựa trên mapping mới approved.

## Verification Plan

### Automated Tests
- `go test -v ./internal/api/...` để kiểm tra các endpoint mới.
- Unit test cho logic `inferType` từ sample data.

### Manual Verification
1. Truy cập CMS Web -> Table Registry.
2. Chọn bảng `wallet_transactions` -> Nhấn "Manage Mapping".
3. Nhấn "Scan Source Schema" để xem các field đang có trong `_raw_data`.
4. Map thử 1 field mới và nhấn "Save".
5. Kiểm tra record mới sync về PostgreSQL đã có column tương ứng được populate (sau khi chạy migration).
