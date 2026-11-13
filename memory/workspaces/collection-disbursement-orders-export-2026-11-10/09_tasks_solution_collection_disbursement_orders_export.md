# 09. Tasks Solution — Kiểm tra GetList & Kế hoạch Export cho `api/v2/admin-portal/ops/collection-disbursement/orders`

## 1. Báo cáo Kiểm tra Luồng GetList `GET /api/v2/admin-portal/ops/collection-disbursement/orders`

### 1.1. Định vị Nhánh & Luồng Kiến trúc Toàn trình (Cross-Repository Trace)
- **Tầng API Gateway (`admin-portal-gateway` — nhánh `origin/testing-1.12.0` / `origin/feature/GP2-5156-ops-api-base`, commit `4ea0d132` & `700b74a6`):**
  - `start.ts:74-76`: Whitelist `"v2.admin-portal.*"` dưới prefix `/api`.
  - `integrate-services/v2/start.ts:1-44`: Khai báo Moleculer service `admin-portal` (`version: 2`), mount `mixins: opsRouteMixins(OPS_SCREENS)`.
  - `integrate-services/v2/ops/screens.ts:30-37`: Định nghĩa cấu hình màn hình `COLLECTION_DISBURSEMENT_ORDERS`:
    - `code: 'COLLECTION_DISBURSEMENT_ORDERS'`
    - `path: '/ops/collection-disbursement/orders'`
    - `service: 'payment-bill-service'`
    - `listAction: 'opsSearchPaymentBills'`
    - `detailAction: 'opsGetPaymentBill'`
    - `conditions: { all: [{ fact: 'bizDomain', operator: 'equal', value: BIZ_DOMAIN.TRANSFER_BIZ }] }` (`"TRANSFER_BIZ"`).
  - `integrate-services/v2/ops/ops-routes.ts:21-39`: Hàm `opsRouteMixins` tự động sinh 2 route cho mỗi phần tử trong `OPS_SCREENS`:
    - `GET /ops/collection-disbursement/orders` $\rightarrow$ action `opsCollectionDisbursementOrdersList` $\rightarrow$ gọi `payment-bill-service.opsSearchPaymentBills` với `{ ...ctx.params, conditions: screen.conditions }` (luôn ép `conditions` từ cấu hình server, chống client ghi đè).
    - `GET /ops/collection-disbursement/orders/:_id` $\rightarrow$ action `opsCollectionDisbursementOrdersDetail` $\rightarrow$ gọi `payment-bill-service.opsGetPaymentBill`.
- **Tầng Downstream Service (`payment-bill-service` — nhánh `origin/testing-1.8.0` commit `6c8dc42` & `origin/feature/GP2-5154-classification-fields` commit `3cc932b`):**
  - `start.ts:136-140`: Đăng ký action `opsSearchPaymentBills` (`params: { conditions: { type: "object" } }`).
  - `logics/payment-bill.logic.ts:428-446` (`opsSearchPaymentBills`):
    1. Dịch `params.conditions` sang Mongo filter `screenFilter` qua `buildOpsScreenFilter(params.conditions)` $\rightarrow$ `toMongoFilter(conditions)` (`{ $and: [{ bizDomain: "TRANSFER_BIZ" }] }`).
    2. Gọi `this.searchPaymentBillsOps(context, "opsSearchPaymentBills", screenFilter)` (`logics/payment-bill.logic.ts:459-676`).
    3. Hậu xử lý từng bản ghi qua `mapOpsPaymentBillDoc` (`utils/payment-bill.helper.ts:171-181`):
       - `doc.orderId = doc.merchantTransId ? doc.merchantTransId : doc.orderId`
       - `doc.payerType = getPartyType(doc.transactionParties?.payer)` (`"MERCHANT"` hoặc `"WALLET_USER"`)
       - `doc.payeeType = getPartyType(doc.transactionParties?.payee)` (`"MERCHANT"` hoặc `"WALLET_USER"`).
- **Tầng Frontend (`admin-portal-web` — nhánh `origin/feature/GP2-4551`, commit `91afeb10`):**
  - `src/services/partner-collection-disbursement/api.ts:5-21`: Gọi `HttpService.get('/api/v2/admin-portal/ops/collection-disbursement/orders', { ...params, withInvoiceDocument: true }, options)`.
  - `src/pages/partner-collection-disbursement/bill/list/index.tsx:238-239`: Đang để chờ tính năng export (`// export waits for the v2 export API (GP2-5170)` và `exportExcel={false}`).

