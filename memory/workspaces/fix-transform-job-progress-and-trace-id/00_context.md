# 00_CONTEXT: TỐI ƯU PROGRESS %, TIẾN ĐỘ ROWS & TRACE ID CHO TRANSFORM JOB

## 1. Bối cảnh Hệ thống
- **Giao diện:** Trang `http://localhost:5173/shadow` (Shadow Registry & Bindings).
- **Thực thi:** Async batch transform chạy dưới worker `centralized-data-service`.
- **Điều phối & Quản lý:** `cdc-cms-service` (tạo job, cấp trace_id, lưu cdc_system.transform_jobs, cung cấp API polling).

## 2. Vấn đề Hiện tại
1. Khi chạy transform, UI hiển thị `Đang chạy 0% 0 rows` và không tính % tiến độ thực tế.
2. Không hiển thị `trace_id` để operator copy và trace trong SigNoz.
3. Khi F5 (tải lại trang), kết quả biến mất trở về `Chưa chạy` do SQL LATERAL JOIN bị trượt.
