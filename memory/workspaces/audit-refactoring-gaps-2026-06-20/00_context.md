# Context: Rà soát & Audit toàn bộ sự khác biệt logic sau Refactor của centralized-data-service

## 1. Bối cảnh
- Dự án `centralized-data-service` đã được refactor từ cấu trúc cũ sang cấu trúc mới (Horizontal Layer-first / Hexagonal).
- Việc refactor này có khả năng làm sai lệch hoặc mất mát các logic nghiệp vụ quan trọng trong quá trình xử lý dữ liệu và cấu trúc DB.
- Chúng ta cần thực hiện đối chiếu trực tiếp các cấu phần logic nghiệp vụ của `centralized-data-service` giữa:
  - **Bản Gốc (Backup)**: Nằm tại `/Users/trainguyen/Documents/work/data-hub-bf/centralized-data-service`
  - **Bản Hiện tại (Refactored)**: Nằm tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`

## 2. Mục tiêu
- Đối chiếu chi tiết logic nghiệp vụ của các chức năng chính trong `centralized-data-service` để tìm ra tất cả sai phạm/sai lệch logic do refactor.
- Khôi phục chính xác logic gốc trong cấu trúc layer mới.
