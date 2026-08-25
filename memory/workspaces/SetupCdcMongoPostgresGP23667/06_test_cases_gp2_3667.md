# 06_test_cases_gp2_3667.md - Kế hoạch & Kết quả Kiểm thử (Validation)

## Danh mục Kịch bản Kiểm thử cho CDC MongoDB -> PostgreSQL

| ID | Tên Kịch bản Test | Các bước Thực hiện | Kỳ vọng Đầu ra | Trạng thái |
| :--- | :--- | :--- | :--- | :--- |
| **TC-01** | Kiểm tra đăng ký Connector MongoDB | Khai báo Mongo Connector qua CMS API | Connector trạng thái `RUNNING` trên Kafka Connect | PENDING |
| **TC-02** | Snapshot V2 Initial Data | Trigger Snapshot Runner cho Collection `transaction_history` | Dữ liệu cũ trong MongoDB được nạp đủ vào Shadow Table | PENDING |
| **TC-03** | Insert Realtime Document | Insert 1 transaction document mới vào MongoDB | Kafka event được consume, Shadow table có record mới | PENDING |
| **TC-04** | Transmuter Processing | Trigger hoặc để Transmuter auto-sync sang Master Table | Master table `transaction_history` có 1 dòng tương ứng với các cột parsed | PENDING |
| **TC-05** | Update & Delete Handling | Update status transaction trên MongoDB | Master table `transaction_history` cập nhật đúng status & timestamp | PENDING |
| **TC-06** | Data Integrity & Performance Check | Query đếm số lượng record (Count Source vs Count Target) | Số lượng record khớp 100%, query latency < 50ms | PENDING |
