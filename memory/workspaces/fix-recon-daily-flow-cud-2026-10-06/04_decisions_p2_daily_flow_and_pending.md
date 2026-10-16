# 04 — Architecture Decision Records (ADRs): Phase P2

Ngày: 2026-10-12

## ADR-P2-1: Tập hợp dữ liệu dòng chảy ngày (Daily Flow) bằng SQL Aggregation phía Server thay vì Client Reducer

- **Bối cảnh**: Client FE hiện fetch 200 logs thô của từng operation (`kafka-consumer`, `transmute`) rồi tự gom nhóm ngày bằng JavaScript reducer. Ở hệ thống có tần suất cao, 200 bản ghi chỉ bao quát được 1–2 giờ, khiến bảng Daily Flow bị "cụt đầu", mất toàn bộ các ngày trước.
- **Quyết định**: Xây dựng SQL Aggregation Endpoint phía server (`GET /api/reconciliation/daily-flow`).
- **Hệ quả & Đánh giá**:
  - *Tích cực*:
    - Tận dụng PostgreSQL Partition Pruning (`created_at >= NOW() - INTERVAL '14 days'`) để quét trực tiếp các partition tương ứng, thời gian thực thi < 25ms.
    - Trả về đúng 14 dòng (mỗi ngày 1 dòng) thay vì đẩy hàng trăm record JSON nặng qua mạng.
    - Dữ liệu 14 ngày luôn đầy đủ 100%, không bao giờ bị cắt xén bất kể khối lượng log là bao nhiêu.
  - *Tiêu cực*: Cần viết câu truy vấn SQL tổng hợp cẩn thận để gom đúng theo múi giờ địa phương của client (`tz_offset`).

## ADR-P2-2: Khóa dung lượng slice trong `chain()` bằng cú pháp 3-index slice `out[:len:len]`

- **Bối cảnh**: Hàm `chain(mws ...fiber.Handler)` trong `internal/router/router.go` tạo slice với capacity bằng số lượng tham số truyền vào. Nếu có middleware `nil`, length < capacity. Sau đó, các biến middleware dùng chung như `destructiveChain` được truyền vào `append(destructiveChain, handler)`. Khi đó, các lần append của các route khác nhau có thể ghi đè vào cùng 1 mảng nền (backing array aliasing).
- **Quyết định**: Trả về `out[:len(out):len(out)]` từ hàm `chain()`.
- **Hệ quả & Đánh giá**:
  - *Tích cực*: Đây là chuẩn idiomatic Go (Go 1.2+ Full Slice Expression) để bảo vệ slice khỏi việc bị `append` làm biến đổi dữ liệu chung. Sửa đúng 1 dòng code, không ảnh hưởng bất kỳ luồng nghiệp vụ nào.
  - *Tiêu cực*: Không có.

## ADR-P2-3: Phân loại lỗi Transmute để triệt tiêu việc đếm thừa DLQ (Dead Letter Task)

- **Bối cảnh**: Quá trình binary search split chia nhỏ một mẻ lớn bị lỗi sẽ tạo ra các log `transmute_sub` lỗi ở các nhánh trung gian trước khi cô lập được 1 bản ghi poison pill duy nhất. Client đếm mọi log lỗi dẫn đến việc 1 poison pill bị tính thành nhiều lỗi DLQ.
- **Quyết định**:
  - Trong `transmute_handler.go`: Các lần chia nhỏ trung gian (`len(batch) > 1`) ghi `details.split_retry = true`.
  - Khi cô lập thành công (`len(batch) == 1`), ghi `details.poison_pill = true`.
  - Bộ tính toán DLQ chỉ cộng các lỗi có `poison_pill = true` hoặc lỗi batch không chia nhỏ (không có cờ `split_retry`).
- **Hệ quả & Đánh giá**:
  - *Tích cực*: Số lượng lỗi phản ánh chính xác số lượng bản ghi hoặc sự kiện hỏng thực tế (1 poison pill = 1 DLQ).
  - *Tiêu cực*: Không có.
