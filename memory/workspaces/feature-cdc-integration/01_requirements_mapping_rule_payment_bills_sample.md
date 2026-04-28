# Requirements — Mapping Rule Seed Data cho `payment_bills` (JsonPath design validation)

> **Date**: 2026-04-21
> **Context**: User đưa 1 sample record `payment_bills` làm ví dụ minh họa tại sao `cdc_mapping_rule` PHẢI hỗ trợ JsonPath. Doc này đóng vai **seed data** cho `02_plan_airbyte_removal_v2_command_center.md` R0 migration + R6 Transmuter tests.
> **Parent plan**: `02_plan_airbyte_removal_v2_command_center.md`

---

## 1. Sample record (ground truth)

```json
{
  "_id": 1,
  "channelID": "BANK_TRANSFER",
  "state": "SUCCESS",
  "orderId": "DH6742593310",
  "trackingId": "240708143819IYQYPK",
  "reason": {},
  "expireTime": { "$date": "2024-07-05T09:07:37.072Z" },
  "requestId": "4640171274",
  "billId": 1,
  "merchantTransId": "T1019900187",
  "amount": 10000,
  "currency": "VND",
  "fxRate": 1,
  "merchant": {
    "reference": "CSJKXK",
    "email": "merchant-test@yopmail.vn",
    "type": "REGULAR_SERVICE",
    "platformClient": null,
    "platformMerchantId": null
  },
  "fee": 88,
  "instrument": {},
  "ewalletInfo": {},
  "connector": "bidv",
  "partnerCode": "d246b80a-6e88-4b9c-b594-015dcb5a5af9",
  "extraInfo": {
    "billAmount": 500000,
    "bankTransfer": {
      "accountNumber": "SUMTINGNaN",
      "bankCode": "bidv",
      "bankName": "BIDV",
      "remark": "Thanh toan don hang GOOPAY1"
    },
    "ruleLogs": []
  },
  "isActive": false,
  "isDelete": false,
  "createdBy": "",
  "lastUpdatedBy": "",
  "createdAt":     { "$date": "2024-07-10T07:17:59.271Z" },
  "lastUpdatedAt": { "$date": "2024-07-10T07:17:59.271Z" },
  "refundedAmount": 0,
  "apiType": "REDIRECT",
  "reqCommand": null,
  "__v": 0
}
```

**Structural observations**:
- **Top-level keys**: 30 keys (chính xác ~30 mà user target ở DoD Phase 2)
- **Flat scalars**: 20+ (string/number/boolean/null)
- **Empty objects** (3): `reason`, `instrument`, `ewalletInfo` — business ý nghĩa 0, **nên skip native column**, chỉ giữ trong `_raw_data`
- **Mongo Extended JSON dates** (3): `expireTime`, `createdAt`, `lastUpdatedAt` (hình dạng `{"$date": "ISO string"}`)
- **Nested object** (1 lvl): `merchant.{5 fields}` — flatten có lợi vì query analyst thường filter theo `merchant.reference`, `merchant.email`
- **Nested object** (2 lvl): `extraInfo.bankTransfer.{4 fields}` — flatten 1 phần (chọn business-critical fields)
- **Array của object** (1): `extraInfo.ruleLogs[]` — keep as JSONB column (unknown length, unknown schema)
- **Nullable fields**: `merchant.platformClient/platformMerchantId`, `reqCommand`
- **Mongo internal**: `__v` (version marker, Mongoose) — skip
- **`_id: 1` (integer)**: NOT standard Mongo ObjectID — sample có thể mock; Transmuter code vẫn handle đúng vì `_gpay_source_id` extract logic ở SinkWorker (`envelope.go::extractSourceID`) đã cover scalar + ObjectID + string paths.

---

## 2. Quyết định thiết kế — cột native vs JSONB

### 2.1 Rule of thumb

