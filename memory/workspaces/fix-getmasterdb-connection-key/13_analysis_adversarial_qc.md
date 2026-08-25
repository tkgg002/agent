# 13_analysis_adversarial_qc.md — Phân tích chi tiết Adversarial QC

## 1. Bản chất sự cố chuỗi nối NULL trong PostgreSQL
Toán tử `||` trong PostgreSQL là toán tử nghiêm ngặt (strict operator) theo chuẩn SQL. Nếu một trong hai biểu thức là `NULL`, toàn bộ biểu thức trả về `NULL`.
- Ví dụ: `'public' || '.' || 'table'` -> `'public.table'` (ĐÚNG)
- Nhưng: `NULL || '.' || 'table'` -> `NULL` (SAI - Mất toàn bộ tên bảng)
Do đó, việc dùng `COALESCE(NULLIF(col, ''), 'public')` là yêu cầu bắt buộc khi định danh FQN.

## 2. Bản chất sự cố Zero-Value trong Golang Struct & HTTP Unmarshaling
Khi thêm một field mới `MasterSchema` vào Domain/Command struct:
- Nếu HTTP Controller chưa có field tương ứng trong DTO `ScheduleCreateRequest`, `json.Unmarshal` sẽ bỏ qua.
- Giá trị `MasterSchema` trong Command sẽ luôn là zero value `""`.
- Tầng Persistence nhận `""` và query `WHERE master_schema = ''` thay vì schema thực tế -> dẫn đến bug nghiệp vụ.

## 3. Bản chất cơ chế Checkpoint & Triết lý Core Systems
- `TransmuterModule` sử dụng `sync_runtime_state` để lưu `last_gpay_id` theo từng batch phục vụ resume.
- Khi hoàn tất full sync, hệ thống tự động gọi:
  ```go
  t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
      item.LastCursorJSON = []byte(`{}`)
  })
  ```
- Việc cố tình xóa thủ công checkpoint trong database vừa vi phạm Rule #12 vừa không cần thiết, vì engine sẽ tự ghi đè khi full sync xong.
