---
tier: T2
suffix: auto_recon_export
workspace: audit-auto-recon-export-2026-11-02
status: propose-only
created_at: 2026-11-02T13:55:20+07:00
---

# Audit Report — Phản Biện Toàn Diện Nhánh `feat/auto-recon-export` (`699a833`) So Với `develop` (`centralized-export-service`)

## 1. Bảng Tổng Hợp Issues & Tổng Kết Hành Động (`Executive Issues Summary Table` — `L002`)

### 1.1. Các Vấn Đề Còn Tồn Đọng Trên Nhánh `feat/auto-recon-export` (`699a833`) Cần Xử Lý (`OPEN`)
| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Mô tả ngắn | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F2`** | Bước 2 Plan (Nuốt lỗi sub-query phí) | 🔴 **CRITICAL** | `logics/processors/BaseExportProcessor.ts:24-36, 236-238`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:100-108` | Commit `699a833` bỏ qua hoàn toàn Bước 2 của Plan: không thêm `failOnSubQueryError?: boolean` vào `BaseExportProcessor.ts` và không bật cờ này trong `pure.ts`. Khi `v_fee_rows` lỗi, job vẫn `COMPLETED` với cột phí trống. | **OPEN** — Chờ User `APPROVE` để thêm `failOnSubQueryError` (mặc định `false` giữ tương thích ngược). |
| **`F3`** | Bước 6 Plan (Lockfile & Dependencies) | 🔴 **CRITICAL** | `package-lock.json`<br>`package.json:43,45`<br>`Dockerfile:12` | Chỉ sửa `yarn.lock` mà bỏ quên `package-lock.json` (trong khi `Dockerfile:12` dùng `npm install`) và chưa cài `@clickhouse/client`, `@hypequery/clickhouse` vào `node_modules`. Hậu quả: `tsc` báo 7 lỗi `TS2307` (`EV-20261102T134637-0c3b`) và `vitest` lỗi ngay khi import (`EV-20261102T134647-ab1c`). | **OPEN** — Chờ User `APPROVE` để chạy `npm install` đồng bộ `package-lock.json` và `node_modules`. |
| **`F1-D`** | Bước 1 Plan (Keyset Pagination) | 🔴 **HIGH** | `models/auto-reconcile/merchant-reconcile-result.model.ts:89-96` | Nếu caller truyền `limit` là string (`"50"` hoặc `"2000"`), `limit + 1` tại dòng `93` nối chuỗi thành `"501"` / `"20001"`. (Qua phản biện Wave 2 `P2/P3`: giữ nguyên `GetAllMerchantReconcileTransactionsExportHandler.ts:17-34` và ép `const pageSize = Number(limit) > 0 ? Number(limit) : DEFAULT_PAGE_SIZE;` + `Number(cursor.updatedAtMs)` ngay tại `merchant-reconcile-result.model.ts`). | **OPEN** — Chờ User `APPROVE` để chuẩn hóa `pageSize = Number(limit)` tại Model. |
| **`F1-A/B/C`** | Bước 1 Plan (SQL Keyset & Immutability) | 🟢 **REFINED (`P2`)** | `models/auto-reconcile/merchant-reconcile-result.model.ts:67-103`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:85-94` | Qua phản biện Wave 2 (`P2`): (1) Giữ nguyên `toUnixTimestamp64Milli(updated_at)` ở cả `SELECT` và `WHERE` để đồng nhất miền `Int64` ms (tránh lệch sub-millisecond khi cột `updated_at` là `DateTime64(6/9)`); (2) Giữ `.limit(pageSize + 1)` ở `getAll()` để bảo toàn unit test `export.test.ts:320`; (3) Giữ `mergeData` in-place trên chunk hiện tại (bảo toàn unit test `export.test.ts:297`), chỉ sửa comment dòng `87-88` từ `"whole result set"` sang `"current chunk"`. | **REFINED** — Chỉ sửa comment dòng `87-88` trong `pure.ts` và ép `Number(cursor.updatedAtMs)` trong `model.ts`. |
| **`F5`** | Bước 4 Plan (Sai Layer / Design Pattern) | 🟠 **HIGH** | `data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts:2-42`<br>`data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts:118-124`<br>`logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts:4-14` | Mới làm 1/3 (thêm `params/index.ts`): `MERCHANT_TRANSACTION_COLUMNS` (nhãn UI `vi`/`en`) vẫn nằm ở `entity.ts`, hàm `validateMerchantTransactionFilters` vẫn nằm ở `params.ts` (sai chuẩn DTO thuần của `develop`), và `pure.ts` chưa import qua barrel `domain/queries` & `data-transfers/params`. Qua phản biện Wave 2 (`P1`): xóa hẳn khỏi `entity.ts` và `params.ts` (tuyệt đối không re-export ngược từ `logics/` để chặn Circular Dependency). | **OPEN** — Chờ User `APPROVE` để chuyển `COLUMNS` và `validate` về `pure.ts`. |
| **`F9`** | Bước 7 & 8 Plan (Integration Test) | 🟠 **MEDIUM** | `test/live/merchant-reconcile-transactions.clickhouse.test.ts:1-118`<br>`vitest.integration.config.ts:10` | File test vẫn nằm ở `test/live/` (bị `vitest.integration.config.ts` bỏ qua), thiếu `console.warn` khi skip (`L006`), và bài test `"205 rows"` chưa chạy phân trang 5 trang với `limit: 50`. Qua phản biện Wave 2 (`P4`): chuyển thẳng về `test/integration/merchant-reconcile-transactions.clickhouse.test.ts` (phẳng như các file integration test hiện có, giữ nguyên độ sâu import `../../`). | **OPEN** — Chờ User `APPROVE` để chuyển sang `test/integration/merchant-reconcile-transactions.clickhouse.test.ts` và cập nhật test 5 trang. |
| **`F6` & `F6-NEW`** | Bước 5 Plan (Infra Neutrality & Timeout) | 🟡 **MEDIUM** | `dbHandler/DbConnectorFactory.ts:108-110, 165-170`<br>`dbHandler/clickhouse.ts:5-8, 43` | Check `!config.host || !config.database || !config.username` chưa đưa lên `switch` của `DbConnectorFactory.ts:108`; `clickhouse.ts:8` tự đặt `DEFAULT_READ_TIMEOUT_MS = 300000` kèm comment nhắc `"dashboard API"`, lệch với `DbConnectorFactory.ts:170` (`20000`) và `.run.local.env.example:119` (`20000`). | **OPEN** — Chờ User `APPROVE` để đưa check lên `switch` và đồng bộ `DEFAULT_READ_TIMEOUT_MS = 20000`. |
| **`F4`** | Bước 3 & 8 Plan (`COLUMN_MAPPING`) | 🟡 **LOW** | `models/auto-reconcile/merchant-reconcile-result.model.ts:195-236` | Đã `throw` khi gặp field lạ nhưng chưa tách `COLUMN_MAPPING` ra cấp module và chưa có unit test `getColumnName("foo")` throw. | **OPEN** — Chờ User `APPROVE` để tách hằng `COLUMN_MAPPING` và thêm test. |
| **`F_LINT` & `F_TSC`** | Bước 8 Plan (Linter & TypeScript) | 🟡 **MEDIUM** | `merchant-reconcile-transactions.entity.ts:39`<br>`clickhouse.test.ts:10,84`<br>`merchant-reconcile-transactions.export.test.ts:105,381,437` | 2 lỗi ESLint (`comma-dangle`, `quotes` — `EV-20261102T134642-0154`) và 5 lỗi `tsc --noEmit` trong 2 file unit test của nhánh (`EV-20261102T134637-0c3b`). | **OPEN** — Chờ User `APPROVE` để sửa sạch 100% lỗi lint & type. |

### 1.2. Các Điểm Chờ Xác Nhận Từ User (`PENDING CONFIRMATION`)
- **`F7` (Tab `list` loại các dòng `reconcile_status IS NULL`):** Hiện `buildMerchantReconcileFilter` sinh `reconcile_status != 'DISCREPANCY'` (trong ClickHouse `NULL != 'DISCREPANCY'` trả về `0`/false nên loại luôn dòng `NULL`). Cần xác nhận hành vi API Go của `auto-reconcile-service`.
- **`F8` (Truy vấn `merchant_reconcile_results` không có `FINAL`):** View `v_fee_rows` có `FINAL`, còn bảng `merchant_reconcile_results` đang đọc trực tiếp không `FINAL`. Cần xác nhận `SHOW CREATE TABLE merchant_reconcile_results`.

### 1.3. Hạng Mục Đã Xử Lý Đúng Trong Commit `699a833` (`RESOLVED`)
- **`F10` (`defined/validation-message.ts:16-19`, `merchant-reconcile-transactions-export.params.ts:8`):** Đã thêm `ERR_TAB_INVALID` và gắn vào `@IsIn(["list", "discrepancy"], { message: "ERR_TAB_INVALID" })` kèm unit test i18n.
- **Bỏ ép kiểu trùng ở `start.ts:260`:** Đã đổi `poolSize: Number(SVC_ENV.get()[config.poolSizeEnvKey] || 10)` thành `poolSize: SVC_ENV.get()[config.poolSizeEnvKey]`.
- **Tách riêng 2 model theo đúng trách nhiệm đơn (`L012`):** `MerchantReconcileResultModel` (`239` dòng) chỉ đọc bảng `merchant_reconcile_results`, `FeeRowViewModel` (`78` dòng) chỉ đọc view `v_fee_rows`, `auto-reconcile.schema.ts` (`70` dòng) khai báo schema.

---

## Mục 1: Phản biện từng dòng update (git diff Line-by-Line Critique) & Từng Task trong 08_tasks
- Tổng diff `develop...HEAD`: **29 files changed, 1749 insertions(+), 2 deletions(-)**.
- Riêng commit `699a833`: **14 files changed, 227 insertions(+), 40 deletions(-)**.
- Số liệu `wc -l` vật lý trên đĩa:
  - `data-transfers/entities/auto-reconcile/merchant-reconcile-transactions.entity.ts`: `109` dòng
  - `data-transfers/params/auto-reconcile/merchant-reconcile-transactions-export.params.ts`: `124` dòng
  - `logics/export/auto-reconcile/merchant-reconcile-transactions-export.pure.ts`: `112` dòng
  - `logics/export/auto-reconcile/merchant-reconcile-transactions.export.ts`: `24` dòng
  - `models/auto-reconcile/merchant-reconcile-result.model.ts`: `239` dòng
  - `models/auto-reconcile/fee-row-view.model.ts`: `78` dòng
  - `models/auto-reconcile/auto-reconcile.schema.ts`: `70` dòng
  - `domain/handlers/auto-reconcile/GetAllMerchantReconcileTransactionsExportHandler.ts`: `35` dòng
  - `domain/handlers/auto-reconcile/GetFeeTotalsByGoopayCodesHandler.ts`: `20` dòng
  - `dbHandler/DbConnectorFactory.ts`: `287` dòng
  - `dbHandler/clickhouse.ts`: `69` dòng
  - `logics/processors/BaseExportProcessor.ts`: `391` dòng (chưa được cập nhật ở `699a833`!)
  - `start.ts`: `333` dòng

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật (EV-...)
- **`EV-20261102T134637-0c3b` (`npx tsc --noEmit`, `exit=2`):** Phát hiện `@clickhouse/client` và `@hypequery/clickhouse` chưa được cài trong `node_modules` (`TS2307`) và 5 lỗi TypeScript trong `test/unit/dbHandler/clickhouse.test.ts:84` + `test/unit/logics/merchant-reconcile-transactions.export.test.ts:112,317,333,381`.
- **`EV-20261102T134642-0154` (`npm run lint`, `exit=1`):** Phát hiện lỗi ESLint `comma-dangle` tại `merchant-reconcile-transactions.entity.ts:39:67` và `@typescript-eslint/quotes` tại `merchant-reconcile-transactions.export.test.ts:437:37`.
- **`EV-20261102T134647-ab1c` (`npx vitest run test/unit/dbHandler/clickhouse.test.ts test/unit/logics/merchant-reconcile-transactions.export.test.ts`, `exit=1`):** Thất bại với `Error: Cannot find package '@hypequery/clickhouse'` do `package-lock.json` và `node_modules` chưa được đồng bộ.
- **`EV-20261102T134907-5445` (`npx vitest run ... DbConnectorFactory.test.ts QueryHandlerFactory.test.ts ...`, `exit=1`):** Các bài unit test cũ cũng bị gãy dây chuyền khi import `DbConnectorFactory.ts` và `models/index.ts` do thiếu `@clickhouse/client` và `@hypequery/clickhouse` trong `node_modules`.

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity
1. **Watermark / Keyset Cursor Continuity:**
   - Trong `test/unit/logics/merchant-reconcile-transactions.export.test.ts:120-126`, hàm mock `afterCursor` dùng regex `/tuple\((\d+),\s*'([^']*)'\)/` để giả lập ClickHouse trong RAM. Đây chỉ là **unit test giả lập trên mảng JS**, hoàn toàn không chứng minh được câu SQL chạy trên ClickHouse thật!
   - Trên ClickHouse thật, việc bọc `toUnixTimestamp64Milli(updated_at)` ở vế trái (`merchant-reconcile-result.model.ts:75`) ép ClickHouse phải tính hàm trên toàn bộ các dòng thay vì dùng trực tiếp cột `updated_at` (`DateTime64(3)`).
