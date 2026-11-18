---
tier: T1
---
# Solution — Shadow Field column max-width

## F1 (VERIFIED)
`MasterMappingFieldsPage.tsx:722-747` cột `Shadow Field` không có `width`/style giới hạn; Table (`:1035-1044`) không có `scroll`/`tableLayout` → tên field dài liền mạch (không khoảng trắng) đẩy giãn cột, phá layout.

## Fix (1 phương án)
```tsx
      title: 'Shadow Field',
      dataIndex: 'source_field',
      width: 260,
      // Giới hạn bề ngang: tên field dài sẽ xuống dòng (không ellipsis để không mất tên)
      onCell: () => ({ style: { maxWidth: 260, whiteSpace: 'normal', wordBreak: 'break-all' } }),
```
- Không dùng ellipsis (L093) — tên vẫn hiển thị đủ, chỉ xuống dòng.
- Impact: chỉ cột này của bảng master rules; không đổi logic/data.
