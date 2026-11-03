---
tier: T2
suffix: auto_recon_export
workspace: audit-auto-recon-export-2026-11-02
status: propose-only
created_at: 2026-11-02T13:51:00+07:00
---

# 09. Technical Solution — Khắc phục Triệt để các Sơ hở Sau Audit Nhánh `feat/auto-recon-export` (`centralized-export-service`)

## 1. Bảng Tổng Hợp Đối Chiếu Plan 8 Bước vs Code Thực Tế (`699a833`) & Các Lỗi Mới Phát Sinh

| Mã Finding | Bước Plan | Mức độ | Vị trí (`file:line`) | Kết quả Audit so với Plan & `develop` | Trạng thái |
| :--- | :---: | :---: | :--- | :--- | :---: |
| **`F2`** | Bước 2 | 🔴 **CRITICAL** | `logics/processors/BaseExportProcessor.ts:24-36, 236-238`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:100-108`<br>`test/unit/logics/merchant-reconcile-transactions.export.test.ts:433-456` | **Bị bỏ qua hoàn toàn (0%):** Chưa thêm cờ opt-in `failOnSubQueryError?: boolean` vào `ExportConfig` và `BaseExportProcessor.ts:236-238`. Khi sub-query phí (`v_fee_rows`) lỗi/timeout, worker vẫn nuốt lỗi và xuất file Excel `COMPLETED` với cột phí trống (`null`), đánh đồng lỗi hệ thống với giao dịch không có phí. | **OPEN** |
| **`F3`** | Bước 6 | 🔴 **CRITICAL** | `package-lock.json`<br>`package.json:43,45`<br>`Dockerfile:12` | **Làm thiếu gây vỡ Build & Test (`EV-20261102T134637-0c3b`, `EV-20261102T134647-ab1c`):** Commit `699a833` chỉ cập nhật `yarn.lock` mà bỏ quên `package-lock.json` (trong khi `Dockerfile:12` dùng `npm install`) và chưa cài đặt `@clickhouse/client`, `@hypequery/clickhouse` vào `node_modules`, khiến `tsc -p tsconfig.build.json` báo 7 lỗi `TS2307` và `npx vitest run` gãy ngay khi import. | **OPEN** |
| **`F1-D`** | Bước 1 | 🔴 **HIGH** | `domain/handlers/auto-reconcile/GetAllMerchantReconcileTransactionsExportHandler.ts:23-26`<br>`models/auto-reconcile/merchant-reconcile-result.model.ts:89-93` | **Bug nối chuỗi `limit + 1` khi `limit` là string:** Handler truyền thẳng `limit` từ `options?.params` vào `transactions.getAll(query.filterConditions, cursor, limit)` mà không ép `Number(limit)`. Nếu `limit` là `"50"` hoặc `"2000"` (từ env/params), `limit + 1` tại dòng `93` thành `"501"` / `"20001"`. | **OPEN** |
| **`F1-A/B/C`** | Bước 1 | 🟠 **HIGH** | `models/auto-reconcile/merchant-reconcile-result.model.ts:67-103, 176-193`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:85-94` | **(1) Sai chữ ký `buildListQuery`:** Chưa nhận `limit` và chưa gắn `.limit(limit + 1)` trong `buildListQuery` như Plan.<br>**(2) Bọc hàm vào cột index vế trái:** Dùng `toUnixTimestamp64Milli(updated_at)` ở vế trái `WHERE` thay vì giữ cột gốc `updated_at` đối chiếu với `fromUnixTimestamp64Milli(toInt64(?))`.<br>**(3) Mutate In-Place sai nguyên lý Pure:** `toExportRow` (`delete item.cursorUpdatedAt`) và `mergeData` (`row.dailyFee = ...`) sửa trực tiếp object đầu vào với lý do "cả tập kết quả nằm trong RAM" — tự mâu thuẫn với việc đã cắt chunk 2,000 dòng. | **OPEN** |
| **`F5`** | Bước 4 | 🟠 **HIGH** | `data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts:2-42`<br>`data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts:2-3, 118-124`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:4-14` | **Mới làm 1/3 (vẫn đặt sai Layer so với Design Pattern gốc):**<br>- `MERCHANT_TRANSACTION_COLUMNS` (nhãn UI `vi`/`en`) và `MERCHANT_TRANSACTION_FIELDS` vẫn nằm trong `entity.ts` thay vì chuyển về `pure.ts`.<br>- Hàm `validateMerchantTransactionFilters` vẫn nằm trong `params.ts` thay vì chuyển về `validate` trong `pure.ts` (tầng `params/` theo chuẩn `develop` chỉ chứa DTO class).<br>- `pure.ts` vẫn chưa import `MerchantReconcileTransactionsExportParams` qua barrel `../../../data-transfers/params`. | **OPEN** |
| **`F9`** | Bước 7 & 8 | 🟠 **MEDIUM** | `test/live/merchant-reconcile-transactions.clickhouse.test.ts:1-118`<br>`vitest.integration.config.ts:10` | **Chưa chuyển thư mục & chưa test multi-page:**<br>- File vẫn nằm ở `test/live/` nên `npm run test:integration` (`test/integration/**`) bỏ sót hoàn toàn.<br>- Thiếu `console.warn` khi skip (`L003`).<br>- Bài test `"205 rows"` (dòng `111-117`) vẫn gọi 1 trang duy nhất (`limit` mặc định 2000), chưa lặp qua 5 trang với `limit: 50` như Plan Bước 8 yêu cầu. | **OPEN** |
| **`F6` & `F6-NEW`** | Bước 5 | 🟡 **MEDIUM** | `dbHandler/DbConnectorFactory.ts:108-110, 165-170`<br>`dbHandler/clickhouse.ts:5-8, 43` | **(1) Vị trí check config:** `case DbType.CLICKHOUSE:` trong `switch` (`DbConnectorFactory.ts:108`) vẫn chưa đặt check `!config.host || !config.database || !config.username` giống các case `MONGODB`/`POSTGRES`/`MARIADB`.<br>**(2) Lệch default timeout & comment nghiệp vụ trong infra:** `clickhouse.ts:8` tự đặt `DEFAULT_READ_TIMEOUT_MS = 300000` kèm comment nhắc `"dashboard API"`, lệch với `DbConnectorFactory.ts:170` (`20000`) và `.run.local.env.example:119` (`20000`). | **OPEN** |
| **`F4`** | Bước 3 & 8 | 🟡 **LOW** | `models/auto-reconcile/merchant-reconcile-result.model.ts:195-236` | Đã `throw new Error` khi gặp field lạ, nhưng vẫn khởi tạo lại object `mapping` (30 keys) mỗi lần gọi `getColumnName` thay vì tách hằng `COLUMN_MAPPING` ở cấp module, và thiếu unit test `getColumnName("foo")` throw. | **OPEN** |
| **`F_LINT` & `F_TSC`** | Bước 8 | 🟡 **MEDIUM** | `merchant-reconcile-transactions.entity.ts:39`<br>`clickhouse.test.ts:10,84`<br>`merchant-reconcile-transactions.export.test.ts:105,381,437` | **Lỗi ESLint & TypeScript (`EV-20261102T134642-0154`, `EV-20261102T134637-0c3b`):**<br>- 2 lỗi ESLint (`comma-dangle` tại `entity.ts:39`, `quotes` tại `export.test.ts:437`).<br>- 5 lỗi `tsc --noEmit` trong `clickhouse.test.ts:84` (`DbConfig` type) và `merchant-reconcile-transactions.export.test.ts:112,317,333,381` (`LoggerInstance` & `sheet.getRow(1).values`). | **OPEN** |
| **`F10`** | Bước 7 | 🟢 **OK** | `defined/validation-message.ts:12-19`<br>`merchant-reconcile-transactions-export.params.ts:6-9` | Đã bổ sung `ERR_TAB_INVALID` vào `validation-message.ts` và gắn `@IsIn(["list", "discrepancy"], { message: "ERR_TAB_INVALID" })` kèm unit test i18n. | **RESOLVED** |

---

## 2. Kiểm tra Độ Phình File (`wc -l`) & Phân Tầng Kiến Trúc (`L012`)

| File | Số dòng trên `develop` | Số dòng hiện tại (`HEAD`) | Đánh giá Single-Responsibility (`L012`) |
| :--- | :---: | :---: | :--- |
| `logics/processors/BaseExportProcessor.ts` | `391` | `391` (`+3` sau khi thêm `failOnSubQueryError`) | Chỉ thêm 1 cờ opt-in `failOnSubQueryError?: boolean` (3 dòng), giữ nguyên 100% hành vi mặc định cho 30+ export cũ. |
| `start.ts` | `316` | `333` (`+17`) | Chỉ thêm nhánh `else if (config.type === "clickhouse")` trong vòng lặp `DB_LIST`, đúng mẫu `mongo` và `mariadb`. |
| `dbHandler/DbConnectorFactory.ts` | `236` | `287` (`+51`) | Thêm `DbType.CLICKHOUSE` và `connectClickHouse`, tách wrapper riêng tại `dbHandler/clickhouse.ts` (`69` dòng). |
| `models/auto-reconcile/merchant-reconcile-result.model.ts` | `0` (file mới) | `239` | Đơn trách nhiệm cho bảng `merchant_reconcile_results`. |
| `models/auto-reconcile/fee-row-view.model.ts` | `0` (file mới) | `78` | Tách riêng model cho view `v_fee_rows` (không nhồi vào model giao dịch). |
| `models/auto-reconcile/auto-reconcile.schema.ts` | `0` (file mới) | `70` | Tách riêng định nghĩa schema ClickHouse. |
| `logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts` | `0` (file mới) | `112` (`~155` sau khi nhận `COLUMNS` + `validate` từ entity/params) | Đúng chuẩn `*-export.pure.ts` của dự án (`centralrecon-export.pure.ts` dài 260 dòng). |
| `data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts` | `0` (file mới) | `109` (`~65` sau khi chuyển `COLUMNS` sang `pure.ts`) | Thuần túy chứa `FILTERS` và các `type`/`interface` domain. |
| `data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts` | `0` (file mới) | `124` (`~115` sau khi chuyển `validate` sang `pure.ts`) | Thuần túy DTO class `class-validator`. |

---

## 3. Bảng Đấu Nối Điểm Gọi Thực Thi (Explicit Runtime Call-Site & Lifecycle Trigger Wiring — `L008`)

| Thành phần / Hàm sửa đổi | Điểm gọi (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle Stage / Event Trigger`) |
| :--- | :--- | :--- |
| `DbConnectorFactory.connect` (`case DbType.CLICKHOUSE`) | `dbHandler/clickhouse.ts:51` <- `start.ts:251` | Khi khởi động service (`CentralizedExportService.initService` duyệt `DB_LIST`). |
| `ExportPure.validate(params, langCode)` | `logics/export/auto-reconcile/merchant-reconcile-transactions.export.ts:8` & `pure.ts:97` | (1) Khi nhận RPC `exportAsync` (`BaseExportProcessor.process()` gọi `this.validate()`); (2) Khi Worker chạy `getConfig()`. |
| `GetAllMerchantReconcileTransactionsExportHandler.execute` | `domain/QueryBus.ts` <- `logics/processors/BaseExportProcessor.ts:209-211` | Trong bước Saga `query_and_generate`, lặp `do ... while (cursor)` cho từng chunk (`limit = chunkSize`). |
| `MerchantReconcileResultModel.getPage` & `buildListQuery` & `countDocuments` | `domain/handlers/auto-reconcile/GetAllMerchantReconcileTransactionsExportHandler.ts:25-28` | Mỗi vòng lặp chunk gọi `getPage(filterConditions, cursor, Number(limit))`; chỉ vòng đầu (`!cursor`) gọi `countDocuments`. |
| `BaseExportProcessor.execute` (`if (config.failOnSubQueryError) throw err`) | `logics/processors/BaseExportProcessor.ts:236-239` | Khi `queryBus.execute(subQueryInstance)` (`GetFeeTotalsByGoopayCodesQuery`) ném lỗi trong bước Saga `query_and_generate` -> kích hoạt `compensate` xóa file tạm và đánh dấu job `FAILED`. |

---

## 4. Phương Án Kỹ Thuật Duy Nhất & Code Demo Từng File (Đã Qua Wave 2 Phản Biện Plan `P1–P4`)

### 4.0. Bốn Điểm Hiệu Chỉnh Quan Trọng Từ Wave 2 Phản Biện Lại Chính Plan (`P1–P4`)
1. **`P1` (Chặn đứng Circular Dependency & Upward Layer Import ở Bước 3 / `F5`):**
   - **Phát hiện phản biện:** Nếu `entity.ts` hoặc `params.ts` dùng `export { ... } from "../../../logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure"` trong khi `pure.ts` lại import từ `data-transfers/params` và `data-transfers/entities`, sẽ tạo ra **Circular Dependency** (`params/index.ts -> params.ts -> pure.ts -> params/index.ts`, làm `MerchantReconcileTransactionsExportParams` bị `undefined` khi nạp module) và vi phạm chiều phụ thuộc 1 chiều (`logics -> domain -> models -> data-transfers`).
   - **Quyết định chuẩn:** Xóa dứt điểm `MERCHANT_TRANSACTION_COLUMNS`/`FIELDS` khỏi `entity.ts` và xóa dứt điểm `validateMerchantTransactionFilters` khỏi `params.ts` (không re-export ngược từ `logics/`). Đặt chúng tại `pure.ts` và cập nhật câu lệnh `import` ở 2 file test (`test/unit/.../export.test.ts` và `test/integration/merchant-reconcile-transactions.clickhouse.test.ts`) trỏ về `ExportPure`. Đồng thời trong `pure.ts`, import cả Queries qua barrel `../../../domain/queries` và Params qua barrel `../../../data-transfers/params` đúng chuẩn `centralrecon-export.pure.ts:3-5`.
2. **`P2` (Bảo toàn miền `Int64` của Keyset Cursor và giữ nguyên 100% các assertion Unit Test hiện có — `L008`):**
   - **Phát hiện phản biện:** Cột cursor ở `SELECT` (`merchant-reconcile-result.model.ts:55`) là `toUnixTimestamp64Milli(updated_at) AS cursorUpdatedAt` (`Int64`). Nếu đổi vế trái `WHERE` sang cột `updated_at` (`DateTime64(6)` hoặc `DateTime64(9)`) so với `fromUnixTimestamp64Milli(toInt64(cursor.updatedAtMs))` (`DateTime64(3)`), phần microsecond/nanosecond lẻ (`.123456` vs `.123000`) sẽ làm mệnh đề `<` bị sai lệch khi 2 dòng cùng millisecond và làm gãy unit test hiện có (`export.test.ts:121, 403`). Tương tự, việc đưa `.limit()` vào `buildListQuery()` sẽ làm gãy unit test `export.test.ts:320` (`expect(sql).not.toMatch(/LIMIT|OFFSET|x' OR 1=1/)`), và việc đổi `mergeData` sang `.map()` sẽ làm gãy unit test `export.test.ts:297` (`expect(merged).toBe(transactions)`).
   - **Quyết định chuẩn:** Giữ nguyên `(toUnixTimestamp64Milli(updated_at), goopay_code) < (Number(cursor.updatedAtMs), cursor.goopayCode)` trong `buildListQuery()`, giữ `.limit(pageSize + 1)` trong `getAll()` với `const pageSize = Number(limit) > 0 ? Number(limit) : DEFAULT_PAGE_SIZE;`, và giữ `mergeData` mutate trên chunk hiện tại (chỉ sửa comment dòng `87` cho đúng bản chất từng chunk).
3. **`P3` (Giữ nguyên 100% `GetAllMerchantReconcileTransactionsExportHandler.ts`):**
   - File `domain/handlers/auto-reconcile/GetAllMerchantReconcileTransactionsExportHandler.ts` ở `699a833` đã viết đúng contract (`const { cursor, limit } = options?.params || {};`, gọi `transactions.getAll` và `transactions.countDocuments`), và `getAll` ở tầng Model đã có default `limit = DEFAULT_PAGE_SIZE` kèm ép kiểu `pageSize`. Không cần chạm vào `GetAllMerchantReconcileTransactionsExportHandler.ts`.
4. **`P4` (Đặt file Integration Test phẳng tại `test/integration/merchant-reconcile-transactions.clickhouse.test.ts`):**
   - Đặt trực tiếp tại `test/integration/merchant-reconcile-transactions.clickhouse.test.ts` (cùng cấp với `test/integration/export-pipeline.integration.test.ts` và `test/integration/payment-bill-export.integration.test.ts`) thay vì tạo thư mục con `test/integration/auto-reconcile/`, giúp bảo toàn 100% độ sâu đường dẫn tương đối `../../` trong file test.

### 4.1. Khắc phục `F1-D` & `F4`: Chuẩn hóa `pageSize = Number(limit)` và tách hằng `COLUMN_MAPPING` trong `models/auto-reconcile/merchant-reconcile-result.model.ts`

```diff
--- a/models/auto-reconcile/merchant-reconcile-result.model.ts
+++ b/models/auto-reconcile/merchant-reconcile-result.model.ts
@@ -55,6 +55,39 @@ const SELECT_COLUMNS = [
 // MERCHANT_TRANSACTION_FIELDS nên cột thừa này không lọt vào file.
 const CURSOR_COLUMN = rawAs<string, "cursorUpdatedAt">("toUnixTimestamp64Milli(updated_at)", "cursorUpdatedAt");
 
+const COLUMN_MAPPING: Readonly<Record<string, string>> = {
+    isDeleted: "is_deleted",
+    goopayCode: "goopay_code",
+    merchantCode: "merchant_code",
+    merchantTransId: "merchant_trans_id",
+    partnerTransId: "partner_trans_id",
+    originalTransId: "original_trans_id",
+    walletPhone: "user_wallet_phone",
+    payerCode: "payer_code",
+    payeeCode: "payee_code",
+    referenceCode: "reference_code",
+    runId: "run_id",
+    transType: "trans_type",
+    partner: "partner_code",
+    serviceType: "service_type",
+    gpStatus: "goopay_status",
+    reconChannel: "reconcile_channel",
+    merchantStatus: "merchant_status",
+    reconcileStatus: "reconcile_status",
+    paymentChannel: "channel_id",
+    connector: "connector",
+    errorCode: "error_code",
+    createdAt: "created_at",
+    createdAtFrom: "created_at",
+    createdAtTo: "created_at",
+    executedAt: "executed_at",
+    executedAtFrom: "executed_at",
+    executedAtTo: "executed_at",
+    updatedAt: "updated_at",
+    updatedAtFrom: "updated_at",
+    updatedAtTo: "updated_at"
+};
+
 class MerchantReconcileResultModel {
     private readonly logger: LoggerInstance;
     private readonly db: AutoReconcileDB;
@@ -73,7 +106,7 @@ class MerchantReconcileResultModel {
             // goopay_code sẽ mất một — hiếm, và an toàn hơn là lặp vô hạn.
             query = query.where(expr => expr.fn("less",
                 expr.fn("tuple", expr.fn("toUnixTimestamp64Milli", expr.col("updated_at")), expr.col("goopay_code")),
-                expr.fn("tuple", expr.literal(cursor.updatedAtMs), expr.literal(cursor.goopayCode))));
+                expr.fn("tuple", expr.literal(Number(cursor.updatedAtMs)), expr.literal(cursor.goopayCode))));
         }
 
         return query.select([
@@ -89,11 +122,12 @@ class MerchantReconcileResultModel {
         limit = DEFAULT_PAGE_SIZE
     ): Promise<{ data: MerchantTransactionRow[]; nextCursor: MerchantTransactionCursor | null }> {
         try {
+            const pageSize = Number(limit) > 0 ? Number(limit) : DEFAULT_PAGE_SIZE;
             // limit+1 để biết còn trang sau mà không phải đếm lại.
-            const rows = await this.buildListQuery(conditions, cursor).limit(limit + 1).execute({ cache: false });
+            const rows = await this.buildListQuery(conditions, cursor).limit(pageSize + 1).execute({ cache: false });
 
-            const hasMore = rows.length > limit;
-            const data = (hasMore ? rows.slice(0, limit) : rows) as MerchantTransactionRow[];
+            const hasMore = rows.length > pageSize;
+            const data = (hasMore ? rows.slice(0, pageSize) : rows) as MerchantTransactionRow[];
             // Đọc khoá trước toExportRow, vì bước đó xoá cột cursor khỏi dòng.
             const nextCursor = hasMore ? this.toCursor(data[data.length - 1]) : null;
 
@@ -192,41 +226,8 @@ class MerchantReconcileResultModel {
         return item;
     }
 
-    private getColumnName(fieldName: string): string {
-        const mapping: Record<string, string> = {
-            isDeleted: "is_deleted",
-            goopayCode: "goopay_code",
-            merchantCode: "merchant_code",
-            merchantTransId: "merchant_trans_id",
-            partnerTransId: "partner_trans_id",
-            originalTransId: "original_trans_id",
-            walletPhone: "user_wallet_phone",
-            payerCode: "payer_code",
-            payeeCode: "payee_code",
-            referenceCode: "reference_code",
-            runId: "run_id",
-            transType: "trans_type",
-            partner: "partner_code",
-            serviceType: "service_type",
-            gpStatus: "goopay_status",
-            reconChannel: "reconcile_channel",
-            merchantStatus: "merchant_status",
-            reconcileStatus: "reconcile_status",
-            paymentChannel: "channel_id",
-            connector: "connector",
-            errorCode: "error_code",
-            createdAt: "created_at",
-            createdAtFrom: "created_at",
-            createdAtTo: "created_at",
-            executedAt: "executed_at",
-            executedAtFrom: "executed_at",
-            executedAtTo: "executed_at",
-            updatedAt: "updated_at",
-            updatedAtFrom: "updated_at",
-            updatedAtTo: "updated_at"
-        };
-
-        const column = mapping[fieldName];
+    public getColumnName(fieldName: string): string {
+        const column = COLUMN_MAPPING[fieldName];
         if (!column) {
             // Lọt tới đây nghĩa là export config thêm field mới mà quên mapping;
             // để nguyên tên camelCase thì ClickHouse chỉ báo "unknown identifier".
             throw new Error(`MerchantReconcileResultModel: no column mapped for field '${fieldName}'`);
         }
         return column;
     }
```

---

### 4.2. Khắc phục `F2`: Bổ sung cờ opt-in `failOnSubQueryError` vào `BaseExportProcessor.ts` và bật cho Auto-Reconcile Export

#### (a) File `logics/processors/BaseExportProcessor.ts`
```diff
--- a/logics/processors/BaseExportProcessor.ts
+++ b/logics/processors/BaseExportProcessor.ts
@@ -31,6 +31,8 @@ export interface ExportConfig {
   subQueryClass?: any;
   buildSubQueryParams?: (primaryData: any[]) => any;
   mergeData?: (primaryData: any[], subData: any[]) => any[];
+  /** true: lỗi sub-query làm job FAILED thay vì xuất file thiếu dữ liệu. Mặc định false (giữ hành vi cũ). */
+  failOnSubQueryError?: boolean;
   getSummaryRow?: (currentSummary: any, chunkData: any[]) => any;
   formatSummary?: (finalSummary: any) => any[];
 }
