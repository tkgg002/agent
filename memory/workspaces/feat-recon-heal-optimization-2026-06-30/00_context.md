# Context: Cải tiến và Tối ưu hóa cơ chế Recon & Heal (Segment A & B) cùng logic Soft-delete ở Master

## Hiện trạng
1. **Đối soát & Heal Segment A (Source Mongo ↔ Shadow Postgres)**:
   - Tiến trình đối soát Tier 2 (`RunTier2` trong `recon_tier_a.go`) so sánh danh sách ID đơn thuần qua `diffIDs`. Do đó, nếu dữ liệu giữa Mongo và Shadow bị lệch timestamp nhưng trùng khớp ID, engine không ghi nhận và không báo cáo các bản ghi bị lệch dữ liệu này (chỉ báo `StaleCount` bằng số lượng window bị drift, nhưng danh sách ID lệch không được thu thập).
   - Khi click heal trên UI, `healSegmentA` chỉ xử lý các ID thiếu hoàn toàn ở shadow (`MissingIDs`). Nó bỏ qua hoàn toàn các ID bị lệch dữ liệu (`stale`). Nếu `MissingCount == 0`, nó sẽ trả về `noop` ngay cả khi có bản ghi bị lệch.
   - Tập dữ liệu MongoDB `payment_bills` rất lớn (~50 triệu records). Việc đối soát và heal cần đảm bảo an toàn tải, không thực hiện CollScan trên Mongo (yêu cầu index trên trường timestamp) và giới hạn/phân trang số lượng heal.

2. **Đối soát & Heal Segment B (Shadow Postgres ↔ Master Postgres)**:
   - Trong `RunSegmentB` (`recon_tier_b.go`), các record bị lệch timestamp được lưu vào `staleIDs`. Tuy nhiên, trong code lưu report, trường `StaleIDs` bị ghi đè bởi danh sách `orphanInMaster`. Danh sách `staleIDs` bị mất hoàn toàn.
   - Do đó, `healSegmentB` chỉ heal các ID bị thiếu, bỏ qua hoàn toàn các ID bị lệch timestamp.

3. **Đồng bộ Soft-delete ở Master Table**:
   - Khi tiến trình Transmute chạy (Shadow -> Master), nó cần đồng bộ và cập nhật cột `_deleted = true` ở table master khi bản ghi ở shadow được đánh dấu soft-delete (`_deleted = true`).

## Mục tiêu
1. Sửa đổi `RunTier2` để so sánh chi tiết ID và timestamp (`ListIDTsInWindow`), tách biệt `missingFromDest`, `missingFromSrc` và `mismatchedIDs` (stale records).
2. Sửa đổi `healSegmentA` để trigger debezium snapshot cho cả missing IDs và mismatched (stale) IDs. Giới hạn số lượng ID được heal mỗi lần để đảm bảo an toàn chịu tải cho tập dữ liệu lớn.
3. Sửa đổi `RunSegmentB` để lưu danh sách `staleIDs` (lệch timestamp) vào report một cách có cấu trúc mà không bị ghi đè.
4. Sửa đổi `healSegmentB` để gửi trigger re-transmute cho cả missing và stale IDs.
5. Cập nhật transmuter master để đồng bộ trạng thái `_deleted = true` sang Master table.
