# Requirements — Phase 11 Wipe Bootstrap Runbook

## Mục tiêu

- Tạo runbook vận hành thật cho đợt `wipe & bootstrap`
- Đồng bộ lệnh thao tác thật trong repo với kiến trúc V2 hiện tại

## Bối cảnh

- `Makefile` cũ chỉ chạy `001_init_schema.sql`
- hệ thống hiện đã cần full migrations tới `038`
- đã có seed template riêng cho `cdc_system`

## Definition of Done

1. Có runbook trong repo cho operator/dev
2. `Makefile` không còn migrate kiểu cũ
3. Runbook bám đúng:
   - `cdc_system`
   - `shadow_<source_db>`
   - seed template V2
