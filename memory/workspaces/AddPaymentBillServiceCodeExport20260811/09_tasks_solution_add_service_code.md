# Solution Specification: Add Service Code to PaymentBillExport

## Technical Solution Details

### File 1: `data-transfers/params/payment-bill/payment-bill-export.params.ts`
```typescript
    @IsOptional()
    @IsString()
    public serviceCode?: string;
```

### File 2: `logics/export/payment-bill/payment-bill-export.pure.ts`
1. Update `buildPaymentBillFilter`:
```typescript
const directFields = ["apiType", "state", "partnerCode", "orderId", "trackingId", "merchantTransId", "serviceCode"];
```

2. Update `getConfig` columns:
```typescript
        columns: [
            { vi: "STT", en: "Order" },
            { vi: "Tài khoản merchant", en: "Merchant account" },
            { vi: "Mã Payer", en: "Payer Code" },
            { vi: "Tên Payer", en: "Payer Name" },
            { vi: "Mã Payee", en: "Payee Code" },
            { vi: "Tên Payee", en: "Payee Name" },
            { vi: "Mã giao dịch merchant", en: "Merchant transaction ID" },
            { vi: "Mã đơn hàng", en: "Order ID" },
            { vi: "Mã dịch vụ", en: "Service code" },
            { vi: "Số tiền", en: "Amount" },
            { vi: "Số tiền thực nhận", en: "Paid amount" },
            { vi: "Tổng tiền đã hoàn", en: "Refunded amount" },
            { vi: "Thông tin đơn hàng", en: "Order information" },
            { vi: "Kênh thanh toán", en: "Channel ID" },
            { vi: "Loại api", en: "Api type" },
            { vi: "Loại tiền tệ", en: "Currency" },
            { vi: "Trạng thái", en: "Status" },
            { vi: "Ngày tạo", en: "Created at" },
            { vi: "Ngày cập nhật", en: "Updated at" }
        ],
```

3. Update `transformRow`:
```typescript
    return [
        rowIndex,
        item.merchantInfoEmail || "",
        item.transactionParties?.payer?.merchantCode || "",
        item.transactionParties?.payer?.name || "",
        item.transactionParties?.payee?.merchantCode || "",
        item.transactionParties?.payee?.name || "",
        item.merchantTransId || "",
        item.orderId || "",
        item.serviceCode || "",
        item.amount || 0,
        item.paidAmount || 0,
        item.refundedAmount || 0,
        item.orderInfo || "",
        channelID || "",
        item.apiType || API_TYPE.REDIRECT,
        item.currency || "VND",
        statusType?.label?.[langCode] || item.state,
        createdAt,
        lastUpdatedAt
    ];
```

### File 3: `test/unit/pure/payment-bill.pure.test.ts`
Cập nhật các assertion của unit test:
- `config.columns` length là 19.
- `row[8]` (Mã dịch vụ) = `item.serviceCode || ''`.
- Các index cột phía sau được đẩy lùi 1 vị trí (Số tiền = `row[9]`, ChannelID = `row[13]`, Status = `row[16]`, CreatedAt = `row[17]`).
