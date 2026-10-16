# Yêu cầu Kỹ thuật: Audit Số liệu c,u,d Đối Soát Theo Ngày & Loại bỏ Read(r)

## 1. Bối cảnh & Hiện trạng
- Người dùng phản ánh số liệu ngày `2026-09-30`:
  `2026-09-30 | 1,562 | 0 | 1,562 | 0 | 0 | 1,562 | 1,562 | 0 | 0 | 2,084`
- Bảng thực tế chỉ có khoảng 500 record (chính xác là 536 records tại cả DB Shadow và DB Master).
- Số liệu $c, u, d$ đang không đúng: toàn bộ dồn vào `c` (1,562), còn `u = 0`, `d = 0`, `r = 0`.
- Người dùng chỉ đạo rõ ràng: "1. KAFKA CDC BẮN VỀ k cần cái read(r)".

## 2. Mục tiêu (Definition of Done)
1. **Truy vết vật lý (Evidence-Based Audit)**:
   - Chứng minh nguồn gốc con số 1,562 của Kafka Consumer và 2,084 của Transmute từ bảng thực tế `cdc_system.cdc_activity_log` trong DB `cdc_dw`.
   - Giải thích cặn kẽ bản chất Throughput (Lưu lượng sự kiện xử lý) vs Table Size (Số bản ghi tồn đọng trong bảng).
2. **Loại bỏ hoàn toàn Read(r) khỏi KAFKA CDC BẮN VỀ**:
   - Loại bỏ cột `Read(r)` ở Modal phân tích theo ngày (chỉ còn Tổng, Insert(c), Update(u), Delete(d)).
   - Loại bỏ hiển thị `r:` ở Card 1 Box tổng hợp số liệu ngoài.
   - Cập nhật hàm tính `totals` và footer `Table.Summary` tương ứng chính xác (18 cột).
3. **Giải pháp (Solution Proposal)**:
   - Backend: Bổ sung trường `Op` vào `UpsertRecord` và ghi nhận chi tiết `c, u, d` vào `details` của `cdc_activity_log` khi Kafka consumer flush batch.
   - Frontend: Xử lý hiển thị chuẩn xác, không tự ý dồn ép bừa vào `c`.
