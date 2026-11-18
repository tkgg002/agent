# Audit Report & Consolidated Validation (Pass 2) — `feat/GP2-5081 -> develop` (`7f4e9a7`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_v2`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Nhánh nguồn → đích:** `origin/feat/GP2-5081` (`7f4e9a7e149dabd75674f9fb3737c81bbfa21794`) → `origin/develop` (`3eef80790b319e4d0f188e3d12cef9f0ac90b302`)
- **Trạng thái Source Code dự án:** `git status -s` rỗng (`0` dòng chỉnh sửa — giữ nguyên chế độ Read-Only Propose-Only).

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng task (Line-by-Line Diff & Task Critique)

### 1.1. Kiểm toán `wc -l` & `git diff` trên 20 files thay đổi (`+1420` insertions, `-4` deletions, tổng `3160` dòng)
1. `data-transfers/params/trans-his/partner-collection-trans-export.params.ts` (`69` dòng): Đã thêm `@MinLength(3)` cho `merchantTransId`, `merchantEmail`, `payeeCode`, kiểm tra `dateDiff >= 0 && dateDiff <= MAX_DAYS_EXPORT`, và mở rộng `sortType?: string | number`. Đóng hoàn toàn `F2` (DTO) & `F4`.
2. `domain/handlers/trans-his/partner-collection-trans-bills.ts` (`66` dòng, mới): Tách dùng chung `applyBillFilter` (`BILL_LOOKUP_LIMIT = 1000`, cache `query.resolvedBillIds` qua các chunk, log cảnh báo khi chạm ngưỡng `1000`) và `attachBillData` (`merchantTransId` + `embedData.transactionType`). Đóng hoàn toàn `F2` (Handler).
3. `domain/handlers/trans-his/GetAllPartnerCollectionTransExportHandler.ts` (`62` dòng) & `GetAllPartnerCollectionTransExportPgHandler.ts` (`34` dòng, mới):
   - Bổ sung đủ `SELECT_FIELDS` (`receiver`, `error`, `updatedAt`, `noted`); thêm `_id` tie-breaker trên Mongo (`sortWithTieBreaker`) và dùng Keyset Cursor trên Postgres; đăng ký phân nhánh đầy đủ trong `domain/QueryHandlerFactory.ts` (`166` dòng) và `partner-collection-trans.export.ts` (`28` dòng). Đóng `F1` (Postgres wiring & columns) và `F3`.
4. `logics/export/trans-his/partner-collection-trans-export.pure.ts` (`159` dòng):
   - Đã sửa `isEmptyValue` có `.trim()`, `billIdVariants` có `Number.isSafeInteger`, `safeText` có `\n`, đồng bộ 15 cột template và fallback `receiver`.
   - **Còn tồn đọng `N1` & `N3.1`:** `baseFilter()` (dòng `35-38`) mới lọc `cdFlow` và `bizType` mà thiếu `isDelete: { $ne: true }`; `transformRow` (dòng `140`) in `"err: undefined"` khi `item.error` tồn tại nhưng không có `code`.
5. `models/trans-his-pg/trans-his-pg.query.ts` (`432` dòng):
   - Đã bổ sung biên dịch tham số hóa an toàn (`$n`) cho `cdFlow` (`eq`), `bizType` (`$nin` giữ `IS NULL`), `info.masterMerchantInfo.email` (`$regex`), `info.merchantInfo.code` (`$regex` + `$exists`), và `info.billId` (`jsonMixedInSql` khớp cả Extended JSON number, JSON number và string).
   - **Còn tồn đọng `N1`:** `fieldSql` (dòng `229-245`) chưa ánh xạ `key === "isDelete"` sang `booleanSql(key, "core.is_delete", spec)` để khớp mệnh đề `WHERE cd_flow IS NOT NULL AND is_delete IS DISTINCT FROM TRUE` của partial index `idx_th_cd_flow_created_at_id`.
