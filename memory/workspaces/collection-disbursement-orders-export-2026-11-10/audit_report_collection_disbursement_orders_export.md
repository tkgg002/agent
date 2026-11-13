---
tier: T2
suffix: collection_disbursement_orders_export
workspace: collection-disbursement-orders-export-2026-11-10
---

# Audit & Validation Report — `CollectionDisbursementOrdersExport` (`centralized-export-service`)

## 1. (a) Line-by-Line Diff Critique & Physical File Verification (`git diff` + `wc -l`)

### 1.1. Thống kê dòng vật lý (`wc -l`) sau lần sửa code cuối
- `defined/app-setting.ts`: `142` dòng (`+1` dòng thêm `PAYMENT_BILL__PAYMENT_BILL_DOCUMENT: "payment-bill-documents"` tại dòng `68`).
- `data-transfers/entities/payment-bill/payment-bill.entity.ts`: `78` dòng (`+5` dòng thêm `transactionParties`, `transHoldingId`, `serviceCode`, `bizDomain`, `bizType` tại dòng `71-75`).
- `data-transfers/entities/payment-bill/payment-bill-document.entity.ts`: `33` dòng (file mới định nghĩa `BillDocumentFile` và `PaymentBillDocumentEntity`).
- `data-transfers/entities/index.ts`: `91` dòng (`+2` dòng import/export `PaymentBillDocumentEntity` tại dòng `38, 80`).
- `models/payment-bill/payment-bill.model.ts`: `196` dòng (`+6/-1` dòng tại dòng `133-138` bổ sung `transactionParties`, `transHoldingId`, `serviceCode`, `bizDomain`, `bizType` vào Mongoose schema).
- `models/payment-bill/payment-bill-document.model.ts`: `106` dòng (file mới định nghĩa `PaymentBillDocumentModel` kèm `aggregateInvoiceFlagsByBillIds(billIds)`).
- `models/index.ts`: `178` dòng (`+3` dòng tại dòng `40, 94, 153` đăng ký `PaymentBillDocumentModel` vào export và `ModelRegistry["payment-bill"]`).
- `data-transfers/params/payment-bill/collection-disbursement-orders-export.params.ts`: `149` dòng (file mới định nghĩa `CollectionDisbursementOrdersExportParams` và `ExceedDateRangeValidator`).
- `data-transfers/params/index.ts`: `102` dòng (`+2` dòng tại dòng `39, 86` export `CollectionDisbursementOrdersExportParams`).
- `domain/queries/payment-bill/GetAllCollectionDisbursementOrdersExportQuery.ts`: `12` dòng (file mới định nghĩa `GetAllCollectionDisbursementOrdersExportQuery`).
- `domain/queries/index.ts`: `70` dòng (`+1` dòng tại dòng `45` export `GetAllCollectionDisbursementOrdersExportQuery`).
- `domain/handlers/payment-bill/GetAllCollectionDisbursementOrdersExportHandler.ts`: `107` dòng (file mới truy vấn `PaymentBillModel` theo chunk và tổng hợp cờ `hasDocument` từ `PaymentBillDocumentModel.aggregateInvoiceFlagsByBillIds` + `embedData.uploaded`).
- `domain/QueryHandlerFactory.ts`: `162` dòng (`+2` dòng tại dòng `49, 130` đăng ký `GetAllCollectionDisbursementOrdersExportQuery` $\rightarrow$ `GetAllCollectionDisbursementOrdersExportHandler`).
- `logics/export/payment-bill/collection-disbursement-orders-export.pure.ts`: `227` dòng (file mới chứa pure functions `buildPartyTypeCondition`, `buildCollectionDisbursementOrdersFilter`, `buildCollectionDisbursementOrdersSort`, `validate`, `getConfig`, `transformRow`).
- `logics/export/payment-bill/collection-disbursement-orders.export.ts`: `28` dòng (file mới định nghĩa `CollectionDisbursementOrdersExport extends BaseExportProcessor` với override `getRequiredModelName(): string[] { return ["PaymentBillModel", "PaymentBillDocumentModel"]; }`).
- `logics/export/index.ts`: `59` dòng (`+1` dòng tại dòng `36` export `CollectionDisbursementOrdersExportPure`).
- `logics/index.ts`: `128` dòng (`+2` dòng tại dòng `41, 105` export `CollectionDisbursementOrdersExport` vào `AllProcessors`).

