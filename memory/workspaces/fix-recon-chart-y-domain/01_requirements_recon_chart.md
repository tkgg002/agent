# Yêu cầu: Tối ưu hoá Miền Y (Y-Axis Domain) của Biểu đồ Biến động Số lượng Phiên Recon

## 1. Bối cảnh & Vấn đề hiện tại
* **Bối cảnh:** Biểu đồ đường ("Biến động số lượng theo phiên recon") hiển thị số lượng record qua các phiên đối soát (Source, Shadow, Master).
* **Vấn đề:** Khi tổng số lượng dữ liệu cực kỳ lớn (ví dụ: 2.000.000 record), sự biến động hoặc sai lệch rất nhỏ (ví dụ: lệch 1-2 record) sẽ hoàn toàn bị ẩn đi/trông như một đường thẳng tắp và trùng lặp hoàn toàn vì trục Y hiện tại tự động scale từ `0` hoặc không có padding hợp lý. Người vận hành không thể phát hiện trực quan việc lệch dữ liệu nhỏ trên biểu đồ.

## 2. Mục tiêu
* Cấu hình trục Y (`YAxis`) của biểu đồ biến động số lượng để tự động phóng to (zoom-in) vào dải giá trị thực tế của dữ liệu.
* Đảm bảo rằng sự chênh lệch nhỏ (lệch 1, 2 record) ở quy mô dữ liệu lớn (hàng triệu record) vẫn hiển thị trực quan dưới dạng các đường tách biệt, uốn lượn rõ ràng.
* Không làm ảnh hưởng đến các trường hợp chênh lệch lớn (hàng chục nghìn record) hoặc khi số lượng record nhỏ.
* Đảm bảo giá trị tối thiểu của miền trục Y không âm (vì số lượng record luôn >= 0).

## 3. Scope & Thành phần bị ảnh hưởng
* **Component:** `src/components/ReconPipelineGrid.tsx` (phần render `<LineChart>` và `<YAxis>`).
* **Trạng thái:** Frontend build thành công, chạy bình thường.
