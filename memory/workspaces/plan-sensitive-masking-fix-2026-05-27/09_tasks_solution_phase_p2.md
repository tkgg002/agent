# 09_tasks_solution_phase_p2 — Hồ sơ giải pháp P2

## M-10: API CRUD mask-config
- **Root cause**: Update config qua SQL thủ công → không audit actor, không validate, risk operator.
- **Solution**: 3 endpoint REST dưới `adminGroup` + `RequireRole("admin")`.
- **Lý do CQRS**: Pattern hiện có của cdc-cms-service (handler → command → repo).
- **Transactional audit**: UPDATE + INSERT audit cùng transaction → không miss record.

## M-11: Admin UI tab Sensitive Masking
- **Root cause**: Operator hiện không có UI nào để cấu hình masking → phải SQL thủ công.
- **Solution**: Tab thứ 3 trong `MappingRuleEditPage` với 3 component:
  - `StrategySelector`: dropdown + Alert hiển thị legal context.
  - `MaskPreview`: client-side preview (HMAC chỉ placeholder vì cần key server).
  - Audit history Table.
- **Lý do client-side preview**: UX nhanh, không round-trip BE cho mỗi keystroke.
- **Lý do show legal context**: Educate operator chọn strategy đúng theo luật.

## M-12: Backfill script
- **Root cause**: Shadow PG có rows `"***"` literal cần dọn.
- **Solution**: Cobra-like script với dry-run mặc định.
- **Lý do dry-run default**: An toàn — operator phải explicit `-dry-run=false`.
- **Idempotent**: WHERE clause filter `LIKE '%"***"%'`, chạy lại không break.
- **Caveat (ADR-005)**: Plaintext gốc đã mất → re-mask theo strategy mới với `null` cho field DROP/HMAC (vì không có plaintext để hash).

## M-13: Compliance evidence doc
- **Root cause**: Audit yêu cầu chứng minh control mapping với điều luật.
- **Solution**: 1 file MD trong `docs/compliance/` với 4 section:
  - Law mapping article ↔ control.
  - Strategy decision matrix.
  - Audit trail evidence.
  - Key rotation procedure.
- **Lý do MD vs PDF**: Code-as-doc, đi cùng repo, version control.

## Tổng impact P2
- Self-service config cho admin, không cần DBA.
- Backfill dọn legacy data.
- Compliance evidence sẵn sàng cho thanh tra.
