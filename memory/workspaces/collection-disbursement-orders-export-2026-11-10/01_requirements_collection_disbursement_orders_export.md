---
tier: T2
suffix: collection_disbursement_orders_export
workspace: collection-disbursement-orders-export-2026-11-10
created_at: 2026-11-10T14:44:45+07:00
status: IN_PROGRESS
---

# 01. Requirements — Kiểm tra GetList & Lên Plan Export cho `api/v2/admin-portal/ops/collection-disbursement/orders`

## 1. Yêu cầu gốc từ User
- Kiểm tra API GetList: `api/v2/admin-portal/ops/collection-disbursement/orders`.
- Lên kế hoạch (Plan + Code Demo) viết tính năng **Export** cho danh sách này.
- **Yêu cầu cốt lõi:** Bắt buộc tuân thủ đúng **Design Pattern** hiện hữu của hệ thống, gọi qua endpoint tập trung `/api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport` (`centralized-export-service`), tuyệt đối không tự chế thêm route mới trên `admin-portal-gateway`. Giữ chế độ **Propose-Only** chờ User `APPROVE`.

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons & GEMINI.md)
- **L001 (Reuse Existing Centralized Export Endpoint Over Inventing New Gateway Routes):** Giữ nguyên endpoint xuất dữ liệu bất đồng bộ tập trung `/api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport`, tuyệt đối không sửa hay mở thêm route trên `admin-portal-gateway`.
- **L031 (Branch-Target Action Verification & List-Export Filter Parity):** Bắt buộc đối chiếu 1-1 toàn bộ các tham số filter, điều kiện lọc, phân trang/sắp xếp giữa luồng `GetList` và luồng `Export` để không lệch số lượng bản ghi hoặc thiếu cờ lọc chuyên biệt.
- **L022 (Respect Live Production Baseline & Codify Existing Design Patterns):** Bám sát 100% kiến trúc và pattern export đang chạy thật trong `centralized-export-service` (`BaseExportProcessor` + `*ExportPure` + `Params` + `CQRS Query/Handler`).
- **L019 & L036 (Verbatim Current-File Code Demo & Strict 1-to-1 Parity):** Mọi khối trích dẫn và Code Demo trong `09_tasks_solution` và trên chat phải trích nguyên văn từ file tại `HEAD` trên đĩa, đầy đủ 100% không dùng dấu `...` cắt dòng, kèm `file:line`.
- **L044 (Explicit Runtime Call-Site & Lifecycle Trigger Wiring):** Chỉ rõ điểm gọi (`caller file:line`) xuyên suốt từ Frontend -> `/api/v1/admin-portal/export-async` -> `centralized-export-service` (`ExportLogic` -> `CollectionDisbursementOrdersExport` -> `QueryBus`).
- **L045 (Propose-Only Discipline & Explicit Approval Gate):** Không sửa bất kỳ file source code nào trước khi User `APPROVE`.
- **L070 (Fail-Fast Config Validation & Interface Signature Integrity):** Bảo toàn chữ ký interface và kiểm tra đầy đủ implementers/callers.
