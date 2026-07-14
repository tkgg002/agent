# Phân tích Kỹ thuật - Loại bỏ handlePrune & Tái cấu trúc Routing Segment

## 1. Hiện trạng & Bối cảnh
Trước khi refactor, luồng điều phối của Reconciliation Check Handler (`recon_check_handler.go`) gặp một số vấn đề:
- Tồn tại phương thức legacy `handlePrune` để dọn dẹp các bản ghi mồ côi (orphan prune), tuy nhiên logic này không còn được kích hoạt qua luồng check chính hoặc đã được chuyển giao cho các job nền chuyên dụng.
- Cơ chế phân phối (routing) chỉ hỗ trợ rẽ nhánh đơn giản giữa Segment A (`source_shadow`) và Segment B (`shadow_master`), chưa hỗ trợ tùy chọn `"both"` (chạy cả hai phân đoạn liên tiếp) và chưa xử lý tự động khi trường `segment` bị bỏ trống.
- Các hàm thực thi kiểm tra cho Segment A và B bị gộp chung hoặc phụ thuộc lẫn nhau, thiếu cấu trúc module hóa rõ ràng.

## 2. Giải pháp Kỹ thuật
Chúng tôi đã áp dụng cấu trúc định tuyến phân đoạn hợp nhất (Unified Segment-Based Routing):
- **Phân tách Rõ ràng**: Tách biệt luồng xử lý Segment A (`executeCheckSegmentA`) và Segment B (`executeCheckSegmentB`).
- **Hỗ trợ "both"**: Khi `payload.Segment` là `"both"` (hoặc trống), handler sẽ chạy cả hai và trả về kết quả tổng hợp:
  ```json
  {
    "status": "drift|error|success",
    "segment_a": {...},
    "segment_b": {...}
  }
  ```
- **Xóa bỏ Code dư thừa**: Loại bỏ hoàn toàn `handlePrune` để giữ codebase tinh gọn, sạch sẽ.
- **Tương thích ngược**: Giữ nguyên cơ chế phản hồi đơn (Single Report JSON) khi chỉ định chính xác một segment đơn lẻ (`source_shadow` hoặc `shadow_master`).
- **Tính toán Trạng thái Tổng hợp**: Trạng thái tổng hợp (`status`) của cả hai segment được tổng hợp theo mức độ ưu tiên: `error/failed` > `drift` > `success`.