### 1.2. Các Phát hiện Kỹ thuật (Findings `F1..F4`)
- **F1 — Khoảng trống bộ lọc giữa `PaymentBillExport` hiện hữu và `opsSearchPaymentBills` (Filter Parity Gap — Lesson `L030`):**
  - Tại `centralized-export-service/logics/export/payment-bill/payment-bill-export.pure.ts:41-103`, hàm `buildPaymentBillFilter` hiện dùng cho màn hình lịch sử đơn hàng cổng thanh toán chung (`PaymentBillExport`) **chưa có**:
    1. Điều kiện cô lập nghiệp vụ Thu/Chi hộ `bizDomain: "TRANSFER_BIZ"` (hoặc `conditions` từ `/ops` screen).
    2. Bộ lọc `payerType` và `payeeType` (`WALLET_USER` / `MERCHANT` suy diễn từ `transactionParties.<party>.partyType` hoặc `merchantCode` vs `userId`/`driverId` theo đúng `buildPartyTypeCondition` tại `payment-bill-service/utils/payment-bill.helper.ts:115-141`).
    3. Bộ lọc `merchantCode`, `isCapturedPayment`, `holdingStep`, và sắp xếp động `sortBy` / `sortType`.
- **F2 — Điểm lệch trong `searchPaymentBillsOps` của GetList so với UI (`payment-bill-service` & `admin-portal-web`):**
  - Tại `payment-bill-service/logics/payment-bill.logic.ts:479-506`, hàm `addLikeFilter` gán trực tiếp `filter[filterKey] = { $regex: value, $options: "i" }` mà **chưa escape ký tự đặc biệt regex** (khác với `centralized-export-service/logics/export/payment-bill/payment-bill-export.pure.ts:32-39` có `escapeRegex`).
  - Tại `admin-portal-web/src/pages/partner-collection-disbursement/bill/list/index.tsx:10,67,92`, cột **Mã bên trả tiền (`payerMerchantCode`)** và **Mã bên thụ hưởng (`payeeMerchantCode`)** hiển thị `getPartyCode(party) = party?.merchantCode || party?.userId` (vì khi Payer/Payee là `WALLET_USER`, định danh nằm ở `userId`). Tuy nhiên ở `payment-bill.logic.ts:591-601`, `payerMerchantCode` / `payeeMerchantCode` mới chỉ tìm trên `transactionParties.<party>.merchantCode` nên khi lọc theo mã `userId` của Ví GooPay sẽ không khớp.
- **F3 — Thiếu trường phân loại mới trên Mongoose Schema `PaymentBillModel` và thiếu `PaymentBillDocumentModel` tại `centralized-export-service`:**
  - Tại `centralized-export-service/models/payment-bill/payment-bill.model.ts:22-135` và `data-transfers/entities/payment-bill/payment-bill.entity.ts:17-71`, schema chưa khai báo các trường `transactionParties`, `transHoldingId`, `serviceCode`, `bizDomain`, `bizType` (đã được thêm ở `payment-bill-service/models/payment-bill.model.ts:138-142`).
  - Màn hình `COLLECTION_DISBURSEMENT_ORDERS` có cột **Chứng từ đính kèm (`invoiceDocument.hasDocument` $\rightarrow$ `Yes` / `-`)** và **Tên nền tảng (`extraInfo.srcPlatformData.platformName || 'GOOPAY'`)**. Để xuất cột Chứng từ đính kèm chính xác như GetList (`enrichInvoiceDocumentFlags`), `centralized-export-service` cần `PaymentBillDocumentModel` (`collection: "payment-bill-documents"` trên kết nối DB `"payment-bill"`) kết hợp kiểm tra `doc.embedData?.uploaded?.length > 0`.
- **F4 — Quy ước bắt buộc `getRequiredModelName()` của `BaseExportProcessor` & Định tuyến `exportType=CollectionDisbursementOrdersExport` tại `centralized-export-service`:**
  - Tại `admin-portal-gateway/integrate-services/v1/start.ts:3900` và `integrate-services/v1/centralized-export/centralizedExport.exportAsync.ts:10-14`, endpoint tập trung `GET /api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport` đã có sẵn và chuyển tiếp nguyên vẹn `ctx.params` xuống `${SVC_ENV.get().CENTRALIZED_EXPORT_SERVICE}.exportAsync` (**tuyệt đối không sửa `admin-portal-gateway`**).
  - Tại `centralized-export-service/logics/processors/BaseExportProcessor.ts:53-79`, phương thức `getRequiredModelName()` mặc định biến đổi tên class `this.constructor.name.replace("Export", "Model")`. Khi tạo processor `CollectionDisbursementOrdersExport`, bắt buộc phải override `protected getRequiredModelName(): string[] { return ["PaymentBillModel", "PaymentBillDocumentModel"]; }` (theo đúng pattern của `PaymentBillForCurrentMerchantExport` tại `logics/export/payment-bill/payment-bill-for-current-merchant.export.ts:24-26` và `TransHisCollectionExport` tại `logics/export/trans-his/trans-his-collection.export.ts:24-26`).

---

## 2. Thiết kế Giải pháp Duy nhất Bám sát 100% Pattern Hệ thống (Ánh xạ 1-1 `F1..F4`)

