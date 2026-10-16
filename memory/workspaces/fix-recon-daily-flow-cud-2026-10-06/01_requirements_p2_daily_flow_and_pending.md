# 01 — Requirements: Phase P2 & Pending Issues Resolution

Ngày lập: 2026-10-12
Trạng thái: Proposed (Chờ User APPROVE để thực thi)

## 1. Bối cảnh & Mục tiêu
Sau khi hoàn tất Phase P1 (Fix bảo mật Clear Activity Log, cách ly đa connector, partition pruning và đồng bộ tham số bare table name), hệ thống còn lại 5 hạng mục tồn đọng cần giải quyết dứt điểm theo tôn chỉ **Simplicity First, Minimal Impact, Tư duy Core Systems**:

1. **R1: Xóa bỏ giới hạn 200 logs bằng Server-side Daily Flow Aggregation API**
   - *Vấn đề*: Hiện tại FE gọi 2 request `/api/activity-log` với `limit=200` và tự chạy reducer nhóm theo ngày. Ở các môi trường lưu lượng cao (hàng trăm batch/giờ), 200 logs chỉ chứa dữ liệu vài giờ gần nhất, làm mất toàn bộ các ngày cũ hơn.
   - *Yêu cầu*: Xây dựng endpoint tổng hợp phía server `GET /api/reconciliation/daily-flow` (hoặc `/api/activity-log/daily-flow`). Truy vấn trực tiếp từ PostgreSQL dùng Partition Pruning (`created_at >= NOW() - INTERVAL 'X days'`), gom nhóm theo ngày địa phương (`tz_offset`), trả về dữ liệu ngày chuẩn xác, không bị giới hạn số lượng log.

2. **R2: Khắc phục đếm thừa DLQ khi Transmute Batch chia để trị (Poison Pill Split Over-count)**
   - *Vấn đề*: Khi 1 mẻ transmute lớn gặp lỗi poison pill, `TransmuteHandler` ghi nhận `status='error'` ở batch cha, sau đó chạy binary search split đệ quy. Mỗi sub-batch nhánh lỗi tiếp tục log `transmute_sub` lỗi. FE hiện tại cứ gặp `status === 'error'` là `+1 DLQ`, khiến 1 lỗi duy nhất bị đội lên 5–7 lần.
   - *Yêu cầu*:
     - Tầng CDS (`transmute_handler.go`): Đánh dấu `"split_retry": true` trong `details` của các lượt lỗi trung gian khi mẻ đang được chia nhỏ (`len(batch) > 1`). Chỉ lượt cô lập được poison pill (`len(batch) == 1`) hoặc thất bại không thể phục hồi mới được đánh dấu là lỗi chốt (`"poison_pill": true`).
     - Tầng CMS Aggregation / Grid: Chỉ đếm các lỗi thực sự vào cột DLQ/Lỗi, bỏ qua các bước chia nhỏ trung gian.

3. **R3: Chuẩn hóa ngữ nghĩa đối soát giữa Khối 2 (Kafka CDC Ingest) và Khối 3 (Shadow DB Write)**
   - *Vấn đề*: Khối 2 và Khối 3 hiện lấy cùng từ log `kafka-consumer`. Khi không có lỗi, số lượng hiển thị bằng nhau tạo cảm giác đối soát giả (tautology).
   - *Yêu cầu*: Phân biệt rõ ràng định nghĩa:
     - **Khối 2 (KAFKA CDC BẮN VỀ)**: Tổng số sự kiện CDC được Debezium/Kafka đẩy vào hệ thống (`total_events` = C + U + D).
     - **Khối 3 (GHI VÀO DB SHADOW)**: Số lượng lệnh đã thực thi thành công vào Shadow DB (`ops_applied`) và số dòng vật lý thực tế được biến đổi (`written_db_rows`).
     - Ghi nhận rõ sự chênh lệch nếu có: Lỗi ghi DB (rơi vào DLQ / `failed_sync_logs`) hoặc Coalescing/Deduplication trong mẻ đệm.

4. **R4: Khắc phục rủi ro Memory Aliasing trong Router `chain()` của CMS (Go Slice Capacity Bug)**
   - *Vấn đề*: `chain()` tạo slice với `cap = len(mws)`. Nếu có middleware `nil`, `len < cap`. Khi gán `destructiveChain := chain(...)` và gọi `append(destructiveChain, handler)` nhiều lần cho các route khác nhau, Go runtime có thể tái sử dụng vùng nhớ capacity thừa, gây ghi đè handler lẫn nhau giữa các route.
   - *Yêu cầu*: Dùng 3-index slice `out[:len(out):len(out)]` trong `chain()` để khóa cứng capacity bằng đúng length, buộc Go cấp phát mảng mới khi gọi `append`.

5. **R5: Chính sách & Tool xử lý Log cũ (Legacy Ambiguous Logs Maintenance)**
   - *Vấn đề*: Log cũ ghi trước khi deploy P1 chưa có `shadow_schema` trong `details`. Nếu bảng có nhiều connector trùng tên, ClearActivity đang bỏ qua để đảm bảo an toàn.
   - *Yêu cầu*: Cung cấp SQL Script bảo trì một lần (One-time enrichment) để backfill `shadow_schema` cho các bảng không bị tranh chấp (chỉ có 1 binding). Với các bảng thực sự tranh chấp đồng thời trong quá khứ, giữ nguyên để partition retention tự giải phóng sau 30 ngày.
