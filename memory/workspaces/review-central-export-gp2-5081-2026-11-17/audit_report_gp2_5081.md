# Audit Report & Consolidated Validation — `feat/GP2-5081 -> develop` (`centralized-export-service`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Nhánh nguồn → đích:** `origin/feat/GP2-5081` (`2304b3b9a2fa80d23abdc9adf58e7634fbdc8ac6`) → `origin/develop` (`3eef80790b319e4d0f188e3d12cef9f0ac90b302`)
- **Trạng thái Source Code dự án:** `git status -s` rỗng (`0` dòng chỉnh sửa — tuân thủ tuyệt đối chế độ Read-Only Propose-Only trước lệnh `APPROVE`).

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng task (Line-by-Line Diff & Task Critique)

### 1.1. Kiểm toán `wc -l` & `git diff` từng file (14 files changed, `+921` insertions, `-1` deletion, tổng `1495` dòng)
1. `data-transfers/params/index.ts` (`104` dòng): Export thêm `PartnerCollectionTransExportParams`. Đúng chuẩn barrel, lưu ý đụng dòng cuối khi merge cùng nhánh `origin/feat/GP2-4633` (`F6`).
2. `data-transfers/params/trans-his/partner-collection-trans-export.params.ts` (`66` dòng, mới):
   - Dòng `18-25`: `ExceedDateRangeValidator` chỉ kiểm tra `dateDiff <= MAX_DAYS_EXPORT`, bỏ lọt `dateTo < dateFr` (`dateDiff < 0` trả về `true`) (`F4`).
   - Dòng `43-44, 52`: `merchantTransId`, `merchantEmail`, `payeeCode` thiếu `@MinLength(3)` khiến tìm kiếm regex 1 ký tự quét toàn bảng (`F2`).
   - Dòng `59`: `@IsString()` trên `sortType` chặn giá trị số `1 | -1`, mâu thuẫn với `pure.ts:61` (`F4`).
3. `domain/queries/trans-his/GetAllPartnerCollectionTransExportQuery.ts` (`14` dòng, mới) & `domain/queries/index.ts` (`71` dòng): Định nghĩa DTO Query gọn; tuy nhiên chưa có thuộc tính cache `resolvedBillIds` dẫn tới tra cứu lại `payment-bills` ở mọi chunk (`F2`).
4. `domain/QueryHandlerFactory.ts` (`165` dòng):
   - Dòng `155-156`: Đăng ký cứng `GetAllPartnerCollectionTransExportHandler` (Mongo) bên ngoài nhánh `if (getTransHisExportSource() === "postgres")` (`F1`).
5. `domain/handlers/trans-his/GetAllPartnerCollectionTransExportHandler.ts` (`94` dòng, mới):
   - Dòng `7-16`: `SELECT_FIELDS` thiếu `receiver: 1` làm mất nguồn fallback thông tin bên thụ hưởng (`F1`).
   - Dòng `19`: `Array<string | number>` vi phạm ESLint `@typescript-eslint/array-type` (`F5`), trùng lặp logic với `pure.ts:48-51` và thiếu `Number.isSafeInteger` (`F4`).
   - Dòng `38-49`: `Bill.find({ merchantTransId: { $regex: ..., $options: "i" } })` không có `.limit()`, bị gọi lặp lại ở mỗi chunk `5000` dòng, và nếu `merchantTransId = "   "` thì `.trim()` sinh `$regex: ""` quét toàn bộ bảng `payment-bills` (`F2`, `F4`).
   - Dòng `51-55, 90`: Phân trang bằng `.skip(cursor)` không có tie-breaker `_id` và tốn $O(N^2)$ khi xuất tới `2_000_000` dòng (`F3`).
6. `logics/export/trans-his/partner-collection-trans-export.pure.ts` (`132` dòng, mới) & `logics/export/index.ts` (`60` dòng):
   - Dòng `19`: `isEmptyValue` không `.trim()`, làm `billId = "   "` bị ép kiểu `Number("   ") === 0` lọc sai sang `billId = 0` (`F4`).
   - Dòng `23-27, 36, 80-82`: `quote-props` ESLint lỗi ở dòng `24, 26` (`F5`); ép cứng `serviceCode: "COLLECTION_DISBURSEMENT"` gây mâu thuẫn khi truyền `params.serviceCode` khác và làm rơi các đơn thu hộ không có `serviceCode` như RADA so với API `GP2-5069` (`F1`).
   - Dòng `58-65`: `buildSort` thiếu tie-breaker `_id` (`F3`).
   - Dòng `68-71`: `safeText` thiếu ký tự `\n` trong `/^[=+\-@\t\r\n]/` (`F5`).
   - Dòng `114-128`: `transformRow` không fallback `receiver.merchantCode` / `receiver.fullName` (`F1`).