### 2.1. Kiến trúc & Luồng Thực thi (Call-Site Wiring — Lesson `L001`, `L045`)

| Thứ tự | Tầng / Repo | File & Vị trí (`file:line`) | Caller $\rightarrow$ Callee & Trách nhiệm |
| :--- | :--- | :--- | :--- |
| **1** | **Frontend (`admin-portal-web`)** | `src/services/common/api.ts:24-31` & `src/pages/partner-collection-disbursement/bill/list/index.tsx:12-255` | `useExportProgress('CollectionDisbursementOrdersExport')` gọi `exportAsync` $\rightarrow$ `GET /api/v1/admin-portal/export-async?exportType=CollectionDisbursementOrdersExport&dateFr=...&dateTo=...` kèm các bộ lọc của màn hình `COLLECTION_DISBURSEMENT_ORDERS`. |
| **2** | **API Gateway (`admin-portal-gateway` — Giữ nguyên 100%, KHÔNG sửa code)** | `integrate-services/v1/start.ts:3900` & `integrate-services/v1/centralized-export/centralizedExport.exportAsync.ts:10-14` | Route có sẵn `RestRoute("GET", "/export-async", "exportAsync", ActionExportAsync)` nhận `ctx.params` (`exportType: "CollectionDisbursementOrdersExport"`) và gọi RPC `${SVC_ENV.get().CENTRALIZED_EXPORT_SERVICE}.exportAsync`. |
| **3** | **Export Entry (`centralized-export-service`)** | `logics/export.logic.ts:30-49` & `logics/index.ts:40-45,103-108` | `ExportLogic.processExport()` tra cứu `ExportLogic.processorMap["CollectionDisbursementOrdersExport"]`, khởi tạo `new CollectionDisbursementOrdersExport(context, queryBus, redisManager)` và gọi `.process()` (validate params $\rightarrow$ đẩy vào Redis queue `"export:queue"` $\rightarrow$ trả về `{ jobId, status: "QUEUED" }`). |
| **4** | **Export Worker & Processor (`centralized-export-service`)** | `logics/export/payment-bill/collection-disbursement-orders.export.ts` $\rightarrow$ `logics/export/payment-bill/collection-disbursement-orders-export.pure.ts` | `ExportWorker` gọi `processor.execute(job, mainProcess)` $\rightarrow$ kiểm tra `getRequiredModelName()` (`["PaymentBillModel", "PaymentBillDocumentModel"]`) $\rightarrow$ lấy `ExportConfig` từ `CollectionDisbursementOrdersExportPure.getConfig(params, langCode)`. |
| **5** | **CQRS Query & Handler (`centralized-export-service`)** | `domain/QueryHandlerFactory.ts:48-53,128-135` $\rightarrow$ `domain/handlers/payment-bill/GetAllCollectionDisbursementOrdersExportHandler.ts` | `QueryBus.execute(new GetAllCollectionDisbursementOrdersExportQuery(filter, sort))` truy vấn theo chunk (`limit`, `cursor`) trên `PaymentBillModel`, tổng hợp cờ chứng từ đính kèm qua `PaymentBillDocumentModel.aggregateInvoiceFlagsByBillIds(billIds)` + `embedData.uploaded` và trả về `{ data, nextCursor, total }`. |
| **6** | **Row Transform & Excel Stream (`centralized-export-service`)** | `logics/export/payment-bill/collection-disbursement-orders-export.pure.ts` | `transformRow(item, rowIndex, langCode)` chuyển đổi từng bản ghi sang 18 cột song ngữ (`vi`/`en`) khớp 100% với bảng UI Danh sách đơn hàng Thu/Chi hộ (`COLLECTION_DISBURSEMENT_ORDERS`). |

