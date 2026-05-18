# Context

- Task: Dọn `flow1`, tập trung vào 3 tab thực dụng:
  - `sources` với chức năng `new connect` cho MongoDB / MySQL / PostgreSQL
  - edit/update config connector/source
  - `registry` đổi tên thành `shadow`
- Scope:
  - `cdc-system/cdc-cms-web`
  - `cdc-system/cdc-cms-service`
- Constraints:
  - Ẩn hoàn toàn UI `flow1`.
  - FE phải tự kiểm thử lại trên browser sau khi sửa.
  - Ưu tiên luồng operator thực tế thay vì wizard thử nghiệm.
