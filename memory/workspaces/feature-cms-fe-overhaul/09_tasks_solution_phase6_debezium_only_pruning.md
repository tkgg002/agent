# Solution — Phase 6 Debezium-only Pruning

## Chốt giải pháp

Phase này không cố "làm đẹp" thêm, mà cắt đúng những gì đã lỗi thời với current operating model:

1. **Queue page**
   - không còn là page độc lập
   - dùng `SystemHealth` làm điểm nhìn chính

2. **Operations**
   - chỉ giữ những operation còn thuộc Debezium-only runtime

3. **Health / Integrity / Mapping / Activity**
   - bỏ cách kể chuyện Airbyte/bridge như những tuyến sống

## Tác động

- FE gọn hơn, đúng mục tiêu vận hành hiện tại hơn.
- Giảm cognitive load cho operator.
- Tạo baseline tốt trước khi dọn backend APIs legacy.