### 2.2. Chi tiết Ánh xạ Giải pháp theo từng Phát hiện (`F1..F4`)
1. **Giải quyết `F1` (Filter Parity 100% giữa GetList và Export):** Tạo `CollectionDisbursementOrdersExportParams` và `buildCollectionDisbursementOrdersFilter(params)` luôn khóa cứng `bizDomain: "TRANSFER_BIZ"` + `isDelete: false`, đồng thời hỗ trợ đầy đủ `payerType`, `payeeType` (`buildPartyTypeCondition`), `merchantId`, `channelID`, `merchantEmail`, `merchantCode`, `apiType`, `state`, `partnerCode`, `serviceCode`, `orderId`, `trackingId`, `merchantTransId`, `isCapturedPayment`, `holdingStep`, `payerMerchantCode`, `payeeMerchantCode`, `dateFr`/`dateTo`, `updatedFr`/`updatedTo`, `sortBy`/`sortType`.
2. **Giải quyết `F2` (Chuẩn hóa bộ lọc `payerMerchantCode` / `payeeMerchantCode` & Regex Escape):** Khi lọc `payerMerchantCode` hoặc `payeeMerchantCode`, tìm kiếm regex đã `escapeRegex` trên cả `transactionParties.<party>.merchantCode` và `transactionParties.<party>.userId` (khớp với `getPartyCode` trên UI cho cả `MERCHANT` lẫn `WALLET_USER`), cập nhật đồng bộ cả trong `payment-bill-service/logics/payment-bill.logic.ts`.
3. **Giải quyết `F3` (Đồng bộ Schema `PaymentBillModel`, bổ sung `PaymentBillDocumentModel` & 18 cột Export):** Bổ sung `transactionParties`, `transHoldingId`, `serviceCode`, `bizDomain`, `bizType` vào `PaymentBillModel` & `PaymentBillEntity`; đăng ký `PaymentBillDocumentModel` vào `ModelRegistry["payment-bill"]`; tổng hợp `hasDocument` trong `GetAllCollectionDisbursementOrdersExportHandler`.
4. **Giải quyết `F4` (Đăng ký Processor `CollectionDisbursementOrdersExport`, QueryHandler & Override `getRequiredModelName()`):** Override `protected getRequiredModelName(): string[] { return ["PaymentBillModel", "PaymentBillDocumentModel"]; }` trong `CollectionDisbursementOrdersExport` và đăng ký đầy đủ vào `domain/queries/index.ts`, `domain/QueryHandlerFactory.ts`, `data-transfers/params/index.ts`, `logics/export/index.ts`, `logics/index.ts` (tuyệt đối không sửa `admin-portal-gateway`).

---

## 3. Code Demo Nguyên văn & Thiết kế Chi tiết (Propose-Only — Lesson `L001`, `L019`, `L036`)

### 3.1. Code Nền Hiện tại tại `HEAD` (Trích nguyên văn 100% từ đĩa)

#### (A) `centralized-export-service/logics/export/payment-bill/payment-bill-for-current-merchant.export.ts:1-29`
```ts
// Thin adapter - delegates to pure functions in export/payment-bill-merchant
import { RequestHelper } from "@goopay/goopay-library/helpers";
import { BaseExportProcessor, ExportConfig } from "../../processors/BaseExportProcessor";
import { PaymentBillForCurrentMerchantExportPure } from "../index";

export class PaymentBillForCurrentMerchantExport extends BaseExportProcessor {
  private rowIndex: number = 0;

  public validate(): void {
    const langCode = RequestHelper.getLangCode(this.context);
    PaymentBillForCurrentMerchantExportPure.validate(this.context.params, langCode);
  }

  protected getConfig(): ExportConfig {
    const langCode = RequestHelper.getLangCode(this.context);
    const currentUser = this.context.meta?.currentUser || {};
    return PaymentBillForCurrentMerchantExportPure.getConfigWithUser(this.context.params, currentUser, langCode);
  }

  protected transformRow(item: any, langCode: string): any[] {
    return PaymentBillForCurrentMerchantExportPure.transformRow(item, ++this.rowIndex, langCode as "vi" | "en");
  }

  protected getRequiredModelName(): string[] {
    return ["PaymentBillModel"];
  }
}

export default PaymentBillForCurrentMerchantExport;
```

#### (B) `centralized-export-service/domain/queries/payment-bill/GetAllPaymentBillExportQuery.ts:1-12`
```ts
// <DOCUMENT filename="domain/queries/GetAllPaymentBillExportQuery.ts">
import { IQuery } from "../../IQuery";

export class GetAllPaymentBillExportQuery implements IQuery {
  public filter: any;
  public sort: object;

  public constructor(filter: any, sort: object = { createdAt: -1 }) {
    this.filter = filter;
    this.sort = sort;
  }
}
```

---

### 3.2. Code Demo Các File Sẽ Tạo Mới / Cập nhật (Đầy đủ 100%, Không Cắt Dòng)