@@ -235,6 +237,7 @@ export abstract class BaseExportProcessor {
 
             } catch (err) {
               this.context.broker.logger.error(`Error executing sub-query for job ${jobId}`, err);
+              if (config.failOnSubQueryError) { throw err; }
             }
           }
```

---

### 4.3. Khắc phục `F5` & `F_LINT`: Đưa `MERCHANT_TRANSACTION_COLUMNS` và `validate` về đúng Layer `pure.ts` theo Design Pattern gốc

#### (a) File `data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts`
- Xóa `MERCHANT_TRANSACTION_COLUMNS` và `MERCHANT_TRANSACTION_FIELDS` khỏi `entity.ts` (chuyển sang `pure.ts`), chỉ giữ `MERCHANT_TRANSACTION_FILTERS` và các `type`/`interface` domain.

```diff
--- a/data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts
+++ b/data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts
@@ -1,49 +1,4 @@
-// Export columns match the merchant table selection and order.
-export const MERCHANT_TRANSACTION_COLUMNS = [
-...
-] as const;
-
-export const MERCHANT_TRANSACTION_FIELDS = MERCHANT_TRANSACTION_COLUMNS.map(column => column.field);
-
 export const MERCHANT_TRANSACTION_FILTERS = [
     "tab",
```

#### (b) File `data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts`
- Xóa `validateMerchantTransactionFilters` khỏi `params.ts`, giữ file thuần DTO `class-validator` như toàn bộ các file `*.params.ts` khác trên `develop`.

```diff
--- a/data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts
+++ b/data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts
@@ -1,6 +1,4 @@
 import { IsDateString, IsIn, IsNotEmpty, IsOptional, IsString } from "class-validator";
-import { validateParams } from "../../../utils/validator";
-import { MERCHANT_TRANSACTION_FILTERS } from "../../entities/auto-reconcile/merchant-reconcile-transactions.entity";
 
 export class MerchantReconcileTransactionsExportParams {
     @IsNotEmpty({ message: "ERR_TAB_REQUIRED" })
@@ -114,11 +112,3 @@ export class MerchantReconcileTransactionsExportParams {
     @IsString()
     public errorCode?: string;
 }
-
-// Select only supported filters: input paging never limits an export.
-export const validateMerchantTransactionFilters = (params: Record<string, unknown>, language = "vi"): MerchantReconcileTransactionsExportParams => {
-    const filters = Object.fromEntries(MERCHANT_TRANSACTION_FILTERS.map(key => {
-        const value = params?.[key];
-        return [key, typeof value === "string" ? value.trim() || undefined : value ?? undefined];
-    }));
-    return validateParams<MerchantReconcileTransactionsExportParams>(MerchantReconcileTransactionsExportParams, filters, language);
-};
```

#### (c) File `logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts`
- Đặt `MERCHANT_TRANSACTION_COLUMNS`, `MERCHANT_TRANSACTION_FIELDS`, và hàm `validate` tại `pure.ts`; import Queries qua barrel `../../../domain/queries` và Params qua barrel `../../../data-transfers/params`; giữ `mergeData` in-place trên chunk hiện tại (cập nhật comment dòng `87-88` cho đúng bản chất từng chunk); bật `failOnSubQueryError: true`.

```diff
--- a/logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts
+++ b/logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts
@@ -1,19 +1,58 @@
 import { ExportConfig } from "../../processors/BaseExportProcessor";
-import { GetAllMerchantReconcileTransactionsExportQuery } from "../../../domain/queries/auto-reconcile/GetAllMerchantReconcileTransactionsExportQuery";
-import { GetFeeTotalsByGoopayCodesQuery } from "../../../domain/queries/auto-reconcile/GetFeeTotalsByGoopayCodesQuery";
 import {
-    MerchantReconcileTransactionsExportParams,
-    validateMerchantTransactionFilters
-} from "../../../data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params";
+    GetAllMerchantReconcileTransactionsExportQuery,
+    GetFeeTotalsByGoopayCodesQuery
+} from "../../../domain/queries";
+import { validateParams } from "../../../utils/validator";
+import { MerchantReconcileTransactionsExportParams } from "../../../data-transfers/params";
 import {
-    MERCHANT_TRANSACTION_COLUMNS,
-    MERCHANT_TRANSACTION_FIELDS,
+    MERCHANT_TRANSACTION_FILTERS,
     MerchantFeeTotalsRow,
     MerchantFilterCondition,
     MerchantTransactionRow
 } from "../../../data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity";
 
+// Export columns match the merchant table selection and order.
+export const MERCHANT_TRANSACTION_COLUMNS = [
+    { field: "goopayCode", vi: "Mã GooPay", en: "GooPay Code" },
+    { field: "createdAt", vi: "Thời gian tạo", en: "Created Time" },
+    { field: "executedAt", vi: "Thời gian đi lệnh", en: "Order Sent Time" },
+    { field: "transType", vi: "Loại giao dịch/yêu cầu", en: "Transaction Type/Request" },
+    { field: "partner", vi: "Đối tác", en: "Partner" },
+    { field: "merchantCode", vi: "Mã Merchant", en: "Merchant Code" },
+    { field: "merchantTransId", vi: "Mã giao dịch merchant", en: "Merchant Transaction ID" },
+    { field: "originalTransId", vi: "Giao dịch gốc", en: "Original Transaction ID" },
+    { field: "serviceType", vi: "Loại dịch vụ", en: "Service Type" },
+    { field: "gpStatus", vi: "Trạng thái giao dịch GP", en: "GP Transaction Status" },
+    { field: "amount", vi: "Giá trị giao dịch", en: "Transaction Amount" },
+    { field: "dailyFee", vi: "Phí tạm tính ngày", en: "Estimated Daily Fee" },
+    { field: "monthlyFee", vi: "Phí tạm tính tháng", en: "Estimated Monthly Fee" },
+    { field: "reconChannel", vi: "Kênh", en: "Channel" },
+    { field: "runId", vi: "Run ID", en: "Run ID" },
+    { field: "merchantStatus", vi: "Trạng thái giao dịch Merchant", en: "Merchant Transaction Status" },
+    { field: "reconcileStatus", vi: "Trạng thái đối soát", en: "Reconciliation Status" },
+    { field: "updatedAt", vi: "Thời gian cập nhật", en: "Updated Time" },
+    { field: "description", vi: "Mô tả", en: "Description" },
+    { field: "walletUserName", vi: "Tên user ví", en: "Wallet User Name" },
+    { field: "walletPhone", vi: "SĐT ví", en: "Wallet Phone" },
+    { field: "walletSystemName", vi: "Tên ví hệ thống", en: "System Wallet Name" },
+    { field: "payerCode", vi: "Mã Payer", en: "Payer Code" },
+    { field: "payerName", vi: "Tên Payer", en: "Payer Name" },
+    { field: "payeeCode", vi: "Mã Payee", en: "Payee Code" },
+    { field: "payeeName", vi: "Tên Payee", en: "Payee Name" },
+    { field: "paymentChannel", vi: "Kênh thanh toán", en: "Payment Channel" },
+    { field: "bankReceiver", vi: "Ngân hàng nhận", en: "Receiving Bank" },
+    { field: "bankReceiverAccount", vi: "Tài khoản nhận", en: "Receiving Account" },
+    { field: "bankSender", vi: "Ngân hàng chuyển", en: "Sending Bank" },
+    { field: "bankSenderAccount", vi: "Tài khoản chuyển", en: "Sending Account" },
+    { field: "connector", vi: "Connector", en: "Connector" },
+    { field: "referenceCode", vi: "Mã tham chiếu", en: "Reference Code" },
+    { field: "errorCode", vi: "Mã lỗi", en: "Error Code" },
+    { field: "reason", vi: "Lý do", en: "Reason" },
+    { field: "note", vi: "Ghi chú", en: "Note" },
+    { field: "updatedBy", vi: "Người cập nhật", en: "Updated By" }
+] as const;
+
+export const MERCHANT_TRANSACTION_FIELDS = MERCHANT_TRANSACTION_COLUMNS.map(column => column.field);
+
 const STATUS_DISCREPANCY = "DISCREPANCY";
 const TAB_DISCREPANCY = "discrepancy";
@@ -64,7 +103,14 @@ export const buildMerchantReconcileFilter = (params: MerchantReconcileTransactio
     return conditions;
 };
 
-export const validate = validateMerchantTransactionFilters;
+// Select only supported filters: input paging never limits an export.
+export const validate = (params: Record<string, unknown>, language = "vi"): MerchantReconcileTransactionsExportParams => {
+    const filters = Object.fromEntries(MERCHANT_TRANSACTION_FILTERS.map(key => {
+        const value = params?.[key];
+        return [key, typeof value === "string" ? value.trim() || undefined : value ?? undefined];
+    }));
+    return validateParams<MerchantReconcileTransactionsExportParams>(MerchantReconcileTransactionsExportParams, filters, language);
+};
 
 const codeOf = (row: { goopayCode?: unknown }): string => String(row.goopayCode ?? "");
@@ -84,8 +130,8 @@ export const mergeData = (
     // A missing entry means the transaction has no fee rows at all, which the API
     // reports as a null fee rather than a zero.
     //
-    // Written in place on purpose: BaseExportProcessor hands us the rows it is about
-    // to write and keeps no other reference, and the whole result set is resident in
-    // memory, so returning a second array would double the export's peak heap.
+    // Written in place on purpose: BaseExportProcessor hands us the current chunk
+    // of rows it is about to write and keeps no other reference, so returning a
+    // second array would double the chunk's peak heap.
     transactions.forEach(row => {
         row.dailyFee = byCode.get(codeOf(row))?.dailyFee ?? null;
@@ -102,6 +148,7 @@ export const getConfig = (params: Record<string, unknown>, language = "vi"): Exp
         subQueryClass: GetFeeTotalsByGoopayCodesQuery,
         buildSubQueryParams,
         mergeData,
+        failOnSubQueryError: true,
         columns: MERCHANT_TRANSACTION_COLUMNS.map(({ vi, en }) => ({ vi, en })),
         fileName: { vi: "Doi_soat_merchant", en: "Merchant_reconciliation" },
         allowEmptyFile: true
```

---

### 4.4. Khắc phục `F6` & `F6-NEW`: Đặt check ClickHouse ngay trong `switch` của `DbConnectorFactory.ts` và thống nhất `readTimeout`

```diff
--- a/dbHandler/DbConnectorFactory.ts
+++ b/dbHandler/DbConnectorFactory.ts
@@ -106,6 +106,7 @@ export class DbConnectorFactory {
             connection = await this.connectMariaDB(config);
             break;
           case DbType.CLICKHOUSE:
+            if (!config.host || !config.database || !config.username) { throw new Error("ClickHouse requires host, database & username"); }
             connection = await this.connectClickHouse(config);
             break;
           default:
@@ -162,9 +163,6 @@ export class DbConnectorFactory {
 
   private async connectClickHouse(config: DbConfig): Promise<DbConnection> {
-    if (!config.host || !config.database || !config.username) {
-      throw new Error("ClickHouse requires host, database & username");
-    }
     const protocol = config.protocol || "http";
     const port = config.port ?? 8123;
     const timeout = config.readTimeout ?? 20000;
```

```diff
--- a/dbHandler/clickhouse.ts
+++ b/dbHandler/clickhouse.ts
@@ -2,9 +2,7 @@ import { ClickHouseClient } from "@clickhouse/client";
 import { LoggerInstance } from "moleculer";
 import DbConnectorFactory, { DbConfig, DbType } from "./DbConnectorFactory";
 
-// Kết nối này chỉ phục vụ export. clickhouse_settings bật wait_end_of_query=1
-// (server gom hết kết quả rồi mới trả byte đầu) nên hạn mức phải rộng hơn
-// nhiều so với 20s của dashboard API.
-const DEFAULT_READ_TIMEOUT_MS = 300000;
+const DEFAULT_READ_TIMEOUT_MS = 20000;
```

---

### 4.5. Khắc phục `F9`, `F_TSC`, `F_LINT` & Bước 8: Chuyển Integration Test về `test/integration/`, in cảnh báo khi skip (`L005`), bổ sung đầy đủ Unit Test & Đồng bộ `package-lock.json` (`F3`)

1. **Di chuyển `test/live/merchant-reconcile-transactions.clickhouse.test.ts` -> `test/integration/merchant-reconcile-transactions.clickhouse.test.ts` (`P4`):**
   - Đặt trực tiếp trong `test/integration/` (cùng cấp với `export-pipeline.integration.test.ts` và `payment-bill-export.integration.test.ts`) để bảo toàn 100% độ sâu `../../` của các câu lệnh import và được `vitest.integration.config.ts` tự động quét.
   - Đổi import `MERCHANT_TRANSACTION_FIELDS` và `validate` sang `ExportPure`.
   - Thêm cảnh báo rõ ràng khi chưa bật `RUN_CLICKHOUSE_EXPORT_TESTS=1` (chống silent skip theo `L005`).
   - Cập nhật hàm `run(filters, options)` nhận `options` và sửa test `"exports all 205 rows"` chạy với `limit: 50` lặp qua đủ **5 trang** (`50 + 50 + 50 + 50 + 5 = 205`), kiểm tra không trùng, không sót, đúng thứ tự `BULK-204 ... BULK-000`.
2. **Cập nhật `test/unit/dbHandler/clickhouse.test.ts` & `test/unit/logics/merchant-reconcile-transactions.export.test.ts`:**
   - Sửa kiểu `const config: DbConfig = ...` và `request_timeout: 20000` trong `clickhouse.test.ts:10,36`.
   - Đổi import `MERCHANT_TRANSACTION_COLUMNS, MERCHANT_TRANSACTION_FIELDS` sang `ExportPure` và bổ sung `fatal: vi.fn(), trace: vi.fn()` trên mock `LoggerInstance` + ép kiểu `(sheet.getRow(1).values as any[]).slice(1)` trong `merchant-reconcile-transactions.export.test.ts` để sạch 100% lỗi `tsc` và `eslint`.
   - Thêm unit test `expect(() => model.getColumnName("unsupportedField")).toThrow(...)` (`F4`) và cập nhật test `failOnSubQueryError: true` làm job `FAILED` khi query phí lỗi (`F2`).
3. **Đồng bộ `package-lock.json` & `node_modules` (`F3`):**
   - Chạy `npm install` để cập nhật `package-lock.json` khớp với `package.json` và `Dockerfile`.
