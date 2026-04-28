# Solution — Phase 4 Mapping Rules Semantics

## Chốt giải pháp

Phase này tiếp tục chiến lược nhất quán:

1. **Nói đúng domain cho operator**
   - page không còn là "mapping fields quanh một target_table"
   - page được framing là rule layer cho source object và shadow target

2. **Không nói sai về backend**
   - FE hiển thị target architecture
   - nhưng luôn gắn note rằng current API vẫn transitional

3. **Chuẩn bị cho refactor backend/CMS phase sau**
   - khi CMS API đổi sang metadata V2 thật, UX hiện tại sẽ ít phải đổi lại

## Tác động

- Operator mental model tiến thêm một bước về V2.
- Blast radius nhỏ vì chưa thay contract network.
- Phase sau có thể đi tiếp vào `ActivityManager` / `DataIntegrity` mà không còn "lệch ngôn ngữ" với các page lõi.
