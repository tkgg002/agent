# Checklist: Tái cấu trúc toàn bộ API Handlers sang chuẩn Hexagonal/CQRS/Screaming

- [x] **Phase 1: Phân tích & Lập Kế Hoạch (Research & Design)**
  - [x] Khởi tạo Workspace, context và progress log.
  - [x] Liệt kê và phân loại các file API Handlers cũ theo package con.
  - [x] Viết thiết kế cấu trúc lớp (Ports, Use Cases, Adapters) chi tiết cho từng nhóm trong `implementation_plan.md`.
  - [x] Nhận phản hồi và sự phê duyệt từ người dùng.

- [x] **Phase 2: Refactor nhóm `shadow` (Mẫu điển hình)**
  - [x] Rút trích Domain Ports cho `shadow`.
  - [x] Tạo các Command/Query handlers cho `shadow`.
  - [x] Implement DB/NATS Adapters cho `shadow`.
  - [x] Slim down các API handlers của `shadow`.
  - [x] Biên dịch và kiểm thử nhóm `shadow`.

- [x] **Phase 3: Refactor nhóm `master` và `governance`**
  - [x] Rút trích Domain Ports cho `master`.
  - [x] Tạo các Command/Query handlers cho `master`.
  - [x] Implement DB/NATS Adapters cho `master`.
  - [x] Slim down API handler `MasterMappingRuleHandler` của `master`.
  - [x] Biên dịch và kiểm thử nhóm `master` thành công.
  - [x] Rút trích Domain Ports cho `governance`.
  - [x] Tạo các Command/Query handlers cho `governance`.
  - [x] Implement DB/NATS Adapters cho `governance`.
  - [x] Slim down các API handlers của `governance`.
  - [x] Biên dịch và kiểm thử nhóm `governance`.

- [x] **Phase 4: Refactor nhóm `source` và `scheduler`**
  - [x] Rút trích Domain Ports cho `source` & `scheduler`.
  - [x] Tạo các Command/Query handlers cho `source` & `scheduler`.
  - [x] Implement DB/NATS Adapters cho `source` & `scheduler`.
  - [x] Slim down các API handlers của `source` & `scheduler`.
  - [x] Biên dịch và kiểm thử nhóm `source` & `scheduler`.

- [x] **Phase 5: Refactor nhóm `recon` và `system`**
  - [x] Rút trích Domain Ports cho `recon` & `system`.
  - [x] Tạo các Command/Query handlers cho `recon` & `system`.
  - [x] Implement DB/NATS Adapters cho `recon` & `system`.
  - [x] Slim down các API handlers của `recon` & `system`.
  - [x] Biên dịch và kiểm thử nhóm `recon` & `system`.

- [x] **Phase 6: Kiểm thử tổng thể & Nghiệm thu (Verify & Wrap-up)**
  - [x] Biên dịch toàn bộ codebase (`go build ./...`).
  - [x] Chạy toàn bộ integration & unit test suite (`go test ./...`).
  - [x] Tạo tài liệu walkthrough.md nghiệm thu.
