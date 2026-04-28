# Solution — Phase 7 Backend API Pruning

## Chốt giải pháp

Phase này đi theo đúng rule:

1. audit usage trước
2. chỉ cắt route khi xác nhận FE/runtime chính không còn dùng
3. update swagger/comment ngay khi đổi API surface

## Kết quả

- Backend API surface gọn hơn.
- Docs không còn nói sai về route stale.
- FE đã được prune trước nên không bị gãy khi cắt router.

## Ý nghĩa

Đây là bước đầu tiên chuyển từ "FE gọn" sang "FE + API cùng gọn".
Muốn đạt mục tiêu không dư thừa API hoàn toàn, các phase sau phải tiếp tục dọn:

- `worker-schedule`
- `masters`
- `mapping-rules`
- các handler/file legacy vật lý chưa xóa
