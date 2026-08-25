# 04_decisions.md - Nhật ký Quyết định Kiến trúc (ADRs)

## ADR-01: Chuẩn hóa Type Inference cho MongoDB BSON Extended JSON v2
- **Bối cảnh**: Khi đọc dữ liệu Mongo từ Debezium CDC hoặc từ Mongo Introspection Driver, các kiểu dữ liệu đặc thù của BSON (Date, ObjectId, Decimal128, Int64) được serialize thành JSON dạng object có chứa wrapper key (`$date`, `$oid`, `$numberDecimal`, `$numberLong`). Hàm `InferTypeFromRawData` hiện tại xem bất kỳ `map[string]interface{}` nào là `JSONB`, dẫn đến việc `createdAt`, `updatedAt` và `_id` bị gán nhầm thành `JSONB` trong PostgreSQL Shadow Table.
- **Quyết định**:
  1. Mở rộng `InferTypeFromRawData` để nhận diện các key BSON Extended JSON v2:
     - Nếu `map` chỉ chứa key `"$date"` -> suy luận thành `TIMESTAMPTZ`.
     - Nếu `map` chỉ chứa key `"$oid"` -> suy luận thành `TEXT` (hoặc `VARCHAR(24)`).
     - Nếu `map` chỉ chứa key `"$numberDecimal"` -> suy luận thành `NUMERIC`.
     - Nếu `map` chỉ chứa key `"$numberLong"` -> suy luận thành `BIGINT`.
  2. Giữ nguyên tính tương thích ngược với JSON thông thường.

## ADR-02: Cơ chế Lấy mẫu Đa chiều (Multi-Stratified Sampling) cho MongoDB Collections
- **Bối cảnh**: MongoDB collections thường có tính chất schema đa hình (polymorphic schema), các document khác nhau (theo thời gian hoặc theo `requestType`) có tập field hoàn toàn khác nhau. Lệnh `Find(..., Limit(10))` lấy 10 record đầu tiên theo insertion order (natural order) nên chỉ thấy schema cũ (`requestData`, `responseData`) mà bỏ lọt schema mới (`bankTransactionId`, `logs`).
- **Quyết định**:
  1. Khi lấy mẫu trực tiếp từ Mongo, kết hợp 2 truy vấn mẫu:
     - Lấy 25 document mới nhất: `Find({}, Sort({"_id": -1}), Limit(25))`
     - Lấy 25 document đầu: `Find({}, Limit(25))`
  2. Gộp tập key từ cả 2 nhóm document để bảo đảm không bỏ sót các trường mới phát sinh trong quá trình tiến hóa hệ thống (Schema Evolution).
