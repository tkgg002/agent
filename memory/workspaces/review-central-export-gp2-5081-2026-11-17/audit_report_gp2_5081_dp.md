# Audit Report & Consolidated Design Pattern Validation — `feat/GP2-5081 -> develop` (`7f4e9a7`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_dp`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Nhánh nguồn → đích:** `origin/feat/GP2-5081` (`7f4e9a7e149dabd75674f9fb3737c81bbfa21794`) → `origin/develop` (`3eef80790b319e4d0f188e3d12cef9f0ac90b302`)
- **Trạng thái Source Code dự án:** `git status -s` rỗng (`0` dòng chỉnh sửa — tuân thủ tuyệt đối Propose-Only).

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng task theo trục Design Pattern (Line-by-Line Diff & Task Critique)

### 1.1. Phản biện Kiến trúc từng file trong `git diff` (`20` files changed, tổng `3160` dòng)
1. `domain/handlers/trans-his/partner-collection-trans-bills.ts` (`66` dòng, mới) & `GetAllPartnerCollectionTransExportHandler.ts` (`62` dòng, mới):
   - Dòng `4` và dòng `5` nhập ngược hàm `billIdVariants` từ `../../../logics/export/trans-his/partner-collection-trans-export.pure` → **vi phạm Clean Architecture (`DP1`)**: trong toàn bộ 37 handlers có sẵn trên `origin/develop`, không có bất kỳ file nào ở tầng `domain/` import ngược lên tầng `logics/`.
   - Dòng `34-55` (`GetAllPartnerCollectionTransExportHandler.ts`) và dòng `17-31` (`GetAllPartnerCollectionTransExportPgHandler.ts`, `34` dòng): trực tiếp gọi `PaymentBillModel` bên trong handler của `trans-his` → **vi phạm ranh giới Bounded Context và bỏ qua Template Method của `BaseExportProcessor` (`DP2`)**, đồng thời đẻ thừa 3 file handler + 1 file query thay vì tái sử dụng `GetAllTransHisExportQuery` như `TransHisExport`, `TransHisCollectionExport` và `InternalTransferExport` (`DP4`).
2. `domain/queries/trans-his/GetAllPartnerCollectionTransExportQuery.ts` (`16` dòng, mới):
   - Dòng `9`: khai báo thuộc tính khả biến `public resolvedBillIds?: (string | number)[]` để handler tự ghi đè lúc chạy (`partner-collection-trans-bills.ts:30`) → **vi phạm tính bất biến của CQRS `IQuery` Value Object (`DP3`)**.
3. `logics/export/trans-his/partner-collection-trans-export.pure.ts` (`159` dòng) & `partner-collection-trans.export.ts` (`28` dòng):
   - Bỏ qua cả `async getConfig(..., queryExecutor)` (như `payout-request.export.ts:14-18`) lẫn bộ 3 hook `subQueryClass`, `buildSubQueryParams`, `mergeData` của `BaseExportProcessor.ts:31-35,219-239` (đang dùng đồng nhất ở toàn bộ 13 exporters đa mô hình trong dự án) (`DP2`).
   - Dòng `134`: rút chữ ký `transformRow(item, langCode)` còn 2 tham số lệch với chuẩn 3 tham số `transformRow(item, rowIndex, langCode)` của các file `*.pure.ts` anh em và chưa khai báo `chunkSize: 500` (`DP5`).

### 1.2. Đối chiếu tiến độ từng Task (`08_tasks_gp2_5081_dp.md`)
- `TDP-01`: Đã thực hiện Mid-Session Fix ghi bài học `L001` vào `lessons.md` (`767 -> 774` dòng) và chạy `build_lessons_index.py` đồng bộ 100 bài học.
- `TDP-02`: Đã quét toàn bộ `domain/` và `logics/export/`, xác thực `DP1`–`DP5` kèm `file:line`, hoàn thành `DG0`.
- `TDP-03`: Đã thiết kế phương án tái cấu trúc chuẩn Design Pattern trong `09_tasks_solution_gp2_5081_dp.md` và hoàn tất `DG1`, `DG2`, `DG3`.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

### 2.1. Bằng chứng thực thi vật lý (`06_evidence/EV-*.json` trên commit `7f4e9a7`)
- **Unit Tests (`EV-20261117T134840-5a70`):** PASSED `100` unit tests (`exit=0`, biên lai `EV-20261117T134840-5a70`).
- **Integration Tests (`EV-20261117T134845-f998`):** chạy thật integration test PASSED `10` tests (`exit=0`, biên lai `EV-20261117T134845-f998`).
- **TypeScript Build Check (`EV-20261117T134849-c7fe`):** PASSED không lỗi kiểu (`exit=0`, biên lai `EV-20261117T134849-c7fe`).
- **ESLint Production Files (`EV-20261117T134853-2990`):** PASSED sạch lỗi cú pháp (`exit=0`, biên lai `EV-20261117T134853-2990`).

### 2.2. Giới hạn & Điểm Chưa Kiểm Chứng Cần Phối Hợp (Limitations & Not Verified)
- Việc 100 unit tests và 10 integration tests đều chạy qua là minh chứng điển hình cho thấy **Test qua ≠ Đúng Kiến trúc & Design Pattern**: các test chỉ kiểm tra hành vi đầu-cuối mà không phát hiện được chiều import ngược `domain -> logics` (`DP1`), trạng thái mutable trên `IQuery` (`DP3`), hay việc phá vỡ Template Method của `BaseExportProcessor` (`DP2`).
- Khi anh duyệt `APPROVE` để thực thi tái cấu trúc sang `GetAllTransHisExportQuery` + `GetPaymentBillExportAuxiliaryHandler`, em sẽ chạy lại đầy đủ unit + integration tests và phối hợp cùng anh kiểm chứng trên môi trường runtime thật nếu cần.