7. `logics/export/trans-his/partner-collection-trans.export.ts` (`28` dòng, mới) & `logics/index.ts` (`130` dòng):
   - `getRequiredModelName()` trả cứng `["TransHisModel", "PaymentBillModel"]` thay vì dùng `transHisModelNames()` (`F1`).
8. 4 file Test (`QueryHandlerFactory.test.ts` `55` dòng, `GetAllPartnerCollectionTransExportHandler.test.ts` `158` dòng, `partner-collection-trans.pure.test.ts` `132` dòng, `partner-collection-trans-export.integration.test.ts` `286` dòng):
   - `partner-collection-trans.pure.test.ts:8-10` dùng `vi.mock("../../../utils/validator")` làm mất hiệu lực kiểm thử `validateParams` thật (`F6`).

### 1.2. Đối chiếu tiến độ từng Task (`08_tasks_gp2_5081.md`)
- `T-01` (Git Branch, Merge-Base & Architecture): Đã xác minh `merge-base = 3eef807` (`origin/develop`), đối chiếu chéo với `admin-portal-gateway` (`GP2-5069`) và `core-trans-proxy-history-service-v2` (`TransHisCollectionOpsLogic`).
- `T-02` (Line-by-Line Review 14 files): Đã hoàn tất phân tích chi tiết, trích xuất 6 phát hiện `F1`–`F6` (`1 P0`, `3 P1`, `2 P2`).
- `T-03` (Test, Typecheck & Lint Verification): Đã thực thi kiểm chứng qua `evidence_run.py` với biên lai `EV-20261117T112350-84e6`, `EV-20261117T112353-763a`, `EV-20261117T112358-fc32` và `EV-20261117T112415-280e` (FAIL 3 lỗi ESLint).
- `T-04` (Decision Gate & Propose-Only Report): Đã kiểm định qua `laya_gate.py` (`DG0`, `DG1`, `DG3`) và xuất báo cáo vật lý.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

### 2.1. Biên lai thực thi vật lý (`06_evidence/EV-*.json`)
- **Unit Tests (`EV-20261117T112350-84e6`):** `npx vitest run test/unit/pure/partner-collection-trans.pure.test.ts test/unit/domain/handlers/GetAllPartnerCollectionTransExportHandler.test.ts test/unit/domain/QueryHandlerFactory.test.ts` → PASSED `24` unit tests (`exit=0`, biên lai `EV-20261117T112350-84e6`).
- **Integration Test (`EV-20261117T112353-763a`):** `npx vitest run --config vitest.integration.config.ts test/integration/partner-collection-trans-export.integration.test.ts` → chạy thật integration test PASSED `9` tests (`exit=0`, biên lai `EV-20261117T112353-763a`) trên `MongoMemoryServer` + `ExcelJS`.
- **TypeScript Build Typecheck (`EV-20261117T112358-fc32`):** `npx tsc -p tsconfig.build.json --noEmit` → PASSED không có lỗi kiểu (`exit=0`, biên lai `EV-20261117T112358-fc32`).
- **ESLint Check (`EV-20261117T112415-280e`):** `npx eslint <10 production files>` → FAILED với `3` errors (`exit=1`, biên lai `EV-20261117T112415-280e`: `@typescript-eslint/array-type` tại `Handler.ts:19` và `quote-props` tại `pure.ts:24,26`).