#### Demo 1: `centralized-export-service/data-transfers/params/payment-bill/collection-disbursement-orders-export.params.ts`
```ts
// data-transfers/params/payment-bill/collection-disbursement-orders-export.params.ts
import {
    IsDateString,
    IsIn,
    IsNotEmpty,
    IsOptional,
    IsString,
    MinLength,
    Validate,
    ValidatorConstraint,
    ValidatorConstraintInterface,
    ValidationArguments
} from "class-validator";
import moment from "moment";
import { MAX_DAYS_EXPORT, MAX_YEAR_DAYS_EXPORT } from "../../../utils/constants";

@ValidatorConstraint({ name: "ExceedDateRangeCollectionDisbursementOrders", async: false })
class ExceedDateRangeValidator implements ValidatorConstraintInterface {
    public validate(value: any, args: ValidationArguments) {
        const params = args.object as any;
        if (!params.dateFr || !params.dateTo) {
            return true;
        }
        const dateFr = moment(params.dateFr, "YYYY-MM-DD").startOf("day");
        const dateTo = moment(params.dateTo, "YYYY-MM-DD").startOf("day");
        const dateCurrent = moment().startOf("day");
        if (dateCurrent.diff(dateFr, "days") > MAX_YEAR_DAYS_EXPORT) {
            return false;
        }
        return dateTo.diff(dateFr, "days") <= MAX_DAYS_EXPORT;
    }

    public defaultMessage(args: ValidationArguments) {
        const params = args.object as any;
        const dateFr = moment(params.dateFr, "YYYY-MM-DD").startOf("day");
        const dateCurrent = moment().startOf("day");
        if (dateCurrent.diff(dateFr, "days") > MAX_YEAR_DAYS_EXPORT) {
            return "EXCEED_YEAR_RANGE_EXPORT";
        }
        return "EXCEED_DATE_RANGE_EXPORT";
    }
}

export class CollectionDisbursementOrdersExportParams {
    @IsNotEmpty({ message: "ERR_DATE_FR_REQUIRED" })
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public dateFr: string;

    @IsNotEmpty({ message: "ERR_DATE_TO_REQUIRED" })
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public dateTo: string;

    @IsOptional()
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public updatedFr?: string;

    @IsOptional()
    @IsDateString({}, { message: "ERR_INVALID_DATE" })
    public updatedTo?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public orderId?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public merchantEmail?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public merchantCode?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public merchantTransId?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public partnerCode?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public payerMerchantCode?: string;

    @IsOptional()
    @IsString()
    @MinLength(3, { message: "ERR_MIN_LENGTH_3" })
    public payeeMerchantCode?: string;

    @IsOptional()
    @IsString()
    public serviceCode?: string;

    @IsOptional()
    @IsString()
    public state?: string;

    @IsOptional()
    @IsString()
    public apiType?: string;

    @IsOptional()
    @IsString()
    public channelID?: string;

    @IsOptional()
    public merchantId?: any;

    @IsOptional()
    @IsString()
    public trackingId?: string;

    @IsOptional()
    @IsString()
    public holdingStep?: string;

    @IsOptional()
    @IsString()
    public isCapturedPayment?: string;

    @IsOptional()
    @IsString()
    @IsIn(["WALLET_USER", "MERCHANT", "USER"], { message: "PARAMS_INVALID_FORMAT" })
    public payerType?: string;

    @IsOptional()
    @IsString()
    @IsIn(["WALLET_USER", "MERCHANT", "USER"], { message: "PARAMS_INVALID_FORMAT" })
    public payeeType?: string;

    @IsOptional()
    @IsString()
    public sortBy?: string;

    @IsOptional()
    public sortType?: string | number;

    @IsOptional()
    @Validate(ExceedDateRangeValidator)
    public get dateRuleCheck(): boolean {
        return true;
    }
}
```

