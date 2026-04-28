# Requirements — Phase 16 Sources Connectors V2 Screen

## Mục tiêu

- Bắt đầu dựng màn V2-native đầu tiên cho CMS FE mà không cần mở thêm API mới.
- Nâng `Sources & Connectors` từ màn chỉ nhìn Kafka Connect runtime thành màn nhìn được cả:
  - Debezium connector runtime
  - source fingerprint đã persist
  - độ lệch giữa hai lớp này

## Yêu cầu

1. Chỉ dùng API hiện có nếu chúng đã đủ:
   - `GET /api/v1/system/connectors`
   - `GET /api/v1/sources`
2. Không làm mất các thao tác runtime hiện có:
   - create connector
   - restart/pause/resume
   - restart task
   - delete connector
3. Màn mới phải giúp operator nhìn ra mismatch:
   - connector runtime có nhưng fingerprint chưa có
   - fingerprint còn nhưng connector runtime đã mất
