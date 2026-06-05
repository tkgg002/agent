# 05_progress.md — Audit Log (APPEND-ONLY)

> ⚠️ CRITICAL: File này là **immutable log** theo CLAUDE.md §7 + §11.
> TUYỆT ĐỐI **không xóa / sửa** entry cũ. Chỉ APPEND entry mới ở cuối file.
> Format: `[YYYY-MM-DD HH:MM] [Role:ModelID] Action — chi tiết`

---

## 2026-06-02

[2026-06-02 09:15] [Brain:Gemini-3.5-Flash (Unverified)] WORKSPACE_CREATE — feature-sensitive-masking-opt-out-2026-06-02
  Reason: Khởi tạo workspace cho tính năng điều chỉnh nhạy cảm (sensitive masking opt-out) theo yêu cầu của user.
  Action: Tạo thư mục workspace và ghi nhận 00_context.md.
  Governance Check: Không phát hiện lỗi vi phạm quy trình Governance nào. Quy trình Workspace-First được tuân thủ nghiêm ngặt.

[2026-06-02 09:16] [Brain:Gemini-3.5-Flash (Unverified)] EXECUTE_START — Bắt đầu sửa đổi mã nguồn
  Files to modify:
    - centralized-data-service/internal/service/masking_service.go (Backend Worker)
    - cdc-cms-service/internal/app/commands/update_mapping_rule.go (Backend API)
    - cdc-cms-web/src/pages/MappingFieldsPage.tsx (Frontend UI)
  Reason: Thực hiện cấu hình Sensitive Masking opt-out và giới hạn UI theo plan được duyệt.

[2026-06-02 09:17] [Brain:Gemini-3.5-Flash (Unverified)] EXECUTE_COMPLETE — Hoàn tất sửa đổi mã nguồn
  Reason: Đã chỉnh sửa toàn bộ 3 files theo đúng kế hoạch.
  Governance Check: Tuân thủ "Simplicity First", sửa đổi tối thiểu và không can thiệp cấu hình DB trực tiếp.

[2026-06-02 09:18] [Brain:Gemini-3.5-Flash (Unverified)] VERIFICATION_COMPLETE — Kiểm thử tự động thành công
  Details:
    - Chạy Unit test trên centralized-data-service thành công (bao gồm cả test case TestResolveMaskMap_OptOutBehavior mới).
    - Compile build cdc-cms-service thành công.
    - Type check cdc-cms-web (npx tsc) thành công.

[2026-06-02 10:13] [Brain:Gemini-3.5-Flash (Unverified)] PLAN_CREATE — Lập kế hoạch Nested JSON Masking & Restore UI Strategy
  Reason: Nhận yêu cầu mới từ user về việc hỗ trợ mã hóa các field con trong JSON (json_mask) và khôi phục cột mask_strategy ở Frontend.
  Action: Tạo implementation_plan.md và cập nhật 00_context.md.
  Governance Check: Không phát hiện lỗi vi phạm quy trình Governance nào. Quy trình Workspace-First được tuân thủ nghiêm ngặt.[2026-06-02 10:15] [Brain:Antigravity] EXECUTE_COMPLETE — Hoàn tất triển khai Nested JSON Masking & Restore UI Strategy
  Details:
    - Thêm strategy `json_mask` vào `rule.go` (`cdc-cms-service`).
    - Cập nhật worker (`centralized-data-service`) hỗ trợ duyệt đệ quy và mã hóa các field nhạy cảm bên trong JSON.
    - Phục hồi cột Mask Strategy dạng Select bên cạnh cột Sensitive dạng Switch trên Frontend UI (`cdc-cms-web`).
    - Viết thêm unit test `TestResolveMaskMap_JSONMaskBehavior` và chạy thành công 100% tests.
    - Compile build API/Worker và typecheck UI hoàn toàn pass không lỗi.
  Governance Check: Tuân thủ "Simplicity First" và quy trình phát triển an toàn.


