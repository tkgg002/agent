# Audit Report & Final Merge Verification (Lượt 3) — `feat/GP2-5081 -> develop` (`92ebb72`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_v3`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Nhánh nguồn → đích:** `origin/feat/GP2-5081` (`92ebb728c008808618b56e93c316c803416fcc02`) → `origin/develop` (`3eef80790b319e4d0f188e3d12cef9f0ac90b302`)
- **Trạng thái Merge Tree:** Sạch xung đột hoàn toàn với `origin/develop` (`git merge-tree --write-tree` mã thoát `0`, tree `3c2af274696d6bca2c1a1a29e114758a38544cb0`).
- **Trạng thái Source Code dự án:** `git status -s` rỗng (`0` dòng chỉnh sửa — tuân thủ tuyệt đối Propose-Only).

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng task (Line-by-Line Diff & Task Critique)

### 1.1. Phản biện chi tiết `19` file thay đổi trên nhánh `92ebb72` (`wc -l` tổng `3055` dòng, thêm `1262` dòng và xóa `5` dòng)
1. `domain/queries/payment-bill/GetPaymentBillExportAuxiliaryQuery.ts` (`18` dòng, mới) & `domain/handlers/payment-bill/GetPaymentBillExportAuxiliaryHandler.ts` (`45` dòng, mới):
   - Đã gỡ bỏ hoàn toàn 4 file vi phạm Clean Architecture ở commit trước (`partner-collection-trans-bills.ts`, `GetAllPartnerCollectionTransExportQuery.ts`, `GetAllPartnerCollectionTransExportHandler.ts`, `GetAllPartnerCollectionTransExportPgHandler.ts`).
   - `GetPaymentBillExportAuxiliaryQuery` khai báo `readonly merchantTransId`, `readonly billIds`, `readonly limit` bất biến tuyệt đối (`DP3`).
   - `GetPaymentBillExportAuxiliaryHandler` đặt đúng Bounded Context `payment-bill`, không còn bất kỳ import ngược nào lên tầng `logics/` (`DP1`), escape regex bằng `_.escapeRegExp` và giới hạn `DEFAULT_BILL_LOOKUP_LIMIT = 1000` kèm `logger.warn` khi chạm trần.
2. `domain/handlers/trans-his/GetAllTransHisExportHandler.ts` (`60` dòng) & `domain/QueryHandlerFactory.ts` (`164` dòng):
   - Dòng `22-25` thêm `_id` tie-breaker (`sortWithTieBreaker`) bằng cách tạo object mới (không mutate `query.sort`), đóng triệt để rủi ro trượt hoặc lặp dòng khi `.skip(cursor)` cho cả 4 màn hình `trans-his` trên Mongo.
   - `QueryHandlerFactory.ts:136,156-162` tái sử dụng trực tiếp `GetAllTransHisExportQuery` cho cả hai nguồn `mongo` và `postgres` (`DP4`).
3. `logics/export/trans-his/partner-collection-trans.export.ts` (`31` dòng, mới) & `partner-collection-trans-export.pure.ts` (`213` dòng, mới):
   - `partner-collection-trans.export.ts:15-23` triển khai `async getConfig()` truyền `queryExecutor` và `transformRow` truyền `++this.rowIndex` đồng bộ chuẩn Template Method của `BaseExportProcessor` (`DP2`, `DP5`).
   - `partner-collection-trans-export.pure.ts:51-55` bổ sung `isDelete: { $ne: true }` vào `baseFilter()` (`N1`).
   - `partner-collection-trans-export.pure.ts:121-152,177` tra cứu `merchantTransId` đúng một lần trước vòng lặp chunk và cấu hình `subQueryClass: GetPaymentBillExportAuxiliaryQuery`, `buildSubQueryParams`, `mergeData`, `chunkSize: 500`.
   - Còn 2 điểm nhỏ mức `P2` không chặn merge tại dòng `194` (`err: ${error.code}` khi `error` thiếu `code`) và dòng `138-139` (`buildSubQueryParams` chưa dùng `isEmptyValue` + `Number.isSafeInteger` đồng bộ với `billIdVariants`).
