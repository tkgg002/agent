# Context: Bug CDC Pipeline Issues 2026-06-23

## Problem Description
User báo cáo 5 vấn đề xảy ra trong hệ thống CDC:
1. **Bật/tắt Active (Debezium Sync)**: Khi bật/tắt Active cho source, hệ thống đang kích hoạt theo status của Source Actions. Yêu cầu: Bỏ kích hoạt kéo theo này.
2. **Sai thông tin Shadow Table ở UI `snapshot-monitor`**: Tại trang `http://localhost:5173/snapshot-monitor`, cột Shadow hiển thị sai thông tin, cụ thể là tên table lấy không đúng.
3. **Sensitive Mask Strategy không chạy**: Sensitive Mask Strategy không chạy khi thực hiện `snapshot` & `upstream`.
4. **transmute shadow -> master không chạy**: Sau khi thực hiện upstream, hệ thống không chạy transmute. 
5. **Chạy cmd-scan-fields trên shadow table rỗng**:
   - Log: `cmd-scan-fields centrallized-export-service.export-jobs shadow_cls_testing.export_jobs_5 error - - nats_command shadow table export_jobs_5 is empty`
   - User hỏi: "sao lại chạy cái này" (tại sao cmd-scan-fields lại chạy trên table trống/mới tạo khi chưa có dữ liệu, hoặc tại sao flow tự động kích hoạt scan-fields khi chưa sẵn sàng).

## Objectives
- Sửa đổi hành vi kích hoạt/deactive Debezium Sync để không tự động kéo theo các Source Actions khác.
- Sửa lỗi UI `/snapshot-monitor` hiển thị sai tên shadow table.
- Khắc phục lỗi Sensitive Mask Strategy không hoạt động trong các luồng snapshot & upstream.
- Sửa lỗi upstream không chạy sau khi config/upstream.
- Tìm hiểu lý do và sửa logic để không tự động hoặc ngăn chặn chạy `scan-fields` trên các shadow table rỗng khi chưa có dữ liệu.
