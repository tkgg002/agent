# Solution — Phase 3 Source Objects Semantics

## Chốt giải pháp

Phase này chọn chiến lược:

1. **Sửa ngữ nghĩa operator-facing trước**
   - hiển thị `shadow_<source_db>.<table>`
   - giải thích rõ row là source object

2. **Giữ payload backward-compatible**
   - backend hiện vẫn dùng `source_shadow` legacy
   - FE không cố submit format V2 khi server chưa nhận được

3. **Tăng context giữa các page**
   - `TableRegistry` không còn điều hướng mù sang `MasterRegistry`
   - page đích biết source object nào đang được thao tác

## Lợi ích

- Operator bớt hiểu sai về namespace shadow.
- Chuyển đổi dần sang V2 mà không tạo downtime hay regression API.
- Chuẩn bị tốt cho phase sau: refactor `MasterRegistry` backend/API thật sự.
