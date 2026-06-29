# Architecture Decisions: Reconcile Overhaul (2026-06-25)

## ADR-01: Giữ nguyên cấu trúc bảng cdc_reconciliation_report & Tối ưu logic ghi
- **Bối cảnh**: Bảng `cdc_reconciliation_report` có lịch sử thay đổi lớn và phình to do log rác `ok`. Tuy nhiên, thay đổi hoàn toàn cấu trúc cột/bảng sẽ phá vỡ tính tương thích ngược với dashboard của `cdc-cms-service`.
- **Quyết định**: Giữ nguyên tên cột và cấu trúc bảng của migration 085. Thay vào đó, áp dụng giải pháp tối ưu logic ghi (Smart Write / Deduplication) để cập nhật mốc thời gian của dòng `ok` hiện tại thay vì chèn mới, và bổ sung job tự động dọn dẹp các dòng OK cũ hơn 7 ngày.
- **Hệ quả**: Giảm 99% lượng ghi rác vào DB, bảo toàn tính tương thích 100% với read-side API và CMS dashboard mà không cần sửa đổi lớn ở CMS service (minimal impact).
