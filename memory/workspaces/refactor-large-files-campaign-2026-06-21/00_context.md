# Context: Chiến dịch Refactor dọn dẹp các file lớn (> 500 LoC) trong centralized-data-service

## 1. Bối cảnh
- Sau khi hoàn thành xuất sắc việc phân rã `kafka_consumer.go`, `recon_source_agent.go` và `recon_dest_agent.go`, chúng ta nhận thấy dự án còn 19 file Go lớn khác có số dòng code vượt quá 500 dòng.
- Các file này có tính phức tạp cao, gộp chung nhiều logic nghiệp vụ, vi phạm nguyên lý Single Responsibility Principle (SRP) và gây khó khăn cho việc bảo trì, tối ưu hóa.

## 2. Mục tiêu chiến dịch
- Thực hiện rà soát, lên kế hoạch chi tiết và tiến hành refactor phân rã toàn bộ các file lớn này theo từng giai đoạn (Phases) và từng nhóm chức năng (Domains).
- Đảm bảo giữ nguyên logic nghiệp vụ, tính tương thích ngược, biên dịch thành công và vượt qua 100% các bài unit test.
- Đảm bảo an toàn bảo mật, bảo vệ dữ liệu và áp dụng các cơ chế Circuit Breaker / Rate Limiting phù hợp.