4. `models/trans-his-pg/trans-his-pg.query.ts` (`433` dòng):
   - Dòng `98-99,119-120`: Ánh xạ `cdFlow` (`eq`) và `bizType` (`$nin`).
   - Dòng `126-127,134,138`: Ánh xạ `info.masterMerchantInfo.email`, `info.merchantInfo.code` (`$regex` & `$exists`), và `info.billId` (`jsonMixedInSql` khớp cả số EJSON lẫn chuỗi).
   - Dòng `240,377-384`: Ánh xạ `isDelete` sang `core.is_delete IS DISTINCT FROM TRUE`, mở khóa hoàn toàn Partial Index `idx_th_cd_flow_created_at_id` (`N1`).
5. Nhóm Barrel, DTO và Test (`12` file còn lại: `params/index.ts` `104` dòng, `partner-collection-trans-export.params.ts` `69` dòng, `domain/queries/index.ts` `71` dòng, `logics/export/index.ts` `60` dòng, `logics/index.ts` `130` dòng, và `7` file kiểm thử):
   - Đấu nối trọn vẹn từ `ExportLogic.processExport` → `ExportWorker.processorMap` → `PartnerCollectionTransExport`, dùng `validateParams` thật không gắn mock.

### 1.2. Đối chiếu tiến độ từng Task (`08_tasks_gp2_5081_v3.md`)
- `T3-01`: Đã rà soát toàn bộ `19` file qua đủ 4 lăng kính (`DP1`–`DP5`, `N1`–`N3`, Contract & Regression, Security) và hoàn thành cổng kiểm định đầu vào.
- `T3-02`: Đã thực thi kiểm tra kiểu TypeScript, ESLint, các bộ unit tests và integration tests lấy biên lai vật lý.
- `T3-03`: Đã tổng hợp `09_tasks_solution_gp2_5081_v3.md` cùng báo cáo hợp nhất này và chạy kiểm định quy trình.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

### 2.1. Bằng chứng thực thi vật lý (`06_evidence/EV-*.json` trên commit `92ebb72`)
- **Targeted Unit Tests (`EV-20261118T093439-fabc`):** PASSED `97` unit tests trên `5` bộ test trọng tâm (`exit=0`, biên lai `EV-20261118T093439-fabc`).
- **Maintained Repo Unit Suites (`EV-20261118T094123-f709`):** PASSED `154` unit tests trên `12` bộ test đang hoạt động của kho mã (`exit=0`, biên lai `EV-20261118T094123-f709`).
- **PartnerCollectionTransExport Integration Suite (`EV-20261118T093454-3d78`):** chạy thật integration test PASSED `10` tests với MongoDB Memory Server + ExcelJS SAGA (`exit=0`, biên lai `EV-20261118T093454-3d78`).
- **Multi-Suite Integration Check (`EV-20261118T094128-9504`):** chạy thật integration test PASSED `13` tests trên luồng thu hộ chi hộ (`exit=0`, biên lai `EV-20261118T094128-9504`).
- **TypeScript Build Check (`EV-20261118T093919-664a`):** PASSED kiểm tra kiểu toàn dự án không phát sinh lỗi (`exit=0`, biên lai `EV-20261118T093919-664a`).
- **ESLint Production Files (`EV-20261118T093924-cab5`):** PASSED kiểm tra quy chuẩn mã nguồn trên toàn bộ 12 file production thay đổi (`exit=0`, biên lai `EV-20261118T093924-cab5`).

