# Validation — Phase 3 Source Objects Semantics

## Commands

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Result

- `tsc -b`: pass
- `vite build`: pass

## Verification Notes

1. `TableRegistry` đã render cột `Shadow Target`.
2. `MasterRegistry` nhận query context và auto-fill form.
3. Placeholder `source_shadow` đã được đổi để phản ánh đúng contract transitional.

## Constraint Confirmed

Trong backend hiện tại, `/api/v1/masters` vẫn validate `source_shadow` bằng regex chỉ cho identifier kiểu `wallet_transactions`, chưa nhận `shadow_<db>.<table>`. FE phase này đã tránh nói sai về điểm đó.
