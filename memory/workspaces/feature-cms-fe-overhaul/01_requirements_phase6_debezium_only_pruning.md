# Requirements — Phase 6 Debezium-only Pruning

## Mục tiêu

- Tối giản CMS FE theo mục tiêu hiện tại:
  - Debezium-only
  - không dư thừa page
  - không dư thừa operation trong UI
  - không giữ các concept Airbyte / bridge như first-class

## Rule bắt buộc

Trước khi cắt UI/API dependency phải audit lại contract hiện hành để chắc:

1. feature đó thực sự không còn nằm trong luồng yêu cầu
2. FE không còn dùng nó như tuyến chính
3. việc cắt không làm gãy flow Debezium-only

## Phạm vi

- `src/App.tsx`
- `src/pages/ActivityManager.tsx`
- `src/pages/DataIntegrity.tsx`
- `src/pages/SystemHealth.tsx`
- `src/pages/MappingFieldsPage.tsx`
- `src/pages/ActivityLog.tsx`

## Điều kiện hoàn thành

1. `QueueMonitoring` không còn là page first-class trong navigation.
2. `bridge` / `airbyte-sync` không còn là operation UI chính.
3. `Airbyte` không còn được hiển thị như infrastructure/component chuẩn trong System Health.
4. Build FE pass.