#### Demo 2: `centralized-export-service/logics/export/payment-bill/collection-disbursement-orders-export.pure.ts`
```ts
// Pure functions cho CollectionDisbursementOrdersExport logic
import moment from "moment";
import _ from "lodash";
import { ExportConfig } from "../../processors/BaseExportProcessor";
import { GetAllCollectionDisbursementOrdersExportQuery } from "../../../domain/queries";
import { CollectionDisbursementOrdersExportParams } from "../../../data-transfers/params";
import { validateParams } from "../../../utils/validator";
import { PAYMENT_STATUS_LIST } from "../../../utils/constants";

export const BIZ_DOMAIN_TRANSFER = "TRANSFER_BIZ";

export const PARTY_TYPE = {
    WALLET_USER: "WALLET_USER",
    MERCHANT: "MERCHANT",
    USER: "USER"
} as const;

export const BILL_PARTY_TYPE_LABELS: Record<string, { vi: string; en: string }> = {
    WALLET_USER: { vi: "Người dùng Ví GooPay", en: "GooPay Wallet user" },
    MERCHANT: { vi: "Merchant", en: "Merchant" }
};

const isEmptyValue = (value: any): boolean => value === undefined || value === null || value === "";
const escapeRegex = (text: string): string => text ? text.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&") : "";

const buildRegexSearch = (val: string) => {
    if (typeof val === "string" && val.trim().length >= 3) {
        return { $regex: escapeRegex(val.trim()), $options: "i" };
    }
    return val;
};

export const buildPartyTypeCondition = (party: "payer" | "payee", type: string): object | null => {
    const path = `transactionParties.${party}`;
    const empty = [null, ""];
    let partyType: string;
    let inferred: object;
    if (type === PARTY_TYPE.MERCHANT) {
        partyType = PARTY_TYPE.MERCHANT;
        inferred = { [`${path}.merchantCode`]: { $nin: empty } };
    } else if (type === PARTY_TYPE.WALLET_USER || type === PARTY_TYPE.USER) {
        partyType = PARTY_TYPE.WALLET_USER;
        inferred = {
            [`${path}.merchantCode`]: { $in: empty },
            $or: [
                { [`${path}.userId`]: { $nin: empty } },
                { [`${path}.driverId`]: { $nin: empty } }
            ]
        };
    } else {
        return null;
    }
    return {
        $or: [
            { [`${path}.partyType`]: partyType },
            { [`${path}.partyType`]: null, ...inferred }
        ]
    };
};

export const getPartyType = (party: any): string | null => {
    if (!party || typeof party !== "object") { return null; }
    if (!isEmptyValue(party.partyType)) {
        return [PARTY_TYPE.MERCHANT, PARTY_TYPE.WALLET_USER].includes(party.partyType) ? party.partyType : null;
    }
    if (!isEmptyValue(party.merchantCode)) { return PARTY_TYPE.MERCHANT; }
    if (!isEmptyValue(party.userId) || !isEmptyValue(party.driverId)) { return PARTY_TYPE.WALLET_USER; }
    return null;
};

export const getPartyCode = (party: any): string => party?.merchantCode || party?.userId || "";
export const getPartyName = (party: any): string => party?.name || party?.userName || "";

export const buildCollectionDisbursementOrdersFilter = (params: any) => {
    const filter: { [k: string]: any } = {};
    const andConditions: object[] = [{ bizDomain: BIZ_DOMAIN_TRANSFER }];

    if ("merchantId" in params && !isEmptyValue(params.merchantId)) {
        filter["merchantInfo.id"] = params.merchantId;
    }
    if ("channelID" in params && !isEmptyValue(params.channelID)) {
        filter.$or = [{ channelID: params.channelID }, { "payments.channelID": params.channelID }];
    }
    if ("merchantEmail" in params && params.merchantEmail) {
        filter["merchantInfo.email"] = buildRegexSearch(params.merchantEmail);
    }
    if ("merchantCode" in params && params.merchantCode) {
        filter["merchantInfo.merchantCode"] = buildRegexSearch(params.merchantCode);
    }

    ["apiType", "state", "partnerCode", "serviceCode", "trackingId"].forEach(field => {
        if (field in params && !isEmptyValue(params[field])) {
            filter[field] = params[field];
        }
    });

    ["orderId", "merchantTransId"].forEach(field => {
        if (field in params && params[field]) {
            filter[field] = buildRegexSearch(params[field]);
        }
    });

    if (params.isCapturedPayment === "true") {
        filter.holdingStep = { $nin: [undefined, null, ""] };
    }
    if ("holdingStep" in params && !isEmptyValue(params.holdingStep)) {
        filter.holdingStep = params.holdingStep;
    }
    if ("payerMerchantCode" in params && params.payerMerchantCode) {
        const regex = buildRegexSearch(params.payerMerchantCode);
        andConditions.push({
            $or: [{ "transactionParties.payer.merchantCode": regex }, { "transactionParties.payer.userId": regex }]
        });
    }
    if ("payeeMerchantCode" in params && params.payeeMerchantCode) {
        const regex = buildRegexSearch(params.payeeMerchantCode);
        andConditions.push({
            $or: [{ "transactionParties.payee.merchantCode": regex }, { "transactionParties.payee.userId": regex }]
        });
    }

    for (const [paramKey, party] of [["payerType", "payer"], ["payeeType", "payee"]] as const) {
        if (!params[paramKey]) { continue; }
        const condition = buildPartyTypeCondition(party, params[paramKey]);
        if (condition) { andConditions.push(condition); }
    }

    if ("dateFr" in params || "dateTo" in params) {
        filter.createdAt = {};
        if ("dateFr" in params && params.dateFr) { filter.createdAt.$gte = new Date(params.dateFr); }
        if ("dateTo" in params && params.dateTo) { filter.createdAt.$lte = new Date(params.dateTo); }
    }
    if ("updatedFr" in params || "updatedTo" in params) {
        filter.lastUpdatedAt = {};
        if ("updatedFr" in params && params.updatedFr) { filter.lastUpdatedAt.$gte = new Date(params.updatedFr); }
        if ("updatedTo" in params && params.updatedTo) { filter.lastUpdatedAt.$lte = new Date(params.updatedTo); }
    }

    filter.isDelete = false;
    filter.$and = [...(filter.$and || []), ...andConditions];
    return filter;
};

export const buildCollectionDisbursementOrdersSort = (params: any): object => {
    if (params?.sortBy && !isEmptyValue(params?.sortType)) {
        const sortOrder = params.sortType === "asc" || params.sortType === "1" || params.sortType === 1 ? 1 : -1;
        return { [params.sortBy]: sortOrder };
    }
    return { createdAt: -1 };
};

export const validate = (params: any, langCode: string = "vi"): CollectionDisbursementOrdersExportParams =>
    validateParams<CollectionDisbursementOrdersExportParams>(CollectionDisbursementOrdersExportParams, params || {}, langCode);

export const getConfig = (params: any, langCode: string = "vi"): ExportConfig => {
    const validatedParams = validate(params, langCode);
    const filter = buildCollectionDisbursementOrdersFilter(validatedParams);
    const sort = buildCollectionDisbursementOrdersSort(validatedParams);

    return {
        query: new GetAllCollectionDisbursementOrdersExportQuery(filter, sort),
        columns: [
            { vi: "STT", en: "Order" },
            { vi: "Ngày tạo", en: "Created at" },
            { vi: "Mã giao dịch merchant", en: "Merchant transaction ID" },
            { vi: "Trạng thái", en: "Status" },
            { vi: "Tài khoản merchant", en: "Merchant account" },
            { vi: "Loại bên trả tiền", en: "Payer type" },
            { vi: "Mã bên trả tiền", en: "Payer code" },
            { vi: "Tên bên trả tiền", en: "Payer name" },
            { vi: "Loại bên thụ hưởng", en: "Payee type" },
            { vi: "Mã bên thụ hưởng", en: "Payee code" },
            { vi: "Tên bên thụ hưởng", en: "Payee name" },
            { vi: "Mã dịch vụ", en: "Service code" },
            { vi: "Số tiền", en: "Amount" },
            { vi: "Số tiền thực nhận", en: "Paid amount" },
            { vi: "Loại tiền tệ", en: "Currency" },
            { vi: "Chứng từ đính kèm", en: "Attached document" },
            { vi: "Tên nền tảng", en: "Platform name" },
            { vi: "Ngày cập nhật", en: "Updated at" }
        ],
        fileName: {
            vi: moment().format("DD-MM-YYYY") + "_Don_Hang_Thu_Chi_Ho",
            en: moment().format("DD-MM-YYYY") + "_Transfer_Orders_List"
        },
        maxRecords: 2_000_000
    };
};

export const transformRow = (item: any, rowIndex: number, langCode: "vi" | "en" = "vi"): any[] => {
    const lang: "vi" | "en" = langCode === "en" ? "en" : "vi";
    const statusType = PAYMENT_STATUS_LIST?.find(x => x.value === item.state);
    const payer = item.transactionParties?.payer;
    const payee = item.transactionParties?.payee;
    const payerTypeKey = getPartyType(payer);
    const payeeTypeKey = getPartyType(payee);

    const payerTypeLabel = payerTypeKey ? (BILL_PARTY_TYPE_LABELS[payerTypeKey]?.[lang] || payerTypeKey) : "";
    const payeeTypeLabel = payeeTypeKey ? (BILL_PARTY_TYPE_LABELS[payeeTypeKey]?.[lang] || payeeTypeKey) : "";

    const createdAt = item.createdAt ? moment(item.createdAt).utcOffset(7).format("DD/MM/YYYY HH:mm:ss") : "";
    const lastUpdatedAt = item.lastUpdatedAt ? moment(item.lastUpdatedAt).utcOffset(7).format("DD/MM/YYYY HH:mm:ss") : "";
    const currency = Array.isArray(item.currency) ? item.currency.join(", ") : (item.currency || "VND");
    const hasDocument = Boolean(item.hasDocument || item.invoiceDocument?.hasDocument);
    const platformName = _.get(item, "extraInfo.srcPlatformData.platformName") || "GOOPAY";

    return [
        rowIndex,
        createdAt,
        item.merchantTransId || "",
        statusType?.label?.[lang] || item.state || "",
        item.merchantInfoEmail || _.get(item, "merchantInfo.email") || "",
        payerTypeLabel,
        getPartyCode(payer),
        getPartyName(payer),
        payeeTypeLabel,
        getPartyCode(payee),
        getPartyName(payee),
        item.serviceCode || "",
        item.amount || 0,
        item.paidAmount || 0,
        currency,
        hasDocument ? "Yes" : "-",
        platformName,
        lastUpdatedAt
    ];
};
```

