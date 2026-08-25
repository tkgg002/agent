# Kết quả kiểm thử & Xác minh: Di chuyển tạo Topic Kafka SFTP sang nút Snapshot

Tài liệu ghi nhận kết quả xác minh kỹ thuật.

---

## 1. Kết quả chạy kiểm thử tự động (Automated Tests)

Các lệnh test đã chạy thành công trong môi trường của Muscle:
- Chạy unit tests cho các commands liên quan:
  `go test ./internal/app/commands/shadow/... ./internal/app/commands/source/... ./internal/api/source/... ./test/internal/api/...`
  **Kết quả:** `PASS 100%`
- Biên dịch CMS server:
  `go build ./cmd/server`
  **Kết quả:** `SUCCESS (exit code 0)`

---

## 2. Kịch bản xác minh thủ công đề xuất (Manual Verification Steps)

Để kiểm tra thực tế hoạt động:
1. Đăng ký một nguồn dữ liệu SFTP mới.
2. Kiểm tra trên Kafka Connect xem connector đã được tạo thành công chưa (đã tạo nhưng chưa có Kafka topic).
3. Tạo Shadow Table, duyệt Mapping Rules thành công.
4. Bật toggle **Active** (ở Header mapping hoặc dòng shadow).
5. Nhấn nút **Snapshot** trên dòng Table Registry.
6. Xác nhận:
   - Một topic mới dạng `cdc.sftp...` được tự động tạo ra.
   - Connector bắt đầu sync dữ liệu tập tin từ SFTP vào Kafka.
   - Worker daemon bắt đầu tiêu thụ bản tin.
