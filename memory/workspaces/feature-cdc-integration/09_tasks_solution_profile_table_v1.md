# 09_tasks_solution_profile_table_v1.md — Task -1.1 (Profiling Tool)

**Ngày**: 2026-04-17
**Scope**: Atomic — chỉ build `cmd/profile_table` Go tool, KHÔNG chạy against DB, KHÔNG migration, KHÔNG Phase 0.
**Reference**: `02_plan_sonyflake_v125_v7_1_final.md` Section 4 (Data Profiling).
**Status**: CODE COMPLETE — awaiting user approval trước khi proceed Task -1.2.

---

## 1. File list

Tất cả nằm tại `cdc-system/centralized-data-service/cmd/profile_table/`:

| File | LOC | Mục đích |
|------|----:|----------|
| `main.go`            | 123 | CLI flags (`--table --sample --output --dsn`), gorm connect, orchestrate sampleAndProfile + streamRows |
| `financial.go`       |  37 | Regex patterns + `IsFinancialField` |
| `financial_test.go`  |  59 | 20 positive + 12 negative cases |
| `locale.go`          | 182 | `DetectNumberLocale` + helper `cleanNumericSample`, `hasAnyDigit`, `allSeparatorsFollowedBy3Digits` |
| `locale_test.go`     | 101 | en_US / vi_VN / de_DE / ambiguous / thousand-group / mixed / empty |
| `profile.go`         | 182 | `FieldProfile`, `TableProfile`, `fieldAccumulator`, `ProfileTable`, `RowIterator` |
| `output.go`          |  56 | `marshalYAML`, `writeOutput` (stdout hoặc file, `MkdirAll`), `sortFieldsByName` |
| **Tổng** | **740** | |

Dependencies: `gorm.io/gorm`, `gorm.io/driver/postgres`, `github.com/tidwall/gjson`, `gopkg.in/yaml.v3` — ALL đã có trong `go.mod` (yaml.v3 được brought in bởi viper). **KHÔNG cần `go mod tidy`.**

---

## 2. Financial regex patterns — EXACT list trong code

`financial.go`:

```go
var financialPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)^(amount|balance|currency|account|price|fee|total|sum|refund|payment|transaction)([_a-z0-9]*)$`),
    regexp.MustCompile(`(?i)^.+_(amount|balance|price|fee|total|sum)$`),
    regexp.MustCompile(`(?i)^(debit|credit|charge|deposit|withdraw|transfer|settlement)([_a-z0-9]*)$`),
}
```

3 patterns tổng:
1. **Prefix form**: `amount*`, `balance*`, `currency*`, `account*`, `price*`, `fee*`, `total*`, `sum*`, `refund*`, `payment*`, `transaction*`
2. **Suffix form**: `*_amount`, `*_balance`, `*_price`, `*_fee`, `*_total`, `*_sum`
3. **Banking verbs**: `debit*`, `credit*`, `charge*`, `deposit*`, `withdraw*`, `transfer*`, `settlement*`

Match logic: OR giữa 3 pattern → `IsFinancialField(name)` = true ngay khi 1 match.

---

## 3. Locale detect algorithm — decision tree

Applied per sample sau khi strip whitespace + currency ($, ₫, €, £, ¥, VND, USD, EUR, đ) + leading '-':

```
IF no ',' AND no '.'                → +en_US +vi_VN +de_DE  (digits-only ambiguous)
ELSE IF has '.' AND no ','
    IF all '.' followed by exactly 3 digits (e.g. "100.000", "1.234.567")
                                    → +en_US +vi_VN +de_DE  (thousand-group ambiguous)
    ELSE                            → +en_US                  (decimal dot: "1234.56")
ELSE IF has ',' AND no '.'
    IF all ',' followed by exactly 3 digits (e.g. "100,000")
                                    → +en_US +vi_VN +de_DE  (thousand-group ambiguous)
    ELSE                            → +vi_VN +de_DE           (decimal comma: "1234,56")
ELSE (both ',' AND '.')
    IF last '.' > last ','          → +en_US                  (thousands ',', decimal '.')
    ELSE                            → +vi_VN +de_DE           (thousands '.', decimal ',')
