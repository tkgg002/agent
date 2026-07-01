# Plan: ReconTimestampWindowing

Kế hoạch chi tiết sửa đổi mã nguồn:

## 1. Thêm Post-Processing kiểm tra chéo trong RunTier2 (recon_tier_a.go)
- Thực hiện kiểm tra sự tồn tại thực tế của các ID trong `missingFromDest` từ Shadow DB.
- Di chuyển các ID tồn tại sang `mismatchedFromDest` (Fake Missing -> Mismatched).
- Loại bỏ các ID này khỏi `missingFromSrc` (Fake Orphan -> Loại bỏ).
- Cập nhật thống kê `StaleCount` bằng tổng của `mismatchedFromDest` và `missingFromSrc`.

## 2. Cập nhật điều kiện lọc của bộ chữa lành (recon_heal_v4.go)
- Sửa đổi 3 chốt chặn an toàn trong `healSegmentA` và `healSegmentB` để đảm bảo bao gồm kiểm tra `OrphanCount == 0`.

## 3. Chạy unit tests kiểm chứng
- Thực hiện chạy `go test` trên package `service/recon` và `handler/recon`.