2. **Batching & String-Limit Bug (`F1-D`):**
   - Trong `BaseExportProcessor.ts:169`: `const chunkSize = Number(config.chunkSize || SVC_ENV.get().CHUNK_SIZE || 5000);` truyền số vào `options.params.limit`, nhưng nếu bất kỳ caller nào gọi `GetAllMerchantReconcileTransactionsExportHandler.execute` với `limit` kiểu chuỗi (hoặc không truyền `options` trong `test/live/...:32`), việc thiếu `Number(limit)` ở Handler và Model tiềm ẩn lỗi nối chuỗi `limit + 1`.
3. **Timeout & Sub-Query Error Handling (`F2`):**
   - Khi xuất dữ liệu lớn qua nhiều chunk, nếu 1 chunk bất kỳ bị timeout ở sub-query `v_fee_rows`, việc thiếu `failOnSubQueryError: true` khiến chỉ riêng chunk đó bị mất toàn bộ phí (`dailyFee = null`, `monthlyFee = null`) trong khi các chunk trước và sau vẫn có phí — tạo ra file đối soát sai lệch ngầm cực kỳ nguy hiểm!

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)
- Không có bất kỳ khẳng định nào dựa trên suy đoán. Mọi lỗi đều đối chiếu trực tiếp với `git diff develop...HEAD`, `git show 699a833`, số dòng `wc -l` thực tế và 4 biên lai `EV-...` (`EV-20261102T134637-0c3b`, `EV-20261102T134642-0154`, `EV-20261102T134647-ab1c`, `EV-20261102T134907-5445`).
- Các điểm chưa đủ dữ kiện runtime (`F7` hành vi API Go đối với `reconcile_status IS NULL`, `F8` DDL `SHOW CREATE TABLE merchant_reconcile_results`) được phân loại rõ là `HYPOTHESIS / PENDING CONFIRMATION`.

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons Check)
- Đã áp dụng nghiêm ngặt:
  - `LS-GLOBAL-001-propose-only-gate`: Giữ nguyên trạng thái 100% Read-Only (Propose-Only) trên `/Users/trainguyen/Documents/work/centralized-export-service`, không sửa bất kỳ dòng code nào trước khi có lệnh `APPROVE`.
  - `L001`: Lập phương án kỹ thuật tối ưu duy nhất tại `09_tasks_solution_auto_recon_export.md` kèm Code Demo.
  - `L002`: Trình bày Bảng Tổng Hợp Issues ở đầu báo cáo.
  - `L003`: Phát hiện silent skip trong `test/live/merchant-reconcile-transactions.clickhouse.test.ts:15`.
  - `L012`: Kiểm tra ranh giới Single-Responsibility và Layering theo Design Pattern gốc trên `develop`.

