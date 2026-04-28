# Phase 24 Solution - Registry Mutation Facade

## Quyết định chính

- Không đổi semantics backend của `Register/Update/BulkRegister` trong phase này.
- Chỉ đổi namespace FE-facing sang `source-objects` để UI không còn bám trực tiếp `/api/registry`.
- Legacy registry write-model vẫn là lớp delegate tạm thời cho tới khi có write-model V2 thật.

## Kết quả mong muốn

- FE runtime sạch hơn về mặt kiến trúc.
- Không thêm API dư thừa theo nghĩa business capability; chỉ tái-namespace 3 mutation hiện có.

## Kết quả thực tế

- `TableRegistry.tsx` đã chuyển hết 3 mutation chính sang namespace `/api/v1/source-objects`.
- FE runtime không còn gọi trực tiếp `/api/registry`.
- Backend vẫn dùng `RegistryHandler` làm delegate nên operator-flow không bị gãy.
- Backend tests và frontend build đều pass.
