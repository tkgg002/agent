# Yêu Cầu: Sửa Lỗi Drift Giả (False Positive Drift) và Đồng Bộ Trạng Thái Đối Soát

## Bối Cảnh
Khi thực hiện đối soát Reconciliation (bấm nút check heal hoặc chạy luồng đối soát), mặc dù không có bất kỳ dòng dữ liệu nào thực sự bị lệch (`mismatches == 0`), hệ thống vẫn báo cáo trạng thái là `drift`.

## Nguyên Nhân Gốc Rễ
1. **Lỗi scan tạm thời hoặc độ trễ đồng bộ (Replication Lag):**
   - Trong quá trình đối soát, do độ trễ đồng bộ (CDC lag) tạm thời giữa MongoDB và Postgres DB, hoặc do micro-lag giữa các câu lệnh truy vấn, hàm tính XOR hash `HashWindow` ghi nhận sự lệch hash/count giữa 2 bên, làm tăng số lượng cửa sổ bị lệch (`driftedWindows > 0`).
   - Tuy nhiên, khi tiến hành drill-down bằng `ListIDTsInWindow` ở bước tiếp theo, dữ liệu có thể đã được đồng bộ hoàn tất hoặc kết quả so khớp chi tiết ID không tìm thấy bất kỳ dòng nào bị lệch thật sự (`mismatches == 0`).
   - Nhưng do logic gán trạng thái trong `recon_tier_a.go` chỉ kiểm tra `driftedWindows > 0` để báo trạng thái `"drift"`, hệ thống đã hiển thị trạng thái `"drift"` giả trên UI mặc dù danh sách mismatch hoàn toàn rỗng.

2. **Bất đối xứng cấu trúc truy vấn drill-down (Phòng ngừa tiềm ẩn):**
   - Khi tính toán XOR hash trong `HashWindow`, hệ thống loại bỏ các hàng đã bị xóa mềm (`_deleted = true`) hoặc có timestamp NULL.
   - Nhưng phương thức `ListIDTsInWindow` ở phía Đích (`ReconDestAgent`) lại thiếu điều kiện lọc `NOT "_deleted"` và `IS NOT NULL`. Mặc dù hiện tại dữ liệu không có record `_deleted = true`, sự bất đối xứng này vẫn là một lỗ hổng logic cần được chuẩn hóa để tránh lỗi lệch dữ liệu drill-down khi có dữ liệu xóa mềm trong tương lai.

3. **Thiếu đồng bộ logic statusStr ở Segment A:**
   - Trong Segment B (`recon_tier_b.go`), nếu `mismatches == 0 && driftedWindows > 0` (lỗi scan/lag nhưng không có diff), trạng thái được chuyển thành `error` thay vì báo `"drift"` giả.
   - Nhưng trong Segment A (`recon_tier_a.go`), logic này chưa được áp dụng, dẫn đến trạng thái vẫn bị gán là `"drift"`.

## Yêu Cầu Chi Tiết (Definition of Done)
1. **Sửa `ListIDTsInWindow` phía Đích (`ReconDestAgent` trong `recon_dest_query.go`):**
   - Thêm điều kiện `AND NOT "_deleted"` và `AND <ts_col> IS NOT NULL` vào các truy vấn lấy danh sách ID + Timestamp để đồng bộ hoàn toàn với logic tính `HashWindow`.
2. **Đồng bộ logic gán `statusStr` cho Segment A (`recon_tier_a.go`):**
   - Nếu `driftedWindows > 0` nhưng tổng số mismatch cụ thể tìm thấy bằng 0, gán trạng thái là `error` thay vì `drift`, tương tự như Segment B.
3. **Đồng bộ logic gán `statusStr` cho Segment B (`recon_tier_b.go`):**
   - Đảm bảo logic gán trạng thái `error` khi có drift window nhưng mismatch = 0 hoạt động chuẩn xác ở cả 2 phương thức check của Segment B.
4. **Kiểm tra chất lượng (Verification):**
   - Đảm bảo code được build thành công.
   - Chạy các unit test liên quan của recon (`recon_tier_a_test.go`, `recon_tier_b_test.go`, v.v.) và đảm bảo chúng vượt qua.