[2026-06-02 10:25] [Brain:Antigravity] EXECUTE_PROGRESS — Cập nhật logic IsGlobalSensitive lên UI & DTO
  Details:
    - Thêm trường `is_global_sensitive` vào file `mapping_rule_dto.go` (cả API DTO và mapper).
    - Thêm `is_global_sensitive` vào frontend interface `MappingRule` (`index.ts`).
    - Sửa đổi UI `MappingFieldsPage.tsx` để Switch (Sensitive) bị disabled khi là trường nhạy cảm toàn cục (`is_global_sensitive` = true), đồng thời vô hiệu hóa option `none` trong Select (Mask Strategy) để đảm bảo tuân thủ.
    - Sửa lỗi "Unterminated string" do lỗi encoding trong file `MappingFieldsPage.tsx`.
    - Build frontend (`npm run build`) và backend API (`go build ./internal/...`) thành công 100%.
    - Dừng và khởi động lại API server `cdc-cms-service` (cổng 8083) để nhận cấu hình mới.
  Governance Check: Tuân thủ "Simplicity First" và quy trình phát triển an toàn.

[2026-06-02 11:35] [Brain:Antigravity] PLAN_CREATE — Lập kế hoạch Fix Column-level JSON Masking trong DynamicMapper
  Reason: Cấu hình `json_mask` cho cột chưa chạy do `MaskByStrategy` bỏ qua `json_mask` trong DynamicMapper.
  Action: Cập nhật implementation_plan.md và bổ sung kế hoạch chi tiết.
  Governance Check: Không phát hiện lỗi vi phạm quy trình Governance nào. Quy trình Workspace-First được tuân thủ nghiêm ngặt.

[2026-06-02 11:40] [Brain:Antigravity] SESSION_RESUME — Tiếp tục phiên làm việc, chuẩn bị kế hoạch sửa lỗi column-level JSON masking.
  Action: Tạo implementation_plan.md chi tiết và xin phê duyệt của User.

[2026-06-02 11:45] [Brain:Antigravity] EXECUTE_PROGRESS — Triển khai sửa đổi Column-level JSON Masking
  Details:
    - Expose MaskJSONFields trong MaskingService.
    - Cập nhật maybeMaskColumn trong DynamicMapper và MapColumnsFromElement trong child_explode.go.
    - Bổ sung unit test TestDynamicMapper_ColumnLevelJSONMask trong masking_service_test.go.
    - Chạy unit test của service pass 100% và build thành công.
    - Graceful restart worker process trong centralized-data-service bằng lệnh make run.
  Governance Check: Tuân thủ "Simplicity First" và quy trình phát triển an toàn.

[2026-06-02 11:50] [Brain:Antigravity] PLAN_CREATE — Phân tích lỗi và lên kế hoạch sửa stale cache & dynamic mapper opt-out check.
  Details:
    - Root cause:
      1. NATS schema.config.reload handler không invalidate cache của MaskingService.
      2. maybeMaskColumn trong DynamicMapper thiếu check rule.IsSensitiveField cho json_mask strategy.
    - Kế hoạch: Sửa worker_server.go, dynamic_mapper.go và thêm test case.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 11:55] [Brain:Antigravity] EXECUTE_COMPLETE — Triển khai sửa lỗi opt-out và stale cache cho masking service.
  Details:
    - Cập nhật dynamic_mapper.go để luôn bypass khi IsSensitiveField = false.
    - Cập nhật worker_server.go gọi maskingSvc.Invalidate(table) khi config reload từ NATS.
    - Thêm 2 unit test mới trong masking_service_test.go thành công.
    - Build compile worker thành công và test pass 100%.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 11:58] [Brain:Antigravity] PLAN_CREATE — Lên kế hoạch gỡ bỏ logic bắt buộc mã hóa đối với trường nhạy cảm toàn cục trên UI.
  Details:
    - Bỏ disable Switch sensitive và Select mask_strategy của các trường is_global_sensitive trên UI MappingFieldsPage.tsx.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 12:00] [Brain:Antigravity] EXECUTE_COMPLETE — Gỡ bỏ logic bắt buộc mã hóa đối với trường nhạy cảm toàn cục trên UI.
  Details:
    - Sửa MappingFieldsPage.tsx để Switch Sensitive và Select Mask Strategy hoạt động bình thường, không bị disabled hay gán cứng bởi is_global_sensitive.
    - Compile build frontend thành công 100% không lỗi.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 12:12] [Brain:Antigravity] PLAN_CREATE — Lên kế hoạch khôi phục nút Tạo Table và Tạo Field MĐ trên UI TableRegistry.tsx.
  Details:
    - Khôi phục import ToolOutlined.
    - Khôi phục hàm handleCreateTable, handleCreateDefaultFields và map nút bấm hiển thị tương ứng trạng thái is_table_created của record.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 12:15] [Brain:Antigravity] EXECUTE_COMPLETE — Khôi phục thành công các nút tạo table trên UI TableRegistry.tsx.
  Details:
    - Sửa TableRegistry.tsx: khôi phục import ToolOutlined, callback handleCreateTable, handleCreateDefaultFields và map hiển thị nút bấm Tạo Table / Tạo Field MĐ theo đúng trạng thái is_table_created của record.
    - Compile build frontend thành công 100% không phát sinh bất kỳ lỗi gì.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 12:17] [Brain:Antigravity] PLAN_CREATE — Kế hoạch đơn giản hóa nút Tạo Table (luôn hiển thị) và bỏ nút Tạo Field MĐ.
  Details:
    - Sửa TableRegistry.tsx để nút Tạo Table luôn hiển thị, bỏ ToolOutlined và logic handleCreateDefaultFields.
  Governance Check: Không phát hiện lỗi vi phạm quy trình.