6. 7 file Test (`QueryHandlerFactory.test.ts` `55` dòng, `GetAllPartnerCollectionTransExportHandler.test.ts` `191` dòng, `partner-collection-trans.pure.test.ts` `207` dòng, `trans-his-pg.handlers.test.ts` `216` dòng, `trans-his-pg.query.test.ts` `483` dòng, `partner-collection-trans-export.integration.test.ts` `298` dòng, `trans-his-pg.integration.test.ts` `313` dòng): Đã gỡ bỏ `vi.mock("../../../utils/validator")`, test trực tiếp validator thật (đóng `F6`).

### 1.2. Đối chiếu tiến độ từng Task (`08_tasks_gp2_5081_v2.md`)
- `T2-01`: Đã kiểm tra chéo toàn bộ 4 commit mới (`51c8898`, `0928958`, `28460e0`, `7f4e9a7`) với migration `011` và `TransactionHistoryRepository` bên `core-trans-proxy-history-service-v2`.
- `T2-02`: Đã thực thi kiểm chứng qua `evidence_run.py` với biên lai `EV-20261117T134840-5a70`, `EV-20261117T134845-f998`, `EV-20261117T134849-c7fe` và `EV-20261117T134853-2990`.
- `T2-03`: Đã kiểm định qua `laya_gate.py` (`DG1`, `DG2`, `DG3`) và lập báo cáo Vòng 2.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

### 2.1. Biên lai thực thi vật lý (`06_evidence/EV-*.json` trên commit `7f4e9a7`)
- **Unit Tests (`EV-20261117T134840-5a70`):** `npx vitest run <5 unit test files>` → PASSED `100` tests (`exit=0`, biên lai `EV-20261117T134840-5a70`).
- **Integration Tests (`EV-20261117T134845-f998`):** `npx vitest run --config vitest.integration.config.ts test/integration/partner-collection-trans-export.integration.test.ts` → chạy thật integration test PASSED `10` tests (`exit=0`, biên lai `EV-20261117T134845-f998`).
- **TypeScript Build Check (`EV-20261117T134849-c7fe`):** `npx tsc -p tsconfig.build.json --noEmit` → PASSED không có lỗi kiểu (`exit=0`, biên lai `EV-20261117T134849-c7fe`).
- **ESLint Production Files (`EV-20261117T134853-2990`):** `npx eslint <13 production files>` → PASSED không còn lỗi lint nào (`exit=0`, biên lai `EV-20261117T134853-2990`).

### 2.2. Giới hạn & Điểm Chưa Kiểm Chứng Cần Phối Hợp (Limitations & Not Verified)
- `test/integration/trans-his-pg.integration.test.ts` là opt-in suite cần biến môi trường `TRANS_HIS_PG_IT_URL` trỏ vào PostgreSQL thật đã chạy migration `011_transaction_history_classification.sql`. Không suy diễn từ unit test `compileFilter` rằng DB môi trường đích đã có cột `cd_flow` / `biz_type` hay đã ăn đúng partial index `idx_th_cd_flow_created_at_id`!
- Cần phối hợp cùng anh xác nhận thứ tự triển khai (`N2`): đảm bảo `core-trans-proxy-history-service-v2` đã apply migration `011` + bật `CLASSIFICATION_COLUMNS=true` và `centralized-export-service` đặt `TRANS_HIS_EXPORT_SOURCE=postgres` trước khi nghiệm thu runtime tính năng UC02 trên môi trường đích.

---

## Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity & Security

