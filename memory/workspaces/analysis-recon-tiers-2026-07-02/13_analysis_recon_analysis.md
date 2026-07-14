# Phân tích Chi tiết Các loại Đối soát (Reconciliation - Recon) trong Hệ thống

Hệ thống đối soát hiện tại được thiết kế theo mô hình phân tầng nhiều lớp (multi-tiered reconciliation) nhằm tối ưu hóa chi phí vận hành cơ sở dữ liệu (đặc biệt là các DB quy mô lớn từ 10M - 50M records) và mạng lưới remote kết nối chập chờn.

Dưới đây là danh sách phân tích toàn diện 10 loại đối soát đang vận hành trong hệ thống:

---

## 1. Phân tầng Đối soát Chính (Segment A: Source ↔ Shadow)

### Tier 0: Count-Total (Fast Path)
- **Mục tiêu**: Đóng vai trò là chốt chặn đầu tiên (Fast Path) nhằm xác định nhanh xem dữ liệu 2 phía có khớp nhau hay không mà không cần quét toàn bộ bảng hay tính toán nặng.
- **Trigger**: Chạy tự động trong mỗi cycle đối soát.
- **Cơ chế hoạt động**:
  - **Source (MongoDB/PostgreSQL)**: Lấy ước lượng tổng số bản ghi bằng các hàm O(1) metadata (ví dụ: `EstimatedCount` từ MongoDB hoặc truy vấn metadata PostgreSQL).
  - **Destination (Shadow PostgreSQL)**: Đọc ước lượng dòng từ bảng hệ thống `pg_class.reltuples` (`EstimatedCountRows`), nếu không đáng tin cậy hoặc bằng 0 thì fallback về `CountRows` (COUNT(*) nhanh theo Primary Key).
  - **So sánh**: Tính toán sai số cho phép (`estTolerance` tỉ lệ thuận với tổng số dòng). Nếu chênh lệch nằm trong mức an toàn, hệ thống sẽ **DỪNG SỚM (Early Exit)** và trả về trạng thái `ok`.
- **Hiệu năng**: Cực kỳ nhanh (~0.3s - 0.8s/bảng). Tránh được tình trạng nghẽn cổ chai DB khi đối soát định kỳ.

### Tier 1: Bucket-Aggregate (Windowed Count)
- **Mục tiêu**: Định vị **khoảng thời gian nào bị lệch** khi Tier 0 phát hiện có sự chênh lệch tổng số lượng bản ghi hoặc lag hệ thống vượt ngưỡng.
- **Trigger**: Kích hoạt khi Tier 0 phát hiện lệch số lượng.
- **Cơ chế hoạt động**:
  - Chia khoảng thời gian lookback (ví dụ: 7 ngày) thành 168 bucket giờ trong memory.
  - Phía Source: Thực hiện 1 lệnh aggregate gom nhóm theo giờ (`BucketCounts` trên Mongo/Postgres) để đếm số lượng dòng trong từng bucket.
  - Phía Destination (Shadow): Thực hiện 1 truy vấn SQL GROUP BY theo giờ (`BucketCounts`) tương tự.
  - So khớp count của 168 bucket giờ. Đánh dấu các bucket có sai lệch để chuẩn bị cho bước drill-down ở Tier 2.
- **Hiệu năng**: Rất tối ưu. Thay thế hoàn toàn cơ chế V4 cũ (bắn 1.344 queries window-count liên tiếp gây sập mạng remote và khóa bảng).

### Tier 2: Hash Window (XOR Hash Drill-down)
- **Mục tiêu**: Đi sâu vào các bucket bị lệch từ Tier 1 để tìm ra **chính xác các ID cụ thể** bị thiếu hoặc sai lệch.
- **Trigger**: Tự động kích hoạt cho các bucket bị lệch sau Tier 1, hoặc gọi thủ công qua API/NATS.
- **Cơ chế hoạt động**:
  - Với mỗi window bị lệch, tính toán mã băm XOR Hash (`HashWindow`) dựa trên tổ hợp `_source_id` + `_source_ts` ở cả 2 phía.
  - Nếu XOR Hash lệch, thực hiện truy vấn chi tiết danh sách ID và Timestamp (`ListIDTsInWindow`) ở 2 phía.
  - Thực hiện diff IDTs để phân loại:
    - `missing_from_dest`: Bản ghi có ở nguồn nhưng thiếu ở đích shadow (cần heal).
    - `missing_from_src`: Bản ghi có ở shadow nhưng không có ở nguồn (orphans).
    - `mismatched`: Lệch timestamp do độ trễ đồng bộ.
  - Thực hiện cross-check ngược lại với Shadow DB để loại bỏ các trường hợp lệch giả do time window skew.
