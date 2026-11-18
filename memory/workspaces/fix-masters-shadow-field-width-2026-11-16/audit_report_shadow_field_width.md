---
tier: T1
---
# Audit Report — Shadow Field column max-width (/masters/)

## (a) Line-by-line diff (`git diff src/pages/MasterMappingFieldsPage.tsx`, +3/-0)
```diff
@@ -722,6 +722,9 @@
       title: 'Shadow Field',
       dataIndex: 'source_field',
+      width: 260,
+      // Giới hạn bề ngang: tên field dài sẽ xuống dòng (không ellipsis để không mất tên)
+      onCell: () => ({ style: { maxWidth: 260, whiteSpace: 'normal', wordBreak: 'break-all' } }),
       ...buildShadowSourceColumnFilter(binding, rules),
```
- `width: 260`: hint độ rộng cột cho AntD Table (Table không có `tableLayout="fixed"`) → thêm `onCell.style.maxWidth` để chặn ở `<td>`.
- `wordBreak: 'break-all'`: tên field không có khoảng trắng → cần break-all để xuống dòng. Không dùng ellipsis (L093).
- Spread `buildShadowSourceColumnFilter` chỉ trả `filters`/`onFilter` (`ShadowSourceBadge.tsx:104-107`) → không ghi đè width/onCell.
- Tree column có icon expand + indent nằm trong cùng `<td>` → chữ bị hẹp thêm theo độ sâu, nhưng wrap nên không mất chữ.

## (b) Phản biện
- Không đổi logic/data/API; render, sorter, filter giữ nguyên. Bảng scanned rules (`:1285`) không bị đụng. Không mock.

## (c) Evidence
- Typecheck `npx tsc -b`: biên lai EV-20261116T164030-6c90 (exit=0).
- Giới hạn: biên lai EV-20261116T164030-6c90 chỉ chứng minh compile; hiển thị trực quan cần anh reload /masters/ để xác nhận (phối hợp trace UI).

## (d) Self-healing
- DG1 lượt đầu (`06_laya_summary_dg1_20261116T160819.json`) bị chặn vì REF lệch dòng do em trích sai; đã sửa REF, không phải lỗi code.

## (e) Gates (Laya)
- DG1: `06_laya_summary_dg1_20261116T160908.json` gate=PASS, các claim UNVALIDATED → kết luận bằng code (grep/sed) + EV-20261116T164030-6c90.
- DG3: `06_laya_summary_dg3_20261116T164144.json` gate=PASS, claim UNVALIDATED → kết luận bằng code + EV-20261116T164030-6c90.

## Security
- Chỉ CSS inline, không secret/PII/input người dùng.
