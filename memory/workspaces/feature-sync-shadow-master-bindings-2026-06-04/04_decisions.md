# Architectural Decisions - Đồng bộ & Duyệt Rules trong Transaction khi Approve Master Binding

Tài liệu này ghi lại quyết định thiết kế kiến trúc để giải quyết vấn đề bảng vật lý trống (chỉ có cột hệ thống mặc định) khi duyệt Master Binding.

## Ngữ cảnh
Khi một Master Binding được duyệt (Approve), hệ thống phát một JetStream command `cdc.cmd.master-create` sang Worker. Worker sẽ đọc các rules từ `mapping_rule_master` của binding đó để thực thi lệnh `ALTER TABLE ADD COLUMN` tạo các cột vật lý trên DB đích.
Tuy nhiên, nếu các rules này chưa được đồng bộ từ shadow mapping rules sang master, hoặc chưa được cập nhật trạng thái `approved`, Worker sẽ không nhận dạng được các cột nghiệp vụ cần tạo, dẫn đến việc bảng đích được tạo ra chỉ có các cột hệ thống mặc định.

## Quyết định thiết kế
1. **Đồng bộ tự động trong Transaction**:
   - Khi API `/api/v1/masters/:name/approve` được gọi, logic handler trong `approve_master.go` sẽ thực thi toàn bộ luồng xử lý trong một database transaction duy nhất (`h.db.Transaction`).
   - Transaction này sẽ thực hiện:
     - Clone các rules từ `mapping_rule_v2` sang `mapping_rule_master` sử dụng `INSERT INTO ... SELECT ... ON CONFLICT DO NOTHING`.
     - Cập nhật tất cả các rules của master binding này sang trạng thái `approved`.
     - Lưu trạng thái Master Binding thành `approved`.
   - Bằng cách này, tính nhất quán dữ liệu được đảm bảo hoàn toàn: rules sẽ được sync và duyệt trước khi command bus phát tán lệnh đi.

2. **Xử lý Command Bus ngoài Transaction**:
   - Lệnh phát command `cdc.cmd.master-create` sang NATS JetStream chỉ được thực thi **sau khi transaction đã được commit thành công** (`err == nil`).
   - Tránh việc phát đi command khi transaction chưa commit (nếu rollback, worker sẽ đọc dữ liệu cũ chưa sync).

3. **Cơ chế Idempotency**:
   - Tận dụng `ON CONFLICT DO NOTHING` dựa trên unique constraint của bảng `mapping_rule_master` để tránh bị trùng lặp dữ liệu khi chạy lại nhiều lần.

## Hệ quả
- Đảm bảo tính nhất quán (Consistency) cao giữa trạng thái logic của rule và bảng vật lý đích.
- Worker luôn tạo bảng vật lý đích với đầy đủ các cột nghiệp vụ đã được định nghĩa.
