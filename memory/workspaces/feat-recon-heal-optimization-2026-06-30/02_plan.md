# Plan: Sửa lỗi mồ côi ảo Segment A và lỗi trigger heal

## 1. Nghiên cứu & Khảo sát (Research Phase)
- [x] **Hành động 1.1**: Đọc `internal/service/recon/recon_stream.go` để xem các method của `ReconSourceAgent`.
- [x] **Hành động 1.2**: Đọc `internal/service/recon/recon_hash.go` để xem logic MongoDB hash window.
- [x] **Hành động 1.3**: Đọc `internal/handler/recon/recon_heal_v4.go` để xem hàm `healSegmentA` và vị trí cần sửa.

## 2. Thiết kế Giải pháp (Design Phase)
- [x] **Hành động 2.1**: Thiết kế filter `$or` bao phủ cả `time.Time` và `int64` cho MongoDB.
- [x] **Hành động 2.2**: Thiết kế sửa đổi `healSegmentA` để query bằng `QualifiedTarget()` và gom `MissingFromSrc` đi heal.

## 3. Thực thi & Sửa code (Execution Phase)
- [ ] **Hành động 3.1**: Thay đổi filter MongoDB trong `ListIDsInWindow` (`recon_stream.go`).
- [ ] **Hành động 3.2**: Thay đổi filter MongoDB trong `ListIDTsInWindow` (`recon_stream.go`).
- [ ] **Hành động 3.3**: Thay đổi filter MongoDB trong `HashWindow` (`recon_hash.go`).
- [ ] **Hành động 3.4**: Sửa đổi `healSegmentA` trong `recon_heal_v4.go` theo thiết kế.

## 4. Xác minh (Verification Phase)
- [ ] **Hành động 4.1**: Chạy `go build` và `go test ./...` để đảm bảo code compile thành công.
- [ ] **Hành động 4.2**: Chạy thử đối soát và trigger heal trên môi trường dev local để xác nhận lỗi được khắc phục hoàn toàn.
