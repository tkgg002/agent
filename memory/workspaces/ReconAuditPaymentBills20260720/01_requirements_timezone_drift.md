# 01 — Yêu cầu: Khắc phục triệt để Timezone Drift trên Production cho payment_bills

> Tạo: 2026-07-20T17:30:00+07:00 | Task: Hotfix/Refactor

## Phạm vi

Khắc phục triệt để lỗi timezone drift gây lệch XOR Hash ở chặng đối soát Segment A (source ↔ shadow) trên production cho bảng `payment_bills` có trường domain timestamp `lastUpdatedAt`.

## Yêu cầu cụ thể

1. **Xử lý case-sensitive/timezone-aware parsing:** Tự động phát hiện kiểu dữ liệu của cột timestamp trong PostgreSQL shadow table (`TIMESTAMP` vs `TIMESTAMPTZ`).
2. **Chuẩn hóa timestamp parse:**
   - Nếu kiểu dữ liệu là `TIMESTAMPTZ` (with timezone), giữ nguyên giờ UTC thực tế mà driver pgx scan được, không áp dụng dịch chuyển múi giờ.
   - Nếu kiểu dữ liệu là `TIMESTAMP` (without timezone), áp dụng dịch chuyển múi giờ dựa trên detected DB timezone để đưa về UTC vật lý đúng.
3. **Caching kiểu cột:** Cache kết quả truy vấn kiểu dữ liệu cột trong `ReconDestAgent` để tránh thâm hụt hiệu năng (slow query `information_schema`).
4. **Đồng bộ hóa test suite:** Cập nhật unit tests tương ứng để phản ánh logic parse mới.

## Definition of Done (DoD)

- [ ] Viết hàm detect kiểu dữ liệu cột `IsColTimestamptz` trong `ReconDestAgent` có tích hợp cache an toàn bằng mutex.
- [ ] Refactor logic parse timestamp trong `HashWindow` (`recon_dest_hash.go`) và `ListIDTsInWindow` (`recon_dest_query.go`).
- [ ] Chạy `go build` thành công.
- [ ] Chạy `go test ./internal/service/recon/...` thành công.
- [ ] Ghi nhận đầy đủ lịch sử progress và report.