### 1.2. Phản biện từng Task trong `08_tasks` và từng dòng `git diff`
- **Task 1–3 (Điều tra & Thiết kế):** Đã truy vết chính xác endpoint `GET /api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport` (`admin-portal-gateway/integrate-services/v1/start.ts:3900` & `centralizedExport.exportAsync.ts:10-14`), giữ nguyên 100% `admin-portal-gateway`.
- **Task 4–5 (Thực thi & Kiểm thử):**
  - Trong `buildCollectionDisbursementOrdersFilter` (`collection-disbursement-orders-export.pure.ts:74-142`): Luôn khởi tạo `const andConditions: object[] = [{ bizDomain: BIZ_DOMAIN_TRANSFER }]`, gán `filter.isDelete = false` và `filter.$and = [...(filter.$and || []), ...andConditions]` ở cuối hàm, đảm bảo không có bất kỳ tham số đầu vào nào có thể ghi đè hoặc làm rò rỉ dữ liệu ngoài `bizDomain: "TRANSFER_BIZ"`.
  - Trong `GetAllCollectionDisbursementOrdersExportHandler.execute` (`GetAllCollectionDisbursementOrdersExportHandler.ts:51-72`): Khi gọi `DocumentModel.aggregateInvoiceFlagsByBillIds(billIds)`, đã bọc `try/catch` có log cảnh báo `this.mainProcess.logger.warn` và fallback an toàn sang `((doc?.embedData as any)?.uploaded?.length ?? 0) > 0` (khớp hành vi `enrichInvoiceDocumentFlags` của `payment-bill-service/logics/payment-bill.logic.ts:36-74`).

---

## 2. (b) Anti-Smoke-Test Speculation Critique (Đa Worker, Chunking, Cursor & Timeout)

- **Chunking & Cursor Continuity (`BaseExportProcessor.ts:169-274` & `GetAllCollectionDisbursementOrdersExportHandler.ts:16,42-63`):**
  - `ExportWorker` xử lý theo từng lô `chunkSize = 5000`. Ở lô đầu tiên (`cursor = null` hoặc `0`), handler chạy song song `findQuery.lean().exec()` và `Model.countDocuments(filter)` để tính tổng tiến độ; ở các lô tiếp theo (`cursor > 0`), handler dùng `.skip(cursor).limit(limit)` và bỏ qua `countDocuments` (`Promise.resolve(0)`) để tránh truy vấn đếm lặp lại gây nghẽn MongoDB.
  - Khi `data.length === limit`, `nextCursor = (cursor || 0) + limit`; khi `data.length < limit`, `nextCursor = null` kết thúc vòng lặp `do ... while (cursor)`.
- **Cô lập Trạng thái Đa Worker / Đa Job (`CollectionDisbursementOrdersExport.ts:7,19-21`):**
  - `ExportLogic.processExport()` và `ExportWorker` luôn khởi tạo instance mới `new ProcessorClass(context, queryBus, redisManager)` cho mỗi job, nên biến đếm `private rowIndex: number = 0` trong `CollectionDisbursementOrdersExport` hoàn toàn cô lập theo từng job, không bị race condition khi nhiều job chạy song song.
