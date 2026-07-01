# Kế hoạch triển khai: Sửa lỗi mồ côi ảo Segment A và lỗi trigger heal

Mục tiêu là sửa đổi cơ chế lọc timestamp trên MongoDB và sửa câu query report trong `healSegmentA` để bóc tách và giải quyết triệt để 1.410 bản ghi mồ côi ảo do lệch hệ quy chiếu Epoch Millisecond (int64) và lỗi `record not found` khi trigger heal.

## 1. Vá lỗi MongoDB timestamp filter
- Sử dụng `$or` bao phủ cả kiểu `ISODate` (`time.Time`) và `Epoch Millisecond` (`int64`) ở 3 hàm query Mongo: `ListIDsInWindow`, `ListIDTsInWindow`, và `HashWindow`.
- Việc này đảm bảo MongoDB filter quét chính xác và trả về đầy đủ bản ghi, không còn bị 0 bản ghi khi gặp cột timestamp kiểu số nguyên.

## 2. Sửa logic trigger heal Segment A
- Dùng `entry.QualifiedTarget()` (tên FQN của bảng) để query report từ database thay vì dùng tên bảng thô (`table`), tránh lỗi `gorm.ErrRecordNotFound`.
- Bóc tách thêm mảng `missing_from_src` (nếu có) từ report `StaleIDs` và gom chung với `missingIDs` và `mismatched` gửi đi để Debezium đồng bộ lại an toàn.

## 3. Proposed Changes

---

### Component: Recon Source Agent (MongoDB side queries)

#### [MODIFY] [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go)
- Sửa hàm `ListIDsInWindow` (dòng 30-31):
  ```diff
  - tsField := resolveTimestampField(timestampField)
  - filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
  + tsField := resolveTimestampField(timestampField)
  + filter := bson.M{
  + 	"$or": []bson.M{
  + 		{tsField: bson.M{"$gte": tLo, "$lt": tHi}},
  + 		{tsField: bson.M{"$gte": tLo.UnixMilli(), "$lt": tHi.UnixMilli()}},
  + 	},
  + }
  ```
- Sửa hàm `ListIDTsInWindow` (dòng 419-420):
  ```diff
  - tsField := resolveTimestampField(timestampField)
  - filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
  + tsField := resolveTimestampField(timestampField)
  + filter := bson.M{
  + 	"$or": []bson.M{
  + 		{tsField: bson.M{"$gte": tLo, "$lt": tHi}},
  + 		{tsField: bson.M{"$gte": tLo.UnixMilli(), "$lt": tHi.UnixMilli()}},
  + 	},
  + }
  ```

#### [MODIFY] [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go)
- Sửa hàm `HashWindow` (dòng 33-34):
  ```diff
  - tsField := resolveTimestampField(timestampField)
  - filter := bson.M{tsField: bson.M{"$gte": tLo, "$lt": tHi}}
  + tsField := resolveTimestampField(timestampField)
  + filter := bson.M{
  + 	"$or": []bson.M{
  + 		{tsField: bson.M{"$gte": tLo, "$lt": tHi}},
  + 		{tsField: bson.M{"$gte": tLo.UnixMilli(), "$lt": tHi.UnixMilli()}},
  + 	},
  + }
  ```

---

### Component: Recon Handler (Heal logic)

#### [MODIFY] [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go)
- Sửa hàm `healSegmentA` để:
  - Query report bằng `entry.QualifiedTarget()` (tên FQN).
  - Ẩn lỗi `ErrRecordNotFound` trong debug log để tránh gây hoang mang.
  - Bóc tách `staleObj.MissingFromSrc` từ `StaleIDs` và gom tất cả `missingIDs`, `staleObj.Mismatched`, `staleObj.MissingFromSrc` đi heal qua Debezium incremental snapshot signal.

## 4. Verification Plan

### Automated Tests
- Chạy unit tests: `go test ./internal/service/recon/...` và `go test ./internal/handler/recon/...`.

### Manual Verification
- Compile-check service.
- Chạy thử tiến trình đối soát local trên bảng `payment_bills` để xác minh số lượng lệch khớp với thực tế.
- Trigger heal và xác nhận report được tìm thấy, signal gửi đi thành công mà không có log đỏ `record not found`.
