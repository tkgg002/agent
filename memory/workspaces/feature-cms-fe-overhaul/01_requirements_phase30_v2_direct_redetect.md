# Phase 30 Requirements — V2 Direct Re-detect

## Mục tiêu
- Giảm phụ thuộc `registry_id` cho operator action `detect-timestamp-field`.
- Giữ đúng mô hình 2 luồng:
  - auto-flow: Debezium runtime là luồng chính
  - cms-fe operator-flow: monitoring / retry / reconcile / re-detect

## Yêu cầu thực chiến
- API phải được audit trước khi sửa FE.
- Không được giả vờ `create-default-columns` hay `scan-fields` đã V2-native nếu worker path chưa đủ dữ liệu.
- Nếu action có thể resolve bằng `source_object_id + active shadow binding`, thì ưu tiên direct V2 route.
- Swagger annotations phải được cập nhật cùng phase.

## Kết luận audit đầu vào
- `detect-timestamp-field` có thể direct V2 vì worker hiện đã support payload fallback theo `target_table`.
- `create-default-columns` chưa direct V2 an toàn vì worker path còn cần metadata bridge/legacy đầy đủ.
