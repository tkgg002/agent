# 00_context.md — Audit cdc-cms-service Pattern Compliance

## Mục tiêu
Audit toàn bộ 251 file .go trong `data-hub/cdc-cms-service/internal/` để:
1. Đảm bảo đúng pattern Hexagonal Architecture (app/commands, app/queries, app/ports, infra/)
2. Phát hiện vi phạm: raw gorm.DB trong app layer, NATS direct call từ handlers, missing port abstraction
3. Phát hiện bug runtime: SQL sai column như log `column "source_table" does not exist`
4. Báo cáo đầy đủ: file thay đổi, LOC, nguyên nhân

## Phạm vi
- Repo: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`
- Layer audit: api/, app/, infra/, domain/, model/, bootstrap/, server/, router/, middleware/
- 251 file Go

## Constraints
- Brain: KHÔNG sửa code (Rule §12)
- Simplicity First, Minimal Impact
- Không cheat DB hay config
- Dựa trên kết quả tính toán thực tế

## Người thực thi
- Brain (Antigravity): Plan, audit, report
- Muscle (CC CLI): Execute fix khi được approve
