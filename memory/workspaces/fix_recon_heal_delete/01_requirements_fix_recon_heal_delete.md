# Yêu cầu Specs - Sửa lỗi Heal không update/delete Master/Shadow (Bổ sung FQN Schema Prefix & Dời logic Resolve Config)

## Hiện trạng & Vấn đề
Khi chạy Heal đối soát chặng Segment B (`shadow_master`), mặc dù đã prune master thành công 100%, hệ thống vẫn văng ra 2 log lỗi record not found từ repo:
```
{"level":"error","ts":1784111923.5318289,"caller":"source/table_registry_repo.go:32","msg":"gorm exec error","error":"record not found","elapsed":0.007030792,"rows":0,"sql":""}
```
Nguyên nhân là do hệ thống luôn cố tìm registry cấu hình trong `cdc_table_registry` bằng tên Master Table FQN (e.g. `master_centrallized_export_service.export_jobs`), trong khi registry chỉ lưu tên Shadow Table (e.g. `shadow_testces.export_jobs`).

## Nguyên nhân gốc rễ (Root Cause)
Trong hàm `processSingleReport` của [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go), logic resolve cấu hình:
```go
entry := h.resolveTargetTableConfig(rpt.TargetTable)
```
được đặt **trước** switch-case phân loại Segment. Do đó, ngay cả khi đối soát Segment B (không cần dùng đến `entry` này), hệ thống vẫn cố gọi `resolveTargetTableConfig` với tên Master table, sinh lỗi "record not found" vô nghĩa làm bẩn log.

## Yêu cầu giải pháp (Cập nhật)
1. **Fully Qualified Name (FQN) cho TargetTable:** Đảm bảo `rpt.TargetTable` luôn có schema prefix.
2. **Dời logic resolve cấu hình vào đúng case cần thiết:**
   - Dời lệnh `entry := h.resolveTargetTableConfig(rpt.TargetTable)` từ ngoài switch-case vào bên trong nhánh `case SegmentSourceShadow, ""` (Segment A) vì chỉ có Segment A mới cần registry config để truy vấn MongoDB nguồn.
   - Segment B (`SegmentShadowMaster`) sẽ không gọi resolve config nữa, tránh hoàn toàn log lỗi "record not found".
3. **Xoá cứng trên Master DB (Segment B) và xoá mềm trên Shadow DB (Segment A):** Giữ nguyên logic đã chạy thành công.
