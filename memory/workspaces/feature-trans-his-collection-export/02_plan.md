# Implementation Plan - Refined Transaction Filtering Logic (Dual Language)

## English
Refactor the `buildTransHisFilter` function in `trans-his-collection-export.pure.ts` to implement a robust filtering logic using an `$or` clause for special transaction types (`REFUND_CASHIN`, `INTERNAL_BANK_TRANSFER`). This ensures these types are included even when the `sysTrans` filter would normally exclude them.

### Proposed Changes
1. **File**: `trans-his-collection-export.pure.ts`
   - Modify `buildTransHisFilter` to identify "special" transaction types.
   - Construct an `$or` query that bypasses the `sysTrans` restriction for these special types while maintaining the rest of the filter's integrity.
   - Ensure compatibility with other `$or` conditions (e.g., `customerId`, `phone`) by using `$and` for composition if necessary, or merging them correctly.

2. **Verification**:
   - Verify that the filter correctly yields the expected MongoDB query structure.
   - Cross-check `getConfig` columns with `transformRow` output array length.
   - Run `npm run build` to ensure no regressions.

## Tiếng Việt
Refactor hàm `buildTransHisFilter` trong `trans-his-collection-export.pure.ts` để triển khai logic lọc mạnh mẽ hơn bằng cách sử dụng toán tử `$or` cho các loại giao dịch đặc biệt (`REFUND_CASHIN`, `INTERNAL_BANK_TRANSFER`). Việc này đảm bảo các loại này được bao gồm ngay cả khi bộ lọc `sysTrans` thông thường sẽ loại bỏ chúng.

### Các thay đổi đề xuất
1. **File**: `trans-his-collection-export.pure.ts`
   - Sửa `buildTransHisFilter` để xác định các loại giao dịch "đặc biệt".
   - Xây dựng một truy vấn `$or` bypass điều kiện `sysTrans` cho các loại đặc biệt này trong khi vẫn giữ nguyên tính toàn vẹn của phần còn lại của bộ lọc.
   - Đảm bảo tương thích với các điều kiện `$or` khác (ví dụ: `customerId`, `phone`) bằng cách sử dụng `$and` để kết hợp nếu cần, hoặc gộp chúng một cách chính xác.

2. **Kiểm tra**:
   - Xác minh rằng bộ lọc tạo ra đúng cấu trúc truy vấn MongoDB mong đợi.
   - Kiểm tra chéo các cột trong `getConfig` với độ dài mảng đầu ra của `transformRow`.
   - Chạy `npm run build` để đảm bảo không có lỗi phát sinh.

## User Review Required
> [!IMPORTANT]
> Logic `$or` mới sẽ thay đổi cách gộp filter. Nếu record `REFUND_CASHIN` có `sysTrans: true` nhưng User truyền `sysTrans: false`, record đó vẫn sẽ xuất hiện. Đây là hành vi mong muốn để User thấy đủ bộ "Giao dịch chuyển khoản nội bộ".

---

## Phase 2: Fix Missing Filters (2026-04-10)

### Root Cause
CMS gửi 3 params mà `buildTransHisFilter` không handle:
- `merchantPaymentOriginalTransactionHisId` → bị ignore hoàn toàn
- `senderAccount` → bị ignore hoàn toàn
- `receiverAccount` → bị ignore hoàn toàn

### Plan
**File**: `trans-his-collection-export.pure.ts` → `buildTransHisFilter`

~~Phase 2 REVERTED — xem Phase 3~~

### Verification
- `npm run build` pass
- Columns/transformRow parity không bị ảnh hưởng (chỉ sửa filter)

---

## Phase 3: Tạo InternalTransferExport mới (2026-04-10)

> Phase 2 đã REVERT. Tạo export processor mới thay vì sửa TransHisCollectionExport.

### Step 1: Revert TransHisCollectionExport
- Bỏ 3 filter thêm ở Phase 2 (`merchantPaymentOriginalTransactionHisId`, `senderAccount`, `receiverAccount`)

### Step 2: Tạo `internal-transfer-export.pure.ts`
- `buildFilter(params)`: Filter chuyên cho INTERNAL_BANK_TRANSFER
  - Mặc định `transType: INTERNAL_BANK_TRANSFER`
  - Handle: transId, merchantPaymentOriginalTransactionHisId, status, partnerCode, senderAccount→sender.bankAccount, receiverAccount→receiver.credit, connector→info.bankConnector, dateFr/dateTo, sysTrans (bypass cho INTERNAL_BANK_TRANSFER via $or)