| Field shape | Native PG column | Stay in `_raw_data` |
|---|---|---|
| Top-level scalar (string/number/bool) | ✅ YES | ✅ (duplicate, luôn còn) |
| Nested 1-lvl, business-critical (merchant.email, merchant.reference) | ✅ YES (via JsonPath) | ✅ |
| Nested 2-lvl, business-critical (extraInfo.bankTransfer.bankCode) | ✅ YES | ✅ |
| Nested object, UI không lookup (merchant.platformClient) | ❌ skip native | ✅ |
| Empty object `{}` | ❌ | ✅ |
| Array (ruleLogs) | ✅ (as JSONB column) | ✅ |
| Mongo `$date` wrapper | ✅ (as TIMESTAMPTZ) | ✅ |
| Internal marker (`__v`, `_id`) | ❌ (`_id` → `_gpay_source_id`) | ✅ |

### 2.2 Master table shape target

```
public.payment_bills_master (
  -- CDC system cols (11) — cùng schema cdc_internal shadow
  _gpay_id          BIGINT PRIMARY KEY,
  _gpay_source_id   TEXT NOT NULL UNIQUE,
  _raw_data         JSONB NOT NULL,    -- full envelope preserved
  _source           TEXT NOT NULL,      -- 'debezium-transmute'
  _source_ts        BIGINT,
  _synced_at        TIMESTAMPTZ NOT NULL,
  _version          BIGINT NOT NULL,
  _hash             TEXT NOT NULL,
  _gpay_deleted     BOOLEAN NOT NULL DEFAULT FALSE,
  _created_at       TIMESTAMPTZ NOT NULL,
  _updated_at       TIMESTAMPTZ NOT NULL,

  -- 30 business cols derived via JsonPath (per Section 3 below)
  bill_id           BIGINT,
  channel_id        TEXT,
  state             TEXT,
  order_id          TEXT,
  tracking_id       TEXT,
  request_id        TEXT,
  merchant_trans_id TEXT,
  amount            NUMERIC(20,4),
  currency          TEXT,
  fx_rate           NUMERIC(10,6),
  fee               NUMERIC(20,4),
  refunded_amount   NUMERIC(20,4),
  connector         TEXT,
  partner_code      TEXT,
  api_type          TEXT,
  req_command       TEXT,
  is_active         BOOLEAN,
  is_delete         BOOLEAN,
  created_by        TEXT,
  last_updated_by   TEXT,
  expire_time       TIMESTAMPTZ,
  created_at        TIMESTAMPTZ,
  last_updated_at   TIMESTAMPTZ,
  -- flatten merchant
  merchant_reference          TEXT,
  merchant_email              TEXT,
  merchant_type               TEXT,
  -- flatten extraInfo
  extra_bill_amount           NUMERIC(20,4),
  extra_bank_account_number   TEXT,
  extra_bank_code             TEXT,
  extra_bank_name             TEXT,
  extra_bank_remark           TEXT,
  extra_rule_logs             JSONB        -- array keep as JSONB
);
```

→ **30 business cols + 11 system = 41 physical cols**. Trùng với expectation user "~30 cột" (business).

---

## 3. Mapping rule seed (YAML format for R0 migration)

Target: `INSERT INTO cdc_mapping_rule (source_table, source_field, target_column, data_type, source_format, jsonpath, transform_fn, is_nullable, default_value, status, rule_type, approved_by_admin, version) VALUES ...`

