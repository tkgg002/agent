# Solution — Phase 13 Legacy Artifact Purge

## Vấn đề

Sau nhiều phase refactor, route/menu/API legacy đã bị gỡ khỏi runtime, nhưng repo vẫn còn giữ các file artifact cũ. Nếu để lại, chúng làm codebase khó đọc và dễ gây hiểu sai về operating model hiện tại.

## Giải pháp

1. Audit usage trước khi xóa.
2. Chỉ purge khi xác nhận:
   - không còn route/menu
   - không còn server wiring
   - không còn call site sống
3. Xóa đồng bộ cả FE và backend để tránh lệch một bên.

## Outcome

- Giảm nhiễu kiến trúc trong repo.
- Không còn artifact trực tiếp gắn với `cdc_internal` runtime cũ.
- Build/test tiếp tục pass sau khi xóa.
