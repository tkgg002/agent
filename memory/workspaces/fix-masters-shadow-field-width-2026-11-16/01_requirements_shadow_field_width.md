---
tier: T1
---
# Requirements — Shadow Field column max-width (/masters/)
- User: cột "Shadow Field" trên trang /masters/ (cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:722) phá layout, cần max-width.
- Ràng buộc (L093): KHÔNG dùng ellipsis cắt tên field -> wrap (word-break) trong max-width.
