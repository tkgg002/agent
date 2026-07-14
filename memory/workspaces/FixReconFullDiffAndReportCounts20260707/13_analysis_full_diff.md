# Analysis: Root Cause of Recon Full Diff Inaccuracy & Missing Fields

## 1. Root Cause Analysis

### Vấn đề 1: `full_diff` không mang lại kết quả chính xác (bỏ sót lệch dữ liệu)
- **Cơ chế cũ:** `TypeReconFullDiff` (Full Search) gọi hàm `TimeBoundedDiffMissingFromShadow` (Segment A) và `TimeBoundedDiffMissingFromMaster` (Segment B).
- **Hạn chế:** Các hàm này chỉ tải lên danh sách các ID (`_source_id` / `_gpay_id`) trong khoảng thời gian xác định, sau đó so sánh xem ID của trạm nguồn có tồn tại ở trạm đích hay không.
- **Hậu quả:** 
  - Nếu một bản ghi có dữ liệu khác nhau ở 2 trạm (mismatched) nhưng ID của nó đã đồng bộ và tồn tại ở cả 2 trạm, hàm so khớp ID thuần túy sẽ báo cáo chênh lệch bằng 0 và trả về `success`.
  - Nếu có các bản ghi thừa ở destination (orphans), các hàm này cũng hoàn toàn bỏ qua.
  - Đây là lý do tại sao Report 49 báo `success` 0 drift, trong khi Report 46 (`hash_window`) báo `drift` với 1 mismatched ID `6a4486a7cb544c04498b9ba2`.

### Vấn đề 2: Thiếu các trường đếm số lượng trong báo cáo `hash_window` (Segment A)
- **Hạn chế:** Trong `recon_tier_a.go`, hàm `RunHashWindowCheck` khởi tạo báo cáo `ReconciliationReport` nhưng KHÔNG gán giá trị cho các cột:
  - `SourceCount` (số lượng scanned nguồn)
  - `DestCount` (số lượng scanned đích)
  - `Diff` (chênh lệch scanned)
  - `TotalSourceCount` (tổng Mongo collection)
  - `TotalDestCount` (tổng shadow table không deleted)
- **Hậu quả:** Các cột này ghi nhận `nil` hoặc `0` trong database, làm cho Dashboard UI hiển thị không đầy đủ thông tin hoặc ghi nhận sai lệch chênh lệch.

### Vấn đề 3: "Nhồi nhét" thông tin lệch vào `stale_ids`
- **Cơ chế cũ:** `RunHashWindowCheck` tổng hợp tất cả các loại lệch (bao gồm cả `missing_from_dest`) vào một đối tượng JSON duy nhất rồi lưu vào cột `stale_ids`.
- **Hậu quả:** Điều này làm cho việc theo dõi trên UI/DB kém phân tách, đồng thời làm mất đi tính rõ ràng của cột `missing_ids`. Thực tế, `missing_from_dest` (những bản ghi bị thiếu thực sự ở đích) nên được lưu vào cả `missing_ids` để dễ dàng hiển thị và heal riêng.

---

## 2. Đề xuất giải pháp (Solution Proposal)

### Giải pháp 1: Đồng bộ hóa cơ chế `full_diff` sang XOR Hash Window Check
Thay vì duy trì một hàm kiểm tra ID thô sơ và không chính xác, chúng ta sẽ cho `TypeReconFullDiff` kế thừa toàn bộ sức mạnh của XOR Hash Window Check (`RunHashWindowCheck` / `RunHashWindowCheckB`).
- Khi chạy `full_diff`, chúng ta bọc context bằng `WithReconTimeRange(ctx, start, end)`.
- Gọi hàm `RunHashWindowCheck` / `RunHashWindowCheckB`.
- Override trường `CheckType` của báo cáo được trả về thành `full_diff`.
- Điều này đảm bảo:
  - Kiểm tra sâu toàn bộ các trường dữ liệu thông qua hashing.
  - Phân loại rõ ràng: `mismatched`, `missing_from_dest`, và `missing_from_src`.
  - Kết quả chính xác 100% và đồng bộ với cơ chế window check tiêu chuẩn.

### Giải pháp 2: Cập nhật các trường đếm trong `RunHashWindowCheck` (Segment A)
Trong `RunHashWindowCheck` tại `recon_tier_a.go`:
- Khai báo 2 biến tích lũy: `totalSrc` và `totalDst`.
- Cộng dồn số lượng bản ghi của từng window:
  ```go
  totalSrc += srcRes.Count
  totalDst += dstRes.Count
  ```
- Thực hiện truy vấn bất đồng bộ hoặc không chặn (non-blocking query) để lấy tổng số bản ghi thực tế của 2 trạm:
  - `srcEst` từ MongoDB `CountDocuments`.
  - `dstTotal` từ Shadow DB Postgres count where `NOT _deleted`.
- Gán tất cả các trường đếm này vào struct `ReconciliationReport` trước khi gọi `rc.stampA(...)`.

### Giải pháp 3: Cập nhật quy trình Chữa lành `proposeFullDiffHealA`
Trong `recon_check_heal_handler.go`, sửa hàm `proposeFullDiffHealA` để:
- Sử dụng `WithReconTimeRange` để chạy `RunHashWindowCheck` thay vì gọi trực tiếp `TimeBoundedDiffMissingFromShadow`.
- Triển khai lệnh heal toàn diện (gồm mismatched, missing_dest, và prune) tương tự như `proposeWindowHealA`.
