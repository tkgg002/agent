# 12 - AI Implementation Plan

## 1. Mục tiêu
Sửa đổi logic Frontend tại `cdc-cms-web` để đảm bảo topic prefix không bị nhân đôi tên service khi tạo connector qua CMS.

## 2. Các bước triển khai
1. Sửa `parseConnectionSeed` trong `SourceConnectors.tsx` cho MongoDB và SQL.
2. Sửa `useEffect` tự động gán giá trị khi mở form tạo mới.
3. Cập nhật thuộc tính `disabled` và `tooltip` của trường `topicPrefix`.
4. Kiểm tra biên dịch TypeScript bằng `npx tsc --noEmit`.
5. Verify git diff.
