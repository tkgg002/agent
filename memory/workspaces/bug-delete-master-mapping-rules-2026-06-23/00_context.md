# Context: Cấu hình quy tắc xóa cho mapping_rule_master (delete master mapping rules)

## Overview
User yêu cầu cấu hình quy tắc xóa cho `mapping_rule_master` (master mapping rules) tại API `DELETE /api/v1/master-mapping-rules/:id`.
Quy tắc xóa cụ thể như sau:
1. **Field sync từ shadow — không xoá được**: Nếu rule được đồng bộ từ shadow (có `CreatedBy != nil && *CreatedBy == "shadow-sync"`), thì bình thường ngăn chặn không cho xóa (trả về lỗi `ErrCannotDeleteSync`).
2. **Ngoại lệ (chỉ field scan-flatten & admin thêm mới xoá được)**:
   - Nếu rule được sync từ shadow nhưng là field **scan-flatten** (tức là có `SourcePath` khác rỗng/NULL, ví dụ: `params.partnerId`) thì **VẪN CHO PHÉP XOÁ**.
   - Nếu rule do **admin thêm mới** (không phải sync từ shadow, tức là `CreatedBy != "shadow-sync"`) thì **VẪN CHO PHÉP XOÁ**.
3. **Thêm cái không cho xoá khi approve & đang có trong table master**:
   - Nếu rule có trạng thái đã duyệt (`Status == "approved"`) và thực tế đã có trong database master (`InMaster == true`), thì **NGĂN CHẶN KHÔNG CHO XÓA** (trả về lỗi mới `ErrCannotDeleteApprovedInMaster`).

## Key Goals
1. Đảm bảo handler `DeleteMasterRuleHandler` áp dụng chính xác các quy tắc trên.
2. Thêm định nghĩa lỗi `ErrCannotDeleteApprovedInMaster`.
3. Kiểm tra tính đúng đắn qua biên dịch dự án và chạy unit test (nếu có).