```yaml
# File: migrations/020_seed_payment_bills_mapping_rules.sql (seeded after R0 migration)
# source_table: "payment-bills" (hyphenated Mongo collection name, will be matched to cdc_internal.payment_bills per R4 gap-fix normalization)
# target_table: "payment_bills_master" (per R0.1 master table convention)
# source_format: all rules use 'debezium_after' since cdc_internal.payment_bills._raw_data = Debezium envelope; jsonpath prefix auto = "after."

rules:
  # ---- 23 flat scalars ----
  - target_column: bill_id
    jsonpath: after.billId
    data_type: BIGINT
    transform_fn: bigint_str    # Mongo can serialize Long as {"$numberLong":"..."} — safe cast
    is_nullable: false
  - target_column: channel_id
    jsonpath: after.channelID
    data_type: TEXT
  - target_column: state
    jsonpath: after.state
    data_type: TEXT
  - target_column: order_id
    jsonpath: after.orderId
    data_type: TEXT
  - target_column: tracking_id
    jsonpath: after.trackingId
    data_type: TEXT
  - target_column: request_id
    jsonpath: after.requestId
    data_type: TEXT
  - target_column: merchant_trans_id
    jsonpath: after.merchantTransId
    data_type: TEXT
  - target_column: amount
    jsonpath: after.amount
    data_type: NUMERIC
    transform_fn: numeric_cast
  - target_column: currency
    jsonpath: after.currency
    data_type: TEXT
  - target_column: fx_rate
    jsonpath: after.fxRate
    data_type: NUMERIC
    transform_fn: numeric_cast
  - target_column: fee
    jsonpath: after.fee
    data_type: NUMERIC
    transform_fn: numeric_cast
  - target_column: refunded_amount
    jsonpath: after.refundedAmount
    data_type: NUMERIC
    transform_fn: numeric_cast
  - target_column: connector
    jsonpath: after.connector
    data_type: TEXT
  - target_column: partner_code
    jsonpath: after.partnerCode
    data_type: TEXT
  - target_column: api_type
    jsonpath: after.apiType
    data_type: TEXT
  - target_column: req_command
    jsonpath: after.reqCommand
    data_type: TEXT
    is_nullable: true
  - target_column: is_active
    jsonpath: after.isActive
    data_type: BOOLEAN
  - target_column: is_delete
    jsonpath: after.isDelete
    data_type: BOOLEAN
  - target_column: created_by
    jsonpath: after.createdBy
    data_type: TEXT
  - target_column: last_updated_by
    jsonpath: after.lastUpdatedBy
    data_type: TEXT

  # ---- 3 Mongo Extended JSON dates ----
  - target_column: expire_time
    jsonpath: after.expireTime.$date
    data_type: TIMESTAMPTZ
    transform_fn: mongo_date_ms  # handles both ISO-string và ms-int forms
  - target_column: created_at
    jsonpath: after.createdAt.$date
    data_type: TIMESTAMPTZ
    transform_fn: mongo_date_ms
  - target_column: last_updated_at
    jsonpath: after.lastUpdatedAt.$date
    data_type: TIMESTAMPTZ
    transform_fn: mongo_date_ms

  # ---- 3 nested merchant.* (1-level dive) ----
  - target_column: merchant_reference
    jsonpath: after.merchant.reference
    data_type: TEXT
  - target_column: merchant_email
    jsonpath: after.merchant.email
    data_type: TEXT
  - target_column: merchant_type
    jsonpath: after.merchant.type
    data_type: TEXT

  # ---- 5 nested extraInfo.* (1-2 level dive, some skipped) ----
  - target_column: extra_bill_amount
    jsonpath: after.extraInfo.billAmount
    data_type: NUMERIC
    transform_fn: numeric_cast
  - target_column: extra_bank_account_number
    jsonpath: after.extraInfo.bankTransfer.accountNumber
    data_type: TEXT
  - target_column: extra_bank_code
    jsonpath: after.extraInfo.bankTransfer.bankCode
    data_type: TEXT
  - target_column: extra_bank_name
    jsonpath: after.extraInfo.bankTransfer.bankName
    data_type: TEXT
  - target_column: extra_bank_remark
    jsonpath: after.extraInfo.bankTransfer.remark
    data_type: TEXT
  - target_column: extra_rule_logs
    jsonpath: after.extraInfo.ruleLogs
    data_type: JSONB
    transform_fn: jsonb_passthrough

  # ---- SKIPPED (giữ trong _raw_data only) ----
  # - after.reason                    → empty object {}, 0 business value
  # - after.instrument                → empty object
  # - after.ewalletInfo               → empty object
  # - after.merchant.platformClient   → null, low lookup value
  # - after.merchant.platformMerchantId → null, low lookup value
  # - after.__v                       → Mongoose version marker, internal
  # - after._id                       → captured as _gpay_source_id (system col)
```

