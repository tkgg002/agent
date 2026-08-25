# 04_decisions_gp2_3667.md - Nhật ký Quyết định Kiến trúc (ADRs)

## ADR-001: Sử dụng Kiến trúc 2 tầng (Source -> Shadow -> Master) cho CDC MongoDB -> PostgreSQL
- **Bối cảnh:** Dữ liệu MongoDB mang tính chất Schema-less (BSON Document), chứa nhiều kiểu dữ liệu phức tạp như BSON Date, BSON ObjectId, NumberLong, Nested Objects. Nếu write trực tiếp vào PostgreSQL Master table mà không qua Shadow table, bất kỳ thay đổi schema đột ngột nào từ MongoDB sẽ làm đứt pipeline CDC.
- **Quyết định:** Sử dụng mô hình chuẩn của `data-hub`:
  1. **Tier 1 (Source -> Shadow):** Nạp nguyên văn ExtJSON BSON từ MongoDB Kafka Change Stream vào PostgreSQL Shadow Table dưới dạng `payload JSONB` với khoá chính `_id`.
  2. **Tier 2 (Shadow -> Master):** Trực thi Transmuter Module trong `centralized-data-service` đọc từ Shadow Table, áp dụng Mapping Rules để bóc tách các trường chính thành các cột SQL Native (VARCHAR, BIGINT, TIMESTAMPTZ, JSONB) trên Master Table `transaction_history`.
- **Hậu quả & Lợi ích:**
  - **Lợi ích:** Đảm bảo đệm an toàn (fault-tolerance), không bao giờ mất dữ liệu gốc, dễ dàng re-transmute dữ liệu khi thay đổi mapping rule mà không phải chạy lại Snapshot từ MongoDB.
  - **Chi phí:** Tốn thêm dung lượng lưu trữ cho Shadow Table (có thể cleanup/partition định kỳ).
