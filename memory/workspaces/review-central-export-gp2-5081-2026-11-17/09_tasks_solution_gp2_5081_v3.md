# 09. Solution & Final Re-Review Assessment (Pass 3) — `feat/GP2-5081 -> develop` (`92ebb72`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_v3`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Commit mới nhất trên nhánh:** `92ebb728c008808618b56e93c316c803416fcc02` (`6` commits trước `origin/develop` `3eef807`, `0` commit bị tụt sau `origin/develop`, `19` files changed, `+1262 / -5`, `git merge-tree --write-tree` sạch xung đột `100%`).
- **Kết luận Tổng thể (Merge Verdict):** **APPROVE / READY TO MERGE (`0` lỗi `P0`, `0` lỗi code `P1`)** — Toàn bộ 5 vi phạm Design Pattern (`DP1`–`DP5`), lỗi thiếu `isDelete` cho Partial Index PostgreSQL (`N1`), cũng như toàn bộ 6 hạng mục `F1`–`F6` từ đầu phiên đã được dev khắc phục triệt để (`100%`).
- **Ràng buộc Thực thi (Technical Constraints từ Global Lessons):**
  - **TC1 (`L002` — Clean Architecture & Design Pattern Audit):** Chiều phụ thuộc `domain/ -> logics/` bằng `0`; `IQuery` bất biến 100%; tái sử dụng `GetAllTransHisExportQuery` + `BaseExportProcessor` hooks (`async getConfig(..., queryExecutor)`, `subQueryClass`, `buildSubQueryParams`, `mergeData`).
  - **TC2 (`L076` — PostgreSQL Partial Index Alignment):** Mệnh đề `isDelete: { $ne: true }` biên dịch sang `core.is_delete IS DISTINCT FROM TRUE`, khớp chính xác điều kiện biên của Partial Index `idx_th_cd_flow_created_at_id`.
  - **TC3 (`L078` & `L080` — Schema & Projection Parity):** Đối chiếu đầy đủ 11 trường `SELECT_FIELDS` với `TransHisPgModel` (`SELECT_SQL` + `toHistoryProjection`) và kiểm tra tương thích ngược với 3 màn hình export `trans-his` hiện hữu.
  - **TC4 (`G1`–`G4`, `L049`):** Giữ chế độ *Propose-Only* (`0` dòng chỉnh sửa source code), kiểm chứng bằng biên lai thực thi vật lý `EV-20261118...`.

---

## 1. Bảng Đối Chiếu Kết Quả Khắc Phục Qua 4 Lăng Kính Trên Bản `92ebb72`

