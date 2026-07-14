# Bối cảnh & Phạm vi (Context & Scope)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

### Vấn đề
- Hệ thống cũ thực hiện chữa lành (recon heal) tự động hoặc thông qua nút "Chữa lành" chạy ngầm Tier 2 check và tự động giải quyết không cho phép người dùng chọn lựa các hành động cụ thể (mismatched, missing_dest, prune_src).
- Nút "Chữa lành" cũ yêu cầu chọn khoảng thời gian (startTime, endTime), chế độ quét và lookback, gây kẹt luồng và vi phạm nguyên lý Single Responsibility Principle (SRP).

### Giải pháp
- Thay thế hoàn toàn luồng chữa lành cũ bằng luồng chữa lành tương tác mới (Interactive Heal):
  - Người dùng bấm nút "Chữa lành" trên UI sẽ hiển thị danh sách các phiên chưa được chữa lành (unhealed reports) cho table đó.
  - Người dùng chọn các checkbox hành động:
    1. `Heal Mismatched` (Chữa lành bản ghi sai khác)
    2. `Heal Missing Dest` (Chữa lành bản ghi thiếu ở đích)
    3. `Prune Missing Src` (Xóa bản ghi thừa ở đích)
  - Lý do chữa lành (Reason) phải được điền tối thiểu 10 ký tự.
  - Khi xác nhận, hệ thống gửi lệnh qua NATS subject `cdc.cmd.recon-heal` chứa danh sách report_ids cần xử lý cùng cấu hình checkboxes hành động.
  - Phía worker nhận lệnh và thực thi tuần tự, lưu trữ thống kê số lượng bản ghi tương tác chữa lành vào DB (`088_recon_interactive_heal_stats.sql`).
