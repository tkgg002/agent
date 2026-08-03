# Yêu Cầu Sửa Lỗi: Scan Fields Không Quét Được Cột ID Của Bảng PostgreSQL

## 1. Hiện Trạng & Sự Cố (Problem Statement)
Trên giao diện `/shadow` (`http://localhost:5173/shadow`), khi nhấn nút **Scan Fields** cho một bảng PostgreSQL (ví dụ bảng có primary key `id`), danh sách các trường được phát hiện và thêm vào `mapping_rules_v2` hoàn toàn thiếu cột `id`.

## 2. Nguyên Nhân Gốc Rễ (Root Cause Analysis)
1. **Tại `centralized-data-service/internal/handler/source/discovery_utils.go` (`inferPGCols`, `inferMySQLCols`, `inferMongoCols`)**:
   Code cũ chứa logic bỏ qua cột primary key:
   ```go
   if strings.EqualFold(name, pkColumn) {
       continue
   }
   ```
   Do `pkColumn` mặc định hoặc được cấu hình là `"id"`, câu lệnh `continue` đã **lọc bỏ trực tiếp cột `id`** ra khỏi danh sách kết quả scan fields.

2. **Tại `centralized-data-service/internal/handler/source/discover_handler_utils.go` (`processDiscoveryRows`)**:
   Khi scan từ `_raw_data` chứa Debezium CDC payload, `doc` ở dạng `{"before": ..., "after": {"id": 123, ...}}`. Code cũ chỉ lấy key ở root level `doc` mà không unwrap `after`, dẫn tới không đọc được các field trong payload `after` khi shadow `_raw_data` chưa được unwrap.

## 3. Phạm Vi Sửa Đổi (Scope)
- Sửa `centralized-data-service/internal/handler/source/discovery_utils.go`:
  - Loại bỏ điều kiện skip `pkColumn` trong `inferPGCols`, `inferMySQLCols`, và `inferMongoCols` để giữ lại 100% các cột nguồn (bao gồm cả `id`).
- Sửa `centralized-data-service/internal/handler/source/discover_handler_utils.go`:
  - Unwrap key `after` từ `doc` nếu `_raw_data` chứa Debezium envelope payload (`after`).
