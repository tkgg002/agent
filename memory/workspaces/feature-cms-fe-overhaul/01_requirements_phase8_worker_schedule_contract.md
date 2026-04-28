# Requirements — Phase 8 Worker Schedule Contract

## Bối cảnh

- `cms-fe` hiện đã bớt page thừa, nhưng page `Operations` vẫn phải enrich dữ liệu schedule bằng cách gọi thêm `/api/registry`.
- `worker-schedule` là API thuộc `cms-fe operator-flow`, không phải `auto-flow`.
- User yêu cầu:
  - luôn audit API trước khi sửa
  - feature phải thực chiến, không chỉ là cái vỏ
  - không cắt nhầm các chức năng monitoring / backup / retry / reconcile của `cms-fe`

## Vấn đề

1. `GET /api/worker-schedule` hiện chỉ trả `target_table`, khiến FE phải tự đoán source/shadow scope.
2. `POST /api/worker-schedule` chỉ nhận payload legacy, không tự resolve scope từ metadata V2.
3. Swagger/comment cho API này còn thiếu, nên contract dễ lệch giữa FE và BE.

## Mục tiêu phase này

1. Làm giàu `worker-schedule` response bằng source/shadow metadata đủ cho operator.
2. Giữ compatibility với `target_table`, nhưng cho phép submit scope giàu hơn để resolve chính xác hơn.
3. Giảm logic “đắp nghĩa” trong `ActivityManager`.
4. Cập nhật swagger/comment ngay khi đổi API.