**Total**: 30 rules tạo cột native (matches "~30 cột" DoD). 6 fields skipped — vẫn queryable qua `_raw_data->>'...'`.

---

## 4. `transform_fn` whitelist — expanded từ plan v2

| Name | Input example | Output | Implementation note |
|---|---|---|---|
| `mongo_date_ms` | `{"$date": "2024-07-10T..."}` OR `{"$date": 1720593479271}` OR `1720593479271` | `time.Time` | Try ISO-8601 parse first, fallback to int64 ms epoch |
| `oid_to_hex` | `{"$oid": "abc..."}` OR string `"abc..."` | `string` (hex) | Unwrap `$oid` key; pass-through if already string |
| `bigint_str` | `{"$numberLong": "1000"}` OR number 1000 OR string "1000" | `int64` | Unwrap `$numberLong`; `strconv.ParseInt` |
| `numeric_cast` | number OR string "10.5" OR `{"$numberDecimal": "10.5"}` | `decimal.Decimal` (GORM) | Use `shopspring/decimal` if already imported, else `strconv.ParseFloat` |
| `lowercase` | string | string | `strings.ToLower` |
| `jsonb_passthrough` | any (object, array, scalar) | `string` (JSON-marshaled) | `json.Marshal` — stored as TEXT, Postgres casts to JSONB at INSERT time |
| `null_if_empty` | `""` or `{}` or `[]` | `nil` | Treat empty as NULL in target column |

**Security gate**: whitelist only — Transmuter rejects unknown `transform_fn` với error `ErrTransformNotWhitelisted` thay vì lookup dynamic.

---

## 5. Edge cases Transmuter phải handle

### 5.1 Missing path
- Rule `after.merchant.phoneNumber` (field không tồn tại) → `gjson.Get` returns `gjson.Result{Type: Null}` → Transmuter set `target_column = NULL` nếu `is_nullable=true`, else **skip row + log WARN** (not error — avoid bloc batch).

### 5.2 Path trả về wrong type
- Rule `after.amount` data_type=BIGINT nhưng gjson returns string → `transform_fn=bigint_str` convert; nếu fail → record error metric `cdc_transmute_type_mismatch_total{table, column}` + set NULL (nếu nullable) else skip.

### 5.3 Mongo Extended JSON vs BSON-native
- Debezium MongoDB connector `capture.mode: change_streams_update_full` + Avro converter → `$date` wrapper preserved trong JSON strings (`"createdAt": {"$date": "2024-07-10T..."}`).
- Nếu switch sang `value.converter.json.converter.mongodb.extended-json-format`: same shape.
- `mongo_date_ms` transform handles both cases.

### 5.4 Nested null
- `after.merchant.platformClient = null` → gjson returns null → Transmuter treats as NULL.
- Rule có thể omit hoàn toàn — plan skip cột `merchant_platform_client` trong master.

### 5.5 Array len variance
- `after.extraInfo.ruleLogs = []` hôm nay, ngày mai `[{"code":"..."}, ...]` → Store cả array trong JSONB column `extra_rule_logs`. Query analyst dùng `jsonb_array_length(extra_rule_logs)` + `jsonb_path_query_array`.

### 5.6 Empty object `{}`
- `after.reason = {}` → skip rule entirely. Nếu cần cờ "has reason": create derived rule `target_column: has_reason, jsonpath: after.reason, transform_fn: not_empty_object, data_type: BOOLEAN` (chưa trong whitelist v1 — defer v2.1).

---

## 6. TransmuterModule test cases (R1 + R6 coverage)

Unit tests dùng sample record trên + các mutation:

