# Validation — Phase 2 FE Nav Refactor

## Command

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Result

- `tsc -b`: pass
- `vite build`: pass

## Regression Found During Validation

1. `SourceToMasterWizard.tsx` dùng API `Steps.Step` kiểu cũ.
2. Build fail với:
   - `Property 'Step' does not exist on type 'Steps'`
   - `Property 'children' does not exist on type 'StepsProps'`
3. Đã sửa bằng cách dùng `items` prop thay cho children API cũ.

## Post-fix State

- FE build sạch.
- Search string còn sót cho thấy `cdc_internal` chỉ còn trong file page legacy `CDCInternalRegistry.tsx`, không còn trong menu runtime mới.
