# Phân tích Kỹ thuật - Đối soát hai chiều (Bidirectional Reconciliation)

Tài liệu này phân tích cấu trúc hiện tại và đề xuất thuật toán tối ưu để thực hiện đối soát hai chiều (Bidirectional Check) cho luồng Full Diff.

## 1. Phân tích thuật toán hiện tại (Một chiều - Source -> Shadow)
Hàm `TimeBoundedDiffMissingFromShadow` hiện tại thực hiện các bước sau:
1. Đọc toàn bộ `_source_id` từ Shadow Postgres DB trong khoảng thời gian `[startTime, endTime]` nạp vào map `shadowSet` (chỉ bọc các bản ghi chưa bị xóa `NOT _deleted`).
2. Stream các ID từ Source DB trong cùng khoảng thời gian.
3. Với mỗi ID đọc được từ Source:
   - Tăng `srcCount`.
   - Kiểm tra nếu ID không tồn tại trong `shadowSet`, đánh dấu là `missing`.
4. Trả về tập hợp `missing`.

**Hạn chế:** Nếu có một bản ghi tồn tại trong Shadow Postgres DB nhưng không hề có ở Source DB (do lỗi CDC hoặc đã bị xóa cứng ở Source), thuật toán này hoàn toàn bỏ qua.

## 2. Thiết kế thuật toán Đối soát hai chiều (Bidirectional) tối ưu
Để phát hiện cả bản ghi thiếu ở Shadow (missing) và bản ghi dư thừa ở Shadow (stale), ta áp dụng thuật toán sau:
1. Nạp toàn bộ `_source_id` từ Shadow Postgres DB vào map `shadowSet` (khóa là ID, giá trị là `struct{}`).
2. Stream các ID từ Source DB.
3. Với mỗi ID nhận được từ Source:
   - Tăng `srcCount`.
   - Nếu ID **không có** trong `shadowSet`: thêm vào danh sách `missing`.
   - Nếu ID **có** trong `shadowSet`: thực hiện **xóa** ID đó khỏi `shadowSet` bằng lệnh `delete(shadowSet, id)`.
4. Sau khi stream từ Source kết thúc:
   - Toàn bộ các ID còn lại trong `shadowSet` chính là các bản ghi tồn tại ở Shadow nhưng không có ở Source (bản ghi **stale**).
   - Duyệt qua `shadowSet` để gom các ID này vào danh sách `stale`.

**Ưu điểm:** 
- Tiết kiệm bộ nhớ: Không cần tạo thêm một `sourceSet` riêng biệt.
- Hiệu năng cao: Các thao tác `delete` trên map trong Go có độ phức tạp trung bình $O(1)$.

## 3. Phân tích ảnh hưởng (Impact Analysis) & Cập nhật Callsite
Việc thay đổi signature của `TimeBoundedDiffMissingFromShadow` từ:
`TimeBoundedDiffMissingFromShadow(...) ([]string, int, error)`
thành:
`TimeBoundedDiffMissingFromShadow(...) (missing []string, stale []string, srcCount int, err error)`
sẽ ảnh hưởng đến hai file sau:

### A. `internal/handler/recon/recon_check_handler.go`
- Cập nhật hàm gọi nhận 4 giá trị trả về.
- Ghi nhận `staleIDs` vào báo cáo `ReconciliationReport`:
  - `StaleCount = len(staleIDs)`
  - `StaleIDs = json.Marshal(staleIDs)`
  - `Diff = len(missing) + len(stale)`
  - Trạng thái `status = "drift"` nếu `len(missing) > 0 || len(stale) > 0`.

### B. `internal/handler/recon/recon_heal_handler.go`
- Tại dòng 574:
  `missing, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctxDiff, *entry, start, end)`
  Cần sửa thành:
  `missing, stale, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctxDiff, *entry, start, end)`
- Trong logic heal segment A mode `full_diff`:
  - Cần gộp cả `missing` và `stale` (nếu cần xử lý hoặc log). Hiện tại, hàm heal chỉ thực hiện `FetchAndWriteByIDs` cho `missing` (nhằm ghi đè/bổ sung bản ghi thiếu).
  - Đối với bản ghi `stale` (dư thừa ở Shadow), logic chữa lành đúng là prune/soft-delete chúng ở Shadow. Tuy nhiên, để đảm bảo an toàn và đúng phạm vi ban đầu, ta sẽ log danh sách `stale` và tập trung sửa lỗi biên dịch trước. Nếu cần, ta sẽ đưa `stale` vào danh sách xử lý phù hợp.
