# Walkthrough: Xoá Shadow & Xoá Master

## 1. Kết quả thực hiện
- Đã bổ sung các nút xoá với modal xác nhận `ConfirmDestructiveModal` ở Frontend:
  - **Xoá Shadow Binding**: Xuất hiện ở cả hai nơi: tab **Shadow Objects** (dưới cột *Shadow Actions*) và tab **Shadow Bindings** (dưới cột *Action*). Cho phép xoá shadow binding, cascade xoá toàn bộ master bindings cùng các rules đi kèm. Tự động khoá nút nếu shadow đang active (`is_active = true`).
  - **Xoá Master Binding** (trang Masters): Cho phép xoá master binding và các rules đi kèm, giữ nguyên shadow binding. Tự động khoá nút nếu master đã approved.
- **Xoá bảng vật lý ở DB Master**: Bổ sung method `DropTable` vào `MasterDDLGenerator` của Worker (`centralized-data-service`). Khi xoá Master Binding hoặc cascade xoá Shadow Binding, Backend (`cdc-cms-service`) sẽ gửi NATS command `cdc.cmd.master-alter-column` với `action="drop_table"` sang Worker. Worker sẽ nạp thông tin kết nối đích chính xác và thực thi lệnh SQL `DROP TABLE IF EXISTS "schema"."table" CASCADE` trên database Master đích.
- **Xoá bảng vật lý Shadow & Dọn dẹp Metadata**: Khi xoá Shadow Binding, API Service thực hiện:
  - Drop bảng vật lý shadow trực tiếp trên database Shadow (`DROP TABLE IF EXISTS "schema"."table" CASCADE`).
  - Xoá sạch record trong bảng legacy metadata `cdc_system.cdc_table_registry` (V1) và `cdc_system.source_object_registry` (V2) để dọn dẹp các ràng buộc duy nhất `(source_db, source_table, target_table)`, giúp người dùng có thể đăng ký lại cặp này mà không bị báo lỗi trùng lặp.
- **Cơ chế Saga / Transaction 2PC:**
  - Áp dụng database transaction (`tx := db.Begin()`) cho cả hai luồng xoá Master và xoá Shadow.
  - Các thay đổi dữ liệu metadata chỉ được lưu trữ (Commit) khi và chỉ khi Worker phản hồi lệnh DROP TABLE vật lý đích thành công qua NATS.
  - Nếu quá trình drop table gặp lỗi hoặc timeout, giao dịch metadata sẽ tự động `Rollback` giúp cơ sở dữ liệu metadata không bị xoá mất dấu, tránh tình trạng để lại bảng rác mồ côi (orphaned table).
- Cập nhật các modal truyền `reason` vào request body (`{ data: { reason } }`) và truyền header `Idempotency-Key` để audit log ghi nhận chính xác lý do thực hiện.
- Sửa lỗi cú pháp thiếu dấu ngoặc đóng và fix lỗi typescript TS6133 unused variable ở file `SourceConnectors.tsx` (FE).

## 2. Kết quả Verify & Audit
- Cả Backend (`go build ./internal/...` cho cả API và Worker) và Frontend (`npm run build`) đều build thành công, không phát sinh bất kỳ lỗi cú pháp hay kiểu dữ liệu nào.
- Linter quy trình đạt kết quả: `GOVERNANCE AUDIT PASSED`.
- Bản báo cáo đối soát chi tiết và đánh giá Gaps đã được tạo tại [gap_analysis_audit.md](file:///Users/trainguyen/.gemini/antigravity/brain/a6806c3d-3b49-4af5-a817-46c27fa464c1/gap_analysis_audit.md). Quá trình thực thi hoàn toàn khớp và vượt tiêu chuẩn thiết kế ban đầu để đảm bảo an toàn giao dịch tối đa.
