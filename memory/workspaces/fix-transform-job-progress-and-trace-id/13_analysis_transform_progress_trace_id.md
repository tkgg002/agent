# 13_ANALYSIS: PHÂN TÍCH GỐC RỄ VÀ GIẢI PHÁP KỸ THUẬT

## 1. Bối cảnh & Phân tích Hiện trạng
- Người dùng thực hiện thao tác Transform trên `/shadow` và Transmute trên `/masters`.
- **Hiện tượng 1:** Khi job đang chạy, tiến độ hiển thị `0%` và `0 rows` (hoặc chỉ tăng số rows mà không có tổng số và không tăng %).
- **Hiện tượng 2:** Khi F5 lại trang sau khi job hoàn thành, trạng thái transform/transmute bị mất hoặc trở về "Chưa chạy".
- **Hiện tượng 3:** Không có SigNoz Trace ID để trace log trên hệ thống quan sát SigNoz.

---

## 2. Root Cause Analysis (5 Whys)
1. **Tại sao tiến độ % luôn bằng 0?**
   - Trong `BatchTransformHandler` và `TransmuterModule`, worker không đếm trước tổng số bản ghi cần xử lý (`totalPendingRows`/`totalShadowRows`) nên không có mẫu số để tính tỉ lệ `progress_percent`.
   - Trong `batch_transform_handler.go`, biến `heartbeatEvery` được đặt là 50 (tương đương 50k - 500k rows), khiến các job có kích thước dưới 50k rows không bao giờ trigger `UpdateProgress` trong quá trình chạy.
2. **Tại sao F5 lại mất trạng thái?**
   - Trong `source_object_read_repo_gorm.go` và `master_read_repo_gorm.go`, LATERAL join nối với `cdc_system.transform_jobs` / `cdc_system.transmute_jobs` chỉ so khớp bằng `target_table` đơn thuần. Khi target table lưu dạng FQN có schema (`shadow_bidv_connector_service.bank_requests`) hoặc `source_object_id` được ghi nhận, mệnh đề WHERE bị trượt dẫn đến kết quả NULL.
3. **Tại sao thiếu Trace ID?**
   - Bảng `transform_jobs` và `transmute_jobs` có trường `trace_id`, tuy nhiên:
     - Handler `TransformV2` không truyền `trace_id` vào NATS message `cdc.cmd.batch-transform`.
     - Endpoints `TransformJobStatusV2` và `TransmuteJobStatus` không đưa `trace_id` và `total_rows` vào JSON response map.
     - Frontend DTO và component không có code render icon copy cho Trace ID.

---

## 3. Giải pháp Kiến trúc & Thiết kế Thực thi
1. **DB Column Alignment:** Bổ sung `total_rows BIGINT` vào cả 2 bảng tracking job để lưu trữ tổng số bản ghi cần xử lý.
2. **Pre-flight Count & Chunk Progress:**
   - Đếm trước số lượng dòng cần xử lý trước khi vào loop chạy.
   - Cập nhật `progress_percent = (rows_affected / total_rows) * 100` ngay sau mỗi chunk/batch hoàn thành.
3. **Trace ID Propagation & Compact UI:**
   - Sinh trace ID (hoặc lấy từ OpenTelemetry SpanContext) và truyền qua NATS payload.
   - UI chỉ hiển thị 1 icon copy Ant Design `<CopyOutlined />` nhỏ gọn với Tooltip `SigNoz Trace ID: ... (Click để copy)`, click vào sẽ sao chép vào clipboard và hiện toast thông báo.
4. **FQN-Safe LATERAL Join:**
   - Cập nhật SQL LATERAL join để so khớp đa tầng: theo `source_object_id`, theo `schema.table`, theo `physical_table_fqn`, và theo `table`.