## Mục 6: Kết quả QC & Nhận xét từ /laya_router (DG3) + Tổng hợp 8 Quality Gates (G1–G8)
- **DG0 (`06_laya_summary_dg0_20261102T135420.json`):** `PASS` (`C-REQ: ACCEPT`, `C-REALITY: ACCEPT`, `C-CONTRA: ACCEPT`, controls calibrated).
- **DG1 & DG2 Wave 1 (`06_laya_summary_dg1_20261102T135424.json`, `06_laya_summary_dg2_20261102T135424.json`):** `PASS`.
- **DG3 Wave 1 (`06_laya_summary_dg3_20261102T135441.json`):** `PASS` (`A-F2-SUBQUERY: ACCEPT`, `A-F9-LIVETEST: ACCEPT`, controls calibrated).
- **DG1 & DG2 Wave 2 — Phản biện lại Plan (`06_laya_summary_dg1_20261102T145939.json`, `06_laya_summary_dg2_20261102T145939.json`):** `PASS` (`S-FIX`, `S-IMPACT`, `S-PATTERN`, `D-EDGE`, `D-API`: `ACCEPT`; `S-RC`, `D-BEHAV`: `FLAG -> RESOLVED` kèm `REF:` tại `04_decisions_auto_recon_export.md`, controls calibrated).
- **DG3 Wave 2 — Hội đồng `/laya_router` nhận xét 4 quyết định tinh chỉnh Plan `P1–P4` (`06_laya_summary_dg3_20261102T150112.json`):** `PASS`:
  - `A-P4-INTEGRATION`: `ACCEPT` (`0.7224`) — Đặt file test phẳng tại `test/integration/merchant-reconcile-transactions.clickhouse.test.ts` khớp `vitest.integration.config.ts:10` và bảo toàn độ sâu import `../../`.
  - `A-P1-BARREL`: `FLAG -> RESOLVED` kèm `REF: data-transfers/params/index.ts:1-3` — Xác nhận không được re-export từ `logics/export` trong `params.ts`/`entity.ts` để chặn Circular Dependency.
  - `A-P2-UNITTEST`: `FLAG -> RESOLVED` kèm `REF: test/unit/logics/merchant-reconcile-transactions.export.test.ts:318-321` — Xác nhận giữ `.limit(pageSize + 1)` ở `getAll()` và giữ `toUnixTimestamp64Milli(updated_at)` ở `buildListQuery()` để bảo toàn miền `Int64` và 100% unit test hiện có.
  - `A-P3-HANDLER`: `FLAG -> RESOLVED` kèm `REF: domain/handlers/auto-reconcile/GetAllMerchantReconcileTransactionsExportHandler.ts:22-29` — Xác nhận giữ nguyên Handler (0 dòng thay đổi).
