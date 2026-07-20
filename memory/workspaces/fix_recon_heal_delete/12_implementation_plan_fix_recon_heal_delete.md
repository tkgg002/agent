# Kế hoạch triển khai của AI - Sửa lỗi Heal Schema Prefix

Kế hoạch này chi tiết hóa các bước sửa lỗi gắn schema prefix cho `TargetTable` trong file `recon_execute_heal_handler.go`.

## Các bước thực hiện

1. **Backup file gốc:**
   - Đã tạo backup `/Users/trainguyen/Documents/work/agent/memory/workspaces/fix_recon_heal_delete/recon_execute_heal_handler.go.bak-heal-schema-prefix-2026-07-15`.

2. **Chỉnh sửa code `internal/handler/recon/recon_execute_heal_handler.go`:**
   - Sửa hàm `processSingleReport` để gán `rpt.TargetTable` kèm theo Schema Prefix (`MasterSchema` hoặc `ShadowSchema`) nếu `rpt.TargetTable` rỗng hoặc thiếu prefix (không chứa `.`):
     - Segment `SegmentShadowMaster`: Nếu `rpt.MasterSchema` không rỗng và `rpt.TargetTable` không chứa `.`, gán `rpt.TargetTable = rpt.MasterSchema + "." + rpt.MasterTable`. Ngược lại nếu `rpt.TargetTable` rỗng, gán `rpt.TargetTable = rpt.MasterTable`.
     - Segment khác (A): Nếu `rpt.ShadowSchema` không rỗng và `rpt.TargetTable` không chứa `.`, gán `rpt.TargetTable = rpt.ShadowSchema + "." + rpt.ShadowTable`. Ngược lại nếu `rpt.TargetTable` rỗng, gán `rpt.TargetTable = rpt.ShadowTable`.

3. **Kiểm tra biên dịch:**
   - Chạy lệnh `go build ./...` trong thư mục `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`.

4. **Cập nhật tài liệu tiến độ:**
   - Thêm log cập nhật vào `05_progress_fix_recon_heal_delete.md`.
   - Cập nhật task hoàn thành vào `08_tasks_fix_recon_heal_delete.md`.

5. **Chạy linter quy trình:**
   - Chạy `python3 agent/tooling/verify_governance.py` từ thư mục `/Users/trainguyen/Documents/work/agent`.

6. **Báo cáo kết quả:**
   - Gửi báo cáo thông qua `send_message` tới parent.