### 2.2. Giới hạn & Điểm Chưa Kiểm Chứng Cần Phối Hợp (Limitations & Not Verified)
- Bộ `test/integration/trans-his-pg.integration.test.ts` (`14` ca kiểm thử kết nối PostgreSQL thật) tự động skip trên máy local do biến môi trường `TRANS_HIS_PG_IT_URL` chưa được cấu hình; phần sinh câu lệnh SQL và bind tham số đã được kiểm chứng qua `47` ca trong `trans-his-pg.query.test.ts` và `13` ca trong `trans-his-pg.handlers.test.ts`.
- Khi triển khai lên môi trường `develop`, `testing`, `prod` với `TRANS_HIS_EXPORT_SOURCE=postgres`, bắt buộc áp dụng trước migration `011_transaction_history_classification.sql` (`GP2-5154`) và bật `CLASSIFICATION_COLUMNS=true` trên `core-trans-proxy-history-service-v2` (`N2`). Nếu anh cần kiểm chứng kết nối PostgreSQL thật qua `TRANS_HIS_PG_IT_URL`, em sẵn sàng phối hợp ngay.
- Một số file unit test cũ có sẵn từ lịch sử `origin/develop` (`1ea8d1f`, như `GetAllTransHisExportHandler.test.ts`, `BaseExportProcessor.edge-cases.test.ts`, `FileGenerator.test.ts`) bị lệch đường dẫn import từ trước nhánh `feat/GP2-5081`; nhánh `feat/GP2-5081` không gây ra bất kỳ lỗi hồi quy mới nào.

---

## Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity & Security

1. **Concurrency & Multi-Worker:** `GetPaymentBillExportAuxiliaryQuery` và `GetAllTransHisExportQuery` hoàn toàn bất biến (`readonly` fields, `sortWithTieBreaker` tạo bản sao mới thay vì ghi đè `query.sort`), kết hợp `private rowIndex` khởi tạo riêng theo từng instance `PartnerCollectionTransExport` trên mỗi job (`ExportWorker.ts:206`), đảm bảo cô lập trạng thái an toàn tuyệt đối khi tăng `EXPORT_WORKER_CONCURRENCY > 1` (không chia sẻ biến tĩnh khả biến, không cần debouncer phụ).
2. **Batching & Timeout / Heartbeat / AckWait:** Hàng đợi `export:queue` của dự án hiện chưa có cơ chế gia hạn lease heartbeat hoặc ackwait theo từng chunk; thiết kế mới giảm thiểu rủi ro treo kết nối bằng cách (a) tra cứu `merchantTransId` đúng 1 lần trước vòng lặp chunk (`limit = 1000`), (b) đặt `chunkSize: 500` giới hạn mỗi truy vấn `$in` phụ tối đa `<= 500` mã đơn hàng, (c) áp dụng `QUERY_TIMEOUT_MS = 300_000` ở `TransHisPgModel`, và (d) cô lập ngoại lệ sub-query tại `BaseExportProcessor.ts:236-238`.
3. **Data Continuity (Watermark / Cursor):** Nguồn PostgreSQL duy trì Keyset Cursor tất định `(sortExpr, core.id)` độ phức tạp $O(N)$ (`trans-his-pg.model.ts:83-98`), còn nguồn MongoDB được bổ sung `_id` tie-breaker (`GetAllTransHisExportHandler.ts:22-25`) bảo toàn tính liên tục dữ liệu (Watermark & Cursor Continuity) giữa các trang, không sót và không nhân bản dòng khi nhiều giao dịch trùng `createdAt`.
4. **Security Review (`/security-agent`):** Không chứa secret, token hay PII cứng; toàn bộ đầu vào regex được escape bằng `_.escapeRegExp` kèm `@MinLength(3)` và `approvedRegex`; toàn bộ truy vấn SQL dùng tham số `$n`; toàn bộ 12 cột văn bản xuất ra Excel đi qua `safeText` chống Formula Injection (`= + - @ \t \r \n`).

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- Mọi nhận định trong báo cáo này đều đối chiếu trực tiếp tới từng dòng mã nguồn của commit `92ebb728c008808618b56e93c316c803416fcc02` và biên lai kiểm thử thực tế ngày `2026-11-18`. Em công bố minh bạch việc 14 ca PostgreSQL trực tiếp trong `trans-his-pg.integration.test.ts` đang skip do thiếu `TRANS_HIS_PG_IT_URL` trên máy local (tuân thủ nghiêm `L043`, không suy diễn thành đã chạy PostgreSQL thật) và báo cáo rõ việc chuyển sang engine dự phòng Laya `:8200` khi endpoint `/v1/predict` của VegaML `:8222` bị treo quá `120s`.

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Các bài học đã áp dụng nghiêm ngặt trong phiên:**
  - `L002` (*Mandatory Design Pattern, Layer Dependency & Template Hook Audit*): Kiểm toán đầu tiên và xuyên suốt cả 5 điểm `DP1`–`DP5`, xác nhận chiều phụ thuộc `domain/ -> logics/` đã về `0` và tái sử dụng đúng `BaseExportProcessor` hooks.
  - `L076` (*Partial Index Alignment*): Xác nhận mệnh đề `isDelete: { $ne: true }` biên dịch thành `core.is_delete IS DISTINCT FROM TRUE` khớp Partial Index `idx_th_cd_flow_created_at_id`.
  - `L043` (*No Silent Live-DB Test Skip*) & `L062` (*Zero-Mockup Evidence*): Công bố minh bạch trạng thái skip của `TRANS_HIS_PG_IT_URL` và đối chiếu các test lỗi cũ trên `origin/develop`.
  - `L049` (*Propose-Only Discipline*), `L064` (*End-to-End Callchain Tracing*), `L066` (*Direct-Answer Summary First*), `G1`–`G4`: Không ghi đè memory, không chạm vào mã nguồn dự án khi chưa có lệnh `APPROVE`.
