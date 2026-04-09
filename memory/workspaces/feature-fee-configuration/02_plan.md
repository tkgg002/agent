# [VI] Kế hoạch triển khai: Cấu hình Phí (Fee Configuration)
# [EN] Implementation Plan: Fee Configuration

Kế hoạch này chi tiết hóa các giải pháp kỹ thuật cho từng giai đoạn để triển khai hệ thống cấu hình phí dịch vụ động.
This plan details technical solutions for each phase to implement a dynamic service fee configuration system.

---

## Phase 2: Core Implementation / Triển khai cốt lõi

### 1. Data Schema Solution / Giải pháp Cấu trúc Dữ liệu
Sử dụng trường `event.params` trong entity `Rule` để cấu hình công thức tính phí.
Use the `event.params` field in the `Rule` entity to configure fee calculation formulas.

**Fields Definition / Định nghĩa các trường:**
- `feeType`: "FIXED" | "PERCENT" | "HYBRID" | "TIERED" | "SUBSCRIPTION"
- `fixedValue`: Phí cố định (VNĐ) / Flat fee amount.
- `percentValue`: Tỷ lệ phần trăm (0.1 = 0.1%) / Percentage rate.
- `minValue` / `maxValue`: Giới hạn dưới/trên cho kết quả / Min/max caps for the result.
- `path`: Key dữ liệu đầu vào (ví dụ: "amount") / Input data key.
- `tiers`: Mảng các mức phí (cho `TIERED`) / Array of fee tiers.
  - `min` / `max`: Khoảng giá trị áp dụng / Range of applicable values.
  - `fixedValue`, `percentValue`, v.v. (tương tự cấp rule).

### 2. Logic Enhancement / Nâng cấp Logic
**File**: `executeBonusRule.processors.ts`

**Solution / Giải pháp:**
- **Hybrid Fee**: Tính `result = fixedValue + (amount * percentValue / 100)`. Sau đó áp dụng `minValue/maxValue` cho tổng kết quả.
  Calculate `result = fixedValue + (amount * percentValue / 100)`. Then apply `minValue/maxValue` to the total result.
- **Tiered Fee**: Duyệt qua mảng `tiers`. Nếu `min <= amount <= max`, sử dụng cấu hình của tier đó để tính phí (hỗ trợ cả Fixed/Percent trong từng tier).
  Iterate through `tiers`. If `min <= amount <= max`, use that tier's configuration to calculate the fee (supports Fixed/Percent within each tier).
- **Subscription**: Xử lý như `FIXED` fee áp dụng định kỳ (trả về giá trị cấu hình).
  Handle as a `FIXED` fee applied periodically (return the configured value).

---

## Phase 3: Integration / Tích hợp Hệ thống

### 1. Transaction Flow / Luồng Giao dịch
**File**: `payment.logic.ts`

**Solution / Giải pháp:**
- Tạo hàm helper `calculateTransactionFee(ctx, data)`.
  Create a helper function `calculateTransactionFee(ctx, data)`.
- Gọi `rule.executeFeeRule` thông qua `ctx.call` với data bao gồm: `amount`, `channelID`, `serviceType`, `userType`, v.v.
  Call `rule.executeFeeRule` via `ctx.call` with data including: `amount`, `channelID`, `serviceType`, `userType`, etc.
- Ghi nhận `feeAmount` vào đối tượng `Payment` trước khi lưu vào database (trong `submitPayment` hoặc `createPayment`).
  Record `feeAmount` in the `Payment` object before saving to the database (in `submitPayment` or `createPayment`).

---

## Verification Plan / Kế hoạch xác minh

### Solution for Testing / Giải pháp Kiểm thử:
- **Unit Tests**: Viết các script test cho `BonusRuleLogic` với mock `ruleList` chứa đủ 5 loại `feeType`.
  Write test scripts for `BonusRuleLogic` with mock `ruleList` containing all 5 `feeType` categories.
- **Integration Test**: Giả lập giao dịch qua API và verify trường `fee` trong MongoDB.
  Simulate transactions via API and verify the `fee` field in MongoDB.
- **Assignee**: Muscle (Chief Engineer)
