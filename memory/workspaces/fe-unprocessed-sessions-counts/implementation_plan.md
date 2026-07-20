# Kế hoạch triển khai: Hiển thị Source/Dest Count trên Modal đối soát

## 1. Goal Description
Cập nhật giao diện modal đối soát (ExecuteHealModal) trong cdc-cms-web: hiển thị thêm 2 cột dữ liệu "Nguồn" (source_count) và "Đích" (dest_count) trong tab "Phiên chưa xử lý" (Unprocessed Sessions), đồng thời hỗ trợ cuộn ngang (scroll) bảng để đảm bảo giao diện hiển thị gọn gàng, trực quan.

## 2. Proposed Changes

### Frontend (cdc-cms-web)

#### [MODIFY] useReconStatus.ts (file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
Bổ sung trường `source_count` và `dest_count` vào interface `UnhealedReport`.

#### [MODIFY] ExecuteHealModal.tsx (file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Thêm hai cột Nguồn/Đích vào mảng `reportColumns` của bảng.
- Định cấu hình `scroll={{ x: 'max-content', y: 200 }}` cho `<Table>` để hỗ trợ cuộn ngang bảng.

## 3. Verification Plan

### Automated Tests
- Chạy build dự án Frontend `npm run build` để kiểm tra lỗi TypeScript biên dịch.

### Manual Verification
- Vận hành kiểm tra trên UI để xem hiển thị dữ liệu Nguồn và Đích.
