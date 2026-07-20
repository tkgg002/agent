# Tiến độ sửa lỗi Timezone Drift trong Recon Pipeline

## Phân tích nguyên nhân gốc rễ (Root Cause Analysis)
- **Triệu chứng:** Hệ thống báo cáo lệch `HashWindow` (Drift) giả mạo tại các dải thời gian dù dữ liệu thực tế đã đồng bộ hoàn chỉnh.
- **Nguyên nhân gốc rễ:** Hàm `parsePostgresTimestamp` ở file `recon_query.go` sử dụng `time.Date(...)` với `time.UTC` khi parse `time.Time` không phải múi giờ UTC. Điều này làm tăng thời gian lên đúng bằng offset timezone (ví dụ: Local timezone là +07:00 thì tăng 7 tiếng), làm sai lệch epoch milliseconds dùng để tạo mã băm XOR vân tay bản ghi.
- **Biện pháp khắc phục:** Sửa logic parse của `time.Time` sang dùng phương thức `.UTC()` chuẩn của Golang để bảo toàn thời điểm vật lý của mốc thời gian.

## Nhật ký tiến độ (Audit Log)
- [2026-07-16T13:57:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Khởi tạo workspace FixTimezoneRecon và thiết lập tài liệu yêu cầu.
- [2026-07-16T14:00:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Sửa đổi mã nguồn trong recon_query.go và recon_postgres_source_test.go, chạy test PASS 100%, chạy script compare_hash.go xác nhận XOR Hash và Count trùng khớp 100%.
- [2026-07-16T14:23:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Reverted parsePostgresTimestamp changes in recon_query.go to preserve original logic.
- [2026-07-16T14:24:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Proposed connection/session-level timezone UTC normalization plan (v2) and waiting for approval.
- [2026-07-16T14:28:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Updated implementation plan v2 based on User feedback to highlight timezone consistency for both hash_win and drift_drill_down.
- [2026-07-16T14:45:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Proposed application-level timezone normalization plan (v3) to resolve timezone shift in Go without any DB-level changes, avoiding connection pool state pollution.
- [2026-07-16T14:49:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Updated implementation plan v4 based on User feedback to detail and clarify the mechanisms/relationship of hash_win and drift_drill_down.
- [2026-07-16T14:55:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Updated implementation plan v5 based on User feedback to confirm that drift_drill_down is triggered dynamically only for windows that failed the hash_win check.
- [2026-07-16T15:00:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Updated implementation plan v6 to detail TIMESTAMP vs TIMESTAMPTZ compatibility on production via Go-level .UTC() normalization.
- [2026-07-16T15:04:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Applied the v6 application-level .UTC() timezone fix in recon_query.go and ran tests successfully. Verified zero drift via compare_hash.go. Created walkthrough reports.
- [2026-07-16T16:15:00+07:00] [Agent:Gemini-3.5-Flash] [DONE] Fixed missing ReconStartTime and ReconEndTime fields for ok status reports in both recon_tier_a.go and recon_tier_b.go. Added assertions and verified tests PASS 100%.
- [2026-07-16T17:22:00+07:00] [Agent:Gemini-Pro] [DONE] Approved by User. Starting to implement Dynamic Timezone Detection in recon package to support both TIMESTAMP and TIMESTAMPTZ dynamically.
- [2026-07-16T17:23:00+07:00] [Agent:Gemini-Pro] [DONE] Finished implementing Dynamic Timezone Detection in recon package. All queries (recon_query, recon_hash, recon_stream, recon_dest_query, recon_dest_hash) refactored. Unit tests verified and PASS 100%.
