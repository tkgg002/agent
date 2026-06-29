# Architectural & Technical Decisions

- **Decision 1: Export internal/private helpers**: Để test các private helper functions trong package `shadow` từ package `service_test`, chúng ta tạo một file `export_helper.go` không có hậu tố `_test` trong package `shadow` và export các hàm này.
- **Decision 2: Use regex SQL match in sqlmock for PostgreSQL compatibility**: PostgreSQL driver sử dụng placeholder `$1` thay cho `?`. Thay vì gán cứng regex placeholder, ta sử dụng regexp hỗ trợ cả `?` và `$1` như `(\?|\$1)`.