- **Tương thích `ExcelJS` Streaming (`BaseExportProcessor.ts:178-179` & `FileGenerator.ts:46`):**
  - Đã kiểm chứng trên `ExcelJS.stream.xlsx.WorkbookWriter` thật và `ExcelJS.Workbook().xlsx.readFile` (`EV-20261110T164834-57d6`, `EV-20261110T165015-24f7`): tên `fileName.vi` (`DD-MM-YYYY_Don_Hang_Thu_Chi_Ho`, 30 ký tự) và `fileName.en` (`DD-MM-YYYY_Transfer_Orders_List`, 31 ký tự) đều nằm trong giới hạn cứng $\le 31$ ký tự của worksheet name trong chuẩn OOXML Excel.

---

## 3. (c) Zero-Mockup & Real Runtime Trace (`EV-...` Receipts)

Toàn bộ kết quả kiểm thử và thực thi đều được lấy biên lai vật lý qua `evidence_run.py`:
1. **Unit Tests (`EV-20261110T165315-5e24`, `exit=0`):**
   - Lệnh: `npx vitest run test/unit/models/payment-bill-document.model.test.ts test/unit/pure/collection-disbursement-orders.pure.test.ts test/unit/processors/CollectionDisbursementOrdersExport.test.ts test/unit/domain/handlers/GetAllCollectionDisbursementOrdersExportHandler.test.ts test/unit/domain/QueryHandlerFactory.test.ts test/unit/logics/export.logic.test.ts` (`EV-20261110T165315-5e24`).
   - Kết quả (`EV-20261110T165315-5e24`): `Test Files 6 passed (6)`, `Tests 41 passed (41)`.
2. **Real MongoDB + Excel SAGA Integration Tests (`EV-20261110T164834-57d6`, `exit=0`):**
   - Lệnh: `npx vitest run --config vitest.integration.config.ts test/integration/collection-disbursement-orders-export.integration.test.ts` (`EV-20261110T164834-57d6`).
   - Kết quả (`EV-20261110T164834-57d6`): Khởi chạy `mongod` thật qua `MongoMemoryServer`, insert 3 bản ghi (`2` bản ghi `TRANSFER_BIZ` + `1` bản ghi `PAYMENT_BIZ`) và `1` chứng từ `payment-bill-documents`, chạy full SAGA 5 bước sinh file `.xlsx` thật trên ổ đĩa và đọc lại bằng `ExcelJS`: `Test Files 1 passed (1)`, `Tests 3 passed (3)`.
3. **Production Build `tsc -p tsconfig.build.json` (`EV-20261110T165012-55d3`, `exit=0`):**
   - Lệnh: `npm run build` (`EV-20261110T165012-55d3`).
   - Kết quả (`EV-20261110T165012-55d3`): Biên dịch TypeScript sang `dist/` hoàn tất (`exit=0`).
4. **Trace trên bản build `dist/` (`EV-20261110T165015-24f7`, `exit=0`):**
   - Lệnh: `node /Users/trainguyen/.gemini/antigravity/brain/6e97e549-1102-4625-96a3-e664f1e29f68/scratch/runtime_verify.js` (`EV-20261110T165015-24f7`).
   - Kết quả (`EV-20261110T165015-24f7`): `{"jobId":"export-1794304216906-1xspekw5n","exportType":"CollectionDisbursementOrdersExport","sheetName":"10-11-2026_Don_Hang_Thu_Chi_Ho","rowCount":2,"exportedRow":[1,"10/11/2026 16:50:16","MT-RT-5001","Thành công","rt@goopay.vn","Người dùng Ví GooPay","WALLET_RT_1","Runtime Payer","Merchant","MC_RT","Runtime Payee","COLLECTION_SERVICE",320000,320000,"VND","Yes","GOOPAY","10/11/2026 16:50:16"]}`.

---

## 4. (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 → S3 → S4 → S5`)

