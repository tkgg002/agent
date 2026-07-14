# Yêu cầu: Chữa lành đối soát phân rã trạng thái & Sửa lỗi heal Segment B

Yêu cầu chi tiết cho việc sửa đổi logic chữa lành đối soát và hiển thị.

## 1. Yêu cầu nghiệp vụ và UI
- **Done toàn bộ mới chuyển sang "Phiên đã xử lý"**:
  Một phiên đối soát (report) chứa tối đa 3 loại lỗi cần chữa lành: Thiếu (missing), Lệch (mismatched), và Thừa (orphan).
  Chỉ khi nào cả 3 loại này đều được chữa lành thành công (counts = 0 hoặc `status = 'healed'`), report mới được chuyển sang tab "Phiên đã xử lý".
- **Chưa hoàn thành toàn bộ vẫn ở "Phiên chưa xử lý"**:
  Nếu report chưa hoàn thành hoàn toàn (status `partially_healed` hoặc vẫn còn lỗi chưa sửa), report phải ở lại tab "Phiên chưa xử lý".
  UI hiển thị cập nhật lại các chỉ số còn lại (Thiếu / Lệch / Thừa) để thể hiện rõ đã thực hiện cái nào.
  Tab "Phiên đã xử lý" không được hiển thị các phiên `partially_healed`.

## 2. Sửa lỗi heal Segment B (Shadow -> Master)
- Hiện tại, trong hàm `executeHealSegB` của backend:
  `rpt.HealedMissingDestCount = len(missingGpayIDs)` và `rpt.HealedMismatchedCount = len(staleB.Mismatched)` được gán trực tiếp mà không kiểm tra xem việc ánh xạ ID và publish transmute có thành công hay không.
  Nếu `mapGpayToSourceIDs` trả về lỗi (ví dụ: `invalid shadow relation`), logic chữa lành bị lỗi nhưng DB vẫn lưu nhận là đã chữa lành, dẫn đến hiện tượng UI báo đã chữa lành nhưng bản ghi thực tế chưa được thêm vào Master.
  **Giải pháp:** Chỉ gán số lượng đã heal dựa trên số lượng ID được publish thành công sang NATS. Ghi log lỗi rõ ràng khi mapping ID thất bại.
