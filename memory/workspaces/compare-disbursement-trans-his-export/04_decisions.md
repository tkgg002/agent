# Decisions: Compare Disbursement Trans His Export

## 04_decisions.md - Các quyết định

### [2026-02-26] Tách biệt Workspace
- **Quyết định**: Tạo workspace riêng `compare-disbursement-trans-his-export`.
- **Lý do**: Tính năng Export Trans History và Ticket Export tuy nằm cùng module Disbursement nhưng có logic mapping và filter khác nhau. Việc gộp chung dẫn đến "pollution" context và gây nhầm lẫn.
- **Hệ quả**: Đảm bảo lịch sử và logic của từng feature được bảo toàn riêng biệt.
