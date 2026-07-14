# Task List - Bổ sung Thời gian Chữa lành Từng Loại Lỗi

- [ ] Thực hiện database migration bổ sung 3 cột timestamp.
- [ ] Cập nhật Go models `ReconciliationReport` trong centralized-data-service và cdc-cms-service.
- [ ] Cập nhật logic `finalizeReport` của transmuter worker để gán timestamp cho từng loại lỗi khi heal thành công.
- [ ] Cập nhật query `GetTableHistory` trong cdc-cms-service để trả về thêm 3 trường này.
- [ ] Cập nhật Frontend hook definitions và các cột hiển thị chi tiết trong `ExecuteHealModal.tsx`.
- [ ] Biên dịch và kiểm thử chất lượng toàn hệ thống.
