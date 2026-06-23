# Context: Refactor dọn dẹp các file quá dài và không rõ ràng trong centralized-data-service

## 1. Bối cảnh
- Dự án `centralized-data-service` (nằm tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`) là một Go service chịu trách nhiệm đồng bộ, xử lý và tích hợp dữ liệu (CDC).
- Sau các đợt refactor trước, kiến trúc của dự án đã chuyển dần sang chuẩn Hexagonal / CQRS / layered architecture.
- Tuy nhiên, User phản hồi rằng vẫn còn rất nhiều file không rõ ràng, chứa quá nhiều dòng code (large files/god files) gây khó khăn cho việc bảo trì và đọc hiểu.

## 2. Mục tiêu
- Rà soát các file Go trong `centralized-data-service` để tìm ra các file có kích thước quá lớn (ví dụ: > 300-500 dòng) hoặc cấu trúc chưa rõ ràng, ôm đồm quá nhiều trách nhiệm (violating Single Responsibility Principle).
- Thực hiện phân tách, tái cấu trúc (refactoring) các file này thành các file nhỏ hơn, tường minh hơn theo đúng chuẩn kiến trúc của dự án (Handler, Service, Repository, Model, CQRS).
- Đảm bảo không làm thay đổi hay phá vỡ logic nghiệp vụ hiện tại.
- Đảm bảo toàn bộ hệ thống biên dịch thành công và vượt qua các bài kiểm thử (Go test).