[2026-06-02 12:19] [Brain:Antigravity] EXECUTE_COMPLETE — Đơn giản hóa thành công nút tạo table trên UI TableRegistry.tsx.
  Details:
    - Sửa TableRegistry.tsx: loại bỏ ToolOutlined icon import, comment lại hàm handleCreateDefaultFields, gỡ bỏ nút Tạo Field MĐ và cập nhật để nút Tạo Table luôn luôn hiển thị trên UI.
    - Compile build frontend thành công 100% không phát sinh lỗi.

[2026-06-02 15:36] [Brain:Antigravity] EXECUTE_START — Bắt đầu sửa đổi mã nguồn để khắc phục lỗi mất dữ liệu khi Snapshot dùng shadow_binding_id.
  Files to modify:
    - centralized-data-service/internal/service/metadata_registry_service.go
    - centralized-data-service/internal/service/dynamic_mapper.go
    - centralized-data-service/internal/handler/event_handler.go
    - centralized-data-service/internal/service/child_explode.go
    - centralized-data-service/internal/service/masking_service.go
  Reason: Cập nhật cache mapping và masking strategy theo shadow_binding_id thay vì shadow_table để tránh xung đột cấu hình giữa các shadow tables trùng tên.
  Governance Check: Tuân thủ "Simplicity First, minimal impact".

[2026-06-02 15:42] [Brain:Antigravity] EXECUTE_COMPLETE — Hoàn tất di chuyển cấu hình Masking và Mapping sang shadow_binding_id (int64)
  Details:
    - Cập nhật signature `GetMappingRules` của legacy `RegistryService` (`registry_service.go`) để nhận `bindingID` `int64`.
    - Refactor toàn bộ integration test và unit test trong `dynamic_mapper_test.go`, `metadata_registry_service_test.go`, `registry_service_test.go` tương thích với `bindingID` `int64`.
    - Cải tiến `MaskingService` với cơ chế Dynamic Resolution (`resolveBindingID` hỗ trợ cả `int64` và string `shadow_table` fallback), sửa các call site ở các handler/consumer tránh lỗi build.
    - Cải tiến `resolveMaskMap` trả về `defaultMasks` khi `bindingID <= 0` để duy trì chính sách bảo mật tối đa (default masks fallback) khi không có DB/shadow_binding trong các unit test.
    - Sắp xếp và phân vùng lại các file nháp của package `scratch` tránh trùng hàm `main` gây lỗi build project.
    - Chạy unit test toàn bộ các packages thành công 100% (pass 100% tests, exit code 0).
  Governance Check: Không phát hiện lỗi vi phạm quy trình Governance. Quy trình "Simplicity First" được tuân thủ trọn vẹn.
