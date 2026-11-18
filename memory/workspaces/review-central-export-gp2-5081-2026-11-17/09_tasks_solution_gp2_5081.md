# 09. Solution & Code Review Proposal — `feat/GP2-5081 -> develop` (`centralized-export-service`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Commit được review:** `2304b3b9a2fa80d23abdc9adf58e7634fbdc8ac6` (`origin/feat/GP2-5081`) → `3eef80790b319e4d0f188e3d12cef9f0ac90b302` (`origin/develop`, 14 files changed, `+921 / -1`, 0 commit lệch sau `origin/develop`).
- **Kết quả S2.5 Pre-Plan Critique (`DG0`):** `gate = PASS` (`06_laya_summary_dg0_20261117T113040.json`, VegaML `:8222` `r3 = YES` với confidence `0.827–0.875`, `trusted_rounds = []` → `UNVALIDATED`, kết luận dựa trên đối chiếu trực tiếp source code `file:line`).
- **Ràng buộc Thực thi (Technical Constraints từ Global Lessons):**
  - **TC1 (`G2` / Rule #1A, #13):** Chế độ *Propose-Only* — không sửa bất kỳ file source code nào trước khi anh `APPROVE`.
  - **TC2 (`L074` — Zero Full-Table Scans & Bounded Queries):** Cấm quét regex không neo đầu thiếu `@MinLength(3)` hoặc lặp lại truy vấn tra cứu đơn hàng qua từng trang; bắt buộc chặn `merchantTransId` rỗng/khoảng trắng để không sinh `$regex: ""` quét toàn bảng `payment-bills`.
  - **TC3 (`L078` — Type Coercion & Timezone Consistency):** Cấm ép kiểu chuỗi khoảng trắng `"   "` qua `Number("   ") === 0`; giữ đồng nhất chuyển đổi múi giờ `+07:00` và kiểm tra `Number.isSafeInteger` khi sinh biến thể `billId`.
  - **TC4 (`G3` & `G4`):** Kiểm định đầy đủ qua Decision Gate (`DG0`, `DG1`, `DG3`) và lấy biên lai test/typecheck/lint thật qua `evidence_run.py`.

---

## 1. Bảng Tổng Hợp Các Phát Hiện (`F1` → `F6`)

| Mã | Mức độ | Trạng thái | Tóm tắt phát hiện & Bằng chứng `file:line` |
| :--- | :---: | :---: | :--- |
| **F1** | **P0 (Critical)** | `VERIFIED` | **Lệch nguồn dữ liệu (`TRANS_HIS_EXPORT_SOURCE=postgres`) & lệch logic lọc/hiển thị so với API danh sách v2 (`GET /ops/collection-disbursement/collections`):**<br>1. `domain/QueryHandlerFactory.ts:155-163` & `logics/export/trans-his/partner-collection-trans.export.ts:23-25` gắn cứng vào Mongo `TransHisModel`, bỏ qua nhánh `getTransHisExportSource() === "postgres"` trong khi hệ thống `trans-his` đang/đã chuyển sang Postgres (`core-trans-proxy-history-service-v2`).<br>2. `logics/export/trans-his/partner-collection-trans-export.pure.ts:23-27,36,80-82` ép cứng `serviceCode: "COLLECTION_DISBURSEMENT"` ở cả `baseFilter()` lẫn `billFilter`. Hệ quả: (a) nếu truyền `params.serviceCode` khác `"COLLECTION_DISBURSEMENT"`, `filter.serviceCode` xung đột với `$and: [{ serviceCode }]` luôn trả về `0` dòng; (b) các đơn thu hộ không mang `serviceCode = "COLLECTION_DISBURSEMENT"` (như đơn RADA có `serviceCode = null` theo chính `transHis.collectionOps.logic.ts:117` ở `GP2-5069`) bị rơi khỏi file xuất.<br>3. `GetAllPartnerCollectionTransExportHandler.ts:7-16` thiếu `receiver: 1` trong `SELECT_FIELDS` và `partner-collection-trans-export.pure.ts:114,127-128` không fallback sang `item.receiver?.merchantCode` / `item.receiver?.fullName` như API danh sách (`transHis.collectionOps.logic.ts:170-175`), làm cột Mã/Tên bên thụ hưởng bị trống khi bản ghi không có `info.merchantInfo`. |
| **F2** | **P1 (High)** | `VERIFIED` | **Quét `Bill.find` bằng `$regex` không giới hạn & bị chạy lặp lại ở MỖI chunk phân trang:**<br>- `data-transfers/params/trans-his/partner-collection-trans-export.params.ts:43-44,52`: thiếu `@MinLength(3)` trên các trường tìm kiếm regex (`merchantTransId`, `merchantEmail`, `payeeCode`) — trái với chuẩn `CollectionDisbursementOrdersExportParams` (`collection-disbursement-orders-export.params.ts:63-95`).<br>- `domain/handlers/trans-his/GetAllPartnerCollectionTransExportHandler.ts:38-49`: `Bill.find({ merchantTransId: { $regex: ..., $options: "i" } })` không có `.limit()`, làm mất tác dụng index B-Tree `{ merchantTransId: 1 }` (Full Collection Scan khi chuỗi ngắn) và có nguy cơ tạo mảng `$in` khổng lồ chạm ngưỡng 16MB BSON. Đồng thời, do `BaseExportProcessor.ts:209` gọi `handler.execute()` ở mỗi chunk `5000` dòng, câu lệnh `Bill.find` này bị thực thi lặp lại $N$ lần vô ích cho cùng 1 job export. |
| **F3** | **P1 (High)** | `VERIFIED` | **Phân trang `.skip(cursor)` thiếu tie-breaker `_id` trong `sort` & suy thoái hiệu năng $O(N^2)$ khi `maxRecords = 2_000_000`:**<br>- `logics/export/trans-his/partner-collection-trans-export.pure.ts:58-65` trả về `{ [field]: asc ? 1 : -1 }` hoặc `{ createdAt: -1 }` không kèm `_id: -1` (hoặc `1`). Khi `sortBy` là `status`, `value`, hoặc nhiều giao dịch trùng mili-giây `createdAt`, `.skip(cursor).limit(limit)` tại `GetAllPartnerCollectionTransExportHandler.ts:51-55` không đảm bảo thứ tự tất định giữa các chunk → gây trùng hoặc mất dòng trong file Excel đối soát. Ngoài ra, `.skip()` tới `2_000_000` dòng (`pure.ts:106`) buộc MongoDB quét bỏ hàng trăm triệu bản ghi cộng dồn. |
| **F4** | **P1 (High)** | `VERIFIED` | **Lỗi biên Validation khoảng ngày (`dateTo < dateFr`) và chuỗi toàn khoảng trắng (`"   "`):**<br>1. `partner-collection-trans-export.params.ts:23-24`: chỉ kiểm tra `dateDiff <= MAX_DAYS_EXPORT` mà thiếu `dateDiff >= 0`, nên `dateTo < dateFr` (`dateDiff < 0`) vẫn pass validation, tạo job rác trên Redis/SAGA.<br>2. `partner-collection-trans-export.pure.ts:19`: `isEmptyValue` không `.trim()` chuỗi:<br>   - Với `billId = "   "`, `Number("   ") === 0` → lọc sai thành `{ "info.billId": { $in: [0, "0"] } }` (`pure.ts:49-50`).<br>   - Với `merchantTransId = "   "`, `billFilter.merchantTransId` vượt qua check ở `pure.ts:80`, vào `Handler.ts:40` bị `.trim()` thành `$regex: ""` → **quét sạch toàn bộ bảng `payment-bills`**!<br>   - Với `status = "SUCCESS, FAILED"`, `.split(",")` không `.trim()`/`.filter(Boolean)` sinh phần tử `" FAILED"` không khớp DB.<br>3. `partner-collection-trans-export.params.ts:59`: `@IsString()` trên `sortType` chặn `sortType: 1 | -1` kiểu `number`, làm nhánh `params.sortType === 1` ở `pure.ts:61` thành dead code. |
| **F5** | **P2 (Medium)** | `VERIFIED` | **3 lỗi ESLint làm fail `eslint` (`EV-20261117T112415-280e`, `exit=1`) & `safeText` thiếu ký tự `\n`:**<br>- `GetAllPartnerCollectionTransExportHandler.ts:19`: dùng `Array<string | number>` vi phạm `@typescript-eslint/array-type`.<br>- `partner-collection-trans-export.pure.ts:24,26`: thiếu quote quanh `"transType"` và `"serviceCode"` vi phạm `quote-props`.<br>- `partner-collection-trans-export.pure.ts:70`: regex chống CSV/Excel Formula Injection `/^[=+\-@\t\r]/` thiếu `\n` (`/^[=+\-@\t\r\n]/`). |
| **F6** | **P2 (Medium)** | `VERIFIED` | **Unit Test mock mất `validateParams` thật & Cảnh báo xung đột merge với nhánh song song `feat/GP2-4633`:**<br>- `test/unit/pure/partner-collection-trans.pure.test.ts:8-10`: `vi.mock("../../../utils/validator", ...)` trả thẳng `params`, khiến `Pure.validate()` / `Pure.getConfig()` trong unit test không thực sự chạy `validateParams` thật.<br>- Lưu ý tích hợp nhánh: nhánh `origin/feat/GP2-4633` (`PartnerUserReserveExport`) sửa cùng dòng cuối ở 6 file barrel/factory (`params/index.ts`, `queries/index.ts`, `QueryHandlerFactory.ts`, `logics/export/index.ts`, `logics/index.ts`, `QueryHandlerFactory.test.ts`), cần rebase giữ cả hai export khi merge nối tiếp vào `develop`. |

---

## 2. Phương Án Khắc Phục Duy Nhất Tối Ưu Nhất (Ánh Xạ 1-1 `F1` → `F6`)

> Thiết kế tuân thủ **Simplicity First, Minimal Impact** và giữ nguyên kiến trúc Thin-Adapter + Pure Functions + CQRS Handler của `centralized-export-service`:

1. **Khắc phục `F1` (Đồng bộ logic lọc, fallback `receiver` & sẵn sàng nguồn `trans-his`):**
   - Trong `partner-collection-trans-export.pure.ts`:
     - Khi `params.serviceCode` được truyền (sau `.trim()`), ghi đè trực tiếp `filter.serviceCode = trimmedServiceCode` thay vì đẩy thêm vào `$and` gây mâu thuẫn với `baseFilter()`.
     - Gỡ `serviceCode: COLLECTION_DISBURSEMENT` khỏi `billFilter` (vì đơn RADA không gắn `serviceCode` trên `payment-bills` theo `transHis.collectionOps.logic.ts:117`, trong khi bản thân `TransHis` đã lọc giao dịch thu hộ).
     - Bổ sung `receiver: 1` vào `SELECT_FIELDS` và fallback `payee?.code ?? item.receiver?.merchantCode`, `payee?.name ?? item.receiver?.fullName`.
     - Đối với `TRANS_HIS_EXPORT_SOURCE=postgres`: xác nhận cùng anh xem môi trường `develop`/`uat`/`prod` đã bật `TRANS_HIS_EXPORT_SOURCE=postgres` và migration `011` (`cd_flow = 'COLLECTION'`) chưa; nếu đã bật, bổ sung `GetAllPartnerCollectionTransExportPgHandler` đăng ký tại `QueryHandlerFactory.ts` và đổi `getRequiredModelName()` sang `[...transHisModelNames(), "PaymentBillModel"]`.
2. **Khắc phục `F2` (Chặn Full-Scan & Cache kết quả tra cứu `merchantTransId` một lần cho cả Job):**
   - Thêm `@MinLength(3, { message: "ERR_MIN_LENGTH_3" })` cho `merchantTransId`, `merchantEmail`, `payeeCode` trong `PartnerCollectionTransExportParams`.
   - Cache `query.resolvedBillIds` ngay ở chunk đầu tiên kèm `.limit(5000)` để các chunk sau tái sử dụng tức thì mà không query lại `PaymentBillModel`, đồng thời tránh tràn 16MB BSON.
3. **Khắc phục `F3` (Phân trang tất định bằng Keyset Cursor `(sortValue, _id)` + Tie-Breaker Sort):**
   - Bổ sung `_id: dir` vào `buildSort` (`{ [field]: dir, _id: dir }` và mặc định `{ createdAt: -1, _id: -1 }`).
   - Hỗ trợ Keyset Cursor `{ field, dir, lastVal, lastId }` trong `GetAllPartnerCollectionTransExportHandler` (giữ tương thích ngược nếu `typeof cursor === "number"`), triệt tiêu lỗi trùng/mất dòng khi có giao dịch mới và đưa chi phí phân trang 2 triệu dòng từ $O(N^2)$ về $O(N)$.
4. **Khắc phục `F4` (Chuẩn hóa Validation khoảng ngày & Trim tham số rỗng):**
   - Sửa `ExceedDateRangeValidator.validate`: kiểm tra `!to.isBefore(from) && dateDiff >= 0 && dateDiff <= MAX_DAYS_EXPORT`.
   - Sửa `isEmptyValue` kiểm tra `.trim() === ""`; gom dùng chung 1 hàm `billIdVariants` có kiểm tra `Number.isSafeInteger`; `.split(",").map(s => s.trim()).filter(Boolean)` cho `status`; đổi `sortType?: string | number`.
5. **Khắc phục `F5` & `F6` (Sửa sạch 3 lỗi ESLint, mở rộng `safeText` & Test thực chất):**
   - Đổi `Array<string | number>` thành `(string | number)[]`, quote đồng nhất `"transType"` / `"serviceCode"`, mở rộng `safeText` sang `/^[=+\-@\t\r\n]/`.
   - Gỡ `vi.mock("../../../utils/validator")` trong `partner-collection-trans.pure.test.ts` để test trực tiếp `Pure.validate()` thật.

---

## 3. Code Demo Đề Xuất (Propose-Only — Chờ Anh `APPROVE`)

### 3.1. `data-transfers/params/trans-his/partner-collection-trans-export.params.ts`
```typescript
@ValidatorConstraint({ name: "ExceedDateRangePartnerCollectionTrans", async: false })
class ExceedDateRangeValidator implements ValidatorConstraintInterface {
    public validate(_value: any, args: ValidationArguments) {
        const params = args.object as any;
        if (!params.dateFr || !params.dateTo) {
            return true;
        }
        const from = moment(new Date(params.dateFr));
        const to = moment(new Date(params.dateTo));
        if (!from.isValid() || !to.isValid() || to.isBefore(from)) {
            return false;
        }
        const dateDiff = moment.duration(to.diff(from)).asDays();
        return dateDiff >= 0 && dateDiff <= MAX_DAYS_EXPORT;
    }

    public defaultMessage(args: ValidationArguments) {
        const params = args.object as any;
        if (params?.dateFr && params?.dateTo && moment(new Date(params.dateTo)).isBefore(moment(new Date(params.dateFr)))) {
            return "PARAMS_INVALID_FORMAT";
        }
        return "EXCEED_DATE_RANGE_EXPORT";
    }
}

export class PartnerCollectionTransExportParams {
    @IsNotEmpty({ message: "ERR_DATE_FR_REQUIRED" })
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public dateFr: string;

    @IsNotEmpty({ message: "ERR_DATE_TO_REQUIRED" })
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public dateTo: string;

    @IsOptional() @IsString() public transId?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public merchantTransId?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public merchantEmail?: string;

    @IsOptional() @IsString() public status?: string;

    @IsOptional()
    @IsString()
    @IsIn(["MERCHANT", "USER"], { message: "PARAMS_INVALID_FORMAT" })
    public payeeType?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public payeeCode?: string;

    @IsOptional() @IsString() public serviceCode?: string;
    @IsOptional() public billId?: string | number;
    @IsOptional() @IsString() public sortBy?: string;
    @IsOptional() public sortType?: string | number;

    @IsOptional()
    @Validate(ExceedDateRangeValidator)
    public get dateRangeCheck(): boolean {
        return true;
    }
}
```

### 3.2. `logics/export/trans-his/partner-collection-trans-export.pure.ts`
```typescript
export const isEmptyValue = (v: any): boolean =>
    v === undefined || v === null || (typeof v === "string" && v.trim() === "");

export const billIdVariants = (billId: string | number): (string | number)[] => {
    const raw = String(billId).trim();
    if (!raw) { return []; }
    const asNumber = Number(raw);
    return !Number.isNaN(asNumber) && Number.isSafeInteger(asNumber)
        ? _.uniq([asNumber, String(asNumber)])
        : [raw];
};

export const baseFilter = (): Record<string, any> => ({
    "transType": "PAY",
    "info.desc": AUGMENT_PROVISIONAL_REVENUE,
    "serviceCode": COLLECTION_DISBURSEMENT
});

export const buildFilter = (params: any): Record<string, any> => {
    const filter: Record<string, any> = baseFilter();
    const and: Record<string, any>[] = [];

    if (!isEmptyValue(params.transId)) { filter.transId = String(params.transId).trim(); }
    if (!isEmptyValue(params.status)) {
        const statuses = String(params.status).split(",").map(s => s.trim()).filter(Boolean);
        if (statuses.length > 0) { filter.status = { $in: statuses }; }
    }
    if (!isEmptyValue(params.serviceCode)) {
        filter.serviceCode = String(params.serviceCode).trim();
    }
    if (!isEmptyValue(params.merchantEmail)) { filter["info.masterMerchantInfo.email"] = like(params.merchantEmail); }
    if (!isEmptyValue(params.payeeCode)) { filter["info.merchantInfo.code"] = like(params.payeeCode); }
    if (params.payeeType === PAYEE_TYPE.USER) {
        filter["info.merchantInfo.code"] = { $exists: false };
    }
    if (!isEmptyValue(params.dateFr) || !isEmptyValue(params.dateTo)) {
        filter.createdAt = {};
        if (!isEmptyValue(params.dateFr)) { filter.createdAt.$gte = new Date(params.dateFr); }
        if (!isEmptyValue(params.dateTo)) { filter.createdAt.$lte = new Date(params.dateTo); }
    }
    if (!isEmptyValue(params.billId)) {
        const variants = billIdVariants(params.billId);
        if (variants.length > 0) {
            and.push({ "info.billId": { $in: variants } });
        }
    }
    if (and.length > 0) { filter.$and = and; }
    return filter;
};

export const buildSort = (params: any): Record<string, 1 | -1> => {
    const field = params?.sortBy ? SORTABLE[params.sortBy] : undefined;
    if (field && !isEmptyValue(params?.sortType)) {
        const dir: 1 | -1 = params.sortType === "asc" || params.sortType === "1" || params.sortType === 1 ? 1 : -1;
        return { [field]: dir, _id: dir };
    }
    return { createdAt: -1, _id: -1 };
};

export const safeText = (value: any): string => {
    const text = value === undefined || value === null ? "" : String(value);
    return /^[=+\-@\t\r\n]/.test(text) ? `'${text}` : text;
};
```

### 3.3. `domain/handlers/trans-his/GetAllPartnerCollectionTransExportHandler.ts`
```typescript
const BILL_SEARCH_LIMIT = 5000;
const SELECT_FIELDS = {
    transId: 1,
    value: 1,
    status: 1,
    noted: 1,
    info: 1,
    receiver: 1,
    serviceCode: 1,
    channelId: 1,
    createdAt: 1
};
...
        const filter: any = { ...query.filter };
        const trimmedMtId = billFilter?.merchantTransId?.trim();
        if (trimmedMtId) {
            if (!query.resolvedBillIds) {
                const bills = await Bill.find({
                    merchantTransId: { $regex: _.escapeRegExp(trimmedMtId), $options: "i" },
                    ...(billFilter?.serviceCode ? { serviceCode: billFilter.serviceCode } : {})
                })
                    .select({ _id: 1 })
                    .limit(BILL_SEARCH_LIMIT)
                    .lean()
                    .exec();
                query.resolvedBillIds = (bills || []).flatMap((b: any) => billIdVariants(b._id));
            }
            const matched = query.resolvedBillIds;
            filter.$and = [...(filter.$and || []), { "info.billId": { $in: matched.length > 0 ? matched : [null] } }];
        }
```
