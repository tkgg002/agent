# Yêu cầu lọc lịch sử đối soát theo check_type = smoke

## Bối cảnh
Hiện tại, API lịch sử đối soát của một bảng (`GetTableHistory` tại endpoint `/api/reconciliation/report/:table`) trả về toàn bộ các bản ghi đối soát bao gồm cả `check_type = 'hash_window'` và `check_type = 'smoke'`.
Khi vẽ biểu đồ biến động số lượng record trên UI (Convergence chart) hoặc hiển thị bảng Nhật ký đối soát, chúng ta muốn chỉ tập trung vào các phiên `smoke` check (là các phiên đối soát toàn bộ bảng, phản ánh đúng tổng số record thực tế). Các phiên `hash_window` check chỉ quét một khoảng thời gian/id nên số lượng của nó chỉ là số lượng trong window, không dùng để vẽ biểu đồ convergence tổng thể được.

## Yêu cầu
1. Sửa đổi API `GetTableHistory` ở backend (`cdc-cms-service`):
   - Khi `exclude_smoke` là `false` (hoặc mặc định): chỉ trả về các bản ghi có `check_type IN ('smoke', 'segment_b_smoke')`.
   - Khi `exclude_smoke` là `true`: chỉ trả về các bản ghi có `check_type NOT IN ('smoke', 'segment_b_smoke')`.
   - Cả tổng số bản ghi phân trang (`total`) và danh sách dữ liệu trả về (`data`) đều phải tuân thủ điều kiện lọc trên.
