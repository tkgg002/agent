# Kế hoạch Sửa đổi Logic Timestamp Đối soát Segment B (Tránh Drift Giả)

## 1. Phân tích nguyên nhân gốc rễ
Hiện tại, đối soát Segment B (chặng Shadow ↔ Master) bị báo lệch **462 bản ghi missing** giả tạo vì lý do sau:
1. Logic watermark và truy vấn ID (`measureAndResolveWatermarksB`, `ListIDTsInWindow`, `TimeBoundedDiffMissingFromMaster`) đang **gán cứng** sử dụng cột kĩ thuật `_source_ts` của CDC.
2. Trong Master DB nghiệp vụ, do đồng bộ/import ban đầu, các bản ghi cũ bị gán cứng `_source_ts` ở mốc ngày **2026-07-02**. Còn ở Shadow DB, các bản ghi có `_source_ts` được cập nhật liên tục theo CDC events (ngày **2026-07-09**).
3. Khi đối soát theo dải thời gian 7 ngày gần nhất, phía Shadow load lên 462 bản ghi mới, trong khi phía Master load lên 0 bản ghi do mốc `_source_ts` của Master bị nằm ngoài dải check. Hệ thống kết luận sai lệch 462 bản ghi missing.

---

## 2. Giải pháp kỹ thuật
Chuyển đổi hoàn toàn đối soát Segment B từ cột kĩ thuật `_source_ts` sang cột **timestamp nghiệp vụ thực tế** (như `updated_at` hoặc `last_updated_at`) được lấy từ `TableRegistry`:

1. **Resolve timestamp nghiệp vụ trong `ReconCore`**:
   Sử dụng `registryRepo.GetByTargetTable` để tìm registry entry tương ứng, sau đó gọi `resolveSourceAndDestTSFields` để lấy ra cột `dstTS` nghiệp vụ thực tế đã được validate tồn tại trên DB.
2. **Cập nhật `measureAndResolveWatermarksB`**:
   - Sửa đổi chữ ký để trả thêm cột timestamp resolved `tsCol`.
   - Sử dụng `tsCol` này cho các hàm `MaxWindowTs`.
3. **Cập nhật `RunHashWindowCheckB` và `RunDeepCheckB`**:
   - Sử dụng `tsCol` trả về từ `measureAndResolveWatermarksB` để truyền vào `BucketCounts` và `ListIDTsInWindow` thay vì gán cứng `_source_ts`.
4. **Cập nhật `TimeBoundedDiffMissingFromMaster`**:
   - Thực hiện resolve `tsCol` tương tự.
   - Sửa câu query SQL: nếu dùng `tsCol` nghiệp vụ, bind biến kiểu `time.Time` và so sánh theo cột nghiệp vụ đó.

---

## 3. Các file cần sửa đổi
- `internal/service/recon/recon_tier_b.go`:
  - Cập nhật hàm `measureAndResolveWatermarksB`.
  - Cập nhật hàm `RunHashWindowCheckB` và `RunDeepCheckB` nhận và truyền `tsCol`.
  - Cập nhật hàm `TimeBoundedDiffMissingFromMaster` hỗ trợ query theo cột nghiệp vụ.

---

## 4. Kế hoạch kiểm thử & Xác minh (DoD)
1. **Biên dịch:** `go build ./cmd/...` thành công.
2. **Unit Tests:** `go test -v ./internal/service/recon/...` PASS 100%.
3. **Chạy thử đối soát thực tế:** Bắn message NATS check Segment B. Kết quả mong đợi: Hệ thống đối soát dựa trên `updated_at` (khớp nhau giữa Shadow và Master), trả về **`missing_count = 0`**, **`stale_count = 0`** và status **`ok`**! (Khắc phục triệt để lỗi drift giả tạo).
