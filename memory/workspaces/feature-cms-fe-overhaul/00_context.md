# Context

- Task: Lập kế hoạch cải tổ `cdc-cms-web` sau khi bootstrap thực chiến local CDC V2.
- Mục tiêu:
  - Xác định page nào còn dùng thực sự.
  - Sắp xếp lại thứ tự/flow theo vận hành thật thay vì menu legacy.
  - Chỉ ra page/chức năng không còn cần thiết hoặc nên gộp/bỏ.
- Scope repo:
  - `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web`
- Nguyên tắc:
  - Workspace chỉ dùng để audit và lập plan, chưa refactor code FE trong phase này.
  - Ưu tiên đọc route thật, page thật, API usage thật trước khi kết luận.

## Flow Model

- Hệ thống hiện có 2 luồng cần tách bạch khi audit/prune:
  - `auto-flow`:
    - luồng chính
    - Debezium-driven
    - chịu trách nhiệm ingest, shadow, transmute, master runtime
  - `cms-fe operator-flow`:
    - luồng phụ trợ nhưng bắt buộc phải giữ
    - phục vụ monitoring, backup operation, retry, reconcile, health, operator control
- Mọi quyết định cắt page hoặc cắt API từ đây phải trả lời rõ:
  - feature đó phục vụ `auto-flow` hay `cms-fe operator-flow`
  - nếu không phục vụ luồng nào hiện tại thì mới xem là dư thừa
