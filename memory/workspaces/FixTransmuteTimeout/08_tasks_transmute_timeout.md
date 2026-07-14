# Kế hoạch Thực thi chi tiết (Checklist)

- [x] Tối ưu hóa truy vấn incremental/heal trong `transmuter.go` (bỏ `_gpay_id` filter và `ORDER BY`).
- [x] Thêm logic tự động tạo index `CONCURRENTLY` cho `_source_id` trên shadow DB dưới nền.
- [x] Thiết lập cơ chế lưu trữ và nạp checkpoint `lastGpayID` vào `LastCursorJSON` trong `Run`.
- [x] Tách biệt tiến trình transmuter trong `HandleTransmute` sang background goroutine với timeout linh hoạt (30m/24h).
- [x] Bảo toàn OTel tracing spans bằng cách quản lý span con đúng cách trong goroutine.
- [x] Biên dịch và chạy thử kiểm thử đơn vị liên quan đến transmuter.
- [x] Cập nhật kết quả xác minh vào tài liệu và hoàn thành báo cáo.
- [x] Tối ưu hook `useTableHistory` để hỗ trợ `pageSize` tùy chỉnh (mặc định 30) phục vụ lấy nhiều bản ghi lịch sử hơn.
- [x] Thêm tab "Phiên đối soát đã xử lý" hiển thị danh sách các phiên có `healed_at != null` vào `ExecuteHealModal.tsx`.
- [x] Biên dịch thành công Frontend (`npm run build` pass).
