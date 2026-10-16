# 01_requirements_smoke_trace_and_trans_his_drift.md
## Yêu cầu Chi tiết: Khắc phục Cây Span Traces Recon Smoke & Xử lý Lệch Số liệu trans_his

### 1. Bối cảnh & Hiện trạng
1. **Lỗi cây Span Traces trên SigNoz (`recon.smoke.cycle_unified`):**
   - Trên SigNoz trace UI, toàn bộ child spans (`prefetch.master`, `prefetch.shadow`, `source_shadow`, `shadow_master`) bị gộp nhầm và lồng vào bên trong pipeline đầu tiên (`export-jobs`).
   - 4 pipelines còn lại (`trans-his` ctphs, `trans-his` mongodevct, `payment_bills`, `payment_bills_1`) không có child spans nào bên trong, hiển thị như các thanh rỗng 273ms ở đáy trace.
   - Nguyên nhân: Cách thức quản lý Span Lifecycle theo mô hình gom gộp tập trung (All-at-once) trong `recon_smoke.go:CheckAllUnified`. Tất cả các pipeline spans được mở cùng microsecond và kết thúc cùng lúc ở cuối chu trình, khiến ngữ cảnh bị rò rỉ và dồn toàn bộ con vào pipeline span đầu tiên.
   - Khóa gom `pipelineFlows` dùng `ShadowRel()` / `QualifiedTarget()` làm mất các master bindings phụ (1 shadow binding có nhiều master binding bị đè mất).
   - Cơ chế khóa Advisory Lock (`withTableLock`) sử dụng tên bảng trần (`entry.TargetTable = "trans_his"`) khiến 2 connector chạy song song bị tranh chấp khóa và 1 bên bị drop ngầm.

2. **Lệch số liệu trên Dashboard đối soát của `trans_his` (`traitestctphs`):**
   - Source (MongoDB): `235,036`
   - Shadow (PostgreSQL): `234,756` (Ingest: `-280`)
   - Master (PostgreSQL): `235,214` (Transmute: `+458`)
   - Bằng chứng toán học rõ ràng: `234,756 (Shadow Active) + 458 = 235,214 (Master Active)`!
     * 458 dòng trong Shadow đã bị đánh dấu `_deleted = true` (xóa mềm từ nguồn).
     * Tầng Shadow tính active = `total - del = 235,214 - 458 = 234,756`.
     * Nhưng Transmuter chạy incremental cursor theo `_gpay_id > cursor`. Các bản ghi xóa mềm này có `_gpay_id` cũ nên cursor không quét lại, dẫn đến chúng vẫn tồn tại sống trên Master (`235,214`).
     * Chênh lệch Source (235,036) vs Shadow (234,756) là do MongoDB `EstimatedDocumentCount()` đọc metadata WiredTiger có độ trôi dạt (metadata drift) thay vì đếm chính xác.

### 2. Yêu cầu Nghiệp vụ & Kỹ thuật
1. **Chuẩn hóa Span Traces Phân cấp Độc lập (Per-Pipeline Execution & Trace Lifecycle):**
   - Chuyển đổi hoàn toàn mô hình lifecycle của Span Trace: Bỏ cơ chế mở hàng loạt pipeline spans cùng microsecond ở đầu hàm.
   - Định danh Pipeline duy nhất theo cặp `[PipelineKey] = [ShadowRel]::[MasterRel]` để không bị đè khi 1 Shadow Table có nhiều Master Table (như `export_jobs` có `export_jobs`, `export_jobs_1`, `export_jobs_2`).
   - Mỗi Pipeline chạy khép kín: Mở Span `pipeline: [...]`, chạy Segment A (nếu chưa scan), chạy Segment B, đóng Span ngay khi pipeline đó hoàn tất.
   - Từng pipeline hiển thị trên SigNoz thành một cây nhánh độc lập, chứa đúng `source_shadow`, `shadow_master` và các span con của chính nó.

2. **Sửa Advisory Lock cho Bảng Đa Kết nối:**
   - Đổi `rc.withTableLock(ctx, entry.TargetTable)` thành `rc.withTableLock(ctx, entry.QualifiedTarget())`.
   - Đổi `rc.withTableLock(ctx, "reconB_"+ref.runName())` thành `rc.withTableLock(ctx, "reconB_"+ref.ShadowRel()+"_"+ref.MasterRel())`.
   - Đảm bảo 2 connector cùng bảng `trans_his` không khóa đè lên nhau.

3. **Xử lý Triệt để Lệch Số liệu Master (+458) và Đối soát Source (-280):**
   - Cơ chế Dọn dẹp Tombstone Master (Orphan Cleanup / Hard Delete Sync):
     Cung cấp cơ chế đối soát hoặc trigger lệnh dọn dẹp các dòng `_deleted = true` từ Shadow sang Master để Master xóa sạch 458 dòng đã bị xóa ở Shadow, đưa Master về đúng `234,756`.
   - Chế độ Verify Exact Count cho Smoke Check khi phát hiện drift:
     Nếu `EstimatedDocumentCount` lệch so với Shadow, cho phép đối soát chính xác hoặc hiển thị chú thích O(1) metadata drift rõ ràng.