- Trong quá trình chạy Integration Test với `ExcelJS` thật, hệ thống phát hiện `BaseExportProcessor.ts:178` lấy trực tiếp `config.fileName[langCode]` làm `sheetName` truyền vào `ExcelJS.addWorksheet(sheetName)`. Chuỗi dài `> 31` ký tự khiến `ExcelJS` cắt chuỗi và báo lỗi trùng worksheet name khi đọc lại file `.xlsx`.
- Đã kích hoạt vòng lặp tự chữa lành khép kín (`S5 → S3 → S4 → S5`):
  1. Cập nhật `09_tasks_solution_collection_disbursement_orders_export.md` (`fileName.vi` = `DD-MM-YYYY_Don_Hang_Thu_Chi_Ho` [30 ký tự], `fileName.en` = `DD-MM-YYYY_Transfer_Orders_List` [31 ký tự]) và xác minh qua `06_laya_summary_dg1_20261110T164906.json`, `06_laya_summary_dg2_20261110T164906.json` (`EV-20261110T164834-57d6`).
  2. Cập nhật `logics/export/payment-bill/collection-disbursement-orders-export.pure.ts:182-185`.
  3. Chạy lại Integration Test (`EV-20261110T164834-57d6`), Unit Test (`EV-20261110T165315-5e24`), Build (`EV-20261110T165012-55d3`) và Script Verification (`EV-20261110T165015-24f7`) hoàn tất 100%.

---

## 5. (e) Hội đồng `/laya_router` (`DG3`) & Cổng Chất lượng (`G1–G8` + Security Gate)

- **Kết quả `DG3` (`laya_gate.py run --stage dg3` sau lần sửa code cuối):**
  - File biên bản: `06_laya_summary_dg3_20261110T165104.json` (`EV-20261110T165315-5e24`).
  - Toàn bộ 4 claims kiểm định code thực tế (`A-FILTER`, `A-PROCESSOR`, `A-HANDLER`, `A-REGISTRY`) đều đạt `ACCEPT` ở vòng `r3` (`confidence`: `0.8502`, `0.7432`, `0.7621`, `0.9016`).
- **Đối chiếu 8 Cổng Chất lượng (`G1–G8`) & `/security-agent`:**
  - **G1 (Traceability):** Khớp 100% yêu cầu trong `01_requirements_collection_disbursement_orders_export.md` (sử dụng `/api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport`, không sửa `admin-portal-gateway`).
  - **G2 (Reproduce Red→Green):** Tái hiện và sửa dứt điểm lỗi giới hạn 31 ký tự tên sheet `.xlsx` (`EV-20261110T164834-57d6`).
  - **G3 (Real DB/File Verification):** Đã kiểm chứng trên `mongod` thật (`MongoMemoryServer`) và đọc/ghi file `.xlsx` thật qua `ExcelJS` (`EV-20261110T164834-57d6`, `EV-20261110T165015-24f7`).
  - **G4 (Edge-case & Concurrency):** Đã kiểm tra null/empty fields, fallback `embedData.uploaded`, lỗi `aggregateInvoiceFlagsByBillIds`, regex escaping (`escapeRegex`) chống ReDoS/NoSQL regex injection (`EV-20261110T165315-5e24`).
  - **G5 (Regression):** Các test hiện hữu (`QueryHandlerFactory.test.ts`, `export.logic.test.ts`, `EV-20261110T165315-5e24`) và `npm run build` (`EV-20261110T165012-55d3`) đều hoàn tất.
  - **G6 (Output Correctness):** Kiểm chứng `count(TRANSFER_BIZ nguồn) == count(dòng dữ liệu trong file Excel đích)` và loại trừ hoàn toàn bản ghi `PAYMENT_BIZ` (`EV-20261110T164834-57d6`, `EV-20261110T165015-24f7`).
  - **G7 (Adversarial Audit & Security Gate):** Không chứa hardcoded secret/PII, escape regex đầy đủ cho mọi input string tìm kiếm, không chạy `git push`.
  - **G8 (Physical Evidence):** Lưu đầy đủ biên lai `EV-20261110T165315-5e24`, `EV-20261110T164834-57d6`, `EV-20261110T165012-55d3`, `EV-20261110T165015-24f7` và `06_laya_summary_dg3_20261110T165104.json`.