### 2.2. Giới hạn & Điểm Chưa Kiểm Chứng Cần Phối Hợp (Limitations & Not Verified)
- **Không suy diễn từ `MongoMemoryServer`:** Integration test hiện chạy trên `MongoMemoryServer` với vỏn vẹn **6 bản ghi** `trans-his` và **3 bản ghi** `payment-bills` (`integration.test.ts:81-125`), hoàn toàn **KHÔNG thể hiện được**:
  1. Giá trị biến môi trường runtime `TRANS_HIS_EXPORT_SOURCE` (`mongo` hay `postgres`) đang cấu hình thực tế trên môi trường `develop`, `staging` và `prod`.
  2. Kế hoạch thực thi (`explain("executionStats")`) của MongoDB Production đối với bộ lọc `{ transType: "PAY", "info.desc": "AUGMENT_PROVISIONAL_REVENUE", serviceCode: "COLLECTION_DISBURSEMENT", createdAt: { $gte, $lte } }` trên collection `trans-his` hàng chục triệu bản ghi.
- **Đề xuất phối hợp cùng anh để trace runtime:** Trước khi chốt phạm vi sửa `F1` và `F3`, em cần phối hợp cùng anh xác nhận: (a) `TRANS_HIS_EXPORT_SOURCE` trên môi trường đích hiện là `mongo` hay `postgres`, và (b) danh sách index thực tế của collection `trans-his` & `payment-bills`.

---

## Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity & Security

1. **Concurrency & Multi-Worker (`ExportWorker.ts` & `PartnerCollectionTransExport`):**
   - `ExportWorker` (`workers/ExportWorker.ts:206`) khởi tạo `new ProcessorClass(...)` riêng biệt cho từng job nên biến instance `private rowIndex: number = 0` trong `partner-collection-trans.export.ts:7` không bị race condition giữa các worker chạy song song (`EXPORT_WORKER_CONCURRENCY > 1`).
   - Tuy nhiên, khi dùng 1 instance `query` qua nhiều chunk (`BaseExportProcessor.ts:208-274`), việc không cache `resolvedBillIds` khiến mỗi chunk đều bắn lại query regex vào `PaymentBillModel`, gây áp lực lên connection pool (`PAYMENT_BILL_MONGO_POOLSIZE = 5` mặc định ở `start.ts:256`) khi multi-worker hoạt động đồng thời (`F2`).
2. **Timeout / Heartbeat / Queue AckWait (`ExportWorker.ts` & `BaseExportProcessor.ts`):**
   - `ExportWorker.ts:64` dùng `redis.dequeue("export:queue")` dạng fire-and-forget không có cơ chế gia hạn lease heartbeat/ackwait tự động nếu pod bị OOM giữa chừng, và không có debouncer cho các request trùng tham số; tiến độ được ghi qua `redis.setJobProgress` sau mỗi chunk (`BaseExportProcessor.ts:270`). Nếu 1 chunk bị treo do quét toàn bảng `payment-bills` (`merchantTransId: "   "` hoặc regex 1 ký tự — `F2`, `F4`) hoặc `.skip(1_500_000)` (`F3`), worker slot sẽ bị kẹt hàng chục phút gây Head-of-Line blocking cho toàn bộ hàng đợi `export:queue`.
3. **Data Continuity (Watermark / Cursor qua các trang dữ liệu đang biến động):**
   - Việc dùng offset `cursor` (`skip(cursor)`) kết hợp `sort: { createdAt: -1 }` không có `_id` tie-breaker (`F3`) làm gãy tính liên tục dữ liệu (Watermark / Cursor Data Continuity): khi có giao dịch thu hộ mới ghi vào `trans-his` trong lúc job đang xuất từ chunk 1 sang chunk 2, cửa sổ `skip` bị trượt làm nhân bản dòng cũ ở biên trang; và khi nhiều bản ghi trùng `status`/`value`/`createdAt`, MongoDB có thể trả trùng hoặc bỏ sót bản ghi. Giải pháp đề xuất chuyển sang Keyset Cursor `(sortField, _id)` giải quyết triệt để lỗi này.
