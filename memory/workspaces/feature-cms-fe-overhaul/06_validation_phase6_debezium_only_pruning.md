# Validation — Phase 6 Debezium-only Pruning

## Command

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Result

- `tsc -b`: pass
- `vite build`: pass

## Regression Found

- `ActivityLog.tsx` ban đầu thiếu `Typography.Text` destructure nên JSX hiểu nhầm `Text` của DOM.
- Đã vá:
  - `const { Title, Text } = Typography;`

## Post-fix State

- FE build sạch sau khi prune route/menu/operations legacy.