| # | Test name | Input mutation | Expect |
|---|---|---|---|
| 1 | `Test_flat_scalar` | — | All 21 flat cols populated correctly |
| 2 | `Test_mongo_date_iso_string` | expireTime=`{"$date":"2024-07-05T..."}` | expire_time = time.Time(2024-07-05T09:07:37.072Z) |
| 3 | `Test_mongo_date_int_ms` | expireTime=`{"$date": 1720163257072}` | same parsed time |
| 4 | `Test_nested_merchant` | merchant.reference="CSJKXK" | merchant_reference='CSJKXK' |
| 5 | `Test_2level_nested_bank` | extraInfo.bankTransfer.bankCode="bidv" | extra_bank_code='bidv' |
| 6 | `Test_array_jsonb_passthrough` | ruleLogs=[{"code":"R1"}] | extra_rule_logs='[{"code":"R1"}]' (JSONB) |
| 7 | `Test_missing_path_nullable` | omit reqCommand | req_command=NULL |
| 8 | `Test_missing_path_nonnullable_skip_row` | omit bill_id | row skipped + metric cdc_transmute_rule_miss_total++ |
| 9 | `Test_empty_object_skipped` | merchant={} | merchant_* all NULL (or row skipped if nonnullable) |
| 10 | `Test_null_in_nested_object` | merchant.platformClient=null | cột nonnative — no effect |
| 11 | `Test_numeric_cast_from_string` | amount="10000" | amount=10000 (NUMERIC) |
| 12 | `Test_numeric_cast_number_decimal` | amount={"$numberDecimal":"10000.5"} | amount=10000.5 |
| 13 | `Test_idempotent_upsert` | Run Transmute twice same payload | 0 second UPDATE (hash unchanged) |
| 14 | `Test_jsonpath_whitelist_reject` | rule jsonpath="after.../*\* malicious */" | save rejected HTTP 400 |
| 15 | `Test_transform_fn_whitelist_reject` | transform_fn="eval" | save rejected HTTP 400 |
| 16 | `Test_version_bump_on_rule_update` | PATCH rule → jsonpath changes | version=2, previous_version_id set |

---

## 7. FE JsonPath Input UX — sample flow

MappingFieldsPage (R6.4):
1. Admin mở `/registry/<shadow>/mappings`
2. Click **"+ Add mapping rule"** → Modal
3. Select `source_format`: **debezium_after** (default)
4. Input `jsonpath`: `merchant.reference` → auto-prepend `after.` khi source_format=debezium_after
5. Input `target_column`: `merchant_reference`
6. Select `data_type`: `TEXT`
7. (Optional) Select `transform_fn`: (dropdown from whitelist)
8. Click **"Preview"** → call `POST /api/v1/mapping-rules/preview` với 3 shadow row samples
   - Response: `[{source: "...CSJKXK...", extracted: "CSJKXK"}, {source: "...", extracted: "ABC"}, ...]`
   - Table hiển thị 3 rows với color = green (parse OK) / red (missing path / type mismatch)
9. Nếu preview passes → enable **Save** button → `POST /api/mapping-rules` → status=pending (unless admin toggle auto-approve)
10. Admin list view → batch **Approve** → status=approved → Transmuter picks up in next cycle

---

## 8. Approval gate (SOP Stage 2 continuation)

Doc này là **seed data addendum** cho plan v2. Không thay đổi 7 phases; chỉ bổ sung:

- R0 migration — seed 30 rules sau khi ALTER TABLE succeed
- R1 Transmuter tests — dùng record trên + 15 mutation cases
- R6 FE preview — gọi với 3 sample shadow rows từ `cdc_internal.payment_bills`

Chờ user + Architect duyệt:
- **(A)** OK seed + proceed R0+R1 implementation (Muscle start code)
- **(B)** Cần refine rules (ví dụ thêm/bớt cột master, change transform_fn)
- **(C)** Ask about multi-collection rollout order (payment_bills vs refund_requests first?)

Muscle **KHÔNG execute** trước khi user duyệt. Doc này = sẵn-sàng-dùng seed để paste vào migration khi time comes.

---

## 9. SOP Stage coverage

| Stage | Status |
|---|---|
| 1 INTAKE | ✅ sample record nhận + hiểu cấu trúc |
| 2 PLAN | ✅ Doc này + cross-ref plan v2 |
| 3-7 | ⏳ Gated on user approval |
