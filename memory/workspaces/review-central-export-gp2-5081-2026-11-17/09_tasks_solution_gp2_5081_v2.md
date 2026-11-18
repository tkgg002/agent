# 09. Solution & Re-Review Proposal (Pass 2) — `feat/GP2-5081 -> develop` (`7f4e9a7`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_v2`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Commit mới nhất trên nhánh:** `7f4e9a7e149dabd75674f9fb3737c81bbfa21794` (`5` commits trước `origin/develop` `3eef807`, `0` commit bị tụt sau `origin/develop`, `20` files changed, `+1420 / -4`).
- **Ràng buộc Thực thi (Technical Constraints từ Global Lessons):**
  - **TC1 (`G2` / Rule #1A, #13):** *Audit Propose-Only* — không tự ý sửa source code khi chưa có lệnh `APPROVE`.
  - **TC2 (`L076` — Partial Index Alignment & No Full Scans):** Mọi truy vấn vào `public.transaction_history` phải khớp điều kiện biên (`WHERE cd_flow IS NOT NULL AND is_delete IS DISTINCT FROM TRUE`) của partial index `idx_th_cd_flow_created_at_id` (migration `011`).
  - **TC3 (`L078` & `L080` — Schema Verification & Type Safety):** Kiểm tra tồn tại cột `cd_flow`/`biz_type` giữa môi trường v1 (Mongo) và v2 (Postgres), cũng như format `failedReason` khi thiếu `error.code`.

---

## 1. Đối Chiếu Kết Quả Khắc Phục 6 Vấn Đề Từ Vòng Review 1 (`F1` → `F6`)

| Mã Vòng 1 | Trạng thái ở `7f4e9a7` | Bằng chứng xác thực (`file:line` & Commit) |
| :--- | :---: | :--- |
| **F1** (Postgres source, 15 cột template, bỏ ép `serviceCode` đơn gốc RADA, fallback `receiver`) | ✅ **ĐÃ SỬA (`90%` — còn điểm `isDelete` ở `N1`)** | - Commit `0928958`: Thêm `GetAllPartnerCollectionTransExportPgHandler.ts:1-34`, cập nhật `QueryHandlerFactory.ts:156-164` và `partner-collection-trans.export.ts:23-25` (`[...transHisModelNames(), "PaymentBillModel"]`), mở rộng `models/trans-his-pg/trans-his-pg.query.ts:97-139,268-365`.<br>- Commit `51c8898` & `28460e0`: Đổi `baseFilter()` sang `{ cdFlow: "COLLECTION", bizType: { $nin: ["COLLATERAL_DEPOSIT", "DEPOSIT_EWALLET"] } }`, bỏ `serviceCode` khỏi `billFilter`, đồng bộ 15 cột template (`transactionType`, `updatedAt`, `noted`, `failedReason`), và fallback `payee?.code ?? item.receiver?.merchantCode` (`partner-collection-trans-export.pure.ts:35-38,95-158`). |
| **F2** (`@MinLength(3)`, giới hạn & cache tra cứu `merchantTransId`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `partner-collection-trans-export.params.ts:46-55`: Đã gắn `@MinLength(3, { message: "ERR_MIN_LENGTH_3" })` cho `merchantTransId`, `merchantEmail`, `payeeCode`.<br>- `domain/handlers/trans-his/partner-collection-trans-bills.ts:7-34`: Giới hạn `BILL_LOOKUP_LIMIT = 1000` (có `logger.warn` khi chạm trần) và cache `query.resolvedBillIds` ở chunk đầu tiên để tái sử dụng cho mọi chunk tiếp theo. |
| **F3** (Tie-breaker `_id` & Keyset Cursor) | ✅ **ĐÃ SỬA (`100%` trên PG, thêm tie-breaker trên Mongo)** | - Trên Postgres (`GetAllPartnerCollectionTransExportPgHandler.ts:26` + `trans-his-pg.sort.ts:192-211`): Dùng Keyset Cursor `(sortExpr, core.id)` độ phức tạp $O(N)$.<br>- Trên Mongo (`GetAllPartnerCollectionTransExportHandler.ts:41-42`): Đã thêm `sortWithTieBreaker = { ...sort, _id: dir }` loại bỏ lỗi trùng/sót dòng do thứ tự không tất định. |
| **F4** (`dateTo < dateFr`, chuỗi khoảng trắng `"   "`, `sortType` số) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `partner-collection-trans-export.params.ts:26,62`: Kiểm tra `dateDiff >= 0 && dateDiff <= MAX_DAYS_EXPORT`; mở rộng `sortType?: string | number`.<br>- `partner-collection-trans-export.pure.ts:21,25-29,47-49`: `isEmptyValue` đã `.trim()`, `billIdVariants` kiểm tra `text !== "" && Number.isSafeInteger(asNumber)`, `status` đã `.trim()` và `.filter(Boolean)`. |
| **F5** (3 lỗi ESLint & `\n` trong `safeText`) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - `npx eslint` trên toàn bộ 13 file production đạt `0 errors, 0 warnings` (`EV-20261117T134853-2990`).<br>- `partner-collection-trans-export.pure.ts:84`: Đã cập nhật `/^[=+\-@\t\r\n]/`. |
| **F6** (Gỡ mock `validateParams` trong unit test) | ✅ **ĐÃ SỬA TRIỆT ĐỂ (`100%`)** | - Commit `7f4e9a7` (`partner-collection-trans.pure.test.ts:60-84`): Đã gỡ bỏ hoàn toàn `vi.mock("../../../utils/validator")` và kiểm thử trực tiếp `Pure.validate()` thật (`100/100` unit tests xanh — `EV-20261117T134840-5a70`). |

---

## 2. Phát Hiện Còn Tồn Đọng / Mới Phát Sinh Ở Bản `7f4e9a7` (`N1` → `N3`)

| Mã | Mức độ | Trạng thái | Chi tiết phát hiện & Bằng chứng `file:line` |
| :--- | :---: | :---: | :--- |
| **N1** | **P1 (High)** | `VERIFIED` | **Thiếu điều kiện `isDelete: { $ne: true }` (`core.is_delete IS DISTINCT FROM TRUE`) làm Postgres KHÔNG ăn được Partial Index `idx_th_cd_flow_created_at_id` VÀ xuất lọt bản ghi xóa mềm:**<br>- Trong `core-trans-proxy-history-service-v2`:<br>  1. Migration `011_transaction_history_classification.sql:11-13` tạo Partial Index: `CREATE INDEX ... idx_th_cd_flow_created_at_id ON public.transaction_history (cd_flow, created_at DESC NULLS LAST, id DESC) INCLUDE (status) WHERE cd_flow IS NOT NULL AND is_delete IS DISTINCT FROM TRUE;`.<br>  2. `transaction-history.repository.ts:379` (`ruleConditionWhere`) luôn đính kèm `NOT_DELETED_SQL` (`core.is_delete IS DISTINCT FROM TRUE`). Ngay cả câu SQL đối chiếu màn hình trong `test/integration/trans-his-pg.integration.test.ts:260` cũng viết `AND c.is_delete IS DISTINCT FROM TRUE`.<br>- Tuy nhiên, `baseFilter()` tại `partner-collection-trans-export.pure.ts:35-38` chỉ trả về `{ cdFlow: COLLECTION_FLOW, bizType: { $nin: [...EXCLUDED_BIZ_TYPES] } }` (thiếu `isDelete: { $ne: true }`), và `fieldSql` tại `models/trans-his-pg/trans-his-pg.query.ts:229-245` mới chỉ ánh xạ `sysTrans` chứ chưa ánh xạ `isDelete` sang `booleanSql(key, "core.is_delete", spec)`.<br>- **Hệ quả:** (1) PostgreSQL planner không thể dùng partial index `idx_th_cd_flow_created_at_id` vì câu query thiếu mệnh đề `is_delete IS DISTINCT FROM TRUE`, phải quét theo `created_at` trên toàn bộ giao dịch mọi luồng trong 31 ngày; (2) Nếu có bản ghi `is_delete = true`, file xuất sẽ thừa dòng so với màn hình CMS (và làm fail chính `expect(all.total).toBe(screen)` ở `trans-his-pg.integration.test.ts:264`). |
| **N2** | **P1 (Deployment Gate)** | `VERIFIED` | **Ràng buộc thứ tự Release (Migration `011` + `TRANS_HIS_EXPORT_SOURCE=postgres`):**<br>- `models/trans-his-pg/trans-his-pg.query.ts:98-99` truy vấn trực tiếp `core.cd_flow` và `core.biz_type` (không check động qua `information_schema` như `idempotency_key`). Đồng thời trên MongoDB v1 (`core-trans-proxy-service` / `core-trans-proxy-history-service`), các bản ghi lịch sử cũ không có trường `cdFlow`.<br>- **Yêu cầu khi triển khai:** Trên môi trường chạy UC02 (`develop`/`testing`/`prod`), bắt buộc phải chạy xong migration `011_transaction_history_classification.sql` (`GP2-5154`), bật ghi `CLASSIFICATION_COLUMNS=true` trên `history-service-v2`, và cấu hình `TRANS_HIS_EXPORT_SOURCE=postgres` cho `centralized-export-service` thì export mới khớp màn hình v2 (`GP2-5069`). (Các export cũ không dùng `cdFlow` nên merge code này vào `develop` hoàn toàn KHÔNG làm hỏng bất kỳ tính năng hiện hữu nào). |
| **N3** | **P2 (Low/Medium)** | `VERIFIED` | **2 lỗi biên nhỏ trong `failedReason` và tìm kiếm ký tự có dấu trên Postgres:**<br>1. `partner-collection-trans-export.pure.ts:140`: `error ? item.info?.failedReason || error.message?.[lang] || \`err: ${error.code}\` : ...` — nếu `item.error` là `{}` hoặc thiếu `code` (ví dụ chỉ có `message` ngôn ngữ khác), cột *Lý do thất bại* sẽ in ra chữ `"err: undefined"`.<br>2. Trên `TRANS_HIS_EXPORT_SOURCE=postgres`, `approvedRegex` (`trans-his-pg.query.test.ts:464`) ném lỗi `"Invalid transaction history regex"` với ký tự non-ASCII (như tiếng Việt có dấu `"Nguyễn"`). Nếu user gõ ký tự có dấu vào `merchantEmail` / `payeeCode`, API `exportAsync` vẫn trả `QUEUED` nhưng job nền trong `ExportWorker` sẽ bị `FAILED` thay vì chặn ngay ở DTO. |

---

## 3. Phương Án Khắc Phục Duy Nhất Tối Ưu Nhất & Code Demo (`N1` & `N3`)

> Bản vá chỉ cần **~8 dòng code** trên 2 file (`partner-collection-trans-export.pure.ts` và `models/trans-his-pg/trans-his-pg.query.ts`) là đóng trọn vẹn cả Partial Index PostgreSQL lẫn edge case `failedReason`:

### 3.1. `models/trans-his-pg/trans-his-pg.query.ts` (Bổ sung 1 dòng ánh xạ `isDelete` ở dòng `240`)
```typescript
	if (key === "sysTrans") { return [booleanSql(key, "core.sys_trans", spec)]; }
	if (key === "isDelete") { return [booleanSql(key, "core.is_delete", spec)]; }
	if (key === "createdAt") { return dateSql(key, "core.created_at", spec, bind); }
```

### 3.2. `logics/export/trans-his/partner-collection-trans-export.pure.ts` (Thêm `isDelete: { $ne: true }` vào `baseFilter` & chặn `"err: undefined"`)
```typescript
export const baseFilter = (): Record<string, any> => ({
    cdFlow: COLLECTION_FLOW,
    bizType: { $nin: [...EXCLUDED_BIZ_TYPES] },
    isDelete: { $ne: true }
});
...
    const fallbackErrCode = error?.code !== undefined && error?.code !== null && error?.code !== ""
        ? `err: ${error.code}`
        : "";
    const failedReason = error
        ? item.info?.failedReason || error.message?.[lang] || fallbackErrCode
        : item.info?.failedReason;
```
