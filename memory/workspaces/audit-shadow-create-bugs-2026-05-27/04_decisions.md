# 04_decisions — Audit Shadow Create Bugs

## D-1: Bug 1 fix dùng `ListActiveBySourceObject` thay vì sửa `GetActiveRulesBySourceTable`
- **Quyết định**: Swap caller `command_handler.go:649` sang `ListActiveBySourceObject(ctx, effectiveID)`. Giữ nguyên `GetActiveRulesBySourceTable` (có thể có caller khác — line 1389 trong `HandleScanFields`).
- **Lý do**: §6 Simplicity First. Đã có API ID-based sẵn (line 37-44), không cần thêm code. `effectiveID` đã được resolve trước đó (line 620-623), không cần thêm SELECT.
- **Risk còn lại**: line 1389 (`HandleScanFields`) cũng dùng `GetActiveRulesBySourceTable(sourceTable)` → vẫn có thể leak nếu trigger Scan Fields thủ công. Audit phase tiếp theo: kiểm tra `HandleScanFields` flow có nên cũng swap không. Ghi vào `10_gap_analysis.md`.

## D-2: Bug 2 fix bổ sung 3 cột vào `command_handler.go` thay vì refactor sang `sinkworker/schema_manager`
- **Quyết định**: Thêm `_source_ts BIGINT`, `_gpay_source_id TEXT UNIQUE`, `_gpay_deleted BOOLEAN DEFAULT FALSE` trực tiếp vào 2 vị trí DDL trong `command_handler.go`.
- **Lý do**: Minimal impact. Refactor để cả 2 service share `schema_manager` là **scope creep** lớn (cross-package import, ports/adapter pattern). Bug fix không nên kéo theo refactor.
- **Risk còn lại**: 2 path build DDL độc lập (`sinkworker/schema_manager.go` + `handler/command_handler.go`) → có thể drift lại trong tương lai. Ghi gap vào `10_gap_analysis.md` để khi có resource refactor.

## D-3: KHÔNG migrate shadow đã tạo lỗi trong phase này
- **Quyết định**: Out-of-scope. Chỉ fix forward.
- **Lý do**: Data migration là phase riêng, cần backup + dry-run plan. Audit-only phase này chỉ ngăn shadow MỚI tạo sai. Shadow `sd_export_jobs_1` hiện tại sẽ tạm dùng workaround riêng (phase migration sẽ rà sau).
- **Action**: Ghi vào `08_tasks_audit.md` task `MIGR-1: rà toàn bộ shadow hiện hữu cho `_source_ts`/`_gpay_source_id` (phase sau).

## D-4: KHÔNG cheat DB hoặc đổi config
- **Quyết định**: Tuyệt đối không `ALTER TABLE ... ADD COLUMN` thủ công, không đổi env, không thay flag.
- **Lý do**: User explicit yêu cầu "Luôn làm theo hướng core systems, không cheat db hay thay đổi các config". Vi phạm §6 GEMINI.

## D-5: Audit-only phase, chờ approval trước khi sửa code
- **Quyết định**: Document đầy đủ `09_tasks_solution_audit.md` với code demo, KHÔNG `Edit` source files.
- **Lý do**: §12 Brain Code Prohibition + user yêu cầu "Khi làm plan phải rõ ràng, có giải pháp cụ thể". User cần thấy plan rõ trước khi cho phép thực thi. Phase audit kết thúc → Muscle dừng, chờ verb "ok/sửa đi/triển khai".
