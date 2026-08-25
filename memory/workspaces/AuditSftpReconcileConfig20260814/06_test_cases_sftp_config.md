# 06_test_cases: Danh Mục Kịch Bản Kiểm Thử (Test Cases Specification)

## 1. Unit & Functional Test Cases

| ID | Tên Kịch Bản | Đầu Vào (Input) | Kết Quả Kỳ Vọng (Expected Outcome) | Trạng Thái |
|---|---|---|---|---|
| **TC-01** | Kiểm tra đọc file CSV chuẩn 1,000 dòng | File `reconcile_20260814_01.csv` có header và 1,000 dòng dữ liệu | Kafka Produce đủ 1,000 messages vào topic `cdc.sftplocal.reconcile.transactions` | PASS |
| **TC-02** | Kiểm tra trích xuất Record Key (SMT) | Record CSV có `transaction_id = "TX100982"` | Message trong Kafka có `Key = "TX100982"` (String) | PASS |
| **TC-03** | Kiểm tra bỏ qua file chưa hoàn tất (`.tmp`) | Upload file `reconcile_20260814_02.csv.tmp` | Connector KHÔNG đọc file `.tmp`, bỏ qua 100% | PASS |
| **TC-04** | Kiểm tra Atomic Rename Protocol | Rename `reconcile_20260814_02.csv.tmp` -> `reconcile_20260814_02.csv` | Connector lập tức phát hiện và đọc toàn bộ file | PASS |
| **TC-05** | Kiểm tra dòng CSV rác & Bẫy DLQ | File CSV chứa 1 dòng bị rác mã hóa UTF-8 hoặc hỏng số cột | Dòng rác được đẩy vào DLQ `dlq.cdc.sftplocal.reconcile.transactions`, Connector giữ trạng thái RUNNING | PASS |
| **TC-06** | Kiểm tra bảo mật Mật khẩu | Gọi API `GET /connectors/sftp-reconcile-source-v1/config` | Mật khẩu hiển thị dưới dạng biến `${file:...}`, không lộ plain-text | PASS |
| **TC-07** | Kiểm tra Dọn rác & Archive | File CSV trong `/reconcile/` có mtime +3 ngày | Cronjob di chuyển file sang `/reconcile_archive/YYYY-MM/` | PASS |

## 2. Boundary & Negative Test Cases
- **TC-08 (File Rỗng 0-byte):** Connector bỏ qua file 0-byte, log WARN nhẹ và tiếp tục poll.
- **TC-09 (Tương tác Mạng SFTP chập chờn):** SFTP Server bị mất kết nối 30 giây -> Connector retry gracefully khi có lại kết nối.
- **TC-10 (Số tiền cực lớn):** Record chứa `amount = 999999999999.99` -> Consumer parse đúng dạng chuỗi/Decimal mà không bị tràn số Float.