- **Vi phạm bài học trong lượt này:** `0` vi phạm.

---

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`laya_gate.py` — `DG0`, `DG1`, `DG3`)

- **Engine sử dụng:** Laya Fallback (`http://127.0.0.1:8200/v1/predict`, theo Rule #10 sau khi VegaML `:8222/v1/predict` timeout `>120s`; `0 LLM token`).
- **Đối chiếu đĩa (`REF:` Verification):** Toàn bộ `14` claims qua các cổng (`DG0`, `DG1`, `DG2`, `DG3`) đều khớp dòng code vật lý trên đĩa (`evidence_error = ""`, không có `BLOCK`). Ở `DG1`, vòng `r3` đạt chuẩn calibration (`trusted_rounds = ["r3"]`), chấp nhận trực tiếp `6` claims (`ACCEPT`) và gắn cờ `S-IMPACT-1` (`FLAG` ở mức `0.32`), đã được giải trình tường minh kèm `REF:` vào `04_decisions_review-central-export-gp2-5081-2026-11-17.md`.

```json
{
  "dg0_v3": {
    "summary_file": "06_laya_summary_dg0_20261118T095915.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.6702 - 0.8213",
    "claims_decision": "UNVALIDATED (3 claims: C-REQ-1, C-REALITY-1, C-CONTRA-1)"
  },
  "dg1_dg2_v3": {
    "summary_file": "06_laya_summary_dg1_20261118T100114.json",
    "dg2_summary_file": "06_laya_summary_dg2_20261118T100114.json",
    "gate": "PASS",
    "trusted_rounds": ["r3"],
    "claims_decision": "6 ACCEPT (S-RC-1, S-FIX-1, S-PATTERN-1, D-BEHAV-1, D-EDGE-1, D-API-1) + 1 FLAG resolved with REF (S-IMPACT-1)"
  },
  "dg3_v3": {
    "summary_file": "06_laya_summary_dg3_20261118T100210.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "claims_decision": "UNVALIDATED (4 claims: A-1..A-4)"
  }
}
```

- **Nhận xét Ground-Truth (Rule #10):** Với `DG1` và `DG2` đã vượt qua control calibration ở vòng `r3` (`6` claims `ACCEPT` + `1` claim `FLAG` đã `resolve` bằng `REF:` thật), còn `DG0` và `DG3` ở trạng thái `UNVALIDATED` (`trusted_rounds = []`), em công bố rõ kết quả calibration và xác lập kết luận cuối cùng trực tiếp từ mã nguồn `92ebb72` cùng chuỗi biên lai PASSED `EV-20261118T093439-fabc`, `EV-20261118T094123-f709`, `EV-20261118T093454-3d78`, `EV-20261118T094128-9504`, `EV-20261118T093919-664a` và `EV-20261118T093924-cab5`.
