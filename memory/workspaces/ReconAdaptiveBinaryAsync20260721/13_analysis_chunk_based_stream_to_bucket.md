# 13 — Thẩm Định Kiến Trúc: "Chunk-Based Stream-to-Bucket" (Hybrid Solution Evaluation)

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Cập nhật:** 2026-07-21  
> **Tác giả:** System Architect Audit  
> **Trạng thái:** ARCHITECTURAL RISK & FEASIBILITY EVALUATION  

---

## 1. Phân Tích Khả Năng Giải Quyết Các Lỗ Hổng Cũ

Mô hình lai ghép **"Chunk-Based Stream-to-Bucket"** (Chia 30 ngày thành 30 Jobs 1-ngày + Cursor Stream 1-ngày + 96 Buckets trên RAM Go) giải quyết trọn vẹn các tử huyệt của những phương án đơn lẻ trước đó:

| Tử Huyệt Cũ | Cơ Cơ Giải Quyết Của Mô Hình Lai | Kết Quả Đạt Được |
|:---|:---|:---|
| **Đứt Cáp Mạng (Network Blip)** | Chia 30 ngày thành 30 Chunks 1-ngày độc lập. Lưu Checkpoint (`checkpoint_ts`) sau mỗi Chunk. | Nếu đứt cáp ở ngày 16, Retry chỉ tốn đúng $3\text{s}$ của ngày 16. Không mất trắng dữ liệu 15 ngày trước. |
| **Phình Ổ Cứng DB (MVCC Bloat / Autovacuum)** | Khống chế thời gian mở DB Cursor từ 30 ngày ($720\text{h}$) xuống **đúng 1 ngày ($24\text{h}$)**, thời gian chạy Cursor chỉ tốn $\approx 3-5\text{s}$. | Triệt tiêu hoàn toàn rủi ro block Autovacuum ở Postgres và WiredTiger Cache pressure ở Mongo. |
| **DB CPU Overload (GROUP BY & Hash Query)** | DB chỉ thực hiện Index Scan lọc dữ liệu 1 ngày (`ORDER BY time ASC`). 100% phép tính Hash & XOR đẩy lên CPU Go App. | DB CPU ở mức tiệm cận $0\%$. Tải tính toán phân bổ đều trên Go Multi-core Nodes. |
| **Phóng Đại I/O Read ($12 \times N$)** | Dữ liệu thô 1 ngày chỉ đọc đúng $1 \times N$. 96 sub-windows 15m được dồn trực tiếp vào 96 Buckets trên RAM Go. | Khống chế tổng lượng đĩa DB phải đọc đúng $1 \times N$. |

---

## 2. Đánh Giá Các Rủi Ro Vận Hành Biên Còn Tồn Tại (Edge-Case Operational Risks)

Dù mô hình lai ghép rất tối ưu về mặt tổng thể kiến trúc, khi triển khai thực tế trên môi trường Production High-Load vẫn cần lưu ý 2 rủi ro biên sau:

### ⚠️ Rủi Ro 1: Lệch Mốc Ranh Giới Giữa Các Chunks Kế Tiếp (Boundary Skew Risk)
- **Bản chất:** Nếu bản ghi có `last_updated_at` nằm đúng mốc ranh giới `23:59:59.999` của Ngày 1 và `00:00:00.000` của Ngày 2.
- **Rủi ro:** Sự khác biệt về độ phân giải thời gian giữa MongoDB (BSON UTC Date - Millisecond) và PostgreSQL (TIMESTAMPTZ - Microsecond) có thể khiến bản ghi bị rơi vào cả 2 Chunks hoặc bị bỏ sót ở ranh giới giữa 2 ngày.
- **Yêu cầu kỹ thuật:** Điều kiện lọc `WHERE time >= start_day AND time < end_day` phải được chuẩn hóa toán tử nửa khoảng $[start, end)$ trên cùng độ phân giải millisecond ở cả 2 phía Database Driver.

### ⚠️ Rủi Ro 2: Biến Động Dữ Liệu Ngày Cao Điểm (Flash Sale / Heavy Data Skew)
- **Bản chất:** Vào các ngày bình thường, 1 ngày có $1.000.000$ bản ghi (Cursor stream trong $3\text{s}$). Nhưng vào ngày Flash Sale (ví dụ 11/11), số lượng bản ghi của 1 ngày có thể vọt lên $50.000.000$ bản ghi.
- **Rủi ro:** Chunk cứng 1-ngày có thể bị phình to đột biến trong ngày cao điểm, tái xuất hiện nguy cơ nghẽn Cursor Stream.
- **Yêu cầu kỹ thuật:** Kích thước Chunk nên hỗ trợ cơ chế **Dynamic Chunking (Phân đoạn linh hoạt)** — mặc định là 1 ngày, nhưng nếu số lượng bản ghi vượt quá trần $M$ records thì tự động chia nhỏ Chunk xuống 6 giờ hoặc 12 giờ.
