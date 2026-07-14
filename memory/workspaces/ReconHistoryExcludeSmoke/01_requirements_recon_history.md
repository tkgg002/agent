# Requirements - Loại bỏ Smoke Check khỏi Lịch sử Chữa lành

Lọc bỏ toàn bộ các bản ghi đối soát nhanh (Smoke Check) khi truy vấn lịch sử chữa lành đối soát trên giao diện (tab Phiên đã xử lý), giải quyết tình trạng trôi bản ghi chữa lành thực sự (ví dụ ID 91).

## Yêu cầu chi tiết
1. **Database level query filter**: Thêm tham số `excludeSmoke` vào phương thức GetTableHistory của repository và query handler để chỉ select từ bảng `cdc_reconciliation_report` khi `excludeSmoke == true`.
2. **Backend controller**: Endpoint `GET /api/reconciliation/report/:table` nhận query parameter `exclude_smoke=true` và truyền xuống tầng query.
3. **Frontend API hook**: `useTableHistory` hỗ trợ thêm tham số `excludeSmoke` và truyền lên API.
4. **Modal UI**: `ExecuteHealModal.tsx` gọi history hook với `excludeSmoke=true` để lấy đúng lịch sử các phiên đối soát thực tế.
