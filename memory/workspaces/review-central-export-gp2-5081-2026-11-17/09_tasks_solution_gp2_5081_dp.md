# 09. Design Pattern & Clean Architecture Audit + Refactoring Proposal — `feat/GP2-5081 -> develop` (`7f4e9a7`)

- **Workspace:** `review-central-export-gp2-5081-2026-11-17`
- **Suffix:** `gp2_5081_dp`
- **Tier:** `T2` (`mode: audit-propose-only`)
- **Commit được kiểm toán:** `7f4e9a7e149dabd75674f9fb3737c81bbfa21794` (`origin/feat/GP2-5081` → `origin/develop` `3eef807`)
- **Ràng buộc Thực thi (Technical Constraints từ Global Lessons):**
  - **TC1 (`L001` — Mandatory Design Pattern, Layer Dependency & Template Hook Audit In Code Review):** Đảm bảo chiều phụ thuộc một chiều (`domain/` cấm import ngược `logics/`), tính bất biến của `IQuery`, trách nhiệm đơn mô hình của `IQueryHandler`, và tận dụng đúng móc mở rộng Template Method của `BaseExportProcessor`.
  - **TC2 (`G2` / Rule #1A, #13):** *Propose-Only* — không sửa bất kỳ file source code nào trước khi anh `APPROVE`.
  - **TC3 (`L077` — Partial Index Alignment):** Kết hợp luôn `isDelete: { $ne: true }` (`N1`) và guard `error?.code` (`N3`).

---

## 1. Kiến Trúc Chuẩn (DNA) Của `centralized-export-service`

Đối chiếu `README.md:11,104-118`, `BaseExportProcessor.ts`, và toàn bộ `37` handlers + `13` exporters đa mô hình hiện có trên `origin/develop`, dự án vận hành trên **4 mẫu thiết kế cốt lõi**:
1. **Clean Architecture (Unidirectional Layer Dependency):** `workers / start` → `logics/` (Application Layer: `*Export` adapter + `*ExportPure` pure functions) → `domain/` (CQRS Layer: `IQuery`, `IQueryHandler`, `QueryBus`, `QueryHandlerFactory`) → `models/` & `data-transfers/`. Trên toàn bộ `origin/develop`, **`domain/` tuyệt đối KHÔNG import bất kỳ file nào từ `logics/`** (`0` vi phạm trong 37 handlers cũ).
2. **Template Method + Hook Methods (`BaseExportProcessor.ts:31-35,158,219-239`):**
   - **Pre-Filter Hook (`async getConfig(..., queryExecutor)`):** Khi bộ lọc của bảng chính cần tra cứu ID từ một DB khác trước khi phân trang (như `PayoutRequestExport` lọc theo `merchantAccount` bên `MerchantModel` tại `payout-request.export.ts:14-18` & `payout-request-export.pure.ts:22-31`), `getConfig` nhận `queryExecutor` gọi qua `QueryBus` **1 lần duy nhất** trước vòng lặp SAGA.
   - **Per-Chunk Sub-Query Hook (`subQueryClass` + `buildSubQueryParams` + `mergeData`):** Toàn bộ **13/13 exporters** cần lấy thêm trường từ model/DB thứ hai (`payout-request-export.pure.ts:54-70`, `payment-history-export.pure.ts:135-160`, `trans-his-export.pure.ts:213-233`, `merchant-export.pure.ts:63`, `customer-export.pure.ts:66`, `reconciliation-report-export.pure.ts:60`...) đều khai báo 3 hook này trong `ExportConfig` để `BaseExportProcessor` điều phối qua `QueryBus` (có sẵn `try/catch` cô lập lỗi tại `BaseExportProcessor.ts:236-238`).
3. **CQRS Stateless Query DTO (`domain/queries/**`):** Toàn bộ `32` class `*Query` hiện có là Value Object bất biến, chỉ mang tham số đầu vào, không lưu state/cache kết quả DB lúc chạy.
4. **Generic Entity Handler Reuse + Auxiliary Handler Pattern:**
   - Cả **3/3 exporters** sẵn có trong `logics/export/trans-his/` (`TransHisExport`, `TransHisCollectionExport`, `InternalTransferExport`) đều **tái sử dụng chung 1 cặp `GetAllTransHisExportQuery` → `GetAllTransHisExportHandler` / `GetAllTransHisExportPgHandler`**, không tạo handler `trans-his` riêng cho từng màn hình!
   - Các truy vấn phụ trợ được đặt đúng bounded context của model đích với hậu tố `*AuxiliaryQuery` / `*AuxiliaryHandler` hoặc `GetList*ByIdsQuery` (ví dụ `GetPaymentHistoryExportAuxiliaryHandler`, `GetMerchantExportAuxiliaryHandler`, `GetListTransHisByTransIdHandler`).

---

## 2. Bảng Tổng Hợp 5 Vi Phạm Design Pattern Tại `feat/GP2-5081` (`DP1` → `DP5`)

| Mã | Mức độ | Trạng thái | Vi phạm Design Pattern & Bằng chứng `file:line` |
| :--- | :---: | :---: | :--- |
| **DP1** | **P1 (High)** | `VERIFIED` | **Vi phạm Clean Architecture — Đảo chiều phụ thuộc tầng (`domain/` import ngược `logics/`):**<br>- `domain/handlers/trans-his/GetAllPartnerCollectionTransExportHandler.ts:5` và `domain/handlers/trans-his/partner-collection-trans-bills.ts:4` đều `import { billIdVariants } from "../../../logics/export/trans-his/partner-collection-trans-export.pure"`, trong khi `partner-collection-trans-export.pure.ts:5` lại import `domain/queries`. Đây là **2 file duy nhất trong toàn bộ `domain/`** nhập ngược từ `logics/`, tạo vòng phụ thuộc hai chiều `domain <-> logics`. |
| **DP2** | **P1 (High)** | `VERIFIED` | **Phá vỡ Template Method của `BaseExportProcessor` & Ranh giới Bounded Context (Cross-DB Join nhét cứng vào Handler `trans-his`):**<br>- Thay vì dùng `async getConfig(..., queryExecutor)` + `subQueryClass` / `buildSubQueryParams` / `mergeData` như 13 exporters chuẩn của dự án, `partner-collection-trans-export.pure.ts:99-125` bỏ trống toàn bộ các hook này và nhét thẳng logic truy vấn `PaymentBillModel` (thuộc DB Mongo `"payment-bill"`) vào bên trong handler của DB `"trans-his"` / `"trans-his-pg"` (`GetAllPartnerCollectionTransExportHandler.ts:35-55`, `GetAllPartnerCollectionTransExportPgHandler.ts:19-31`). |
| **DP3** | **P1 (High)** | `VERIFIED` | **Vi phạm tính bất biến của CQRS `IQuery` (Stateful Mutable Query Anti-Pattern):**<br>- Để vá việc không dùng `async getConfig(..., queryExecutor)`, `domain/queries/trans-his/GetAllPartnerCollectionTransExportQuery.ts:9` khai báo biến trạng thái khả biến `public resolvedBillIds?: (string | number)[]` và cho `applyBillFilter` (`partner-collection-trans-bills.ts:30`) tự ghi đè `query.resolvedBillIds = ...` ngay trong lúc thực thi handler. |
| **DP4** | **P1 (High)** | `VERIFIED` | **Phình đại Handler trùng lặp (Handler Proliferation) thay vì tái sử dụng `GetAllTransHisExportQuery` + Auxiliary Handler:**<br>- Trong khi `TransHisExport`, `TransHisCollectionExport` và `InternalTransferExport` đều tái sử dụng `GetAllTransHisExportQuery`, nhánh này tạo mới 4 file riêng trong `domain/*/trans-his/` (`GetAllPartnerCollectionTransExportQuery.ts`, `GetAllPartnerCollectionTransExportHandler.ts`, `GetAllPartnerCollectionTransExportPgHandler.ts`, `partner-collection-trans-bills.ts`) chỉ để bọc lại chính `TransHisModel.find` / `TransHisPgModel.getPage` cộng thêm phần gọi `PaymentBillModel`. |
| **DP5** | **P2 (Medium)** | `VERIFIED` | **Lệch chữ ký Pure Function Contract & thiếu `chunkSize` chuẩn cho Sub-Query:**<br>- `partner-collection-trans-export.pure.ts:134` đổi chữ ký thành `transformRow(item: any, langCode: string = "vi")` (2 tham số) lệch khỏi chuẩn 3 tham số `transformRow(item, rowIndex, langCode)` của toàn bộ các `*ExportPure` anh em; đồng thời chưa đặt `chunkSize: 500` như `trans-his-export.pure.ts:258`, `payment-history-export.pure.ts:198` và `payout-request-export.pure.ts:89` để giới hạn kích thước mảng `$in` của mỗi lần tra cứu đơn hàng. |

---

## 3. Phương Án Tái Cấu Trúc Duy Nhất Đúng Chuẩn Design Pattern Dự Án (Kèm `N1` & `N3`)

> Ánh xạ 1-1 giải quyết đồng thời cả `DP1`–`DP5` lẫn `N1`/`N3`, giúp **xóa bỏ 4 file dư thừa/sai tầng trong `domain/*/trans-his/`** và chỉ thêm 1 cặp Auxiliary chuẩn CQRS bên `domain/*/payment-bill/`:

1. **Tạo cặp `GetPaymentBillExportAuxiliaryQuery` + `GetPaymentBillExportAuxiliaryHandler` đúng Bounded Context `payment-bill` (Đóng `DP1`, `DP2`, `DP4`):**
   - Đặt tại `domain/queries/payment-bill/GetPaymentBillExportAuxiliaryQuery.ts` (bất biến, nhận `{ billIds?: number[]; merchantTransId?: string; limit?: number }`) và `domain/handlers/payment-bill/GetPaymentBillExportAuxiliaryHandler.ts` (chỉ đọc `PaymentBillModel`, không chạm `TransHis`, không import `logics/`). Có thể tái sử dụng cho mọi màn hình export khác cần lấy thông tin đơn hàng gốc!
2. **Tái sử dụng `GetAllTransHisExportQuery` cho bảng chính & thêm `_id` tie-breaker vào `GetAllTransHisExportHandler` chung (Đóng `DP4`):**
   - Xóa 4 file `GetAllPartnerCollectionTransExportQuery.ts`, `GetAllPartnerCollectionTransExportHandler.ts`, `GetAllPartnerCollectionTransExportPgHandler.ts`, `partner-collection-trans-bills.ts`.
   - Nâng cấp 2 dòng `sortWithTieBreaker` ngay trong `GetAllTransHisExportHandler.ts` để toàn bộ các export `trans-his` trên Mongo đều được phân trang tất định.
3. **Sử dụng đúng 2 Hook của `BaseExportProcessor` trong `PartnerCollectionTransExport` & `partner-collection-trans-export.pure.ts` (Đóng `DP2`, `DP3`, `DP5`, `N1`, `N3`):**
   - **Pre-Filter Hook:** `PartnerCollectionTransExport.getConfig()` là `async`, truyền `queryExecutor` vào `PartnerCollectionTransExportPure.getConfig(params, langCode, queryExecutor)` (giống hệt `PayoutRequestExport` tại `payout-request.export.ts:14-18`). Khi có `merchantTransId`, `getConfig` gọi `queryExecutor(new GetPaymentBillExportAuxiliaryQuery({ merchantTransId, limit: 1000 }))` **1 lần duy nhất** trước SAGA loop để dựng `filter.$and` — xóa sạch biến mutable `resolvedBillIds` trên `IQuery`!
   - **Per-Chunk Enrichment Hook:** Khai báo `subQueryClass: GetPaymentBillExportAuxiliaryQuery`, `buildSubQueryParams`, `mergeData`, và `chunkSize: 500` ngay trong `ExportConfig` (giống hệt `PaymentHistoryExport` & `PayoutRequestExport`).
   - Chuẩn hóa chữ ký `transformRow(item: any, _rowIndex: number, langCode: string = "vi")`, thêm `isDelete: { $ne: true }` vào `baseFilter()` + `trans-his-pg.query.ts` (`N1`), và guard `error?.code` (`N3`).

---

## 4. Code Demo Đầy Đủ Theo Chuẩn Design Pattern Dự Án

### 4.1. `domain/queries/payment-bill/GetPaymentBillExportAuxiliaryQuery.ts` (Stateless `IQuery`)
```typescript
import { IQuery } from "../../IQuery";

export interface PaymentBillAuxiliaryParams {
    readonly billIds?: number[];
    readonly merchantTransId?: string;
    readonly limit?: number;
}

export class GetPaymentBillExportAuxiliaryQuery implements IQuery {
    public readonly billIds?: number[];
    public readonly merchantTransId?: string;
    public readonly limit?: number;

    public constructor(params: PaymentBillAuxiliaryParams = {}) {
        this.billIds = params.billIds;
        this.merchantTransId = params.merchantTransId;
        this.limit = params.limit;
    }
}
```

### 4.2. `domain/handlers/payment-bill/GetPaymentBillExportAuxiliaryHandler.ts` (Single-Model Handler, Zero `logics/` Import)
```typescript
import * as _ from "lodash";
import { IQueryHandler } from "../../IQueryHandler";
import { GetPaymentBillExportAuxiliaryQuery } from "../../queries";
import MainProcess from "../../../data-transfers/main-process";

const DEFAULT_LOOKUP_LIMIT = 1000;

export class GetPaymentBillExportAuxiliaryHandler implements IQueryHandler {
    private readonly mainProcess: MainProcess;

    public constructor(mainProcess: MainProcess) {
        this.mainProcess = mainProcess;
    }

    public async execute(query: GetPaymentBillExportAuxiliaryQuery): Promise<{ data: any[] }> {
        const Bill = this.mainProcess.models.PaymentBillModel;
        const trimmedMtId = query.merchantTransId?.trim();

        if (trimmedMtId) {
            const cap = query.limit || DEFAULT_LOOKUP_LIMIT;
            const bills = await Bill.find({
                merchantTransId: { $regex: _.escapeRegExp(trimmedMtId), $options: "i" }
            })
                .select({ _id: 1 })
                .limit(cap)
                .lean()
                .exec();
            if ((bills || []).length >= cap) {
                this.mainProcess.logger?.warn(
                    `[GetPaymentBillExportAuxiliaryHandler] merchantTransId "${trimmedMtId}" reached limit ${cap}`
                );
            }
            return { data: bills || [] };
        }

        if (!query.billIds || query.billIds.length === 0) {
            return { data: [] };
        }

        const bills = await Bill.find({ _id: { $in: query.billIds } })
            .select({ "_id": 1, "merchantTransId": 1, "embedData.transactionType": 1 })
            .lean()
            .exec();
        return { data: bills || [] };
    }
}
```

### 4.3. `logics/export/trans-his/partner-collection-trans.export.ts` (Đúng chuẩn `PayoutRequestExport` Adapter)
```typescript
import { RequestHelper } from "@goopay/goopay-library/helpers";
import { BaseExportProcessor, ExportConfig } from "../../processors/BaseExportProcessor";
import { transHisModelNames } from "../../../utils/trans-his-export-source";
import { PartnerCollectionTransExportPure } from "../index";

export class PartnerCollectionTransExport extends BaseExportProcessor {
    private rowIndex: number = 0;

    protected validate(): void {
        const langCode = RequestHelper.getLangCode(this.context);
        PartnerCollectionTransExportPure.validate(this.context.params, langCode);
    }

    protected async getConfig(): Promise<ExportConfig> {
        const langCode = RequestHelper.getLangCode(this.context);
        const queryExecutor = async (query: any) => this.queryBus.execute(query, this.context, {}, this.mainProcess);
        return PartnerCollectionTransExportPure.getConfig(this.context.params, langCode, queryExecutor);
    }

    protected transformRow(item: any, langCode: string): any[] {
        return PartnerCollectionTransExportPure.transformRow(item, ++this.rowIndex, langCode);
    }

    protected getRequiredModelName(): string[] {
        return [...transHisModelNames(), "PaymentBillModel"];
    }
}

export default PartnerCollectionTransExport;
```

### 4.4. `logics/export/trans-his/partner-collection-trans-export.pure.ts` (Tái sử dụng `GetAllTransHisExportQuery` + `subQueryClass` / `mergeData`)
```typescript
export const baseFilter = (): Record<string, any> => ({
    cdFlow: COLLECTION_FLOW,
    bizType: { $nin: [...EXCLUDED_BIZ_TYPES] },
    isDelete: { $ne: true }
});

export const getConfig = async (
    params: any,
    langCode: string = "vi",
    queryExecutor?: (q: any) => Promise<any>
): Promise<ExportConfig> => {
    const validated = validate(params, langCode);
    const filter = buildFilter(validated);
    const sort = buildSort(validated);

    if (!isEmptyValue(validated.merchantTransId) && queryExecutor) {
        const auxRes = await queryExecutor(
            new GetPaymentBillExportAuxiliaryQuery({
                merchantTransId: String(validated.merchantTransId).trim(),
                limit: BILL_LOOKUP_LIMIT
            })
        );
        const matched = (auxRes?.data || []).flatMap((b: any) => billIdVariants(b._id));
        filter.$and = [...(filter.$and || []), { "info.billId": { $in: matched } }];
    }

    return {
        query: new GetAllTransHisExportQuery(filter, sort, SELECT_FIELDS),
        subQueryClass: GetPaymentBillExportAuxiliaryQuery,
        buildSubQueryParams: (primaryData: any[]) => ({
            billIds: _.uniq(
                (primaryData || [])
                    .map(d => d?.info?.billId)
                    .filter(v => v !== undefined && v !== null && v !== "")
            ).map(v => (Number.isNaN(Number(v)) ? v : Number(v)))
        }),
        mergeData: (primaryData: any[], bills: any[]) => {
            const billMap = new Map<string, any>((bills || []).map(b => [String(b._id), b]));
            return (primaryData || []).map(doc => {
                const bill = billMap.get(String(doc?.info?.billId));
                return {
                    ...doc,
                    merchantTransId: bill?.merchantTransId || "",
                    transactionType: bill?.embedData?.transactionType || ""
                };
            });
        },
        columns: [ /* giữ nguyên 15 cột template UC02 */ ],
        fileName: {
            vi: moment().format("DD-MM-YYYY") + "_Giao_dich_thu_ho",
            en: moment().format("DD-MM-YYYY") + "_Collection_payments"
        },
        chunkSize: 500,
        maxRecords: 2_000_000
    };
};

export const transformRow = (item: any, _rowIndex: number, langCode: string = "vi"): any[] => {
    // Giữ chữ ký 3 tham số chuẩn của toàn bộ *ExportPure + guard error?.code
    ...
};
```
