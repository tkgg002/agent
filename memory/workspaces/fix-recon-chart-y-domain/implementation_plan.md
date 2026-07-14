# Kế hoạch triển khai: Tối ưu miền Y (Y-Axis Domain) của Biểu đồ Biến động Số lượng Phiên đối soát (Recon Chart)

## 1. Bối cảnh & Vấn đề hiện tại
* **Bối cảnh:** Biểu đồ đối soát "Biến động số lượng theo phiên recon" hiển thị số lượng bản ghi của 3 trạm (Source, Shadow, Master) qua các phiên đối soát.
* **Vấn đề:** Khi kích thước bảng cực kỳ lớn (ví dụ: 2.000.000 bản ghi), nếu có chênh lệch rất nhỏ (ví dụ: lệch 1-2 bản ghi), các đường vẽ Source, Shadow, Master sẽ hoàn toàn trùng khít lên nhau và phẳng lỳ. Điều này do trục Y mặc định của Recharts scale từ `0` hoặc sử dụng miền mặc định rộng, khiến độ lệch nhỏ biến mất trên mặt trực quan.

## 2. Giải pháp đề xuất
Chúng ta sẽ tính toán miền trục Y (`yDomain`) động ngay trong React trước khi truyền vào Recharts `<YAxis>`.

### Thuật toán tính toán miền Y động (`yDomain`):
1. Duyệt qua mảng `history.data` để tìm giá trị nhỏ nhất (`min`) và lớn nhất (`max`) của cả `source_count` và `dest_count`.
2. Tính dải dao động: `range = max - min`.
3. Xác định khoảng đệm (`padding`):
   - Nếu `range === 0` (dữ liệu hoàn toàn phẳng): đặt `padding = 5` đơn vị.
   - Nếu `range > 0`: đặt `padding = Math.max(1, Math.ceil(range * 0.1))` (lấy 10% dải dao động, tối thiểu là 1 đơn vị).
4. Miền trục Y được giới hạn: `[Math.max(0, Math.floor(min - padding)), Math.ceil(max + padding)]`. Việc sử dụng `Math.max(0, ...)` đảm bảo chặn dưới của trục Y không bao giờ bị âm (vì số lượng record luôn `>= 0`).

Giải pháp này hoàn toàn xử lý ở phía frontend và truyền mảng số `domain={yDomain}` cho `<YAxis />`.

---

## 3. Các thay đổi đề xuất

### 3.1. Frontend (`cdc-cms-web`)

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
* Thêm hàm `useMemo` tính toán `yDomain` dựa trên `history?.data`.
* Cập nhật thẻ `<YAxis>` để truyền prop `domain={yDomain}`.

---

## 4. Kế hoạch xác minh & Kiểm thử

### Kiểm thử tự động (CI/CD Gates)
* Chạy `npm run build` để kiểm tra lỗi build/TypeScript của frontend.

### Kiểm thử thủ công
1. Mở drawer drill-down của một pipeline có dữ liệu lớn và có lệch bản ghi nhỏ (ví dụ: lệch 1-2 record).
2. Kiểm tra trực quan xem các đường vẽ trên chart có hiển thị rõ khoảng cách dao động thay vì trùng khít/phẳng lỳ hay không.
3. Đảm bảo trục Y hiển thị dải số xung quanh mốc dữ liệu thay vì bắt đầu từ 0.