| Nhóm / Mã | Trạng thái ở `92ebb72` | Bằng chứng xác thực (`file:line`) |
| :--- | :---: | :--- |
| **DP1** (Vi phạm Clean Architecture `domain -> logics`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | Đã xóa sạch toàn bộ 4 file vi phạm (`partner-collection-trans-bills.ts`, `GetAllPartnerCollectionTransExportQuery.ts`, `GetAllPartnerCollectionTransExportHandler.ts`, `GetAllPartnerCollectionTransExportPgHandler.ts`). Quét toàn bộ `domain/` không còn bất kỳ import nào trỏ lên `logics/`. `GetPaymentBillExportAuxiliaryHandler.ts:1-5` chỉ import `lodash`, `IQueryHandler`, `GetPaymentBillExportAuxiliaryQuery` và `MainProcess`. |
| **DP2** (Bỏ qua Template Method Hooks của `BaseExportProcessor`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `partner-collection-trans.export.ts:15-19`: Cài đặt `protected async getConfig(): Promise<ExportConfig>` truyền `queryExecutor` xuống tầng pure (đúng chuẩn `reconciliation-report.export.ts:14-18` & `payout-request.export.ts:14-18`).<br>- `partner-collection-trans-export.pure.ts:121-152`: Tra cứu `merchantTransId` **1 lần duy nhất** trước vòng lặp chunk qua `queryExecutor(new GetPaymentBillExportAuxiliaryQuery(...))`, và khai báo chuẩn bộ 3 hook `subQueryClass: GetPaymentBillExportAuxiliaryQuery`, `buildSubQueryParams`, `mergeData` cho từng chunk. |
| **DP3** (Biến trạng thái khả biến `resolvedBillIds` trên `IQuery`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | `GetPaymentBillExportAuxiliaryQuery.ts:8-18` khai báo toàn bộ thuộc tính `public readonly` (`merchantTransId`, `billIds`, `limit`); `GetAllTransHisExportQuery` không chứa bất kỳ biến mutable nào. |
| **DP4** (Đẻ thừa Query/Handler trùng lặp thay vì tái sử dụng) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `partner-collection-trans-export.pure.ts:131`: Tái sử dụng `new GetAllTransHisExportQuery(filter, sort, SELECT_FIELDS)` giống hệt 3 màn hình `TransHisExport`, `TransHisCollectionExport` và `InternalTransferExport`.<br>- `QueryHandlerFactory.ts:136,156-162`: Chỉ đăng ký thêm duy nhất `GetPaymentBillExportAuxiliaryQuery` dưới nhóm `payment-bill`, giữ gọn phân nhánh `postgres` / `mongo` của `trans-his`. |
| **DP5** (Chữ ký `transformRow` 2 tham số & thiếu `chunkSize: 500`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | `partner-collection-trans-export.pure.ts:177,188` đặt `chunkSize: 500` (giới hạn `<= 500` ID mỗi truy vấn `$in` phụ) và chuẩn hóa chữ ký 3 tham số `transformRow(item: any, _rowIndex: number, langCode: string = "vi")` đồng bộ với `partner-collection-trans.export.ts:21-23` (`++this.rowIndex`). |
| **N1** (Thiếu `isDelete: { $ne: true }` làm lọt xóa mềm & trượt Partial Index PG) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `partner-collection-trans-export.pure.ts:51-55`: `baseFilter()` đã bổ sung `isDelete: { $ne: true }`.<br>- `models/trans-his-pg/trans-his-pg.query.ts:240,377-384`: Ánh xạ `isDelete` qua `booleanSql(key, "core.is_delete", spec)` sinh ra `core.is_delete IS DISTINCT FROM TRUE`, khớp 100% điều kiện Partial Index `idx_th_cd_flow_created_at_id` (`WHERE cd_flow IS NOT NULL AND is_delete IS DISTINCT FROM TRUE`) và được kiểm chứng bằng case `T-X-DELETED` tại `partner-collection-trans-export.integration.test.ts:133`. |
| **Contract & Regression** (Ảnh hưởng tới các luồng export hiện hữu) | ✅ **AN TOÀN (`0` Regression)** | - `GetAllTransHisExportHandler.ts:22-25`: Tạo bản sao `sortWithTieBreaker` (không mutate `query.sort`) thêm `_id: 1 | -1` khi `sort` chưa có `_id`, giúp phân trang `.skip(cursor)` tất định cho toàn bộ 4 màn hình `trans-his` trên Mongo.<br>- Toàn bộ 11 trường trong `SELECT_FIELDS` (`partner-collection-trans-export.pure.ts:14-26`) đều có mặt trong `SELECT_SQL` (`trans-his-pg.model.ts:18-42`) và `toHistoryProjection` (`trans-his-pg.mapper.ts:207-226`).<br>- TypeScript build (`EV-20261118T093919-664a`), ESLint 12 file production (`EV-20261118T093924-cab5`), 12 bộ unit test (`154` tests — `EV-20261118T094123-f709`) và các bộ integration test (`EV-20261118T093454-3d78`, `EV-20261118T094128-9504`) đều thoát mã `0`. |
| **Security** (NoSQL/SQL Injection, ReDoS, Formula Injection, DoS) | ✅ **ĐẠT CHUẨN (`0` Lỗ hổng)** | - Escape regex đầy đủ bằng `_.escapeRegExp` (`partner-collection-trans-export.pure.ts:38`, `GetPaymentBillExportAuxiliaryHandler.ts:23`) kèm `@MinLength(3)` và trần `limit = 1000`.<br>- Toàn bộ câu lệnh SQL trong `trans-his-pg.query.ts` dùng bind tham số `$n` và `approvedRegex`.<br>- `safeText` (`partner-collection-trans-export.pure.ts:99-102`) trung hòa đầy đủ `= + - @ \t \r \n` trên tất cả 12 cột chuỗi; khoảng ngày khóa cứng `0 <= diff <= 31` ngày (`partner-collection-trans-export.params.ts:26`). |

---

## 2. Lưu Ý Vận Hành (`N2`) & Điểm Nhỏ Không Chặn Merge (`P2` Optional Polish)

| Mã | Phân loại | Trạng thái | Chi tiết & Khuyến nghị |
| :--- | :---: | :---: | :--- |
| **N2** | **Deployment Order Gate (Vận hành — Không phải lỗi code)** | `VERIFIED` | Khi bật `TRANS_HIS_EXPORT_SOURCE=postgres` cho UC02 trên `develop`/`testing`/`prod`, môi trường đích cần chạy trước migration `011_transaction_history_classification.sql` (`GP2-5154` — tạo cột `cd_flow`, `biz_type` và partial index `idx_th_cd_flow_created_at_id`) và bật `CLASSIFICATION_COLUMNS=true` trên `core-trans-proxy-history-service-v2`. Việc merge nhánh này vào `develop` **không ảnh hưởng** bất kỳ export hiện hữu nào (do các export cũ không truy vấn `cdFlow`/`bizType`). Ngoài ra khi merge song song với `origin/feat/GP2-4633` chỉ cần giữ cả 2 dòng thêm mới (keep-both) tại 5 file barrel/factory (`params/index.ts`, `queries/index.ts`, `QueryHandlerFactory.ts`, `logics/export/index.ts`, `logics/index.ts`). |
| **P2-1** | **P2 (Minor / Non-blocking)** | `VERIFIED` | Tại `partner-collection-trans-export.pure.ts:194`: `const failedReason = error ? item.info?.failedReason || error.message?.[lang] || \`err: ${error.code}\` : item.info?.failedReason;` — nếu `item.error` là object rỗng `{}` hoặc chỉ có `message` ở ngôn ngữ khác mà thiếu `error.code`, cột *Lý do thất bại* sẽ ghi `"err: undefined"`. |
| **P2-2** | **P2 (Minor / Non-blocking)** | `VERIFIED` | Tại `partner-collection-trans-export.pure.ts:138-139`: `buildSubQueryParams` dùng `.filter(id => id !== undefined && id !== null && id !== "").map(id => (Number.isNaN(Number(id)) ? id : Number(id)))` — nếu gặp chuỗi toàn khoảng trắng `"   "` thì `Number("   ")` sẽ ép thành `0`, và số dài >53-bit không kiểm tra `Number.isSafeInteger` như hàm `billIdVariants` ở dòng `42-44`. Trong thực tế `PaymentBillModel._id` là chuỗi/số sequence nguyên `< MAX_SAFE_INTEGER` nên không gây lỗi sản xuất, nhưng có thể đồng bộ cho sạch tuyệt đối. |

---

## 3. Phương Án Tinh Chỉnh Tùy Chọn Duy Nhất & Code Demo (Không Chặn Merge — Chỉ Áp Dụng Nếu Cần Polish `P2-1` & `P2-2`)

> Nhánh hiện tại **đã đủ tiêu chuẩn merge ngay vào `develop`**. Trường hợp anh muốn gọt sạch luôn 2 edge case `P2-1` và `P2-2` trong `logics/export/trans-his/partner-collection-trans-export.pure.ts` (chỉ thay 6 dòng):

```typescript
// logics/export/trans-his/partner-collection-trans-export.pure.ts (dòng 133-141 & 193-195)
        buildSubQueryParams: (primaryData: any[]) => ({
            // numeric safe integers first, then unique: "101" and 101 are the same order
            billIds: _.uniq(
                (primaryData || [])
                    .map(row => row?.info?.billId)
                    .filter(id => !isEmptyValue(id))
                    .map(id => {
                        const text = String(id).trim();
                        const asNum = Number(text);
                        return Number.isSafeInteger(asNum) ? asNum : text;
                    })
            )
        }),
// ...
    const error = item.error;
    const errorCodeText = error?.code !== undefined && error?.code !== null && error?.code !== "" ? `err: ${error.code}` : "";
    const failedReason = error ? item.info?.failedReason || error.message?.[lang] || errorCodeText : item.info?.failedReason;
```
