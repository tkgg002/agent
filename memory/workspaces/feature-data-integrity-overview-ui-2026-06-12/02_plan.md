# Kế hoạch thực hiện Tách cột Source / Shadow thành 3 cột Source, Shadow, Master

## Các bước thực hiện
1. **Phân tích dữ liệu & Mapping logic**:
   - Sử dụng danh sách báo cáo `reportList` (là mảng các `ReconReport`) để ánh xạ thông tin chéo giữa các segment A (`source_shadow`) và segment B (`shadow_master`) nhằm khôi phục đầy đủ mối quan hệ Source -> Shadow -> Master cho mỗi dòng riêng biệt.
   - Định nghĩa hàm helper `resolvePipelineNames(record, allReports)` để trả về FQN của `sourceFqn`, `sourceConnector`, `shadowFqn` và `masterFqn` tương ứng.

2. **Chỉnh sửa cột trong `reportColumns` của `DataIntegrity.tsx`**:
   - Xóa cột `Source / Shadow` hiện tại.
   - Thêm 3 cột mới:
     - **Source**: Dùng `resolvePipelineNames` để lấy `tbl` và `db` nguồn, render dạng text/sub-text, kèm `sourceConnector` tag nếu có.
     - **Shadow**: Dùng `resolvePipelineNames` để lấy `tbl` và `db` shadow, render dạng code block, kèm `Ambiguous` tag nếu `record.scope_ambiguous` là true.
     - **Master**: Dùng `resolvePipelineNames` để lấy `tbl` và `db` master, render dạng code block. Nếu chưa map thì hiển thị `—`.
   
3. **Kiểm tra và xác thực**:
   - Khởi chạy frontend server để xem giao diện tab "Tổng quan".
   - Kiểm tra xem 3 cột mới hiển thị đúng vị trí, thông tin đầy đủ, căn lề và style hài hòa, premium.
