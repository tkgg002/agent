# 14 - Walkthrough & Verification Guide

## 1. Các bước kiểm tra trên giao diện CMS
1. Mở trang **Source Connectors** trên CMS Web.
2. Bấm **Tạo Nguồn Dữ Liệu Mới**.
3. Chọn loại **MongoDB**:
   - Nhập tên Connector: `payment-service`
   - Quan sát trường **Topic Prefix**: Tự động hiển thị `cdc.goopay` (không bị nối `.payment_service`).
   - Kiểm tra khả năng chỉnh sửa: Ô nhập liệu có thể gõ và thay đổi được.
   - Di chuột vào icon Tooltip: Hiển thị hướng dẫn về quy tắc đặt tên Debezium và cách tránh va chạm.
4. Chọn loại **SFTP**:
   - Nhập tên Connector: `testsftp1`
   - Quan sát trường **Kafka Topic (topic)**: Tự động hiển thị `cdc.sftp.testsftp1` và bị khóa mờ (disabled) đúng theo thiết kế.

## 2. Kiểm tra biên dịch mã nguồn
```bash
cd cdc-cms-web
npx tsc --noEmit --project tsconfig.app.json
```
Kết quả: `Exit code 0` — Toàn bộ type và JSX đều hợp lệ.