4. **Security Review (`/security-agent`):**
   - Không có secret/token/PII cứng trong diff.
   - Cần vá lỗ hổng `\n` trong `safeText` chống CSV/Excel Formula Injection (`F5`) và đảm bảo ở tầng API Gateway rằng `exportType = "PartnerCollectionTransExport"` chỉ mở cho role Vận hành (Ops/CMS), không mở cho Merchant Portal (do query không khóa cứng theo `currentUser.merchantId`).

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- Toàn bộ nhận định trong báo cáo đều được kiểm chứng trung thực (honesty check, zero speculation) bằng dòng lệnh thực tế và `file:line`:
  - Minh chứng chống suy diễn trong quá trình điều tra S2: Khi thấy `TransHisModel` (`models/trans-his/trans-his.model.ts:22-65`) không khai báo `serviceCode` trong Mongoose Schema, thay vì suy diễn rằng Mongoose 8 sẽ strip `serviceCode` do `strictQuery`, em đã kiểm tra trực tiếp cấu hình `mongoose 8.20.0` (`strictQuery` mặc định `false`) và đối chiếu integration test (`EV-20261117T112353-763a`) nên đã loại bỏ giả thuyết sai đó, chỉ giữ lại các phát hiện `VERIFIED` có bằng chứng vật lý.
- Tuyệt đối không có bất kỳ chỉnh sửa lén nào vào source code (`git status -s` sạch).

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Các Global Lessons đã áp dụng nghiêm ngặt trong phiên:**
  - `L074` (Chống lệnh DDL động và chống quét toàn bảng trên DB lớn): Phát hiện trực diện lỗi thiếu `@MinLength(3)`, `$regex: ""` khi truyền khoảng trắng, lặp `Bill.find` mỗi chunk (`F2`, `F4`) và `.skip()` $O(N^2)$ tới `2_000_000` dòng (`F3`).
  - `L078` (Bảo toàn dữ liệu gốc, ép kiểu BSON & chuẩn hóa múi giờ): Phát hiện lỗi ép kiểu `Number("   ") === 0` trên `billId` (`F4`) và kiểm chứng múi giờ `utcOffset(7)` trên `transformRow`.
  - `L041` (Tách lệnh đơn sạch khi gọi công cụ kiểm định): Tách riêng từng lệnh `evidence_run.py` và `laya_gate.py` độc lập.
  - Core Tripwires `G1` (No Overwrite memory), `G2` (Audit Propose-Only), `G3` (Decision Gate `DG0`/`DG1`/`DG3`), `G4` (Thực thi test/lint qua `evidence_run.py`).
- **Kiểm tra vi phạm Lesson:** Không mắc vi phạm nào trong phiên.

---

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`laya_gate.py` — `DG0`, `DG1`, `DG3`)

- **Engine sử dụng:** VegaML Primary (`http://127.0.0.1:8222/v1/predict`, `mode=auto`, `0 LLM token`).
- **Đối chiếu đĩa (`REF:` Verification):** Toàn bộ `16` claims thực tế qua 3 cổng (`DG0`, `DG1`/`DG2`, `DG3`) đều khớp dòng code vật lý trên đĩa (`evidence_error = ""`, không có `BLOCK`).
- **Tóm tắt kết quả các cổng Decision Gate (`06_laya_summary_*.json`) & Công bố trung thực theo Rule #10:**

```json
{
  "dg0": {
    "summary_file": "06_laya_summary_dg0_20261117T113040.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.8271 - 0.8747",
    "claims_decision": "UNVALIDATED (3 claims: C-REQ-1, C-REALITY-1, C-CONTRA-1)"
  },
  "dg1_dg2": {
    "summary_file": "06_laya_summary_dg1_20261117T113546.json",
    "dg2_summary_file": "06_laya_summary_dg2_20261117T113546.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.6835 - 0.9257",
    "claims_decision": "UNVALIDATED (8 claims: S-RC-1..2, S-FIX-1, S-IMPACT-1, S-PATTERN-1, D-BEHAV-1, D-EDGE-1, D-API-1)"
  },
  "dg3": {
    "summary_file": "06_laya_summary_dg3_20261117T113544.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.7447 - 0.7880",
    "claims_decision": "UNVALIDATED (5 claims: A-F1..A-F5)"
  }
}
```

- **Nhận xét Ground-Truth (Rule #10):** Do vòng control của VegaML `:8222` chưa tách biệt vượt ngưỡng `DECISIVE_CONF` (`trusted_rounds = []` → `UNVALIDATED`), em báo cáo rõ trạng thái `UNVALIDATED` của engine và xác lập kết luận cuối cùng trực tiếp từ source code `file:line` cùng 4 biên lai thực thi `EV-20261117T112350-84e6`, `EV-20261117T112353-763a`, `EV-20261117T112358-fc32` và `EV-20261117T112415-280e` (FAIL 3 lỗi ESLint).
