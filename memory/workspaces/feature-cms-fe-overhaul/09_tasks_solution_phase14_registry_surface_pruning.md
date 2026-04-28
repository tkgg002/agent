# Solution — Phase 14 Registry Surface Pruning

## Vấn đề

User muốn CMS FE và API surface “fix” hơn, không dư thừa. `/api/registry` là điểm dễ hiểu nhầm nhất vì nó vừa còn đang dùng thật, vừa chứa nhiều route dead/retired.

## Giải pháp

1. Audit caller thật trước khi cắt.
2. Chỉ prune các route không còn FE/runtime usage.
3. Giữ nguyên phần compatibility surface đang phục vụ operator-flow.
4. Đổi semantics swagger/comment của nhóm còn sống sang `Source Objects`.

## Outcome

- API surface gọn hơn mà không làm gãy FE.
- Không còn mount các route registry dead chỉ để trả `410`.
- `/api/registry` được định vị đúng hơn: transitional but still practical.
