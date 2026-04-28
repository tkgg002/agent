# Phase 28 Requirements - V2 Status Visibility

## Mục tiêu

- Cho operator nhìn rõ row nào đã sync V2 đầy đủ.
- Giảm mơ hồ của cảnh báo `legacy bridge only`.

## Yêu cầu

1. Read model `source objects` phải trả thêm trạng thái metadata/bridge.
2. UI `Source Objects` phải hiển thị:
   - `V2 Ready`
   - `Shadow Bound`
   - `Source Only`
   - `Bridge OK` / `No Bridge`
3. Copy cảnh báo phải phản ánh đúng thực tế mới sau Phase 27.
