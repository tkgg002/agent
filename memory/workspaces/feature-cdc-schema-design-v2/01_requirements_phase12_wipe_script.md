# Requirements — Phase 12 Wipe Script

## Mục tiêu

- Tạo script wipe dữ liệu V2 có thể dùng thật cho đợt reset
- Dùng metadata hiện có để drop đúng master/shadow trước khi xóa control-plane

## Definition of Done

1. Có file SQL wipe riêng trong repo
2. Runbook tham chiếu file wipe này
3. Query verify trong runbook khớp schema thật
