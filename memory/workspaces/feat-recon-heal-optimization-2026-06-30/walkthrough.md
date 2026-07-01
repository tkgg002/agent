# Walkthrough: Sửa lỗi mồ côi ảo Segment A và lỗi trigger heal

Tài liệu này tóm tắt kết quả thực hiện các chỉnh sửa lỗi mồ côi ảo và lỗi query report khi trigger heal.

## Các thay đổi đã thực hiện

### 1. `internal/service/recon/recon_stream.go` & `recon_hash.go`
- **Sửa đổi**: Cập nhật bộ lọc MongoDB trong 3 hàm `ListIDsInWindow` (`recon_stream.go`), `ListIDTsInWindow` (`recon_stream.go`), và `HashWindow` (`recon_hash.go`).
- **Chi tiết**: Thay thế filter cứng cũ bằng filter dùng toán tử `$or` bao phủ cả kiểu `ISODate` (`time.Time`) và `Epoch Millisecond` (`int64`):
  ```go
  filter := bson.M{
      "$or": []bson.M{
          {tsField: bson.M{"$gte": tLo, "$lt": tHi}},
          {tsField: bson.M{"$gte": tLo.UnixMilli(), "$lt": tHi.UnixMilli()}},
      },
  }
  ```
- **Mục tiêu**: Đảm bảo query MongoDB bắt chính xác các bản ghi sử dụng Epoch Millisecond, triệt tiêu hoàn toàn 1.410 mồ côi ảo do lệch hệ quy chiếu.

### 2. `internal/handler/recon/recon_heal_v4.go`
- **Sửa đổi**: Sửa hàm `healSegmentA`.
- **Chi tiết**:
  - Dùng `targetFQN := entry.QualifiedTarget()` (tên FQN của bảng) để query report từ database thay vì dùng tên bảng thô (`table`), loại bỏ lỗi `gorm.ErrRecordNotFound`.
  - Ẩn lỗi `ErrRecordNotFound` trong warning log để tránh spam log đỏ gây hiểu lầm.
  - Bóc tách mảng `missing_from_src` từ report `StaleIDs` (nếu có) và gom chung với `missingIDs` và `staleObj.Mismatched` làm danh sách `healIDs` gửi đi để Debezium đồng bộ lại an toàn.

---

## Kết quả kiểm thử (Verification Results)

Tất cả các unit test và compilation check đều đã thành công 100%:

```bash
go build ./internal/...
go test ./internal/...
```

**Output**:
- Build compile thành công không có lỗi.
- Toàn bộ unit tests của project (bao gồm package `recon` và `master`) đều PASS:
  ```
  ok  	centralized-data-service/internal/service/master	0.614s
  ok  	centralized-data-service/internal/service/recon	0.915s
  ok  	centralized-data-service/internal/handler/recon	1.325s
  ```
