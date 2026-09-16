# 11 Change Report — Force Transform & Bug Fix

## Danh sách các file đã thay đổi

| STT | File Path | Số dòng thay đổi | Mô tả thay đổi |
|---|---|---|---|
| 1 | `centralized-data-service/internal/service/metadata/mapping_utils.go` | +17, -2 | Tách case `timestamptz` / `timestamp with time zone`, dùng `::TIMESTAMPTZ` tại nhánh fallback ELSE. |
| 2 | `centralized-data-service/internal/handler/shadow/batch_transform_handler.go` | +32, -3 | Thêm `Force`, `ForceFields` vào `BatchTransformPayload`, thêm validate và logic tách nhánh force transform (WHERE TRUE, SET chỉ force fields). |
| 3 | `centralized-data-service/internal/handler/shadow/batch_transform_handler_test.go` | +60, -8 | Cập nhật sqlmock args khớp 3 params của `GetActiveRulesBySourceTable` và thêm unit test `TestHandleBatchTransform_ForceMode`. |
| 4 | `cdc-cms-service/internal/api/source/source_object_actions_handler.go` | +18, -4 | Đọc `force` & `force_fields` từ JSON request body trong `TransformV2`, validate và forward vào NATS message payload. |
| 5 | `cdc-cms-web/src/pages/TableRegistry.tsx` | +120, -45 | Đổi key index `activeJobId` ở child table, tạo component `TransformModal` tích hợp Switch Force mode và Select chọn `force_fields` trực tiếp trong modal confirm của thao tác Transform. |
| 6 | `cdc-cms-web/src/pages/MappingFieldsPage.tsx` | 0 | Giữ nguyên thuần túy cấu hình mapping rules (không đặt nút trigger action sai ngữ cảnh). |