```

Confidence = `count_locale / total_scored_samples`. de_DE luôn tie với vi_VN (shape identical — không disambiguate được từ dạng chữ).

**Key design decision**: Thousand-group-only shapes (vd `100.000 đ`) là thực sự ambiguous và được contribute vào CẢ 3 locale — explicit ambiguity tốt hơn false confidence.

---

## 4. Test coverage summary

### `financial_test.go`
- **Positive (20)**: amount, total_amount, refund_amount, user_balance, transaction_id, payment_method, currency, price_usd, fee, debit_note, credit_card, charge_id, deposit_amount, withdraw, transfer_ref, settlement_date, final_price, service_fee, grand_total, line_sum
- **Negative (12)**: created_at, updated_at, status, description, user_id, email, phone, address, merchant_name, country_code, note, tags
- **ALL PASS** ✓

### `locale_test.go`
- `TestDetectNumberLocale_EnUS`: `["1,234.56", "1234.56", "$100.00", "-1,000.00"]` → en_US=1.0 ✓
- `TestDetectNumberLocale_ViVN`: `["1.234,56", "1234,56"]` → vi_VN=1.0, de_DE=1.0, no en_US ✓
- `TestDetectNumberLocale_DeDE`: `["1.234,56"]` → de_DE==vi_VN ✓
- `TestDetectNumberLocale_Ambiguous`: `["1234"]` → all 3 = 1.0 ✓
- `TestDetectNumberLocale_ThousandGroupAmbiguous`: `["100.000 đ"]`, `["100,000"]` → all 3 = 1.0 ✓
- `TestDetectNumberLocale_Mixed`: `["1,234.56", "1234.56", "1.234,56", "1234,56"]` → 0.5 / 0.5 / 0.5 ✓
- `TestDetectNumberLocale_Empty`: nil, non-numeric → empty map ✓

Tổng: **7 locale tests, ALL PASS**.

---

## 5. Build output

```
$ go build ./cmd/profile_table
(no output, exit 0)

$ ls -la profile_table
-rwxr-xr-x  18301730 bytes

$ go build ./...
(no output, exit 0)  — full project build PASS

$ go vet ./cmd/profile_table/...
(no output)

$ go test ./cmd/profile_table/... -v -count=1
=== RUN   TestIsFinancialField_Positive
--- PASS: TestIsFinancialField_Positive (0.00s)
    [20 subtests PASS]
=== RUN   TestIsFinancialField_Negative
--- PASS: TestIsFinancialField_Negative (0.00s)
    [12 subtests PASS]
--- PASS: TestDetectNumberLocale_EnUS (0.00s)
--- PASS: TestDetectNumberLocale_ViVN (0.00s)
--- PASS: TestDetectNumberLocale_ThousandGroupAmbiguous (0.00s)
--- PASS: TestDetectNumberLocale_DeDE (0.00s)
--- PASS: TestDetectNumberLocale_Ambiguous (0.00s)
--- PASS: TestDetectNumberLocale_Mixed (0.00s)
--- PASS: TestDetectNumberLocale_Empty (0.00s)
PASS
ok  centralized-data-service/cmd/profile_table  0.307s
```

Binary đã bị xóa sau verify (không commit artifact).

---

## 6. Sample CLI invocation (NOT executed, for user to run khi ready)

```bash
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service

# Build
go build -o ./bin/profile_table ./cmd/profile_table

# Export DSN (khuyên dùng env hơn flag để tránh leak vào shell history)
export DB_DSN="host=localhost port=5432 user=postgres password=XXX dbname=gpay sslmode=disable"

# Run profile against payment_bills (5% BERNOULLI sample, cap 5000 rows)
./bin/profile_table --table=payment_bills --sample=5 --output=./profiles/payment_bills.profile.yaml

# Stdout variant
./bin/profile_table --table=transactions --sample=10

# Explicit DSN variant
./bin/profile_table --table=accounts --sample=5 \
  --dsn="host=... user=... password=... dbname=... sslmode=disable" \
  --output=./profiles/accounts.profile.yaml
```

Output YAML schema (matches spec):
```yaml
table: payment_bills
sample_size: 4987
generated_at: "2026-04-17T..."
fields:
  - field: amount
    detected_type: mixed
    number_locale:
      en_US: 0.72
      vi_VN: 0.15
      de_DE: 0.15
    confidence: 0.72
    null_rate: 0.02
    sample_size: 4900
    is_financial: true
    admin_override: REQUIRED
  - field: bill_no
    detected_type: string
    confidence: 0.99
    null_rate: 0
    sample_size: 5000
    is_financial: false
```

---

## 7. Scope compliance confirmation

- [x] KHÔNG chạy binary against bất kỳ DB nào (gpay-postgres hay khác)
- [x] KHÔNG tạo migration
- [x] KHÔNG touch code ngoài `cmd/profile_table/`
- [x] KHÔNG start/restart Worker / CMS
- [x] KHÔNG proceed Task -1.2 / Phase 0
- [x] Build PASS (`go build ./...`)
- [x] Tests PASS (`go test ./cmd/profile_table/...`)
- [x] `go vet` clean
- [x] Rule 11: Workspace doc created as NEW file (no overwrite)

Chờ Brain/user review + approve trước khi move Task -1.2.
