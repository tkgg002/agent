# Context

**Project**: centralized-data-service
**Feature/Task**: Config Audit & Cleanup
**Description**: Audit/clean `config-local.yml` và `config.go`.
User chê layout `db:` "vớ vẩn" và `config.go` còn dead code. 
- Phase 3: rename `db:` → `dbPool:` trong yaml.
- Phase 2: xóa DEAD code trong `config.go`.

**Definition of Done**:
1. Đọc và phân tích `config.go` và `config-local.yml` (và cả production config nếu cần).
2. Lập plan chi tiết cho việc đổi `db:` thành `dbPool:` và dọn dẹp `config.go`.
3. Nhận sự phê duyệt từ user trước khi thực thi.