- `getConfig(params, langCode)`: 19 columns, fileName = "Gd_ck_noi_bo"
- `transformRow(history, rowIndex, langCode)`: Mapping theo doc UC03.R01
- `validate(params, langCode)`: Reuse TransHisExportParams

### Step 3: Tạo `internal-transfer.export.ts`
- Thin adapter extends BaseExportProcessor, delegate to pure functions
- requiredModelName: ["TransHisModel"]

### Step 4: Register
- `logics/export/index.ts`: export pure
- `logics/index.ts`: import + export adapter class

### Step 5: Build + Verify
- `npm run build`
- Verify 19 columns ↔ transformRow array length

---

## Phase 3.1: Fix filter patterns theo reference code (2026-04-10)

### Root Cause
`buildFilter` trong `internal-transfer-export.pure.ts` dùng logic tự chế thay vì follow pattern từ service gốc (`trans-his.manage.logic.ts`).

### Fixes cần làm
1. **sysTrans**: Bỏ `$or` bypass sai → đổi sang `filter.sysTrans = params.sysTrans` (đơn giản vì đã hardcode transType=INTERNAL_BANK_TRANSFER)
2. **connector**: Đổi `filter["info.connector"]` → `$expr` + `$ifNull` check cả `$info.bankConnector` lẫn `$info.connector` (theo pattern gốc)
3. **senderAccount**: Đổi exact match → `$expr` + `$regexMatch` + `$ifNull` trên `$info.bankAccountNumber` và `$sender.bankAccount` (theo pattern gốc)
4. **receiverAccount**: Đổi exact match → `$regex` trên `receiver.credit` (theo pattern gốc)
5. **originalTransHisId**: Đổi exact match → `$regex` + `$options: "i"` (theo pattern gốc)

---

## Phase 3.2: Fix transformRow mapping bugs (2026-04-10)

### Bugs phát hiện từ actual DB record
Verified với record `transId: 260409073353ZVCWXT`:

1. **connector**: `info.bankConnector` không tồn tại ở `info` root → dùng `info.connector` (fallback `info.bankTransferData.bankConnector`)
2. **transType label**: `INTERNAL_BANK_TRANSFER` không có entry trong `TRANSACTION_TYPE_LIST` → cần hardcode label
3. **originalInternalBankTransId**: Typo `info?.tranHis?` → phải là `info?.transHis?` (data: `info.transHis.originalInternalBankTransId`)
4. **senderBankName**: `sender.bankName` không tồn tại → dùng `sender.bankCode` (data: "BIDV")
5. **receiverBankName**: `receiver.bankName` không tồn tại → dùng `receiver.bankCode` (data: "bidv")

---

## Phase 3.3: Port đầy đủ transType + sysTrans logic (2026-04-10)

### Root Cause
Chỉ copy nửa vời logic từ reference code. Thiếu 2 phần quan trọng:

1. **transType override**: Khi CMS gửi `params.transType`, phải override default `$in` array
2. **$or bypass cho REFUND_CASHIN + sysTrans**: Khi `sysTrans === "true"` và có REFUND_CASHIN trong transType, tạo `$or` để:
   - Branch 1: REFUND_CASHIN không bị ràng buộc sysTrans
   - Branch 2: Các type còn lại giữ nguyên sysTrans filter

### Reference pattern (từ service gốc):
```typescript
if (params.transType) {
    const arrayTransType = params.transType.split(",");
    filter.transType = { $in: arrayTransType };
}
// ... other filters ...
if ("sysTrans" in params) {
    filter.sysTrans = params.sysTrans;
}
// resolve REFUND_CASHIN + sysTrans conflict
const arrTransTypes = params.transType?.split(",") || [];
const hasRefundCashin = arrTransTypes.includes(TRANS_TYPE.REFUND_CASHIN);
const otherTransTypes = arrTransTypes.filter(t => t !== TRANS_TYPE.REFUND_CASHIN);
const { sysTrans, ...restFilter } = filter;
if ("transType" in params && hasRefundCashin && "sysTrans" in filter && sysTrans === "true") {
    filter.transType = { $in: otherTransTypes };
    const newFilter = {$or: [
        { ...restFilter, transType: TRANS_TYPE.REFUND_CASHIN },
        { ...filter }
    ]};
    filter = _.cloneDeep(newFilter);
}
```

### Cần import thêm: `lodash` cho `_.cloneDeep`
