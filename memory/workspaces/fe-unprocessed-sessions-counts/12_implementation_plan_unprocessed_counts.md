# Kế hoạch triển khai chi tiết (Local Workspace)

## 1. Các file cần thay đổi
- `cdc-cms-web/src/hooks/useReconStatus.ts`: Thêm `source_count` và `dest_count` vào interface `UnhealedReport`.
- `cdc-cms-web/src/components/ExecuteHealModal.tsx`: Thêm cột "Nguồn" và "Đích" vào `reportColumns` hiển thị trong tab "Phiên chưa xử lý". Cấu hình cuộn ngang cho table.

## 2. Các bước thực hiện
1. Thay đổi interface trong `useReconStatus.ts`.
2. Sửa component `ExecuteHealModal.tsx` để render thêm cột.
3. Chạy `npm run build` trong `cdc-cms-web` để verify compiler.
4. Chạy kiểm chứng UI.
