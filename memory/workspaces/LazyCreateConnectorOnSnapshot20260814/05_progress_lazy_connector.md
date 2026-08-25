# Progress Audit Log: Lazy Create Connector On Snapshot

- [2026-08-14T15:54:00+07:00] [Agent:Gemini-3.6-Flash] Khởi tạo workspace LazyCreateConnectorOnSnapshot20260814.
- [2026-08-14T15:54:00+07:00] [Agent:Gemini-3.6-Flash] Đọc GEMINI.md và lessons.md.
- [2026-08-14T15:54:00+07:00] [Agent:Gemini-3.6-Flash] Lập kế hoạch thiết kế kỹ thuật (Plan & Technical Solution) cho luồng Lazy Connector Creation & Preserve Legacy Pause/Resume code.
- [2026-08-14T16:01:00+07:00] [Agent:Gemini-3.6-Flash] Nhận lệnh APPROVE từ User. Chuyển sang Role Muscle để thực thi thay đổi mã nguồn.
- [2026-08-14T16:03:00+07:00] [Agent:Gemini-3.6-Flash] Refactor CreateSystemConnectorHandler trong debezium_connector.go (cdc-cms-service): Phân nhánh isSFTP thành công, lưu status="configured" cho SFTP, giữ nguyên Eager flow cho DB types khác.
- [2026-08-14T16:05:00+07:00] [Agent:Gemini-3.6-Flash] Refactor snapshot_runner_handler.go (centralized-data-service): Bổ sung logic kiểm tra GET status và POST lazy create SFTP connector khi runSnapshot.
- [2026-08-14T16:07:00+07:00] [Agent:Gemini-3.6-Flash] Tiến hành Adversarial Self-Review Audit theo yêu cầu của User: Phát hiện lỗ hổng UpdateConfig bị 404 khi connector ở trạng thái lazy configured.
- [2026-08-14T16:08:00+07:00] [Agent:Gemini-3.6-Flash] Đã fix lỗ hổng UpdateConfig trong debezium_connector.go, bổ sung GetByConnectorName vào system_connector_repo_gorm.go và ports/repository.go.
- [2026-08-14T16:11:00+07:00] [Agent:Gemini-3.6-Flash] Nhận phản hồi từ User về việc khoanh vùng 100% SFTP cho UpdateConfig. Đã bổ sung kiểm tra isSFTP trong UpdateSystemConnectorConfigHandler.
- [2026-08-14T16:17:00+07:00] [Agent:Gemini-3.6-Flash] Nhận phản hồi từ User về việc cập nhật PostgreSQL CHECK Constraint. Đã tạo file migration 088_update_sources_status_check_configured.sql, chạy ALTER TABLE bổ sung 'configured' vào sources_status_check constraint trên DB gpay-postgres-cdc thành công!
- [2026-08-14T16:32:00+07:00] [Agent:Gemini-3.6-Flash] Nhận báo lỗi từ User về việc fs.uris chứa chuỗi mờ ***:*** làm connector không connect được SFTP. Đã bổ sung logic giải mã unmasking fs.uris từ connection_registry.options_json trong snapshot_runner_handler.go!
- [2026-08-14T17:26:00+07:00] [Agent:Gemini-3.6-Flash] Sửa triệt để fallback policy.regexp trong SourceConnectors.tsx thành động '^.*\\.csv$'. Đã biên dịch lại npm run build thành công!
- [2026-08-17T09:15:00+07:00] [Agent:Gemini-3.6-Flash] Áp dụng giải pháp triệt để ngăn ngừa rác schema cho SFTP shadow tables: (1) Cập nhật discovery_utils.go chèn case engine == "sftp" trả về nil (Lean Schema Mode); (2) Cập nhật provisioning_shadow_bind.go chèn SFTP clean provisioning guard ép businessCols = nil. Chạy test suite PASS 100%!
