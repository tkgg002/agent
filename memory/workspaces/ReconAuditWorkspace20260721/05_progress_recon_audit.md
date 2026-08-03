# 05 — Nhật Ký Tiến Độ Audit (Audit Log — Append ONLY)

> ⚠️ FILE BẤT DI BẤT DỊCH: Chỉ APPEND, không sửa/xóa nội dung cũ.

---

- [2026-07-21T15:33:00+07:00] [Agent:Gemini-3.5-Pro] Khởi tạo workspace audit độc lập ReconAuditWorkspace20260721 theo yêu cầu của User để rà soát toàn bộ workspace ReconAdaptiveBinaryAsync20260721.
- [2026-07-21T15:33:15+07:00] [Agent:Gemini-3.5-Pro] Khởi tạo bộ tài liệu quy chuẩn (00_context, 01_requirements, 02_plan, 03_implementation, 04_decisions, 05_progress, 06_validation, 07_status_report, 08_tasks, 09_tasks_solution, 10_gap_analysis, 11_report, 12_implementation_plan, 13_analysis).
- [2026-07-21T15:34:00+07:00] [Agent:Gemini-3.5-Pro] Rà soát 14 tệp tin code thuộc dự án centralized-data-service. Phát hiện và sửa 2 lỗi lập trình (sub-window boundary clamping & mock test interface) và xóa 1 file rác (test_write.go).
- [2026-07-21T15:34:30+07:00] [Agent:Gemini-3.5-Pro] Thực thi kiểm thử unit test tự động. Kết quả: internal/service/recon và internal/handler/recon PASSED 100%.
- [2026-07-21T15:35:00+07:00] [Agent:Gemini-3.5-Pro] Hoàn tất xuất báo cáo audit vật lý 11_report_recon_audit.md và 13_analysis_recon_audit.md. Chạy verify_governance.py đạt PASSED 🟢.
- [2026-07-21T16:17:00+07:00] [Agent:Gemini-3.5-Pro] Cập nhật ReconDestAgent (HashWindow & ListIDTsInWindow) hỗ trợ động cả 2 kiểu cột Timestamp trên Shadow Postgres (TIMESTAMPTZ vs TIMESTAMP WITHOUT TIME ZONE). Trích xuất data_type từ information_schema.columns để binding đúng UTC cho TIMESTAMPTZ và dbLoc cho TIMESTAMP naive. Đã thêm unit test TestDestAgent_HashWindow_DomainTS_TimestampWithoutTZ và pass 100%.
- [2026-07-21T16:37:00+07:00] [Agent:Gemini-3.5-Pro] Phân tích 2 nguyên nhân gốc rễ: (1) ReconDestAgent query nhầm cột _id (bigint) thay vì _source_id (string) dẫn tới hash fingerprint không khớp giữa Mongo và Postgres gây 40 drift sub-windows giả; (2) recon_job_worker gán len(drifts) (số sub-windows) làm diff và missing_count trong cdc_reconciliation_report thay vì tổng số record. Đã lập kế hoạch giải pháp chi tiết tại 09_tasks_solution_recon_audit.md và 12_implementation_plan_recon_audit.md.
- [2026-07-21T16:44:00+07:00] [Agent:Gemini-3.5-Pro] Đính chính cấu trúc bảng Shadow: _gpay_id mới là cột Sonyflake, _id là cột từ Mongo gốc. Phát hiện nguyên nhân thực sự gây lệch XorHash: extractMongoIDFromRaw trong recon_hash.go thiếu Int64OK/Int32OK, làm BSON Int64 _id bị format thành JSON {"$numberLong":"504"} thay vì "504". Đã cập nhật lại toàn bộ hồ sơ kỹ thuật 09_tasks_solution_recon_audit.md và 12_implementation_plan_recon_audit.md.
- [2026-07-21T16:52:00+07:00] [Agent:Gemini-3.5-Pro] Đã hoàn thành triển khai fix full loop:
  1. Fix `extractMongoIDFromRaw` hỗ trợ BSON numeric (Int64, Int32, Double) trong `recon_hash.go`.
  2. Implement `resolvePKFields` động `_source_id` vs `_id` trên `ChunkStreamBucketEngine`.
  3. Chuyển `ChunkStreamBucketEngine.Execute` trả về `*ChunkEngineResult` (record-level totals + diffs + missing counts).
  4. Cập nhật `ReconJobWorker` ghi đúng `SourceCount`, `DestCount`, `Diff`, `MissingCount` vào `cdc_reconciliation_report`.
  5. Chạy unit tests: PASS 100% (0.82s).
  6. Chạy live verification via NATS & local DB: ID=68, SrcCount=1230, DstCount=1230, Diff=0, Missing=0, Status=COMPLETED. Hoàn thành 100%.
- [2026-07-21T17:03:00+07:00] [Agent:Gemini-3.5-Pro] Lập kế hoạch bổ sung các trường `total_record_diff_count`, `source_count`, `dest_count` và `stale_ids` (định dạng `{"mismatched": null, "missing_from_master": [...], "missing_from_shadow": null}`) vào `ReconJob`, `cdc_reconciliation_report`, `ChunkEngineResult` và `ReconJobWorker`. Ghi nhận giải pháp tại `09_tasks_solution_recon_audit.md` và `12_implementation_plan_recon_audit.md`.


