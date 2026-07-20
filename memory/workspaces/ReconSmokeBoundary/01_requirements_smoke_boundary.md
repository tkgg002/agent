# Yêu cầu: Tối ưu hóa Smoke Check tránh cảnh báo giả khi Pipeline đang hoạt động (Recon Smoke Boundary Yêu cầu Cải tiến)

## Bối cảnh
Hiện tại, cơ chế Smoke Check (đối soát nhanh bằng số lượng) của chặng A (Source ↔ Shadow) và chặng B (Shadow ↔ Master) đang đếm toàn bộ dữ liệu từ trước đến nay (`all -> now`). Khi hệ thống đồng bộ (CDC Sink hoặc Transmuter) đang hoạt động, dữ liệu mới ghi ở nguồn chưa kịp đồng bộ xuống đích sẽ tạo ra độ lệch số lượng tức thời, gây ra cảnh báo `drift` giả (false positive).

## Yêu cầu Nghiệp vụ
1. **Giải pháp trừ bù cửa sổ gần đây:**
   - Thay vì so sánh trực tiếp tổng số lượng đến hiện tại, ta sẽ lấy tổng số lượng (`all -> now`) trừ đi số lượng bản ghi phát sinh trong cửa sổ gần đây (ví dụ: `120 giây` gần nhất).
   - Công thức tính số lượng sạch (clean count): `cleanCount = totalCount - recentCount` (nếu `cleanCount < 0` thì gán bằng `0`).
   - So sánh các `cleanCount` này giữa các tầng để kết luận drift.

2. **Quy tắc làm tròn mốc thời gian (from):**
   - Mốc thời gian bắt đầu của cửa sổ gần đây `from` được tính bằng: `now - 120s`.
   - Nếu có giây lẻ, làm tròn lùi về đầu phút (ví dụ: `15:04:22` trừ 120s thành `15:02:22`, làm tròn lùi về `15:02:00`).
   - Trong Go, sử dụng: `from := now.Add(-120 * time.Second).Truncate(time.Minute)`.

3. **Yêu cầu Hiệu năng & Tránh đánh sập MongoDB (Không scan toàn bảng):**
   - Không được phép sử dụng `CountDocuments` (exact count) mặc định trên MongoDB của Chặng A vì sẽ gây full index/table scan trên hàng chục triệu bản ghi, làm quá tải CPU/IO và gây nghẽn dịch vụ thanh toán (đánh sập hệ thống).
   - Sử dụng `EstimatedCount` (O(1) đọc từ metadata của collection) cho MongoDB làm mặc định để đảm bảo an toàn tuyệt đối cho database.
   - Để giải quyết sai số (drift xấp xỉ) của `EstimatedCount` dưới ràng buộc Zero-tolerance (lệch 1 row cũng là drift):
     - Nếu phép toán trừ bù `srcEstClean - dstActiveClean == 0`: Xác nhận khớp, trả về `"ok"`.
     - Nếu có độ lệch (`diff != 0`): Hệ thống **không báo drift ngay**, mà thực hiện kiểm tra chéo bằng **Hash Window** trên cửa sổ thời gian tĩnh trước đó (`lo` đến `hi`).
     - **Mốc thời gian tĩnh:** Để loại bỏ hoàn toàn nhiễu trễ đồng bộ (replication lag), mốc trên `hi` của Hash Window phải được đặt chính xác bằng `from` (`now - 120s` làm tròn phút) thay vì dùng `now` hay `now - lag`. Mốc dưới `lo` được tính là `hi.Add(-lookback)`. Dải kiểm tra sẽ là `[hi - lookback, hi)`.
     - Nếu kết quả `HashWindow` của nguồn (MongoDB) và đích (Shadow) trùng khớp cả về số lượng (`Count`) và chữ ký (`XorHash`): Xác nhận hệ thống thực tế hoàn toàn đồng bộ, độ lệch ban đầu chỉ là sai số metadata của `EstimatedCount`. Hệ thống tự động thiết lập `diff = 0` và trả về `"ok"` (không bắn cảnh báo giả).
     - Nếu `HashWindow` không khớp: Xác nhận có drift thực sự, báo trạng thái `"drift"` để vận hành xử lý.

4. **Xử lý xóa vật lý ở nguồn và xóa mềm ở đích (Soft Delete in Window):**
   - Khi nguồn (MongoDB) xảy ra Hard Delete trong cửa sổ 120s, số lượng `srcTotal` giảm đi, nhưng `srcRecent` bằng 0.
   - Ở đích (Postgres Shadow/Master), bản ghi được cập nhật xóa mềm (`_deleted = true`). Điều này làm `dstActive` giảm đi, nhưng bản ghi vẫn nằm trong cửa sổ thời gian gần đây (`dstRecentTotal` tăng lên 1).
   - Để loại bỏ sai số này, ta thực hiện đếm thêm số bản ghi bị xóa mềm trong cửa sổ 120s gần nhất (`dstRecentDeleted` với cờ `_deleted = true`).
   - Số lượng recent active thực tế là: `dstRecentActive = dstRecentTotal - dstRecentDeleted`.
   - Số lượng sạch ở đích là: `dstActiveClean = dstActive - dstRecentActive`.
   - Công thức này loại bỏ hoàn toàn sai số khi có thao tác xóa (Hard Delete ở nguồn / Soft Delete ở đích) trong cửa sổ 120s gần nhất.