- **Hiệu năng**: Trung bình ~2.3s cho mỗi window lệch.

### Tier 3: Whole-Table Bucket Hash (Full-Table Fingerprinting)
- **Mục tiêu**: Quét toàn bộ cơ sở dữ liệu để tìm ra các sai lệch tiềm ẩn nằm ngoài lookback window thông thường (ví dụ dữ liệu lịch sử bị sửa đổi/xóa thủ công).
- **Trigger**: Chạy định kỳ vào khung giờ thấp điểm (off-peak 02:00 - 05:00 sáng).
- **Cơ chế hoạt động**:
  - Chia toàn bộ bảng dữ liệu thành 256 bucket dựa trên thuật toán băm ID.
  - Tính toán XOR Hash cho từng bucket ở cả 2 phía (Mongo và Shadow).
  - So sánh hash của 256 bucket để phát hiện các bucket bị lệch (drifted buckets).
  - Cơ chế bảo vệ (Budget Guard): Nếu tổng số dòng vượt quá `Tier3MaxDocsPerRun`, hệ thống sẽ tự động fallback về Tier 2 với lookback window để bảo vệ DB.
- **Hiệu năng**: Quét toàn bộ bảng, chi phí I/O cao nên chỉ chạy off-peak.

---

## 2. Các cơ chế bổ trợ và đối soát nâng cao

### Segment B (Shadow ↔ Master)
- **Mục tiêu**: Đối soát dữ liệu giữa lớp Shadow (lưu trữ trung gian sau CDC) và Master DB (DB thực tế của ứng dụng).
- **Trigger**: Chạy theo scheduler định kỳ hoặc trigger NATS.
- **Cơ chế hoạt động**:
  - Áp dụng các tầng đối soát tương tự Segment A (Tier 0, Tier 1, Tier 2) để so sánh số lượng và mã băm.
  - Điểm khác biệt: Dữ liệu ở Master DB đã được chuyển đổi (transmute) theo các Mapping Rules đã phê duyệt, do đó hệ thống cần map ID thực tế (ví dụ: `gpay_id`) về `source_id` để đối khớp.

### Orphan Prune
- **Mục tiêu**: Tìm kiếm và soft-delete các bản ghi "mồ côi" (orphan) - tức là bản ghi đã bị xóa ở Source DB nhưng shadow vẫn còn tồn tại (do CDC bị mất sự kiện DELETE).
- **Trigger**: Gọi qua NATS `tier=prune`.
- **Cơ chế hoạt động**:
  - Stream toàn bộ danh sách ID từ Source DB và so sánh với danh sách ID của Shadow DB.
  - Các ID chỉ tồn tại ở Shadow DB sẽ được đưa vào hàng đợi soft-delete (`_deleted = true`).
  - Có cơ chế an toàn: Nếu source trả về 0 ID (ví dụ do lỗi kết nối), hệ thống sẽ bỏ qua prune để tránh xóa nhầm toàn bộ bảng shadow.

### Heal-A & Heal-B (Tự động phục hồi dữ liệu)
- **Mục tiêu**: Tự động sửa đổi các bản ghi bị lệch hoặc thiếu đã được phát hiện từ các báo cáo đối soát.
- **Trigger**: NATS `recon-heal` sau khi có báo cáo drift.
- **Cơ chế hoạt động**:
  - **Heal-A**: Gửi các debezium-signal chunks để ép Debezium thực hiện incremental snapshot lại các bản ghi bị thiếu ở Shadow.
  - **Heal-B**: Thực hiện map ID từ Master DB về Shadow DB, sau đó trigger transmute chunks để ghi đè lại dữ liệu xuống Master.

### Row-Diff L3-B (Deep Field-by-Field Diff)
- **Mục tiêu**: So sánh sâu chi tiết từng trường dữ liệu (field-by-field) của các bản ghi lệch để phát hiện drift về mặt nội dung thuộc tính (ví dụ lệch trạng thái, số tiền...).
- **Trigger**: Chạy khi payload check có `deep: true`.
- **Cơ chế hoạt động**:
  - Fetch dữ liệu gốc của cả 2 phía theo ID (giới hạn tối đa 200 dòng).
  - Chạy dữ liệu shadow qua bộ engine Mapping Rules để tái lập lại giá trị kỳ vọng (expected).
  - So sánh chi tiết từng trường giữa expected và actual của Master để chỉ ra chính xác thuộc tính nào bị lệch.

### Timestamp Detector / Full Count Agg (Phụ trợ)
- **Mục tiêu**: Đo đạc sự phân bố timestamp của dữ liệu để tối ưu hóa kích thước window đối soát và phát hiện các mẫu drift lạ.
- **Trạng thái**: Hầu như đã bị disable để giảm tải tài nguyên hệ thống.
