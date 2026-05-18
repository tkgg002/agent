# Requirements — Audit 3 Repo + Flow 1

## Mục tiêu

1. Xác nhận repo `api`, `fe`, `cdc-worker` hiện đang ở trạng thái nào:
   - working tree sạch hay bẩn
   - build/test pass hay fail
   - có dấu hiệu drift giữa code, docs, và runtime không
2. Kiểm tra Flow 1 trên CMS:
   - flow manual còn đi được hay không
   - blocker đang nằm ở CMS, FE, hay worker
   - các bước nào pass bằng chứng, bước nào chỉ mới "giả định"
3. Tạo kết luận ngắn gọn, có mức độ ưu tiên rõ ràng để user biết nên xử lý chỗ nào trước.

## Definition of Done

- Có bảng tóm tắt trạng thái cho cả 3 repo.
- Có kết luận riêng cho Flow 1 với root cause hoặc blocker hiện tại.
- Mọi kết luận chính đều có chứng cứ từ ít nhất một trong các nguồn:
  - `git status` / commit / branch
  - test/build output
  - log runtime
  - endpoint response
  - artifact/workspace report đã tồn tại

## Non-Goals

- Không refactor hoặc implement feature mới trong phiên audit này trừ khi phát hiện fix rất nhỏ và cần thiết.
- Không báo "xong" nếu chưa có verify tối thiểu trên từng repo/flow.
