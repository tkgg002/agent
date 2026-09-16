# 13_analysis_recon_no_index.md

## Phân tích phản biện chuyên sâu (Adversarial Analysis)

### 1. Tại sao KHÔNG ĐƯỢC chỉ đổi sang `_id: ObjectId` vô điều kiện?
- **Khuyết điểm 1: Lệch miền dữ liệu (Data Domain Shift):**
  `_id` thể hiện thời gian tạo (Insert Time). Trường `_source_ts` ở Postgres thể hiện thời gian xảy ra sự kiện CDC (bao gồm cả Update).
  Nếu bảng có update bản ghi cũ, query theo `_id` trên Mongo sẽ miss các bản ghi update đó, trong khi Postgres vẫn lấy theo `_source_ts`. Dẫn đến hash mismatch 100% -> False Drift.
- **Khuyết điểm 2: Type Incompatibility:**
  Nếu `_id` là String (UUID) hoặc Number, query so sánh với `primitive.ObjectID` trả về 0 kết quả do BSON type sorting.
- **Khuyết điểm 3: Vi phạm Single Source of Truth:**
  Không suy diễn mọi collection đều dùng `ObjectId` và đều append-only. CDC platform generic phải hoạt động đúng cho N collections.

### 2. Định lượng hiệu năng
- Khi tăng `BatchSize` từ 1.000 lên 5.000: Giảm 80% network round-trips giữa Mongo và worker.
- Khi nâng `QueryTimeout` lên 120s: Cho phép MongoDB hoàn thành COLLSCAN 15M records trên replica node mà không bị hủy ngang.
