# Bối cảnh Workspace (00_context.md)

> **Workspace**: bug-mapping-rules-and-snapshot-v2-2026-05-29
> **Dự án**: CDC System (GooPay 2026)
> **Trạng thái**: Đang điều tra bối cảnh và chuẩn bị kế hoạch sửa lỗi.

## Vấn đề hiện tại
Hệ thống CDC đang gặp 4 lỗi nghiêm trọng cần khắc phục gấp:

1. **Lỗi Mapping Rules Leak (Bug 1)**:
   - Khi tạo shadow binding mới (ví dụ `wallet_capsets_1` cho source `wallet-capsets`), mặc dù chưa thực hiện quét fields (scan fields), trang mapping (`/shadow/2/mappings`) vẫn hiển thị các mapping rules của shadow binding cũ (`wallet_capsets`).
   - Lý do: Logic API list mapping rules hoặc logic lưu trữ cache trong worker/BFF đang bị gộp chung theo `source_object_id` (hoặc `source_database`/`source_table`) mà không phân biệt theo `shadow_binding_id`.

2. **Thiếu thông tin Data Type Source & Status Sai (Bug 2)**:
   - Cột "Data Type source" trên table hiển thị các fields thiếu thông tin kiểu dữ liệu gốc.
   - Cột "Status" đang hiển thị trạng thái duyệt chung (Approved/Pending), nhưng thực tế Status là trạng thái duyệt của rule, còn In Shadow chỉ là trạng thái audit.
   - Yêu cầu: Thêm cột `source_data_type` vào bảng `cdc_system.mapping_rule_v2` trong DB. Bổ sung logic xác định kiểu dữ liệu từ raw data scan và ghi nhận vào DB. Hiển thị thông tin này trên UI.

3. **Ẩn Action Thừa trên Frontend (Bug 3)**:
   - Cần ẩn 2 nút hành động "Preview" và "Backfill" trong cột "Action" ở trang Mapping Fields (FE).
   - Yêu cầu: Chỉ ẩn (hide/comment/disable conditional) chứ không xóa bỏ hoàn toàn code logic để giữ nguyên cấu trúc và khả năng phục hồi sau này.

4. **Lỗi Snapshot V2 Registry Lookup Fail (Bug 4)**:
   - Khi trigger Snapshot V2 cho shadow binding mới tạo (`shadow_binding_id = 4`), worker báo lỗi:
     `nats-command => shadow_binding_id=4 not in active registry routes for source_db=wallet-service source_collection=wallet-capsets — registry`
   - Lý do: Worker sử dụng cache registry để map route. Khi reload cache, do trùng `source_object_id = 1` giữa các bindings, map `routeBySourceID` bị ghi đè dẫn đến mất hoặc lệch route, hoặc cơ chế reload cache bị race condition/stale.
   - Yêu cầu: Khắc phục triệt để lỗi ghi đè route cache trong `MetadataRegistryService.ReloadAll` và xử lý triệt để bug lookup route trong snapshot runner.
