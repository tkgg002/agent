# Requirements — Phase 7 Cutover Readiness

## Scope

- Chốt khả năng cutover sau chuỗi refactor V2.
- Ghi checklist vận hành để wipe dữ liệu và bootstrap lại hệ thống theo metadata V2.
- Đánh dấu các dependency legacy còn lại nhưng không còn là blocker trực tiếp cho luồng chính.

## Required Outcomes

1. Runtime lookup của các path chính phải ưu tiên metadata V2.
2. Có checklist rõ ràng cho:
   - migrate
   - seed V2 metadata
   - wipe data
   - restart services
   - verify luồng bootstrap
3. Có gap list rõ phần nào còn legacy nhưng không chặn cutover đầu tiên.
