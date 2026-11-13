# 08. Tasks — Kiểm tra GetList & Thiết kế Export cho `collection-disbursement/orders`

## Danh sách Task

### Task 1: Điều tra luồng GetList `api/v2/admin-portal/ops/collection-disbursement/orders` (S2)
- **Mô tả & Context (`file:line`):** Truy vết toàn trình từ `admin-portal-gateway` xuống downstream service xử lý `collection-disbursement/orders` (route, controller/handler, DTO request/response, NATS/gRPC/HTTP subject, service, repository/query DB).
- **Logs/Test:** Kiểm tra git status/branch hiện tại của các repo liên quan và đối chiếu code thực tế tại `HEAD`.
- **Definition of Done (DoD):** Xác định chính xác 100% `file:line` của luồng `GetList`, danh sách filter parameters, cấu trúc dữ liệu trả về và nguồn dữ liệu.

### Task 2: Điều tra Pattern Export chuẩn của hệ thống (S2 & S2.5 DG0)
- **Mô tả & Context (`file:line`):** Kiểm tra cách các API `/ops/...` hoặc tương đương trong `admin-portal-gateway` và `centralized-export-service` (hoặc `export-service` / downstream service) triển khai tính năng Export. Chạy cổng phản biện `DG0` (`laya_gate.py run --stage dg0`).
- **Logs/Test:** Đối chiếu các luồng export mẫu cùng domain/layer, chạy `DG0` đạt `PASS`.
- **Definition of Done (DoD):** Nắm rõ 100% chuẩn pattern Export (route, request DTO,filter parity với GetList, message queue/RPC contract, processor/excel builder, permission/scope nếu có).

### Task 3: Lập Giải pháp & Code Demo chi tiết (S3 + DG1/DG2)
- **Mô tả & Context (`file:line`):** Soạn `09_tasks_solution_collection_disbursement_orders_export.md` với duy nhất 1 phương án chuẩn pattern, ánh xạ 1-1 các findings `F1..F4`, kèm bảng Call-Site Wiring và Code Demo nguyên văn từ `HEAD` (không dùng `...`).
- **Logs/Test:** Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` đạt `PASS`.
- **Definition of Done (DoD):** Hồ sơ giải pháp `09_tasks_solution` hoàn chỉnh, `DG1` `PASS`, hiển thị trực tiếp Code Demo trên chat và dừng chờ `APPROVE` từ User.

### Task 4: Thực thi tại `centralized-export-service` (S4 — Đã hoàn tất sau `APPROVE`)
- **Mô tả & Context (`file:line`):**
  1. [x] Đồng bộ schema `PaymentBillModel` (`models/payment-bill/payment-bill.model.ts:130-139`) và `PaymentBillEntity` (`data-transfers/entities/payment-bill/payment-bill.entity.ts:68-76`) bổ sung `transactionParties`, `transHoldingId`, `serviceCode`, `bizDomain`, `bizType`; thêm `PaymentBillDocumentEntity` (`data-transfers/entities/payment-bill/payment-bill-document.entity.ts:1-33`) và `PaymentBillDocumentModel` (`models/payment-bill/payment-bill-document.model.ts:1-106`, `defined/app-setting.ts:67-69`, `models/index.ts:38-41,91-95,150-154`).
  2. [x] Tạo DTO `CollectionDisbursementOrdersExportParams` tại `data-transfers/params/payment-bill/collection-disbursement-orders-export.params.ts:1-149` và export tại `data-transfers/params/index.ts:38-44,85-91`.
  3. [x] Tạo `GetAllCollectionDisbursementOrdersExportQuery` (`domain/queries/payment-bill/GetAllCollectionDisbursementOrdersExportQuery.ts:1-12`) và `GetAllCollectionDisbursementOrdersExportHandler` (`domain/handlers/payment-bill/GetAllCollectionDisbursementOrdersExportHandler.ts:1-107`), đăng ký vào `domain/queries/index.ts:44-50` và `domain/QueryHandlerFactory.ts:48-55,129-135`.
  4. [x] Tạo `collection-disbursement-orders-export.pure.ts` (`logics/export/payment-bill/collection-disbursement-orders-export.pure.ts:1-227`) và `CollectionDisbursementOrdersExport` (`logics/export/payment-bill/collection-disbursement-orders.export.ts:1-28` có override `getRequiredModelName(): string[]`), đăng ký vào `logics/export/index.ts:35-41` và `logics/index.ts:40-46,104-110`.
- **Logs/Test:** Viết unit test tại `test/unit/models/payment-bill-document.model.test.ts`, `test/unit/pure/collection-disbursement-orders.pure.test.ts`, `test/unit/processors/CollectionDisbursementOrdersExport.test.ts`, `test/unit/domain/handlers/GetAllCollectionDisbursementOrdersExportHandler.test.ts`, `test/unit/domain/QueryHandlerFactory.test.ts`; chạy unit test (`EV-20261110T165315-5e24`) và `npm run build` (`EV-20261110T165012-55d3`) qua `evidence_run.py`.
- **Definition of Done (DoD):** Toàn bộ unit test & build pass 100%, filter parity 1-1 với `opsSearchPaymentBills`.

### Task 5: Kiểm thử Tích hợp MongoDB thật & Runtime SAGA Excel (S4 & S5 — Đã hoàn tất)
- **Mô tả & Context (`file:line`):**
  1. [x] Viết integration test `test/integration/collection-disbursement-orders-export.integration.test.ts:1-311` sử dụng `MongoMemoryServer` (`mongod` thật) + `ExcelJS` để kiểm chứng toàn trình: `GET /api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport` $\rightarrow$ `ExportLogic.processExport()` $\rightarrow$ `CollectionDisbursementOrdersExport.execute()` (SAGA 5 bước) $\rightarrow$ `QueryBus.execute(GetAllCollectionDisbursementOrdersExportQuery)` $\rightarrow$ `GetAllCollectionDisbursementOrdersExportHandler.execute()` $\rightarrow$ xuất file `.xlsx` thật đúng 18 cột và chỉ lấy bản ghi `bizDomain: "TRANSFER_BIZ"`.
  2. [x] Tự động phát hiện và sửa dứt điểm giới hạn tên sheet Excel $\le 31$ ký tự (`_Don_Hang_Thu_Chi_Ho` / `_Transfer_Orders_List`) qua vòng lặp Self-Healing `S5 -> S3 -> S4 -> S5`.
- **Logs/Test:** Chạy integration test (`EV-20261110T164834-57d6`) và runtime verification trên bản build `dist/` (`EV-20261110T165015-24f7`) qua `evidence_run.py`.
- **Definition of Done (DoD):** Toàn bộ unit test, integration test trên MongoDB thật, runtime SAGA Excel `.xlsx` và cổng `DG3` (`06_laya_summary_dg3_20261110T165104.json`) đạt `PASS` 100%.
