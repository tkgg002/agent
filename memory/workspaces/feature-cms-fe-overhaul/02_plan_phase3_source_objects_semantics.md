# Plan — Phase 3 Source Objects Semantics

1. Rà backend `registry` và `master` để biết current contract còn chỗ nào transitional.
2. Thêm helper shadow namespace ở `TableRegistry`.
3. Đổi cột / text / action copy để row được hiểu là source object + shadow target.
4. Chuyển context từ `TableRegistry` sang `MasterRegistry` bằng query params giàu ngữ cảnh hơn.
5. Giữ payload submit tương thích backend legacy.
6. Build thật và ghi lại residual gaps.
