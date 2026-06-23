# Context: Cataloging Internal Handlers

## Goal
Liệt kê danh sách từng tệp tin trong thư mục `internal/handler/` và các thư mục con của nó. Với mỗi tệp tin, liệt kê các handler được định nghĩa, và cung cấp mô tả ngắn về chức năng của từng handler. Cuối cùng, xuất danh sách này thành một báo cáo markdown.

## Active Directory
- [internal/handler](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler)

## Current Status
- Đã kiểm tra cấu trúc thư mục của `internal/handler/`. Thư mục này chứa 6 thư mục con: `base`, `master`, `orchestration`, `recon`, `shadow`, `source`.
- Cần quét tất cả các file Go trong các thư mục con này, tìm kiếm các cấu trúc handler, các phương thức xử lý (như các hàm handling lệnh NATS hoặc HTTP request), phân tích logic để đưa ra mô tả chức năng của từng handler.
