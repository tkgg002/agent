# Context: Refactor and Drainage of DB & NATS from API and App Layers

## 1. Mô tả
Trong kiến trúc Hexagonal (Ports and Adapters), các tầng bên trong (`internal/domain`, `internal/app`, `internal/api`) không nên phụ thuộc trực tiếp vào các adapter hạ tầng cụ thể như GORM (`*gorm.DB` hay `h.db`) hoặc NATS (`nats.Conn` hay client nats). Thay vào đó:
- Tầng `internal/api` và `internal/app` phải giao tiếp qua các Interfaces (Ports) định nghĩa ở tầng `internal/domain` hoặc `internal/app/ports`.
- Các implement thực tế sử dụng GORM (`*gorm.DB`) hoặc NATS client phải nằm hoàn toàn ở tầng `internal/infra/persistence/...` hoặc `internal/infra/messaging/...`.

## 2. Mục tiêu
- Loại bỏ toàn bộ sự xuất hiện trực tiếp của `h.db` (hoặc `*gorm.DB`) và client nats (hoặc `natsConn`) ra khỏi `internal/api` và `internal/app`.
- Di chuyển toàn bộ các logic truy vấn DB và gửi/nhận NATS message về tầng `internal/infra`.
- Đối chiếu, audit các câu SQL và logic NATS ở tầng `internal/infra` của workspace hiện tại (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`) so với codebase backup (`/Users/trainguyen/Documents/work/data-hub-bf/cdc-cms-service`) để đảm bảo không có sai lệch logic nghiệp vụ hoặc hành vi hệ thống.
