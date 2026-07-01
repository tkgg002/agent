# Report: Fix False Drift on Recon payment_bills / Báo cáo: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

Chi tiết báo cáo hiện trạng và quá trình thực thi đã được ghi nhận đầy đủ tại các tài liệu chuẩn của Workspace:
- Báo cáo chi tiết kỹ thuật: [03_implementation.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/bug-recon-false-drift-payment-bills-2026-06-30/03_implementation.md)
- Quyết định kiến trúc: [04_decisions.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/bug-recon-false-drift-payment-bills-2026-06-30/04_decisions.md)
- Kịch bản kiểm thử và kết quả xác minh: [06_validation.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/bug-recon-false-drift-payment-bills-2026-06-30/06_validation.md)

## Thay đổi mã nguồn (307 dòng code):
1. **[MODIFY]** [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go): +54 dòng code (Nhận `timestampField` và đổi cách map query Postgres sang epoch milliseconds).
2. **[MODIFY]** [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go): +28 dòng code (Cập nhật hàm tính XOR hash nhận `timestampField`).
3. **[MODIFY]** [recon_dest_legacy.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_legacy.go): +12 dòng code (Cập nhật wrapper tương thích ngược).
4. **[MODIFY]** [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go): +25 dòng code (Sử dụng dynamic `resolvedTS` cấu hình trong mapping registry cho Tier 1).
5. **[MODIFY]** [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go): +18 dòng code (Truyền explicit `_source_ts` cho Tier 2).
6. **[NEW]** [recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go): +170 dòng code (Bộ unit test độc lập kiểm thử dynamic timestamp query & hash).

## Kết quả kiểm tra:
- Dự án build/compile thành công 100% bằng `go build`.
- Kiểm thử unit test gói `recon` PASS 100% (bao gồm cả các test cases mới và cũ).
