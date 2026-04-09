# Task: Fee Configuration Implementation

- [x] Phase 1: Research & Planning
    - [x] Read `Fee Configuration.ini` requirements [Brain]
    - [x] Research existing fee logic in GooPay codebase [Brain]
    - [x] Define data schema for Fee Rules [Brain]
    - [x] Create implementation plan with technical solutions [Brain]
- [/] Phase 2: Core Implementation
    - [ ] Mở rộng logic `BonusRuleLogic` (executeBonusRule.processors.ts) [Muscle]
        - [ ] Hỗ trợ logic Hybrid Fee (Sum + Max/Min Caps) [Muscle]
        - [ ] Hỗ trợ logic Tiered Fee (Iterator qua list tiers) [Muscle]
    - [ ] Cập nhật Interface/DTO cho Rule event params (nếu cần) [Muscle]
- [ ] Phase 3: Integration & Testing
    - [ ] Tạo helper tính phí trong `payment.logic.ts` [Muscle]
    - [ ] Tích hợp gọi Fee Engine vào luồng `submitPayment` [Muscle]
    - [ ] Viết unit tests chứng minh solution cho 5 loại fee [Muscle]
- [ ] Phase 4: Finalization
    - [ ] Chạy /security-agent review [Muscle]
    - [ ] Brain verify DoD và bài học kinh nghiệm [Brain]

## Solutions Overview
### Fee Engine Solution
- Sử dụng JSON configuration trong `event.params`.
- Hỗ trợ formula: `result = fixed + (val * rate / 100)` capped by `[min, max]`.
- Tiered logic: `find tier where tier.min <= amount <= tier.max`.

### Integration Solution
- Service-to-service call: `payment-service` -> `rule-service.executeFeeRule`.
- Data persistence: Lưu data fee gốc và fee sau tính toán vào record Payment.
