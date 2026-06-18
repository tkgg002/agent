# Workspace Context: Tái cấu trúc toàn bộ API Handlers sang chuẩn Hexagonal Architecture, CQRS, và Screaming Architecture

## Ngữ cảnh
Project `cdc-cms-service` (CMS điều phối CDC System) đang chứa các API Handlers cũ ở dạng "Fat Handlers". Các handlers này vi phạm nguyên tắc Clean Architecture khi trực tiếp gọi các thư viện hạ tầng như GORM (raw SQL, Transaction) và NATS Client (Publish, Request-Reply) trực tiếp trong tầng Delivery (HTTP/Fiber).

Trong các bước trước:
- Chúng ta đã chuyển các file API Handler thô vào các package con tương ứng theo Screaming Architecture: `governance`, `master`, `recon`, `scheduler`, `shadow`, `source`, `system`.
- Chúng ta đã tái cấu trúc thành công `master_mapping_rule_handler.go` và `mapping_preview_handler.go` (lập kế hoạch).

## Mục tiêu
1. Liệt kê toàn bộ các API Handlers trong các package con.
2. Kiểm tra xem các file nào đang là "Fat Handlers" (chứa GORM, NATS client, logic Use Case).
3. Thiết lập lộ trình bóc tách chi tiết (Implementation Plan).
4. Thực thi bóc tách các file này sang chuẩn Hexagonal Architecture, CQRS và Screaming Architecture theo nguyên lý "Inside-Out" (Domain Ports -> Application commands/queries -> Infrastructure adapters -> Slim API Handlers).
