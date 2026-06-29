# Plan: Bug Sync Master Mapping Rules From Shadow 500 Error

### Root Cause Analysis (RCA)
- **Vấn đề**: API `/api/v1/master-mapping-rules/sync-from-shadow` bị lỗi 500 do vi phạm unique constraint `ux_mapping_rule_master_target` (trên bảng `mapping_rule_master`, các cột `master_binding_id, target_column`).
- **Nguyên nhân**: Khi thực hiện Phase 2a và 2b (update rename) trong repository `SyncRulesFromShadow`, câu lệnh join giữa `mapping_rule_master m` và `mapping_rule_v2 v2` qua `m.mapping_v2_id = v2.id` đã tìm thấy tất cả các cột con được flatten (ví dụ các cột con có `mapping_v2_id = 5` đại diện cho `params` như `params_partnerId`, `params_transId`,...). Bởi vì tên của các cột con này khác với tên của shadow field cha (`v2.target_column = 'params'`), câu lệnh UPDATE cố gắng đổi tên tất cả các cột con này thành `'params'`, dẫn đến việc trùng lặp tên cột và vi phạm ràng buộc unique.
- **Giải pháp**: Thêm điều kiện `AND (m.source_path IS NULL OR m.source_path = '')` vào câu lệnh UPDATE trong cả Phase 2a và Phase 2b để đảm bảo chỉ có cột direct đại diện chính thức (không có path con) mới được đổi tên khi shadow field tương ứng đổi tên.

## Proposed Steps

### Phase 1: Research & Root Cause Analysis (RCA)
- [x] Xác định service chứa route `/api/v1/master-mapping-rules/sync-from-shadow` (nhiều khả năng là `cdc-cms-service` chạy ở cổng 8083).
- [x] Dùng `grep_search` định vị handler của API này trong mã nguồn.
- [x] Đọc mã nguồn handler và các services/repositories liên quan để phân tích logic xử lý của API.
- [x] Kiểm tra logs hoạt động của service để thu thập thông tin về stack trace của lỗi 500 (file logs hoặc output terminal).

### Phase 2: Implementation & Fix
- [/] Thực hiện chỉnh sửa mã nguồn để khắc phục lỗi 500:
  - [/] Thêm điều kiện lọc `AND (m.source_path IS NULL OR m.source_path = '')` vào Phase 2a và Phase 2b trong `SyncRulesFromShadow` của [master_mapping_rule_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go).

### Phase 3: Compile & Verification
- [ ] Biên dịch lại các service bị ảnh hưởng để đảm bảo không lỗi cú pháp (`cdc-cms-service`).
- [ ] Chạy tests để kiểm tra độ ổn định.
- [ ] Sử dụng `curl` để gửi lại request và xác minh lỗi 500 biến mất, trả về trạng thái 200 OK.
