# Yêu cầu: Review & Quy hoạch lại cụm Reconciliation

## Bối cảnh
- User yêu cầu review toàn bộ file recon trong 7 thư mục trải dài 2 backend service + 1 frontend
- Mục tiêu: Xác định technical debt, file sai domain, dead code, và lên plan quy hoạch lại
- Kết hợp với kế hoạch feature `ExecuteHealCommand` (tách biệt đối soát & thực thi)

## Phạm vi
- CDS: `service/recon/`, `repository/recon/`, `handler/recon/`
- CMS: `domain/recon/`, `queries/recon/`, `commands/recon/`, `api/recon/`
- FE: `cdc-cms-web/src/` (hooks, pages, components liên quan recon)
- Migrations: `schema/recon_dlq/`

## Definition of Done
1. ✅ Review toàn bộ file, liệt kê purpose + tech debt cho từng file
2. ✅ Xác định file thiếu cho feature ExecuteHealCommand
3. ✅ Lên plan quy hoạch lại (4 phase) có xếp hạng ưu tiên
4. ⬜ User review + approve plan