#### Demo 3: `centralized-export-service/logics/export/payment-bill/collection-disbursement-orders.export.ts`
```ts
// Thin adapter - delegates to pure functions in export/payment-bill
import { RequestHelper } from "@goopay/goopay-library/helpers";
import { BaseExportProcessor, ExportConfig } from "../../processors/BaseExportProcessor";
import { CollectionDisbursementOrdersExportPure } from "../index";

export class CollectionDisbursementOrdersExport extends BaseExportProcessor {
  private rowIndex: number = 0;

  public validate(): void {
    const langCode = RequestHelper.getLangCode(this.context);
    CollectionDisbursementOrdersExportPure.validate(this.context.params, langCode);
  }

  protected getConfig(): ExportConfig {
    const langCode = RequestHelper.getLangCode(this.context);
    return CollectionDisbursementOrdersExportPure.getConfig(this.context.params, langCode);
  }

  protected transformRow(item: any, langCode: string): any[] {
    return CollectionDisbursementOrdersExportPure.transformRow(item, ++this.rowIndex, langCode as "vi" | "en");
  }

  protected getRequiredModelName(): string[] {
    return ["PaymentBillModel", "PaymentBillDocumentModel"];
  }
}

export default CollectionDisbursementOrdersExport;
```

#### Demo 4: `centralized-export-service/domain/queries/payment-bill/GetAllCollectionDisbursementOrdersExportQuery.ts` & `domain/handlers/payment-bill/GetAllCollectionDisbursementOrdersExportHandler.ts`
```ts
// <DOCUMENT filename="domain/queries/payment-bill/GetAllCollectionDisbursementOrdersExportQuery.ts">
import { IQuery } from "../../IQuery";

export class GetAllCollectionDisbursementOrdersExportQuery implements IQuery {
  public filter: any;
  public sort: object;

  public constructor(filter: any, sort: object = { createdAt: -1 }) {
    this.filter = filter;
    this.sort = sort;
  }
}
```
```ts
// <DOCUMENT filename="domain/handlers/payment-bill/GetAllCollectionDisbursementOrdersExportHandler.ts">
import * as _ from "lodash";
import { IQueryHandler } from "../../IQueryHandler";
import { GetAllCollectionDisbursementOrdersExportQuery } from "../../queries";
import MainProcess from "../../../data-transfers/main-process";

export class GetAllCollectionDisbursementOrdersExportHandler implements IQueryHandler {
  private mainProcess: MainProcess;

  public constructor(mainProcess: MainProcess) {
    this.mainProcess = mainProcess;
  }

  public async execute(query: GetAllCollectionDisbursementOrdersExportQuery, context: any, options: any): Promise<any> {
    const { filter, sort } = query;
    const { cursor, limit = 5000 } = options?.params || {};

    const Model = this.mainProcess.models.PaymentBillModel;
    const DocumentModel = this.mainProcess.models.PaymentBillDocumentModel;

    const selectFields = {
      "_id": 1,
      "merchantInfo.email": 1,
      "merchantTransId": 1,
      "orderId": 1,
      "amount": 1,
      "paidAmount": 1,
      "currency": 1,
      "state": 1,
      "createdAt": 1,
      "lastUpdatedAt": 1,
      "transactionParties": 1,
      "serviceCode": 1,
      "extraInfo": 1,
      "embedData": 1
    };

    let findQuery = Model.find(filter)
      .sort(sort)
      .select(selectFields)
      .limit(limit);

    if (cursor && cursor > 0) {
      findQuery = findQuery.skip(cursor);
    }

    const [rawData, total] = await Promise.all([
      findQuery.lean().exec(),
      cursor ? Promise.resolve(0) : Model.countDocuments(filter)
    ]);

    const billIds = (rawData || [])
      .map((doc: any) => (doc?._id !== undefined && doc?._id !== null ? String(doc._id) : null))
      .filter((id: string | null): id is string => Boolean(id));

    const docFlagSet = new Set<string>();
    if (billIds.length > 0 && DocumentModel && typeof DocumentModel.aggregateInvoiceFlagsByBillIds === "function") {
      try {
        const grouped = await DocumentModel.aggregateInvoiceFlagsByBillIds(billIds);
        for (const group of grouped || []) {
          const sources: string[] = group?.sources || [];
          if (sources.includes("SYNC") || sources.includes("UPLOAD")) {
            docFlagSet.add(String(group._id));
          }
        }
      } catch (err) {
        if (this.mainProcess.logger) {
          this.mainProcess.logger.warn(
            `[GetAllCollectionDisbursementOrdersExportHandler] aggregateInvoiceFlagsByBillIds failed: ${JSON.stringify(err)}`
          );
        }
      }
    }

    const data = (rawData || []).map((doc: any) => {
      const embedded = ((doc?.embedData as any)?.uploaded?.length ?? 0) > 0;
      const hasDocInCollection = doc?._id !== undefined && doc?._id !== null
        ? docFlagSet.has(String(doc._id))
        : false;
      const hasDocument = hasDocInCollection || embedded;

      return {
        _id: doc._id,
        merchantInfoEmail: _.get(doc, "merchantInfo.email") || "",
        merchantTransId: doc.merchantTransId,
        orderId: doc.merchantTransId ? doc.merchantTransId : doc.orderId,
        amount: doc.amount,
        paidAmount: doc.paidAmount,
        currency: doc.currency,
        state: doc.state,
        createdAt: doc.createdAt,
        lastUpdatedAt: doc.lastUpdatedAt,
        transactionParties: doc.transactionParties,
        serviceCode: doc.serviceCode,
        extraInfo: doc.extraInfo,
        hasDocument,
        invoiceDocument: { hasDocument }
      };
    });

    return {
      data,
      nextCursor: Array.isArray(data) && data.length === limit ? (cursor || 0) + limit : null,
      total
    };
  }
}
```