---

## Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity & Security

1. **Concurrency & Multi-Worker:** Khi chuyển sang `async getConfig(..., queryExecutor)` + Stateless `IQuery` (`DP3`) thay vì gắn biến mutable `resolvedBillIds` lên `IQuery`, đối tượng `GetAllTransHisExportQuery` hoàn toàn bất biến (an toàn tuyệt đối giữa multi-worker ngay cả khi tái sử dụng hoặc retry job, không cần debouncer phụ).
2. **Timeout / Heartbeat / AckWait:** Đặt `chunkSize: 500` (`DP5`, khớp `PaymentHistoryExport` & `PayoutRequestExport`) giúp mỗi vòng `subQueryClass` (`GetPaymentBillExportAuxiliaryQuery`) chỉ truy vấn tối đa `<= 500` `_id` đơn hàng cực nhanh, không giữ kết nối lâu làm nghẽn worker trong bối cảnh `export:queue` chưa có lease heartbeat/ackwait tự động; đồng thời `BaseExportProcessor.ts:236-238` đã bọc sẵn `try/catch` quanh sub-query nên lỗi DB phụ không làm đứt tiến trình xuất chính.
3. **Data Continuity (Watermark / Cursor):** Tái sử dụng `GetAllTransHisExportPgHandler` giữ nguyên Keyset Cursor `(sortExpr, core.id)` trên Postgres, và bổ sung `_id` tie-breaker vào `GetAllTransHisExportHandler` nâng cấp tính liên tục dữ liệu (Watermark / Cursor Data Continuity) cho toàn bộ 4 màn hình xuất `trans-his` trên Mongo cùng lúc!
4. **Security Review (`/security-agent`):** Tách `GetPaymentBillExportAuxiliaryHandler` giữ nguyên `_.escapeRegExp` và `limit = 1000`, không lộ secret/PII.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- Em xin nhận thiếu sót cầu thị (honesty check, zero speculation): ở 2 lượt review trước em đã tập trung vào lỗi nghiệp vụ, SQL, index, validation và lint mà **chưa kiểm toán chuyên sâu trục Design Pattern & ranh giới tầng Clean Architecture** cho đến khi anh nhắc nhở. Toàn bộ 5 phát hiện `DP1`–`DP5` trong báo cáo này đều được đối chiếu thực chứng trên toàn bộ `37` handlers và `13` cross-model exporters của dự án.

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Thực thi nghiêm ngặt Rule #5 (Mid-Session Fix):** Ngay khi nhận lời nhắc của anh, em đã dừng lại, đúc kết bài học tổng quát `L001` (*Mandatory Design Pattern, Layer Dependency & Template Hook Audit In Code Review*) chèn vào đầu `memory/global/lessons.md` (`wc -l`: `767 -> 774`) và chạy `build_lessons_index.py` đồng bộ đầy đủ 100 bài học vào `lessons/INDEX.md` trước khi tiến hành kiểm toán.
- **Các bài học khác cùng áp dụng:** `L003` (tái sử dụng cấu trúc `centralized-export-service` hiện hữu), `L077` (Partial Index PostgreSQL), `L043` (tách lệnh CLI sạch), `G1`–`G4`.

---

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`laya_gate.py` — `DG0`, `DG1`, `DG3`)

- **Engine sử dụng:** VegaML Primary (`http://127.0.0.1:8222/v1/predict`, `mode=auto`, `0 LLM token`).
- **Đối chiếu đĩa (`REF:` Verification):** Toàn bộ `14` claims qua 3 cổng (`DG0`, `DG1`/`DG2`, `DG3`) đều khớp dòng code vật lý trên đĩa (`evidence_error = ""`, không có `BLOCK`).

```json
{
  "dg0_dp": {
    "summary_file": "06_laya_summary_dg0_20261117T140821.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.7610 - 0.8951",
    "claims_decision": "UNVALIDATED (3 claims: C-REQ-1, C-REALITY-1, C-CONTRA-1)"
  },
  "dg1_dg2_dp": {
    "summary_file": "06_laya_summary_dg1_20261117T140828.json",
    "dg2_summary_file": "06_laya_summary_dg2_20261117T140828.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.7807 - 0.9276",
    "claims_decision": "UNVALIDATED (7 claims: S-RC-1, S-FIX-1, S-IMPACT-1, S-PATTERN-1, D-BEHAV-1, D-EDGE-1, D-API-1)"
  },
  "dg3_dp": {
    "summary_file": "06_laya_summary_dg3_20261117T140825.json",
    "gate": "PASS",
    "trusted_rounds": [],
    "r3_yes_confidence_range": "0.6996 - 0.9129",
    "claims_decision": "UNVALIDATED (4 claims: A-DP1..A-DP4)"
  }
}
```

- **Nhận xét Ground-Truth (Rule #10):** Do vòng control của VegaML `:8222` chưa tách biệt vượt ngưỡng `DECISIVE_CONF` (`trusted_rounds = []` → `UNVALIDATED`), em công bố rõ trạng thái `UNVALIDATED` và xác lập kết luận trực tiếp bằng đối chiếu mã nguồn thật trên toàn dự án kèm các biên lai PASSED `EV-20261117T134840-5a70`, `EV-20261117T134845-f998`, `EV-20261117T134849-c7fe` và `EV-20261117T134853-2990`.