1. **Concurrency & Multi-Worker:** `query.resolvedBillIds` được gắn trên từng instance `GetAllPartnerCollectionTransExportQuery` riêng của mỗi job (`partner-collection-trans-export.pure.ts:100`), nên hoàn toàn cô lập giữa các worker chạy song song (`EXPORT_WORKER_CONCURRENCY > 1`) và giảm tải connection pool `PaymentBillModel` xuống đúng `1` lần tra cứu cho cả job (không cần debouncer riêng).
2. **Timeout / Heartbeat / AckWait:** Mặc dù `BILL_LOOKUP_LIMIT = 1000` đã chặn rủi ro nổ RAM/BSON trên `PaymentBillModel`, việc thiếu `isDelete: { $ne: true }` (`N1`) trên Postgres khiến câu lệnh `SELECT count(*)` và `getPage` trong `TransHisPgModel` (`QUERY_TIMEOUT_MS = 300_000`, worker queue không có lease heartbeat/ackwait tự động) không tận dụng được partial index `idx_th_cd_flow_created_at_id`, có nguy cơ giữ slot worker lâu trên bảng lớn.
3. **Data Continuity (Watermark / Cursor):**
   - Trên `TRANS_HIS_EXPORT_SOURCE=postgres`: Keyset Cursor `(sortExpr, core.id)` qua `pageSegments` bảo toàn tuyệt đối tính liên tục dữ liệu (Watermark / Cursor Data Continuity — không lặp, không mất bản ghi khi có giao dịch mới ghi vào lúc đang xuất nhiều trang).
   - Trên `TRANS_HIS_EXPORT_SOURCE=mongo`: Đã có `_id` tie-breaker đảm bảo thứ tự tất định giữa các trang cùng timestamp/status.
4. **Security Review (`/security-agent`):**
   - Mọi giá trị trong `trans-his-pg.query.ts` đều bind qua `$1..$n` (không có SQL injection); `safeText` đã chặn đủ `[=+\-@\t\r\n]`; không có secret/PII cứng trong toàn bộ 20 files.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- Toàn bộ đánh giá ở Vòng 2 đều đối chiếu trực tiếp source code thực tế giữa `centralized-export-service` (`7f4e9a7`) và `core-trans-proxy-history-service-v2` (`011_transaction_history_classification.sql`, `transaction-history.repository.ts`), trung thực tuyệt đối (honesty check, zero speculation).

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Các Global Lessons đã áp dụng:** `L076` (đối chiếu điều kiện biên Partial Index PostgreSQL `WHERE cd_flow IS NOT NULL AND is_delete IS DISTINCT FROM TRUE`), `L078` & `L080` (kiểm tra schema cột phân loại `cd_flow`/`biz_type` giữa v1/v2 & ép kiểu JSONB), `L042` (tách lệnh CLI sạch), cùng Core Tripwires `G1`–`G4`.
- **Vi phạm Lesson trong phiên:** `0`.

---

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`laya_gate.py` — `DG0`, `DG1`, `DG3`)

- **Engine sử dụng:** VegaML Primary (`http://127.0.0.1:8222/v1/predict`, `mode=auto`, `0 LLM token`).
- **Đối chiếu đĩa (`REF:` Verification):** Toàn bộ claims thực tế trên commit `7f4e9a7` qua `DG1`/`DG2` và `DG3` đều khớp dòng code vật lý trên đĩa (`evidence_error = ""`, không có `BLOCK`).

```json
{
  "dg1_dg2_v2": {
    "summary_file": "06_laya_summary_dg1_20261117T135454.json",
    "dg2_summary_file": "06_laya_summary_dg2_20261117T135454.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.7138 - 0.8804",
    "claims_decision": "UNVALIDATED (7 claims: S-RC-1, S-FIX-1, S-IMPACT-1, S-PATTERN-1, D-BEHAV-1, D-EDGE-1, D-API-1)"
  },
  "dg3_v2": {
    "summary_file": "06_laya_summary_dg3_20261117T135455.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.6866 - 0.9086",
    "claims_decision": "UNVALIDATED (4 claims: A-N1..A-N3, A-R1)"
  }
}
```

- **Nhận xét Ground-Truth (Rule #10):** Do vòng control của VegaML `:8222` chưa tách biệt vượt ngưỡng `DECISIVE_CONF` (`trusted_rounds = []` → `UNVALIDATED`), em báo cáo rõ trạng thái `UNVALIDATED` của engine và xác lập kết luận cuối cùng trực tiếp từ source code `file:line` cùng các biên lai thực thi PASSED `EV-20261117T134840-5a70`, `EV-20261117T134845-f998`, `EV-20261117T134849-c7fe` và `EV-20261117T134853-2990`.
